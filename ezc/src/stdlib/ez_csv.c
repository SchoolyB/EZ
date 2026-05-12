/*
 * ez_csv.c - @csv module implementation
 *
 * Simple RFC 4180 compliant CSV parser.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_csv.h"
#include <string.h>
#include <stdio.h>

EzArray ez_csv_parse(EzArena *arena, EzString csv_string) {
    EzArray rows = ez_array_new(arena, sizeof(EzArray), 8);
    const char *s = csv_string.data;
    const char *end = s + csv_string.len;

    while (s < end) {
        EzArray row = ez_array_new(arena, sizeof(EzString), 8);
        while (s < end && *s != '\n' && *s != '\r') {
            const char *field_start;
            int32_t field_len;

            if (*s == '"') {
                /* Quoted field */
                s++;
                field_start = s;
                while (s < end && !(*s == '"' && (s + 1 >= end || *(s + 1) != '"'))) {
                    if (*s == '"' && *(s + 1) == '"') s += 2;
                    else s++;
                }
                field_len = (int32_t)(s - field_start);
                if (s < end) s++; /* skip closing quote */
            } else {
                /* Unquoted field */
                field_start = s;
                while (s < end && *s != ',' && *s != '\n' && *s != '\r') s++;
                field_len = (int32_t)(s - field_start);
            }

            EzString field = ez_string_new(arena, field_start, field_len);
            ez_array_push(arena, &row, &field);

            if (s < end && *s == ',') s++;
        }
        ez_array_push(arena, &rows, &row);

        /* Skip line ending */
        if (s < end && *s == '\r') s++;
        if (s < end && *s == '\n') s++;
    }
    return rows;
}

EzString ez_csv_stringify(EzArena *arena, EzArray *data) {
    /* Accept [string] — each string is a pre-formatted CSV row.
     * Join with newlines. */
    if (data->elem_size == (int32_t)sizeof(EzString)) {
        int32_t total = 0;
        for (int32_t i = 0; i < data->len; i++) {
            EzString s = EZ_ARRAY_GET(*data, EzString, i);
            total += s.len + 1;
        }
        char *buf = ez_arena_alloc(arena, (size_t)total + 1);
        int32_t pos = 0;
        for (int32_t i = 0; i < data->len; i++) {
            EzString s = EZ_ARRAY_GET(*data, EzString, i);
            memcpy(buf + pos, s.data, (size_t)s.len);
            pos += s.len;
            if (i < data->len - 1) buf[pos++] = '\n';
        }
        buf[pos] = '\0';
        return (EzString){ buf, pos };
    }

    /* Fallback: array of arrays ([[string]] rows).
     * First pass: compute exact required size to avoid heap overflow. */
    int32_t total = 0;
    for (int32_t i = 0; i < data->len; i++) {
        EzArray *row = (EzArray *)((char *)data->data + (size_t)i * sizeof(EzArray));
        for (int32_t j = 0; j < row->len; j++) {
            if (j > 0) total++; /* comma */
            EzString *field = (EzString *)((char *)row->data + (size_t)j * sizeof(EzString));
            total += field->len;
        }
        total++; /* newline */
    }
    char *buf = ez_arena_alloc(arena, (size_t)total + 1);
    int32_t pos = 0;
    for (int32_t i = 0; i < data->len; i++) {
        EzArray *row = (EzArray *)((char *)data->data + (size_t)i * sizeof(EzArray));
        for (int32_t j = 0; j < row->len; j++) {
            if (j > 0) buf[pos++] = ',';
            EzString *field = (EzString *)((char *)row->data + (size_t)j * sizeof(EzString));
            memcpy(buf + pos, field->data, (size_t)field->len);
            pos += field->len;
        }
        buf[pos++] = '\n';
    }
    buf[pos] = '\0';
    return (EzString){ buf, pos };
}

EzArray ez_csv_headers(EzArena *arena, EzArray *data) {
    if (data->len > 0) {
        EzArray first_row = EZ_ARRAY_GET(*data, EzArray, 0);
        return ez_array_copy(arena, &first_row);
    }
    return ez_array_new(arena, sizeof(EzString), 0);
}

EzArray ez_csv_read(EzArena *arena, EzString path) {
    FILE *f = fopen(path.data, "rb");
    if (!f) return ez_array_new(arena, sizeof(EzArray), 1);
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);
    char *content = ez_arena_alloc(arena, (size_t)size + 1);
    size_t read = fread(content, 1, (size_t)size, f);
    content[read] = '\0';
    fclose(f);
    EzString s = { content, (int32_t)read };
    return ez_csv_parse(arena, s);
}

bool ez_csv_write(EzArena *arena, EzString path, EzArray *data) {
    EzString csv = ez_csv_stringify(arena, data);
    FILE *f = fopen(path.data, "wb");
    if (!f) return false;
    fwrite(csv.data, 1, (size_t)csv.len, f);
    fclose(f);
    return true;
}

/* _result variants */

EzResult_array ez_csv_read_result(EzArena *arena, EzString path) {
    EzResult_array r;
    FILE *f = fopen(path.data, "rb");
    if (!f) {
        r.v0 = ez_array_new(arena, sizeof(EzArray), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot read CSV file '%s'", path.data));
        return r;
    }
    fclose(f);
    r.v0 = ez_csv_read(arena, path);
    r.v1 = NULL;
    return r;
}

EzResult_bool ez_csv_write_result(EzArena *arena, EzString path, EzArray *data) {
    EzResult_bool r;
    if (ez_csv_write(arena, path, data)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot write CSV file '%s'", path.data));
    }
    return r;
}
