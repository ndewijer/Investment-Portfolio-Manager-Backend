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
