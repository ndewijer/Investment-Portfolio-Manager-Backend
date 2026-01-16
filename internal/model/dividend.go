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

// DividendFund represents a dividend payment with enriched fund information.
// This structure is used for API responses that combine dividend data with fund metadata.
// It includes all dividend payment details along with the associated fund name and dividend type.
type DividendFund struct {
	ID                        string     `json:"ID"`                                  // Unique dividend record ID
	FundID                    string     `json:"fundID"`                              // Fund identifier
	FundName                  string     `json:"fundName"`                            // Name of the fund paying the dividend
	PortfolioFundID           string     `json:"portfolioFundID"`                     // Portfolio fund relationship ID
	RecordDate                time.Time  `json:"recordDate"`                          // Date of record for dividend eligibility
	ExDividendDate            time.Time  `json:"exDividendDate"`                      // Ex-dividend date
	SharesOwned               float64    `json:"sharesOwned"`                         // Number of shares owned on ex-dividend date
	DividendPerShare          float64    `json:"dividendPerShare"`                    // Dividend amount per share
	TotalAmount               float64    `json:"totalAmount"`                         // Total dividend amount (sharesOwned × dividendPerShare)
	ReinvestmentStatus        string     `json:"reinvestmentStatus"`                  // Status: "reinvested", "paid", etc.
	BuyOrderDate              *time.Time `json:"buyOrderDate,omitempty"`              // Date when reinvestment buy order was placed (nil if not reinvested)
	ReinvestmentTransactionId string     `json:"reinvestmentTransactionID,omitempty"` // Transaction ID if dividend was reinvested (empty if not)
	DividendType              string     `json:"dividendType"`                        // Type of dividend: "accumulating", "distributing"
}
