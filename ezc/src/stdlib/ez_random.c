/*
 * ez_random.c - @random module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_random.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

static bool _seeded = false;
static void ensure_seed(void) {
    if (!_seeded) { srand((unsigned)time(NULL)); _seeded = true; }
}

void ez_random_seed(int64_t value) {
    srand((unsigned)value);
    _seeded = true;
}

double ez_random_float_unit(void) {
    ensure_seed();
    return (double)rand() / RAND_MAX;
}

double ez_random_float_range(double min, double max) {
    return min + ez_random_float_unit() * (max - min);
}

int64_t ez_random_int_max(int64_t max) {
    ensure_seed();
    if (max <= 0) return 0;
    return (int64_t)(rand() % (int)max);
}

int64_t ez_random_int_range(int64_t min, int64_t max) {
    ensure_seed();
    if (min >= max) return min;
    return min + (int64_t)(rand() % (int)(max - min));
}

bool ez_random_bool(void) {
    ensure_seed();
    return rand() % 2 == 0;
}

uint8_t ez_random_byte(void) {
    ensure_seed();
    return (uint8_t)(rand() % 256);
}

int32_t ez_random_char(void) {
    ensure_seed();
    return (int32_t)(32 + rand() % 95); /* printable ASCII */
}

int32_t ez_random_char_range(int32_t min, int32_t max) {
    ensure_seed();
    if (min >= max) return min;
    return min + (int32_t)(rand() % (max - min));
}

EzArray ez_random_shuffle(EzArena *arena, EzArray *arr) {
    ensure_seed();
    EzArray result = ez_array_copy(arena, arr);
    char *data = (char *)result.data;
    size_t es = (size_t)result.elem_size;
    char tmp[64]; /* max element size */
    for (int32_t i = result.len - 1; i > 0; i--) {
        int32_t j = rand() % (i + 1);
        memcpy(tmp, data + i * es, es);
        memcpy(data + i * es, data + j * es, es);
        memcpy(data + j * es, tmp, es);
    }
    return result;
}

EzArray ez_random_sample(EzArena *arena, EzArray *arr, int32_t n) {
    if (n > arr->len) {
        fflush(stdout);
        fprintf(stderr, "panic: random.sample() count %d exceeds array length %d\n",
            (int)n, (int)arr->len);
        exit(1);
    }
    if (n < 0) {
        fflush(stdout);
        fprintf(stderr, "panic: random.sample() count cannot be negative (%d)\n", (int)n);
        exit(1);
    }
    EzArray shuffled = ez_random_shuffle(arena, arr);
    shuffled.len = n;
    return shuffled;
}

EzString ez_random_hex(EzArena *arena, int64_t length) {
    ensure_seed();
    char *hex = ez_arena_alloc(arena, (size_t)length + 1);
    for (int64_t i = 0; i < length; i++) {
        snprintf(hex + i, 2, "%x", rand() % 16);
    }
    hex[length] = '\0';
    return (EzString){ hex, (int32_t)length };
}
