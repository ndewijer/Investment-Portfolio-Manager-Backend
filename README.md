# Investment Portfolio Manager - Backend

A Go backend for managing investment fund portfolios, transactions, dividend tracking, and IBKR Flex integration. Built with Chi router and SQLite.

## Quick Start

### Prerequisites

- Go 1.26+
- SQLite database (created automatically on first run)

### Run Locally

```bash
cp .env.example .env    # configure as needed
make deps               # download dependencies
make run                # start on http://localhost:5000
```

### Docker

```bash
docker build -t ipm-backend .
docker run -p 5000:5000 -v ./data:/data/db ipm-backend
```

### Verify

```bash
curl http://localhost:5000/api/system/health
# {"status":"healthy","database":"connected"}
```

## Features

- Portfolio management with fund allocation tracking
- Transaction recording (buy/sell) with realized gain/loss calculation
- Cash and stock dividend processing
- Automated daily fund price updates via Yahoo Finance
- IBKR Flex integration for automatic transaction imports
- Materialized views for portfolio summary/history performance
- Currency conversion (EUR/USD)
- CSV import/export for fund prices and transactions
- Structured logging with configurable levels
- Automatic database migrations via Goose

## API

All endpoints are served under `/api`. See [docs/API.md](docs/API.md) for the full reference.

| Namespace      | Endpoints | Description                                    |
|----------------|-----------|------------------------------------------------|
| `/system`      | 2         | Health check, version                          |
| `/portfolio`   | 13        | CRUD, archive, fund assignments, summary       |
| `/fund`        | 11        | CRUD, price history, symbol lookup             |
| `/transaction` | 6         | CRUD, per-portfolio listing                    |
| `/dividend`    | 7         | CRUD, per-portfolio and per-fund listing       |
| `/ibkr`        | 19        | Config, import, inbox, allocation, matching    |
| `/developer`   | 13        | Logs, settings, CSV import, exchange rates     |

## Documentation

- [API Reference](docs/API.md) - Complete endpoint documentation
- [Architecture](docs/ARCHITECTURE.md) - System design and layered architecture
- [Development](docs/DEVELOPMENT.md) - Local development setup and workflow
- [Configuration](docs/CONFIGURATION.md) - Environment variables and settings

### Learning Notes

The notes from building this project are preserved in [docs/learning/](docs/learning/).

## Tech Stack

- **Language:** Go 1.26+
- **Router:** [Chi](https://github.com/go-chi/chi) (stdlib-compatible)
- **Database:** SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO)
- **Migrations:** [Goose](https://github.com/pressly/goose)
- **Scheduling:** [robfig/cron](https://github.com/robfig/cron)
- **Testing:** Go testing + testify/assert

## License

Apache License, Version 2.0
