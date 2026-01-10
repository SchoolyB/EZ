#!/bin/bash

# Build EZ interpreter for WebAssembly
# Usage: ./build.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.."

echo "Building EZ WASM..."
GOOS=js GOARCH=wasm go build -o cmd/wasm/ez.wasm ./cmd/wasm

echo "Copying wasm_exec.js..."
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" cmd/wasm/

echo ""
echo "Build complete!"
echo "  - cmd/wasm/ez.wasm"
echo "  - cmd/wasm/wasm_exec.js"
echo ""
echo "To use in a website, copy both files to your public directory."
