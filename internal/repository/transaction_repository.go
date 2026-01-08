package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (s *TransactionRepository) GetTransactions(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Transaction, error) {
	if len(pfIDs) == 0 {
		return make(map[string][]model.Transaction), nil
	}

	transactionPlaceholders := make([]string, len(pfIDs))
	for i := range transactionPlaceholders {
		transactionPlaceholders[i] = "?"
	}

	// Retrieve all transactions based on returned portfolio_fund IDs
	transactionQuery := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(transactionPlaceholders, ",") + `)
		ORDER BY date ASC
	`

	transactiondArgs := make([]interface{}, len(pfIDs))
	for i, id := range pfIDs {
		transactiondArgs[i] = id
	}

	rows, err := s.db.Query(transactionQuery, transactiondArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionsByPortfolio := make(map[string][]model.Transaction)

	for rows.Next() {

		var dateStr, createdAtStr string
		var t model.Transaction

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		portfolioID := portfolioFundToPortfolio[t.PortfolioFundID]
		transactionsByPortfolio[portfolioID] = append(transactionsByPortfolio[portfolioID], t)
		if portfolioID == "" {
			fmt.Printf("WARNING: Transaction %s has unmapped portfolio_fund_id: %s\n", t.ID, t.PortfolioFundID)
		}
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("ERROR during iteration: %v\n", err)
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionsByPortfolio, nil
}
