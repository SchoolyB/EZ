# EZ Language Build System
.PHONY: build stubs install uninstall clean help \
       test test-unit test-e2e test-integration test-go \
       test-ubsan test-asan

BINARY_NAME=ez
INSTALL_PATH=/usr/local/bin
GO=go

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

EMBED_DIR=internal/ezc/runtime

help:
	@echo "EZ Language Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build     - Build the ez binary (compiler embedded)"
	@echo "  make stubs     - Create empty embed stubs (for dev go build)"
	@echo "  make install   - Install ez to $(INSTALL_PATH)"
	@echo "  make uninstall - Remove ez from $(INSTALL_PATH)"
	@echo "  make clean     - Remove built binaries"
	@echo ""
	@echo "Test targets:"
	@echo "  make test             - Run the full test suite (unit + e2e + integration + Go)"
	@echo "  make test-unit        - Run C unit tests (lexer, parser, typechecker)"
	@echo "  make test-e2e         - Run end-to-end codegen tests"
	@echo "  make test-integration - Run integration tests (pass + fail)"
	@echo "  make test-go          - Run Go unit tests"
	@echo "  make test-ubsan       - Run UBSan sanitizer tests"
	@echo "  make test-asan        - Run ASan+UBSan sanitizer tests (Linux recommended)"

# ===== Test targets =====
# Delegate compiler tests to ezc/Makefile; Go tests run from the root.

test: build test-unit test-e2e test-integration test-go
	@echo ""
	@echo "All test suites completed."

test-unit:
	@$(MAKE) -C ezc test-unit

test-e2e: build
	@$(MAKE) -C ezc test-e2e

test-integration: build
	@bash scripts/run_tests.sh

test-go: stubs
	@echo ""
	@echo "=== Go Unit Tests ==="
	$(GO) test ./pkg/lineeditor/... ./cmd/ez/... ./internal/ezc/...

test-ubsan:
	@$(MAKE) -C ezc test-ubsan

test-asan:
	@$(MAKE) -C ezc test-asan

# Create zero-length embed stubs. go:embed directives in
# internal/ezc/embedded.go require these files to exist at `go build`
# time; the extractor detects empty stubs and falls back to the legacy
# path search so dev builds still work. Both the runtime binaries
# themselves are gitignored — `make build` overwrites the stubs with
# real content before invoking `go build`.
stubs:
	@mkdir -p $(EMBED_DIR)/src/runtime $(EMBED_DIR)/src/stdlib
	@test -f $(EMBED_DIR)/ezc || : > $(EMBED_DIR)/ezc
	@test -f $(EMBED_DIR)/libezrt.a || : > $(EMBED_DIR)/libezrt.a
	@test -f $(EMBED_DIR)/src/runtime/.stub || : > $(EMBED_DIR)/src/runtime/.stub
	@test -f $(EMBED_DIR)/src/stdlib/.stub || : > $(EMBED_DIR)/src/stdlib/.stub

# Single-binary build (#1461): compile the C compiler first, stage the
# artifacts into internal/ezc/runtime/ so go:embed picks them up, then
# build the Go CLI. The final `ez` binary contains `ezc` + `libezrt.a`
# as embedded assets and extracts them on first use to ~/.ez/runtime/.
build: stubs
	@echo "Building compiler..."
	@$(MAKE) -C ezc build
	@echo "Staging embedded runtime assets..."
	@cp ezc/ezc $(EMBED_DIR)/ezc
	@cp ezc/libezrt.a $(EMBED_DIR)/libezrt.a
	@cp ezc/src/runtime/*.h ezc/src/runtime/*.c $(EMBED_DIR)/src/runtime/
	@cp ezc/src/stdlib/*.h ezc/src/stdlib/*.c $(EMBED_DIR)/src/stdlib/
	@echo "Building ez CLI (with embedded runtime)..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ez
	@echo ""
	@echo "Build complete: ./$(BINARY_NAME)"
	@echo "Run with: ./$(BINARY_NAME) <file.ez>"

install: build
	@echo "Installing EZ to $(INSTALL_PATH)..."
	@if [ -w $(INSTALL_PATH) ]; then \
		mkdir -p $(INSTALL_PATH); \
		cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME); \
		chmod +x $(INSTALL_PATH)/$(BINARY_NAME); \
	else \
		echo "Need sudo permissions to install to $(INSTALL_PATH)"; \
		sudo mkdir -p $(INSTALL_PATH); \
		sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME); \
		sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME); \
	fi
	@echo ""
	@echo ' _____ ____'
	@echo '| ____|__  |'
	@echo '|  _|   / /'
	@echo '| |___ / /_'
	@echo '|_____/____|'
	@echo 'Programming made EZ'
	@echo ""
	@echo "EZ installed successfully!"

uninstall:
	@echo "Uninstalling EZ..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@# Remove any legacy standalone ezc from prior install layouts (#1461)
	@rm -f $(INSTALL_PATH)/ezc
	@echo "EZ uninstalled"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@# Embed assets are gitignored — delete entirely, not truncate
	@rm -f $(EMBED_DIR)/ezc $(EMBED_DIR)/libezrt.a
	@rm -rf $(EMBED_DIR)/src
	@$(MAKE) -C ezc clean
	@echo "Clean complete"

