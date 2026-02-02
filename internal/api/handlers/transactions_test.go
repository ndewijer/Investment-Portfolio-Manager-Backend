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

func TestTransactionHandler_AllTransactions(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns empty array when no transactions exist", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d transactions", len(response))
		}
	})

	t.Run("returns all transactions successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx1 := testutil.NewTransaction(pf.ID).Build(t, db)
		tx2 := testutil.NewTransaction(pf.ID).Build(t, db)

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(response))
		}

		// Verify transaction IDs are present
		foundTransactions := make(map[string]bool)
		for _, tx := range response {
			foundTransactions[tx.ID] = true
		}

		if !foundTransactions[tx1.ID] {
			t.Error("Expected to find tx1 in response")
		}
		if !foundTransactions[tx2.ID] {
			t.Error("Expected to find tx2 in response")
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
		w := httptest.NewRecorder()

		handler.AllTransactions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_TransactionPerPortfolio(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns transactions for portfolio successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		testutil.NewTransaction(pf.ID).Build(t, db)
		testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if len(response) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(response))
		}
	})

	t.Run("returns empty array when portfolio has no transactions", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response []model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response == nil {
			t.Error("Expected non-nil array, got nil")
		}

		if len(response) != 0 {
			t.Errorf("Expected empty array, got %d transactions", len(response))
		}
	})

	t.Run("returns 400 when portfolio ID is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/",
			map[string]string{"portfolioId": ""},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/portfolio/"+portfolio.ID,
			map[string]string{"portfolioId": portfolio.ID},
		)
		w := httptest.NewRecorder()

		handler.TransactionPerPortfolio(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTransactionHandler_GetTransaction(t *testing.T) {
	setupHandler := func(t *testing.T) (*TransactionHandler, *sql.DB) {
		t.Helper()
		db := testutil.SetupTestDB(t)
		ts := testutil.NewTestTransactionService(t, db)
		return NewTransactionHandler(ts), db
	}

	t.Run("returns transaction by ID successfully", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+tx.ID,
			map[string]string{"transactionId": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response model.TransactionResponse
		//nolint:errcheck // Test assertion - decode failure would cause test to fail anyway
		json.NewDecoder(w.Body).Decode(&response)

		if response.ID != tx.ID {
			t.Errorf("Expected transaction ID %s, got %s", tx.ID, response.ID)
		}
	})

	t.Run("returns 400 when transaction ID is missing", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/",
			map[string]string{"transactionId": ""},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 for invalid UUID format", func(t *testing.T) {
		handler, _ := setupHandler(t)

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/invalid-uuid",
			map[string]string{"transactionId": "invalid-uuid"},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 404 when transaction not found", func(t *testing.T) {
		handler, _ := setupHandler(t)

		nonExistentID := testutil.MakeID()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+nonExistentID,
			map[string]string{"transactionId": nonExistentID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 500 on database error", func(t *testing.T) {
		handler, db := setupHandler(t)

		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		tx := testutil.NewTransaction(pf.ID).Build(t, db)

		db.Close()

		req := testutil.NewRequestWithURLParams(
			http.MethodGet,
			"/api/transaction/"+tx.ID,
			map[string]string{"transactionId": tx.ID},
		)
		w := httptest.NewRecorder()

		handler.GetTransaction(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}
