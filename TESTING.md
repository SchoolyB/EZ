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

### Unit Tests (313 tests)

Unit tests validate individual compiler components:

- **Lexer Tests** (`ezc/tests/test_lexer.c` — 69 tests): Token scanning, keyword recognition, literal formats, comment handling, attribute tokens, v3 keyword aliases, operator disambiguation, edge cases (empty strings, zero literals, multiline comments, adjacent operators).
- **Parser Tests** (`ezc/tests/test_parser.c` — 86 tests): Declarations, imports, control flow, structs, enums, function references, attributes, map/array types, visibility, error reporting, grouped params, compound assignments, nested structures, import aliases, precedence, parser error codes (E2025 array size, E2059 empty when, E2060 too many returns, E2068 mut struct, E2069 semicolons, E2070 wildcard in var, E2071 empty interpolation).
- **Typechecker Tests** (`ezc/tests/test_typechecker.c` — 158 tests): Scope management, type resolution, expression inference, built-in return types, error detection, enum/map type resolution, deep scope nesting, bigint types, char/uint numerics, valid program verification (nested structs, enums, multi-return, when, struct functions), error code coverage (E2014 duplicate enum variant, E2015 duplicate struct field init, E2063 duplicate named return, E2064 field-func conflict, E2065-E2067 naming constraints, E3047 enum no member, E3048 string plus, E3057 invalid map key, E3059 const map, E3061 recursive struct, E3063 return addr of local, E3072 return nil, E3073 return in main, E3074 array compare, E3076 map compare, E3078 pointer arithmetic, E3080 named return mismatch, E3082 wildcard named return, E4008 main params, E5025 invalid assign target).

### End-to-End Tests (101 tests)

E2E tests (`ezc/tests/test_codegen.c`) compile EZ programs, run them, and verify output. Covers variables, control flow, functions, data structures, string features, function references, struct functions, enums, maps, pointers, runtime checks, boolean logic, string comparison, negative arithmetic, variable shadowing, nested control flow, map mutation, for_each with index, const values, empty containers, nested function calls, string indexing, enum comparison, grouped parameters, scope lifetime, #strict when, C interop, and atomic operations.

**Running:**

```bash
# From repo root
make test-unit        # unit tests (lexer + parser + typechecker)
make test-e2e         # e2e codegen tests

# Individual test suites (from ezc/)
./ezc/tests/test_lexer
./ezc/tests/test_parser
./ezc/tests/test_typechecker
./ezc/tests/test_codegen
```

### Integration Tests

Integration tests compile and run `.ez` programs end-to-end through the full compiler pipeline.

**Structure:**

- `integration-tests/pass/core/` — 169 core language feature tests (arrays, control flow, structs, enums, maps, typeof, named returns, import variants, C interop, #strict when, or_return, raw strings, bigint arrays, pointer collections, non-standard map key ops, wide numeric map keys, [func] arrays, func var calls, enum map keys, map deep copy, wildcard types, [func] arrays of struct-namespaced refs, nested array deep copy, struct deep copy, nested map types, container compound assign, grouped struct fields, struct func mutable params, func ref mutable params, for_each pointer arrays, pointer-to-pointer types, struct self dispatch, struct func fields, wildcard nested calls, wildcard named returns, wildcard multi-return, wildcard multi-return destructuring, func param calls, func ref default params, enum implicit values, int-to-float coercion, uint precision at UINT64_MAX, default params referencing earlier params, when/is branch scoping, c_string NULL safety, >32 ensures per function, float map key normalization with +0.0/-0.0 collision and NaN dedup, etc.)
- `integration-tests/pass/stdlib/` — 51 stdlib module tests (includes random.shuffle/sample on >64-byte struct elements and url_encode on UTF-8 high-bit bytes)
- `integration-tests/pass/warnings/` — 24 warning detection tests (includes W2012 float-when)
- `integration-tests/pass/multi-file/` — 38 multi-file import tests (basic, alias, structs, enums, constants, private visibility, transitive, circular, collision, nested, struct functions, imported struct with enum field, import-use-types, extensionless imports, directory imports, directory alias, mixed import styles, directory with stdlib, transitive directory)
- `integration-tests/pass/stress/` — 42 stress tests (29 core + 13 stdlib)
- `integration-tests/fail/errors/` — 596 error detection tests (includes negation-overflow and TYPE_MIN/-1 div/mod panics for int/i64/i32/i16/i8, E3046 hex-top-bit-to-int and above-UINT64_MAX, E3068 void in typed-func return, compound-assign overflow panics for int/uint across +=/-=/*=//=/%=, E3069 inverted &-on-param, E2001 missing param comma, E3070 nested ensure in if/for/when, E3071 return nil from wildcard-return, E3082 wildcard named return, E4001 named return hint, E6002 extensionless not found, E6003 empty directory import, E9006 array mutation during for_each across append/clear/insert_at/remove_at/remove_last/sort_asc/sort_desc and direct index assignment, E16001 base64 wrong length and misplaced padding, E16002 hex odd length)
- `integration-tests/fail/multi-file/` — 20 multi-file error detection tests (collision, private access, type mismatch, undefined module, cross-file error attribution)

**Running:**

```bash
make test-integration

# With verbose output on failures
bash scripts/run_tests.sh --verbose
```

### Sanitizer Tests

ASan (AddressSanitizer) and UBSan (Undefined Behavior Sanitizer) builds catch memory bugs and undefined behavior.

```bash
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
make test-go
```

---

## Running Everything

```bash
make test
```

This builds the compiler, then runs unit tests, e2e tests, integration tests, and Go tests in sequence.

---

## CI

All tests run automatically on push to `main` and `v3.0.0` via GitHub Actions:

| Platform | Compiler | Sanitizers | Go Tooling |
|----------|:--------:|:----------:|:----------:|
| Ubuntu   | unit + e2e + integration | UBSan + ASan | lineeditor |
| macOS    | unit + e2e + integration | UBSan | lineeditor |

CI workflow: `.github/workflows/ci.yml`
