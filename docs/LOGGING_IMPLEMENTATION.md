# Logging Implementation Guide

How to implement a robust logging system in Go that integrates with the existing Python backend's database logging.

---

## Table of Contents

1. [Current State Analysis](#current-state-analysis)
2. [Python Logging Architecture](#python-logging-architecture)
3. [Go Logging Options](#go-logging-options)
4. [Recommended Approach](#recommended-approach)
5. [Implementation Guide](#implementation-guide)
6. [Database Logging](#database-logging)
7. [Log Retrieval API](#log-retrieval-api)
8. [Migration Path](#migration-path)

---

## Current State Analysis

### What You Have Now

Your Go backend uses basic `log.Printf()`:

```go
// cmd/server/main.go
log.Printf("Server starting on %s", cfg.Server.Addr)
log.Fatalf("Server error: %v", err)

// internal/api/middleware/logger.go
log.Printf("%s %s - %d (%s)", method, path, statusCode, elapsed)
```

**Limitations:**
- No log levels (DEBUG, INFO, WARNING, ERROR)
- No log categories (PORTFOLIO, FUND, TRANSACTION, etc.)
- No structured output (can't parse/filter)
- Doesn't write to database (no `/developer/logs` support)
- No request context (request_id, IP, user_agent)

### What You Need

The Python backend has a sophisticated logging system that:
1. Writes to a `Log` database table
2. Supports levels: DEBUG, INFO, WARNING, ERROR, CRITICAL
3. Supports categories: PORTFOLIO, FUND, TRANSACTION, DIVIDEND, SYSTEM, DATABASE, SECURITY, IBKR, DEVELOPER
4. Captures request context (request_id, IP, user_agent)
5. Includes stack traces for errors
6. Falls back to file logging if database unavailable
7. Can be configured via `SystemSetting` table

---

## Python Logging Architecture

From `logging_service.py`:

```python
class LoggingService:
    def log(self, level, category, message, details=None, source=None, http_status=None, stack_trace=None):
        # Check if should log (level threshold)
        if not self.should_log(level):
            return response, status

        # Create log entry
        log_entry = Log(
            id=str(uuid.uuid4()),
            level=level,
            category=category,
            message=message,
            details=json.dumps(details),
            source=source or traceback.extract_stack()[-2][2],
            request_id=getattr(g, 'request_id', None),
            stack_trace=stack_trace,
            http_status=http_status,
            ip_address=request.remote_addr,
            user_agent=request.user_agent.string,
        )

        # Try database, fall back to file
        try:
            db.session.add(log_entry)
            db.session.commit()
        except Exception:
            self.logger.log(...)  # File fallback
```

### Log Table Schema

```sql
CREATE TABLE log (
    id TEXT PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL,          -- 'debug', 'info', 'warning', 'error', 'critical'
    category TEXT NOT NULL,       -- 'portfolio', 'fund', 'transaction', etc.
    message TEXT NOT NULL,
    details TEXT,                 -- JSON string
    source TEXT,                  -- Function name
    request_id TEXT,              -- UUID for request tracing
    stack_trace TEXT,
    http_status INTEGER,
    ip_address TEXT,
    user_agent TEXT
);
```

---

## Go Logging Options

### Option 1: Standard Library (Not Recommended)

```go
import "log"
log.Printf("message")
```

**Pros:** Simple, no dependencies
**Cons:** No levels, no structure, no context

### Option 2: log/slog (Go 1.21+) - Recommended

```go
import "log/slog"
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("user created", "id", userID, "name", userName)
```

**Pros:**
- Standard library (Go 1.21+)
- Structured logging (JSON/text)
- Built-in levels (Debug, Info, Warn, Error)
- Context support
- Extensible handlers

**Cons:**
- No categories (need custom attribute)
- Need custom handler for database

### Option 3: Zap (Uber) - Alternative

```go
import "go.uber.org/zap"
logger, _ := zap.NewProduction()
logger.Info("user created", zap.String("id", userID))
```

**Pros:** Very fast, structured, widely used
**Cons:** Third-party dependency, more complex setup

### Option 4: Zerolog - Alternative

```go
import "github.com/rs/zerolog"
log.Info().Str("id", userID).Msg("user created")
```

**Pros:** Fastest, zero allocation
**Cons:** Third-party dependency, chained API style

---

## Recommended Approach

**Use Go's `log/slog` with a custom database handler.**

Reasons:
1. Standard library - no external dependencies
2. You're on Go 1.25.6 (slog available since 1.21)
3. Extensible - can add database handler
4. Matches Python's approach (levels, structured data)
5. Easy to add categories as custom attributes

---

## Implementation Guide

### Step 1: Define Log Types

Create `internal/model/log.go`:

```go
package model

import (
    "time"
)

// LogLevel represents logging severity
type LogLevel string

const (
    LogLevelDebug    LogLevel = "debug"
    LogLevelInfo     LogLevel = "info"
    LogLevelWarning  LogLevel = "warning"
    LogLevelError    LogLevel = "error"
    LogLevelCritical LogLevel = "critical"
)

// LogCategory represents logging domain
type LogCategory string

const (
    LogCategoryPortfolio   LogCategory = "portfolio"
    LogCategoryFund        LogCategory = "fund"
    LogCategoryTransaction LogCategory = "transaction"
    LogCategoryDividend    LogCategory = "dividend"
    LogCategorySystem      LogCategory = "system"
    LogCategoryDatabase    LogCategory = "database"
    LogCategorySecurity    LogCategory = "security"
    LogCategoryIBKR        LogCategory = "ibkr"
    LogCategoryDeveloper   LogCategory = "developer"
)

// Log represents a log entry
type Log struct {
    ID         string      `json:"id"`
    Timestamp  time.Time   `json:"timestamp"`
    Level      LogLevel    `json:"level"`
    Category   LogCategory `json:"category"`
    Message    string      `json:"message"`
    Details    string      `json:"details,omitempty"`    // JSON string
    Source     string      `json:"source,omitempty"`     // Function name
    RequestID  string      `json:"request_id,omitempty"`
    StackTrace string      `json:"stack_trace,omitempty"`
    HTTPStatus int         `json:"http_status,omitempty"`
    IPAddress  string      `json:"ip_address,omitempty"`
    UserAgent  string      `json:"user_agent,omitempty"`
}
```

### Step 2: Create Log Repository

Create `internal/repository/log_repository.go`:

```go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/model"
)

type LogRepository struct {
    db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
    return &LogRepository{db: db}
}

func (r *LogRepository) Insert(ctx context.Context, log *model.Log) error {
    query := `
        INSERT INTO log (id, timestamp, level, category, message, details, source,
                        request_id, stack_trace, http_status, ip_address, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    _, err := r.db.ExecContext(ctx, query,
        log.ID,
        log.Timestamp,
        string(log.Level),
        string(log.Category),
        log.Message,
        log.Details,
        log.Source,
        log.RequestID,
        log.StackTrace,
        log.HTTPStatus,
        log.IPAddress,
        log.UserAgent,
    )

    if err != nil {
        return fmt.Errorf("failed to insert log: %w", err)
    }

    return nil
}

// GetLogs retrieves logs with filtering
func (r *LogRepository) GetLogs(ctx context.Context, filter LogFilter) ([]model.Log, int, error) {
    // Build query with filters
    query := `SELECT id, timestamp, level, category, message, details, source,
                     request_id, stack_trace, http_status, ip_address, user_agent
              FROM log WHERE 1=1`
    var args []any

    if len(filter.Levels) > 0 {
        query += " AND level IN (?" + strings.Repeat(",?", len(filter.Levels)-1) + ")"
        for _, l := range filter.Levels {
            args = append(args, string(l))
        }
    }

    if len(filter.Categories) > 0 {
        query += " AND category IN (?" + strings.Repeat(",?", len(filter.Categories)-1) + ")"
        for _, c := range filter.Categories {
            args = append(args, string(c))
        }
    }

    if !filter.StartDate.IsZero() {
        query += " AND timestamp >= ?"
        args = append(args, filter.StartDate)
    }

    if !filter.EndDate.IsZero() {
        query += " AND timestamp <= ?"
        args = append(args, filter.EndDate)
    }

    // Get total count
    countQuery := "SELECT COUNT(*) FROM (" + query + ")"
    var total int
    if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
        return nil, 0, fmt.Errorf("failed to count logs: %w", err)
    }

    // Add sorting and pagination
    query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
    args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to query logs: %w", err)
    }
    defer rows.Close()

    var logs []model.Log
    for rows.Next() {
        var log model.Log
        var details, source, requestID, stackTrace, ipAddr, userAgent sql.NullString
        var httpStatus sql.NullInt64

        err := rows.Scan(
            &log.ID, &log.Timestamp, &log.Level, &log.Category, &log.Message,
            &details, &source, &requestID, &stackTrace, &httpStatus, &ipAddr, &userAgent,
        )
        if err != nil {
            return nil, 0, fmt.Errorf("failed to scan log: %w", err)
        }

        log.Details = details.String
        log.Source = source.String
        log.RequestID = requestID.String
        log.StackTrace = stackTrace.String
        log.HTTPStatus = int(httpStatus.Int64)
        log.IPAddress = ipAddr.String
        log.UserAgent = userAgent.String

        logs = append(logs, log)
    }

    return logs, total, nil
}

func (r *LogRepository) Clear(ctx context.Context) error {
    _, err := r.db.ExecContext(ctx, "DELETE FROM log")
    return err
}

type LogFilter struct {
    Levels     []model.LogLevel
    Categories []model.LogCategory
    StartDate  time.Time
    EndDate    time.Time
    Page       int
    PerPage    int
}
```

### Step 3: Create Logger Service

Create `internal/service/logger_service.go`:

```go
package service

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "runtime"
    "time"

    "github.com/google/uuid"
    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/model"
    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/repository"
)

// Logger provides unified logging to database and console
type Logger struct {
    repo       *repository.LogRepository
    slogger    *slog.Logger
    minLevel   model.LogLevel
    enabled    bool
}

// NewLogger creates a new logger instance
func NewLogger(db *sql.DB) *Logger {
    repo := repository.NewLogRepository(db)

    // Console logger for fallback
    slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    return &Logger{
        repo:     repo,
        slogger:  slogger,
        minLevel: model.LogLevelInfo, // Default, can be configured
        enabled:  true,
    }
}

// SetMinLevel sets the minimum log level
func (l *Logger) SetMinLevel(level model.LogLevel) {
    l.minLevel = level
}

// SetEnabled enables or disables logging
func (l *Logger) SetEnabled(enabled bool) {
    l.enabled = enabled
}

// shouldLog checks if the given level should be logged
func (l *Logger) shouldLog(level model.LogLevel) bool {
    if !l.enabled {
        return false
    }

    levelPriority := map[model.LogLevel]int{
        model.LogLevelDebug:    0,
        model.LogLevelInfo:     1,
        model.LogLevelWarning:  2,
        model.LogLevelError:    3,
        model.LogLevelCritical: 4,
    }

    return levelPriority[level] >= levelPriority[l.minLevel]
}

// LogEntry contains data for a log entry
type LogEntry struct {
    Level      model.LogLevel
    Category   model.LogCategory
    Message    string
    Details    map[string]interface{}
    HTTPStatus int
    RequestID  string
    IPAddress  string
    UserAgent  string
}

// Log writes a log entry
func (l *Logger) Log(ctx context.Context, entry LogEntry) {
    if !l.shouldLog(entry.Level) {
        return
    }

    // Get source (calling function)
    _, file, line, ok := runtime.Caller(2)
    source := "unknown"
    if ok {
        source = fmt.Sprintf("%s:%d", file, line)
    }

    // Serialize details to JSON
    var detailsJSON string
    if entry.Details != nil {
        if b, err := json.Marshal(entry.Details); err == nil {
            detailsJSON = string(b)
        }
    }

    // Get stack trace for errors
    var stackTrace string
    if entry.Level == model.LogLevelError || entry.Level == model.LogLevelCritical {
        buf := make([]byte, 4096)
        n := runtime.Stack(buf, false)
        stackTrace = string(buf[:n])
    }

    log := &model.Log{
        ID:         uuid.New().String(),
        Timestamp:  time.Now().UTC(),
        Level:      entry.Level,
        Category:   entry.Category,
        Message:    entry.Message,
        Details:    detailsJSON,
        Source:     source,
        RequestID:  entry.RequestID,
        StackTrace: stackTrace,
        HTTPStatus: entry.HTTPStatus,
        IPAddress:  entry.IPAddress,
        UserAgent:  entry.UserAgent,
    }

    // Try database insert
    if err := l.repo.Insert(ctx, log); err != nil {
        // Fallback to console
        l.slogger.Error("Failed to write log to database, using console",
            "error", err,
            "original_message", entry.Message,
        )
        l.logToConsole(entry)
        return
    }

    // Also log to console for immediate visibility (optional)
    l.logToConsole(entry)
}

func (l *Logger) logToConsole(entry LogEntry) {
    level := slog.LevelInfo
    switch entry.Level {
    case model.LogLevelDebug:
        level = slog.LevelDebug
    case model.LogLevelWarning:
        level = slog.LevelWarn
    case model.LogLevelError, model.LogLevelCritical:
        level = slog.LevelError
    }

    l.slogger.Log(context.Background(), level, entry.Message,
        "category", string(entry.Category),
        "request_id", entry.RequestID,
    )
}

// Convenience methods

func (l *Logger) Debug(ctx context.Context, category model.LogCategory, message string, details map[string]interface{}) {
    l.Log(ctx, LogEntry{Level: model.LogLevelDebug, Category: category, Message: message, Details: details})
}

func (l *Logger) Info(ctx context.Context, category model.LogCategory, message string, details map[string]interface{}) {
    l.Log(ctx, LogEntry{Level: model.LogLevelInfo, Category: category, Message: message, Details: details})
}

func (l *Logger) Warning(ctx context.Context, category model.LogCategory, message string, details map[string]interface{}) {
    l.Log(ctx, LogEntry{Level: model.LogLevelWarning, Category: category, Message: message, Details: details})
}

func (l *Logger) Error(ctx context.Context, category model.LogCategory, message string, details map[string]interface{}) {
    l.Log(ctx, LogEntry{Level: model.LogLevelError, Category: category, Message: message, Details: details})
}

func (l *Logger) Critical(ctx context.Context, category model.LogCategory, message string, details map[string]interface{}) {
    l.Log(ctx, LogEntry{Level: model.LogLevelCritical, Category: category, Message: message, Details: details})
}
```

### Step 4: Request Context Middleware

Create or update `internal/api/middleware/request_context.go`:

```go
package middleware

import (
    "context"
    "net/http"

    "github.com/google/uuid"
)

type contextKey string

const (
    RequestIDKey contextKey = "request_id"
    IPAddressKey contextKey = "ip_address"
    UserAgentKey contextKey = "user_agent"
)

// RequestContext adds request metadata to context
func RequestContext(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Add request ID
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        ctx = context.WithValue(ctx, RequestIDKey, requestID)

        // Add IP address
        ip := r.RemoteAddr
        if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
            ip = forwarded
        }
        ctx = context.WithValue(ctx, IPAddressKey, ip)

        // Add user agent
        ctx = context.WithValue(ctx, UserAgentKey, r.UserAgent())

        // Set request ID in response header for debugging
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Helper functions to extract from context
func GetRequestID(ctx context.Context) string {
    if v := ctx.Value(RequestIDKey); v != nil {
        return v.(string)
    }
    return ""
}

func GetIPAddress(ctx context.Context) string {
    if v := ctx.Value(IPAddressKey); v != nil {
        return v.(string)
    }
    return ""
}

func GetUserAgent(ctx context.Context) string {
    if v := ctx.Value(UserAgentKey); v != nil {
        return v.(string)
    }
    return ""
}
```

### Step 5: Using the Logger in Handlers

```go
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req request.CreatePortfolioRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.logger.Warning(ctx, model.LogCategoryPortfolio, "Invalid JSON in create portfolio request", map[string]interface{}{
            "error": err.Error(),
        })
        api.RespondError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
        return
    }

    portfolio, err := h.portfolioService.CreatePortfolio(ctx, req)
    if err != nil {
        h.logger.Error(ctx, model.LogCategoryPortfolio, "Failed to create portfolio", map[string]interface{}{
            "error": err.Error(),
            "name":  req.Name,
        })
        api.RespondError(w, http.StatusInternalServerError, "Create failed", err.Error())
        return
    }

    h.logger.Info(ctx, model.LogCategoryPortfolio, "Portfolio created", map[string]interface{}{
        "portfolio_id": portfolio.ID,
        "name":         portfolio.Name,
    })

    api.RespondJSON(w, http.StatusCreated, portfolio)
}
```

### Step 6: Register Middleware

In `router.go`:

```go
r.Use(middleware.RequestContext) // Add this before other middleware
r.Use(middleware.Logger)
r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
```

---

## Log Retrieval API

### Implement Developer Logs Handler

Create `internal/api/handlers/developer.go`:

```go
package handlers

import (
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/api"
    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/model"
    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/repository"
    "github.com/yourname/Investment-Portfolio-Manager-Backend/internal/service"
)

type DeveloperHandler struct {
    logRepo *repository.LogRepository
    logger  *service.Logger
}

func NewDeveloperHandler(logRepo *repository.LogRepository, logger *service.Logger) *DeveloperHandler {
    return &DeveloperHandler{
        logRepo: logRepo,
        logger:  logger,
    }
}

// GetLogs handles GET /api/developer/logs
func (h *DeveloperHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    query := r.URL.Query()

    // Parse filter parameters
    filter := repository.LogFilter{
        Page:    1,
        PerPage: 50,
    }

    if levels := query.Get("levels"); levels != "" {
        for _, l := range strings.Split(levels, ",") {
            filter.Levels = append(filter.Levels, model.LogLevel(strings.ToLower(l)))
        }
    }

    if categories := query.Get("categories"); categories != "" {
        for _, c := range strings.Split(categories, ",") {
            filter.Categories = append(filter.Categories, model.LogCategory(strings.ToLower(c)))
        }
    }

    if startDate := query.Get("start_date"); startDate != "" {
        if t, err := time.Parse(time.RFC3339, startDate); err == nil {
            filter.StartDate = t
        }
    }

    if endDate := query.Get("end_date"); endDate != "" {
        if t, err := time.Parse(time.RFC3339, endDate); err == nil {
            filter.EndDate = t
        }
    }

    if page := query.Get("page"); page != "" {
        if p, err := strconv.Atoi(page); err == nil && p > 0 {
            filter.Page = p
        }
    }

    if perPage := query.Get("per_page"); perPage != "" {
        if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 && pp <= 100 {
            filter.PerPage = pp
        }
    }

    logs, total, err := h.logRepo.GetLogs(ctx, filter)
    if err != nil {
        h.logger.Error(ctx, model.LogCategoryDeveloper, "Failed to retrieve logs", map[string]interface{}{
            "error": err.Error(),
        })
        api.RespondError(w, http.StatusInternalServerError, "Failed to retrieve logs", err.Error())
        return
    }

    pages := (total + filter.PerPage - 1) / filter.PerPage

    response := map[string]interface{}{
        "logs":         logs,
        "total":        total,
        "pages":        pages,
        "current_page": filter.Page,
    }

    api.RespondJSON(w, http.StatusOK, response)
}

// ClearLogs handles DELETE /api/developer/logs
func (h *DeveloperHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    if err := h.logRepo.Clear(ctx); err != nil {
        h.logger.Error(ctx, model.LogCategoryDeveloper, "Failed to clear logs", map[string]interface{}{
            "error": err.Error(),
        })
        api.RespondError(w, http.StatusInternalServerError, "Failed to clear logs", err.Error())
        return
    }

    h.logger.Info(ctx, model.LogCategoryDeveloper, "Logs cleared", nil)
    w.WriteHeader(http.StatusNoContent)
}
```

### Add Routes

```go
r.Route("/developer", func(r chi.Router) {
    r.Get("/logs", developerHandler.GetLogs)
    r.Delete("/logs", developerHandler.ClearLogs)
})
```

---

## Migration Path

### Phase 1: Basic Logging (Week 1)
1. Create `model/log.go` with types
2. Create `repository/log_repository.go`
3. Create `service/logger_service.go`
4. Add `RequestContext` middleware
5. Wire up in `main.go`

### Phase 2: Integration (Week 2)
1. Add logger to existing handlers
2. Log key operations (create, update, delete)
3. Log errors with context

### Phase 3: API & Settings (Week 3)
1. Implement `GET /api/developer/logs`
2. Implement `DELETE /api/developer/logs`
3. Add `SystemSetting` support for log level/enabled

### Phase 4: Advanced (Later)
1. Log export functionality
2. Log rotation/cleanup
3. Performance monitoring

---

## Key Differences from Python

| Aspect | Python | Go |
|--------|--------|-----|
| Levels | LogLevel enum | const with LogLevel type |
| Categories | LogCategory enum | const with LogCategory type |
| Request context | Flask's `g` object | `context.Context` with values |
| Stack traces | `traceback.format_exc()` | `runtime.Stack()` |
| JSON serialization | `json.dumps()` | `json.Marshal()` |
| Database ORM | SQLAlchemy | `database/sql` |

---

## Testing the Logger

```go
// internal/service/logger_service_test.go
func TestLogger_LogLevels(t *testing.T) {
    // Create test database
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()

    // Create log table
    db.Exec(`CREATE TABLE log (...)`)

    logger := NewLogger(db)
    logger.SetMinLevel(model.LogLevelWarning)

    ctx := context.Background()

    // This should NOT be logged (below threshold)
    logger.Info(ctx, model.LogCategorySystem, "test info", nil)

    // This should be logged
    logger.Warning(ctx, model.LogCategorySystem, "test warning", nil)

    // Verify
    var count int
    db.QueryRow("SELECT COUNT(*) FROM log").Scan(&count)
    assert.Equal(t, 1, count) // Only warning was logged
}
```

---

*Document created: 2026-01-22*
*For: Investment Portfolio Manager Go Backend*
