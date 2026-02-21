package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client defines the interface for fetching financial data from Yahoo Finance API.
// This interface enables dependency injection and testing with mock implementations.
type Client interface {
	// QueryYahooFiveDaySymbol fetches the last 5 days of daily price data for a symbol.
	QueryYahooFiveDaySymbol(symbol string) (Response, error)

	// QueryYahooSymbolByDateRange fetches daily price data for a symbol within a date range.
	QueryYahooSymbolByDateRange(symbol string, startDate, endDate time.Time) (Response, error)

	// ParseChart converts a raw Yahoo Finance API response into a structured price chart.
	ParseChart(yahooResult Response) (PriceChart, error)
}

// FinanceClient provides methods for fetching financial data from Yahoo Finance API.
// It wraps an HTTP client and provides convenient methods for querying stock prices
// and related financial data.
type FinanceClient struct {
	httpClient *http.Client
}

// NewFinanceClient creates a new Yahoo Finance client with default HTTP settings.
// The client uses a standard http.Client for making requests to Yahoo Finance endpoints.
//
// Returns:
//   - *FinanceClient: A new client instance ready for use
func NewFinanceClient() *FinanceClient {
	return &FinanceClient{
		httpClient: &http.Client{},
	}
}

// ParseChart converts a raw Yahoo Finance API response into a structured price chart.
// This method extracts price data (open, close, high, low, volume) and metadata
// (symbol, currency, exchange) from the Yahoo response format.
//
// The method performs validation to ensure:
//   - Timestamp data is present
//   - Close price data is present
//   - Data arrays have matching lengths
//
// Parameters:
//   - yahooResult: Raw response from Yahoo Finance API
//
// Returns:
//   - PriceChart: Structured chart with indicators and metadata
//   - error: If data is missing, malformed, or arrays have mismatched lengths
func (c *FinanceClient) ParseChart(yahooResult Response) (PriceChart, error) {
	if len(yahooResult.Chart.Result) == 0 {
		return PriceChart{}, fmt.Errorf("no results returned from Yahoo Finance")
	}

	result := yahooResult.Chart.Result[0]

	if len(result.Timestamp) == 0 {
		return PriceChart{}, fmt.Errorf("no price data returned")
	}
	if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) == 0 {
		return PriceChart{}, fmt.Errorf("no close prices returned")
	}

	if len(result.Indicators.Quote[0].Close) != len(result.Timestamp) {
		return PriceChart{}, fmt.Errorf("mismatched data lengths")
	}

	indicators := make([]Indicators, 0, len(result.Timestamp))

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i, v := range result.Timestamp {
		indicatorDate := time.Unix(v, 0).UTC().Truncate(24 * time.Hour)

		// Skip today's data - it's provisional intraday data that changes throughout the day.
		// Only use completed trading days (yesterday and older) to ensure data quality.
		if !indicatorDate.Before(today) {
			continue
		}

		if result.Indicators.Quote[0].Close[i] == nil {
			continue
		}

		indicator := Indicators{
			Date:       time.Unix(v, 0).UTC(),
			PriceClose: *result.Indicators.Quote[0].Close[i],
		}

		if result.Indicators.Quote[0].Open[i] != nil {
			indicator.PriceOpen = *result.Indicators.Quote[0].Open[i]
		}
		if result.Indicators.Quote[0].Volume[i] != nil {
			indicator.Volume = *result.Indicators.Quote[0].Volume[i]
		}
		if result.Indicators.Quote[0].High[i] != nil {
			indicator.PriceHigh = *result.Indicators.Quote[0].High[i]
		}
		if result.Indicators.Quote[0].Low[i] != nil {
			indicator.PriceLow = *result.Indicators.Quote[0].Low[i]
		}

		indicators = append(indicators, indicator)
	}

	return PriceChart{
		Symbol:           result.Meta.Symbol,
		Currency:         result.Meta.Currency,
		ExchangeName:     result.Meta.ExchangeName,
		FullExchangeName: result.Meta.FullExchangeName,
		LongName:         result.Meta.LongName,
		Shortname:        result.Meta.Shortname,
		Indicators:       indicators,
	}, nil
}

// GetIndicatorForDate searches for price data matching a specific date.
// The method performs date-only comparison by truncating both the target and
// indicator dates to midnight UTC, ignoring time components.
//
// This is useful for finding a specific day's data when the exact timestamp
// is unknown or variable.
//
// Parameters:
//   - target: The date to search for (time component is ignored)
//
// Returns:
//   - Indicators: The price data for the matching date
//   - bool: true if a match was found, false otherwise
func (c PriceChart) GetIndicatorForDate(target time.Time) (Indicators, bool) {
	targetDay := target.UTC().Truncate(24 * time.Hour)
	for _, ind := range c.Indicators {
		if ind.Date.UTC().Truncate(24 * time.Hour).Equal(targetDay) {
			return ind, true
		}
	}
	return Indicators{}, false
}

// QueryYahooFiveDaySymbol fetches the last 5 days of daily price data for a symbol.
// This method is optimized for retrieving recent price history, typically used
// to get the latest available closing price.
//
// The method uses Yahoo Finance's range-based query format (range=5d) which
// automatically selects the most recent 5 trading days.
//
// Parameters:
//   - symbol: Stock ticker symbol (e.g., "AAPL", "MSFT")
//
// Returns:
//   - Response: Raw API response containing price data
//   - error: If the HTTP request fails, API returns an error, or no results found
func (c *FinanceClient) QueryYahooFiveDaySymbol(symbol string) (Response, error) {
	queryURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=5d", url.PathEscape(symbol))
	result, err := c.queryYahoo(queryURL)
	if err != nil {
		return Response{}, err
	}
	if len(result.Chart.Result) == 0 {
		return Response{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	return result, nil
}

// QueryYahooSymbolByDateRange fetches daily price data for a symbol within a specific date range.
// This method is optimized for historical data backfilling, allowing retrieval of
// price data for any arbitrary date range.
//
// The method uses Yahoo Finance's period-based query format with Unix timestamps,
// providing precise control over the requested date range.
//
// Parameters:
//   - symbol: Stock ticker symbol (e.g., "AAPL", "MSFT")
//   - startDate: Beginning of date range (inclusive)
//   - endDate: End of date range (inclusive)
//
// Returns:
//   - Response: Raw API response containing price data for the range
//   - error: If the HTTP request fails, API returns an error, or no results found
func (c *FinanceClient) QueryYahooSymbolByDateRange(symbol string, startDate, endDate time.Time) (Response, error) {
	// Yahoo's period2 parameter is exclusive, so add 1 day to endDate to ensure
	// we get all data up to and including the endDate
	adjustedEndDate := endDate.AddDate(0, 0, 1)

	queryURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d",
		url.PathEscape(symbol),
		startDate.Unix(),
		adjustedEndDate.Unix(),
	)
	result, err := c.queryYahoo(queryURL)
	if err != nil {
		return Response{}, err
	}
	if len(result.Chart.Result) == 0 {
		return Response{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	return result, nil
}

// queryYahoo is an internal helper that executes HTTP requests to Yahoo Finance API.
// This method handles the common logic for making authenticated requests, reading responses,
// parsing JSON, and checking for API errors.
//
// The method sets required headers:
//   - User-Agent: Mimics a browser to avoid API blocking
//   - Accept: Requests JSON response format
//
// Parameters:
//   - url: Fully-formed Yahoo Finance API URL
//
// Returns:
//   - Response: Parsed API response
//   - error: If HTTP request fails, response parsing fails, or Yahoo API returns an error
func (c *FinanceClient) queryYahoo(url string) (Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Response{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	//nolint:gosec // G704: URL is constructed from a hardcoded Yahoo Finance base with url.PathEscape applied to the symbol; the host cannot be redirected by user input.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var response Response
	if err := json.Unmarshal(data, &response); err != nil {
		return Response{}, err
	}

	if response.Chart.Error != nil {
		return response, fmt.Errorf("yahoo error: %s", *response.Chart.Error)
	}

	return response, nil
}
