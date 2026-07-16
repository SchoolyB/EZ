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
    Arena *a = malloc(sizeof(Arena));
    if (!a) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    a->default_block_size = initial_size;
    a->first = arena_block_create(initial_size);
    a->current = a->first;
    return a;
}

void *arena_alloc(Arena *a, size_t size) {
    size = ALIGN_UP(size, 8);

    if (a->current->used + size > a->current->size) {
        size_t block_size = a->default_block_size;
        if (size > block_size) {
            block_size = size;
        }
        ArenaBlock *block = arena_block_create(block_size);
        a->current->next = block;
        a->current = block;
    }

    void *ptr = a->current->data + a->current->used;
    a->current->used += size;
    return ptr;
}

char *arena_strdup(Arena *a, const char *s) {
    size_t len = strlen(s);
    char *dup = arena_alloc(a, len + 1);
    memcpy(dup, s, len + 1);
    return dup;
}

char *arena_strndup(Arena *a, const char *s, size_t len) {
    char *dup = arena_alloc(a, len + 1);
    memcpy(dup, s, len);
    dup[len] = '\0';
    return dup;
}

void arena_destroy(Arena *a) {
    ArenaBlock *block = a->first;
    while (block) {
        ArenaBlock *next = block->next;
        free(block);
        block = next;
    }
    free(a);
}
