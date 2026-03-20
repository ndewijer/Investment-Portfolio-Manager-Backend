# Write Operations Guide - POST/PUT/DELETE Patterns

A comprehensive guide for implementing write operations (POST/PUT/DELETE) in the Go backend.

---

## Table of Contents

1. [Overview - Why Writes Are Different](#overview---why-writes-are-different)
2. [Request Handling Pattern](#request-handling-pattern)
3. [Validation Patterns](#validation-patterns)
4. [Transaction Management](#transaction-management)
5. [Repository Write Methods](#repository-write-methods)
6. [Service Layer Patterns](#service-layer-patterns)
7. [Handler Implementation](#handler-implementation)
8. [Error Response Patterns](#error-response-patterns)
9. [Complete Example: POST /api/portfolio](#complete-example-post-apiportfolio)
10. [Recommended Implementation Order](#recommended-implementation-order)

---

## Overview - Why Writes Are Different

### Read Operations (What You've Built)
```
HTTP GET → Handler → Service → Repository → DB
                                    ↓
HTTP 200 ← Handler ← Service ← Repository ← rows
```
- Single database query
- No side effects
- Easy to retry
- Idempotent

### Write Operations (What You're Adding)
```
HTTP POST → Handler → Validate → Service → Transaction Begin
                                     ↓
                              Repository Insert (1)
                                     ↓
                              Repository Insert (2)
                                     ↓
HTTP 201 ← Handler ← Service ← Transaction Commit
```
- Request body parsing
- Input validation
- Database transactions
- Potential for partial failure
- Not idempotent (POST)
- Side effects on other records

---

## Request Handling Pattern

### 1. Define Request DTOs

Create a new file `internal/api/request/portfolio.go`:

```go
package request

// CreatePortfolioRequest represents the request body for creating a portfolio
type CreatePortfolioRequest struct {
    Name               string `json:"name"`
    Description        string `json:"description"`
    IsArchived         bool   `json:"is_archived"`
    ExcludeFromOverview bool  `json:"exclude_from_overview"`
}

// UpdatePortfolioRequest represents the request body for updating a portfolio
type UpdatePortfolioRequest struct {
    Name               *string `json:"name,omitempty"`               // Pointer for optional
    Description        *string `json:"description,omitempty"`
    IsArchived         *bool   `json:"is_archived,omitempty"`
    ExcludeFromOverview *bool  `json:"exclude_from_overview,omitempty"`
}
```

**Why pointers for Update?**
- Distinguishes between "not provided" (nil) and "set to empty" ("")
- Allows partial updates without overwriting unmentioned fields

### 2. Parse Request Body

```go
func parseJSON[T any](r *http.Request) (T, error) {
    var req T

    // Limit body size (prevent memory exhaustion attacks)
    r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB max

    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields() // Strict parsing

    if err := decoder.Decode(&req); err != nil {
        return req, fmt.Errorf("invalid JSON: %w", err)
    }

    return req, nil
}
```

### 3. Handler Pattern

```go
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    req, err := parseJSON[request.CreatePortfolioRequest](r)
    if err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
        return
    }

    // 2. Validate
    if err := validation.ValidateCreatePortfolio(req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Validation failed", err.Error())
        return
    }

    // 3. Call service
    portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), req)
    if err != nil {
        api.RespondError(w, http.StatusInternalServerError, "Failed to create portfolio", err.Error())
        return
    }

    // 4. Return created resource
    api.RespondJSON(w, http.StatusCreated, portfolio)
}
```

---

## Validation Patterns

### Create Validation File

`internal/validation/portfolio.go`:

```go
package validation

import (
    "fmt"
    "strings"

    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/api/request"
)

// ValidationError holds multiple field errors
type ValidationError struct {
    Fields map[string]string
}

func (e *ValidationError) Error() string {
    var msgs []string
    for field, msg := range e.Fields {
        msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
    }
    return strings.Join(msgs, "; ")
}

func ValidateCreatePortfolio(req request.CreatePortfolioRequest) error {
    errors := make(map[string]string)

    // Required field
    if strings.TrimSpace(req.Name) == "" {
        errors["name"] = "name is required"
    } else if len(req.Name) > 100 {
        errors["name"] = "name must be 100 characters or less"
    }

    // Optional but has constraints
    if len(req.Description) > 500 {
        errors["description"] = "description must be 500 characters or less"
    }

    if len(errors) > 0 {
        return &ValidationError{Fields: errors}
    }
    return nil
}

func ValidateUpdatePortfolio(req request.UpdatePortfolioRequest) error {
    errors := make(map[string]string)

    // Only validate provided fields
    if req.Name != nil {
        if strings.TrimSpace(*req.Name) == "" {
            errors["name"] = "name cannot be empty"
        } else if len(*req.Name) > 100 {
            errors["name"] = "name must be 100 characters or less"
        }
    }

    if req.Description != nil && len(*req.Description) > 500 {
        errors["description"] = "description must be 500 characters or less"
    }

    if len(errors) > 0 {
        return &ValidationError{Fields: errors}
    }
    return nil
}
```

### Use Existing Validation

Extend your existing `validation/validation.go`:

```go
// ValidatePortfolioID validates and returns a portfolio ID from URL params
func ValidatePortfolioID(id string) (string, error) {
    if err := ValidateUUID(id); err != nil {
        return "", fmt.Errorf("invalid portfolio ID: %w", err)
    }
    return id, nil
}
```

---

## Transaction Management

### Why Transactions Matter

Consider creating a portfolio with funds:

```go
// Without transaction - DANGEROUS
portfolio, err := repo.InsertPortfolio(...)     // Success
portfolioFund, err := repo.InsertPortfolioFund(...) // Fails!
// Now you have orphaned portfolio with no funds
```

```go
// With transaction - SAFE
tx, _ := db.Begin()
portfolio, err := repo.InsertPortfolioTx(tx, ...) // Success
portfolioFund, err := repo.InsertPortfolioFundTx(tx, ...) // Fails!
tx.Rollback() // Both changes are reverted
```

### Transaction Pattern 1: Repository-Level

Add to your repository:

```go
// PortfolioRepository with transaction support
type PortfolioRepository struct {
    db *sql.DB
    tx *sql.Tx // Optional transaction
}

// WithTx returns a new repository that uses the given transaction
func (r *PortfolioRepository) WithTx(tx *sql.Tx) *PortfolioRepository {
    return &PortfolioRepository{
        db: r.db,
        tx: tx,
    }
}

// getQuerier returns either the transaction or the db
func (r *PortfolioRepository) getQuerier() interface {
    Query(query string, args ...any) (*sql.Rows, error)
    QueryRow(query string, args ...any) *sql.Row
    Exec(query string, args ...any) (sql.Result, error)
} {
    if r.tx != nil {
        return r.tx
    }
    return r.db
}

// Use getQuerier() in all methods
func (r *PortfolioRepository) Insert(ctx context.Context, p *model.Portfolio) error {
    q := r.getQuerier()
    _, err := q.Exec(`INSERT INTO portfolio ...`, p.ID, p.Name, ...)
    return err
}
```

### Transaction Pattern 2: Service-Level (Recommended)

Create a transaction manager:

```go
// internal/database/transaction.go
package database

import (
    "context"
    "database/sql"
    "fmt"
)

// TxFunc is a function that runs inside a transaction
type TxFunc func(tx *sql.Tx) error

// WithTransaction executes fn inside a transaction
func WithTransaction(ctx context.Context, db *sql.DB, fn TxFunc) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p) // Re-throw panic after rollback
        }
    }()

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx failed: %v, rollback failed: %w", err, rbErr)
        }
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

Usage in service:

```go
func (s *PortfolioService) CreatePortfolioWithFund(ctx context.Context, portfolioReq, fundReq) error {
    return database.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
        // All operations use the same transaction
        portfolioRepo := s.portfolioRepo.WithTx(tx)
        portfolioFundRepo := s.portfolioFundRepo.WithTx(tx)

        portfolio := &model.Portfolio{ID: uuid.New().String(), ...}
        if err := portfolioRepo.Insert(ctx, portfolio); err != nil {
            return err // Transaction will be rolled back
        }

        portfolioFund := &model.PortfolioFund{PortfolioID: portfolio.ID, ...}
        if err := portfolioFundRepo.Insert(ctx, portfolioFund); err != nil {
            return err // Transaction will be rolled back
        }

        return nil // Transaction will be committed
    })
}
```

---

## Repository Write Methods

### Insert Pattern

```go
func (r *PortfolioRepository) Insert(ctx context.Context, p *model.Portfolio) error {
    query := `
        INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
        VALUES (?, ?, ?, ?, ?)
    `

    _, err := r.getQuerier().ExecContext(ctx, query,
        p.ID,
        p.Name,
        p.Description,
        p.IsArchived,
        p.ExcludeFromOverview,
    )

    if err != nil {
        return fmt.Errorf("failed to insert portfolio: %w", err)
    }

    return nil
}
```

### Update Pattern

```go
func (r *PortfolioRepository) Update(ctx context.Context, p *model.Portfolio) error {
    query := `
        UPDATE portfolio
        SET name = ?, description = ?, is_archived = ?, exclude_from_overview = ?
        WHERE id = ?
    `

    result, err := r.getQuerier().ExecContext(ctx, query,
        p.Name,
        p.Description,
        p.IsArchived,
        p.ExcludeFromOverview,
        p.ID,
    )

    if err != nil {
        return fmt.Errorf("failed to update portfolio: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound // Define this error
    }

    return nil
}
```

### Delete Pattern

```go
func (r *PortfolioRepository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM portfolio WHERE id = ?`

    result, err := r.getQuerier().ExecContext(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to delete portfolio: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrNotFound
    }

    return nil
}
```

### Upsert Pattern (Insert or Update)

```go
func (r *PortfolioRepository) Upsert(ctx context.Context, p *model.Portfolio) error {
    query := `
        INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name = excluded.name,
            description = excluded.description,
            is_archived = excluded.is_archived,
            exclude_from_overview = excluded.exclude_from_overview
    `

    _, err := r.getQuerier().ExecContext(ctx, query,
        p.ID, p.Name, p.Description, p.IsArchived, p.ExcludeFromOverview,
    )

    if err != nil {
        return fmt.Errorf("failed to upsert portfolio: %w", err)
    }

    return nil
}
```

---

## Service Layer Patterns

### Create Method

```go
func (s *PortfolioService) CreatePortfolio(
    ctx context.Context,
    req request.CreatePortfolioRequest,
) (*model.Portfolio, error) {
    // Generate ID
    portfolio := &model.Portfolio{
        ID:                  uuid.New().String(),
        Name:                req.Name,
        Description:         req.Description,
        IsArchived:          req.IsArchived,
        ExcludeFromOverview: req.ExcludeFromOverview,
    }

    // Insert
    if err := s.portfolioRepo.Insert(ctx, portfolio); err != nil {
        return nil, fmt.Errorf("failed to create portfolio: %w", err)
    }

    // Log (once logging is implemented)
    // s.logger.Info("portfolio created", "id", portfolio.ID)

    return portfolio, nil
}
```

### Update Method

```go
func (s *PortfolioService) UpdatePortfolio(
    ctx context.Context,
    id string,
    req request.UpdatePortfolioRequest,
) (*model.Portfolio, error) {
    // Fetch existing
    portfolio, err := s.portfolioRepo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Apply updates (only non-nil fields)
    if req.Name != nil {
        portfolio.Name = *req.Name
    }
    if req.Description != nil {
        portfolio.Description = *req.Description
    }
    if req.IsArchived != nil {
        portfolio.IsArchived = *req.IsArchived
    }
    if req.ExcludeFromOverview != nil {
        portfolio.ExcludeFromOverview = *req.ExcludeFromOverview
    }

    // Save
    if err := s.portfolioRepo.Update(ctx, portfolio); err != nil {
        return nil, fmt.Errorf("failed to update portfolio: %w", err)
    }

    return portfolio, nil
}
```

### Delete Method with Cascade

```go
func (s *PortfolioService) DeletePortfolio(ctx context.Context, id string) error {
    return database.WithTransaction(ctx, s.db, func(tx *sql.Tx) error {
        portfolioRepo := s.portfolioRepo.WithTx(tx)
        portfolioFundRepo := s.portfolioFundRepo.WithTx(tx)

        // Check if portfolio exists
        _, err := portfolioRepo.GetByID(ctx, id)
        if err != nil {
            return err
        }

        // Delete related portfolio_funds first (foreign key constraint)
        if err := portfolioFundRepo.DeleteByPortfolioID(ctx, id); err != nil {
            return fmt.Errorf("failed to delete portfolio funds: %w", err)
        }

        // Delete portfolio
        if err := portfolioRepo.Delete(ctx, id); err != nil {
            return fmt.Errorf("failed to delete portfolio: %w", err)
        }

        return nil
    })
}
```

---

## Handler Implementation

### POST Handler

```go
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req request.CreatePortfolioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
        return
    }

    if err := validation.ValidateCreatePortfolio(req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Validation failed", err.Error())
        return
    }

    portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), req)
    if err != nil {
        api.RespondError(w, http.StatusInternalServerError, "Failed to create portfolio", err.Error())
        return
    }

    api.RespondJSON(w, http.StatusCreated, portfolio)
}
```

### PUT Handler

```go
func (h *PortfolioHandler) Update(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := validation.ValidateUUID(id); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid portfolio ID", err.Error())
        return
    }

    var req request.UpdatePortfolioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
        return
    }

    if err := validation.ValidateUpdatePortfolio(req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Validation failed", err.Error())
        return
    }

    portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), id, req)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            api.RespondError(w, http.StatusNotFound, "Portfolio not found", nil)
            return
        }
        api.RespondError(w, http.StatusInternalServerError, "Failed to update portfolio", err.Error())
        return
    }

    api.RespondJSON(w, http.StatusOK, portfolio)
}
```

### DELETE Handler

```go
func (h *PortfolioHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := validation.ValidateUUID(id); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid portfolio ID", err.Error())
        return
    }

    err := h.portfolioService.DeletePortfolio(r.Context(), id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            api.RespondError(w, http.StatusNotFound, "Portfolio not found", nil)
            return
        }
        api.RespondError(w, http.StatusInternalServerError, "Failed to delete portfolio", err.Error())
        return
    }

    w.WriteHeader(http.StatusNoContent) // 204 No Content
}
```

### Register Routes

In `router.go`:

```go
r.Route("/portfolio", func(r chi.Router) {
    r.Get("/", portfolioHandler.List)
    r.Post("/", portfolioHandler.Create)          // NEW
    r.Get("/{id}", portfolioHandler.Get)
    r.Put("/{id}", portfolioHandler.Update)       // NEW
    r.Delete("/{id}", portfolioHandler.Delete)    // NEW
})
```

---

## Error Response Patterns

### Standard Error Responses

Update `internal/api/response.go`:

```go
// Error response with optional validation details
type ErrorResponse struct {
    Error   string            `json:"error"`
    Details interface{}       `json:"details,omitempty"`
    Fields  map[string]string `json:"fields,omitempty"` // Validation errors
}

func RespondValidationError(w http.ResponseWriter, err *validation.ValidationError) {
    response := ErrorResponse{
        Error:  "Validation failed",
        Fields: err.Fields,
    }
    RespondJSON(w, http.StatusBadRequest, response)
}

func RespondNotFound(w http.ResponseWriter, resource string) {
    RespondJSON(w, http.StatusNotFound, ErrorResponse{
        Error: fmt.Sprintf("%s not found", resource),
    })
}

func RespondCreated(w http.ResponseWriter, data interface{}) {
    RespondJSON(w, http.StatusCreated, data)
}
```

### HTTP Status Code Guidelines

| Operation | Success | Not Found | Validation Error | Server Error |
|-----------|---------|-----------|------------------|--------------|
| POST (create) | 201 Created | N/A | 400 Bad Request | 500 Internal Server Error |
| PUT (update) | 200 OK | 404 Not Found | 400 Bad Request | 500 Internal Server Error |
| DELETE | 204 No Content | 404 Not Found | 400 Bad Request | 500 Internal Server Error |

---

## Complete Example: POST /api/portfolio

Here's everything together:

### 1. Request DTO (`internal/api/request/portfolio.go`)

```go
package request

type CreatePortfolioRequest struct {
    Name               string `json:"name"`
    Description        string `json:"description"`
    ExcludeFromOverview bool  `json:"exclude_from_overview"`
}
```

### 2. Validation (`internal/validation/portfolio.go`)

```go
package validation

func ValidateCreatePortfolio(req request.CreatePortfolioRequest) error {
    if strings.TrimSpace(req.Name) == "" {
        return fmt.Errorf("name is required")
    }
    if len(req.Name) > 100 {
        return fmt.Errorf("name too long (max 100)")
    }
    return nil
}
```

### 3. Repository (`internal/repository/portfolio_repository.go`)

```go
func (r *PortfolioRepository) Insert(ctx context.Context, p *model.Portfolio) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
         VALUES (?, ?, ?, ?, ?)`,
        p.ID, p.Name, p.Description, p.IsArchived, p.ExcludeFromOverview,
    )
    return err
}
```

### 4. Service (`internal/service/portfolio_service.go`)

```go
func (s *PortfolioService) CreatePortfolio(
    ctx context.Context,
    req request.CreatePortfolioRequest,
) (*model.Portfolio, error) {
    portfolio := &model.Portfolio{
        ID:                  uuid.New().String(),
        Name:                req.Name,
        Description:         req.Description,
        IsArchived:          false,
        ExcludeFromOverview: req.ExcludeFromOverview,
    }

    if err := s.portfolioRepo.Insert(ctx, portfolio); err != nil {
        return nil, err
    }

    return portfolio, nil
}
```

### 5. Handler (`internal/api/handlers/portfolios.go`)

```go
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req request.CreatePortfolioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
        return
    }

    if err := validation.ValidateCreatePortfolio(req); err != nil {
        api.RespondError(w, http.StatusBadRequest, "Validation failed", err.Error())
        return
    }

    portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), req)
    if err != nil {
        api.RespondError(w, http.StatusInternalServerError, "Create failed", err.Error())
        return
    }

    api.RespondJSON(w, http.StatusCreated, portfolio)
}
```

### 6. Route (`internal/api/router.go`)

```go
r.Post("/", portfolioHandler.Create)
```

---

## Recommended Implementation Order

### Phase 1: Foundation (Do First)
1. Add transaction support to database package
2. Add `ErrNotFound` to repository package
3. Extend response helpers for 201, 204 status codes
4. Create `internal/api/request/` package

### Phase 2: Simple CRUD (Portfolio)
1. `POST /api/portfolio` - Create portfolio
2. `PUT /api/portfolio/{id}` - Update portfolio
3. `DELETE /api/portfolio/{id}` - Delete portfolio
4. `POST /api/portfolio/{id}/archive` - Archive portfolio

### Phase 3: Related Resources (Fund)
1. `POST /api/fund` - Create fund
2. `PUT /api/fund/{id}` - Update fund
3. `DELETE /api/fund/{id}` - Delete fund (with cascade to prices)

### Phase 4: Complex Writes (Transaction/Dividend)
1. `POST /api/transaction` - Requires validation, affects portfolio calculations
2. `POST /api/dividend` - Requires FIFO calculations, affects gains/losses

### Phase 5: IBKR Integration
1. `POST /api/ibkr/config` - With encryption
2. `POST /api/ibkr/inbox/{id}/allocate` - Complex multi-table transaction

---

## Key Takeaways

1. **Always validate before service call** - Don't trust input
2. **Use transactions for multi-table operations** - Prevent partial failures
3. **Check affected rows on update/delete** - Return 404 if nothing changed
4. **Use proper HTTP status codes** - 201 for create, 204 for delete
5. **Log write operations** - You'll need audit trails
6. **Test write operations** - More important than read tests

---

*Document created: 2026-01-22*
*For: Investment Portfolio Manager Go Backend*
