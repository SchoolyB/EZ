# Contribute to EZ

EZ is open source and welcomes contributions of all kinds. Whether you're fixing a typo or implementing a new stdlib module, every contribution helps.

## Getting Started

### Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/EZ.git
cd EZ
```

### Build the Project

```bash
go build ./cmd/ez
```

### Run Tests

```bash
go test ./...
```

### Verify Installation

```bash
./ez version
```

## Development Workflow

### Branch Naming

Use descriptive branch names with a prefix:

| Prefix | Purpose |
| :--- | :--- |
| `fix/` | Bug fixes (e.g., `fix/parser-struct-literal`) |
| `feature/` | New features (e.g., `feature/sets-module`) |
| `docs/` | Documentation changes (e.g., `docs/contributing-guide`) |
| `chore/` | Maintenance tasks (e.g., `chore/update-deps`) |

### Commit Message Format

Write clear, concise commit messages:

```
<type>: <short description>

<optional longer description>
```

**Examples:**
- `fix: handle nil pointer in parser`
- `feature: add @sets stdlib module`
- `docs: update CONTRIBUTING.md`

### PR Process

1. Create a branch from `main`
2. Make your changes with clear commits
3. Run tests locally: `go test ./...`
4. Push to your fork and open a PR
5. Fill out the PR template with details
6. Wait for review and address feedback

## Code Style

### Go Code

- Run `gofmt` before committing (or use editor integration)
- Follow standard Go conventions
- Keep functions focused and testable
- Add comments for exported functions

### EZ Code (Examples & Tests)

- Use `snake_case` for variables and functions
- Use `SCREAMING_SNAKE_CASE` for constants
- Use `PascalCase` for struct/enum types
- Keep examples simple and focused

## Project Structure

```
EZ/
├── cmd/
│   └── ez/              # CLI entry point (main.go)
├── pkg/
│   ├── ast/             # Abstract Syntax Tree definitions
│   ├── errors/          # Error types and formatting
│   ├── interpreter/     # Runtime evaluation (evaluator.go)
│   ├── lexer/           # Tokenizer (lexer.go)
│   ├── lineeditor/      # REPL line editing
│   ├── object/          # Runtime object types
│   ├── parser/          # Parser (parser.go)
│   ├── stdlib/          # Standard library modules
│   ├── tokenizer/       # Token types and definitions
│   └── typechecker/     # Static type checking
├── examples/            # Example EZ programs
├── fix_docs/            # Internal documentation
└── scripts/             # Build and release scripts
```

### Key Components

| Component | Location | Purpose |
| :--- | :--- | :--- |
| Lexer | `pkg/lexer/` | Converts source code to tokens |
| Parser | `pkg/parser/` | Converts tokens to AST |
| Type Checker | `pkg/typechecker/` | Static analysis and type validation |
| Interpreter | `pkg/interpreter/` | Executes AST nodes |
| Stdlib | `pkg/stdlib/` | Built-in library functions |

## Testing

### Running All Tests

```bash
go test ./...
```

### Running Tests for a Specific Package

```bash
go test ./pkg/parser/...
go test ./pkg/interpreter/...
```

### Running Tests with Verbose Output

```bash
go test -v ./pkg/parser/...
```

### Adding New Tests

1. Create test functions in `*_test.go` files
2. Follow existing test patterns in the package
3. Use table-driven tests for multiple cases
4. Test both success and error cases

### Example Test Structure

```go
func TestParseIfStatement(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"if true { }", "if true { }"},
        // ... more cases
    }
    
    for _, tt := range tests {
        // test logic
    }
}
```

## Ways to Contribute


### Report Bugs
Found something broken? Open an issue with reproduction steps.

### Request Features
Have an idea? Propose new functionality or improvements.

### Write Examples
Help others learn by contributing real-world code examples.

### Improve Stdlib
Add functions to `@math`, `@random`, `@strings`, `@arrays`, `@maps`, and more.

### Fix Bugs
Pick up an open issue and submit a pull request.

### Improve Docs
Clarify explanations, fix typos, or add missing documentation.

## Creating Issues
Good issues help maintainers understand and fix problems quickly. Follow these guidelines when creating an issue.

### Issue Title Format
All issue titles must use one of these prefixes:

| Prefix | When to Use |
| :--- | :--- |
| `Bug:` | Something is broken or produces incorrect results |
| `Feature:` | New functionality or enhancement |
| `Documentation:` | Docs improvements or additions |
| `Tests:` | Unit tests or test infrastructure |
| `Cleanup:` | Removing dead code or organizing |

### Bug Reports
Include these sections in every bug report:

* **Description** — 1-3 sentences describing the bug
* **Severity** — High, Medium, or Low with justification
* **Reproduction** — Minimal EZ code that reproduces the bug
* **Expected Behavior** — What should happen
* **Actual Behavior** — What actually happens

### Feature Requests
Include these sections in feature requests:

* **Description** — What the feature does and why it's useful
* **Tasks** — Actionable items with checkboxes
* **Proposed Syntax** — Example EZ code showing usage
* **Expected Behavior** — How it should work

[View full issue template →](https://github.com/SchoolyB/EZ/issues/new/choose)

## Pull Requests
Ready to contribute code? Here's how to submit a pull request.

1. **Fork & Clone**
   Fork the repository and clone it locally.

2. **Create a Branch**
   Create a branch with a descriptive name: `fix/parser-struct-literal` or `feature/sets-module`

3. **Make Changes**
   Write your code. Follow existing patterns and style.

4. **Test**
   Run existing tests and add new ones for your changes.

5. **Submit PR**
   Push your branch and open a pull request with a clear description of changes.

## Contributing Examples
Good examples help people learn EZ faster. When writing examples:

* **Keep it focused** — One concept per example
* **Make it runnable** — Examples should work as-is when copied
* **Add comments** — Explain non-obvious parts
* **Show real use cases** — Practical > contrived
* **Test edge cases** — Show how to handle errors or unusual inputs

## Contributing to Stdlib
The standard library is where EZ gets its "batteries included" feel. Here's how to contribute:

### Available Modules
* `@std`
* `@math`
* `@random`
* `@strings`
* `@arrays`
* `@maps`
* `@time`
* `@bytes`
* `@io`
* `@os`
* `@json`
* `@binary`

### Guidelines
* **Check for duplicates** — Make sure the function doesn't already exist
* **Follow naming conventions** — `snake_case` for functions, `SCREAMING_SNAKE_CASE` for constants, `PascalCase` for types
* **Write tests** — Every stdlib function needs test coverage
* **Document it** — Add to the docs with examples
* **Keep it simple** — EZ values simplicity over cleverness

### Proposing New Modules
Have an idea for an entirely new stdlib module? You can propose one. Open a feature issue with:

* **Module name** — What would it be called? (e.g., `@sets`, `@regex`, `@http`)
* **Purpose** — What problem does it solve?
* **Proposed API** — List the functions, constants, and types it would include
* **Example usage** — Show how it would be used in EZ code

> Not every proposal will be accepted — EZ intentionally keeps a limited scope — but good ideas that fit the language's goals are welcome.