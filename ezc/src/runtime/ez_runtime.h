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

/* --- Arena Defaults --- */
#define EZ_DEFAULT_ARENA_SIZE   (1024 * 1024)
#define EZ_MIN_ARENA_SIZE       4096

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

/* Per-thread default arena — each thread (including spawned threads) gets its own. */
extern _Thread_local EzArena *ez_default_arena;

/* Persistent heap arena — lives for the lifetime of the program.
 * Used by new() so returned pointers are never dangling. */
extern _Thread_local EzArena *ez_heap_arena;

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

/* --- SourceLocation: compile-time substituted by the here() builtin --- */

typedef struct {
    EzString file;
    int64_t line;
    int64_t column;
} EzStruct_SourceLocation;

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

/* Create an EZ string from a C char* by copying onto the arena.
 * NULL input -> empty string. Length is clamped at INT32_MAX. The
 * result has the same lifetime contract as every other arena string,
 * regardless of what happens to the source pointer afterwards. */
EzString ez_c_string_dup(EzArena *arena, const char *s);

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

/* --- Scope-based memory management (#1521) --- */

/* Watermark for mark-and-reset scoping. Save the arena's usage at
 * scope entry; on scope exit, reset to the watermark to free all
 * allocations made during the scope. */
typedef struct {
    EzArenaBlock *block;
    size_t used;
} EzScopeMark;

/* Save current arena state */
EzScopeMark ez_scope_save(EzArena *arena);

/* Restore arena to a saved state — frees everything allocated after
 * the mark. Only called for void scopes (no return value to preserve). */
void ez_scope_restore(EzArena *arena, EzScopeMark mark);

/* --- Panic --- */

void ez_panic(const char *file, int line, const char *fmt, ...)
    __attribute__((format(printf, 3, 4), noreturn));

void ez_panic_code(const char *code, const char *fmt, ...)
    __attribute__((format(printf, 2, 3), noreturn));

/* --- Stack depth guard --- */

#define EZ_MAX_CALL_DEPTH 10000
extern int ez_call_depth;

static inline void ez_enter_func(const char *file, int line) {
    (void)file; (void)line;
    if (++ez_call_depth > EZ_MAX_CALL_DEPTH) {
        ez_panic_code("P0003", "maximum recursion depth exceeded (%d calls deep); your function is calling itself too many times",
            EZ_MAX_CALL_DEPTH);
    }
}

static inline void ez_exit_func(void) {
    ez_call_depth--;
}

/* Overflow-checked integer arithmetic */
static inline int64_t ez_add_check(int64_t a, int64_t b, const char *file, int line) {
    (void)file; (void)line;
    int64_t result;
    if (__builtin_add_overflow(a, b, &result))
        ez_panic_code("P0004", "addition result is too large; value exceeds the range of int");
    return result;
}

static inline int64_t ez_sub_check(int64_t a, int64_t b, const char *file, int line) {
    (void)file; (void)line;
    int64_t result;
    if (__builtin_sub_overflow(a, b, &result))
        ez_panic_code("P0005", "subtraction result is too large; value exceeds the range of int");
    return result;
}

static inline int64_t ez_mul_check(int64_t a, int64_t b, const char *file, int line) {
    (void)file; (void)line;
    int64_t result;
    if (__builtin_mul_overflow(a, b, &result))
        ez_panic_code("P0006", "multiplication result is too large; value exceeds the range of int");
    return result;
}

static inline int64_t ez_neg_check(int64_t a, const char *file, int line) {
    (void)file; (void)line;
    int64_t result;
    if (__builtin_sub_overflow((int64_t)0, a, &result))
        ez_panic_code("P0007", "negation result is too large; value exceeds the range of int");
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
    (void)file; (void)line;
    uint64_t result;
    if (__builtin_add_overflow(a, b, &result))
        ez_panic_code("P0008", "addition result is too large; value exceeds the range of uint");
    return result;
}

static inline uint64_t ez_usub_check(uint64_t a, uint64_t b, const char *file, int line) {
    (void)file; (void)line;
    if (b > a)
        ez_panic_code("P0009", "subtraction result is negative, but uint cannot hold negative values");
    return a - b;
}

static inline uint64_t ez_umul_check(uint64_t a, uint64_t b, const char *file, int line) {
    (void)file; (void)line;
    uint64_t result;
    if (__builtin_mul_overflow(a, b, &result))
        ez_panic_code("P0010", "multiplication result is too large; value exceeds the range of uint");
    return result;
}

/* Sized signed integer overflow checks (i8, i16, i32) */
static inline int64_t ez_sized_add_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a + b;
    if (result < min_val || result > max_val)
        ez_panic_code("P0011", "%s addition result is too large; value exceeds the range of this type", type_name);
    return result;
}

static inline int64_t ez_sized_sub_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a - b;
    if (result < min_val || result > max_val)
        ez_panic_code("P0012", "%s subtraction result is too large; value exceeds the range of this type", type_name);
    return result;
}

static inline int64_t ez_sized_mul_check(int64_t a, int64_t b, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a * b;
    if (result < min_val || result > max_val)
        ez_panic_code("P0013", "%s multiplication result is too large; value exceeds the range of this type", type_name);
    return result;
}

static inline int64_t ez_sized_neg_check(int64_t a, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = -a;
    if (result < min_val || result > max_val)
        ez_panic_code("P0014", "%s negation result is too large; value exceeds the range of this type", type_name);
    return result;
}

/* Sized unsigned integer overflow checks (u8, u16, u32, byte).
 * Operands are int64_t so that signed operands (e.g. byte + int) are
 * handled correctly: a negative right-hand side must fire P0016, not
 * silently wrap to a large uint64 and trigger the wrong P0015 path. */
static inline uint64_t ez_usized_add_check(int64_t a, int64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a + b;
    if (result < 0)
        ez_panic_code("P0016", "%s addition result is negative, but this unsigned type cannot hold negative values", type_name);
    if ((uint64_t)result > max_val)
        ez_panic_code("P0015", "%s addition result is too large; value exceeds the range of this unsigned type", type_name);
    return (uint64_t)result;
}

static inline uint64_t ez_usized_sub_check(int64_t a, int64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a - b;
    if (result < 0)
        ez_panic_code("P0016", "%s subtraction result is negative, but this unsigned type cannot hold negative values", type_name);
    if ((uint64_t)result > max_val)
        ez_panic_code("P0015", "%s subtraction result is too large; value exceeds the range of this unsigned type", type_name);
    return (uint64_t)result;
}

static inline uint64_t ez_usized_mul_check(int64_t a, int64_t b, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    int64_t result = a * b;
    if (result < 0)
        ez_panic_code("P0016", "%s multiplication result is negative, but this unsigned type cannot hold negative values", type_name);
    if ((uint64_t)result > max_val)
        ez_panic_code("P0017", "%s multiplication result is too large; value exceeds the range of this unsigned type", type_name);
    return (uint64_t)result;
}

/* Safe narrowing cast with overflow check */
static inline int64_t ez_cast_check(int64_t v, int64_t min_val, int64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    if (v < min_val || v > max_val)
        ez_panic_code("P0018", "cast to %s failed; value %lld is outside the valid range (%lld to %lld)",
            type_name, (long long)v, (long long)min_val, (long long)max_val);
    return v;
}

static inline uint64_t ez_ucast_check(int64_t v, uint64_t max_val,
    const char *type_name, const char *file, int line) {
    (void)file; (void)line;
    if (v < 0 || (uint64_t)v > max_val)
        ez_panic_code("P0019", "cast to %s failed; value %lld is outside the valid range (0 to %llu)",
            type_name, (long long)v, (unsigned long long)max_val);
    return (uint64_t)v;
}

/* Safe uint64 → int64 conversion: panics if value exceeds INT64_MAX */
static inline int64_t ez_uint_to_int_check(uint64_t v, const char *file, int line) {
    (void)file; (void)line;
    if (v > (uint64_t)9223372036854775807LL)
        ez_panic_code("P0018", "cast to int failed; value %llu is outside the valid range (0 to 9223372036854775807)",
            (unsigned long long)v);
    return (int64_t)v;
}

/* Safe float-to-int conversion with overflow check */
static inline int64_t ez_float_to_int(double v, const char *file, int line) {
    (void)file; (void)line;
    if (v > 9.223372036854775e+18 || v < -9.223372036854775e+18 ||
        v != v /* NaN */)
        ez_panic_code("P0020", "cannot convert float to int; the value is too large, too small, or NaN");
    return (int64_t)v;
}

/* Safe float-to-uint conversion with range check */
static inline uint64_t ez_float_to_uint(double v, const char *file, int line) {
    (void)file; (void)line;
    /* 1.8446744073709552e+19 == 2^64 exactly as a double; any value >= it overflows uint64 */
    if (v < 0.0 || v >= 1.8446744073709552e+19 || v != v /* NaN */)
        ez_panic_code("P0091", "cannot convert float to uint; the value is negative, too large, or NaN");
    return (uint64_t)v;
}

/* --- Result types for (value, Error) destructuring --- */

/* Forward declarations for types defined in other headers */
struct EzArray_tag;
struct EzMap_tag;

typedef struct { int64_t v0; EzError *v1; } EzResult_int;
typedef struct { void *v0; EzError *v1; } EzResult_ptr;

#endif
