# EZ Language Testing Guide

## Quick Start

If you've already run `make build && make install`, you can run the entire test suite with a single command:

```bash
ez test
```

This runs all Go tooling tests, compiler unit tests, compiler e2e tests, and integration tests in sequence.

---

## Compiler Tests

The EZ compiler has a comprehensive test suite written in C, located in `ezc/tests/`.

### Unit Tests (284 tests)

Unit tests validate individual compiler components:

- **Lexer Tests** (`ezc/tests/test_lexer.c` — 69 tests): Token scanning, keyword recognition, literal formats, comment handling, attribute tokens, v3 keyword aliases, operator disambiguation, edge cases (empty strings, zero literals, multiline comments, adjacent operators).
- **Parser Tests** (`ezc/tests/test_parser.c` — 79 tests): Declarations, imports, control flow, structs, enums, function references, attributes, map/array types, visibility, error reporting, grouped params, compound assignments, nested structures, import aliases, precedence.
- **Typechecker Tests** (`ezc/tests/test_typechecker.c` — 136 tests): Scope management, type resolution, expression inference, built-in return types, error detection, enum/map type resolution, deep scope nesting, bigint types, char/uint numerics, valid program verification (nested structs, enums, multi-return, when, struct functions).

### End-to-End Tests (101 tests)

E2E tests (`ezc/tests/test_codegen.c`) compile EZ programs, run them, and verify output. Covers variables, control flow, functions, data structures, string features, function references, struct functions, enums, maps, pointers, runtime checks, boolean logic, string comparison, negative arithmetic, variable shadowing, nested control flow, map mutation, for_each with index, const values, empty containers, nested function calls, string indexing, enum comparison, grouped parameters, scope lifetime, #strict when, C interop, and atomic operations.

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

### Integration Tests

Integration tests compile and run `.ez` programs end-to-end through the full compiler pipeline.

**Structure:**

- `integration-tests/pass/core/` — 146 core language feature tests (arrays, control flow, structs, enums, maps, typeof, named returns, import variants, C interop, #strict when, or_return, raw strings, bigint arrays, pointer collections, non-standard map key ops, wide numeric map keys, [func] arrays, func var calls, enum map keys, map deep copy, wildcard types, [func] arrays of struct-namespaced refs, nested array deep copy, struct deep copy, nested map types, container compound assign, grouped struct fields, struct func mutable params, func ref mutable params, for_each pointer arrays, pointer-to-pointer types, struct self dispatch, struct func fields, wildcard nested calls, wildcard named returns, wildcard multi-return, func param calls, func ref default params, enum implicit values, int-to-float coercion, etc.)
- `integration-tests/pass/stdlib/` — 29 stdlib module tests
- `integration-tests/pass/warnings/` — 23 warning detection tests
- `integration-tests/pass/multi-file/` — 32 multi-file import tests (basic, alias, structs, enums, constants, private visibility, transitive, circular, collision, nested, struct functions, imported struct with enum field, import-use-types)
- `integration-tests/pass/stress/` — 42 stress tests (29 core + 13 stdlib)
- `integration-tests/fail/errors/` — 514 error detection tests
- `integration-tests/fail/multi-file/` — 20 multi-file error detection tests (collision, private access, type mismatch, undefined module, cross-file error attribution)

**Running:**

```bash
bash scripts/run_tests.sh

# With verbose output on failures
bash scripts/run_tests.sh --verbose
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

---

## Go Tooling Tests (271 tests total)

The Go CLI (`ez`) has unit tests for the packages it uses:

- `pkg/lineeditor` (177 tests) — REPL line editing, keystroke parsing, escape sequences, cursor movement, history navigation, UTF-8 handling
- `cmd/ez` (84 tests) — updater semver parsing/comparison (parseSemver, compareSemver, isNewerVersion, pickLatestPrerelease), exact-version install validation (exactSemverRE, normalizeTag)
- `internal/ezc` (10 tests) — compiler binary lookup (statFile, Find EZ_COMPILER_PATH override behavior)

```bash
go test ./pkg/lineeditor/...
go test ./cmd/ez/...
go test ./internal/ezc/...
```

---

## Running Everything

The simplest way (requires `make build && make install`):

```bash
ez test
```

Or manually:

```bash
cd ezc && make test-unit && make test-e2e && cd ..
bash scripts/run_tests.sh
go test ./pkg/lineeditor/...
go test ./cmd/ez/...
go test ./internal/ezc/...
```

---

## CI

All tests run automatically on push to `main` and `v3.0.0` via GitHub Actions:

| Platform | Compiler | Sanitizers | Go Tooling |
|----------|:--------:|:----------:|:----------:|
| Ubuntu   | unit + e2e + integration | UBSan + ASan | lineeditor |
| macOS    | unit + e2e + integration | UBSan | lineeditor |
| Windows  | — | — | — |

CI workflow: `.github/workflows/ci.yml`
