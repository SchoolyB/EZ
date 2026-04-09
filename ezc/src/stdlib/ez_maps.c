/*
 * ez_maps.c - @maps module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_maps.h"
#include <string.h>

EzArray ez_maps_get_keys(EzArena *arena, EzMap *m) {
    EzArray arr = ez_array_new(arena, m->key_size, m->count > 0 ? m->count : 4);
    /* Iterate in insertion order */
    for (int32_t oi = 0; oi < m->order_len; oi++) {
        int32_t slot = m->order[oi];
        if (m->states[slot] == 1) {
            void *key = (char *)m->keys + (size_t)slot * (size_t)m->key_size;
            ez_array_push(arena, &arr, key);
        }
    }
    return arr;
}

EzArray ez_maps_get_values(EzArena *arena, EzMap *m) {
    EzArray arr = ez_array_new(arena, m->value_size, m->count > 0 ? m->count : 4);
    /* Iterate in insertion order */
    for (int32_t oi = 0; oi < m->order_len; oi++) {
        int32_t slot = m->order[oi];
        if (m->states[slot] == 1) {
            void *val = (char *)m->values + (size_t)slot * (size_t)m->value_size;
            ez_array_push(arena, &arr, val);
        }
    }
    return arr;
}

bool ez_maps_has_key(EzMap *m, const void *key) {
    return ez_map_has(m, key);
}

bool ez_maps_is_empty(EzMap *m) {
    return m->count == 0;
}

EzMap ez_maps_merge(EzArena *arena, EzMap *m1, EzMap *m2) {
    EzMap result = ez_map_new(arena, m1->key_size, m1->value_size,
        m1->count + m2->count > 8 ? (m1->count + m2->count) * 2 : 8);
    /* Copy all entries from m1 */
    for (int32_t oi = 0; oi < m1->order_len; oi++) {
        int32_t slot = m1->order[oi];
        if (m1->states[slot] == 1) {
            void *key = (char *)m1->keys + (size_t)slot * (size_t)m1->key_size;
            void *val = (char *)m1->values + (size_t)slot * (size_t)m1->value_size;
            ez_map_set(arena, &result, key, val);
        }
    }
    /* Copy all entries from m2 (overwrites m1 on conflict) */
    for (int32_t oi = 0; oi < m2->order_len; oi++) {
        int32_t slot = m2->order[oi];
        if (m2->states[slot] == 1) {
            void *key = (char *)m2->keys + (size_t)slot * (size_t)m2->key_size;
            void *val = (char *)m2->values + (size_t)slot * (size_t)m2->value_size;
            ez_map_set(arena, &result, key, val);
        }
    }
    return result;
}

bool ez_maps_contains_value(EzMap *m, const void *value) {
    for (int32_t oi = 0; oi < m->order_len; oi++) {
        int32_t slot = m->order[oi];
        if (m->states[slot] == 1) {
            void *val = (char *)m->values + (size_t)slot * (size_t)m->value_size;
            if (memcmp(val, value, (size_t)m->value_size) == 0) return true;
        }
    }
    return false;
}
