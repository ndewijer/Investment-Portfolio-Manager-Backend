package model

import "time"

// Allocation represents a portfolio allocation percentage for IBKR imports.
type Allocation struct {
	PortfolioID string  `json:"portfolioId" sql:"portfolio_id"`
	Percentage  float64 `json:"percentage"`
}

// IbkrConfig represents the IBKR (Interactive Brokers) integration configuration.
// Contains settings for flex queries, token management, and default allocation rules.
type IbkrConfig struct {
	Configured               bool         `json:"configured"`
	FlexQueryID              string       `json:"flexQueryId"`
	TokenExpiresAt           *time.Time   `json:"tokenExpiresAt,omitempty"`
	TokenWarning             string       `json:"tokenWarning,omitempty"`
	LastImportDate           *time.Time   `json:"lastImportDate,omitempty"`
	AutoImportEnabled        bool         `json:"autoImportEnabled"`
	Enabled                  bool         `json:"enabled"`
	DefaultAllocationEnabled bool         `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
	CreatedAt                time.Time    `json:"createdAt"`
	UpdatedAt                time.Time    `json:"updatedAt"`
}

// IBKRTransaction represents a transaction imported from Interactive Brokers.
// Stores transaction details including trades, dividends, fees, and other account activities.
// Transactions are initially imported with status "pending" and require allocation to portfolios.
type IBKRTransaction struct {
	ID                string    `json:"id"`
	IBKRTransactionID string    `json:"ibkrTransactionId"`
	TransactionDate   time.Time `json:"transactionDate"`
	Symbol            string    `json:"symbol,omitempty"`
	ISIN              string    `json:"isin,omitempty"`
	Description       string    `json:"description,omitempty"`
	TransactionType   string    `json:"transactionType"`
	Quantity          float64   `json:"quantity,omitempty"`
	Price             float64   `json:"price,omitempty"`
	TotalAmount       float64   `json:"totalAmount"`
	Currency          string    `json:"currency"`
	Fees              float64   `json:"fees"`
	Status            string    `json:"status"`
	ImportedAt        time.Time `json:"importedAt"`
}

// IBKRInboxCount represents the count of IBKR imported transactions.
// Used as the response payload for the inbox count endpoint.
type IBKRInboxCount struct {
	Count int `json:"count"`
}

// IBKRAllocation represents the allocation details for an IBKR transaction.
// Contains the transaction status and a list of how the transaction was allocated across portfolios.
// Used as the response payload for the transaction allocations endpoint.
type IBKRAllocation struct {
	IBKRTransactionID string                              `json:"ibkrTransactionId"`
	Status            string                              `json:"status"`
	Allocations       []IBKRTransactionAllocationResponse `json:"allocations"`
}

// IBKRTransactionAllocationResponse represents a single portfolio allocation for an IBKR transaction.
// Used in API responses to show how a transaction's amount, shares, and fees were allocated to a specific portfolio.
// Fees are aggregated from separate fee transactions and included in the AllocatedCommission field.
type IBKRTransactionAllocationResponse struct {
	PortfolioID          string  `json:"portfolioID"`
	PortfolioName        string  `json:"PortfolioName"`
	AllocationPercentage float64 `json:"allocationPercentage"`
	AllocatedAmount      float64 `json:"allocatedAmount"`
	AllocatedShares      float64 `json:"allocatedShares"`
	AllocatedCommission  float64 `json:"allocatedCommission"`
}

// IBKRTransactionAllocation represents the full database model for an IBKR transaction allocation.
// Stores the complete record of how an IBKR transaction was allocated to a portfolio,
// including the created transaction reference and allocation type (e.g., "trade", "fee").
type IBKRTransactionAllocation struct {
	ID                   string
	IBKRTransactionID    string
	PortfolioID          string
	PortfolioName        string
	AllocationPercentage float64
	AllocatedAmount      float64
	AllocatedShares      float64
	TransactionID        string
	Type                 string
	CreatedAt            time.Time
}

type IBKREligiblePortfolioResponse struct {
	Found      bool        `json:"found"`
	MatchedBy  string      `json:"matchedBy"`
	FundID     string      `json:"fundId"`
	FundName   string      `json:"fundName"`
	FundSymbol string      `json:"fundSymbol"`
	FundISIN   string      `json:"fundIsin"`
	Portfolios []Portfolio `json:"portfolios"`
	Warning    string      `json:"warning,omitempty"`
}
