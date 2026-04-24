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
    EZ_ERROR("E1003", "syntax", "multi-line comment was never closed — add */ to close it") \
    EZ_ERROR("E1005", "syntax", "character literal was never closed — add a closing single quote") \
    EZ_ERROR("E1006", "syntax", "invalid escape sequence in string — valid escapes are \\n \\t \\\\ \\\" and \\x") \
    EZ_ERROR("E1007", "syntax", "invalid escape sequence in character literal") \
    EZ_ERROR("E1010", "syntax", "invalid number format — hex (0x), octal (0o), or binary (0b) prefix must be followed by digits") \
    EZ_ERROR("E1011", "syntax", "number cannot have consecutive underscores") \
    EZ_ERROR("E1012", "syntax", "numeric literals cannot start with an underscore") \
    EZ_ERROR("E1013", "syntax", "number cannot end with an underscore") \
    EZ_ERROR("E1014", "syntax", "number cannot have an underscore before the decimal point") \
    EZ_ERROR("E1015", "syntax", "number cannot have an underscore after the decimal point") \
    EZ_ERROR("E1016", "syntax", "number cannot end with a trailing decimal point — add a digit after the dot") \
    EZ_ERROR("E1017", "syntax", "raw string literal was never closed — add a closing backtick") \
    EZ_ERROR("E1018", "syntax", "char literal must contain exactly one character — use a string for multiple characters") \
    EZ_ERROR("E1019", "syntax", "unexpected '#' character — use '//' for comments") \
    EZ_ERROR("E1020", "syntax", "unexpected '|' character — use '||' for logical OR") \
    EZ_ERROR("E1021", "syntax", "string literal was never closed — add a closing double quote") \
    EZ_ERROR("E1022", "syntax", "unexpected character")

/* --- E2xxx: Understanding Your Code (Parser) --- */
#define EZ_PARSER_ERRORS \
    EZ_ERROR("E2001", "syntax", "unexpected symbol — the compiler found something it did not expect here") \
    EZ_ERROR("E2002", "syntax", "missing symbol — a bracket, parenthesis, or keyword is missing") \
    EZ_ERROR("E2010", "syntax", "cannot use a module before importing it — add the import statement before using") \
    EZ_ERROR("E2011", "syntax", "constants must have a value — add = followed by a value after the type") \
    EZ_ERROR("E2012", "syntax", "duplicate parameter name — each parameter must have a unique name") \
    EZ_ERROR("E2013", "syntax", "duplicate struct field name — each field must have a unique name") \
    EZ_ERROR("E2014", "syntax", "duplicate enum variant name — each variant must have a unique name") \
    EZ_ERROR("E2015", "syntax", "duplicate field in struct literal — each field can only be initialized once") \
    EZ_ERROR("E2016", "syntax", "empty enum — an enum must have at least one value") \
    EZ_ERROR("E2017", "syntax", "trailing comma — remove the extra comma before the closing bracket or brace") \
    EZ_ERROR("E2025", "syntax", "expected integer for array size — the second value in [type, size] must be a positive integer") \
    EZ_ERROR("E2036", "syntax", "imports must be at the top of the file, not inside a function") \
    EZ_ERROR("E2037", "syntax", "duplicate function name in struct — each function must have a unique name") \
    EZ_ERROR("E2038", "syntax", "reserved name for struct or enum — this name is used by the language") \
    EZ_ERROR("E2039", "syntax", "required parameter cannot appear after a parameter with a default value") \
    EZ_ERROR("E2043", "syntax", "duplicate case value in when statement — each case must be unique") \
    EZ_ERROR("E2050", "syntax", "break and continue can only be used inside a loop") \
    EZ_ERROR("E2051", "syntax", "functions cannot be defined inside other functions — move it to the top level") \
    EZ_ERROR("E2053", "syntax", "structs and enums must be defined at the file scope, not inside a function") \
    EZ_ERROR("E2056", "syntax", "executable statements are not allowed at file scope — put code inside do main() { }") \
    EZ_ERROR("E2057", "syntax", "invalid interpolation syntax — use ${variable} instead of $variable") \
    EZ_ERROR("E2058", "syntax", "nested type declarations are not allowed — structs and enums must be defined at the file scope") \
    EZ_ERROR("E2059", "syntax", "empty when block — a when statement must have at least one 'is' branch") \
    EZ_ERROR("E2060", "syntax", "too many return values — a function can return at most 16 values") \
    EZ_ERROR("E2061", "syntax", "'module' declarations are not supported — imported files are identified by their file path") \
    EZ_ERROR("E2062", "syntax", "too many variables in multi-variable declaration — maximum is 16") \
    EZ_ERROR("E2063", "syntax", "duplicate or conflicting named return value — each name must be unique and not collide with parameters") \
    EZ_ERROR("E2064", "syntax", "struct function name conflicts with a field name — functions and fields must have distinct names") \
    EZ_ERROR("E2065", "syntax", "enum variant cannot have the same name as its enum type") \
    EZ_ERROR("E2066", "syntax", "struct field cannot have the same name as its struct type") \
    EZ_ERROR("E2067", "syntax", "empty struct — a struct must have at least one field") \
    EZ_ERROR("E2068", "syntax", "structs and enums must be declared with 'const', not 'mut'") \
    EZ_ERROR("E2069", "syntax", "unexpected semicolon — statements and declarations are separated by newlines, not semicolons") \
    EZ_ERROR("E2070", "syntax", "wildcard type '?' is only allowed in function parameter and return types — not in variable declarations, struct fields, or enum types") \
    EZ_ERROR("E2071", "syntax", "empty string interpolation '${}' — interpolation requires an expression between the braces") \
    EZ_ERROR("E2072", "syntax", "'&' is not a valid operator — use 'addr(x)' to take the address of a variable")

/* --- E3xxx: Type Problems (Typechecker) --- */
#define EZ_TYPE_ERRORS \
    EZ_ERROR("E3001", "types", "wrong type — you are using a value of one type where a different type is expected") \
    EZ_ERROR("E3002", "types", "this operator does not work on this type — for example, you cannot subtract strings") \
    EZ_ERROR("E3003", "types", "invalid array index type — array indices must be integers") \
    EZ_ERROR("E3005", "types", "cannot change a constant — use mut instead of const if you need to modify this value") \
    EZ_ERROR("E3006", "types", "return value problem — either returning a value from a function that should not return one, or missing a return value") \
    EZ_ERROR("E3007", "types", "++ and -- only work on integers — you cannot increment or decrement a non-integer") \
    EZ_ERROR("E3008", "types", "cannot use [] on this type — only arrays, maps, and strings support indexing") \
    EZ_ERROR("E3009", "types", "cannot loop over this type — for_each only works with arrays, maps, and strings") \
    EZ_ERROR("E3010", "types", "this struct does not have a field with that name") \
    EZ_ERROR("E3011", "types", "a type name like int or string cannot be used as a value — did you forget to declare a variable?") \
    EZ_ERROR("E3012", "types", "addr() needs a variable, field, or array element — you cannot take the address of a literal like 42") \
    EZ_ERROR("E3013", "types", "only structs have fields — you cannot use .field on a number, string, or bool") \
    EZ_ERROR("E3015", "types", "this value is not a function and cannot be called") \
    EZ_ERROR("E3016", "types", "only pointers can be dereferenced with ^ — this value is not a pointer") \
    EZ_ERROR("E3017", "types", "fmt.printf/sprintf cannot format composite types — use println() or access individual fields") \
    EZ_ERROR("E3018", "types", "type mismatch in when/is — the case value type does not match the scrutinee type") \
    EZ_ERROR("E3019", "types", "cannot assign a signed integer to an unsigned type — the value may be negative") \
    EZ_ERROR("E3024", "types", "this function must return a value but the body has no return statement") \
    EZ_ERROR("E3027", "types", "cannot pass a constant to a mutable parameter — the function wants to modify this value") \
    EZ_ERROR("E3031", "types", "function name cannot be used as a value — call it with '()' or take a reference with '()name'") \
    EZ_ERROR("E3032", "types", "cannot compare different enum types — they are never equal") \
    EZ_ERROR("E3033", "types", "duplicate value in enum — each enum member must have a unique value") \
    EZ_ERROR("E3034", "types", "'any' type is reserved for internal use and cannot be used in declarations") \
    EZ_ERROR("E3035", "types", "not all code paths return a value — add return statements to all branches") \
    EZ_ERROR("E3036", "types", "value is out of range for this type — for example, 200 does not fit in an i8 (-128 to 127)") \
    EZ_ERROR("E3038", "types", "'void' cannot be used as a variable type or in expressions like typeof()") \
    EZ_ERROR("E3039", "types", "ensure expects a function call — for example: ensure close(file)") \
    EZ_ERROR("E3040", "types", "this function returns multiple values but you are assigning to a single variable — use mut a, b = func()") \
    EZ_ERROR("E3041", "types", "cannot interpolate expression — interpolation supports primitives, strings, arrays, and maps") \
    EZ_ERROR("E3042", "types", "struct functions must be called on the type, not an instance — use Type.func() instead of variable.func()") \
    EZ_ERROR("E3043", "types", "cannot cast between incompatible types — only numeric, enum, and string conversions are allowed") \
    EZ_ERROR("E3044", "types", "cannot access a field on a struct type — use an instance variable instead") \
    EZ_ERROR("E3045", "types", "or_return requires a function that returns (T, Error) — the called function does not return an error") \
    EZ_ERROR("E3046", "types", "integer literal overflows 64-bit integer — max value is 9223372036854775807") \
    EZ_ERROR("E3047", "types", "this enum does not have a member with that name") \
    EZ_ERROR("E3048", "types", "operator '+' is not defined for strings — use string interpolation or fmt.format() instead") \
    EZ_ERROR("E3049", "types", "arithmetic operators are not valid on enum values — enums only support == and != comparisons") \
    EZ_ERROR("E3050", "types", "array literal requires a type annotation — declare as [T] (e.g., mut x [int] = {1, 2, 3})") \
    EZ_ERROR("E3051", "types", "map literal requires a type annotation — declare as map[K:V] (e.g., mut x map[string:int] = {\"a\": 1})") \
    EZ_ERROR("E3052", "types", "too many elements in array initializer — declared size is %d, got %d") \
    EZ_ERROR("E3053", "types", "type mismatch in array initializer — expected '%s', got '%s'") \
    EZ_ERROR("E3054", "types", "mutable arrays cannot have a fixed size — remove the size or use 'const'") \
    EZ_ERROR("E3055", "types", "const arrays must have a fixed size — declare as [T, N]") \
    EZ_ERROR("E3056", "types", "non-exhaustive #strict when — all enum variants must be handled or add a default branch") \
    EZ_ERROR("E3057", "types", "type cannot be used as a map key — only primitive types (int, string, bool, char, byte, float) and enums are hashable") \
    EZ_ERROR("E3058", "types", "generic function instantiation failed — the body operation is not supported for the concrete type at this call site") \
    EZ_ERROR("E3059", "types", "maps cannot be declared const — use 'mut' for maps or a struct for fixed data") \
    EZ_ERROR("E3060", "types", "wildcard '?' in return type cannot be resolved — at least one parameter must also use '?' to bind the concrete type") \
    EZ_ERROR("E3061", "types", "struct cannot contain itself by value — use a pointer field '^T' for recursive types") \
    EZ_ERROR("E3062", "types", "handle types (channels, mutexes, threads) cannot be declared const — use 'mut'") \
    EZ_ERROR("E3063", "types", "cannot return address of local variable — the variable's memory is freed when the function returns") \
    EZ_ERROR("E3064", "types", "mem.destroy() called twice on the same arena — each arena can only be destroyed once") \
    EZ_ERROR("E3065", "types", "bare 'func' is not a valid type — use func(<params>) -> <return> with an explicit signature") \
    EZ_ERROR("E3066", "types", "function reference signature mismatch — expected and actual function types differ") \
    EZ_ERROR("E3067", "types", "argument passed by value to a '&' parameter — pass an lvalue (variable) instead") \
    EZ_ERROR("E3068", "types", "'void' is not a user-facing type — omit the '-> R' clause to declare a function with no return value") \
    EZ_ERROR("E3069", "types", "'&' on a parameter must come before the name, not the type — write '&name type' to mark a parameter mutable")

/* --- E4xxx: Name Problems (References) --- */
#define EZ_REFERENCE_ERRORS \
    EZ_ERROR("E4001", "names", "this variable does not exist — check the spelling or make sure it is declared above this line") \
    EZ_ERROR("E4002", "names", "this function does not exist — check the spelling or make sure it is defined") \
    EZ_ERROR("E4003", "names", "a variable with this name already exists in this scope — use a different name") \
    EZ_ERROR("E4004", "names", "a function with this name already exists — each function must have a unique name") \
    EZ_ERROR("E4005", "names", "no main() function found — every program needs a do main() { } function") \
    EZ_ERROR("E4006", "names", "this name is reserved by the compiler — choose a different name that does not start with ez_ or Ez") \
    EZ_ERROR("E4007", "names", "a type with this name already exists — each struct and enum must have a unique name") \
    EZ_ERROR("E4008", "names", "main() cannot have parameters or a return type — it must be declared as do main() { }") \
    EZ_ERROR("E4012", "names", "variable shadows a type definition with the same name") \
    EZ_ERROR("E4013", "names", "variable shadows a function with the same name") \
    EZ_ERROR("E4014", "names", "variable shadows an imported module name") \
    EZ_ERROR("E4015", "names", "cannot access a private function or constant from outside its file")

/* --- E5xxx: Usage Problems --- */
#define EZ_USAGE_ERRORS \
    EZ_ERROR("E5007", "usage", "cannot modify an immutable value — declare with mut to allow modification") \
    EZ_ERROR("E5008", "arguments", "wrong number of arguments — the function expects a different number of values than you provided") \
    EZ_ERROR("E5011", "usage", "return value of function is not used — assign it to a variable or use _ to discard") \
    EZ_ERROR("E5015", "usage", "postfix ++ and -- require a variable, not a literal or expression") \
    EZ_ERROR("E5023", "usage", "++ and -- only work on integer types, not floats") \
    EZ_ERROR("E5024", "usage", "return type mismatch — cannot return a signed value as an unsigned type") \
    EZ_ERROR("E5025", "usage", "invalid assignment target — left side of '=' must be a variable, field, or index expression") \
    EZ_ERROR("E5026", "arguments", "argument type mismatch — the function expects a different type than what was provided")

/* --- E6xxx: Import Problems --- */
#define EZ_IMPORT_ERRORS \
    EZ_ERROR("E6001", "imports", "unknown module — this is not a built-in module. Check the spelling or see the docs for available modules")

/* --- E7xxx+: Standard Library --- */
#define EZ_STDLIB_ERRORS \
    EZ_ERROR("E7004", "stdlib", "function argument must be an integer, not a float") \
    EZ_ERROR("E7006", "stdlib", "threads.spawn() needs a function reference — use ()function_name to pass a function") \
    EZ_ERROR("E7014", "stdlib", "cannot convert negative value to char — must be a valid Unicode code point") \
    EZ_ERROR("E7015", "stdlib", "len() only works on string, array, and map types — other types have no meaningful length") \
    EZ_ERROR("E9002", "stdlib", "array operation requires a numeric array — cannot use sum/min/max on string or bool arrays") \
    EZ_ERROR("E9005", "stdlib", "range bounds must be valid — start must be less than end") \
    EZ_ERROR("E12001", "stdlib", "maps function requires a map argument, not an array") \
    EZ_ERROR("E12006", "stdlib", "duplicate key in map literal — each key must be unique")

/* --- Warnings --- */
#define EZ_WARNINGS \
    EZ_WARNING("W1001", "cleanup", "variable is declared but never used — remove it or use it") \
    EZ_WARNING("W1003", "cleanup", "function is declared but never called — remove it or call it") \
    EZ_WARNING("W1005", "cleanup", "typed blank identifier — '_' doesn't need a type annotation, use 'mut _ = <expr>' instead") \
    EZ_WARNING("W2001", "cleanup", "this module is imported but never used — remove the import or use the module") \
    EZ_WARNING("W1002", "cleanup", "this import is never used — remove it or use a function from the module") \
    EZ_WARNING("W2002", "safety", "this variable shadows a variable with the same name in an outer scope") \
    EZ_WARNING("W2003", "safety", "unreachable code — this statement will never execute because it comes after a return") \
    EZ_WARNING("W2007", "safety", "this variable shadows a global constant or variable") \
    EZ_WARNING("W2011", "safety", "named return variable is not used in return — the returned value may not match the named variable") \
    EZ_WARNING("W2012", "safety", "when condition is a float — equality checks on floats are imprecise; prefer math.abs(x - y) < epsilon") \
    EZ_WARNING("W3003", "safety", "fixed-size array is not fully initialized — remaining elements will be zero-valued") \
    EZ_WARNING("W3004", "safety", "pointer may reference memory from a scope that has ended — assigning addr() of an inner-scope variable to an outer-scope pointer")

#endif
