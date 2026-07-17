/*
 * gray_bytes.c — Implementation of the bytes stdlib module.
 * Converts between strings, byte arrays, hex, and base64
 * representations for raw byte manipulation.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_bytes.h"
#include "gray_encoding.h"
#include <string.h>
#include <stdio.h>

GrayArray gray_bytes_from_string(GrayArena *arena, GrayString s) {
    GrayArray arr = gray_array_new(arena, sizeof(uint8_t), s.len);
    for (int32_t i = 0; i < s.len; i++) {
        uint8_t b = (uint8_t)s.data[i];
        gray_array_push(arena, &arr, &b);
    }
    return arr;
}

GrayString gray_bytes_to_string(GrayArena *arena, GrayArray *bytes) {
    return gray_string_new(arena, (const char *)bytes->data, bytes->len);
}

GrayArray gray_bytes_from_hex(GrayArena *arena, GrayString hex) {
    int32_t out_len = hex.len / 2;
    GrayArray arr = gray_array_new(arena, sizeof(uint8_t), out_len);
    for (int32_t i = 0; i < out_len; i++) {
        unsigned int byte;
        sscanf(hex.data + i * 2, "%02x", &byte);
        uint8_t b = (uint8_t)byte;
        gray_array_push(arena, &arr, &b);
    }
    return arr;
}

GrayString gray_bytes_to_hex(GrayArena *arena, GrayArray *bytes) {
    int32_t out_len = bytes->len * 2;
    char *hex = gray_arena_alloc(arena, (size_t)out_len + 1);
    uint8_t *data = (uint8_t *)bytes->data;
    for (int32_t i = 0; i < bytes->len; i++) {
        snprintf(hex + i * 2, 3, "%02x", data[i]);
    }
    hex[out_len] = '\0';
    GrayString r = { hex, out_len };
    return r;
}

GrayArray gray_bytes_from_base64(GrayArena *arena, GrayString b64) {
    GrayString decoded = gray_encoding_base64_decode(arena, b64);
    return gray_bytes_from_string(arena, decoded);
}

GrayString gray_bytes_to_base64(GrayArena *arena, GrayArray *bytes) {
    GrayString s = gray_bytes_to_string(arena, bytes);
    return gray_encoding_base64_encode(arena, s);
}
