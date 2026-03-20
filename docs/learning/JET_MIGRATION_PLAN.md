# Goose + Jet Migration Plan

**Date:** 2026-03-09
**Status:** Proof-of-concept phase
**Related:** GitHub issue #55, `DATABASE_MIGRATIONS.md`, `ARCHITECTURE_DECISIONS.md` ADR #3 and #9

---

## Context

The codebase has ~4,000 lines across 8 repository files, containing 88 methods and 44 manual `Scan()` calls. The bulk of that code is:

- Writing SQL as string literals
- Calling `.Scan()` with ordered field lists that must match the SELECT columns exactly
- The `getQuerier()` / `WithTx()` boilerplate (repeated identically in all 7 repos)
- `sql.NullString` handling and manual date parsing

After evaluating sqlc (rejected — model duplication, loss of repository boundaries, primitive transaction composition) and sqlx (viable but only fixes Scan, not SQL strings), **go-jet/jet** emerged as the best fit:

- **Type-safe SQL DSL** — queries are Go code, not string literals. Column typos = compile errors.
- **No ORM magic** — you think in SQL, write in Go. JOINs, subqueries, CASE WHEN all look like SQL.
- **Generated from schema** — Goose runs migrations, Jet generates type-safe table constants and model types from the result.
- **Same `database/sql` interface** — `*sql.DB` and `*sql.Tx` both work as query executors. The `WithTx()` pattern translates directly.

## Tool Stack

| Tool | Role | Version |
|------|------|---------|
| **Goose** (pressly/goose v3) | Schema migrations — embedded SQL files with `-- +goose Up/Down` | Already planned (issue #55) |
| **Jet** (go-jet/jet v2) | Type-safe SQL builder + code generation from schema | v2.14.1 |
| **modernc.org/sqlite** | Pure Go SQLite driver (existing) | v1.46.1 |

## Key Findings from Research

### What works

| Concern | Status | Details |
|---------|--------|---------|
| SQLite date handling | **Solved** | `_texttotime=1` DSN param (modernc.org/sqlite v1.46.0+) makes the driver auto-parse `DATE`/`DATETIME`/`TIMESTAMP` TEXT columns into `time.Time`. All 17 date columns in the schema are declared with these types. |
| Reserved keyword `"transaction"` | **Solved** | Jet auto-quotes reserved words since v2.2.0. |
| SQLite functions | **Solved** | `substr`, `strftime`, `COALESCE`, `CASE WHEN` — natively supported. `instr` — use `sqlite.Func("INSTR", ...)` wrapper. |
| Batch inserts | **Solved** | `.MODELS([]model.X{...})` generates multi-row INSERT. |
| Complex dynamic queries | **Solved** | `jet.RawStatement()` with named args + QRM scanning as escape hatch. |
| modernc.org/sqlite compat | **Solved** | Works at runtime (same `database/sql` interface). Generator needs a small Go wrapper script. |
| Transaction support | **Solved** | `*sql.Tx` satisfies Jet's `qrm.DB` interface. Pass `tx` where you'd pass `db`. |

### Known issues requiring workarounds

| Issue | Workaround |
|-------|-----------|
| **int32/float32 bug** ([#302](https://github.com/go-jet/jet/issues/302)) — Jet maps SQLite `INTEGER`→`int32`, `REAL`→`float32` | Custom type override in generator config: `INTEGER`→`int64`, `REAL`→`float64` |
| **Generator uses mattn/go-sqlite3** — CGo dependency at codegen time | Write a small `cmd/jetgen/main.go` that uses `sqlitegen.GenerateDB()` with modernc.org/sqlite. Only needed at dev time, not runtime. |
| **Model duplication** — Jet generates its own model types in `gen/model/` | Decision needed: use Jet models directly, map to existing `model.*`, or configure Jet to skip model generation (`-skip-model`) |
| **JSON columns** (`default_allocations` in `ibkr_config`) | Jet scans as `string`/`*string`. Post-scan `json.Unmarshal` still needed. |
| **Computed fields** (`Configured` flag on `IbkrConfig`) | Post-scan logic remains in repository. |

### `_texttotime` deep dive

Added in modernc.org/sqlite v1.46.0 (2026-02-17) via [cznic/sqlite#245](https://gitlab.com/cznic/sqlite/-/work_items/245). We're already on v1.46.1.

**How it works:**
- `Next()` already converts `DATE`/`DATETIME`/`TIMESTAMP` TEXT columns to `time.Time` (regardless of the flag)
- `_texttotime=1` makes `ColumnTypeScanType()` report `time.Time` consistently, so Jet's QRM scans correctly
- Nullable columns scan into `*time.Time` (nil for SQL NULL)

**Supported formats** (all match our data):
| Format | Example | Supported |
|--------|---------|-----------|
| Date only | `2006-01-02` | Yes |
| Datetime | `2006-01-02 15:04:05` | Yes |
| RFC3339 UTC | `2006-01-02T15:04:05Z` | Yes |
| RFC3339 offset | `2006-01-02T15:04:05+00:00` | Yes |

**Caveat 1:** `TIME`-only columns have an inconsistency (ColumnTypeScanType reports `time.Time` but `Next()` delivers `string`). We have no `TIME`-only columns, so this doesn't affect us.

**Caveat 2: Aggregate functions lose column type information.** `MAX()`, `MIN()`, `COALESCE()`, and other SQL functions/aggregates return results as `string` even when the underlying column is `DATE`/`DATETIME`. The driver only auto-parses direct column reads — aggregate expressions don't carry the column type through to `ColumnTypeScanType()`. This means queries like `SELECT MAX(calculated_at) FROM ...` must still scan into `string`/`sql.NullString` and parse manually. Discovered during materialized view testing (Issue #35) — `GetLatestMaterializedDate` and `GetLatestSourceDates` both use `MAX()` and required manual parsing despite `_texttotime=1` being enabled. Reported upstream: [cznic/sqlite#248](https://gitlab.com/cznic/sqlite/-/work_items/248).

## Current Pain Points (What Jet Eliminates)

### 1. Positional Scan() — 44 call sites

Every SELECT requires a matching Scan with fields in exact column order:

```go
// Current: column-order bug if you swap any two fields
rows.Scan(&t.ID, &t.PortfolioFundID, &dateStr, &t.Type, &t.Shares, &t.CostPerShare, &createdAtStr)
```

```go
// With Jet: scan by column name, type-safe
var transactions []model.Transaction
err := Transaction.SELECT(Transaction.AllColumns).
    WHERE(Transaction.PortfolioFundID.IN(ids...)).
    Query(db, &transactions)
```

### 2. Manual date parsing — every date column

Every date column is currently scanned into a `string`, then parsed with `ParseTime()`:

```go
// Current: 6 lines per date field
var dateStr string
rows.Scan(..., &dateStr, ...)
t.Date, err = ParseTime(dateStr)
if err != nil || t.Date.IsZero() {
    return nil, fmt.Errorf("failed to parse date: %w", err)
}
```

With `_texttotime=1`, the driver scans directly into `time.Time`. No parsing code needed.

### 3. sql.NullString intermediaries — nullable columns

```go
// Current: 4 lines per nullable field
var tokenExpiresStr sql.NullString
rows.Scan(..., &tokenExpiresStr, ...)
if tokenExpiresStr.Valid {
    t, err := ParseTime(tokenExpiresStr.String)
    ic.TokenExpiresAt = &t
}
```

Jet generates `*time.Time` for nullable DATETIME columns. The driver handles nil.

### 4. String SQL — typos caught at runtime

```go
// Current: typo in "portfolio_fund_id" → runtime error
query := `SELECT id, portfolio_fnd_id, date FROM "transaction" WHERE id = ?`
```

```go
// With Jet: typo → compile error
Transaction.SELECT(Transaction.ID, Transaction.PortfolioFndID, Transaction.Date)
// ❌ Transaction.PortfolioFndID undefined
```

### 5. Duplicate getQuerier/WithTx boilerplate — 7 copies

Every repository has identical `getQuerier()` and `WithTx()` methods. With Jet, you pass `db` or `tx` directly to `.Query(db, &dest)` — the interface is the same for both.

## SQL Patterns Inventory

Audit of all SQL patterns in the 8 repository files, and how each maps to Jet:

| Pattern | Count | Files | Jet approach |
|---------|-------|-------|-------------|
| Simple SELECT + Scan | 22 | All repos | `Table.SELECT(...).WHERE(...).Query(db, &dest)` |
| Dynamic WHERE (`WHERE 1=1` + conditional) | 8 | portfolio, fund, pf, ibkr, dividend, developer | Conditional `if` with `.WHERE(predicate)` — Jet supports `BoolExpression` composition with `.AND()` |
| IN clause with placeholder list | 10 | transaction, fund, pf, dividend, materialized, realizedGainLoss | `Column.IN(stringExpList...)` — type-safe, no `strings.Join` |
| CASE WHEN | 3 | transaction | `CASE().WHEN(cond).THEN(val).ELSE(def)` — native Jet |
| COALESCE + SUM aggregate | 6 | materialized, transaction, pf | `COALESCE(SUM(expr), Int(0))` — native Jet |
| Multi-table JOIN (2-4 tables) | 8 | transaction, ibkr, fund, pf, dividend | `.FROM(T1.INNER_JOIN(T2, T1.Col.EQ(T2.Col)))` — native Jet |
| INSERT + VALUES | 12 | All repos | `Table.INSERT(Table.AllColumns).MODEL(obj).Exec(db)` |
| Batch INSERT (prepared stmt loop) | 2 | fund (InsertFundPrices), ibkr (AddIbkrTransactions) | `.MODELS(slice)` — multi-row INSERT |
| UPDATE + SET | 6 | portfolio, transaction, ibkr, developer | `Table.UPDATE(cols).MODEL(obj).WHERE(...).Exec(db)` |
| DELETE | 7 | portfolio, transaction, ibkr | `Table.DELETE().WHERE(...).Exec(db)` |
| ON CONFLICT / INSERT OR REPLACE | 6 | fund, developer, ibkr | `INSERT(...).ON_CONFLICT(col).DO_UPDATE(SET(...))` or `RawStatement` for `INSERT OR REPLACE` |
| Correlated subqueries | 4 | materialized | `RawStatement` — too complex for DSL |
| SQLite functions (`substr`, `instr`, `date`, `strftime`, `datetime`) | 7 | fund, materialized | `substr`/`strftime` native; `instr`/`date`/`datetime` via `sqlite.Func(...)` |
| Cursor pagination with dynamic sort | 1 | developer (GetLogs) | `RawStatement` — too dynamic for DSL |

### Methods best kept as RawStatement

These are complex enough that the Jet DSL would be more verbose than helpful:

1. **`materialized_repository.go:buildMaterializedQuery`** — 4 correlated subqueries, `date()` function, `strftime()`, `GROUP BY`, dynamic IN clause. ~45 lines of SQL.
2. **`developer_repository.go:GetLogs`** — cursor pagination with dynamic WHERE clauses assembled from `[]string` slices, dynamic sort direction. ~90 lines of Go.
3. **`fund_repository.go:GetFundBySymbolOrIsin`** — `substr(symbol, 1, instr(symbol || '.', '.') - 1)` expression is SQLite-specific and awkward in DSL.

Everything else (75+ methods) maps cleanly to Jet's DSL.

## Model Type Strategy

### Option A: Use Jet-generated models directly

Jet generates a `model` package from the schema. We'd replace `internal/model/` with Jet's output.

**Pros:** Single source of truth. No mapping layer.
**Cons:** Jet models don't have JSON tags by default (fixable with `-model-json-tag=snake-case`, but our tags use `camelCase`). Jet models don't include response/computed types (`PortfolioSummary`, `TransactionResponse`, etc.) — those would still live in a separate package.

### Option B: Keep existing models, use alias tags for scanning

Keep `internal/model/` as-is. Add `alias:"table.column"` struct tags for Jet's QRM scanning. Skip Jet model generation (`-skip-model`).

**Pros:** Zero disruption to existing code. Models retain custom JSON tags. Response types stay where they are.
**Cons:** Need to add `alias` tags to every model struct. Two things to keep in sync (schema + tags).

### Option C: Hybrid — use Jet models for table types, keep response types separate

Use Jet-generated models for direct table mappings (`Portfolio`, `Fund`, `Transaction`, `Dividend`, etc.). Keep hand-written response/computed types (`PortfolioSummary`, `TransactionResponse`, `DividendFund`, etc.) with a thin mapping function.

**Pros:** Clean separation. Table types are auto-generated. Response types are hand-crafted for API needs.
**Cons:** Need mapping functions between Jet models and response types. The generator's JSON tags may not match existing API contracts.

### Recommendation: Decide during PoC

The PoC will test Option A (simplest) and evaluate whether the JSON tag mismatch and response type separation are dealbreakers.

## Proof-of-Concept Test Cases

Three test cases covering the risk spectrum. Each rewrites specific repository methods using Jet's DSL and runs them against an in-memory SQLite database with `_texttotime=1`.

### PoC 1: PortfolioRepository — basics work

**Proves:** Jet + modernc.org/sqlite + struct scanning + dynamic WHERE + CRUD

**Methods to rewrite:**

| Method | Current pattern | Jet pattern |
|--------|---------------|-------------|
| `GetPortfolios(filter)` | `WHERE 1=1` + conditional `+=` | `SELECT(...).WHERE(conditionalPredicate).Query(db, &portfolios)` |
| `GetPortfolioOnID(id)` | `QueryRow` + `Scan` 5 fields | `SELECT(AllColumns).WHERE(ID.EQ(String(id))).Query(db, &portfolio)` |
| `InsertPortfolio(p)` | `ExecContext` with 5 positional args | `INSERT(AllColumns).MODEL(p).Exec(db)` |
| `UpdatePortfolio(p)` | `ExecContext` with 5 positional args + WHERE | `UPDATE(cols).MODEL(p).WHERE(ID.EQ(String(p.ID))).Exec(db)` |
| `DeletePortfolio(id)` | `ExecContext` + `RowsAffected` check | `DELETE().WHERE(ID.EQ(String(id))).Exec(db)` + result check |

**What we validate:**
- Code generation from schema works with modernc.org/sqlite
- int64/float64 type overrides work
- Basic CRUD operations
- Dynamic WHERE clause composition
- Struct scanning without manual Scan()

**Risk level:** Low — no dates, no nullable fields, no complex SQL

---

### PoC 2: TransactionRepository — expressions + reserved keyword

**Proves:** `"transaction"` table quoting, CASE WHEN, multi-JOIN, COALESCE+SUM aggregate

**Methods to rewrite:**

| Method | Current pattern | Jet pattern |
|--------|---------------|-------------|
| `GetTransactionsPerPortfolio(portfolioID)` | 4-table JOIN + CASE WHEN + NullString | `SELECT(...).FROM(Transaction.INNER_JOIN(...)).WHERE(...)` with `CASE().WHEN().THEN().ELSE()` |
| `GetSharesOnDate(pfID, date)` | `COALESCE(SUM(CASE ... END), 0)` | `SELECT(COALESCE(SUM(CASE()...), Float(0))).FROM(Transaction).WHERE(...)` |
| `InsertTransaction(t)` | `ExecContext` into `"transaction"` | `Transaction.INSERT(...).MODEL(t).Exec(db)` |

**What we validate:**
- Reserved keyword `"transaction"` is auto-quoted in generated SQL
- CASE WHEN expressions compile and execute correctly
- COALESCE + SUM aggregate works
- Multi-table JOINs with LEFT JOIN
- `_texttotime=1` handles DATE columns in the transaction table

**Risk level:** Medium — reserved keyword, complex expressions, date handling

---

### PoC 3: IbkrRepository — dates + nullable + JSON

**Proves:** `_texttotime` works end-to-end, `*time.Time` scanning, JSON column handling, batch INSERT

**Methods to rewrite:**

| Method | Current pattern | Jet pattern |
|--------|---------------|-------------|
| `GetIbkrConfig()` | `sql.NullString` for dates + JSON + `ParseTime()` | `SELECT(AllColumns).Query(db, &config)` — driver handles dates, post-scan for JSON |
| `AddIbkrTransactions(txns)` | Prepared stmt + loop + `NullString` for processedAt | `.MODELS(txns)` multi-row INSERT |
| `GetIbkrTransaction(id)` | 17-column Scan + 3 date parses + 1 nullable date | `SELECT(AllColumns).WHERE(ID.EQ(String(id))).Query(db, &txn)` |

**What we validate:**
- `_texttotime=1` correctly parses DATETIME columns into `time.Time`
- Nullable `*time.Time` fields (`token_expires_at`, `last_import_date`, `processed_at`) scan as nil/value
- JSON column (`default_allocations`) requires post-scan unmarshal (expected)
- Computed field (`Configured`) requires post-scan logic (expected)
- Batch INSERT with `.MODELS()` works for IBKR transactions

**Risk level:** High — this is the make-or-break test for date handling

---

### PoC evaluation criteria

| Outcome | Action |
|---------|--------|
| All three PoCs pass cleanly | Plan full migration — repo by repo |
| Basics work but complex queries too verbose | Use Jet for simple repos, keep raw SQL for complex ones |
| Date handling fails or driver incompatibilities | Fall back to sqlx (fixes Scan but not SQL strings) |

## Implementation Sequence

### Phase 0: Setup (PoC prerequisites)

1. `go get github.com/go-jet/jet/v2`
2. Create `cmd/jetgen/main.go`:
   - Opens temp SQLite file with modernc.org/sqlite
   - Runs Goose migrations (from `internal/database/migrations/`)
   - Calls `sqlitegen.GenerateDB()` with custom type overrides (`INTEGER`→`int64`, `REAL`→`float64`)
   - Generates into `internal/jetgen/` (table/ + model/)
3. Update `database.Open()` DSN to include `_texttotime=1`
4. Update `testutil.SetupTestDB()` DSN to include `_texttotime=1`
5. Run generator, verify output

### Phase 1: PoC (validate feasibility)

6. Write PoC 1 tests — PortfolioRepository with Jet
7. Write PoC 2 tests — TransactionRepository with Jet
8. Write PoC 3 tests — IbkrRepository with Jet
9. Evaluate results, decide go/no-go

### Phase 2: Migration (if PoC passes)

10. Decide model strategy (Option A/B/C from above)
11. Migrate repositories one at a time, starting with simplest:
    - `portfolio_repository.go` (5 methods)
    - `transaction_repository.go` (8 methods)
    - `portfolio_fund_repository.go` (11 methods)
    - `dividend_repository.go` (7 methods)
    - `fund_repository.go` (16 methods — most complex, has `substr`/`instr`)
    - `realizedGainLoss_repository.go` (4 methods)
    - `ibkr_repository.go` (17 methods — dates, JSON, batch)
    - `developer_repository.go` (20 methods — GetLogs stays as RawStatement)
    - `materialized_repository.go` (5 methods — buildMaterializedQuery stays as RawStatement)
12. Remove `ParseTime()` helper and `sql.NullString` intermediaries
13. Remove (or simplify) `getQuerier()` / `WithTx()` boilerplate
14. Update existing tests to work with new repository implementations
15. Add `make generate` target for Goose + Jet codegen

### Phase 3: Cleanup

16. Remove `internal/repository/db.go` (Querier interface) if no longer needed
17. Update documentation (`REPOSITORY_TRANSACTION_PATTERNS.md`, etc.)
18. Update `ARCHITECTURE_DECISIONS.md` with Jet ADR

## Makefile Integration

```makefile
.PHONY: generate jetgen

# Run Goose migrations on a temp DB, then generate Jet code
jetgen:
	go run ./cmd/jetgen

# Full generation pipeline
generate: jetgen
	@echo "Jet code generated in internal/jetgen/"
```

## Directory Structure (after migration)

```
internal/
├── jetgen/                    # ← NEW: Jet-generated code (DO NOT EDIT)
│   ├── table/                 # Table/column constants for SQL builder
│   │   ├── portfolio.go
│   │   ├── transaction.go     # Auto-quoted reserved keyword
│   │   ├── fund.go
│   │   ├── ibkr_config.go
│   │   └── ...
│   └── model/                 # Generated model structs (if using Option A/C)
│       ├── portfolio.go
│       ├── transaction.go
│       └── ...
├── model/                     # Existing models (keep for response types + Option B/C)
├── repository/                # Repositories — rewritten to use Jet DSL
├── database/
│   ├── database.go            # Open() with _texttotime=1
│   ├── migrate.go             # Goose migrations
│   └── migrations/
│       └── 00001_initial_schema.sql
└── ...
```

## Verification Checklist

- [ ] `go run ./cmd/jetgen` — generates code without errors
- [ ] Generated `int64`/`float64` types (not `int32`/`float32`)
- [ ] Generated `"transaction"` table is properly quoted
- [ ] PoC 1: Portfolio CRUD works with Jet
- [ ] PoC 2: Transaction JOIN + CASE WHEN works
- [ ] PoC 3: IBKR dates scan correctly with `_texttotime=1`
- [ ] PoC 3: Nullable `*time.Time` fields are nil for NULL
- [ ] PoC 3: Batch INSERT with `.MODELS()` works
- [ ] `go test ./...` — full suite still passes
- [ ] `go build ./...` — clean build

## References

- [go-jet/jet GitHub](https://github.com/go-jet/jet)
- [Jet Wiki](https://github.com/go-jet/jet/wiki)
- [Jet SQLite INTEGER/REAL bug #302](https://github.com/go-jet/jet/issues/302)
- [modernc.org/sqlite _texttotime](https://pkg.go.dev/modernc.org/sqlite)
- [Goose (pressly/goose)](https://github.com/pressly/goose)
- [SQLite Date/Time Functions](https://sqlite.org/lang_datefunc.html)
- Project docs: `DATABASE_MIGRATIONS.md`, `ARCHITECTURE_DECISIONS.md`, `REPOSITORY_TRANSACTION_PATTERNS.md`
