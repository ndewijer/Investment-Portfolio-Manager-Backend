# DRAFT: Database Migrations & Auto-Initialization Issue

> **Status**: Draft for review and iteration
> **Date**: 2026-02-06 (Updated)
> **Notes**: Save as GitHub issue once finalized
> **Target Version**: v1.5.0 (feature parity with Python backend)

---

## Problem

Currently:
- No migration system (can't evolve schema over time)
- No auto-initialization (users must manually create database)
- Python backend will be decommissioned (can't rely on it)
- Need Docker-friendly deployment approach
- Lots of boilerplate scanning code (no sqlc yet)

## Goal

Implement a complete database management system for **v1.5.0 release**:
1. **Migration system** (Goose - track schema changes)
2. **Type-safe queries** (sqlc - reduce boilerplate)
3. **Auto-initialize** database on first run
4. **Config-driven** behavior (production vs development)
5. **Docker-first** approach (no CLI dependency)

## Proposed Solution: Goose + sqlc

### Migration Management: Goose
- ✅ Simple SQL-based migrations
- ✅ Version tracking (knows what's applied)
- ✅ Embedded migrations (bundle in binary)
- ✅ Up/Down migrations (rollback support)
- ✅ Works well with SQLite

### Query Generation: sqlc
- ✅ Type-safe Go code from SQL
- ✅ Compile-time query validation
- ✅ No manual scanning boilerplate
- ✅ Schema-aware code generation
- ✅ Works alongside Goose perfectly

**Why both?**
- Goose: Manages schema evolution
- sqlc: Makes working with that schema easier
- Complementary, not competing

## Version Alignment Strategy

**Initial release: v1.5.0** (match Python backend)

Migration naming reflects version:
```
00001_v1.5.0_complete_schema.sql    ← Snapshot at v1.5.0
00002_v1.5.0_indexes.sql            ← Part of v1.5.0
00003_v1.6.0_add_log_table.sql      ← Future: v1.6.0
00004_v1.6.1_add_user_auth.sql      ← Future: v1.6.1
```

**Benefits:**
- Clear version alignment with Python backend
- Easy audit: "what changed in v1.6.0?"
- Migration history tracks product versions

## Project Structure

```
Investment-Portfolio-Manager-Backend/
├── config.yaml.example              # Configuration template
├── internal/
│   ├── config/
│   │   └── config.go                # Config loading with defaults
│   └── database/
│       ├── migrations/              # Goose migrations
│       │   ├── 00001_v1.5.0_complete_schema.sql
│       │   └── 00002_v1.5.0_indexes.sql
│       ├── queries/                 # sqlc query definitions
│       │   ├── portfolio.sql
│       │   ├── fund.sql
│       │   └── transaction.sql
│       ├── generated/               # sqlc generated code (gitignored)
│       │   ├── db.go
│       │   ├── models.go
│       │   ├── portfolio.sql.go
│       │   └── fund.sql.go
│       ├── migrate.go               # Migration runner
│       ├── init.go                  # Auto-initialization
│       └── seed.go                  # Seed data (uses repositories!)
├── sqlc.yaml                        # sqlc configuration
└── Dockerfile                       # Production deployment
```

## Implementation

### 1. Configuration System

**`config.yaml.example`**
```yaml
# Application Configuration
app:
  environment: production  # development, production, test
  version: 1.5.0

database:
  path: /data/portfolio_manager.db
  auto_migrate: true
  auto_seed: false  # Only true in development

encryption:
  # For IBKR token encryption (issue #33)
  key_path: /secrets/encryption.key
  # OR use environment variable:
  # key_env: ENCRYPTION_KEY

server:
  port: 8080
  host: 0.0.0.0
  read_timeout: 10s
  write_timeout: 10s
```

**Config with Environment Variable Overrides:**
```go
// internal/config/config.go
package config

type Config struct {
    App        AppConfig
    Database   DatabaseConfig
    Encryption EncryptionConfig
    Server     ServerConfig
}

type DatabaseConfig struct {
    Path       string `yaml:"path" env:"DB_PATH"`
    AutoMigrate bool  `yaml:"auto_migrate" env:"DB_AUTO_MIGRATE"`
    AutoSeed   bool   `yaml:"auto_seed" env:"DB_AUTO_SEED"`
}

func Load() *Config {
    // 1. Load from config.yaml if exists
    cfg := loadFromFile("config.yaml")
    if cfg == nil {
        cfg = &Config{} // Use defaults
    }

    // 2. Override with environment variables
    overrideFromEnv(cfg)

    // 3. Apply sensible defaults
    applyDefaults(cfg)

    return cfg
}

func applyDefaults(cfg *Config) {
    if cfg.Database.Path == "" {
        cfg.Database.Path = "./data/portfolio_manager.db"
    }
    if cfg.App.Environment == "" {
        cfg.App.Environment = "development"
    }
    // Auto-migrate always enabled (safe operation)
    cfg.Database.AutoMigrate = true
}
```

### 2. Version-Aligned Migrations

**`internal/database/migrations/00001_v1.5.0_complete_schema.sql`**
```sql
-- +goose Up
-- Complete schema for Investment Portfolio Manager v1.5.0
-- This migration creates the initial database matching Python backend v1.5.0

CREATE TABLE portfolio (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_archived INTEGER NOT NULL DEFAULT 0,
    exclude_from_overview INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE fund (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    symbol TEXT,
    isin TEXT UNIQUE,
    currency TEXT,
    exchange TEXT,
    investment_type TEXT,
    dividend_type TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE portfolio_fund (
    id TEXT PRIMARY KEY,
    portfolio_id TEXT NOT NULL,
    fund_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
    FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE
);

CREATE TABLE transaction (
    id TEXT PRIMARY KEY,
    portfolio_fund_id TEXT NOT NULL,
    date TEXT NOT NULL,
    transaction_type TEXT NOT NULL,
    shares REAL NOT NULL,
    price_per_share REAL NOT NULL,
    total_amount REAL NOT NULL,
    currency TEXT,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE
);

CREATE TABLE dividend (
    id TEXT PRIMARY KEY,
    portfolio_fund_id TEXT NOT NULL,
    ex_dividend_date TEXT NOT NULL,
    payment_date TEXT,
    dividend_per_share REAL NOT NULL,
    shares_owned REAL NOT NULL,
    total_amount REAL NOT NULL,
    is_reinvested INTEGER NOT NULL DEFAULT 0,
    currency TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE
);

CREATE TABLE fund_price (
    id TEXT PRIMARY KEY,
    fund_id TEXT NOT NULL,
    date TEXT NOT NULL,
    price REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE,
    UNIQUE(fund_id, date)
);

CREATE TABLE fund_history_materialized (
    id TEXT PRIMARY KEY,
    portfolio_fund_id TEXT NOT NULL,
    fund_id TEXT NOT NULL,
    date TEXT NOT NULL,
    shares REAL NOT NULL,
    price REAL NOT NULL,
    value REAL NOT NULL,
    cost REAL NOT NULL,
    realized_gain REAL NOT NULL,
    unrealized_gain REAL NOT NULL,
    total_gain_loss REAL NOT NULL,
    dividends REAL NOT NULL,
    fees REAL NOT NULL,
    calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
    UNIQUE(portfolio_fund_id, date)
);

-- +goose Down
DROP TABLE IF EXISTS fund_history_materialized;
DROP TABLE IF EXISTS fund_price;
DROP TABLE IF EXISTS dividend;
DROP TABLE IF EXISTS transaction;
DROP TABLE IF EXISTS portfolio_fund;
DROP TABLE IF EXISTS fund;
DROP TABLE IF EXISTS portfolio;
```

**`internal/database/migrations/00002_v1.5.0_indexes.sql`**
```sql
-- +goose Up
-- Performance indexes for v1.5.0

CREATE INDEX idx_portfolio_fund_portfolio ON portfolio_fund(portfolio_id);
CREATE INDEX idx_portfolio_fund_fund ON portfolio_fund(fund_id);
CREATE INDEX idx_transaction_pf_date ON transaction(portfolio_fund_id, date);
CREATE INDEX idx_dividend_pf_date ON dividend(portfolio_fund_id, ex_dividend_date);
CREATE INDEX idx_fund_price_fund_date ON fund_price(fund_id, date);
CREATE INDEX idx_fund_history_pf_date ON fund_history_materialized(portfolio_fund_id, date);
CREATE INDEX idx_fund_history_date ON fund_history_materialized(date);
CREATE INDEX idx_fund_history_fund_id ON fund_history_materialized(fund_id);

-- +goose Down
DROP INDEX IF EXISTS idx_fund_history_fund_id;
DROP INDEX IF EXISTS idx_fund_history_date;
DROP INDEX IF EXISTS idx_fund_history_pf_date;
DROP INDEX IF EXISTS idx_fund_price_fund_date;
DROP INDEX IF EXISTS idx_dividend_pf_date;
DROP INDEX IF EXISTS idx_transaction_pf_date;
DROP INDEX IF EXISTS idx_portfolio_fund_fund;
DROP INDEX IF EXISTS idx_portfolio_fund_portfolio;
```

**Future example: `00003_v1.6.0_add_log_table.sql`**
```sql
-- +goose Up
-- Structured logging support for v1.6.0

CREATE TABLE log (
    id TEXT PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL,
    category TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT,
    source TEXT,
    request_id TEXT,
    http_status INTEGER,
    ip_address TEXT,
    user_agent TEXT
);

CREATE INDEX idx_log_timestamp ON log(timestamp);
CREATE INDEX idx_log_level ON log(level);
CREATE INDEX idx_log_category ON log(category);

-- +goose Down
DROP INDEX IF EXISTS idx_log_category;
DROP INDEX IF EXISTS idx_log_level;
DROP INDEX IF EXISTS idx_log_timestamp;
DROP TABLE IF EXISTS log;
```

### 3. sqlc Configuration

**`sqlc.yaml`**
```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "internal/database/queries/"
    schema: "internal/database/migrations/"
    gen:
      go:
        package: "generated"
        out: "internal/database/generated"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_db_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
        emit_empty_slices: true
```

**Example Query Definition: `internal/database/queries/portfolio.sql`**
```sql
-- name: GetPortfolio :one
SELECT * FROM portfolio
WHERE id = ?;

-- name: ListPortfolios :many
SELECT * FROM portfolio
WHERE (sqlc.arg(include_archived) = 1 OR is_archived = 0)
ORDER BY name;

-- name: CreatePortfolio :exec
INSERT INTO portfolio (id, name, description, is_archived, exclude_from_overview)
VALUES (?, ?, ?, ?, ?);

-- name: UpdatePortfolio :exec
UPDATE portfolio
SET name = ?, description = ?, is_archived = ?, exclude_from_overview = ?
WHERE id = ?;

-- name: DeletePortfolio :exec
DELETE FROM portfolio WHERE id = ?;

-- name: ArchivePortfolio :exec
UPDATE portfolio SET is_archived = 1 WHERE id = ?;

-- name: UnarchivePortfolio :exec
UPDATE portfolio SET is_archived = 0 WHERE id = ?;
```

**Generated code (automatic):**
```go
// internal/database/generated/portfolio.sql.go (auto-generated)

func (q *Queries) GetPortfolio(ctx context.Context, id string) (Portfolio, error)
func (q *Queries) ListPortfolios(ctx context.Context, includeArchived int) ([]Portfolio, error)
func (q *Queries) CreatePortfolio(ctx context.Context, arg CreatePortfolioParams) error
// ... etc
```

### 4. Migration Runner with Embedded Files

```go
// internal/database/migrate.go
package database

import (
    "database/sql"
    "embed"
    "fmt"
    "log"

    "github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func RunMigrations(db *sql.DB) error {
    goose.SetBaseFS(embedMigrations)

    if err := goose.SetDialect("sqlite3"); err != nil {
        return fmt.Errorf("failed to set dialect: %w", err)
    }

    if err := goose.Up(db, "migrations"); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil
}

func GetMigrationVersion(db *sql.DB) (int64, error) {
    goose.SetBaseFS(embedMigrations)

    if err := goose.SetDialect("sqlite3"); err != nil {
        return 0, err
    }

    return goose.GetDBVersion(db)
}
```

### 5. Auto-Initialize with Config

```go
// internal/database/init.go
package database

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    "yourproject/internal/config"
)

func Initialize(cfg *config.Config) (*sql.DB, error) {
    // 1. Check if database file exists
    isNew := !fileExists(cfg.Database.Path)

    // 2. Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
        return nil, fmt.Errorf("failed to create database directory: %w", err)
    }

    // 3. Open/create connection
    db, err := sql.Open("sqlite", cfg.Database.Path)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // 4. Enable foreign keys (SQLite requires this)
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
    }

    // 5. Run migrations if enabled
    if cfg.Database.AutoMigrate {
        log.Printf("Running database migrations...")
        if err := RunMigrations(db); err != nil {
            db.Close()
            return nil, fmt.Errorf("migration failed: %w", err)
        }

        version, _ := GetMigrationVersion(db)
        log.Printf("✅ Database schema at version %d (v1.5.0)", version)
    }

    // 6. Seed if new database and auto-seed enabled
    if isNew && cfg.Database.AutoSeed {
        log.Printf("Seeding database with sample data...")
        if err := SeedDatabase(db); err != nil {
            log.Printf("⚠️  Warning: seeding failed: %v", err)
            // Don't fail startup
        } else {
            log.Printf("✅ Database seeded with sample data")
        }
    }

    return db, nil
}

func fileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}
```

### 6. Seed Data Using Go Functions (Not Raw SQL!)

```go
// internal/database/seed.go
package database

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/google/uuid"
    "yourproject/internal/model"
    "yourproject/internal/repository"
)

func SeedDatabase(db *sql.DB) error {
    // Create repositories (reuse existing code!)
    portfolioRepo := repository.NewPortfolioRepository(db)
    fundRepo := repository.NewFundRepository(db)
    portfolioFundRepo := repository.NewPortfolioFundRepository(db)
    transactionRepo := repository.NewTransactionRepository(db)

    // 1. Create sample portfolios using actual models
    portfolios := []*model.Portfolio{
        {
            ID:                 uuid.New().String(),
            Name:               "Growth Portfolio",
            Description:        "Aggressive growth strategy focused on capital appreciation",
            IsArchived:         false,
            ExcludeFromOverview: false,
        },
        {
            ID:                 uuid.New().String(),
            Name:               "Income Portfolio",
            Description:        "Dividend-focused strategy for passive income",
            IsArchived:         false,
            ExcludeFromOverview: false,
        },
        {
            ID:                 uuid.New().String(),
            Name:               "Balanced Portfolio",
            Description:        "Mixed growth and income allocation",
            IsArchived:         false,
            ExcludeFromOverview: false,
        },
    }

    for _, p := range portfolios {
        if err := portfolioRepo.Create(p); err != nil {
            return fmt.Errorf("failed to seed portfolio %s: %w", p.Name, err)
        }
    }

    // 2. Create sample funds
    funds := []*model.Fund{
        {
            ID:             uuid.New().String(),
            Name:           "Vanguard FTSE All-World UCITS ETF",
            Symbol:         "VWCE",
            ISIN:           "IE00BK5BQT80",
            Currency:       "EUR",
            Exchange:       "XETRA",
            InvestmentType: "etf",
            DividendType:   "distributing",
        },
        {
            ID:             uuid.New().String(),
            Name:           "Vanguard S&P 500 UCITS ETF",
            Symbol:         "VUSA",
            ISIN:           "IE00B3XXRP09",
            Currency:       "USD",
            Exchange:       "LSE",
            InvestmentType: "etf",
            DividendType:   "accumulating",
        },
        {
            ID:             uuid.New().String(),
            Name:           "iShares Core MSCI World UCITS ETF",
            Symbol:         "IWDA",
            ISIN:           "IE00B4L5Y983",
            Currency:       "USD",
            Exchange:       "XETRA",
            InvestmentType: "etf",
            DividendType:   "accumulating",
        },
    }

    for _, f := range funds {
        if err := fundRepo.Create(f); err != nil {
            return fmt.Errorf("failed to seed fund %s: %w", f.Name, err)
        }
    }

    // 3. Link funds to portfolios
    // Portfolio 0: VWCE + VUSA
    // Portfolio 1: VUSA + IWDA
    // Portfolio 2: All three

    links := []struct {
        PortfolioIdx int
        FundIdx      int
    }{
        {0, 0}, {0, 1},  // Growth: VWCE + VUSA
        {1, 1}, {1, 2},  // Income: VUSA + IWDA
        {2, 0}, {2, 1}, {2, 2},  // Balanced: All
    }

    portfolioFundIDs := make(map[string]string) // Track for transactions

    for _, link := range links {
        pf := &model.PortfolioFund{
            ID:          uuid.New().String(),
            PortfolioID: portfolios[link.PortfolioIdx].ID,
            FundID:      funds[link.FundIdx].ID,
        }

        if err := portfolioFundRepo.Create(pf); err != nil {
            return fmt.Errorf("failed to link portfolio-fund: %w", err)
        }

        // Store for transaction seeding
        key := fmt.Sprintf("%d-%d", link.PortfolioIdx, link.FundIdx)
        portfolioFundIDs[key] = pf.ID
    }

    // 4. Create sample transactions (past 6 months)
    baseDate := time.Now().AddDate(0, -6, 0)

    transactions := []*model.Transaction{
        {
            ID:              uuid.New().String(),
            PortfolioFundID: portfolioFundIDs["0-0"], // Growth - VWCE
            Date:            baseDate.AddDate(0, 0, 0),
            TransactionType: "buy",
            Shares:          10.0,
            PricePerShare:   95.50,
            TotalAmount:     955.00,
            Currency:        "EUR",
        },
        {
            ID:              uuid.New().String(),
            PortfolioFundID: portfolioFundIDs["0-1"], // Growth - VUSA
            Date:            baseDate.AddDate(0, 1, 0),
            TransactionType: "buy",
            Shares:          5.0,
            PricePerShare:   82.30,
            TotalAmount:     411.50,
            Currency:        "USD",
        },
        // ... more transactions
    }

    for _, t := range transactions {
        if err := transactionRepo.Create(t); err != nil {
            return fmt.Errorf("failed to seed transaction: %w", err)
        }
    }

    // 5. Create sample fund prices (past year, weekly)
    // 6. Create sample dividends

    return nil
}
```

**Benefits of this approach:**
- ✅ Uses existing repositories (DRY)
- ✅ Type-safe (compiler catches schema changes)
- ✅ Validated (uses same validation as real data)
- ✅ When schema changes, seed data breaks at compile time (good!)
- ✅ Easy to maintain and extend

### 7. Main.go Integration

```go
// cmd/server/main.go
package main

import (
    "log"
    "os"

    "yourproject/internal/config"
    "yourproject/internal/database"
    "yourproject/internal/api"
)

func main() {
    // 1. Load configuration (file + env vars)
    cfg := config.Load()

    log.Printf("Starting Investment Portfolio Manager v%s", cfg.App.Version)
    log.Printf("Environment: %s", cfg.App.Environment)

    // 2. Initialize database (auto-migrate + optional seed)
    db, err := database.Initialize(cfg)
    if err != nil {
        log.Fatalf("Database initialization failed: %v", err)
    }
    defer db.Close()

    // 3. Start HTTP server
    server := api.NewServer(cfg, db)

    log.Printf("Server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
    if err := server.Start(); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### 8. Docker Deployment (Primary Use Case)

**`Dockerfile`**
```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /ipm-backend ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /ipm-backend .
COPY config.yaml.example /app/config.yaml.example

# Create data directory
RUN mkdir -p /data

# Environment defaults
ENV ENVIRONMENT=production
ENV DB_PATH=/data/portfolio_manager.db
ENV DB_AUTO_MIGRATE=true
ENV DB_AUTO_SEED=false

EXPOSE 8080

CMD ["./ipm-backend"]
```

**`docker-compose.yml`**
```yaml
version: '3.8'

services:
  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      ENVIRONMENT: production
      DB_PATH: /data/portfolio_manager.db
      DB_AUTO_MIGRATE: true
      DB_AUTO_SEED: false
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}  # From .env file
    volumes:
      - ./data:/data
      - ./config.yaml:/app/config.yaml  # Optional config override
    restart: unless-stopped
```

**Development override: `docker-compose.dev.yml`**
```yaml
version: '3.8'

services:
  backend:
    environment:
      ENVIRONMENT: development
      DB_AUTO_SEED: true  # Seed in development
    volumes:
      - ./data:/data
      - ./config.yaml:/app/config.yaml
      - ./internal:/app/internal  # Live reload with air
```

**Usage:**
```bash
# Production
docker-compose up

# Development (with seeding)
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

# First run output:
# Running database migrations...
# Applying migration: 00001_v1.5.0_complete_schema.sql
# Applying migration: 00002_v1.5.0_indexes.sql
# ✅ Database schema at version 2 (v1.5.0)
# Seeding database with sample data...
# ✅ Database seeded with sample data
# Server starting on 0.0.0.0:8080
```

### 9. Optional: Admin API Endpoints (Instead of CLI)

For Docker environments, expose admin operations via API:

```go
// internal/api/handlers/admin.go (protected by auth middleware)

// POST /api/admin/migrate - Manually trigger migrations
func (h *AdminHandler) Migrate(w http.ResponseWriter, r *http.Request) {
    if err := database.RunMigrations(h.db); err != nil {
        response.RespondError(w, 500, "Migration failed", err.Error())
        return
    }

    version, _ := database.GetMigrationVersion(h.db)
    response.RespondJSON(w, 200, map[string]any{
        "status": "success",
        "version": version,
    })
}

// GET /api/admin/db-status - Check migration status
func (h *AdminHandler) DBStatus(w http.ResponseWriter, r *http.Request) {
    version, _ := database.GetMigrationVersion(h.db)

    var count int
    h.db.QueryRow("SELECT COUNT(*) FROM portfolio").Scan(&count)

    response.RespondJSON(w, 200, map[string]any{
        "schema_version": version,
        "portfolio_count": count,
        "is_seeded": count > 0,
    })
}

// POST /api/admin/seed - Manually trigger seeding (dev only)
func (h *AdminHandler) Seed(w http.ResponseWriter, r *http.Request) {
    if h.cfg.App.Environment != "development" {
        response.RespondError(w, 403, "Forbidden", "Seeding only allowed in development")
        return
    }

    if err := database.SeedDatabase(h.db); err != nil {
        response.RespondError(w, 500, "Seeding failed", err.Error())
        return
    }

    response.RespondJSON(w, 200, map[string]string{"status": "seeded"})
}
```

**Usage in Docker:**
```bash
# Check database status
curl http://localhost:8080/api/admin/db-status

# Manually trigger migration (if needed)
curl -X POST http://localhost:8080/api/admin/migrate

# Seed database (dev only)
curl -X POST http://localhost:8080/api/admin/seed
```

## Implementation Phases

### Phase 1: Configuration & Goose Setup (2 days)
- [ ] Add dependencies: `go get github.com/pressly/goose/v3`
- [ ] Create `internal/config/config.go` with env override support
- [ ] Create `config.yaml.example`
- [ ] Create `internal/database/migrations/` folder
- [ ] Write `00001_v1.5.0_complete_schema.sql`
- [ ] Write `00002_v1.5.0_indexes.sql`
- [ ] Implement migration runner with embedded files
- [ ] Test: Migrations create schema correctly

### Phase 2: sqlc Integration (2 days)
- [ ] Add sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- [ ] Create `sqlc.yaml` configuration
- [ ] Create `internal/database/queries/portfolio.sql` (example)
- [ ] Generate code: `sqlc generate`
- [ ] Update one repository to use sqlc (proof of concept)
- [ ] Test: Generated code works correctly

### Phase 3: Auto-Initialize (1 day)
- [ ] Implement `Initialize()` function
- [ ] Integrate into `main.go`
- [ ] Environment-based behavior
- [ ] Test: Fresh DB auto-migrates, existing DB updates

### Phase 4: Seed Data with Repositories (2 days)
- [ ] Implement `SeedDatabase()` using existing repositories
- [ ] Create comprehensive sample data:
  - 3 portfolios
  - 3-4 funds (real ETFs)
  - 10-15 transactions (past 6 months)
  - 5-10 dividends
  - 90 days of fund prices
- [ ] Test: Seeded database is immediately usable

### Phase 5: Docker Integration (1 day)
- [ ] Create production `Dockerfile`
- [ ] Create `docker-compose.yml`
- [ ] Create `docker-compose.dev.yml` override
- [ ] Test: Docker deployment works end-to-end

### Phase 6: Admin API (Optional, 1 day)
- [ ] Implement admin endpoints
- [ ] Add authentication middleware
- [ ] Document API usage

### Phase 7: Documentation (0.5 days)
- [ ] Update README with configuration
- [ ] Create MIGRATIONS.md guide
- [ ] Document Docker deployment
- [ ] Update development setup guide

## Testing Requirements

- [ ] Test fresh database creation + migration
- [ ] Test existing database detects and applies new migrations
- [ ] Test migration version tracking
- [ ] Test seeding in development environment
- [ ] Test NO seeding in production environment
- [ ] Test config file + env var override precedence
- [ ] Test Docker deployment with volume mount
- [ ] Test seed data is valid and queryable
- [ ] Test sqlc generated code compiles
- [ ] Test migration rollback (if needed)

## Success Criteria

✅ Fresh database auto-creates with v1.5.0 schema
✅ Existing database auto-updates with new migrations
✅ Development environment gets seeded automatically
✅ Production environment stays empty
✅ Config-driven behavior (file + env vars)
✅ Docker deployment works out of the box
✅ sqlc reduces repository boilerplate significantly
✅ Seed data uses type-safe Go code (not raw SQL)
✅ Version alignment clear (migrations named v1.5.0, v1.6.0, etc.)

## Dependencies

```bash
# Required
go get github.com/pressly/goose/v3
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Optional (config parsing)
go get gopkg.in/yaml.v3
```

## Priority
**Critical** - Foundation for v1.5.0 release and all future schema changes

## Estimated Effort
8-9 days (with sqlc integration)

## Future Enhancements
- [ ] Migration dry-run mode
- [ ] Database backup/restore endpoints
- [ ] Seed data variants (minimal, realistic, stress-test)
- [ ] Health check endpoint includes migration status
- [ ] Metrics: track migration execution time

---

## Key Decisions Made

### ✅ Version Alignment
- Migrations named with version: `00001_v1.5.0_*.sql`
- Clear tracking of "what changed when"
- Easy audit trail

### ✅ Goose + sqlc Together
- Goose: Migration management
- sqlc: Type-safe queries
- Complementary tools, not competing

### ✅ Configuration Strategy
- YAML config file with sensible defaults
- Environment variables override config
- Docker-friendly

### ✅ Docker-First Deployment
- Auto-migrate on startup (no CLI needed)
- Environment-driven seeding
- Admin API for manual operations (optional)

### ✅ Seed Data Implementation
- Uses repositories/models (not raw SQL)
- Type-safe and validated
- Schema changes caught by compiler

### ✅ No CLI Dependency
- CLI commands optional (local dev convenience)
- Docker doesn't rely on CLI
- Admin API alternative for production

---

## Open Questions (Resolved)

~~1. CLI vs API for admin operations?~~
**Decision**: Both. CLI for local dev, Admin API for Docker.

~~2. Seed data: raw SQL vs Go functions?~~
**Decision**: Go functions using repositories (type-safe, validated).

~~3. Migration numbering: sequential vs timestamp?~~
**Decision**: Sequential with version prefix (00001_v1.5.0).

~~4. When to add sqlc?~~
**Decision**: Now, alongside Goose setup (before write operations).

~~5. Config file required or optional?~~
**Decision**: Optional (sensible defaults), overridable by env vars.

## New Questions

1. **Existing Python database migration**: How to handle users upgrading from Python v1.5.0?
   - Option A: Provide migration script
   - Option B: Recommend fresh start
   - **Recommendation**: Fresh start for v1.5.0 Go release (clean slate)

2. **sqlc adoption timeline**: Migrate all repositories at once or gradually?
   - **Recommendation**: Gradually, starting with new write operations
