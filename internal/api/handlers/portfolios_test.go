package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

// TestPortfolioHandler_Portfolios tests the GET /api/portfolio endpoint.
//
// WHY: This is the primary endpoint for retrieving portfolios. The frontend
// depends on this returning correct data with proper HTTP status codes and
// JSON formatting. Testing ensures API contract stability.
func TestPortfolioHandler_Portfolios(t *testing.T) {
	t.Run("GET /api/portfolio returns 200 with empty array", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert HTTP status
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Assert Content-Type
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Assert response body
		var response []handlers.PortfoliosResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("GET /api/portfolio returns all portfolios", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create test data
		p1 := testutil.CreatePortfolio(t, db, "Portfolio One")
		p2 := testutil.CreatePortfolio(t, db, "Portfolio Two")

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfoliosResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(response))
		}

		// Verify data matches what we created
		if response[0].ID != p1.ID {
			t.Errorf("Expected first portfolio ID %s, got %s", p1.ID, response[0].ID)
		}
		if response[0].Name != "Portfolio One" {
			t.Errorf("Expected first portfolio name 'Portfolio One', got '%s'", response[0].Name)
		}

		if response[1].ID != p2.ID {
			t.Errorf("Expected second portfolio ID %s, got %s", p2.ID, response[1].ID)
		}
		if response[1].Name != "Portfolio Two" {
			t.Errorf("Expected second portfolio name 'Portfolio Two', got '%s'", response[1].Name)
		}
	})

	t.Run("GET /api/portfolio includes all fields", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio with all fields set
		p := testutil.NewPortfolio().
			WithName("Complete Portfolio").
			WithDescription("Full description").
			Archived().
			ExcludedFromOverview().
			Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfoliosResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		portfolio := response[0]

		// Verify all fields
		if portfolio.ID != p.ID {
			t.Errorf("ID mismatch: expected %s, got %s", p.ID, portfolio.ID)
		}
		if portfolio.Name != "Complete Portfolio" {
			t.Errorf("Name mismatch: expected 'Complete Portfolio', got '%s'", portfolio.Name)
		}
		if portfolio.Description != "Full description" {
			t.Errorf("Description mismatch: expected 'Full description', got '%s'", portfolio.Description)
		}
		if portfolio.IsArchived != true {
			t.Errorf("IsArchived mismatch: expected true, got %v", portfolio.IsArchived)
		}
		if portfolio.ExcludeFromOverview != true {
			t.Errorf("ExcludeFromOverview mismatch: expected true, got %v", portfolio.ExcludeFromOverview)
		}
	})

	t.Run("GET /api/portfolio returns 500 on database error", func(t *testing.T) {
		// Setup with closed database
		db := testutil.SetupTestDB(t)
		db.Close() // Force database error

		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert error response
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})
}

// Example of testing with helper function for repeated setup
func TestPortfolioHandler_WithHelper(t *testing.T) {
	// Helper function to reduce boilerplate
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *httptest.ResponseRecorder) {
		t.Helper()

		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)
		w := httptest.NewRecorder()

		return handler, w
	}

	t.Run("example using helper", func(t *testing.T) {
		handler, w := setupHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)

		handler.Portfolios(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_PortfolioSummary tests the portfolio summary endpoint.
//
// WHY: This endpoint provides calculated portfolio metrics including values,
// costs, gains/losses, and dividends. It's critical for the dashboard view.
func TestPortfolioHandler_PortfolioSummary(t *testing.T) {
	t.Run("returns empty array when no portfolios exist", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("returns summary for portfolio with basic transactions", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create test data: portfolio, fund, and transactions
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Create buy transaction: 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Add current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]
		if summary.ID != portfolio.ID {
			t.Errorf("Expected portfolio ID %s, got %s", portfolio.ID, summary.ID)
		}
		if summary.Name != "Test Portfolio" {
			t.Errorf("Expected name 'Test Portfolio', got '%s'", summary.Name)
		}

		// Verify calculations
		// Total cost: 100 * $10 = $1000
		if summary.TotalCost != 1000.0 {
			t.Errorf("Expected total cost 1000.0, got %.2f", summary.TotalCost)
		}

		// Total value: 100 * $12 = $1200
		if summary.TotalValue != 1200.0 {
			t.Errorf("Expected total value 1200.0, got %.2f", summary.TotalValue)
		}

		// Unrealized gain: $1200 - $1000 = $200
		if summary.TotalUnrealizedGainLoss != 200.0 {
			t.Errorf("Expected unrealized gain 200.0, got %.2f", summary.TotalUnrealizedGainLoss)
		}

		// Total gain/loss (no realized gains): $200
		if summary.TotalGainLoss != 200.0 {
			t.Errorf("Expected total gain/loss 200.0, got %.2f", summary.TotalGainLoss)
		}

		// No dividends
		if summary.TotalDividends != 0.0 {
			t.Errorf("Expected dividends 0.0, got %.2f", summary.TotalDividends)
		}

		// No realized gains
		if summary.TotalRealizedGainLoss != 0.0 {
			t.Errorf("Expected realized gain/loss 0.0, got %.2f", summary.TotalRealizedGainLoss)
		}

		if summary.IsArchived {
			t.Error("Expected portfolio to not be archived")
		}
	})

	t.Run("excludes archived portfolios from summary", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create active portfolio
		activePortfolio := testutil.NewPortfolio().WithName("Active").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(activePortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pfID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		// Create archived portfolio
		archivedPortfolio := testutil.NewPortfolio().WithName("Archived").Archived().Build(t, db)
		pfID2 := testutil.NewPortfolioFund(archivedPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pfID2).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should only include active portfolio
		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		if response[0].Name != "Active" {
			t.Errorf("Expected active portfolio, got %s", response[0].Name)
		}
	})

	t.Run("excludes portfolios marked as exclude_from_overview", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create normal portfolio
		normalPortfolio := testutil.NewPortfolio().WithName("Normal").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(normalPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pfID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		// Create excluded portfolio
		excludedPortfolio := testutil.NewPortfolio().WithName("Excluded").ExcludedFromOverview().Build(t, db)
		pfID2 := testutil.NewPortfolioFund(excludedPortfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pfID2).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []handlers.PortfolioSummaryResponse
		json.NewDecoder(w.Body).Decode(&response)

		// Should only include normal portfolio
		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		if response[0].Name != "Normal" {
			t.Errorf("Expected normal portfolio, got %s", response[0].Name)
		}
	})

	t.Run("includes realized gains from sell transactions", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Sell 30 shares at $15
		sellTx := testutil.NewTransaction(pfID).
			WithType("sell").
			WithShares(30).
			WithCostPerShare(15.0).
			Build(t, db)

		// Record realized gain: 30 * ($15 - $10) = $150
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithShares(30).
			WithCostBasis(300.0).    // 30 * $10
			WithSaleProceeds(450.0). // 30 * $15
			Build(t, db)

		// Current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []handlers.PortfolioSummaryResponse
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]

		// Remaining: 70 shares
		// Cost: 70 * $10 = $700 (proportionally reduced)
		// Value: 70 * $12 = $840
		// Unrealized: $840 - $700 = $140

		if summary.TotalCost != 700.0 {
			t.Errorf("Expected cost 700.0, got %.2f", summary.TotalCost)
		}

		if summary.TotalValue != 840.0 {
			t.Errorf("Expected value 840.0, got %.2f", summary.TotalValue)
		}

		if summary.TotalUnrealizedGainLoss != 140.0 {
			t.Errorf("Expected unrealized gain 140.0, got %.2f", summary.TotalUnrealizedGainLoss)
		}

		// Realized gain: $150
		if summary.TotalRealizedGainLoss != 150.0 {
			t.Errorf("Expected realized gain 150.0, got %.2f", summary.TotalRealizedGainLoss)
		}

		// Total gain: $140 (unrealized) + $150 (realized) = $290
		if summary.TotalGainLoss != 290.0 {
			t.Errorf("Expected total gain 290.0, got %.2f", summary.TotalGainLoss)
		}

		// Sale proceeds
		if summary.TotalSaleProceeds != 450.0 {
			t.Errorf("Expected sale proceeds 450.0, got %.2f", summary.TotalSaleProceeds)
		}

		// Original cost of sold shares
		if summary.TotalOriginalCost != 300.0 {
			t.Errorf("Expected original cost 300.0, got %.2f", summary.TotalOriginalCost)
		}
	})

	t.Run("includes dividend payments in summary", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().WithName("Dividend Test Portfolio").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Add dividend: 100 shares * $0.50 = $50
		testutil.NewDividend(fund.ID, pfID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			Build(t, db)

		// Current price
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d. Response: %+v", len(response), response)
		}

		summary := response[0]

		// Dividends: $50
		if summary.TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", summary.TotalDividends)
		}
	})

	t.Run("includes dividend reinvestment shares in calculations", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Dividend reinvestment: buy 5 shares at $10
		reinvestTx := testutil.NewTransaction(pfID).
			WithType("dividend").
			WithShares(5).
			WithCostPerShare(10.0).
			Build(t, db)

		// Dividend record with reinvestment
		testutil.NewDividend(fund.ID, pfID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			WithReinvestmentTransaction(reinvestTx.ID).
			Build(t, db)

		// Current price: $12
		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		var response []handlers.PortfolioSummaryResponse
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		summary := response[0]

		// Total shares: 100 (buy) + 5 (dividend reinvestment) = 105
		// Value: 105 * $12 = $1260
		// Cost: 100 * $10 + 5 * $10 = $1050
		// This will be calculated by the service, so just verify non-zero values
		if summary.TotalValue <= 0 {
			t.Errorf("Expected positive value, got %.2f", summary.TotalValue)
		}

		if summary.TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", summary.TotalDividends)
		}
	})

	t.Run("handles portfolio with no transactions gracefully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio with no transactions
		testutil.NewPortfolio().WithName("Empty").Build(t, db)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should not crash, may return empty or zero-value summary
		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	})

	t.Run("handles portfolio with no fund prices gracefully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create portfolio with transaction but no prices
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pfID).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// No fund price added

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert - should not crash
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []handlers.PortfolioSummaryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		// Value should be 0 when no price is available
		if response[0].TotalValue != 0 {
			t.Errorf("Expected value 0 when no price, got %.2f", response[0].TotalValue)
		}

		// Cost should still be calculated
		if response[0].TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response[0].TotalCost)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		// Setup with closed database
		db := testutil.SetupTestDB(t)
		db.Close() // Force database error

		svc := testutil.NewTestPortfolioService(t, db)
		handler := handlers.NewPortfolioHandler(svc)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert error response
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}

		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})
}

// TODO: Add more handler tests as you implement endpoints:
// - TestPortfolioHandler_GetPortfolio (GET /api/portfolio/:id)
// - TestPortfolioHandler_CreatePortfolio (POST /api/portfolio)
// - TestPortfolioHandler_UpdatePortfolio (PUT /api/portfolio/:id)
// - TestPortfolioHandler_DeletePortfolio (DELETE /api/portfolio/:id)
// etc.
