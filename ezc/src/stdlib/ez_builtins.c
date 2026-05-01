/*
 * ez_builtins.c - Built-in functions that require no import
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_builtins.h"
#include "../util/constants.h"
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <inttypes.h>
#include <unistd.h>
#include <time.h>

#define EZ_TOSTRING_BUF_SIZE    4096
#define EZ_TOSTRING_SAFE_LIMIT  (EZ_TOSTRING_BUF_SIZE - 96)
#define EZ_INT_STR_BUF          32
#define EZ_FLOAT_STR_BUF        64
#define EZ_INPUT_BUF_SIZE       4096

/* Format a double using the shortest representation that round-trips */
static int fmt_shortest_float(char *buf, size_t bufsz, double v) {
    int n = 0;
    for (int prec = 15; prec <= 17; prec++) {
        n = snprintf(buf, bufsz, "%.*g", prec, v);
        double rt;
        if (sscanf(buf, "%lf", &rt) == 1 && rt == v) break;
    }
    if (!strchr(buf, '.') && !strchr(buf, 'e') && !strchr(buf, 'i') &&
        !strchr(buf, 'n') && n + 2 < (int)bufsz) {
        buf[n++] = '.';
        buf[n++] = '0';
        buf[n] = '\0';
    }
    return n;
}

/* Write a Unicode code point as UTF-8 to a FILE stream */
static void fput_utf8(int32_t c, FILE *f) {
    if (c < 0x80) {
        fputc(c, f);
    } else if (c < 0x800) {
        fputc(0xC0 | (c >> 6), f);
        fputc(0x80 | (c & 0x3F), f);
    } else if (c < 0x10000) {
        fputc(0xE0 | (c >> 12), f);
        fputc(0x80 | ((c >> 6) & 0x3F), f);
        fputc(0x80 | (c & 0x3F), f);
    } else if (c < 0x110000) {
        fputc(0xF0 | (c >> 18), f);
        fputc(0x80 | ((c >> 12) & 0x3F), f);
        fputc(0x80 | ((c >> 6) & 0x3F), f);
        fputc(0x80 | (c & 0x3F), f);
    }
}

/* --- println --- */

void ez_builtin_println_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
    putchar('\n');
}

void ez_builtin_println_int(int64_t v) {
    printf("%" PRId64 "\n", v);
}

void ez_builtin_println_uint(uint64_t v) {
    printf("%" PRIu64 "\n", v);
}

void ez_builtin_println_float(double v) {
    char buf[EZ_FLOAT_STR_BUF];
    fmt_shortest_float(buf, sizeof(buf), v);
    printf("%s\n", buf);
}

void ez_builtin_println_bool(bool v) {
    printf("%s\n", v ? "true" : "false");
}

void ez_builtin_println_char(int32_t c) {
    fput_utf8(c, stdout);
    putchar('\n');
}

void ez_builtin_println_addr(uintptr_t v) {
    printf("0x%" PRIxPTR "\n", v);
}

/* --- print --- */

void ez_builtin_print_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
}

void ez_builtin_print_int(int64_t v) {
    printf("%" PRId64, v);
}

void ez_builtin_print_uint(uint64_t v) {
    printf("%" PRIu64, v);
}

void ez_builtin_print_float(double v) {
    char buf[EZ_FLOAT_STR_BUF];
    fmt_shortest_float(buf, sizeof(buf), v);
    printf("%s", buf);
}

void ez_builtin_print_bool(bool v) {
    printf("%s", v ? "true" : "false");
}

void ez_builtin_print_char(int32_t c) {
    fput_utf8(c, stdout);
}

void ez_builtin_print_addr(uintptr_t v) {
    printf("0x%" PRIxPTR, v);
}

/* --- eprintln / eprint --- */

void ez_builtin_eprintln_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
    fputc('\n', stderr);
}

void ez_builtin_eprintln_int(int64_t v) {
    fprintf(stderr, "%" PRId64 "\n", v);
}

void ez_builtin_eprintln_uint(uint64_t v) {
    fprintf(stderr, "%" PRIu64 "\n", v);
}

void ez_builtin_eprintln_char(int32_t c) {
    fput_utf8(c, stderr);
    fputc('\n', stderr);
}

void ez_builtin_eprintln_addr(uintptr_t v) {
    fprintf(stderr, "0x%" PRIxPTR "\n", v);
}

void ez_builtin_eprint_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
}

void ez_builtin_eprint_char(int32_t c) {
    fput_utf8(c, stderr);
}

void ez_builtin_eprint_addr(uintptr_t v) {
    fprintf(stderr, "0x%" PRIxPTR, v);
}

/* --- input --- */

EzString ez_builtin_input(EzArena *arena) {
    char buf[EZ_INPUT_BUF_SIZE];
    if (fgets(buf, sizeof(buf), stdin) == NULL) {
        return ez_string_lit("");
    }
    size_t len = strlen(buf);
    if (len > 0 && buf[len - 1] == '\n') len--;
    return ez_string_new(arena, buf, (int32_t)len);
}

/* --- assert --- */

void ez_builtin_assert(bool condition, EzString message, const char *file, int line) {
    if (!condition) {
        fprintf(stderr, "assertion failed at %s:%d: ", file, line);
        fwrite(message.data, 1, (size_t)message.len, stderr);
        fputc('\n', stderr);
        exit(1);
    }
}

/* --- panic --- */

void ez_builtin_panic_msg(EzString message) {
    fprintf(stderr, "panic: ");
    fwrite(message.data, 1, (size_t)message.len, stderr);
    fputc('\n', stderr);
    exit(1);
}

/* --- exit --- */

void ez_builtin_exit(int64_t code) {
    exit((int)code);
}

/* --- sleep --- */

void ez_builtin_sleep_s(int64_t seconds) {
    if (seconds < 0) { fflush(stdout); fprintf(stderr, "panic: sleep duration cannot be negative (%lld)\n", (long long)seconds); exit(1); }
    if (seconds > 0) sleep((unsigned int)seconds);
}

void ez_builtin_sleep_ms(int64_t ms) {
    if (ms < 0) { fflush(stdout); fprintf(stderr, "panic: sleep duration cannot be negative (%lld ms)\n", (long long)ms); exit(1); }
    if (ms > 0) {
        struct timespec ts;
        ts.tv_sec = ms / MS_PER_SEC;
        ts.tv_nsec = (ms % MS_PER_SEC) * NS_PER_MS;
        nanosleep(&ts, NULL);
    }
}

void ez_builtin_sleep_ns(int64_t ns) {
    if (ns > 0) {
        struct timespec ts;
        ts.tv_sec = ns / NS_PER_SEC;
        ts.tv_nsec = ns % NS_PER_SEC;
        nanosleep(&ts, NULL);
    }
}

/* --- to_string --- */

EzString ez_builtin_to_string_int(EzArena *arena, int64_t v) {
    char buf[EZ_INT_STR_BUF];
    int len = snprintf(buf, sizeof(buf), "%" PRId64, v);
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_builtin_to_string_uint(EzArena *arena, uint64_t v) {
    char buf[EZ_INT_STR_BUF];
    int len = snprintf(buf, sizeof(buf), "%" PRIu64, v);
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_builtin_to_string_float(EzArena *arena, double v) {
    char buf[EZ_FLOAT_STR_BUF];
    int len = fmt_shortest_float(buf, sizeof(buf), v);
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_builtin_format_float(EzArena *arena, double v) {
    return ez_builtin_to_string_float(arena, v);
}

EzString ez_builtin_to_string_bool(EzArena *arena, bool v) {
    return v ? ez_string_lit("true") : ez_string_lit("false");
}

/* --- from_string --- */

int64_t ez_builtin_string_to_int(EzString s) {
    char buf[EZ_FLOAT_STR_BUF];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    char *end = NULL;
    int64_t result = strtoll(buf, &end, 10);
    if (end == buf || (*end != '\0' && *end != ' ')) {
        fflush(stdout);
        fprintf(stderr, "panic: cannot convert \"%s\" to int\n", buf);
        exit(1);
    }
    return result;
}

double ez_builtin_string_to_float(EzString s) {
    char buf[EZ_FLOAT_STR_BUF];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    char *end = NULL;
    double result = strtod(buf, &end);
    if (end == buf || (*end != '\0' && *end != ' ')) {
        fflush(stdout);
        fprintf(stderr, "panic: cannot convert \"%s\" to float\n", buf);
        exit(1);
    }
    return result;
}

/* --- composite to_string --- */

EzString ez_builtin_array_to_string(EzArena *arena, EzArray *arr, int elem_kind) {
    char buf[EZ_TOSTRING_BUF_SIZE];
    int pos = 0;
    buf[pos++] = '{';
    for (int32_t i = 0; i < arr->len && pos < EZ_TOSTRING_SAFE_LIMIT; i++) {
        if (i > 0) { buf[pos++] = ','; buf[pos++] = ' '; }
        switch (elem_kind) {
        case 0:
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%" PRId64,
                EZ_ARRAY_GET(*arr, int64_t, i));
            break;
        case 1: {
            char fbuf[EZ_FLOAT_STR_BUF];
            fmt_shortest_float(fbuf, sizeof(fbuf), EZ_ARRAY_GET(*arr, double, i));
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%s", fbuf);
            break;
        }
        case 2: {
            EzString s = EZ_ARRAY_GET(*arr, EzString, i);
            pos += snprintf(buf + pos, sizeof(buf) - pos, "\"%.*s\"",
                (int)s.len, s.data ? s.data : "");
            break;
        }
        case 3:
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%s",
                EZ_ARRAY_GET(*arr, bool, i) ? "true" : "false");
            break;
        }
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return ez_string_new(arena, buf, (int32_t)pos);
}

/* --- to_char / char_count — Unicode codepoint access --- */

int32_t ez_builtin_to_char(EzString s, int64_t index, const char *file, int line) {
    if (index < 0) {
        ez_panic(file, line, "to_char() index out of bounds — index %lld is negative", (long long)index);
    }
    const uint8_t *p = (const uint8_t *)s.data;
    const uint8_t *end = p + s.len;
    int64_t cp_idx = 0;
    while (p < end) {
        int32_t cp;
        int bytes;
        uint8_t b = *p;
        if (b < 0x80) {
            cp = b; bytes = 1;
        } else if ((b & 0xE0) == 0xC0) {
            cp = b & 0x1F; bytes = 2;
        } else if ((b & 0xF0) == 0xE0) {
            cp = b & 0x0F; bytes = 3;
        } else if ((b & 0xF8) == 0xF0) {
            cp = b & 0x07; bytes = 4;
        } else {
            cp = 0xFFFD; bytes = 1; /* replacement character for invalid lead byte */
        }
        /* Validate we have enough continuation bytes */
        if (p + bytes > end) {
            cp = 0xFFFD; bytes = 1;
        } else {
            for (int i = 1; i < bytes; i++) {
                if ((p[i] & 0xC0) != 0x80) {
                    cp = 0xFFFD; bytes = 1;
                    break;
                }
                cp = (cp << 6) | (p[i] & 0x3F);
            }
        }
        if (cp_idx == index) return cp;
        p += bytes;
        cp_idx++;
    }
    ez_panic(file, line, "to_char() index out of bounds — index %lld but string has %lld characters",
        (long long)index, (long long)cp_idx);
    return 0; /* unreachable */
}

int64_t ez_builtin_char_count(EzString s) {
    const uint8_t *p = (const uint8_t *)s.data;
    const uint8_t *end = p + s.len;
    int64_t count = 0;
    while (p < end) {
        uint8_t b = *p;
        if (b < 0x80) p += 1;
        else if ((b & 0xE0) == 0xC0) p += 2;
        else if ((b & 0xF0) == 0xE0) p += 3;
        else if ((b & 0xF8) == 0xF0) p += 4;
        else p += 1; /* invalid lead byte, skip one */
        if (p > end) p = end; /* clamp if truncated */
        count++;
    }
    return count;
}

EzString ez_builtin_char_to_utf8(EzArena *arena, int32_t cp) {
    char buf[4];
    int len = 0;
    if (cp < 0x80) {
        buf[0] = (char)cp; len = 1;
    } else if (cp < 0x800) {
        buf[0] = (char)(0xC0 | (cp >> 6));
        buf[1] = (char)(0x80 | (cp & 0x3F));
        len = 2;
    } else if (cp < 0x10000) {
        buf[0] = (char)(0xE0 | (cp >> 12));
        buf[1] = (char)(0x80 | ((cp >> 6) & 0x3F));
        buf[2] = (char)(0x80 | (cp & 0x3F));
        len = 3;
    } else if (cp < 0x110000) {
        buf[0] = (char)(0xF0 | (cp >> 18));
        buf[1] = (char)(0x80 | ((cp >> 12) & 0x3F));
        buf[2] = (char)(0x80 | ((cp >> 6) & 0x3F));
        buf[3] = (char)(0x80 | (cp & 0x3F));
        len = 4;
    } else {
        /* Invalid codepoint — replacement character U+FFFD */
        buf[0] = (char)0xEF; buf[1] = (char)0xBF; buf[2] = (char)0xBD;
        len = 3;
    }
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_builtin_map_to_string(EzArena *arena, EzMap *m, int val_kind) {
    char buf[EZ_TOSTRING_BUF_SIZE];
    int pos = 0;
    buf[pos++] = '{';
    for (int32_t oi = 0; oi < m->order_len && pos < EZ_TOSTRING_SAFE_LIMIT; oi++) {
        int32_t i = m->order[oi];
        if (m->states[i] != 1) continue;
        if (oi > 0) { buf[pos++] = ','; buf[pos++] = ' '; }
        EzString *kp = (EzString *)((char *)m->keys + (size_t)i * m->key_size);
        pos += snprintf(buf + pos, sizeof(buf) - pos, "\"%.*s\": ",
            (int)kp->len, kp->data ? kp->data : "");
        void *vp = (char *)m->values + (size_t)i * m->value_size;
        switch (val_kind) {
        case 0: pos += snprintf(buf + pos, sizeof(buf) - pos, "%" PRId64, *(int64_t *)vp); break;
        case 1: {
            char fbuf[EZ_FLOAT_STR_BUF];
            fmt_shortest_float(fbuf, sizeof(fbuf), *(double *)vp);
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%s", fbuf);
            break;
        }
        case 2: {
            EzString *sp = (EzString *)vp;
            pos += snprintf(buf + pos, sizeof(buf) - pos, "\"%.*s\"",
                (int)sp->len, sp->data ? sp->data : "");
            break;
        }
        case 3: pos += snprintf(buf + pos, sizeof(buf) - pos, "%s",
            *(bool *)vp ? "true" : "false"); break;
        }
    }
    if (m->order_len == 0) { buf[pos++] = ':'; }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return ez_string_new(arena, buf, (int32_t)pos);
}
