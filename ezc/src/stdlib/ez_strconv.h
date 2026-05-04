/*
 * ez_strconv.h - strconv module for EZ (string/type conversions)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STRCONV_H
#define EZ_STRCONV_H

#include "../runtime/ez_runtime.h"

/* Result types for fallible conversions */
#ifndef EZRESULT_BOOL_DEFINED
#define EZRESULT_BOOL_DEFINED
typedef struct { bool v0; EzError *v1; } EzResult_bool;
#endif
typedef struct { uint64_t v0; EzError *v1; } EzResult_uint;
typedef struct { double v0; EzError *v1; } EzResult_float;

/* Panicking conversions (single-var assignment) */
int64_t ez_strconv_to_int(EzString s, int base);
uint64_t ez_strconv_to_uint(EzString s, int base);
double ez_strconv_to_float(EzString s);
bool ez_strconv_to_bool(EzString s);

/* Fallible conversions (multi-var destructuring) */
EzResult_int ez_strconv_to_int_result(EzString s, int base);
EzResult_uint ez_strconv_to_uint_result(EzString s, int base);
EzResult_float ez_strconv_to_float_result(EzString s);
EzResult_bool ez_strconv_to_bool_result(EzString s);

/* Type to string conversions */
EzString ez_strconv_from_int(EzArena *arena, int64_t n);
EzString ez_strconv_from_uint(EzArena *arena, uint64_t n);
EzString ez_strconv_from_float(EzArena *arena, double f);
EzString ez_strconv_from_bool(bool b);

/* Query functions */
bool ez_strconv_is_numeric(EzString s);
bool ez_strconv_is_integer(EzString s);

#endif
