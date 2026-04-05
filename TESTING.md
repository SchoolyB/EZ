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

### Unit Tests (230 tests)

Unit tests validate individual compiler components:

- **Lexer Tests** (`ezc/tests/test_lexer.c` — 55 tests): Token scanning, keyword recognition, literal formats, comment handling, attribute tokens, v3 keyword aliases, operator disambiguation.
- **Parser Tests** (`ezc/tests/test_parser.c` — 66 tests): Declarations, imports, control flow, structs, enums, function references, attributes, map/array types, visibility, error reporting.
- **Typechecker Tests** (`ezc/tests/test_typechecker.c` — 109 tests): Scope management, type resolution, expression inference, built-in return types, error detection, enum/map type resolution.

### End-to-End Tests (81 tests)

E2E tests (`ezc/tests/test_codegen.c`) compile EZ programs, run them, and verify output. Covers variables, control flow, functions, data structures, string features, function references, struct functions, enums, maps, pointers, and runtime checks.

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

- `integration-tests/pass/core/` — 97 core language feature tests (arrays, control flow, structs, enums, maps, typeof, named returns, etc.)
- `integration-tests/pass/stdlib/` — 42 stdlib module tests
- `integration-tests/fail/errors/` — 409 error detection tests

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

## Go Tooling Tests (362 tests)

The Go CLI (`ez`) has unit tests for the packages it uses:

- `pkg/errors` — Error handling and formatting
- `pkg/lineeditor` — REPL line editing

```bash
go test ./pkg/errors/... ./pkg/lineeditor/...
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
