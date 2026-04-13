# EZ Language Build System
.PHONY: build install uninstall clean test help

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
	@echo "  make install   - Install ez to $(INSTALL_PATH)"
	@echo "  make uninstall - Remove ez from $(INSTALL_PATH)"
	@echo "  make clean     - Remove built binaries"
	@echo "  make test      - Run all tests"

# Single-binary build (#1461): compile the C compiler first, stage the
# artifacts into internal/ezc/runtime/ so go:embed picks them up, then
# build the Go CLI. The final `ez` binary contains `ezc` + `libezrt.a`
# as embedded assets and extracts them on first use to ~/.ez/runtime/.
build:
	@echo "Building compiler..."
	@$(MAKE) -C ezc build
	@echo "Staging embedded runtime assets..."
	@cp ezc/ezc $(EMBED_DIR)/ezc
	@cp ezc/libezrt.a $(EMBED_DIR)/libezrt.a
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
	@echo "Try: ez examples/basic/hello.ez"

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
	@# Truncate staged embed assets back to empty stubs
	@: > $(EMBED_DIR)/ezc
	@: > $(EMBED_DIR)/libezrt.a
	@$(MAKE) -C ezc clean
	@echo "Clean complete"

test:
	@echo "=== CLI/tooling tests ==="
	$(GO) test ./pkg/errors/... ./pkg/lineeditor/...
	@echo ""
	@echo "=== Compiler unit tests ==="
	@$(MAKE) -C ezc test-unit
	@echo ""
	@echo "=== Compiler e2e tests ==="
	@$(MAKE) -C ezc test-e2e
