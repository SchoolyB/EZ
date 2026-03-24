# EZ Language Testing Guide

## Interpreter Tests

### Go Unit Tests

Unit tests are written in Go and test individual interpreter/runtime components. Each package in `pkg/` has corresponding `*_test.go` files.

**Packages with Unit Tests:**

- `pkg/ast` — Abstract syntax tree nodes
- `pkg/errors` — Error handling and formatting
- `pkg/interpreter` — Runtime evaluation (380+ tests including 3.0 features)
- `pkg/lexer` — Tokenization
- `pkg/lineeditor` — REPL line editing
- `pkg/object` — Runtime object system
- `pkg/parser` — Syntax parsing (210+ tests)
- `pkg/stdlib` — Standard library functions
- `pkg/tokenizer` — Token definitions
- `pkg/typechecker` — Static type checking (465+ tests including 3.0 features)

**Running:**

```bash
# Run all unit tests
go test ./pkg/...

# Run with verbose output
go test -v ./pkg/...

# Run tests for a specific package
go test ./pkg/interpreter/...
go test ./pkg/typechecker/...

# Run a specific test
go test ./pkg/interpreter/ -run TestFunctionReferenceBasic
```

### Integration Tests

Integration tests are `.ez` files that test the language end-to-end. Located in `integration-tests/` (404 tests).

**Structure:**

- `integration-tests/pass/` — Tests that should execute successfully
  - `core/` — Core language features (control flow, structs, enums, function references, struct-namespaced functions, map iteration, etc.)
  - `stdlib/` — Standard library modules
  - `multi-file/` — Multi-file project tests (imports, modules, cross-module struct access, struct-namespaced functions)
  - `named_returns/` — Named return value tests
  - `warnings/` — Warning detection tests
- `integration-tests/fail/` — Tests that should fail with expected errors
  - `errors/` — Error detection tests (named by error code)
  - `multi-file/` — Multi-file error tests

**Running:**

```bash
# Run all integration tests
bash integration-tests/run_tests.sh
```

### Test Coverage

```bash
# Generate coverage report
go test ./pkg/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

---

## EZC Compiler Tests

The EZC compiler has its own test suite written in C, located in `ezc/tests/`.

### Unit Tests (98 tests)

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
- Type resolution (primitives, arrays, structs, maps, pointers)
- Expression type inference (literals, arithmetic, comparison, logical)
- Built-in function return types (`len`, `typeof`, `to_float`, `addr`)
- Error detection:
  - E3001: type mismatch on variable declarations
  - E5008: wrong argument count on function calls
  - E3016: dereference of non-pointer type
- String enum member type resolution
- Map type from name parsing

**Running:**

```bash
cd ezc

# Run all unit tests (lexer + parser + typechecker)
make test-unit

# Individual test suites
./tests/test_lexer
./tests/test_parser
./tests/test_typechecker
```

### End-to-End Tests (64 tests)

E2e tests compile EZ programs, run them, and verify output. Located in `ezc/tests/test_codegen.c`.

**Coverage areas:**
- Basic: hello world, variables, arithmetic, strings, booleans
- Control flow: if/else, for, while, loop/break, when/is
- Functions: calls, recursion, multi-return, named returns, default params, mutable params
- Mutable params: simple variables, array elements (`arr[i]`), struct fields (`s.x`)
- Data structures: arrays, fixed-size arrays, nested arrays, maps, structs, enums
- String features: interpolation, char literals, raw strings, hex/octal/binary literals
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

# Run all e2e tests
make test-e2e
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

Compares EZC compiler output against the EZ interpreter to verify identical behavior.

```bash
# Run from project root
bash ezc/tests/run_parity.sh
```

Current parity: 7/15 basic examples pass. Known differences: float rounding (IEEE 754 ULP), array/map printing format, reference semantics.

**Running all compiler tests:**

```bash
cd ezc

# Everything
make test-unit && make test-e2e && make test-ubsan
```

---

## CI

All tests run automatically on push to `main` and `v3.0.0` via GitHub Actions:

| Platform | Interpreter | Compiler | Sanitizers |
|----------|:-----------:|:--------:|:----------:|
| Ubuntu | ✅ unit + integration | ✅ unit + e2e | ✅ UBSan + ASan |
| macOS | ✅ unit + integration | ✅ unit + e2e | ✅ UBSan |
| Windows | ✅ unit (skip lineeditor) | — | — |

CI workflow: `.github/workflows/ci.yml`
