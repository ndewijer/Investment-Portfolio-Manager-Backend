package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type YahooFinanceClient struct {
	httpClient *http.Client
}

func NewYahooFinanceClient() *YahooFinanceClient {
	return &YahooFinanceClient{
		httpClient: &http.Client{},
	}
}

func (c *YahooFinanceClient) ParseChart(yahooResult YahooResponse) (YahooPriceChart, error) {

	result := yahooResult.Chart.Result[0]

	if len(result.Timestamp) == 0 {
		return YahooPriceChart{}, fmt.Errorf("no price data returned")
	}
	if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) == 0 {
		return YahooPriceChart{}, fmt.Errorf("no close prices returned")
	}

	if len(result.Indicators.Quote[0].Close) != len(result.Timestamp) {
		return YahooPriceChart{}, fmt.Errorf("mismatched data lengths")
	}

	indicators := make([]YahooIndicators, len(result.Timestamp))
	for i, v := range result.Timestamp {
		indicators[i].Date = time.Unix(v, 0).UTC()
		indicators[i].PriceOpen = result.Indicators.Quote[0].Open[i]
		indicators[i].PriceClose = result.Indicators.Quote[0].Close[i]
		indicators[i].Volume = result.Indicators.Quote[0].Volume[i]
		indicators[i].PriceHigh = result.Indicators.Quote[0].High[i]
		indicators[i].PriceLow = result.Indicators.Quote[0].Low[i]
	}

	return YahooPriceChart{
		Symbol:           result.Meta.Symbol,
		Currency:         result.Meta.Currency,
		ExchangeName:     result.Meta.ExchangeName,
		FullExchangeName: result.Meta.FullExchangeName,
		LongName:         result.Meta.LongName,
		Shortname:        result.Meta.Shortname,
		Indicators:       indicators,
	}, nil
}

func (c YahooPriceChart) GetIndicatorForDate(target time.Time) (YahooIndicators, bool) {
	targetDay := target.UTC().Truncate(24 * time.Hour)
	for _, ind := range c.Indicators {
		if ind.Date.UTC().Truncate(24 * time.Hour).Equal(targetDay) {
			return ind, true
		}
	}
	return YahooIndicators{}, false
}

func (c *YahooFinanceClient) QueryYahooFiveDaySymbol(symbol string) (YahooResponse, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=5d", symbol)
	result, err := c.queryYahoo(url)
	if err != nil {
		return YahooResponse{}, err
	}
	if len(result.Chart.Result) == 0 {
		return YahooResponse{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	return result, nil
}

func (c *YahooFinanceClient) QueryYahooSymbolByDateRange(symbol string, startDate, endDate time.Time) (YahooResponse, error) {
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d",
		symbol,
		startDate.Unix(),
		endDate.Unix(),
	)
	result, err := c.queryYahoo(url)
	if err != nil {
		return YahooResponse{}, err
	}
	if len(result.Chart.Result) == 0 {
		return YahooResponse{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	return result, nil
}

func (c *YahooFinanceClient) queryYahoo(url string) (YahooResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return YahooResponse{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return YahooResponse{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return YahooResponse{}, err
	}

	var yahooResponse YahooResponse
	if err := json.Unmarshal(data, &yahooResponse); err != nil {
		return YahooResponse{}, err
	}

	if yahooResponse.Chart.Error != nil {
		return yahooResponse, fmt.Errorf("yahoo error: %s", *yahooResponse.Chart.Error)
	}

	return yahooResponse, nil
}
