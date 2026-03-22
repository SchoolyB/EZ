/*
 * ez_map.c - Hash map implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_map.h"
#include <string.h>

/* FNV-1a hash */
static uint64_t hash_bytes(const void *data, int32_t size) {
    const uint8_t *bytes = (const uint8_t *)data;
    uint64_t hash = 14695981039346656037ULL;
    for (int32_t i = 0; i < size; i++) {
        hash ^= bytes[i];
        hash *= 1099511628211ULL;
    }
    return hash;
}

/* Hash for EzString keys — hash the string data, not the struct */
static uint64_t hash_ez_string(const void *key, int32_t key_size) {
    if (key_size == (int32_t)sizeof(EzString)) {
        const EzString *s = (const EzString *)key;
        return hash_bytes(s->data, s->len);
    }
    return hash_bytes(key, key_size);
}

/* Compare keys — special handling for EzString */
static bool keys_equal(const void *a, const void *b, int32_t key_size) {
    if (key_size == (int32_t)sizeof(EzString)) {
        const EzString *sa = (const EzString *)a;
        const EzString *sb = (const EzString *)b;
        if (sa->len != sb->len) return false;
        return memcmp(sa->data, sb->data, (size_t)sa->len) == 0;
    }
    return memcmp(a, b, (size_t)key_size) == 0;
}

static void *key_ptr(EzMap *m, int32_t idx) {
    return (char *)m->keys + (size_t)idx * (size_t)m->key_size;
}

static void *val_ptr(EzMap *m, int32_t idx) {
    return (char *)m->values + (size_t)idx * (size_t)m->value_size;
}

EzMap ez_map_new(EzArena *arena, int32_t key_size, int32_t value_size, int32_t initial_cap) {
    if (initial_cap < 8) initial_cap = 8;
    EzMap m;
    m.key_size = key_size;
    m.value_size = value_size;
    m.count = 0;
    m.capacity = initial_cap;
    m.keys = ez_arena_alloc(arena, (size_t)initial_cap * (size_t)key_size);
    m.values = ez_arena_alloc(arena, (size_t)initial_cap * (size_t)value_size);
    m.states = ez_arena_alloc(arena, (size_t)initial_cap);
    memset(m.states, 0, (size_t)initial_cap);
    return m;
}

static int32_t find_slot(EzMap *m, const void *key) {
    uint64_t h = hash_ez_string(key, m->key_size);
    int32_t idx = (int32_t)(h % (uint64_t)m->capacity);
    for (int32_t i = 0; i < m->capacity; i++) {
        int32_t probe = (idx + i) % m->capacity;
        if (m->states[probe] == 0) return -1; /* empty — not found */
        if (m->states[probe] == 1 && keys_equal(key_ptr(m, probe), key, m->key_size)) {
            return probe;
        }
    }
    return -1;
}

static void rehash(EzArena *arena, EzMap *m) {
    int32_t old_cap = m->capacity;
    void *old_keys = m->keys;
    void *old_values = m->values;
    uint8_t *old_states = m->states;

    m->capacity = old_cap * 2;
    m->keys = ez_arena_alloc(arena, (size_t)m->capacity * (size_t)m->key_size);
    m->values = ez_arena_alloc(arena, (size_t)m->capacity * (size_t)m->value_size);
    m->states = ez_arena_alloc(arena, (size_t)m->capacity);
    memset(m->states, 0, (size_t)m->capacity);
    m->count = 0;

    for (int32_t i = 0; i < old_cap; i++) {
        if (old_states[i] == 1) {
            ez_map_set(arena, m,
                (char *)old_keys + (size_t)i * (size_t)m->key_size,
                (char *)old_values + (size_t)i * (size_t)m->value_size);
        }
    }
}

void *ez_map_get(EzMap *m, const void *key) {
    int32_t idx = find_slot(m, key);
    if (idx < 0) return NULL;
    return val_ptr(m, idx);
}

void ez_map_set(EzArena *arena, EzMap *m, const void *key, const void *value) {
    /* Check load factor */
    if (m->count * 4 >= m->capacity * 3) {
        rehash(arena, m);
    }

    uint64_t h = hash_ez_string(key, m->key_size);
    int32_t idx = (int32_t)(h % (uint64_t)m->capacity);
    for (int32_t i = 0; i < m->capacity; i++) {
        int32_t probe = (idx + i) % m->capacity;
        if (m->states[probe] == 0 || m->states[probe] == 2) {
            /* Empty or tombstone — insert here */
            memcpy(key_ptr(m, probe), key, (size_t)m->key_size);
            memcpy(val_ptr(m, probe), value, (size_t)m->value_size);
            m->states[probe] = 1;
            m->count++;
            return;
        }
        if (m->states[probe] == 1 && keys_equal(key_ptr(m, probe), key, m->key_size)) {
            /* Update existing */
            memcpy(val_ptr(m, probe), value, (size_t)m->value_size);
            return;
        }
    }
}

bool ez_map_has(EzMap *m, const void *key) {
    return find_slot(m, key) >= 0;
}

bool ez_map_remove(EzMap *m, const void *key) {
    int32_t idx = find_slot(m, key);
    if (idx < 0) return false;
    m->states[idx] = 2; /* tombstone */
    m->count--;
    return true;
}

void *ez_map_get_str(EzMap *m, EzString key) {
    return ez_map_get(m, &key);
}

void ez_map_set_str(EzArena *arena, EzMap *m, EzString key, const void *value) {
    ez_map_set(arena, m, &key, value);
}

void *ez_map_key_at(EzMap *m, int32_t internal_idx) {
    return key_ptr(m, internal_idx);
}

void *ez_map_value_at(EzMap *m, int32_t internal_idx) {
    return val_ptr(m, internal_idx);
}
