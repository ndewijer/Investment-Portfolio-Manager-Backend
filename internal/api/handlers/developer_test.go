package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

//nolint:gocyclo // Test functions naturally have high complexity due to many test cases
func TestDeveloperHandler_GetLogs(t *testing.T) {
	setupHandler := func(t *testing.T) (*DeveloperHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDeveloperService(t, db)
		return NewDeveloperHandler(ds), db
	}

	t.Run("returns logs with default parameters", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Logs == nil {
			t.Error("Expected logs array to be initialized")
		}
	})

	t.Run("filters by level parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?level=error", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// If there are logs returned, verify they match the filter
		for _, log := range response.Logs {
			if log.Level != "ERROR" {
				t.Errorf("Expected level ERROR, got %s", log.Level)
			}
		}
	})

	t.Run("filters by multiple levels", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?level=error,critical", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify returned logs match one of the requested levels
		validLevels := map[string]bool{"ERROR": true, "CRITICAL": true}
		for _, log := range response.Logs {
			if !validLevels[log.Level] {
				t.Errorf("Expected level ERROR or CRITICAL, got %s", log.Level)
			}
		}
	})

	t.Run("filters by category parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?category=system", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("filters by source parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?source=Handler", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("filters by message parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?message=failed", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("filters by date range", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?startDate=2024-01-01&endDate=2024-12-31", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("respects perPage parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?perPage=5", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Logs) > 5 {
			t.Errorf("Expected at most 5 logs, got %d", len(response.Logs))
		}
	})

	t.Run("respects sortDir parameter", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?sortDir=asc", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("handles combined filters", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?level=error&category=system&sortDir=desc&perPage=10", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid level", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?level=invalid", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid category", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?category=invalid", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid sortDir", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?sortDir=invalid", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid perPage", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?perPage=0", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for perPage above maximum", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?perPage=101", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid date format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?startDate=invalid-date", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accepts case-insensitive level values", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?level=ERROR", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("accepts case-insensitive category values", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs?category=SYSTEM", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDeveloperHandler_GetLoggingConfig(t *testing.T) {

	setupHandler := func(t *testing.T) (*DeveloperHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDeveloperService(t, db)
		return NewDeveloperHandler(ds), db
	}

	t.Run("returns default logging config", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/system-settings/logging", nil)
		w := httptest.NewRecorder()

		handler.GetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LoggingSetting
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Enabled != true {
			t.Errorf("Expected logging to be enabled. value is: '%v'", response.Enabled)
		}

		if response.Level != "info" {
			t.Errorf("Expected logging level to be set to 'info', set to '%s'", response.Level)
		}
	})

	t.Run("returns set config.", func(t *testing.T) {
		handler, db := setupHandler(t)

		testutil.NewLoggingEnabled(false).Build(t, db)
		testutil.NewLoggingLevel("warning").Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/system-settings/logging", nil)
		w := httptest.NewRecorder()

		handler.GetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.LoggingSetting
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Enabled != false {
			t.Errorf("Expected logging to be disabled. value is: '%v'", response.Enabled)
		}

		if response.Level != "warning" {
			t.Errorf("Expected logging level to be set to 'warning', set to '%s'", response.Level)
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/developer/system-settings/logging", nil)
		w := httptest.NewRecorder()

		handler.GetLoggingConfig(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

}

func TestDeveloperHandler_GetFundPriceCSVTemplate(t *testing.T) {
	setupHandler := func(t *testing.T) (*DeveloperHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDeveloperService(t, db)
		return NewDeveloperHandler(ds), db
	}

	handler, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/developer/csv/fund-prices/template", nil)
	w := httptest.NewRecorder()

	handler.GetFundPriceCSVTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	type TemplateModel struct {
		Headers     []string          `json:"headers"`
		Example     map[string]string `json:"example"`
		Description string            `json:"description"`
	}

	var response TemplateModel
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Headers[0] != "date" {
		t.Errorf("Expected Header to be date. value is: '%v'", response.Headers[0])
	}
}

func TestDeveloperHandler_GetTransactionCSVTemplate(t *testing.T) {
	setupHandler := func(t *testing.T) (*DeveloperHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDeveloperService(t, db)
		return NewDeveloperHandler(ds), db
	}

	handler, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/developer/csv/transactions/template", nil)
	w := httptest.NewRecorder()

	handler.GetTransactionCSVTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	type TemplateModel struct {
		Headers     []string          `json:"headers"`
		Example     map[string]string `json:"example"`
		Description string            `json:"description"`
	}

	var response TemplateModel
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Headers[3] != "cost_per_share" {
		t.Errorf("Expected Header to be cost_per_share. value is: '%v'", response.Headers[3])
	}
}

//nolint:gocyclo // Test functions naturally have high complexity due to many test cases
func TestDeveloperHandler_GetExchangeRate(t *testing.T) {
	setupHandler := func(t *testing.T) (*DeveloperHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDeveloperService(t, db)
		return NewDeveloperHandler(ds), db
	}

	t.Run("returns 200 with rate when exchange rate exists", func(t *testing.T) {
		handler, db := setupHandler(t)

		// Insert test exchange rate
		testutil.NewExchangeRate("USD", "EUR", "2024-01-01", 0.85).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR&date=2024-01-01", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.FromCurrency != "USD" {
			t.Errorf("Expected fromCurrency to be USD, got %s", response.FromCurrency)
		}

		if response.ToCurrency != "EUR" {
			t.Errorf("Expected toCurrency to be EUR, got %s", response.ToCurrency)
		}

		if response.Date != "2024-01-01" {
			t.Errorf("Expected date to be 2024-01-01, got %s", response.Date)
		}

		if response.Rate == nil {
			t.Error("Expected rate to be set, got nil")
		} else if response.Rate.Rate != 0.85 {
			t.Errorf("Expected rate to be 0.85, got %f", response.Rate.Rate)
		}
	})

	t.Run("returns 200 with nil rate when exchange rate not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR&date=2024-01-01", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.FromCurrency != "USD" {
			t.Errorf("Expected fromCurrency to be USD, got %s", response.FromCurrency)
		}

		if response.ToCurrency != "EUR" {
			t.Errorf("Expected toCurrency to be EUR, got %s", response.ToCurrency)
		}

		if response.Date != "2024-01-01" {
			t.Errorf("Expected date to be 2024-01-01, got %s", response.Date)
		}

		if response.Rate != nil {
			t.Errorf("Expected rate to be nil when not found, got %v", response.Rate)
		}
	})

	t.Run("returns 400 when fromCurrency is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?toCurrency=EUR&date=2024-01-01", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when toCurrency is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&date=2024-01-01", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when date is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 when all parameters are missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("panics on invalid date format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR&date=invalid-date", nil)
		w := httptest.NewRecorder()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on invalid date format, but no panic occurred")
			}
		}()

		handler.GetExchangeRate(w, req)
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR&date=2024-01-01", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("handles different currency pairs", func(t *testing.T) {
		handler, db := setupHandler(t)

		testutil.NewExchangeRate("GBP", "USD", "2024-06-15", 1.27).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=GBP&toCurrency=USD&date=2024-06-15", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Rate == nil {
			t.Error("Expected rate to be set, got nil")
		} else if response.Rate.Rate != 1.27 {
			t.Errorf("Expected rate to be 1.27, got %f", response.Rate.Rate)
		}
	})

	t.Run("handles valid date formats", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/exchange-rate?fromCurrency=USD&toCurrency=EUR&date=2024-12-31", nil)
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Date != "2024-12-31" {
			t.Errorf("Expected date to be 2024-12-31, got %s", response.Date)
		}
	})
}
