# Testing Framework Peer Review

**Date:** 2026-01-29
**Reviewer:** Claude Opus 4.5
**Branch:** Testing-Framework

---

## Overall Assessment

This is a **well-designed testing framework** that demonstrates thoughtful architecture and Go testing best practices. The foundation you've built is solid and will scale well as you add more endpoint coverage.

---

## Strengths

### 1. Builder Pattern Implementation (factories.go)

The fluent builder pattern is textbook Go testing:

```go
portfolio := testutil.NewPortfolio().
    WithName("Custom Portfolio").
    Archived().
    Build(t, db)
```

This approach is readable, self-documenting, and makes test data setup clear at a glance.

### 2. Database Setup (database.go)

- In-memory SQLite is the right choice for fast, isolated tests
- Schema creation is clean and matches production tables
- `t.Cleanup()` for automatic teardown is correct
- `CleanDatabase()` and `AssertRowCount()` helpers reduce boilerplate

### 3. HTTP Helpers (http_helpers.go)

The chi-specific URL parameter injection is a common pain point you've solved elegantly:

```go
req := testutil.NewRequestWithURLParams(
    http.MethodGet,
    "/api/portfolio/"+id,
    map[string]string{"portfolioId": id},
)
```

### 4. Test Organization (portfolios_test.go)

- **WHY comments** explaining the purpose of each test group are valuable for maintainability
- Tests cover the full spectrum: happy paths, edge cases, error conditions, database errors
- Subtests (`t.Run`) with descriptive names
- No assumptions about ordering (you search by ID instead of assuming array position)

### 5. Domain Errors Package (errors/errors.go)

Sentinel errors are the Go-idiomatic way to handle domain errors. This enables clean error checking in handlers:

```go
if errors.Is(err, apperrors.ErrPortfolioNotFound) {
    respondJSON(w, http.StatusNotFound, ...)
}
```

### 6. Makefile Targets

The coverage tooling is comprehensive: `coverage-gaps`, `coverage-by-file`, `coverage-handlers` are all useful for tracking progress.

---

## Areas for Improvement

### 1. Service Factory Duplication (helpers.go)

Each `NewTest*Service` function recreates the entire dependency tree. This leads to redundancy:

```go
// In NewTestMaterializedService, these are created again:
transactionService := service.NewTransactionService(repository.NewTransactionRepository(db))
dividendService := service.NewDividendService(repository.NewDividendRepository(db))
// ... same pattern repeated
```

**Recommendation:** Consider a `TestServiceContainer` that lazily initializes services once and shares them:

```go
type TestServices struct {
    db          *sql.DB
    portfolio   *service.PortfolioService
    fund        *service.FundService
    // ... cached instances
}

func NewTestServices(t *testing.T, db *sql.DB) *TestServices {
    return &TestServices{db: db}
}

func (s *TestServices) Portfolio() *service.PortfolioService {
    if s.portfolio == nil {
        s.portfolio = service.NewPortfolioService(repository.NewPortfolioRepository(s.db))
    }
    return s.portfolio
}
```

### 2. Handler Setup Boilerplate (portfolios_test.go)

Every test repeats the same setup:

```go
db := testutil.SetupTestDB(t)
ps := testutil.NewTestPortfolioService(t, db)
fs := testutil.NewTestFundService(t, db)
ms := testutil.NewTestMaterializedService(t, db)
handler := handlers.NewPortfolioHandler(ps, fs, ms)
```

You've started addressing this with `setupHandler()` local functions, but they're re-defined in each test group.

**Recommendation:** A shared test helper or test suite struct could reduce this:

```go
// In testutil/http_helpers.go
func NewTestPortfolioHandler(t *testing.T) (*handlers.PortfolioHandler, *sql.DB) {
    t.Helper()
    db := SetupTestDB(t)
    ps := NewTestPortfolioService(t, db)
    fs := NewTestFundService(t, db)
    ms := NewTestMaterializedService(t, db)
    return handlers.NewPortfolioHandler(ps, fs, ms), db
}
```

### 3. Deprecated rand.Seed (helpers.go:142)

```go
func init() {
    rand.Seed(time.Now().UnixNano())
}
```

`rand.Seed` is deprecated in Go 1.20+. The global random generator is now auto-seeded. Remove this entirely.

### 4. Missing Table-Driven Tests

Some tests could benefit from table-driven patterns, especially validation tests. For example, in `TestPortfolioHandler_GetPortfolio`:

```go
// Instead of separate subtests for each invalid input:
tests := []struct {
    name           string
    portfolioID    string
    expectedStatus int
    expectedError  string
}{
    {"empty ID", "", 400, "portfolio ID is required"},
    {"invalid UUID", "not-a-uuid", 400, "invalid UUID format"},
    {"non-existent", testutil.MakeID(), 404, ""},
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // ...
    })
}
```

### 5. shared_test.go Location

`shared_test.go` is in `package handlers` (internal test) while `portfolios_test.go` is in `package handlers_test` (external test). This is fine, but document why—it's because `respondJSON` is unexported.

### 6. Missing Test Isolation Verification

Consider adding a test that verifies test isolation:

```go
func TestDatabaseIsolation(t *testing.T) {
    t.Run("first creates data", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        testutil.CreatePortfolio(t, db, "Test")
        testutil.AssertRowCount(t, db, "portfolio", 1)
    })

    t.Run("second starts clean", func(t *testing.T) {
        db := testutil.SetupTestDB(t)
        testutil.AssertRowCount(t, db, "portfolio", 0)
    })
}
```

### 7. Consider Response Helpers

You repeatedly decode JSON responses. A helper could reduce noise:

```go
func DecodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
    t.Helper()
    var result T
    if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
        t.Fatalf("Failed to decode response: %v", err)
    }
    return result
}

// Usage:
response := testutil.DecodeJSON[[]model.Portfolio](t, w)
```

---

## Minor Observations

1. **Commented-out t.Skip blocks** - Good that you documented blockers with references to docs. Clean pattern.

2. **Constants for HTTP status codes** - Consider `http.StatusOK` over `200` for consistency (you use both).

3. **Missing test for Content-Type header** in most subtests - You check it in the first test, could be a shared assertion.

---

## Summary

| Aspect | Rating |
|--------|--------|
| Architecture | 5/5 |
| Code Quality | 4/5 |
| Test Coverage Pattern | 5/5 |
| Documentation | 4/5 |
| Maintainability | 4/5 |

This is a strong foundation. The patterns you've established (builders, helpers, WHY comments, edge case testing) will pay dividends as you expand coverage to other handlers. The main improvements are reducing boilerplate through shared setup and adding table-driven tests where appropriate.

---

## Files Reviewed

- `internal/testutil/database.go` - Test database setup
- `internal/testutil/factories.go` - Builder pattern implementations
- `internal/testutil/helpers.go` - Service factories and utilities
- `internal/testutil/http_helpers.go` - HTTP request helpers
- `internal/api/handlers/portfolios_test.go` - Handler tests
- `internal/api/handlers/shared_test.go` - Internal handler tests
- `internal/errors/errors.go` - Domain error definitions
- `Makefile` - Test and coverage targets
