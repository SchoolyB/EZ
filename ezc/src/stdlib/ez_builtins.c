/*
 * ez_builtins.c - Built-in functions that require no import
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_builtins.h"
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <time.h>

/* --- input --- */

EzString ez_builtin_input(EzArena *arena) {
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
        ts.tv_sec = ms / 1000;
        ts.tv_nsec = (ms % 1000) * 1000000;
        nanosleep(&ts, NULL);
    }
}

void ez_builtin_sleep_ns(int64_t ns) {
    if (ns > 0) {
        struct timespec ts;
        ts.tv_sec = ns / 1000000000;
        ts.tv_nsec = ns % 1000000000;
        nanosleep(&ts, NULL);
    }
}
