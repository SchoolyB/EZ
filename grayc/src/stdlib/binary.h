/*
 * binary.h — Public interface for the binary stdlib module.
 * Binary encoding/decoding for integers and floats in little-endian
 * and big-endian byte order.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_BINARY_H
#define GRAY_BINARY_H

#include "../runtime/runtime.h"
#include "../runtime/array.h"
#include "../runtime/bigint.h"

/* --- 8-bit --- */
GrayArray gray_binary_encode_i8(GrayArena *arena, int8_t val);
GrayArray gray_binary_encode_u8(GrayArena *arena, uint8_t val);
int8_t gray_binary_decode_i8(GrayArray *bytes);
uint8_t gray_binary_decode_u8(GrayArray *bytes);

/* --- 16-bit signed --- */
GrayArray gray_binary_encode_i16_le(GrayArena *arena, int16_t val);
GrayArray gray_binary_encode_i16_be(GrayArena *arena, int16_t val);
int16_t gray_binary_decode_i16_le(GrayArray *bytes);
int16_t gray_binary_decode_i16_be(GrayArray *bytes);

/* --- 16-bit unsigned --- */
GrayArray gray_binary_encode_u16_le(GrayArena *arena, uint16_t val);
GrayArray gray_binary_encode_u16_be(GrayArena *arena, uint16_t val);
uint16_t gray_binary_decode_u16_le(GrayArray *bytes);
uint16_t gray_binary_decode_u16_be(GrayArray *bytes);

/* --- 32-bit signed --- */
GrayArray gray_binary_encode_i32_le(GrayArena *arena, int32_t val);
GrayArray gray_binary_encode_i32_be(GrayArena *arena, int32_t val);
int32_t gray_binary_decode_i32_le(GrayArray *bytes);
int32_t gray_binary_decode_i32_be(GrayArray *bytes);

/* --- 32-bit unsigned --- */
GrayArray gray_binary_encode_u32_le(GrayArena *arena, uint32_t val);
GrayArray gray_binary_encode_u32_be(GrayArena *arena, uint32_t val);
uint32_t gray_binary_decode_u32_le(GrayArray *bytes);
uint32_t gray_binary_decode_u32_be(GrayArray *bytes);

/* --- 64-bit signed --- */
GrayArray gray_binary_encode_i64_le(GrayArena *arena, int64_t val);
GrayArray gray_binary_encode_i64_be(GrayArena *arena, int64_t val);
int64_t gray_binary_decode_i64_le(GrayArray *bytes);
int64_t gray_binary_decode_i64_be(GrayArray *bytes);

/* --- 64-bit unsigned --- */
GrayArray gray_binary_encode_u64_le(GrayArena *arena, uint64_t val);
GrayArray gray_binary_encode_u64_be(GrayArena *arena, uint64_t val);
uint64_t gray_binary_decode_u64_le(GrayArray *bytes);
uint64_t gray_binary_decode_u64_be(GrayArray *bytes);

/* --- 128-bit --- */
GrayArray gray_binary_encode_i128_le(GrayArena *arena, gray_i128 val);
GrayArray gray_binary_encode_i128_be(GrayArena *arena, gray_i128 val);
gray_i128 gray_binary_decode_i128_le(GrayArray *bytes);
gray_i128 gray_binary_decode_i128_be(GrayArray *bytes);
GrayArray gray_binary_encode_u128_le(GrayArena *arena, gray_u128 val);
GrayArray gray_binary_encode_u128_be(GrayArena *arena, gray_u128 val);
gray_u128 gray_binary_decode_u128_le(GrayArray *bytes);
gray_u128 gray_binary_decode_u128_be(GrayArray *bytes);

/* --- 256-bit --- */
GrayArray gray_binary_encode_i256_le(GrayArena *arena, gray_i256 val);
GrayArray gray_binary_encode_i256_be(GrayArena *arena, gray_i256 val);
gray_i256 gray_binary_decode_i256_le(GrayArray *bytes);
gray_i256 gray_binary_decode_i256_be(GrayArray *bytes);
GrayArray gray_binary_encode_u256_le(GrayArena *arena, gray_u256 val);
GrayArray gray_binary_encode_u256_be(GrayArena *arena, gray_u256 val);
gray_u256 gray_binary_decode_u256_le(GrayArray *bytes);
gray_u256 gray_binary_decode_u256_be(GrayArray *bytes);

/* --- Float encode/decode --- */
GrayArray gray_binary_encode_f32_le(GrayArena *arena, float val);
GrayArray gray_binary_encode_f32_be(GrayArena *arena, float val);
float gray_binary_decode_f32_le(GrayArray *bytes);
float gray_binary_decode_f32_be(GrayArray *bytes);
GrayArray gray_binary_encode_f64_le(GrayArena *arena, double val);
GrayArray gray_binary_encode_f64_be(GrayArena *arena, double val);
double gray_binary_decode_f64_le(GrayArray *bytes);
double gray_binary_decode_f64_be(GrayArray *bytes);

#endif
