# ez-stdlib.md  
### *The Standard Library of the EZ Programming Language*

---

## 1. Purpose and Philosophy

The EZ standard library exists to give programmers **enough power to do real work**
without bloating the language or hiding behavior.

It follows the same core principles as the language:

- **Beginner-first:**  
  Functions must be obvious, explicit, and predictable. A new programmer should be able
  to guess what a function does from the name and a simple example.

- **High skill ceiling:**  
  Advanced users can chain functions, use more powerful modules (@net, @crypto, @sys),
  and write serious scripts and tools.

- **No magic:**  
  No implicit conversions, no hidden side-effects beyond what the module name suggests,
  no “surprise” allocations or global state.

- **Consistency:**  
  Naming, parameter order, return patterns, and error behavior should feel the same
  across all modules.

- **Small but complete:**  
  The stdlib should cover the most common needs (I/O, collections, time, randomness,
  JSON, basic OS + networking) without turning into a kitchen sink.

All stdlib design must pass the two core questions:

1. Does this make it easier for a beginner to start?  
2. Does this give an expert more power without confusing beginners?

If not, it doesn’t belong.

---

## 2. Modules Overview

### 2.1 Current & Planned Modules

| Module      | Status         | Category   | Purpose |
|-------------|----------------|-----------|---------|
| `@std`      | Implemented    | Core      | Basic printing and standard utilities |
| `@strings`  | Implemented    | Core/Data | String formatting and manipulation |
| `@math`     | Implemented    | Core      | Math functions and constants |
| `@arrays`   | Implemented    | Data      | Array operations (append, pop, etc.) |
| `@maps`     | Implemented    | Data      | Map/dictionary operations (get, set, keys, values, etc.) |
| `@time`     | Implemented    | System    | Time, dates, sleeping, timestamps |
| `@sets`     | Issue Created  | Data      | Set helpers once set types exist |
| `@io`       | Not Implemented | System   | Basic I/O streams (stdin/stdout/stderr) |
| `@fs`       | Not Implemented | System   | Filesystem operations (files/dirs/paths) |
| `@os`       | Not Implemented | System   | OS/environment utilities |
| `@sys`      | Not Implemented | System   | Interpreter/runtime/system info (advanced) |
| `@json`     | Not Implemented | Utility  | JSON encode/decode |
| `@random`   | Not Implemented | Utility  | Random numbers, choices, shuffling |
| `@crypto`   | Not Implemented | Utility  | Simple hashing and crypto helpers |
| `@net`      | Not Implemented | Advanced | Networking (HTTP/basic sockets) |

---

## 3. Global Stdlib Design Rules

### 3.1 Import Style

Canonical form (beginner-friendly):

```ez
import @arrays
import @strings
import @math
```

Advanced form with optional alaising (for power users):

```
import arr@arrays, str@strings, @math
```

Docs and examples must always show the canonical form first.
Aliases are optional and never required to understand or use a module.


### 3.2 Function Naming
- Names must be short, clear and action-based:
    - `append()`, `pop()`, `shift()`, `unshift()`
    - `read_file()`, `write_file()`
    - `now()`, `sleep()`
    - `encode_json()`, `decode_json()`, `marshal()`, `unmarshal()`
- No cute or "clever" names.
- Avoid overload meanings. If a name can be misread, pick a better one.

---

### 3.3 Parameter and Return Conventions

Stdlib functions fall into three main categories with different return rules:

1. **Pure / always-valid functions**  
   Examples: `math.abs()`, `strings.upper()`, `arrays.length()`

   - These return a single value.
   - They do not return errors or status flags.
   - If something goes wrong, it is a programming/runtime error handled by EZ’s normal error system.

2. **Collection operations (`@arrays`, later `@maps`, `@sets`)**

   - The simple canonical functions (e.g. `append()`, `pop()`, `shift()`) may:
     - mutate in place and return nothing, or
     - return a useful value (like the popped element or new length).
   - If used incorrectly (e.g. `pop` on an empty array), they are allowed to raise a
     clear, beginner-friendly runtime error.

   For advanced/safe usage, each risky operation may also have a `try_` variant:
   - `arrays.pop(arr)` → returns value or errors on empty.
   - `arrays.try_pop(arr)` → returns `(value, ok bool)` and never errors.

   This keeps the beginner path simple while giving experts explicit control.

3. **External / environment-dependent functions (`@fs`, `@io`, `@net`, `@json`, `@crypto`, `@os`)**

   - These must never silently fail.
   - They should use explicit status in their return types, e.g.:

     - `(result, ok bool)` for pre-error-type EZ, or  
     - `(result, err error)` once the language has a proper `error` type.

   Examples:
   - `fs.read_file(path string) (string, bool)`
   - `json.decode(text string) (Value, bool)`
   - `net.http_get(url string) (Response, bool)`

In all cases, stdlib APIs must:
- avoid hidden global state,
- avoid silent failures,
- use patterns that are easy to teach to beginners and powerful for advanced users.

---

### 3.4 Error Behavior

- Errors from stdlib must:
  - Explain what wnet wrong
  - Include relevant values, files, paths when safe
  - Be consistent with EZ's global error style
- No "hidden" expectations or global error states. If something fails, it is visible

---

### 3.5 Dependencies and Layering
- Core modules (@std, @math, @strings, @arrays, @time) should be as independent as possible.
- Higher-level modules (@net, @os, etc. )may depend on core modules, but not vice versa.
- Circular dependencies between stdlib modules are not allowed.

---

## 4. Conclusion

### 4.1 Standard Library Expansion Rules
Before adding a new function or module to the stdlib, ask:

- Does this clearly help a common use case?
- Can a beginner understand it with one example and a short explanation?
- Does it stay explicit and predictable?
- Is this better as a user built library than core stdlib?
If it fails any of those, it doesn’t go in the standard library.

--- 

### 4.2 Summary
The EZ standard library is:
- small but capable
- beginner-friendly but powerful
- predictable, explicit, and honest.
It should help programmers write real code quickly, while still reinforcing the core
philosophy of EZ: low barrier to entry, high skill ceiling, no bullshit.