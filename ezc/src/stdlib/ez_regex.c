/*
 * ez_regex.c - @regex module implementation
 *
 * Uses POSIX <regex.h> for pattern matching.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_regex.h"
#include <regex.h>
#include <string.h>
#include <stdio.h>

/* Helper: compile pattern into a null-terminated C string and regex_t.
 * Returns 0 on success, non-zero on error. Caller must regfree on success. */
static int compile_pattern(EzString pattern, regex_t *re, int flags) {
    /* Null-terminate the pattern */
    char pat_buf[4096];
    int plen = pattern.len < (int32_t)sizeof(pat_buf) - 1 ? pattern.len : (int32_t)sizeof(pat_buf) - 1;
    memcpy(pat_buf, pattern.data, (size_t)plen);
    pat_buf[plen] = '\0';

    return regcomp(re, pat_buf, flags | REG_EXTENDED);
}

/* Helper: null-terminate an EzString into a buffer */
static const char *to_cstr(EzString s, char *buf, size_t bufsz) {
    size_t len = (size_t)s.len < bufsz - 1 ? (size_t)s.len : bufsz - 1;
    memcpy(buf, s.data, len);
    buf[len] = '\0';
    return buf;
}

bool ez_regex_is_valid(EzString pattern) {
    regex_t re;
    if (compile_pattern(pattern, &re, REG_EXTENDED | REG_NOSUB) != 0) return false;
    regfree(&re);
    return true;
}

bool ez_regex_match(EzString pattern, EzString text) {
    regex_t re;
    if (compile_pattern(pattern, &re, REG_NOSUB) != 0) return false;

    char txt_buf[8192];
    to_cstr(text, txt_buf, sizeof(txt_buf));

    int result = regexec(&re, txt_buf, 0, NULL, 0);
    regfree(&re);
    return result == 0;
}

EzString ez_regex_find(EzArena *arena, EzString pattern, EzString text) {
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        return (EzString){"", 0};
    }

    char txt_buf[8192];
    to_cstr(text, txt_buf, sizeof(txt_buf));

    regmatch_t match;
    if (regexec(&re, txt_buf, 1, &match, 0) != 0) {
        regfree(&re);
        return (EzString){"", 0};
    }

    int32_t mlen = (int32_t)(match.rm_eo - match.rm_so);
    EzString result = ez_string_new(arena, txt_buf + match.rm_so, mlen);
    regfree(&re);
    return result;
}

EzArray ez_regex_find_all(EzArena *arena, EzString pattern, EzString text) {
    EzArray arr = ez_array_new(arena, sizeof(EzString), 8);
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) return arr;

    char txt_buf[8192];
    to_cstr(text, txt_buf, sizeof(txt_buf));

    const char *cursor = txt_buf;
    regmatch_t match;

    while (regexec(&re, cursor, 1, &match, 0) == 0) {
        int32_t mlen = (int32_t)(match.rm_eo - match.rm_so);
        EzString s = ez_string_new(arena, cursor + match.rm_so, mlen);
        ez_array_push(arena, &arr, &s);

        /* Advance past this match (at least 1 char to avoid infinite loop) */
        cursor += match.rm_eo;
        if (match.rm_eo == 0) cursor++;
        if (*cursor == '\0') break;
    }

    regfree(&re);
    return arr;
}

EzString ez_regex_replace(EzArena *arena, EzString pattern, EzString text, EzString replacement) {
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        return text;
    }

    char txt_buf[8192];
    to_cstr(text, txt_buf, sizeof(txt_buf));

    char repl_buf[4096];
    to_cstr(replacement, repl_buf, sizeof(repl_buf));

    /* Build result by replacing all matches */
    char result[16384];
    int pos = 0;
    const char *cursor = txt_buf;
    regmatch_t match;

    while (regexec(&re, cursor, 1, &match, 0) == 0 && pos < (int)sizeof(result) - 4096) {
        /* Copy text before match */
        int pre_len = (int)match.rm_so;
        memcpy(result + pos, cursor, (size_t)pre_len);
        pos += pre_len;

        /* Copy replacement */
        int repl_len = (int)strlen(repl_buf);
        memcpy(result + pos, repl_buf, (size_t)repl_len);
        pos += repl_len;

        /* Advance past match */
        cursor += match.rm_eo;
        if (match.rm_eo == 0) {
            if (*cursor) result[pos++] = *cursor++;
            else break;
        }
    }

    /* Copy remaining text */
    int remaining = (int)strlen(cursor);
    memcpy(result + pos, cursor, (size_t)remaining);
    pos += remaining;

    regfree(&re);
    return ez_string_new(arena, result, (int32_t)pos);
}

EzArray ez_regex_split(EzArena *arena, EzString pattern, EzString text) {
    EzArray arr = ez_array_new(arena, sizeof(EzString), 8);
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        /* On bad pattern, return array with original string */
        ez_array_push(arena, &arr, &text);
        return arr;
    }

    char txt_buf[8192];
    to_cstr(text, txt_buf, sizeof(txt_buf));

    const char *cursor = txt_buf;
    regmatch_t match;

    while (regexec(&re, cursor, 1, &match, 0) == 0) {
        /* Piece before the match */
        int32_t plen = (int32_t)match.rm_so;
        EzString piece = ez_string_new(arena, cursor, plen);
        ez_array_push(arena, &arr, &piece);

        cursor += match.rm_eo;
        if (match.rm_eo == 0) {
            if (*cursor) cursor++;
            else break;
        }
    }

    /* Remaining text after last match */
    int32_t remaining = (int32_t)strlen(cursor);
    EzString last = ez_string_new(arena, cursor, remaining);
    ez_array_push(arena, &arr, &last);

    regfree(&re);
    return arr;
}

/* _result variants */

EzResult_string ez_regex_find_result(EzArena *arena, EzString pattern, EzString text) {
    EzResult_string r;
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        r.v0 = (EzString){"", 0};
        r.v1 = ez_error_new(arena, ez_string_format(arena, "invalid regex pattern '%.*s'",
            pattern.len, pattern.data));
        return r;
    }
    regfree(&re);
    r.v0 = ez_regex_find(arena, pattern, text);
    r.v1 = NULL;
    return r;
}

EzResult_array ez_regex_find_all_result(EzArena *arena, EzString pattern, EzString text) {
    EzResult_array r;
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        r.v0 = ez_array_new(arena, sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "invalid regex pattern '%.*s'",
            pattern.len, pattern.data));
        return r;
    }
    regfree(&re);
    r.v0 = ez_regex_find_all(arena, pattern, text);
    r.v1 = NULL;
    return r;
}

EzResult_string ez_regex_replace_result(EzArena *arena, EzString pattern, EzString text, EzString replacement) {
    EzResult_string r;
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        r.v0 = text;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "invalid regex pattern '%.*s'",
            pattern.len, pattern.data));
        return r;
    }
    regfree(&re);
    r.v0 = ez_regex_replace(arena, pattern, text, replacement);
    r.v1 = NULL;
    return r;
}

EzResult_array ez_regex_split_result(EzArena *arena, EzString pattern, EzString text) {
    EzResult_array r;
    regex_t re;
    if (compile_pattern(pattern, &re, 0) != 0) {
        r.v0 = ez_array_new(arena, sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "invalid regex pattern '%.*s'",
            pattern.len, pattern.data));
        return r;
    }
    regfree(&re);
    r.v0 = ez_regex_split(arena, pattern, text);
    r.v1 = NULL;
    return r;
}
