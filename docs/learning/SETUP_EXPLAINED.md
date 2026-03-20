# Go Backend Setup - Detailed Explanation

This document explains in detail what was built, how it works, and the Go patterns used.

## Table of Contents

1. [Overview](#overview)
2. [Project Structure](#project-structure)
3. [Component Breakdown](#component-breakdown)
4. [How It All Works Together](#how-it-all-works-together)
5. [Go Patterns & Concepts Used](#go-patterns--concepts-used)
6. [Next Steps](#next-steps)

---

## Overview

### What Was Built

A minimal but complete Go web application that:
- Starts an HTTP server on port 5001
- Connects to your existing SQLite database
- Exposes a health check endpoint at `/api/system/health`
- Uses industry-standard patterns and libraries

### Architecture Layers

```
HTTP Request → Router → Handler → Service → Database
                ↓
          Middleware (logging, CORS)
```

**Flow Example:**
1. Client makes GET request to `/api/system/health`
2. Request passes through middleware (logging, CORS)
3. Router matches route and calls `SystemHandler.Health()`
4. Handler calls `SystemService.CheckHealth()`
5. Service executes `SELECT 1` query on database
6. Response bubbles back up: Service → Handler → Router → Client

---

## Project Structure

### Layout Explained

```
Investment-Portfolio-Manager-Backend/
├── cmd/                           # Application entry points
│   └── server/
│       └── main.go               # Main application (starts server)
│
├── internal/                      # Private application code
│   ├── api/                      # HTTP layer
│   │   ├── handlers/             # HTTP request handlers
│   │   ├── middleware/           # HTTP middleware
│   │   ├── response.go           # Response helpers
│   │   └── router.go             # Route definitions
│   │
│   ├── config/                   # Configuration management
│   │   └── config.go
│   │
│   ├── database/                 # Database layer
│   │   └── database.go
│   │
│   └── service/                  # Business logic layer
│       └── system_service.go
│
├── data/                         # Database files
│   └── portfolio_manager.db
│
├── .env.example                  # Environment template
├── Makefile                      # Build automation
├── go.mod                        # Go module definition
└── go.sum                        # Dependency checksums
```

### Why This Structure?

**`cmd/` for entry points:**
- Go convention: executable commands go in `cmd/`
- Allows multiple binaries from same codebase
- `cmd/server/main.go` is the web server
- Later could add `cmd/migrate/main.go` for migrations

**`internal/` for private code:**
- Go special directory: code here CANNOT be imported by other projects
- Ensures your implementation stays encapsulated
- Forces clean API boundaries

**Layered architecture:**
- **API layer**: HTTP concerns (routing, middleware, request/response)
- **Service layer**: Business logic (what to do)
- **Database layer**: Data access (how to get data)

This separation makes testing easier and code more maintainable.

---

## Component Breakdown

### 1. Entry Point: `cmd/server/main.go`

**Purpose:** Start the application, wire everything together

**Key Responsibilities:**
```go
func main() {
    // 1. Load configuration
    cfg, err := config.Load()

    // 2. Open database
    db, err := database.Open(cfg.Database.Path)
    defer db.Close()

    // 3. Create services
    systemService := service.NewSystemService(db)

    // 4. Create router with routes
    router := api.NewRouter(systemService, cfg)

    // 5. Create and start HTTP server
    server := &http.Server{
        Addr:    cfg.Server.Addr,
        Handler: router,
    }
    server.ListenAndServe()

    // 6. Graceful shutdown handling
}
```

**Go Patterns Used:**

**Error Handling:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```
- Go returns errors as values (not exceptions)
- Convention: last return value is error
- Check `err != nil` after every operation that can fail
- Use `log.Fatalf()` for fatal errors at startup

**Deferred Cleanup:**
```go
defer db.Close()
```
- `defer` schedules function call when surrounding function returns
- Runs even if panic occurs
- Common pattern: acquire resource, immediately defer its cleanup
- Ensures database connection closes when main() exits

**Graceful Shutdown:**
```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit  // Block until signal received

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```
- Catches Ctrl+C (SIGINT) and kill (SIGTERM)
- Allows 30 seconds for in-flight requests to complete
- Prevents data loss or broken connections

---

### 2. Configuration: `internal/config/config.go`

**Purpose:** Load settings from environment variables and `.env` file

**How It Works:**

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    CORS     CORSConfig
}

func Load() (*Config, error) {
    // 1. Load .env file (ignores error if missing)
    _ = godotenv.Load()

    // 2. Read environment variables with defaults
    config := &Config{
        Server: ServerConfig{
            Port: getEnv("SERVER_PORT", "5001"),
            Host: getEnv("SERVER_HOST", "localhost"),
        },
        // ...
    }

    return config, nil
}
```

**Go Patterns Used:**

**Struct Composition:**
```go
type Config struct {
    Server   ServerConfig    // Embedded struct
    Database DatabaseConfig  // Keeps related config together
    CORS     CORSConfig
}
```
- Organizes related configuration into logical groups
- Each sub-config is its own type
- Makes code self-documenting

**Pointer Returns:**
```go
func Load() (*Config, error) {
    config := &Config{...}  // Create on heap
    return config, nil       // Return pointer
}
```
- `&Config{}` creates struct on heap, returns pointer
- Allows modification after return
- Standard for larger structs (more efficient than copying)

**Error Ignoring:**
```go
_ = godotenv.Load()
```
- Underscore `_` explicitly ignores return value
- Okay here because `.env` file is optional
- Go compiler enforces you handle or explicitly ignore errors

---

### 3. Database: `internal/database/database.go`

**Purpose:** Open SQLite connection, configure it properly

**How It Works:**

```go
func Open(dbPath string) (*sql.DB, error) {
    // 1. Open database connection
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // 2. Test the connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    // 3. Configure SQLite
    db.Exec("PRAGMA foreign_keys = ON")
    db.Exec("PRAGMA timezone = 'UTC'")

    return db, nil
}
```

**Go Patterns Used:**

**Import Side Effects:**
```go
import _ "modernc.org/sqlite"
```
- Underscore before import name
- Imports package only for its side effects (registers SQLite driver)
- Doesn't use package directly in this file
- Driver registers itself with `database/sql` package

**Error Wrapping:**
```go
return nil, fmt.Errorf("failed to open database: %w", err)
```
- `%w` verb wraps the original error
- Creates error chain: outer error wraps inner error
- Allows `errors.Is()` and `errors.As()` to work
- Provides context while preserving original error

**Multiple Return Values:**
```go
func Open(dbPath string) (*sql.DB, error) {
    // Returns both database AND error
}
```
- Go commonly returns `(value, error)` pairs
- Convention: check error first, then use value
- Replaces try/catch exception handling

---

### 4. Service Layer: `internal/service/system_service.go`

**Purpose:** Business logic for system operations

**How It Works:**

```go
type SystemService struct {
    db *sql.DB  // Holds database connection
}

func NewSystemService(db *sql.DB) *SystemService {
    return &SystemService{db: db}
}

func (s *SystemService) CheckHealth() error {
    return database.HealthCheck(s.db)
}
```

**Go Patterns Used:**

**Constructor Function:**
```go
func NewSystemService(db *sql.DB) *SystemService {
    return &SystemService{db: db}
}
```
- Go has no constructors
- Convention: `New<Type>()` function creates instances
- Takes dependencies as parameters (dependency injection)
- Returns pointer to struct

**Methods with Receivers:**
```go
func (s *SystemService) CheckHealth() error {
    //  ^--- receiver (like 'self' in Python)
}
```
- `(s *SystemService)` is the receiver
- Makes `CheckHealth()` a method on `SystemService`
- `s` is like `this` or `self` in other languages
- `*SystemService` is a pointer receiver (can modify struct)

**Why a Service Layer?**
- Separates business logic from HTTP layer
- Makes testing easier (no HTTP mocking needed)
- Can be reused by different interfaces (HTTP, CLI, etc.)
- Follows Single Responsibility Principle

---

### 5. HTTP Router: `internal/api/router.go`

**Purpose:** Define routes and wire up middleware

**How It Works:**

```go
func NewRouter(systemService *service.SystemService, cfg *config.Config) http.Handler {
    r := chi.NewRouter()

    // Global middleware (applies to all routes)
    r.Use(middleware.RequestID)
    r.Use(custommiddleware.Logger)
    r.Use(middleware.Recoverer)

    corsMiddleware := custommiddleware.NewCORS(cfg.CORS.AllowedOrigins)
    r.Use(corsMiddleware.Handler)

    // Define routes
    r.Route("/api", func(r chi.Router) {
        r.Route("/system", func(r chi.Router) {
            systemHandler := handlers.NewSystemHandler(systemService)
            r.Get("/health", systemHandler.Health)
        })
    })

    return r
}
```

**Go Patterns Used:**

**Interface Return Type:**
```go
func NewRouter(...) http.Handler {
    //                 ^^^^^^^^^^^ interface, not concrete type
    r := chi.NewRouter()  // Returns *chi.Mux
    return r              // *chi.Mux implements http.Handler
}
```
- Returns interface, not concrete type
- Caller doesn't need to know about Chi
- Can swap router implementation without changing callers
- Interface: any type with `ServeHTTP(ResponseWriter, *Request)` method

**Middleware Pattern:**
```go
r.Use(middleware.RequestID)
```
- Middleware wraps handler with additional behavior
- Executes before handler (logging, auth, etc.)
- Chi's `.Use()` applies middleware to all subsequent routes

**Nested Routes:**
```go
r.Route("/api", func(r chi.Router) {
    r.Route("/system", func(r chi.Router) {
        r.Get("/health", handler)  // Becomes: GET /api/system/health
    })
})
```
- Routes defined in nested functions
- Path segments combine: `/api` + `/system` + `/health`
- Groups related routes together
- Makes route structure clear

---

### 6. HTTP Handler: `internal/api/handlers/system.go`

**Purpose:** Handle HTTP requests for system endpoints

**How It Works:**

```go
type SystemHandler struct {
    systemService *service.SystemService
}

func NewSystemHandler(systemService *service.SystemService) *SystemHandler {
    return &SystemHandler{systemService: systemService}
}

func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
    // 1. Call service layer
    if err := h.systemService.CheckHealth(); err != nil {
        // 2. Build error response
        response := HealthResponse{
            Status:   "unhealthy",
            Database: "disconnected",
            Error:    err.Error(),
        }
        // 3. Send JSON response with 503 status
        respondJSON(w, http.StatusServiceUnavailable, response)
        return
    }

    // 4. Build success response
    response := HealthResponse{
        Status:   "healthy",
        Database: "connected",
    }
    // 5. Send JSON response with 200 status
    respondJSON(w, http.StatusOK, response)
}
```

**Go Patterns Used:**

**HTTP Handler Signature:**
```go
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
    //                          ^^^^^^^^^^^^^^^^^^  ^^^^^^^^^^^^^
    //                          write response      read request
}
```
- Standard Go HTTP handler signature
- `w` writes HTTP response
- `r` reads HTTP request
- Must match `http.HandlerFunc` or `http.Handler` interface

**Struct Tags:**
```go
type HealthResponse struct {
    Status   string `json:"status"`
    Database string `json:"database"`
    Error    string `json:"error,omitempty"`
}
```
- Backtick strings are struct tags
- `json:"status"` controls JSON field name
- `omitempty` omits field if zero value (empty string, 0, nil)
- Used by `json.Marshal()` and `json.Unmarshal()`

**Early Return Pattern:**
```go
if err != nil {
    respondJSON(w, http.StatusServiceUnavailable, response)
    return  // Exit early on error
}
// Success path continues here
```
- Handle error case first, return early
- Success path at bottom (not nested in else)
- Reduces indentation, improves readability
- Very common Go pattern

---

### 7. Middleware: `internal/api/middleware/`

#### A. Logger Middleware (`logger.go`)

**Purpose:** Log every HTTP request with method, path, status, duration

**How It Works:**

```go
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter to capture status code
        wrapped := &responseWriter{
            ResponseWriter: w,
            statusCode:     http.StatusOK,
        }

        // Call the next handler
        next.ServeHTTP(wrapped, r)

        // Log after handler completes
        log.Printf("%s %s %d %s",
            r.Method,
            r.URL.Path,
            wrapped.statusCode,
            time.Since(start),
        )
    })
}
```

**Go Patterns Used:**

**Higher-Order Functions:**
```go
func Logger(next http.Handler) http.Handler {
    //         ^^^^ input        ^^^^ output
    // Function that takes a handler and returns a handler
}
```
- Middleware is a function that wraps another function
- Takes handler, returns new handler that does something extra
- Classic decorator pattern

**Type Embedding:**
```go
type responseWriter struct {
    http.ResponseWriter  // Embedded interface
    statusCode int
}
```
- Embeds `http.ResponseWriter` interface
- Automatically gets all `ResponseWriter` methods
- Can override specific methods (like `WriteHeader`)
- Adds new field `statusCode` to track status

**Closure:**
```go
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // This function "closes over" variables from outer scope:
    start := time.Now()  // Captured from outer function
    next.ServeHTTP(...)  // 'next' is from parameter
})
```
- Inner function can access outer function's variables
- `start` and `next` are captured
- Allows middleware to maintain state per request

#### B. CORS Middleware (`cors.go`)

**Purpose:** Configure Cross-Origin Resource Sharing for frontend

**How It Works:**

```go
func NewCORS(allowedOrigins []string) *cors.Cors {
    return cors.New(cors.Options{
        AllowedOrigins: allowedOrigins,
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type", "X-API-Key"},
        AllowCredentials: true,
        MaxAge: 300,
    })
}
```

- Uses third-party `go-chi/cors` package
- Configures which origins can call your API
- Allows frontend on different port to make requests
- Handles preflight OPTIONS requests automatically

---

### 8. Response Helpers: `internal/api/response.go`

**Purpose:** Standardize JSON responses

**How It Works:**

```go
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
    // 1. Set Content-Type header
    w.Header().Set("Content-Type", "application/json")

    // 2. Write status code
    w.WriteHeader(status)

    // 3. Encode data as JSON to response
    if data != nil {
        json.NewEncoder(w).Encode(data)
    }
}
```

**Go Patterns Used:**

**Interface{} (Empty Interface):**
```go
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
    //                                                  ^^^^^^^^^^^
    // Accepts any type
}
```
- `interface{}` means "any type"
- Similar to `Object` in Java or `any` in TypeScript
- Allows function to accept any data structure
- `json.Encoder` handles any type that can be serialized

**Method Chaining:**
```go
json.NewEncoder(w).Encode(data)
// 1. NewEncoder(w) returns *Encoder
// 2. .Encode(data) is called on that *Encoder
```
- Create encoder that writes to `w`
- Immediately encode `data` to JSON
- Data flows: struct → JSON → http.ResponseWriter → network

---

## How It All Works Together

### Request Flow Example

Let's trace a request to `GET /api/system/health`:

```
1. HTTP Request arrives
   └─> "GET /api/system/health HTTP/1.1"

2. Go's http.Server receives it
   └─> Calls router's ServeHTTP()

3. Router (Chi) processes it
   ├─> RequestID middleware: adds unique request ID
   ├─> Logger middleware: records start time
   ├─> Recoverer middleware: sets up panic recovery
   ├─> CORS middleware: adds CORS headers
   └─> Matches route: /api/system/health → SystemHandler.Health()

4. SystemHandler.Health() executes
   ├─> Calls systemService.CheckHealth()
   │   └─> database.HealthCheck(db)
   │       └─> db.Ping() // Executes "SELECT 1" internally
   │           └─> Returns nil (success) or error
   ├─> If error:
   │   ├─> Creates HealthResponse{status: "unhealthy", ...}
   │   └─> Calls respondJSON(w, 503, response)
   └─> If success:
       ├─> Creates HealthResponse{status: "healthy", ...}
       └─> Calls respondJSON(w, 200, response)

5. respondJSON() sends response
   ├─> Sets Content-Type: application/json
   ├─> Writes HTTP status code (200 or 503)
   └─> Encodes struct to JSON → writes to ResponseWriter

6. Logger middleware logs the request
   └─> "GET /api/system/health 200 1.234ms"

7. HTTP Response sent to client
   └─> {"status":"healthy","database":"connected"}
```

### Data Flow

```
main.go:
    ├─> Creates *sql.DB (database connection)
    ├─> Passes to SystemService constructor
    │   └─> SystemService stores reference to *sql.DB
    ├─> Passes SystemService to NewRouter()
    │   └─> Router passes to NewSystemHandler()
    │       └─> Handler stores reference to SystemService
    └─> Passes Router to http.Server
        └─> Server calls Router for each request

Dependency Chain:
    http.Server → Router → Handler → Service → Database
```

This is **Dependency Injection**: each component receives its dependencies as constructor parameters.

---

## Go Patterns & Concepts Used

### 1. Error Handling as Values

**Python approach (exceptions):**
```python
try:
    db.connect()
except DatabaseError as e:
    log.error(f"Failed: {e}")
```

**Go approach (errors as values):**
```go
db, err := database.Open(path)
if err != nil {
    log.Fatalf("Failed: %v", err)
}
```

**Why:**
- Makes error handling explicit
- Can't accidentally ignore errors (compiler checks)
- Error flow is visible in code (not hidden in try/catch)

### 2. Interfaces

```go
// http.Handler is an interface:
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// Any type with this method satisfies the interface:
type MyHandler struct{}
func (h *MyHandler) ServeHTTP(w ResponseWriter, r *Request) {}
// MyHandler is now an http.Handler (no explicit declaration needed!)
```

**Key concepts:**
- Implicit implementation (no `implements` keyword)
- Types satisfy interfaces automatically if they have the right methods
- Small interfaces (often one method) are idiomatic
- Enables polymorphism without inheritance

### 3. Pointers vs Values

**When to use pointers:**
```go
func NewService(db *sql.DB) *Service {
    //              ^            ^
    //         accept pointer  return pointer
    return &Service{db: db}
}
```

**Rules of thumb:**
- Pointers for large structs (more efficient)
- Pointers when method needs to modify receiver
- Values for small, immutable data (int, bool, small structs)
- Interfaces already contain pointers (don't need `*interface{}`)

### 4. Package Organization

```go
package handlers  // Package name

import (
    "net/http"  // Standard library

    "github.com/go-chi/chi/v5"  // Third-party

    "github.com/ndewijer/...internal/service"  // Internal
)
```

**Import rules:**
- Group standard library, third-party, internal
- Blank line between groups
- Alphabetical within groups
- Unused imports cause compile error

### 5. Exported vs Unexported

```go
type SystemService struct {  // Exported (capital S)
    db *sql.DB              // unexported (lowercase d)
}

func NewSystemService() {}   // Exported
func (s *SystemService) checkHealth() {}  // unexported
```

**Rules:**
- Capital first letter = exported (public)
- Lowercase first letter = unexported (private to package)
- Applies to: types, functions, methods, fields, constants

### 6. Defer for Cleanup

```go
func main() {
    db, _ := database.Open("db.sqlite")
    defer db.Close()  // Called when main() exits

    // ... do work ...

}  // db.Close() called here automatically
```

**Common uses:**
- Close files: `defer file.Close()`
- Unlock mutexes: `defer mu.Unlock()`
- Recover panics: `defer recover()`

### 7. Zero Values

```go
var s string        // s = ""
var i int          // i = 0
var b bool         // b = false
var p *int         // p = nil
var slice []int    // slice = nil
```

**Why it matters:**
- Every type has a zero value (no undefined)
- Makes types usable immediately
- `nil` for pointers, slices, maps, interfaces, channels

---

## Next Steps

### To Learn More About This Code

1. **Experiment:**
   - Change the response message
   - Add a new field to `HealthResponse`
   - Create a new endpoint `/api/system/version`
   - Break things and see error messages

2. **Read the Code:**
   - Start at `main.go`, follow the flow
   - Add `log.Printf()` statements to trace execution
   - Use a debugger to step through

3. **Understand the Imports:**
   - Look up `go-chi/chi` documentation
   - Read about `database/sql` package
   - Explore `modernc.org/sqlite`

### To Build On This

1. **Add Tests:**
   - Create `internal/api/handlers/system_test.go`
   - Use `httptest` package to test handlers
   - Mock the service layer

2. **Add More Endpoints:**
   - Implement `GET /api/system/version`
   - Add portfolio endpoints
   - Follow the implementation plan

3. **Add Database Operations:**
   - Create repository layer
   - Write SQL queries
   - Learn `database/sql` patterns

### Go Learning Resources

- **Official Tour:** https://go.dev/tour/
- **Effective Go:** https://go.dev/doc/effective_go
- **Go by Example:** https://gobyexample.com/
- **Standard Library:** https://pkg.go.dev/std

---

## Summary

You now have:
- ✅ Working HTTP server
- ✅ Database connection
- ✅ Health check endpoint
- ✅ Proper project structure
- ✅ Middleware stack
- ✅ Service layer pattern
- ✅ Clean code following Go conventions

The foundation is solid and ready to build upon. Each file is small, focused, and easy to understand. The patterns used are standard Go practices you'll see in professional codebases.

Happy learning! 🚀
