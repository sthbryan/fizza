BINARY      := fizza
PKG         := ./cmd/fizza
DIST        := dist
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X main.version=$(VERSION)
INSTALL_DIR := $(HOME)/.local/bin

GO          := go
GOBIN       ?= $(shell $(GO) env GOPATH)/bin

.PHONY: all build test test-race lint vet clean install uninstall run mcp-test help

all: build

help:
	@echo "fizza — make targets:"
	@echo "  build         compile the binary to ./$(BINARY)"
	@echo "  test          run unit and integration tests"
	@echo "  test-race     run tests with -race detector"
	@echo "  vet           run go vet"
	@echo "  lint          run gofmt + go vet"
	@echo "  install       install binary to $(INSTALL_DIR)"
	@echo "  uninstall     remove installed binary"
	@echo "  run           build and run with --help"
	@echo "  mcp-test      smoke test the MCP server end-to-end"
	@echo "  clean         remove build artifacts"

build:
	@echo "→ building $(BINARY) ($(VERSION))"
	@$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	@$(GO) test ./...

test-race:
	@$(GO) test -race ./...

vet:
	@$(GO) vet ./...

lint:
	@test -z "$$(gofmt -l . | tee /dev/stderr)"
	@$(GO) vet ./...

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "→ installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "  make sure $(INSTALL_DIR) is on your PATH"

uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY)

run: build
	@./$(BINARY) --help

mcp-test: build
	@echo "→ smoke-testing fizza mcp over stdio"
	@$(GO) run ./scripts/mcp-smoke ./$(BINARY)

clean:
	@rm -f $(BINARY)
	@rm -rf $(DIST)
	@$(GO) clean -cache -testcache 2>/dev/null || true
