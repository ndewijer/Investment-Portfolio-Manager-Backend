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

func TestDividendHandler_GetAllDividends(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns empty array when no dividends exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Dividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d dividends", len(response))
		}
	})

	t.Run("returns all dividends successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		dividend1 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.00).
			WithSharesOwned(100).
			Build(t, db)
		dividend2 := testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(0.50).
			WithSharesOwned(100).
			Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.Dividend
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 dividends, got %d", len(response))
		}

		// Verify dividend IDs are present
		foundDividends := make(map[string]bool)
		for _, d := range response {
			foundDividends[d.ID] = true
		}

		if !foundDividends[dividend1.ID] {
			t.Error("Expected to find dividend1 in response")
		}
		if !foundDividends[dividend2.ID] {
			t.Error("Expected to find dividend2 in response")
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/dividend", nil)
		w := httptest.NewRecorder()

		handler.GetAllDividend(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDividendHandler_DividendPerPortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*DividendHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ds := testutil.NewTestDividendService(t, db)
		return NewDividendHandler(ds), db
	}

	t.Run("returns dividends for portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewDividend(fund.ID, pf.ID).
			WithDividendPerShare(1.00).
			WithSharesOwned(100).
			Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 1 {
			t.Errorf("Expected 1 dividend, got %d", len(response))
		}
	})

	t.Run("returns empty array when portfolio has no dividends", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.DividendFund
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d dividends", len(response))
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/dividend/portfolio/"+portfolio.ID,
			map[string]string{"uuid": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.DividendPerPortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}
