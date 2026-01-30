# Tooling Recommendations

Essential tools and practices for Go development that will improve code quality, catch bugs early, and make development more productive.

---

## Table of Contents

1. [Essential Tools (Use These)](#essential-tools-use-these)
2. [Recommended Tools (Consider These)](#recommended-tools-consider-these)
3. [IDE Configuration](#ide-configuration)
4. [Makefile Improvements](#makefile-improvements)
5. [Pre-commit Hooks](#pre-commit-hooks)
6. [CI/CD Recommendations](#cicd-recommendations)

---

## Essential Tools (Use These)

### 1. golangci-lint

**You mentioned discovering this. Use it!**

**What it does:** Meta-linter that runs 50+ linters in parallel.

**Installation:**
```bash
# macOS
brew install golangci-lint

# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Create `.golangci.yml` in project root:**
```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    # Bugs
    - errcheck      # Check error returns
    - gosec         # Security issues
    - staticcheck   # Static analysis
    - govet         # Go vet checks

    # Style
    - gofmt         # Formatting
    - goimports     # Import organization
    - misspell      # Spelling mistakes
    - revive        # Replacement for golint

    # Complexity
    - gocyclo       # Cyclomatic complexity
    - funlen        # Function length

    # Performance
    - prealloc      # Suggest preallocations

linters-settings:
  funlen:
    lines: 100      # Warn on functions over 100 lines
    statements: 50

  gocyclo:
    min-complexity: 15  # Warn on complex functions

  errcheck:
    check-type-assertions: true
    check-blank: true

issues:
  exclude-rules:
    # Ignore long lines in generated files
    - path: "_test\\.go"
      linters:
        - funlen
```

**Run it:**
```bash
golangci-lint run
```

**What it catches:**
- Unchecked errors (`err` not handled)
- Security vulnerabilities
- Common mistakes
- Style inconsistencies
- Unused code

### 2. go test (Built-in)

**What it does:** Runs tests and checks coverage.

**Commands you should use:**
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View in browser

# Run specific test
go test -run TestPortfolioService ./internal/service/
```

**You have no tests yet.** Priority: Write at least one test file before implementing writes.

### 3. go fmt / gofmt (Built-in)

**What it does:** Formats code to standard style.

**Run before every commit:**
```bash
go fmt ./...
```

**Or use goimports (better):**
```bash
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

goimports does everything gofmt does plus organizes imports.

### 4. go vet (Built-in)

**What it does:** Reports suspicious constructs.

```bash
go vet ./...
```

**Catches things like:**
- Printf format string mismatches
- Unreachable code
- Copying locks
- Struct tag issues

### 5. go mod tidy (Built-in)

**What it does:** Cleans up go.mod and go.sum.

```bash
go mod tidy
```

Run this:
- After adding new imports
- Before committing
- When dependencies seem broken

---

## Recommended Tools (Consider These)

### 1. air - Live Reload

**What it does:** Restarts your server when files change.

**Installation:**
```bash
go install github.com/air-verse/air@latest
```

**Create `.air.toml`:**
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ./cmd/server"
  bin = "./tmp/main"
  delay = 1000
  exclude_dir = ["tmp", "vendor", "docs"]
  include_ext = ["go"]
  exclude_regex = ["_test.go"]

[log]
  time = true

[misc]
  clean_on_exit = true
```

**Run:**
```bash
air
```

Now your server restarts automatically when you save files.

### 2. dlv (Delve) - Debugger

**What it does:** Go debugger for stepping through code.

**Installation:**
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Usage:**
```bash
# Debug a program
dlv debug ./cmd/server

# Debug a test
dlv test ./internal/service/

# In debugger:
# (dlv) break main.main
# (dlv) continue
# (dlv) next
# (dlv) print variableName
```

VS Code integrates with Delve automatically.

### 3. gotests - Test Generator

**What it does:** Generates table-driven test stubs.

**Installation:**
```bash
go install github.com/cweill/gotests/...@latest
```

**Usage:**
```bash
# Generate tests for all functions in a file
gotests -all -w internal/service/portfolio_service.go

# Generate tests for specific function
gotests -only GetPortfolio -w internal/service/portfolio_service.go
```

**Output example:**
```go
func TestPortfolioService_GetPortfolio(t *testing.T) {
    type args struct {
        ctx context.Context
        id  string
    }
    tests := []struct {
        name    string
        args    args
        want    *model.Portfolio
        wantErr bool
    }{
        // TODO: Add test cases.
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### 4. mockgen - Mock Generator

**What it does:** Generates mock implementations for interfaces.

**Installation:**
```bash
go install go.uber.org/mock/mockgen@latest
```

**Usage:**
```bash
# Generate mock for an interface
mockgen -source=internal/repository/portfolio_repository.go -destination=internal/repository/mocks/portfolio_repository_mock.go
```

**Why useful:** Enables testing services without real database.

### 5. goose - Migrations (Covered in SQLC_DECISION_GUIDE)

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

---

## IDE Configuration

### VS Code (Recommended for Go)

**Install the Go extension** by the Go team.

**Settings (`.vscode/settings.json`):**
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.formatTool": "goimports",
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.organizeImports": "explicit"
    },
    "[go]": {
        "editor.insertSpaces": false,
        "editor.tabSize": 4,
        "editor.defaultFormatter": "golang.go"
    },
    "go.testFlags": ["-v", "-race"],
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "highlight"
    }
}
```

**Recommended extensions:**
- Go (by Go Team at Google) - Essential
- Error Lens - Inline error display
- GitLens - Git integration
- REST Client - Test API endpoints

### GoLand (JetBrains)

GoLand has most tools built in:
- Automatic gofmt on save
- Integrated testing
- Database tools
- Debugger

If using GoLand, less external tooling needed.

---

## Makefile Improvements

Update your Makefile:

```makefile
.PHONY: run build test coverage lint fmt vet clean deps help air check

# Variables
VERSION_PKG := github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/version
BINARY := bin/server

# Development
run:
	go run -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" cmd/server/main.go

air:
	air

# Build
build:
	go build -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" -o $(BINARY) cmd/server/main.go

# Testing
test:
	go test -v -race ./...

test-short:
	go test -short ./...

coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Code quality
lint:
	golangci-lint run

fmt:
	goimports -w .

vet:
	go vet ./...

# Combined check (run before commit)
check: fmt vet lint test
	@echo "All checks passed!"

# Dependencies
deps:
	go mod download
	go mod tidy

# Cleanup
clean:
	rm -rf bin/ coverage.out coverage.html tmp/

# Help
help:
	@echo "Available targets:"
	@echo "  run       - Run development server"
	@echo "  air       - Run with live reload"
	@echo "  build     - Build binary"
	@echo "  test      - Run all tests"
	@echo "  coverage  - Generate coverage report"
	@echo "  lint      - Run golangci-lint"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  check     - Run all checks (fmt, vet, lint, test)"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  clean     - Remove build artifacts"
```

**New workflow:**
```bash
# During development
make air

# Before committing
make check

# Generate coverage report
make coverage
```

---

## Pre-commit Hooks

Automate checks before every commit.

### Option 1: Git Hooks (Simple)

Create `.git/hooks/pre-commit`:
```bash
#!/bin/sh

echo "Running pre-commit checks..."

# Format check
echo "Checking formatting..."
if [ -n "$(gofmt -l .)" ]; then
    echo "Files need formatting. Run 'go fmt ./...'"
    exit 1
fi

# Vet
echo "Running go vet..."
go vet ./... || exit 1

# Lint
echo "Running golangci-lint..."
golangci-lint run || exit 1

# Tests
echo "Running tests..."
go test -short ./... || exit 1

echo "All checks passed!"
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Option 2: pre-commit Framework (Better)

**Installation:**
```bash
# macOS
brew install pre-commit

# pip
pip install pre-commit
```

**Create `.pre-commit-config.yaml`:**
```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-imports
      - id: go-mod-tidy

  - repo: local
    hooks:
      - id: go-test
        name: go test
        entry: go test -short ./...
        language: system
        pass_filenames: false
```

**Install hooks:**
```bash
pre-commit install
```

Now checks run automatically on `git commit`.

---

## CI/CD Recommendations

### GitHub Actions (Recommended)

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install dependencies
        run: go mod download

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: coverage.out

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Build
        run: go build -v ./cmd/server
```

### What This Gives You

1. **On every push/PR:**
   - Runs linter
   - Runs tests
   - Checks build succeeds

2. **Benefits:**
   - Catch issues before merge
   - Document code quality
   - Coverage tracking

---

## Tool Priority List

### Must Have (Start Using Now)

1. **golangci-lint** - Catches bugs and style issues
2. **go test** - You need tests
3. **goimports** - Auto-format and organize imports
4. **VS Code Go extension** - IDE support

### Should Have (Add Soon)

5. **air** - Live reload saves time
6. **pre-commit hooks** - Automated quality gates
7. **goose** - Database migrations

### Nice to Have (Add Later)

8. **gotests** - Test generation
9. **mockgen** - Mock generation
10. **GitHub Actions** - CI/CD

---

## Summary Checklist

```
[ ] Install golangci-lint
[ ] Create .golangci.yml configuration
[ ] Install goimports
[ ] Configure VS Code settings
[ ] Update Makefile with new targets
[ ] Add pre-commit hook
[ ] Write first test file
[ ] Run `make check` before commits
```

### Quick Start

```bash
# Install essential tools
brew install golangci-lint
go install golang.org/x/tools/cmd/goimports@latest

# Create config
echo 'run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - gosec
    - staticcheck
    - gofmt
    - revive' > .golangci.yml

# Run checks
golangci-lint run
go test ./...
```

---

*Document created: 2026-01-22*
*For: Investment Portfolio Manager Go Backend*
