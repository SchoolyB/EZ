/*
 * buf.h — Public interface for the growable string buffer used by the
 * code generator to emit C source output.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAYC_BUF_H
#define GRAYC_BUF_H

#include <stddef.h>

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} Buf;

Buf buffer_create(size_t initial_cap);
void append_string_to_buffer(Buf *buffer, const char *string);
void append_bytes_to_buffer(Buf *buffer, const char *data, size_t len);
void append_format_to_buffer(Buf *buffer, const char *format, ...) __attribute__((format(printf, 2, 3)));
void append_char_to_buffer(Buf *buffer, char character);
void append_indent_to_buffer(Buf *buffer, int depth);
const char *buffer_to_string(Buf *buffer);
void buffer_destroy(Buf *buffer);

#endif
