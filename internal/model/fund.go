package model

import "time"

// Fund represents a fund from the database
type Fund struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Isin           string `json:"isin"`
	Symbol         string `json:"symbol"`
	Currency       string `json:"currency"`
	Exchange       string `json:"exchange"`
	InvestmentType string `json:"investmentType"`
	DividendType   string `json:"dividendType"`
}

type FundPrice struct {
	ID     string
	FundID string
	Date   time.Time
	Price  float64
}

type PortfolioFund struct {
	ID                 string  `json:"id"`
	FundId             string  `json:"fundId"`
	FundName           string  `json:"fundName"`
	TotalShares        float64 `json:"totalShares"`
	LatestPrice        float64 `json:"latestPrice"`
	AverageCost        float64 `json:"averageCost"`
	TotalCost          float64 `json:"totalCost"`
	CurrentValue       float64 `json:"currentValue"`
	UnrealizedGainLoss float64 `json:"unrealizedGainLoss"`
	RealizedGainLoss   float64 `json:"realizedGainLoss"`
	TotalGainLoss      float64 `json:"totalGainLoss"`
	TotalDividends     float64 `json:"totalDividends"`
	TotalFees          float64 `json:"totalFees"`
	DividendType       string  `json:"dividendType"`
	InvestmentType     string  `json:"investmentType"`
}
