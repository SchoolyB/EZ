# EZ 3.0 — What's New

Reference document for updating official EZ documentation. Covers all language changes, new features, removals, and EZC compiler additions in version 3.0.

---

## Breaking Changes

### `temp` renamed to `mut`

The keyword for mutable variable declarations changed from `temp` to `mut`.

```ez
// Before (2.x)
temp x int = 42
temp name string = "Alice"

// After (3.0)
mut x int = 42
mut name string = "Alice"
```

`const` is unchanged. The pairing is now `const` / `mut`.

### Module declarations removed

The `module` keyword is no longer used. Module identity comes from the filesystem.

```ez
// Before (2.x) — required at top of file
module helpers

do greet() -> string { return "hi" }

// After (3.0) — no declaration needed
do greet() -> string { return "hi" }
```

Existing files with `module` declarations still parse (silently ignored), but they have no effect. Module name is determined by the filename or directory name.

### Server module API changed

`server.route()` now takes a function reference handler instead of a static Response.

```ez
// Before (2.x)
server.route(router, "GET", "/", server.text(200, "Hello"))

// After (3.0)
do home(req Request) -> Response {
    return server.text(200, "Hello")
}
server.route(router, "GET", "/", ()home)
```

---

## New Keywords

### `mut` (replaces `temp`)

```ez
mut count int = 0
mut name = "Alice"    // type inference works
```

### `while` (alias for `as_long_as`)

Both are valid. User's choice.

```ez
while count < 10 { count++ }
as_long_as count < 10 { count++ }  // still works
```

### `or_return`

Error propagation shorthand. If a function call returns a non-nil error, automatically returns from the enclosing function.

```ez
// Before — 4 lines of boilerplate every time
mut content, err = read_file("data.txt")
if err != nil {
    return "", err
}

// After — one line
mut content = read_file("data.txt") or_return
```

Supports custom fallback values:
```ez
mut content = read_file("data.txt") or_return "", error("failed to load")
```

---

## New Language Features

### Function References — `()func_name`

Pass named functions as values using the `()` prefix. No anonymous functions, no lambdas.

```ez
do is_positive(n int) -> bool {
    return n > 0
}

// Store in a variable
mut check = ()is_positive
check(5)   // true

// Pass as argument
mut positives = filter(numbers, ()is_positive)

// Works with struct-namespaced functions
mut result = apply(data, ()MathUtils.square)
```

The `()` prefix means "reference this function, don't call it." The receiving function calls it when ready.

When declaring a parameter that accepts a function reference, use `func` as the type:

```ez
do filter(arr [int], test func) -> [int] {
    mut result [int] = {}
    for_each item in arr {
        if test(item) {
            arrays.append(result, item)
        }
    }
    return result
}
```

### Struct-Namespaced Functions

Functions declared inside struct blocks. No self/this — every parameter is explicit.

```ez
const Point struct {
    x int
    y int

    do create(x int, y int) -> Point {
        return Point{x: x, y: y}
    }

    do distance(a Point, b Point) -> float {
        return math.sqrt(math.pow(float(a.x - b.x), 2) + math.pow(float(a.y - b.y), 2))
    }

    private do validate(p Point) -> bool {
        return p.x >= 0 && p.y >= 0
    }
}

// Called as Type.func()
mut p1 = Point.create(3, 4)
mut d = Point.distance(p1, p2)

// Cross-module: module.Type.func()
mut v = geometry.Vec2.create(1.0, 2.0)
```

Rules:
- No `self`, no `this` — all parameters explicit
- `private` restricts to same-struct access
- Called as `StructName.func_name(args...)`

### Map Iteration via `for_each`

```ez
mut ages map[string:int] = {"alice": 30, "bob": 25}

// Two variables: key, value
for_each k, v in ages {
    println("${k}: ${v}")
}

// One variable: keys only
for_each key in ages {
    println(key)
}

// Blank identifier for values only
for_each _, v in ages {
    total += v
}
```

Iteration order is undefined (maps are unordered).

### Lifted `when`/`is` Restrictions

`when` statements now accept bools, nil, and floats as conditions.

```ez
// Bools — previously banned
when flag {
    is true { println("yes") }
    is false { println("no") }
}

// Nil — previously banned
when value {
    is nil { println("empty") }
    default { println("has value") }
}

// Floats — allowed with W2012 warning about imprecision
when temperature {
    is 98.6 { println("normal") }
    default { println("abnormal") }
}
```

Collections (arrays/maps) as `when` conditions remain banned.

### Full Literal Type Inference

Type annotations are optional for all literal types:

```ez
mut x = 42          // inferred: int
mut f = 3.14        // inferred: float
mut s = "hello"     // inferred: string
mut b = true        // inferred: bool
mut c = 'A'         // inferred: char

// Explicit annotation still available for non-default types
mut y uint = 42
mut b byte = 255
```

---

## Module System Overhaul

### Import Rules

| Syntax | Meaning |
|--------|---------|
| `import @std` | Stdlib module, alias is `std` |
| `import "./foo.ez"` | Single file, alias is `foo` |
| `import "./dir"` | All `.ez` files in dir merge, alias is `dir` |
| `import myalias @std` | Explicit alias |
| `import mymod "./path"` | Explicit alias for local module |
| `import & use @std` | Import + bring into scope |

### Key Rules

- **Collision = error.** Two imports resolving to the same alias without aliasing is an error.
- **Directory imports are top-level only.** `import "./utils"` merges `.ez` files directly in `utils/`. Subdirectories are their own modules.
- **Double-import = error.** Can't import a file individually AND its parent directory.
- **`module` declarations are ignored.** Kept for backward compat but have no effect.
- **No `module` declaration needed.** Files in a directory automatically merge.

### Private Declarations

`private` on functions, constants, structs, etc. keeps them internal to the module. Works the same as before — just no `module` declaration needed.

```ez
// utils/helpers.ez — no module declaration
private do internal_helper() -> int { return 42 }
do public_api() -> int { return internal_helper() }
```

---

## Server Module — Full API

The `@server` module was redesigned with dynamic handlers.

### Response Builders

```ez
server.text(200, "Hello")                    // text/plain
server.json(200, {"status": "ok"})           // application/json
server.html(200, "<h1>Hello</h1>")           // text/html
server.redirect(302, "/new-location")        // redirect
server.set_header(response, "key", "value")  // add header
```

### Request Object

Every handler receives a `Request` struct:

| Field | Type | Description |
|-------|------|-------------|
| `req.method` | `string` | HTTP method (GET, POST, etc.) |
| `req.path` | `string` | Request path |
| `req.query` | `map[string:string]` | Query parameters |
| `req.headers` | `map[string:string]` | Request headers |
| `req.params` | `map[string:string]` | Path parameters (from `:param`) |
| `req.body` | `string` | Raw request body |

### Path Parameters

```ez
server.route(router, "GET", "/users/:id", ()get_user)

do get_user(req Request) -> Response {
    mut id = req.params["id"]
    return server.json(200, {"id": id})
}
// GET /users/42 → req.params["id"] = "42"
```

### Middleware

```ez
do logger(req Request) -> nil {
    println("[${req.method}] ${req.path}")
    return nil  // continue to next handler
}

do auth(req Request) -> Response {
    if req.headers["authorization"] == "" {
        return server.json(401, {"error": "unauthorized"})
    }
    return nil  // continue
}

server.use(router, ()logger)
server.use(router, ()auth)
```

Return `nil` to continue. Return a `Response` to short-circuit.

### Other Features

```ez
server.cors(router, "*")                              // CORS
server.static(router, "/assets", "./public")          // static files
server.parse_json(req)                                // parse body as JSON
```

### Complete Example

```ez
import @std, @server
using std

do home(req Request) -> Response {
    return server.html(200, "<h1>Welcome</h1>")
}

do get_user(req Request) -> Response {
    return server.json(200, {"id": req.params["id"]})
}

do main() {
    mut router = server.router()
    server.cors(router, "*")
    server.route(router, "GET", "/", ()home)
    server.route(router, "GET", "/users/:id", ()get_user)
    server.listen(8080, router)
}
```

---

## Bug Fixes

### Dangling `ref()` — Runtime Error E5025

Accessing a reference whose target variable went out of scope now emits a clear runtime error instead of silently returning nil.

### Module File Loading Order

`os.ReadDir()` returns files in filesystem order which varies by OS. Fixed by sorting file list before processing. Eliminates intermittent "function not found" bugs in multi-file modules.

### Regex Test — String/Nil Comparison

Fixed `string == nil` comparison that the typechecker correctly blocks. Test now checks the error value instead.

---

## EZC Compiler — New Features

### Pointers (`^Type`)

| Operation | C | EZ (EZC) |
|-----------|---|----------|
| Declare pointer | `int *p;` | `mut p ^int` |
| Get address | `p = &x;` | `p = addr(x)` |
| Dereference (read) | `*p` | `p^` |
| Dereference (write) | `*p = 42;` | `p^ = 42` |
| Null pointer | `NULL` | `nil` |
| Null check | `if (p == NULL)` | `if p == nil` |
| Heap allocate | `malloc(sizeof(int))` | `mem.new(arena, int)` |
| Free memory | `free(p)` | `mem.destroy(arena)` |
| Struct field via pointer | `p->x` | `p^.x` |

### `@threads` Module (EZC-only)

Threading primitives built on POSIX pthreads.

```ez
import @threads

// Spawn threads
mut t1 = threads.spawn(()worker, 1)
mut t2 = threads.spawn(()worker, 2)
threads.join(t1)
threads.join(t2)

// Mutex
mut mtx = threads.mutex()
threads.lock(mtx)
// critical section
threads.unlock(mtx)

// Channels
mut ch = threads.channel(8)
threads.send(ch, 42)
mut val = threads.recv(ch)
threads.close(ch)

// Sleep
threads.sleep_ms(100)
```

| Function | Description |
|----------|-------------|
| `threads.spawn(()func)` | Spawn thread (no args) |
| `threads.spawn(()func, arg)` | Spawn thread with int arg |
| `threads.join(handle)` | Wait for thread |
| `threads.sleep_ms(ms)` | Sleep current thread |
| `threads.id()` | Current thread ID |
| `threads.mutex()` | Create mutex |
| `threads.lock(m)` / `threads.unlock(m)` | Lock/unlock |
| `threads.channel(capacity)` | Create bounded channel |
| `threads.send(ch, value)` | Send (blocks if full) |
| `threads.recv(ch)` | Receive (blocks if empty) |
| `threads.close(ch)` | Close channel |

### `@sqlite` Module (EZC-only, replaces `@db`)

```ez
import @sqlite

mut db = sqlite.open("app.db")
sqlite.exec(db, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
sqlite.exec(db, "INSERT INTO users (name) VALUES ('Alice')")
mut rows = sqlite.query(db, "SELECT * FROM users")
sqlite.close(db)
```

### Fixed-Size Arrays

```ez
const arr [int, 3] = {10, 20, 30}
```

### Multi-Dimensional Arrays

```ez
mut matrix [[int]] = {{1, 2, 3}, {4, 5, 6}}
mut row = matrix[0]
println(row[2])  // 3
```

### `@mem` Module (EZC-only)

Arena-based memory management for explicit control.

```ez
import @mem

mut arena = mem.arena(4096)
ensure mem.destroy(arena)

mut p ^int = mem.new(arena, int)
p^ = 42
println(mem.usage(arena))  // bytes used
mem.reset(arena)           // reuse without freeing
```

---

## New Error Codes

| Code | Name | Description |
|------|------|-------------|
| E5025 | dangling-reference | Reference target no longer exists |
| E6010 | import-alias-collision | Two imports resolve to same alias |
| E6011 | double-import | File already included via directory import |
| W2012 | float-when-imprecise | Float equality in `when` may be imprecise |

---

## Removed / Deprecated

| Item | Status |
|------|--------|
| `temp` keyword | **Replaced** by `mut` |
| `module` declarations | **Removed** — silently ignored if present |
| E6005 (module-name-mismatch) | **Removed** — no longer applicable |
| E6006 (module-name-conflict) | **Removed** — no longer applicable |
| W4001 (module-name-mismatch) | **Removed** — no longer applicable |
| E2044 (when-float-not-allowed) | **Removed** — floats now allowed in `when` |
| E2048 (when-bool-condition) | **Removed** — bools now allowed in `when` |
| E2049 (when-nil-condition) | **Removed** — nil now allowed in `when` |
| `@db` module (interpreter) | **Being replaced** by `@sqlite` |
| Static `server.route()` API | **Replaced** by function reference handlers |
