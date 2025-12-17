# Typechecker Status Report

**Date:** December 17, 2024
**Version:** 0.23.1

## Overview

The typechecker (`./ez check`) runs before runtime as a defense layer to catch errors early. This document assesses what it currently catches, what slips through, and what improvements are needed.

---

## Does the typechecker run before runtime?

**YES** - `./ez check` runs the typechecker independently. It's the wall before execution.

---

## What it CURRENTLY catches

| Check | Status |
|-------|--------|
| Type mismatches (string where int expected) | Caught |
| Wrong argument count | Caught |
| Missing return statement | Caught |
| Dead code after return | Caught |
| Modifying const | Caught |
| Invalid struct field access | Caught |
| Calling non-function as function | Caught |
| Invalid operation types (string + int) | Caught |
| Function return type mismatch | Caught |

---

## What it SHOULD catch but DOESN'T (Gaps)

| Issue | Example | Check Result | Runtime Result |
|-------|---------|--------------|----------------|
| Undefined variable | `println(x)` | Pass | Crash |
| Undefined function | `foo()` | Pass | Crash |
| Use before declaration | `println(x); temp x = 5` | Pass | Crash |
| Out-of-range literal | `temp x i8 = 200` | Pass | Overflow |
| Compile-time div by zero | `temp x = 10 / 0` | Pass | Crash |
| Variable redeclaration | `temp x = 5; temp x = 10` | Pass | Crash |

---

## What CAN'T be caught at check time (fundamentally impossible)

- **Runtime user input** - `temp x = input(); temp y = 10 / x` (can't know if x is 0)
- **Dynamic array indices** - `arr[i]` where i comes from runtime
- **External data** - File contents, network responses
- **Complex control flow** - Values that depend on runtime conditions

---

## Priority Ranking for Fixes

1. **HIGH** - Undefined variables/functions (most common beginner error)
2. **HIGH** - Variable redeclaration in same scope
3. **MEDIUM** - Use before declaration
4. **MEDIUM** - Literal range checking for sized types
5. **LOW** - Constant expression div-by-zero (edge case)

---

## Related Issues

- [ ] #663 - Undefined variable/function detection (HIGH)
- [ ] #664 - Variable redeclaration in same scope (HIGH)
- [ ] #665 - Use before declaration (MEDIUM)
- [ ] #666 - Literal range checking for sized types (MEDIUM)
- [ ] #667 - Constant expression evaluation (LOW)

---

## Bottom Line

The typechecker catches type-system errors well but **misses name resolution errors entirely**. Users can write code that "passes check" but crashes immediately at runtime due to typos or undefined references. That's the biggest hole in the wall.
