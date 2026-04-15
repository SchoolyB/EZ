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

/* --- Encoder --- */

static void json_append_escaped(char *buf, int *pos, EzString s) {
    buf[(*pos)++] = '"';
    for (int32_t i = 0; i < s.len; i++) {
        char c = s.data[i];
        if (c == '"') { buf[(*pos)++] = '\\'; buf[(*pos)++] = '"'; }
        else if (c == '\\') { buf[(*pos)++] = '\\'; buf[(*pos)++] = '\\'; }
        else if (c == '\n') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 'n'; }
        else if (c == '\t') { buf[(*pos)++] = '\\'; buf[(*pos)++] = 't'; }
        else buf[(*pos)++] = c;
    }
    buf[(*pos)++] = '"';
}

EzString ez_json_encode_map(EzArena *arena, EzMap *m) {
    /* Estimate size */
    int est = m->count * 128 + 2;
    char *buf = ez_arena_alloc(arena, (size_t)est);
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
        /* Try to detect if value is a number or bool */
        if (val->len > 0 && (isdigit(val->data[0]) || val->data[0] == '-')) {
            memcpy(buf + pos, val->data, (size_t)val->len);
            pos += val->len;
        } else if (val->len == 4 && memcmp(val->data, "true", 4) == 0) {
            memcpy(buf + pos, "true", 4); pos += 4;
        } else if (val->len == 5 && memcmp(val->data, "false", 5) == 0) {
            memcpy(buf + pos, "false", 5); pos += 5;
        } else if (val->len == 4 && memcmp(val->data, "null", 4) == 0) {
            memcpy(buf + pos, "null", 4); pos += 4;
        } else {
            json_append_escaped(buf, &pos, *val);
        }
        entry++;
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    EzString r = { buf, (int32_t)pos };
    return r;
}

/* --- Decoder --- */

static void skip_ws(const char **s, const char *end) {
    while (*s < end && isspace(**s)) (*s)++;
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
    while (*s < end && **s != ',' && **s != '}' && **s != ']' && !isspace(**s)) (*s)++;
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
    int est = m->count * 128 + 64;
    char *buf = ez_arena_alloc(arena, (size_t)est);
    int pos = 0;
    buf[pos++] = '{';
    buf[pos++] = '\n';
    int entry = 0;
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] != 1) continue;
        if (entry > 0) { buf[pos++] = ','; buf[pos++] = '\n'; }
        for (int64_t j = 0; j < indent_size; j++) buf[pos++] = ' ';
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
