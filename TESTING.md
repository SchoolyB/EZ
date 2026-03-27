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
- Error detection:
  - E3001: type mismatch on variable declarations
  - E5008: wrong argument count on function calls
  - E3016: dereference of non-pointer type
- String enum member type resolution
- Map type from name parsing

**End-to-End Tests** (`ezc/tests/test_codegen.c` — 65 tests):

E2E tests compile EZ programs, run them, and verify output:

- Basic: hello world, variables, arithmetic, strings, booleans
- Control flow: if/else, for, while, loop/break, when/is
- Functions: calls, recursion, multi-return, named returns, default params, mutable params
- Mutable params: simple variables, array elements (`arr[i]`), struct fields (`s.x`)
- Data structures: arrays, fixed-size arrays, nested arrays, maps, structs, enums
- String features: interpolation, char literals, hex/octal/binary literals
- Function references: `()func`, `ref(func)`, reassignment
- Struct-namespaced functions: `Type.func()` calls
- `or_return` error propagation
- Enum attributes: `#flags` (powers of 2), string enums
- Map operations: literal creation, key access, `for_each` iteration
- `@mem` module: arena create/destroy, alloc, usage, reset
- Pointers: `addr()`, dereference (`p^`), nil check, write-through, struct pointers
- `@threads` module: spawn/join, channels, sleep
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

### Integration Pass Tests (42 tests)

Integration tests compile and run `.ez` programs end-to-end through the full compiler pipeline.

**Structure:**

- `integration-tests/pass/core/` — Core language features (arrays, control flow, structs, enums, maps, typeof, etc.)
- `integration-tests/pass/named_returns/` — Named return value tests
- `integration-tests/pass/stdlib/` — Stdlib module tests (http, net, regex)

**Running:**

```bash
bash ezc/tests/run_integration.sh

# With verbose output on failures
bash ezc/tests/run_integration.sh --verbose
```

### Error Detection Tests (291 tests — 100% coverage)

Every error test is an `.ez` file that **must** be rejected by the compiler or runtime. Tests are named by error code (e.g., `E3001_type_mismatch.ez`).

**Structure:**

- `integration-tests/fail/errors/` — 305 total tests (14 skipped as interpreter-only)
- Each test has a comment header with the expected error code and message pattern

**What's tested:**

- 269 compile-time errors (typechecker, parser, lexer)
- 22 runtime panics (division by zero, nil deref, overflow, bounds, stack overflow)
- Covers all 92 error and warning codes in `error_codes.h`

**Running:**

```bash
bash ezc/tests/run_integration.sh   # includes error tests
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

### Parity Tests

Compares EZC compiler output against expected results for basic examples.

```bash
bash ezc/tests/run_parity.sh
```

---

## Go Tooling Tests

The Go CLI (`ez`) has unit tests for the packages it still uses:

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
bash ezc/tests/run_integration.sh
go test ./pkg/errors/... ./pkg/lineeditor/...
```

---

## CI

All tests run automatically on push to `main` and `v3.0.0` via GitHub Actions:

| Platform | Compiler | Sanitizers | Go Tooling |
|----------|:--------:|:----------:|:----------:|
| Ubuntu   | unit + e2e + integration | UBSan + ASan | errors + lineeditor |
| macOS    | unit + e2e + integration | UBSan | errors + lineeditor |

CI workflow: `.github/workflows/ci.yml`
