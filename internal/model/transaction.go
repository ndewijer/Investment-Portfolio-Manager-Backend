package model

import "time"

// Transaction represents a buy or sell transaction for a portfolio fund.
// Used internally for calculations and data processing.
type Transaction struct {
	ID              string    `json:"id"`
	PortfolioFundID string    `json:"portfolioFundId"`
	Date            time.Time `json:"date"`
	Type            string    `json:"type"`
	Shares          float64   `json:"shares"`
	CostPerShare    float64   `json:"costPerShare"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
}

// TransactionResponse represents a transaction with enriched data for API responses.
// Includes fund name and IBKR linkage information.
type TransactionResponse struct {
	ID                string    `json:"id"`
	PortfolioFundID   string    `json:"portfolioFundId"`
	FundName          string    `json:"fundName"`
	Date              time.Time `json:"date"`
	Type              string    `json:"type"`
	Shares            float64   `json:"shares"`
	CostPerShare      float64   `json:"costPerShare"`
	IbkrTransactionID string    `json:"ibkrTransactionId,omitempty"`
	IbkrLinked        bool      `json:"ibkrLinked"`
}
