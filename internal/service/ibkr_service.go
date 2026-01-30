package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// IbkrService handles IBKR (Interactive Brokers) integration business logic operations.
type IbkrService struct {
	ibkrRepo           *repository.IbkrRepository
	portfolioRepo      *repository.PortfolioRepository
	transactionService *TransactionService
	fundRepository     *repository.FundRepository
}

// NewIbkrService creates a new IbkrService with the provided repository dependencies.
func NewIbkrService(
	ibkrRepo *repository.IbkrRepository, portfolioRepo *repository.PortfolioRepository, transactionService *TransactionService, fundRepository *repository.FundRepository,
) *IbkrService {
	return &IbkrService{
		ibkrRepo:           ibkrRepo,
		portfolioRepo:      portfolioRepo,
		transactionService: transactionService,
		fundRepository:     fundRepository,
	}
}

// GetIbkrConfig retrieves the IBKR integration configuration.
// Adds a token expiration warning if the token expires within 30 days.
func (s *IbkrService) GetIbkrConfig() (*model.IbkrConfig, error) {
	config, err := s.ibkrRepo.GetIbkrConfig()

	if err != nil {
		return config, err // Return whatever we got
	}
	if config == nil {
		return nil, fmt.Errorf("unexpected nil config")
	}

	if !config.TokenExpiresAt.IsZero() {
		diff := time.Until(*config.TokenExpiresAt)
		if diff.Hours() <= 720.0 {
			config.TokenWarning = fmt.Sprintf("Token expires in %d days",
				int64(diff.Hours()/24))
		}
	}

	return config, err
}

// GetActivePortfolios retrieves all active portfolios that can be used for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
func (s *IbkrService) GetActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

// GetPendingDividends retrieves dividend records with PENDING reinvestment status.
// These dividends can be matched to incoming IBKR dividend transactions.
// Optionally filters by fund symbol or ISIN.
func (s *IbkrService) GetPendingDividends(symbol, isin string) ([]model.PendingDividend, error) {
	return s.ibkrRepo.GetPendingDividends(symbol, isin)
}

// GetInbox retrieves IBKR imported transactions from the inbox.
// Returns transactions filtered by status (defaults to "pending") and optionally by transaction type.
// Used to display imported IBKR transactions that need to be allocated to portfolios.
func (s *IbkrService) GetInbox(status, transactionType string) ([]model.IBKRTransaction, error) {
	return s.ibkrRepo.GetInbox(status, transactionType)
}

// GetInboxCount retrieves the count of IBKR imported transactions with status "pending".
// Returns only the count without fetching full transaction records for efficiency.
func (s *IbkrService) GetInboxCount() (model.IBKRInboxCount, error) {
	return s.ibkrRepo.GetIbkrInboxCount()
}

// GetTransactionAllocations retrieves the allocation details for an IBKR transaction.
// Fetches the transaction and its allocations, then processes and aggregates the data:
//   - Separates fee allocations from trade allocations
//   - Aggregates fees by portfolio ID and includes them in AllocatedCommission
//   - Rounds monetary values to standard precision
//   - Filters out fee transactions from the final response
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction
//
// Returns the transaction allocation summary with portfolio-level details,
// or an error if the transaction is not found or a database error occurs.
func (s *IbkrService) GetTransactionAllocations(transactionID string) (model.IBKRAllocation, error) {

	ibkrTransaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, err
	}

	allocationDetails, err := s.ibkrRepo.GetIbkrTransactionAllocations(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, err
	}

	feesByID := make(map[string]float64)
	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			feesByID[allocation.PortfolioID] += allocation.AllocatedAmount
		}
	}

	allocationDetailsResponse := make([]model.IBKRTransactionAllocationResponse, 0, len(allocationDetails))

	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			continue
		}

		allocationDetailsResponse = append(allocationDetailsResponse, model.IBKRTransactionAllocationResponse{
			PortfolioID:          allocation.PortfolioID,
			PortfolioName:        allocation.PortfolioName,
			AllocationPercentage: allocation.AllocationPercentage,
			AllocatedAmount:      round(allocation.AllocatedAmount),
			AllocatedShares:      round(allocation.AllocatedShares),
			AllocatedCommission:  round(feesByID[allocation.PortfolioID]),
		})
	}

	allocationReturn := model.IBKRAllocation{
		IBKRTransactionID: ibkrTransaction.ID,
		Status:            ibkrTransaction.Status,
		Allocations:       allocationDetailsResponse,
	}

	return allocationReturn, nil
}

func (s *IbkrService) GetEligiblePortfolios(transactionID string) (model.IBKREligiblePortfolioResponse, error) {
	transaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}

	ibkrEPR := model.IBKREligiblePortfolioResponse{}

	// First we try to find the fund on ISIN as it's most reliable
	fund, err := s.fundRepository.GetFundOnSymbolIsin("", transaction.ISIN)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}
	if fund.ID == "" {
		// Second, we try on Symbol.
		fund, err = s.fundRepository.GetFundOnSymbolIsin(transaction.Symbol, "")

		if fund.ID == "" {
			return model.IBKREligiblePortfolioResponse{}, errors.ErrFundNotFound
		} else {
			ibkrEPR.MatchedBy = "symbol"
		}
	} else {
		ibkrEPR.MatchedBy = "isin"
	}
	ibkrEPR.Found = true
	ibkrEPR.FundID = fund.ID
	ibkrEPR.FundName = fund.Name
	ibkrEPR.FundSymbol = fund.Symbol
	ibkrEPR.FundISIN = fund.Isin

	porfolios, err := s.portfolioRepo.GetPortfoliosOnFundID(fund.ID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}
	if len(porfolios) == 0 {
		ibkrEPR.Warning = fmt.Sprintf("Fund '%s' (%s) exists but is not assigned to any portfolio. Please add this fund to a portfolio first.", fund.Name, fund.Symbol)
	}

	ibkrEPR.Portfolios = porfolios

	return ibkrEPR, nil
}
