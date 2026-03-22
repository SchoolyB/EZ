/*
 * ez_mem.c - @mem module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_mem.h"

EzArena *ez_mem_arena(int64_t size) {
    if (size <= 0) size = 4096;
    return ez_arena_create((size_t)size);
}

void ez_mem_destroy(EzArena *arena) {
    if (arena) ez_arena_destroy(arena);
}

void ez_mem_reset(EzArena *arena) {
    if (arena) ez_arena_reset(arena);
}

int64_t ez_mem_usage(EzArena *arena) {
    if (!arena) return 0;
    return (int64_t)ez_arena_usage(arena);
}
