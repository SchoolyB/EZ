/*
 * gray_runtime.c - EZC runtime implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_runtime.h"
#include <stdarg.h>

/* --- Per-thread default arena --- */

_Thread_local EzArena *gray_default_arena = NULL;

/* --- Persistent heap arena (used by new()) --- */

_Thread_local EzArena *gray_heap_arena = NULL;

/* --- Arena Allocator --- */

#define ALIGN_UP(x, a) (((x) + (a) - 1) & ~((a) - 1))

static EzArenaBlock *gray_arena_block_create(size_t size) {
    EzArenaBlock *block = (EzArenaBlock *)malloc(sizeof(EzArenaBlock) + size);
    if (!block) {
        fprintf(stderr, "EZ runtime: out of memory\n");
        exit(1);
    }
    block->next = NULL;
    block->size = size;
    block->used = 0;
    return block;
}

EzArena *gray_arena_create(size_t initial_size) {
    EzArena *arena = (EzArena *)malloc(sizeof(EzArena));
    if (!arena) {
        fprintf(stderr, "EZ runtime: out of memory\n");
        exit(1);
    }
    arena->default_block_size = initial_size;
    arena->first = gray_arena_block_create(initial_size);
    arena->current = arena->first;
    arena->destroyed = false;
    return arena;
}

void *gray_arena_alloc(EzArena *arena, size_t size) {
    if (arena->destroyed)
        gray_panic_code("P0001", "cannot allocate from a destroyed arena; mem.destroy() was already called on this arena");
    size = ALIGN_UP(size, 8);
    if (arena->current->used + size > arena->current->size) {
        size_t block_size = arena->default_block_size;
        if (size > block_size) block_size = size;
        EzArenaBlock *block = gray_arena_block_create(block_size);
        arena->current->next = block;
        arena->current = block;
    }
    void *ptr = arena->current->data + arena->current->used;
    arena->current->used += size;
    memset(ptr, 0, size);
    return ptr;
}

void gray_arena_reset(EzArena *arena) {
    /* Reset all blocks — reuse memory without freeing */
    EzArenaBlock *block = arena->first;
    while (block) {
        block->used = 0;
        block = block->next;
    }
    arena->current = arena->first;
}

void gray_arena_destroy(EzArena *arena, const char *file, int line) {
    if (!arena) return;
    if (arena->destroyed)
        gray_panic_code("P0002", "mem.destroy() called on an arena that was already destroyed; each arena can only be destroyed once");
    EzArenaBlock *block = arena->first;
    while (block) {
        EzArenaBlock *next = block->next;
        free(block);
        block = next;
    }
    arena->first = NULL;
    arena->current = NULL;
    arena->destroyed = true;
    /* Don't free the arena struct — keep it alive so the destroyed flag
     * can be checked if the user calls destroy again. It will be cleaned
     * up at process exit. */
}

size_t gray_arena_usage(EzArena *arena) {
    size_t total = 0;
    EzArenaBlock *block = arena->first;
    while (block) {
        total += block->used;
        block = block->next;
    }
    return total;
}

/* --- Error --- */

EzError *gray_error_new(EzArena *arena, EzString message) {
    EzError *err = (EzError *)gray_arena_alloc(arena, sizeof(EzError));
    err->message = gray_string_new(arena, message.data, message.len);
    err->code = gray_string_lit("");
    return err;
}

/* --- String --- */

EzString gray_string_new(EzArena *arena, const char *s, int32_t len) {
    char *data = (char *)gray_arena_alloc(arena, (size_t)len + 1);
    memcpy(data, s, (size_t)len);
    data[len] = '\0';
    EzString str;
    str.data = data;
    str.len = len;
    return str;
}

EzString gray_c_string_dup(EzArena *arena, const char *s) {
    if (s == NULL) return gray_string_lit("");
    size_t n = strlen(s);
    if (n > (size_t)INT32_MAX) n = (size_t)INT32_MAX;
    char *data = (char *)gray_arena_alloc(arena, n + 1);
    memcpy(data, s, n);
    data[n] = '\0';
    EzString str;
    str.data = data;
    str.len = (int32_t)n;
    return str;
}

EzString gray_string_format(EzArena *arena, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);
    int needed = vsnprintf(NULL, 0, fmt, args);
    va_end(args);

    if (needed < 0) {
        return gray_string_lit("");
    }

    char *data = (char *)gray_arena_alloc(arena, (size_t)needed + 1);
    va_start(args, fmt);
    vsnprintf(data, (size_t)needed + 1, fmt, args);
    va_end(args);

    EzString str;
    str.data = data;
    str.len = (int32_t)needed;
    return str;
}

EzString gray_string_concat(EzArena *arena, EzString a, EzString b) {
    if (b.len > INT32_MAX - a.len) {
        fprintf(stderr, "EZ runtime: string concatenation overflow\n");
        exit(1);
    }
    int32_t new_len = a.len + b.len;
    char *data = (char *)gray_arena_alloc(arena, (size_t)new_len + 1);
    memcpy(data, a.data, (size_t)a.len);
    memcpy(data + a.len, b.data, (size_t)b.len);
    data[new_len] = '\0';
    EzString s = { data, new_len };
    return s;
}

/* --- Scope-based memory management (#1521) --- */

EzScopeMark gray_scope_save(EzArena *arena) {
    EzScopeMark m;
    m.block = arena->current;
    m.used = arena->current ? arena->current->used : 0;
    return m;
}

void gray_scope_restore(EzArena *arena, EzScopeMark mark) {
    /* Reset all blocks AFTER the marked block */
    if (!mark.block) return;
    EzArenaBlock *block = mark.block->next;
    while (block) {
        block->used = 0;
        block = block->next;
    }
    /* Reset the marked block to the saved position */
    mark.block->used = mark.used;
    arena->current = mark.block;
}

/* --- Stack depth guard --- */
int gray_call_depth = 0;

/* --- Runtime Init/Shutdown --- */

void gray_runtime_init(void) {
    gray_default_arena = gray_arena_create(GRAY_DEFAULT_ARENA_SIZE);
    gray_heap_arena = gray_arena_create(GRAY_DEFAULT_ARENA_SIZE);
}

void gray_runtime_shutdown(void) {
    if (gray_default_arena) {
        gray_arena_destroy(gray_default_arena, __FILE__, __LINE__);
        gray_default_arena = NULL;
    }
    if (gray_heap_arena) {
        gray_arena_destroy(gray_heap_arena, __FILE__, __LINE__);
        gray_heap_arena = NULL;
    }
}

/* --- Panic --- */

void gray_panic(const char *file, int line, const char *fmt, ...) {
    fflush(stdout);
    fprintf(stderr, "panic at %s:%d: ", file, line);
    va_list args;
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fprintf(stderr, "\n");
    exit(1);
}

void gray_panic_code(const char *code, const char *fmt, ...) {
    fflush(stdout);
    fprintf(stderr, "panic[%s]: ", code);
    va_list args;
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fprintf(stderr, "\n");
    exit(1);
}
