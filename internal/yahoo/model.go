package yahoo

import "time"

type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency         string `json:"currency"`
				Symbol           string `json:"symbol"`
				ExchangeName     string `json:"exchangeName"`
				FullExchangeName string `json:"fullExchangeName"`
				LongName         string `json:"longName"`
				Shortname        string `json:"shortName"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *string `json:"error"`
	} `json:"chart"`
}

type YahooPriceChart struct {
	Currency         string            `json:"currency"`
	Symbol           string            `json:"symbol"`
	ExchangeName     string            `json:"exchangeName"`
	FullExchangeName string            `json:"fullExchangeName"`
	LongName         string            `json:"longName"`
	Shortname        string            `json:"shortName"`
	Indicators       []YahooIndicators `json:"indicators"`
}

type YahooIndicators struct {
	Date       time.Time
	PriceOpen  float64
	PriceClose float64
	Volume     int64
	PriceHigh  float64
	PriceLow   float64
}
