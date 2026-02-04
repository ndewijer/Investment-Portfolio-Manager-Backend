# SQLC Migration Decision Guide

Analysis of whether and when to migrate from `database/sql` to `sqlc`, `Atlas`, and `Goose`.

---

## Table of Contents

1. [Current State](#current-state)
2. [What Are These Tools?](#what-are-these-tools)
3. [The Decision Matrix](#the-decision-matrix)
4. [My Recommendation](#my-recommendation)
5. [If You Stay with database/sql](#if-you-stay-with-databasesql)
6. [If You Migrate to sqlc](#if-you-migrate-to-sqlc)
7. [Migration Strategy](#migration-strategy)

---

## Current State

### What You're Using Now

**`database/sql`** - Go's standard library database interface:
```go
rows, err := db.Query("SELECT id, name FROM portfolio WHERE is_archived = ?", false)
for rows.Next() {
    err := rows.Scan(&p.ID, &p.Name)
    // ...
}
```

### Pain Points You're Experiencing

1. **Boilerplate** - Every query needs similar scanning code
2. **No compile-time checking** - SQL errors discovered at runtime
3. **Column order dependence** - `Scan()` arguments must match SELECT order
4. **Schema changes** - No migration system, manual ALTER TABLE

### What You've Learned

By using `database/sql` directly, you now understand:
- How Go talks to databases
- Pointer semantics for `Scan()`
- Connection pooling
- Query vs QueryRow vs Exec
- NULL handling with sql.NullString, etc.
- Error handling patterns

**This knowledge is valuable.** It will make you better at using any tool.

---

## What Are These Tools?

### sqlc - Query Code Generator

**What it does:** Generates type-safe Go code from SQL queries.

**You write:**
```sql
-- name: GetPortfolio :one
SELECT id, name, description, is_archived
FROM portfolio
WHERE id = ?;
```

**sqlc generates:**
```go
func (q *Queries) GetPortfolio(ctx context.Context, id string) (Portfolio, error) {
    row := q.db.QueryRowContext(ctx, getPortfolio, id)
    var i Portfolio
    err := row.Scan(&i.ID, &i.Name, &i.Description, &i.IsArchived)
    return i, err
}
```

**Benefits:**
- Compile-time SQL validation
- Type-safe parameters and returns
- No manual scanning
- Query changes automatically update generated code

**Costs:**
- Learning curve
- Configuration files
- Generated code in repo
- Requires sqlc CLI in build process

### Atlas - Schema Management

**What it does:** Declarative database schema management and migrations.

**You write (desired schema):**
```hcl
table "portfolio" {
  schema = schema.main
  column "id" { type = text }
  column "name" { type = text }
  primary_key { columns = [column.id] }
}
```

**Atlas generates migrations:**
```sql
-- 20240122_add_column.sql
ALTER TABLE portfolio ADD COLUMN description TEXT;
```

**Benefits:**
- Declarative schema (state what you want, not how to get there)
- Automatic migration generation
- Schema validation
- Drift detection

### Goose - Migration Runner

**What it does:** Executes SQL migration files in order.

**You write:**
```sql
-- 001_create_portfolio.sql
-- +goose Up
CREATE TABLE portfolio (id TEXT PRIMARY KEY, name TEXT);

-- +goose Down
DROP TABLE portfolio;
```

**Goose runs:**
```bash
goose up      # Apply pending migrations
goose down    # Rollback last migration
goose status  # Show migration status
```

**Benefits:**
- Simple and focused
- SQL migrations (you control the SQL)
- Up and down migrations
- Embedded or CLI usage

---

## The Decision Matrix

### When to Stay with database/sql

| Situation | Recommendation |
|-----------|----------------|
| Still learning Go fundamentals | Stay |
| Queries are working fine | Stay |
| Minimal new features planned | Stay |
| Tight deadline | Stay |
| Team unfamiliar with sqlc | Stay |

### When to Migrate to sqlc

| Situation | Recommendation |
|-----------|----------------|
| Adding many new queries | Migrate |
| Frequent SQL bugs at runtime | Migrate |
| Tired of scanning boilerplate | Migrate |
| Planning major feature additions | Migrate |
| Want compile-time safety | Migrate |

### When to Add Atlas/Goose

| Situation | Recommendation |
|-----------|----------------|
| Making schema changes | Add migrations |
| Multiple developers | Add migrations |
| Production deployment | Add migrations |
| Need rollback capability | Add migrations |

---

## My Recommendation

### Short Answer: **Not Yet, But Soon**

Here's my reasoning:

### Why Not Right Now

1. **You haven't finished learning** - You're about to implement write operations. Doing this with `database/sql` will teach you:
   - Transaction management
   - Error handling in writes
   - Race condition awareness
   - The pain that sqlc solves

2. **Migration during learning is harder** - Converting existing code while learning new patterns is cognitively expensive.

3. **Your current code works** - The GET endpoints are functional. Don't fix what isn't broken.

### When to Switch

**Switch to sqlc after:**
- You've implemented 2-3 POST endpoints manually
- You've felt the pain of manual scanning
- You've had at least one SQL bug caught at runtime
- You understand what sqlc is saving you from

**This is probably 2-4 weeks of development time.**

### The Learning-Then-Tool Pattern

```
Week 1-2: Implement POST /portfolio with database/sql
Week 3-4: Implement POST /transaction with database/sql
          ↓
       Feel the pain
          ↓
Week 5: Evaluate sqlc seriously
Week 6: Migrate if it makes sense
```

### Migration Tool Recommendation

**Use Goose, not Atlas** for your project:

Why Goose:
- Simpler mental model
- You write the SQL (good for learning)
- Works with existing database
- No new configuration language (HCL)

Atlas is powerful but overkill for a personal learning project with SQLite.

---

## If You Stay with database/sql

### Improvements You Can Make Now

#### 1. Create a Query Builder Helper

```go
// internal/database/query.go
type QueryBuilder struct {
    query strings.Builder
    args  []any
}

func NewQuery(base string) *QueryBuilder {
    qb := &QueryBuilder{}
    qb.query.WriteString(base)
    return qb
}

func (qb *QueryBuilder) Where(condition string, arg any) *QueryBuilder {
    qb.query.WriteString(" AND ")
    qb.query.WriteString(condition)
    qb.args = append(qb.args, arg)
    return qb
}

func (qb *QueryBuilder) Build() (string, []any) {
    return qb.query.String(), qb.args
}
```

Usage:
```go
qb := database.NewQuery("SELECT * FROM portfolio WHERE 1=1")
if !filter.IncludeArchived {
    qb.Where("is_archived = ?", false)
}
query, args := qb.Build()
rows, err := db.Query(query, args...)
```

#### 2. Create Scanner Helpers

```go
// internal/repository/scanner.go
type PortfolioScanner struct {
    Portfolio model.Portfolio
}

func (s *PortfolioScanner) Scan(row interface{ Scan(...any) error }) error {
    return row.Scan(
        &s.Portfolio.ID,
        &s.Portfolio.Name,
        &s.Portfolio.Description,
        &s.Portfolio.IsArchived,
        &s.Portfolio.ExcludeFromOverview,
    )
}
```

Usage:
```go
row := db.QueryRow("SELECT id, name, description, is_archived, exclude_from_overview FROM portfolio WHERE id = ?", id)
scanner := &PortfolioScanner{}
if err := scanner.Scan(row); err != nil {
    return nil, err
}
return &scanner.Portfolio, nil
```

#### 3. Define Column Constants

```go
// internal/repository/columns.go
const portfolioColumns = "id, name, description, is_archived, exclude_from_overview"
const portfolioColumnsInsert = "(id, name, description, is_archived, exclude_from_overview)"
const portfolioPlaceholders = "(?, ?, ?, ?, ?)"
```

Usage:
```go
query := fmt.Sprintf("SELECT %s FROM portfolio WHERE id = ?", portfolioColumns)
```

---

## If You Migrate to sqlc

### Setup Process

#### 1. Install sqlc

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

#### 2. Create Configuration

`sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "internal/database/queries/"
    schema: "internal/database/schema/"
    gen:
      go:
        package: "db"
        out: "internal/database/db"
        emit_json_tags: true
        emit_empty_slices: true
```

#### 3. Define Schema

`internal/database/schema/schema.sql`:
```sql
CREATE TABLE portfolio (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_archived INTEGER NOT NULL DEFAULT 0,
    exclude_from_overview INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE fund (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    isin TEXT UNIQUE,
    symbol TEXT,
    currency TEXT,
    exchange TEXT,
    investment_type TEXT,
    dividend_type TEXT
);

-- ... rest of schema
```

#### 4. Write Queries

`internal/database/queries/portfolio.sql`:
```sql
-- name: GetPortfolio :one
SELECT id, name, description, is_archived, exclude_from_overview
FROM portfolio
WHERE id = ?;

-- name: ListPortfolios :many
SELECT id, name, description, is_archived, exclude_from_overview
FROM portfolio
WHERE (? = 1 OR is_archived = 0)
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
```

#### 5. Generate Code

```bash
sqlc generate
```

#### 6. Use Generated Code

```go
import "yourproject/internal/database/db"

func (s *PortfolioService) GetPortfolio(ctx context.Context, id string) (*db.Portfolio, error) {
    return s.queries.GetPortfolio(ctx, id)
}
```

### Migration from Existing Code

**Don't do a big-bang migration.** Instead:

1. Keep existing `database/sql` code working
2. Add sqlc alongside for new features
3. Gradually migrate existing code
4. Remove old code when no longer needed

```go
// Phase 1: Both systems coexist
type PortfolioRepository struct {
    db      *sql.DB        // Old
    queries *db.Queries    // New (sqlc)
}

// Phase 2: New methods use sqlc
func (r *PortfolioRepository) CreatePortfolio(ctx context.Context, p *model.Portfolio) error {
    return r.queries.CreatePortfolio(ctx, db.CreatePortfolioParams{
        ID:          p.ID,
        Name:        p.Name,
        Description: p.Description,
        // ...
    })
}

// Phase 3: Migrate old methods one by one
```

---

## Migration Strategy

### Adding Goose (Database Migrations)

#### 1. Install Goose

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

#### 2. Create Migrations Directory

```bash
mkdir -p internal/database/migrations
```

#### 3. Create Initial Migration

Since you already have a database, create a baseline:

```bash
goose -dir internal/database/migrations create baseline sql
```

Edit `internal/database/migrations/00001_baseline.sql`:
```sql
-- +goose Up
-- This migration documents the existing schema
-- The database already has these tables

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS portfolio (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_archived INTEGER NOT NULL DEFAULT 0,
    exclude_from_overview INTEGER NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- ... rest of existing schema

-- +goose Down
-- Cannot roll back initial schema
```

#### 4. Add Goose to Your App

```go
// cmd/server/main.go
import (
    "github.com/pressly/goose/v3"
)

func runMigrations(db *sql.DB) error {
    goose.SetBaseFS(embeddedMigrations) // Or use filesystem
    if err := goose.SetDialect("sqlite3"); err != nil {
        return err
    }
    return goose.Up(db, ".")
}
```

#### 5. Create New Migrations

```bash
goose -dir internal/database/migrations create add_log_table sql
```

```sql
-- +goose Up
CREATE TABLE log (
    id TEXT PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL,
    category TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT,
    source TEXT,
    request_id TEXT,
    stack_trace TEXT,
    http_status INTEGER,
    ip_address TEXT,
    user_agent TEXT
);

-- +goose Down
DROP TABLE log;
```

---

## Summary

| Tool | Recommendation | When |
|------|----------------|------|
| database/sql | Keep using | Now - next 2-4 weeks |
| sqlc | Consider | After 2-3 POST endpoints |
| Goose | Add soon | When adding logging table |
| Atlas | Skip | Overkill for this project |

### Action Items

1. **Now:** Continue with database/sql for write operations
2. **Soon:** Add Goose for migration management (you need the log table)
3. **Later:** Evaluate sqlc after feeling the manual scanning pain
4. **Maybe never:** Atlas (Goose is sufficient)

---

*Document created: 2026-01-22*
*For: Investment Portfolio Manager Go Backend*
