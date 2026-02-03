# Go Backend Framework Implementation Plan
## Investment Portfolio Manager - Backend Rewrite in Go

---

## Overview

Transform the Python/Flask backend into a Go-based backend that maintains API compatibility with the existing React frontend and works with the existing SQLite database at `Investment-Portfolio-Manager-Backend/data/portfolio_manager.db`.

**Tech Stack:**
- **Web Framework**: Chi router (stdlib-compatible, middleware ecosystem)
- **Database (Phase 1-2)**: modernc.org/sqlite + database/sql (learn fundamentals)
- **Database (Phase 3+)**: modernc.org/sqlite + sqlc + Atlas (modern, production-ready)
- **API Documentation**: Swaggo (OpenAPI/Swagger generation)
- **Testing**: Go testing + testify/assert
- **Logging**: Structured logging with levels and categories

**Development Approach:**
1. **Phase 1-2**: Start with database/sql (health check + basic CRUD) - Learn raw patterns
2. **Phase 3**: Migrate to sqlc + Atlas - Modern type-safe code generation
3. Incrementally add functionality to reach feature parity
4. Follow Python backend's development principles (service layer, comprehensive testing, structured logging)
5. Maintain API compatibility for seamless frontend integration

**Why the Hybrid Approach?**
- Start with database/sql to understand what happens under the hood
- Appreciate the boilerplate that sqlc eliminates
- See both approaches in practice (great learning experience)
- Migrate to sqlc + Atlas for production-ready, type-safe code generation

---

## Project Structure

```
Investment-Portfolio-Manager-Backend/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/                  # HTTP handlers (7 namespaces)
│   │   │   ├── system.go             # Health, version endpoints
│   │   │   ├── portfolio.go          # Portfolio CRUD
│   │   │   ├── fund.go               # Fund CRUD
│   │   │   ├── transaction.go        # Transaction CRUD
│   │   │   ├── dividend.go           # Dividend CRUD
│   │   │   ├── ibkr.go               # IBKR integration
│   │   │   └── developer.go          # Developer utilities
│   │   ├── middleware/                # HTTP middleware
│   │   │   ├── logger.go             # Request/response logging
│   │   │   ├── cors.go               # CORS configuration
│   │   │   ├── recovery.go           # Panic recovery
│   │   │   └── auth.go               # API key authentication
│   │   ├── router.go                  # Chi router setup
│   │   └── response.go                # Standard response helpers
│   ├── service/                       # Business logic layer
│   │   ├── portfolio_service.go
│   │   ├── fund_service.go
│   │   ├── transaction_service.go
│   │   ├── dividend_service.go
│   │   ├── system_service.go
│   │   ├── logging_service.go
│   │   └── ibkr_flex_service.go
│   ├── repository/                    # Database access layer
│   │   ├── portfolio_repository.go
│   │   ├── fund_repository.go
│   │   ├── transaction_repository.go
│   │   ├── dividend_repository.go
│   │   └── log_repository.go
│   ├── model/                         # Data models (structs)
│   │   ├── portfolio.go
│   │   ├── fund.go
│   │   ├── transaction.go
│   │   ├── dividend.go
│   │   ├── log.go
│   │   └── enums.go                   # Enum type definitions
│   ├── database/                      # Database management
│   │   ├── database.go                # Connection, setup
│   │   ├── migrations.go              # Migration runner
│   │   └── migrations/                # SQL migration files
│   │       ├── 001_initial_schema.sql
│   │       └── 002_add_indexes.sql
│   ├── util/                          # Utility functions
│   │   ├── uuid.go                    # UUID generation
│   │   ├── validation.go              # Input validation
│   │   └── date.go                    # Date utilities
│   └── config/
│       └── config.go                  # Configuration management
├── pkg/                               # Public packages (if needed)
├── tests/
│   ├── integration/                   # Integration tests
│   └── fixtures/                      # Test fixtures
├── data/                              # Database location
│   └── portfolio_manager.db           # SQLite database
├── docs/                              # Go-specific documentation
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── DEVELOPMENT.md
├── go.mod                             # Go module definition
├── go.sum                             # Dependency checksums
├── Makefile                           # Build commands
├── .env.example                       # Environment variables template
└── README.md                          # Project documentation
```

---

## Phase 1: Project Bootstrap & Health Check

### 1.1 Initialize Go Module

**File:** `Investment-Portfolio-Manager-Backend/go.mod`

```bash
cd Investment-Portfolio-Manager-Backend
go mod init github.com/ndewijer/Investment-Portfolio-Manager-Backend
```

**Dependencies to add:**
```bash
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get modernc.org/sqlite
go get github.com/joho/godotenv              # .env file support
go get github.com/swaggo/swag/cmd/swag       # Swagger generation
go get github.com/swaggo/http-swagger        # Swagger UI
go get github.com/stretchr/testify           # Testing assertions
go get github.com/google/uuid                # UUID generation
```

### 1.2 Configuration Management

**File:** `internal/config/config.go`

- Load environment variables from `.env`
- Define configuration struct:
  - Server port (default: 5001)
  - Database path
  - CORS origins
  - API keys (INTERNAL_API_KEY, IBKR_ENCRYPTION_KEY)
  - Logging configuration

**Pattern:**
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    CORS     CORSConfig
    APIKeys  APIKeyConfig
    Logging  LoggingConfig
}

func Load() (*Config, error) {
    // Load .env file
    // Parse environment variables
    // Set defaults
    // Validate required fields
}
```

### 1.3 Database Connection

**File:** `internal/database/database.go`

- Open SQLite connection using modernc.org/sqlite
- Enable foreign keys: `PRAGMA foreign_keys = ON`
- Set timezone: `PRAGMA timezone = 'UTC'`
- Configure connection pool
- Implement health check query: `SELECT 1`

**Pattern:**
```go
func Open(dbPath string) (*sql.DB, error) {
    db, err := sql.Open("sqlite", dbPath)
    // Configure connection
    // Execute pragmas
    // Test connection
    return db, err
}

func HealthCheck(db *sql.DB) error {
    return db.Ping()
}
```

### 1.4 HTTP Server Setup

**File:** `internal/api/router.go`

- Create Chi router
- Mount middleware:
  - Request logging
  - CORS
  - Panic recovery
  - Request ID
- Define namespace sub-routers:
  - `/api/system` - System handlers
  - `/api/portfolio` - Portfolio handlers
  - `/api/fund` - Fund handlers
  - `/api/transaction` - Transaction handlers
  - `/api/dividend` - Dividend handlers
  - `/api/ibkr` - IBKR handlers
  - `/api/developer` - Developer handlers
- Mount Swagger UI at `/api/docs`

**Pattern:**
```go
func NewRouter(services *service.Container) http.Handler {
    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(cors.Handler(corsOptions))

    // API routes
    r.Route("/api", func(r chi.Router) {
        // System namespace
        r.Route("/system", func(r chi.Router) {
            h := handlers.NewSystemHandler(services.SystemService)
            r.Get("/health", h.Health)
            r.Get("/version", h.Version)
        })

        // Other namespaces...
    })

    return r
}
```

### 1.5 System Handler - Health Check

**File:** `internal/api/handlers/system.go`

Implement health check endpoint:
- **Method**: GET
- **Path**: `/api/system/health`
- **Response** (200 OK):
  ```json
  {
    "status": "healthy",
    "database": "connected"
  }
  ```
- **Response** (503 Service Unavailable):
  ```json
  {
    "status": "unhealthy",
    "database": "disconnected",
    "error": "connection refused"
  }
  ```

**Pattern:**
```go
// @Summary Health check
// @Description Check system health and database connectivity
// @Tags system
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} ErrorResponse
// @Router /api/system/health [get]
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
    if err := h.systemService.CheckHealth(); err != nil {
        RespondError(w, 503, "unhealthy", err.Error())
        return
    }

    RespondJSON(w, 200, map[string]string{
        "status": "healthy",
        "database": "connected",
    })
}
```

### 1.6 Application Entry Point

**File:** `cmd/server/main.go`

- Load configuration
- Initialize database connection
- Create service container
- Create router
- Start HTTP server
- Graceful shutdown handling

**Pattern:**
```go
func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Open database
    db, err := database.Open(cfg.Database.Path)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create services
    services := service.NewContainer(db, cfg)

    // Create router
    router := api.NewRouter(services)

    // Start server
    server := &http.Server{
        Addr:    cfg.Server.Addr,
        Handler: router,
    }

    // Graceful shutdown
    // ...
}
```

### 1.7 Makefile for Common Commands

**File:** `Makefile`

```makefile
.PHONY: run test build swagger clean

run:
	go run cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

coverage:
	go tool cover -html=coverage.out

build:
	go build -o bin/server cmd/server/main.go

swagger:
	swag init -g cmd/server/main.go -o docs/swagger

clean:
	rm -rf bin/ coverage.out docs/swagger/

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy
```

---

## Phase 2: Core Infrastructure

### 2.1 Structured Logging Service

**File:** `internal/service/logging_service.go`

Replicate Python's LoggingService:
- Log levels: DEBUG, INFO, WARNING, ERROR, CRITICAL
- Log categories: SYSTEM, PORTFOLIO, FUND, TRANSACTION, DIVIDEND, IBKR, DEVELOPER, SECURITY, DATABASE
- Dual logging: Database + file fallback
- Structured data in details field (JSON)

**Pattern:**
```go
type LogLevel string
type LogCategory string

type LoggingService struct {
    db         *sql.DB
    fileLogger *log.Logger
}

func (s *LoggingService) Log(level LogLevel, category LogCategory, message string, details map[string]interface{}) {
    logEntry := &model.Log{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Level:     level,
        Category:  category,
        Message:   message,
        Details:   toJSON(details),
    }

    // Try database first
    if err := s.saveToDatabase(logEntry); err != nil {
        // Fallback to file
        s.fileLogger.Printf("[%s] [%s] %s - %v", level, category, message, details)
    }
}
```

### 2.2 Response Helpers

**File:** `internal/api/response.go`

Standard response functions:
- `RespondJSON(w, status, data)` - Success responses
- `RespondError(w, status, message, details)` - Error responses
- `RespondCreated(w, data)` - 201 Created
- `RespondNoContent(w)` - 204 No Content

**Error response format:**
```go
type ErrorResponse struct {
    Error   string      `json:"error"`
    Details interface{} `json:"details,omitempty"`
}
```

### 2.3 Middleware Layer

**Files:**
- `internal/api/middleware/logger.go` - Request/response logging with duration
- `internal/api/middleware/cors.go` - CORS configuration for frontend
- `internal/api/middleware/recovery.go` - Panic recovery
- `internal/api/middleware/auth.go` - API key validation (for protected endpoints)

### 2.4 Models Definition

**Files in `internal/model/`:**

Define structs matching database schema:
- `portfolio.go` - Portfolio struct with JSON tags
- `fund.go` - Fund struct with enums
- `transaction.go` - Transaction struct
- `dividend.go` - Dividend struct
- `log.go` - Log struct
- `enums.go` - All enum types (DividendType, InvestmentType, ReinvestmentStatus, etc.)

**Pattern:**
```go
type Portfolio struct {
    ID                  string    `json:"id" db:"id"`
    Name                string    `json:"name" db:"name"`
    Description         *string   `json:"description,omitempty" db:"description"`
    IsArchived          bool      `json:"is_archived" db:"is_archived"`
    ExcludeFromOverview bool      `json:"exclude_from_overview" db:"exclude_from_overview"`
}

type DividendType string
const (
    DividendTypeNone  DividendType = "none"
    DividendTypeCash  DividendType = "cash"
    DividendTypeStock DividendType = "stock"
)
```

---

## Phase 3: Migration to sqlc + Atlas

**⚠️ MIGRATION POINT: Switch from database/sql to sqlc + Atlas**

At this point, you'll have:
- Working health check endpoint
- Basic understanding of database/sql patterns
- Repository layer with manual SQL queries
- Appreciation for boilerplate code

**Why Migrate Now?**
1. You've seen raw database/sql in action
2. You understand what sqlc will generate for you
3. Ready for production-ready patterns
4. Will speed up remaining 60+ endpoints

### 3.1 Install sqlc and Atlas

```bash
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install Atlas
curl -sSf https://atlasgo.sh | sh
```

### 3.2 Create sqlc Configuration

**File:** `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "internal/database/queries/"
    schema: "internal/database/schema.sql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_db_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_empty_slices: true
```

### 3.3 Create Atlas Configuration

**File:** `atlas.hcl`

```hcl
env "local" {
  src = "file://internal/database/schema.sql"
  url = "sqlite://data/portfolio_manager.db"
  dev = "sqlite://file?mode=memory"

  migration {
    dir = "file://internal/database/migrations"
  }

  diff {
    skip {
      drop_schema = true
      drop_table  = true
    }
  }
}

env "test" {
  src = "file://internal/database/schema.sql"
  url = "sqlite://file::memory:?cache=shared"
  dev = "sqlite://file?mode=memory"
}
```

### 3.4 Convert Existing Schema to schema.sql

Extract your existing database schema:

```bash
# Dump existing schema from Python backend's database
sqlite3 ../Investment-Portfolio-Manager/backend/data/db/portfolio_manager.db .schema > internal/database/schema.sql
```

**File:** `internal/database/schema.sql`

```sql
-- Core tables
CREATE TABLE portfolio (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_archived INTEGER DEFAULT 0,
    exclude_from_overview INTEGER DEFAULT 0
);

CREATE TABLE fund (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    isin TEXT UNIQUE NOT NULL,
    symbol TEXT,
    currency TEXT NOT NULL,
    exchange TEXT NOT NULL,
    investment_type TEXT NOT NULL,
    dividend_type TEXT NOT NULL
);

-- Add all other tables from existing schema...
```

### 3.5 Write SQL Queries for sqlc

**File:** `internal/database/queries/portfolio.sql`

```sql
-- name: GetPortfolio :one
SELECT * FROM portfolio
WHERE id = ? LIMIT 1;

-- name: ListPortfolios :many
SELECT * FROM portfolio
ORDER BY name;

-- name: ListActivePortfolios :many
SELECT * FROM portfolio
WHERE is_archived = 0
ORDER BY name;

-- name: CreatePortfolio :exec
INSERT INTO portfolio (
    id, name, description, is_archived, exclude_from_overview
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: UpdatePortfolio :exec
UPDATE portfolio
SET name = ?,
    description = ?,
    is_archived = ?,
    exclude_from_overview = ?
WHERE id = ?;

-- name: DeletePortfolio :exec
DELETE FROM portfolio
WHERE id = ?;

-- name: ArchivePortfolio :exec
UPDATE portfolio
SET is_archived = 1
WHERE id = ?;

-- name: UnarchivePortfolio :exec
UPDATE portfolio
SET is_archived = 0
WHERE id = ?;
```

**File:** `internal/database/queries/fund.sql`

```sql
-- name: GetFund :one
SELECT * FROM fund
WHERE id = ? LIMIT 1;

-- name: GetFundByISIN :one
SELECT * FROM fund
WHERE isin = ? LIMIT 1;

-- name: ListFunds :many
SELECT * FROM fund
ORDER BY name;

-- name: CreateFund :exec
INSERT INTO fund (
    id, name, isin, symbol, currency, exchange, investment_type, dividend_type
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateFund :exec
UPDATE fund
SET name = ?,
    symbol = ?,
    currency = ?,
    exchange = ?,
    investment_type = ?,
    dividend_type = ?
WHERE id = ?;

-- name: DeleteFund :exec
DELETE FROM fund
WHERE id = ?;
```

**Continue for all other tables...**

### 3.6 Generate Code with sqlc

```bash
sqlc generate
```

This creates `internal/db/` with:
- `models.go` - All struct definitions
- `db.go` - Database connection interface
- `querier.go` - Query interface
- `portfolio.sql.go` - Generated portfolio queries
- `fund.sql.go` - Generated fund queries
- etc.

### 3.7 Update Repository Layer to Use Generated Code

**Before (manual database/sql):**

```go
func (r *PortfolioRepository) GetByID(id string) (*model.Portfolio, error) {
    query := `SELECT id, name, description, is_archived, exclude_from_overview
              FROM portfolio WHERE id = ?`
    var p model.Portfolio
    err := r.db.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Description,
                                           &p.IsArchived, &p.ExcludeFromOverview)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &p, err
}
```

**After (sqlc generated):**

```go
type PortfolioRepository struct {
    queries *db.Queries
}

func NewPortfolioRepository(database *sql.DB) *PortfolioRepository {
    return &PortfolioRepository{
        queries: db.New(database),
    }
}

func (r *PortfolioRepository) GetByID(ctx context.Context, id string) (*db.Portfolio, error) {
    portfolio, err := r.queries.GetPortfolio(ctx, id)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &portfolio, err
}
```

**Benefits:**
- ✅ No manual Scan() code
- ✅ Type-safe at compile time
- ✅ Auto-generated from SQL
- ✅ Less boilerplate
- ✅ Easier to maintain

### 3.8 Create Initial Migration with Atlas

```bash
# Generate migration from schema.sql to current database
atlas migrate diff initial \
  --env local \
  --to file://internal/database/schema.sql

# Apply migration
atlas migrate apply --env local
```

### 3.9 Future Schema Changes with Atlas

**Workflow:**

1. **Modify `schema.sql`** - Add/change tables
2. **Generate migration**:
   ```bash
   atlas migrate diff add_new_feature --env local
   ```
3. **Review generated SQL** in `internal/database/migrations/`
4. **Lint migration** (checks for destructive changes):
   ```bash
   atlas migrate lint --env local
   ```
5. **Apply migration**:
   ```bash
   atlas migrate apply --env local
   ```

**Example: Adding a new column**

```sql
-- schema.sql
ALTER TABLE portfolio ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
```

```bash
atlas migrate diff add_created_at --env local
```

Atlas generates:
```sql
-- 20240115_add_created_at.sql
ALTER TABLE portfolio ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
```

### 3.10 Update Makefile

```makefile
# Add to Makefile
.PHONY: sqlc-generate atlas-diff atlas-apply

sqlc-generate:
	sqlc generate

atlas-diff:
	atlas migrate diff $(name) --env local

atlas-apply:
	atlas migrate apply --env local

atlas-lint:
	atlas migrate lint --env local

db-generate: sqlc-generate
	@echo "Code generated from SQL"
```

### 3.11 Update Project Structure

```
Investment-Portfolio-Manager-Backend/
├── internal/
│   ├── db/                            # ✨ NEW: sqlc generated code
│   │   ├── models.go                  # Generated structs
│   │   ├── db.go                      # DB interface
│   │   ├── querier.go                 # Query interface
│   │   ├── portfolio.sql.go           # Generated portfolio queries
│   │   └── fund.sql.go                # Generated fund queries
│   ├── database/
│   │   ├── schema.sql                 # ✨ NEW: Single source of truth
│   │   ├── queries/                   # ✨ NEW: SQL query files
│   │   │   ├── portfolio.sql
│   │   │   ├── fund.sql
│   │   │   └── transaction.sql
│   │   ├── migrations/                # ✨ NEW: Atlas migrations
│   │   │   ├── 20240115_initial.sql
│   │   │   └── atlas.sum
│   │   └── database.go                # Connection management
│   ├── repository/                    # ✨ UPDATED: Thin wrappers around sqlc
│   │   ├── portfolio_repository.go
│   │   └── fund_repository.go
│   └── model/                         # ⚠️ DEPRECATED: Use internal/db instead
├── sqlc.yaml                          # ✨ NEW: sqlc configuration
└── atlas.hcl                          # ✨ NEW: Atlas configuration
```

### 3.12 Migration Checklist

- [ ] Install sqlc and Atlas
- [ ] Create `sqlc.yaml` configuration
- [ ] Create `atlas.hcl` configuration
- [ ] Extract schema to `schema.sql`
- [ ] Write SQL queries in `queries/*.sql`
- [ ] Run `sqlc generate`
- [ ] Update repositories to use generated code
- [ ] Update tests to use new code
- [ ] Generate initial Atlas migration
- [ ] Update documentation
- [ ] Remove old `internal/model/` package
- [ ] Update Makefile with new commands

**After Migration:**
- All new queries written in `.sql` files
- Run `make sqlc-generate` after query changes
- Use Atlas for all schema changes
- Type-safe database access throughout

---

## Phase 4: Repository Pattern (Post-Migration)

### 4.1 Repository Interface Pattern (Using sqlc)

**After sqlc migration**, repositories become thin wrappers around generated code:

**File:** `internal/repository/portfolio_repository.go`

```go
type PortfolioRepository struct {
    queries *db.Queries
}

func NewPortfolioRepository(database *sql.DB) *PortfolioRepository {
    return &PortfolioRepository{
        queries: db.New(database),
    }
}

func (r *PortfolioRepository) Create(ctx context.Context, portfolio *db.CreatePortfolioParams) error {
    return r.queries.CreatePortfolio(ctx, portfolio)
}

func (r *PortfolioRepository) GetByID(ctx context.Context, id string) (*db.Portfolio, error) {
    portfolio, err := r.queries.GetPortfolio(ctx, id)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &portfolio, err
}

func (r *PortfolioRepository) GetAll(ctx context.Context) ([]db.Portfolio, error) {
    return r.queries.ListPortfolios(ctx)
}

func (r *PortfolioRepository) Update(ctx context.Context, params db.UpdatePortfolioParams) error {
    return r.queries.UpdatePortfolio(ctx, params)
}

func (r *PortfolioRepository) Delete(ctx context.Context, id string) error {
    return r.queries.DeletePortfolio(ctx, id)
}

// Complex queries - add to queries/portfolio.sql
func (r *PortfolioRepository) GetWithFunds(ctx context.Context, id string) (*PortfolioWithFunds, error) {
    // Use a JOIN query defined in portfolio.sql
    // sqlc generates the scanning code
}
```

**Key Differences from Manual database/sql:**
- ✅ No manual Scan() code
- ✅ sqlc generates parameter structs
- ✅ Type-safe at compile time
- ✅ Still separates database access from business logic
- ✅ Complex queries defined in SQL files, not Go code

**Repository Pattern Benefits (Still Apply):**
- Separates SQL from business logic
- Easy to test with mock repositories
- Centralized query optimization
- Follows Go best practices for database access

### 4.2 Transaction Management (Using sqlc)

**Pattern for multi-operation transactions with sqlc:**

```go
func (r *TransactionRepository) CreateWithGainLoss(ctx context.Context, tx *db.Transaction, gainLoss *db.RealizedGainLoss) error {
    // Begin transaction
    dbTx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer dbTx.Rollback()

    // Create queries with transaction
    qtx := r.queries.WithTx(dbTx)

    // Insert transaction (sqlc generated)
    err = qtx.CreateTransaction(ctx, db.CreateTransactionParams{
        ID:              tx.ID,
        PortfolioFundID: tx.PortfolioFundID,
        Date:            tx.Date,
        Type:            tx.Type,
        Shares:          tx.Shares,
        CostPerShare:    tx.CostPerShare,
    })
    if err != nil {
        return err
    }

    // Insert gain/loss record (sqlc generated)
    err = qtx.CreateRealizedGainLoss(ctx, db.CreateRealizedGainLossParams{
        ID:                gainLoss.ID,
        TransactionID:     tx.ID,
        SharesSold:        gainLoss.SharesSold,
        CostBasis:         gainLoss.CostBasis,
        SaleProceeds:      gainLoss.SaleProceeds,
        RealizedGainLoss:  gainLoss.RealizedGainLoss,
    })
    if err != nil {
        return err
    }

    // Commit transaction
    return dbTx.Commit()
}
```

**Original Pattern (manual database/sql):**

```go
func (r *TransactionRepository) CreateWithGainLoss(tx *model.Transaction, gainLoss *model.RealizedGainLoss) error {
    // Start transaction
    dbTx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer dbTx.Rollback() // Rollback if not committed

    // Insert transaction
    _, err = dbTx.Exec(`INSERT INTO transaction (...) VALUES (...)`, ...)
    if err != nil {
        return err
    }

    // Insert gain/loss record
    _, err = dbTx.Exec(`INSERT INTO realized_gain_loss (...) VALUES (...)`, ...)
    if err != nil {
        return err
    }

    // Commit transaction
    return dbTx.Commit()
}
```

### 4.3 Query Optimization Patterns (Using sqlc)

**Batch Loading (Avoid N+1) with sqlc:**

Write optimized JOIN queries in SQL files:

```sql
-- name: GetPortfoliosWithFunds :many
SELECT
    p.id as portfolio_id,
    p.name as portfolio_name,
    p.is_archived,
    pf.id as portfolio_fund_id,
    f.id as fund_id,
    f.name as fund_name,
    f.symbol as fund_symbol
FROM portfolio p
LEFT JOIN portfolio_fund pf ON p.id = pf.portfolio_id
LEFT JOIN fund f ON pf.fund_id = f.id
ORDER BY p.name, f.name;
```

sqlc generates the struct and scanning code automatically:

```go
func (r *PortfolioRepository) GetAllWithFunds(ctx context.Context) ([]PortfolioWithFunds, error) {
    rows, err := r.queries.GetPortfoliosWithFunds(ctx)
    if err != nil {
        return nil, err
    }

    // Group by portfolio (sqlc gives us flat rows)
    portfolioMap := make(map[string]*PortfolioWithFunds)
    for _, row := range rows {
        if _, exists := portfolioMap[row.PortfolioID]; !exists {
            portfolioMap[row.PortfolioID] = &PortfolioWithFunds{
                ID:   row.PortfolioID,
                Name: row.PortfolioName,
                Funds: []Fund{},
            }
        }
        if row.FundID.Valid {
            portfolioMap[row.PortfolioID].Funds = append(
                portfolioMap[row.PortfolioID].Funds,
                Fund{
                    ID:     row.FundID.String,
                    Name:   row.FundName.String,
                    Symbol: row.FundSymbol.String,
                },
            )
        }
    }

    // Convert map to slice
    result := make([]PortfolioWithFunds, 0, len(portfolioMap))
    for _, pwf := range portfolioMap {
        result = append(result, *pwf)
    }

    return result, nil
}
```

**Benefits:**
- ✅ sqlc generates the row scanning
- ✅ You control the SQL (optimization)
- ✅ Type-safe struct definitions
- ✅ No manual scanning errors

**Original Pattern (manual database/sql):**

```go
func (r *PortfolioRepository) GetAllWithFunds() ([]PortfolioWithFunds, error) {
    // Load all portfolios
    portfolios, err := r.GetAll()
    if err != nil {
        return nil, err
    }

    // Extract portfolio IDs
    portfolioIDs := make([]string, len(portfolios))
    for i, p := range portfolios {
        portfolioIDs[i] = p.ID
    }

    // Batch load all portfolio_fund relationships
    query := `SELECT pf.id, pf.portfolio_id, pf.fund_id, f.name, f.symbol
              FROM portfolio_fund pf
              JOIN fund f ON pf.fund_id = f.id
              WHERE pf.portfolio_id IN (` + placeholders(len(portfolioIDs)) + `)`

    rows, err := r.db.Query(query, toInterfaceSlice(portfolioIDs)...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Group by portfolio_id
    fundsMap := make(map[string][]model.Fund)
    for rows.Next() {
        var portfolioID string
        var fund model.Fund
        // Scan...
        fundsMap[portfolioID] = append(fundsMap[portfolioID], fund)
    }

    // Combine portfolios with funds
    result := make([]PortfolioWithFunds, len(portfolios))
    for i, p := range portfolios {
        result[i] = PortfolioWithFunds{
            Portfolio: p,
            Funds:     fundsMap[p.ID],
        }
    }

    return result, nil
}
```

---

## Phase 4: Service Layer (Business Logic)

### 4.1 Service Container

**File:** `internal/service/container.go`

Dependency injection container:

```go
type Container struct {
    DB               *sql.DB
    Logger           *LoggingService
    SystemService    *SystemService
    PortfolioService *PortfolioService
    FundService      *FundService
    // ... other services
}

func NewContainer(db *sql.DB, cfg *config.Config) *Container {
    logger := NewLoggingService(db, cfg.Logging)

    // Create repositories
    portfolioRepo := repository.NewPortfolioRepository(db)
    fundRepo := repository.NewFundRepository(db)
    // ... other repos

    return &Container{
        DB:               db,
        Logger:           logger,
        SystemService:    NewSystemService(db, logger),
        PortfolioService: NewPortfolioService(portfolioRepo, logger),
        FundService:      NewFundService(fundRepo, logger),
        // ... other services
    }
}
```

### 4.2 Service Implementation Pattern

**File:** `internal/service/portfolio_service.go`

```go
type PortfolioService struct {
    repo   *repository.PortfolioRepository
    logger *LoggingService
}

func NewPortfolioService(repo *repository.PortfolioRepository, logger *LoggingService) *PortfolioService {
    return &PortfolioService{
        repo:   repo,
        logger: logger,
    }
}

func (s *PortfolioService) GetPortfolio(id string) (*model.Portfolio, error) {
    portfolio, err := s.repo.GetByID(id)
    if err != nil {
        if err == repository.ErrNotFound {
            s.logger.Log(LogLevelWarning, LogCategoryPortfolio,
                "Portfolio not found", map[string]interface{}{"id": id})
            return nil, ErrNotFound
        }
        s.logger.Log(LogLevelError, LogCategoryPortfolio,
            "Error retrieving portfolio", map[string]interface{}{"id": id, "error": err.Error()})
        return nil, err
    }

    s.logger.Log(LogLevelInfo, LogCategoryPortfolio,
        "Portfolio retrieved", map[string]interface{}{"id": id, "name": portfolio.Name})

    return portfolio, nil
}

func (s *PortfolioService) CreatePortfolio(name, description string) (*model.Portfolio, error) {
    // Validation
    if name == "" {
        return nil, ErrInvalidInput("name is required")
    }

    portfolio := &model.Portfolio{
        ID:                  uuid.New().String(),
        Name:                name,
        Description:         &description,
        IsArchived:          false,
        ExcludeFromOverview: false,
    }

    if err := s.repo.Create(portfolio); err != nil {
        s.logger.Log(LogLevelError, LogCategoryPortfolio,
            "Error creating portfolio", map[string]interface{}{"error": err.Error()})
        return nil, err
    }

    s.logger.Log(LogLevelInfo, LogCategoryPortfolio,
        "Portfolio created", map[string]interface{}{"id": portfolio.ID, "name": name})

    return portfolio, nil
}

func (s *PortfolioService) DeletePortfolio(id string) error {
    // Check if portfolio exists
    _, err := s.repo.GetByID(id)
    if err != nil {
        return err
    }

    // TODO: Check if portfolio has funds/transactions
    // If yes, either prevent deletion or cascade

    if err := s.repo.Delete(id); err != nil {
        s.logger.Log(LogLevelError, LogCategoryPortfolio,
            "Error deleting portfolio", map[string]interface{}{"id": id, "error": err.Error()})
        return err
    }

    s.logger.Log(LogLevelInfo, LogCategoryPortfolio,
        "Portfolio deleted", map[string]interface{}{"id": id})

    return nil
}

// Complex business logic
func (s *PortfolioService) GetPortfolioSummary() ([]PortfolioSummary, error) {
    // Batch load portfolios with funds
    portfoliosWithFunds, err := s.repo.GetAllWithFunds()
    if err != nil {
        return nil, err
    }

    // Load all transactions in one query
    // Load all prices in one query
    // Calculate metrics in memory
    // Return aggregated summary
}
```

### 4.3 Error Handling Strategy

**File:** `internal/service/errors.go`

Define custom error types:

```go
var (
    ErrNotFound      = errors.New("resource not found")
    ErrInvalidInput  = errors.New("invalid input")
    ErrConflict      = errors.New("resource conflict")
    ErrUnauthorized  = errors.New("unauthorized")
)

// Helper to create detailed errors
func ErrInvalidInputWithField(field, message string) error {
    return fmt.Errorf("%w: %s - %s", ErrInvalidInput, field, message)
}
```

**In handlers, map errors to HTTP status codes:**

```go
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    portfolio, err := h.service.GetPortfolio(id)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrNotFound):
            RespondError(w, 404, "Portfolio not found", nil)
        case errors.Is(err, service.ErrInvalidInput):
            RespondError(w, 400, err.Error(), nil)
        default:
            RespondError(w, 500, "Internal server error", err.Error())
        }
        return
    }

    RespondJSON(w, 200, portfolio)
}
```

---

## Phase 5: Testing Strategy (Post-sqlc Migration)

**Note:** Testing becomes easier with sqlc because:
- Generated code is already tested by sqlc
- You only test business logic
- Mock the `Querier` interface sqlc generates

### 5.1 Repository Tests

**File:** `internal/repository/portfolio_repository_test.go`

- Use in-memory SQLite database (`:memory:`)
- Set up schema in `TestMain`
- Test each repository method
- Use testify/assert for assertions

**Pattern:**
```go
func TestPortfolioRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewPortfolioRepository(db)

    portfolio := &model.Portfolio{
        ID:   uuid.New().String(),
        Name: "Test Portfolio",
    }

    err := repo.Create(portfolio)
    assert.NoError(t, err)

    // Verify it was created
    retrieved, err := repo.GetByID(portfolio.ID)
    assert.NoError(t, err)
    assert.Equal(t, portfolio.Name, retrieved.Name)
}
```

### 5.2 Service Tests (Using sqlc Mocks)

**File:** `internal/service/portfolio_service_test.go`

- Mock the `db.Querier` interface (generated by sqlc with `emit_interface: true`)
- Test business logic without database
- Verify logging calls

**Pattern with sqlc:**
```go
type MockQuerier struct {
    mock.Mock
}

func (m *MockQuerier) GetPortfolio(ctx context.Context, id string) (db.Portfolio, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(db.Portfolio), args.Error(1)
}

func (m *MockQuerier) CreatePortfolio(ctx context.Context, params db.CreatePortfolioParams) error {
    args := m.Called(ctx, params)
    return args.Error(0)
}

func TestPortfolioService_GetPortfolio(t *testing.T) {
    mockRepo := new(MockPortfolioRepository)
    mockLogger := &LoggingService{} // Or mock this too

    service := NewPortfolioService(mockRepo, mockLogger)

    expectedPortfolio := &model.Portfolio{
        ID:   "test-id",
        Name: "Test Portfolio",
    }

    mockRepo.On("GetByID", "test-id").Return(expectedPortfolio, nil)

    result, err := service.GetPortfolio("test-id")

    assert.NoError(t, err)
    assert.Equal(t, expectedPortfolio.Name, result.Name)
    mockRepo.AssertExpectations(t)
}
```

### 5.3 Integration Tests

**File:** `tests/integration/health_test.go`

- Test full HTTP stack
- Use real database (test database)
- Verify HTTP responses

**Pattern:**
```go
func TestHealthCheckEndpoint(t *testing.T) {
    // Setup test server
    db := setupTestDB(t)
    services := service.NewContainer(db, testConfig)
    router := api.NewRouter(services)

    server := httptest.NewServer(router)
    defer server.Close()

    // Make request
    resp, err := http.Get(server.URL + "/api/system/health")
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)

    // Parse response
    var result map[string]string
    json.NewDecoder(resp.Body).Decode(&result)
    assert.Equal(t, "healthy", result["status"])
    assert.Equal(t, "connected", result["database"])
}
```

### 5.4 Test Coverage Goals

Match Python backend standards:
- **Overall**: 95%+ coverage
- **Repository layer**: 90%+ (all CRUD operations)
- **Service layer**: 95%+ (business logic)
- **Handler layer**: 85%+ (HTTP edge cases)

**Run coverage:**
```bash
make test
make coverage  # Opens HTML coverage report
```

---

## Phase 6: Swagger/OpenAPI Documentation

### 6.1 Swaggo Setup

**File:** `cmd/server/main.go` (add Swagger metadata)

```go
// @title Investment Portfolio Manager API
// @version 1.0
// @description REST API for managing investment portfolios
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:5001
// @BasePath /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
    // ... application code
}
```

### 6.2 Handler Annotations

Add Swagger comments to each handler:

```go
// @Summary Get portfolio by ID
// @Description Retrieve a single portfolio by its UUID
// @Tags portfolios
// @Accept json
// @Produce json
// @Param id path string true "Portfolio ID (UUID)"
// @Success 200 {object} model.Portfolio
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /portfolio/{id} [get]
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### 6.3 Generate Swagger Spec

```bash
make swagger
```

This generates `docs/swagger/swagger.json` and `docs/swagger/swagger.yaml`

### 6.4 Serve Swagger UI

Mount Swagger UI in router:

```go
import httpSwagger "github.com/swaggo/http-swagger"
import _ "github.com/ndewijer/Investment-Portfolio-Manager-Backend/docs/swagger" // Import generated docs

func NewRouter(services *service.Container) http.Handler {
    r := chi.NewRouter()

    // ... other routes

    // Swagger UI
    r.Get("/api/docs/*", httpSwagger.Handler(
        httpSwagger.URL("/api/docs/swagger.json"),
    ))

    return r
}
```

Access Swagger UI at: `http://localhost:5001/api/docs/index.html`

---

## Phase 7: Development Workflow

### 7.1 Environment Setup

**File:** `.env.example`

```env
# Server Configuration
SERVER_PORT=5001
SERVER_HOST=localhost

# Database
DB_PATH=./data/portfolio_manager.db

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# API Keys
INTERNAL_API_KEY=your-api-key-here
IBKR_ENCRYPTION_KEY=your-encryption-key-here

# Logging
LOG_LEVEL=info
LOG_TO_FILE=true
LOG_FILE_PATH=./logs/app.log
```

Copy to `.env` and customize:
```bash
cp .env.example .env
```

### 7.2 Development Commands

```bash
# Install dependencies
make deps

# Run development server (with hot reload using air)
make run

# Run tests
make test

# Run tests with coverage
make coverage

# Generate Swagger docs
make swagger

# Build binary
make build

# Run linter
make lint

# Format code
go fmt ./...
```

### 7.3 Database Migrations

**Initial approach**: Use the existing SQLite database

**Future migrations**: Create SQL migration files in `internal/database/migrations/`

**Migration runner pattern:**
```go
func RunMigrations(db *sql.DB) error {
    migrations := []string{
        "001_initial_schema.sql",
        "002_add_indexes.sql",
    }

    for _, migration := range migrations {
        sql, err := os.ReadFile("internal/database/migrations/" + migration)
        if err != nil {
            return err
        }

        if _, err := db.Exec(string(sql)); err != nil {
            return fmt.Errorf("migration %s failed: %w", migration, err)
        }
    }

    return nil
}
```

---

## Phase 8: Incremental Feature Implementation

### Priority Order (after health check):

1. **System Namespace** (2 endpoints)
   - ✅ GET /api/system/health (Phase 1)
   - GET /api/system/version

2. **Portfolio Namespace** (13 endpoints)
   - GET /api/portfolio (list all)
   - POST /api/portfolio (create)
   - GET /api/portfolio/{id} (get one)
   - PUT /api/portfolio/{id} (update)
   - DELETE /api/portfolio/{id} (delete)
   - POST /api/portfolio/{id}/archive
   - POST /api/portfolio/{id}/unarchive
   - GET /api/portfolio-summary
   - GET /api/portfolio-history
   - GET /api/portfolio/{id}/fund-history
   - GET /api/portfolio-funds
   - POST /api/portfolio-funds
   - DELETE /api/portfolio-funds/{id}

3. **Fund Namespace** (13 endpoints)
4. **Transaction Namespace** (5 endpoints)
5. **Dividend Namespace** (6 endpoints)
6. **Developer Namespace** (15 endpoints)
7. **IBKR Namespace** (19 endpoints) - Most complex, implement last

### Implementation Pattern for Each Endpoint:

1. Define handler function with Swagger annotations
2. Extract and validate input
3. Call service layer
4. Handle errors (map to HTTP status codes)
5. Return response
6. Write tests (repository, service, integration)
7. Verify with frontend

---

## Phase 9: Documentation

### 9.1 Go-Specific Documentation

**File:** `docs/GO_DEVELOPMENT.md`

- Project structure explanation
- How to add new endpoints
- Testing conventions
- Database patterns
- Error handling
- Logging best practices

### 9.2 Architecture Documentation

**File:** `docs/ARCHITECTURE.md`

- System architecture diagram
- Layer separation (handler → service → repository)
- Data flow
- Dependency injection pattern
- Middleware stack

### 9.3 API Documentation

- Auto-generated via Swagger at `/api/docs`
- Keep Python backend's `API_DOCUMENTATION.md` as reference

---

## Key Implementation Principles

### From Python Backend Documentation:

1. **Service Layer Pattern**
   - Handlers are thin (HTTP concerns only)
   - Services contain business logic
   - Repositories handle database access

2. **Comprehensive Testing**
   - Unit tests for services and repositories
   - Integration tests for HTTP endpoints
   - 95%+ coverage goal

3. **Structured Logging**
   - All operations logged with category and level
   - Dual logging (database + file fallback)
   - JSON details for structured data

4. **Error Handling**
   - Custom error types
   - Proper HTTP status codes
   - Detailed error messages in logs, generic to clients

5. **API Compatibility**
   - Maintain exact same endpoints as Python backend
   - Same request/response formats
   - Same error response structure

6. **Security**
   - API key authentication for protected endpoints
   - CORS configuration for frontend
   - Input validation
   - SQL injection prevention (parameterized queries)

---

## Success Criteria

### Phase 1 Complete:
- ✅ Health check endpoint returns 200 OK
- ✅ Frontend can connect to `/api/system/health`
- ✅ Database connection verified
- ✅ Structured logging working
- ✅ Tests pass with >90% coverage

### Feature Parity Complete:
- ✅ All 72 endpoints implemented
- ✅ Frontend works without modifications
- ✅ All tests pass (95%+ coverage)
- ✅ Swagger documentation complete
- ✅ Performance meets or exceeds Python backend

---

## Development Timeline Approach

**Incremental Development:**
1. Get health check working first ✓
2. Add one namespace at a time
3. Test with frontend after each namespace
4. Refactor and optimize as you learn

**Learning Opportunities:**
- database/sql patterns and best practices
- Go error handling with errors.Is/As
- Go project structure and conventions
- Chi router and middleware patterns
- Testing in Go with testify
- Swagger documentation generation

---

## Notes

- Use existing database at `Investment-Portfolio-Manager-Backend/data/portfolio_manager.db`
- Keep frontend pointing to same API endpoints
- Python backend can run alongside Go backend on different port for comparison
- Use this as learning project - prioritize understanding over speed
- Follow Go idioms: simple, explicit, readable code
- **Phase 1-2**: Leverage database/sql for deeper understanding of database operations
- **Phase 3+**: Use sqlc + Atlas for type-safe, production-ready code

---

## 📋 Next Steps

**Copy this plan to the project:**

```bash
cp ~/.claude/plans/abstract-wibbling-brook.md Investment-Portfolio-Manager-Backend/docs/GO_IMPLEMENTATION_PLAN.md
```

This will serve as your living implementation guide that you can update as you progress through the development phases.
