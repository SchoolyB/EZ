<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

<p align="center">
  A statically typed, compiled programming language.
</p>

<h3 align="center">Programming made EZ.</h3>

<p align="center">
  <a href="https://schoolyb.github.io/EZ-Language-Webapp/playground/">Try EZ Online</a> •
  <a href="https://schoolyb.github.io/EZ-Language-Webapp/docs">Learn About EZ</a>
</p>

<p align="center">
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml"><img src="https://github.com/SchoolyB/EZ/actions/workflows/codeql-analysis.yml/badge.svg" alt="CodeQL Security Scan"></a>
</p>

---

## What is EZ

EZ is a statically typed, compiled programming language that produces native binaries. Source files (`.ez`) are compiled to C, then to machine code. The result is a single binary with no runtime dependencies.

- Static type system with type inference
- Structs with scoped functions
- Multi-return values and error handling
- String interpolation, enums, when/is expressions
- 26 standard library modules (HTTP, JSON, crypto, SQLite, threads, and more)
- Compiles in milliseconds

```ez
const Circle struct {
    radius float

    do area(c Circle) -> float {
        return 3.14159 * c.radius * c.radius
    }
}
```

```ez
do divide(a float, b float) -> (float, Error) {
    if b == 0.0 {
        return 0.0, error("division by zero")
    }
    return a / b, nil
}
```

```ez
do main() {
    mut names [string] = {"Alice", "Bob", "Charlie"}
    for_each name in names {
        println("Hello, ${name}!")
    }
}
```

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

## Quick Start

Create a file called `main.ez`:

```ez
do main() {
    println("Hello, World!")
}
```

Run it:

```bash
ez main.ez
```

EZ compiles your code to a native binary, executes it, and cleans up.

---

## Commands

```
ez <file.ez>              Compile and run
ez build <file.ez> -o app Compile to a distributable binary
ez check <file.ez>        Type check without compiling
ez repl                   Interactive REPL
ez watch <file.ez>        Watch for changes, re-run on save
ez doc ./...              Generate docs from #doc attributes
ez pz <name>             Scaffold a new project
ez test                   Run the full test suite
ez report                 Print system info for bug reports
ez update                 Update to the latest stable version
ez update --pre           Update to the latest pre-release (alpha/beta/rc)
ez install <version>      Install a specific version (e.g. 2.5.0, 3.0.0-beta.2)
ez version                Show version info
```

---

## Updating

```bash
ez update              # latest stable
ez update --pre        # latest pre-release
ez install 2.5.0       # pin to an exact version
```

`ez update` checks for new versions, shows the changelog, and upgrades both the `ez` CLI and the compiler. Pass `--pre` to pick up the latest alpha, beta, or rc. Use `ez install <version>` to install an exact version by semver — downgrades and pre-release tags (e.g. `3.0.0-beta.2`) are supported.

---

## Bug Reports

Found a bug? Run `ez report` to gather your system info, then open an issue at [github.com/SchoolyB/EZ/issues](https://github.com/SchoolyB/EZ/issues) and paste the output:

```bash
ez report
```

```
EZ Bug Report Info
======================
EZ Version:  v3.0.0-alpha.13  (pre-release)
Commit:      (released build)
Install:     /usr/local/bin/ez
OS:          darwin/arm64  Darwin 24.5.0
CPU:         Apple M2
RAM:         8 GB
C compiler:  /usr/bin/clang
             Apple clang version 17.0.0 (clang-1700.0.13.5)
             target: arm64-apple-darwin24.5.0
```

Include this output along with a description of the bug, the EZ code that triggers it, and what you expected to happen.

---

## Standard Library Reference

EZ ships with 26 standard library modules. Import with `import @module_name`.

Functions marked with `-> (T, Error)` support error-handling via multi-variable destructuring:

```ez
mut content, err = io.read_file("config.json")
if err != nil {
    println("failed to read config: ${err}")
}
```

### Builtins

Available everywhere, no import needed.

```
do println(value any)
do print(value any)
do eprintln(value any)
do eprint(value any)
do input() -> string
do len(value any) -> int
do type_of(value any) -> string
do size_of(Type) -> int
do copy(value any) -> any
do new(Type) -> Type
do ref(func_name) -> func
do addr(lvalue) -> ptr<T>
do error(message string) -> Error
do cast(value any, Type) -> Type
do range(start int, end int) -> [int]
do exit(code int)
do panic(message string)
do assert(condition bool, message string)
do sleep_s(seconds int)
do sleep_ms(ms int)
do sleep_ns(ns int)
do to_char(s string, index int) -> char
do char_count(s string) -> int
do c_string(value string) -> ptr
do int(value any) -> int
do uint(value any) -> uint
do float(value any) -> float
do string(value any) -> string
do char(value any) -> char
do byte(value any) -> byte
do bool(value any) -> bool
```

### io

```
do io.read_file(path string) -> string
do io.write_file(path string, content string) -> bool
do io.append_file(path string, content string) -> bool
do io.delete_file(path string) -> bool
do io.rename_file(old_path string, new_path string) -> bool
do io.copy_file(src string, dst string) -> bool
do io.move_file(src string, dst string) -> bool
do io.file_exists(path string) -> bool
do io.is_file(path string) -> bool
do io.is_directory(path string) -> bool
do io.file_size(path string) -> int
do io.list_dir(path string) -> [string]
do io.make_dir(path string) -> bool
do io.make_dir_all(path string) -> bool
do io.remove_dir(path string) -> bool
do io.remove_dir_all(path string) -> bool
do io.walk(path string) -> [string]
do io.glob(pattern string) -> [string]
do io.path_join(a string, b string) -> string
do io.dirname(path string) -> string
do io.basename(path string) -> string
do io.extension(path string) -> string
do io.is_absolute(path string) -> bool
do io.normalize(path string) -> string
```

Error-returning variants: `read_file`, `write_file`, `delete_file`, `append_file`, `rename_file`, `copy_file`, `move_file`, `list_dir`, `make_dir`, `make_dir_all`, `remove_dir`, `remove_dir_all`, `walk`

### strings

```
do strings.to_upper(s string) -> string
do strings.to_lower(s string) -> string
do strings.trim(s string) -> string
do strings.trim_left(s string) -> string
do strings.trim_right(s string) -> string
do strings.contains(s string, sub string) -> bool
do strings.starts_with(s string, prefix string) -> bool
do strings.ends_with(s string, suffix string) -> bool
do strings.index_of(s string, sub string) -> int
do strings.count(s string, sub string) -> int
do strings.is_empty(s string) -> bool
do strings.replace(s string, old string, new string) -> string
do strings.repeat(s string, count int) -> string
do strings.reverse(s string) -> string
do strings.slice(s string, start int, end int) -> string
do strings.split(s string, sep string) -> [string]
do strings.join(arr [string], sep string) -> string
```

### fmt

```
do fmt.printf(format string, ...)
do fmt.sprintf(format string, ...) -> string
do fmt.format(format string, ...) -> string
do fmt.println(value any)
do fmt.eprintln(value any)
do fmt.eprint(value string)
do fmt.pad_left(s string, width int, ch char) -> string
do fmt.pad_right(s string, width int, ch char) -> string
do fmt.center(s string, width int, ch char) -> string
do fmt.int_to_hex(n int) -> string
do fmt.int_to_binary(n int) -> string
do fmt.int_to_octal(n int) -> string
do fmt.float_fixed(f float, decimals int) -> string
do fmt.float_sci(f float) -> string
```

### math

```
do math.abs(n int) -> int
do math.abs(n float) -> float
do math.sign(n int) -> int
do math.neg(n int) -> int
do math.neg(n float) -> float
do math.min(a int, b int) -> int
do math.min(a float, b float) -> float
do math.max(a int, b int) -> int
do math.max(a float, b float) -> float
do math.clamp(v int, lo int, hi int) -> int
do math.clamp(v float, lo float, hi float) -> float
do math.floor(n float) -> float
do math.ceil(n float) -> float
do math.round(n float) -> float
do math.trunc(n float) -> float
do math.pow(base float, exp float) -> float
do math.sqrt(n float) -> float
do math.cbrt(n float) -> float
do math.hypot(x float, y float) -> float
do math.exp(n float) -> float
do math.exp2(n float) -> float
do math.log(n float) -> float
do math.log2(n float) -> float
do math.log10(n float) -> float
do math.log_base(v float, base float) -> float
do math.sin(r float) -> float
do math.cos(r float) -> float
do math.tan(r float) -> float
do math.asin(n float) -> float
do math.acos(n float) -> float
do math.atan(n float) -> float
do math.atan2(y float, x float) -> float
do math.sinh(n float) -> float
do math.cosh(n float) -> float
do math.tanh(n float) -> float
do math.deg_to_rad(d float) -> float
do math.rad_to_deg(r float) -> float
do math.is_even(n int) -> bool
do math.is_odd(n int) -> bool
do math.is_infinite(n float) -> bool
do math.is_nan(n float) -> bool
do math.is_finite(n float) -> bool
do math.is_prime(n int) -> bool
do math.factorial(n int) -> int
do math.gcd(a int, b int) -> int
do math.lcm(a int, b int) -> int
do math.lerp(a float, b float, t float) -> float
do math.distance(x1 float, y1 float, x2 float, y2 float) -> float
do math.random_int(min int, max int) -> int
do math.random_float(min float, max float) -> float
```

Constants: `math.PI`, `math.E`, `math.TAU`, `math.PHI`, `math.SQRT2`, `math.LN2`, `math.LN10`, `math.INF`, `math.NEG_INF`, `math.EPSILON`

### random

```
do random.rand_float() -> float
do random.rand_float(min float, max float) -> float
do random.rand_int(max int) -> int
do random.rand_int(min int, max int) -> int
do random.rand_bool() -> bool
do random.rand_byte() -> byte
do random.rand_char() -> char
do random.rand_char(min char, max char) -> char
do random.shuffle(arr [any]) -> [any]
do random.sample(arr [any], n int) -> [any]
do random.choice(arr [any]) -> any
do random.random_hex(length int) -> string
do random.seed(value int)
```

### time

```
do time.now() -> int
do time.now_ms() -> int
do time.now_ns() -> int
do time.year(ts int) -> int
do time.month(ts int) -> int
do time.day(ts int) -> int
do time.hour(ts int) -> int
do time.minute(ts int) -> int
do time.second(ts int) -> int
do time.weekday(ts int) -> int
do time.format(fmt string, ts int) -> string
do time.to_iso(ts int) -> string
do time.date(ts int) -> string
do time.to_time(ts int) -> string
do time.tick() -> int
do time.elapsed_ms(start_tick int) -> int
```

### os

```
do os.args() -> [string]
do os.get_env(name string) -> string
do os.set_env(name string, value string)
do os.current_dir() -> string
do os.hostname() -> string
do os.current_os() -> int
do os.arch() -> string
do os.pid() -> int
```

Constants: `os.MAC_OS`, `os.LINUX`, `os.WINDOWS`, `os.OTHER`

### uuid

```
do uuid.generate() -> string
do uuid.generate_hyphenated() -> string
do uuid.is_valid(s string) -> bool
```

### encoding

```
do encoding.base64_encode(s string) -> string
do encoding.base64_decode(s string) -> string
do encoding.hex_encode(s string) -> string
do encoding.hex_decode(s string) -> string
do encoding.url_encode(s string) -> string
do encoding.url_decode(s string) -> string
```

### crypto

```
do crypto.sha256(data string) -> string
do crypto.md5(data string) -> string
do crypto.random_hex(length int) -> string
```

### bytes

```
do bytes.from_string(s string) -> [byte]
do bytes.to_string(b [byte]) -> string
do bytes.from_hex(hex string) -> [byte]
do bytes.to_hex(b [byte]) -> string
do bytes.from_base64(b64 string) -> [byte]
do bytes.to_base64(b [byte]) -> string
```

### binary

```
do binary.encode_i8(val int) -> [byte]
do binary.decode_i8(b [byte]) -> int
do binary.encode_u8(val int) -> [byte]
do binary.decode_u8(b [byte]) -> int
do binary.encode_i16_le(val int) -> [byte]
do binary.decode_i16_le(b [byte]) -> int
do binary.encode_i16_be(val int) -> [byte]
do binary.decode_i16_be(b [byte]) -> int
do binary.encode_u16_le(val int) -> [byte]
do binary.decode_u16_le(b [byte]) -> int
do binary.encode_u16_be(val int) -> [byte]
do binary.decode_u16_be(b [byte]) -> int
do binary.encode_i32_le(val int) -> [byte]
do binary.decode_i32_le(b [byte]) -> int
do binary.encode_i32_be(val int) -> [byte]
do binary.decode_i32_be(b [byte]) -> int
do binary.encode_u32_le(val int) -> [byte]
do binary.decode_u32_le(b [byte]) -> int
do binary.encode_u32_be(val int) -> [byte]
do binary.decode_u32_be(b [byte]) -> int
do binary.encode_i64_le(val int) -> [byte]
do binary.decode_i64_le(b [byte]) -> int
do binary.encode_i64_be(val int) -> [byte]
do binary.decode_i64_be(b [byte]) -> int
do binary.encode_u64_le(val int) -> [byte]
do binary.decode_u64_le(b [byte]) -> int
do binary.encode_u64_be(val int) -> [byte]
do binary.decode_u64_be(b [byte]) -> int
do binary.encode_f32_le(val float) -> [byte]
do binary.decode_f32_le(b [byte]) -> float
do binary.encode_f32_be(val float) -> [byte]
do binary.decode_f32_be(b [byte]) -> float
do binary.encode_f64_le(val float) -> [byte]
do binary.decode_f64_le(b [byte]) -> float
do binary.encode_f64_be(val float) -> [byte]
do binary.decode_f64_be(b [byte]) -> float
do binary.encode_i128_le(val i128) -> [byte]
do binary.decode_i128_le(b [byte]) -> i128
do binary.encode_i128_be(val i128) -> [byte]
do binary.decode_i128_be(b [byte]) -> i128
do binary.encode_u128_le(val u128) -> [byte]
do binary.decode_u128_le(b [byte]) -> u128
do binary.encode_u128_be(val u128) -> [byte]
do binary.decode_u128_be(b [byte]) -> u128
do binary.encode_i256_le(val i256) -> [byte]
do binary.decode_i256_le(b [byte]) -> i256
do binary.encode_i256_be(val i256) -> [byte]
do binary.decode_i256_be(b [byte]) -> i256
do binary.encode_u256_le(val u256) -> [byte]
do binary.decode_u256_le(b [byte]) -> u256
do binary.encode_u256_be(val u256) -> [byte]
do binary.decode_u256_be(b [byte]) -> u256
```

### csv

```
do csv.parse(data string) -> [[string]]
do csv.decode(data string) -> [[string]]
do csv.encode(data [[string]]) -> string
do csv.format(data [[string]]) -> string
do csv.read_file(path string) -> [[string]]
do csv.write_file(path string, data [[string]]) -> bool
do csv.headers(data [[string]]) -> [string]
```

Error-returning variants: `read_file`, `write_file`

### json

```
do json.decode(text string) -> map[string:string]
do json.parse(text string) -> any
do json.encode(value any) -> string
do json.stringify(value any) -> string
do json.format(value any) -> string
do json.pretty_print(m map, indent int) -> string
do json.is_valid(text string) -> bool
```

Error-returning variant: `decode`

### sqlite

```
do sqlite.open(path string) -> Database
do sqlite.close(db Database)
do sqlite.exec(db Database, sql string) -> bool
do sqlite.query(db Database, sql string) -> [map]
```

Error-returning variants: `open`, `exec`, `query`

### net

```
do net.connect(host string, port int) -> Socket
do net.close(sock Socket)
do net.send(sock Socket, data string) -> int
do net.receive(sock Socket, max_bytes int) -> string
do net.listen(port int) -> Listener
do net.accept(listener Listener) -> Socket
do net.set_timeout(sock Socket, milliseconds int)
do net.resolve(hostname string) -> string
```

Error-returning variants: `connect`, `listen`, `accept`, `send`, `receive`, `resolve`

### http

```
do http.get(url string) -> HttpResponse
do http.post(url string, body string) -> HttpResponse
do http.put(url string, body string) -> HttpResponse
do http.delete(url string) -> HttpResponse
do http.head(url string) -> HttpResponse
do http.patch(url string, body string) -> HttpResponse
```

Error-returning variants: `get`, `post`, `put`, `delete`, `head`, `patch`

`HttpResponse` fields: `.status` (int), `.body` (string), `.headers` (map)

### regex

```
do regex.is_valid(pattern string) -> bool
do regex.is_match(pattern string, text string) -> bool
do regex.find(pattern string, text string) -> string
do regex.find_all(pattern string, text string) -> [string]
do regex.replace(pattern string, text string, replacement string) -> string
do regex.split(pattern string, text string) -> [string]
```

Error-returning variants: `find`, `find_all`, `replace`, `split`

### mem

```
do mem.arena(size int) -> Arena
do mem.destroy(arena Arena)
do mem.reset(arena Arena)
do mem.usage(arena Arena) -> int
do mem.alloc(arena Arena, value any) -> any
do mem.make(arena Arena, Type) -> ptr<Type>
do mem.init(arena Arena, Type) -> ptr<Type>
do mem.raw_copy(dest ptr, src ptr, n int)
do mem.zero(p ptr, n int)
do mem.fill(p ptr, val int, n int)
do mem.free(arena Arena)
do mem.size_of(Type) -> int
```

### arrays

```
do arrays.append(arr [any], value any)
do arrays.insert_at(arr [any], index int, value any)
do arrays.prepend(arr [any], value any)
do arrays.remove_at(arr [any], index int)
do arrays.clear(arr [any])
do arrays.fill(arr [any], value any, count int)
do arrays.get_first(arr [any]) -> any
do arrays.get_last(arr [any]) -> any
do arrays.remove_last(arr [any]) -> any
do arrays.remove_first(arr [any]) -> any
do arrays.is_empty(arr [any]) -> bool
do arrays.contains(arr [any], value any) -> bool
do arrays.index_of(arr [any], value any) -> int
do arrays.count(arr [any], value any) -> int
do arrays.reverse(arr [any]) -> [any]
do arrays.slice(arr [any], start int, end int) -> [any]
do arrays.concat(a [any], b [any]) -> [any]
do arrays.deduplicate(arr [any]) -> [any]
do arrays.flatten(arr [[any]]) -> [any]
do arrays.split_every(arr [any], size int) -> [[any]]
do arrays.pair(a [any], b [any]) -> [[any]]
do arrays.get_sum(arr [int]) -> int
do arrays.get_min(arr [int]) -> int
do arrays.get_max(arr [int]) -> int
do arrays.sort_asc(arr [any])
do arrays.sort_desc(arr [any])
```

### maps

```
do maps.get_keys(m map) -> [any]
do maps.get_values(m map) -> [any]
do maps.has_key(m map, key any) -> bool
do maps.is_empty(m map) -> bool
do maps.merge(m1 map, m2 map) -> map
do maps.contains_value(m map, value any) -> bool
do maps.get_or_default(m map, key any, default any) -> any
do maps.remove_key(m map, key any)
do maps.clear(m map)
```

### threads

```
do threads.spawn(func_ref func) -> Thread
do threads.spawn(func_ref func, arg int) -> Thread
do threads.join(t Thread)
do threads.get_id() -> int
```

### sync

```
do sync.mutex() -> Mutex
do sync.lock(m Mutex)
do sync.unlock(m Mutex)
do sync.destroy(m Mutex)
do sync.try_lock(m Mutex)
```

### channels

```
do channels.open(capacity int) -> Channel
do channels.send(ch Channel, value int)
do channels.receive(ch Channel) -> int
do channels.close(ch Channel)
```

### atomic

```
do atomic.load(p ptr<int>) -> int
do atomic.store(p ptr<int>, val int)
do atomic.add(p ptr<int>, val int) -> int
do atomic.sub(p ptr<int>, val int) -> int
do atomic.exchange(p ptr<int>, val int) -> int
do atomic.cas(p ptr<int>, expected int, desired int) -> bool
do atomic.and(p ptr<int>, val int) -> int
do atomic.or(p ptr<int>, val int) -> int
do atomic.xor(p ptr<int>, val int) -> int
do atomic.spinlock() -> SpinLock
do atomic.spin_lock(lk SpinLock)
do atomic.spin_trylock(lk SpinLock) -> bool
do atomic.spin_unlock(lk SpinLock)
do atomic.fence()
```

### server

```
do server.add_router() -> Router
do server.add_route(router Router, method string, pattern string, handler func)
do server.listen(port int, router Router)
do server.text(status int, body string) -> HttpResponse
do server.json(status int, body string) -> HttpResponse
do server.html(status int, body string) -> HttpResponse
do server.redirect(status int, location string) -> HttpResponse
do server.cors(router Router)
do server.use(router Router, middleware func)
```

---

## Learn More

- [Official documentation](https://schoolyb.github.io/EZ-Language-Webapp/docs)
- [Contributing guide](CONTRIBUTING.md)

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
