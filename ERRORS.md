# EZ Error Code Reference

> Auto-generated from `ezc/src/util/error_codes.h`. Do not edit manually.
> Run `./scripts/generate_errors.sh` to regenerate.

**Total: 96 codes** (89 errors, 7 warnings)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
| `E1003` | syntax | multi-line comment was never closed — add */ to close it |
| `E1005` | syntax | character literal was never closed — add a closing single quote |
| `E1006` | syntax | invalid escape sequence in string — valid escapes are \\n \\t \\\\ \\\ |
| `E1007` | syntax | invalid escape sequence in character literal |
| `E1010` | syntax | invalid number format — hex (0x), octal (0o), or binary (0b) prefix must be followed by digits |
| `E1011` | syntax | number cannot have consecutive underscores |
| `E1013` | syntax | number cannot end with an underscore |
| `E1014` | syntax | number cannot have an underscore before the decimal point |
| `E1015` | syntax | number cannot have an underscore after the decimal point |
| `E1016` | syntax | number cannot end with a trailing decimal point — add a digit after the dot |
| `E1017` | syntax | raw string literal was never closed — add a closing backtick |
| `E1018` | syntax | char literal must contain exactly one character — use a string for multiple characters |
| `E1019` | syntax | unexpected '#' character — use '//' for comments |
| `E2001` | syntax | unexpected symbol — the compiler found something it did not expect here |
| `E2002` | syntax | missing symbol — a bracket, parenthesis, or keyword is missing |
| `E2011` | syntax | constants must have a value — add = followed by a value after the type |
| `E2012` | syntax | duplicate parameter name — each parameter must have a unique name |
| `E2013` | syntax | duplicate struct field name — each field must have a unique name |
| `E2016` | syntax | empty enum — an enum must have at least one value |
| `E2017` | syntax | trailing comma — remove the extra comma before the closing bracket or brace |
| `E2025` | syntax | expected integer for array size — the second value in [type, size] must be a positive integer |
| `E2036` | syntax | imports must be at the top of the file, not inside a function |
| `E2037` | syntax | duplicate function name in struct — each function must have a unique name |
| `E2038` | syntax | reserved name for struct or enum — this name is used by the language |
| `E2039` | syntax | required parameter cannot appear after a parameter with a default value |
| `E2043` | syntax | duplicate case value in when statement — each case must be unique |
| `E2050` | syntax | break and continue can only be used inside a loop |
| `E2051` | syntax | functions cannot be defined inside other functions — move it to the top level |
| `E2053` | syntax | structs and enums must be defined at the file scope, not inside a function |
| `E2056` | syntax | executable statements are not allowed at file scope — put code inside do main() { } |
| `E2057` | syntax | invalid interpolation syntax — use ${variable} instead of $variable |
| `E2058` | syntax | nested type declarations are not allowed — structs and enums must be defined at the file scope |
| `E2059` | syntax | empty when block — a when statement must have at least one 'is' branch |
| `E2060` | syntax | too many return values — a function can return at most 16 values |
| `E3001` | types | wrong type — you are using a value of one type where a different type is expected |
| `E3002` | types | this operator does not work on this type — for example, you cannot subtract strings |
| `E3003` | types | invalid array index type — array indices must be integers |
| `E3005` | types | cannot change a constant — use mut instead of const if you need to modify this value |
| `E3006` | types | return value problem — either returning a value from a function that should not return one, or missing a return value |
| `E3007` | types | ++ and -- only work on integers — you cannot increment or decrement a non-integer |
| `E3008` | types | cannot use [] on this type — only arrays, maps, and strings support indexing |
| `E3009` | types | cannot loop over this type — for_each only works with arrays, maps, and strings |
| `E3010` | types | this struct does not have a field with that name |
| `E3011` | types | a type name like int or string cannot be used as a value — did you forget to declare a variable? |
| `E3012` | types | addr() needs a variable, field, or array element — you cannot take the address of a literal like 42 |
| `E3013` | types | only structs have fields — you cannot use .field on a number, string, or bool |
| `E3015` | types | this value is not a function and cannot be called |
| `E3016` | types | only pointers can be dereferenced with ^ — this value is not a pointer |
| `E3017` | types | fmt.printf/sprintf cannot format composite types — use println() or access individual fields |
| `E3018` | types | type mismatch in when/is — the case value type does not match the scrutinee type |
| `E3019` | types | cannot assign a signed integer to an unsigned type — the value may be negative |
| `E3024` | types | this function must return a value but the body has no return statement |
| `E3027` | types | cannot pass a constant to a mutable parameter — the function wants to modify this value |
| `E3032` | types | cannot compare different enum types — they are never equal |
| `E3033` | types | duplicate value in enum — each enum member must have a unique value |
| `E3034` | types | 'any' type is reserved for internal use and cannot be used in declarations |
| `E3035` | types | not all code paths return a value — add return statements to all branches |
| `E3036` | types | value is out of range for this type — for example, 200 does not fit in an i8 (-128 to 127) |
| `E3038` | types | 'void' cannot be used as a variable type or in expressions like typeof() |
| `E3039` | types | ensure expects a function call — for example: ensure close(file) |
| `E3040` | types | this function returns multiple values but you are assigning to a single variable — use mut a, b = func() |
| `E3041` | types | cannot interpolate void expression — the function does not return a value |
| `E3042` | types | struct functions must be called on the type, not an instance — use Type.func() instead of variable.func() |
| `E3043` | types | cannot cast between incompatible types — structs and maps cannot be cast to primitives |
| `E3044` | types | cannot access a field on a struct type — use an instance variable instead |
| `E3045` | types | or_return requires a function that returns (T, Error) — the called function does not return an error |
| `E4001` | names | this variable does not exist — check the spelling or make sure it is declared above this line |
| `E4002` | names | this function does not exist — check the spelling or make sure it is defined |
| `E4003` | names | a variable with this name already exists in this scope — use a different name |
| `E4004` | names | a function with this name already exists — each function must have a unique name |
| `E4005` | names | no main() function found — every EZ program needs a do main() { } function |
| `E4006` | names | this name is reserved by the compiler — choose a different name that does not start with ez_ or Ez |
| `E4012` | names | variable shadows a type definition with the same name |
| `E4013` | names | variable shadows a function with the same name |
| `E4014` | names | variable shadows an imported module name |
| `E5007` | usage | cannot modify an immutable value — declare with mut to allow modification |
| `E5008` | arguments | wrong number of arguments — the function expects a different number of values than you provided |
| `E5011` | usage | return value of function is not used — assign it to a variable or use _ to discard |
| `E5015` | usage | postfix ++ and -- require a variable, not a literal or expression |
| `E5023` | usage | ++ and -- only work on integer types, not floats |
| `E5024` | usage | return type mismatch — cannot return a signed value as an unsigned type |
| `E6001` | imports | unknown module — this is not a built-in EZ module. Check the spelling or see the docs for available modules |
| `E7004` | stdlib | function argument must be an integer, not a float |
| `E7006` | stdlib | threads.spawn() needs a function reference — use ()function_name to pass a function |
| `E7014` | stdlib | cannot convert negative value to char — must be a valid Unicode code point |
| `E9002` | stdlib | array operation requires a numeric array — cannot use sum/min/max on string or bool arrays |
| `E9005` | stdlib | range bounds must be valid — start must be less than end |
| `E12001` | stdlib | maps function requires a map argument, not an array |
| `E12006` | stdlib | duplicate key in map literal — each key must be unique |

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
| `W1001` | cleanup | variable is declared but never used — remove it or use it |
| `W1003` | cleanup | function is declared but never called — remove it or call it |
| `W1005` | cleanup | typed blank identifier — adding a type to _ is unnecessary, use plain _ instead |
| `W2001` | cleanup | this module is imported but never used — remove the import or use the module |
| `W2002` | safety | this variable shadows a variable with the same name in an outer scope |
| `W2011` | safety | named return variable is not used in return — the returned value may not match the named variable |
| `W3003` | safety | fixed-size array is not fully initialized — remaining elements will be zero-valued |

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

*Generated on 2026-04-01 17:35:54 UTC*
