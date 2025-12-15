#!/usr/bin/env bash
# EZ Language Installer
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="ez"

echo "EZ Language Installer"
echo "===================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# Build the binary
echo "Building EZ..."
go build -o "$BINARY_NAME" ./cmd/ez

# Install to system
echo "Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Need sudo permissions to install to $INSTALL_DIR"
    sudo cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

# Cleanup local binary
rm -f "$BINARY_NAME"

echo ""
echo "EZ installed successfully!"
echo "Run 'ez run <file>' to execute EZ programs"
echo ""
echo "To uninstall, run: sudo rm $INSTALL_DIR/$BINARY_NAME"
