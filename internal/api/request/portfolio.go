package request

// CreatePortfolioRequest represents the request body for creating a portfolio
type CreatePortfolioRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	ExcludeFromOverview bool   `json:"excludeFromOverview"`
}

type UpdatePortfolioRequest struct {
	ID                  *string `json:"id"`
	Name                *string `json:"name,omitempty"`
	Description         *string `json:"description,omitempty"`
	IsArchived          *bool   `json:"isArchived,omitempty"`
	ExcludeFromOverview *bool   `json:"excludeFromOverview,omitempty"`
}

type CreatePortfolioFundRequest struct {
	PortfolioID string `json:"portfolioId"`
	FundID      string `json:"fundId"`
}
