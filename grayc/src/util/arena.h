/*
 * arena.h — Public interface for the block-based arena allocator used to
 * allocate AST nodes, strings, and other compiler data structures.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAYC_ARENA_H
#define GRAYC_ARENA_H

#include <stddef.h>

typedef struct ArenaBlock {
    struct ArenaBlock *next;
    size_t size;
    size_t used;
    char data[];
} ArenaBlock;

typedef struct {
    ArenaBlock *first;
    ArenaBlock *current;
    size_t default_block_size;
} Arena;

Arena *arena_create(size_t initial_size);
void *arena_alloc(Arena *a, size_t size);
char *arena_strdup(Arena *a, const char *s);
char *arena_strndup(Arena *a, const char *s, size_t len);
void arena_destroy(Arena *a);

#endif
