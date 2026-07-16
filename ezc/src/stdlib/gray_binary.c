/*
 * gray_binary.c - @binary module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_binary.h"
#include <string.h>

static EzArray make_bytes(EzArena *arena, const void *data, int32_t size) {
    EzArray arr = gray_array_new(arena, sizeof(uint8_t), size);
    for (int32_t i = 0; i < size; i++) {
        uint8_t b = ((const uint8_t *)data)[i];
        gray_array_push(arena, &arr, &b);
    }
    return arr;
}

static EzArray make_bytes_reversed(EzArena *arena, const void *data, int32_t size) {
    EzArray arr = gray_array_new(arena, sizeof(uint8_t), size);
    for (int32_t i = size - 1; i >= 0; i--) {
        uint8_t b = ((const uint8_t *)data)[i];
        gray_array_push(arena, &arr, &b);
    }
    return arr;
}

/* --- 8-bit --- */
EzArray gray_binary_encode_i8(EzArena *arena, int8_t val) { return make_bytes(arena, &val, 1); }
EzArray gray_binary_encode_u8(EzArena *arena, uint8_t val) { return make_bytes(arena, &val, 1); }
int8_t gray_binary_decode_i8(EzArray *bytes) { return *(int8_t *)bytes->data; }
uint8_t gray_binary_decode_u8(EzArray *bytes) { return *(uint8_t *)bytes->data; }

/* --- 16-bit signed --- */
EzArray gray_binary_encode_i16_le(EzArena *arena, int16_t val) { return make_bytes(arena, &val, 2); }
EzArray gray_binary_encode_i16_be(EzArena *arena, int16_t val) { return make_bytes_reversed(arena, &val, 2); }
int16_t gray_binary_decode_i16_le(EzArray *bytes) { int16_t v; memcpy(&v, bytes->data, 2); return v; }
int16_t gray_binary_decode_i16_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return (int16_t)((d[0] << 8) | d[1]);
}

/* --- 16-bit unsigned --- */
EzArray gray_binary_encode_u16_le(EzArena *arena, uint16_t val) { return make_bytes(arena, &val, 2); }
EzArray gray_binary_encode_u16_be(EzArena *arena, uint16_t val) { return make_bytes_reversed(arena, &val, 2); }
uint16_t gray_binary_decode_u16_le(EzArray *bytes) { uint16_t v; memcpy(&v, bytes->data, 2); return v; }
uint16_t gray_binary_decode_u16_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return (uint16_t)((d[0] << 8) | d[1]);
}

/* --- 32-bit signed --- */
EzArray gray_binary_encode_i32_le(EzArena *arena, int32_t val) { return make_bytes(arena, &val, 4); }
EzArray gray_binary_encode_i32_be(EzArena *arena, int32_t val) { return make_bytes_reversed(arena, &val, 4); }
int32_t gray_binary_decode_i32_le(EzArray *bytes) { int32_t v; memcpy(&v, bytes->data, 4); return v; }
int32_t gray_binary_decode_i32_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return (int32_t)(((uint32_t)d[0] << 24) | ((uint32_t)d[1] << 16) | ((uint32_t)d[2] << 8) | d[3]);
}

/* --- 32-bit unsigned --- */
EzArray gray_binary_encode_u32_le(EzArena *arena, uint32_t val) { return make_bytes(arena, &val, 4); }
EzArray gray_binary_encode_u32_be(EzArena *arena, uint32_t val) { return make_bytes_reversed(arena, &val, 4); }
uint32_t gray_binary_decode_u32_le(EzArray *bytes) { uint32_t v; memcpy(&v, bytes->data, 4); return v; }
uint32_t gray_binary_decode_u32_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return ((uint32_t)d[0] << 24) | ((uint32_t)d[1] << 16) | ((uint32_t)d[2] << 8) | d[3];
}

/* --- 64-bit signed --- */
EzArray gray_binary_encode_i64_le(EzArena *arena, int64_t val) { return make_bytes(arena, &val, 8); }
EzArray gray_binary_encode_i64_be(EzArena *arena, int64_t val) { return make_bytes_reversed(arena, &val, 8); }
int64_t gray_binary_decode_i64_le(EzArray *bytes) { int64_t v; memcpy(&v, bytes->data, 8); return v; }
int64_t gray_binary_decode_i64_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    int64_t v = 0;
    for (int i = 0; i < 8; i++) v = (v << 8) | d[i];
    return v;
}

/* --- 64-bit unsigned --- */
EzArray gray_binary_encode_u64_le(EzArena *arena, uint64_t val) { return make_bytes(arena, &val, 8); }
EzArray gray_binary_encode_u64_be(EzArena *arena, uint64_t val) { return make_bytes_reversed(arena, &val, 8); }
uint64_t gray_binary_decode_u64_le(EzArray *bytes) { uint64_t v; memcpy(&v, bytes->data, 8); return v; }
uint64_t gray_binary_decode_u64_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    uint64_t v = 0;
    for (int i = 0; i < 8; i++) v = (v << 8) | d[i];
    return v;
}

/* --- 128-bit --- */
EzArray gray_binary_encode_i128_le(EzArena *arena, gray_i128 val) { return make_bytes(arena, &val, 16); }
EzArray gray_binary_encode_i128_be(EzArena *arena, gray_i128 val) { return make_bytes_reversed(arena, &val, 16); }
gray_i128 gray_binary_decode_i128_le(EzArray *bytes) { gray_i128 v; memcpy(&v, bytes->data, 16); return v; }
gray_i128 gray_binary_decode_i128_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    gray_i128 v;
    uint64_t hi = 0, lo = 0;
    for (int i = 0; i < 8; i++) hi = (hi << 8) | d[i];
    for (int i = 8; i < 16; i++) lo = (lo << 8) | d[i];
    v.hi = (int64_t)hi;
    v.lo = lo;
    return v;
}

EzArray gray_binary_encode_u128_le(EzArena *arena, gray_u128 val) { return make_bytes(arena, &val, 16); }
EzArray gray_binary_encode_u128_be(EzArena *arena, gray_u128 val) { return make_bytes_reversed(arena, &val, 16); }
gray_u128 gray_binary_decode_u128_le(EzArray *bytes) { gray_u128 v; memcpy(&v, bytes->data, 16); return v; }
gray_u128 gray_binary_decode_u128_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    gray_u128 v;
    uint64_t hi = 0, lo = 0;
    for (int i = 0; i < 8; i++) hi = (hi << 8) | d[i];
    for (int i = 8; i < 16; i++) lo = (lo << 8) | d[i];
    v.hi = hi;
    v.lo = lo;
    return v;
}

/* --- 256-bit --- */
EzArray gray_binary_encode_i256_le(EzArena *arena, gray_i256 val) { return make_bytes(arena, &val, 32); }
EzArray gray_binary_encode_i256_be(EzArena *arena, gray_i256 val) { return make_bytes_reversed(arena, &val, 32); }
gray_i256 gray_binary_decode_i256_le(EzArray *bytes) { gray_i256 v; memcpy(&v, bytes->data, 32); return v; }
gray_i256 gray_binary_decode_i256_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    gray_i256 v;
    for (int w = 3; w >= 0; w--) {
        uint64_t limb = 0;
        for (int i = 0; i < 8; i++) limb = (limb << 8) | d[(3 - w) * 8 + i];
        v.w[w] = limb;
    }
    return v;
}

EzArray gray_binary_encode_u256_le(EzArena *arena, gray_u256 val) { return make_bytes(arena, &val, 32); }
EzArray gray_binary_encode_u256_be(EzArena *arena, gray_u256 val) { return make_bytes_reversed(arena, &val, 32); }
gray_u256 gray_binary_decode_u256_le(EzArray *bytes) { gray_u256 v; memcpy(&v, bytes->data, 32); return v; }
gray_u256 gray_binary_decode_u256_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    gray_u256 v;
    for (int w = 3; w >= 0; w--) {
        uint64_t limb = 0;
        for (int i = 0; i < 8; i++) limb = (limb << 8) | d[(3 - w) * 8 + i];
        v.w[w] = limb;
    }
    return v;
}

/* --- Float --- */
EzArray gray_binary_encode_f32_le(EzArena *arena, float val) { return make_bytes(arena, &val, 4); }
EzArray gray_binary_encode_f32_be(EzArena *arena, float val) { return make_bytes_reversed(arena, &val, 4); }
float gray_binary_decode_f32_le(EzArray *bytes) { float v; memcpy(&v, bytes->data, 4); return v; }
float gray_binary_decode_f32_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    uint8_t reversed[4] = {d[3], d[2], d[1], d[0]};
    float v; memcpy(&v, reversed, 4); return v;
}

EzArray gray_binary_encode_f64_le(EzArena *arena, double val) { return make_bytes(arena, &val, 8); }
EzArray gray_binary_encode_f64_be(EzArena *arena, double val) { return make_bytes_reversed(arena, &val, 8); }
double gray_binary_decode_f64_le(EzArray *bytes) { double v; memcpy(&v, bytes->data, 8); return v; }
double gray_binary_decode_f64_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    uint8_t reversed[8] = {d[7], d[6], d[5], d[4], d[3], d[2], d[1], d[0]};
    double v; memcpy(&v, reversed, 8); return v;
}
