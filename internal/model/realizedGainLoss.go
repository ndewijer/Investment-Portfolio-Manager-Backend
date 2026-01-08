package model

import "time"

type RealizedGainLoss struct {
	ID               string
	PortfolioID      string
	FundID           string
	TransactionID    string
	TransactionDate  time.Time
	SharesSold       float64
	CostBasis        float64
	SaleProceeds     float64
	RealizedGainLoss float64
	CreatedAt        time.Time
}
