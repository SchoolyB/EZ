<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

<h3 align="center">Programming Made EZ</h3>

<p align="center">
  A simple, statically-typed programming language designed for clarity and ease of use.
</p>

<p align="center">
  <a href="https://schoolyb.github.io/EZ-Language-Webapp/playground/">Try EZ Online</a> •
  <a href="https://schoolyb.github.io/EZ-Language-Webapp/docs">Learn About EZ</a>
</p>

<p align="center">
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL Security Scan"></a>
</p>

---

## Two Ways to Run EZ

EZ programs can be **interpreted** or **compiled to native binaries**. Same language, same `.ez` files, you choose how to run them.

| | `ez` (interpreter) | `ezc` (compiler) |
|---|---|---|
| **Run** | `ez hello.ez` | `ezc hello.ez -o hello && ./hello` |
| **Speed** | Instant startup | Native performance |
| **Best for** | Scripting, REPL, learning | Shipping binaries, performance |
| **Written in** | Go | C |

---

## Quick Start

```bash
git clone https://github.com/SchoolyB/EZ.git
cd EZ
```

### Interpreter (`ez`)

```bash
make build
./ez examples/basic/hello.ez
```

**Requirements:** Go 1.23.1 or higher

### Compiler (`ezc`)

```bash
cd ezc && make build
./ezc ../examples/basic/hello.ez -o hello
./hello
```

**Requirements:** C compiler (gcc or clang)

`ezc` works from any directory — it finds its runtime headers automatically. You can also install it system-wide:

```bash
sudo make install    # installs to /usr/local/bin/ezc
ezc hello.ez -o hello
```

For pre-built binaries and full documentation, visit the [documentation](https://schoolyb.github.io/EZ-Language-Webapp/docs).

Want to contribute or build from source? See the [Contributing Guide](CONTRIBUTING.md).

---

## Updating

EZ includes a built-in update command:

```bash
ez update
```

This will check for new versions, show the changelog, and prompt you to upgrade. If EZ is installed in a system directory (like `/usr/local/bin`), it will automatically prompt for your password.

## Running Tests

```bash
# Interpreter tests
make intergration-tests       # or ./integration-tests/run_tests.sh
go test ./...

# Compiler tests
cd ezc
make test-unit                # 35 unit tests (lexer + parser)
make test-parity              # compare ezc output against ez interpreter
make test                     # run both
```

For more details, see the [Testing Guide](TESTING.md).

---

## License

MIT License - Copyright (c) 2025-Present Marshall A Burns

See [LICENSE](LICENSE) for details.

---

## Contributors

Thank you to everyone who has contributed to EZ!

<a href="https://github.com/akamikado"><img src="https://github.com/akamikado.png" width="50" height="50" alt="akamikado"/></a>
<a href="https://github.com/CobbCoding1"><img src="https://github.com/CobbCoding1.png" width="50" height="50" alt="CobbCoding1"/></a>
<a href="https://github.com/CFFinch62"><img src="https://github.com/CFFinch62.png" width="50" height="50" alt="CFFinch62"/></a>
<a href="https://github.com/Aryan-Shrivastva"><img src="https://github.com/Aryan-Shrivastva.png" width="50" height="50" alt="Aryan-Shrivastva"/></a>
<a href="https://github.com/arjunpathak072"><img src="https://github.com/arjunpathak072.png" width="50" height="50" alt="arjunpathak072"/></a>
<a href="https://github.com/deepika1214"><img src="https://github.com/deepika1214.png" width="50" height="50" alt="deepika1214"/></a>
<a href="https://github.com/blackgirlbytes"><img src="https://github.com/blackgirlbytes.png" width="50" height="50" alt="blackgirlbytes"/></a>
<a href="https://github.com/majiayu000"><img src="https://github.com/majiayu000.png" width="50" height="50" alt="majiayu000"/></a>
<a href="https://github.com/prjctimg"><img src="https://github.com/prjctimg.png" width="50" height="50" alt="prjctimg"/></a>
<a href="https://github.com/jaideepkathiresan"><img src="https://github.com/jaideepkathiresan.png" width="50" height="50" alt="jaideepkathiresan"/></a>
<a href="https://github.com/Abhishek022001"><img src="https://github.com/Abhishek022001.png" width="50" height="50" alt="Abhishek022001"/></a>
<a href="https://github.com/Scanf-s"><img src="https://github.com/Scanf-s.png" width="50" height="50" alt="Scanf-s"/></a>
<a href="https://github.com/HCH1212"><img src="https://github.com/HCH1212.png" width="50" height="50" alt="HCH1212"/></a>
<a href="https://github.com/elect0"><img src="https://github.com/elect0.png" width="50" height="50" alt="elect0"/></a>
<a href="https://github.com/jgafnea"><img src="https://github.com/jgafnea.png" width="50" height="50" alt="jgafnea"/></a>
<a href="https://github.com/madhav-murali"><img src="https://github.com/madhav-murali.png" width="50" height="50" alt="madhav-murali"/></a>
<a href="https://github.com/preettrank53"><img src="https://github.com/preettrank53.png" width="50" height="50" alt="preettrank53"/></a>
<a href="https://github.com/TechLateef"><img src="https://github.com/TechLateef.png" width="50" height="50" alt="TechLateef"/></a>
<a href="https://github.com/dtee1"><img src="https://github.com/dtee1.png" width="50" height="50" alt="dtee1"/></a>
