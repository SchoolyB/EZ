#!/bin/bash
#
# EZ Integration Test Runner
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.
#
# This script runs all integration tests and reports results.
# - Tests in pass/ should execute successfully
# - Tests in fail/ should fail with expected errors
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EZ_BIN="$PROJECT_ROOT/ez"

# Check if ez binary exists
if [ ! -f "$EZ_BIN" ]; then
    # Try building it
    echo "EZ binary not found, building..."
    cd "$PROJECT_ROOT"
    go build -o ez ./cmd/ez
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

echo "========================================"
echo "  EZ Integration Test Suite"
echo "========================================"
echo ""

# Run pass tests (should succeed)
echo "Running PASS tests..."
echo "----------------------------------------"

# Core tests
for test_file in "$SCRIPT_DIR"/pass/core/*.ez; do
    if [ -f "$test_file" ]; then
        test_name=$(basename "$test_file" .ez)
        printf "  core/%s... " "$test_name"

        if output=$("$EZ_BIN" "$test_file" 2>&1); then
            if echo "$output" | grep -q "SOME TESTS FAILED"; then
                echo -e "${RED}FAIL${NC} (test assertions failed)"
                ((FAIL_COUNT++))
            else
                echo -e "${GREEN}PASS${NC}"
                ((PASS_COUNT++))
            fi
        else
            echo -e "${RED}FAIL${NC} (execution error)"
            ((FAIL_COUNT++))
        fi
    fi
done

# Stdlib tests
for test_file in "$SCRIPT_DIR"/pass/stdlib/*.ez; do
    if [ -f "$test_file" ]; then
        test_name=$(basename "$test_file" .ez)
        printf "  stdlib/%s... " "$test_name"

        if output=$("$EZ_BIN" "$test_file" 2>&1); then
            if echo "$output" | grep -q "SOME TESTS FAILED"; then
                echo -e "${RED}FAIL${NC} (test assertions failed)"
                ((FAIL_COUNT++))
            else
                echo -e "${GREEN}PASS${NC}"
                ((PASS_COUNT++))
            fi
        else
            echo -e "${RED}FAIL${NC} (execution error)"
            ((FAIL_COUNT++))
        fi
    fi
done

# Multi-file tests
for dir in "$SCRIPT_DIR"/pass/multi-file/*/; do
    if [ -d "$dir" ]; then
        dir_name=$(basename "$dir")
        # Find main.ez in the directory (may be nested)
        main_file=$(find "$dir" -name "main.ez" | head -1)

        if [ -n "$main_file" ]; then
            printf "  multi-file/%s... " "$dir_name"

            if output=$("$EZ_BIN" "$main_file" 2>&1); then
                echo -e "${GREEN}PASS${NC}"
                ((PASS_COUNT++))
            else
                echo -e "${RED}FAIL${NC} (execution error)"
                ((FAIL_COUNT++))
            fi
        fi
    fi
done

echo ""
echo "Running FAIL tests (expecting errors)..."
echo "----------------------------------------"

# Error tests (should fail)
for test_file in "$SCRIPT_DIR"/fail/errors/*.ez; do
    if [ -f "$test_file" ]; then
        test_name=$(basename "$test_file" .ez)
        printf "  errors/%s... " "$test_name"

        if "$EZ_BIN" "$test_file" >/dev/null 2>&1; then
            echo -e "${RED}FAIL${NC} (expected error, got success)"
            ((FAIL_COUNT++))
        else
            echo -e "${GREEN}PASS${NC}"
            ((PASS_COUNT++))
        fi
    fi
done

# Cleanup any .ezdb files created by error tests (they are created in cwd, not the test directory)
rm -f ./*.ezdb

# Multi-file error tests (single files)
for test_file in "$SCRIPT_DIR"/fail/multi-file/*.ez; do
    if [ -f "$test_file" ]; then
        test_name=$(basename "$test_file" .ez)
        printf "  multi-file/%s... " "$test_name"

        if "$EZ_BIN" "$test_file" >/dev/null 2>&1; then
            echo -e "${RED}FAIL${NC} (expected error, got success)"
            ((FAIL_COUNT++))
        else
            echo -e "${GREEN}PASS${NC}"
            ((PASS_COUNT++))
        fi
    fi
done

# Multi-file error tests (directories with main.ez)
for dir in "$SCRIPT_DIR"/fail/multi-file/*/; do
    if [ -d "$dir" ]; then
        dir_name=$(basename "$dir")
        main_file=$(find "$dir" -name "main.ez" | head -1)

        if [ -n "$main_file" ]; then
            printf "  multi-file/%s... " "$dir_name"

            if "$EZ_BIN" "$main_file" >/dev/null 2>&1; then
                echo -e "${RED}FAIL${NC} (expected error, got success)"
                ((FAIL_COUNT++))
            else
                echo -e "${GREEN}PASS${NC}"
                ((PASS_COUNT++))
            fi
        fi
    fi
done

# Summary
echo ""
echo "========================================"
echo "  Test Summary"
echo "========================================"
echo ""
echo -e "  ${GREEN}Passed:${NC}  $PASS_COUNT"
echo -e "  ${RED}Failed:${NC}  $FAIL_COUNT"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "  ${GREEN}ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "  ${RED}SOME TESTS FAILED${NC}"
    exit 1
fi
