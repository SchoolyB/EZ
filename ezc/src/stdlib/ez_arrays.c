/*
 * ez_arrays.c - arrays module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_arrays.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

/* === Modification === */

void ez_arrays_append(EzArena *arena, EzArray *arr, const void *value) {
    ez_array_push(arena, arr, value);
}

void ez_arrays_insert_at(EzArena *arena, EzArray *arr, int32_t index, const void *value) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (index < 0 || index > arr->len) {
        ez_panic_code("P0043",
            "arrays.insert_at: index %d is out of bounds for an array of length %d",
            index, arr->len);
    }

    /* Grow if needed — same policy as ez_array_push */
    if (arr->len >= arr->cap) {
        int32_t new_cap = arr->cap < EZ_ARRAY_MIN_CAP ? EZ_ARRAY_MIN_CAP : arr->cap * 2;
        if (new_cap < arr->cap) {
            fprintf(stderr, "EZ runtime: array capacity overflow\n");
            exit(1);
        }
        void *new_data = ez_arena_alloc(arena, (size_t)new_cap * (size_t)arr->elem_size);
        if (arr->data && arr->len > 0) {
            memcpy(new_data, arr->data, (size_t)arr->len * (size_t)arr->elem_size);
        }
        arr->data = new_data;
        arr->cap = new_cap;
    }

    size_t es = (size_t)arr->elem_size;
    char *data = (char *)arr->data;
    if (index < arr->len) {
        memmove(data + (index + 1) * es, data + index * es,
                (size_t)(arr->len - index) * es);
    }
    memcpy(data + index * es, value, es);
    arr->len++;
}

void ez_arrays_prepend(EzArena *arena, EzArray *arr, const void *value) {
    ez_arrays_insert_at(arena, arr, 0, value);
}

void ez_arrays_remove_at(EzArray *arr, int32_t index) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (index < 0 || index >= arr->len)
        ez_panic_code("P0044",
            "arrays.remove_at: index %d is out of bounds for an array of length %d",
            index, arr->len);
    char *data = (char *)arr->data;
    size_t es = (size_t)arr->elem_size;
    memmove(data + index * es, data + (index + 1) * es, (arr->len - 1 - index) * es);
    arr->len--;
}

void ez_arrays_remove_int(EzArray *arr, int64_t value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(int64_t *)((char *)arr->data + i * arr->elem_size) == value) {
            ez_arrays_remove_at(arr, i);
            return;
        }
    }
}

void ez_arrays_remove_float(EzArray *arr, double value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(double *)((char *)arr->data + i * arr->elem_size) == value) {
            ez_arrays_remove_at(arr, i);
            return;
        }
    }
}

void ez_arrays_remove_str(EzArray *arr, EzString value) {
    for (int32_t i = 0; i < arr->len; i++) {
        EzString *s = (EzString *)((char *)arr->data + i * arr->elem_size);
        if (s->len == value.len && memcmp(s->data, value.data, s->len) == 0) {
            ez_arrays_remove_at(arr, i);
            return;
        }
    }
}

void ez_arrays_clear(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    arr->len = 0;
}

void ez_arrays_fill(EzArena *arena, EzArray *arr, const void *value, int32_t count) {
    ez_arrays_clear(arr);
    for (int32_t i = 0; i < count; i++) {
        ez_array_push(arena, arr, value);
    }
}

/* === Access === */

void *ez_arrays_first_ptr(EzArray *arr) {
    if (arr->len == 0)
        ez_panic_code("P0045", "arrays.get_first called on an empty array");
    return arr->data;
}

void *ez_arrays_last_ptr(EzArray *arr) {
    if (arr->len == 0)
        ez_panic_code("P0046", "arrays.get_last called on an empty array");
    return (char *)arr->data + (arr->len - 1) * arr->elem_size;
}

void ez_arrays_remove_first_raw(EzArray *arr, void *out) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len == 0)
        ez_panic_code("P0047", "arrays.remove_first called on an empty array");
    memcpy(out, arr->data, arr->elem_size);
    ez_arrays_remove_at(arr, 0);
}

void ez_arrays_remove_last_raw(EzArray *arr, void *out) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len == 0)
        ez_panic_code("P0048", "arrays.remove_last called on an empty array");
    memcpy(out, (char *)arr->data + (arr->len - 1) * arr->elem_size, arr->elem_size);
    arr->len--;
}

int64_t ez_arrays_get_first(EzArray *arr) {
    if (arr->len == 0) ez_panic_code("P0045", "arrays.get_first called on an empty array");
    return *(int64_t *)arr->data;
}

int64_t ez_arrays_get_last(EzArray *arr) {
    if (arr->len == 0) ez_panic_code("P0046", "arrays.get_last called on an empty array");
    return *(int64_t *)((char *)arr->data + (arr->len - 1) * arr->elem_size);
}

int64_t ez_arrays_remove_last(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len == 0) ez_panic_code("P0048", "arrays.remove_last called on an empty array");
    int64_t val = *(int64_t *)((char *)arr->data + (arr->len - 1) * arr->elem_size);
    arr->len--;
    return val;
}

int64_t ez_arrays_remove_first(EzArray *arr) {
    if (arr->len == 0) ez_panic_code("P0047", "arrays.remove_first called on an empty array");
    int64_t val = *(int64_t *)arr->data;
    ez_arrays_remove_at(arr, 0);
    return val;
}

/* === Query === */

bool ez_arrays_is_empty(EzArray *arr) {
    return arr->len == 0;
}

bool ez_arrays_contains_int(EzArray *arr, int64_t value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(int64_t *)((char *)arr->data + i * arr->elem_size) == value) return true;
    }
    return false;
}

bool ez_arrays_contains_float(EzArray *arr, double value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(double *)((char *)arr->data + i * arr->elem_size) == value) return true;
    }
    return false;
}

bool ez_arrays_contains_str(EzArray *arr, EzString value) {
    for (int32_t i = 0; i < arr->len; i++) {
        EzString *s = (EzString *)((char *)arr->data + i * arr->elem_size);
        if (s->len == value.len && memcmp(s->data, value.data, s->len) == 0) return true;
    }
    return false;
}

int64_t ez_arrays_index_of_int(EzArray *arr, int64_t value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(int64_t *)((char *)arr->data + i * arr->elem_size) == value) return i;
    }
    return -1;
}

int64_t ez_arrays_index_of_str(EzArray *arr, EzString value) {
    for (int32_t i = 0; i < arr->len; i++) {
        EzString *s = (EzString *)((char *)arr->data + i * arr->elem_size);
        if (s->len == value.len && memcmp(s->data, value.data, s->len) == 0) return i;
    }
    return -1;
}

int64_t ez_arrays_count(EzArray *arr, int64_t value) {
    int64_t c = 0;
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(int64_t *)((char *)arr->data + i * arr->elem_size) == value) c++;
    }
    return c;
}

bool ez_arrays_is_equal_prim(EzArray *a, EzArray *b) {
    if (a->len != b->len) return false;
    if (a->elem_size != b->elem_size) return false;
    if (a->len == 0) return true;
    return memcmp(a->data, b->data, (size_t)a->len * (size_t)a->elem_size) == 0;
}

bool ez_arrays_is_equal_str(EzArray *a, EzArray *b) {
    if (a->len != b->len) return false;
    for (int32_t i = 0; i < a->len; i++) {
        EzString *sa = (EzString *)((char *)a->data + i * a->elem_size);
        EzString *sb = (EzString *)((char *)b->data + i * b->elem_size);
        if (sa->len != sb->len) return false;
        if (sa->len > 0 && memcmp(sa->data, sb->data, sa->len) != 0) return false;
    }
    return true;
}

/* === Transformation === */

EzArray ez_arrays_reverse(EzArena *arena, EzArray *arr) {
    EzArray result = ez_array_new(arena, arr->elem_size, arr->len);
    char *src = (char *)arr->data;
    for (int32_t i = arr->len - 1; i >= 0; i--) {
        ez_array_push(arena, &result, src + i * arr->elem_size);
    }
    return result;
}

EzArray ez_arrays_slice(EzArena *arena, EzArray *arr, int32_t start, int32_t end) {
    if (start < 0) start = 0;
    if (end > arr->len) end = arr->len;
    if (start >= end) return ez_array_new(arena, arr->elem_size, 1);
    int32_t count = end - start;
    return ez_array_from(arena, (char *)arr->data + start * arr->elem_size, arr->elem_size, count);
}

EzArray ez_arrays_concat(EzArena *arena, EzArray *a, EzArray *b) {
    EzArray result = ez_array_copy(arena, a);
    char *src = (char *)b->data;
    for (int32_t i = 0; i < b->len; i++) {
        ez_array_push(arena, &result, src + i * b->elem_size);
    }
    return result;
}

EzArray ez_arrays_deduplicate(EzArena *arena, EzArray *arr) {
    EzArray result = ez_array_new(arena, arr->elem_size, arr->len);
    char *data = (char *)arr->data;
    size_t es = (size_t)arr->elem_size;
    for (int32_t i = 0; i < arr->len; i++) {
        bool found = false;
        char *rdata = (char *)result.data;
        for (int32_t j = 0; j < result.len; j++) {
            if (memcmp(data + i * es, rdata + j * es, es) == 0) { found = true; break; }
        }
        if (!found) ez_array_push(arena, &result, data + i * es);
    }
    return result;
}

EzArray ez_arrays_flatten(EzArena *arena, EzArray *arr) {
    /* Flatten one level: [[int]] -> [int]. Each element is an EzArray. */
    EzArray result = ez_array_new(arena, sizeof(int64_t), 8);
    for (int32_t i = 0; i < arr->len; i++) {
        EzArray *inner = (EzArray *)((char *)arr->data + i * arr->elem_size);
        char *idata = (char *)inner->data;
        for (int32_t j = 0; j < inner->len; j++) {
            ez_array_push(arena, &result, idata + j * inner->elem_size);
        }
    }
    return result;
}

EzArray ez_arrays_split_every(EzArena *arena, EzArray *arr, int32_t size) {
    if (size <= 0) size = 1;
    EzArray result = ez_array_new(arena, sizeof(EzArray), 4);
    char *data = (char *)arr->data;
    size_t es = (size_t)arr->elem_size;
    for (int32_t i = 0; i < arr->len; i += size) {
        int32_t chunk_len = (i + size <= arr->len) ? size : (arr->len - i);
        EzArray chunk = ez_array_from(arena, data + i * es, arr->elem_size, chunk_len);
        ez_array_push(arena, &result, &chunk);
    }
    return result;
}

EzArray ez_arrays_pair(EzArena *arena, EzArray *a, EzArray *b) {
    int32_t len = a->len < b->len ? a->len : b->len;
    EzArray result = ez_array_new(arena, sizeof(EzArray), len);
    for (int32_t i = 0; i < len; i++) {
        EzArray pair_arr = ez_array_new(arena, a->elem_size, 2);
        ez_array_push(arena, &pair_arr, (char *)a->data + i * a->elem_size);
        ez_array_push(arena, &pair_arr, (char *)b->data + i * b->elem_size);
        ez_array_push(arena, &result, &pair_arr);
    }
    return result;
}

/* === Computation === */

int64_t ez_arrays_get_sum(EzArray *arr) {
    int64_t total = 0;
    for (int32_t i = 0; i < arr->len; i++) {
        total += *(int64_t *)((char *)arr->data + i * arr->elem_size);
    }
    return total;
}

int64_t ez_arrays_get_min(EzArray *arr) {
    if (arr->len == 0) return 0;
    int64_t m = *(int64_t *)arr->data;
    for (int32_t i = 1; i < arr->len; i++) {
        int64_t v = *(int64_t *)((char *)arr->data + i * arr->elem_size);
        if (v < m) m = v;
    }
    return m;
}

int64_t ez_arrays_get_max(EzArray *arr) {
    if (arr->len == 0) return 0;
    int64_t m = *(int64_t *)arr->data;
    for (int32_t i = 1; i < arr->len; i++) {
        int64_t v = *(int64_t *)((char *)arr->data + i * arr->elem_size);
        if (v > m) m = v;
    }
    return m;
}

/* === Sort === */

static int cmp_i64_asc(const void *a, const void *b) {
    int64_t va = *(const int64_t *)a;
    int64_t vb = *(const int64_t *)b;
    return (va > vb) - (va < vb);
}

static int cmp_i64_desc(const void *a, const void *b) {
    int64_t va = *(const int64_t *)a;
    int64_t vb = *(const int64_t *)b;
    return (vb > va) - (vb < va);
}

static int cmp_f64_asc(const void *a, const void *b) {
    double va = *(const double *)a;
    double vb = *(const double *)b;
    return (va > vb) - (va < vb);
}

static int cmp_f64_desc(const void *a, const void *b) {
    double va = *(const double *)a;
    double vb = *(const double *)b;
    return (vb > va) - (vb < va);
}

static int cmp_str_asc(const void *a, const void *b) {
    const EzString *sa = (const EzString *)a;
    const EzString *sb = (const EzString *)b;
    int32_t min_len = sa->len < sb->len ? sa->len : sb->len;
    int cmp = memcmp(sa->data, sb->data, (size_t)min_len);
    if (cmp != 0) return cmp;
    return (sa->len > sb->len) - (sa->len < sb->len);
}

static int cmp_str_desc(const void *a, const void *b) {
    return cmp_str_asc(b, a);
}

void ez_arrays_sort_asc(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_i64_asc);
}

void ez_arrays_sort_asc_float(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_f64_asc);
}

void ez_arrays_sort_asc_str(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_str_asc);
}

void ez_arrays_sort_desc(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_i64_desc);
}

void ez_arrays_sort_desc_float(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_f64_desc);
}

void ez_arrays_sort_desc_str(EzArray *arr) {
    if (arr->iterating > 0)
        ez_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_str_desc);
}
