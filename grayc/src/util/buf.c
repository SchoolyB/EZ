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

static void grow_buffer(Buf *buffer, size_t needed) {
    if (buffer->len + needed + 1 <= buffer->cap) return;
    size_t new_cap = buffer->cap * 2;
    if (new_cap < buffer->len + needed + 1) {
        new_cap = buffer->len + needed + 1;
    }
    buffer->data = realloc(buffer->data, new_cap);
    if (!buffer->data) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    buffer->cap = new_cap;
}

Buf buffer_create(size_t initial_cap) {
    Buf buffer;
    buffer.data = malloc(initial_cap);
    if (!buffer.data) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    buffer.data[0] = '\0';
    buffer.len = 0;
    buffer.cap = initial_cap;
    return buffer;
}

void append_string_to_buffer(Buf *buffer, const char *string) {
    size_t len = strlen(string);
    append_bytes_to_buffer(buffer, string, len);
}

void append_bytes_to_buffer(Buf *buffer, const char *data, size_t len) {
    grow_buffer(buffer, len);
    memcpy(buffer->data + buffer->len, data, len);
    buffer->len += len;
    buffer->data[buffer->len] = '\0';
}

void append_format_to_buffer(Buf *buffer, const char *format, ...) {
    va_list args;

    va_start(args, format);
    int needed = vsnprintf(NULL, 0, format, args);
    va_end(args);

    if (needed < 0) return;

    grow_buffer(buffer, (size_t)needed);

    va_start(args, format);
    vsnprintf(buffer->data + buffer->len, (size_t)needed + 1, format, args);
    va_end(args);

    buffer->len += (size_t)needed;
}

void append_char_to_buffer(Buf *buffer, char character) {
    grow_buffer(buffer, 1);
    buffer->data[buffer->len++] = character;
    buffer->data[buffer->len] = '\0';
}

#define BUF_INDENT_WIDTH 4

void append_indent_to_buffer(Buf *buffer, int depth) {
    static const char spaces[] =
        "                                                                ";
    int n = depth * BUF_INDENT_WIDTH;
    if (n < (int)sizeof(spaces)) {
        append_bytes_to_buffer(buffer, spaces, (size_t)n);
    } else {
        for (int i = 0; i < depth; i++)
            append_bytes_to_buffer(buffer, spaces, BUF_INDENT_WIDTH);
    }
}

const char *buffer_to_string(Buf *buffer) {
    return buffer->data;
}

void buffer_destroy(Buf *buffer) {
    free(buffer->data);
    buffer->data = NULL;
    buffer->len = 0;
    buffer->cap = 0;
}
