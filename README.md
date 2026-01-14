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

**macOS/Linux:**
```bash
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Build the binary
make build

# Run a program
./ez examples/hello.ez
```

**Windows (PowerShell):**
```powershell
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Build the binary
go build -o ez.exe ./cmd/ez

# Run a program
.\ez.exe examples\hello.ez
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

# Create install directory and move binary
New-Item -ItemType Directory -Path "$env:ProgramFiles\ez" -Force
Move-Item ez.exe "$env:ProgramFiles\ez\ez.exe" -Force

# Add to PATH (if not already present)
$path = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($path -notlike "*$env:ProgramFiles\ez*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$env:ProgramFiles\ez", "Machine")
    Write-Host "Added to PATH - restart your terminal for changes to take effect"
}
```

Alternatively, if building from source, you can use the install script:
```powershell
# Run as Administrator for system-wide install, or as regular user for user-local install
.\install.ps1
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
