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
#include "ez_bigint.h"

/* --- 8-bit --- */
EzArray ez_binary_encode_i8(EzArena *arena, int8_t val);
EzArray ez_binary_encode_u8(EzArena *arena, uint8_t val);
int8_t ez_binary_decode_i8(EzArray *bytes);
uint8_t ez_binary_decode_u8(EzArray *bytes);

/* --- 16-bit signed --- */
EzArray ez_binary_encode_i16_le(EzArena *arena, int16_t val);
EzArray ez_binary_encode_i16_be(EzArena *arena, int16_t val);
int16_t ez_binary_decode_i16_le(EzArray *bytes);
int16_t ez_binary_decode_i16_be(EzArray *bytes);

/* --- 16-bit unsigned --- */
EzArray ez_binary_encode_u16_le(EzArena *arena, uint16_t val);
EzArray ez_binary_encode_u16_be(EzArena *arena, uint16_t val);
uint16_t ez_binary_decode_u16_le(EzArray *bytes);
uint16_t ez_binary_decode_u16_be(EzArray *bytes);

/* --- 32-bit signed --- */
EzArray ez_binary_encode_i32_le(EzArena *arena, int32_t val);
EzArray ez_binary_encode_i32_be(EzArena *arena, int32_t val);
int32_t ez_binary_decode_i32_le(EzArray *bytes);
int32_t ez_binary_decode_i32_be(EzArray *bytes);

/* --- 32-bit unsigned --- */
EzArray ez_binary_encode_u32_le(EzArena *arena, uint32_t val);
EzArray ez_binary_encode_u32_be(EzArena *arena, uint32_t val);
uint32_t ez_binary_decode_u32_le(EzArray *bytes);
uint32_t ez_binary_decode_u32_be(EzArray *bytes);

/* --- 64-bit signed --- */
EzArray ez_binary_encode_i64_le(EzArena *arena, int64_t val);
EzArray ez_binary_encode_i64_be(EzArena *arena, int64_t val);
int64_t ez_binary_decode_i64_le(EzArray *bytes);
int64_t ez_binary_decode_i64_be(EzArray *bytes);

/* --- 64-bit unsigned --- */
EzArray ez_binary_encode_u64_le(EzArena *arena, uint64_t val);
EzArray ez_binary_encode_u64_be(EzArena *arena, uint64_t val);
uint64_t ez_binary_decode_u64_le(EzArray *bytes);
uint64_t ez_binary_decode_u64_be(EzArray *bytes);

/* --- 128-bit --- */
EzArray ez_binary_encode_i128_le(EzArena *arena, ez_i128 val);
EzArray ez_binary_encode_i128_be(EzArena *arena, ez_i128 val);
ez_i128 ez_binary_decode_i128_le(EzArray *bytes);
ez_i128 ez_binary_decode_i128_be(EzArray *bytes);
EzArray ez_binary_encode_u128_le(EzArena *arena, ez_u128 val);
EzArray ez_binary_encode_u128_be(EzArena *arena, ez_u128 val);
ez_u128 ez_binary_decode_u128_le(EzArray *bytes);
ez_u128 ez_binary_decode_u128_be(EzArray *bytes);

/* --- 256-bit --- */
EzArray ez_binary_encode_i256_le(EzArena *arena, ez_i256 val);
EzArray ez_binary_encode_i256_be(EzArena *arena, ez_i256 val);
ez_i256 ez_binary_decode_i256_le(EzArray *bytes);
ez_i256 ez_binary_decode_i256_be(EzArray *bytes);
EzArray ez_binary_encode_u256_le(EzArena *arena, ez_u256 val);
EzArray ez_binary_encode_u256_be(EzArena *arena, ez_u256 val);
ez_u256 ez_binary_decode_u256_le(EzArray *bytes);
ez_u256 ez_binary_decode_u256_be(EzArray *bytes);

/* --- Float encode/decode --- */
EzArray ez_binary_encode_f32_le(EzArena *arena, float val);
EzArray ez_binary_encode_f32_be(EzArena *arena, float val);
float ez_binary_decode_f32_le(EzArray *bytes);
float ez_binary_decode_f32_be(EzArray *bytes);
EzArray ez_binary_encode_f64_le(EzArena *arena, double val);
EzArray ez_binary_encode_f64_be(EzArena *arena, double val);
double ez_binary_decode_f64_le(EzArray *bytes);
double ez_binary_decode_f64_be(EzArray *bytes);

#endif
