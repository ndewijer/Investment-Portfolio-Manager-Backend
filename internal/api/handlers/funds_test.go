package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestFundHandler_GetAllFunds(t *testing.T) {
	t.Run("returns empty array when no funds exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		req := httptest.NewRequest(http.MethodGet, "/api/fund/", nil)
		w := httptest.NewRecorder()

		handler.GetAllFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		var response []model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d items", len(response))
		}
	})

	t.Run("returns all funds successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		f1 := testutil.NewFund().WithName("AAPL").Build(t, db)
		f2 := testutil.NewFund().WithName("GOOGL").Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/fund/", nil)
		w := httptest.NewRecorder()

		handler.GetAllFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 funds, got %d", len(response))
		}

		// Find funds by ID - don't assume order
		var fund1, fund2 *model.Fund
		for i := range response {
			if response[i].ID == f1.ID {
				fund1 = &response[i]
			}
			if response[i].ID == f2.ID {
				fund2 = &response[i]
			}
		}

		// Verify we found both
		if fund1 == nil {
			t.Fatal("Fund One not found in response")
		}
		if fund2 == nil {
			t.Fatal("Fund2 Two not found in response")
		}

		// Verify data matches what we created
		if fund1.Name != "AAPL" {
			t.Errorf("Expected first portfolio name 'AAPL', got '%s'", fund1.Name)
		}
		if fund2.Name != "GOOGL" {
			t.Errorf("Expected second portfolio name 'GOOGL', got '%s'", fund2.Name)
		}
	})

	t.Run("returns fund with all fields populated", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		f := testutil.NewFund().
			WithName("Apple").
			WithCurrency("USD").
			WithExchange("NSE").
			WithISIN("ISIN12345").
			WithSymbol("APPL").
			WithInvestementType("stock").
			WithDividendType("none").
			Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/fund", nil)
		w := httptest.NewRecorder()

		handler.GetAllFunds(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response []model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 1 {
			t.Fatalf("Expected 1 portfolio, got %d", len(response))
		}

		fund := response[0]

		// Verify all fields
		if fund.ID != f.ID {
			t.Errorf("ID mismatch: expected %s, got %s", f.ID, fund.ID)
		}
		if fund.Name != "Apple" {
			t.Errorf("Name mismatch: expected 'Apple', got '%s'", fund.Name)
		}
		if fund.Currency != "USD" {
			t.Errorf("Currency mismatch: expected 'USD', got '%s'", fund.Currency)
		}
		if fund.Exchange != "NSE" {
			t.Errorf("Exchange mismatch: expected 'NSE', got %s", fund.Exchange)
		}
		if fund.Isin != "ISIN12345" {
			t.Errorf("Isin mismatch: expected 'ISIN12345', got %s", fund.Isin)
		}
		if fund.Symbol != "APPL" {
			t.Errorf("Symbol mismatch: expected 'APPL', got %s", fund.Symbol)
		}
		if fund.InvestmentType != "stock" {
			t.Errorf("InvestmentType mismatch: expected 'APPL', got %s", fund.InvestmentType)
		}
		if fund.DividendType != "none" {
			t.Errorf("DividendType mismatch: expected 'APPL', got %s", fund.DividendType)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		db.Close() // Force database error

		req := httptest.NewRequest(http.MethodGet, "/api/fund/", nil)
		w := httptest.NewRecorder()

		handler.GetAllFunds(w, req)

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

func TestFundHandler_GetFund(t *testing.T) {
	t.Run("returns fund by UUID successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithName("AAPL").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFund(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Assert response body
		var response model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != fund.ID {
			t.Errorf("Expected fund ID %s, got %s", fund.ID, response.ID)
		}

		if response.Name != fund.Name {
			t.Errorf("Expected name '%s', got '%s'", fund.Name, response.Name)
		}
	})

	t.Run("returns 404 when fund not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/",
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.GetFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrFundNotFound.Error(), response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().WithName("AAPL").Build(t, db)

		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFund(w, req)

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

func TestFundHandler_GetSymbol(t *testing.T) {
	t.Run("returns symbol by ticker successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		symbol := testutil.NewSymbol().WithSymbol("AAPL").Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/symbol/"+symbol.Symbol,
			map[string]string{"symbol": symbol.Symbol},
		)
		w := httptest.NewRecorder()

		handler.GetSymbol(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response model.Symbol
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != symbol.ID {
			t.Errorf("Expected symbol ID %s, got %s", symbol.ID, response.ID)
		}

		if response.Name != symbol.Name {
			t.Errorf("Expected name '%s', got '%s'", symbol.Name, response.Name)
		}
	})

	t.Run("returns 400 when symbol is missing", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/symbol/",
			map[string]string{"symbol": ""},
		)
		w := httptest.NewRecorder()

		handler.GetSymbol(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrInvalidSymbol.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrInvalidSymbol.Error(), response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		symbol := testutil.NewSymbol().WithSymbol("AAPL").Build(t, db)

		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/symbol/"+symbol.Symbol,
			map[string]string{"symbol": symbol.Symbol},
		)
		w := httptest.NewRecorder()

		handler.GetSymbol(w, req)

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

func TestFundHandler_GetFundHistory(t *testing.T) {
	t.Run("returns fund history for portfolio", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/history/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundHistory(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.FundHistoryResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Empty portfolio should return empty array
		if len(response) != 0 {
			t.Errorf("Expected empty history, got %d entries", len(response))
		}
	})

	t.Run("returns 400 for invalid date parameters", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithQueryAndURLParams(
			http.MethodGet,
			"/api/fund/history/"+portfolio.ID+"?start_date=invalid-date",
			map[string]string{"uuid": portfolio.ID},
			map[string]string{"start_date": "invalid-date"},
		)
		w := httptest.NewRecorder()

		handler.GetFundHistory(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "Invalid date parameters" {
			t.Errorf("Expected 'Invalid date parameters' error, got '%s'", response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		portfolio := testutil.NewPortfolio().Build(t, db)

		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/history/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundHistory(w, req)

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

func TestFundHandler_GetFundPrices(t *testing.T) {
	t.Run("returns fund price history", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/fund-prices/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundPrices(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.FundPrice
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Fund without prices should return empty array
		if len(response) != 0 {
			t.Errorf("Expected empty price array, got %d prices", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/fund-prices/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.GetFundPrices(w, req)

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
func TestFundHandler_CheckUsage(t *testing.T) {
	t.Run("returns not in use when fund has no portfolio associations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/check-usage/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.CheckUsage(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.FundUsage
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.InUsage {
			t.Error("Expected fund to not be in use")
		}

		if len(response.Portfolios) != 0 {
			t.Errorf("Expected empty portfolios list, got %d entries", len(response.Portfolios))
		}
	})

	t.Run("returns in use when fund has portfolio associations", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)
		portfolio := testutil.NewPortfolio().Build(t, db)
		testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/check-usage/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.CheckUsage(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.FundUsage
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.InUsage {
			t.Error("Expected fund to be in use")
		}

		if len(response.Portfolios) != 1 {
			t.Errorf("Expected 1 portfolio entry, got %d", len(response.Portfolios))
		}

		if response.Portfolios[0].ID != portfolio.ID {
			t.Errorf("Expected portfolio ID %s, got %s", portfolio.ID, response.Portfolios[0].ID)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)
		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/fund/check-usage/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.CheckUsage(w, req)

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

func TestFundHandler_CreateFund(t *testing.T) {
	t.Run("creates fund successfully with valid data", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		payload := map[string]interface{}{
			"name":           "Apple Inc.",
			"isin":           "US0378331005",
			"symbol":         "AAPL",
			"currency":       "USD",
			"exchange":       "NASDAQ",
			"investmentType": "STOCK",
			"dividendType":   "CASH",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "Apple Inc." {
			t.Errorf("Expected name 'Apple Inc.', got '%s'", response.Name)
		}
		if response.Isin != "US0378331005" {
			t.Errorf("Expected ISIN 'US0378331005', got '%s'", response.Isin)
		}
		if response.Symbol != "AAPL" {
			t.Errorf("Expected symbol 'AAPL', got '%s'", response.Symbol)
		}
		if response.Currency != "USD" {
			t.Errorf("Expected currency 'USD', got '%s'", response.Currency)
		}
		if response.Exchange != "NASDAQ" {
			t.Errorf("Expected exchange 'NASDAQ', got '%s'", response.Exchange)
		}
		if response.InvestmentType != "STOCK" {
			t.Errorf("Expected investment type 'STOCK', got '%s'", response.InvestmentType)
		}
		if response.DividendType != "CASH" {
			t.Errorf("Expected dividend type 'CASH', got '%s'", response.DividendType)
		}

		// Verify ID was generated
		if response.ID == "" {
			t.Error("Expected ID to be generated")
		}
	})

	t.Run("returns 400 when name is missing", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		payload := map[string]interface{}{
			"isin":           "US0378331005",
			"currency":       "USD",
			"exchange":       "NASDAQ",
			"investmentType": "STOCK",
			"dividendType":   "CASH",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "validation failed" {
			t.Errorf("Expected validation error, got '%v'", response["error"])
		}
	})

	t.Run("returns 400 when ISIN format is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		payload := map[string]interface{}{
			"name":           "Apple Inc.",
			"isin":           "INVALID",
			"currency":       "USD",
			"exchange":       "NASDAQ",
			"investmentType": "STOCK",
			"dividendType":   "CASH",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when investment type is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		payload := map[string]interface{}{
			"name":           "Apple Inc.",
			"isin":           "US0378331005",
			"currency":       "USD",
			"exchange":       "NASDAQ",
			"investmentType": "INVALID",
			"dividend_type":  "CASH",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when dividend type is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		payload := map[string]interface{}{
			"name":            "Apple Inc.",
			"isin":            "US0378331005",
			"currency":        "USD",
			"exchange":        "NASDAQ",
			"investment_type": "STOCK",
			"dividendType":    "INVALID",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when JSON is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "invalid request body" {
			t.Errorf("Expected 'invalid request body' error, got '%s'", response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		db.Close() // Force database error

		payload := map[string]interface{}{
			"name":           "Apple Inc.",
			"isin":           "US0378331005",
			"currency":       "USD",
			"exchange":       "NASDAQ",
			"investmentType": "STOCK",
			"dividendType":   "CASH",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/fund", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateFund(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestFundHandler_UpdateFund(t *testing.T) {
	t.Run("updates fund successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().
			WithName("Old Name").
			WithSymbol("OLD").
			Build(t, db)

		newName := "New Name"
		newSymbol := "NEW"
		payload := map[string]interface{}{
			"name":   newName,
			"symbol": newSymbol,
		}

		body, _ := json.Marshal(payload)
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateFund(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.Fund
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != newName {
			t.Errorf("Expected name '%s', got '%s'", newName, response.Name)
		}
		if response.Symbol != newSymbol {
			t.Errorf("Expected symbol '%s', got '%s'", newSymbol, response.Symbol)
		}
	})

	t.Run("returns 404 when fund not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		nonExistentID := testutil.MakeID()
		payload := map[string]interface{}{
			"name": "New Name",
		}

		body, _ := json.Marshal(payload)
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/fund/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrFundNotFound.Error(), response["error"])
		}
	})

	t.Run("returns 400 when validation fails", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		payload := map[string]interface{}{
			"isin": "INVALID",
		}

		body, _ := json.Marshal(payload)
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 when JSON is invalid", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		req.Body = io.NopCloser(bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateFund(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != "invalid request body" {
			t.Errorf("Expected 'invalid request body' error, got '%s'", response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)
		db.Close() // Force database error

		payload := map[string]interface{}{
			"name": "New Name",
		}

		body, _ := json.Marshal(payload)
		req := testutil.NewRequestWithURLParams(
			http.MethodPut,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.UpdateFund(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestFundHandler_DeleteFund(t *testing.T) {
	t.Run("deletes fund successfully", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteFund(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d: %s", w.Code, w.Body.String())
		}

		// Verify fund was deleted
		_, err := fs.GetFund(fund.ID)
		if err == nil {
			t.Error("Expected fund to be deleted")
		}
		if err != apperrors.ErrFundNotFound {
			t.Errorf("Expected ErrFundNotFound, got %v", err)
		}
	})

	t.Run("returns 404 when fund not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/fund/"+nonExistentID,
			map[string]string{"uuid": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.DeleteFund(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)

		if response["error"] != apperrors.ErrFundNotFound.Error() {
			t.Errorf("Expected '%s' error, got '%s'", apperrors.ErrFundNotFound.Error(), response["error"])
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fs := testutil.NewTestFundService(t, db)
		ms := testutil.NewTestMaterializedService(t, db)
		handler := handlers.NewFundHandler(fs, ms)

		fund := testutil.NewFund().Build(t, db)
		db.Close() // Force database error

		req := testutil.NewRequestWithURLParams(
			http.MethodDelete,
			"/api/fund/"+fund.ID,
			map[string]string{"uuid": fund.ID},
		)
		w := httptest.NewRecorder()

		handler.DeleteFund(w, req)

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
