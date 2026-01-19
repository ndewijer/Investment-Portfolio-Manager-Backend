package model

import "time"

type Transaction struct {
	ID              string
	PortfolioFundID string
	Date            time.Time
	Type            string
	Shares          float64
	CostPerShare    float64
	CreatedAt       time.Time
}
type TransactionResponse struct {
	Id                string    `json:"id"`
	PortfolioFundId   string    `json:"portfolioFundId"`
	FundName          string    `json:"fundName"`
	Date              time.Time `json:"date"`
	Type              string    `json:"type"`
	Shares            float64   `json:"shares"`
	CostPerShare      float64   `json:"costPerShare"`
	IbkrTransactionId string    `json:"ibkrTransactionId,omitempty"`
	IbkrLinked        bool      `json:"ibkrLinked"`
}
