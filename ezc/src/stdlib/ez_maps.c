/*
 * ez_maps.c - @maps module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_maps.h"
#include <string.h>

EzArray ez_maps_keys(EzArena *arena, EzMap *m) {
    EzArray arr = ez_array_new(arena, m->key_size, m->count > 0 ? m->count : 4);
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] == 1) {
            void *key = (char *)m->keys + (size_t)i * (size_t)m->key_size;
            ez_array_push(arena, &arr, key);
        }
    }
    return arr;
}

EzArray ez_maps_values(EzArena *arena, EzMap *m) {
    EzArray arr = ez_array_new(arena, m->value_size, m->count > 0 ? m->count : 4);
    for (int32_t i = 0; i < m->capacity; i++) {
        if (m->states[i] == 1) {
            void *val = (char *)m->values + (size_t)i * (size_t)m->value_size;
            ez_array_push(arena, &arr, val);
        }
    }
    return arr;
}

bool ez_maps_has_key(EzMap *m, const void *key) {
    return ez_map_has(m, key);
}
