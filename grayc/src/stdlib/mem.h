/*
 * mem.h — Public interface for the mem stdlib module.
 * Declares arena creation, destruction, reset, usage queries, and
 * raw memory copy/zero operations for manual memory management.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_MEM_H
#define GRAY_MEM_H

#include "../runtime/runtime.h"
#include "../runtime/array.h"

/* mem.arena(size) — create a new arena with initial block size */
GrayArena *gray_mem_arena(int64_t size);

/* mem.destroy(arena) — destroy an arena and free all its memory */
void gray_mem_destroy(GrayArena *arena, const char *file, int line);

/* mem.reset(arena) — reset arena for reuse (keeps allocated blocks) */
void gray_mem_reset(GrayArena *arena);

/* mem.usage(arena) — return bytes currently used in the arena */
int64_t gray_mem_usage(GrayArena *arena);

/*
 * mem.alloc(arena, value) is handled by the codegen directly —
 * it emits the allocation on the specified arena instead of
 * gray_default_arena. No C function needed.
 */

/* mem.copy(dest, src, n) — copy n bytes from src to dest (memcpy) */
void gray_mem_copy(void *dest, const void *src, int64_t n);

/* mem.zero(ptr, n) — zero n bytes starting at ptr (memset 0) */
void gray_mem_zero(void *ptr, int64_t n);

/* mem.set(ptr, val, n) — set n bytes to val (memset) */
void gray_mem_set(void *ptr, int64_t val, int64_t n);

/* mem.size_of(Type) — handled by codegen (sizeof) */

#endif
