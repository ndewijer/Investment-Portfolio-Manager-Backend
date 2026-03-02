# Database Initialization with Goose Migrations

## Context

The Go backend has no database initialization logic. `database.Open()` just connects — if the DB doesn't exist, the app crashes on the first query. The Python backend auto-creates the schema on first startup.

We need: automatic schema creation on first run + a migration system to track schema changes across versions without relying on git.

## Tool Choice: Goose

[Goose](https://github.com/pressly/goose) (pressly/goose v3) — chosen because:
- Numbered SQL migration files with `-- +goose Up` / `-- +goose Down` annotations
- Native SQLite support
- Can be embedded in the binary via Go's `embed` package (no external files needed at runtime)
- Runs programmatically as a library (no CLI dependency in production)
- Lightweight — fits a single-user portfolio manager
- Already referenced in `docs/TOOLING_RECOMMENDATIONS.md`

This is independent of the planned sqlc migration (Phase 3) — goose handles schema versioning, sqlc handles query code generation.

## Implementation

### Step 1: Add goose dependency

```
go get github.com/pressly/goose/v3
```

### Step 2: Create initial migration file

`migrations/00001_initial_schema.sql` — contains the full DDL extracted from `internal/testutil/database.go` (all 17 tables + 24 indexes), plus default `system_setting` rows.

Format:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS portfolio (...);
CREATE TABLE IF NOT EXISTS fund (...);
-- ... all tables and indexes ...

-- Default system settings
INSERT INTO system_setting (...) VALUES (..., 'LOGGING_ENABLED', 'true', ...);
INSERT INTO system_setting (...) VALUES (..., 'LOGGING_LEVEL', 'info', ...);

-- +goose Down
DROP TABLE IF EXISTS fund_history_materialized;
DROP TABLE IF EXISTS ibkr_import_cache;
-- ... all tables in reverse dependency order ...
```

### Step 3: Create `internal/database/migrate.go`

Embeds migrations and runs them programmatically:

```go
package database

import (
    "database/sql"
    "embed"

    "github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate runs all pending migrations.
// On a fresh DB this creates the full schema.
// On an existing DB it applies only new migrations.
func Migrate(db *sql.DB) error {
    goose.SetBaseFS(migrations)
    goose.SetDialect("sqlite")
    return goose.Up(db, "migrations")
}
```

The `migrations/` directory lives inside `internal/database/` so the `//go:embed` directive can reach it.

### Step 4: Update `internal/database/database.go`

Add `EnsureDir()` to create the DB directory if missing:
```go
func EnsureDir(dbPath string) error {
    dir := filepath.Dir(dbPath)
    if dir == "" || dir == "." {
        return nil
    }
    return os.MkdirAll(dir, 0o750)
}
```

### Step 5: Update `cmd/server/main.go`

Add directory creation and migration between `Open` and service creation:
```go
database.EnsureDir(cfg.Database.Path)       // before Open
db, err := database.Open(cfg.Database.Path)
database.Migrate(db)                        // runs pending migrations
```

### Step 6: Update `internal/testutil/database.go`

Replace inline DDL with goose migrations:
```go
func createTestSchema(db *sql.DB) error {
    return database.Migrate(db)
}
```

Remove the ~250 lines of inline SQL. Single source of truth for schema.

### Step 7: Update `internal/service/system_service.go`

Replace `getDbVersion()` (reads `alembic_version`) with a goose query:
```go
func (s *SystemService) getDbVersion() (string, error) {
    var versionID int64
    err := s.db.QueryRow(
        "SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1",
    ).Scan(&versionID)
    if err == nil {
        return fmt.Sprintf("%d", versionID), nil
    }
    // Fallback for legacy Python DBs
    var versionNum string
    err = s.db.QueryRow("SELECT version_num FROM alembic_version").Scan(&versionNum)
    if err != nil {
        return "", err
    }
    return versionNum, nil
}
```

### Step 8: Tests

`internal/database/migrate_test.go`:

**Functional tests:**
- `TestMigrate_NewDatabase` — fresh in-memory DB → all 17 tables exist, goose version table exists
- `TestMigrate_Idempotent` — call twice → no errors, no duplicate rows
- `TestMigrate_DefaultSettings` — verify LOGGING_ENABLED and LOGGING_LEVEL defaults inserted

**Schema correctness test (golden file):**
- `TestMigrate_SchemaMatchesGoldenFile` — after running migrations on a fresh DB, dump the full schema via:
  ```sql
  SELECT sql FROM sqlite_master WHERE sql IS NOT NULL ORDER BY name
  ```
  Compare against a committed golden file at `internal/database/testdata/golden_schema.sql`.

  If the test fails, it means either:
  1. A migration was changed without updating the golden file → run `go test -update-golden` to regenerate
  2. The migration produces unexpected schema → investigate

  This catches everything: columns, types, constraints, indexes, foreign keys, unique constraints — without writing individual PRAGMA assertions for each table. It directly addresses the past finding where a constraint existed in the DB but not in the DDL.

  The golden file is regenerated via a `-update` test flag:
  ```go
  var updateGolden = flag.Bool("update-golden", false, "update golden schema file")

  func TestMigrate_SchemaMatchesGoldenFile(t *testing.T) {
      db := setupFreshDB(t)
      database.Migrate(db)

      actual := dumpSchema(t, db) // SELECT sql FROM sqlite_master ...

      goldenPath := "testdata/golden_schema.sql"
      if *updateGolden {
          os.WriteFile(goldenPath, []byte(actual), 0o644)
          return
      }

      expected, _ := os.ReadFile(goldenPath)
      if actual != string(expected) {
          t.Errorf("schema drift detected; run with -update-golden to regenerate")
          // Show diff for easy debugging
      }
  }
  ```

## Directory Structure

```
internal/database/
├── database.go          # Open(), HealthCheck(), EnsureDir()
├── migrate.go           # Migrate(), embed directive
├── migrate_test.go      # migration tests (functional + golden file)
├── testdata/
│   └── golden_schema.sql  # committed schema snapshot for drift detection
└── migrations/
    └── 00001_initial_schema.sql
```

## Files Changed

| File | Action |
|------|--------|
| `go.mod` / `go.sum` | MODIFY — add `github.com/pressly/goose/v3` |
| `internal/database/migrate.go` | CREATE — embed + Migrate() |
| `internal/database/migrations/00001_initial_schema.sql` | CREATE — full DDL |
| `internal/database/migrate_test.go` | CREATE — functional + golden file schema tests |
| `internal/database/testdata/golden_schema.sql` | CREATE — committed schema snapshot |
| `internal/testutil/database.go` | MODIFY — replace inline DDL with `database.Migrate(db)` |
| `cmd/server/main.go` | MODIFY — add EnsureDir + Migrate calls |
| `internal/service/system_service.go` | MODIFY — update getDbVersion() |

## Verification

1. `go build ./...` — clean build
2. `go test ./internal/database/...` — migration tests pass (functional + golden file)
3. `go test ./...` — full suite passes (existing tests still work with `database.Migrate` in testutil)
4. `rm -f ./data/portfolio_manager.db && make run` — fresh start, tables created via migration
5. `make run` again — existing DB, migration is a no-op
6. `GET /api/system/version` — returns goose migration version

### Golden file workflow
- **First time**: run `go test ./internal/database/... -update-golden` to generate `testdata/golden_schema.sql`, then commit it
- **After adding a new migration**: run `-update-golden` again, review the diff, commit
- **CI**: the golden file test runs without the flag — any schema drift fails the build

## Schema Update Guide

This section documents the full workflow for making schema changes. Follow these steps in order.

### 1. Create a new migration file

Migration files live in `internal/database/migrations/` and follow the naming convention `NNNNN_description.sql` (zero-padded, sequential). Never modify an existing migration that has been merged to main.

```bash
# Find the next number
ls internal/database/migrations/
# 00001_initial_schema.sql

# Create the next migration
touch internal/database/migrations/00002_add_portfolio_color.sql
```

### 2. Write the migration SQL

Each file has `-- +goose Up` and `-- +goose Down` sections. Both are required.

```sql
-- +goose Up
ALTER TABLE portfolio ADD COLUMN color VARCHAR(7) DEFAULT '#000000';

-- +goose Down
-- SQLite <3.35.0 doesn't support DROP COLUMN.
-- Recreate the table without the column if a rollback is needed.
CREATE TABLE portfolio_backup AS SELECT id, name, description, is_archived, exclude_from_overview FROM portfolio;
DROP TABLE portfolio;
ALTER TABLE portfolio_backup RENAME TO portfolio;
```

**SQLite-specific notes:**
- `ALTER TABLE ... ADD COLUMN` works, but `DROP COLUMN` requires SQLite 3.35.0+
- For down migrations on older SQLite, use the create-copy-drop-rename pattern shown above
- `CREATE INDEX IF NOT EXISTS` is recommended for safety
- Foreign key constraints are only enforced when `PRAGMA foreign_keys = ON` (set by `database.Open()`)

### 3. Update the golden schema file

After writing the migration, regenerate the golden file so tests pass:

```bash
go test ./internal/database/... -update-golden
```

This runs all migrations on a fresh in-memory DB, dumps the resulting schema, and writes it to `internal/database/testdata/golden_schema.sql`.

**Review the diff before committing:**
```bash
git diff internal/database/testdata/golden_schema.sql
```

Verify that only the expected changes appear (e.g., one new column, one new index). If unexpected changes appear, investigate — they could indicate a problem with the migration or a drift from a previous migration.

### 4. Run the full test suite

```bash
go test ./...
```

This verifies:
- The golden file matches the migrated schema (no drift)
- Existing repository/service tests still pass with the updated schema
- Migration is idempotent (the test calls `Migrate` twice)

### 5. Test locally against a real database

```bash
# Option A: fresh database
rm -f ./data/portfolio_manager.db
make run
# App should start, new migration applied

# Option B: existing database (incremental migration)
make run
# Only the new migration should run; existing data preserved

# Verify via the version endpoint
curl http://localhost:8080/api/system/version
# Should show the new migration number
```

### 6. Commit

Commit these files together as a single change:
- `internal/database/migrations/NNNNN_description.sql`
- `internal/database/testdata/golden_schema.sql`
- Any code changes that depend on the new schema (repos, services, etc.)

### How goose tracks migrations

Goose creates a `goose_db_version` table in the database:

| id | version_id | is_applied | tstamp |
|----|-----------|------------|--------|
| 1  | 1         | true       | ...    |
| 2  | 2         | true       | ...    |

On startup, `Migrate()` calls `goose.Up()` which:
1. Reads the `goose_db_version` table to find already-applied migrations
2. Scans `migrations/` for any files with a higher version number
3. Applies only the new ones, in order
4. Records each applied migration in `goose_db_version`

If the database is brand new (no `goose_db_version` table), goose creates the table and runs all migrations from scratch.

### Rolling back a migration

Rolling back is a manual operation — it does **not** happen automatically on startup:

```bash
# Install the goose CLI (one-time)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Roll back the last migration
goose -dir internal/database/migrations sqlite3 ./data/portfolio_manager.db down

# Roll back to a specific version
goose -dir internal/database/migrations sqlite3 ./data/portfolio_manager.db down-to 1
```

After rolling back, regenerate the golden file and test:
```bash
go test ./internal/database/... -update-golden
go test ./...
```

### Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `TestMigrate_SchemaMatchesGoldenFile` fails | Migration changed or golden file stale | Run `go test -update-golden`, review diff |
| `no such table` in repo tests | Migration missing a table | Check migration SQL, compare with golden file |
| `goose_db_version` has gaps | Migration file was deleted/renumbered | Never delete or renumber merged migrations |
| Down migration fails on SQLite | Used `DROP COLUMN` on SQLite <3.35 | Use create-copy-drop-rename pattern |
| Duplicate rows in `system_setting` | Initial migration missing `INSERT OR IGNORE` | Use `INSERT OR IGNORE` for seed data |
