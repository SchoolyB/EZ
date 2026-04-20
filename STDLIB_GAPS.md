# EZ Stdlib Gap Report — Engineering Action Items

**Date:** 2026-04-20
**Scope:** Every function in STANDARD.md vs actual compiler implementation.
**Files involved:**
- Typechecker: `ezc/src/typechecker/typechecker.c`
- Codegen: `ezc/src/codegen/codegen.c`
- C runtime headers: `ezc/src/stdlib/ez_*.h`
- C runtime implementations: `ezc/src/stdlib/ez_*.c`

---

## FIX 1: json.encode segfaults on arrays and maps [CRITICAL]

**Symptom:** `json.encode(arr)` or `json.encode(m)` crashes at runtime (exit 139).

**Root cause:** In `codegen.c` `emit_json_call()` (line 3197), the `"encode"` handler has cases for `TK_MAP`, `TK_INT`, `TK_FLOAT`, `TK_BOOL`, `TK_STRING` — but **no case for `TK_ARRAY`**. Arrays fall through to the `else` block (line 3226) which casts the value to `(EzMap *)` and calls `ez_json_encode_map()` — this reinterprets array memory as a map and segfaults.

**File:** `ezc/src/codegen/codegen.c`, lines 3197-3231

**Current code (lines 3225-3230):**
```c
        } else {
            /* Fallback: store in temp to allow & */
            emit(cg, "({ __auto_type _jtmp = ");
            emit_expression(cg, arg);
            emit(cg, "; ez_json_encode_map(ez_default_arena, (EzMap *)&_jtmp); })");
        }
```

**Fix:** Add a `TK_ARRAY` handler before the `else` fallback. You need a new C function `ez_json_encode_array()` in `ez_json.h`/`ez_json.c`, or serialize the array inline using a loop that encodes each element. Insert before line 3225:
```c
        } else if (arg_t && arg_t->kind == TK_ARRAY) {
            /* Array: serialize as JSON array */
            emit(cg, "ez_json_encode_array(ez_default_arena, &");
            emit_expression(cg, arg);
            emit(cg, ")");
        }
```

Then implement `ez_json_encode_array()` in `ez_json.h` and `ez_json.c`.

**Verify:**
```bash
echo 'import @json
do main() {
    mut arr [int] = {1, 2, 3}
    println(json.encode(arr))
    mut m map[string:int] = {"a": 1}
    println(json.encode(m))
}' > /tmp/fix1.ez && ./ezc/ezc /tmp/fix1.ez -o /tmp/fix1 && /tmp/fix1
# Expected: [1,2,3] and {"a":1} — no segfault
```

---

## FIX 2: Register 8 missing math functions in typechecker [CRITICAL]

**Symptom:** `math.cbrt(27.0)` → `E4005: module 'math' has no function named 'cbrt'`

**Root cause:** The codegen handles these functions (they exist in `ez_math.h`/`ez_math.c` and in the `func_to_mod` dispatch table), but the typechecker doesn't register them.

**Missing functions:** `cbrt`, `hypot`, `exp2`, `log_base`, `deg_to_rad`, `rad_to_deg`, `lerp`, `distance`

**File:** `ezc/src/typechecker/typechecker.c`, lines 1846-1858

**Current code (float-returning math functions):**
```c
                } else if (strcmp(mfn, "sqrt") == 0 || strcmp(mfn, "pow") == 0 ||
                           strcmp(mfn, "exp") == 0 || strcmp(mfn, "log") == 0 ||
                           strcmp(mfn, "log2") == 0 || strcmp(mfn, "log10") == 0 ||
                           strcmp(mfn, "sin") == 0 || strcmp(mfn, "cos") == 0 ||
                           strcmp(mfn, "tan") == 0 || strcmp(mfn, "asin") == 0 ||
                           strcmp(mfn, "acos") == 0 || strcmp(mfn, "atan") == 0 ||
                           strcmp(mfn, "atan2") == 0 || strcmp(mfn, "sinh") == 0 ||
                           strcmp(mfn, "cosh") == 0 || strcmp(mfn, "tanh") == 0 ||
                           strcmp(mfn, "floor") == 0 || strcmp(mfn, "ceil") == 0 ||
                           strcmp(mfn, "round") == 0 || strcmp(mfn, "trunc") == 0 ||
                           strcmp(mfn, "mod") == 0 || strcmp(mfn, "pi") == 0 ||
                           strcmp(mfn, "e") == 0 || strcmp(mfn, "tau") == 0) {
                    result = &TYPE_FLOAT;
```

**Fix:** Add the 8 missing functions to this block. They all return float:
```c
                } else if (strcmp(mfn, "sqrt") == 0 || strcmp(mfn, "pow") == 0 ||
                           strcmp(mfn, "exp") == 0 || strcmp(mfn, "exp2") == 0 ||
                           strcmp(mfn, "log") == 0 ||
                           strcmp(mfn, "log2") == 0 || strcmp(mfn, "log10") == 0 ||
                           strcmp(mfn, "log_base") == 0 ||
                           strcmp(mfn, "sin") == 0 || strcmp(mfn, "cos") == 0 ||
                           strcmp(mfn, "tan") == 0 || strcmp(mfn, "asin") == 0 ||
                           strcmp(mfn, "acos") == 0 || strcmp(mfn, "atan") == 0 ||
                           strcmp(mfn, "atan2") == 0 || strcmp(mfn, "sinh") == 0 ||
                           strcmp(mfn, "cosh") == 0 || strcmp(mfn, "tanh") == 0 ||
                           strcmp(mfn, "floor") == 0 || strcmp(mfn, "ceil") == 0 ||
                           strcmp(mfn, "round") == 0 || strcmp(mfn, "trunc") == 0 ||
                           strcmp(mfn, "cbrt") == 0 || strcmp(mfn, "hypot") == 0 ||
                           strcmp(mfn, "deg_to_rad") == 0 || strcmp(mfn, "rad_to_deg") == 0 ||
                           strcmp(mfn, "lerp") == 0 || strcmp(mfn, "distance") == 0 ||
                           strcmp(mfn, "mod") == 0 || strcmp(mfn, "pi") == 0 ||
                           strcmp(mfn, "e") == 0 || strcmp(mfn, "tau") == 0) {
                    result = &TYPE_FLOAT;
```

**Verify:**
```bash
echo 'import @math
do main() {
    println(math.cbrt(27.0))
    println(math.hypot(3.0, 4.0))
    println(math.exp2(3.0))
    println(math.log_base(81.0, 3.0))
    println(math.deg_to_rad(180.0))
    println(math.rad_to_deg(3.14159))
    println(math.lerp(0.0, 10.0, 0.5))
    println(math.distance(0.0, 0.0, 3.0, 4.0))
}' > /tmp/fix2.ez && ./ezc/ezc /tmp/fix2.ez -o /tmp/fix2 && /tmp/fix2
# Expected: 3.0, 5.0, 8.0, 4.0, ~3.14, ~180.0, 5.0, 5.0
```

**No changes needed in codegen or C runtime — they already exist.**

---

## FIX 3: fmt.pad_left/pad_right/center char codegen bug [CRITICAL]

**Symptom:** `fmt.pad_left("hi", 5, ch)` → C compile error: `member reference base type 'int32_t' is not a structure or union`

**Root cause:** Codegen emits `(char_expr).data[0]` — this treats the char argument as an EzString and tries to access `.data[0]`. But `char` is `int32_t` in C, not a struct.

**File:** `ezc/src/codegen/codegen.c`, lines 3905-3936

**Current code (line 3910-3912 for pad_left, same pattern on 3921-3923, 3932-3934):**
```c
        emit(cg, ", (");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ").data[0])");
```

**Fix:** Replace `.data[0]` with just `)` — the char value is already a `char`-compatible int:
```c
        emit(cg, ", (char)(");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "))");
```

Apply this change in all three places (pad_left at line 3912, pad_right at line 3923, center at line 3934).

**Verify:**
```bash
echo 'import @fmt
do main() {
    mut ch char = '0'
    println(fmt.pad_left("hi", 5, ch))
    println(fmt.pad_right("hi", 5, ch))
    println(fmt.center("hi", 6, ch))
}' > /tmp/fix3.ez && ./ezc/ezc /tmp/fix3.ez -o /tmp/fix3 && /tmp/fix3
# Expected: 000hi, hi000, 00hi00
```

---

## FIX 4: csv.parse/decode name mismatch [HIGH]

**Symptom:** `csv.parse()` passes typechecker but codegen emits broken C (missing arena arg). `csv.decode()` fails at typechecker with E4005.

**Root cause:** Typechecker registers `"parse"` (line 1609). Codegen handler checks for `"decode"` (line 3165). Neither path works end-to-end.

**File 1:** `ezc/src/typechecker/typechecker.c`, line 1609

**Current code:**
```c
                if (strcmp(mfn, "parse") == 0 || strcmp(mfn, "read") == 0 ||
```

**Fix (option A — rename typechecker to match STANDARD):** Change `"parse"` to `"decode"`:
```c
                if (strcmp(mfn, "decode") == 0 || strcmp(mfn, "read") == 0 ||
```

**OR Fix (option B — rename codegen to match typechecker):** In `codegen.c` line 3165, change `"decode"` to `"parse"`:
```c
    if (strcmp(func, "parse") == 0) {
```

**Recommendation:** Use option A — the STANDARD says `decode`, so align the typechecker to `decode`.

**Verify:**
```bash
echo 'import @csv
do main() {
    mut rows [[string]] = csv.decode("a,b,c\n1,2,3")
    println(rows)
}' > /tmp/fix4.ez && ./ezc/ezc /tmp/fix4.ez -o /tmp/fix4 && /tmp/fix4
```

---

## FIX 5: json.pretty/pretty_print name mismatch [HIGH]

**Symptom:** `json.pretty(m, 2)` passes typechecker but codegen emits `ez_json_pretty()` via generic fallback → C linker error (undefined function). `json.pretty_print()` fails at typechecker with E4005.

**Root cause:** Typechecker registers `"pretty"` (line 1630). Codegen handler checks for `"pretty_print"` (line 3286). The generic fallback emits `ez_json_pretty()` which doesn't exist — the C function is `ez_json_pretty_map()`.

**File 1:** `ezc/src/typechecker/typechecker.c`, line 1629-1630

**Current code:**
```c
        } else if (strcmp(mfn, "encode") == 0 || strcmp(mfn, "stringify") == 0 ||
                   strcmp(mfn, "format") == 0 || strcmp(mfn, "pretty") == 0) {
```

**Fix:** Change `"pretty"` to `"pretty_print"` to match the STANDARD and codegen:
```c
        } else if (strcmp(mfn, "encode") == 0 || strcmp(mfn, "stringify") == 0 ||
                   strcmp(mfn, "format") == 0 || strcmp(mfn, "pretty_print") == 0) {
```

**Also update** the `func_to_mod` dispatch table (search for `{"pretty","json"}` and change to `{"pretty_print","json"}`).

**Verify:**
```bash
echo 'import @json
do main() {
    mut m map[string:int] = {"a": 1, "b": 2}
    mut p string = json.pretty_print(m, 2)
    println(p)
}' > /tmp/fix5.ez && ./ezc/ezc /tmp/fix5.ez -o /tmp/fix5 && /tmp/fix5
```

---

## FIX 6: Register net.set_timeout in typechecker [HIGH]

**Symptom:** `net.set_timeout(sock, 5000)` → E4005: module 'net' has no function named 'set_timeout'

**Root cause:** Codegen handles it (line 3087) but typechecker doesn't register it.

**File:** `ezc/src/typechecker/typechecker.c`, lines 1943-1957

**Current code:**
```c
            } else if (strcmp(mod, "net") == 0) {
                if (strcmp(mfn, "listen") == 0) {
                    result = type_struct("Listener");
                } else if (strcmp(mfn, "connect") == 0 || strcmp(mfn, "accept") == 0) {
                    result = type_struct("Socket");
                } else if (strcmp(mfn, "send") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "receive") == 0 || strcmp(mfn, "resolve") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "close") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
```

**Fix:** Add `set_timeout` to the void-returning branch:
```c
                } else if (strcmp(mfn, "close") == 0 || strcmp(mfn, "set_timeout") == 0) {
                    result = &TYPE_VOID;
```

**Verify:**
```bash
echo 'import @net
do main() {
    mut sock = net.connect("example.com", 80)
    net.set_timeout(sock, 5000)
    net.close(sock)
}' > /tmp/fix6.ez && ./ezc/ezc /tmp/fix6.ez -o /tmp/fix6
# Should compile without errors
```

---

## FIX 7: strings.pad_left/pad_right/center codegen emits wrong C function [HIGH]

**Symptom:** `strings.pad_left("hi", 5, ch)` → C linker error: `ez_strings_pad_left` undefined.

**Root cause:** The typechecker registers `pad_left`/`pad_right`/`center` in the strings module (line 1534), but the C functions are `ez_fmt_pad_left`, `ez_fmt_pad_right`, `ez_fmt_center` — they live in the fmt module. The strings codegen handler (`emit_strings_call`) falls through to the generic `emitf(cg, "ez_strings_%s(", func)` which doesn't exist.

**Fix (option A — remove from strings typechecker):** Remove `pad_left`, `pad_right`, `center` from the strings module registration at line 1534. Users should call `fmt.pad_left()` instead.

**Fix (option B — add C aliases):** Add `ez_strings_pad_left` etc. as wrappers in `ez_strings.h`/`ez_strings.c` that call the `ez_fmt_*` versions. BUT this still has the char `.data[0]` bug from FIX 3.

**Recommendation:** Remove from strings module (option A). The STANDARD lists them under fmt only.

**File:** `ezc/src/typechecker/typechecker.c`, line 1534

**Current code:**
```c
                           strcmp(mfn, "pad_left") == 0 || strcmp(mfn, "pad_right") == 0 ||
                           strcmp(mfn, "center") == 0) {
```

**Fix:** Remove those three strcmp checks from the strings block.

---

## FIX 8: sqlite — 6 missing functions [MEDIUM]

**Symptom:** `sqlite.prepare()`, `sqlite.begin()`, etc. → E4005

**Root cause:** Not implemented anywhere — not in typechecker, codegen, or C runtime.

**Missing functions per STANDARD:**
| Function | Signature |
|----------|-----------|
| `prepare` | `(db Database, sql string) -> Statement` |
| `step` | `(stmt Statement, ...params string) -> [map[string:string]]` |
| `finalize` | `(stmt Statement)` |
| `begin` | `(db Database)` |
| `commit` | `(db Database)` |
| `rollback` | `(db Database)` |

**Files to touch (4 files per the stdlib wiring pattern):**
1. `ezc/src/stdlib/ez_sqlite.h` — Declare the 6 new C functions
2. `ezc/src/stdlib/ez_sqlite.c` — Implement them using sqlite3 API
3. `ezc/src/typechecker/typechecker.c` — Register return types in the `strcmp(mod, "sqlite")` block (around line 1636)
4. `ezc/src/codegen/codegen.c` — Add emit handlers in `emit_sqlite_call()` (around line 3299)

**Current sqlite typechecker block (lines 1636-1645):**
```c
            } else if (strcmp(mod, "sqlite") == 0) {
                if (strcmp(mfn, "open") == 0) result = &TYPE_UNKNOWN;
                else if (strcmp(mfn, "exec") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "query") == 0) result = type_array("map[string:string]");
                else if (strcmp(mfn, "close") == 0) result = &TYPE_VOID;
                ...
```

**Current sqlite codegen (`emit_sqlite_call`, lines 3299-3332):**
Handles `open`, `close`, `exec`, `query` only.

---

## FIX 9: binary module — 20 missing C functions [MEDIUM]

**Symptom:** `binary.encode_i128_le(val)` compiles EZ successfully but fails at C link time with "undefined symbol".

**Root cause:** The typechecker and codegen are generic (accept any `encode_*`/`decode_*` name), but `ez_binary.h`/`ez_binary.c` only implement 8/16/32/64-bit functions. The 128-bit, 256-bit, and some big-endian float variants don't exist in C.

**Missing C functions (documented in STANDARD):**
- 128-bit: `ez_binary_encode_i128_le`, `_be`, `ez_binary_decode_i128_le`, `_be`, `_u128` variants (8 total)
- 256-bit: Same pattern (8 total)
- Big-endian floats: `ez_binary_encode_f32_be`, `_f64_be`, `decode_f32_be`, `decode_f64_be` (4 total)

**Files to touch:**
1. `ezc/src/stdlib/ez_binary.h` — Declare the 20 new functions
2. `ezc/src/stdlib/ez_binary.c` — Implement them

No typechecker or codegen changes needed (generic dispatch already handles them).

---

## FIX 10: Spec vs implementation mismatches [LOW — DECISION REQUIRED]

These need a product decision — either fix the implementation or update the STANDARD.

| Issue | STANDARD Says | Implementation Does | Recommendation |
|-------|---------------|---------------------|----------------|
| Error tuples | Many functions return `(T, Error)` | Return just `T` | Update STANDARD to match reality (widespread pattern across io, encoding, csv, json) |
| `strings.is_empty` | "Check if empty (after trim)" | Only checks `len == 0` | Fix implementation to trim first, or update STANDARD |
| `csv.encode` | Takes `[[string]]` (2D), returns `(string, Error)` | Takes `[string]` (1D), returns `string` | Update STANDARD |
| `csv.decode` | Returns `([[string]], Error)` | Returns `[[string]]` | Update STANDARD |

---

## Appendix: Undocumented functions that exist in the compiler

These work but are not in STANDARD.md. Decide whether to document or remove.

| Module | Function | What It Does |
|--------|----------|--------------|
| arrays | `insert` | Alias for `insert_at` |
| arrays | `remove` | Alias for `remove_at` |
| arrays | `set` | Set element at index |
| json | `stringify` | Alias for `encode` |
| json | `format` | Alias for `encode` |
| csv | `headers` | Returns first row of CSV |
| os | `exec` | Execute shell command |
| random | `seed` | Seed the RNG |
| uuid | `v4`, `v7`, `to_string` | Extra UUID variants |
| server | `cors`, `use`, `add_router` | Middleware functions |
| mem | `make`, `free`, `raw_copy`, `zero`, `fill` | Additional memory ops |
| sync | `try_lock` | Non-blocking lock attempt |
