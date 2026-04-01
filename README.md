<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

<h3 align="center">Programming Made EZ</h3>

<p align="center">
  A statically-typed programming language that compiles to fast native binaries.
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

## Quick Start

```bash
git clone https://github.com/SchoolyB/EZ.git
cd EZ
make build
make install
```

```bash
ez examples/basic/hello.ez
```

That's it. EZ compiles your code to a native binary, executes it, and cleans up.

**Requirements:** Go 1.21+ and a C compiler (gcc or clang)

---

## Commands

```
ez <file.ez>              Compile and run
ez run <file.ez>          Compile and run (alias)
ez build <file.ez> -o app Compile to a distributable binary
ez check <file.ez>        Type check without compiling
ez repl                   Interactive REPL
ez watch <file.ez>        Watch for changes, re-run on save
ez doc ./...              Generate docs from #doc attributes
ez pz <name>              Scaffold a new project
ez test                   Run the full test suite
ez report                 Print system info for bug reports
ez update                 Update to the latest version
ez version                Show version info
```

---

## What EZ Looks Like

```ez
import @std
using std

const Person struct {
    name string
    age int
}

do greet(p Person) -> string {
    return "Hello, ${p.name}! You are ${p.age} years old."
}

do main() {
    mut p = Person{name: "Alice", age: 30}
    println(greet(p))
}
```

---

## Updating

```bash
ez update
```

Checks for new versions, shows the changelog, and upgrades both the `ez` CLI and the compiler.

---

## Bug Reports

Found a bug? Run `ez report` to gather your system info, then open an issue at [github.com/SchoolyB/EZ/issues](https://github.com/SchoolyB/EZ/issues) and paste the output:

```bash
ez report
```

```
EZ Bug Report Info
==================
EZ Version:  v3.0.0
Commit:      abc1234
Compiler:    ezc 3.0.0
OS:          darwin/arm64
RAM:         16 GB
```

Include this output along with a description of the bug, the EZ code that triggers it, and what you expected to happen.

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
