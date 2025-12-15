# EZ Language Testing Guide

## Unit Tests

Unit tests are written in Go and test individual compiler/runtime components. Each package in `pkg/` has corresponding `*_test.go` files.

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

- `integration-tests/pass/` - Tests that should execute successfully
  - `core/` - Core language feature tests
  - `stdlib/` - Standard library tests
  - `multi-file/` - Multi-file project tests
- `integration-tests/fail/` - Tests that should fail with expected errors
  - `errors/` - Error detection tests
  - `multi-file/` - Multi-file error tests

### Running Integration Tests

```bash
# Run all integration tests
./integration-tests/run_tests.sh
```

The test runner will:
- Execute all pass tests and verify they succeed
- Execute all fail tests and verify they produce errors
- Report a summary of passed/failed tests

## Test Coverage

```bash
# Generate coverage file
go test ./pkg/... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```
