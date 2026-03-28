/*
 * ez_random.h - @random module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_RANDOM_H
#define EZ_RANDOM_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

double ez_random_float_range(double min, double max);
double ez_random_float_unit(void);
int64_t ez_random_int_range(int64_t min, int64_t max);
int64_t ez_random_int_max(int64_t max);
bool ez_random_bool(void);
uint8_t ez_random_byte(void);
int32_t ez_random_char(void);
int32_t ez_random_char_range(int32_t min, int32_t max);

/* Array operations */
EzArray ez_random_shuffle(EzArena *arena, EzArray *arr);
EzArray ez_random_sample(EzArena *arena, EzArray *arr, int32_t n);

/* Random hex string */
EzString ez_random_hex(EzArena *arena, int64_t length);

#endif
