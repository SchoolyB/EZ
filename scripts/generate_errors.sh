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

# Count errors and warnings (only lines starting with whitespace + EZ_ERROR/EZ_WARNING)
ERROR_COUNT=$(grep -c '^ *EZ_ERROR("' "$CODES_FILE" || echo 0)
WARNING_COUNT=$(grep -c '^ *EZ_WARNING("' "$CODES_FILE" || echo 0)
TOTAL=$((ERROR_COUNT + WARNING_COUNT))

# Generate markdown
cat > "$OUTPUT" << HEADER
# EZ Error Code Reference

> Auto-generated from \`ezc/src/util/error_codes.h\`. Do not edit manually.
> Run \`./scripts/generate_errors.sh\` to regenerate.

**Total: ${TOTAL} codes** (${ERROR_COUNT} errors, ${WARNING_COUNT} warnings)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
HEADER

# Extract errors (only actual definitions, not comments)
grep '^ *EZ_ERROR("' "$CODES_FILE" | while IFS= read -r line; do
    code=$(echo "$line" | sed 's/.*EZ_ERROR("\([^"]*\)".*/\1/')
    category=$(echo "$line" | sed 's/.*EZ_ERROR("[^"]*", "\([^"]*\)".*/\1/')
    desc=$(echo "$line" | sed 's/.*EZ_ERROR("[^"]*", "[^"]*", "\([^"]*\)".*/\1/')
    echo "| \`${code}\` | ${category} | ${desc} |"
done >> "$OUTPUT"

cat >> "$OUTPUT" << WARN_HEADER

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
WARN_HEADER

# Extract warnings (only actual definitions, not comments)
grep '^ *EZ_WARNING("' "$CODES_FILE" | while IFS= read -r line; do
    code=$(echo "$line" | sed 's/.*EZ_WARNING("\([^"]*\)".*/\1/')
    category=$(echo "$line" | sed 's/.*EZ_WARNING("[^"]*", "\([^"]*\)".*/\1/')
    desc=$(echo "$line" | sed 's/.*EZ_WARNING("[^"]*", "[^"]*", "\([^"]*\)".*/\1/')
    echo "| \`${code}\` | ${category} | ${desc} |"
done >> "$OUTPUT"

cat >> "$OUTPUT" << FOOTER

---

## Error Code Ranges

| Range | Category |
|-------|----------|
| E1xxx | Lexer errors |
| E2xxx | Parser errors |
| E3xxx | Type errors |
| E4xxx | Reference/name errors |
| E5xxx | Runtime errors |
| E6xxx | Import errors |
| E7xxx | Stdlib validation errors |
| W2xxx | Unused code warnings |
| W3xxx | Type warnings |

---

*Generated on $(date -u '+%Y-%m-%d %H:%M:%S UTC')*
FOOTER

echo "Generated $OUTPUT ($TOTAL codes: $ERROR_COUNT errors, $WARNING_COUNT warnings)"
