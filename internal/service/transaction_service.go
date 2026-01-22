package service

import (
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// TransactionService handles fund-related business logic operations.
type TransactionService struct {
	transactionRepo *repository.TransactionRepository
}

// NewTransactionService creates a new TransactionService with the provided repository dependencies.
func NewTransactionService(
	transactionRepo *repository.TransactionRepository,
) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
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

// GetTransactionsperPortfolio retrieves all transactions for a specific portfolio or all transactions if portfolioId is empty.
// Returns enriched transaction data including fund names and IBKR linkage status.
func (s *TransactionService) GetTransactionsperPortfolio(portfolioId string) ([]model.TransactionResponse, error) {
	return s.transactionRepo.GetTransactionsPerPortfolio(portfolioId)
}

// GetTransaction retrieves a single transaction by its ID.
// Returns enriched transaction data including fund name and IBKR linkage status.
func (s *TransactionService) GetTransaction(transactionId string) (model.TransactionResponse, error) {
	return s.transactionRepo.GetTransaction(transactionId)
}
