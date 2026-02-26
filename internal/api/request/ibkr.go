package request

import "time"

type UpdateIbkrConfigRequest struct {
	Configured               *bool        `json:"configured"`
	FlexToken                *string      `json:"flexToken"`
	FlexQueryID              *int         `json:"flexQueryId"`
	TokenExpiresAt           *time.Time   `json:"tokenExpiresAt,omitempty"`
	LastImportDate           *time.Time   `json:"lastImportDate,omitempty"`
	AutoImportEnabled        *bool        `json:"autoImportEnabled"`
	Enabled                  *bool        `json:"enabled"`
	DefaultAllocationEnabled *bool        `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
	CreatedAt                *time.Time   `json:"createdAt"`
	UpdatedAt                *time.Time   `json:"updatedAt"`
}

type Allocation struct {
	PortfolioID *string  `json:"portfolioId"`
	Percentage  *float64 `json:"percentage"`
}
