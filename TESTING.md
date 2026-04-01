# EZ Language Testing Guide

## Compiler Tests (EZC)

The EZC compiler has a comprehensive test suite written in C, located in `ezc/tests/`.

### Unit Tests (163 tests)

Unit tests validate individual compiler components:

**Lexer Tests** (`ezc/tests/test_lexer.c` — 30 tests):
- Token scanning (single/multi-char, strings, chars, numbers)
- Keyword recognition (`mut`, `while`, `or_return`, `as_long_as`)
- Literal formats (hex, octal, binary, float with underscores)
- Comment handling (single-line, multi-line)
- Attribute tokens (`#flags`, `#strict`, `#suppress`, `#doc`)
- v3 keyword aliases and coexistence
- `!in` vs `!` disambiguation

**Parser Tests** (`ezc/tests/test_parser.c` — 33 tests):
- Variable/constant declarations
- Function declarations (params, return types, defaults)
- Import statements (single, multi, aliases)
- Control flow (if/or/otherwise, for, while, loop, when)
- Struct and enum declarations
- Struct-namespaced functions (`do` inside struct blocks)
- Function references (`()func_name`)
- `or_return` parsing
- `#flags` and string enum attributes
- Map types, fixed-size arrays, nested arrays
- Private visibility on struct functions
- `for_each` with index variable
- Error reporting on invalid syntax

**Typechecker Tests** (`ezc/tests/test_typechecker.c` — 35 tests):
- Scope management (define, lookup, nested, shadow)
- Type resolution (primitives including signed/unsigned, arrays, structs, maps, pointers)
- Expression type inference (literals, arithmetic, comparison, logical)
- Built-in function return types (`len`, `typeof`, `to_float`, `addr`)
- Error detection (E3001 type mismatch, E5008 wrong arg count, E3016 deref non-pointer)
- String enum member type resolution
- Map type from name parsing

**End-to-End Tests** (`ezc/tests/test_codegen.c` — 65 tests):

E2E tests compile EZ programs, run them, and verify output:

- Basic: hello world, variables, arithmetic, strings, booleans
- Control flow: if/else, for, while, loop/break, when/is
- Functions: calls, recursion, multi-return, named returns, default params, mutable params
- Data structures: arrays, fixed-size arrays, nested arrays, maps, structs, enums
- String features: interpolation, char literals, hex/octal/binary literals
- Function references: `()func`, `ref(func)`, reassignment
- Struct-namespaced functions: `Type.func()` calls
- `or_return` error propagation
- Enum attributes: `#flags` (powers of 2), string enums
- Map operations: literal creation, key access, `for_each` iteration
- Pointers: `addr()`, dereference (`p^`), nil check, write-through, struct pointers
- Runtime checks: division by zero panic

**Running:**

```bash
cd ezc

# Run all unit tests (lexer + parser + typechecker)
make test-unit

# Run all e2e tests
make test-e2e

# Individual test suites
./tests/test_lexer
./tests/test_parser
./tests/test_typechecker
./tests/test_codegen
```

### Integration Tests (65 pass + 24 stdlib + 291 error = 380 tests)

Integration tests compile and run `.ez` programs end-to-end through the full compiler pipeline.

**Structure:**

- `integration-tests/pass/core/` — Core language features (arrays, control flow, structs, enums, maps, typeof, named returns, etc.)
- `integration-tests/pass/stdlib/` — All 24 stdlib module tests (`*_c.ez` files)
- `integration-tests/fail/errors/` — 305 error detection tests (14 skipped as interpreter-only)

**What's tested:**

- 65 passing programs covering all core language features
- 24 stdlib modules: std, strings, math, arrays, maps, io, os, time, encoding, crypto, json, csv, uuid, bytes, binary, mem, fmt, random, regex, http, net, server, sqlite, threads
- 269 compile-time errors (typechecker, parser, lexer)
- 22 runtime panics (division by zero, nil deref, overflow, bounds, stack overflow)
- Covers all 81 error and warning codes

**Running:**

```bash
bash scripts/run_integration.sh

# With verbose output on failures
bash scripts/run_integration.sh --verbose
```

### Interpreter Integration Tests

Runs all integration tests against the EZ interpreter (the `ez` CLI):

```bash
bash scripts/run_tests.sh
```

### Parity Tests

Compares EZC compiled output against the EZ interpreter to verify identical behavior:

```bash
bash scripts/run_parity.sh

# Run specific files only
bash scripts/run_parity.sh examples/basic/hello.ez
```

### Sanitizer Tests

ASan (AddressSanitizer) and UBSan (Undefined Behavior Sanitizer) builds catch memory bugs and undefined behavior.

```bash
cd ezc

# UBSan — runs on all platforms
make test-ubsan

# ASan + UBSan — Linux recommended
make test-asan
```

### Fuzz Testing

A Python fuzzer generates random EZ programs and feeds them to the compiler, looking for crashes, segfaults, and hangs. The compiler should never crash on any input — it should always produce a clean error or compile successfully.

**Generators (4 categories, 23 generators):**

| Category | What it generates |
|----------|-------------------|
| `valid` | Correct programs — random structs, enums, control flow, arrays, maps, function refs, string interpolation |
| `edge` | Stress tests — empty files, very long strings, deep nesting, many parameters, many variables |
| `garbage` | Random bytes, random token sequences, broken strings, type mismatches, malformed imports |
| `mutation` | Takes a valid program and randomly corrupts it — character deletions, insertions, swaps, truncation |

**Running:**

```bash
# Basic run (500 iterations, random seed)
python3 scripts/fuzz.py

# More iterations
python3 scripts/fuzz.py -n 2000

# Reproducible run (same seed = same programs every time)
python3 scripts/fuzz.py --seed 42

# Only test a specific category
python3 scripts/fuzz.py --mode garbage

# Save every generated program to fuzz_output/ for inspection
python3 scripts/fuzz.py --save-all

# Point to a specific ezc binary
python3 scripts/fuzz.py --ezc ./ezc/ezc

# Clean up all crash files and saved output
python3 scripts/fuzz.py --clean
```

**What it checks:** Any crash (segfault, abort, signal), hang (exceeds 10s timeout), or internal compiler error is a failure. Clean error messages on invalid input are expected and count as passes.

**When a crash is found:** The offending program is saved to `fuzz_crash_NNN_generator.ez` in the working directory. Rerun with the same `--seed` to reproduce.

---

## Go Tooling Tests

The Go CLI (`ez`) has unit tests for the packages it uses:

- `pkg/errors` — Error handling and formatting
- `pkg/lineeditor` — REPL line editing

```bash
go test ./pkg/errors/... ./pkg/lineeditor/...
```

---

## Running Everything

```bash
# Full test suite
cd ezc && make test-unit && make test-e2e && cd ..
bash scripts/run_integration.sh
bash scripts/run_tests.sh
bash scripts/run_parity.sh
go test ./pkg/errors/... ./pkg/lineeditor/...
```

---

## CI

All tests run automatically on push to `main` and `v3.0.0` via GitHub Actions:

| Platform | Compiler | Sanitizers | Go Tooling |
|----------|:--------:|:----------:|:----------:|
| Ubuntu   | unit + e2e + integration | UBSan + ASan | errors + lineeditor |
| macOS    | unit + e2e + integration | UBSan | errors + lineeditor |
| Windows  | — | — | errors |

CI workflow: `.github/workflows/ci.yml`
