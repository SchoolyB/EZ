#!/bin/bash
#
# Error Test Runner
# Runs all error test files and verifies they produce errors
#
# Usage: ./run_error_tests.sh [ez_binary_path]
#

EZ_BIN="${1:-../../ez}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

total=0
passed=0
failed=0

echo "========================================"
echo "  EZ Error Test Runner"
echo "========================================"
echo ""

run_error_test() {
    local test_file="$1"
    local test_name=$(basename "$test_file" .ez)

    total=$((total + 1))

    # Run the test and capture output (stderr and stdout)
    output=$("$EZ_BIN" run "$test_file" 2>&1)

    # Error tests should produce error output (contains "error" or "ERROR")
    if echo "$output" | grep -qi "error"; then
        # Extract error code if present
        error_line=$(echo "$output" | grep -i "error" | head -1)
        echo -e "  ${GREEN}PASS${NC}: $test_name"
        echo "        $error_line"
        passed=$((passed + 1))
    else
        echo -e "  ${RED}FAIL${NC}: $test_name"
        echo "        Expected error but got: $(echo "$output" | head -1)"
        failed=$((failed + 1))
    fi
}

# Run comprehensive error tests
echo "------------------------------------------"
echo "  Comprehensive Error Tests"
echo "------------------------------------------"
for test_file in "$SCRIPT_DIR/comprehensive/"*.ez; do
    if [ -f "$test_file" ]; then
        run_error_test "$test_file"
    fi
done
echo ""

# Run multi-file error tests
echo "------------------------------------------"
echo "  Multi-File Error Tests"
echo "------------------------------------------"
for test_file in "$SCRIPT_DIR/multi-file/"*.ez; do
    if [ -f "$test_file" ]; then
        run_error_test "$test_file"
    fi
done
echo ""

# Run nested-test error tests
echo "------------------------------------------"
echo "  Nested Module Error Tests"
echo "------------------------------------------"
for test_file in "$SCRIPT_DIR/nested-test/"*.ez; do
    if [ -f "$test_file" ]; then
        run_error_test "$test_file"
    fi
done
echo ""

# Summary
echo "========================================"
echo "  Error Tests Complete!"
echo "========================================"
echo ""
echo "  Total:  $total"
echo "  Passed: $passed"
echo "  Failed: $failed"
echo ""

if [ $failed -eq 0 ]; then
    echo -e "  Status: ${GREEN}ALL ERROR TESTS PASSED!${NC}"
    exit 0
else
    echo -e "  Status: ${RED}SOME ERROR TESTS FAILED${NC}"
    exit 1
fi
