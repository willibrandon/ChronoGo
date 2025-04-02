# ChronoGo Makefile

.PHONY: all build clean test lint release

# Default target
all: clean lint build test

# Build variables
BINARY_NAME := chrono
VERSION := 0.1.0
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X 'github.com/willibrandon/ChronoGo/pkg/version.Version=$(VERSION)' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$(BUILD_TIME)'"
DEBUG_FLAGS := -gcflags "all=-N -l" $(LDFLAGS)
RELEASE_FLAGS := -ldflags "-s -w -X 'github.com/willibrandon/ChronoGo/pkg/version.Version=$(VERSION)' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$(BUILD_TIME)'"

# Detect OS
ifeq ($(OS),Windows_NT)
    BINARY_NAME := $(BINARY_NAME).exe
endif

# Build the main binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(DEBUG_FLAGS) -trimpath -o $(BINARY_NAME) ./cmd/chrono

# Build release version (optimized, stripped)
release:
	@echo "Building release version of $(BINARY_NAME)..."
	@go build $(RELEASE_FLAGS) -trimpath -o $(BINARY_NAME) ./cmd/chrono

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		lint_output=$$(golangci-lint run ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Linting failed with the following errors:"; \
			echo "$$lint_output"; \
			exit 1; \
		else \
			echo "Linting passed"; \
		fi \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@if [ -f $(BINARY_NAME) ]; then rm $(BINARY_NAME); fi
	@go clean -testcache

# Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help target
help:
	@echo "ChronoGo Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all         - Clean, lint, build, and test"
	@echo "  build       - Build debug version"
	@echo "  release     - Build release version (optimized, stripped)"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linter"
	@echo "  clean       - Clean build artifacts"
	@echo "  install-lint - Install golangci-lint"
	@echo "  help        - Show this help" 