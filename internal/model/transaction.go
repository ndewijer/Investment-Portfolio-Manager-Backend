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
