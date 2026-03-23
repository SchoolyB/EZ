/*
 * ez_arrays.h - @arrays module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_ARRAYS_H
#define EZ_ARRAYS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* Modification */
void ez_arrays_append(EzArena *arena, EzArray *arr, const void *value);
void ez_arrays_insert(EzArena *arena, EzArray *arr, int32_t index, const void *value);
void ez_arrays_remove_at(EzArray *arr, int32_t index);
void ez_arrays_clear(EzArray *arr);

/* Query */
bool ez_arrays_is_empty(EzArray *arr);
bool ez_arrays_contains_int(EzArray *arr, int64_t value);
bool ez_arrays_contains_str(EzArray *arr, EzString value);
int64_t ez_arrays_index_of_int(EzArray *arr, int64_t value);
int64_t ez_arrays_index_of_str(EzArray *arr, EzString value);

/* Transformation */
EzArray ez_arrays_reverse(EzArena *arena, EzArray *arr);
EzArray ez_arrays_slice(EzArena *arena, EzArray *arr, int32_t start, int32_t end);
EzArray ez_arrays_concat(EzArena *arena, EzArray *a, EzArray *b);

/* Computation */
int64_t ez_arrays_sum(EzArray *arr);
int64_t ez_arrays_min(EzArray *arr);
int64_t ez_arrays_max(EzArray *arr);

/* Sort */
void ez_arrays_sort(EzArray *arr);
void ez_arrays_sort_desc(EzArray *arr);

#endif
