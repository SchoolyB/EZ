# EZ Language Testing Guide

## Quick Start

To run the full test suite, use:

```bash
make test
```

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

- `integration-tests/pass/core/` — 236 core language feature tests (arrays, control flow, structs, enums, maps, typeof, named returns, import variants, C interop, #strict when, or_return, raw strings, bigint arrays, pointer collections, non-standard map key ops, wide numeric map keys, [func] arrays, func var calls, enum map keys, map deep copy, wildcard types, [func] arrays of struct-namespaced refs, nested array deep copy, struct deep copy, nested map types, container compound assign, grouped struct fields, struct func mutable params, func ref mutable params, for_each pointer arrays, pointer-to-pointer types, struct self dispatch, struct func fields, wildcard nested calls, wildcard named returns, wildcard multi-return, wildcard multi-return destructuring, func param calls, func ref default params, enum implicit values, int-to-float coercion, uint precision at UINT64_MAX, default params referencing earlier params, when/is branch scoping, c_string NULL safety, >32 ensures per function, float map key normalization with +0.0/-0.0 collision and NaN dedup, here() builtin returning SourceLocation, wildcard unifier recursion into nested composite types, pointer parameter auto-deref field assignment, recursive struct pointer fields (self, binary tree, mutual, forward-ref), for while-style loop, when enum pointer deref, embed() builtin, not_bool_valid, chained_field_access, array_type_safety, map_type_safety, pointer_correct_struct_arg, uint_inc_dec_valid, map_string_key_deep_copy, recursive struct auto-deref pointer field access, struct func multi-return typedef, eprintln all primitive types, eprint all primitive types, wildcard print dispatch all variants, struct wildcard function monomorphization per call site, circular value-type struct references with multi-return, circular value-type struct references with println, pointer deref type inference, pointer array field indexing, struct array field deep copy, addr() on pointer-deref field, for_each over inline array literals, float array integer literal promotion, range equal start/end empty sequence, for loop blank identifier, tagged enum plain enum comparison unaffected by E3124, string(Error) conversion consistency)
- `integration-tests/pass/stdlib/` — 59 stdlib module tests (includes random.shuffle/sample on >64-byte struct elements, url_encode on UTF-8 high-bit bytes, uuid v4/v7/parse/NIL_UUID coverage, threads detach/is_alive/current/yield/sleep/thread_count, strconv whitespace rejection and is_numeric digit requirement, regex find_all on empty input, crypto.random_hex CSPRNG edge cases, io read_bytes/read_lines/file_size_result/glob_result, O_RDONLY/O_WRONLY/O_RDWR constants, directory-path error returns in _result variants, strings char classification is_alpha/is_digit/is_alnum/is_whitespace/is_upper/is_lower)
- `integration-tests/pass/warnings/` — 23 warning detection tests (includes W2012 float-when)
- `integration-tests/pass/multi-file/` — 45 multi-file import tests (basic, alias, structs, enums, constants, private visibility, transitive, circular, collision, nested, struct functions, imported struct with enum field, import-use-types, extensionless imports, directory imports, directory alias, mixed import styles, directory with stdlib, transitive directory)
- `integration-tests/pass/stress/` — 43 stress tests (30 core, 13 stdlib)
- `integration-tests/fail/errors/` — 741 error detection tests (includes negation-overflow and TYPE_MIN/-1 div/mod panics for int/i64/i32/i16/i8, E3046 hex-top-bit-to-int and above-UINT64_MAX, E3068 void in typed-func return, compound-assign overflow panics for int/uint across +=/-=/*=//=/%=, E3069 inverted &-on-param, E2001 missing param comma, E3070 nested ensure in if/for/when, E3071 return nil from wildcard-return, E3082 wildcard named return, E4001 named return hint, E6002 extensionless not found, E6003 empty directory import, E9006 array mutation during for_each across append/clear/insert_at/remove_at/remove_last/sort_asc/sort_desc and direct index assignment, E9006 arrays.contains() with struct element type, E12007 maps.contains_value() with struct value type, E3006 base64 wrong length, E3006 base64 misplaced padding, E3006 hex odd length, E5014 here() with args, E5016 'here' as variable / function / parameter / enum / struct, strconv to_int/to_uint/to_float leading-whitespace panics, encoding url_decode invalid percent-escape panic, crypto.random_hex negative length panic, E3089 can-fail stdlib single-var assignment, E5017 embed() non-literal arg, E5018 embed() file not found, E3090 '!' on non-bool types, E3091 non-scalar as if/for condition, E3092 non-nullable type compared to nil, E3093 arithmetic on map/array/struct, E3094 wrong type assigned to array element by index, E3010 invalid field on chained struct access, E3001 struct returned from primitive function, E3001 array element type mismatch in assignment/arg/return, E3001 map key/value type mismatch in assignment/arg, E3001 addr() of wrong struct type passed to pointer parameter, uint ++ overflow panic, uint -- underflow panic, E3097 addr() of inner-scope struct assigned to outer pointer, E3098 struct mismatch through pointer deref, E3099 reserved stdlib struct name, E3001 struct mismatch in field assignment and nested struct literals, E5028 func reference passed to print functions, E2082 typed func signature as array element type, E3102 func-type return value assigned to variable, E5030 chained call on function return value, E4019 func reference to builtin/stdlib functions across all three surfaces: direct builtin, module.func, and using-scope, E3105 unknown format directives in printf/sprintf, E3106 dangling % in format strings, E3107 too few args for format directives, E3108 too many args for format directives, E3120 pointer ordering comparisons, P0090 range step zero panic, E3121 invalid when condition type for struct/array/map/pointer, E2085 duplicate default branch in when, E5037 copy() on pointer, E3122 addr() on const variable or const struct field, P0091 cast negative float to uint/u64 panic, E3123 for_each both positions discarded, E3124 tagged enum equality, E5038 tagged enum passed to print/println/eprint/eprintln)
- `integration-tests/fail/multi-file/` — 21 multi-file error detection tests (collision, private access, type mismatch, undefined module, cross-file error attribution)

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

## Go Tooling Tests (94 tests total)

The Go CLI (`ez`) has unit tests for the packages it uses:

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
