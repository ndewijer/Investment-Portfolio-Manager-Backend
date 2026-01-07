package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

type DividendRepository struct {
	db *sql.DB
}

func NewDividendRepository(db *sql.DB) *DividendRepository {
	return &DividendRepository{db: db}
}

func (s *DividendRepository) GetDividend(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Dividend, error) {
	if len(pfIDs) == 0 {
		return make(map[string][]model.Dividend), nil
	}

	dividendPlaceholders := make([]string, len(pfIDs))
	for i := range dividendPlaceholders {
		dividendPlaceholders[i] = "?"
	}

	// Retrieve all dividend based on returned portfolio_fund IDs
	dividendQuery := `
		SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned, 
		dividend_per_share, total_amount, reinvestment_status, buy_order_date, reinvestment_transaction_id, created_at
		FROM dividend
		WHERE portfolio_fund_id IN (` + strings.Join(dividendPlaceholders, ",") + `)
	`

	dividendArgs := make([]interface{}, len(pfIDs))
	for i, id := range pfIDs {
		dividendArgs[i] = id
	}

	rows, err := s.db.Query(dividendQuery, dividendArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividendByPortfolio := make(map[string][]model.Dividend)

	for rows.Next() {
		var recordDateStr, exDividendStr, buyOrderStr, createdAtStr string
		var t model.Dividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
			&t.ReinvestmentStatus,
			&buyOrderStr,
			&t.ReinvestmentTransactionId,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend table results: %w", err)
		}

		t.RecordDate, err = ParseTime(recordDateStr)
		if err != nil || t.RecordDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ExDividendDate, err = ParseTime(exDividendStr)
		if err != nil || t.ExDividendDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.BuyOrderDate, err = ParseTime(buyOrderStr)
		if err != nil || t.BuyOrderDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		portfolioID := portfolioFundToPortfolio[t.PortfolioFundID]
		dividendByPortfolio[portfolioID] = append(dividendByPortfolio[portfolioID], t)

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating dividend table: %w", err)
		}
	}

	return dividendByPortfolio, nil
}
