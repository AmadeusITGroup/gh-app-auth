# gh-app-auth Makefile

.PHONY: help build test lint clean install dev-setup security-scan release deps

# Default target
help:
	@echo "gh-app-auth - GitHub App Authentication Extension"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build the extension binary"
	@echo "  test         Run all tests"
	@echo "  test-race    Run tests with race detection"
	@echo "  test-cover   Run tests with coverage report"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  clean        Clean build artifacts"
	@echo "  install      Install extension to GitHub CLI"
	@echo "  uninstall    Uninstall extension from GitHub CLI"
	@echo "  dev-setup    Set up development environment"
	@echo "  security-scan Run security scans"
	@echo "  deps         Download and verify dependencies"
	@echo "  release      Build release binaries for all platforms"

# Build variables
BINARY_NAME := gh-app-auth
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Build the extension
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint:
	@echo "Running linters..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

# Install extension to GitHub CLI
install: build
	@echo "Installing extension to GitHub CLI..."
	gh extension install .

# Uninstall extension from GitHub CLI
uninstall:
	@echo "Uninstalling extension from GitHub CLI..."
	gh extension remove app-auth || true

# Set up development environment
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Setting up git commit template..."
	git config commit.template .gitmessage
	@echo "Development environment ready!"
	@echo ""
	@echo "💡 Tip: Use 'git commit' (without -m) to use the conventional commit template"
	@echo "📖 See CONTRIBUTING.md for conventional commit guidelines"

# Run security scans
security-scan:
	@echo "Running security scans..."
	$(shell go env GOPATH)/bin/gosec -fmt sarif -out gosec.sarif ./... || true
	@echo "Running vulnerability check..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	$(shell go env GOPATH)/bin/govulncheck ./... || true

# Download and verify dependencies  
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	go mod tidy

# Build release binaries
release: clean
	@echo "Building release binaries..."
	mkdir -p dist
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	
	# Windows ARM64
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-arm64.exe .
	
	@echo "Release binaries built in dist/"
	@ls -la dist/

# Validate that all required tools are installed
validate-tools:
	@echo "Validating required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed"; exit 1; }
	@command -v gh >/dev/null 2>&1 || { echo "GitHub CLI is required but not installed"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "Git is required but not installed"; exit 1; }
	@echo "All required tools are installed."

# Quick development cycle
dev: fmt lint test build
	@echo "Development cycle complete!"

# CI pipeline simulation
ci: deps validate-tools lint test-race security-scan build
	@echo "CI pipeline complete!"
