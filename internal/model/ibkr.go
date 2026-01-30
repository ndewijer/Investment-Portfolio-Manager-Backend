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

type IBKRAllocation struct {
	IBKRTransactionID string                              `json:"ibkrTransactionId"`
	Status            string                              `json:"status"`
	Allocations       []IBKRTransactionAllocationResponse `json:"allocations"`
}

// returned to API
type IBKRTransactionAllocationResponse struct {
	PortfolioID          string  `json:"portfolioID"`
	PortfolioName        string  `json:"PortfolioName"`
	AllocationPercentage float64 `json:"allocationPercentage"`
	AllocatedAmount      float64 `json:"allocatedAmount"`
	AllocatedShares      float64 `json:"allocatedShares"`
	AllocatedCommission  float64 `json:"allocatedCommission"`
}

// full datamodel of database
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
