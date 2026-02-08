package request

type CreateFundRequest struct {
	Name           string `json:"name"`
	Isin           string `json:"isin"`
	Symbol         string `json:"symbol"`
	Currency       string `json:"currency"`
	Exchange       string `json:"exchange"`
	DividendType   string `json:"dividendType"`
	InvestmentType string `json:"investmentType"`
}
