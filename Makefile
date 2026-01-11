.PHONY: run build test test-short test-verbose coverage coverage-html clean deps help

# Module and version settings
MODULE=github.com/ndewijer/Investment-Portfolio-Manager-Backend
VERSION_PKG=$(MODULE)/internal/version

# Run the application
run:
	go run -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" cmd/server/main.go

# Build the application
build:
	go build -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" -o bin/server cmd/server/main.go

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

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

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
	@echo "  run            - Run the application"
	@echo "  build          - Build the application binary"
	@echo "  test           - Run all tests with race detector"
	@echo "  test-short     - Run tests in short mode (skip slow tests)"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  coverage       - Run tests with coverage summary"
	@echo "  coverage-html  - Generate HTML coverage report"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  help           - Display this help message"
