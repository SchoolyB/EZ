# EZ Memory Model Proposal: Scope-Based Automatic Memory Management

*Drafted: 2026-04-17 | Target: v3.0.0 GA | Status: Proposal*

---

## Overview

EZ replaces its current single-global-arena memory model with **scope-based automatic memory management**. Memory allocated inside a block of code is freed when that block ends. If a value needs to survive (because it's returned, or stored into an outer variable), EZ moves it automatically.

The user writes normal code. Memory just works. No new keywords, no annotations, no concepts to learn.

For power users who want manual control, the `@mem` module remains available as an opt-in tool — not a requirement.

---

## Core Principle

**One rule:** when a block of code ends, any memory it created gets cleaned up. If something needs to survive, EZ handles it.

---

## How It Works

### Default behavior — automatic cleanup

Every scope (function body, loop iteration, if/otherwise block) owns the memory it creates. When the scope ends, that memory is freed:

```ez
do process(name string) {
    mut upper = strings.to_upper(name)    // created here
    mut parts = strings.split(upper, ",") // created here
    println(parts[0])
}
// function ends -> upper and parts are freed
```

The user doesn't do anything. No imports, no annotations, no cleanup calls.

### Automatic survival — values that escape their scope

When a value is stored somewhere that outlives the current scope, EZ keeps it alive by moving it to the outer scope's memory:

```ez
mut results [string] = {}

for_each line in lines {
    mut upper = strings.to_upper(line)
    arrays.append(results, upper)         // upper goes into results, which lives outside the loop
}
// upper survives each iteration because it was stored in results
// results lives until main ends
```

Three cases where EZ keeps values alive:

1. **Returning a value** — ownership transfers to the caller
2. **Storing into an outer-scope container** — array append, map insert, struct field assignment
3. **Assigning to a variable declared in an outer scope** — `outer_var = expr`

Everything else is freed when the block ends.

### Manual control — `@mem` module (opt-in)

For the rare case where a user wants explicit control over memory (pre-allocating buffers, managing pools, bulk allocation), the `@mem` module remains available:

```ez
import @mem

mut scratch = mem.arena(4096)
mut node = mem.init(scratch, Node)
// ... use node ...
mem.destroy(scratch)
```

Most users never import `@mem`. It's a power tool, not a requirement.

---

## What This Replaces

### Current model (v3.0-alpha)

- One global arena created at program start (1MB, grows as needed)
- Every string, array, map, and `new()` allocation goes into this arena
- Nothing is freed until the program exits
- Works fine for short-lived CLI tools
- Long-running programs (servers, daemons, data pipelines) grow forever

### Proposed model

- Each scope gets its own memory region
- Allocations are freed at scope exit
- Values that escape are moved automatically
- Long-running programs stay memory-flat
- Short-lived programs work identically to today — no behavior change

---

## Safety Features

The scope model provides several safety guarantees automatically, with additional compile-time checks layered on top.

### Built-in safety (free with the scope model)

These safety properties come directly from the scope model's design — no extra implementation work beyond the model itself:

| Hazard | How the scope model prevents it |
|--------|-------------------------------|
| **Memory leaks** | Scopes own their allocations and free them on exit. No accumulation over time. |
| **Use-after-free (common case)** | If a value is in scope, its memory is alive. If it's out of scope, you can't name it. Can't access what you can't name. |
| **Dangling returns (common case)** | Returning a value transfers ownership to the caller. The data moves, the pointer stays valid. |
| **Loop accumulation** | Each iteration is a scope. Iteration ends, memory is reclaimed. Next iteration starts clean. |
| **Server/daemon growth** | Each request handler is a scope. Request ends, memory is reclaimed. Process stays flat. |

### Compile-time safety checks (added on top)

These are typechecker rules that catch memory bugs before the program runs. Zero runtime cost.

#### Check 1: Reject `addr()` of locals in return or outer-scope assignment

**Target: 3.0 GA**

```ez
do bad() -> ^int {
    mut x int = 42
    return addr(x)    // COMPILE ERROR: cannot return address of local variable
}
```

The compiler knows `x` is local. The compiler knows `return` sends the pointer outside this scope. The scope model will free `x` when `bad()` returns, making the pointer dangle. Reject it.

**Rule:** `addr()` of a local variable cannot appear in a `return` statement or be assigned to a variable declared in an outer scope. One check in the typechecker. No annotations.

#### Check 2: Warn on cross-scope pointer assignments

**Target: 3.0 GA**

```ez
mut p ^Node = nil
if some_condition {
    mut n = new(Node)
    p = addr(n)           // WARNING: pointer may reference memory from a scope that has ended
}
println(p^.val)            // n was freed when the if-block ended — p dangles
```

The compiler sees that `p` was assigned from something inside a narrower scope. The inner scope will be cleaned up before `p` is used. Warn the user.

**Rule:** when a pointer-typed variable in scope A is assigned from `addr()` of a value in scope B, and B is nested inside A, emit a warning. Covers the 80% case without full lifetime analysis.

#### Check 3: Non-nullable pointers by default

**Target: 3.0 GA**

Make `^T` non-nullable. Nullable pointers use an explicit syntax (e.g., `?^T`):

```ez
mut p ^Node = new(Node)    // OK — new() always returns non-nil
mut q ^Node = nil           // COMPILE ERROR — ^Node is not nullable
mut r ?^Node = nil          // OK — explicitly nullable

println(p^.val)             // safe — compiler knows p is non-nil

if r != nil {
    println(r^.val)         // OK — compiler knows r is non-nil inside this block
}
```

Eliminates the entire class of nil-dereference panics at compile time for non-nullable pointers. Users who actually need nullable pointers opt in with `?^T`.

#### Check 4: Double-free detection on `@mem` arenas

**Target: 3.0 GA (already implemented)**

```ez
import @mem
mut a = mem.arena(1024)
mem.destroy(a)
mem.destroy(a)    // RUNTIME PANIC: arena already destroyed
```

The arena struct carries a `destroyed` flag. Second destroy panics with a clear message. Already implemented in `ez_runtime.c`.

### Runtime safety checks (unchanged from today)

These existing checks remain in place:

| Hazard | Behavior |
|--------|----------|
| Nil pointer dereference | Runtime panic |
| Array out of bounds | Runtime panic |
| Map key not found | Runtime panic |
| Division by zero | Runtime panic (int and float) |
| Integer overflow | Runtime panic (checked arithmetic) |
| Stack overflow | Detected and reported |

### Not checked (programmer responsibility)

| Hazard | Mitigation |
|--------|-----------|
| Data races | Use `sync.lock()`. EZ does not detect races. |
| `@mem` use-after-free | Manual arena lifetime is the user's responsibility when opting into `@mem`. |
| Pointer arithmetic | Not supported in the language (disallowed by design). |

---

## Safety Comparison

| Hazard | C | EZ (today) | EZ (proposed) | Go | Rust |
|--------|---|------------|---------------|-----|------|
| Memory leaks | Manual free | Leaks until exit | Scope cleanup | GC | Ownership |
| Use-after-free | Silent corruption | Possible (arena) | Mostly eliminated | GC prevents | Borrow checker prevents |
| Dangling pointers | Silent corruption | Possible | Compile-time rejection | GC prevents | Borrow checker prevents |
| Nil deref | Silent corruption | Runtime panic | Compile-time (3.1) | Runtime panic | No null type |
| Bounds checking | None (UB) | Runtime panic | Runtime panic | Runtime panic | Runtime panic |
| Double free | Silent corruption | Runtime panic | Runtime panic | GC prevents | Compile-time |
| Data races | Silent corruption | Silent corruption | Silent corruption | Race detector (opt-in) | Compile-time |
| Integer overflow | Silent wrap | Runtime panic | Runtime panic | Silent wrap | Panic (debug) / wrap (release) |

**EZ's position:** safer than C on every axis. Comparable to Go on most axes (minus GC's use-after-free protection, plus overflow checking that Go lacks). Behind Rust on data races and lifetime analysis, but without any of Rust's syntactic complexity.

---

## Implementation Roadmap

### 3.0 GA

| Item | Type | Scope |
|------|------|-------|
| Scope-based memory model | Core runtime change | Replace global arena with per-scope allocation regions in codegen |
| Automatic survival on return | Codegen | Detect return of scope-local value, copy to caller's region |
| Automatic survival on outer-scope assignment | Codegen | Detect store into outer-scope container/variable, copy to that scope's region |
| `addr()` of locals rejected in return/outer-assign | Typechecker | One check, new error code |
| Cross-scope pointer assignment warning | Typechecker | Warn when pointer assigned from narrower scope |
| Non-nullable pointers (`^T` non-null, `?^T` nullable) | Typechecker + syntax | New nullable pointer syntax, null-check narrowing in if-blocks |
| Double-free detection on `@mem` arenas | Runtime + typechecker | Compile-time for straight-line cases, runtime for conditional/cross-function cases. Already partially implemented |
| `@mem` module remains as opt-in power tool | No change | Existing module, existing API |

---

## What This Is NOT

- **Not a garbage collector.** No tracing, no runtime thread, no pauses.
- **Not reference counting.** No counters on objects, no retain/release overhead.
- **Not a borrow checker.** No lifetime annotations, no ownership types, no fighting the compiler.
- **Not manual memory management.** No malloc, no free, no thinking about memory for normal code.

It is **scope-based automatic cleanup with mechanical escape detection**, compiled to C, with zero user-facing syntax. The closest analog is C++ RAII without destructors or move semantics — but simpler, because the compiler handles ownership transfer automatically based on three mechanical rules (return, outer-scope assignment, container insertion).

---

## The Pitch

"EZ cleans up memory automatically when your code is done with it. Dangling pointers and null dereferences are caught at compile time. If you want manual control, it's there. You never have to think about memory unless you choose to."
