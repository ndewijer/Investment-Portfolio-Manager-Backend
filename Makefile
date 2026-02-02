.PHONY: run build test test-short test-verbose coverage coverage-html coverage-func coverage-by-file coverage-open coverage-handlers coverage-handlers-full coverage-validation coverage-full-html clean deps help

# Module and version settings
MODULE=github.com/ndewijer/Investment-Portfolio-Manager-Backend
VERSION_PKG=$(MODULE)/internal/version

# Run the application
run:
	go run -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" ./cmd/server/main.go

# Build the application
build:
	go build -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" -o bin/server ./cmd/server/main.go

# Run all tests
test:
	go test -race ./...

# Run tests with short mode (skip slow tests)
test-short:
	go test -short -race ./...

# Run tests with verbose output
test-verbose:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage.out | grep total

# Generate and open HTML coverage report in browser
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Open existing HTML coverage report in browser
coverage-open:
	@if [ -f coverage.html ]; then \
		open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Please open coverage.html manually"; \
	else \
		echo "Error: coverage.html not found. Run 'make coverage-html' first."; \
	fi

# Show detailed per-function coverage
coverage-func: coverage
	@echo ""
	@echo "Per-function coverage:"
	@go tool cover -func=coverage.out

# Show per-file coverage sorted by percentage
coverage-by-file: coverage
	@echo ""
	@echo "Per-file coverage (sorted by %):"
	@go tool cover -func=coverage.out | grep -v "total:" | awk '{print $$3 "\t" $$1}' | sort -rn

# Show files with less than 100% coverage
coverage-gaps: coverage
	@echo ""
	@echo "Files with less than 100% coverage:"
	@go tool cover -func=coverage.out | grep -v "100.0%" | grep -v "total:"

# Run coverage for handlers package only
coverage-handlers:
	go test -coverprofile=coverage.out ./internal/api/handlers
	@echo ""
	@echo "Handler Coverage:"
	@go tool cover -func=coverage.out | grep "handlers/"
	@echo ""
	@go tool cover -func=coverage.out | grep total

# Run ALL tests with detailed coverage breakdown
coverage-handlers-full:
	@echo "Running ALL tests with comprehensive coverage..."
	go test -coverpkg=./internal/... \
		-coverprofile=coverage_full.out ./...
	@echo ""
	@echo "=== Validation Coverage ==="
	@go tool cover -func=coverage_full.out | grep "validation/" || echo "No validation coverage"
	@echo ""
	@echo "=== Handler Coverage ==="
	@go tool cover -func=coverage_full.out | grep "handlers/" | tail -10
	@echo ""
	@echo "=== Middleware Coverage ==="
	@go tool cover -func=coverage_full.out | grep "middleware/" || echo "No middleware coverage"
	@echo ""
	@echo "=== Repository Coverage ==="
	@go tool cover -func=coverage_full.out | grep "repository/" | tail -5
	@echo ""
	@echo "=== Service Coverage ==="
	@go tool cover -func=coverage_full.out | grep "service/" | tail -5
	@echo ""
	@echo "=== Total Coverage ==="
	@go tool cover -func=coverage_full.out | grep "total:"

# Show detailed validation package coverage from ALL tests
coverage-validation:
	@echo "Validation package coverage from ALL tests:"
	@go test -coverpkg=./internal/validation \
		-coverprofile=coverage_validation.out ./... 2>/dev/null
	@echo ""
	@go tool cover -func=coverage_validation.out | grep "validation/"
	@echo ""
	@go tool cover -func=coverage_validation.out | grep "total:"

# Generate HTML report for ALL packages
coverage-full-html:
	@echo "Generating comprehensive HTML coverage report..."
	@go test -coverpkg=./internal/... \
		-coverprofile=coverage_full.out ./...
	@go tool cover -html=coverage_full.out -o coverage_full.html
	@echo "Coverage report generated: coverage_full.html"

# Clean build artifacts and coverage files
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html coverage_*.out

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  run              - Run the application"
	@echo "  build            - Build the application binary"
	@echo ""
	@echo "Testing:"
	@echo "  test             - Run all tests with race detector"
	@echo "  test-short       - Run tests in short mode (skip slow tests)"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo ""
	@echo "Coverage:"
	@echo "  coverage              - Run tests with coverage summary"
	@echo "  coverage-html         - Generate and view HTML coverage report"
	@echo "  coverage-open         - Open existing HTML coverage report"
	@echo "  coverage-func         - Show detailed per-function coverage"
	@echo "  coverage-by-file      - Show per-file coverage sorted by %"
	@echo "  coverage-gaps         - Show files with less than 100% coverage"
	@echo "  coverage-handlers     - Run coverage for handlers package only"
	@echo "  coverage-handlers-full- Run ALL tests with detailed coverage breakdown ⭐"
	@echo "  coverage-validation   - Show validation package coverage from ALL tests"
	@echo "  coverage-full-html    - Generate comprehensive HTML coverage report"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean            - Clean build artifacts and coverage files"
	@echo "  deps             - Download and tidy dependencies"
	@echo "  fmt              - Format code"
	@echo "  lint             - Run linter"
	@echo "  help             - Display this help message"
