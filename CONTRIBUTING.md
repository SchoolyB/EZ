# Contributing to EZ

Thanks for your interest in contributing to EZ! This guide will help you get started.

## Table of Contents

- [Getting Started](#getting-started)
- [Installing Go](#installing-go)
- [Building from Source](#building-from-source)
- [How the Compiler Works](#how-the-compiler-works)
- [Making Changes](#making-changes)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Reporting Issues](#reporting-issues)
- [AI-Assisted Contributions](#ai-assisted-contributions)
- [Good First Issues](#good-first-issues)

---

## Getting Started

### Prerequisites

- **Go 1.21 or higher** (for the `ez` CLI tooling)
- **C compiler** (gcc or clang)
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

## How the Compiler Works

EZ compiles source code to native binaries through a pipeline of stages. Understanding this helps you know which files to touch for different kinds of changes.

```
Source Code → Lexer → Parser → Type Checker → Code Gen → C Compiler → Binary
  (.ez)      tokens    AST     validated AST    .c file    cc/gcc       native
```

All compiler stages live in `ezc/src/`:

1. **Lexer** (`ezc/src/lexer/`) — Reads source text and produces tokens.
2. **Parser** (`ezc/src/parser/`) — Turns tokens into an Abstract Syntax Tree (AST).
3. **Type Checker** (`ezc/src/typechecker/`) — Walks the AST and validates types, catches errors before compilation.
4. **Code Generator** (`ezc/src/codegen/`) — Translates the AST into C source code.
5. **Runtime** (`ezc/src/runtime/`) — Core types (strings, arrays, maps, arenas) linked into every binary.
6. **Stdlib** (`ezc/src/stdlib/`) — Standard library modules (24 modules) compiled into `libezrt.a`.

The `ez` CLI (`cmd/ez/`) is a Go tooling wrapper that invokes `ezc` for compilation. It provides the REPL, watch mode, doc generation, project scaffolding, and self-update.

**What this means in practice:** If you're adding a new language feature, you'll touch the C compiler stages (lexer → parser → typechecker → codegen). If you're adding a stdlib function, you add it to `ezc/src/stdlib/` and wire it into the codegen. If you're improving developer tooling (REPL, watch, etc.), you work in the Go code under `cmd/ez/`.

---

## Making Changes

The fastest way to iterate on a change:

```bash
# 1. Edit a file (e.g., ezc/src/stdlib/ez_strings.c)

# 2. Rebuild
make build

# 3. Test with a quick .ez file
EZC_PATH=./ezc/ezc ./ez run examples/basic/hello.ez

# Or test with the REPL
EZC_PATH=./ezc/ezc ./ez repl
```

This is a great way to quickly validate your change while developing. When your feature is working, make sure to add proper tests before submitting your PR (see [Writing Tests](#writing-tests)).

---

## Project Structure

```
EZ/
├── cmd/ez/              # Go CLI tooling wrapper
│   ├── main.go          # Entry point, version
│   ├── commands.go      # Subcommand definitions
│   ├── repl.go          # Interactive REPL (backed by ezc)
│   ├── watch.go         # File watcher
│   ├── doc.go           # Documentation generator
│   ├── pz.go            # Project scaffolding
│   └── update.go        # Self-update
├── internal/ezc/        # Go wrapper for invoking ezc binary
├── pkg/                 # Go packages (tooling only)
│   ├── errors/          # Error codes and formatting
│   └── lineeditor/      # REPL line editor
├── ezc/                 # EZC compiler (C)
│   ├── src/lexer/       # Tokenization
│   ├── src/parser/      # Parsing (tokens → AST)
│   ├── src/typechecker/ # Static type checking
│   ├── src/codegen/     # Code generation (AST → C)
│   ├── src/runtime/     # Runtime (strings, arrays, maps, arenas)
│   ├── src/stdlib/      # Standard library (24 modules)
│   └── tests/           # Unit, e2e, integration tests
├── examples/            # Example EZ programs
├── integration-tests/   # End-to-end test suite
├── STANDARD.md          # Language specification
└── Makefile             # Build commands (builds both ez + ezc)
```

### Key Files for Common Tasks

| Task | Where to Look |
|------|---------------|
| Add a stdlib function | `ezc/src/stdlib/` + wire in `ezc/src/codegen/codegen.c` |
| Change syntax/grammar | `ezc/src/parser/parser.c` + `ezc/src/parser/ast.h` |
| Add a new keyword/token | `ezc/src/lexer/lexer.c` + `ezc/src/lexer/token.h` |
| Modify type checking | `ezc/src/typechecker/typechecker.c` |
| Change code generation | `ezc/src/codegen/codegen.c` |
| Add/modify error codes | `pkg/errors/codes.go` (Go) or `ezc/src/util/error.c` (C) |
| Improve CLI tooling | `cmd/ez/*.go` |
| Add a new language feature | Parser → Typechecker → Codegen (all in `ezc/src/`) |

---

## Running Tests

### Unit Tests

```bash
# Run all tests (CLI + compiler)
make test

# Compiler tests only
cd ezc
make test-unit     # unit tests
make test-e2e      # end-to-end tests

# CLI/tooling tests only
go test ./pkg/errors/... ./pkg/lineeditor/...
```

### Integration Tests

```bash
bash scripts/run_integration.sh
```

---

## Writing Tests

New features and bug fixes should include tests. The type of test depends on what you're changing.

### Unit Tests (C)

For compiler internals. Run from `ezc/`:

```bash
cd ezc
make test-unit          # lexer, parser, typechecker tests
make test-e2e           # full compilation pipeline tests
```

### Integration Tests (EZ)

For end-to-end behavior — verifying that EZ programs produce the right output. These are `.ez` files in the `integration-tests/` directory.

**Pass tests** (`integration-tests/pass/core/`) should run successfully and verify results:

```ez

do main() {
    temp passed int = 0
    temp failed int = 0

    // Test 1: description
    temp result int = 1 + 1
    if result == 2 {
        println("  [PASS] 1 + 1 = 2")
        passed += 1
    } otherwise {
        println("  [FAIL] 1 + 1: expected 2, got ${result}")
        failed += 1
    }

    println("Results: ${passed} passed, ${failed} failed")
    if failed > 0 {
        println("SOME TESTS FAILED")
    } otherwise {
        println("ALL TESTS PASSED")
    }
}
```

The test runner checks for `SOME TESTS FAILED` in the output — if it's present, the test fails.

**Fail tests** (`integration-tests/fail/errors/`) are minimal programs that should trigger a specific error. The test runner expects a non-zero exit code:

```ez
/*
 * Error Test: E3001 - type-mismatch
 * Expected: "type mismatch"
 */

do main() {
    temp x int = "hello"  // Should produce E3001
}
```

Name fail tests by error code: `E3001_type_mismatch.ez`.

**Multi-file tests** (`integration-tests/pass/multi-file/`) use subdirectories with a `main.ez` and supporting module files. Useful for testing imports, module visibility, and cross-file features.

For more details, see `TESTING.md`.

---

## Code Style

- Run `gofmt` before committing (the pre-commit hook does this automatically)
- Follow standard Go conventions
- Keep functions focused and readable
- Add comments for non-obvious logic
- When adding error codes, register them in `pkg/errors/codes.go`

---

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/). The format is:

```
type(scope): short description
```

**Types:**

| Type | When to Use |
|------|-------------|
| `feat` | New feature or functionality |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `refactor` | Code change that doesn't fix a bug or add a feature |
| `chore` | Build, CI, tooling, or maintenance |

**Scope** is optional but helpful — it indicates what area of the codebase is affected:

```bash
feat(parser): add named return variables support
fix(typechecker): prevent false positive on nested struct init
test(integration): add 7 core language integration tests
docs(stdlib): add doc comments to @math module
feat(stdlib/strings): add 12 new string utility functions
chore(ci): add workflow_dispatch trigger to release-please
```

Keep the description short (under ~72 characters), lowercase, no period at the end.

---

## Submitting a Pull Request

1. **Create a branch** for your feature:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes** and test them

3. **Commit** with a clear message:
   ```bash
   git commit -m "feat(stdlib): add `shout()` function to strings module"
   ```

4. **Push** to your fork:
   ```bash
   git push origin feat/my-feature
   ```

5. **Open a Pull Request** against `main`

### Branch Naming

- `feat/` — New features
- `fix/` — Bug fixes
- `docs/` — Documentation changes
- `refactor/` — Code refactoring
- `test/` — Test additions

### PR Guidelines

- Keep changes focused and small when possible
- Include a clear description of what changed and why
- Make sure all tests pass (`go test ./...` or `make integration-test`)
- Include unit and/or integration tests for new features and bug fixes

---

## Reporting Issues

### Bug Reports

When reporting a bug, please include:
- System info from `ez report` (paste the full output)
- Minimal code example that reproduces the issue
- Expected vs actual behavior

### Feature Requests

- Describe the use case
- Explain why existing features don't solve it
- Provide examples of how it would work

---

## AI-Assisted Contributions

EZ was built with heavy use of AI tooling from day one, and AI-assisted contributions are absolutely welcome. Whether you're using Claude, Copilot, ChatGPT, or any other tool to help write code, that's fine. What matters is the end result.

That said, AI-generated code still needs a human behind it. You're responsible for what you submit. Don't just paste AI output into a PR — build it, run it, and make sure it actually works. If you can't explain what your code does and why, it's not ready to submit. AI gets things wrong all the time: hallucinated APIs, subtle logic bugs, code that looks right but breaks edge cases. Read through it critically before pushing.

The bar for AI-assisted contributions is the same as any other contribution: does it work, is it tested, and does it solve the problem?

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
