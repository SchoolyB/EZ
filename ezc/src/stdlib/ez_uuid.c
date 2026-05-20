/*
 * ez_uuid.c - @uuid module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_uuid.h"
#include "ez_builtins.h"
#include <time.h>
#include <stdio.h>
#include <stdint.h>

#if defined(__APPLE__) || defined(__OpenBSD__) || defined(__FreeBSD__) || defined(__linux__)
#include <unistd.h>
#endif
#if defined(__APPLE__)
#include <sys/random.h>
#endif

#define EZ_UUID_LEN             36
#define EZ_UUID_COMPACT_LEN     32

/* Cryptographically-suitable random bytes. Prefers getentropy() (macOS,
 * BSDs, glibc 2.25+) and falls back to /dev/urandom. On failure, returns
 * false; callers should treat that as fatal since UUID uniqueness is the
 * whole point. */
static bool ez_uuid_random_bytes(uint8_t *buf, size_t n) {
#if defined(__APPLE__) || defined(__OpenBSD__) || defined(__FreeBSD__) || defined(__linux__)
    if (n <= 256 && getentropy(buf, n) == 0) return true;
#endif
    FILE *f = fopen("/dev/urandom", "rb");
    if (!f) return false;
    size_t got = fread(buf, 1, n, f);
    fclose(f);
    return got == n;
}

static void ez_uuid_format_hyphenated(const uint8_t *bytes, char *buf) {
    snprintf(buf, EZ_UUID_LEN + 1,
        "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
        bytes[0], bytes[1], bytes[2], bytes[3],
        bytes[4], bytes[5], bytes[6], bytes[7],
        bytes[8], bytes[9], bytes[10], bytes[11],
        bytes[12], bytes[13], bytes[14], bytes[15]);
}

EzString ez_uuid_generate(EzArena *arena) {
    uint8_t bytes[16];
    if (!ez_uuid_random_bytes(bytes, sizeof(bytes))) {
        return ez_string_lit("");
    }
    bytes[6] = (bytes[6] & 0x0F) | 0x40; /* version 4 */
    bytes[8] = (bytes[8] & 0x3F) | 0x80; /* variant 1 (RFC 4122) */
    char buf[EZ_UUID_LEN + 1];
    ez_uuid_format_hyphenated(bytes, buf);
    return ez_string_new(arena, buf, EZ_UUID_LEN);
}

EzString ez_uuid_generate_compact(EzArena *arena) {
    uint8_t bytes[16];
    if (!ez_uuid_random_bytes(bytes, sizeof(bytes))) {
        return ez_string_lit("");
    }
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

EzString ez_uuid_generate_random(EzArena *arena) {
    /* Alias for the canonical hyphenated v4 form. */
    return ez_uuid_generate(arena);
}

/* RFC 9562 §5.7 — UUID v7:
 *   bytes[0..5]   48-bit Unix timestamp (ms), big-endian
 *   bytes[6]      version 7 (high nibble) + 4 random bits
 *   bytes[7]      8 random bits
 *   bytes[8]      variant 10 (high 2 bits) + 6 random bits
 *   bytes[9..15]  56 random bits
 */
EzString ez_uuid_generate_time_ordered(EzArena *arena) {
    uint8_t bytes[16];
    if (!ez_uuid_random_bytes(bytes, sizeof(bytes))) {
        return ez_string_lit("");
    }

    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    uint64_t ms = (uint64_t)ts.tv_sec * 1000ULL + (uint64_t)(ts.tv_nsec / 1000000);

    bytes[0] = (uint8_t)(ms >> 40);
    bytes[1] = (uint8_t)(ms >> 32);
    bytes[2] = (uint8_t)(ms >> 24);
    bytes[3] = (uint8_t)(ms >> 16);
    bytes[4] = (uint8_t)(ms >> 8);
    bytes[5] = (uint8_t)(ms);
    bytes[6] = (bytes[6] & 0x0F) | 0x70; /* version 7 */
    bytes[8] = (bytes[8] & 0x3F) | 0x80; /* variant 1 */

    char buf[EZ_UUID_LEN + 1];
    ez_uuid_format_hyphenated(bytes, buf);
    return ez_string_new(arena, buf, EZ_UUID_LEN);
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

/* Strict parser: panics on invalid input. Callers that want a fallible
 * check should gate with ez_uuid_is_valid() first. Returns the input
 * normalized to lowercase. */
EzString ez_uuid_parse(EzArena *arena, EzString s) {
    if (!ez_uuid_is_valid(s)) {
        ez_builtin_panic_msg(ez_string_lit("uuid.parse: invalid UUID string"));
    }
    char *buf = (char *)ez_arena_alloc(arena, EZ_UUID_LEN + 1);
    for (int i = 0; i < EZ_UUID_LEN; i++) {
        char c = s.data[i];
        buf[i] = (c >= 'A' && c <= 'F') ? (char)(c - 'A' + 'a') : c;
    }
    buf[EZ_UUID_LEN] = '\0';
    EzString out;
    out.data = buf;
    out.len = EZ_UUID_LEN;
    return out;
}
