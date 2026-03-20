# Architecture

## Layered Design

```
HTTP Request → Router → Handler → Service → Repository → Database
                ↓
          Middleware (logging, CORS, recovery, UUID validation)
```

**Handler** — Parses HTTP requests, calls services, writes responses. No business logic.

**Service** — Business logic, validation, orchestration. Services may call other services and multiple repositories. Owns transaction boundaries for write operations.

**Repository** — Data access. Each repository maps to one database table/domain. Accepts `*sql.DB` or `*sql.Tx` so services can compose multiple repository calls in a single transaction.

**Middleware** — Cross-cutting concerns: request logging, CORS, panic recovery, UUID path parameter validation.

## Project Layout

```
cmd/server/main.go          Entry point — wires dependencies, starts server
internal/
  api/
    router.go               Route definitions (Chi)
    handlers/                HTTP handlers, one file per domain
    middleware/              CORS, logging, UUID validation, API key auth
    request/                Request parsing types, one file per domain
    response/               Shared response helpers
  model/                    Domain model types
  service/                  Business logic, one file per domain
  repository/               Database access, one file per domain
  database/
    database.go             Connection setup
    migrate.go              Migration runner
    migrations/             Goose SQL migration files
  config/                   Environment-based configuration
  logging/                  Structured logging with DB-configurable levels
  validation/               Input validation helpers
  apperrors/                Typed application errors
  yahoo/                    Yahoo Finance price client
  ibkr/                     IBKR Flex report client
  version/                  Build-time version injection
  testutil/                 Shared test helpers and DB setup
data/
  portfolio_manager.db      SQLite database (gitignored)
```

## Key Design Decisions

### Pure Go SQLite

Uses `modernc.org/sqlite` — a pure Go SQLite implementation requiring no CGO. This simplifies cross-compilation and Docker builds at the cost of slightly lower performance than CGO-based drivers. For a single-user portfolio app, this is the right trade-off.

### Repository + Service Pattern

Repositories handle raw SQL. Services own the business rules and transaction boundaries. This keeps SQL out of handlers and makes services testable with real database calls (no mocks).

Write operations follow a consistent pattern:
1. Service begins transaction
2. Service calls one or more repositories with the `*sql.Tx`
3. Service commits or rolls back

### Materialized Views

Portfolio summary and history endpoints are backed by in-application materialized views (not SQLite views). These are computed on demand and cached, then invalidated when transactions, dividends, or fund prices change. See `service/materialized_service.go`.

### Scheduled Tasks

Two cron jobs run in-process via `robfig/cron`:
- **Fund price update** — weekdays at 00:55 UTC
- **IBKR import** — Tue–Sat at 05:30–07:30 UTC (retries hourly)

Both use `SkipIfStillRunning` to prevent overlap and have 15-minute timeouts.

### Encryption

IBKR credentials are encrypted at rest using Fernet symmetric encryption. The key is resolved in priority order: `IBKR_ENCRYPTION_KEY` env var → `data/.ibkr_encryption_key` file → auto-generated on first run.

### Database Migrations

Managed by Goose, embedded in the binary. Migrations run automatically on startup. The base migration (`162_base.sql`) establishes the full schema matching the original Python backend's database.
