/*
 * ez_runtime.h - EZC runtime support
 *
 * This header is included in all generated C code.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_RUNTIME_H
#define EZ_RUNTIME_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* --- Arena Allocator --- */

typedef struct EzArenaBlock {
    struct EzArenaBlock *next;
    size_t size;
    size_t used;
    char data[];
} EzArenaBlock;

typedef struct {
    EzArenaBlock *first;
    EzArenaBlock *current;
    size_t default_block_size;
    bool destroyed;
} EzArena;

EzArena *ez_arena_create(size_t initial_size);
void *ez_arena_alloc(EzArena *arena, size_t size);
void ez_arena_reset(EzArena *arena);
void ez_arena_destroy(EzArena *arena, const char *file, int line);
size_t ez_arena_usage(EzArena *arena);

/* Global default arena */
extern EzArena *ez_default_arena;

/* --- String --- */

typedef struct {
    const char *data;
    int32_t len;
} EzString;

/* --- Error --- */

typedef struct {
    EzString message;
    EzString code;
} EzError;

/* Create an error on the default arena */
EzError *ez_error_new(EzArena *arena, EzString message);

/* Create a string from a C string literal (no copy, points to static data) */
static inline EzString ez_string_lit(const char *s) {
    EzString str;
    str.data = s;
    str.len = (int32_t)strlen(s);
    return str;
}

/* String literal with explicit length — for strings containing null bytes */
static inline EzString ez_string_lit_len(const char *s, int32_t len) {
    EzString str;
    str.data = s;
    str.len = len;
    return str;
}

/* Compile-time string literal — works at file scope (C11 compliant) */
#define EZ_STRING_LIT(s) ((EzString){ (s), sizeof(s) - 1 })

/* Create a string with a copy on the arena */
EzString ez_string_new(EzArena *arena, const char *s, int32_t len);

/* String formatting (for interpolation) */
EzString ez_string_format(EzArena *arena, const char *fmt, ...);

/* String comparison */
static inline bool ez_string_eq(EzString a, EzString b) {
    if (a.len != b.len) return false;
    return memcmp(a.data, b.data, (size_t)a.len) == 0;
}

/* String concatenation */
EzString ez_string_concat(EzArena *arena, EzString a, EzString b);

/* --- Runtime Init/Shutdown --- */

void ez_runtime_init(void);
void ez_runtime_shutdown(void);

/* --- Stack depth guard --- */

#define EZ_MAX_CALL_DEPTH 10000
extern int ez_call_depth;

static inline void ez_enter_func(const char *file, int line) {
    if (++ez_call_depth > EZ_MAX_CALL_DEPTH) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: stack overflow (recursion depth exceeded %d)\n",
            file, line, EZ_MAX_CALL_DEPTH);
        exit(1);
    }
}

static inline void ez_exit_func(void) {
    ez_call_depth--;
}

/* --- Panic --- */

void ez_panic(const char *file, int line, const char *fmt, ...)
    __attribute__((format(printf, 3, 4), noreturn));

/* Overflow-checked integer arithmetic */
static inline int64_t ez_add_check(int64_t a, int64_t b, const char *file, int line) {
    int64_t result;
    if (__builtin_add_overflow(a, b, &result)) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: integer overflow in addition\n", file, line);
        exit(1);
    }
    return result;
}

static inline int64_t ez_sub_check(int64_t a, int64_t b, const char *file, int line) {
    int64_t result;
    if (__builtin_sub_overflow(a, b, &result)) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: integer overflow in subtraction\n", file, line);
        exit(1);
    }
    return result;
}

static inline int64_t ez_mul_check(int64_t a, int64_t b, const char *file, int line) {
    int64_t result;
    if (__builtin_mul_overflow(a, b, &result)) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: integer overflow in multiplication\n", file, line);
        exit(1);
    }
    return result;
}

static inline int64_t ez_inc_check(int64_t a, const char *file, int line) {
    return ez_add_check(a, 1, file, line);
}

static inline int64_t ez_dec_check(int64_t a, const char *file, int line) {
    return ez_sub_check(a, 1, file, line);
}

/* Overflow-checked unsigned integer arithmetic */
static inline uint64_t ez_uadd_check(uint64_t a, uint64_t b, const char *file, int line) {
    uint64_t result;
    if (__builtin_add_overflow(a, b, &result)) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: unsigned integer overflow in addition\n", file, line);
        exit(1);
    }
    return result;
}

static inline uint64_t ez_usub_check(uint64_t a, uint64_t b, const char *file, int line) {
    if (b > a) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: unsigned integer underflow in subtraction\n", file, line);
        exit(1);
    }
    return a - b;
}

static inline uint64_t ez_umul_check(uint64_t a, uint64_t b, const char *file, int line) {
    uint64_t result;
    if (__builtin_mul_overflow(a, b, &result)) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: unsigned integer overflow in multiplication\n", file, line);
        exit(1);
    }
    return result;
}

/* Sized signed integer overflow checks (i8, i16, i32) */
static inline int64_t ez_sized_add_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    int64_t result = a + b;
    if (result < min_val || result > max_val) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer overflow in addition\n", file, line, type_name);
        exit(1);
    }
    return result;
}

static inline int64_t ez_sized_sub_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    int64_t result = a - b;
    if (result < min_val || result > max_val) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer overflow in subtraction\n", file, line, type_name);
        exit(1);
    }
    return result;
}

static inline int64_t ez_sized_mul_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    int64_t result = a * b;
    if (result < min_val || result > max_val) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer overflow in multiplication\n", file, line, type_name);
        exit(1);
    }
    return result;
}

/* Sized unsigned integer overflow checks (u8, u16, u32, byte) */
static inline uint64_t ez_usized_add_check(uint64_t a, uint64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    uint64_t result = a + b;
    if (result > max_val) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer overflow in addition\n", file, line, type_name);
        exit(1);
    }
    return result;
}

static inline uint64_t ez_usized_sub_check(uint64_t a, uint64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    if (b > a) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer underflow in subtraction\n", file, line, type_name);
        exit(1);
    }
    return a - b;
}

static inline uint64_t ez_usized_mul_check(uint64_t a, uint64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    uint64_t result = a * b;
    if (result > max_val) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: %s integer overflow in multiplication\n", file, line, type_name);
        exit(1);
    }
    return result;
}

/* Safe float-to-int conversion with overflow check */
static inline int64_t ez_float_to_int(double v, const char *file, int line) {
    if (v > 9.223372036854775e+18 || v < -9.223372036854775e+18 ||
        v != v /* NaN */) {
        fflush(stdout);
        fprintf(stderr, "panic at %s:%d: conversion overflow — float value exceeds integer range\n", file, line);
        exit(1);
    }
    return (int64_t)v;
}

#endif
