# Investment Portfolio Manager - Go Backend

This is a Go rewrite of the Python/Flask backend, designed to maintain API compatibility with the existing React frontend.

## Project Structure

```
Investment-Portfolio-Manager-Backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP handlers
│   │   │   └── system.go       # System/health handlers
│   │   ├── middleware/          # HTTP middleware
│   │   │   ├── cors.go         # CORS configuration
│   │   │   └── logger.go       # Request logging
│   │   ├── response.go          # Response helpers
│   │   └── router.go            # Chi router setup
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   └── database.go          # Database connection
│   └── service/
│       └── system_service.go    # System service
├── data/                        # Database location
├── .env.example                 # Environment variables template
├── Makefile                     # Build commands
└── go.mod                       # Go module definition
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite database from Python backend at `data/portfolio_manager.db`

### Installation

1. Clone the repository (already done!)

2. Copy the environment file:
```bash
cp .env.example .env
```

3. Download dependencies:
```bash
make deps
```

### Running the Application

```bash
make run
```

The server will start on `http://localhost:5001`

### Testing the Health Check

Open your browser or use curl:

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

## Available Commands

- `make run` - Run the application
- `make build` - Build the application binary
- `make test` - Run tests
- `make coverage` - Run tests with coverage report
- `make clean` - Clean build artifacts
- `make deps` - Download and tidy dependencies
- `make fmt` - Format code
- `make help` - Display all available commands

## Development

This project follows the implementation plan in `docs/GO_IMPLEMENTATION_PLAN.md`.

**Current Status:** Phase 1 - Health Check Endpoint ✅

### Next Steps

1. Add more system endpoints (version info)
2. Implement Portfolio namespace
3. Add comprehensive testing
4. Migrate to sqlc + Atlas (Phase 3)

## Tech Stack

- **Web Framework:** Chi router (stdlib-compatible)
- **Database:** modernc.org/sqlite (pure Go)
- **Configuration:** godotenv (.env file support)
- **Testing:** Go testing (to be added)

## API Compatibility

This Go backend maintains the same API contract as the Python backend:
- Same endpoints
- Same request/response formats
- Same error handling

## Learning Resources

- [Go Implementation Plan](docs/GO_IMPLEMENTATION_PLAN.md) - Complete implementation guide
- [Python Backend API Docs](https://github.com/ndewijer/Investment-Portfolio-Manager/docs/API_DOCUMENTATION.md) - API reference
