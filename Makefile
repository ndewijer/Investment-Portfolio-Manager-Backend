.PHONY: run build test clean deps help

# Module and version settings
MODULE=github.com/ndewijer/Investment-Portfolio-Manager-Backend
VERSION_PKG=$(MODULE)/internal/version

# Run the application
run:
	go run -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" cmd/server/main.go

# Build the application
build:
	go build -ldflags "-X $(VERSION_PKG).Version=$$(cat VERSION)" -o bin/server cmd/server/main.go

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

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
	@echo "  run       - Run the application"
	@echo "  build     - Build the application binary"
	@echo "  test      - Run tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  clean     - Clean build artifacts"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  fmt       - Format code"
	@echo "  lint      - Run linter"
	@echo "  help      - Display this help message"
