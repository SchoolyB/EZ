/*
 * buf.h - Growable string buffer for code emission
 */

#ifndef EZC_BUF_H
#define EZC_BUF_H

#include <stddef.h>

typedef struct {
    char *data;
    size_t len;
    size_t cap;
} Buf;

Buf buf_create(size_t initial_cap);
void buf_append(Buf *b, const char *s);
void buf_appendn(Buf *b, const char *s, size_t len);
void buf_appendf(Buf *b, const char *fmt, ...) __attribute__((format(printf, 2, 3)));
void buf_append_char(Buf *b, char c);
void buf_append_indent(Buf *b, int depth);
const char *buf_cstr(Buf *b);
void buf_destroy(Buf *b);

#endif
