# Integration Test Suite Overhaul

This document tracks the work needed to fix and restructure the EZ integration test suite.

## Related Issues
- [#408](https://github.com/SchoolyB/EZ/issues/408) - Test Suite Lacks Assertions for Standard Library Functions
- [#356](https://github.com/SchoolyB/EZ/issues/356) - Restructure Integration Test Directory

---

## Problem Summary

### Issue #408: Broken Test Assertions
The comprehensive test suite gives a false sense of security. Tests pass unconditionally without validating actual behavior.

**Current state:**
- 253 `passed += 1` statements (many unconditional)
- 116 `failed += 1` statements
- ~137 tests auto-pass without any assertion

**Example of broken pattern:**
```ez
// BAD: Just prints and passes regardless of result
temp arr [int] = {3, 1, 2}
arrays.reverse(arr)
println("  arrays.reverse() = ${arr}")
passed += 1  // <-- Always passes even if reverse() did nothing!
```

**Bugs that shipped due to this:**
- `arrays.reverse()` - does nothing, returns unchanged array
- `arrays.remove()` - off-by-one bug (not even tested)
- `arrays.sort()` - fails to sort (#406)
- `arrays.insert()` - fails at valid indices (#405)

### Issue #356: Poor Directory Structure
Tests are disorganized and hard to maintain. Need clear separation of pass/fail tests.

---

## Phase 1: Directory Restructuring

### Current Structure
```
tests/
├── comprehensive.ez          # Monolithic 81KB file
├── errors/
│   ├── comprehensive/        # Individual error code tests
│   └── multi-file/
├── import-use/
├── multi-file/
├── nested-test/
└── stdlib/
```

### Target Structure
```
integration-tests/
├── pass/
│   ├── core/
│   │   ├── variables.ez
│   │   ├── primitives.ez
│   │   ├── operators.ez
│   │   ├── control-flow.ez
│   │   ├── functions.ez
│   │   ├── structs.ez
│   │   ├── enums.ez
│   │   ├── arrays.ez
│   │   ├── maps.ez
│   │   └── matrices.ez
│   ├── stdlib/
│   │   ├── std.ez
│   │   ├── math.ez
│   │   ├── arrays.ez
│   │   ├── strings.ez
│   │   ├── maps.ez
│   │   ├── time.ez
│   │   ├── io.ez
│   │   ├── os.ez
│   │   └── random.ez
│   └── multi-file/
│       ├── basic/
│       ├── imports/
│       └── nested/
├── fail/
│   ├── errors/               # Move from tests/errors/comprehensive
│   └── multi-file/           # Move from tests/errors/multi-file
└── run_tests.sh              # Unified test runner
```

### Tasks
- [ ] Create `integration-tests/` directory structure
- [ ] Move `tests/errors/` → `integration-tests/fail/`
- [ ] Move multi-file tests → `integration-tests/pass/multi-file/`
- [ ] Split `comprehensive.ez` into focused test files (see Phase 2)
- [ ] Consolidate `tests/stdlib/` into `integration-tests/pass/stdlib/`
- [ ] Create unified `run_tests.sh` runner
- [ ] Update documentation/README
- [ ] Delete old `tests/` directory

---

## Phase 2: Fix Test Assertions

The core problem: tests need to actually assert expected vs actual values.

### Create Assertion Helper
```ez
// In each test file or a shared test_utils module
do assert_equal(actual int, expected int, name string, &passed int, &failed int) {
    if actual == expected {
        println("  ✓ ${name}")
        passed += 1
    } otherwise {
        println("  ✗ ${name}: expected ${expected}, got ${actual}")
        failed += 1
    }
}

do assert_array_equal(actual [int], expected [int], name string, &passed int, &failed int) {
    if len(actual) != len(expected) {
        println("  ✗ ${name}: length mismatch (${len(actual)} vs ${len(expected)})")
        failed += 1
        return
    }
    for (temp i int = 0; i < len(actual); i += 1) {
        if actual[i] != expected[i] {
            println("  ✗ ${name}: mismatch at index ${i}")
            failed += 1
            return
        }
    }
    println("  ✓ ${name}")
    passed += 1
}
```

### Files to Fix (extracted from comprehensive.ez)

Each section needs proper assertions:

#### Core Language Tests
- [ ] `core/variables.ez` - var, const, temp declarations
- [ ] `core/primitives.ez` - int, float, string, bool, char
- [ ] `core/operators.ez` - arithmetic, comparison, logical, assignment
- [ ] `core/control-flow.ez` - if/or/otherwise, for, while, for_each, range
- [ ] `core/functions.ez` - params, returns, multiple returns, mutable params
- [ ] `core/structs.ez` - struct definitions and usage
- [ ] `core/enums.ez` - int enums, string enums, skip attribute
- [ ] `core/arrays.ez` - array literals, indexing, slicing
- [ ] `core/maps.ez` - map literals, access, iteration
- [ ] `core/matrices.ez` - 2D array operations

#### Standard Library Tests (CRITICAL - most broken)
- [ ] `stdlib/std.ez` - print, println, printf, input
- [ ] `stdlib/math.ez` - abs, floor, ceil, sqrt, pow, sin, cos, etc.
- [ ] `stdlib/arrays.ez` - **PRIORITY** - reverse, sort, insert, remove, etc.
- [ ] `stdlib/strings.ez` - split, join, upper, lower, trim, etc.
- [ ] `stdlib/maps.ez` - keys, values, has_key, etc.
- [ ] `stdlib/time.ez` - now, sleep, format, etc.
- [ ] `stdlib/io.ez` - file operations
- [ ] `stdlib/os.ez` - env, exec, args

### Example: Fixed arrays.reverse Test
```ez
// BAD (current):
temp arr [int] = {3, 1, 2}
arrays.reverse(arr)
println("  arrays.reverse() = ${arr}")
passed += 1

// GOOD (fixed):
temp arr [int] = {3, 1, 2}
arrays.reverse(arr)
temp expected [int] = {2, 1, 3}
assert_array_equal(arr, expected, "arrays.reverse({3,1,2})", passed, failed)
```

---

## Phase 3: Add Missing Tests

Functions that aren't tested at all:

### @arrays
- [ ] `arrays.remove()` - has known off-by-one bug
- [ ] `arrays.insert()` - edge cases at start/end
- [ ] `arrays.slice()` - boundary conditions
- [ ] Empty array edge cases for all functions
- [ ] Single element array edge cases

### @strings
- [ ] Edge cases: empty strings, single char
- [ ] Unicode handling

### @maps
- [ ] Empty map operations
- [ ] Key type variations

---

## Phase 4: Test Runner

Create `integration-tests/run_tests.sh`:

```bash
#!/bin/bash

PASS_COUNT=0
FAIL_COUNT=0

echo "Running pass tests..."
for test in integration-tests/pass/**/*.ez; do
    if ./ez run "$test"; then
        ((PASS_COUNT++))
    else
        echo "UNEXPECTED FAILURE: $test"
        ((FAIL_COUNT++))
    fi
done

echo "Running fail tests..."
for test in integration-tests/fail/**/*.ez; do
    if ./ez run "$test" 2>/dev/null; then
        echo "UNEXPECTED SUCCESS: $test"
        ((FAIL_COUNT++))
    else
        ((PASS_COUNT++))
    fi
done

echo "================================"
echo "Pass: $PASS_COUNT  Fail: $FAIL_COUNT"
exit $FAIL_COUNT
```

---

## Execution Order

1. **Phase 1** - Create directory structure (don't delete old tests yet)
2. **Phase 2** - Fix assertions in new test files as we split comprehensive.ez
3. **Phase 3** - Add missing test coverage
4. **Phase 4** - Create test runner and CI integration
5. **Cleanup** - Delete old `tests/` directory, close issues

---

## Notes

- Keep old `tests/` until new structure is fully working
- Run both old and new tests during migration
- Each new test file should be self-contained and runnable
- Use consistent assertion helper pattern across all files
