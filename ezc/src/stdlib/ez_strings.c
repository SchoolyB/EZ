/*
 * ez_strings.c - @strings module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_strings.h"
#include <ctype.h>
#include <stdlib.h>
#include <string.h>

EzString ez_strings_to_upper(EzArena *arena, EzString s) {
    char *buf = ez_arena_alloc(arena, (size_t)s.len + 1);
    for (int32_t i = 0; i < s.len; i++) buf[i] = (char)toupper((unsigned char)s.data[i]);
    buf[s.len] = '\0';
    EzString r = { buf, s.len };
    return r;
}

EzString ez_strings_to_lower(EzArena *arena, EzString s) {
    char *buf = ez_arena_alloc(arena, (size_t)s.len + 1);
    for (int32_t i = 0; i < s.len; i++) buf[i] = (char)tolower((unsigned char)s.data[i]);
    buf[s.len] = '\0';
    EzString r = { buf, s.len };
    return r;
}

EzString ez_strings_trim(EzArena *arena, EzString s) {
    int32_t start = 0, end = s.len;
    while (start < end && isspace((unsigned char)s.data[start])) start++;
    while (end > start && isspace((unsigned char)s.data[end - 1])) end--;
    return ez_string_new(arena, s.data + start, end - start);
}

EzString ez_strings_trim_left(EzArena *arena, EzString s) {
    int32_t start = 0;
    while (start < s.len && isspace((unsigned char)s.data[start])) start++;
    return ez_string_new(arena, s.data + start, s.len - start);
}

EzString ez_strings_trim_right(EzArena *arena, EzString s) {
    int32_t end = s.len;
    while (end > 0 && isspace((unsigned char)s.data[end - 1])) end--;
    return ez_string_new(arena, s.data, end);
}

bool ez_strings_contains(EzString s, EzString sub) {
    if (sub.len == 0) return true;
    if (sub.len > s.len) return false;
    for (int32_t i = 0; i <= s.len - sub.len; i++) {
        if (memcmp(s.data + i, sub.data, (size_t)sub.len) == 0) return true;
    }
    return false;
}

bool ez_strings_starts_with(EzString s, EzString prefix) {
    if (prefix.len > s.len) return false;
    return memcmp(s.data, prefix.data, (size_t)prefix.len) == 0;
}

bool ez_strings_ends_with(EzString s, EzString suffix) {
    if (suffix.len > s.len) return false;
    return memcmp(s.data + s.len - suffix.len, suffix.data, (size_t)suffix.len) == 0;
}

int64_t ez_strings_index_of(EzString s, EzString sub) {
    if (sub.len == 0) return 0;
    if (sub.len > s.len) return -1;
    for (int32_t i = 0; i <= s.len - sub.len; i++) {
        if (memcmp(s.data + i, sub.data, (size_t)sub.len) == 0) return i;
    }
    return -1;
}

int64_t ez_strings_last_index_of(EzString s, EzString sub) {
    if (sub.len == 0) return s.len;
    if (sub.len > s.len) return -1;
    for (int32_t i = s.len - sub.len; i >= 0; i--) {
        if (memcmp(s.data + i, sub.data, (size_t)sub.len) == 0) return i;
    }
    return -1;
}

int64_t ez_strings_count(EzString s, EzString sub) {
    if (sub.len == 0) return 0;
    int64_t count = 0;
    for (int32_t i = 0; i <= s.len - sub.len; i++) {
        if (memcmp(s.data + i, sub.data, (size_t)sub.len) == 0) {
            count++;
            i += sub.len - 1;
        }
    }
    return count;
}

bool ez_strings_is_empty(EzString s) {
    return s.len == 0;
}

EzString ez_strings_remove_prefix(EzArena *arena, EzString s, EzString prefix) {
    if (prefix.len > s.len || memcmp(s.data, prefix.data, (size_t)prefix.len) != 0) return s;
    int32_t new_len = s.len - prefix.len;
    return ez_string_new(arena, s.data + prefix.len, new_len);
}

EzString ez_strings_remove_suffix(EzArena *arena, EzString s, EzString suffix) {
    if (suffix.len > s.len || memcmp(s.data + s.len - suffix.len, suffix.data, (size_t)suffix.len) != 0) return s;
    int32_t new_len = s.len - suffix.len;
    return ez_string_new(arena, s.data, new_len);
}

EzString ez_strings_replace(EzArena *arena, EzString s, EzString old_s, EzString new_s) {
    if (old_s.len == 0) return s;
    /* Count occurrences to size the buffer */
    int64_t cnt = ez_strings_count(s, old_s);
    if (cnt == 0) return s;
    int64_t new_len64 = (int64_t)s.len + cnt * ((int64_t)new_s.len - (int64_t)old_s.len);
    if (new_len64 < 0 || new_len64 > INT32_MAX) {
        ez_panic_code("P0071", "strings.replace() result exceeds maximum string length");
    }
    int32_t new_len = (int32_t)new_len64;
    char *buf = ez_arena_alloc(arena, (size_t)new_len + 1);
    int32_t pos = 0;
    for (int32_t i = 0; i < s.len; ) {
        if (i <= s.len - old_s.len && memcmp(s.data + i, old_s.data, (size_t)old_s.len) == 0) {
            memcpy(buf + pos, new_s.data, (size_t)new_s.len);
            pos += new_s.len;
            i += old_s.len;
        } else {
            buf[pos++] = s.data[i++];
        }
    }
    buf[pos] = '\0';
    EzString r = { buf, pos };
    return r;
}

EzString ez_strings_repeat(EzArena *arena, EzString s, int64_t count) {
    if (count < 0) ez_panic_code("P0072", "strings.repeat() count cannot be negative (%lld)", (long long)count);
    if (count == 0 || s.len == 0) return ez_string_lit("");
    int64_t new_len64 = (int64_t)s.len * count;
    if (new_len64 > INT32_MAX) {
        ez_panic_code("P0073", "strings.repeat() result exceeds maximum string length");
    }
    int32_t new_len = (int32_t)new_len64;
    char *buf = ez_arena_alloc(arena, (size_t)new_len + 1);
    for (int64_t i = 0; i < count; i++) {
        memcpy(buf + i * s.len, s.data, (size_t)s.len);
    }
    buf[new_len] = '\0';
    EzString r = { buf, new_len };
    return r;
}

EzString ez_strings_reverse(EzArena *arena, EzString s) {
    char *buf = ez_arena_alloc(arena, (size_t)s.len + 1);
    for (int32_t i = 0; i < s.len; i++) buf[i] = s.data[s.len - 1 - i];
    buf[s.len] = '\0';
    EzString r = { buf, s.len };
    return r;
}

EzString ez_strings_slice(EzArena *arena, EzString s, int64_t start, int64_t end) {
    if (start < 0) start = 0;
    if (end > s.len) end = s.len;
    if (start >= end) return ez_string_lit("");
    return ez_string_new(arena, s.data + start, (int32_t)(end - start));
}

EzArray ez_strings_split(EzArena *arena, EzString s, EzString sep) {
    EzArray arr = ez_array_new(arena, sizeof(EzString), 4);
    if (sep.len == 0) {
        ez_array_push(arena, &arr, &s);
        return arr;
    }
    int32_t start = 0;
    for (int32_t i = 0; i <= s.len - sep.len; i++) {
        if (memcmp(s.data + i, sep.data, (size_t)sep.len) == 0) {
            EzString part = ez_string_new(arena, s.data + start, i - start);
            ez_array_push(arena, &arr, &part);
            i += sep.len - 1;
            start = i + 1;
        }
    }
    EzString last = ez_string_new(arena, s.data + start, s.len - start);
    ez_array_push(arena, &arr, &last);
    return arr;
}

EzString ez_strings_join(EzArena *arena, EzArray arr, EzString sep) {
    if (arr.len == 0) return ez_string_lit("");
    /* Calculate total length */
    int32_t total = 0;
    for (int32_t i = 0; i < arr.len; i++) {
        EzString *s = (EzString *)((char *)arr.data + (size_t)i * sizeof(EzString));
        total += s->len;
        if (i > 0) total += sep.len;
    }
    char *buf = ez_arena_alloc(arena, (size_t)total + 1);
    int32_t pos = 0;
    for (int32_t i = 0; i < arr.len; i++) {
        if (i > 0) { memcpy(buf + pos, sep.data, (size_t)sep.len); pos += sep.len; }
        EzString *s = (EzString *)((char *)arr.data + (size_t)i * sizeof(EzString));
        memcpy(buf + pos, s->data, (size_t)s->len);
        pos += s->len;
    }
    buf[pos] = '\0';
    EzString r = { buf, pos };
    return r;
}


char ez_strings_char_at(EzString s, int64_t index) {
    if (index < 0 || index >= s.len) {
        ez_panic_code("P0082", "string index %d out of bounds (length %d)",
                      (int)index, (int)s.len);
    }
    return s.data[index];
}

bool ez_strings_is_alpha(char c)      { return isalpha((unsigned char)c) != 0; }
bool ez_strings_is_digit(char c)      { return isdigit((unsigned char)c) != 0; }
bool ez_strings_is_alnum(char c)      { return isalnum((unsigned char)c) != 0; }
bool ez_strings_is_whitespace(char c) { return isspace((unsigned char)c) != 0; }
bool ez_strings_is_upper(char c)      { return isupper((unsigned char)c) != 0; }
bool ez_strings_is_lower(char c)      { return islower((unsigned char)c) != 0; }
