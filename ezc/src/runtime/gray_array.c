/*
 * gray_array.c - Dynamic array implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_array.h"
#include <string.h>

EzArray gray_array_new(EzArena *arena, int32_t elem_size, int32_t initial_cap) {
    EzArray arr;
    arr.elem_size = elem_size;
    arr.len = 0;
    arr.cap = initial_cap > 0 ? initial_cap : GRAY_ARRAY_MIN_CAP;
    arr.iterating = 0;
    arr.data = gray_arena_alloc(arena, (size_t)arr.cap * (size_t)arr.elem_size);
    return arr;
}

EzArray gray_array_from(EzArena *arena, const void *data, int32_t elem_size, int32_t count) {
    EzArray arr;
    arr.elem_size = elem_size;
    arr.len = count;
    arr.cap = count > 0 ? count : GRAY_ARRAY_MIN_CAP;
    arr.iterating = 0;
    arr.data = gray_arena_alloc(arena, (size_t)arr.cap * (size_t)arr.elem_size);
    if (count > 0 && data) {
        memcpy(arr.data, data, (size_t)count * (size_t)elem_size);
    }
    return arr;
}

void *gray_array_get_ptr(EzArray *arr, int32_t index, const char *file, int line) {
    if (index < 0 || index >= arr->len) {
        gray_panic_code("P0033", "index out of bounds; tried to access index %d but the length is %d", index, arr->len);
    }
    return (char *)arr->data + (size_t)index * (size_t)arr->elem_size;
}

void gray_array_set(EzArray *arr, int32_t index, const void *value, const char *file, int line) {
    if (arr->iterating > 0)
        gray_panic_code("P0034", "cannot modify array during for_each iteration");
    if (index < 0 || index >= arr->len) {
        gray_panic_code("P0033", "index out of bounds; tried to access index %d but the length is %d", index, arr->len);
    }
    memcpy((char *)arr->data + (size_t)index * (size_t)arr->elem_size,
           value, (size_t)arr->elem_size);
}

void gray_array_push(EzArena *arena, EzArray *arr, const void *value) {
    if (arr->iterating > 0)
        gray_panic_code("P0034", "cannot modify array during for_each iteration");
    if (arr->len >= arr->cap) {
        int32_t new_cap = arr->cap < GRAY_ARRAY_MIN_CAP ? GRAY_ARRAY_MIN_CAP : arr->cap * 2;
        if (new_cap < arr->cap) {
            fprintf(stderr, "EZ runtime: array capacity overflow\n");
            exit(1);
        }
        void *new_data = gray_arena_alloc(arena, (size_t)new_cap * (size_t)arr->elem_size);
        if (arr->data && arr->len > 0) {
            memcpy(new_data, arr->data, (size_t)arr->len * (size_t)arr->elem_size);
        }
        arr->data = new_data;
        arr->cap = new_cap;
    }
    memcpy((char *)arr->data + (size_t)arr->len * (size_t)arr->elem_size,
           value, (size_t)arr->elem_size);
    arr->len++;
}

EzArray gray_array_copy(EzArena *arena, EzArray *src) {
    return gray_array_from(arena, src->data, src->elem_size, src->len);
}
