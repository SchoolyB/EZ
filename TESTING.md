# EZ Language Testing Guide

## Unit Tests

Unit tests are written in Go and test individual interpreter/runtime components. Each package in `pkg/` has corresponding `*_test.go` files.

### Packages with Unit Tests

- `pkg/ast` - Abstract syntax tree nodes
- `pkg/errors` - Error handling and formatting
- `pkg/interpreter` - Runtime evaluation
- `pkg/lexer` - Tokenization
- `pkg/lineeditor` - REPL line editing
- `pkg/object` - Runtime object system
- `pkg/parser` - Syntax parsing
- `pkg/stdlib` - Standard library functions
- `pkg/tokenizer` - Token definitions
- `pkg/typechecker` - Static type checking

### Running Unit Tests

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage summary
go test ./pkg/... -cover

# Run tests for a specific package
go test ./pkg/lexer/...
go test ./pkg/parser/...
go test ./pkg/ast/...
go test ./pkg/object/...
go test ./pkg/errors/...
go test ./pkg/interpreter/...
go test ./pkg/typechecker/...
go test ./pkg/stdlib/...
go test ./pkg/lineeditor/...
go test ./pkg/tokenizer/...
```

## Integration Tests

Integration tests are `.ez` files that test the language end-to-end. They are located in the `integration-tests/` directory and organized into:

- `integration-tests/pass/` - Tests that should execute successfully (exit 0)
  - `core/` - Core language feature tests (control flow, operators, structs, enums, string interpolation, raw strings, etc.)
  - `stdlib/` - Standard library module tests (io, strings, arrays, maps, math, crypto, binary, etc.)
  - `multi-file/` - Multi-file project tests
  - `named_returns/` - Named return value tests
  - `warnings/` - Warning detection tests (expected to trigger specific warnings via `ez check`)
- `integration-tests/fail/` - Tests that should fail with expected errors (exit non-zero)
  - `errors/` - Error detection tests (named by error code, e.g. `E3001_type_mismatch.ez`)
  - `multi-file/` - Multi-file error tests

### Running Integration Tests

```bash
# Run all integration tests (recommended)
make integration-test

# Or run the script directly
./integration-tests/run_tests.sh
```

The test runner will:
- Execute all `pass/core/` and `pass/stdlib/` tests and verify they succeed
- Execute all `pass/warnings/` tests with `ez check` and verify the expected warning code appears in output
- Execute all `fail/` tests and verify they produce errors
- Report a summary of passed/failed tests

### Warning Tests

Warning tests live in `integration-tests/pass/warnings/` and are named with their expected warning code (e.g. `W2001_unreachable_code.ez`). The test runner extracts the warning code from the filename and runs `ez check` on the file, verifying that the expected `warning[WXXXX]` appears in the output. Multi-file warning tests use a directory (e.g. `W4001_module_name_mismatch/`) with a `main.ez` entry point.

## Test Coverage

```bash
# Generate coverage file
go test ./pkg/... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```
