# EZ Error Code Reference

> Auto-generated from `ezc/src/util/error_codes.h`. Do not edit manually.
> Run `./scripts/generate_errors.sh` to regenerate.

**Total: 153 codes** (141 errors, 12 warnings)

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
| `E1012` | syntax | numeric literals cannot start with an underscore |
| `E1013` | syntax | number cannot end with an underscore |
| `E1014` | syntax | number cannot have an underscore before the decimal point |
| `E1015` | syntax | number cannot have an underscore after the decimal point |
| `E1016` | syntax | number cannot end with a trailing decimal point — add a digit after the dot |
| `E1017` | syntax | raw string literal was never closed — add a closing backtick |
| `E1018` | syntax | char literal must contain exactly one character — use a string for multiple characters |
| `E1019` | syntax | unexpected '#' character — use '//' for comments |
| `E1020` | syntax | unexpected '|' character — use '||' for logical OR |
| `E1021` | syntax | string literal was never closed — add a closing double quote |
| `E1022` | syntax | unexpected character |
| `E2001` | syntax | unexpected symbol — the compiler found something it did not expect here |
| `E2002` | syntax | missing symbol — a bracket, parenthesis, or keyword is missing |
| `E2010` | syntax | cannot use a module before importing it — add the import statement before using |
| `E2011` | syntax | constants must have a value — add = followed by a value after the type |
| `E2012` | syntax | duplicate parameter name — each parameter must have a unique name |
| `E2013` | syntax | duplicate struct field name — each field must have a unique name |
| `E2014` | syntax | duplicate enum variant name — each variant must have a unique name |
| `E2015` | syntax | duplicate field in struct literal — each field can only be initialized once |
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
| `E2061` | syntax | 'module' declarations are not supported — imported files are identified by their file path |
| `E2062` | syntax | too many variables in multi-variable declaration — maximum is 16 |
| `E2063` | syntax | duplicate or conflicting named return value — each name must be unique and not collide with parameters |
| `E2064` | syntax | struct function name conflicts with a field name — functions and fields must have distinct names |
| `E2065` | syntax | enum variant cannot have the same name as its enum type |
| `E2066` | syntax | struct field cannot have the same name as its struct type |
| `E2067` | syntax | empty struct — a struct must have at least one field |
| `E2068` | syntax | structs and enums must be declared with 'const', not 'mut' |
| `E2069` | syntax | unexpected semicolon — statements and declarations are separated by newlines, not semicolons |
| `E2070` | syntax | wildcard type '?' is only allowed in function parameter and return types — not in variable declarations, struct fields, or enum types |
| `E2071` | syntax | empty string interpolation '${}' — interpolation requires an expression between the braces |
| `E2072` | syntax | '&' is not a valid operator — use 'addr(x)' to take the address of a variable |
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
| `E3031` | types | function name cannot be used as a value — call it with '()' or take a reference with '()name' |
| `E3032` | types | cannot compare different enum types — they are never equal |
| `E3033` | types | duplicate value in enum — each enum member must have a unique value |
| `E3034` | types | 'any' type is reserved for internal use and cannot be used in declarations |
| `E3035` | types | not all code paths return a value — add return statements to all branches |
| `E3036` | types | value is out of range for this type — for example, 200 does not fit in an i8 (-128 to 127) |
| `E3038` | types | 'void' cannot be used as a variable type or in expressions like typeof() |
| `E3039` | types | ensure expects a function call — for example: ensure close(file) |
| `E3040` | types | this function returns multiple values but you are assigning to a single variable — use mut a, b = func() |
| `E3041` | types | cannot interpolate expression — interpolation supports primitives, strings, arrays, and maps |
| `E3042` | types | struct functions must be called on the type, not an instance — use Type.func() instead of variable.func() |
| `E3043` | types | cannot cast between incompatible types — only numeric, enum, and string conversions are allowed |
| `E3044` | types | cannot access a field on a struct type — use an instance variable instead |
| `E3045` | types | or_return requires a function that returns (T, Error) — the called function does not return an error |
| `E3046` | types | integer literal overflows 64-bit integer — max value is 9223372036854775807 |
| `E3047` | types | this enum does not have a member with that name |
| `E3048` | types | operator '+' is not defined for strings — use string interpolation or fmt.format() instead |
| `E3049` | types | arithmetic operators are not valid on enum values — enums only support == and != comparisons |
| `E3050` | types | array literal requires a type annotation — declare as [T] (e.g., mut x [int] = {1, 2, 3}) |
| `E3051` | types | map literal requires a type annotation — declare as map[K:V] (e.g., mut x map[string:int] = {\ |
| `E3052` | types | too many elements in array initializer — declared size is %d, got %d |
| `E3053` | types | type mismatch in array initializer — expected '%s', got '%s' |
| `E3054` | types | mutable arrays cannot have a fixed size — remove the size or use 'const' |
| `E3055` | types | const arrays must have a fixed size — declare as [T, N] |
| `E3056` | types | non-exhaustive #strict when — all enum variants must be handled or add a default branch |
| `E3057` | types | type cannot be used as a map key — only primitive types (int, string, bool, char, byte, float) and enums are hashable |
| `E3058` | types | generic function instantiation failed — the body operation is not supported for the concrete type at this call site |
| `E3059` | types | maps cannot be declared const — use 'mut' for maps or a struct for fixed data |
| `E3060` | types | wildcard '?' in return type cannot be resolved — at least one parameter must also use '?' to bind the concrete type |
| `E3061` | types | struct cannot contain itself by value — use a pointer field '^T' for recursive types |
| `E3062` | types | handle types (channels, mutexes, threads) cannot be declared const — use 'mut' |
| `E3063` | types | cannot return address of local variable — the variable's memory is freed when the function returns |
| `E3064` | types | mem.destroy() called twice on the same arena — each arena can only be destroyed once |
| `E3065` | types | bare 'func' is not a valid type — use func(<params>) -> <return> with an explicit signature |
| `E3066` | types | function reference signature mismatch — expected and actual function types differ |
| `E3067` | types | argument passed by value to a '&' parameter — pass an lvalue (variable) instead |
| `E3068` | types | 'void' is not a user-facing type — omit the '-> R' clause to declare a function with no return value |
| `E3069` | types | '&' on a parameter must come before the name, not the type — write '&name type' to mark a parameter mutable |
| `E3070` | types | 'ensure' may only appear at the top level of a function body — lift it out of the enclosing block |
| `E3071` | types | cannot 'return nil' from a function whose return type contains '?' — 'nil' is not a valid value for every binding |
| `E4001` | names | this variable does not exist — check the spelling or make sure it is declared above this line |
| `E4002` | names | this function does not exist — check the spelling or make sure it is defined |
| `E4003` | names | a variable with this name already exists in this scope — use a different name |
| `E4004` | names | a function with this name already exists — each function must have a unique name |
| `E4005` | names | no main() function found — every program needs a do main() { } function |
| `E4006` | names | this name is reserved by the compiler — choose a different name that does not start with ez_ or Ez |
| `E4007` | names | a type with this name already exists — each struct and enum must have a unique name |
| `E4008` | names | main() cannot have parameters or a return type — it must be declared as do main() { } |
| `E4012` | names | variable shadows a type definition with the same name |
| `E4013` | names | variable shadows a function with the same name |
| `E4014` | names | variable shadows an imported module name |
| `E4015` | names | cannot access a private function or constant from outside its file |
| `E5007` | usage | cannot modify an immutable value — declare with mut to allow modification |
| `E5008` | arguments | wrong number of arguments — the function expects a different number of values than you provided |
| `E5011` | usage | return value of function is not used — assign it to a variable or use _ to discard |
| `E5015` | usage | postfix ++ and -- require a variable, not a literal or expression |
| `E5023` | usage | ++ and -- only work on integer types, not floats |
| `E5024` | usage | return type mismatch — cannot return a signed value as an unsigned type |
| `E5025` | usage | invalid assignment target — left side of '=' must be a variable, field, or index expression |
| `E5026` | arguments | argument type mismatch — the function expects a different type than what was provided |
| `E6001` | imports | unknown module — this is not a built-in module. Check the spelling or see the docs for available modules |
| `E7004` | stdlib | function argument must be an integer, not a float |
| `E7006` | stdlib | threads.spawn() needs a function reference — use ()function_name to pass a function |
| `E7014` | stdlib | cannot convert negative value to char — must be a valid Unicode code point |
| `E7015` | stdlib | len() only works on string, array, and map types — other types have no meaningful length |
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
| `W1005` | cleanup | typed blank identifier — '_' doesn't need a type annotation, use 'mut _ = <expr>' instead |
| `W2001` | cleanup | this module is imported but never used — remove the import or use the module |
| `W1002` | cleanup | this import is never used — remove it or use a function from the module |
| `W2002` | safety | this variable shadows a variable with the same name in an outer scope |
| `W2003` | safety | unreachable code — this statement will never execute because it comes after a return |
| `W2007` | safety | this variable shadows a global constant or variable |
| `W2011` | safety | named return variable is not used in return — the returned value may not match the named variable |
| `W2012` | safety | when condition is a float — equality checks on floats are imprecise; prefer math.abs(x - y) < epsilon |
| `W3003` | safety | fixed-size array is not fully initialized — remaining elements will be zero-valued |
| `W3004` | safety | pointer may reference memory from a scope that has ended — assigning addr() of an inner-scope variable to an outer-scope pointer |

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
| E8xxx | Math errors (sqrt of negative, log of non-positive) |
| E9xxx | Array errors (empty array ops, invalid ranges) |
| E10xxx | String errors (index out of bounds, repeat count) |
| E11xxx | Time errors (parsing failures) |
| E12xxx | Map errors (unhashable keys, immutable maps) |
| E13xxx | JSON errors (syntax, unsupported types) |
| W1xxx | Cleanup suggestions (unused variables, functions) |
| W2xxx | Safety warnings (shadowing, unused imports) |
| W3xxx | Quality warnings (empty blocks) |

---

*Generated on 2026-04-24 01:43:18 UTC*
