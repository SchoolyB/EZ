/*
 * arena.c — Block-based arena allocator for compiler-internal memory,
 * providing bump allocation with bulk free on compilation finish.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "arena.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

#define ALIGN_UP(x, align) (((x) + (align) - 1) & ~((align) - 1))

static ArenaBlock *arena_block_create(size_t size) {
    ArenaBlock *block = malloc(sizeof(ArenaBlock) + size);
    if (!block) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    block->next = NULL;
    block->size = size;
    block->used = 0;
    return block;
}

Arena *arena_create(size_t initial_size) {
    Arena *arena = malloc(sizeof(Arena));
    if (!arena) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    arena->default_block_size = initial_size;
    arena->first = arena_block_create(initial_size);
    arena->current = arena->first;
    return arena;
}

void *arena_alloc(Arena *arena, size_t size) {
    size = ALIGN_UP(size, 8);

    if (arena->current->used + size > arena->current->size) {
        size_t block_size = arena->default_block_size;
        if (size > block_size) {
            block_size = size;
        }
        ArenaBlock *block = arena_block_create(block_size);
        arena->current->next = block;
        arena->current = block;
    }

    void *ptr = arena->current->data + arena->current->used;
    arena->current->used += size;
    return ptr;
}

char *arena_copy_string(Arena *arena, const char *source) {
    size_t len = strlen(source);
    char *duplicated = arena_alloc(arena, len + 1);
    memcpy(duplicated, source, len + 1);
    return duplicated;
}

char *arena_copy_string_with_length(Arena *arena, const char *source, size_t len) {
    char *duplicated = arena_alloc(arena, len + 1);
    memcpy(duplicated, source, len);
    duplicated[len] = '\0';
    return duplicated;
}

void arena_destroy(Arena *arena) {
    ArenaBlock *block = arena->first;
    while (block) {
        ArenaBlock *next = block->next;
        free(block);
        block = next;
    }
    free(arena);
}
