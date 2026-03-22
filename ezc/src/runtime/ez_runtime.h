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
} EzArena;

EzArena *ez_arena_create(size_t initial_size);
void *ez_arena_alloc(EzArena *arena, size_t size);
void ez_arena_reset(EzArena *arena);
void ez_arena_destroy(EzArena *arena);
size_t ez_arena_usage(EzArena *arena);

/* Global default arena */
extern EzArena *ez_default_arena;

/* --- String --- */

typedef struct {
    const char *data;
    int32_t len;
} EzString;

/* Create a string from a C string literal (no copy, points to static data) */
static inline EzString ez_string_lit(const char *s) {
    EzString str;
    str.data = s;
    str.len = (int32_t)strlen(s);
    return str;
}

/* Create a string with a copy on the arena */
EzString ez_string_new(EzArena *arena, const char *s, int32_t len);

/* String formatting (for interpolation) */
EzString ez_string_format(EzArena *arena, const char *fmt, ...);

/* --- Runtime Init/Shutdown --- */

void ez_runtime_init(void);
void ez_runtime_shutdown(void);

/* --- Panic --- */

void ez_panic(const char *file, int line, const char *fmt, ...)
    __attribute__((format(printf, 3, 4), noreturn));

#endif
