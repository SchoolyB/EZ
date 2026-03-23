/*
 * ez_bytes.c - @bytes module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_bytes.h"
#include "ez_encoding.h"
#include <string.h>
#include <stdio.h>

EzArray ez_bytes_from_string(EzArena *arena, EzString s) {
    EzArray arr = ez_array_new(arena, sizeof(uint8_t), s.len);
    for (int32_t i = 0; i < s.len; i++) {
        uint8_t b = (uint8_t)s.data[i];
        ez_array_push(arena, &arr, &b);
    }
    return arr;
}

EzString ez_bytes_to_string(EzArena *arena, EzArray *bytes) {
    return ez_string_new(arena, (const char *)bytes->data, bytes->len);
}

EzArray ez_bytes_from_hex(EzArena *arena, EzString hex) {
    int32_t out_len = hex.len / 2;
    EzArray arr = ez_array_new(arena, sizeof(uint8_t), out_len);
    for (int32_t i = 0; i < out_len; i++) {
        unsigned int byte;
        sscanf(hex.data + i * 2, "%02x", &byte);
        uint8_t b = (uint8_t)byte;
        ez_array_push(arena, &arr, &b);
    }
    return arr;
}

EzString ez_bytes_to_hex(EzArena *arena, EzArray *bytes) {
    int32_t out_len = bytes->len * 2;
    char *hex = ez_arena_alloc(arena, (size_t)out_len + 1);
    uint8_t *data = (uint8_t *)bytes->data;
    for (int32_t i = 0; i < bytes->len; i++) {
        snprintf(hex + i * 2, 3, "%02x", data[i]);
    }
    hex[out_len] = '\0';
    EzString r = { hex, out_len };
    return r;
}

EzArray ez_bytes_from_base64(EzArena *arena, EzString b64) {
    EzString decoded = ez_encoding_base64_decode(arena, b64);
    return ez_bytes_from_string(arena, decoded);
}

EzString ez_bytes_to_base64(EzArena *arena, EzArray *bytes) {
    EzString s = ez_bytes_to_string(arena, bytes);
    return ez_encoding_base64_encode(arena, s);
}
