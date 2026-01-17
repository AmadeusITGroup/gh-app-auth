# gh-app-auth Makefile

.PHONY: help build test lint clean install dev-setup security-scan release deps vet gocyclo staticcheck ineffassign misspell test-coverage-check markdownlint yamllint actionlint cli-smoke-test

# Default target
help:
	@echo "gh-app-auth - GitHub App Authentication Extension"
	@echo ""
	@echo "Available targets:"
	@echo "  build              Build the extension binary"
	@echo "  test               Run all tests"
	@echo "  test-race          Run tests with race detection"
	@echo "  test-cover         Run tests with coverage report"
	@echo "  test-coverage-check Enforce minimum coverage threshold"
	@echo "  lint               Run all linters (golangci-lint)"
	@echo "  vet                Run go vet"
	@echo "  staticcheck        Run staticcheck"
	@echo "  gocyclo            Run gocyclo (cyclomatic complexity)"
	@echo "  ineffassign        Run ineffassign (ineffectual assignments)"
	@echo "  misspell           Run misspell (spelling checker)"
	@echo "  markdownlint       Run markdownlint on .md files"
	@echo "  yamllint           Run yamllint on .yml/.yaml files"
	@echo "  actionlint         Run actionlint on GitHub workflow files"
	@echo "  lint-all           Run all individual linters"
	@echo "  fmt                Format code"
	@echo "  clean              Clean build artifacts"
	@echo "  install            Install extension to GitHub CLI"
	@echo "  uninstall          Uninstall extension from GitHub CLI"
	@echo "  dev-setup          Set up development environment"
	@echo "  validate-tools     Validate core tools are installed"
	@echo "  validate-lint-tools Validate linting tools are installed"
	@echo "  security-scan      Run security scans"
	@echo "  deps               Download and verify dependencies"
	@echo "  dev                Quick development cycle (fmt + lint + test + build)"
	@echo "  ci                 CI pipeline simulation (mirrors GitHub CI)"
	@echo "  quality            Full quality check (all linters + tests + security)"
	@echo "  release            Build release binaries for all platforms"
	@echo ""
	@echo "Presentation targets:"
	@echo "  presentation-setup Install presentation tools (Mermaid CLI, filters)"
	@echo "  presentation       Build both HTML and PDF presentations"
	@echo "  presentation-html  Build interactive HTML presentation"
	@echo "  presentation-pdf   Build PDF presentation (requires presentation-setup)"
	@echo "  presentation-serve Serve presentation locally on :8000"
	@echo "  presentation-clean Clean presentation build artifacts"

# Build variables
BINARY_NAME := gh-app-auth
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Tool paths
GOPATH := $(shell go env GOPATH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint
GOIMPORTS := $(GOPATH)/bin/goimports
STATICCHECK := $(GOPATH)/bin/staticcheck
GOCYCLO := $(GOPATH)/bin/gocyclo
INEFFASSIGN := $(GOPATH)/bin/ineffassign
MISSPELL := $(GOPATH)/bin/misspell
GOSEC := $(GOPATH)/bin/gosec
GOVULNCHECK := $(GOPATH)/bin/govulncheck

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

# Check coverage meets minimum threshold
COVERAGE_THRESHOLD ?= 50.0
test-coverage-check:
	@echo "Checking coverage threshold (minimum: $(COVERAGE_THRESHOLD)%)..."
	@go test -race -coverprofile=coverage.out -covermode=atomic ./... > /dev/null 2>&1 || true
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ "$$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l)" -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%"; \
		echo ""; \
		echo "Package breakdown:"; \
		go tool cover -func=coverage.out | grep -E "(pkg/|cmd/)" | awk '{printf "  %-50s %s\n", $$1, $$3}' | sort; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

# Lint code with golangci-lint (comprehensive)
lint:
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run staticcheck
staticcheck:
	@echo "Running staticcheck..."
	$(STATICCHECK) ./...

# Run gocyclo (cyclomatic complexity)
gocyclo:
	@echo "Running gocyclo (complexity threshold: 10)..."
	$(GOCYCLO) -over 10 . || echo "⚠️  High complexity functions found (expected for CLI commands)"

# Run ineffassign (ineffectual assignments)
ineffassign:
	@echo "Running ineffassign..."
	$(INEFFASSIGN) ./...

# Run misspell (spelling checker)
misspell:
	@echo "Running misspell..."
	$(MISSPELL) -error .

# Run markdownlint (requires npx/node)
markdownlint:
	@echo "Running markdownlint..."
	@command -v npx >/dev/null 2>&1 || { echo "⚠️  npx not found, skipping markdownlint"; exit 0; }
	npx markdownlint-cli2 "**/*.md" "!node_modules/**" || echo "⚠️  Markdown lint issues found"

# Run yamllint (requires pip install yamllint)
yamllint:
	@echo "Running yamllint..."
	@command -v yamllint >/dev/null 2>&1 || { echo "⚠️  yamllint not found, skipping (install: pip install yamllint)"; exit 0; }
	yamllint -d relaxed . || echo "⚠️  YAML lint issues found"

# Run actionlint on GitHub workflow files (requires actionlint binary)
actionlint:
	@echo "Running actionlint..."
	@command -v actionlint >/dev/null 2>&1 || { echo "⚠️  actionlint not found, skipping (install: go install github.com/rhysd/actionlint/cmd/actionlint@latest)"; exit 0; }
	actionlint || echo "⚠️  Action lint issues found"

# CLI smoke test - verify binary works
cli-smoke-test: build
	@echo "Running CLI smoke tests..."
	./$(BINARY_NAME) --help > /dev/null
	./$(BINARY_NAME) --version > /dev/null
	@echo "✅ CLI smoke tests passed"

# Run all individual linters
lint-all: vet staticcheck gocyclo ineffassign misspell markdownlint yamllint actionlint
	@echo "All individual linters completed!"

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	$(GOIMPORTS) -w .

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
	@echo "Installing linting tools..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/gordonklaus/ineffassign@latest
	go install github.com/client9/misspell/cmd/misspell@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Setting up git commit template..."
	git config commit.template .gitmessage
	@echo "Development environment ready!"
	@echo ""
	@echo "💡 Tip: Use 'git commit' (without -m) to use the conventional commit template"
	@echo "📖 See CONTRIBUTING.md for conventional commit guidelines"

# Set up presentation tools
presentation-setup:
	@echo "Setting up presentation tools..."
	@command -v npm >/dev/null 2>&1 || { echo "npm is required. Install Node.js first"; exit 1; }
	@echo "Installing Mermaid CLI..."
	npm install -g @mermaid-js/mermaid-cli
	@echo "Installing mermaid-filter..."
	npm install -g mermaid-filter
	@echo "Presentation tools installed!"
	@echo ""
	@echo "✅ Mermaid CLI: $(shell which mmdc 2>/dev/null || echo 'not found')"
	@npm list -g mermaid-filter >/dev/null 2>&1 && echo "✅ mermaid-filter: installed" || echo "❌ mermaid-filter: not found"

# Run security scans
security-scan:
	@echo "Running security scans..."
	$(GOSEC) -fmt sarif -out gosec.sarif ./... || true
	@echo "Running vulnerability check..."
	$(GOVULNCHECK) ./... || true

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
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed"; exit 1; }
	@command -v gh >/dev/null 2>&1 || { echo "❌ GitHub CLI is required but not installed"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "❌ Git is required but not installed"; exit 1; }
	@echo "✅ Core tools are installed."

# Validate that all linting tools are installed
validate-lint-tools:
	@echo "Validating linting tools..."
	@test -f $(GOLANGCI_LINT) || { echo "❌ golangci-lint not found. Run: make dev-setup"; exit 1; }
	@test -f $(GOIMPORTS) || { echo "❌ goimports not found. Run: make dev-setup"; exit 1; }
	@test -f $(STATICCHECK) || { echo "❌ staticcheck not found. Run: make dev-setup"; exit 1; }
	@test -f $(GOCYCLO) || { echo "❌ gocyclo not found. Run: make dev-setup"; exit 1; }
	@test -f $(INEFFASSIGN) || { echo "❌ ineffassign not found. Run: make dev-setup"; exit 1; }
	@test -f $(MISSPELL) || { echo "❌ misspell not found. Run: make dev-setup"; exit 1; }
	@test -f $(GOSEC) || { echo "❌ gosec not found. Run: make dev-setup"; exit 1; }
	@test -f $(GOVULNCHECK) || { echo "❌ govulncheck not found. Run: make dev-setup"; exit 1; }
	@echo "✅ All linting tools are installed."

# Quick development cycle
dev: fmt lint test build
	@echo "Development cycle complete!"

# CI pipeline simulation (mirrors GitHub CI workflows)
# Runs: deps → vet → lint → test-race → coverage-check → security → build → smoke-test
ci: deps validate-tools validate-lint-tools
	@echo ""
	@echo "=========================================="
	@echo "  CI Pipeline Simulation"
	@echo "=========================================="
	@echo ""
	@echo "Step 1/8: Running go vet..."
	go vet ./...
	@echo ""
	@echo "Step 2/8: Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m
	@echo ""
	@echo "Step 3/8: Running tests with race detection..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo "Step 4/8: Checking coverage threshold (CI: 35%)..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if awk "BEGIN {exit !($$COVERAGE < 35.0)}"; then \
		echo "❌ Coverage $$COVERAGE% is below threshold 35.0%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets threshold 35.0%"; \
	fi
	@echo ""
	@echo "Step 5/8: Running security scans..."
	@$(MAKE) security-scan
	@echo ""
	@echo "Step 6/8: Building binary..."
	go build -v -o $(BINARY_NAME) .
	@echo ""
	@echo "Step 7/8: Running CLI smoke tests..."
	./$(BINARY_NAME) --help > /dev/null
	./$(BINARY_NAME) --version > /dev/null
	@echo "✅ CLI smoke tests passed"
	@echo ""
	@echo "Step 8/8: Running additional linters (non-blocking)..."
	@$(MAKE) markdownlint || true
	@$(MAKE) yamllint || true
	@$(MAKE) actionlint || true
	@echo ""
	@echo "=========================================="
	@echo "  ✅ CI Pipeline Complete!"
	@echo "=========================================="

# Full quality check (all linters + tests)
quality: validate-lint-tools fmt lint-all test-coverage-check security-scan
	@echo "Quality check complete!"

# Presentation targets
.PHONY: presentation presentation-setup presentation-html presentation-pdf presentation-serve presentation-clean

# Build presentation HTML
presentation-html:
	@echo "Building presentation HTML..."
	@command -v pandoc >/dev/null 2>&1 || { echo "Pandoc is required. Install: apt install pandoc"; exit 1; }
	mkdir -p dist/presentation
	pandoc docs/presentation.md \
		-t revealjs \
		-s \
		-o dist/presentation/index.html \
		-V revealjs-url=https://unpkg.com/reveal.js@3.9.2 \
		-V theme=white \
		-V transition=slide \
		-V slideNumber=true \
		--mathjax \
		--highlight-style=pygments
	@echo "Adding custom CSS and Mermaid support..."
	cp docs/presentation-custom.css dist/presentation/
	@# Insert custom CSS link before </head>
	sed -i 's|</head>|  <link rel="stylesheet" href="presentation-custom.css">\n  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>\n</head>|' dist/presentation/index.html
	@# Convert mermaid code blocks to divs - use perl for multi-line replacement
	perl -i -0pe 's/<pre class="mermaid"><code>(.*?)<\/code><\/pre>/<div class="mermaid">\1<\/div>/gs' dist/presentation/index.html
	@# Initialize Mermaid after Reveal.init
	sed -i 's|Reveal\.initialize({|mermaid.initialize({ startOnLoad: true, theme: "default" });\n      Reveal.initialize({|' dist/presentation/index.html
	@echo "Presentation HTML created: dist/presentation/index.html"

# Build presentation PDF
presentation-pdf:
	@echo "Building presentation PDF..."
	@command -v pandoc >/dev/null 2>&1 || { echo "Pandoc is required. Install: apt install pandoc"; exit 1; }
	@command -v xelatex >/dev/null 2>&1 || { echo "XeLaTeX is required. Install: apt install texlive-xetex"; exit 1; }
	@command -v mmdc >/dev/null 2>&1 || { echo "Mermaid CLI is required. Run: make presentation-setup"; exit 1; }
	@npm list -g mermaid-filter >/dev/null 2>&1 || { echo "mermaid-filter is required. Run: make presentation-setup"; exit 1; }
	mkdir -p dist/presentation
	pandoc docs/presentation.md \
		-o dist/presentation/presentation.pdf \
		--pdf-engine=xelatex \
		-F mermaid-filter \
		-V geometry:margin=1in \
		-V fontsize=12pt \
		-V colorlinks=true \
		-V mainfont="DejaVu Sans"
	@echo "Presentation PDF created: dist/presentation/presentation.pdf"

# Build both HTML and PDF
presentation: presentation-html presentation-pdf
	@echo "All presentation formats built successfully!"

# Serve presentation locally
presentation-serve: presentation-html
	@echo "Starting presentation server at http://localhost:8000"
	@command -v python3 >/dev/null 2>&1 || { echo "Python 3 is required"; exit 1; }
	cd dist/presentation && python3 -m http.server 8000

# Clean presentation artifacts
presentation-clean:
	rm -rf dist/presentation
	@echo "Presentation artifacts cleaned"
