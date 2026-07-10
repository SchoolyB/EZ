<p align="center">
  <img src="images/Flip.png" alt="Flip - EZ Mascot" width="275">
</p>
<h1 align="center">EZ</h1>

<p align="center">
  A compiled language that's actually approachable!
</p>

<h3 align="center">Programming made EZ.</h3>

<p align="center">
  <a href="STANDARD.md">Learn More About EZ</a>
</p>

<p align="center">
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL Security Scan"></a>
</p>

---

```ez
do main() {
    println("Hello, World!")
}
```

```ez
import @json

#json
const User struct {
    name string
    age int
}

do main() {
    mut u = new(User)^
    u.name = "Marshall"
    u.age = 31
    
    println(json.stringify(u))
}
// Output: {"name":"Marshall","age":31}
```

```ez
import @io

do main() {
    mut content, err = io.read_file("config.json")
    if err != nil {
        eprintln("Failed: ${err}")
        exit(1)
    } or len(content) == 0 {
        println("No content found")
        exit(1)
    } otherwise {
        println("Content found: ${content}")
    }
}
```

```ez
do main() {
    mut score int = 82
    mut grade char

    when score {
        is range(90, 101) { grade = 'A' }
        is range(80, 90)  { grade = 'B' }
        is range(70, 80)  { grade = 'C' }
        is range(60, 70)  { grade = 'D' }
        default           { grade = 'F' }
    }

    println("Your grade is a ${grade}")
}
```

---

## Why EZ?

- **Readable syntax** — `for_each`, `as_long_as`, `if`/`or`/`otherwise`, `when`/`is`, `in`/`not_in`, `or_return`, `ensure`, `bit_and`/`bit_or`/`bit_xor`.
- **Versatility through simplicity** — Scripts, microservices, CLI tools, or a project where you want to learn systems programming fundamentals EZ is designed for you!
- **A compiler that works with the programmer** — Clear error messages, a builtin formatter, live reloading, builtin docs via `ez man`. You should never have to wonder what went wrong or where.

---

## Standard Library

<p align="center">

`arrays` · `strings` · `maps` · `math` · `time` · `random` · `json` · `io` · `os`
`http` · `server` · `crypto` · `encoding` · `uuid` · `bytes` · `binary` · `sqlite`
`regex` · `csv` · `net` · `threads` · `sync` · `channels` · `mem` · `atomic` · `fmt` · `strconv`

</p>

---

## Install

### Binary download (recommended)

Download the latest release for your platform from the [Releases page](https://github.com/SchoolyB/EZ/releases). No dependencies required.

### Build from source

Requires Go 1.23+ and a C compiler (gcc or clang).

```bash
git clone https://github.com/SchoolyB/EZ.git
cd EZ
make build
make install
```

> **Note:** EZ currently supports **macOS** and **Linux** only.

---

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `ez <file>` | Compile and run | `ez main.ez` |
| `ez build <file> -o <name>` | Compile to a distributable binary | `ez build main.ez -o myapp` |
| `ez check <file>` | Type check without compiling | `ez check main.ez` |
| `ez watch <file>` | Watch for changes, re-run on save | `ez watch main.ez` |
| `ez fmt <path>` | Format `.ez` source files in place | `ez fmt .` or `ez fmt ./...` |
| `ez fmt --check <path>` | Check formatting without modifying files (CI gate) | `ez fmt --check ./...` |
| `ez doc <file>` | Generate docs from `#doc` attributes | `ez doc main.ez` |
| `ez pz <name>` | Scaffold a new project | `ez pz myproject` |
| `ez report` | Print system info for bug reports | `ez report` |
| `ez update` | Update to the latest stable version | `ez update` |
| `ez update --pre` | Update to the latest pre-release (alpha/beta) | `ez update --pre` |
| `ez install <version>` | Install a specific version by semver | `ez install 2.5.0` |
| `ez version` | Show version info | `ez version` |
| `ez man` | Show help for the man command | `ez man` |
| `ez man <module>` | Show info about a stdlib module | `ez man strings` |
| `ez man <function>` | Show info about a stdlib function | `ez man to_upper` |
| `ez man <struct>` | Show info about a stdlib struct type | `ez man HttpRequest` |

---

## Updating

```bash
ez update              # latest stable
ez update --pre        # latest pre-release
ez install 2.5.0       # pin to an exact version
```

`ez update` checks for new versions, shows the changelog, and upgrades both the `ez` CLI and the compiler. Pass `--pre` to pick up the latest alpha, beta, or rc. Use `ez install <version>` to install an exact version by semver — downgrades and pre-release tags (e.g. `3.0.0-beta.2`) are supported.

---

## Learn More

- [Language standard](STANDARD.md)
- [Contributing guide](CONTRIBUTING.md)

## Tooling
- [The EZ Language Server Protocol](https://github.com/SchoolyB/EZLS)
---

## Status

EZ is in active development. The language is usable for personal projects and dev tools. Breaking changes may occur frequently.

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
<a href="https://github.com/SAY-5"><img src="https://github.com/SAY-5.png" width="50" height="50" alt="SAY-5"/></a>
<a href="https://github.com/mvanhorn"><img src="https://github.com/mvanhorn.png" width="50" height="50" alt="mvanhorn"/></a>
<a href="https://github.com/kas2804"><img src="https://github.com/kas2804.png" width="50" height="50" alt="kas2804"/></a>
<a href="https://github.com/su-s2008"><img src="https://github.com/su-s2008.png" width="50" height="50" alt="su-s2008"/></a>
