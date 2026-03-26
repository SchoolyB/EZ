/*
 * ez_std.c - @std module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_std.h"
#include <stdio.h>
#include <stdlib.h>
#include <inttypes.h>
#include <unistd.h>
#include <time.h>

/* Format a double using the shortest representation that round-trips,
 * matching Go's strconv.FormatFloat(v, 'g', -1, 64) behavior. */
static int fmt_shortest_float(char *buf, size_t bufsz, double v) {
    /* Try increasing precision until the round-trip matches */
    int n = 0;
    for (int prec = 15; prec <= 17; prec++) {
        n = snprintf(buf, bufsz, "%.*g", prec, v);
        double rt;
        if (sscanf(buf, "%lf", &rt) == 1 && rt == v) break;
    }
    /* Ensure decimal point for whole-number floats (e.g. "88" → "88.0")
     * but NOT for special values like inf, -inf, nan */
    if (!strchr(buf, '.') && !strchr(buf, 'e') && !strchr(buf, 'i') &&
        !strchr(buf, 'n') && n + 2 < (int)bufsz) {
        buf[n++] = '.';
        buf[n++] = '0';
        buf[n] = '\0';
    }
    return n;
}

/* --- println --- */

void ez_std_println_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
    putchar('\n');
}

void ez_std_println_int(int64_t v) {
    printf("%" PRId64 "\n", v);
}

void ez_std_println_float(double v) {
    char buf[64];
    fmt_shortest_float(buf, sizeof(buf), v);
    printf("%s\n", buf);
}

void ez_std_println_bool(bool v) {
    printf("%s\n", v ? "true" : "false");
}

/* --- print --- */

void ez_std_print_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
}

void ez_std_print_int(int64_t v) {
    printf("%" PRId64, v);
}

/* --- eprintln / eprint --- */

void ez_std_eprintln_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
    fputc('\n', stderr);
}

void ez_std_eprintln_int(int64_t v) {
    fprintf(stderr, "%" PRId64 "\n", v);
}

void ez_std_eprint_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
}

/* --- input --- */

EzString ez_std_input(EzArena *arena) {
    char buf[4096];
    if (fgets(buf, sizeof(buf), stdin) == NULL) {
        return ez_string_lit("");
    }
    /* Strip trailing newline */
    size_t len = strlen(buf);
    if (len > 0 && buf[len - 1] == '\n') len--;
    return ez_string_new(arena, buf, (int32_t)len);
}

/* --- assert --- */

void ez_std_assert(bool condition, EzString message, const char *file, int line) {
    if (!condition) {
        fprintf(stderr, "assertion failed at %s:%d: ", file, line);
        fwrite(message.data, 1, (size_t)message.len, stderr);
        fputc('\n', stderr);
        exit(1);
    }
}

/* --- panic --- */

void ez_std_panic_msg(EzString message) {
    fprintf(stderr, "panic: ");
    fwrite(message.data, 1, (size_t)message.len, stderr);
    fputc('\n', stderr);
    exit(1);
}

/* --- exit --- */

void ez_std_exit(int64_t code) {
    exit((int)code);
}

/* --- error --- */

EzString ez_std_error(EzString message) {
    return message;
}

/* --- sleep --- */

void ez_std_sleep_seconds(int64_t seconds) {
    if (seconds < 0) { fflush(stdout); fprintf(stderr, "panic: sleep duration cannot be negative (%lld)\n", (long long)seconds); exit(1); }
    if (seconds > 0) sleep((unsigned int)seconds);
}

void ez_std_sleep_milliseconds(int64_t ms) {
    if (ms < 0) { fflush(stdout); fprintf(stderr, "panic: sleep duration cannot be negative (%lld ms)\n", (long long)ms); exit(1); }
    if (ms > 0) {
        struct timespec ts;
        ts.tv_sec = ms / 1000;
        ts.tv_nsec = (ms % 1000) * 1000000;
        nanosleep(&ts, NULL);
    }
}

void ez_std_sleep_nanoseconds(int64_t ns) {
    if (ns > 0) {
        struct timespec ts;
        ts.tv_sec = ns / 1000000000;
        ts.tv_nsec = ns % 1000000000;
        nanosleep(&ts, NULL);
    }
}

/* --- to_string --- */

EzString ez_std_to_string_int(EzArena *arena, int64_t v) {
    char buf[32];
    int len = snprintf(buf, sizeof(buf), "%" PRId64, v);
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_std_to_string_float(EzArena *arena, double v) {
    char buf[64];
    int len = fmt_shortest_float(buf, sizeof(buf), v);
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_std_format_float(EzArena *arena, double v) {
    return ez_std_to_string_float(arena, v);
}

EzString ez_std_to_string_bool(EzArena *arena, bool v) {
    return v ? ez_string_lit("true") : ez_string_lit("false");
}

/* Array to string: {elem1, elem2, ...}
 * elem_kind: 0=int, 1=float, 2=string, 3=bool */
EzString ez_std_array_to_string(EzArena *arena, EzArray *arr, int elem_kind) {
    char buf[4096];
    int pos = 0;
    buf[pos++] = '{';
    for (int32_t i = 0; i < arr->len && pos < 4000; i++) {
        if (i > 0) { buf[pos++] = ','; buf[pos++] = ' '; }
        switch (elem_kind) {
        case 0: /* int */
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%" PRId64,
                EZ_ARRAY_GET(*arr, int64_t, i));
            break;
        case 1: { /* float */
            char fbuf[64];
            fmt_shortest_float(fbuf, sizeof(fbuf), EZ_ARRAY_GET(*arr, double, i));
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%s", fbuf);
            break;
        }
        case 2: { /* string */
            EzString s = EZ_ARRAY_GET(*arr, EzString, i);
            pos += snprintf(buf + pos, sizeof(buf) - pos, "\"%.*s\"",
                (int)s.len, s.data ? s.data : "");
            break;
        }
        case 3: /* bool */
            pos += snprintf(buf + pos, sizeof(buf) - pos, "%s",
                EZ_ARRAY_GET(*arr, bool, i) ? "true" : "false");
            break;
        }
    }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return ez_string_new(arena, buf, (int32_t)pos);
}

/* Map to string: {"key1": val1, "key2": val2, ...}
 * val_kind: 0=int, 1=float, 2=string, 3=bool */
EzString ez_std_map_to_string(EzArena *arena, EzMap *m, int val_kind) {
    char buf[4096];
    int pos = 0;
    buf[pos++] = '{';
    /* Iterate in insertion order */
    for (int32_t oi = 0; oi < m->order_len && pos < 4000; oi++) {
        int32_t i = m->order[oi];
        if (m->states[i] != 1) continue;
        if (oi > 0) { buf[pos++] = ','; buf[pos++] = ' '; }
        /* Key (assume string for now — most common) */
        EzString *kp = (EzString *)((char *)m->keys + (size_t)i * m->key_size);
        pos += snprintf(buf + pos, sizeof(buf) - pos, "\"%.*s\": ",
            (int)kp->len, kp->data ? kp->data : "");
        /* Value */
        void *vp = (char *)m->values + (size_t)i * m->value_size;
        switch (val_kind) {
        case 0: pos += snprintf(buf + pos, sizeof(buf) - pos, "%" PRId64, *(int64_t *)vp); break;
        case 1: {
            char fbuf[64];
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
    if (m->count == 0) { /* empty map */ }
    buf[pos++] = '}';
    buf[pos] = '\0';
    return ez_string_new(arena, buf, (int32_t)pos);
}
