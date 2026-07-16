#!/bin/bash
#
# generate_errors.sh - Generate ERRORS.md from the centralized error registry
#
# Reads error codes from ezc/src/util/error_codes.h and produces
# ERRORS.md for the documentation website.
#
# Usage: ./scripts/generate_errors.sh
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CODES_FILE="$ROOT_DIR/ezc/src/util/error_codes.h"
OUTPUT="$ROOT_DIR/ERRORS.md"

if [ ! -f "$CODES_FILE" ]; then
    echo "Error: $CODES_FILE not found"
    exit 1
fi

# Count errors, warnings, and panics
ERROR_COUNT=$(grep -c '^ *GRAY_ERROR("' "$CODES_FILE" || echo 0)
WARNING_COUNT=$(grep -c '^ *GRAY_WARNING("' "$CODES_FILE" || echo 0)
PANIC_COUNT=$(grep -c '^ *GRAY_PANIC("' "$CODES_FILE" || echo 0)
TOTAL=$((ERROR_COUNT + WARNING_COUNT + PANIC_COUNT))

# Generate markdown
cat > "$OUTPUT" << HEADER
# Grayscale Error Code Reference

> Auto-generated from \`ezc/src/util/error_codes.h\`. Do not edit manually.
> Run \`./scripts/generate_errors.sh\` to regenerate.

**Total: ${TOTAL} codes** (${ERROR_COUNT} errors, ${WARNING_COUNT} warnings, ${PANIC_COUNT} panics)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
HEADER

# Extract errors
grep '^ *GRAY_ERROR("' "$CODES_FILE" | while IFS= read -r line; do
    code=$(echo "$line" | sed 's/.*GRAY_ERROR("\([^"]*\)".*/\1/')
    category=$(echo "$line" | sed 's/.*GRAY_ERROR("[^"]*", "\([^"]*\)".*/\1/')
    desc=$(echo "$line" | sed 's/\\"/DQUOTE/g' | sed 's/.*GRAY_ERROR("[^"]*", "[^"]*", "\([^"]*\)".*/\1/' | sed 's/DQUOTE/\\"/g')
    echo "| \`${code}\` | ${category} | ${desc} |"
done >> "$OUTPUT"

cat >> "$OUTPUT" << WARN_HEADER

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
WARN_HEADER

# Extract warnings
grep '^ *GRAY_WARNING("' "$CODES_FILE" | while IFS= read -r line; do
    code=$(echo "$line" | sed 's/.*GRAY_WARNING("\([^"]*\)".*/\1/')
    category=$(echo "$line" | sed 's/.*GRAY_WARNING("[^"]*", "\([^"]*\)".*/\1/')
    desc=$(echo "$line" | sed 's/\\"/DQUOTE/g' | sed 's/.*GRAY_WARNING("[^"]*", "[^"]*", "\([^"]*\)".*/\1/' | sed 's/DQUOTE/\\"/g')
    echo "| \`${code}\` | ${category} | ${desc} |"
done >> "$OUTPUT"

cat >> "$OUTPUT" << PANIC_HEADER

---

## Panics

Runtime panics are fatal errors that terminate the program immediately. They are not compiler errors — they happen at runtime when the program encounters an unrecoverable condition.

| Code | Category | Description |
|------|----------|-------------|
PANIC_HEADER

# Extract panics
grep '^ *GRAY_PANIC("' "$CODES_FILE" | while IFS= read -r line; do
    code=$(echo "$line" | sed 's/.*GRAY_PANIC("\([^"]*\)".*/\1/')
    category=$(echo "$line" | sed 's/.*GRAY_PANIC("[^"]*", *"\([^"]*\)".*/\1/')
    desc=$(echo "$line" | sed 's/\\"/DQUOTE/g' | sed 's/.*GRAY_PANIC("[^"]*", *"[^"]*", *"\([^"]*\)".*/\1/' | sed 's/DQUOTE/\\"/g')
    echo "| \`${code}\` | ${category} | ${desc} |"
done >> "$OUTPUT"

cat >> "$OUTPUT" << FOOTER

---

## Code Ranges

| Range | What It Means |
|-------|---------------|
| E1xxx | Problems reading your code (invalid characters, malformed literals) |
| E2xxx | Problems understanding your code (missing brackets, unexpected symbols) |
| E3xxx | Type problems (wrong types, invalid operations) |
| E4xxx | Name problems (undefined variables, duplicate names) |
| E5xxx | Usage problems (wrong number of arguments, invalid assignment targets) |
| E6xxx | Import problems (unknown modules, missing files) |
| E7xxx | Standard library problems (wrong usage of built-in functions) |
| E8xxx | Bitwise operator errors (non-integer operands) |
| E9xxx | Array errors (empty array ops, invalid ranges) |
| E12xxx | Map errors (unhashable keys, duplicate keys) |
| W1xxx | Cleanup suggestions (unused variables, functions) |
| W2xxx | Safety warnings (shadowing, unused imports) |
| W3xxx | Quality warnings (uninitialized elements, pointer lifetime) |
| P0xxx | Runtime panics (overflow, out-of-bounds, nil dereference) |

---

*Generated on $(date -u '+%Y-%m-%d %H:%M:%S UTC')*
FOOTER

echo "Generated $OUTPUT ($TOTAL codes: $ERROR_COUNT errors, $WARNING_COUNT warnings, $PANIC_COUNT panics)"
