/*
 * ez_binary.h - @binary module for EZC
 *
 * Binary encoding/decoding for integers and floats
 * in little-endian and big-endian byte order.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_BINARY_H
#define EZ_BINARY_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* Encode integers to byte arrays */
EzArray ez_binary_encode_u8(EzArena *arena, uint8_t val);
EzArray ez_binary_encode_i16_le(EzArena *arena, int16_t val);
EzArray ez_binary_encode_i16_be(EzArena *arena, int16_t val);
EzArray ez_binary_encode_i32_le(EzArena *arena, int32_t val);
EzArray ez_binary_encode_i32_be(EzArena *arena, int32_t val);
EzArray ez_binary_encode_i64_le(EzArena *arena, int64_t val);
EzArray ez_binary_encode_i64_be(EzArena *arena, int64_t val);

/* Decode byte arrays to integers */
uint8_t ez_binary_decode_u8(EzArray *bytes);
int16_t ez_binary_decode_i16_le(EzArray *bytes);
int16_t ez_binary_decode_i16_be(EzArray *bytes);
int32_t ez_binary_decode_i32_le(EzArray *bytes);
int32_t ez_binary_decode_i32_be(EzArray *bytes);
int64_t ez_binary_decode_i64_le(EzArray *bytes);
int64_t ez_binary_decode_i64_be(EzArray *bytes);

/* Float encode/decode */
EzArray ez_binary_encode_f32_le(EzArena *arena, float val);
EzArray ez_binary_encode_f64_le(EzArena *arena, double val);
float ez_binary_decode_f32_le(EzArray *bytes);
double ez_binary_decode_f64_le(EzArray *bytes);

#endif
