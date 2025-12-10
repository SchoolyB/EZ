<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

<h3 align="center">Programming Made EZ</h3>

<p align="center">
  A simple, statically-typed programming language designed for clarity and ease of use.
</p>

<p align="center">
  <a href="https://schoolyb.github.io/language.ez">Website</a> •
  <a href="https://schoolyb.github.io/language.ez/docs">Documentation</a>
</p>

---

## Quick Start (For Developers)

Want to contribute or build from source?

```bash
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Build the binary
make build

# Run a program
./ez examples/hello.ez
```

**Requirements:** Go 1.23.1 or higher

For pre-built binaries and installation instructions, visit the [documentation](https://schoolyb.github.io/language.ez/docs).

---

## Updating

EZ includes a built-in update command:

```bash
ez update
```

This will check for new versions, show the changelog, and prompt you to upgrade. If EZ is installed in a system directory (like `/usr/local/bin`), it will automatically prompt for your password.

### Upgrading from v0.17.1 or earlier

If you're upgrading from **v0.17.1 or earlier**, you'll need to manually update once:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/SchoolyB/EZ/releases/latest/download/ez-darwin-arm64.tar.gz | tar xz
sudo mv ez /usr/local/bin/ez

# macOS (Intel)
curl -L https://github.com/SchoolyB/EZ/releases/latest/download/ez-darwin-amd64.tar.gz | tar xz
sudo mv ez /usr/local/bin/ez

# Linux (x86_64)
curl -L https://github.com/SchoolyB/EZ/releases/latest/download/ez-linux-amd64.tar.gz | tar xz
sudo mv ez /usr/local/bin/ez

# Linux (ARM64)
curl -L https://github.com/SchoolyB/EZ/releases/latest/download/ez-linux-arm64.tar.gz | tar xz
sudo mv ez /usr/local/bin/ez
```

After this one-time manual update, `ez update` will work automatically for all future versions.

---

## Running Tests

EZ has two types of tests: **EZ language tests** (written in EZ) and **Go unit tests** (for the interpreter internals).

### EZ Language Tests

```bash
# Run passing tests
./ez tests/comprehensive.ez
./ez tests/multi-file/main.ez
./ez tests/nested-test/main/main.ez

# Run error tests
./tests/errors/run_error_tests.sh ./ez
```

### Go Unit Tests

```bash
# Run all unit tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/lexer/...
go test ./pkg/parser/...
go test ./pkg/ast/...
go test ./pkg/object/...
go test ./pkg/errors/...

# Run with coverage report
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Unit Test Coverage

| Package | Status |
|---------|--------|
| pkg/tokenizer | Implemented |
| pkg/errors | Implemented |
| pkg/lexer | Implemented |
| pkg/ast | Implemented |
| pkg/object | Implemented |
| pkg/parser | Implemented |
| pkg/typechecker | Implemented |
| pkg/interpreter | Implemented |
| pkg/stdlib | Implemented |

---

## License

MIT License - Copyright (c) 2025-Present Marshall A Burns

See [LICENSE](LICENSE) for details.
