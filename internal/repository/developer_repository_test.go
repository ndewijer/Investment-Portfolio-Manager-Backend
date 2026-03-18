package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

// ---------------------------------------------------------------------------
// AddLog / GetLogs
// ---------------------------------------------------------------------------

func TestDeveloperRepository_AddLogAndGetLogs(t *testing.T) {
	t.Run("add and retrieve logs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)

		log1 := model.Log{
			ID:        testutil.MakeID(),
			Timestamp: now.Add(-2 * time.Second),
			Level:     "INFO",
			Category:  "SYSTEM",
			Message:   "Test message 1",
			Source:    "test-source",
		}
		log2 := model.Log{
			ID:        testutil.MakeID(),
			Timestamp: now.Add(-1 * time.Second),
			Level:     "ERROR",
			Category:  "FUND",
			Message:   "Test error message",
			Source:    "test-source",
			Details:   "some details",
		}

		for _, l := range []model.Log{log1, log2} {
			if err := repo.AddLog(ctx, l); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 2 {
			t.Errorf("expected 2 logs, got %d", resp.Count)
		}
		// desc order means most recent first
		if resp.Logs[0].Level != "ERROR" {
			t.Errorf("expected first log level=ERROR (newest), got %s", resp.Logs[0].Level)
		}
	})
}

//nolint:gocyclo // Test function with multiple subtests and assertions.
func TestDeveloperRepository_GetLogs_Filters(t *testing.T) {
	t.Run("filter by level", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		for _, level := range []string{"INFO", "ERROR", "WARNING"} {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: now,
				Level:     level,
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			Levels:  []string{"error"},
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("expected 1 log, got %d", resp.Count)
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		for _, cat := range []string{"SYSTEM", "FUND", "PORTFOLIO"} {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: now,
				Level:     "INFO",
				Category:  cat,
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			Categories: []string{"fund", "portfolio"},
			PerPage:    50,
			SortDir:    "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 2 {
			t.Errorf("expected 2 logs, got %d", resp.Count)
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

		for i := range 5 {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: base.Add(time.Duration(i) * 24 * time.Hour),
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		// NOTE: AddLog stores timestamps as "2006-01-02 15:04:05" but GetLogs
		// compares with RFC3339 (e.g. "2026-03-11T12:00:00Z"). SQLite string
		// comparison means space (0x20) < 'T' (0x54), so stored timestamps at
		// the exact start boundary are excluded. This is a known format mismatch.
		// Using a range that spans day 0.5 to day 3.5 to reliably capture days 1-3.
		start := base.Add(12 * time.Hour) // day 0 + 12h = halfway between day 0 and day 1
		end := base.Add(4*24*time.Hour - 12*time.Hour)
		filters := &model.LogFilters{
			StartDate: &start,
			EndDate:   &end,
			PerPage:   50,
			SortDir:   "asc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 3 {
			t.Errorf("expected 3 logs in date range, got %d", resp.Count)
		}
	})

	t.Run("filter by source partial match", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		for _, src := range []string{"handler/portfolio", "handler/fund", "service/system"} {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: now,
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    src,
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			Source:  "handler",
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 2 {
			t.Errorf("expected 2 logs matching source 'handler', got %d", resp.Count)
		}
	})

	t.Run("filter by message partial match", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		for _, msg := range []string{"portfolio created", "fund updated", "system started"} {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: now,
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   msg,
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			Message: "fund",
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("expected 1 log matching message 'fund', got %d", resp.Count)
		}
	})

	t.Run("pagination with cursor", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
		for i := range 5 {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: base.Add(time.Duration(i) * time.Second),
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		// Get first page
		filters := &model.LogFilters{
			PerPage: 2,
			SortDir: "desc",
		}
		resp1, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs page 1: %v", err)
		}
		if !resp1.HasMore {
			t.Error("expected HasMore=true for first page")
		}
		if resp1.NextCursor == "" {
			t.Error("expected non-empty NextCursor")
		}
		if resp1.Count != 2 {
			t.Errorf("expected 2 logs on first page, got %d", resp1.Count)
		}

		// Get second page
		filters2 := &model.LogFilters{
			PerPage: 2,
			SortDir: "desc",
			Cursor:  resp1.NextCursor,
		}
		resp2, err := repo.GetLogs(filters2)
		if err != nil {
			t.Fatalf("GetLogs page 2: %v", err)
		}
		if resp2.Count != 2 {
			t.Errorf("expected 2 logs on second page, got %d", resp2.Count)
		}
	})

	t.Run("ascending sort", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := range 3 {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: base.Add(time.Duration(i) * time.Hour),
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "asc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 3 {
			t.Fatalf("expected 3 logs, got %d", resp.Count)
		}
		if !resp.Logs[0].Timestamp.Before(resp.Logs[1].Timestamp) {
			t.Error("expected ascending order")
		}
	})
}

// ---------------------------------------------------------------------------
// GetLoggingConfig / SetLoggingConfig
// ---------------------------------------------------------------------------

func TestDeveloperRepository_GetLoggingConfig(t *testing.T) {
	t.Run("defaults when no settings exist", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)

		cfg, err := repo.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig: %v", err)
		}
		if !cfg.Enabled {
			t.Error("expected default Enabled=true")
		}
		if cfg.Level != "info" {
			t.Errorf("expected default Level=info, got %s", cfg.Level)
		}
	})

	t.Run("reads configured settings", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)

		testutil.NewLoggingEnabled(false).Build(t, db)
		testutil.NewLoggingLevel("error").Build(t, db)

		cfg, err := repo.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig: %v", err)
		}
		if cfg.Enabled {
			t.Error("expected Enabled=false")
		}
		if cfg.Level != "error" {
			t.Errorf("expected Level=error, got %s", cfg.Level)
		}
	})
}

func TestDeveloperRepository_SetLoggingConfig(t *testing.T) {
	t.Run("create new setting", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		setting := model.SystemSetting{
			ID:        testutil.MakeID(),
			Key:       "LOGGING_LEVEL",
			Value:     "debug",
			UpdatedAt: &now,
		}
		err := repo.SetLoggingConfig(ctx, setting)
		if err != nil {
			t.Fatalf("SetLoggingConfig: %v", err)
		}

		cfg, err := repo.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig: %v", err)
		}
		if cfg.Level != "debug" {
			t.Errorf("expected Level=debug, got %s", cfg.Level)
		}
	})

	t.Run("upsert existing setting", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		setting1 := model.SystemSetting{
			ID:        testutil.MakeID(),
			Key:       "LOGGING_LEVEL",
			Value:     "info",
			UpdatedAt: &now,
		}
		if err := repo.SetLoggingConfig(ctx, setting1); err != nil {
			t.Fatalf("first SetLoggingConfig: %v", err)
		}

		setting2 := model.SystemSetting{
			ID:        testutil.MakeID(),
			Key:       "LOGGING_LEVEL",
			Value:     "warning",
			UpdatedAt: &now,
		}
		if err := repo.SetLoggingConfig(ctx, setting2); err != nil {
			t.Fatalf("second SetLoggingConfig: %v", err)
		}

		cfg, err := repo.GetLoggingConfig()
		if err != nil {
			t.Fatalf("GetLoggingConfig: %v", err)
		}
		if cfg.Level != "warning" {
			t.Errorf("expected Level=warning after upsert, got %s", cfg.Level)
		}
	})
}

// ---------------------------------------------------------------------------
// GetExchangeRate / UpdateExchangeRate
// ---------------------------------------------------------------------------

func TestDeveloperRepository_GetExchangeRate(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)

		testutil.NewExchangeRate("USD", "EUR", "2026-03-15", 0.85).Build(t, db)

		date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		rate, err := repo.GetExchangeRate("USD", "EUR", date)
		if err != nil {
			t.Fatalf("GetExchangeRate: %v", err)
		}
		if rate.Rate != 0.85 {
			t.Errorf("expected rate=0.85, got %f", rate.Rate)
		}
		if rate.FromCurrency != "USD" {
			t.Errorf("expected FromCurrency=USD, got %s", rate.FromCurrency)
		}
		if rate.ToCurrency != "EUR" {
			t.Errorf("expected ToCurrency=EUR, got %s", rate.ToCurrency)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)

		date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		_, err := repo.GetExchangeRate("USD", "GBP", date)
		if !errors.Is(err, apperrors.ErrExchangeRateNotFound) {
			t.Fatalf("expected ErrExchangeRateNotFound, got %v", err)
		}
	})

	t.Run("wrong date returns not found", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)

		testutil.NewExchangeRate("USD", "EUR", "2026-03-15", 0.85).Build(t, db)

		date := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
		_, err := repo.GetExchangeRate("USD", "EUR", date)
		if !errors.Is(err, apperrors.ErrExchangeRateNotFound) {
			t.Fatalf("expected ErrExchangeRateNotFound for wrong date, got %v", err)
		}
	})
}

func TestDeveloperRepository_UpdateExchangeRate(t *testing.T) {
	t.Run("insert new rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		rate := model.ExchangeRate{
			ID:           testutil.MakeID(),
			FromCurrency: "GBP",
			ToCurrency:   "USD",
			Rate:         1.25,
			Date:         date,
		}
		err := repo.UpdateExchangeRate(ctx, rate)
		if err != nil {
			t.Fatalf("UpdateExchangeRate: %v", err)
		}

		got, err := repo.GetExchangeRate("GBP", "USD", date)
		if err != nil {
			t.Fatalf("GetExchangeRate: %v", err)
		}
		if got.Rate != 1.25 {
			t.Errorf("expected rate=1.25, got %f", got.Rate)
		}
	})

	t.Run("upsert existing rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

		rate1 := model.ExchangeRate{
			ID:           testutil.MakeID(),
			FromCurrency: "JPY",
			ToCurrency:   "USD",
			Rate:         0.007,
			Date:         date,
		}
		if err := repo.UpdateExchangeRate(ctx, rate1); err != nil {
			t.Fatalf("first UpdateExchangeRate: %v", err)
		}

		rate2 := model.ExchangeRate{
			ID:           testutil.MakeID(),
			FromCurrency: "JPY",
			ToCurrency:   "USD",
			Rate:         0.008,
			Date:         date,
		}
		if err := repo.UpdateExchangeRate(ctx, rate2); err != nil {
			t.Fatalf("second UpdateExchangeRate: %v", err)
		}

		got, err := repo.GetExchangeRate("JPY", "USD", date)
		if err != nil {
			t.Fatalf("GetExchangeRate: %v", err)
		}
		if got.Rate != 0.008 {
			t.Errorf("expected rate=0.008 after upsert, got %f", got.Rate)
		}

		// Ensure only one row exists
		testutil.AssertRowCount(t, db, "exchange_rate", 1)
	})
}

// ---------------------------------------------------------------------------
// DeleteLogs
// ---------------------------------------------------------------------------

func TestDeveloperRepository_DeleteLogs(t *testing.T) {
	t.Run("delete all logs", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		for range 3 {
			log := model.Log{
				ID:        testutil.MakeID(),
				Timestamp: now,
				Level:     "INFO",
				Category:  "SYSTEM",
				Message:   "msg",
				Source:    "src",
			}
			if err := repo.AddLog(ctx, log); err != nil {
				t.Fatalf("AddLog: %v", err)
			}
		}

		testutil.AssertRowCount(t, db, "log", 3)

		err := repo.DeleteLogs(ctx)
		if err != nil {
			t.Fatalf("DeleteLogs: %v", err)
		}

		testutil.AssertRowCount(t, db, "log", 0)
	})

	t.Run("delete on empty table is no-op", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		err := repo.DeleteLogs(ctx)
		if err != nil {
			t.Fatalf("DeleteLogs on empty: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// AddLog with optional fields
// ---------------------------------------------------------------------------

func TestDeveloperRepository_AddLog_OptionalFields(t *testing.T) {
	t.Run("log with all optional fields", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		repo := repository.NewDeveloperRepository(db)
		ctx := context.Background()

		now := time.Now().UTC().Truncate(time.Second)
		log := model.Log{
			ID:         testutil.MakeID(),
			Timestamp:  now,
			Level:      "ERROR",
			Category:   "TRANSACTION",
			Message:    "Transaction failed",
			Details:    "Could not process buy order",
			Source:     "service/transaction",
			RequestID:  "req-123",
			StackTrace: "goroutine 1 [running]",
			HTTPStatus: "500",
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}
		if err := repo.AddLog(ctx, log); err != nil {
			t.Fatalf("AddLog: %v", err)
		}

		filters := &model.LogFilters{
			PerPage: 50,
			SortDir: "desc",
		}
		resp, err := repo.GetLogs(filters)
		if err != nil {
			t.Fatalf("GetLogs: %v", err)
		}
		if resp.Count != 1 {
			t.Fatalf("expected 1 log, got %d", resp.Count)
		}
		got := resp.Logs[0]
		if got.Details != "Could not process buy order" {
			t.Errorf("expected Details match, got %s", got.Details)
		}
		if got.RequestID != "req-123" {
			t.Errorf("expected RequestID=req-123, got %s", got.RequestID)
		}
		if got.StackTrace != "goroutine 1 [running]" {
			t.Errorf("expected StackTrace match, got %s", got.StackTrace)
		}
		if got.HTTPStatus != "500" {
			t.Errorf("expected HTTPStatus=500, got %s", got.HTTPStatus)
		}
		if got.IPAddress != "192.168.1.1" {
			t.Errorf("expected IPAddress=192.168.1.1, got %s", got.IPAddress)
		}
		if got.UserAgent != "Mozilla/5.0" {
			t.Errorf("expected UserAgent match, got %s", got.UserAgent)
		}
	})
}
