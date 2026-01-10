#!/bin/bash

# Build EZ interpreter for WebAssembly
# Usage: ./build.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.."

echo "Building EZ WASM..."
GOOS=js GOARCH=wasm go build -o cmd/wasm/ez.wasm ./cmd/wasm

# Note: wasm_exec.js is a customized version that properly routes
# stderr to console.error (Go's default sends everything to console.log)

echo ""
echo "Build complete!"
echo "  - cmd/wasm/ez.wasm"
echo "  - cmd/wasm/wasm_exec.js"
echo ""
echo "To use in a website, copy both files to your public directory."
