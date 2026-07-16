/*
 * buf.c — Growable string buffer implementation used by the code generator
 * to build C source output via append, formatted append, and indentation.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "buf.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <stdarg.h>

static void buf_grow(Buf *b, size_t needed) {
    if (b->len + needed + 1 <= b->cap) return;
    size_t new_cap = b->cap * 2;
    if (new_cap < b->len + needed + 1) {
        new_cap = b->len + needed + 1;
    }
    b->data = realloc(b->data, new_cap);
    if (!b->data) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    b->cap = new_cap;
}

Buf buf_create(size_t initial_cap) {
    Buf b;
    b.data = malloc(initial_cap);
    if (!b.data) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    b.data[0] = '\0';
    b.len = 0;
    b.cap = initial_cap;
    return b;
}

void buf_append(Buf *b, const char *s) {
    size_t len = strlen(s);
    buf_appendn(b, s, len);
}

void buf_appendn(Buf *b, const char *s, size_t len) {
    buf_grow(b, len);
    memcpy(b->data + b->len, s, len);
    b->len += len;
    b->data[b->len] = '\0';
}

void buf_appendf(Buf *b, const char *fmt, ...) {
    va_list args;

    va_start(args, fmt);
    int needed = vsnprintf(NULL, 0, fmt, args);
    va_end(args);

    if (needed < 0) return;

    buf_grow(b, (size_t)needed);

    va_start(args, fmt);
    vsnprintf(b->data + b->len, (size_t)needed + 1, fmt, args);
    va_end(args);

    b->len += (size_t)needed;
}

void buf_append_char(Buf *b, char c) {
    buf_grow(b, 1);
    b->data[b->len++] = c;
    b->data[b->len] = '\0';
}

#define BUF_INDENT_WIDTH 4

void buf_append_indent(Buf *b, int depth) {
    static const char spaces[] =
        "                                                                ";
    int n = depth * BUF_INDENT_WIDTH;
    if (n < (int)sizeof(spaces)) {
        buf_appendn(b, spaces, (size_t)n);
    } else {
        for (int i = 0; i < depth; i++)
            buf_appendn(b, spaces, BUF_INDENT_WIDTH);
    }
}

const char *buf_cstr(Buf *b) {
    return b->data;
}

void buf_destroy(Buf *b) {
    free(b->data);
    b->data = NULL;
    b->len = 0;
    b->cap = 0;
}
