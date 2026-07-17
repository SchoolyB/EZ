/*
 * math.c - @math module implementation (non-inline functions)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "math.h"
#include <stdlib.h>
#include <time.h>

static bool _rand_seeded = false;
static void ensure_seeded(void) {
    if (!_rand_seeded) {
        srand((unsigned int)time(NULL));
        _rand_seeded = true;
    }
}

int64_t gray_math_random_int(int64_t min, int64_t max) {
    ensure_seeded();
    if (min >= max) return min;
    return min + (int64_t)(rand() % (int)(max - min));
}

double gray_math_random_float(double min, double max) {
    ensure_seeded();
    return min + ((double)rand() / RAND_MAX) * (max - min);
}

int64_t gray_math_factorial(int64_t n) {
    if (n < 0) gray_panic_code("P0070", "math.factorial() requires a non-negative integer, got %lld", (long long)n);
    if (n <= 1) return 1;
    int64_t result = 1;
    for (int64_t i = 2; i <= n; i++) result *= i;
    return result;
}

int64_t gray_math_gcd(int64_t a, int64_t b) {
    if (a < 0) a = -a;
    if (b < 0) b = -b;
    while (b != 0) { int64_t t = b; b = a % b; a = t; }
    return a;
}

int64_t gray_math_lcm(int64_t a, int64_t b) {
    if (a == 0 || b == 0) return 0;
    int64_t g = gray_math_gcd(a, b);
    return (a / g) * b;
}

bool gray_math_is_prime(int64_t n) {
    if (n < 2) return false;
    if (n < 4) return true;
    if (n % 2 == 0 || n % 3 == 0) return false;
    for (int64_t i = 5; i * i <= n; i += 6) {
        if (n % i == 0 || n % (i + 2) == 0) return false;
    }
    return true;
}
