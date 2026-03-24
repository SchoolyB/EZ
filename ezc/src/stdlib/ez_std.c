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

/* --- println --- */

void ez_std_println_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
    putchar('\n');
}

void ez_std_println_int(int64_t v) {
    printf("%" PRId64 "\n", v);
}

void ez_std_println_float(double v) {
    /* Match interpreter: print full precision, always show decimal point */
    char buf[64];
    snprintf(buf, sizeof(buf), "%.15g", v);
    /* Ensure there's always a decimal point for whole-number floats */
    if (!strchr(buf, '.') && !strchr(buf, 'e')) {
        size_t len = strlen(buf);
        buf[len] = '.';
        buf[len+1] = '0';
        buf[len+2] = '\0';
    }
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
    if (seconds > 0) sleep((unsigned int)seconds);
}

void ez_std_sleep_milliseconds(int64_t ms) {
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
    int len = snprintf(buf, sizeof(buf), "%.15g", v);
    /* Ensure decimal point for whole-number floats */
    if (!strchr(buf, '.') && !strchr(buf, 'e')) {
        buf[len++] = '.';
        buf[len++] = '0';
        buf[len] = '\0';
    }
    return ez_string_new(arena, buf, (int32_t)len);
}

EzString ez_std_to_string_bool(EzArena *arena, bool v) {
    return v ? ez_string_lit("true") : ez_string_lit("false");
}
