/*
 * error_codes.h - Centralized error code registry for EZC
 *
 * This is the SINGLE SOURCE OF TRUTH for all error and warning codes.
 * The scripts/generate_errors.sh script reads this file to produce ERRORS.md.
 *
 * Format: EZ_ERROR(code, category, description)
 *         EZ_WARNING(code, category, description)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_ERROR_CODES_H
#define EZC_ERROR_CODES_H

/* --- E1xxx: Reading Your Code --- */
#define EZ_LEXER_ERRORS \
    EZ_ERROR("E1010", "syntax", "number is too large — the maximum integer value is 9223372036854775807")

/* --- E2xxx: Understanding Your Code --- */
#define EZ_PARSER_ERRORS \
    EZ_ERROR("E2001", "syntax", "unexpected symbol — the compiler found something it did not expect here") \
    EZ_ERROR("E2002", "syntax", "missing symbol — a bracket, parenthesis, or keyword is missing") \
    EZ_ERROR("E2050", "syntax", "break and continue can only be used inside a loop") \
    EZ_ERROR("E2051", "syntax", "functions cannot be defined inside other functions — move it to the top level")

/* --- E3xxx: Type Problems --- */
#define EZ_TYPE_ERRORS \
    EZ_ERROR("E3001", "types", "wrong type — you are using a value of one type where a different type is expected") \
    EZ_ERROR("E3002", "types", "this operator does not work on this type — for example, you cannot subtract strings") \
    EZ_ERROR("E3005", "types", "cannot change a constant — use mut instead of const if you need to modify this value") \
    EZ_ERROR("E3006", "types", "return value problem — either returning a value from a function that should not return one, or missing a return value") \
    EZ_ERROR("E3007", "types", "++ and -- only work on numbers — you cannot increment or negate a non-number") \
    EZ_ERROR("E3008", "types", "cannot use [] on this type — only arrays, maps, and strings support indexing") \
    EZ_ERROR("E3009", "types", "cannot loop over this type — for_each only works with arrays, maps, and strings") \
    EZ_ERROR("E3010", "types", "this struct does not have a field with that name") \
    EZ_ERROR("E3011", "types", "a type name like int or string cannot be used as a value — did you forget to declare a variable?") \
    EZ_ERROR("E3012", "types", "addr() needs a variable, field, or array element — you cannot take the address of a literal like 42") \
    EZ_ERROR("E3013", "types", "only structs have fields — you cannot use .field on a number, string, or bool") \
    EZ_ERROR("E3016", "types", "only pointers can be dereferenced with ^ — this value is not a pointer")

/* --- E4xxx: Name Problems --- */
#define EZ_REFERENCE_ERRORS \
    EZ_ERROR("E4001", "names", "this variable does not exist — check the spelling or make sure it is declared above this line") \
    EZ_ERROR("E4002", "names", "this function does not exist — check the spelling or make sure it is defined") \
    EZ_ERROR("E4003", "names", "a variable with this name already exists in this scope — use a different name") \
    EZ_ERROR("E4004", "names", "a function with this name already exists — each function must have a unique name") \
    EZ_ERROR("E4005", "names", "no main() function found — every EZ program needs a do main() { } function") \
    EZ_ERROR("E4006", "names", "this name is reserved by the compiler — choose a different name that does not start with ez_ or Ez")

/* --- E5xxx: Wrong Usage --- */
#define EZ_RUNTIME_ERRORS \
    EZ_ERROR("E5008", "arguments", "wrong number of arguments — the function expects a different number of values than you provided") \
    EZ_ERROR("R5001", "runtime", "nil pointer dereference — you tried to use a pointer that has no value (nil)") \
    EZ_ERROR("R5002", "runtime", "array index out of bounds — you tried to access an element that does not exist") \
    EZ_ERROR("R5003", "runtime", "division by zero — you cannot divide a number by zero") \
    EZ_ERROR("R5004", "runtime", "key not found in map — the key you are looking for does not exist") \
    EZ_ERROR("R5005", "runtime", "stack overflow — too many nested function calls (possible infinite recursion)") \
    EZ_ERROR("R5006", "runtime", "assertion failed — a condition checked with assert() was false")

/* --- E6xxx: Import Problems --- */
#define EZ_IMPORT_ERRORS \
    EZ_ERROR("E6001", "imports", "unknown module — this is not a built-in EZ module. Check the spelling or see the docs for available modules")

/* --- E7xxx: Standard Library --- */
#define EZ_STDLIB_ERRORS \
    EZ_ERROR("E7006", "stdlib", "threads.spawn() needs a function reference — use ()function_name to pass a function")

/* --- Warnings --- */
#define EZ_WARNINGS \
    EZ_WARNING("W2001", "cleanup", "this module is imported but never used — remove the import or use the module") \
    EZ_WARNING("W3001", "safety", "this function might not always return a value — make sure all paths through the function end with a return")

#endif
