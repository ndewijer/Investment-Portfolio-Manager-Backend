package request

type UpdateIbkrConfigRequest struct {
	Enabled                  *bool        `json:"enabled"`
	FlexToken                *string      `json:"flexToken"`
	FlexQueryID              *string      `json:"flexQueryId"`
	TokenExpiresAt           *string      `json:"tokenExpiresAt,omitempty"`
	AutoImportEnabled        *bool        `json:"autoImportEnabled"`
	DefaultAllocationEnabled *bool        `json:"defaultAllocationEnabled"`
	DefaultAllocations       []Allocation `json:"defaultAllocations,omitempty"`
}

type Allocation struct {
	PortfolioID *string  `json:"portfolioId"`
	Percentage  *float64 `json:"percentage"`
}
