# EZ Error Code Reference

> Auto-generated from `ezc/src/util/error_codes.h`. Do not edit manually.
> Run `./scripts/generate_errors.sh` to regenerate.

**Total: 191 codes** (176 errors, 15 warnings)

---

## Errors

| Code | Category | Description |
|------|----------|-------------|
| `E1003` | syntax | unclosed multi-line comment; add */ |
| `E1005` | syntax | unclosed character; add a closing single quote |
| `E1006` | syntax | invalid escape sequence in string; valid escapes are \\n \\t \\\\ \\\ |
| `E1007` | syntax | invalid escape sequence in character |
| `E1010` | syntax | invalid number format; hex (0x), octal (0o), or binary (0b) prefix must be followed by digits |
| `E1011` | syntax | number cannot have consecutive underscores |
| `E1012` | syntax | numeric literals cannot start with an underscore; did you mean '%s'? |
| `E1013` | syntax | number cannot end with an underscore |
| `E1014` | syntax | number cannot have an underscore before the decimal point |
| `E1015` | syntax | number cannot have an underscore after the decimal point |
| `E1016` | syntax | number cannot end with a trailing decimal point; add a digit after the dot |
| `E1017` | syntax | unclosed raw string; add a closing backtick |
| `E1018` | syntax | a char holds exactly one character; use a string for multiple |
| `E1019` | syntax | unexpected '#' character; use '//' for comments |
| `E1020` | syntax | unexpected '|' character; use '||' for logical OR |
| `E1021` | syntax | unclosed string; add a closing double quote |
| `E1022` | syntax | unexpected character |
| `E2001` | syntax | unexpected symbol |
| `E2002` | syntax | missing symbol; expected a bracket, parenthesis, or keyword |
| `E2010` | syntax | cannot use module '%s' before importing it; add 'import @%s' before the using statement |
| `E2011` | syntax | constant '%s' must have a value; add = followed by a value |
| `E2012` | syntax | duplicate parameter name '%s' |
| `E2013` | syntax | duplicate field name '%s' in struct '%s' |
| `E2014` | syntax | duplicate variant name '%s' in enum '%s' |
| `E2015` | syntax | duplicate field '%s' in struct literal; field can only be initialized once |
| `E2016` | syntax | enum '%s' has no values; an enum must have at least one value |
| `E2017` | syntax | stray comma; remove the extra ',' |
| `E2025` | syntax | expected integer for array size; the second value in [type, size] must be a positive integer |
| `E2036` | syntax | imports must be at the top of the file, not inside a function |
| `E2037` | syntax | duplicate function name in struct; each function must have a unique name |
| `E2038` | syntax | reserved name for struct or enum; this name is used by the language |
| `E2039` | syntax | required parameter '%s' cannot come after a parameter with a default value |
| `E2043` | syntax | duplicate case value in when statement |
| `E2050` | syntax | break and continue can only be used inside a loop |
| `E2051` | syntax | nested function declarations are not allowed; define '%s' at the top level |
| `E2053` | syntax | %s '%s' must be defined at the file scope, not inside a function |
| `E2056` | syntax | statements cannot sit at file scope; move this into do main() |
| `E2057` | syntax | invalid interpolation syntax; use ${variable} instead of $variable |
| `E2058` | syntax | cannot declare a struct or enum inside %s '%s'; define it at the file scope |
| `E2059` | syntax | empty when block; add at least one 'is' branch |
| `E2060` | syntax | too many return values; a function can return at most 16 values |
| `E2061` | syntax | 'module' declarations are not supported; imported files are identified by their file path |
| `E2062` | syntax | too many variables in multi-variable declaration; maximum is %d |
| `E2063` | syntax | duplicate or conflicting named return value; each name must be unique and not collide with parameters |
| `E2064` | syntax | function '%s' conflicts with field '%s' in struct '%s' |
| `E2065` | syntax | enum variant '%s' cannot have the same name as its enum type '%s' |
| `E2066` | syntax | struct field '%s' cannot have the same name as its struct type '%s' |
| `E2067` | syntax | struct '%s' has no fields; a struct must have at least one field |
| `E2068` | syntax | %ss must be declared with 'const', not 'mut'; change 'mut' to 'const' |
| `E2069` | syntax | unexpected semicolon; statements and declarations are separated by newlines, not semicolons |
| `E2070` | syntax | wildcard type '?' is only allowed in function parameter and return types; not in variable declarations, struct fields, or enum types |
| `E2071` | syntax | empty string interpolation '${}'; interpolation requires an expression between the braces |
| `E2072` | syntax | '&' is not a valid operator; use 'addr(x)' to take the address of a variable |
| `E2073` | syntax | function calls cannot have whitespace between the name and the opening parenthesis; write 'name(...)' with no space or newline |
| `E2074` | syntax | member access cannot have whitespace before the dot; write 'obj.field' or 'Enum.VARIANT' with no space or newline |
| `E2075` | syntax | index expressions cannot have whitespace before the opening bracket; write 'arr[i]' with no space or newline |
| `E2076` | syntax | postfix operators ('++', '--', '^') cannot have whitespace before them; write 'x++', 'x--', or 'p^' with no space or newline |
| `E2078` | syntax | variable declarations must start with 'const' or 'mut'; did you mean 'const %s' or 'mut %s'? |
| `E2079` | syntax | 'nil' is a value, not a type; for a function that returns nothing, omit the '-> ...' clause |
| `E2080` | syntax | invalid character in C header path; only [A-Za-z0-9./_+-] are permitted |
| `E3001` | types | type mismatch; a value of one type is used where a different type is expected |
| `E3002` | types | this operator does not work on this type; for example, strings cannot be subtracted |
| `E3003` | types | invalid array index type; array indices must be integers |
| `E3005` | types | cannot modify constant '%s'; declare with 'mut' to make it mutable |
| `E3006` | types | too many variables; the function returns %d value(s) but variable %d was requested |
| `E3007` | types | cannot negate type '%s'; only numeric types support negation |
| `E3008` | types | type '%s' does not support indexing; only arrays, maps, and strings can be indexed |
| `E3009` | types | cannot iterate over type '%s'; for_each requires an array, map, or string |
| `E3010` | types | struct '%s' has no field '%s' |
| `E3011` | types | '%s' is a type, not a value; did you mean to declare a type? (e.g., mut x %s = ...) |
| `E3012` | types | addr() needs a variable, field, or array element; the address of a value like 42 cannot be taken |
| `E3013` | types | only structs have fields; .field is not valid on a number, string, or bool |
| `E3015` | types | '%s' is a %s, not a function; it cannot be called |
| `E3016` | types | cannot dereference non-pointer type '%s'; only ^T types can use ^ |
| `E3017` | types | fmt.%s() cannot format value of type '%s'; use println() for composite types, or access individual fields |
| `E3018` | types | type mismatch in 'when'; comparing '%s' with '%s' |
| `E3019` | types | cannot assign signed type '%s' to unsigned type '%s'; value may be negative |
| `E3024` | types | function '%s' must return a value but has no return statement |
| `E3027` | types | cannot pass a constant to a mutable parameter; the function wants to modify this value |
| `E3031` | types | function '%s' cannot be used as a value; did you mean '%s()' or '()%s'? |
| `E3032` | types | cannot compare enum '%s' with enum '%s'; different enum types are never equal |
| `E3033` | types | duplicate value in enum '%s': '%s' and '%s' both have the same value |
| `E3034` | types | 'any' type is reserved for internal use and cannot be used in declarations |
| `E3035` | types | not all code paths in '%s' return a value |
| `E3036` | types | value %lld is out of range for type '%s' (valid range: %lld to %lld) |
| `E3038` | types | 'void' cannot be used as a variable type or in expressions like typeof() |
| `E3039` | types | ensure expects a function call; for example: ensure close(file) |
| `E3040` | types | '%s' returns %d values; use mut a, b = %s() to capture all of them |
| `E3041` | types | cannot interpolate expression; interpolation supports primitives, strings, arrays, and maps |
| `E3042` | types | struct functions must be called on the type; use '%s.%s()' instead of '%s.%s()' |
| `E3043` | types | cannot cast between incompatible types; only numeric, enum, and string conversions are allowed |
| `E3044` | types | cannot access field '%s' on type '%s'; use an instance variable instead |
| `E3045` | types | 'or_return' requires a function that returns (T, Error); '%s()' does not return an error |
| `E3046` | types | integer too large for 64 bits; max is 9223372036854775807 |
| `E3047` | types | enum '%s' has no member '%s' |
| `E3048` | types | operator '+' is not defined for strings; use string interpolation or fmt.format() instead |
| `E3049` | types | cannot use '%s' on enum values; enums only support == and != comparisons |
| `E3050` | types | array needs a type annotation; declare as [T] (e.g., mut x [int] = {1, 2, 3}) |
| `E3051` | types | map needs a type annotation; declare as map[K:V] (e.g., mut x map[string:int] = {\ |
| `E3052` | types | too many elements in array initializer; declared size is %d, got %d |
| `E3053` | types | type mismatch in array initializer; expected '%s', got '%s' |
| `E3054` | types | mutable arrays cannot have a fixed size; remove the size or use 'const' (e.g., mut %s %.*s] = ...) |
| `E3055` | types | const arrays must have a fixed size; declare as [T, N] (e.g., const %s [%.*s, %d] = ...) |
| `E3056` | types | #strict when is not exhaustive; missing variant '%s.%s' |
| `E3057` | types | type '%s' cannot be used as a map key; only primitive types (int, string, bool, char, byte, float) and enums are hashable |
| `E3058` | types | in instantiation of generic function '%s' with '?' = %s |
| `E3059` | types | maps cannot be declared const; use 'mut' for maps or a struct for fixed data |
| `E3060` | types | wildcard '?' in return type cannot be resolved; at least one parameter must also use '?' to bind the concrete type |
| `E3061` | types | struct '%s' cannot contain itself by value through '%s'; break the cycle with a pointer field '^%s' |
| `E3062` | types | %s cannot be declared const; use 'mut' (every operation on a %s mutates its state) |
| `E3063` | types | cannot return addr(%s); '%s' is a local variable whose memory is freed when this function returns |
| `E3064` | types | %s(%s) called again; '%s' was already destroyed |
| `E3065` | types | bare 'func' is not a valid type; use func(<params>) -> <return> with an explicit signature |
| `E3066` | types | function reference signature mismatch; expected and actual function types differ |
| `E3067` | types | argument %d of '%s' is passed to a '&' parameter; pass a mutable variable, not a literal or expression |
| `E3068` | types | 'void' is not a user-facing type; omit the '-> R' clause to declare a function with no return value |
| `E3069` | types | '&' on a parameter must come before the name, not the type; write '&%s %s' to mark this parameter mutable |
| `E3070` | types | 'ensure' may only appear at the top level of a function body; lift it out of the enclosing block |
| `E3071` | types | cannot 'return nil' from a function whose return type contains '?'; 'nil' is not a valid value for every binding (e.g. int, string) |
| `E3072` | types | cannot return 'nil' from a function that returns '%s'; nil is only valid for pointer and error types |
| `E3073` | types | 'return' is not allowed in main(); main exits when control reaches the closing brace |
| `E3074` | types | arrays cannot be compared with comparison operators; use arrays.is_equal(a, b) for equality, or compare elements individually for ordering |
| `E3075` | types | chained struct function calls are not supported; assign the intermediate result to a variable, then call the next struct function on it |
| `E3076` | types | maps cannot be compared with comparison operators; use maps.is_equal(a, b) for equality (maps have no defined ordering) |
| `E3077` | types | structs cannot be compared with comparison operators; compare individual fields instead (e.g., a.x == b.x, a.x < b.x) |
| `E3078` | types | pointer arithmetic is not supported; '^T' is the address of one value, not a buffer |
| `E3079` | types | cannot take a mutable reference to a const variable; declare the reference as 'const', or copy() the value to get an independent mutable instance |
| `E3080` | types | function must return named variable '%s', not a different expression |
| `E3081` | types | function '%s' used as a statement without being called; did you mean '%s()'? |
| `E3082` | types | wildcard type '?' cannot be used in named return positions; use an unnamed return instead |
| `E3083` | types | c_string() requires a raw C pointer; cannot convert a non-pointer type |
| `E3084` | types | type_of() expects a value, not a type name; use type_of(instance) instead |
| `E3085` | types | 'in' operator type mismatch: cannot check if '%s' is in '%s' |
| `E3086` | types | fmt.%s format string must be a string literal; use string interpolation for dynamic values |
| `E3087` | types | %%n is not permitted in fmt format strings |
| `E3088` | types | fmt.%s format directive '%%%s' expects %s but argument %d has type '%s' |
| `E4001` | names | this variable does not exist; check the spelling or make sure it is declared above this line |
| `E4002` | names | this function does not exist; check the spelling or make sure it is defined |
| `E4003` | names | variable '%s' already declared in this scope (line %d) |
| `E4004` | names | function '%s' already declared |
| `E4005` | names | module '%s' has no function named '%s' |
| `E4006` | names | name '%s' uses reserved prefix (ez_, _ez_, Ez); these are reserved for the compiler |
| `E4007` | names | a type with this name already exists; each struct and enum must have a unique name |
| `E4008` | names | main() cannot have parameters or a return type; it must be declared as do main() { } |
| `E4012` | names | variable '%s' shadows a type definition with the same name |
| `E4013` | names | variable '%s' shadows a function with the same name |
| `E4014` | names | variable '%s' shadows an imported module with the same name |
| `E4015` | names | '%s' is private and cannot be accessed from outside its file |
| `E4016` | names | undefined type '%s'; check the spelling or import the module that defines it |
| `E4017` | names | function '%s.%s' is private and cannot be called from outside the struct |
| `E4018` | names | struct '%s' has no function named '%s' |
| `E5007` | usage | cannot modify immutable %s '%s'; declare with 'mut' to allow modification |
| `E5008` | arguments | wrong number of arguments; the function expects a different count than was provided |
| `E5009` | arguments | invalid base for integer conversion; base must be between 2 and 36 |
| `E5011` | usage | return value of '%s' is not used; assign it to a variable or use '_' to discard |
| `E5012` | usage | the throwaway '_' is only meaningful when discarding the result of a function call; the right-hand side has no return value to discard |
| `E5013` | usage | function calls are not allowed in file-scope initializers; move this declaration into a function body |
| `E5014` | usage | here() takes no arguments; the call site's file, line, and column are substituted at compile time |
| `E5015` | usage | postfix ++ and -- require a variable, not a value or expression |
| `E5016` | naming | this name is reserved by a builtin function and cannot be redeclared |
| `E5023` | usage | cannot use '%s' on type '%s'; only integer types support increment/decrement |
| `E5024` | usage | return type mismatch: cannot return signed '%s' as unsigned '%s' |
| `E5025` | usage | invalid assignment target; left side of '=' must be a variable, field, or index expression |
| `E5026` | arguments | argument type mismatch; the function expects a different type than what was provided |
| `E6001` | imports | unknown module '@%s' |
| `E6002` | imports | cannot find file or directory '%s' |
| `E6003` | imports | directory '%s' contains no .ez files |
| `E6004` | imports | cannot import own module directory |
| `E7004` | stdlib | function argument must be an integer, not a float |
| `E7006` | stdlib | threads.spawn() needs a function reference; use ()function_name to pass a function |
| `E7014` | stdlib | cannot convert %lld to char; value must be a valid Unicode code point (0 or greater) |
| `E7015` | stdlib | len() is not supported for type '%s'; len() works on string, array, and map types |
| `E9002` | stdlib | arrays.%s() requires a numeric array, got array of %s |
| `E9005` | stdlib | invalid range: start (%lld) must be less than end (%lld) |
| `E12001` | stdlib | maps.%s() requires a map argument, got an array |
| `E12006` | stdlib | duplicate key in map literal |

---

## Warnings

| Code | Category | Description |
|------|----------|-------------|
| `W1001` | cleanup | variable is declared but never used; remove it or use it |
| `W1003` | cleanup | function is declared but never called; remove it or call it |
| `W1005` | cleanup | typed blank identifier; '_' doesn't need a type annotation, use 'mut _ = <expr>' instead |
| `W2001` | cleanup | this module is imported but never used; remove the import or use the module |
| `W1002` | cleanup | this import is never used; remove it or use a function from the module |
| `W2002` | safety | this variable shadows a variable with the same name in an outer scope |
| `W2003` | safety | unreachable code; this statement will never execute because it comes after a return |
| `W2007` | safety | this variable shadows a global constant or variable |
| `W2011` | safety | named return value is declared in the signature but no matching variable exists in the function body |
| `W2012` | safety | when condition is a float; equality checks on floats are imprecise; prefer math.abs(x - y) < epsilon |
| `W2013` | imports | duplicate import of already-imported module |
| `W2014` | imports | intra-directory import already included by directory import |
| `W2015` | imports | file already imported as part of a directory import; redundant import |
| `W3003` | safety | fixed-size array is not fully initialized; remaining elements will be zero-valued |
| `W3004` | safety | pointer may reference memory from a scope that has ended; assigning addr() of an inner-scope variable to an outer-scope pointer |

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

*Generated on 2026-05-20 02:08:49 UTC*
