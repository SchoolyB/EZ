/*
 * error_codes.c - Runtime lookup for error/warning messages
 *
 * The registry in error_codes.h is the single source of truth for both
 * ERRORS.md (via scripts/generate_errors.sh) and the messages shown to
 * users at compile time. ez_error_message() performs the string-keyed
 * lookup so emission sites never hold their own copy of the text.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "error_codes.h"
#include <stdlib.h>
#include <string.h>

typedef struct { const char *code; const char *msg; } ErrorEntry;

static ErrorEntry entries[] = {
#define EZ_ERROR(c, cat, msg) { c, msg },
#define EZ_WARNING(c, cat, msg) { c, msg },
#define EZ_PANIC(c, cat, msg) { c, msg },
    EZ_LEXER_ERRORS
    EZ_PARSER_ERRORS
    EZ_TYPE_ERRORS
    EZ_REFERENCE_ERRORS
    EZ_USAGE_ERRORS
    EZ_IMPORT_ERRORS
    EZ_STDLIB_ERRORS
    EZ_BITWISE_ERRORS
    EZ_PANIC_CODES
    EZ_WARNINGS
#undef EZ_ERROR
#undef EZ_WARNING
#undef EZ_PANIC
};

#define ENTRY_COUNT (sizeof(entries) / sizeof(entries[0]))

static int entry_cmp(const void *a, const void *b) {
    return strcmp(((const ErrorEntry *)a)->code, ((const ErrorEntry *)b)->code);
}

static int sorted = 0;

const char *ez_error_message(const char *code) {
    if (!code) return NULL;
    if (!sorted) {
        qsort(entries, ENTRY_COUNT, sizeof(ErrorEntry), entry_cmp);
        sorted = 1;
    }
    ErrorEntry key = { code, NULL };
    ErrorEntry *hit = bsearch(&key, entries, ENTRY_COUNT, sizeof(ErrorEntry), entry_cmp);
    return hit ? hit->msg : NULL;
}
