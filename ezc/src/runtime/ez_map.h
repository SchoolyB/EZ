/*
 * ez_map.h - Hash map type for EZC
 *
 * Open-addressing hash table with linear probing.
 * Keys and values stored as fixed-size blobs.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_MAP_H
#define EZ_MAP_H

#include "ez_runtime.h"

typedef struct {
    void *keys;
    void *values;
    uint8_t *states;        /* 0=empty, 1=occupied, 2=tombstone */
    int32_t *order;         /* insertion-order slot indices */
    int32_t count;
    int32_t capacity;
    int32_t key_size;
    int32_t value_size;
    int32_t order_len;      /* number of entries in order array */
    int32_t iterating;      /* >0 while a for_each is active */
} EzMap;

/* Create an empty map */
EzMap ez_map_new(EzArena *arena, int32_t key_size, int32_t value_size, int32_t initial_cap);

/* Get a pointer to the value for a key, or NULL if not found */
void *ez_map_get(EzMap *m, const void *key);

/* Set a key-value pair (inserts or updates) */
void ez_map_set(EzArena *arena, EzMap *m, const void *key, const void *value);

/* Check if a key exists */
bool ez_map_has(EzMap *m, const void *key);

/* Remove a key */
bool ez_map_remove(EzMap *m, const void *key);

/* Clear all entries */
void ez_map_clear(EzMap *m);

/* String-keyed convenience functions */
void *ez_map_get_str(EzMap *m, EzString key);
void ez_map_set_str(EzArena *arena, EzMap *m, EzString key, const void *value);

/* Get key at internal index (for iteration) */
void *ez_map_key_at(EzMap *m, int32_t internal_idx);
void *ez_map_value_at(EzMap *m, int32_t internal_idx);

/* Deep copy: allocate a fresh map with independent backing storage
 * (keys, values, states, order) so mutations to the copy do not affect
 * the original. */
EzMap ez_map_copy(EzArena *arena, const EzMap *src);

#endif
