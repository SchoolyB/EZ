/*
 * ez_math.h - @math module for EZ
 *
 * Most functions are thin wrappers around <math.h>.
 * Constants are handled by codegen (math.PI, math.E, etc.)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_MATH_H
#define EZ_MATH_H

#include "../runtime/ez_runtime.h"
#include <math.h>

/* Arithmetic */
static inline int64_t ez_math_abs_int(int64_t n) { return n < 0 ? -n : n; }
static inline double  ez_math_abs_float(double n) { return fabs(n); }
static inline int64_t ez_math_sign(int64_t n) { return n > 0 ? 1 : (n < 0 ? -1 : 0); }
static inline int64_t ez_math_min_int(int64_t a, int64_t b) { return a < b ? a : b; }
static inline double  ez_math_min_float(double a, double b) { return a < b ? a : b; }
static inline int64_t ez_math_max_int(int64_t a, int64_t b) { return a > b ? a : b; }
static inline double  ez_math_max_float(double a, double b) { return a > b ? a : b; }
static inline int64_t ez_math_clamp_int(int64_t v, int64_t lo, int64_t hi) {
    return v < lo ? lo : (v > hi ? hi : v);
}
static inline double  ez_math_clamp_float(double v, double lo, double hi) {
    return v < lo ? lo : (v > hi ? hi : v);
}

/* Rounding */
static inline double ez_math_floor(double n) { return floor(n); }
static inline double ez_math_ceil(double n) { return ceil(n); }
static inline double ez_math_round(double n) { return round(n); }
static inline double ez_math_trunc(double n) { return trunc(n); }

/* Powers/Roots */
static inline double ez_math_pow(double b, double e) { return pow(b, e); }
static inline double ez_math_sqrt(double n) {
    if (n < 0) { fflush(stdout); fprintf(stderr, "panic: math.sqrt() requires a non-negative number, got %g\n", n); exit(1); }
    return sqrt(n);
}
static inline double ez_math_cbrt(double n) { return cbrt(n); }
static inline double ez_math_hypot(double x, double y) { return hypot(x, y); }
static inline double ez_math_exp(double n) { return exp(n); }
static inline double ez_math_exp2(double n) { return exp2(n); }

/* Logarithms */
static inline double ez_math_log(double n) {
    if (n <= 0) { fflush(stdout); fprintf(stderr, "panic: math.log() requires a positive number, got %g\n", n); exit(1); }
    return log(n);
}
static inline double ez_math_log2(double n) {
    if (n <= 0) { fflush(stdout); fprintf(stderr, "panic: math.log2() requires a positive number, got %g\n", n); exit(1); }
    return log2(n);
}
static inline double ez_math_log10(double n) {
    if (n <= 0) { fflush(stdout); fprintf(stderr, "panic: math.log10() requires a positive number, got %g\n", n); exit(1); }
    return log10(n);
}
static inline double ez_math_log_base(double v, double b) { return log(v) / log(b); }

/* Trigonometry */
static inline double ez_math_sin(double r) { return sin(r); }
static inline double ez_math_cos(double r) { return cos(r); }
static inline double ez_math_tan(double r) { return tan(r); }
static inline double ez_math_asin(double n) {
    if (n < -1.0 || n > 1.0) { fflush(stdout); fprintf(stderr, "panic: math.asin() requires value in [-1, 1], got %g\n", n); exit(1); }
    return asin(n);
}
static inline double ez_math_acos(double n) {
    if (n < -1.0 || n > 1.0) { fflush(stdout); fprintf(stderr, "panic: math.acos() requires value in [-1, 1], got %g\n", n); exit(1); }
    return acos(n);
}
static inline double ez_math_atan(double n) { return atan(n); }
static inline double ez_math_atan2(double y, double x) { return atan2(y, x); }
static inline double ez_math_sinh(double n) { return sinh(n); }
static inline double ez_math_cosh(double n) { return cosh(n); }
static inline double ez_math_tanh(double n) { return tanh(n); }
static inline double ez_math_deg_to_rad(double d) { return d * 3.14159265358979323846 / 180.0; }
static inline double ez_math_rad_to_deg(double r) { return r * 180.0 / 3.14159265358979323846; }

/* Properties */
static inline bool ez_math_is_even(int64_t n) { return n % 2 == 0; }
static inline bool ez_math_is_odd(int64_t n) { return n % 2 != 0; }
static inline bool ez_math_is_infinite(double n) { return isinf(n); }
static inline bool ez_math_is_nan(double n) { return isnan(n); }
static inline bool ez_math_is_finite(double n) { return isfinite(n); }

/* Random */
int64_t ez_math_random_int(int64_t min, int64_t max);
double ez_math_random_float(double min, double max);

/* Statistical */
int64_t ez_math_factorial(int64_t n);
int64_t ez_math_gcd(int64_t a, int64_t b);
int64_t ez_math_lcm(int64_t a, int64_t b);
bool ez_math_is_prime(int64_t n);

/* Utility */
static inline double ez_math_lerp(double a, double b, double t) { return a + (b - a) * t; }
static inline double ez_math_distance(double x1, double y1, double x2, double y2) {
    return hypot(x2 - x1, y2 - y1);
}

#endif
