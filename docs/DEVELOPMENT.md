# Development

## Prerequisites

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/) (for linting)
- [pre-commit](https://pre-commit.com/) (optional, for git hooks)

## Setup

```bash
git clone https://github.com/ndewijer/Investment-Portfolio-Manager-Backend.git
cd Investment-Portfolio-Manager-Backend

cp .env.example .env     # edit as needed
make deps                # download dependencies
make run                 # start server
```

The server runs on `http://localhost:5000` by default. The database is created automatically on first run with all migrations applied.

## Make Targets

### Build & Run

| Target   | Description                  |
|----------|------------------------------|
| `run`    | Run the application          |
| `build`  | Build binary to `bin/server` |

### Testing

| Target         | Description                              |
|----------------|------------------------------------------|
| `test`         | Run all tests with race detector         |
| `test-short`   | Skip slow tests                          |
| `test-verbose` | Verbose output                           |

### Coverage

| Target                 | Description                                     |
|------------------------|-------------------------------------------------|
| `coverage`             | Run tests with coverage summary                 |
| `coverage-html`        | Generate HTML coverage report                   |
| `coverage-func`        | Per-function coverage                           |
| `coverage-by-file`     | Per-file coverage sorted by %                   |
| `coverage-gaps`        | Files below 100% coverage                       |
| `coverage-threshold`   | Check coverage meets 75% threshold (CI)         |
| `coverage-full-html`   | Comprehensive HTML report across all packages   |

### Code Quality

| Target               | Description                          |
|----------------------|--------------------------------------|
| `fmt`                | Format code                          |
| `lint`               | Run golangci-lint                    |
| `lint-fix`           | Run linter with auto-fix             |
| `govulncheck`        | Security vulnerability scan          |
| `pre-commit-install` | Install pre-commit git hooks         |
| `pre-commit-run`     | Run pre-commit on all files          |
| `ci-local`           | Run full CI pipeline locally         |

### Maintenance

| Target  | Description                              |
|---------|------------------------------------------|
| `clean` | Remove build artifacts and coverage files|
| `deps`  | Download and tidy dependencies           |

## Testing

Tests use real SQLite databases (in-memory, per-test) — no mocks. Each test gets an isolated database instance via `testutil.SetupTestDB()`.

```bash
make test                              # all tests
go test ./internal/service/...         # specific package
go test -run TestCreatePortfolio ./... # specific test
```

## Docker

The Dockerfile uses a multi-stage build:

1. **Builder stage** — compiles the Go binary with version injection
2. **Runtime stage** — minimal Alpine image with just the binary

```bash
docker build -t ipm-backend .
docker run -p 5000:5000 -v ./data:/data/db ipm-backend
```

Environment variables in Docker:
- `DB_DIR` — directory for the database file (default: `/data/db`)
- `LOG_DIR` — directory for log files (default: `/data/logs`)
- `DOMAIN` — used for CORS origin generation
- `IBKR_ENCRYPTION_KEY` — optional, auto-generated if not set
- `INTERNAL_API_KEY` — for protected endpoints (e.g., update-all-prices)

## Database

The application uses SQLite with automatic migrations on startup. The database file lives at the path configured by `DB_PATH` (local) or `DB_DIR` (Docker).

To inspect the database:

```bash
sqlite3 ./data/portfolio_manager.db ".tables"
sqlite3 ./data/portfolio_manager.db ".schema funds"
```
