# EZ Error Code Reference

> Auto-generated from `ezc/src/util/error_codes.h`. Do not edit manually.
> Run `./scripts/generate_errors.sh` to regenerate.

**Total: 54 codes** (49 errors, 5 warnings)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
| `E1003` | syntax | multi-line comment was never closed — add */ to close it |
| `E1004` | syntax | string literal was never closed — add a closing quote |
| `E1005` | syntax | character literal was never closed — add a closing single quote |
| `E1006` | syntax | invalid escape sequence in string — valid escapes are \\n \\t \\\\ \\\ |
| `E1010` | syntax | number is too large — the maximum integer value is 9223372036854775807 |
| `E2001` | syntax | unexpected symbol — the compiler found something it did not expect here |
| `E2002` | syntax | missing symbol — a bracket, parenthesis, or keyword is missing |
| `E2008` | syntax | invalid assignment target — you can only assign to variables, fields, or array elements |
| `E2011` | syntax | constants must have a value — add = followed by a value after the type |
| `E2012` | syntax | duplicate parameter name — each parameter must have a unique name |
| `E2013` | syntax | duplicate struct field name — each field must have a unique name |
| `E2016` | syntax | empty enum — an enum must have at least one value |
| `E2036` | syntax | imports must be at the top of the file, not inside a function |
| `E2043` | syntax | duplicate case value in when statement — each case must be unique |
| `E2050` | syntax | break and continue can only be used inside a loop |
| `E2051` | syntax | functions cannot be defined inside other functions — move it to the top level |
| `E2053` | syntax | structs and enums must be defined at the top level, not inside a function |
| `E3001` | types | wrong type — you are using a value of one type where a different type is expected |
| `E3002` | types | this operator does not work on this type — for example, you cannot subtract strings |
| `E3005` | types | cannot change a constant — use mut instead of const if you need to modify this value |
| `E3006` | types | return value problem — either returning a value from a function that should not return one, or missing a return value |
| `E3007` | types | ++ and -- only work on numbers — you cannot increment or negate a non-number |
| `E3008` | types | cannot use [] on this type — only arrays, maps, and strings support indexing |
| `E3009` | types | cannot loop over this type — for_each only works with arrays, maps, and strings |
| `E3010` | types | this struct does not have a field with that name |
| `E3011` | types | a type name like int or string cannot be used as a value — did you forget to declare a variable? |
| `E3012` | types | addr() needs a variable, field, or array element — you cannot take the address of a literal like 42 |
| `E3013` | types | only structs have fields — you cannot use .field on a number, string, or bool |
| `E3015` | types | this value is not a function and cannot be called |
| `E3016` | types | only pointers can be dereferenced with ^ — this value is not a pointer |
| `E3024` | types | this function must return a value but the body has no return statement |
| `E3027` | types | cannot pass a constant to a mutable parameter — the function wants to modify this value |
| `E3039` | types | ensure expects a function call — for example: ensure close(file) |
| `E3040` | types | this function returns multiple values but you are assigning to a single variable — use mut a, b = func() |
| `E4001` | names | this variable does not exist — check the spelling or make sure it is declared above this line |
| `E4002` | names | this function does not exist — check the spelling or make sure it is defined |
| `E4003` | names | a variable with this name already exists in this scope — use a different name |
| `E4004` | names | a function with this name already exists — each function must have a unique name |
| `E4005` | names | no main() function found — every EZ program needs a do main() { } function |
| `E4006` | names | this name is reserved by the compiler — choose a different name that does not start with ez_ or Ez |
| `E5008` | arguments | wrong number of arguments — the function expects a different number of values than you provided |
| `R5001` | runtime | nil pointer dereference — you tried to use a pointer that has no value (nil) |
| `R5002` | runtime | array index out of bounds — you tried to access an element that does not exist |
| `R5003` | runtime | division by zero — you cannot divide a number by zero |
| `R5004` | runtime | key not found in map — the key you are looking for does not exist |
| `R5005` | runtime | stack overflow — too many nested function calls (possible infinite recursion) |
| `R5006` | runtime | assertion failed — a condition checked with assert() was false |
| `E6001` | imports | unknown module — this is not a built-in EZ module. Check the spelling or see the docs for available modules |
| `E7006` | stdlib | threads.spawn() needs a function reference — use ()function_name to pass a function |

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
| `W1001` | cleanup | variable is declared but never used — remove it or use it |
| `W1003` | cleanup | function is declared but never called — remove it or call it |
| `W2001` | cleanup | this module is imported but never used — remove the import or use the module |
| `W2002` | safety | this variable shadows a variable with the same name in an outer scope |
| `W3001` | safety | this function might not always return a value — make sure all paths through the function end with a return |

---

## Error Code Ranges

| Range | What It Means |
|-------|---------------|
| E1xxx | Problems reading your code (invalid characters, numbers too large) |
| E2xxx | Problems understanding your code (missing brackets, unexpected symbols) |
| E3xxx | Type problems (wrong types, invalid operations) |
| E4xxx | Name problems (undefined variables, duplicate names) |
| E5xxx | Usage problems (wrong number of arguments) |
| E6xxx | Import problems (unknown modules) |
| E7xxx | Standard library problems (wrong usage of built-in functions) |
| W2xxx | Cleanup suggestions (unused imports) |
| W3xxx | Safety warnings (possible missing return values) |

---

*Generated on 2026-03-26 03:18:21 UTC*
