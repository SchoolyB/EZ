# Contributing to EZ

Thanks for your interest in contributing to EZ! This guide will help you get started.

> **Platform note:** EZ v3.0 supports **macOS** and **Linux** only. Windows is not a supported contributor platform — you won't be able to build or test locally. Windows support is on the roadmap (tracked in #1342).

Before you start coding, skim [`STANDARD.md`](./STANDARD.md) — the language specification. It's the canonical reference for syntax, types, and every stdlib module.

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

- **Go 1.23 or higher** (for the `ez` CLI tooling)
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
| Linux | `go1.23.x.linux-amd64.tar.gz` |

### macOS

1. Download the `.pkg` installer
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

# Verify it works — scaffold a throwaway project and run it
./ez pz /tmp/hello-ez && ./ez /tmp/hello-ez/main.ez
```

> **Iterating on the compiler?** When you edit anything under `ezc/src/` and rebuild locally, you need to tell the `ez` wrapper to use your *local* `ezc` binary instead of the system-installed one. Set `EZ_COMPILER_PATH=./ezc/ezc` on every command while iterating:
>
> ```bash
> EZ_COMPILER_PATH=./ezc/ezc ./ez /tmp/hello-ez/main.ez
> ```
>
> Without this, you'll silently run against whatever `ezc` is installed on your system and wonder why your changes had no effect.

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the `ez` binary |
| `make install` | Install to `/usr/local/bin` |
| `make test` | Run Go unit tests |
| `make clean` | Remove build artifacts |

Integration tests run via a shell script rather than a Makefile target — see [Running Tests](#running-tests).

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
6. **Stdlib** (`ezc/src/stdlib/`) — Standard library modules compiled into `libezrt.a`.

The `ez` CLI (`cmd/ez/`) is a Go tooling wrapper that invokes the compiler for compilation. It provides the REPL, watch mode, doc generation, project scaffolding, and self-update.

**What this means in practice:** If you're adding a new language feature, you'll touch the C compiler stages (lexer → parser → typechecker → codegen). If you're adding a stdlib function, you add it to `ezc/src/stdlib/` and wire it into the codegen. If you're improving developer tooling (REPL, watch, etc.), you work in the Go code under `cmd/ez/`.

---

## Making Changes

The fastest way to iterate on a change:

```bash
# 1. Edit a file (e.g., ezc/src/stdlib/ez_strings.c)

# 2. Rebuild
make build

# 3. Test with a quick .ez file (scaffolded via the project templates)
./ez pz /tmp/hello-ez
EZ_COMPILER_PATH=./ezc/ezc ./ez /tmp/hello-ez/main.ez

# Or test with the REPL
EZ_COMPILER_PATH=./ezc/ezc ./ez repl
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
│   └── lineeditor/      # REPL line editor
├── ezc/                 # EZ compiler (C)
│   ├── src/lexer/       # Tokenization
│   ├── src/parser/      # Parsing (tokens → AST)
│   ├── src/typechecker/ # Static type checking
│   ├── src/codegen/     # Code generation (AST → C)
│   ├── src/runtime/     # Runtime (strings, arrays, maps, arenas)
│   ├── src/stdlib/      # Standard library modules
│   └── tests/           # Unit, e2e, integration tests
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
| Add/modify error codes | `ezc/src/util/error_codes.h` |
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
go test ./pkg/lineeditor/...
```

### Integration Tests

```bash
bash scripts/run_tests.sh
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
    mut passed int = 0
    mut failed int = 0

    // Test 1: description
    mut result int = 1 + 1
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
    mut x int = "hello"  // Should produce E3001
}
```

Name fail tests by error code: `E3001_type_mismatch.ez`.

**Multi-file tests** (`integration-tests/pass/multi-file/`) use subdirectories with a `main.ez` and supporting module files. Useful for testing imports, module visibility, and cross-file features.

For more details, see `TESTING.md`.

---

## Code Style

### Go code (`cmd/ez/`, `pkg/`, `internal/`)

- Run `gofmt` before committing
- Follow standard Go conventions
- Keep functions focused and readable

### C code (`ezc/src/`)

- The repo ships a `.clang-format` under `ezc/` — run `clang-format -i` on files you touch
- Match the style of the surrounding code; the compiler is consistent throughout

### General

- Add comments only for non-obvious logic — don't explain what the code already says
- When adding error codes, register them in `ezc/src/util/error_codes.h` (the canonical source). After adding or changing codes, run `scripts/generate_errors.sh` to regenerate `ERRORS.md` for the docs site.

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

**Scope** is optional but helpful — it indicates what area of the codebase is affected. Canonical scopes:

| Scope | Area |
|-------|------|
| `lexer` | Tokenization (`ezc/src/lexer/`) |
| `parser` | Parsing / AST construction (`ezc/src/parser/`) |
| `typechecker` | Static type checking (`ezc/src/typechecker/`) |
| `codegen` | C code generation (`ezc/src/codegen/`) |
| `runtime` | Runtime library (`ezc/src/runtime/`) |
| `stdlib` | Standard library (`ezc/src/stdlib/`) — optionally nested, e.g. `stdlib/strings` |
| `cli` | Go CLI wrapper (`cmd/ez/`) |
| `tests` | Test additions or test infrastructure |
| `docs` | Documentation only |
| `ci` | GitHub Actions, release workflows |

Examples:

```bash
feat(parser): add named return variables support
fix(typechecker): prevent false positive on nested struct init
test(integration): add 7 core language integration tests
docs(stdlib): add doc comments to math module
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
- Make sure all tests pass (`make test` for unit, `bash scripts/run_tests.sh` for integration)
- Include unit and/or integration tests for new features and bug fixes

---

## Reporting Issues

### Bug Reports

**Every bug report must include the full output of `ez report`.** No exceptions. Run it and paste the complete output at the top of your issue — version, commit, install path, OS, CPU, RAM, and C compiler info are all load-bearing for triage. Bugs filed without `ez report` output will be asked for it before anything else happens.

Use this template:

```markdown
## Severity
Critical / High / Medium / Low — and why.

## System info
(paste full `ez report` output here)

## Summary
One paragraph. What's broken, in plain language.

## Reproduction
Minimal `.ez` program that triggers the bug. Inline the code in a fenced block.

## Expected
What should happen.

## Actual
What actually happens. Include the error message verbatim if there is one.

## Where to look (optional)
If you've poked around and have a guess at the root cause, note it. Saves triage time.

## Related (optional)
Link any related issues or recent commits that touched the same area.
```

Bugs filed in this shape get triaged quickly. One-paragraph bugs without a repro get parked.

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
- Fix typos in docs

---

## Questions?

Feel free to open an issue if you get stuck or have questions!
