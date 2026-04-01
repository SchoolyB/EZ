#!/bin/bash
#
# run_parity.sh - Compare ezc output against expected output
#
# Usage: ./scripts/run_parity.sh [file.ez ...]
#   No args: runs all examples/basic/*.ez
#   With args: runs only the specified files
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

# No set -e: we handle errors ourselves

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# Paths
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
EZC="$ROOT_DIR/ezc/ezc"
export EZC_PATH="$EZC"
EZ="go run $ROOT_DIR/cmd/ez/..."
TMP_DIR="/tmp/ezc_parity_$$"

mkdir -p "$TMP_DIR"
trap "rm -rf $TMP_DIR" EXIT

# Counters
PASS=0
FAIL=0
SKIP=0
COMPILE_FAIL=0
TOTAL=0

# Examples that use features not yet fully working in EZC
SKIP_LIST=(
    "function_references" # arrays.append inside FCF causes OOM in codegen
)

should_skip() {
    local name="$1"
    for skip in "${SKIP_LIST[@]}"; do
        if [ "$name" = "$skip" ]; then
            return 0
        fi
    done
    return 1
}

run_test() {
    local file="$1"
    local name
    name=$(basename "$file" .ez)
    TOTAL=$((TOTAL + 1))

    if should_skip "$name"; then
        printf "  ${YELLOW}SKIP${RESET}  %s (unimplemented features)\n" "$name"
        SKIP=$((SKIP + 1))
        return
    fi

    # Compile with ezc
    local bin="$TMP_DIR/$name"
    local compile_out
    compile_out=$("$EZC" "$file" -o "$bin" 2>&1)
    if [ $? -ne 0 ]; then
        printf "  ${RED}FAIL${RESET}  %s (compile error)\n" "$name"
        echo "$compile_out" | head -3 | sed 's/^/        /'
        COMPILE_FAIL=$((COMPILE_FAIL + 1))
        FAIL=$((FAIL + 1))
        return
    fi

    # Run compiled binary
    local ezc_out
    ezc_out=$("$bin" 2>&1) || true

    # Run with ez CLI (expected output)
    local ez_out
    ez_out=$(cd "$ROOT_DIR" && $EZ "$file" 2>&1 | sed '/^Note:/d' | sed '/^$/{ N; /^\n$/d; }' | sed '1{ /^$/d; }') || true

    # Compare
    if [ "$ezc_out" = "$ez_out" ]; then
        printf "  ${GREEN}PASS${RESET}  %s\n" "$name"
        PASS=$((PASS + 1))
    else
        printf "  ${RED}FAIL${RESET}  %s (output mismatch)\n" "$name"
        # Show first diff
        diff <(echo "$ezc_out") <(echo "$ez_out") | head -10 | sed 's/^/        /'
        FAIL=$((FAIL + 1))
    fi
}

# Header
echo ""
printf "${BOLD}EZC Parity Tests${RESET}\n"
echo "Comparing compiled output against expected output"
echo ""

# Build ezc if needed
if [ ! -f "$EZC" ]; then
    echo "Building ezc..."
    (cd "$ROOT_DIR/ezc" && make build) || { echo "Build failed"; exit 1; }
fi

# Run tests
if [ $# -gt 0 ]; then
    # Run specified files
    for file in "$@"; do
        run_test "$file"
    done
else
    # Run all basic examples
    for file in "$ROOT_DIR"/examples/basic/*.ez; do
        run_test "$file"
    done
fi

# Summary
echo ""
printf "${BOLD}Results:${RESET} "
if [ $FAIL -eq 0 ]; then
    printf "${GREEN}%d passed${RESET}" "$PASS"
else
    printf "${GREEN}%d passed${RESET}, ${RED}%d failed${RESET}" "$PASS" "$FAIL"
fi
if [ $SKIP -gt 0 ]; then
    printf ", ${YELLOW}%d skipped${RESET}" "$SKIP"
fi
printf " (${TOTAL} total)\n"

if [ $COMPILE_FAIL -gt 0 ]; then
    printf "${RED}%d compilation failures${RESET}\n" "$COMPILE_FAIL"
fi

echo ""

# Exit code
if [ $FAIL -gt 0 ]; then
    exit 1
fi
exit 0
