/*
 * ez_mem.h - @mem module for EZC (memory management)
 *
 * This is an EZC-only module. The EZ interpreter does not need it
 * because it uses Go's garbage collector.
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
void ez_mem_destroy(EzArena *arena);

/* mem.reset(arena) — reset arena for reuse (keeps allocated blocks) */
void ez_mem_reset(EzArena *arena);

/* mem.usage(arena) — return bytes currently used in the arena */
int64_t ez_mem_usage(EzArena *arena);

/*
 * mem.alloc(arena, value) is handled by the codegen directly —
 * it emits the allocation on the specified arena instead of
 * ez_default_arena. No C function needed.
 */

#endif
