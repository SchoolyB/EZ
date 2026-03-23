/*
 * ez_binary.c - @binary module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_binary.h"
#include <string.h>

static EzArray make_bytes(EzArena *arena, const void *data, int32_t size) {
    EzArray arr = ez_array_new(arena, sizeof(uint8_t), size);
    for (int32_t i = 0; i < size; i++) {
        uint8_t b = ((const uint8_t *)data)[i];
        ez_array_push(arena, &arr, &b);
    }
    return arr;
}

static EzArray make_bytes_reversed(EzArena *arena, const void *data, int32_t size) {
    EzArray arr = ez_array_new(arena, sizeof(uint8_t), size);
    for (int32_t i = size - 1; i >= 0; i--) {
        uint8_t b = ((const uint8_t *)data)[i];
        ez_array_push(arena, &arr, &b);
    }
    return arr;
}

EzArray ez_binary_encode_u8(EzArena *arena, uint8_t val) { return make_bytes(arena, &val, 1); }
EzArray ez_binary_encode_i16_le(EzArena *arena, int16_t val) { return make_bytes(arena, &val, 2); }
EzArray ez_binary_encode_i16_be(EzArena *arena, int16_t val) { return make_bytes_reversed(arena, &val, 2); }
EzArray ez_binary_encode_i32_le(EzArena *arena, int32_t val) { return make_bytes(arena, &val, 4); }
EzArray ez_binary_encode_i32_be(EzArena *arena, int32_t val) { return make_bytes_reversed(arena, &val, 4); }
EzArray ez_binary_encode_i64_le(EzArena *arena, int64_t val) { return make_bytes(arena, &val, 8); }
EzArray ez_binary_encode_i64_be(EzArena *arena, int64_t val) { return make_bytes_reversed(arena, &val, 8); }
EzArray ez_binary_encode_f32_le(EzArena *arena, float val) { return make_bytes(arena, &val, 4); }
EzArray ez_binary_encode_f64_le(EzArena *arena, double val) { return make_bytes(arena, &val, 8); }

uint8_t ez_binary_decode_u8(EzArray *bytes) {
    return *(uint8_t *)bytes->data;
}

int16_t ez_binary_decode_i16_le(EzArray *bytes) { int16_t v; memcpy(&v, bytes->data, 2); return v; }
int16_t ez_binary_decode_i16_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return (int16_t)((d[0] << 8) | d[1]);
}
int32_t ez_binary_decode_i32_le(EzArray *bytes) { int32_t v; memcpy(&v, bytes->data, 4); return v; }
int32_t ez_binary_decode_i32_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    return (int32_t)((d[0] << 24) | (d[1] << 16) | (d[2] << 8) | d[3]);
}
int64_t ez_binary_decode_i64_le(EzArray *bytes) { int64_t v; memcpy(&v, bytes->data, 8); return v; }
int64_t ez_binary_decode_i64_be(EzArray *bytes) {
    uint8_t *d = (uint8_t *)bytes->data;
    int64_t v = 0;
    for (int i = 0; i < 8; i++) v = (v << 8) | d[i];
    return v;
}
float ez_binary_decode_f32_le(EzArray *bytes) { float v; memcpy(&v, bytes->data, 4); return v; }
double ez_binary_decode_f64_le(EzArray *bytes) { double v; memcpy(&v, bytes->data, 8); return v; }
