package model

import "time"

// Fund represents a dividend from the database
type Dividend struct {
	ID                        string
	FundID                    string
	PortfolioFundID           string
	RecordDate                time.Time
	ExDividendDate            time.Time
	SharesOwned               float64
	DividendPerShare          float64
	TotalAmount               float64
	ReinvestmentStatus        string
	BuyOrderDate              time.Time
	ReinvestmentTransactionId string
	CreatedAt                 time.Time
}
