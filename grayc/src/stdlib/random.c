/*
 * random.c — Implementation of the random stdlib module.
 * Provides pseudo-random number generation for floating-point numbers and integers,
 * array shuffling, random element selection, and manual seeding.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

/* Must precede every <stdlib.h> inclusion (also transitive ones via random.h)
 * so the CRT declares rand_s, the Windows entropy source. */
#ifdef _WIN32
#define _CRT_RAND_S
#include <stdlib.h>
#endif

#include "random.h"
#include "builtins.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#if defined(_WIN32)
#include <process.h>
#else
#include <unistd.h>
#endif

/* arc4random_buf is hidden by _POSIX_C_SOURCE on Apple/BSD — declare explicitly */
#if defined(__APPLE__) || defined(__FreeBSD__)
void arc4random_buf(void *buf, size_t nbytes);
#endif

static bool _seeded = false;
static bool _user_seeded = false;
static void ensure_seed(void) {
    if (!_seeded) {
        unsigned seed;
#if defined(__APPLE__) || defined(__FreeBSD__)
        arc4random_buf(&seed, sizeof(seed));
#elif defined(_WIN32)
        /* rand_s is RtlGenRandom under the hood: CSPRNG, no extra link lib. */
        if (rand_s(&seed) != 0) seed = (unsigned)time(NULL) ^ (unsigned)_getpid();
#else
        FILE *urandom = fopen("/dev/urandom", "rb");
        if (urandom) { fread(&seed, sizeof(seed), 1, urandom); fclose(urandom); }
        else { seed = (unsigned)time(NULL) ^ (unsigned)getpid(); }
#endif
        srand(seed);
        _seeded = true;
    }
}

void gray_random_seed(int64_t value) {
    srand((unsigned)value);
    _seeded = true;
    _user_seeded = true;
}

double gray_random_f64_unit(void) {
    ensure_seed();
    return (double)rand() / RAND_MAX;
}

double gray_random_f64_range(double minimum, double maximum) {
    if (!(minimum < maximum))
        gray_panic_code("P0134", "random.rand_f64: range [%g, %g) is empty; min must be less than max", minimum, maximum);
    return minimum + gray_random_f64_unit() * (maximum - minimum);
}

/* Generate a uniform random uint64_t across the full 64-bit range.
 * When the user has explicitly seeded via random.seed(), use srand/rand
 * so the sequence is deterministic.  Otherwise use OS entropy. */
static uint64_t rand64(void) {
    if (_user_seeded) {
        return ((uint64_t)(unsigned)rand() << 33) ^
               ((uint64_t)(unsigned)rand() << 2)  ^
               ((uint64_t)(unsigned)rand());
    }
    uint64_t bits;
#if defined(__APPLE__) || defined(__FreeBSD__)
    arc4random_buf(&bits, sizeof(bits));
#elif defined(_WIN32)
    unsigned int high_bits, lo;
    if (rand_s(&high_bits) == 0 && rand_s(&lo) == 0) {
        bits = ((uint64_t)high_bits << 32) | lo;
    } else {
        bits = ((uint64_t)(unsigned)rand() << 33) ^
            ((uint64_t)(unsigned)rand() << 2)  ^
            ((uint64_t)(unsigned)rand());
    }
#else
    FILE *urandom = fopen("/dev/urandom", "rb");
    if (urandom) { fread(&bits, sizeof(bits), 1, urandom); fclose(urandom); }
    else {
        bits = ((uint64_t)(unsigned)rand() << 33) ^
            ((uint64_t)(unsigned)rand() << 2)  ^
            ((uint64_t)(unsigned)rand());
    }
#endif
    return bits;
}

int64_t gray_random_i64_max(int64_t maximum) {
    ensure_seed();
    if (maximum <= 0) return 0;
    return (int64_t)(rand64() % (uint64_t)maximum);
}

int32_t gray_random_choice_index(int32_t length, const char *file, int line) {
    if (length == 0) gray_panic_code_at(file, line, "P0133", "random.choice called on an empty array");
    return (int32_t)gray_random_i64_max(length);
}

int64_t gray_random_i64_range(int64_t minimum, int64_t maximum) {
    if (minimum >= maximum)
        gray_panic_code("P0134", "random.rand_i64: range [%lld, %lld) is empty; min must be less than max", (long long)minimum, (long long)maximum);
    ensure_seed();
    return minimum + (int64_t)(rand64() % (uint64_t)(maximum - minimum));
}

GrayArray gray_random_bytes(GrayArena *arena, int64_t count) {
    if (count < 0)
        gray_panic_code("P0152", "random.rand_bytes: length %lld is negative", (long long)count);
    GrayArray array = gray_array_new(arena, sizeof(uint8_t), (int32_t)count, GRAY_ELEM_U8);
    ensure_seed();
    for (int64_t i = 0; i < count; i++) {
        uint8_t byte = (uint8_t)(rand() % 256);
        GRAY_ARRAY_PUSH(arena, &array, &byte);
    }
    return array;
}

int64_t gray_random_roll_dice(int64_t count, int64_t sides) {
    if (count < 1 || sides < 1)
        gray_panic_code("P0153", "random.roll_dice: count %lld and sides %lld must both be at least 1", (long long)count, (long long)sides);
    ensure_seed();
    int64_t total = 0;
    for (int64_t i = 0; i < count; i++) {
        int64_t roll = 1 + (int64_t)(rand64() % (uint64_t)sides);
        if (gray_add_overflows_i64(total, roll, &total))
            gray_panic_code("P0156", "random.roll_dice: total of %lld dice with %lld sides overflows i64", (long long)count, (long long)sides);
    }
    return total;
}

bool gray_random_bool(void) {
    ensure_seed();
    return rand() % 2 == 0;
}

uint8_t gray_random_u8(void) {
    ensure_seed();
    return (uint8_t)(rand() % 256);
}

int32_t gray_random_char(void) {
    ensure_seed();
    return (int32_t)(32 + rand() % 95); /* printable ASCII */
}

#define GRAY_SURROGATE_FIRST 0xD800
#define GRAY_SURROGATE_LAST  0xDFFF

int32_t gray_random_char_range(int32_t minimum, int32_t maximum) {
    if (minimum >= maximum)
        gray_panic_code("P0134", "random.rand_char: range [%d, %d) is empty; min must be less than max", (int)minimum, (int)maximum);
    if (minimum >= GRAY_SURROGATE_FIRST && maximum - 1 <= GRAY_SURROGATE_LAST)
        gray_panic_code("P0158", "random.rand_char: range [%d, %d) holds only surrogate codepoints", (int)minimum, (int)maximum);
    ensure_seed();
    /* A surrogate is not a valid char; draw again so the result stays
     * uniform over the codepoints that are. */
    int32_t codepoint;
    do {
        codepoint = minimum + (int32_t)(rand() % (maximum - minimum));
    } while (codepoint >= GRAY_SURROGATE_FIRST && codepoint <= GRAY_SURROGATE_LAST);
    return codepoint;
}

/* Appends `codepoint` as UTF-8 (1 to 4 bytes) and returns the byte count. */
static size_t random_encode_utf8(int32_t codepoint, char *output) {
    if (codepoint < 0x80) { output[0] = (char)codepoint; return 1; }
    if (codepoint < 0x800) {
        output[0] = (char)(0xC0 | (codepoint >> 6));
        output[1] = (char)(0x80 | (codepoint & 0x3F));
        return 2;
    }
    if (codepoint < 0x10000) {
        output[0] = (char)(0xE0 | (codepoint >> 12));
        output[1] = (char)(0x80 | ((codepoint >> 6) & 0x3F));
        output[2] = (char)(0x80 | (codepoint & 0x3F));
        return 3;
    }
    output[0] = (char)(0xF0 | (codepoint >> 18));
    output[1] = (char)(0x80 | ((codepoint >> 12) & 0x3F));
    output[2] = (char)(0x80 | ((codepoint >> 6) & 0x3F));
    output[3] = (char)(0x80 | (codepoint & 0x3F));
    return 4;
}

GrayString gray_random_string(GrayArena *arena, int64_t length, GrayString alphabet) {
    if (length <= 0) return gray_string_lit("");
    if (alphabet.len == 0) {
        gray_panic_code("P0123",
            "random.rand_string: alphabet is empty but length is %lld", (long long)length);
    }
    ensure_seed();
    /* Draw whole characters, not bytes: decode the alphabet once, then pick
     * from its codepoints and encode each pick. */
    int32_t *codepoints = gray_arena_alloc_uninitialized(arena, sizeof(int32_t) * (size_t)alphabet.len);
    int32_t codepoint_count = 0;
    const uint8_t *cursor = (const uint8_t *)alphabet.data;
    const uint8_t *end_cursor = cursor + alphabet.len;
    while (cursor < end_cursor)
        cursor += gray_builtin_utf8_next(cursor, end_cursor, &codepoints[codepoint_count++]);

    char *buffer = gray_arena_alloc_uninitialized(arena, (size_t)length * 4 + 1);
    size_t byte_length = 0;
    for (int64_t i = 0; i < length; i++)
        byte_length += random_encode_utf8(codepoints[rand() % codepoint_count], buffer + byte_length);
    buffer[byte_length] = '\0';
    return gray_string_new(arena, buffer, (int32_t)byte_length);
}

GrayArray gray_random_shuffle(GrayArena *arena, GrayArray *array) {
    ensure_seed();
    GrayArray result = gray_array_copy(arena, array);
    char *data = (char *)result.data;
    size_t element_size = (size_t)result.elem_size;
    /* Scratch slot sized to the actual element width. The previous
     * fixed char tmp[64] overflowed the stack for any element type
     * larger than 64 bytes (struct arrays, nested arrays/maps). */
    void *scratch = gray_arena_alloc_uninitialized(arena, element_size);
    for (int32_t i = result.len - 1; i > 0; i--) {
        int32_t swap_index = rand() % (i + 1);
        memcpy(scratch, data + i * element_size, element_size);
        memcpy(data + i * element_size, data + swap_index * element_size, element_size);
        memcpy(data + swap_index * element_size, scratch, element_size);
    }
    return result;
}

GrayArray gray_random_sample(GrayArena *arena, GrayArray *array, int64_t count) {
    if (count > array->len)
        gray_panic_code("P0062", "random.sample() count %lld exceeds array length %d", (long long)count, (int)array->len);
    if (count < 0)
        gray_panic_code("P0063", "random.sample() count cannot be negative (%lld)", (long long)count);
    GrayArray shuffled = gray_random_shuffle(arena, array);
    shuffled.len = (int32_t)count;
    return shuffled;
}

