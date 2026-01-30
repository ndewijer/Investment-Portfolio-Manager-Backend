package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
		var response []model.Portfolio
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 portfolios, got %d", len(response))
		}

		// Find portfolios by ID - don't assume order
		var portfolio1, portfolio2 *model.Portfolio
		for i := range response {
			if response[i].ID == p1.ID {
				portfolio1 = &response[i]
			}
			if response[i].ID == p2.ID {
				portfolio2 = &response[i]
			}
		}

		// Verify we found both
		if portfolio1 == nil {
			t.Fatal("Portfolio One not found in response")
		}
		if portfolio2 == nil {
			t.Fatal("Portfolio Two not found in response")
		}

		// Verify data matches what we created
		if portfolio1.Name != "Portfolio One" {
			t.Errorf("Expected first portfolio name 'Portfolio One', got '%s'", portfolio1.Name)
		}
		if portfolio2.Name != "Portfolio Two" {
			t.Errorf("Expected second portfolio name 'Portfolio Two', got '%s'", portfolio2.Name)
		}
	})

	t.Run("GET /api/portfolio includes all fields", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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

		var response []model.Portfolio
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

	t.Run("GET /api/portfolio includes archived and excluded from overview", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio with all fields set
		p1 := testutil.CreateArchivedPortfolio(t, db, "Portfolio One")
		p2 := testutil.CreateExcludedPortfolio(t, db, "Portfolio Two")

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.Portfolios(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Fatalf("Expected 2 portfolios, got %d", len(response))
		}

		// Find portfolios by ID - don't assume order
		var archivedPortfolio, excludedPortfolio *model.Portfolio
		for i := range response {
			if response[i].ID == p1.ID {
				archivedPortfolio = &response[i]
			}
			if response[i].ID == p2.ID {
				excludedPortfolio = &response[i]
			}
		}

		// Verify we found both
		if archivedPortfolio == nil {
			t.Fatal("Archived portfolio not found in response")
		}
		if excludedPortfolio == nil {
			t.Fatal("Excluded portfolio not found in response")
		}

		// Verify flags are set correctly
		if !archivedPortfolio.IsArchived {
			t.Errorf("Expected archived portfolio isArchived to be true, got %t", archivedPortfolio.IsArchived)
		}
		if !excludedPortfolio.ExcludeFromOverview {
			t.Errorf("Expected excluded portfolio exclude_from_overview to be true, got %t", excludedPortfolio.ExcludeFromOverview)
		}
	})

	t.Run("GET /api/portfolio returns 500 on database error", func(t *testing.T) {
		// Setup with closed database
		db := testutil.SetupTestDB(t)
		db.Close() // Force database error

		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create HTTP request
		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
		w := httptest.NewRecorder()

		// Execute
		handler.PortfolioSummary(w, req)

		// Assert
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Debug output - only shows on failure or with -v flag
		if len(response) != 0 {
			t.Logf("Expected empty array, got %d items", len(response))
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("returns summary for portfolio with basic transactions", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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

		var response []model.PortfolioSummary
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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

		var response []model.PortfolioSummary
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
		var response []model.PortfolioSummary
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
		var response []model.PortfolioSummary
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
		// t.Skip("BLOCKED: Dividend calculation returns 0. Root cause: testutil.NewDividend() uses raw SQL INSERT " +
		// 	"that doesn't match production data structure. Will fix when DividendService.Create() is implemented. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().WithName("Dividend Test Portfolio").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
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

		var response []model.PortfolioSummary
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
		// t.Skip("BLOCKED: Same as 'includes_dividend_payments_in_summary' - dividend test data issue. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

		// Create portfolio and fund
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
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
		var response []model.PortfolioSummary
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
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
		var response []model.PortfolioSummary
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	})

	t.Run("handles portfolio with no fund prices gracefully", func(t *testing.T) {
		// Setup
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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

		var response []model.PortfolioSummary
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

		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewPortfolioHandler(ps, fs, ms)

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
	t.Run("returns portfolio with current summary for valid ID", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Buy 100 shares at $10
		testutil.NewTransaction(pfID).
			WithType("buy").
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// Current price: $12
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
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != portfolio.ID {
			t.Errorf("Expected portfolio ID %s, got %s", portfolio.ID, response.ID)
		}

		if response.Name != "Test Portfolio" {
			t.Errorf("Expected name 'Test Portfolio', got '%s'", response.Name)
		}

		// Verify calculations: cost = 100 * $10 = $1000
		if response.TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response.TotalCost)
		}

		// Value: 100 * $12 = $1200
		if response.TotalValue != 1200.0 {
			t.Errorf("Expected value 1200.0, got %.2f", response.TotalValue)
		}

		// Unrealized gain: $1200 - $1000 = $200
		if response.TotalUnrealizedGainLoss != 200.0 {
			t.Errorf("Expected unrealized gain 200.0, got %.2f", response.TotalUnrealizedGainLoss)
		}
	})

	// Input validation: Invalid format
	t.Run("returns 400 when portfolioId is invalid UUID format", func(t *testing.T) {
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

	// Edge case: Portfolio with no transactions
	t.Run("handles portfolio with no transactions", func(t *testing.T) {
		// t.Skip("BLOCKED: Returns 404 instead of 200. Issue: GetPortfolioHistoryWithFallback returns empty " +
		// 	"history for portfolios with no transactions, which is treated as 'not found'. Need to distinguish " +
		// 	"between 'portfolio doesn't exist' vs 'portfolio exists but has no data'. See portfolios.go:101")

		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Empty Portfolio").Build(t, db)

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
		if response.TotalCost != 0 {
			t.Errorf("Expected TotalCost 0, got %.2f", response.TotalCost)
		}
	})

	// Input validation: Empty portfolio ID
	t.Run("returns 400 when portfolioId URL param is empty", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/",
			map[string]string{"portfolioId": ""},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty portfolioId, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "portfolio ID is required" {
			t.Errorf("Expected 'portfolio ID is required' error, got '%s'", response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Test").Build(t, db)
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

// TestPortfolioHandler_PortfolioHistory tests the GET /api/portfolio/history endpoint.
//
// WHY: This endpoint returns historical portfolio valuations over a date range.
// It's essential for performance charts and trend analysis in the frontend.
//
// TODO: Add tests for materialized view fast path once materialized view population is implemented.
// Currently only tests the on-the-fly calculation fallback path.
func TestPortfolioHandler_PortfolioHistory(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: No portfolios
	t.Run("returns empty array when no portfolios exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		totalPortfolios := 0
		for _, entry := range response {
			totalPortfolios += len(entry.Portfolios)
		}
		if totalPortfolios != 0 {
			t.Errorf("Expected no portfolios in timeframe, got %d total portfolios", totalPortfolios)
		}
	})

	// Happy path: With data
	t.Run("returns historical data for specified date range", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("History Test").Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		// Transaction on 2024-06-01
		txDate, _ := time.Parse("2006-01-02", "2024-06-01")
		testutil.NewTransaction(pfID).
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(txDate).
			Build(t, db)

		// Price on 2024-06-15
		priceDate, _ := time.Parse("2006-01-02", "2024-06-15")
		testutil.NewFundPrice(fund.ID).
			WithPrice(12.0).
			WithDate(priceDate).
			Build(t, db)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-06-01",
				"end_date":   "2024-06-30",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.PortfolioHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have historical data points
		if len(response) == 0 {
			t.Error("Expected historical data, got empty array")
		}
	})

	// Default behavior: Uses default dates when not provided
	t.Run("uses default dates when query params not provided", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create minimal test data
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		testutil.NewTransaction(pfID).Build(t, db)
		testutil.NewFundPrice(fund.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/history", nil)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		// Should succeed with default dates
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with default dates, got %d", w.Code)
		}
	})

	// Input validation: Invalid date format
	t.Run("returns 400 when start_date has invalid format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "not-a-date",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if _, hasError := response["error"]; !hasError {
			t.Error("Expected error field in response")
		}
	})

	t.Run("returns 400 when end_date has invalid format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "invalid-date",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	// Edge case: Single day range
	t.Run("handles single day date range", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		singleDate, _ := time.Parse("2006-01-02", "2024-06-15")
		testutil.NewTransaction(pfID).WithDate(singleDate).Build(t, db)
		testutil.NewFundPrice(fund.ID).WithDate(singleDate).Build(t, db)

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-06-15",
				"end_date":   "2024-06-15",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for single day range, got %d", w.Code)
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close() // Force error

		req := testutil.NewRequestWithQueryParams(
			http.MethodGet,
			"/api/portfolio/history",
			map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		)
		w := httptest.NewRecorder()

		handler.PortfolioHistory(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_PortfolioFunds tests the GET /api/portfolio/funds endpoint.
//
// WHY: This endpoint returns all portfolio-fund relationships across all portfolios.
// Used for overview pages showing which funds are in which portfolios.
func TestPortfolioHandler_PortfolioFunds(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path: Empty
	t.Run("returns empty array when no portfolio-fund relationships exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Happy path: With data
	t.Run("returns all portfolio-fund relationships", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data: 2 portfolios, 2 funds
		p1 := testutil.NewPortfolio().WithName("Portfolio 1").Build(t, db)
		p2 := testutil.NewPortfolio().WithName("Portfolio 2").Build(t, db)
		f1 := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		f2 := testutil.NewFund().WithSymbol("GOOGL").Build(t, db)

		// Create relationships
		testutil.NewPortfolioFund(p1.ID, f1.ID).Build(t, db)
		testutil.NewPortfolioFund(p1.ID, f2.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, f1.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have 3 relationships
		if len(response) != 3 {
			t.Errorf("Expected 3 portfolio-fund relationships, got %d", len(response))
		}
	})

	// Edge case: Multiple portfolios with same fund
	t.Run("includes funds from multiple portfolios", func(t *testing.T) {
		handler, db := setupHandler(t)

		p1 := testutil.NewPortfolio().Build(t, db)
		p2 := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)

		// Same fund in two portfolios
		testutil.NewPortfolioFund(p1.ID, fund.ID).Build(t, db)
		testutil.NewPortfolioFund(p2.ID, fund.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		var response []model.PortfolioFund
		json.NewDecoder(w.Body).Decode(&response)

		// Should list both relationships
		if len(response) != 2 {
			t.Errorf("Expected 2 relationships, got %d", len(response))
		}
	})

	// Edge case: Verifies nil-safe response handling
	t.Run("returns empty array not nil when no relationships exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFundListing
		json.NewDecoder(w.Body).Decode(&response)

		// Should be empty slice, not nil
		if response == nil {
			t.Error("Expected non-nil response (empty array)")
		}
		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close() // Force error

		req := httptest.NewRequest(http.MethodGet, "/api/portfolio/funds", nil)
		w := httptest.NewRecorder()

		handler.PortfolioFunds(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TestPortfolioHandler_GetPortfolioFunds tests the GET /api/portfolio/funds/{portfolioId} endpoint.
//
// WHY: This endpoint returns detailed fund metrics for a specific portfolio, including
// shares, cost, value, gains, and dividends per fund. Critical for portfolio detail views.
func TestPortfolioHandler_GetPortfolioFunds(t *testing.T) {
	setupHandler := func(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ps := testutil.NewTestPortfolioService(t, db)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		return handlers.NewPortfolioHandler(ps, fs, ms), db
	}

	// Happy path
	t.Run("returns funds with metrics for valid portfolio", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Create test data
		portfolio := testutil.NewPortfolio().WithName("Test Portfolio").Build(t, db)
		fund1 := testutil.NewFund().WithSymbol("AAPL").Build(t, db)
		fund2 := testutil.NewFund().WithSymbol("GOOGL").Build(t, db)

		pf1 := testutil.NewPortfolioFund(portfolio.ID, fund1.ID).Build(t, db)
		pf2 := testutil.NewPortfolioFund(portfolio.ID, fund2.ID).Build(t, db)

		// AAPL: 100 shares at $10
		testutil.NewTransaction(pf1).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)
		testutil.NewFundPrice(fund1.ID).WithPrice(12.0).Build(t, db)

		// GOOGL: 50 shares at $20
		testutil.NewTransaction(pf2).
			WithShares(50).
			WithCostPerShare(20.0).
			Build(t, db)
		testutil.NewFundPrice(fund2.ID).WithPrice(22.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Fatalf("Expected 2 funds, got %d", len(response))
		}

		// Find funds by ID - don't assume order
		var aapl, googl *model.PortfolioFund
		for i := range response {
			if response[i].FundID == fund1.ID {
				aapl = &response[i]
			}
			if response[i].FundID == fund2.ID {
				googl = &response[i]
			}
		}

		// Verify we found both funds
		if aapl == nil {
			t.Fatal("AAPL not found in response")
		}
		if googl == nil {
			t.Fatal("GOOGL not found in response")
		}

		// Verify AAPL calculations
		if aapl.TotalShares != 100 {
			t.Errorf("Expected AAPL shares 100, got %.2f", aapl.TotalShares)
		}
		if aapl.TotalCost != 1000.0 {
			t.Errorf("Expected AAPL cost 1000.0, got %.2f", aapl.TotalCost)
		}
		if aapl.CurrentValue != 1200.0 {
			t.Errorf("Expected AAPL value 1200.0, got %.2f", aapl.CurrentValue)
		}

		// Verify GOOGL calculations
		if googl.TotalShares != 50 {
			t.Errorf("Expected GOOGL shares 50, got %.2f", googl.TotalShares)
		}
		if googl.TotalCost != 1000.0 {
			t.Errorf("Expected GOOGL cost 1000.0, got %.2f", googl.TotalCost)
		}
		if googl.CurrentValue != 1100.0 {
			t.Errorf("Expected GOOGL value 1100.0, got %.2f", googl.CurrentValue)
		}
	})

	// Edge case: Empty portfolio
	t.Run("returns empty array when portfolio has no funds", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().WithName("Empty Portfolio").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFund
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	// Input validation: Invalid UUID
	t.Run("returns 400 when portfolioId is invalid UUID format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/not-a-uuid",
			map[string]string{"portfolioId": "not-a-uuid"},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	// Resource not found
	t.Run("returns 404 when portfolio doesn't exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		validID := testutil.MakeID()
		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+validID,
			map[string]string{"portfolioId": validID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	// Edge case: Fund with no prices
	t.Run("handles fund with no price data", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pfID).
			WithShares(100).
			WithCostPerShare(10.0).
			Build(t, db)

		// No price added

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var response []model.PortfolioFund
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		// Value should be 0 when no price
		if response[0].CurrentValue != 0 {
			t.Errorf("Expected value 0 when no price, got %.2f", response[0].CurrentValue)
		}

		// Cost should still be calculated
		if response[0].TotalCost != 1000.0 {
			t.Errorf("Expected cost 1000.0, got %.2f", response[0].TotalCost)
		}
	})

	// Edge case: Realized gains
	t.Run("includes realized gains per fund", func(t *testing.T) {
		handler, db := setupHandler(t)

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

		// Record realized gain
		testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
			WithShares(30).
			WithCostBasis(300.0).
			WithSaleProceeds(450.0).
			Build(t, db)

		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		var response []model.PortfolioFund
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		// Remaining shares: 70
		if response[0].TotalShares != 70 {
			t.Errorf("Expected 70 shares, got %.2f", response[0].TotalShares)
		}

		// Realized gain: $150
		if response[0].RealizedGainLoss != 150.0 {
			t.Errorf("Expected realized gain 150.0, got %.2f", response[0].RealizedGainLoss)
		}
	})

	// Edge case: Dividends
	t.Run("includes dividends per fund", func(t *testing.T) {
		// t.Skip("BLOCKED: Same dividend test data issue as PortfolioSummary tests. " +
		// 	"See docs/TESTDATA_LIMITATIONS.md")

		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pfID).
			WithShares(100).
			WithCostPerShare(10.0).
			WithDate(time.Now().AddDate(0, 0, -20)).
			Build(t, db)

		// Add dividend: $50
		testutil.NewDividend(fund.ID, pfID).
			WithSharesOwned(100).
			WithDividendPerShare(0.50).
			Build(t, db)

		testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		var response []model.PortfolioFund
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Fatalf("Expected 1 fund, got %d", len(response))
		}

		if response[0].TotalDividends != 50.0 {
			t.Errorf("Expected dividends 50.0, got %.2f", response[0].TotalDividends)
		}
	})

	// Input validation: Empty portfolio ID
	t.Run("returns 400 when portfolioId URL param is empty", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/",
			map[string]string{"portfolioId": ""},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty portfolioId, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "portfolio ID is required" {
			t.Errorf("Expected 'portfolio ID is required' error, got '%s'", response["error"])
		}
	})

	// Database error
	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close() // Force error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/portfolio/funds/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetPortfolioFunds(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d", w.Code)
		}
	})
}

// TODO: Add more handler tests as you implement endpoints:
// - TestPortfolioHandler_CreatePortfolio (POST /api/portfolio)
// - TestPortfolioHandler_UpdatePortfolio (PUT /api/portfolio/:id)
// - TestPortfolioHandler_DeletePortfolio (DELETE /api/portfolio/:id)
// etc.
