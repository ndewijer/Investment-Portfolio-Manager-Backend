package ibkr

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client defines the interface for fetching financial data from Interactive Brokers.
// This interface enables dependency injection and testing with mock implementations.
type Client interface {
	RequestIBKRFlexReport(ctx context.Context, token string, queryID int) (FlexQueryResponse, []byte, error)
}

// FinanceClient provides methods for fetching financial data from Interactive Brokers.
// It wraps an HTTP client and provides convenient methods for querying flex queries.
type FinanceClient struct {
	httpClient *http.Client
}

// NewFinanceClient creates a new IBKR client with default HTTP settings.
// The client uses a standard http.Client for making requests to Interactive Brokers endpoints.
//
// Returns:
//   - *FinanceClient: A new client instance ready for use
func NewFinanceClient() *FinanceClient {
	return &FinanceClient{
		httpClient: &http.Client{},
	}
}

func (c *FinanceClient) RequestIBKRFlexReport(ctx context.Context, token string, queryID int) (FlexQueryResponse, []byte, error) {

	if token == "" || queryID == 0 {
		return FlexQueryResponse{}, nil, fmt.Errorf("Missing variables")
	}

	request, err := c.requestIBKRFlexReport(token, queryID)
	if err != nil {
		return FlexQueryResponse{}, nil, err
	}
	report, data, err := c.retrieveIBKRFlexReport(ctx, token, request)
	if err != nil {
		return FlexQueryResponse{}, nil, err
	}

	return report, data, nil
}

func (c *FinanceClient) requestIBKRFlexReport(token string, queryID int) (FlexRequestResponse, error) {
	if queryID == 0 || token == "" {
		return FlexRequestResponse{}, fmt.Errorf("Not all parameters set")
	}

	queryURL := fmt.Sprintf("https://ndcdyn.interactivebrokers.com/AccountManagement/FlexWebService/SendRequest?t=%s&q=%d&v=3", url.PathEscape(token), queryID)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return FlexRequestResponse{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FlexRequestResponse{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return FlexRequestResponse{}, err
	}

	var response FlexRequestResponse
	if err := xml.Unmarshal(data, &response); err != nil {
		return FlexRequestResponse{}, err
	}

	if response.ErrorCode != nil && response.ErrorMessage != nil {
		return response, fmt.Errorf("ibkr error %d: %s", *response.ErrorCode, *response.ErrorMessage)
	}

	return response, nil
}

func (c *FinanceClient) retrieveIBKRFlexReport(ctx context.Context, token string, request FlexRequestResponse) (FlexQueryResponse, []byte, error) {
	if request.Status == "fail" {
		return FlexQueryResponse{}, nil, fmt.Errorf("failed request submitted")
	}

	var response FlexQueryResponse
	var errResponse FlexRequestResponse
	var data []byte

	queryURL := fmt.Sprintf("%s?t=%s&q=%d&v=3", request.URL, token, request.ReferenceCode)
	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return FlexQueryResponse{}, nil, err
	}

	backoff := 2 * time.Second // start at 2s
	maxBackoff := 30 * time.Second
	maxAttempts := 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		fmt.Printf("attempt: %d\n", attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return FlexQueryResponse{}, nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return FlexQueryResponse{}, nil, err
		}

		data, err = io.ReadAll(resp.Body) // = not :=, assigns to outer data
		resp.Body.Close()                 // explicit close, not defer
		if err != nil {
			return FlexQueryResponse{}, nil, err
		}

		errResponse = FlexRequestResponse{} // reset between attempts
		if err := xml.Unmarshal(data, &response); err != nil {
			if err := xml.Unmarshal(data, &errResponse); err != nil {
				return FlexQueryResponse{}, nil, err
			}
		}

		if errResponse.ErrorCode != nil &&
			(*errResponse.ErrorCode == 1018 || *errResponse.ErrorCode == 1019 || *errResponse.ErrorCode == 1021) {
			continue // not ready yet, retry
		}

		if errResponse.ErrorCode != nil {
			return FlexQueryResponse{}, nil, fmt.Errorf("ibkr error %d: %s", *errResponse.ErrorCode, *errResponse.ErrorMessage)
		}

		break // success
	}
	response.ImportedAt = time.Now().UTC()
	response.QueryID = request.ReferenceCode
	return response, data, nil
}
