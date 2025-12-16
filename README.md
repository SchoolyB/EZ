<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

<h3 align="center">Programming Made EZ</h3>

<p align="center">
  A simple, interpreted, statically-typed programming language designed for clarity and ease of use.
</p>

<p align="center">
  <a href="https://schoolyb.github.io/language.ez">Website</a> •
  <a href="https://schoolyb.github.io/language.ez/docs">Documentation</a>
</p>

<p align="center">
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL Security Scan"></a>
</p>

---

## Developer Quick Start

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

**macOS/Linux:**
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

**Windows (PowerShell as Administrator):**
```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/SchoolyB/EZ/releases/latest/download/ez-windows-amd64.zip" -OutFile ez.zip
Expand-Archive ez.zip -DestinationPath .
Remove-Item ez.zip

# Move to a directory in your PATH (e.g., C:\Program Files\ez)
Move-Item ez.exe "C:\Program Files\ez\ez.exe"
```

After this one-time manual update, `ez update` will work automatically for all future versions.

---

## Running Tests

```bash
# Run integration tests
./integration-tests/run_tests.sh

# Run unit tests
go test ./...
```

For more details, see the [Testing Guide](TESTING.md).

---

## License

MIT License - Copyright (c) 2025-Present Marshall A Burns

See [LICENSE](LICENSE) for details.
