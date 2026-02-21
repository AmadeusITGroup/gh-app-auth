# gh-app-auth Makefile

.PHONY: help build test lint clean install dev-setup security-scan release deps vet gocyclo staticcheck ineffassign misspell test-coverage-check markdownlint yamllint actionlint cli-smoke-test package-deb package-rpm packages

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
	@echo "  dev-setup          Set up development environment (config only)"
	@echo "  validate-tools     Validate core tools are installed"
	@echo "  validate-lint-tools Validate linting tools can run"
	@echo "  security-scan      Run security scans (gosec, govulncheck)"
	@echo "  deps               Download and verify dependencies"
	@echo "  dev                Quick development cycle (fmt + lint + test + build)"
	@echo "  ci                 CI pipeline simulation (mirrors GitHub CI)"
	@echo "  quality            Full quality check (all linters + tests + security)"
	@echo "  release            Build release binaries for all platforms"
	@echo ""
	@echo "Packaging targets:"
	@echo "  package-deb          Build DEB package for amd64"
	@echo "  package-deb-arm64    Build DEB package for arm64 (requires 'make release')"
	@echo "  package-deb-arm      Build DEB package for arm/armhf (requires 'make release')"
	@echo "  package-rpm          Build RPM package for amd64"
	@echo "  package-rpm-arm64    Build RPM package for arm64 (requires 'make release')"
	@echo "  packages             Build all packages (deb/rpm for all architectures)"
	@echo "  packages-local       Build packages for local architecture only"
	@echo "  validate-packages    Verify binary/package architectures match targets"
	@echo ""
	@echo "Presentation targets:"
	@echo "  presentation       Build both HTML and PDF presentations"
	@echo "  presentation-html  Build interactive HTML presentation"
	@echo "  presentation-pdf   Build PDF presentation"
	@echo "  presentation-serve Serve presentation locally on :8000"
	@echo "  presentation-clean Clean presentation build artifacts"

# Build variables
BINARY_NAME := gh-app-auth
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
PKG_VERSION := $(shell echo "$(VERSION)" | sed 's/^v//')
RPM_RELEASE ?= 1
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Go tool commands
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
GOIMPORTS := go run golang.org/x/tools/cmd/goimports@latest
STATICCHECK := go run honnef.co/go/tools/cmd/staticcheck@latest
GOCYCLO := go run github.com/fzipp/gocyclo/cmd/gocyclo@latest
INEFFASSIGN := go run github.com/gordonklaus/ineffassign@latest
MISSPELL := go run github.com/client9/misspell/cmd/misspell@latest
GOSEC := go run github.com/securego/gosec/v2/cmd/gosec@latest
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@latest
ACTIONLINT := go run github.com/rhysd/actionlint/cmd/actionlint@latest

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

# Lint code with golangci-lint
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

# Run actionlint on GitHub workflow files
actionlint:
	@echo "Running actionlint..."
	@$(ACTIONLINT) || echo "⚠️  Action lint issues found"

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
	$(GOIMPORTS) -w .
	gofmt -s -w .

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

# Set up development environment (configuration only, no tool installation)
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	@echo "Setting up git commit template..."
	git config commit.template .gitmessage
	@echo "Development environment ready!"
	@echo ""
	@echo "💡 Tip: Use 'git commit' (without -m) to use the conventional commit template"
	@echo "📖 See CONTRIBUTING.md for conventional commit guidelines"

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

# Build matrix: os-arch (gh extension install compatible)
# Format: OS-ARCH (no prefix, no dots, windows has .exe)
# Based on download analysis:
#   linux-amd64: 16999 (critical - primary platform)
#   darwin-arm64: 12, darwin-amd64: 9 (real but low usage)
#   windows/*: 8 each (likely checks only, but kept for completeness)
#   freebsd/*: 8 each (likely checks only - commented out)
BUILD_MATRIX := \
	linux-amd64 \
	linux-arm64 \
	darwin-amd64 \
	darwin-arm64 \
	windows-amd64:.exe \
	windows-arm64:.exe

# Additional platforms (uncomment if needed):
# windows-386:.exe, linux-386, linux-arm, freebsd-amd64, freebsd-arm64, freebsd-386

# Build release binaries
release: clean
	@echo "Building release binaries..."
	mkdir -p dist
	@echo "Build matrix: $(BUILD_MATRIX)"
	@echo ""
	$(foreach entry,$(BUILD_MATRIX),\
		$(eval platform := $(entry)) \
		$(eval ext := $(word 2,$(subst :, ,$(entry)))) \
		$(eval platform_clean := $(word 1,$(subst :, ,$(entry)))) \
		$(eval os := $(word 1,$(subst -, ,$(platform_clean)))) \
		$(eval arch := $(word 2,$(subst -, ,$(platform_clean)))) \
		echo "  Building $(platform_clean)..."; \
		CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
			go build $(LDFLAGS) \
			-o dist/$(platform_clean)$(or $(ext),) .; \
	)
	@echo ""
	@echo "Release binaries built in dist/"
	@ls -la dist/

# Validate that all required tools are installed
validate-tools:
	@echo "Validating required tools..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed"; exit 1; }
	@command -v gh >/dev/null 2>&1 || { echo "❌ GitHub CLI is required but not installed"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "❌ Git is required but not installed"; exit 1; }
	@echo "✅ Core tools are installed."

# Validate that linting tools can be executed
validate-lint-tools:
	@echo "Validating linting tools can be executed..."
	@$(GOLANGCI_LINT) version >/dev/null 2>&1 || { echo "❌ golangci-lint failed"; exit 1; }
	@$(GOIMPORTS) -l . >/dev/null 2>&1 || { echo "❌ goimports failed"; exit 1; }
	@$(STATICCHECK) --help >/dev/null 2>&1 || { echo "❌ staticcheck failed"; exit 1; }
	@$(GOCYCLO) . >/dev/null 2>&1 || { echo "❌ gocyclo failed"; exit 1; }
	@$(INEFFASSIGN) . >/dev/null 2>&1 || { echo "❌ ineffassign failed"; exit 1; }
	@$(MISSPELL) --help >/dev/null 2>&1 || { echo "❌ misspell failed"; exit 1; }
	@$(GOSEC) --help >/dev/null 2>&1 || { echo "❌ gosec failed"; exit 1; }
	@$(GOVULNCHECK) --help >/dev/null 2>&1 || { echo "❌ govulncheck failed"; exit 1; }
	@echo "✅ All linting tools work"

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

# Packaging targets
.PHONY: package-deb package-deb-arm64 package-deb-arm package-rpm package-rpm-arm64 package-rpm-arm packages packages-local validate-packages

# Build DEB package for amd64
package-deb:
	@echo "Building DEB package for amd64..."
	@mkdir -p dist
	@echo "Building Linux amd64 binary..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/linux-amd64 .
	@export GOARCH=amd64 ARCH=amd64 VERSION=$(PKG_VERSION); \
	envsubst '$$GOARCH $$ARCH $$VERSION' < nfpm.yaml > nfpm-temp.yaml; \
	GOARCH=$(shell go env GOARCH) $(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_amd64.deb; \
	rm nfpm-temp.yaml
	@echo "✅ DEB package created: dist/$(BINARY_NAME)_$(PKG_VERSION)_amd64.deb"

# Build DEB package for arm64
package-deb-arm64: release
	@echo "Building DEB package for arm64..."
	@test -f dist/linux-arm64 || { echo "❌ Linux ARM64 binary not found. Run 'make release' first."; exit 1; }
	@export GOARCH=arm64 ARCH=arm64 VERSION=$(PKG_VERSION); \
	envsubst '$$GOARCH $$ARCH $$VERSION' < nfpm.yaml > nfpm-temp.yaml; \
	GOARCH=$(shell go env GOARCH) $(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_arm64.deb; \
	rm nfpm-temp.yaml
	@echo "✅ DEB package created: dist/$(BINARY_NAME)_$(PKG_VERSION)_arm64.deb"


# Build RPM package for amd64
package-rpm:
	@echo "Building RPM package for amd64..."
	@mkdir -p dist
	@echo "Building Linux amd64 binary..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/linux-amd64 .
	@export GOARCH=amd64 ARCH=amd64 VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	GOARCH=$(shell go env GOARCH) $(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_x86_64.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ RPM package created: dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_x86_64.rpm"

# Build RPM package for arm64
package-rpm-arm64: release
	@echo "Building RPM package for arm64..."
	@test -f dist/linux-arm64 || { echo "❌ Linux ARM64 binary not found. Run 'make release' first."; exit 1; }
	@export GOARCH=arm64 ARCH=arm64 VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	GOARCH=$(shell go env GOARCH) $(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_aarch64.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ RPM package created: dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_aarch64.rpm"



# Build all packages (requires release binaries)
packages: dev-setup release package-deb package-rpm package-deb-arm64 package-rpm-arm64 package-deb-arm package-rpm-arm
	@echo ""
	@echo "=========================================="
	@echo "  All packages built successfully!"
	@echo "=========================================="
	@ls -lh dist/*.deb dist/*.rpm 2>/dev/null || echo "⚠️  Some packages may not have been created"

# Validate package architectures match targets
validate-packages:
	@echo "Validating binary and package architectures..."
	@echo ""
	@echo "=== Linux Binary Architecture Verification ==="
	@for binary in dist/linux-*; do \
		if [ -f "$$binary" ]; then \
			echo -n "$$binary: "; \
			file $$binary | grep -oP '(x86-64|ARM aarch64|Intel 80386|ARM EABI)'; \
		fi; \
	done
	@echo ""
	@echo "=== macOS Binary Architecture Verification ==="
	@for binary in dist/darwin-*; do \
		if [ -f "$$binary" ]; then \
			echo -n "$$binary: "; \
			file $$binary | grep -oP '(x86-64|arm64)'; \
		fi; \
	done
	@echo ""
	@echo "=== Windows Binary Architecture Verification ==="
	@for binary in dist/windows-*.exe; do \
		if [ -f "$$binary" ]; then \
			echo -n "$$binary: "; \
			file $$binary | grep -oP '(x86-64|Aarch64)'; \
		fi; \
	done
	@echo ""
	@echo "=== DEB Package Architecture Verification ==="
	@for deb in dist/*.deb; do \
		if [ -f "$$deb" ]; then \
			echo -n "$$deb: "; \
			dpkg-deb -I "$$deb" | grep Architecture | awk '{print $$2}'; \
		fi; \
	done
	@echo ""
	@echo "=== RPM Package Architecture Verification ==="
	@for rpm in dist/*.rpm; do \
		if [ -f "$$rpm" ]; then \
			echo -n "$$rpm: "; \
			rpm -qip "$$rpm" 2>/dev/null | grep Architecture | awk '{print $$2}' || echo "N/A (rpm not installed)"; \
		fi; \
	done
	@echo ""
	@echo "✅ Validation complete!"

# Build packages for local architecture only
packages-local:
	@echo "Building packages for local architecture ($(shell go env GOARCH))..."
	@mkdir -p dist
	@echo "Building Linux binary for local architecture..."
	@GOOS=linux GOARCH=$(shell go env GOARCH) CGO_ENABLED=0 go build $(LDFLAGS) -o dist/linux-$(shell go env GOARCH) .
ifeq ($(shell go env GOARCH),amd64)
	@export GOARCH=amd64 ARCH=amd64 VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_amd64.deb; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_x86_64.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ Local packages created (amd64)"
else ifeq ($(shell go env GOARCH),arm64)
	@export GOARCH=arm64 ARCH=arm64 VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_arm64.deb; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_aarch64.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ Local packages created (arm64)"
else ifeq ($(shell go env GOARCH),386)
	@export GOARCH=386 ARCH=386 VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_i386.deb; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_i386.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ Local packages created (386)"
else ifeq ($(shell go env GOARCH),arm)
	@export GOARCH=arm ARCH=arm VERSION=$(PKG_VERSION) RPM_RELEASE=$(RPM_RELEASE); \
	envsubst '$$GOARCH $$ARCH $$VERSION $$RPM_RELEASE' < nfpm.yaml > nfpm-temp.yaml; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager deb --target dist/$(BINARY_NAME)_$(PKG_VERSION)_armhf.deb; \
	$(NFPM_CMD) pkg --config nfpm-temp.yaml --packager rpm --target dist/$(BINARY_NAME)_$(PKG_VERSION)-$(RPM_RELEASE)_armv7hl.rpm; \
	rm nfpm-temp.yaml
	@echo "✅ Local packages created (arm)"
else
	@echo "⚠️  Unsupported architecture: $(shell go env GOARCH)"
	@exit 1
endif

# Presentation targets
.PHONY: presentation presentation-html presentation-pdf presentation-serve presentation-clean

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
	@echo "Ensuring mermaid-filter is available..."
	@command -v mermaid-filter >/dev/null 2>&1 || { echo "Installing mermaid-filter..." && npm install -g mermaid-filter; }
	@mkdir -p dist/presentation
	export PATH="$$(npm config get prefix)/bin:$$PATH"; \
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
