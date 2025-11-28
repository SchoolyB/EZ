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
	@echo "  make build      - Build the ez binary"
	@echo "  make install    - Install ez to $(INSTALL_PATH)"
	@echo "  make uninstall  - Remove ez from $(INSTALL_PATH)"
	@echo "  make clean      - Remove built binaries"
	@echo "  make test       - Run tests"
	@echo "  make release    - Build for multiple platforms"

build:
	@echo "Building EZ..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ez
	@echo "Build complete! Binary: ./$(BINARY_NAME)"

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
	@echo "EZ installed successfully!"
	@echo "Try: ez help"

uninstall:
	@echo "Uninstalling EZ..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "EZ uninstalled"

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f ez.bin
	@rm -rf dist/
	@echo "Clean complete"

test:
	$(GO) test ./... -v

# Build for multiple platforms
release:
	@echo "Building releases..."
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/ez
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/ez
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/ez
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/ez
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/ez
	@echo "Release builds complete in dist/"
