# Investment Portfolio Manager - Go Backend

A personal learning project rebuilding the Investment Portfolio Manager backend in Go. This project is part of my journey to learn Go by reimplementing a production Python/Flask backend that manages investment fund portfolios, transactions, and dividend tracking.

**Important:** This backend is being built **manually by me, not AI-generated**. Every line of code is written to understand Go fundamentals, patterns, and best practices. The implementation follows a phased approach starting with raw `database/sql` to learn the foundations before migrating to modern tools like `sqlc` and Atlas.

## Project Status
🚧 **In Active Development** - 100% complete (72/72 endpoints implemented)

This is a ground-up rewrite of the [Investment Portfolio Manager backend](https://github.com/ndewijer/Investment-Portfolio-Manager) from Python/Flask to Go. The goal is to achieve feature parity while learning Go idioms, patterns, and ecosystem.

### Tech Stack

- **Language:** Go 1.23+
- **Web Framework:** Chi router (stdlib-compatible)
- **Database Driver:** modernc.org/sqlite (pure Go, no CGO)
- **Database Access:**
  - Phase 1-2: `database/sql` (learning fundamentals)
  - Phase 3+: `sqlc` + Atlas (type-safe code generation)
- **Testing:** Go testing + testify/assert
- **Logging:** Structured logging with levels and categories

## API Implementation Status

This backend aims to replicate all 73 endpoints from the Python backend. Below is the current implementation status:

```
/api
├── /system (2/2 endpoints) ✅
│   ├── GET    /health                          ✅ Health check
│   └── GET    /version                         ✅ Version information
│
├── /portfolio (13/13 endpoints) ✅
│   ├── GET    /                                ✅ List all portfolios
│   ├── POST   /                                ✅ Create portfolio
│   ├── GET    /{id}                            ✅ Get portfolio by ID
│   ├── PUT    /{id}                            ✅ Update portfolio
│   ├── DELETE /{id}                            ✅ Delete portfolio
│   ├── POST   /{id}/archive                    ✅ Archive portfolio
│   ├── POST   /{id}/unarchive                  ✅ Unarchive portfolio
│   ├── GET    /summary                         ✅ Portfolio summary
│   ├── GET    /history                         ✅ Portfolio history
│   ├── GET    /{id}/fund-history               ✅ MOVED TO FUND/HISTORY/
│   ├── GET    /funds/{portfolioID}             ✅ Portfolio funds per ID
│   ├── POST   /fund                            ✅ Add fund to portfolio
│   └── DELETE /fund/{id}                       ✅ Remove fund from portfolio
│
├── /fund (13/13 endpoints) ✅
│   ├── GET         /                           ✅ Get all funds
│   ├── POST        /                           ✅ Create a new fund
│   ├── GET         /{id}                       ✅ Get fund details
│   ├── PUT         /{id}                       ✅ Update fund information
│   ├── DELETE      /{id}                       ✅ Delete a fund
│   ├── GET         /fund-prices/{id}           ✅ Get price history for a fund
│   ├── POST        /fund-prices/{id}/update    ✅ Update fund prices from external source
│   ├── GET         /history/{portfolioID}      ✅ Get historical fund values for a portfolio
│   ├── GET         /symbol/{symbol}            ✅ Get information about a trading symbol
│   ├── POST        /update-all-prices          ✅ Update prices for all funds
│   ├── GET         /{id}/check-usage           ✅ Check if fund is being used
│   ├── POST        /{id}/price/historical      ⬜ SKIPPING - Covered by /fund-prices/{id}/update for now - Update historical prices for a fund
│   └── POST        /{id}/price/today           ⬜ SKIPPING - Covered by /fund-prices/{id}/update for now - Update today's price for a fund
|
├── /transaction (6/6 endpoints) ✅
│   ├── GET    /                                ✅ List all transactions
│   ├── POST   /                                ✅ Create transaction
│   ├── GET    /{id}                            ✅ Get transaction by ID
│   ├── PUT    /{id}                            ✅ Update transaction
│   ├── DELETE /{id}                            ✅ Delete transaction
│   └── GET    /portfolio/{portfolioID}         ✅ Get transaction by ID
│
├── /dividend (7/7 endpoints) ✅
│   ├── GET    /                                ✅ List all dividends
│   ├── POST   /                                ✅ Create dividend
│   ├── GET    /{id}                            ✅ Get dividend by ID
|   |── GET    /portfolio/{portfolioId}         ✅ Get dividends per portfolioID
|   |── GET    /fund/{fundId}                   ✅ Get dividends per FundID
│   ├── PUT    /{id}                            ✅ Update dividend
│   └── DELETE /{id}                            ✅ Delete dividend
│
├── /ibkr (19/19 endpoints) ✅
│   ├── POST    /config                                    ✅ Create or update IBKR configuration
│   ├── GET     /config                                    ✅ Get IBKR configuration status
│   ├── DELETE  /config                                    ✅ Delete IBKR configuration
│   ├── POST    /config/test                               ✅ Test IBKR connection with provided credentials
│   ├── GET     /dividend/pending                          ✅ Get pending dividend records for matching
│   ├── POST    /import                                    ✅ Trigger IBKR import mechanism
│   ├── GET     /inbox                                     ✅ List IBKR imported transactions
│   ├── POST    /inbox/bulk-allocate                       ✅ Allocate multiple IBKR transactions with same allocations
│   ├── GET     /inbox/count                               ✅ Get count of IBKR transactions
│   ├── GET     /inbox/{transactionId}                     ✅ Get IBKR transaction details
│   ├── DELETE  /inbox/{transactionId}                     ✅ Delete IBKR transaction
│   ├── POST    /inbox/{transactionId}/allocate            ✅ Allocate IBKR transaction to portfolios
│   ├── GET     /inbox/{transactionId}/allocations         ✅ Get allocation details for a processed IBKR transaction
│   ├── PUT     /inbox/{transactionId}/allocations         ✅ Modify allocation percentages for a processed IBKR transaction
│   ├── GET     /inbox/{transactionId}/eligible-portfolios ✅ Get eligible portfolios for allocating this transaction
│   ├── POST    /inbox/{transactionId}/ignore              ✅ Mark IBKR transaction as ignored
│   ├── POST    /inbox/{transactionId}/match-dividend      ✅ Match IBKR dividend transaction to existing dividend records
│   ├── POST    /inbox/{transactionId}/unallocate          ✅ Unallocate a processed IBKR transaction
│   └── GET     /portfolios                                ✅ Get available portfolios for transaction allocation
│
└── /developer (12/12 endpoints) ✅
    ├── GET /csv/fund-prices/template     ✅ Get CSV template for fund price import
    ├── GET /csv/transactions/template    ✅ Get CSV template for transaction import
    ├── GET /exchange-rate                ✅ Get exchange rate for currency pair
    ├── POST /exchange-rate               ✅ Set exchange rate for currency pair
    ├── GET /fund-price                   ✅ Get fund price for specific date
    ├── POST /fund-price                  ✅ Set fund price for specific date
    ├── POST /import-fund-prices          ✅ Import fund prices from CSV file
    ├── POST /import-transactions         ✅ Import transactions from CSV file
    ├── DELETE /logs                      ✅ Clear all system logs
    ├── GET /logs                         ✅ Get system logs with cursor-based pagination
    ├── GET /system-settings/logging      ✅ Get current logging configuration settings
    └── PUT /system-settings/logging      ✅ Update logging configuration settings

Legend: ✅ Implemented | 🚧 In Progress | ⬜ Planned
Overall Progress: 72/72 endpoints (100%)
```

## Quick Start

### Prerequisites

- Go 1.23 or higher
- SQLite database from the Python backend (or create new)

### Setup

```bash
# Clone the repository
git clone https://github.com/ndewijer/Investment-Portfolio-Manager-Backend.git
cd Investment-Portfolio-Manager-Backend

# Create environment file
cp .env.example .env
# Edit .env with your configuration

# Install dependencies
go mod download

# Run the server
make run
# or
go run cmd/server/main.go
```

The server will start on `http://localhost:5001` by default.

### Testing the Health Check

```bash
curl http://localhost:5001/api/system/health
```

Expected response:
```json
{
  "status": "healthy",
  "database": "connected"
}
```

### Development Commands

```bash
make run          # Run development server
make test         # Run all tests
make coverage     # Generate coverage report
make build        # Build binary
make lint         # Run linter
make fmt          # Format code
make deps         # Download dependencies
```

## Project Structure

```
Investment-Portfolio-Manager-Backend/
├── cmd/
│   └── server/
│       └── main.go                   # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/                 # HTTP request handlers
│   │   ├── middleware/               # HTTP middleware (CORS, logging)
│   │   ├── response.go               # Response helpers
│   │   └── router.go                 # Route definitions
│   ├── config/
│   │   └── config.go                 # Configuration management
│   ├── database/
│   │   └── database.go               # Database connection
│   ├── service/                      # Business logic layer
│   │   ├── system_service.go
│   │   ├── portfolio_service.go
│   │   ├── portfolio_data_loader.go
│   │   ├── fund_service.go
│   │   ├── transaction_service.go
│   │   ├── dividend_service.go
│   │   ├── ibkr_service.go
│   │   ├── developer_service.go
│   │   ├── materialized_service.go
│   │   └── realizedGainLoss_service.go
│   ├── apperrors/                    # Centralized error types
│   ├── validation/                   # Input validation helpers
│   ├── yahoo/                        # Yahoo Finance integration
│   ├── testutil/                     # Test utilities and helpers
│   └── version/
│       └── version.go                # Version information
├── data/
│   └── portfolio_manager.db          # SQLite database
├── docs/                             # Documentation
│   ├── ARCHITECTURE_DECISIONS.md     # Why specific choices were made
│   ├── GO_IMPLEMENTATION_PLAN.md     # Detailed implementation roadmap
│   ├── GO_POINTERS_EXPLAINED.md      # Go pointers guide
│   ├── GO_TESTING_GUIDE.md           # Testing patterns
│   └── SETUP_EXPLAINED.md            # Detailed setup walkthrough
├── .env.example                      # Environment template
├── Makefile                          # Build automation
├── go.mod                            # Go module definition
└── README.md                         # This file
```

## Documentation

### Learning Resources

- **[Architecture Decisions](docs/ARCHITECTURE_DECISIONS.md)** - Deep dive into why specific technologies and patterns were chosen
- **[Implementation Plan](docs/GO_IMPLEMENTATION_PLAN.md)** - Phased roadmap for building the backend
- **[Setup Explained](docs/SETUP_EXPLAINED.md)** - Detailed explanation of how everything works
- **[Go Pointers Guide](docs/GO_POINTERS_EXPLAINED.md)** - Understanding pointers in Go
- **[Testing Guide](docs/GO_TESTING_GUIDE.md)** - Testing patterns and practices
- **[Testing Quick Reference](docs/TESTING_QUICK_REFERENCE.md)** - Quick reference for test commands and builders
- **[Endpoint Testing Guide](docs/ENDPOINT_TESTING_GUIDE.md)** - Patterns for testing HTTP endpoints
- **[Repository Transaction Patterns](docs/REPOSITORY_TRANSACTION_PATTERNS.md)** - Transaction management patterns
- **[Write Operations Guide](docs/WRITE_OPERATIONS_GUIDE.md)** - Patterns for POST/PUT/DELETE handlers
- **[Go Best Practices](docs/GO_BEST_PRACTICES.md)** - Project-specific Go conventions
- **[Logging Implementation](docs/LOGGING_IMPLEMENTATION.md)** - Structured logging setup and usage
- **[Tooling Recommendations](docs/TOOLING_RECOMMENDATIONS.md)** - Recommended tools and CI/CD setup

### Architecture Overview

The application follows a clean layered architecture:

```
HTTP Request → Router → Handler → Service → Database
                ↓
          Middleware (logging, CORS, recovery)
```

**Layers:**
- **API Layer** (`internal/api/`): HTTP concerns (routing, middleware, request/response)
- **Service Layer** (`internal/service/`): Business logic, validation, orchestration
- **Database Layer** (`internal/database/`): Data access patterns

This separation ensures:
- Testable components (can test services without HTTP)
- Reusable logic (services can be called from CLI, workers, etc.)
- Clear dependencies (API → Service → Database, never backwards)

## Development Approach

### Phase 1-2: Learning with database/sql ✅ Complete

Built with raw `database/sql` to understand:
- How query execution works
- Pointer semantics and scanning
- Transaction management
- Error handling patterns
- NULL value handling

**Benefits:**
- Deep understanding of database operations
- Appreciation for what code generation solves
- Foundation for all Go database work

### Phase 3: Migration to sqlc + Atlas (Planned)

After gaining solid understanding, migrating to:
- **sqlc**: Type-safe code generation from SQL
- **Atlas**: Database schema management and migrations

**Benefits:**
- Compile-time SQL validation
- Type-safe queries
- Reduced boilerplate
- Better maintainability

See [Implementation Plan](docs/GO_IMPLEMENTATION_PLAN.md) for detailed migration strategy.

## Key Principles

### 1. Explicit Over Magic
- Prefer readable code over clever abstractions
- Avoid over-engineering
- Keep it simple and clear

### 2. Standard Library First
- Use `database/sql` before ORMs
- Use `net/http` patterns with Chi router
- Leverage Go's excellent standard library

### 3. Learn By Doing
- Write code manually to understand patterns
- Feel the pain points before adding tools
- Build appreciation for productivity tools

### 4. Production-Ready Patterns
- Structured logging with categories
- Proper error handling and wrapping
- Comprehensive testing (95%+ coverage goal)
- Service layer pattern for testability

## Testing

Tests follow Go conventions with comprehensive coverage:

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific package
go test ./internal/service/...
```

**Test Organization:**
- Unit tests alongside code (`*_test.go`)
- Integration tests for HTTP endpoints
- Test utilities in `internal/testutil/`
- 95%+ coverage target

## Configuration

Configuration is managed through environment variables with `.env` file support:

```env
# Server
SERVER_PORT=5001
SERVER_HOST=localhost

# Database
DB_PATH=./data/portfolio_manager.db

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

See `.env.example` for all available options.

## Contributing

This is a personal learning project, but feedback and suggestions are welcome! Feel free to:
- Open issues for bugs or questions
- Suggest improvements to code or documentation
- Share Go best practices I might have missed

## Relationship to Original Project

This backend is designed to be a drop-in replacement for the [Python/Flask backend](https://github.com/ndewijer/Investment-Portfolio-Manager). It:
- Uses the same SQLite database schema
- Implements the same REST API endpoints
- Returns the same JSON response formats
- Works with the existing React frontend

The Python backend can run alongside this Go backend during development for comparison and testing.

## License

Apache License, Version 2.0

## Acknowledgments

This project is a learning exercise inspired by the need to consolidate multiple Excel spreadsheets into a single application. The original Python backend was built with LLM assistance; this Go rewrite is being built manually to learn Go from the ground up.

---

**Current Focus:** Completing IBKR and developer endpoints (remaining 25% of API surface).

**Next Steps:** Full IBKR import flow, remaining developer utilities, then Phase 3 migration to sqlc + Atlas.
