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

// TestIbkrConnectionRequest holds the credentials for a one-off IBKR connection test.
// The token is supplied in plaintext here because this endpoint is used to verify
// credentials before they are saved to the encrypted config.
type TestIbkrConnectionRequest struct {
	FlexQueryID string `json:"flexQueryId"`
	FlexToken   string `json:"flexToken"`
}
