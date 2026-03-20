# Endpoint Testing Guide

A systematic approach to testing HTTP endpoints with cookie-cutter templates and failure path identification.

## Cookie-Cutter Test Template

Here's a reusable template you can copy-paste for each endpoint:

```go
// Test{Handler}_{EndpointName} tests the {METHOD} {PATH} endpoint.
//
// WHY: [Explain what this endpoint does and why it's important]
func Test{Handler}_{EndpointName}(t *testing.T) {
    // Happy path
    t.Run("returns success with valid data", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        ps := testutil.NewTestPortfolioService(t, db)
        fs := testutil.NewTestFundService(t, db)
        ms := testutil.NewTestMaterializedService(t, db)
        handler := handlers.NewPortfolioHandler(ps, fs, ms)

        // Create test data
        // ... use testutil builders

        // Create request
        req := httptest.NewRequest(http.MethodGet, "/api/path", nil)
        w := httptest.NewRecorder()

        // Execute
        handler.EndpointName(w, req)

        // Assert
        if w.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d", w.Code)
        }

        var response []model.ResponseType
        json.NewDecoder(w.Body).Decode(&response)
        // ... more assertions
    })

    // Failure paths - see checklist below
}
```

## Failure Path Checklist

Here's a systematic approach to identify failure scenarios for any endpoint:

### 1. **Input Validation Failures**
- [ ] Missing required parameters
- [ ] Invalid parameter format (e.g., malformed UUID)
- [ ] Invalid parameter types (string where number expected)
- [ ] Parameter out of valid range
- [ ] Invalid date formats
- [ ] Empty strings where non-empty required

### 2. **Resource Not Found**
- [ ] Requested ID doesn't exist
- [ ] Related resources don't exist (e.g., portfolio has no funds)

### 3. **Database/Service Errors**
- [ ] Database connection closed/failed
- [ ] Query timeout
- [ ] Foreign key constraint violation

### 4. **Business Logic Errors**
- [ ] Invalid state transition
- [ ] Business rule violation (e.g., can't archive portfolio with open transactions)

### 5. **Edge Cases**
- [ ] Empty result sets
- [ ] Very large result sets
- [ ] Null/nil values in optional fields
- [ ] Concurrent modifications

### 6. **Data Consistency**
- [ ] Stale data
- [ ] Missing related data (e.g., fund without prices)

## Decision Tree for Failure Paths

```
For each endpoint:

1. Does it have path parameters (e.g., {id})?
   YES → Test: missing param, invalid format, not found
   NO  → Skip to step 2

2. Does it have query parameters?
   YES → Test: invalid format, out of range, missing required
   NO  → Skip to step 3

3. Does it require a request body?
   YES → Test: malformed JSON, missing required fields, invalid types
   NO  → Skip to step 4

4. Does it perform calculations/business logic?
   YES → Test: edge cases (empty, zero, null), boundary conditions
   NO  → Skip to step 5

5. Does it access database/external services?
   YES → Test: database closed, query timeout
   NO  → Skip to step 6

6. Does it filter/exclude data?
   YES → Test: verify filtering works correctly
   NO  → Done!

Always test: Happy path with realistic data
```

## Test Helper Functions

Add these to `internal/testutil/http_helpers.go`:

```go
package testutil

import (
    "context"
    "net/http"
    "net/http/httptest"

    "github.com/go-chi/chi/v5"
)

// NewRequestWithURLParams creates an HTTP request with chi URL parameters
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

// NewRequestWithQueryParams creates an HTTP request with query parameters
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
```

## Portfolio Endpoints Test Scenarios

### 1. GET /api/portfolio/ (Portfolios)

**What it does**: Returns all portfolios

**Test scenarios**:
- ✅ Returns empty array when no portfolios
- ✅ Returns all portfolios with data
- ✅ Includes archived portfolios
- ✅ Includes all fields correctly
- ⚠️ Returns 500 when database error

### 2. GET /api/portfolio/{portfolioId} (GetPortfolio)

**What it does**: Returns single portfolio with current summary

**Test scenarios**:
- ✅ Returns portfolio with valid ID
- ✅ Returns portfolio with current valuations
- ⚠️ Returns 400 when portfolio ID is missing
- ⚠️ Returns 400 when portfolio ID is invalid format
- ⚠️ Returns 404 when portfolio doesn't exist
- ⚠️ Returns 500 when database error

### 3. GET /api/portfolio/summary (PortfolioSummary)

**What it does**: Returns current summaries for all active portfolios

**Test scenarios**:
- ✅ Returns empty array when no portfolios
- ✅ Returns summary with basic transactions
- ✅ Excludes archived portfolios
- ✅ Excludes portfolios marked exclude_from_overview
- ✅ Includes realized gains
- ✅ Includes dividends
- ✅ Handles portfolio with no transactions
- ✅ Handles portfolio with no fund prices
- ⚠️ Returns 500 on database error

### 4. GET /api/portfolio/history (PortfolioHistory)

**What it does**: Returns historical valuations over date range

**Test scenarios**:
- ✅ Returns empty array when no portfolios
- ✅ Returns history for date range
- ✅ Uses default dates when not provided
- ✅ Returns data only within transaction date range
- ⚠️ Returns 400 when start_date invalid format
- ⚠️ Returns 400 when end_date invalid format
- ⚠️ Handles start_date after end_date gracefully
- ✅ Handles single day range
- ✅ Handles very large date range
- ⚠️ Returns 500 on database error

### 5. GET /api/portfolio/funds (PortfolioFunds)

**What it does**: Returns all portfolio-fund relationships

**Test scenarios**:
- ✅ Returns empty array when no portfolio-funds
- ✅ Returns all portfolio-fund relationships
- ✅ Includes funds from multiple portfolios
- ✅ Handles portfolio with no funds
- ⚠️ Returns 500 on database error

### 6. GET /api/portfolio/funds/{portfolioId} (GetPortfolioFunds)

**What it does**: Returns detailed fund metrics for a specific portfolio

**Test scenarios**:
- ✅ Returns funds with metrics for valid portfolio
- ✅ Returns empty array when portfolio has no funds
- ✅ Calculates shares, cost, value correctly
- ✅ Includes realized gains per fund
- ✅ Includes dividends per fund
- ⚠️ Returns 400 when portfolio ID missing
- ⚠️ Returns 400 when portfolio ID invalid format
- ⚠️ Returns 404 when portfolio doesn't exist
- ✅ Handles fund with no prices
- ✅ Handles fund with only sell transactions
- ⚠️ Returns 500 on database error

## Complete Example: GetPortfolio Tests

```go
package handlers_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

// TestPortfolioHandler_GetPortfolio tests the GET /api/portfolio/{portfolioId} endpoint.
//
// WHY: This endpoint retrieves a single portfolio with its current valuation summary.
// It's critical for portfolio detail views and dashboard widgets showing individual portfolio performance.
func TestPortfolioHandler_GetPortfolio(t *testing.T) {
    setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
        t.Helper()
        db := testutil.SetupTestDB(t)
        ps := testutil.NewTestPortfolioService(t, db)
        fs := testutil.NewTestFundService(t, db)
        ms := testutil.NewTestMaterializedService(t, db)
        return handlers.NewPortfolioHandler(ps, fs, ms), db
    }

    // Happy path
    t.Run("returns portfolio with correct summary", func(t *testing.T) {
        handler, db := setupHandler(t)

        // Create test data
        portfolio := testutil.CreatePortfolio(t, db, "Test Portfolio")
        fund := testutil.CreateFund(t, db, "AAPL")
        pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

        testutil.NewTransaction(pfID).
            WithType("buy").
            WithShares(100).
            WithCostPerShare(10.0).
            Build(t, db)

        testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

        // Create request with URL params
        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/portfolio/"+portfolio.ID,
            map[string]string{"portfolioId": portfolio.ID},
        )
        w := httptest.NewRecorder()

        // Execute
        handler.GetPortfolio(w, req)

        // Assert HTTP status
        if w.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d", w.Code)
        }

        // Assert response body
        var response model.PortfolioSummary
        json.NewDecoder(w.Body).Decode(&response)

        if response.TotalCost != 1000.0 {
            t.Errorf("Expected cost 1000.0, got %.2f", response.TotalCost)
        }
        if response.TotalValue != 1200.0 {
            t.Errorf("Expected value 1200.0, got %.2f", response.TotalValue)
        }
    })

    // Input validation: Invalid format
    t.Run("returns 400 when portfolioId is invalid format", func(t *testing.T) {
        handler, _ := setupHandler(t)

        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/portfolio/not-a-uuid",
            map[string]string{"portfolioId": "not-a-uuid"},
        )
        w := httptest.NewRecorder()

        handler.GetPortfolio(w, req)

        if w.Code != http.StatusBadRequest {
            t.Errorf("Expected 400, got %d", w.Code)
        }

        var response map[string]string
        json.NewDecoder(w.Body).Decode(&response)

        if _, hasError := response["error"]; !hasError {
            t.Error("Expected error field in response")
        }
    })

    // Resource not found
    t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
        handler, _ := setupHandler(t)

        validID := testutil.MakeID()
        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/portfolio/"+validID,
            map[string]string{"portfolioId": validID},
        )
        w := httptest.NewRecorder()

        handler.GetPortfolio(w, req)

        if w.Code != http.StatusNotFound {
            t.Errorf("Expected 404, got %d", w.Code)
        }
    })

    // Edge case: No transactions
    t.Run("handles portfolio with no transactions", func(t *testing.T) {
        handler, db := setupHandler(t)

        portfolio := testutil.CreatePortfolio(t, db, "Empty Portfolio")

        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/portfolio/"+portfolio.ID,
            map[string]string{"portfolioId": portfolio.ID},
        )
        w := httptest.NewRecorder()

        handler.GetPortfolio(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d", w.Code)
        }

        var response model.PortfolioSummary
        json.NewDecoder(w.Body).Decode(&response)

        // Should have zero values, not errors
        if response.TotalValue != 0 {
            t.Errorf("Expected TotalValue 0, got %.2f", response.TotalValue)
        }
    })

    // Database error
    t.Run("returns 500 on database error", func(t *testing.T) {
        handler, db := setupHandler(t)

        portfolio := testutil.CreatePortfolio(t, db, "Test")
        db.Close() // Force error

        req := testutil.NewRequestWithURLParams(
            http.MethodGet,
            "/api/portfolio/"+portfolio.ID,
            map[string]string{"portfolioId": portfolio.ID},
        )
        w := httptest.NewRecorder()

        handler.GetPortfolio(w, req)

        if w.Code != http.StatusInternalServerError {
            t.Errorf("Expected 500, got %d", w.Code)
        }
    })
}
```

## Quick Reference Card

| Endpoint Type | Required Tests |
|---------------|----------------|
| No params | Happy path, empty data, DB error |
| With {id} | + Invalid format, not found |
| With query | + Invalid query values, out of range |
| With body | + Malformed JSON, missing fields |
| Calculations | + Edge cases (zero, null, empty) |
| Filtering | + Verify filter logic |

## Next Steps

1. Create `internal/testutil/http_helpers.go` with helper functions
2. Copy cookie-cutter template for each endpoint
3. Run decision tree to identify failure paths
4. Add tests systematically
5. Run `make test` to verify

Happy testing! 🧪
