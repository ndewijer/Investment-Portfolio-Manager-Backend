package handlers_test

import (
	"encoding/json"
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

		var response []model.Portfolio
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

		var response []model.Portfolio
		err := json.NewDecoder(w.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 funds, got %d", len(response))
		}

		// Find funds by ID - don't assume order
		var fund1, fund2 *model.Portfolio
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
