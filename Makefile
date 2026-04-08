# EZ Language Build System
.PHONY: build install uninstall clean test help

BINARY_NAME=ez
INSTALL_PATH=/usr/local/bin
GO=go

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

help:
	@echo "EZ Language Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build     - Build ez CLI + ezc compiler"
	@echo "  make install   - Install both to $(INSTALL_PATH)"
	@echo "  make uninstall - Remove ez and ezc from $(INSTALL_PATH)"
	@echo "  make clean     - Remove built binaries"
	@echo "  make test      - Run all tests"

build:
	@echo "Building ez CLI..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ez
	@echo "Building compiler..."
	@$(MAKE) -C ezc build
	@echo ""
	@echo "Build complete!"
	@echo "  CLI:      ./$(BINARY_NAME)"
	@echo "  Compiler: ./ezc/ezc"
	@echo ""
	@echo "Run with: EZC_PATH=./ezc/ezc ./ez <file.ez>"

install: build
	@echo "Installing EZ to $(INSTALL_PATH)..."
	@if [ -w $(INSTALL_PATH) ]; then \
		mkdir -p $(INSTALL_PATH); \
		cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME); \
		chmod +x $(INSTALL_PATH)/$(BINARY_NAME); \
		cp ezc/ezc $(INSTALL_PATH)/ezc; \
		chmod +x $(INSTALL_PATH)/ezc; \
	else \
		echo "Need sudo permissions to install to $(INSTALL_PATH)"; \
		sudo mkdir -p $(INSTALL_PATH); \
		sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME); \
		sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME); \
		sudo cp ezc/ezc $(INSTALL_PATH)/ezc; \
		sudo chmod +x $(INSTALL_PATH)/ezc; \
	fi
	@echo ""
	@echo "EZ installed successfully!"
	@echo "Try: ez examples/basic/hello.ez"

uninstall:
	@echo "Uninstalling EZ..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@rm -f $(INSTALL_PATH)/ezc
	@echo "EZ uninstalled"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
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
