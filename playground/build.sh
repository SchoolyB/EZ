#!/bin/bash
#
# build.sh - Build the EZ playground WASM module
#
# Compiles the ezc compiler core (lexer, parser, typechecker, codegen)
# to WebAssembly using Emscripten. No stdlib runtime needed — the
# playground only parses, type-checks, and generates C code.
#
# Usage: ./playground/build.sh
#
# Prerequisites: emcc (Emscripten) must be installed and in PATH
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
EZC_SRC="$ROOT_DIR/ezc/src"
OUT_DIR="$SCRIPT_DIR"

# Check for emcc
if ! command -v emcc &> /dev/null; then
    echo "Error: emcc not found. Install Emscripten:"
    echo "  brew install emscripten"
    echo "  or: https://emscripten.org/docs/getting_started/downloads.html"
    exit 1
fi

echo "Building EZ playground WASM..."

# Source files — only the compiler core, no stdlib runtime
SOURCES=(
    "$SCRIPT_DIR/ezc_wasm.c"
    "$EZC_SRC/util/arena.c"
    "$EZC_SRC/util/buf.c"
    "$EZC_SRC/util/error.c"
    "$EZC_SRC/lexer/token.c"
    "$EZC_SRC/lexer/lexer.c"
    "$EZC_SRC/parser/ast.c"
    "$EZC_SRC/parser/parser.c"
    "$EZC_SRC/typechecker/types.c"
    "$EZC_SRC/typechecker/scope.c"
    "$EZC_SRC/typechecker/typechecker.c"
    "$EZC_SRC/codegen/codegen.c"
)

emcc "${SOURCES[@]}" \
    -I"$EZC_SRC" \
    -O2 \
    -s WASM=1 \
    -s EXPORTED_FUNCTIONS='["_ezc_compile","_ezc_check","_malloc","_free"]' \
    -s EXPORTED_RUNTIME_METHODS='["ccall","cwrap","UTF8ToString"]' \
    -s MODULARIZE=1 \
    -s EXPORT_NAME='createEzcModule' \
    -s ALLOW_MEMORY_GROWTH=1 \
    -s TOTAL_MEMORY=16777216 \
    -s NO_EXIT_RUNTIME=1 \
    -o "$OUT_DIR/ezc.js"

echo ""
echo "Build complete!"
echo "  WASM: $OUT_DIR/ezc.wasm ($(du -h "$OUT_DIR/ezc.wasm" | cut -f1))"
echo "  JS:   $OUT_DIR/ezc.js ($(du -h "$OUT_DIR/ezc.js" | cut -f1))"
echo ""
echo "To test locally:"
echo "  cd playground && python3 -m http.server 8080"
echo "  Open http://localhost:8080"
