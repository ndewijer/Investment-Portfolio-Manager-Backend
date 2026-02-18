package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DividendService handles dividend-related business logic operations.
type DividendService struct {
	db              *sql.DB
	dividendRepo    *repository.DividendRepository
	fundRepo        *repository.FundRepository
	transactionRepo *repository.TransactionRepository
}

// NewDividendService creates a new DividendService with the provided dependencies.
//
// Parameters:
//   - db: Raw database connection, used to manage database transactions in CreateDividend.
//   - dividendRepo: Repository for dividend table operations.
//   - fundRepo: Repository for fund and portfolio-fund lookups.
//   - transactionRepo: Repository for transaction table operations, including share calculations.
func NewDividendService(
	db *sql.DB,
	dividendRepo *repository.DividendRepository,
	fundRepo *repository.FundRepository,
	transactionRepo *repository.TransactionRepository,
) *DividendService {
	return &DividendService{
		db:              db,
		dividendRepo:    dividendRepo,
		fundRepo:        fundRepo,
		transactionRepo: transactionRepo,
	}
}

// GetAllDividends retrieves all dividend records from the database.
// Returns raw dividend data without fund enrichment.
func (s *DividendService) GetAllDividend() ([]model.Dividend, error) {
	return s.dividendRepo.GetAllDividend()
}

// GetDividend retrieves a single dividend record by its ID.
// Returns ErrDividendNotFound if no record with the given ID exists.
func (s *DividendService) GetDividend(DividendID string) (model.Dividend, error) {
	return s.dividendRepo.GetDividend(DividendID)
}

// GetDividendFund retrieves all dividend records for a specific portfolio with enriched fund information.
// This is the public service method that returns complete dividend details including fund names,
// dividend types, and reinvestment status for all funds held in the portfolio.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve dividends for
//
// Returns a slice of DividendFund containing all historical dividend payments.
func (s *DividendService) GetDividendFund(portfolioID string) ([]model.DividendFund, error) {
	dividendFund, err := s.dividendRepo.GetDividendPerPortfolioFund(portfolioID)
	if err != nil {
		return nil, err
	}
	return dividendFund, nil
}

// loadDividend retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *DividendService) loadDividendPerPF(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividendPerPF(pfIDs, startDate, endDate)
}

// ProcessDividendSharesForDate calculates shares acquired through dividend reinvestment as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
// Returns a map of portfolio_fund ID to total reinvested shares.
func (s *DividendService) processDividendSharesForDate(dividendMap map[string][]model.Dividend, transactions []model.Transaction, date time.Time) (map[string]float64, error) {
	totalDividendMap := make(map[string]float64)

	for pfID, dividend := range dividendMap {
		var dividendShares float64

		for _, div := range dividend {
			if div.ExDividendDate.Before(date) || div.ExDividendDate.Equal(date) {
				if div.ReinvestmentTransactionID != "" {
					// Find the transaction with this ID
					for _, transaction := range transactions {
						if transaction.ID == div.ReinvestmentTransactionID {
							dividendShares += transaction.Shares
							break
						}
					}
				}
			} else {
				break
			}
		}
		totalDividendMap[pfID] = dividendShares
	}

	return totalDividendMap, nil
}

// ProcessDividendAmountForDate calculates the cumulative dividend amount as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
func (s *DividendService) processDividendAmountForDate(dividend []model.Dividend, date time.Time) (float64, error) {
	if len(dividend) == 0 {
		return 0.0, nil
	}
	var totalDividend float64

	for _, d := range dividend {

		if d.ExDividendDate.Before(date) || d.ExDividendDate.Equal(date) {
			totalDividend += d.TotalAmount
		} else {
			break
		}
	}

	return totalDividend, nil
}

// CreateDividend creates a new dividend record, calculating SharesOwned and TotalAmount
// from transactions as of the ex-dividend date.
//
// If BuyOrderDate is provided, a reinvestment transaction is also created atomically
// within the same database transaction. ReinvestmentStatus is determined as follows:
//
//   - STOCK fund, no BuyOrderDate:                              "PENDING"
//   - STOCK fund, BuyOrderDate set, price/shares missing:       "PENDING"
//   - STOCK fund, BuyOrderDate set, reinvested == total amount: "COMPLETED"
//   - STOCK fund, BuyOrderDate set, reinvested < total amount:  "PARTIAL"
//   - Non-STOCK fund, no BuyOrderDate:                          "COMPLETED"
//   - Non-STOCK fund, BuyOrderDate and price/shares provided:   "COMPLETED"
//
// Note: Once a dividend is used in a portfolio (has transactions), it becomes permanent
// and cannot be deleted. This preserves portfolio history and dividend price data.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: CreateDividendRequest containing all required dividend fields
//
// Returns the created dividend with its generated ID, or an error if creation fails.
func (s *DividendService) CreateDividend(ctx context.Context, req request.CreateDividendRequest) (*model.DividendFund, error) {
	portfolioFund, err := s.findPortfolioFund(req.PortfolioFundID)
	if err != nil {
		return nil, err
	}

	if portfolioFund.DividendType == "None" {
		return nil, fmt.Errorf("this fund does not pay out dividends")
	}

	recordDate, err := time.Parse("2006-01-02", req.RecordDate)
	if err != nil {
		return nil, err
	}

	exDividendDate, err := time.Parse("2006-01-02", req.ExDividendDate)
	if err != nil {
		return nil, err
	}

	shares, err := s.transactionRepo.GetSharesOnDate(req.PortfolioFundID, exDividendDate)
	if err != nil {
		return nil, err
	}

	dividend := &model.Dividend{
		ID:               uuid.New().String(),
		FundID:           portfolioFund.FundID,
		PortfolioFundID:  req.PortfolioFundID,
		RecordDate:       recordDate,
		ExDividendDate:   exDividendDate,
		DividendPerShare: req.DividendPerShare,
		SharesOwned:      shares,
		TotalAmount:      shares * req.DividendPerShare,
		CreatedAt:        time.Now().UTC(),
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if req.BuyOrderDate != "" {
		if err := s.applyReinvestment(ctx, tx, portfolioFund, dividend, req); err != nil {
			return nil, err
		}
	} else if portfolioFund.DividendType == "STOCK" {
		dividend.ReinvestmentStatus = "PENDING"
	} else {
		dividend.ReinvestmentStatus = "COMPLETED"
	}

	if err := s.dividendRepo.WithTx(tx).InsertDividend(ctx, dividend); err != nil {
		return nil, fmt.Errorf("failed to create dividend: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return dividendToFund(*dividend, portfolioFund), nil
}

// UpdateDividend updates an existing dividend with the provided changes.
// Only fields present in the request (non-nil) are updated; all others retain their current values.
// SharesOwned and TotalAmount are always recalculated from transactions as of the (possibly updated)
// ex-dividend date.
//
// ReinvestmentStatus is re-evaluated after the update using the same rules as CreateDividend:
//
//   - STOCK fund, no BuyOrderDate:                                       "PENDING"
//   - STOCK fund, BuyOrderDate set, price/shares missing:                "PENDING"
//   - STOCK fund, BuyOrderDate set, reinvested == total amount:          "COMPLETED"
//   - STOCK fund, BuyOrderDate set, reinvested < total amount:           "PARTIAL"
//   - Non-STOCK fund, no BuyOrderDate:                                   "COMPLETED"
//   - Non-STOCK fund, BuyOrderDate and price/shares provided:            "COMPLETED"
//   - Any fund, existing COMPLETED status, no new reinvestment info:     "COMPLETED" (preserved)
//
// If reinvestment info is provided and no reinvestment transaction exists yet, a new one is created.
// If a reinvestment transaction already exists, it is updated in the same database transaction.
//
// Returns ErrDividendNotFound if the dividend does not exist.
// Returns an error if date parsing fails or any database operation fails.
func (s *DividendService) UpdateDividend(
	ctx context.Context,
	id string,
	req request.UpdateDividendRequest,
) (*model.DividendFund, error) {
	dividend, err := s.dividendRepo.GetDividend(id)
	if err != nil {
		return nil, err
	}

	if req.PortfolioFundID != nil {
		dividend.PortfolioFundID = *req.PortfolioFundID
	}

	portfolioFund, err := s.findPortfolioFund(dividend.PortfolioFundID)
	if err != nil {
		return nil, err
	}

	if err := applyUpdateFields(&dividend, req); err != nil {
		return nil, err
	}

	shares, err := s.transactionRepo.GetSharesOnDate(dividend.PortfolioFundID, dividend.ExDividendDate)
	if err != nil {
		return nil, err
	}

	dividend.SharesOwned = shares
	dividend.TotalAmount = shares * dividend.DividendPerShare
	dividend.CreatedAt = time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if req.BuyOrderDate != nil || !dividend.BuyOrderDate.IsZero() {
		if err := s.applyUpdateReinvestment(ctx, tx, portfolioFund, &dividend, req); err != nil {
			return nil, err
		}
	} else if portfolioFund.DividendType == "STOCK" {
		dividend.ReinvestmentStatus = "PENDING"
	} else {
		dividend.ReinvestmentStatus = "COMPLETED"
	}

	if err := s.dividendRepo.WithTx(tx).UpdateDividend(ctx, &dividend); err != nil {
		return nil, fmt.Errorf("failed to update dividend: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return dividendToFund(dividend, portfolioFund), nil
}

// findPortfolioFund looks up a PortfolioFundListing by ID from the full listing.
// Returns ErrFailedToRetrievePortfolioFunds if the ID is not found.
func (s *DividendService) findPortfolioFund(portfolioFundID string) (model.PortfolioFundListing, error) {
	pfs, err := s.fundRepo.GetAllPortfolioFundListings()
	if err != nil {
		return model.PortfolioFundListing{}, err
	}

	for _, v := range pfs {
		if v.ID == portfolioFundID {
			return v, nil
		}
	}

	return model.PortfolioFundListing{}, apperrors.ErrFailedToRetrievePortfolioFunds
}

// applyReinvestment parses the BuyOrderDate and sets ReinvestmentStatus on the dividend.
// For STOCK funds with reinvestment price and shares, it also creates a "dividend" transaction
// within the provided database transaction.
func (s *DividendService) applyReinvestment(ctx context.Context, tx *sql.Tx, portfolioFund model.PortfolioFundListing, dividend *model.Dividend, req request.CreateDividendRequest) error {
	var err error
	dividend.BuyOrderDate, err = time.Parse("2006-01-02", req.BuyOrderDate)
	if err != nil {
		return err
	}

	hasReinvestmentInfo := req.ReinvestmentPrice > 0.0 && req.ReinvestmentShares > 0.0

	if portfolioFund.DividendType == "STOCK" && hasReinvestmentInfo {
		return s.createReinvestmentTransaction(ctx, tx, dividend, req)
	}

	if hasReinvestmentInfo {
		// Non-STOCK fund with reinvestment info: mark complete, no transaction needed.
		dividend.ReinvestmentStatus = "COMPLETED"
		return nil
	}

	dividend.ReinvestmentStatus = "PENDING"
	return nil
}

// applyUpdateReinvestment updates or creates a reinvestment transaction and recalculates
// ReinvestmentStatus for an existing dividend.
//
// If BuyOrderDate is in the request it replaces the existing value on the dividend.
// When reinvestment price and shares are both provided:
//   - STOCK funds: updates the existing reinvestment transaction (if any), or creates a new one.
//   - Non-STOCK funds: marks the dividend COMPLETED without creating a transaction.
//
// When no reinvestment info is provided, COMPLETED status is preserved; all other statuses
// are set to PENDING.
func (s *DividendService) applyUpdateReinvestment(ctx context.Context, tx *sql.Tx, portfolioFund model.PortfolioFundListing, dividend *model.Dividend, req request.UpdateDividendRequest) error {
	if req.BuyOrderDate != nil {
		buyOrderDate, err := time.Parse("2006-01-02", *req.BuyOrderDate)
		if err != nil {
			return err
		}
		dividend.BuyOrderDate = buyOrderDate.UTC()
	}

	var hasReinvestmentInfo bool
	if req.ReinvestmentShares != nil && req.ReinvestmentPrice != nil {
		hasReinvestmentInfo = *req.ReinvestmentPrice > 0.0 && *req.ReinvestmentShares > 0.0
	}

	if portfolioFund.DividendType == "STOCK" && hasReinvestmentInfo {
		if dividend.ReinvestmentTransactionID != "" {
			return s.updateReinvestmentTransaction(ctx, tx, dividend, req)
		}
		reqCreate := request.CreateDividendRequest{
			PortfolioFundID:    dividend.PortfolioFundID,
			ReinvestmentShares: *req.ReinvestmentShares,
			ReinvestmentPrice:  *req.ReinvestmentPrice,
		}
		return s.createReinvestmentTransaction(ctx, tx, dividend, reqCreate)
	}

	if hasReinvestmentInfo {
		// Non-STOCK fund with reinvestment info: mark complete, no transaction needed.
		dividend.ReinvestmentStatus = "COMPLETED"
		return nil
	}

	if dividend.ReinvestmentStatus != "COMPLETED" {
		dividend.ReinvestmentStatus = "PENDING"
	}

	return nil
}

// createReinvestmentTransaction inserts a "dividend" transaction for a STOCK fund reinvestment
// and sets ReinvestmentStatus to "COMPLETED" or "PARTIAL" based on whether the reinvested
// amount matches the total dividend amount.
func (s *DividendService) createReinvestmentTransaction(ctx context.Context, tx *sql.Tx, dividend *model.Dividend, req request.CreateDividendRequest) error {
	transactionID := uuid.New().String()

	transaction := &model.Transaction{
		ID:              transactionID,
		PortfolioFundID: req.PortfolioFundID,
		Date:            dividend.BuyOrderDate,
		Type:            "dividend",
		Shares:          req.ReinvestmentShares,
		CostPerShare:    req.ReinvestmentPrice,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to create reinvestment transaction: %w", err)
	}

	dividend.ReinvestmentTransactionID = transactionID
	if round(req.ReinvestmentShares*req.ReinvestmentPrice) == round(dividend.TotalAmount) {
		dividend.ReinvestmentStatus = "COMPLETED"
	} else {
		dividend.ReinvestmentStatus = "PARTIAL"
	}

	return nil
}

// updateReinvestmentTransaction updates the fields of the existing reinvestment transaction
// linked to the dividend, and recalculates ReinvestmentStatus (COMPLETED or PARTIAL) based
// on whether the new reinvested amount equals the dividend's TotalAmount.
func (s *DividendService) updateReinvestmentTransaction(ctx context.Context, tx *sql.Tx, dividend *model.Dividend, req request.UpdateDividendRequest) error {
	transaction, err := s.transactionRepo.GetTransactionByID(dividend.ReinvestmentTransactionID)
	if err != nil {
		return err
	}
	transaction.PortfolioFundID = dividend.PortfolioFundID
	transaction.Date = dividend.BuyOrderDate
	transaction.Shares = *req.ReinvestmentShares
	transaction.CostPerShare = *req.ReinvestmentPrice
	transaction.CreatedAt = time.Now().UTC()

	if err := s.transactionRepo.WithTx(tx).UpdateTransaction(ctx, &transaction); err != nil {
		return fmt.Errorf("failed to update reinvestment transaction: %w", err)
	}

	if round(*req.ReinvestmentShares**req.ReinvestmentPrice) == round(dividend.TotalAmount) {
		dividend.ReinvestmentStatus = "COMPLETED"
	} else {
		dividend.ReinvestmentStatus = "PARTIAL"
	}

	return nil
}

// dividendToFund maps a Dividend and its associated PortfolioFundListing into a DividendFund response.
func dividendToFund(d model.Dividend, pf model.PortfolioFundListing) *model.DividendFund {
	var buyOrderDate *time.Time
	if !d.BuyOrderDate.IsZero() {
		t := d.BuyOrderDate
		buyOrderDate = &t
	}

	return &model.DividendFund{
		ID:                        d.ID,
		FundID:                    d.FundID,
		FundName:                  pf.FundName,
		PortfolioFundID:           d.PortfolioFundID,
		RecordDate:                d.RecordDate,
		ExDividendDate:            d.ExDividendDate,
		SharesOwned:               d.SharesOwned,
		DividendPerShare:          d.DividendPerShare,
		TotalAmount:               d.TotalAmount,
		ReinvestmentStatus:        d.ReinvestmentStatus,
		BuyOrderDate:              buyOrderDate,
		ReinvestmentTransactionID: d.ReinvestmentTransactionID,
		DividendType:              pf.DividendType,
	}
}

// applyUpdateFields applies optional field updates from an UpdateDividendRequest onto a Dividend.
// Parses and assigns RecordDate, ExDividendDate, and DividendPerShare when provided.
func applyUpdateFields(dividend *model.Dividend, req request.UpdateDividendRequest) error {
	if req.RecordDate != nil {
		recordDate, err := time.Parse("2006-01-02", *req.RecordDate)
		if err != nil {
			return err
		}
		dividend.RecordDate = recordDate.UTC()
	}

	if req.ExDividendDate != nil {
		exDividendDate, err := time.Parse("2006-01-02", *req.ExDividendDate)
		if err != nil {
			return err
		}
		dividend.ExDividendDate = exDividendDate.UTC()
	}

	if req.DividendPerShare != nil {
		dividend.DividendPerShare = *req.DividendPerShare
	}

	return nil
}
