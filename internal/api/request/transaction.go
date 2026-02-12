package request

type CreateTransactionRequest struct {
	PortfolioFundID string  `json:"portfolioFundId"`
	Date            string  `json:"date"`
	Type            string  `json:"type"`
	Shares          float64 `json:"shares"`
	CostPerShare    float64 `json:"costPerShare"`
}

type UpdateTransactionRequest struct {
	PortfolioFundID *string  `json:"portfolioFundId,omitempty"`
	Date            *string  `json:"date,omitempty"`
	Type            *string  `json:"type,omitempty"`
	Shares          *float64 `json:"shares,omitempty"`
	CostPerShare    *float64 `json:"costPerShare,omitempty"`
}
