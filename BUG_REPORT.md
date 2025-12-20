# EZ Language Bug Report

**Date:** 2025-12-19
**Branch:** `bugfixes/test-coverage-bug-fixes`

---

## Fixed Bugs

### BUG #1: Unchecked Array Access in ReturnValue - **FIXED** (PR #743)
- **File:** `pkg/interpreter/evaluator.go:774`
- **Issue:** Accessed `result.Values[0]` without bounds check
- **Fix:** Added length check before access

### BUG #2: Ignored Deref() Error in Compound Assignment - **FIXED** (PR #744)
- **File:** `pkg/interpreter/evaluator.go:1142`
- **Issue:** Ignored error return from `ref.Deref()`
- **Fix:** Added error check and proper error return

### BUG #3: Missing Error Checks on File Close - **FIXED** (PR #745)
- **File:** `pkg/stdlib/io.go:63,69,75`
- **Issue:** Ignored `Close()` errors in error paths
- **Fix:** Made ignored errors explicit with blank identifier

---

## Open Bugs

### BUG #4: Unchecked Type Assertions in arrays.range()

**File:** `pkg/stdlib/arrays.go`
**Lines:** 923, 927-928, 931-933
**Severity:** High

**Code:**
```go
if len(args) == 1 {
    end = args[0].(*object.Integer).Value.Int64()  // No type check!
} else if len(args) == 2 {
    start = args[0].(*object.Integer).Value.Int64()  // No type check!
    end = args[1].(*object.Integer).Value.Int64()    // No type check!
} else {
    start = args[0].(*object.Integer).Value.Int64()  // No type check!
    end = args[1].(*object.Integer).Value.Int64()    // No type check!
    step = args[2].(*object.Integer).Value.Int64()   // No type check!
}
```

**Description:**
The function validates argument count but does NOT validate that arguments are Integers before casting. Passing a Float or String will cause a panic.

**Impact:** Crash on `arrays.range(1.5)` or `arrays.range("a")`

**Suggested Fix:**
```go
intArg, ok := args[0].(*object.Integer)
if !ok {
    return &object.Error{Code: "E7004", Message: "arrays.range() requires integer arguments"}
}
end = intArg.Value.Int64()
```

---

### BUG #5: Integer-to-Byte Conversion Without Bounds Check

**File:** `pkg/stdlib/bytes.go`
**Lines:** 167, 1014
**Severity:** Medium

**Code:**
```go
// Line 167 (bytes.to_string)
if intVal, ok := elem.(*object.Integer); ok {
    data[i] = byte(intVal.Value.Int64())  // No bounds check!
    continue
}

// Line 1014 (slice_data helper)
case *object.Integer:
    data[i] = byte(e.Value.Int64())  // No bounds check!
```

**Description:**
When converting integers to bytes, there's no validation that the value is in the valid range (0-255). Go silently truncates out-of-range values.

**Impact:** `bytes.to_string([256, 0])` silently converts 256 to 0 instead of erroring

**Suggested Fix:**
```go
val := intVal.Value.Int64()
if val < 0 || val > 255 {
    return &object.Error{Code: "E7005", Message: "byte value must be 0-255"}
}
data[i] = byte(val)
```

---

## Additional Patterns to Review

| File | Line | Pattern |
|------|------|---------|
| `pkg/typechecker/typechecker.go` | 637 | `varType, _ = tc.inferExpressionType(node.Value)` |
| `pkg/typechecker/typechecker.go` | 714 | `structType, _ := tc.GetType(node.Name.Value)` |
| `pkg/stdlib/json.go` | 472 | `hash, _ := object.HashKey(keyObj)` |
| `pkg/stdlib/json.go` | 498 | `bytes, _ := json.Marshal(v.Value)` |
| `pkg/stdlib/json.go` | 511 | `bytes, _ := json.Marshal(string(v.Value))` |

---

## Verified Safe Patterns

| Location | Pattern | Reason |
|----------|---------|--------|
| math.go:113,148 | `args[0].(*object.Integer)` | `getNumber()` validates type first |
| evaluator.go:836 | `returnVal.Values[i]` | Bounds checked at lines 825-827 |
| evaluator.go:2911-2912 | `v.Values[0]` | `len(v.Values) == 1` checked first |

---

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| High | 1 | BUG #4 - arrays.range() |
| Medium | 1 | BUG #5 - bytes.go bounds |
| Fixed | 3 | PRs #743, #744, #745 |

**Open bugs:** 2
**Fixed bugs:** 3
