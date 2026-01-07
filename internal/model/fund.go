package model

import "time"

// Fund represents a fund from the database
type Fund struct {
	ID             string
	Name           string
	Isin           string
	Symbol         string
	Currency       string
	Exchange       string
	InvestmentType string
	DividendType   string
}

type FundPrice struct {
	ID     string
	FundID string
	Date   time.Time
	Price  float64
}
