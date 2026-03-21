/*
 * arena.h - Compiler-internal arena allocator
 *
 * Used for allocating AST nodes, strings, and other compiler data.
 * All memory is freed in one shot when compilation of a file finishes.
 */

#ifndef EZC_ARENA_H
#define EZC_ARENA_H

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
