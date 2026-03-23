/*
 * ez_encoding.c - @encoding module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_encoding.h"
#include <string.h>
#include <stdio.h>
#include <ctype.h>

static const char b64_table[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

EzString ez_encoding_base64_encode(EzArena *arena, EzString s) {
    int32_t out_len = ((s.len + 2) / 3) * 4;
    char *out = ez_arena_alloc(arena, (size_t)out_len + 1);
    int j = 0;
    for (int i = 0; i < s.len; i += 3) {
        uint32_t a = (uint8_t)s.data[i];
        uint32_t b = (i + 1 < s.len) ? (uint8_t)s.data[i + 1] : 0;
        uint32_t c = (i + 2 < s.len) ? (uint8_t)s.data[i + 2] : 0;
        uint32_t triple = (a << 16) | (b << 8) | c;
        out[j++] = b64_table[(triple >> 18) & 0x3F];
        out[j++] = b64_table[(triple >> 12) & 0x3F];
        out[j++] = (i + 1 < s.len) ? b64_table[(triple >> 6) & 0x3F] : '=';
        out[j++] = (i + 2 < s.len) ? b64_table[triple & 0x3F] : '=';
    }
    out[j] = '\0';
    EzString r = { out, (int32_t)j };
    return r;
}

static int b64_val(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '+') return 62;
    if (c == '/') return 63;
    return -1;
}

EzString ez_encoding_base64_decode(EzArena *arena, EzString s) {
    int32_t out_len = (s.len / 4) * 3;
    char *out = ez_arena_alloc(arena, (size_t)out_len + 1);
    int j = 0;
    for (int i = 0; i < s.len; i += 4) {
        int a = b64_val(s.data[i]);
        int b = b64_val(s.data[i + 1]);
        int c = (s.data[i + 2] != '=') ? b64_val(s.data[i + 2]) : 0;
        int d = (s.data[i + 3] != '=') ? b64_val(s.data[i + 3]) : 0;
        uint32_t triple = ((uint32_t)a << 18) | ((uint32_t)b << 12) | ((uint32_t)c << 6) | (uint32_t)d;
        out[j++] = (char)((triple >> 16) & 0xFF);
        if (s.data[i + 2] != '=') out[j++] = (char)((triple >> 8) & 0xFF);
        if (s.data[i + 3] != '=') out[j++] = (char)(triple & 0xFF);
    }
    out[j] = '\0';
    EzString r = { out, (int32_t)j };
    return r;
}

EzString ez_encoding_hex_encode(EzArena *arena, EzString s) {
    int32_t out_len = s.len * 2;
    char *out = ez_arena_alloc(arena, (size_t)out_len + 1);
    for (int i = 0; i < s.len; i++) {
        snprintf(out + i * 2, 3, "%02x", (uint8_t)s.data[i]);
    }
    out[out_len] = '\0';
    EzString r = { out, out_len };
    return r;
}

EzString ez_encoding_hex_decode(EzArena *arena, EzString s) {
    int32_t out_len = s.len / 2;
    char *out = ez_arena_alloc(arena, (size_t)out_len + 1);
    for (int i = 0; i < out_len; i++) {
        unsigned int byte;
        sscanf(s.data + i * 2, "%02x", &byte);
        out[i] = (char)byte;
    }
    out[out_len] = '\0';
    EzString r = { out, out_len };
    return r;
}

EzString ez_encoding_url_encode(EzArena *arena, EzString s) {
    /* Worst case: every char becomes %XX (3x) */
    char *out = ez_arena_alloc(arena, (size_t)s.len * 3 + 1);
    int j = 0;
    for (int i = 0; i < s.len; i++) {
        char c = s.data[i];
        if (isalnum(c) || c == '-' || c == '_' || c == '.' || c == '~') {
            out[j++] = c;
        } else {
            j += snprintf(out + j, 4, "%%%02X", (uint8_t)c);
        }
    }
    out[j] = '\0';
    EzString r = { out, (int32_t)j };
    return r;
}

EzString ez_encoding_url_decode(EzArena *arena, EzString s) {
    char *out = ez_arena_alloc(arena, (size_t)s.len + 1);
    int j = 0;
    for (int i = 0; i < s.len; i++) {
        if (s.data[i] == '%' && i + 2 < s.len) {
            unsigned int byte;
            sscanf(s.data + i + 1, "%02x", &byte);
            out[j++] = (char)byte;
            i += 2;
        } else if (s.data[i] == '+') {
            out[j++] = ' ';
        } else {
            out[j++] = s.data[i];
        }
    }
    out[j] = '\0';
    EzString r = { out, (int32_t)j };
    return r;
}
