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

## Running Tests

EZ has two test suites: **passing tests** (verify features work) and **error tests** (verify error messages).

```bash
# Run passing tests
./ez tests/comprehensive.ez
./ez tests/multi-file/main.ez
./ez tests/nested-test/main/main.ez

# Run error tests
./tests/errors/run_error_tests.sh ./ez
```

---

## License

MIT License - Copyright (c) 2025-Present Marshall A Burns

See [LICENSE](LICENSE) for details.
