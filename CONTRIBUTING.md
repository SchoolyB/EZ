# Contributing to EZ

Thanks for your interest in contributing to EZ! This guide will help you get started.

## Table of Contents

- [Getting Started](#getting-started)
- [Installing Go](#installing-go)
- [Building from Source](#building-from-source)
- [Making Changes](#making-changes)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Good First Issues](#good-first-issues)

---

## Getting Started

### Prerequisites

- **Go 1.23.1 or higher** ([Installation Guide](#installing-go))
- Git

### Fork and Clone the Repository

1. Fork the repository on GitHub
2. Clone your fork:

```bash
git clone https://github.com/YOUR-USERNAME/EZ.git
cd EZ
```

3. Add the upstream remote:

```bash
git remote add upstream https://github.com/SchoolyB/EZ.git
```

---

## Installing Go

> ⚠️ **Important:** Download the **pre-built binary**, not the source code!

Go to the official download page and select the installer for your platform:

👉 **https://go.dev/dl/**

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | `go1.23.x.darwin-arm64.pkg` |
| macOS (Intel) | `go1.23.x.darwin-amd64.pkg` |
| Windows | `go1.23.x.windows-amd64.msi` |
| Linux | `go1.23.x.linux-amd64.tar.gz` |

### macOS / Windows

1. Download the `.pkg` (macOS) or `.msi` (Windows) installer
2. Double-click to run it
3. Follow the installation prompts
4. **Restart your terminal**

### Linux

```bash
# Download (replace version as needed)
curl -LO https://go.dev/dl/go1.23.4.linux-amd64.tar.gz

# Extract to /usr/local
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz

# Add to PATH (add this to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
```

### Verify Installation

```bash
go version
# Should output: go version go1.23.x ...
```

---

## Building from Source

```bash
# Build the binary
make build

# Verify it works
./ez examples/hello.ez
```

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the `ez` binary |
| `make install` | Install to `/usr/local/bin` |
| `make test` | Run Go unit tests |
| `make integration-test` | Run integration test suite |
| `make clean` | Remove build artifacts |

---

## Making Changes

The fastest way to test a change:

```bash
# 1. Edit a file (e.g., pkg/stdlib/strings.go)

# 2. Rebuild
make build

# 3. Test with the REPL
./ez repl

# Or test with a quick .ez file
./ez examples/hello.ez
```

No need to write formal tests just to see if your change works!

---

## Project Structure

```
EZ/
├── cmd/ez/              # CLI entry point
│   ├── main.go          # Command handling
│   ├── repl.go          # Interactive REPL
│   └── update.go        # Self-update feature
├── pkg/                 # Core language implementation
│   ├── lexer/           # Tokenization (source → tokens)
│   ├── parser/          # Parsing (tokens → AST)
│   ├── typechecker/     # Static type checking
│   ├── interpreter/     # Runtime evaluation
│   ├── object/          # Runtime object system
│   ├── stdlib/          # Standard library modules
│   ├── ast/             # Abstract Syntax Tree
│   └── errors/          # Error handling
├── examples/            # Example EZ programs
├── integration-tests/   # End-to-end tests
└── Makefile             # Build commands
```

### Key Files for Common Tasks

| Task | Where to Look |
|------|---------------|
| Add a stdlib function | `pkg/stdlib/*.go` |
| Change syntax/grammar | `pkg/parser/parser.go` |
| Add a new keyword | `pkg/lexer/lexer.go` + `pkg/tokenizer/token.go` |
| Modify type checking | `pkg/typechecker/typechecker.go` |
| Change runtime behavior | `pkg/interpreter/evaluator.go` |
| Improve error messages | `pkg/errors/` |

---

## Running Tests

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/lexer/...
go test ./pkg/parser/...
```

### Integration Tests

```bash
./integration-tests/run_tests.sh
```

### Test Coverage

```bash
# Generate coverage report
go test ./pkg/... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out
```

---

## Code Style

- Run `gofmt` before committing: `gofmt -w .`
- Follow standard Go conventions
- Keep functions focused and readable
- Add comments for non-obvious logic

---

## Submitting a Pull Request

1. **Create a branch** for your feature:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** and test them

3. **Commit** with a clear message:
   ```bash
   git commit -m "feat: Add shout() function to strings module"
   ```

4. **Push** to your fork:
   ```bash
   git push origin feat/my-feature
   ```

5. **Open a Pull Request** against `main`

### Branch Naming

- `feat/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring

### PR Guidelines

- Keep changes focused and small when possible
- Include a clear description of what changed and why
- Make sure tests pass (`go test ./...`)
- Add tests for new features when appropriate

---

## Reporting Issues

### Bug Reports

When reporting a bug, please include:
- EZ version (`ez version`)
- Operating system
- Minimal code example that reproduces the issue
- Expected vs actual behavior

### Feature Requests

- Describe the use case
- Explain why existing features don't solve it
- Provide examples of how it would work

---

## Good First Issues

Looking for something to work on? Check out issues labeled **"good first issue"**:

👉 [Good First Issues](https://github.com/SchoolyB/EZ/labels/good%20first%20issue)

Some ideas:
- Add documentation comments to stdlib functions
- Improve error messages
- Add new examples
- Fix typos in docs

---

## Questions?

Feel free to open an issue if you get stuck or have questions!
