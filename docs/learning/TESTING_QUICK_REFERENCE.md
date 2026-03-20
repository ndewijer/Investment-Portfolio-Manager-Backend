# Testing Quick Reference

Quick reference for writing and running tests in Go, comparing with your Python/pytest workflow.

## Running Tests

### Python/pytest Commands

```bash
# Run all tests
pytest

# Run specific test file
pytest tests/services/test_portfolio_service.py

# Run specific test
pytest tests/services/test_portfolio_service.py::TestPortfolioRetrieval::test_get_all_portfolios

# Run with coverage
pytest --cov

# Run verbose
pytest -v

# Run only services
pytest tests/services/

# Run marked tests
pytest -m "not slow"
```

### Go Equivalent

```bash
# Run all tests
make test
# or: go test ./...

# Run specific package
go test ./internal/service/

# Run specific test
go test -run TestPortfolioService_GetAllPortfolios ./internal/service/

# Run specific subtest
go test -run TestPortfolioService_GetAllPortfolios/returns_empty_slice ./internal/service/

# Run with coverage
make coverage
# or: go test -cover ./...

# HTML coverage report
make coverage-html

# Verbose output
make test-verbose
# or: go test -v ./...

# Short mode (skip slow tests)
make test-short
# or: go test -short ./...

# Watch mode (requires external tool)
# Install: go install github.com/cespare/reflex@latest
reflex -r '\.go$' -s -- go test ./...
```

## Writing Tests

### Test File Structure

**Python:**
```python
# tests/services/test_portfolio_service.py
import pytest
from tests.factories import PortfolioFactory

class TestPortfolioRetrieval:
    def test_get_all_portfolios(self, db_session):
        """Test retrieving all portfolios"""
        portfolio = PortfolioFactory()

        result = PortfolioService.get_all_portfolios()

        assert len(result) == 1
        assert result[0].name == portfolio.name
```

**Go:**
```go
// internal/service/portfolio_service_test.go
package service_test

import (
    "testing"
    "github.com/ndewijer/.../internal/testutil"
)

func TestPortfolioService_GetAllPortfolios(t *testing.T) {
    t.Run("retrieves all portfolios", func(t *testing.T) {
        // Setup
        db := testutil.SetupTestDB(t)
        svc := service.NewPortfolioService(db)
        portfolio := testutil.CreatePortfolio(t, db, "Test")

        // Execute
        result, err := svc.GetAllPortfolios()

        // Assert
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(result) != 1 {
            t.Errorf("expected 1 portfolio, got %d", len(result))
        }
        if result[0].Name != portfolio.Name {
            t.Errorf("expected name %s, got %s", portfolio.Name, result[0].Name)
        }
    })
}
```

### Fixtures vs Setup Functions

**Python Fixtures:**
```python
@pytest.fixture(scope="function")
def db_session(app_context):
    clear_database()
    yield db.session

def test_something(db_session):
    # db_session automatically injected
    pass
```

**Go Setup:**
```go
func setupTest(t *testing.T) (*sql.DB, *service.PortfolioService) {
    t.Helper()
    db := testutil.SetupTestDB(t)
    svc := service.NewPortfolioService(db)
    t.Cleanup(func() { db.Close() })
    return db, svc
}

func TestSomething(t *testing.T) {
    db, svc := setupTest(t)  // Explicitly called
    // test code
}
```

### Factory Pattern

**Python (factory-boy):**
```python
# Create with defaults
portfolio = PortfolioFactory()

# Create with overrides
portfolio = PortfolioFactory(name="Custom Name")

# Create multiple
portfolios = PortfolioFactory.create_batch(5)

# Build without saving
portfolio = PortfolioFactory.build()
```

**Go (Builder Pattern) - 7 Builders Implemented:**

```go
// Portfolio
portfolio := testutil.NewPortfolio().Build(t, db)
portfolio := testutil.NewPortfolio().WithName("Custom").Archived().Build(t, db)
portfolio := testutil.CreatePortfolio(t, db, "Name")
portfolios := testutil.CreatePortfolios(t, db, 5)

// Fund
fund := testutil.NewFund().Build(t, db)
fund := testutil.NewFund().WithSymbol("AAPL").WithCurrency("USD").Build(t, db)
fund := testutil.CreateFund(t, db, "AAPL")

// Portfolio-Fund Relationship
pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

// Transaction
tx := testutil.NewTransaction(pfID).WithType("buy").WithShares(100).WithCostPerShare(10.0).Build(t, db)

// Fund Price
price := testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

// Dividend
div := testutil.NewDividend(fund.ID, pfID).WithSharesOwned(100).WithDividendPerShare(0.50).Build(t, db)

// Realized Gain/Loss
rgl := testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, tx.ID).WithCostBasis(300.0).WithSaleProceeds(450.0).Build(t, db)
```

### Assertions

**Python (pytest):**
```python
assert portfolio is not None
assert portfolio.name == "Test"
assert len(portfolios) == 5
```

**Go (standard):**
```go
if portfolio == nil {
    t.Error("expected portfolio, got nil")
}
if portfolio.Name != "Test" {
    t.Errorf("expected name 'Test', got '%s'", portfolio.Name)
}
if len(portfolios) != 5 {
    t.Errorf("expected 5 portfolios, got %d", len(portfolios))
}
```

**Go (with testify - optional):**
```go
import "github.com/stretchr/testify/assert"

assert.NotNil(t, portfolio)
assert.Equal(t, "Test", portfolio.Name)
assert.Len(t, portfolios, 5)
```

### Test Organization

**Python Classes:**
```python
class TestPortfolioRetrieval:
    def test_get_all(self):
        pass

    def test_get_one(self):
        pass

class TestPortfolioCreation:
    def test_create(self):
        pass
```

**Go Subtests:**
```go
func TestPortfolioRetrieval(t *testing.T) {
    t.Run("get all", func(t *testing.T) {
        // test code
    })

    t.Run("get one", func(t *testing.T) {
        // test code
    })
}

func TestPortfolioCreation(t *testing.T) {
    t.Run("create", func(t *testing.T) {
        // test code
    })
}
```

## Common Patterns

### Database Testing

**Python:**
```python
@pytest.fixture(scope="function")
def db_session(app_context):
    clear_database()
    yield db.session

def test_something(db_session):
    # Fresh database
    pass
```

**Go:**
```go
func TestSomething(t *testing.T) {
    db := testutil.SetupTestDB(t)  // In-memory, fresh DB
    // Automatically cleaned up after test
}
```

### Creating Test Data

**Python:**
```python
# Create portfolio
portfolio = PortfolioFactory(name="Test")

# Create with relationships
fund = FundFactory()
transaction = TransactionFactory(fund=fund, portfolio=portfolio)
```

**Go:**
```go
// Create portfolio
portfolio := testutil.CreatePortfolio(t, db, "Test")

// Create fund
fund := testutil.CreateFund(t, db, "AAPL")

// Create portfolio-fund relationship
pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

// Create transaction
tx := testutil.NewTransaction(pfID).
    WithType("buy").
    WithShares(100).
    WithCostPerShare(10.0).
    Build(t, db)

// Add current price
testutil.NewFundPrice(fund.ID).WithPrice(12.0).Build(t, db)

// Add dividend
testutil.NewDividend(fund.ID, pfID).
    WithSharesOwned(100).
    WithDividendPerShare(0.50).
    Build(t, db)
```

### Table-Driven Tests

**Python:**
```python
@pytest.mark.parametrize("name,expected", [
    ("Portfolio 1", True),
    ("", False),
])
def test_create_portfolio(name, expected):
    result = create_portfolio(name)
    assert (result is not None) == expected
```

**Go:**
```go
func TestCreatePortfolio(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        hasError bool
    }{
        {"valid name", "Portfolio 1", false},
        {"empty name", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := createPortfolio(tt.input)
            if (err != nil) != tt.hasError {
                t.Errorf("expected error=%v, got error=%v", tt.hasError, err)
            }
        })
    }
}
```

### Mocking

**Python:**
```python
from unittest.mock import Mock, patch

@patch('module.external_api_call')
def test_something(mock_api):
    mock_api.return_value = {"data": "mocked"}
    result = function_that_calls_api()
    assert result == {"data": "mocked"}
```

**Go (using interfaces):**
```go
// Define interface
type ExternalAPI interface {
    FetchData() (Data, error)
}

// Mock implementation
type MockAPI struct {
    FetchDataFunc func() (Data, error)
}

func (m *MockAPI) FetchData() (Data, error) {
    return m.FetchDataFunc()
}

// In test
func TestSomething(t *testing.T) {
    mockAPI := &MockAPI{
        FetchDataFunc: func() (Data, error) {
            return Data{Value: "mocked"}, nil
        },
    }

    result := FunctionThatCallsAPI(mockAPI)
    // assert result
}
```

## Test Utilities Reference

### Database Helpers

```go
// Setup fresh in-memory database
db := testutil.SetupTestDB(t)

// Create service with test database
svc := testutil.NewTestPortfolioService(t, db)
```

### Factory Builders (7 Available)

```go
// 1. Portfolio Builder
portfolio := testutil.NewPortfolio().
    WithName("Custom").
    WithDescription("Desc").
    Archived().
    ExcludedFromOverview().
    Build(t, db)

// Convenience functions
p := testutil.CreatePortfolio(t, db, "Name")
pArchived := testutil.CreateArchivedPortfolio(t, db, "Old")
portfolios := testutil.CreatePortfolios(t, db, 10)

// 2. Fund Builder
fund := testutil.NewFund().
    WithSymbol("AAPL").
    WithName("Apple Inc.").
    WithCurrency("USD").
    WithExchange("NASDAQ").
    Build(t, db)

// Convenience functions
f := testutil.CreateFund(t, db, "AAPL")
funds := testutil.CreateFunds(t, db, 10)

// 3. Portfolio-Fund Builder
pfID := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)

// 4. Transaction Builder
tx := testutil.NewTransaction(portfolioFundID).
    WithType("buy").           // or "sell"
    WithShares(100).
    WithCostPerShare(10.0).
    WithDate(time.Now()).
    Build(t, db)

// 5. Fund Price Builder
price := testutil.NewFundPrice(fund.ID).
    WithPrice(12.0).
    WithDate(time.Now()).
    Build(t, db)

// 6. Dividend Builder
div := testutil.NewDividend(fund.ID, portfolioFundID).
    WithSharesOwned(100).
    WithDividendPerShare(0.50).
    WithReinvestmentTransaction(txID).  // Optional
    Build(t, db)

// 7. Realized Gain/Loss Builder
rgl := testutil.NewRealizedGainLoss(portfolio.ID, fund.ID, tx.ID).
    WithShares(30).
    WithCostBasis(300.0).
    WithSaleProceeds(450.0).
    WithDate(time.Now()).
    Build(t, db)
```

### ID Generation

```go
id := testutil.MakeID()                          // UUID
isin := testutil.MakeISIN("US")                  // US1A2B3C4D5E
symbol := testutil.MakeSymbol("AAPL")            // AAPL1A2B
portfolioName := testutil.MakePortfolioName("Portfolio")  // Portfolio ABC123
fundName := testutil.MakeFundName("Fund")        // Fund XYZ789
```

## Coverage

### Python

```bash
# Run with coverage
pytest --cov

# Coverage report
pytest --cov --cov-report=html

# Fail if below threshold
pytest --cov --cov-fail-under=90
```

### Go

```bash
# Run with coverage
make coverage

# HTML report
make coverage-html
open coverage.html

# Check threshold (requires custom script)
./scripts/check-coverage.sh 90
```

## CI/CD Integration

### Python (GitHub Actions)

```yaml
- name: Run tests
  run: pytest --cov --cov-fail-under=90
```

### Go (GitHub Actions)

```yaml
- name: Run tests
  run: go test -race -coverprofile=coverage.out ./...

- name: Check coverage
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 90" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 90%"
      exit 1
    fi
```

## Tips

### Python to Go Cheat Sheet

| Python | Go |
|--------|-----|
| `@pytest.fixture` | Setup function + `t.Cleanup()` |
| `PortfolioFactory()` | `testutil.NewPortfolio().Build(t, db)` |
| `assert x == y` | `if x != y { t.Errorf(...) }` |
| `pytest.mark.parametrize` | Table-driven tests |
| `with pytest.raises(Error)` | `if err == nil { t.Error(...) }` |
| `db_session` fixture | `testutil.SetupTestDB(t)` |
| `@pytest.mark.slow` | `if testing.Short() { t.Skip() }` |

### Best Practices

1. **Use `t.Helper()`** in helper functions to get correct line numbers in errors
2. **Use `t.Cleanup()`** instead of `defer` for better subtest handling
3. **Use subtests** (`t.Run()`) to organize related tests
4. **Use table-driven tests** for testing multiple similar scenarios
5. **Keep tests next to code** (`*_test.go` in same directory)
6. **Use descriptive test names** that explain what's being tested
7. **Write WHY comments** explaining the purpose of each test
8. **Test one thing per test** - keep tests focused

### Common Gotchas

1. **Database must be closed** - Use `t.Cleanup()` or `defer`
2. **Subtests share state** - Each subtest should set up its own data
3. **Table tests need `t.Run()`** - Otherwise all run in same test
4. **Error messages need context** - Use `t.Errorf()` with formatted messages
5. **Race detector** - Always run with `-race` flag

## Current Implementation Status

### ✅ Completed
- 7 test builders for all major entities
- In-memory SQLite database testing
- 17 comprehensive handler tests for PortfolioSummary endpoint
- All tests passing

### 📚 Learn More

1. [GO_TESTING_GUIDE.md](GO_TESTING_GUIDE.md) - Comprehensive testing guide
2. `internal/testutil/factories.go` - All 7 test builders implementation
3. `internal/api/handlers/portfolios_test.go` - 17 working test examples
4. [TESTING_SETUP_SUMMARY.md](testing/TESTING_SETUP_SUMMARY.md) - Implementation status

### 🚀 Quick Commands

```bash
make test              # Run all tests with race detector
make test-verbose      # Run with detailed output
make coverage          # Show coverage summary
make coverage-html     # Generate HTML coverage report
make help              # See all available commands
```

### 🎯 Next Steps

1. Add service layer tests for `portfolio_service.go`
2. Add repository layer tests
3. Expand test coverage for edge cases
4. Add CI/CD integration

Happy testing! 🧪
