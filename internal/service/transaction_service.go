package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// TransactionService handles fund-related business logic operations.
type TransactionService struct {
	db              *sql.DB
	transactionRepo *repository.TransactionRepository
	pfRepo          *repository.PortfolioFundRepository
}

// NewTransactionService creates a new TransactionService with the provided repository dependencies.
func NewTransactionService(
	db *sql.DB, transactionRepo *repository.TransactionRepository, pfRepo *repository.PortfolioFundRepository,
) *TransactionService {
	return &TransactionService{
		db:              db,
		transactionRepo: transactionRepo,
		pfRepo:          pfRepo,
	}
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
// Generates a new UUID for the transaction and sets the creation timestamp.
//
// Returns the created transaction on success.
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

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return transaction, nil
}

// UpdateTransaction updates an existing transaction with the provided changes.
// Only fields present in the request (non-nil) are updated.
// Updates the createdAt timestamp to reflect the modification time.
//
// Returns the updated transaction on success.
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns an error if date parsing fails or database update fails.
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

	if req.PortfolioFundID != nil {
		_, err = s.pfRepo.WithTx(tx).GetPortfolioFund(*req.PortfolioFundID)
		if err != nil {
			return nil, err
		}
		transaction.PortfolioFundID = *req.PortfolioFundID
	}
	if req.Date != nil {
		transactionDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, err
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

	if err := s.transactionRepo.WithTx(tx).UpdateTransaction(ctx, &transaction); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &transaction, nil
}

// DeleteTransaction removes a transaction from the system.
// Verifies the transaction exists before attempting deletion.
//
// Returns ErrTransactionNotFound if the transaction does not exist.
// Returns an error if the database deletion fails.
func (s *TransactionService) DeleteTransaction(ctx context.Context, id string) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	_, err = s.transactionRepo.WithTx(tx).GetTransactionByID(id)
	if err != nil {
		return err
	}

	err = s.transactionRepo.WithTx(tx).DeleteTransaction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
