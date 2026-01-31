package request

// CreatePortfolioRequest represents the request body for creating a portfolio
type CreatePortfolioRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	ExcludeFromOverview bool   `json:"excludeFromOverview"`
}

type UpdatePortfolioRequest struct {
	Name                *string `json:"name,omitempty"`
	Description         *string `json:"description,omitempty"`
	IsArchived          *bool   `json:"isArchived,omitempty"`
	ExcludeFromOverview *bool   `json:"excludeFromOverview,omitempty"`
}
