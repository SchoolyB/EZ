/*
 * ez_uuid.c - @uuid module implementation (UUID v4)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_uuid.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <stdio.h>

#define EZ_UUID_LEN             36
#define EZ_UUID_COMPACT_LEN     32

static bool _seeded = false;

EzString ez_uuid_generate(EzArena *arena) {
    if (!_seeded) { srand((unsigned)time(NULL)); _seeded = true; }
    uint8_t bytes[16];
    for (int i = 0; i < 16; i++) bytes[i] = (uint8_t)(rand() % 256);
    bytes[6] = (bytes[6] & 0x0F) | 0x40; /* version 4 */
    bytes[8] = (bytes[8] & 0x3F) | 0x80; /* variant 1 */
    char buf[EZ_UUID_LEN + 1];
    snprintf(buf, sizeof(buf),
        "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5], bytes[6], bytes[7],
        bytes[8], bytes[9], bytes[10], bytes[11],
        bytes[12], bytes[13], bytes[14], bytes[15]);
    return ez_string_new(arena, buf, EZ_UUID_LEN);
}

EzString ez_uuid_generate_compact(EzArena *arena) {
    if (!_seeded) { srand((unsigned)time(NULL)); _seeded = true; }
    uint8_t bytes[16];
    for (int i = 0; i < 16; i++) bytes[i] = (uint8_t)(rand() % 256);
    bytes[6] = (bytes[6] & 0x0F) | 0x40;
    bytes[8] = (bytes[8] & 0x3F) | 0x80;
    char buf[EZ_UUID_COMPACT_LEN + 1];
    snprintf(buf, sizeof(buf),
        "%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5], bytes[6], bytes[7],
        bytes[8], bytes[9], bytes[10], bytes[11],
        bytes[12], bytes[13], bytes[14], bytes[15]);
    return ez_string_new(arena, buf, EZ_UUID_COMPACT_LEN);
}

bool ez_uuid_is_valid(EzString s) {
    if (s.len != EZ_UUID_LEN) return false;
    for (int i = 0; i < EZ_UUID_LEN; i++) {
        if (i == 8 || i == 13 || i == 18 || i == 23) {
            if (s.data[i] != '-') return false;
        } else {
            char c = s.data[i];
            if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')))
                return false;
        }
    }
    return true;
}
