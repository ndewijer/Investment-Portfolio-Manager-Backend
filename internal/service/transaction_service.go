package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// TransactionService handles transaction-related business logic operations.
// This includes sell processing with realized gain/loss tracking, insufficient shares
// validation, and IBKR allocation cleanup on deletion.
type TransactionService struct {
	db                      *sql.DB
	transactionRepo         *repository.TransactionRepository
	pfRepo                  *repository.PortfolioFundRepository
	realizedGainLossRepo    *repository.RealizedGainLossRepository
	ibkrRepo                *repository.IbkrRepository
	materializedInvalidator MaterializedInvalidator
}

// NewTransactionService creates a new TransactionService with the provided repository dependencies.
func NewTransactionService(
	db *sql.DB,
	transactionRepo *repository.TransactionRepository,
	pfRepo *repository.PortfolioFundRepository,
	realizedGainLossRepo *repository.RealizedGainLossRepository,
	ibkrRepo *repository.IbkrRepository,
) *TransactionService {
	return &TransactionService{
		db:                   db,
		transactionRepo:      transactionRepo,
		pfRepo:               pfRepo,
		realizedGainLossRepo: realizedGainLossRepo,
		ibkrRepo:             ibkrRepo,
	}
}

// SetMaterializedInvalidator injects the MaterializedInvalidator after construction.
// This breaks the circular initialization order between TransactionService and MaterializedService.
func (s *TransactionService) SetMaterializedInvalidator(m MaterializedInvalidator) {
	s.materializedInvalidator = m
}

// GetOldestTransaction returns the date of the earliest transaction across the given portfolio_fund IDs.
// This is used to determine the earliest date for which portfolio calculations can be performed.
func (s *TransactionService) getOldestTransaction(pfIDs []string) time.Time {
	return s.transactionRepo.GetOldestTransaction(pfIDs)
}

// loadTransactions retrieves transactions for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *TransactionService) loadTransactions(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	return s.transactionRepo.GetTransactions(pfIDs, startDate, endDate)
}

// GetTransactionsperPortfolio retrieves all transactions for a specific portfolio or all transactions if portfolioID is empty.
// Returns enriched transaction data including fund names and IBKR linkage status.
func (s *TransactionService) GetTransactionsperPortfolio(portfolioID string) ([]model.TransactionResponse, error) {
	return s.transactionRepo.GetTransactionsPerPortfolio(portfolioID)
}

// GetTransaction retrieves a single transaction by its ID.
// Returns enriched transaction data including fund name and IBKR linkage status.
func (s *TransactionService) GetTransaction(transactionID string) (model.TransactionResponse, error) {
	return s.transactionRepo.GetTransaction(transactionID)
}

// CreateTransaction creates a new transaction from the provided request.
// For sell transactions, validates sufficient shares and automatically creates
// a RealizedGainLoss record with the calculated cost basis and gain/loss.
//
// Returns the created transaction on success.
// Returns ErrInsufficientShares if selling more shares than currently held.
// Returns an error if date parsing fails or database insertion fails.
func (s *TransactionService) CreateTransaction(ctx context.Context, req request.CreateTransactionRequest) (*model.Transaction, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.pfRepo.WithTx(tx).GetPortfolioFund(req.PortfolioFundID)
	if err != nil {
		return nil, err
	}

	transactionDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	transaction := &model.Transaction{
		ID:              uuid.New().String(),
		PortfolioFundID: req.PortfolioFundID,
		Date:            transactionDate.UTC(),
		Type:            req.Type,
		Shares:          req.Shares,
		CostPerShare:    req.CostPerShare,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// For sell transactions, validate shares and create realized gain/loss record
	if req.Type == "sell" {
		if err := s.createRealizedGainLoss(ctx, tx, transaction.ID, transaction); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), transaction.Date, "", "", req.PortfolioFundID); err != nil {
				log.Printf("failed to regenerate materialized table: %v", err)
			}
		}()
	}

	return transaction, nil
}

// UpdateTransaction updates an existing transaction with the provided changes.
// Only fields present in the request (non-nil) are updated.
//
// Handles realized gain/loss lifecycle:
//   - If the old type was "sell", deletes the existing RealizedGainLoss record
//   - If the new type is "sell", validates shares and creates a new RealizedGainLoss record
//   - If both old and new are "sell", recalculates the RealizedGainLoss record
//
// Returns the updated transaction on success.
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns ErrInsufficientShares if selling more shares than currently held.
func (s *TransactionService) UpdateTransaction(
	ctx context.Context,
	id string,
	req request.UpdateTransactionRequest,
) (*model.Transaction, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	transaction, err := s.transactionRepo.WithTx(tx).GetTransactionByID(id)
	if err != nil {
		return nil, err
	}

	oldType := transaction.Type

	if err := s.applyTransactionUpdates(tx, &transaction, req); err != nil {
		return nil, err
	}

	// Handle realized gain/loss updates when sell type is involved
	if oldType == "sell" || transaction.Type == "sell" {
		if oldType == "sell" {
			if err := s.realizedGainLossRepo.WithTx(tx).DeleteRealizedGainLossByTransactionID(ctx, id); err != nil {
				return nil, fmt.Errorf("failed to delete old realized gain/loss: %w", err)
			}
		}
		if transaction.Type == "sell" {
			if err := s.createRealizedGainLoss(ctx, tx, id, &transaction); err != nil {
				return nil, err
			}
		}
	}

	if err := s.transactionRepo.WithTx(tx).UpdateTransaction(ctx, &transaction); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), transaction.Date, "", "", transaction.PortfolioFundID); err != nil {
				log.Printf("failed to regenerate materialized table: %v", err)
			}
		}()
	}

	return &transaction, nil
}

// DeleteTransaction removes a transaction from the system with proper cleanup.
//
// This method handles:
//   - Realized gain/loss record deletion for sell transactions
//   - IBKR allocation cleanup and status reversion to "pending" when deleting
//     the last allocation for an IBKR transaction
//   - Materialized view invalidation
//
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns an error if the database deletion fails.
func (s *TransactionService) DeleteTransaction(ctx context.Context, id string) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	transaction, err := s.transactionRepo.WithTx(tx).GetTransactionByID(id)
	if err != nil {
		return err
	}

	// Clean up realized gain/loss for sell transactions
	if transaction.Type == "sell" {
		if err := s.realizedGainLossRepo.WithTx(tx).DeleteRealizedGainLossByTransactionID(ctx, id); err != nil {
			return fmt.Errorf("failed to delete realized gain/loss: %w", err)
		}
	}

	// Handle IBKR allocation cleanup and status reversion
	ibkrTxnID, err := s.ibkrRepo.WithTx(tx).GetIbkrTransactionIDByTransactionID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check ibkr allocation: %w", err)
	}

	if ibkrTxnID != "" {
		// Count remaining allocations for this IBKR transaction
		allocCount, err := s.ibkrRepo.WithTx(tx).CountIbkrAllocationsByIbkrTransactionID(ctx, ibkrTxnID)
		if err != nil {
			return fmt.Errorf("failed to count ibkr allocations: %w", err)
		}

		// If this is the last allocation, revert IBKR transaction status to pending
		if allocCount == 1 {
			if err := s.ibkrRepo.WithTx(tx).UpdateIbkrTransactionStatus(ctx, ibkrTxnID, "pending", nil); err != nil {
				return fmt.Errorf("failed to revert ibkr transaction status: %w", err)
			}
		}
	}

	err = s.transactionRepo.WithTx(tx).DeleteTransaction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	if s.materializedInvalidator != nil {
		go func() {
			if err := s.materializedInvalidator.RegenerateMaterializedTable(context.Background(), transaction.Date, "", "", transaction.PortfolioFundID); err != nil {
				log.Printf("failed to regenerate materialized table: %v", err)
			}
		}()
	}

	return nil
}

// calculateCurrentPosition computes the current total shares and total cost for a portfolio fund
// using the weighted average cost method. Optionally excludes a specific transaction by ID
// (used when updating an existing transaction to avoid counting it in position calculation).
//
// Transaction processing:
//   - "buy": increase shares and cost
//   - "dividend": increase shares and cost (dividend reinvestment transactions carry actual shares)
//   - "sell": decrease shares and adjust cost proportionally
//   - "fee": increase cost (fees are part of cost basis)
//
// Note: This differs from fund_metrics.go, which receives dividend reinvestment shares via a
// separate parameter. Here we count dividend transaction shares directly because this method
// operates solely on the transaction table for position validation.
//
// Returns (totalShares, totalCost, error).
func (s *TransactionService) calculateCurrentPosition(tx *sql.Tx, pfID string, excludeTransactionID string) (float64, float64, error) {
	transactions, err := s.transactionRepo.WithTx(tx).GetTransactionsByPortfolioFundID(pfID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load transactions for position: %w", err)
	}

	var shares, cost float64

	for _, t := range transactions {
		if t.ID == excludeTransactionID {
			continue
		}
		switch t.Type {
		case "buy", "dividend":
			shares += t.Shares
			cost += t.Shares * t.CostPerShare
		case "sell":
			shares -= t.Shares
			if shares > 0 {
				cost = (cost / (shares + t.Shares)) * shares
			} else {
				cost = 0
			}
		case "fee":
			cost += t.CostPerShare
		default:
			return 0, 0, fmt.Errorf("unknown transaction type %q for transaction %s", t.Type, t.ID)
		}
	}

	return shares, cost, nil
}

// applyTransactionUpdates patches a transaction with the non-nil fields from an update request.
// Validates that the portfolio fund exists if it's being changed.
func (s *TransactionService) applyTransactionUpdates(tx *sql.Tx, transaction *model.Transaction, req request.UpdateTransactionRequest) error {
	if req.PortfolioFundID != nil {
		_, err := s.pfRepo.WithTx(tx).GetPortfolioFund(*req.PortfolioFundID)
		if err != nil {
			return err
		}
		transaction.PortfolioFundID = *req.PortfolioFundID
	}
	if req.Date != nil {
		transactionDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return err
		}
		transaction.Date = transactionDate.UTC()
	}
	if req.Type != nil {
		transaction.Type = *req.Type
	}
	if req.Shares != nil {
		transaction.Shares = *req.Shares
	}
	if req.CostPerShare != nil {
		transaction.CostPerShare = *req.CostPerShare
	}
	return nil
}

// createRealizedGainLoss validates sufficient shares and creates a RealizedGainLoss record
// for a sell transaction. Used by both CreateTransaction and UpdateTransaction.
func (s *TransactionService) createRealizedGainLoss(ctx context.Context, tx *sql.Tx, transactionID string, transaction *model.Transaction) error {
	shares, cost, err := s.calculateCurrentPosition(tx, transaction.PortfolioFundID, transactionID)
	if err != nil {
		return fmt.Errorf("failed to calculate position: %w", err)
	}

	if shares < transaction.Shares {
		return apperrors.ErrInsufficientShares
	}

	pf, err := s.pfRepo.WithTx(tx).GetPortfolioFund(transaction.PortfolioFundID)
	if err != nil {
		return err
	}

	avgCost := 0.0
	if shares > 0 {
		avgCost = cost / shares
	}
	costBasis := avgCost * transaction.Shares
	saleProceeds := transaction.Shares * transaction.CostPerShare
	realizedGL := saleProceeds - costBasis

	rgl := &model.RealizedGainLoss{
		ID:               uuid.New().String(),
		PortfolioID:      pf.PortfolioID,
		FundID:           pf.FundID,
		TransactionID:    transactionID,
		TransactionDate:  transaction.Date,
		SharesSold:       transaction.Shares,
		CostBasis:        costBasis,
		SaleProceeds:     saleProceeds,
		RealizedGainLoss: realizedGL,
		CreatedAt:        time.Now().UTC(),
	}

	if err := s.realizedGainLossRepo.WithTx(tx).InsertRealizedGainLoss(ctx, rgl); err != nil {
		return fmt.Errorf("failed to record realized gain/loss: %w", err)
	}

	return nil
}
