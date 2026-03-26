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

/* --- E1xxx: Lexer Errors --- */
#define EZ_LEXER_ERRORS \
    EZ_ERROR("E1010", "lexer", "integer literal overflows 64-bit integer")

/* --- E2xxx: Parser Errors --- */
#define EZ_PARSER_ERRORS \
    EZ_ERROR("E2001", "parser", "unexpected token in expression or statement") \
    EZ_ERROR("E2002", "parser", "expected token not found (missing brace, paren, etc.)") \
    EZ_ERROR("E2050", "parser", "break or continue used outside of a loop") \
    EZ_ERROR("E2051", "parser", "nested function declarations are not allowed")

/* --- E3xxx: Type Errors --- */
#define EZ_TYPE_ERRORS \
    EZ_ERROR("E3001", "type", "type mismatch (assignment, argument, return, comparison, struct field)") \
    EZ_ERROR("E3002", "type", "invalid operator on type (string arithmetic, ordering)") \
    EZ_ERROR("E3005", "type", "cannot modify a constant (const reassignment, index, field, ++)") \
    EZ_ERROR("E3006", "type", "return value issue (value in void function, bare return in non-void)") \
    EZ_ERROR("E3007", "type", "operator on incompatible type (++/-- or negation on non-numeric)") \
    EZ_ERROR("E3008", "type", "indexing a non-indexable type (only arrays, maps, strings)") \
    EZ_ERROR("E3009", "type", "for_each on non-iterable type (only arrays, maps, strings)") \
    EZ_ERROR("E3010", "type", "struct has no such field (literal or access)") \
    EZ_ERROR("E3011", "type", "type keyword used as a value (mut x = int)") \
    EZ_ERROR("E3012", "type", "addr() requires an lvalue (variable, field, or index)") \
    EZ_ERROR("E3013", "type", "field access on non-struct type") \
    EZ_ERROR("E3016", "type", "cannot dereference non-pointer type") \
    EZ_ERROR("E3018", "type", "cannot use module before importing it")

/* --- E4xxx: Reference/Name Errors --- */
#define EZ_REFERENCE_ERRORS \
    EZ_ERROR("E4001", "reference", "undefined variable") \
    EZ_ERROR("E4002", "reference", "undefined function") \
    EZ_ERROR("E4003", "reference", "variable already declared in this scope") \
    EZ_ERROR("E4004", "reference", "function already declared") \
    EZ_ERROR("E4005", "reference", "program has no main() function") \
    EZ_ERROR("E4006", "reference", "name uses reserved compiler prefix (ez_, _ez_, Ez)")

/* --- E5xxx: Runtime Errors --- */
#define EZ_RUNTIME_ERRORS \
    EZ_ERROR("E5008", "runtime", "wrong number of function arguments")

/* --- E6xxx: Import Errors --- */
#define EZ_IMPORT_ERRORS \
    EZ_ERROR("E6001", "import", "unknown standard library module")

/* --- E7xxx: Stdlib Validation Errors --- */
#define EZ_STDLIB_ERRORS \
    EZ_ERROR("E7006", "stdlib", "threads.spawn() requires a function reference")

/* --- Warnings --- */
#define EZ_WARNINGS \
    EZ_WARNING("W2001", "unused", "imported module is not used") \
    EZ_WARNING("W3001", "type", "function may not return a value on all paths")

#endif
