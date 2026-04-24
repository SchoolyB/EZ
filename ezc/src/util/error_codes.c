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
#include <string.h>

const char *ez_error_message(const char *code) {
    if (!code) return NULL;
#define EZ_ERROR(c, cat, msg) if (strcmp(code, c) == 0) return msg;
#define EZ_WARNING(c, cat, msg) if (strcmp(code, c) == 0) return msg;
    EZ_LEXER_ERRORS
    EZ_PARSER_ERRORS
    EZ_TYPE_ERRORS
    EZ_REFERENCE_ERRORS
    EZ_USAGE_ERRORS
    EZ_IMPORT_ERRORS
    EZ_STDLIB_ERRORS
    EZ_WARNINGS
#undef EZ_ERROR
#undef EZ_WARNING
    return NULL;
}
