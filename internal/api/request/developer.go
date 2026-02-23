package request

type SetExchangeRateRequest struct {
	Date         string `json:"date"`
	FromCurrency string `json:"fromCurrency"`
	ToCurrency   string `json:"toCurrency"`
	Rate         string `json:"rate"`
}

type SetFundPriceRequest struct {
	Date   string `json:"date"`
	FundID string `json:"fundId"`
	Price  string `json:"price"`
}
