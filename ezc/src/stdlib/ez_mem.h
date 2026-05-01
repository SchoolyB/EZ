/*
 * ez_mem.h - @mem module for EZ (memory management)
 *
 * Manual memory management via arena allocation.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_MEM_H
#define EZ_MEM_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* mem.arena(size) — create a new arena with initial block size */
EzArena *ez_mem_arena(int64_t size);

/* mem.destroy(arena) — destroy an arena and free all its memory */
void ez_mem_destroy(EzArena *arena, const char *file, int line);

/* mem.reset(arena) — reset arena for reuse (keeps allocated blocks) */
void ez_mem_reset(EzArena *arena);

/* mem.usage(arena) — return bytes currently used in the arena */
int64_t ez_mem_usage(EzArena *arena);

/*
 * mem.alloc(arena, value) is handled by the codegen directly —
 * it emits the allocation on the specified arena instead of
 * ez_default_arena. No C function needed.
 */

/* mem.copy(dest, src, n) — copy n bytes from src to dest (memcpy) */
void ez_mem_copy(void *dest, const void *src, int64_t n);

/* mem.zero(ptr, n) — zero n bytes starting at ptr (memset 0) */
void ez_mem_zero(void *ptr, int64_t n);

/* mem.set(ptr, val, n) — set n bytes to val (memset) */
void ez_mem_set(void *ptr, int64_t val, int64_t n);

/* mem.size_of(Type) — handled by codegen (sizeof) */

#endif
