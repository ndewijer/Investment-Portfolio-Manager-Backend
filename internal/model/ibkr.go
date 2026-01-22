package model

import "time"

// Allocation represents a portfolio allocation percentage for IBKR imports.
type Allocation struct {
	PortfolioId string  `json:"portfolioId" sql:"portfolio_id"`
	Percentage  float64 `json:"percentage"`
}

// IbkrConfig represents the IBKR (Interactive Brokers) integration configuration.
// Contains settings for flex queries, token management, and default allocation rules.
type IbkrConfig struct {
	Configured               bool         `json:"configured"`
	FlexQueryId              string       `json:"flexQueryId"`
	TokenExpiresAt           time.Time    `json:"tokenExpiresAt,omitempty"`
	TokenWarning             string       `json:"tokenWarning,omitempty"`
	LastImportDate           time.Time    `json:"lastImportDate,omitempty"`
	AutoImportEnabled        bool         `json:"autoImportEnabled"`
	Enabled                  bool         `json:"enabled"`
	DefaultAllocationEnabled bool         `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
	CreatedAt                time.Time    `json:"createdAt"`
	UpdatedAt                time.Time    `json:"updatedAt"`
}
