# Go Best Practices & FAQ

This document addresses common questions and best practices for the Investment Portfolio Manager backend codebase.

---

## Table of Contents

1. [Context Support for Cancellation](#context-support-for-cancellation)
2. [Observability & Logging](#observability--logging)
3. [Prepared Statements](#prepared-statements)
4. [Input Validation Helpers](#input-validation-helpers)

---

## Context Support for Cancellation

### What is Context?

Context (`context.Context`) is a Go standard library package that carries deadlines, cancellation signals, and request-scoped values across API boundaries and between processes.

### When Should You Use Context?

**Use Context when**:
- Requests can take a long time (>1 second)
- You want users to be able to cancel ongoing operations
- You're making database queries that could hang
- You're calling external APIs
- You're processing large datasets

**NOT needed when**:
- Operations are guaranteed to be fast (<100ms)
- Operations are already cached (like materialized views)
- Internal calculations with no I/O

### Why Add Context Support?

#### Without Context:
```go
// User closes browser/cancels request
// Query keeps running for 30 seconds
// Database resources are wasted
// Connection pool gets exhausted

portfolioHistory, err := GetPortfolioHistory(startDate, endDate)
```

#### With Context:
```go
// User closes browser
// Context is cancelled
// Query stops immediately
// Resources are freed

ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()
portfolioHistory, err := GetPortfolioHistory(ctx, startDate, endDate)
```

### Example: Adding Context to a Query

**Before**:
```go
func (r *MaterializedRepository) GetMaterializedHistory(
    portfolioIDs []string,
    startDate, endDate time.Time,
    callback func(record model.PortfolioHistoryMaterialized) error,
) error {
    rows, err := r.db.Query(query, args...)
    if err != nil {
        return fmt.Errorf("failed to query: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        // Process rows
    }
    return nil
}
```

**After**:
```go
func (r *MaterializedRepository) GetMaterializedHistory(
    ctx context.Context,  // ✅ Add context parameter
    portfolioIDs []string,
    startDate, endDate time.Time,
    callback func(record model.PortfolioHistoryMaterialized) error,
) error {
    rows, err := r.db.QueryContext(ctx, query, args...)  // ✅ Use QueryContext
    if err != nil {
        return fmt.Errorf("failed to query: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        // Check if context was cancelled during iteration
        select {
        case <-ctx.Done():  // ✅ Respect cancellation
            return ctx.Err()
        default:
            // Process row
        }
    }
    return nil
}
```

### Should You Add Context to All Queries?

**Recommendation for this project**:

1. **Add to Long-Running Queries**:
   - `GetPortfolioHistory` (processes multiple portfolios, multiple dates)
   - `GetMaterializedHistory` (scans large date ranges)
   - `GetFundPrice` (retrieves historical price data)

2. **Lower Priority for Fast Queries**:
   - `GetPortfolioOnID` (single row lookup)
   - `GetFunds` (small table, likely cached by DB)
   - Materialized view queries (already optimized, ~3-10ms)

3. **Always Add to Handler Layer**:
```go
func (h *PortfolioHandler) PortfolioHistory(w http.ResponseWriter, r *http.Request) {
    // Extract context from HTTP request
    ctx := r.Context()

    // Pass context down the call chain
    history, err := h.portfolioService.GetPortfolioHistory(ctx, startDate, endDate)
    if err != nil {
        // Handle errors (including context.Canceled)
        if errors.Is(err, context.Canceled) {
            // Client cancelled, no need to respond
            return
        }
        respondJSON(w, http.StatusInternalServerError, errorResponse)
        return
    }
    respondJSON(w, http.StatusOK, history)
}
```

---

## Observability & Logging

### What is the Added Value?

Observability helps you understand:
- **Performance bottlenecks**: Which functions are slow?
- **Error patterns**: What's failing and why?
- **Usage patterns**: Which endpoints are called most?
- **Production debugging**: What happened when a user reported an issue?

### Why Not Just Add Logs Everywhere?

Good question! Logging has costs:
- **Performance**: I/O operations are slow
- **Storage**: Logs consume disk space
- **Noise**: Too many logs make it hard to find issues
- **Maintenance**: Logs need to be maintained/updated

### When to Add Logging

#### ✅ DO Log:
1. **External interactions**: API calls, database queries that fail
2. **State changes**: Creating transactions, updating balances
3. **Errors**: All errors with context
4. **Slow operations**: Operations >1 second

#### ❌ DON'T Log:
1. **Hot paths**: Functions called thousands of times per second
2. **Sensitive data**: Passwords, tokens, full portfolio values
3. **Successful operations**: "Successfully retrieved 10 portfolios" (noise)
4. **Internal calculations**: "Calculating total value..." (noise)

### Structured Logging Example

Instead of:
```go
fmt.Printf("ERROR during iteration: %v\n", err)  // ❌ Bad
```

Use structured logging:
```go
log.WithFields(log.Fields{
    "operation": "transaction_query",
    "portfolio_fund_ids": pfIDs,
    "error": err,
}).Error("failed to iterate transaction results")  // ✅ Good
```

### Should You Add Debug Flags?

**Yes, but strategically**. Here's a recommended approach:

```go
// In config/config.go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Logging  LoggingConfig  // ✅ Add this
}

type LoggingConfig struct {
    Level           string  // "debug", "info", "warn", "error"
    EnableTiming    bool    // Enable performance timing logs
    EnableQueryLog  bool    // Log all SQL queries (expensive!)
    EnableProfiling bool    // Enable pprof endpoints
}
```

**Usage in code**:
```go
func (s *FundService) calculateFundMetrics(...) (FundMetrics, error) {
    var start time.Time
    if s.config.Logging.EnableTiming {
        start = time.Now()
        defer func() {
            log.WithFields(log.Fields{
                "fund_id": fundID,
                "duration_ms": time.Since(start).Milliseconds(),
            }).Debug("calculateFundMetrics completed")
        }()
    }

    // Function logic...
}
```

### Recommendation for This Project

**Start simple, add as needed**:

1. **Now** (already in place):
   - Error logging with context (you have this ✅)
   - HTTP request logging (via middleware ✅)

2. **Next Sprint**:
   - Add structured logging library (logrus or zap)
   - Add timing logs behind a debug flag for slow operations:
     - `GetPortfolioHistory`
     - `calculateFundMetrics`
     - Materialized view generation

3. **Future** (when you have production issues):
   - Add correlation IDs to track requests across services
   - Add distributed tracing (OpenTelemetry)
   - Add metrics (Prometheus)

**Example Config**:
```yaml
# config.yaml
logging:
  level: "info"           # In production
  enable_timing: false    # In production
  enable_query_log: false # In production

# Development override
logging:
  level: "debug"
  enable_timing: true
  enable_query_log: true  # Warning: very verbose!
```

---

## Prepared Statements

### What Are Prepared Statements?

A prepared statement is a SQL query that's compiled once and executed many times with different parameters.

### How Do They Work?

#### Without Prepared Statements:
```go
// Every call parses and plans the query
for _, fundID := range fundIDs {
    rows, err := db.Query("SELECT * FROM fund WHERE id = ?", fundID)
    // Database: Parse → Plan → Execute (repeated 1000 times)
}
```

#### With Prepared Statements:
```go
// Parse and plan once
stmt, err := db.Prepare("SELECT * FROM fund WHERE id = ?")
defer stmt.Close()

// Execute many times
for _, fundID := range fundIDs {
    rows, err := stmt.Query(fundID)
    // Database: Execute only (1000 times)
}
```

### Performance Benefits

**When They Help**:
- Same query executed repeatedly in a loop
- Query is complex (joins, subqueries)
- Database is under high load

**Typical Gains**:
- Simple queries: 5-15% faster
- Complex queries: 20-40% faster
- High concurrency: Even better (reduced lock contention)

### When to Use Prepared Statements

#### ✅ Good Candidates:
```go
// Called in a loop
for _, portfolioID := range portfolios {
    GetPortfolioFundsOnPortfolioID(portfolioID)  // ✅ Prepare this
}

// Called frequently (hot path)
func GetFundPrice(fundID string) {  // ✅ Prepare this
    // SELECT * FROM fund_price WHERE fund_id = ?
}
```

#### ❌ Not Worth It:
```go
// Called once per HTTP request
func GetPortfolioHistory(startDate, endDate) {  // ❌ Don't prepare
    // Already doing bulk query with IN clause
}

// Dynamic query that changes
func GetFund(filter Filter) {  // ❌ Can't prepare
    query := "SELECT * FROM fund WHERE 1=1"
    if filter.Currency != "" {
        query += " AND currency = ?"
    }
    // Query structure changes, can't prepare
}
```

### Example Implementation

**Before** (fund_repository.go):
```go
type FundRepository struct {
    db *sql.DB
}

func (r *FundRepository) GetFunds(fundID string) (model.Fund, error) {
    query := "SELECT id, name, isin FROM fund WHERE id = ?"
    row := r.db.QueryRow(query, fundID)
    // Parsed every time
}
```

**After** (with prepared statements):
```go
type FundRepository struct {
    db                *sql.DB
    getFundStmt       *sql.Stmt  // ✅ Add prepared statements
    getFundPriceStmt  *sql.Stmt
}

func NewFundRepository(db *sql.DB) (*FundRepository, error) {
    repo := &FundRepository{db: db}

    // Prepare frequently-used queries
    var err error
    repo.getFundStmt, err = db.Prepare(
        "SELECT id, name, isin, symbol, currency, exchange, investment_type, dividend_type FROM fund WHERE id = ?",
    )
    if err != nil {
        return nil, fmt.Errorf("failed to prepare getFund: %w", err)
    }

    repo.getFundPriceStmt, err = db.Prepare(
        "SELECT id, fund_id, date, price FROM fund_price WHERE fund_id = ? AND date >= ? AND date <= ? ORDER BY date ASC",
    )
    if err != nil {
        repo.getFundStmt.Close()  // Cleanup previous
        return nil, fmt.Errorf("failed to prepare getFundPrice: %w", err)
    }

    return repo, nil
}

func (r *FundRepository) Close() error {
    // Remember to close statements
    r.getFundStmt.Close()
    r.getFundPriceStmt.Close()
    return nil
}

func (r *FundRepository) GetFunds(fundID string) (model.Fund, error) {
    // Use prepared statement
    row := r.getFundStmt.QueryRow(fundID)  // ✅ Already prepared
    // ... scan logic
}
```

### Recommendation for This Project

**Start with measurement**:

1. **Profile first**: Don't optimize blindly
   ```bash
   # Add timing to identify slow queries
   go test -bench=. -cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

2. **Prepare selectively**: Based on profiling results
   - If `GetFund` shows up in top 10: prepare it
   - If it doesn't: don't bother yet

3. **Consider IN clause optimization first**:
   Your current code already uses bulk queries:
   ```go
   // This is already efficient - loads 100 funds in 1 query
   WHERE fund_id IN (?, ?, ?, ...)
   ```

   Preparing this won't help much because:
   - Query structure changes (different number of IDs)
   - SQLite/Most DBs optimize IN clauses well already

4. **Best candidates in your codebase**:
   - `GetOldestTransaction` - called once per portfolio, same query
   - Simple lookups in tight loops (if you add them later)

### Trade-offs

**Pros**:
- Faster execution for repeated queries
- Protection against SQL injection (you already have this via parameterization)

**Cons**:
- More complex code (need to manage statement lifecycle)
- Memory overhead (statements stay in memory)
- Connection pool implications (statements tied to connections)

**Verdict**: Optimize this later based on production metrics. Your current bulk queries are already efficient.

---

## Input Validation Helpers

### Why Validation Helpers?

**Without helpers** (current state):
```go
// Duplicated validation logic
func GetFundPrice(fundIDs []string, startDate, endDate time.Time) error {
    if startDate.After(endDate) {
        return fmt.Errorf("startDate must be before endDate")
    }
    // ...
}

func GetTransactions(pfIDs []string, startDate, endDate time.Time) error {
    if startDate.After(endDate) {
        return fmt.Errorf("startDate must be before endDate")
    }
    // ...
}
```

**With helpers**:
```go
func GetFundPrice(fundIDs []string, startDate, endDate time.Time) error {
    if err := ValidateDateRange(startDate, endDate); err != nil {
        return err
    }
    // ...
}
```

### Suggested Validation Helpers

Create `internal/validation/validation.go`:

```go
package validation

import (
    "fmt"
    "time"

    "github.com/google/uuid"
)

// Common validation errors
var (
    ErrInvalidUUID      = fmt.Errorf("invalid UUID format")
    ErrInvalidDateRange = fmt.Errorf("invalid date range")
    ErrEmptySlice       = fmt.Errorf("slice cannot be empty")
)

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(id string) error {
    if _, err := uuid.Parse(id); err != nil {
        return fmt.Errorf("%w: %s", ErrInvalidUUID, id)
    }
    return nil
}

// ValidateUUIDs validates a slice of UUIDs
func ValidateUUIDs(ids []string) error {
    if len(ids) == 0 {
        return ErrEmptySlice
    }
    for _, id := range ids {
        if err := ValidateUUID(id); err != nil {
            return err
        }
    }
    return nil
}

// ValidateDateRange ensures start date is before or equal to end date
func ValidateDateRange(startDate, endDate time.Time) error {
    if startDate.IsZero() {
        return fmt.Errorf("start date is zero value")
    }
    if endDate.IsZero() {
        return fmt.Errorf("end date is zero value")
    }
    if startDate.After(endDate) {
        return fmt.Errorf("%w: start (%s) is after end (%s)",
            ErrInvalidDateRange,
            startDate.Format("2006-01-02"),
            endDate.Format("2006-01-02"))
    }
    return nil
}

// ValidateDateRangeOptional allows zero values (for optional date parameters)
func ValidateDateRangeOptional(startDate, endDate time.Time) error {
    // Allow zero values
    if startDate.IsZero() || endDate.IsZero() {
        return nil
    }
    return ValidateDateRange(startDate, endDate)
}

// ValidatePortfolioID validates a portfolio ID
func ValidatePortfolioID(portfolioID string) error {
    if portfolioID == "" {
        return fmt.Errorf("portfolio ID cannot be empty")
    }
    return ValidateUUID(portfolioID)
}

// ValidatePortfolioIDOptional allows empty string (for "all portfolios")
func ValidatePortfolioIDOptional(portfolioID string) error {
    if portfolioID == "" {
        return nil  // Empty means "all portfolios"
    }
    return ValidateUUID(portfolioID)
}

// ValidateSortOrder checks if sort order is valid
func ValidateSortOrder(sortOrder string) error {
    switch strings.ToUpper(sortOrder) {
    case "ASC", "DESC":
        return nil
    default:
        return fmt.Errorf("invalid sort order: %s (must be ASC or DESC)", sortOrder)
    }
}

// ValidatePageSize ensures pagination size is reasonable
func ValidatePageSize(size int) error {
    if size < 1 {
        return fmt.Errorf("page size must be at least 1")
    }
    if size > 1000 {
        return fmt.Errorf("page size cannot exceed 1000")
    }
    return nil
}
```

### Usage Examples

#### In Repositories:
```go
func (r *FundRepository) GetFundPrice(fundIDs []string, startDate, endDate time.Time, sortOrder string) (map[string][]model.FundPrice, error) {
    // Validate inputs
    if err := validation.ValidateUUIDs(fundIDs); err != nil {
        return nil, err
    }
    if err := validation.ValidateDateRange(startDate, endDate); err != nil {
        return nil, err
    }
    if err := validation.ValidateSortOrder(sortOrder); err != nil {
        return nil, err
    }

    // Continue with query...
}
```

#### In Handlers:
```go
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
    portfolioID := chi.URLParam(r, "portfolioId")

    // Validate input
    if err := validation.ValidatePortfolioID(portfolioID); err != nil {
        respondJSON(w, http.StatusBadRequest, map[string]string{
            "error":  "invalid portfolio ID",
            "detail": err.Error(),
        })
        return
    }

    // Continue with business logic...
}
```

### Advanced: Struct Validation

For more complex validation, consider using a validation library:

```go
import "github.com/go-playground/validator/v10"

type PortfolioRequest struct {
    StartDate string `json:"start_date" validate:"required,datetime=2006-01-02"`
    EndDate   string `json:"end_date" validate:"required,datetime=2006-01-02"`
    Limit     int    `json:"limit" validate:"min=1,max=1000"`
}

var validate = validator.New()

func (h *PortfolioHandler) PortfolioHistory(w http.ResponseWriter, r *http.Request) {
    var req PortfolioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondJSON(w, http.StatusBadRequest, errorResponse)
        return
    }

    // Validate with tags
    if err := validate.Struct(req); err != nil {
        respondJSON(w, http.StatusBadRequest, map[string]string{
            "error":  "validation failed",
            "detail": err.Error(),
        })
        return
    }

    // Continue...
}
```

### Recommendation for This Project

**Start simple, grow as needed**:

1. **Phase 1** (Do now):
   - Create `internal/validation/validation.go`
   - Add: `ValidateUUID`, `ValidateDateRange`, `ValidateSortOrder`
   - Use in repositories that accept these parameters

2. **Phase 2** (Later):
   - Add handler-level validation
   - Add custom validators for business rules
   - Consider struct validation library if API grows

3. **Phase 3** (Much later):
   - Generate OpenAPI spec from validation tags
   - Auto-generate validation tests
   - Add validation middleware

### Where to Apply Validation?

**Layer Decision**:

| Layer      | What to Validate                           | Why                                    |
|------------|--------------------------------------------|----------------------------------------|
| Handler    | Request format, auth, rate limits          | Fail fast, clear error messages        |
| Service    | Business rules (e.g., can't sell more than owned) | Domain logic belongs here              |
| Repository | Data integrity (UUIDs, date ranges)        | Protect database from invalid queries  |

**Example**:
```go
// Handler: Check format
if portfolioID == "" {
    return http.StatusBadRequest
}

// Service: Check business rule
if sellShares > ownedShares {
    return ErrInsufficientShares
}

// Repository: Check data integrity
if err := validation.ValidateUUID(portfolioID); err != nil {
    return err
}
```

---

## Summary & Recommendations

### Priority Order:

1. **✅ Do Now**:
   - Create validation helpers for UUID and date ranges
   - Apply validation in repository methods

2. **📋 Next Sprint**:
   - Add structured logging library
   - Add timing logs behind debug flag
   - Profile to identify slow queries

3. **🔮 Future**:
   - Add context support to long-running queries
   - Consider prepared statements based on profiling
   - Add distributed tracing when you have multiple services

### Key Principles:

1. **Measure before optimizing**: Profile, benchmark, then optimize
2. **Start simple**: Add complexity only when needed
3. **User-facing first**: Optimize what users notice (response times)
4. **Developer experience**: Make debugging easier with good logging

---

**Document Version**: 1.0
**Last Updated**: 2026-01-16
**Maintained By**: Engineering Team
