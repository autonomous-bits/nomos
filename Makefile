.PHONY: help build test test-race lint work-sync clean build-cli test-module build-module
.PHONY: test-unit test-integration test-integration-module test-coverage
.PHONY: fmt mod-tidy install watch
.PHONY: release-lib list-tags release-check

# Module discovery - find all modules dynamically
MODULES := $(shell find . -name 'go.mod' -not -path './go.work' -exec dirname {} \; | sort)
APP_MODULES := $(shell find ./apps -name 'go.mod' -exec dirname {} \; | sort)
LIB_MODULES := $(shell find ./libs -name 'go.mod' -exec dirname {} \; | sort)

# Default target
help:
	@echo "Available targets:"
	@echo "  help              - Show this help message"
	@echo "  build             - Build all applications"
	@echo "  build-cli         - Build the CLI application"
	@echo "  build-module      - Build a specific module (usage: make build-module MODULE=libs/compiler)"
	@echo "  test              - Run all tests across all modules"
	@echo "  test-unit         - Run only unit tests (excludes integration tests)"
	@echo "  test-integration  - Run all integration tests across all modules"
	@echo "  test-coverage     - Generate coverage reports for all modules"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-module       - Test a specific module (usage: make test-module MODULE=libs/compiler)"
	@echo "  test-integration-module - Run integration tests for a specific module"
	@echo "  fmt               - Format all Go code"
	@echo "  mod-tidy          - Tidy all module dependencies"
	@echo "  install           - Install the nomos binary to GOPATH/bin"
	@echo "  lint              - Run linters across all modules"
	@echo "  watch             - Auto-rebuild on file changes (requires air)"
	@echo "  work-sync         - Sync Go workspace dependencies"
	@echo "  clean             - Clean build artifacts"
	@echo ""
	@echo "Release targets:"
	@echo "  release-lib       - Tag and release a library (usage: make release-lib LIB=compiler VERSION=v0.1.0)"
	@echo "  release-check     - Verify pre-release checklist for a library (usage: make release-check LIB=compiler)"
	@echo "  list-tags         - List all library tags"

# Sync Go workspace
work-sync:
	@echo "Syncing Go workspace..."
	@go work sync

# Build all applications
build: work-sync
	@echo "Building all applications..."
	@cd apps/command-line && go build -o ../../bin/nomos ./cmd/nomos

# Build CLI application
build-cli: work-sync
	@echo "Building CLI application..."
	@cd apps/command-line && go build -o ../../bin/nomos ./cmd/nomos

# Build specific module
build-module: work-sync
	@if [ -z "$(MODULE)" ]; then \
		echo "Error: MODULE variable not set. Usage: make build-module MODULE=libs/compiler"; \
		exit 1; \
	fi
	@echo "Building module $(MODULE)..."
	@cd $(MODULE) && go build ./...

# Run all tests
test: work-sync
	@echo "Running all tests across workspace..."
	@for dir in $(MODULES); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test -v ./...) || exit 1; \
	done

# Run unit tests only (exclude integration tests)
test-unit: work-sync
	@echo "Running unit tests across workspace (excluding integration tests)..."
	@for dir in $(MODULES); do \
		echo "Testing $$dir (unit tests only)..."; \
		(cd $$dir && go test -v -short ./...) || exit 1; \
	done

# Run integration tests only
test-integration: work-sync
	@echo "Running integration tests across workspace..."
	@for dir in $(MODULES); do \
		echo "Testing $$dir (integration tests)..."; \
		(cd $$dir && go test -v -tags=integration ./...) || exit 1; \
	done

# Generate coverage reports for all modules
test-coverage: work-sync
	@echo "Generating coverage reports for all modules..."
	@mkdir -p coverage
	@for dir in $(MODULES); do \
		module_name=$$(basename $$dir); \
		echo "Generating coverage for $$dir..."; \
		(cd $$dir && go test -coverprofile=$(CURDIR)/coverage/$$module_name.out -covermode=atomic ./...) || exit 1; \
		(cd $$dir && go tool cover -html=$(CURDIR)/coverage/$$module_name.out -o $(CURDIR)/coverage/$$module_name.html) || exit 1; \
	done
	@echo "Coverage reports generated in ./coverage/"
	@echo "Open coverage/*.html in a browser to view reports"

# Run tests with race detector
test-race: work-sync
	@echo "Running tests with race detector..."
	@for dir in $(MODULES); do \
		echo "Testing $$dir with race detector..."; \
		(cd $$dir && go test -race ./...) || exit 1; \
	done

# Test specific module
test-module: work-sync
	@if [ -z "$(MODULE)" ]; then \
		echo "Error: MODULE variable not set. Usage: make test-module MODULE=libs/compiler"; \
		exit 1; \
	fi
	@echo "Testing module $(MODULE)..."
	@cd $(MODULE) && go test -v ./...

# Run integration tests for specific module
test-integration-module: work-sync
	@if [ -z "$(MODULE)" ]; then \
		echo "Error: MODULE variable not set. Usage: make test-integration-module MODULE=libs/compiler"; \
		exit 1; \
	fi
	@echo "Running integration tests for module $(MODULE)..."
	@cd $(MODULE) && go test -v -tags=integration ./...

# Format all Go code
fmt:
	@echo "Formatting all Go code..."
	@for dir in $(MODULES); do \
		echo "Formatting $$dir..."; \
		(cd $$dir && go fmt ./...) || exit 1; \
	done
	@echo "✅ All code formatted"

# Tidy all module dependencies
mod-tidy:
	@echo "Tidying all module dependencies..."
	@for dir in $(MODULES); do \
		echo "Tidying $$dir..."; \
		(cd $$dir && go mod tidy) || exit 1; \
	done
	@go work sync
	@echo "✅ All modules tidied and workspace synced"

# Install nomos binary to GOPATH/bin
install: build-cli
	@echo "Installing nomos binary..."
	@go install ./apps/command-line/cmd/nomos
	@echo "✅ nomos installed to $$(go env GOPATH)/bin/nomos"
	@echo "Make sure $$(go env GOPATH)/bin is in your PATH"

# Lint all modules
lint:
	@echo "Linting all modules..."
	@for dir in $(MODULES); do \
		echo "Linting $$dir..."; \
		(cd $$dir && golangci-lint run ./...) || exit 1; \
	done

# Watch mode - auto-rebuild on file changes (requires air: go install github.com/cosmtrek/air@latest)
watch:
	@echo "Starting watch mode (auto-rebuild)..."
	@if ! command -v air >/dev/null 2>&1; then \
		echo "Error: 'air' not found. Install with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi
	@air

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean -cache

# Release targets
# Usage: make release-check LIB=compiler
release-check:
	@if [ -z "$(LIB)" ]; then \
		echo "Error: LIB variable not set. Usage: make release-check LIB=compiler"; \
		exit 1; \
	fi
	@echo "Running pre-release checks for libs/$(LIB)..."
	@echo "1. Checking if on main branch..."
	@if [ "$$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then \
		echo "   ❌ Not on main branch"; \
		exit 1; \
	fi
	@echo "   ✅ On main branch"
	@echo "2. Checking for uncommitted changes..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "   ❌ Uncommitted changes detected"; \
		git status --short; \
		exit 1; \
	fi
	@echo "   ✅ No uncommitted changes"
	@echo "3. Checking if up-to-date with remote..."
	@git fetch origin main >/dev/null 2>&1
	@if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/main)" ]; then \
		echo "   ❌ Local branch not in sync with origin/main"; \
		exit 1; \
	fi
	@echo "   ✅ Up-to-date with origin/main"
	@echo "4. Running tests for libs/$(LIB)..."
	@cd libs/$(LIB) && go test ./... >/dev/null 2>&1 && echo "   ✅ Tests pass" || (echo "   ❌ Tests failed"; exit 1)
	@echo "5. Checking CHANGELOG.md..."
	@if [ ! -f "libs/$(LIB)/CHANGELOG.md" ]; then \
		echo "   ❌ CHANGELOG.md not found"; \
		exit 1; \
	fi
	@echo "   ✅ CHANGELOG.md exists"
	@echo ""
	@echo "✅ All pre-release checks passed for libs/$(LIB)"

# Usage: make release-lib LIB=compiler VERSION=v0.1.0
# This creates an annotated tag and displays push instructions
release-lib: release-check
	@if [ -z "$(LIB)" ] || [ -z "$(VERSION)" ]; then \
		echo "Error: LIB and VERSION variables required."; \
		echo "Usage: make release-lib LIB=compiler VERSION=v0.1.0"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "Error: VERSION must be in format vX.Y.Z (e.g., v0.1.0)"; \
		exit 1; \
	fi
	@echo "Creating annotated tag for libs/$(LIB) $(VERSION)..."
	@TAG="libs/$(LIB)/$(VERSION)"; \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
		echo "Error: Tag $$TAG already exists"; \
		exit 1; \
	fi; \
	git tag -a "$$TAG" -m "libs/$(LIB) $(VERSION)"; \
	echo "✅ Tag $$TAG created successfully"; \
	echo ""; \
	echo "To push the tag to GitHub, run:"; \
	echo "  git push origin $$TAG"; \
	echo ""; \
	echo "Or to push all tags:"; \
	echo "  git push origin --tags"

# List all library tags
list-tags:
	@echo "Library tags in repository:"
	@git tag -l "libs/*" | sort
