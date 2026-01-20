package model

import "time"

type Allocation struct {
	PortfolioId string `json:"portfolioId"`
	Percentage  int64  `json:"percentage"`
}

type IbkrConfig struct {
	Configured               bool         `json:"configured"`
	FlexQueryId              string       `json:"flexQueryId"`
	TokenExpiresAt           time.Time    `json:"tokenExpiresAt,omitempty"`
	TokenWarning             string       `json:"tokenWarning,omitempty"`
	LastImportDate           time.Time    `json:"lastImportDate,omitempty"`
	AutoImprtEnabled         bool         `json:"autoImprtEnabled"`
	Enabled                  bool         `json:"enabled"`
	DefaultAllocationEnabled bool         `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
	CreatedAt                time.Time    `json:"createdAt"`
	UpdatedAt                time.Time    `json:"updatedAt"`
}
