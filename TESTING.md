# EZ Language Testing Guide

## Unit Tests

Unit tests are written in Go and test individual compiler/runtime components (lexer, parser, interpreter, typechecker, stdlib). Each package in `pkg/` has corresponding `*_test.go` files.

```bash
# Run all unit tests
go test ./...

# Run with coverage summary
go test ./pkg/... -cover

# Run tests for a specific package
go test ./pkg/interpreter/...
go test ./pkg/typechecker/...
go test ./pkg/stdlib/...
```

## Integration Tests

Integration tests are `.ez` files that test the language end-to-end. Error tests verify the compiler catches errors correctly. Pass tests verify language features work correctly.

```bash
# Run error tests (files that should fail)
./tests/errors/run_error_tests.sh ./ez

# Run comprehensive language test (should pass)
./ez run tests/comprehensive.ez
```

## Test Coverage

```bash
# Generate coverage file
go test ./pkg/... -coverprofile=coverage.out

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```
