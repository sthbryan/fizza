# Fizza Makefile

# Variables
BINARY_NAME=fizza
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILT=$(shell date +%Y-%m-%d)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILT)"
CGO_ENABLED=0
BUILD_DIR=bin
PKG=./cmd/fizza

# Default target
.DEFAULT_GOAL:=build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for current platform... ($(VERSION))"
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(PKG)

# Build for all platforms
.PHONY: build-all
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building $(BINARY_NAME) for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(PKG)

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building $(BINARY_NAME) for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(PKG)

.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(PKG)

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building $(BINARY_NAME) for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(PKG)

.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building $(BINARY_NAME) for windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(PKG)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Run tests with race detector
.PHONY: test-race
test-race:
	@echo "Running tests with -race..."
	go test -race ./...

# Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Format Go source files with gofmt (CI checks this with `gofmt -l .`)
.PHONY: fmt
fmt:
	@echo "Running gofmt -w..."
	gofmt -w .
	@echo "Checking for unformatted files..."
	@test -z "$$(gofmt -l .)" || (echo "ERROR: gofmt found unformatted files:" && gofmt -l . && exit 1)
	@echo "All files formatted."

# Smoke-test the MCP server end-to-end
.PHONY: mcp-test
mcp-test: build
	@echo "Smoke-testing $(BINARY_NAME) mcp server..."
	@go run ./scripts/mcp-smoke $(BUILD_DIR)/$(BINARY_NAME)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	@go clean -cache -testcache 2>/dev/null || true

# Install binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) go install $(LDFLAGS) $(PKG)

# Uninstall binary
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $$(go env GOPATH)/bin/$(BINARY_NAME)

# Help target
.PHONY: help
help:
	@echo "Fizza Makefile targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all platforms (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64)"
	@echo "  test           - Run go tests"
	@echo "  test-race      - Run tests with -race detector"
	@echo "  vet            - Run go vet"
	@echo "  fmt            - Run gofmt -w and verify CI formatting check passes"
	@echo "  mcp-test       - Smoke-test the MCP server end-to-end"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install binary using go install"
	@echo "  uninstall      - Remove installed binary"
	@echo "  help           - Show this help message"
