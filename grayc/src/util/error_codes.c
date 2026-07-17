/*
 * error_codes.c — Runtime lookup for error and warning messages, backed
 * by the registry in error_codes.h via sorted binary search.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "error_codes.h"
#include <stdlib.h>
#include <string.h>

typedef struct { const char *code; const char *msg; } ErrorEntry;

static ErrorEntry entries[] = {
#define GRAY_ERROR(c, cat, msg) { c, msg },
#define GRAY_WARNING(c, cat, msg) { c, msg },
#define GRAY_PANIC(c, cat, msg) { c, msg },
    GRAY_LEXER_ERRORS
    GRAY_PARSER_ERRORS
    GRAY_TYPE_ERRORS
    GRAY_REFERENCE_ERRORS
    GRAY_USAGE_ERRORS
    GRAY_IMPORT_ERRORS
    GRAY_STDLIB_ERRORS
    GRAY_BITWISE_ERRORS
    GRAY_PANIC_CODES
    GRAY_WARNINGS
#undef GRAY_ERROR
#undef GRAY_WARNING
#undef GRAY_PANIC
};

#define ENTRY_COUNT (sizeof(entries) / sizeof(entries[0]))

static int error_code_compare(const void *a, const void *b) {
    return strcmp(((const ErrorEntry *)a)->code, ((const ErrorEntry *)b)->code);
}

static int sorted = 0;

const char *gray_error_message(const char *code) {
    if (!code) return NULL;
    if (!sorted) {
        qsort(entries, ENTRY_COUNT, sizeof(ErrorEntry), error_code_compare);
        sorted = 1;
    }
    ErrorEntry key = { code, NULL };
    ErrorEntry *hit = bsearch(&key, entries, ENTRY_COUNT, sizeof(ErrorEntry), error_code_compare);
    return hit ? hit->msg : NULL;
}
