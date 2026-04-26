/*
 * error_codes.h - Centralized error code registry for EZC
 *
 * This is the SINGLE SOURCE OF TRUTH for all error and warning codes.
 * The scripts/generate_errors.sh script reads this file to produce ERRORS.md.
 *
 * IMPORTANT: Every code listed here MUST be emitted somewhere in the compiler
 * source. Do not add codes that are not actually used.
 *
 * Format: EZ_ERROR(code, category, description)
 *         EZ_WARNING(code, category, description)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_ERROR_CODES_H
#define EZC_ERROR_CODES_H

/* --- E1xxx: Reading Your Code (Lexer) --- */
#define EZ_LEXER_ERRORS \
    EZ_ERROR("E1003", "syntax", "unclosed multi-line comment; add */") \
    EZ_ERROR("E1005", "syntax", "unclosed character; add a closing single quote") \
    EZ_ERROR("E1006", "syntax", "invalid escape sequence in string; valid escapes are \\n \\t \\\\ \\\" and \\x") \
    EZ_ERROR("E1007", "syntax", "invalid escape sequence in character") \
    EZ_ERROR("E1010", "syntax", "invalid number format; hex (0x), octal (0o), or binary (0b) prefix must be followed by digits") \
    EZ_ERROR("E1011", "syntax", "number cannot have consecutive underscores") \
    EZ_ERROR("E1012", "syntax", "numeric literals cannot start with an underscore; did you mean '%s'?") \
    EZ_ERROR("E1013", "syntax", "number cannot end with an underscore") \
    EZ_ERROR("E1014", "syntax", "number cannot have an underscore before the decimal point") \
    EZ_ERROR("E1015", "syntax", "number cannot have an underscore after the decimal point") \
    EZ_ERROR("E1016", "syntax", "number cannot end with a trailing decimal point; add a digit after the dot") \
    EZ_ERROR("E1017", "syntax", "unclosed raw string; add a closing backtick") \
    EZ_ERROR("E1018", "syntax", "a char holds exactly one character; use a string for multiple") \
    EZ_ERROR("E1019", "syntax", "unexpected '#' character; use '//' for comments") \
    EZ_ERROR("E1020", "syntax", "unexpected '|' character; use '||' for logical OR") \
    EZ_ERROR("E1021", "syntax", "unclosed string; add a closing double quote") \
    EZ_ERROR("E1022", "syntax", "unexpected character")

/* --- E2xxx: Understanding Your Code (Parser) --- */
#define EZ_PARSER_ERRORS \
    EZ_ERROR("E2001", "syntax", "unexpected symbol") \
    EZ_ERROR("E2002", "syntax", "missing symbol; expected a bracket, parenthesis, or keyword") \
    EZ_ERROR("E2010", "syntax", "cannot use module '%s' before importing it; add 'import @%s' before the using statement") \
    EZ_ERROR("E2011", "syntax", "constant '%s' must have a value; add = followed by a value") \
    EZ_ERROR("E2012", "syntax", "duplicate parameter name '%s'") \
    EZ_ERROR("E2013", "syntax", "duplicate field name '%s' in struct '%s'") \
    EZ_ERROR("E2014", "syntax", "duplicate variant name '%s' in enum '%s'") \
    EZ_ERROR("E2015", "syntax", "duplicate field '%s' in struct literal; field can only be initialized once") \
    EZ_ERROR("E2016", "syntax", "enum '%s' has no values; an enum must have at least one value") \
    EZ_ERROR("E2017", "syntax", "stray comma; remove the extra ','") \
    EZ_ERROR("E2025", "syntax", "expected integer for array size; the second value in [type, size] must be a positive integer") \
    EZ_ERROR("E2036", "syntax", "imports must be at the top of the file, not inside a function") \
    EZ_ERROR("E2037", "syntax", "duplicate function name in struct; each function must have a unique name") \
    EZ_ERROR("E2038", "syntax", "reserved name for struct or enum; this name is used by the language") \
    EZ_ERROR("E2039", "syntax", "required parameter '%s' cannot come after a parameter with a default value") \
    EZ_ERROR("E2043", "syntax", "duplicate case value in when statement") \
    EZ_ERROR("E2050", "syntax", "break and continue can only be used inside a loop") \
    EZ_ERROR("E2051", "syntax", "nested function declarations are not allowed; define '%s' at the top level") \
    EZ_ERROR("E2053", "syntax", "%s '%s' must be defined at the file scope, not inside a function") \
    EZ_ERROR("E2056", "syntax", "statements cannot sit at file scope; move this into do main()") \
    EZ_ERROR("E2057", "syntax", "invalid interpolation syntax; use ${variable} instead of $variable") \
    EZ_ERROR("E2058", "syntax", "cannot declare a struct or enum inside %s '%s'; define it at the file scope") \
    EZ_ERROR("E2059", "syntax", "empty when block; add at least one 'is' branch") \
    EZ_ERROR("E2060", "syntax", "too many return values; a function can return at most 16 values") \
    EZ_ERROR("E2061", "syntax", "'module' declarations are not supported; imported files are identified by their file path") \
    EZ_ERROR("E2062", "syntax", "too many variables in multi-variable declaration; maximum is %d") \
    EZ_ERROR("E2063", "syntax", "duplicate or conflicting named return value; each name must be unique and not collide with parameters") \
    EZ_ERROR("E2064", "syntax", "function '%s' conflicts with field '%s' in struct '%s'") \
    EZ_ERROR("E2065", "syntax", "enum variant '%s' cannot have the same name as its enum type '%s'") \
    EZ_ERROR("E2066", "syntax", "struct field '%s' cannot have the same name as its struct type '%s'") \
    EZ_ERROR("E2067", "syntax", "struct '%s' has no fields; a struct must have at least one field") \
    EZ_ERROR("E2068", "syntax", "%ss must be declared with 'const', not 'mut'; change 'mut' to 'const'") \
    EZ_ERROR("E2069", "syntax", "unexpected semicolon; statements and declarations are separated by newlines, not semicolons") \
    EZ_ERROR("E2070", "syntax", "wildcard type '?' is only allowed in function parameter and return types; not in variable declarations, struct fields, or enum types") \
    EZ_ERROR("E2071", "syntax", "empty string interpolation '${}'; interpolation requires an expression between the braces") \
    EZ_ERROR("E2072", "syntax", "'&' is not a valid operator; use 'addr(x)' to take the address of a variable") \
    EZ_ERROR("E2073", "syntax", "function calls cannot have whitespace between the name and the opening parenthesis; write 'name(...)' with no space or newline") \
    EZ_ERROR("E2074", "syntax", "member access cannot have whitespace before the dot; write 'obj.field' or 'Enum.VARIANT' with no space or newline") \
    EZ_ERROR("E2075", "syntax", "index expressions cannot have whitespace before the opening bracket; write 'arr[i]' with no space or newline") \
    EZ_ERROR("E2076", "syntax", "postfix operators ('++', '--', '^') cannot have whitespace before them; write 'x++', 'x--', or 'p^' with no space or newline") \
    EZ_ERROR("E2077", "syntax", "cannot have whitespace between a struct name and its opening brace; write 'Name{...}' with no space or newline") \
    EZ_ERROR("E2078", "syntax", "variable declarations must start with 'const' or 'mut'; did you mean 'const %s' or 'mut %s'?") \
    EZ_ERROR("E2079", "syntax", "'nil' is a value, not a type; for a function that returns nothing, omit the '-> ...' clause")

/* --- E3xxx: Type Problems (Typechecker) --- */
#define EZ_TYPE_ERRORS \
    EZ_ERROR("E3001", "types", "type mismatch; a value of one type is used where a different type is expected") \
    EZ_ERROR("E3002", "types", "this operator does not work on this type; for example, strings cannot be subtracted") \
    EZ_ERROR("E3003", "types", "invalid array index type; array indices must be integers") \
    EZ_ERROR("E3005", "types", "cannot modify constant '%s'; declare with 'mut' to make it mutable") \
    EZ_ERROR("E3006", "types", "too many variables; the function returns %d value(s) but variable %d was requested") \
    EZ_ERROR("E3007", "types", "cannot negate type '%s'; only numeric types support negation") \
    EZ_ERROR("E3008", "types", "type '%s' does not support indexing; only arrays, maps, and strings can be indexed") \
    EZ_ERROR("E3009", "types", "cannot iterate over type '%s'; for_each requires an array, map, or string") \
    EZ_ERROR("E3010", "types", "struct '%s' has no field '%s'") \
    EZ_ERROR("E3011", "types", "'%s' is a type, not a value; did you mean to declare a type? (e.g., mut x %s = ...)") \
    EZ_ERROR("E3012", "types", "addr() needs a variable, field, or array element; the address of a value like 42 cannot be taken") \
    EZ_ERROR("E3013", "types", "only structs have fields; .field is not valid on a number, string, or bool") \
    EZ_ERROR("E3015", "types", "'%s' is a %s, not a function; it cannot be called") \
    EZ_ERROR("E3016", "types", "cannot dereference non-pointer type '%s'; only ^T types can use ^") \
    EZ_ERROR("E3017", "types", "fmt.%s() cannot format value of type '%s'; use println() for composite types, or access individual fields") \
    EZ_ERROR("E3018", "types", "type mismatch in 'when'; comparing '%s' with '%s'") \
    EZ_ERROR("E3019", "types", "cannot assign signed type '%s' to unsigned type '%s'; value may be negative") \
    EZ_ERROR("E3024", "types", "function '%s' must return a value but has no return statement") \
    EZ_ERROR("E3027", "types", "cannot pass a constant to a mutable parameter; the function wants to modify this value") \
    EZ_ERROR("E3031", "types", "function '%s' cannot be used as a value; did you mean '%s()' or '()%s'?") \
    EZ_ERROR("E3032", "types", "cannot compare enum '%s' with enum '%s'; different enum types are never equal") \
    EZ_ERROR("E3033", "types", "duplicate value in enum '%s': '%s' and '%s' both have the same value") \
    EZ_ERROR("E3034", "types", "'any' type is reserved for internal use and cannot be used in declarations") \
    EZ_ERROR("E3035", "types", "not all code paths in '%s' return a value") \
    EZ_ERROR("E3036", "types", "value %lld is out of range for type '%s' (valid range: %lld to %lld)") \
    EZ_ERROR("E3038", "types", "'void' cannot be used as a variable type or in expressions like typeof()") \
    EZ_ERROR("E3039", "types", "ensure expects a function call; for example: ensure close(file)") \
    EZ_ERROR("E3040", "types", "'%s' returns %d values; use mut a, b = %s() to capture all of them") \
    EZ_ERROR("E3041", "types", "cannot interpolate expression; interpolation supports primitives, strings, arrays, and maps") \
    EZ_ERROR("E3042", "types", "struct functions must be called on the type; use '%s.%s()' instead of '%s.%s()'") \
    EZ_ERROR("E3043", "types", "cannot cast between incompatible types; only numeric, enum, and string conversions are allowed") \
    EZ_ERROR("E3044", "types", "cannot access field '%s' on type '%s'; use an instance variable instead") \
    EZ_ERROR("E3045", "types", "'or_return' requires a function that returns (T, Error); '%s()' does not return an error") \
    EZ_ERROR("E3046", "types", "integer too large for 64 bits; max is 9223372036854775807") \
    EZ_ERROR("E3047", "types", "enum '%s' has no member '%s'") \
    EZ_ERROR("E3048", "types", "operator '+' is not defined for strings; use string interpolation or fmt.format() instead") \
    EZ_ERROR("E3049", "types", "cannot use '%s' on enum values; enums only support == and != comparisons") \
    EZ_ERROR("E3050", "types", "array needs a type annotation; declare as [T] (e.g., mut x [int] = {1, 2, 3})") \
    EZ_ERROR("E3051", "types", "map needs a type annotation; declare as map[K:V] (e.g., mut x map[string:int] = {\"a\": 1})") \
    EZ_ERROR("E3052", "types", "too many elements in array initializer; declared size is %d, got %d") \
    EZ_ERROR("E3053", "types", "type mismatch in array initializer; expected '%s', got '%s'") \
    EZ_ERROR("E3054", "types", "mutable arrays cannot have a fixed size; remove the size or use 'const' (e.g., mut %s %.*s] = ...)") \
    EZ_ERROR("E3055", "types", "const arrays must have a fixed size; declare as [T, N] (e.g., const %s [%.*s, %d] = ...)") \
    EZ_ERROR("E3056", "types", "#strict when is not exhaustive; missing variant '%s.%s'") \
    EZ_ERROR("E3057", "types", "type '%s' cannot be used as a map key; only primitive types (int, string, bool, char, byte, float) and enums are hashable") \
    EZ_ERROR("E3058", "types", "in instantiation of generic function '%s' with '?' = %s") \
    EZ_ERROR("E3059", "types", "maps cannot be declared const; use 'mut' for maps or a struct for fixed data") \
    EZ_ERROR("E3060", "types", "wildcard '?' in return type cannot be resolved; at least one parameter must also use '?' to bind the concrete type") \
    EZ_ERROR("E3061", "types", "struct '%s' cannot contain itself by value through '%s'; break the cycle with a pointer field '^%s'") \
    EZ_ERROR("E3062", "types", "%s cannot be declared const; use 'mut' (every operation on a %s mutates its state)") \
    EZ_ERROR("E3063", "types", "cannot return addr(%s); '%s' is a local variable whose memory is freed when this function returns") \
    EZ_ERROR("E3064", "types", "%s(%s) called again; '%s' was already destroyed") \
    EZ_ERROR("E3065", "types", "bare 'func' is not a valid type; use func(<params>) -> <return> with an explicit signature") \
    EZ_ERROR("E3066", "types", "function reference signature mismatch; expected and actual function types differ") \
    EZ_ERROR("E3067", "types", "argument %d of '%s' is passed to a '&' parameter; pass a mutable variable, not a literal or expression") \
    EZ_ERROR("E3068", "types", "'void' is not a user-facing type; omit the '-> R' clause to declare a function with no return value") \
    EZ_ERROR("E3069", "types", "'&' on a parameter must come before the name, not the type; write '&%s %s' to mark this parameter mutable") \
    EZ_ERROR("E3070", "types", "'ensure' may only appear at the top level of a function body; lift it out of the enclosing block") \
    EZ_ERROR("E3071", "types", "cannot 'return nil' from a function whose return type contains '?'; 'nil' is not a valid value for every binding (e.g. int, string)") \
    EZ_ERROR("E3072", "types", "cannot return 'nil' from a function that returns '%s'; nil is only valid for pointer and error types") \
    EZ_ERROR("E3073", "types", "'return' is not allowed in main(); main exits when control reaches the closing brace") \
    EZ_ERROR("E3074", "types", "arrays cannot be compared with '==' or '!='; use arrays.is_equal(a, b) for structural equality") \
    EZ_ERROR("E3075", "types", "chained struct function calls are not supported; assign the intermediate result to a variable, then call the next struct function on it") \
    EZ_ERROR("E3076", "types", "maps cannot be compared with '==' or '!='; use maps.is_equal(a, b) for structural equality") \
    EZ_ERROR("E3077", "types", "structs cannot be compared with '==' or '!='; compare fields individually (e.g., a.x == b.x && a.y == b.y)")

/* --- E4xxx: Name Problems (References) --- */
#define EZ_REFERENCE_ERRORS \
    EZ_ERROR("E4001", "names", "this variable does not exist; check the spelling or make sure it is declared above this line") \
    EZ_ERROR("E4002", "names", "this function does not exist; check the spelling or make sure it is defined") \
    EZ_ERROR("E4003", "names", "variable '%s' already declared in this scope (line %d)") \
    EZ_ERROR("E4004", "names", "function '%s' already declared") \
    EZ_ERROR("E4005", "names", "module '%s' has no function named '%s'") \
    EZ_ERROR("E4006", "names", "name '%s' uses reserved prefix (ez_, _ez_, Ez); these are reserved for the compiler") \
    EZ_ERROR("E4007", "names", "a type with this name already exists; each struct and enum must have a unique name") \
    EZ_ERROR("E4008", "names", "main() cannot have parameters or a return type; it must be declared as do main() { }") \
    EZ_ERROR("E4012", "names", "variable '%s' shadows a type definition with the same name") \
    EZ_ERROR("E4013", "names", "variable '%s' shadows a function with the same name") \
    EZ_ERROR("E4014", "names", "variable '%s' shadows an imported module with the same name") \
    EZ_ERROR("E4015", "names", "'%s' is private and cannot be accessed from outside its file")

/* --- E5xxx: Usage Problems --- */
#define EZ_USAGE_ERRORS \
    EZ_ERROR("E5007", "usage", "cannot modify immutable %s '%s'; declare with 'mut' to allow modification") \
    EZ_ERROR("E5008", "arguments", "wrong number of arguments; the function expects a different count than was provided") \
    EZ_ERROR("E5011", "usage", "return value of '%s' is not used; assign it to a variable or use '_' to discard") \
    EZ_ERROR("E5015", "usage", "postfix ++ and -- require a variable, not a value or expression") \
    EZ_ERROR("E5023", "usage", "cannot use '%s' on type '%s'; only integer types support increment/decrement") \
    EZ_ERROR("E5024", "usage", "return type mismatch: cannot return signed '%s' as unsigned '%s'") \
    EZ_ERROR("E5025", "usage", "invalid assignment target; left side of '=' must be a variable, field, or index expression") \
    EZ_ERROR("E5026", "arguments", "argument type mismatch; the function expects a different type than what was provided")

/* --- E6xxx: Import Problems --- */
#define EZ_IMPORT_ERRORS \
    EZ_ERROR("E6001", "imports", "unknown module '@%s'")

/* --- E7xxx+: Standard Library --- */
#define EZ_STDLIB_ERRORS \
    EZ_ERROR("E7004", "stdlib", "function argument must be an integer, not a float") \
    EZ_ERROR("E7006", "stdlib", "threads.spawn() needs a function reference; use ()function_name to pass a function") \
    EZ_ERROR("E7014", "stdlib", "cannot convert %lld to char; value must be a valid Unicode code point (0 or greater)") \
    EZ_ERROR("E7015", "stdlib", "len() is not supported for type '%s'; len() works on string, array, and map types") \
    EZ_ERROR("E9002", "stdlib", "arrays.%s() requires a numeric array, got array of %s") \
    EZ_ERROR("E9005", "stdlib", "invalid range: start (%lld) must be less than end (%lld)") \
    EZ_ERROR("E12001", "stdlib", "maps.%s() requires a map argument, got an array") \
    EZ_ERROR("E12006", "stdlib", "duplicate key in map literal")

/* --- Warnings --- */
#define EZ_WARNINGS \
    EZ_WARNING("W1001", "cleanup", "variable is declared but never used; remove it or use it") \
    EZ_WARNING("W1003", "cleanup", "function is declared but never called; remove it or call it") \
    EZ_WARNING("W1005", "cleanup", "typed blank identifier; '_' doesn't need a type annotation, use 'mut _ = <expr>' instead") \
    EZ_WARNING("W2001", "cleanup", "this module is imported but never used; remove the import or use the module") \
    EZ_WARNING("W1002", "cleanup", "this import is never used; remove it or use a function from the module") \
    EZ_WARNING("W2002", "safety", "this variable shadows a variable with the same name in an outer scope") \
    EZ_WARNING("W2003", "safety", "unreachable code; this statement will never execute because it comes after a return") \
    EZ_WARNING("W2007", "safety", "this variable shadows a global constant or variable") \
    EZ_WARNING("W2011", "safety", "named return variable is not used in return; the returned value may not match the named variable") \
    EZ_WARNING("W2012", "safety", "when condition is a float; equality checks on floats are imprecise; prefer math.abs(x - y) < epsilon") \
    EZ_WARNING("W3003", "safety", "fixed-size array is not fully initialized; remaining elements will be zero-valued") \
    EZ_WARNING("W3004", "safety", "pointer may reference memory from a scope that has ended; assigning addr() of an inner-scope variable to an outer-scope pointer")

/* Look up the canonical message for a code like "E3050" or "W2001".
 * Returns NULL if the code is unknown. The returned pointer is a
 * string literal and lives for the lifetime of the program. */
const char *ez_error_message(const char *code);

#endif
