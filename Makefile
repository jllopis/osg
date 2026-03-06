# OSG Makefile

# Variables
BINARY_NAME := osg
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X osg/internal/app.Version=$(VERSION) -X osg/internal/app.Commit=$(COMMIT) -X osg/internal/app.Date=$(DATE)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod

# Directories
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

# OSG runtime vars
CONFIG ?= config.yaml
VAULT_PATH ?=
OSG_CONTENT_DIR ?=
PUBLIC_DIR ?=
INCLUDE_DRAFTS ?= false
DRY_RUN ?= false
VERBOSE ?= false
ARGS ?=
SERVE_ADDR ?= :1313
INSTALL_PATH := $(HOME)/.local/bin

.PHONY: all build build-all clean test test-coverage lint fmt vet tidy deps run init update-content serve tui install uninstall version help
.PHONY: plugins plugins-search plugins-llmstxt plugins-mermaid plugins-archives plugins-example-feed install-plugins

## Default target
all: tidy fmt vet test build

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/osg

## Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/osg
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/osg
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/osg
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/osg
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/osg

## Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f cover.out

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

## Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## Run the CLI
run:
	@$(GO) run ./cmd/osg $(ARGS)

## Serve public directory
serve:
	@$(GO) run ./cmd/osg serve -c $(CONFIG) \
		$(if $(PUBLIC_DIR),--public-dir $(PUBLIC_DIR),) \
		--addr $(SERVE_ADDR)

## Launch TUI
tui:
	@$(GO) run ./cmd/osg tui

## Initialize project structure
init:
	@$(GO) run ./cmd/osg init -c $(CONFIG)

## Sync content from vault
update-content:
	@$(GO) run ./cmd/osg -c $(CONFIG) \
		$(if $(VAULT_PATH),--vault-path $(VAULT_PATH),) \
		$(if $(OSG_CONTENT_DIR),--osg-content-dir $(OSG_CONTENT_DIR),) \
		$(if $(filter true,$(INCLUDE_DRAFTS)),--include-drafts,) \
		$(if $(filter true,$(DRY_RUN)),--dry-run,) \
		$(if $(filter true,$(VERBOSE)),--verbose,) \
		$(ARGS)

## Install to ~/.local/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed: $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo ""
	@echo "Ensure $(INSTALL_PATH) is in your PATH:"
	@echo "  export PATH=\"\$$HOME/.local/bin:\$$PATH\""

## Uninstall from ~/.local/bin
uninstall:
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled: $(INSTALL_PATH)/$(BINARY_NAME)"

## Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

## Help
help:
	@echo "OSG - Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Usage: make [target]"

## Build bundled WASM plugins (from plugins-src/)
plugins: plugins-search

## Build all official WASM plugins (from plugins-src/)
plugins-all: plugins-search plugins-llmstxt plugins-mermaid plugins-archives

## Build search plugin (bundled, Rust WASM)
plugins-search:
	@echo "Building search plugin..."
	@cd plugins-src/search && ./build.sh
	@echo "Built: plugins-src/search/search.wasm"

## Build llmstxt plugin (official, Rust WASM)
plugins-llmstxt:
	@echo "Building llmstxt plugin..."
	@cd plugins-src/llmstxt && ./build.sh
	@echo "Built: plugins-src/llmstxt/llmstxt.wasm"

## Build mermaid plugin (official, Rust WASM)
plugins-mermaid:
	@echo "Building mermaid plugin..."
	@cd plugins-src/mermaid && ./build.sh
	@echo "Built: plugins-src/mermaid/mermaid.wasm"

## Build archives plugin (official, Rust WASM)
plugins-archives:
	@echo "Building archives plugin..."
	@cd plugins-src/archives && ./build.sh
	@echo "Built: plugins-src/archives/archives.wasm"

## Install compiled plugins into plugins/ directory
install-plugins: plugins-all
	@echo "Installing plugins..."
	@mkdir -p plugins
	@cp plugins-src/search/search.wasm plugins/
	@cp plugins-src/llmstxt/llmstxt.wasm plugins/
	@cp plugins-src/mermaid/mermaid.wasm plugins/
	@cp plugins-src/archives/archives.wasm plugins/
	@echo "Installed: plugins/*.wasm"

## Build example feed plugin (reference only, not bundled)
plugins-example-feed:
	@echo "Building example feed plugin..."
	@cd examples/plugins/feed && ./build.sh
	@echo "Built: examples/plugins/feed/feed.wasm"
