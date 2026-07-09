/*
 * ez_json.c - @json module implementation
 *
 * Minimal recursive descent JSON parser.
 * Supports: strings, numbers, bools, null, objects, arrays.
 * Objects are parsed as EzMap[string:string] (values stored as strings).
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_json.h"
#include <string.h>
#include <stdio.h>
#include <ctype.h>
#include <inttypes.h>

/* --- Encoder --- */

/* Exact byte count that json_append_escaped would write (includes quotes). */
size_t json_escaped_len(EzString s) {
    size_t n = 2; /* opening + closing quote */
    for (int32_t i = 0; i < s.len; i++) {
        unsigned char c = (unsigned char)s.data[i];
        if (c == '"' || c == '\\') n += 2;
        else if (c == '\b' || c == '\f' || c == '\n' || c == '\r' || c == '\t') n += 2;
        else if (c < 0x20) n += 6; /* \uXXXX */
        else n += 1;
    }
    return n;
}

void json_append_escaped(char *buf, int *pos, EzString s) {
    static const char hex[] = "0123456789abcdef";
    buf[(*pos)++] = '"';
    for (int32_t i = 0; i < s.len; i++) {
        unsigned char c = (unsigned char)s.data[i];
        if (c == '"') { buf[(*pos)++] = '\\'; buf[(*pos)++] = '"'; }
        else if (c == '\\') { buf[(*pos)++] = '\\'; buf[(*pos)++] = '\\'; }
        else if (c == '\b') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 'b'; }
        else if (c == '\f') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 'f'; }
        else if (c == '\n') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 'n'; }
        else if (c == '\r') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 'r'; }
        else if (c == '\t') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 't'; }
        else if (c < 0x20) {
            buf[(*pos)++] = '\\'; buf[(*pos)++] = 'u';
            buf[(*pos)++] = '0'; buf[(*pos)++] = '0';
            buf[(*pos)++] = hex[c >> 4]; buf[(*pos)++] = hex[c & 0xf];
        }
        else buf[(*pos)++] = (char)c;
    }
    buf[(*pos)++] = '"';
}

/* Helper: byte length of a map[string:string] value in JSON output.
 * All string values are always quoted — never infer JSON types from content. */
static size_t json_map_val_len(EzString *val) {
    return json_escaped_len(*val);
}

EzString ez_json_encode_map(EzArena *arena, EzMap *m) {
    /* Pass 1: compute exact size */
    size_t need = 2; /* { } */
    int counted = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (counted > 0) need += 1; /* comma */
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        EzString *val = (EzString *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        need += json_escaped_len(*key) + 1 /* colon */ + json_map_val_len(val);
        counted++;
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '{';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; }
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        EzString *val = (EzString *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        json_append_escaped(buf, &pos, *key);
        buf[pos++] = ':';
        json_append_escaped(buf, &pos, *val);
        entry++;
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    EzString r = { buf, (int32_t)pos };
    return r;
}

/* --- Array Encoders --- */

EzString ez_json_encode_array_int(EzArena *arena, EzArray *arr) {
    /* 21 chars max per int64 + comma, plus brackets + nul */
    size_t need = 2 + (arr->len > 0 ? (size_t)arr->len * 22 - 1 : 0);
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '[';
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) { buf[pos++] = ','; }
        int64_t val = *(int64_t *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        int w = snprintf(buf + pos, need + 1 - (size_t)pos, "%" PRId64, val);
        if (w > 0 && (size_t)w < need + 1 - (size_t)pos) pos += w;
        else { pos = (int)need; break; }
    }
    buf[pos++] = ']';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

EzString ez_json_encode_array_float(EzArena *arena, EzArray *arr) {
    /* 24 chars max per %g double + comma, plus brackets + nul */
    size_t need = 2 + (arr->len > 0 ? (size_t)arr->len * 25 - 1 : 0);
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '[';
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) { buf[pos++] = ','; }
        double val = *(double *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        int w = snprintf(buf + pos, need + 1 - (size_t)pos, "%g", val);
        if (w > 0 && (size_t)w < need + 1 - (size_t)pos) pos += w;
        else { pos = (int)need; break; }
    }
    buf[pos++] = ']';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

EzString ez_json_encode_array_string(EzArena *arena, EzArray *arr) {
    /* Pass 1: exact size */
    size_t need = 2; /* [ ] */
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) need += 1; /* comma */
        EzString *val = (EzString *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        need += json_escaped_len(*val);
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '[';
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) { buf[pos++] = ','; }
        EzString *val = (EzString *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        json_append_escaped(buf, &pos, *val);
    }
    buf[pos++] = ']';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

EzString ez_json_encode_array_bool(EzArena *arena, EzArray *arr) {
    /* Pass 1: exact size */
    size_t need = 2; /* [ ] */
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) need += 1;
        bool val = *(bool *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        need += val ? 4 : 5;
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '[';
    for (int32_t i = 0; i < arr->len; i++) {
        if (i > 0) { buf[pos++] = ','; }
        bool val = *(bool *)((char *)arr->data + (size_t)i * (size_t)arr->elem_size);
        if (val) { memcpy(buf + pos, "true", 4); pos += 4; }
        else { memcpy(buf + pos, "false", 5); pos += 5; }
    }
    buf[pos++] = ']';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

/* --- Typed Map Encoders --- */

EzString ez_json_encode_map_int(EzArena *arena, EzMap *m) {
    /* Pass 1: exact size — key (escaped) + colon + int (max 21) */
    size_t need = 2;
    int counted = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (counted > 0) need += 1;
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        need += json_escaped_len(*key) + 1 + 21;
        counted++;
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '{';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; }
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        int64_t *val = (int64_t *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        json_append_escaped(buf, &pos, *key);
        buf[pos++] = ':';
        int w = snprintf(buf + pos, need + 1 - (size_t)pos, "%" PRId64, *val);
        if (w > 0 && (size_t)w < need + 1 - (size_t)pos) pos += w;
        else { pos = (int)need; break; }
        entry++;
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

EzString ez_json_encode_map_float(EzArena *arena, EzMap *m) {
    /* Pass 1: exact size — key (escaped) + colon + float (max 24) */
    size_t need = 2;
    int counted = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (counted > 0) need += 1;
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        need += json_escaped_len(*key) + 1 + 24;
        counted++;
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '{';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; }
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        double *val = (double *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        json_append_escaped(buf, &pos, *key);
        buf[pos++] = ':';
        int w = snprintf(buf + pos, need + 1 - (size_t)pos, "%g", *val);
        if (w > 0 && (size_t)w < need + 1 - (size_t)pos) pos += w;
        else { pos = (int)need; break; }
        entry++;
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

EzString ez_json_encode_map_bool(EzArena *arena, EzMap *m) {
    /* Pass 1: exact size */
    size_t need = 2;
    int counted = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (counted > 0) need += 1;
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        bool *val = (bool *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        need += json_escaped_len(*key) + 1 + (*val ? 4 : 5);
        counted++;
    }
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '{';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; }
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        bool *val = (bool *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        json_append_escaped(buf, &pos, *key);
        buf[pos++] = ':';
        if (*val) { memcpy(buf + pos, "true", 4); pos += 4; }
        else { memcpy(buf + pos, "false", 5); pos += 5; }
        entry++;
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return (EzString){ buf, (int32_t)pos };
}

/* --- Decoder --- */

static void skip_ws(const char **s, const char *end) {
    while (*s < end && isspace((unsigned char)**s)) (*s)++;
}

static EzString parse_json_string(EzArena *arena, const char **s, const char *end) {
    if (**s != '"') return ez_string_lit("");
    (*s)++;
    const char *start = *s;
    while (*s < end && **s != '"') {
        if (**s == '\\') (*s)++;
        (*s)++;
    }
    EzString r = ez_string_new(arena, start, (int32_t)(*s - start));
    if (*s < end) (*s)++; /* skip closing quote */
    return r;
}

static EzString parse_json_value_as_string(EzArena *arena, const char **s, const char *end) {
    skip_ws(s, end);
    if (*s >= end) return ez_string_lit("");

    if (**s == '"') {
        return parse_json_string(arena, s, end);
    }

    /* Number, bool, null — read until delimiter */
    const char *start = *s;
    while (*s < end && **s != ',' && **s != '}' && **s != ']' && !isspace((unsigned char)**s)) (*s)++;
    return ez_string_new(arena, start, (int32_t)(*s - start));
}

EzMap ez_json_decode(EzArena *arena, EzString text) {
    EzMap m = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 8);
    const char *s = text.data;
    const char *end = s + text.len;
    skip_ws(&s, end);
    if (s >= end || *s != '{') return m;
    s++; /* skip { */

    while (s < end) {
        skip_ws(&s, end);
        if (s >= end || *s == '}') break;

        EzString key = parse_json_string(arena, &s, end);
        skip_ws(&s, end);
        if (s < end && *s == ':') s++;
        EzString val = parse_json_value_as_string(arena, &s, end);
        ez_map_set(arena, &m, &key, &val);

        skip_ws(&s, end);
        if (s < end && *s == ',') s++;
    }
    return m;
}

/* --- Validator (#1498) ---
 *
 * Proper recursive descent validator. The old implementation just
 * peeked at the first non-whitespace character and dispatched on it,
 * which meant anything starting with '{', '[', '"', a digit, '-',
 * 't', 'f', or 'n' was silently accepted (so `{broken` was "valid").
 * The new one walks the full grammar and requires the consumed
 * region to end at text.len with only trailing whitespace.
 *
 * Mutually recursive with v_array / v_object because a JSON value
 * can be an object or array of values. Parameters are (const char **s,
 * const char *end) so each helper advances `*s` on success and leaves
 * it unspecified on failure. */

static bool v_value(const char **s, const char *end);

static void v_skip_ws(const char **s, const char *end) {
    while (*s < end && isspace((unsigned char)**s)) (*s)++;
}

static bool v_string_lit(const char **s, const char *end) {
    if (*s >= end || **s != '"') return false;
    (*s)++;
    while (*s < end && **s != '"') {
        unsigned char c = (unsigned char)**s;
        if (c == '\\') {
            (*s)++;
            if (*s >= end) return false;
            char esc = **s;
            if (esc == '"' || esc == '\\' || esc == '/' || esc == 'b' ||
                esc == 'f' || esc == 'n' || esc == 'r' || esc == 't') {
                (*s)++;
            } else if (esc == 'u') {
                (*s)++;
                for (int i = 0; i < 4; i++) {
                    if (*s >= end || !isxdigit((unsigned char)**s)) return false;
                    (*s)++;
                }
            } else {
                return false;
            }
        } else if (c < 0x20) {
            /* Control characters must be escaped per RFC 8259. */
            return false;
        } else {
            (*s)++;
        }
    }
    if (*s >= end) return false;
    (*s)++; /* skip closing " */
    return true;
}

static bool v_number(const char **s, const char *end) {
    if (*s >= end) return false;
    if (**s == '-') (*s)++;
    if (*s >= end) return false;
    if (**s == '0') {
        (*s)++;
    } else if (**s >= '1' && **s <= '9') {
        while (*s < end && isdigit((unsigned char)**s)) (*s)++;
    } else {
        return false;
    }
    /* Fractional part */
    if (*s < end && **s == '.') {
        (*s)++;
        if (*s >= end || !isdigit((unsigned char)**s)) return false;
        while (*s < end && isdigit((unsigned char)**s)) (*s)++;
    }
    /* Exponent */
    if (*s < end && (**s == 'e' || **s == 'E')) {
        (*s)++;
        if (*s < end && (**s == '+' || **s == '-')) (*s)++;
        if (*s >= end || !isdigit((unsigned char)**s)) return false;
        while (*s < end && isdigit((unsigned char)**s)) (*s)++;
    }
    return true;
}

static bool v_literal(const char **s, const char *end, const char *lit) {
    size_t n = strlen(lit);
    if ((size_t)(end - *s) < n) return false;
    if (memcmp(*s, lit, n) != 0) return false;
    *s += n;
    return true;
}

static bool v_array(const char **s, const char *end) {
    if (*s >= end || **s != '[') return false;
    (*s)++;
    v_skip_ws(s, end);
    if (*s < end && **s == ']') { (*s)++; return true; }
    for (;;) {
        v_skip_ws(s, end);
        if (!v_value(s, end)) return false;
        v_skip_ws(s, end);
        if (*s >= end) return false;
        if (**s == ',') { (*s)++; continue; }
        if (**s == ']') { (*s)++; return true; }
        return false;
    }
}

static bool v_object(const char **s, const char *end) {
    if (*s >= end || **s != '{') return false;
    (*s)++;
    v_skip_ws(s, end);
    if (*s < end && **s == '}') { (*s)++; return true; }
    for (;;) {
        v_skip_ws(s, end);
        if (!v_string_lit(s, end)) return false;
        v_skip_ws(s, end);
        if (*s >= end || **s != ':') return false;
        (*s)++;
        v_skip_ws(s, end);
        if (!v_value(s, end)) return false;
        v_skip_ws(s, end);
        if (*s >= end) return false;
        if (**s == ',') { (*s)++; continue; }
        if (**s == '}') { (*s)++; return true; }
        return false;
    }
}

static bool v_value(const char **s, const char *end) {
    v_skip_ws(s, end);
    if (*s >= end) return false;
    char c = **s;
    if (c == '{') return v_object(s, end);
    if (c == '[') return v_array(s, end);
    if (c == '"') return v_string_lit(s, end);
    if (c == '-' || (c >= '0' && c <= '9')) return v_number(s, end);
    if (c == 't') return v_literal(s, end, "true");
    if (c == 'f') return v_literal(s, end, "false");
    if (c == 'n') return v_literal(s, end, "null");
    return false;
}

bool ez_json_is_valid(EzString text) {
    if (text.len <= 0 || !text.data) return false;
    const char *s = text.data;
    const char *end = s + text.len;
    v_skip_ws(&s, end);
    if (s >= end) return false;
    if (!v_value(&s, end)) return false;
    v_skip_ws(&s, end);
    return s == end;
}

EzString ez_json_pretty_map(EzArena *arena, EzMap *m, int64_t indent_size) {
    /* Pass 1: exact size */
    size_t ind = indent_size > 0 ? (size_t)indent_size : 0;
    size_t need = 3; /* { \n } */
    int counted = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (counted > 0) need += 2; /* ,\n */
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        EzString *val = (EzString *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        need += ind + json_escaped_len(*key) + 2 /* ": " */ + json_escaped_len(*val);
        counted++;
    }
    if (counted > 0) need += 1; /* trailing \n before } */
    /* Pass 2: write */
    char *buf = ez_arena_alloc(arena, need + 1);
    int pos = 0;
    buf[pos++] = '{';
    buf[pos++] = '\n';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; buf[pos++] = '\n'; }
        for (size_t j = 0; j < ind; j++) buf[pos++] = ' ';
        EzString *key = (EzString *)((char *)m->keys + (size_t)i * (size_t)m->key_size);
        EzString *val = (EzString *)((char *)m->values + (size_t)i * (size_t)m->value_size);
        json_append_escaped(buf, &pos, *key);
        buf[pos++] = ':'; buf[pos++] = ' ';
        json_append_escaped(buf, &pos, *val);
        entry++;
    }
    buf[pos++] = '\n';
    buf[pos++] = '}';
    buf[pos] = '\0';
    EzString r = { buf, (int32_t)pos };
    return r;
}

/* --- Array splitter ---
 * Splits a JSON array "[{...},{...},...]" into an EzArray of EzString,
 * where each element is the raw JSON text of one top-level element.
 * Handles nested braces, brackets, and quoted strings correctly. */

EzArray ez_json_split_array(EzArena *arena, EzString text) {
    EzArray arr = ez_array_new(arena, sizeof(EzString), 4);
    const char *s = text.data;
    const char *end = s + text.len;
    skip_ws(&s, end);
    if (s >= end || *s != '[') return arr;
    s++; /* skip [ */

    while (s < end) {
        skip_ws(&s, end);
        if (s >= end || *s == ']') break;

        /* Mark start of this element */
        const char *elem_start = s;
        int depth_brace = 0, depth_bracket = 0;
        bool in_string = false;

        /* Scan to end of element (respecting nesting and strings) */
        while (s < end) {
            char c = *s;
            if (in_string) {
                if (c == '\\') { s++; if (s < end) s++; continue; }
                if (c == '"') in_string = false;
                s++;
                continue;
            }
            if (c == '"') { in_string = true; s++; continue; }
            if (c == '{') { depth_brace++; s++; continue; }
            if (c == '}') { depth_brace--; s++; continue; }
            if (c == '[') { depth_bracket++; s++; continue; }
            if (c == ']') {
                if (depth_bracket == 0) break; /* end of outer array */
                depth_bracket--;
                s++;
                continue;
            }
            if (c == ',' && depth_brace == 0 && depth_bracket == 0) break;
            s++;
        }

        int32_t elem_len = (int32_t)(s - elem_start);
        if (elem_len > 0) {
            EzString elem = ez_string_new(arena, elem_start, elem_len);
            ez_array_push(arena, &arr, &elem);
        }

        skip_ws(&s, end);
        if (s < end && *s == ',') s++;
    }
    return arr;
}

/* _result variant */

EzResult_map ez_json_decode_result(EzArena *arena, EzString text) {
    EzResult_map r;
    if (text.len <= 0 || !text.data) {
        r.v0 = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "empty JSON input"));
        return r;
    }
    if (!ez_json_is_valid(text)) {
        r.v0 = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "invalid JSON"));
        return r;
    }
    r.v0 = ez_json_decode(arena, text);
    r.v1 = NULL;
    return r;
}
