#!/bin/bash
#
# run_integration.sh - Run EZC against the interpreter's integration test suite
#
# Usage: ./ezc/tests/run_integration.sh [--verbose]
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
EZC="$ROOT_DIR/ezc/ezc"
TMP_DIR="/tmp/ezc_integration_$$"

mkdir -p "$TMP_DIR"
trap "rm -rf $TMP_DIR" EXIT

PASS=0
FAIL=0
SKIP=0
TOTAL=0
VERBOSE=false

if [ "$1" = "--verbose" ]; then
    VERBOSE=true
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
RESET='\033[0m'

run_test() {
    local file="$1"
    local name
    name=$(basename "$file" .ez)
    TOTAL=$((TOTAL + 1))

    local bin="$TMP_DIR/$name"
    local compile_out
    compile_out=$("$EZC" "$file" -o "$bin" 2>&1)
    if [ $? -eq 0 ]; then
        # Try to run the binary
        local run_out
        run_out=$("$bin" 2>&1)
        if [ $? -eq 0 ]; then
            printf "  ${GREEN}PASS${RESET}  %s\n" "$name"
            PASS=$((PASS + 1))
        else
            printf "  ${RED}FAIL${RESET}  %s (runtime error)\n" "$name"
            if $VERBOSE; then echo "$run_out" | head -3 | sed 's/^/        /'; fi
            FAIL=$((FAIL + 1))
        fi
    else
        printf "  ${RED}FAIL${RESET}  %s (compile error)\n" "$name"
        if $VERBOSE; then echo "$compile_out" | head -3 | sed 's/^/        /'; fi
        FAIL=$((FAIL + 1))
    fi
}

echo ""
printf "${BOLD}EZC Integration Tests${RESET}\n"
echo "Running EZC against the interpreter's test suite"
echo ""

# Build if needed
if [ ! -f "$EZC" ]; then
    echo "Building ezc..."
    (cd "$ROOT_DIR/ezc" && make build) || { echo "Build failed"; exit 1; }
fi

# Core tests
printf "${BOLD}Core Tests:${RESET}\n"
for file in "$ROOT_DIR"/integration-tests/pass/core/*.ez; do
    run_test "$file"
done

# Named returns tests
echo ""
printf "${BOLD}Named Returns:${RESET}\n"
if [ -d "$ROOT_DIR/integration-tests/pass/named_returns" ]; then
    for file in "$ROOT_DIR"/integration-tests/pass/named_returns/*.ez; do
        [ -f "$file" ] && run_test "$file"
    done
fi

# Stdlib tests (compiler-specific _c.ez files)
echo ""
printf "${BOLD}Stdlib Tests:${RESET}\n"
if [ -d "$ROOT_DIR/integration-tests/pass/stdlib" ]; then
    for file in "$ROOT_DIR"/integration-tests/pass/stdlib/*_c.ez; do
        [ -f "$file" ] && run_test "$file"
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
printf " (${TOTAL} total)\n"
echo ""

exit 0
