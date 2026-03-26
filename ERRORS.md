# EZ Error Code Reference

> Auto-generated from `ezc/src/util/error_codes.h`. Do not edit manually.
> Run `./scripts/generate_errors.sh` to regenerate.

**Total: 29 codes** (27 errors, 2 warnings)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
| `E1010` | lexer | integer literal overflows 64-bit integer |
| `E2001` | parser | unexpected token in expression or statement |
| `E2002` | parser | expected token not found (missing brace, paren, etc.) |
| `E2050` | parser | break or continue used outside of a loop |
| `E2051` | parser | nested function declarations are not allowed |
| `E3001` | type | type mismatch (assignment, argument, return, comparison, struct field) |
| `E3002` | type | invalid operator on type (string arithmetic, ordering) |
| `E3005` | type | cannot modify a constant (const reassignment, index, field, ++) |
| `E3006` | type | return value issue (value in void function, bare return in non-void) |
| `E3007` | type | operator on incompatible type (++/-- or negation on non-numeric) |
| `E3008` | type | indexing a non-indexable type (only arrays, maps, strings) |
| `E3009` | type | for_each on non-iterable type (only arrays, maps, strings) |
| `E3010` | type | struct has no such field (literal or access) |
| `E3011` | type | type keyword used as a value (mut x = int) |
| `E3012` | type | addr() requires an lvalue (variable, field, or index) |
| `E3013` | type | field access on non-struct type |
| `E3016` | type | cannot dereference non-pointer type |
| `E3018` | type | cannot use module before importing it |
| `E4001` | reference | undefined variable |
| `E4002` | reference | undefined function |
| `E4003` | reference | variable already declared in this scope |
| `E4004` | reference | function already declared |
| `E4005` | reference | program has no main() function |
| `E4006` | reference | name uses reserved compiler prefix (ez_, _ez_, Ez) |
| `E5008` | runtime | wrong number of function arguments |
| `E6001` | import | unknown standard library module |
| `E7006` | stdlib | threads.spawn() requires a function reference |

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
| `W2001` | unused | imported module is not used |
| `W3001` | type | function may not return a value on all paths |

---

## Error Code Ranges

| Range | Category |
|-------|----------|
| E1xxx | Lexer errors |
| E2xxx | Parser errors |
| E3xxx | Type errors |
| E4xxx | Reference/name errors |
| E5xxx | Runtime errors |
| E6xxx | Import errors |
| E7xxx | Stdlib validation errors |
| W2xxx | Unused code warnings |
| W3xxx | Type warnings |

---

*Generated on 2026-03-26 02:50:16 UTC*
