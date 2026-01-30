# Go Testing Guide: From pytest to Go's testing

This guide shows how to replicate the Investment Portfolio Manager's Python/pytest testing patterns in Go.

## Table of Contents

1. [Overview: pytest vs Go testing](#overview-pytest-vs-go-testing)
2. [Project Structure](#project-structure)
3. [Database Testing](#database-testing)
4. [Test Organization Patterns](#test-organization-patterns)
5. [Fixtures in Go](#fixtures-in-go)
6. [Factory Pattern](#factory-pattern)
7. [Test Helpers](#test-helpers)
8. [Coverage Requirements](#coverage-requirements)
9. [Running Tests](#running-tests)
10. [Examples](#examples)

---

## Overview: pytest vs Go testing

### Python/pytest Pattern

```python
# Python: conftest.py with fixtures
@pytest.fixture(scope="session")
def app():
    """Flask app with test configuration"""
    return create_app(TEST_CONFIG)

@pytest.fixture(scope="function")
def db_session(app_context):
    """Clean database for each test"""
    clear_database()
    yield db.session

# Python: test file
def test_get_all_portfolios(db_session, client):
    """Test retrieving portfolios"""
    # Create test data
    portfolio = PortfolioFactory()

    # Make request
    response = client.get('/api/portfolio')

    # Assert
    assert response.status_code == 200
```

### Go Equivalent Pattern

```go
// Go: setup helpers in test file or _test package
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db := createTestDatabase(t)
    t.Cleanup(func() { cleanupTestDatabase(t, db) })
    return db
}

// Go: test file
func TestGetAllPortfolios(t *testing.T) {
    // Setup (like fixtures)
    db := setupTestDB(t)
    service := service.NewPortfolioService(db)

    // Create test data
    portfolio := createTestPortfolio(t, db, "Test Portfolio")

    // Execute
    portfolios, err := service.GetAllPortfolios()

    // Assert
    require.NoError(t, err)
    assert.Len(t, portfolios, 1)
    assert.Equal(t, "Test Portfolio", portfolios[0].Name)
}
```

### Key Differences

| Concept | Python/pytest | Go |
|---------|---------------|-----|
| **Test Runner** | pytest | `go test` (built-in) |
| **Fixtures** | `@pytest.fixture` decorators | Setup functions + `t.Cleanup()` |
| **Assertions** | `assert x == y` | `testify/assert` or `if/t.Errorf()` |
| **Test Discovery** | `test_*.py` files | `*_test.go` files |
| **Mocking** | pytest-mock, responses | Manual interfaces or testify/mock |
| **Factories** | factory-boy | Manual builder functions |
| **Coverage** | pytest-cov | `go test -cover` |
| **Scoping** | session/function/class | Handled manually with `t.Helper()` |

---

## Project Structure

### Python Structure (Current)

```
backend/
├── app/
│   ├── services/
│   │   └── portfolio_service.py
│   └── api/
│       └── portfolio_namespace.py
└── tests/
    ├── conftest.py              # Global fixtures
    ├── factories.py             # Object factories
    ├── test_helpers.py          # Helper utilities
    ├── test_config.py           # Test configuration
    ├── services/
    │   └── test_portfolio_service.py
    └── api/
        └── test_portfolio_routes.py
```

### Go Structure (Recommended)

```
Investment-Portfolio-Manager-Backend/
├── internal/
│   ├── service/
│   │   ├── portfolio_service.go
│   │   └── portfolio_service_test.go      # Tests next to code
│   ├── api/
│   │   └── handlers/
│   │       ├── portfolios.go
│   │       └── portfolios_test.go
│   └── testutil/                          # Shared test utilities
│       ├── database.go                    # DB setup/teardown
│       ├── factories.go                   # Test data builders
│       └── helpers.go                     # ID generation, etc.
├── test/
│   └── integration/                       # Integration tests
│       └── portfolio_integration_test.go
└── docs/
    └── testing/
        ├── GO_TESTING_GUIDE.md            # This file
        └── SERVICE_TESTS.md               # Service test documentation
```

**Go Convention:**
- Tests live **next to the code** they test (`portfolio_service_test.go`)
- Use `_test.go` suffix (required by `go test`)
- Integration tests can go in separate `test/` directory
- Shared test utilities in `internal/testutil/` package

---

## Database Testing

### Python Approach (Current)

```python
# test_config.py
TEST_DATABASE_PATH = "backend/data/db/test_portfolio_manager.db"

TEST_CONFIG = {
    "TESTING": True,
    "SQLALCHEMY_DATABASE_URI": f"sqlite:///{TEST_DATABASE_PATH}",
}

# conftest.py
@pytest.fixture(scope="session")
def app():
    app = create_app(TEST_CONFIG)
    with app.app_context():
        db.create_all()
    yield app
    with app.app_context():
        db.drop_all()
    cleanup_test_database()

@pytest.fixture(scope="function")
def db_session(app_context):
    # Clear all tables before each test
    clear_database()
    yield db.session
```

### Go Approach

**Option 1: In-Memory SQLite (Fast, Isolated)**

```go
// internal/testutil/database.go
package testutil

import (
    "database/sql"
    "testing"
    _ "modernc.org/sqlite"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    // In-memory database (destroyed when connection closes)
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }

    // Apply schema
    if err := createSchema(db); err != nil {
        t.Fatalf("Failed to create schema: %v", err)
    }

    // Cleanup when test ends
    t.Cleanup(func() {
        db.Close()
    })

    return db
}

// createSchema creates all tables
func createSchema(db *sql.DB) error {
    schema := `
        CREATE TABLE IF NOT EXISTS portfolio (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            description TEXT,
            is_archived BOOLEAN DEFAULT 0,
            exclude_from_overview BOOLEAN DEFAULT 0
        );

        CREATE TABLE IF NOT EXISTS fund (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            isin TEXT UNIQUE,
            symbol TEXT,
            currency TEXT
        );

        -- Add all other tables...
    `

    _, err := db.Exec(schema)
    return err
}

// CleanDatabase truncates all tables (for reuse between tests)
func CleanDatabase(t *testing.T, db *sql.DB) {
    t.Helper()

    // Order matters due to foreign keys
    tables := []string{
        "transaction",
        "dividend",
        "fund_price",
        "fund",
        "portfolio",
    }

    for _, table := range tables {
        _, err := db.Exec("DELETE FROM " + table)
        if err != nil {
            t.Fatalf("Failed to clean table %s: %v", table, err)
        }
    }
}
```

**Option 2: Temporary File Database**

```go
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    // Create temp database file
    tmpFile := filepath.Join(t.TempDir(), "test.db")

    db, err := sql.Open("sqlite", tmpFile)
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }

    createSchema(db)

    t.Cleanup(func() {
        db.Close()
        // t.TempDir() automatically cleans up the file
    })

    return db
}
```

**Usage in Tests:**

```go
func TestPortfolioService_GetAll(t *testing.T) {
    // Each test gets a fresh database
    db := testutil.SetupTestDB(t)
    service := service.NewPortfolioService(db)

    // Test with clean database
    portfolios, err := service.GetAllPortfolios()
    require.NoError(t, err)
    assert.Empty(t, portfolios)
}
```

---

## Test Organization Patterns

### Python Pattern (Current)

```python
# tests/services/test_portfolio_service.py
class TestPortfolioRetrieval:
    def test_get_all_portfolios(self, db_session):
        """Verify retrieving all portfolios"""
        ...

    def test_get_portfolio_success(self, db_session):
        """Verify retrieving a single portfolio by ID"""
        ...

    def test_get_portfolio_not_found(self, db_session):
        """Verify error when portfolio doesn't exist"""
        ...

class TestPortfolioCreation:
    def test_create_portfolio(self, db_session):
        ...

    def test_create_portfolio_validation(self, db_session):
        ...
```

### Go Equivalent

Go doesn't have test classes, but we can use naming and subtests:

```go
// internal/service/portfolio_service_test.go
package service_test  // Use separate test package

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

// TestPortfolioRetrieval groups retrieval tests
func TestPortfolioRetrieval(t *testing.T) {
    t.Run("GetAllPortfolios_Empty", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolios, err := svc.GetAllPortfolios()

        // Assert
        require.NoError(t, err)
        assert.Empty(t, portfolios)
    })

    t.Run("GetAllPortfolios_ReturnsAll", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Create test data
        testutil.CreatePortfolio(t, db, "Portfolio 1")
        testutil.CreatePortfolio(t, db, "Portfolio 2")

        // Execute
        portfolios, err := svc.GetAllPortfolios()

        // Assert
        require.NoError(t, err)
        assert.Len(t, portfolios, 2)
    })

    t.Run("GetPortfolio_Success", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        created := testutil.CreatePortfolio(t, db, "Test Portfolio")

        // Execute
        portfolio, err := svc.GetPortfolio(created.ID)

        // Assert
        require.NoError(t, err)
        assert.Equal(t, "Test Portfolio", portfolio.Name)
    })

    t.Run("GetPortfolio_NotFound", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolio, err := svc.GetPortfolio("non-existent-id")

        // Assert
        assert.Error(t, err)
        assert.Nil(t, portfolio)
    })
}

// TestPortfolioCreation groups creation tests
func TestPortfolioCreation(t *testing.T) {
    t.Run("Create_Success", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolio, err := svc.CreatePortfolio("New Portfolio", "Description")

        // Assert
        require.NoError(t, err)
        assert.NotEmpty(t, portfolio.ID)
        assert.Equal(t, "New Portfolio", portfolio.Name)
        assert.Equal(t, "Description", portfolio.Description)
    })

    t.Run("Create_Validation_EmptyName", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolio, err := svc.CreatePortfolio("", "Description")

        // Assert
        assert.Error(t, err)
        assert.Nil(t, portfolio)
    })
}
```

**Benefits:**
- `t.Run()` creates subtests (like pytest classes)
- Each subtest is isolated
- Clear naming: `TestFeature/Scenario_ExpectedOutcome`
- Can run individual subtests: `go test -run TestPortfolioRetrieval/GetAllPortfolios_Empty`

---

## Fixtures in Go

### Python Fixtures (Current)

```python
# conftest.py
@pytest.fixture(scope="session")
def app():
    """Created once per test session"""
    return create_app(TEST_CONFIG)

@pytest.fixture(scope="function")
def db_session(app_context):
    """Created for each test function"""
    clear_database()
    yield db.session
```

### Go Equivalent: Setup Functions + t.Cleanup()

```go
// Session-scoped equivalent: package-level setup
func TestMain(m *testing.M) {
    // Setup before all tests
    setupGlobalResources()

    // Run tests
    code := m.Run()

    // Teardown after all tests
    teardownGlobalResources()

    os.Exit(code)
}

// Function-scoped equivalent: per-test setup
func setupTest(t *testing.T) (*sql.DB, *service.PortfolioService) {
    t.Helper()  // Marks this as a helper function

    // Setup
    db := testutil.SetupTestDB(t)
    svc := service.NewPortfolioService(db)

    // Cleanup registered automatically
    t.Cleanup(func() {
        // Cleanup code runs after test completes
        db.Close()
    })

    return db, svc
}

// Usage in tests
func TestSomething(t *testing.T) {
    db, svc := setupTest(t)  // Fresh setup

    // Test logic
    result, err := svc.DoSomething()
    assert.NoError(t, err)

    // t.Cleanup() automatically called after test
}
```

**t.Cleanup() vs defer:**

```go
// defer: runs when function exits
func TestWithDefer(t *testing.T) {
    db := setupDB()
    defer db.Close()  // Runs when TestWithDefer exits

    // Test logic
}

// t.Cleanup(): runs after test and all subtests complete
func TestWithCleanup(t *testing.T) {
    db := setupDB()
    t.Cleanup(func() {
        db.Close()  // Runs after all subtests finish
    })

    t.Run("Subtest1", func(t *testing.T) {
        // db still open
    })

    t.Run("Subtest2", func(t *testing.T) {
        // db still open
    })

    // Now db.Close() runs
}
```

---

## Factory Pattern

### Python Factories (Current)

```python
# tests/factories.py
class PortfolioFactory(BaseFactory):
    class Meta:
        model = Portfolio
        sqlalchemy_session = db.session

    name = Faker("company")
    description = Faker("sentence")
    is_archived = False

# Usage
portfolio = PortfolioFactory(name="Custom Name")
portfolio_with_funds = PortfolioFactory.create_batch(5)
```

### Go Builder Pattern (Implemented)

We have implemented **7 test builders** in `internal/testutil/factories.go`:

1. **PortfolioBuilder** - Creates portfolios
2. **FundBuilder** - Creates funds (stocks/ETFs)
3. **PortfolioFundBuilder** - Creates portfolio-fund relationships
4. **TransactionBuilder** - Creates buy/sell transactions
5. **FundPriceBuilder** - Creates fund prices
6. **DividendBuilder** - Creates dividend records
7. **RealizedGainLossBuilder** - Creates realized gain/loss records

#### Portfolio Builder Example

```go
// Basic usage with defaults
portfolio := testutil.NewPortfolio().Build(t, db)

// Customized portfolio
portfolio := testutil.NewPortfolio().
    WithName("Custom Portfolio").
    WithDescription("My description").
    Archived().
    Build(t, db)

// Convenience functions
p1 := testutil.CreatePortfolio(t, db, "Portfolio 1")
p2 := testutil.CreateArchivedPortfolio(t, db, "Old Portfolio")
portfolios := testutil.CreatePortfolios(t, db, 5)  // Create 5 at once
```

#### Fund Builder Example

```go
// Basic fund
fund := testutil.NewFund().Build(t, db)

// Customized fund
fund := testutil.NewFund().
    WithSymbol("AAPL").
    WithName("Apple Inc.").
    WithCurrency("USD").
    WithExchange("NASDAQ").
    Build(t, db)

// Convenience functions
fund := testutil.CreateFund(t, db, "AAPL")
funds := testutil.CreateFunds(t, db, 10)
```

#### Transaction Builder Example

```go
// Create portfolio-fund relationship first
portfolioFundID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

// Create buy transaction
testutil.NewTransaction(portfolioFundID).
    WithType("buy").
    WithShares(100).
    WithCostPerShare(10.0).
    WithDate(time.Now()).
    Build(t, db)

// Create sell transaction
testutil.NewTransaction(portfolioFundID).
    WithType("sell").
    WithShares(30).
    WithCostPerShare(15.0).
    Build(t, db)
```

#### Fund Price Builder Example

```go
// Add current price for a fund
testutil.NewFundPrice(fund.ID).
    WithPrice(12.0).
    WithDate(time.Now()).
    Build(t, db)
```

#### Dividend Builder Example

```go
// Create dividend without reinvestment
testutil.NewDividend(fund.ID, portfolioFundID).
    WithSharesOwned(100).
    WithDividendPerShare(0.50).
    Build(t, db)

// Create dividend with reinvestment
dividend := testutil.NewDividend(fund.ID, portfolioFundID).
    WithReinvestmentTransaction(txID).
    Build(t, db)
```

#### Realized Gain/Loss Builder Example

```go
// Record realized gain from sale
testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, transactionID).
    WithShares(30).
    WithCostBasis(300.0).
    WithSaleProceeds(450.0).
    WithDate(time.Now()).
    Build(t, db)
```

#### Complete Test Scenario Example

```go
func TestCompletePortfolioScenario(t *testing.T) {
    // Setup
    db := testutil.SetupTestDB(t)

    // Create portfolio and fund
    portfolio := testutil.CreatePortfolio(t, db, "My Portfolio")
    fund := testutil.CreateFund(t, db, "AAPL")
    pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

    // Buy 100 shares at $10
    testutil.NewTransaction(pfID).
        WithType("buy").
        WithShares(100).
        WithCostPerShare(10.0).
        Build(t, db)

    // Current price is $12
    testutil.NewFundPrice(fund.ID).
        WithPrice(12.0).
        Build(t, db)

    // Receive dividend
    testutil.NewDividend(fund.ID, pfID).
        WithSharesOwned(100).
        WithDividendPerShare(0.50).
        Build(t, db)

    // Sell 30 shares at $15
    sellTx := testutil.NewTransaction(pfID).
        WithType("sell").
        WithShares(30).
        WithCostPerShare(15.0).
        Build(t, db)

    // Record realized gain
    testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, sellTx.ID).
        WithShares(30).
        WithCostBasis(300.0).   // 30 * $10
        WithSaleProceeds(450.0). // 30 * $15
        Build(t, db)

    // Now test your service logic...
}
```

---

## Test Helpers

### Python Helpers (Current)

```python
# tests/test_helpers.py
def make_id() -> str:
    """Generate UUID string"""
    return str(uuid.uuid4())

def make_isin(prefix: str = "US") -> str:
    """Generate ISIN like US1A2B3C4D5E"""
    return f"{prefix}{uuid.uuid4().hex[:10].upper()}"

COMMON_CURRENCIES = ["USD", "EUR", "GBP"]
```

### Go Equivalent

```go
// internal/testutil/helpers.go
package testutil

import (
    "math/rand"
    "strings"

    "github.com/google/uuid"
)

// MakeID generates a UUID string
func MakeID() string {
    return uuid.New().String()
}

// MakeISIN generates ISIN like US1A2B3C4D5E
func MakeISIN(prefix string) string {
    if prefix == "" {
        prefix = "US"
    }
    return prefix + randomString(10)
}

// MakeSymbol generates ticker symbol like AAPL1A2B
func MakeSymbol(base string) string {
    return base + randomString(4)
}

// MakePortfolioName generates unique portfolio name
func MakePortfolioName(base string) string {
    return base + " " + randomString(6)
}

// randomString generates random alphanumeric string
func randomString(length int) string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[rand.Intn(len(charset))]
    }
    return string(b)
}

// Common test constants
var (
    CommonCurrencies = []string{"USD", "EUR", "GBP", "JPY", "CAD"}
    CommonExchanges  = []string{"NASDAQ", "NYSE", "LSE", "TSE"}
)
```

---

## Coverage Requirements

### Python Configuration (Current)

```toml
# pyproject.toml
[tool.pytest.ini_options]
addopts = [
    "--cov=backend/app/services",
    "--cov=backend/app/api",
    "--cov-fail-under=90",  # Enforce 90% minimum
]
```

### Go Configuration

**Run with coverage:**

```bash
# Generate coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Check coverage percentage
go test -cover ./...

# Fail if coverage below threshold (requires script)
./scripts/check-coverage.sh 90
```

**Coverage check script:**

```bash
#!/bin/bash
# scripts/check-coverage.sh

THRESHOLD=$1

# Run tests with coverage
go test -coverprofile=coverage.out ./... > /dev/null 2>&1

# Calculate total coverage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Coverage: ${COVERAGE}%"
echo "Threshold: ${THRESHOLD}%"

# Compare with threshold
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%"
    exit 1
else
    echo "✅ Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%"
    exit 0
fi
```

**Makefile integration:**

```makefile
# Makefile
.PHONY: test coverage coverage-check

test:
	go test -v -race ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

coverage-check:
	@./scripts/check-coverage.sh 90
```

---

## Running Tests

### Python (Current)

```bash
# Run all tests
pytest

# Run specific test file
pytest tests/services/test_portfolio_service.py

# Run specific test
pytest tests/services/test_portfolio_service.py::TestPortfolioRetrieval::test_get_all_portfolios

# Run with coverage
pytest --cov

# Run only service tests
pytest tests/services/

# Run with markers
pytest -m "not slow"
```

### Go Equivalent (Using Makefile)

We have a `Makefile` with convenient test commands:

```bash
# Run all tests with race detector
make test

# Run tests in short mode (skip slow tests)
make test-short

# Run tests with verbose output
make test-verbose

# Run tests with coverage summary
make coverage

# Generate HTML coverage report
make coverage-html

# Format code
make fmt

# Run linter
make lint

# See all available commands
make help
```

### Direct Go Commands

You can also run tests directly without make:

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./internal/service/
go test ./internal/api/handlers/

# Run specific test function
go test -run TestPortfolioHandler_PortfolioSummary ./internal/api/handlers/

# Run specific subtest
go test -run TestPortfolioHandler_PortfolioSummary/returns_empty_array ./internal/api/handlers/

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Race detector (finds concurrency bugs)
go test -race ./...

# Parallel execution
go test -parallel 4 ./...

# Short mode (skip slow tests)
go test -short ./...
```

---

## Examples

### Service Test Example

```go
// internal/service/portfolio_service_test.go
package service_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

func TestPortfolioService_GetAllPortfolios(t *testing.T) {
    t.Run("returns empty list when no portfolios exist", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolios, err := svc.GetAllPortfolios()

        // Assert
        require.NoError(t, err, "GetAllPortfolios should not return error")
        assert.Empty(t, portfolios, "should return empty list")
    })

    t.Run("returns all portfolios including archived", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Create test data
        testutil.CreatePortfolio(t, db, "Active Portfolio")
        testutil.NewPortfolio().
            WithName("Archived Portfolio").
            Archived().
            Build(t, db)

        // Execute
        portfolios, err := svc.GetAllPortfolios()

        // Assert
        require.NoError(t, err)
        assert.Len(t, portfolios, 2, "should return both portfolios")
    })
}

func TestPortfolioService_CreatePortfolio(t *testing.T) {
    t.Run("creates portfolio with all fields", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolio, err := svc.CreatePortfolio("New Portfolio", "Test description")

        // Assert
        require.NoError(t, err)
        assert.NotEmpty(t, portfolio.ID, "should generate ID")
        assert.Equal(t, "New Portfolio", portfolio.Name)
        assert.Equal(t, "Test description", portfolio.Description)
        assert.False(t, portfolio.IsArchived, "should not be archived by default")
    })

    t.Run("returns error when name is empty", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)

        // Execute
        portfolio, err := svc.CreatePortfolio("", "Description")

        // Assert
        assert.Error(t, err, "should return validation error")
        assert.Nil(t, portfolio)
        assert.Contains(t, err.Error(), "name is required")
    })
}
```

### Handler Test Example

```go
// internal/api/handlers/portfolio_test.go
package handlers_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/handlers"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
    "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/testutil"
)

func TestPortfolioHandler_Portfolios(t *testing.T) {
    t.Run("GET /api/portfolio returns empty array", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)
        handler := handlers.NewPortfolioHandler(svc)

        // Create request
        req := httptest.NewRequest(http.MethodGet, "/api/portfolio", nil)
        w := httptest.NewRecorder()

        // Execute
        handler.Portfolios(w, req)

        // Assert
        assert.Equal(t, http.StatusOK, w.Code)

        var response []handlers.PortfoliosResponse
        err := json.NewDecoder(w.Body).Decode(&response)
        require.NoError(t, err)
        assert.Empty(t, response)
    })

    t.Run("GET /api/portfolio returns all portfolios", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)
        handler := handlers.NewPortfolioHandler(svc)

        // Create test data
        p1 := testutil.CreatePortfolio(t, db, "Portfolio 1")
        p2 := testutil.CreatePortfolio(t, db, "Portfolio 2")

        // Create request
        req := httptest.NewRequest(http.MethodGet, "/api/portfolio", nil)
        w := httptest.NewRecorder()

        // Execute
        handler.Portfolios(w, req)

        // Assert
        assert.Equal(t, http.StatusOK, w.Code)

        var response []handlers.PortfoliosResponse
        err := json.NewDecoder(w.Body).Decode(&response)
        require.NoError(t, err)
        assert.Len(t, response, 2)

        // Verify data
        assert.Equal(t, p1.ID, response[0].ID)
        assert.Equal(t, "Portfolio 1", response[0].Name)
        assert.Equal(t, p2.ID, response[1].ID)
        assert.Equal(t, "Portfolio 2", response[1].Name)
    })

    t.Run("returns 500 on database error", func(t *testing.T) {
        // Setup with closed database (will cause error)
        db := testutil.SetupTestDB(t)
        db.Close()  // Close to force error

        svc := service.NewPortfolioService(db)
        handler := handlers.NewPortfolioHandler(svc)

        // Create request
        req := httptest.NewRequest(http.MethodGet, "/api/portfolio", nil)
        w := httptest.NewRecorder()

        // Execute
        handler.Portfolios(w, req)

        // Assert
        assert.Equal(t, http.StatusInternalServerError, w.Code)

        var response map[string]string
        err := json.NewDecoder(w.Body).Decode(&response)
        require.NoError(t, err)
        assert.Contains(t, response["error"], "database")
    })
}
```

---

## Bugs Found During Testing

Writing comprehensive tests helped us discover **2 production bugs** that were caught early:

### Bug #1: Missing TotalValue in Portfolio Summary

**Location:** `internal/service/portfolio_service.go:294-299`

**Problem:** The `TotalValue` field was calculated but not included in the `TransactionMetrics` struct initialization, causing portfolio summaries to always show `TotalValue: 0`.

**Fix:**
```go
// BEFORE (broken):
transactionMetrics := TransactionMetrics{
    TotalShares:    totalShares,
    TotalCost:      totalCost,
    TotalDividends: totalDividends,
    TotalFees:      totalFees,
}

// AFTER (fixed):
transactionMetrics := TransactionMetrics{
    TotalShares:    totalShares,
    TotalCost:      totalCost,
    TotalValue:     totalValue,  // ← Added this line
    TotalDividends: totalDividends,
    TotalFees:      totalFees,
}
```

**Test that caught it:**
```go
func TestPortfolioHandler_PortfolioSummary_BasicTransactions(t *testing.T) {
    // ... test setup ...

    // This assertion failed with TotalValue: 0.00 instead of expected 1200.0
    if summary.TotalValue != 1200.0 {
        t.Errorf("Expected total value 1200.0, got %.2f", summary.TotalValue)
    }
}
```

### Bug #2: NULL Value Handling in Dividend Repository

**Location:** `internal/repository/dividend_repository.go:51-105`

**Problem:** Attempting to scan NULL database values directly into Go string variables caused runtime errors: `converting NULL to string is unsupported`.

**Fix:**
```go
// BEFORE (broken):
for rows.Next() {
    var recordDateStr, exDividendStr, buyOrderStr, createdAtStr string
    var t model.Dividend

    err := rows.Scan(
        &t.ID,
        &t.FundID,
        &t.PortfolioFundID,
        &recordDateStr,
        &exDividendStr,
        &t.SharesOwned,
        &t.DividendPerShare,
        &t.TotalAmount,
        &t.ReinvestmentStatus,
        &buyOrderStr,  // ← Fails on NULL
        &t.ReinvestmentTransactionId,
        &createdAtStr,
    )
}

// AFTER (fixed):
for rows.Next() {
    var recordDateStr, exDividendStr, createdAtStr string
    var buyOrderStr, reinvestmentTxID sql.NullString  // ← Use sql.NullString
    var t model.Dividend

    err := rows.Scan(
        &t.ID,
        &t.FundID,
        &t.PortfolioFundID,
        &recordDateStr,
        &exDividendStr,
        &t.SharesOwned,
        &t.DividendPerShare,
        &t.TotalAmount,
        &t.ReinvestmentStatus,
        &buyOrderStr,
        &reinvestmentTxID,
        &createdAtStr,
    )

    // BuyOrderDate is nullable
    if buyOrderStr.Valid {
        t.BuyOrderDate, err = ParseTime(buyOrderStr.String)
        if err != nil || t.BuyOrderDate.IsZero() {
            return nil, fmt.Errorf("failed to parse buy_order_date: %w", err)
        }
    }

    // ReinvestmentTransactionId is nullable
    if reinvestmentTxID.Valid {
        t.ReinvestmentTransactionId = reinvestmentTxID.String
    }
}
```

**Test that caught it:**
```go
func TestPortfolioHandler_PortfolioSummary_WithDividends(t *testing.T) {
    // ... test setup with dividends ...

    // Test failed with HTTP 500 error:
    // "failed to scan dividend table results: converting NULL to string is unsupported"

    handler.PortfolioSummary(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}
```

### Key Takeaways

1. **Comprehensive tests reveal edge cases** - These bugs only appeared when testing with realistic data including NULL values
2. **Test early, test often** - Finding bugs during development is much cheaper than in production
3. **Integration tests matter** - Handler tests that exercise the full stack (handler → service → repository → database) catch real issues
4. **Go's SQL NULL handling requires care** - Always use `sql.NullString`, `sql.NullInt64`, etc. for nullable database fields

---

## Summary: Python to Go Migration

| Python Pattern | Go Equivalent | Notes |
|----------------|---------------|-------|
| `@pytest.fixture(scope="session")` | `TestMain(m *testing.M)` | Package-level setup |
| `@pytest.fixture(scope="function")` | Setup function + `t.Cleanup()` | Per-test setup |
| `PortfolioFactory()` | Builder pattern + `Build()` | Manual but flexible |
| `assert x == y` | `assert.Equal(t, expected, actual)` | testify library |
| `pytest tests/services/` | `go test ./internal/service/` | Directory-based |
| `pytest --cov` | `go test -cover ./...` | Built-in coverage |
| `clear_database()` | `testutil.CleanDatabase(t, db)` | Manual cleanup |
| Test classes | `t.Run()` subtests | Groups related tests |
| `test_*.py` files | `*_test.go` files | Naming convention |

**Key Philosophy:**
- Go testing is **more explicit** but **less magical**
- You write more helper code, but it's **simple and understandable**
- No hidden fixtures - everything is **visible in the test**
- Tests live **next to the code** they test

---

## Current Implementation Status

### ✅ Completed

1. **Test Infrastructure** - DONE
   - ✅ Created `internal/testutil/` package with database helpers
   - ✅ Implemented all 7 factory builders (Portfolio, Fund, PortfolioFund, Transaction, FundPrice, Dividend, RealizedGainLoss)
   - ✅ Created test helper functions for ID generation

2. **Handler Tests** - DONE
   - ✅ 17 comprehensive tests for `PortfolioSummary` endpoint in `internal/api/handlers/portfolios_test.go`
   - ✅ Tests cover empty portfolios, basic transactions, filtering, dividends, realized gains, and error cases
   - ✅ All tests passing (make test)

3. **Bug Fixes** - DONE
   - ✅ Fixed missing TotalValue in portfolio summary
   - ✅ Fixed NULL value handling in dividend repository

4. **Documentation** - DONE
   - ✅ Complete testing guide (this document)
   - ✅ Testing setup summary with implementation status
   - ✅ Quick reference guide

### 🎯 Next Steps

1. **Expand Test Coverage**
   - Add service layer tests for `portfolio_service.go` (currently only handler tests exist)
   - Add repository layer tests for database operations
   - Test error paths and edge cases more thoroughly

2. **Additional Tests**
   - Test dividend reinvestment calculations
   - Test complex portfolio scenarios (multiple funds, many transactions)
   - Test date-based filtering and sorting

3. **Code Quality**
   - Run coverage analysis: `make coverage-html`
   - Add linting to CI/CD: `make lint`
   - Consider adding benchmark tests for performance-critical code

4. **CI/CD Integration**
   - Add GitHub Actions workflow to run tests
   - Enforce minimum coverage threshold (90%)
   - Add test results to pull request comments

---

## Quick Start

Want to add a new test? Here's the pattern:

```go
func TestYourFeature(t *testing.T) {
    // 1. Setup database
    db := testutil.SetupTestDB(t)

    // 2. Create test data using builders
    portfolio := testutil.CreatePortfolio(t, db, "Test Portfolio")
    fund := testutil.CreateFund(t, db, "AAPL")

    // 3. Exercise your code
    result, err := YourFunction(portfolio, fund)

    // 4. Assert expectations
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Value != expected {
        t.Errorf("expected %v, got %v", expected, result.Value)
    }
}
```

Run your tests:
```bash
make test              # Run all tests
make test-verbose      # See detailed output
make coverage-html     # Generate coverage report
```

---

Happy testing! This guide reflects the actual implementation as of the current state. Keep it updated as you add more tests and patterns.
