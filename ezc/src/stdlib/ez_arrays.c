/*
 * ez_arrays.c - @arrays module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_arrays.h"
#include <string.h>
#include <stdlib.h>

void ez_arrays_append(EzArena *arena, EzArray *arr, const void *value) {
    ez_array_push(arena, arr, value);
}

void ez_arrays_insert(EzArena *arena, EzArray *arr, int32_t index, const void *value) {
    /* Make room by pushing a dummy, then shift elements right */
    char zero[32] = {0};
    ez_array_push(arena, arr, zero);
    char *data = (char *)arr->data;
    size_t es = (size_t)arr->elem_size;
    memmove(data + (index + 1) * es, data + index * es, (arr->len - 1 - index) * es);
    memcpy(data + index * es, value, es);
}

void ez_arrays_remove_at(EzArray *arr, int32_t index) {
    if (index < 0 || index >= arr->len) return;
    char *data = (char *)arr->data;
    size_t es = (size_t)arr->elem_size;
    memmove(data + index * es, data + (index + 1) * es, (arr->len - 1 - index) * es);
    arr->len--;
}

void ez_arrays_clear(EzArray *arr) {
    arr->len = 0;
}

bool ez_arrays_is_empty(EzArray *arr) {
    return arr->len == 0;
}

bool ez_arrays_contains_int(EzArray *arr, int64_t value) {
    for (int32_t i = 0; i < arr->len; i++) {
        if (*(int64_t *)((char *)arr->data + i * arr->elem_size) == value) return true;
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

int64_t ez_arrays_sum(EzArray *arr) {
    int64_t total = 0;
    for (int32_t i = 0; i < arr->len; i++) {
        total += *(int64_t *)((char *)arr->data + i * arr->elem_size);
    }
    return total;
}

int64_t ez_arrays_min(EzArray *arr) {
    if (arr->len == 0) return 0;
    int64_t m = *(int64_t *)arr->data;
    for (int32_t i = 1; i < arr->len; i++) {
        int64_t v = *(int64_t *)((char *)arr->data + i * arr->elem_size);
        if (v < m) m = v;
    }
    return m;
}

int64_t ez_arrays_max(EzArray *arr) {
    if (arr->len == 0) return 0;
    int64_t m = *(int64_t *)arr->data;
    for (int32_t i = 1; i < arr->len; i++) {
        int64_t v = *(int64_t *)((char *)arr->data + i * arr->elem_size);
        if (v > m) m = v;
    }
    return m;
}

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

void ez_arrays_sort(EzArray *arr) {
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_i64_asc);
}

void ez_arrays_sort_desc(EzArray *arr) {
    if (arr->len <= 1) return;
    qsort(arr->data, (size_t)arr->len, (size_t)arr->elem_size, cmp_i64_desc);
}
