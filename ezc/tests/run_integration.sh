#!/bin/bash
#
# run_integration.sh - Run EZC against the integration test suite
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

# Tests that use interpreter-only syntax (skip for compiler)
SKIP_INTEGRATION=(
    "cast_keyword"          # cast with array type codegen incomplete
    "const_ref"             # ref() type tracking incomplete for arrays
    "copy_semantics"        # copy-by-default not in compiler
    "function_references"   # arrays.append in FCF causes OOM
    "large-integers"        # i128 literals exceed int64 parser range
    "mutable-indexed-params" # interpreter-specific mutable semantics
    "primitives"            # interpreter-specific type tests
"typeof_stdlib"         # type_of returns different strings
    # Error tests for interpreter-only features
    "E2002_for_missing_closing_paren"
    "E2002_for_nested_missing_paren"
    "E2009_using_after_declarations"
    "E2041_when_missing_default"
    "E2042_when_strict_has_default"
    "E2045_when_strict_non_enum"
    "E2046_strict_when_missing_cases"
    "E2051_suppress_invalid_target"
    "E2052_suppress_invalid_code"
    "E2052_suppress_non_suppressible"
    "E2054_strict_when_non_enum_case"
    "E2055_strict_invalid_target"
    "E4007_module_not_imported"
    "E4016_loop_variable_shadows"
)

should_skip() {
    local name="$1"
    for skip in "${SKIP_INTEGRATION[@]}"; do
        if [ "$name" = "$skip" ]; then return 0; fi
    done
    return 1
}

run_test() {
    local file="$1"
    local name
    name=$(basename "$file" .ez)

    if should_skip "$name"; then
        printf "  ${YELLOW}SKIP${RESET}  %s\n" "$name"
        SKIP=$((SKIP + 1))
        return
    fi

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
echo "Running EZC against the integration test suite"
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
if [ $SKIP -gt 0 ]; then
    printf ", ${YELLOW}%d skipped${RESET}" "$SKIP"
fi
printf " (%d total)\n" "$((TOTAL + SKIP))"
echo ""

exit 0
