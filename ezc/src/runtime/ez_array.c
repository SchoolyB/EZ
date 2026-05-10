/*
 * ez_array.c - Dynamic array implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_array.h"
#include <string.h>

EzArray ez_array_new(EzArena *arena, int32_t elem_size, int32_t initial_cap) {
    EzArray arr;
    arr.elem_size = elem_size;
    arr.len = 0;
    arr.cap = initial_cap > 0 ? initial_cap : EZ_ARRAY_MIN_CAP;
    arr.iterating = 0;
    arr.data = ez_arena_alloc(arena, (size_t)arr.cap * (size_t)arr.elem_size);
    return arr;
}

EzArray ez_array_from(EzArena *arena, const void *data, int32_t elem_size, int32_t count) {
    EzArray arr;
    arr.elem_size = elem_size;
    arr.len = count;
    arr.cap = count > 0 ? count : EZ_ARRAY_MIN_CAP;
    arr.iterating = 0;
    arr.data = ez_arena_alloc(arena, (size_t)arr.cap * (size_t)arr.elem_size);
    if (count > 0 && data) {
        memcpy(arr.data, data, (size_t)count * (size_t)elem_size);
    }
    return arr;
}

void *ez_array_get_ptr(EzArray *arr, int32_t index, const char *file, int line) {
    if (index < 0 || index >= arr->len) {
        ez_panic(file, line, "array index out of bounds — tried to access index %d but the array only has %d elements", index, arr->len);
    }
    return (char *)arr->data + (size_t)index * (size_t)arr->elem_size;
}

void ez_array_set(EzArray *arr, int32_t index, const void *value, const char *file, int line) {
    if (arr->iterating > 0)
        ez_panic(file, line, "cannot modify array during for_each iteration");
    if (index < 0 || index >= arr->len) {
        ez_panic(file, line, "array index out of bounds — tried to access index %d but the array only has %d elements", index, arr->len);
    }
    memcpy((char *)arr->data + (size_t)index * (size_t)arr->elem_size,
           value, (size_t)arr->elem_size);
}

void ez_array_push(EzArena *arena, EzArray *arr, const void *value) {
    if (arr->iterating > 0)
        ez_panic(__FILE__, __LINE__, "cannot modify array during for_each iteration");
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
    memcpy((char *)arr->data + (size_t)arr->len * (size_t)arr->elem_size,
           value, (size_t)arr->elem_size);
    arr->len++;
}

EzArray ez_array_copy(EzArena *arena, EzArray *src) {
    return ez_array_from(arena, src->data, src->elem_size, src->len);
}
