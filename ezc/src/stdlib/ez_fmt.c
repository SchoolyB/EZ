/*
 * ez_fmt.c - @fmt module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_fmt.h"
#include <inttypes.h>

void ez_fmt_eprintln_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
    fputc('\n', stderr);
}

void ez_fmt_eprintln_int(int64_t v) {
    fprintf(stderr, "%" PRId64 "\n", v);
}

void ez_fmt_eprintln_float(double v) {
    fprintf(stderr, "%g\n", v);
}

void ez_fmt_eprintln_bool(bool v) {
    fprintf(stderr, "%s\n", v ? "true" : "false");
}

void ez_fmt_eprint_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
}

EzString ez_fmt_pad_left(EzArena *arena, EzString s, int64_t width, char ch) {
    if (s.len >= width) return s;
    int64_t pad = width - s.len;
    char *buf = (char *)ez_arena_alloc(arena, (size_t)width);
    memset(buf, ch, (size_t)pad);
    memcpy(buf + pad, s.data, (size_t)s.len);
    return (EzString){buf, width};
}

EzString ez_fmt_pad_right(EzArena *arena, EzString s, int64_t width, char ch) {
    if (s.len >= width) return s;
    int64_t pad = width - s.len;
    char *buf = (char *)ez_arena_alloc(arena, (size_t)width);
    memcpy(buf, s.data, (size_t)s.len);
    memset(buf + s.len, ch, (size_t)pad);
    return (EzString){buf, width};
}

EzString ez_fmt_center(EzArena *arena, EzString s, int64_t width, char ch) {
    if (s.len >= width) return s;
    int64_t total_pad = width - s.len;
    int64_t left_pad = total_pad / 2;
    int64_t right_pad = total_pad - left_pad;
    char *buf = (char *)ez_arena_alloc(arena, (size_t)width);
    memset(buf, ch, (size_t)left_pad);
    memcpy(buf + left_pad, s.data, (size_t)s.len);
    memset(buf + left_pad + s.len, ch, (size_t)right_pad);
    return (EzString){buf, width};
}

EzString ez_fmt_int_to_hex(EzArena *arena, int64_t n) {
    char tmp[32];
    int len = snprintf(tmp, sizeof(tmp), "%" PRIx64, (uint64_t)n);
    char *buf = (char *)ez_arena_alloc(arena, (size_t)len);
    memcpy(buf, tmp, (size_t)len);
    return (EzString){buf, len};
}

EzString ez_fmt_int_to_binary(EzArena *arena, int64_t n) {
    if (n == 0) {
        char *buf = (char *)ez_arena_alloc(arena, 1);
        buf[0] = '0';
        return (EzString){buf, 1};
    }
    char tmp[65];
    int pos = 64;
    uint64_t v = (uint64_t)n;
    while (v > 0) {
        tmp[--pos] = (char)('0' + (v & 1));
        v >>= 1;
    }
    int len = 64 - pos;
    char *buf = (char *)ez_arena_alloc(arena, (size_t)len);
    memcpy(buf, tmp + pos, (size_t)len);
    return (EzString){buf, len};
}

EzString ez_fmt_int_to_octal(EzArena *arena, int64_t n) {
    char tmp[32];
    int len = snprintf(tmp, sizeof(tmp), "%" PRIo64, (uint64_t)n);
    char *buf = (char *)ez_arena_alloc(arena, (size_t)len);
    memcpy(buf, tmp, (size_t)len);
    return (EzString){buf, len};
}

EzString ez_fmt_float_fixed(EzArena *arena, double f, int64_t decimals) {
    char tmp[64];
    int len = snprintf(tmp, sizeof(tmp), "%.*f", (int)decimals, f);
    char *buf = (char *)ez_arena_alloc(arena, (size_t)len);
    memcpy(buf, tmp, (size_t)len);
    return (EzString){buf, len};
}

EzString ez_fmt_float_sci(EzArena *arena, double f) {
    char tmp[64];
    int len = snprintf(tmp, sizeof(tmp), "%e", f);
    char *buf = (char *)ez_arena_alloc(arena, (size_t)len);
    memcpy(buf, tmp, (size_t)len);
    return (EzString){buf, len};
}
