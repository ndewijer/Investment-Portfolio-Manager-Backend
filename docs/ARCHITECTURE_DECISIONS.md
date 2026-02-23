# Architecture Decisions & Alternatives

This document explains **why** we made specific implementation choices and **what alternatives** were considered. Each section follows this format:

- **Context:** What decision needed to be made
- **Decision:** What we chose
- **Alternatives Considered:** Other viable options
- **Rationale:** Why we chose this approach
- **Trade-offs:** Pros and cons

---

## Table of Contents

1. [Web Framework Choice](#1-web-framework-choice)
2. [Database Driver](#2-database-driver)
3. [Database Access Layer](#3-database-access-layer)
4. [Project Structure](#4-project-structure)
5. [Service Layer Pattern](#5-service-layer-pattern)
6. [Middleware Stack](#6-middleware-stack)
7. [Configuration Management](#7-configuration-management)
8. [Error Handling Strategy](#8-error-handling-strategy)
9. [Hybrid Database Approach](#9-hybrid-database-approach)
10. [PortfolioFundRepository — Separate Join Table Repository](#10-portfoliofundrepository--separate-join-table-repository)

---

## 1. Web Framework Choice

### Context
We needed to choose an HTTP router/framework for building REST APIs in Go. The Python backend uses Flask-RESTX, which provides routing, request handling, and automatic API documentation.

### Decision
**Chi Router** (`github.com/go-chi/chi/v5`)

### Alternatives Considered

#### A. Standard Library `net/http`
**What it is:**
- Go's built-in HTTP package
- Since Go 1.22, includes enhanced routing with path patterns

**Example:**
```go
mux := http.NewServeMux()
mux.HandleFunc("GET /api/system/health", healthHandler)
mux.HandleFunc("POST /api/portfolio/{id}", createHandler)
```

**Pros:**
- Zero dependencies
- Fastest performance (no abstraction overhead)
- Never breaks (stdlib is stable)
- Forces you to learn Go fundamentals

**Cons:**
- No middleware chain (you build it yourself)
- Basic route matching (less powerful than Chi)
- No route grouping or subrouters
- More boilerplate for common patterns

**Why not chosen:**
- While excellent for learning, it would require building middleware infrastructure from scratch
- Chi provides better route organization for 72 endpoints
- Chi is still stdlib-compatible (same interfaces), so the learning value remains

---

#### B. Gin Framework
**What it is:**
- Most popular Go web framework
- Full-featured with built-in validation, rendering, middleware

**Example:**
```go
r := gin.Default()
r.GET("/api/system/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy"})
})
```

**Pros:**
- Very fast (custom HTTP implementation)
- Large ecosystem and community
- Built-in JSON validation
- Excellent documentation

**Cons:**
- Uses custom `gin.Context` instead of standard `http.ResponseWriter`
- Harder to use with stdlib tools
- More "magical" (hides Go fundamentals)
- Might be overkill for API-only backend

**Why not chosen:**
- Your goal is to **learn Go**, and Gin's abstractions hide too much
- Custom context makes code less portable to other frameworks
- Chi keeps you closer to standard library patterns

---

#### C. Echo or Fiber
**What they are:**
- Echo: Similar to Gin, but slightly different API
- Fiber: Express.js-inspired framework, uses fasthttp (not net/http)

**Why not considered seriously:**
- Fiber uses `fasthttp` instead of `net/http` (completely different ecosystem)
- Echo is similar to Gin but with smaller community
- Both move you further from Go idioms

---

### Rationale for Chi

**Chi is the "Goldilocks" choice:**
1. **Stdlib-compatible:** Uses standard `http.Handler` and `http.HandlerFunc`
   - Everything you learn applies to any Go HTTP code
   - Can mix Chi with stdlib code seamlessly

2. **Lightweight:** Minimal abstraction over stdlib
   - ~1000 lines of code
   - Only adds routing logic, not a new paradigm

3. **Good middleware ecosystem:**
   - Includes common middleware (logger, CORS, recovery)
   - Easy to write custom middleware

4. **Excellent route organization:**
   ```go
   r.Route("/api", func(r chi.Router) {
       r.Route("/system", func(r chi.Router) {
           r.Get("/health", handler)
       })
   })
   ```
   - Nested routes keep 72 endpoints organized
   - Clear visual structure

5. **Production-ready:** Used by companies like Basecamp, CloudFlare

**Learning perspective:**
- You'll learn `http.ResponseWriter` and `http.Request` (transferable skills)
- Not locked into Chi-specific patterns
- Easy migration to stdlib or other frameworks later

---

## 2. Database Driver

### Context
We need a SQLite driver to connect to `portfolio_manager.db`. In Python, we used SQLAlchemy with sqlite3. Go's `database/sql` requires a driver.

### Decision
**modernc.org/sqlite** (Pure Go driver)

### Alternatives Considered

#### A. mattn/go-sqlite3 (CGO-based)
**What it is:**
- Wraps the C SQLite library using CGO
- Most popular SQLite driver for Go

**Pros:**
- Battle-tested (most widely used)
- Slightly faster for some operations
- Full SQLite feature support
- Large community

**Cons:**
- **Requires CGO:** Must have C compiler installed
- **Cross-compilation is painful:** Building for different OS/architecture is complex
- **Deployment complexity:** Binary is platform-specific
- **Slower builds:** CGO compilation adds overhead

**Example of CGO pain:**
```bash
# Building for Linux from Mac:
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc go build
# Need to install cross-compiler toolchains!
```

---

#### B. crawshaw.io/sqlite
**What it is:**
- Another CGO-based SQLite driver
- Different API (not database/sql compatible)

**Why not considered:**
- Also requires CGO
- Non-standard API (doesn't use database/sql)
- Smaller community than mattn

---

### Rationale for modernc.org/sqlite

**Pure Go = Better Developer Experience:**

1. **No CGO required:**
   ```bash
   go build  # Just works on any machine with Go installed
   ```
   - No C compiler needed
   - Faster builds
   - Simpler development environment

2. **Cross-compilation is trivial:**
   ```bash
   GOOS=linux go build    # Build for Linux
   GOOS=windows go build  # Build for Windows
   GOOS=darwin go build   # Build for Mac
   ```
   - Single command, no additional tools
   - Perfect for Docker containers

3. **Database/sql compatible:**
   - Uses standard `database/sql` interface
   - Same API as mattn/go-sqlite3
   - Easy to switch if needed

4. **Good performance:**
   - Slightly slower than CGO version (5-10%)
   - But fast enough for most applications
   - SQLite is already fast

**Trade-off accepted:**
- Marginally slower than CGO version
- Worth it for deployment simplicity and ease of use

**Why it matters for learning:**
- Get started immediately without installing C tools
- Builds are fast, encouraging experimentation
- Works in GitHub Actions, Docker, etc. without extra setup

---

## 3. Database Access Layer

### Context
How should we write and execute SQL queries? Python backend uses SQLAlchemy ORM. We need to decide on the Go equivalent.

### Decision
**Hybrid Approach:**
- Phase 1-2: `database/sql` (stdlib)
- Phase 3: Migrate to `sqlc` + `Atlas`

### Alternatives Considered

#### A. GORM (Full ORM)
**What it is:**
- Go's most popular ORM
- Similar to SQLAlchemy, ActiveRecord

**Example:**
```go
type User struct {
    ID   uint
    Name string
}

db.Create(&User{Name: "John"})
db.First(&user, 1)
db.Model(&user).Update("Name", "Jane")
```

**Pros:**
- Feels familiar (like SQLAlchemy)
- Automatic migrations
- Less SQL to write
- Relationship management
- Hooks and callbacks

**Cons:**
- **Hides SQL:** You don't learn what's actually happening
- **Magic queries:** `db.Where("age > ?", 18)` - what SQL is generated?
- **Performance surprises:** N+1 queries, inefficient joins
- **Debugging is harder:** Error messages reference GORM internals
- **Limited for complex queries:** Ends up requiring raw SQL anyway

**Why not chosen:**
- Your goal is to **learn Go and SQL fundamentals**
- GORM's magic defeats the learning purpose
- Raw SQL gives you full control and understanding

---

#### B. sqlx (Lightweight extension)
**What it is:**
- Extensions to `database/sql`
- Adds struct scanning, named parameters

**Example:**
```go
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
}

var user User
err := db.Get(&user, "SELECT * FROM users WHERE id = $1", 1)
```

**Pros:**
- Less boilerplate than database/sql
- Still write SQL
- Struct mapping is convenient
- Lightweight, minimal magic

**Cons:**
- Still requires manual SQL writing
- No compile-time SQL validation
- Mapping can be brittle with schema changes

**Why not chosen:**
- Similar effort to raw SQL, but less explicit
- Doesn't solve the main pain points
- Better to go straight to sqlc for code generation

---

#### C. sqlc (Code Generator)
**What it is:**
- Generates type-safe Go code from SQL queries
- Write SQL, get Go functions automatically

**Example:**

SQL file (`queries/users.sql`):
```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :exec
INSERT INTO users (name) VALUES ($1);
```

Generated Go code:
```go
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) {
    // Implementation generated automatically
}
```

**Pros:**
- **Compile-time validation:** SQL is checked when you build
- **Type-safe:** Knows parameter and return types
- **No reflection:** Fast performance
- **Explicit SQL:** You see exactly what queries run
- **Schema awareness:** Knows your database schema

**Cons:**
- Requires code generation step
- One more tool to learn
- Need to organize SQL queries in files

---

### Rationale for Hybrid Approach

**Phase 1-2: Start with `database/sql`**

Why start here:
```go
// You write this:
row := db.QueryRow("SELECT portfolio_name FROM portfolios WHERE id = ?", id)

var name string
err := row.Scan(&name)
```

**Learning benefits:**
1. **See SQL execution flow:** Query → Scan → Error handling
2. **Understand pointer semantics:** Why `&name` in Scan?
3. **Learn database/sql patterns:** QueryRow vs Query vs Exec
4. **Handle NULL values:** Using `sql.NullString`
5. **Manage transactions manually:** Begin, Commit, Rollback
6. **Feel the pain:** Motivates why tools like sqlc exist

**Phase 3: Migrate to `sqlc`**

Why migrate later:
1. **Appreciation:** You'll understand what sqlc does for you
2. **Productivity:** 72 endpoints = lots of repetitive SQL code
3. **Type safety:** Catch errors at compile time
4. **Maintainability:** Schema changes update code automatically

**This approach:**
- Teaches fundamentals first
- Introduces productivity tools when you can appreciate them
- Provides a clear migration path (documented in implementation plan)

---

#### D. Atlas (Migration Tool)
**What it is:**
- Database migration tool
- Generates migrations by comparing schema to database

**Why pair with sqlc:**
- sqlc needs schema definitions
- Atlas manages schema evolution
- They complement each other perfectly

**Phase 3 stack:**
```
Atlas (migrations) → Schema → sqlc (code gen) → Your code
```

---

## 4. Project Structure

### Context
How should we organize the codebase? Go projects can be structured many ways. We need a layout that scales from 1 endpoint to 72 endpoints.

### Decision
**Standard Go Project Layout** with internal packages

```
cmd/
  server/
    main.go
internal/
  api/
  config/
  database/
  service/
```

### Alternatives Considered

#### A. Flat Structure
```
main.go
handlers.go
database.go
config.go
```

**Pros:**
- Simple to start
- Easy navigation
- No import paths

**Cons:**
- Everything in one package
- Can't have multiple binaries
- Doesn't scale beyond ~1000 lines
- No encapsulation

**Why not chosen:**
- We're building 72 endpoints (will be thousands of lines)
- Need logical separation

---

#### B. Domain-Driven Design (DDD) Structure
```
internal/
  portfolio/
    entity/
    repository/
    service/
    handler/
  transaction/
    entity/
    repository/
    service/
    handler/
```

**Pros:**
- Groups by domain concept
- Clear boundaries
- Matches business logic

**Cons:**
- More complex
- Harder for small projects
- Can lead to over-engineering

**Why not chosen:**
- Overkill for a learning project
- Layered architecture is simpler to understand first
- Can refactor to DDD later if needed

---

### Rationale for Chosen Structure

**`cmd/` directory:**
```
cmd/
  server/
    main.go     # Web server entry point
  migrate/      # (future) Database migrations
  worker/       # (future) Background jobs
```

**Why:**
- Go convention for executable commands
- Allows multiple binaries from one codebase
- Clear "this is the entry point"

**`internal/` directory:**
```
internal/
  api/          # HTTP layer (handlers, routing, middleware)
  service/      # Business logic
  database/     # Data access
  config/       # Configuration
```

**Why:**
- `internal/` is a special Go directory
- Code here **cannot be imported by external projects**
- Enforces encapsulation
- Prevents accidental API surface

**Layered by technical concern:**
```
api → service → database
```

- **API layer:** HTTP concerns (request/response, routing)
- **Service layer:** Business logic (portfolio calculations, validations)
- **Database layer:** Data access (queries, transactions)

**Benefits:**
1. **Separation of concerns:** Each layer has one job
2. **Testable:** Can test service without HTTP
3. **Reusable:** Service can be called from HTTP, CLI, or worker
4. **Clear dependencies:** api → service → database (never backwards)

**Comparison to Python backend:**
```
Python:                     Go:
backend/app/
  api/system_namespace.py → internal/api/handlers/system.go
  services/system.py       → internal/service/system_service.go
  models/*.py              → internal/database/ (later: repository/)
```

Structure mirrors Python backend's layering, just using Go idioms.

---

## 5. Service Layer Pattern

### Context
Where should business logic live? Handlers could call the database directly, or we could add an intermediate layer.

### Decision
**Service Layer** - Separate business logic from HTTP handlers

### Alternatives Considered

#### A. Handlers Call Database Directly
```go
func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
    // Parse request
    id := chi.URLParam(r, "id")

    // Execute query directly
    row := h.db.QueryRow("SELECT * FROM portfolios WHERE id = ?", id)
    var p Portfolio
    row.Scan(&p.ID, &p.Name, ...)

    // Return response
    json.NewEncoder(w).Encode(p)
}
```

**Pros:**
- Fewer files
- Simpler for very small projects
- Direct data flow

**Cons:**
- **Can't test business logic without HTTP:** Need to mock HTTP requests
- **Mixes concerns:** HTTP logic + business logic + SQL
- **Hard to reuse:** What if you add a CLI? Duplicate logic?
- **Violates SRP:** Handler does too many things

---

#### B. Repository Pattern (with Service)
```go
// Repository: Data access
type PortfolioRepository interface {
    GetByID(id int) (*Portfolio, error)
    Create(p *Portfolio) error
}

// Service: Business logic
type PortfolioService struct {
    repo PortfolioRepository
}

// Handler: HTTP
type PortfolioHandler struct {
    service *PortfolioService
}
```

**Pros:**
- Clear separation: Handler → Service → Repository → Database
- Repository can be mocked for service tests
- Follows "clean architecture" principles

**Cons:**
- More abstraction layers
- More files to navigate
- Can feel over-engineered for simple CRUD

**Why not chosen initially:**
- Service layer is sufficient for Phase 1-2
- Repository pattern can be added in Phase 3 with sqlc
- Don't want to over-engineer before understanding needs

---

### Rationale for Service Layer

**Current architecture:**
```go
Handler → Service → Database
```

**Why service layer:**

1. **Testable business logic:**
```go
// Test without HTTP:
service := NewSystemService(db)
err := service.CheckHealth()
assert.NoError(t, err)

// vs testing handler requires HTTP mocking
```

2. **Reusable logic:**
```go
// HTTP handler uses it:
func (h *Handler) Health(w, r) {
    err := h.systemService.CheckHealth()
}

// CLI command uses it:
func HealthCommand() {
    err := systemService.CheckHealth()
}

// No duplication!
```

3. **Clear responsibilities:**
   - **Handler:** HTTP concerns (parse request, send response)
   - **Service:** Business logic (validation, calculations, orchestration)
   - **Database:** Data access (queries)

4. **Matches Python backend:**
```python
# Python:
class SystemService:
    @staticmethod
    def check_health():
        # Business logic here

# Go equivalent:
type SystemService struct {}
func (s *SystemService) CheckHealth() error {
    // Business logic here
}
```

**Evolution path:**
- Phase 1-2: Handler → Service → SQL
- Phase 3: Handler → Service → Repository → SQL (if needed)

---

## 6. Middleware Stack

### Context
We need cross-cutting concerns like logging, CORS, panic recovery. Where should they live?

### Decision
**Middleware Chain** using Chi's middleware system

```go
r.Use(middleware.RequestID)
r.Use(custommiddleware.Logger)
r.Use(middleware.Recoverer)
r.Use(custommiddleware.NewCORS(...).Handler)
```

### Alternatives Considered

#### A. Manual in Each Handler
```go
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    // Log request
    log.Printf("Request: %s %s", r.Method, r.URL)

    // Add CORS headers
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Recover from panic
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Panic: %v", err)
        }
    }()

    // Actual handler logic
    // ...
}
```

**Pros:**
- No middleware concept to learn
- All code in one place

**Cons:**
- Massive duplication across 72 endpoints
- Easy to forget
- Hard to maintain (change in one place = change everywhere)

**Why not chosen:**
- Violates DRY principle
- Doesn't scale

---

#### B. Middleware in main.go
```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        log.Printf("...")
        next.ServeHTTP(w, r)
    })
}

handler = loggingMiddleware(corsMiddleware(recoverMiddleware(router)))
```

**Pros:**
- Pure stdlib approach
- No framework needed

**Cons:**
- Nested wrapping gets confusing
- No clear "stack" visualization
- Hard to add/remove middleware

---

### Rationale for Middleware Chain

**Why Chi middleware:**

1. **Clear chain visualization:**
```go
r.Use(middleware.RequestID)    // 1. Generate request ID
r.Use(custommiddleware.Logger) // 2. Log request
r.Use(middleware.Recoverer)    // 3. Catch panics
r.Use(corsMiddleware)          // 4. Add CORS headers
// Then: route to handler
```
Order is obvious, top to bottom.

2. **Standard pattern:**
```go
func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        // Before handler
        next.ServeHTTP(w, r)
        // After handler
    })
}
```
This is idiomatic Go, used by many frameworks.

3. **Composable:**
```go
// Global middleware
r.Use(middleware.Logger)

// Group-specific middleware
r.Route("/admin", func(r chi.Router) {
    r.Use(middleware.AdminAuth)  // Only for /admin routes
})
```

**Middleware we use:**

| Middleware | Purpose | Why |
|------------|---------|-----|
| `RequestID` | Unique ID per request | Trace requests through logs |
| `Logger` | Log method, path, status, duration | Debug and monitoring |
| `Recoverer` | Catch panics | Prevent server crashes |
| `CORS` | Cross-origin headers | Allow frontend on different port |

**Custom vs Built-in:**
- `RequestID`, `Recoverer`: Use Chi's built-in (already correct)
- `Logger`, `CORS`: Custom implementation (allows configuration)

---

## 7. Configuration Management

### Context
How should we handle configuration (port, database path, CORS origins, etc.)? Need something simple but flexible.

### Decision
**Environment variables** with `.env` file support using `godotenv`

### Alternatives Considered

#### A. Hardcoded Constants
```go
const ServerPort = "5001"
const DBPath = "./data/portfolio_manager.db"
```

**Pros:**
- Simple
- No dependencies
- Fast

**Cons:**
- Can't change without recompiling
- No dev/staging/prod differences
- Security risk (secrets in code)

**Why not chosen:**
- Need different configs for different environments

---

#### B. Config Files (YAML/JSON/TOML)
```go
// config.yaml
server:
  port: 5001
  host: localhost
database:
  path: ./data/portfolio_manager.db
```

**Pros:**
- Structured configuration
- Comments possible
- Complex nested config

**Cons:**
- Need parser library (Viper, etc.)
- One more file to manage
- Overkill for simple needs

**Why not chosen:**
- Our config is simple (5-6 values)
- Environment variables are standard in deployment (Docker, K8s, Heroku)

---

#### C. Command-line Flags
```go
port := flag.String("port", "5001", "Server port")
flag.Parse()
```

**Pros:**
- Stdlib only
- Override at runtime

**Cons:**
- Tedious for many options
- Not persistent
- Doesn't work well with Docker

**Why not chosen:**
- Environment variables are more standard for 12-factor apps

---

### Rationale for Environment Variables

**Why environment variables:**

1. **12-Factor App methodology:**
   - Strict separation of config from code
   - Same code runs in dev, staging, prod
   - Config changes don't require rebuilds

2. **Deployment-friendly:**
```bash
# Development
SERVER_PORT=5001 go run cmd/server/main.go

# Production
SERVER_PORT=8080 go run cmd/server/main.go

# Docker
docker run -e SERVER_PORT=8080 myapp

# Kubernetes
env:
  - name: SERVER_PORT
    value: "8080"
```

3. **Security:**
   - Secrets (API keys, passwords) don't go in code
   - Can use secret management (AWS Secrets Manager, etc.)

**Why `.env` file support:**

```bash
# .env file for local development
SERVER_PORT=5001
SERVER_HOST=localhost
DB_PATH=./data/portfolio_manager.db
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

- Convenience for local development
- Ignored by Git (in `.gitignore`)
- Optional (doesn't error if missing)
- Overridden by actual environment variables

**Implementation:**
```go
func Load() (*Config, error) {
    // Try to load .env (ignores error if missing)
    _ = godotenv.Load()

    // Read from environment with defaults
    config := &Config{
        Server: ServerConfig{
            Port: getEnv("SERVER_PORT", "5001"),
            Host: getEnv("SERVER_HOST", "localhost"),
        },
    }

    return config, nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

**Benefits:**
- Simple (no complex parser)
- Default values for development
- Easy to override for deployment
- Standard practice in Go community

---

## 8. Error Handling Strategy

### Context
How should we handle and propagate errors? Go returns errors as values, but there are patterns to follow.

### Decision
**Explicit error checking** with **error wrapping** and **early returns**

### Alternatives Considered

#### A. Panic for Errors
```go
db, err := database.Open(path)
if err != nil {
    panic(err)  // Crashes the program
}
```

**When it's appropriate:**
- Unrecoverable errors at startup
- Programming bugs (like index out of bounds)

**When it's not:**
- Normal error conditions (user not found, etc.)
- Recoverable errors

**Why not as main strategy:**
- Go philosophy: errors are values, not exceptions
- Panics should be rare
- Prefer returning errors

---

#### B. Ignoring Errors
```go
db, _ := database.Open(path)  // Ignore error
```

**Never acceptable except:**
- When you're certain error can't happen
- Side effects that don't matter (e.g., optional .env file)

**Go compiler:**
- Forces you to acknowledge errors
- Use `_` to explicitly ignore

---

### Rationale for Error Handling Approach

**Pattern 1: Check and return early**
```go
func DoSomething() error {
    db, err := database.Open(path)
    if err != nil {
        return err  // Return immediately
    }

    data, err := db.Query(...)
    if err != nil {
        return err  // Return immediately
    }

    // Success path at end
    return nil
}
```

**Why:**
- Errors are handled immediately
- Happy path isn't nested in else blocks
- Clear error flow
- Standard Go idiom

---

**Pattern 2: Error wrapping**
```go
if err != nil {
    return fmt.Errorf("failed to open database: %w", err)
    //                                            ^^
    //                                      wraps original error
}
```

**Why:**
- Adds context at each layer
- Preserves original error
- Creates error chain
- Enables `errors.Is()` and `errors.As()`

**Example error chain:**
```
failed to load configuration: failed to open database: unable to open database file: no such file
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  ^^^^^^^^^^^^^^^^^^^^^^^^  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
Added by main.go               Added by database.Open()  Original error from OS
```

---

**Pattern 3: Different handling by layer**

**In main.go (startup):**
```go
if err != nil {
    log.Fatalf("Failed to start: %v", err)  // Fatal: exits program
}
```
Startup errors should crash the program.

**In handlers (request handling):**
```go
if err != nil {
    respondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
        Error: err.Error(),
    })
    return  // Return error to client, don't crash
}
```
Request errors should return HTTP error, not crash server.

**In services (business logic):**
```go
if err != nil {
    return fmt.Errorf("failed to get portfolio: %w", err)  // Return error up
}
```
Let caller decide how to handle.

---

**Comparison to Python:**
```python
# Python:
try:
    db = open_database(path)
except DatabaseError as e:
    log.error(f"Failed: {e}")
    raise

# Go:
db, err := database.Open(path)
if err != nil {
    log.Printf("Failed: %v", err)
    return err
}
```

**Go approach:**
- More verbose (check every error)
- But more explicit (can't forget to catch)
- Error flow is visible in code

---

## 9. Hybrid Database Approach

### Context
We need to balance **learning fundamentals** with **eventual productivity**. Should we start with advanced tools or build up?

### Decision
**Phased approach:** raw SQL first, then migrate to sqlc + Atlas

### Alternatives Considered

#### A. Start with GORM (ORM from day 1)
**Path:**
```
Phase 1: GORM
Phase 2: GORM
Phase 3: GORM
```

**Pros:**
- Fast to build features
- Less SQL to write
- Familiar if you know SQLAlchemy

**Cons:**
- Don't learn SQL or database/sql
- Hidden complexity
- Hard to debug
- Performance surprises later

---

#### B. Stay with database/sql forever
**Path:**
```
Phase 1: database/sql
Phase 2: database/sql
Phase 3: database/sql
```

**Pros:**
- Simple mental model
- No code generation
- Full control

**Cons:**
- Lots of boilerplate for 72 endpoints
- No compile-time SQL validation
- Schema changes require manual code updates
- Repetitive error handling

---

#### C. Jump straight to sqlc
**Path:**
```
Phase 1: sqlc
Phase 2: sqlc
Phase 3: sqlc
```

**Pros:**
- Best productivity from start
- Type-safe immediately

**Cons:**
- Don't appreciate what sqlc does for you
- Miss learning database/sql patterns
- Harder to debug if you don't understand underlying layer

---

### Rationale for Hybrid Approach

**The learning journey:**

```
Phase 1-2: database/sql
    ↓
    Learn fundamentals
    Feel the pain
    ↓
Phase 3: sqlc + Atlas
    ↓
    Appreciate automation
    Use productivity tools
```

**Phase 1-2: Fundamentals with database/sql**

What you'll write:
```go
func (s *Service) GetPortfolio(id int) (*Portfolio, error) {
    query := `
        SELECT id, name, user_id, total_value
        FROM portfolios
        WHERE id = ?
    `

    row := s.db.QueryRow(query, id)

    var p Portfolio
    err := row.Scan(&p.ID, &p.Name, &p.UserID, &p.TotalValue)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to scan: %w", err)
    }

    return &p, nil
}
```

**What you learn:**
- ✅ How `QueryRow` vs `Query` vs `Exec` work
- ✅ Pointer semantics (`&p.ID`)
- ✅ Error handling patterns (`sql.ErrNoRows`)
- ✅ NULL handling (`sql.NullString`)
- ✅ Transaction management
- ✅ Connection pooling concepts

**Pain points you'll feel:**
- ❌ Typing same Scan pattern repeatedly
- ❌ Easy to forget a field in Scan (runtime error)
- ❌ Schema changes = hunt for all queries
- ❌ No compile-time validation of SQL
- ❌ Lots of boilerplate

---

**Phase 3: Productivity with sqlc**

What you'll write instead:

SQL file (`queries/portfolio.sql`):
```sql
-- name: GetPortfolio :one
SELECT id, name, user_id, total_value
FROM portfolios
WHERE id = ?;

-- name: CreatePortfolio :exec
INSERT INTO portfolios (name, user_id)
VALUES (?, ?);

-- name: ListPortfolios :many
SELECT id, name, user_id, total_value
FROM portfolios
WHERE user_id = ?;
```

Generated Go code (automatic):
```go
// sqlc generates:
type Portfolio struct {
    ID         int64
    Name       string
    UserID     int64
    TotalValue float64
}

func (q *Queries) GetPortfolio(ctx context.Context, id int64) (Portfolio, error)
func (q *Queries) CreatePortfolio(ctx context.Context, arg CreatePortfolioParams) error
func (q *Queries) ListPortfolios(ctx context.Context, userID int64) ([]Portfolio, error)
```

Your service layer:
```go
func (s *Service) GetPortfolio(id int64) (*Portfolio, error) {
    // sqlc handles all the boilerplate
    p, err := s.queries.GetPortfolio(context.Background(), id)
    if err != nil {
        return nil, err
    }
    return &p, nil
}
```

**What you gained:**
- ✅ **Compile-time validation:** SQL is checked during build
- ✅ **Type safety:** Wrong types = compile error
- ✅ **Auto-generated code:** No manual Scan() calls
- ✅ **Schema awareness:** Knows your database structure
- ✅ **Less boilerplate:** 3 lines instead of 15

**What you appreciate (because you learned the hard way):**
- "Oh, sqlc generates all those Scan() calls I was writing!"
- "It validates SQL at compile time, catching my typos early!"
- "Schema changes automatically update the generated code!"

---

**Migration path documented:**

`docs/GO_IMPLEMENTATION_PLAN.md` includes:
- Complete before/after comparison
- Step-by-step migration guide
- How to install and configure sqlc
- How to use Atlas for migrations
- Example conversions from database/sql to sqlc

---

**Why this is optimal:**

1. **Foundation first:**
   - Understand what's happening under the hood
   - Learn Go's database patterns
   - Build muscle memory

2. **Tools second:**
   - Appreciate what they solve
   - Use them effectively
   - Debug them when needed

3. **Realistic for production:**
   - Phase 1-2: Learn (few endpoints)
   - Phase 3: Scale (many endpoints)
   - Real projects do this evolution

---

## Summary of Decision Philosophy

The common theme in all these decisions:

### **Learn Fundamentals First, Then Add Productivity Tools**

| Layer | Phase 1-2 (Learning) | Phase 3 (Productivity) |
|-------|---------------------|------------------------|
| **Web Framework** | Chi (stdlib-like) | Chi (same) |
| **Database Driver** | modernc.org/sqlite (pure Go) | Same |
| **Query Layer** | database/sql (manual) | sqlc (generated) |
| **Migrations** | Manual SQL | Atlas (automated) |
| **Error Handling** | Explicit checks | Explicit checks (same) |

### **Why This Matters**

**You're not just building a project, you're learning Go.**

The choices prioritize:
1. **Understanding** over magic
2. **Explicit** over implicit
3. **Standard library** over frameworks
4. **Build up** rather than start abstracted

When you finish Phase 3, you'll:
- ✅ Understand how web servers work in Go
- ✅ Know database/sql patterns (used in all projects)
- ✅ Appreciate why tools like sqlc exist
- ✅ Be able to make informed decisions on future projects
- ✅ Have working knowledge of production-ready tools

### **Comparison to "Fast Start" Approach**

If we'd chosen Gin + GORM from day 1:
- ❌ Wouldn't learn stdlib http package
- ❌ Wouldn't understand database/sql
- ❌ Would be locked into framework patterns
- ❌ Harder to debug or optimize later
- ✅ Faster initial development

**Our approach:**
- ✅ Transferable skills (works with any Go project)
- ✅ Deep understanding
- ✅ Smooth transition to productivity tools
- ✅ Can work on any Go codebase
- ❌ Slightly slower initial development

**The trade-off is worth it for a learning project.**

---

## Questions to Guide Future Decisions

As you continue implementing, ask:

1. **Am I learning something valuable?**
   - If a tool hides too much, maybe do it manually first

2. **Is this pattern standard Go?**
   - Check Go blog, standard library, popular projects

3. **Will this scale to 72 endpoints?**
   - Early decisions should anticipate growth

4. **Can I explain how this works?**
   - If not, maybe it's too magical

5. **What's the migration path?**
   - Don't get locked in without an escape hatch

---

## Additional Resources

### Learning Why These Patterns Exist

- **Go Proverbs:** https://go-proverbs.github.io/
  - "A little copying is better than a little dependency"
  - "Clear is better than clever"

- **Effective Go:** https://go.dev/doc/effective_go
  - Official guide to Go idioms

- **Standard Library as Example:**
  - Read `net/http` source code
  - See how `database/sql` is designed

### Architecture Patterns

- **Standard Go Project Layout:** https://github.com/golang-standards/project-layout
- **Clean Architecture in Go:** https://github.com/bxcodec/go-clean-arch
- **Practical Go:** https://dave.cheney.net/practical-go

### Understanding the Tools

- **Chi Router:** https://github.com/go-chi/chi
- **sqlc:** https://docs.sqlc.dev/
- **Atlas:** https://atlasgo.io/getting-started

---

## 10. PortfolioFundRepository — Separate Join Table Repository

### Context

The `portfolio_fund` table is a join table linking portfolios to funds. Methods for querying it were initially split between `fund_repository.go` (e.g. `GetPortfolioFunds`, `InsertPortfolioFund`, `CheckUsage`) and `portfolio_repository.go` (e.g. `GetPortfolioFundsOnPortfolioID`). This worked early on but caused problems as the codebase grew:

- Service constructors had to accept both `FundRepository` and `PortfolioRepository` even when only portfolio_fund queries were needed.
- The `DividendService` ended up with a `findPortfolioFund()` helper that fetched **all** portfolio fund listings and looped to find one by ID — purely because there was no single-record lookup function that fit without creating yet another near-duplicate SQL variant.
- The service layer was directly invoking `fundRepo.GetPortfolioFund()` for what is conceptually a join-table operation, blurring the entity boundary.

### Decision

**Extract all `portfolio_fund` operations into a dedicated `PortfolioFundRepository`** in `internal/repository/portfolio_fund_repository.go`.

Methods consolidated:
- `GetPortfolioFund(id)` — raw join table row
- `GetPortfolioFundListing(id)` — single enriched listing (no archive filter — ID lookup must work regardless of archive status)
- `GetAllPortfolioFundListings()` — all active (non-archived) enriched listings
- `GetPortfolioFunds(fundID)` — all funds in a portfolio by fund ID
- `GetPortfolioFundsbyFundID(fundID)` — portfolios using a given fund
- `GetPortfolioFundsOnPortfolioID(portfolioID)` — portfolio_fund rows for a portfolio
- `CheckUsage(fundID)` — usage check before deletion
- `InsertPortfolioFund(req, tx)` — insert
- `DeletePortfolioFund(id, tx)` — delete

The `findPortfolioFund()` antipattern in `DividendService` was deleted and replaced with a direct call to `pfRepo.GetPortfolioFundListing(id)`.

### Alternatives Considered

#### A. Keep methods in FundRepository and PortfolioRepository

**Pros:**
- No change required; less churn during active endpoint development.

**Cons:**
- Ownership is ambiguous — `portfolio_fund` is neither a fund nor a portfolio.
- Service constructors were already being modified frequently; keeping the split would continue forcing double-repository injection.
- The `findPortfolioFund` antipattern would need to be replicated into IBKR and developer services as they were built.

#### B. Wait until sqlc migration

**Pros:**
- sqlc will generate typed query functions anyway; the repository struct is boilerplate that could be skipped.

**Cons:**
- sqlc migration is planned only after all ~72 endpoints are complete. With ~25% remaining (IBKR + developer), leaving the antipattern in place would let it spread into new code.
- Constructor churn was already ongoing; deferring would not reduce it.

#### C. Add a PortfolioFundService

**Pros:**
- Consistent with other domain objects that have both a repo and a service.

**Cons:**
- `portfolio_fund` has no business logic — it is purely a join table. All validation and orchestration lives in the consuming services. A service layer here would be empty indirection.

### Rationale

A join table that is queried from multiple services, inserted and deleted transactionally, and used in listing endpoints is its own data entity. Grouping its queries in one file makes ownership clear, removes the double-injection from constructors, and enables the single-record `GetPortfolioFundListing` lookup that eliminates the antipattern without creating another SQL variant.

The timing was right because constructor signatures were in flux anyway (recently added `*sql.DB` and `*sql.Tx` for transaction orchestration), so the migration cost was low and the benefit — preventing the antipattern from reaching new IBKR/developer code — was immediate.

### Trade-offs

**Pros:**
- Clear entity ownership: all `portfolio_fund` SQL lives in one file.
- Eliminates the `findPortfolioFund` antipattern (fetch-all + loop).
- Service constructors only inject `pfRepo` instead of both `fundRepo` and `portfolioRepo` where only join-table queries are needed.
- Blocks the antipattern from spreading into remaining endpoints.

**Cons:**
- Constructor updates were required across all affected services and test helpers — a one-time migration cost.
- Adds another file to the repository layer, increasing total file count.
- Will be partially superseded by sqlc-generated code, but the conceptual separation will remain valid.

---

Happy learning! The decisions documented here will make more sense as you implement more endpoints. Revisit this document when you're ready for Phase 3 migration.
