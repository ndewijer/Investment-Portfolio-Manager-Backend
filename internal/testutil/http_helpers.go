package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

// NewRequestWithURLParams creates an HTTP request with chi URL parameters.
// This helper simplifies testing chi handlers that use chi.URLParam() to extract path parameters.
//
// Example:
//
//	req := testutil.NewRequestWithURLParams(
//	    http.MethodGet,
//	    "/api/portfolio/123-456",
//	    map[string]string{"portfolioId": "123-456"},
//	)
func NewRequestWithURLParams(method, path string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range params {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	return req
}

// NewRequestWithQueryParams creates an HTTP request with query parameters.
// This helper simplifies testing handlers that use r.URL.Query() to extract query string parameters.
//
// Example:
//
//	req := testutil.NewRequestWithQueryParams(
//	    http.MethodGet,
//	    "/api/portfolio/history",
//	    map[string]string{
//	        "start_date": "2024-01-01",
//	        "end_date": "2024-12-31",
//	    },
//	)
func NewRequestWithQueryParams(method, path string, queryParams map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	if len(queryParams) > 0 {
		q := req.URL.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req
}
