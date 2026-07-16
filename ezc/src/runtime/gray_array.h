/*
 * gray_array.h - Dynamic array type for EZ
 *
 * EzArray is a fat pointer: data + len + cap + elem_size.
 * All backing storage is allocated from an arena.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_ARRAY_H
#define GRAY_ARRAY_H

#include "gray_runtime.h"

#define GRAY_ARRAY_MIN_CAP            4
#define GRAY_MAX_INLINE_ELEM_SIZE     64

typedef struct {
    void *data;
    int32_t len;
    int32_t cap;
    int32_t elem_size;
    int32_t iterating;          /* >0 while a for_each is active */
} EzArray;

/* Create an empty array with given element size and initial capacity */
EzArray gray_array_new(EzArena *arena, int32_t elem_size, int32_t initial_cap);

/* Create an array from a C literal (copies data into arena) */
EzArray gray_array_from(EzArena *arena, const void *data, int32_t elem_size, int32_t count);

/* Get a pointer to element at index (with bounds checking) */
void *gray_array_get_ptr(EzArray *arr, int32_t index, const char *file, int line);

/* Set element at index (with bounds checking) */
void gray_array_set(EzArray *arr, int32_t index, const void *value, const char *file, int line);

/* Append an element (may reallocate on the arena) */
void gray_array_push(EzArena *arena, EzArray *arr, const void *value);

/* Deep copy an array */
EzArray gray_array_copy(EzArena *arena, EzArray *src);

/* Typed access macros */
#define GRAY_ARRAY_GET(arr, type, i) (*(type *)gray_array_get_ptr(&(arr), (i), __FILE__, __LINE__))
#define GRAY_ARRAY_SET(arr, type, i, val) do { type _v = (val); gray_array_set(&(arr), (i), &_v, __FILE__, __LINE__); } while(0)

/* Create from typed literal — helper macros */
#define GRAY_ARRAY_FROM_I64(arena, ...) \
    gray_array_from((arena), (int64_t[]){__VA_ARGS__}, sizeof(int64_t), \
        sizeof((int64_t[]){__VA_ARGS__}) / sizeof(int64_t))

#define GRAY_ARRAY_FROM_F64(arena, ...) \
    gray_array_from((arena), (double[]){__VA_ARGS__}, sizeof(double), \
        sizeof((double[]){__VA_ARGS__}) / sizeof(double))

#define GRAY_ARRAY_FROM_BOOL(arena, ...) \
    gray_array_from((arena), (bool[]){__VA_ARGS__}, sizeof(bool), \
        sizeof((bool[]){__VA_ARGS__}) / sizeof(bool))

#define GRAY_ARRAY_FROM_STR(arena, ...) \
    gray_array_from((arena), (EzString[]){__VA_ARGS__}, sizeof(EzString), \
        sizeof((EzString[]){__VA_ARGS__}) / sizeof(EzString))

#endif
