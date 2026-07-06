/*
 * ez_bigint.h - Wide integer types (i128, u128, i256, u256) for EZ
 *
 * Struct-based portable implementation backed by uint64_t limbs.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_BIGINT_H
#define EZ_BIGINT_H

#include "../runtime/ez_runtime.h"
#include <inttypes.h>

/* --- Type Definitions --- */

typedef struct { uint64_t lo; int64_t hi; } ez_i128;
typedef struct { uint64_t lo; uint64_t hi; } ez_u128;
typedef struct { uint64_t w[4]; } ez_i256;  /* w[0]=lo ... w[3]=hi */
typedef struct { uint64_t w[4]; } ez_u256;

/* --- Max decimal digits for string rendering --- */
#define I128_MAX_DIGITS     21
#define U128_MAX_DIGITS     21
#define I256_MAX_DIGITS     40
#define U256_MAX_DIGITS     80

/* --- Zero Constants --- */

#define EZ_I128_ZERO  ((ez_i128){0, 0})
#define EZ_U128_ZERO  ((ez_u128){0, 0})
#define EZ_I256_ZERO  ((ez_i256){{0, 0, 0, 0}})
#define EZ_U256_ZERO  ((ez_u256){{0, 0, 0, 0}})

/* --- Constructors --- */

static inline ez_i128 ez_i128_from_i64(int64_t v) {
    ez_i128 r;
    r.lo = (uint64_t)v;
    r.hi = (v < 0) ? -1 : 0;
    return r;
}

static inline ez_u128 ez_u128_from_u64(uint64_t v) {
    ez_u128 r;
    r.lo = v;
    r.hi = 0;
    return r;
}

static inline ez_i256 ez_i256_from_i64(int64_t v) {
    ez_i256 r;
    r.w[0] = (uint64_t)v;
    int64_t sign = (v < 0) ? -1 : 0;
    r.w[1] = (uint64_t)sign;
    r.w[2] = (uint64_t)sign;
    r.w[3] = (uint64_t)sign;
    return r;
}

static inline ez_u256 ez_u256_from_u64(uint64_t v) {
    ez_u256 r;
    r.w[0] = v;
    r.w[1] = 0;
    r.w[2] = 0;
    r.w[3] = 0;
    return r;
}

/* --- Size Casting (range-checked) --- */

static inline int64_t ez_i128_to_i64(ez_i128 a) {
    if (a.hi != ((int64_t)a.lo >> 63)) {
        ez_panic_code("P0093", "cast from i128 failed; value is outside the representable range of int64");
    }
    return (int64_t)a.lo;
}

static inline uint64_t ez_i128_to_u64(ez_i128 a) {
    if (a.hi != 0) {
        ez_panic_code("P0094", "cast from i128 failed; value is negative or outside the representable range of uint64");
    }
    return a.lo;
}

static inline int64_t ez_u128_to_i64(ez_u128 a) {
    if (a.hi != 0 || a.lo > (uint64_t)INT64_MAX) {
        ez_panic_code("P0095", "cast from u128 failed; value exceeds the representable range of int64");
    }
    return (int64_t)a.lo;
}

static inline uint64_t ez_u128_to_u64(ez_u128 a) {
    if (a.hi != 0) {
        ez_panic_code("P0096", "cast from u128 failed; value exceeds the representable range of uint64");
    }
    return a.lo;
}

static inline int64_t ez_i256_to_i64(ez_i256 a) {
    uint64_t sign_ext = (uint64_t)((int64_t)a.w[0] >> 63);
    if (a.w[1] != sign_ext || a.w[2] != sign_ext || a.w[3] != sign_ext) {
        ez_panic_code("P0097", "cast from i256 failed; value is outside the representable range of int64");
    }
    return (int64_t)a.w[0];
}

static inline uint64_t ez_i256_to_u64(ez_i256 a) {
    if (a.w[1] != 0 || a.w[2] != 0 || a.w[3] != 0) {
        ez_panic_code("P0098", "cast from i256 failed; value is negative or outside the representable range of uint64");
    }
    return a.w[0];
}

static inline int64_t ez_u256_to_i64(ez_u256 a) {
    if (a.w[1] != 0 || a.w[2] != 0 || a.w[3] != 0 || a.w[0] > (uint64_t)INT64_MAX) {
        ez_panic_code("P0099", "cast from u256 failed; value exceeds the representable range of int64");
    }
    return (int64_t)a.w[0];
}

static inline uint64_t ez_u256_to_u64(ez_u256 a) {
    if (a.w[1] != 0 || a.w[2] != 0 || a.w[3] != 0) {
        ez_panic_code("P0100", "cast from u256 failed; value exceeds the representable range of uint64");
    }
    return a.w[0];
}

static inline ez_u128 ez_u128_from_i128(ez_i128 a) {
    ez_u128 r;
    r.lo = a.lo;
    r.hi = (uint64_t)a.hi;
    return r;
}

static inline ez_i128 ez_i128_from_u128(ez_u128 a) {
    ez_i128 r;
    r.lo = a.lo;
    r.hi = (int64_t)a.hi;
    return r;
}

static inline ez_i256 ez_i256_from_i128(ez_i128 a) {
    ez_i256 r;
    r.w[0] = a.lo;
    r.w[1] = (uint64_t)a.hi;
    int64_t sign = (a.hi < 0) ? -1 : 0;
    r.w[2] = (uint64_t)sign;
    r.w[3] = (uint64_t)sign;
    return r;
}

static inline ez_u256 ez_u256_from_u128(ez_u128 a) {
    ez_u256 r;
    r.w[0] = a.lo;
    r.w[1] = a.hi;
    r.w[2] = 0;
    r.w[3] = 0;
    return r;
}

static inline ez_i128 ez_i128_from_i256(ez_i256 a) {
    ez_i128 r;
    r.lo = a.w[0];
    r.hi = (int64_t)a.w[1];
    return r;
}

static inline ez_u128 ez_u128_from_u256(ez_u256 a) {
    ez_u128 r;
    r.lo = a.w[0];
    r.hi = a.w[1];
    return r;
}

/* --- i128 Arithmetic --- */

static inline ez_i128 ez_i128_add(ez_i128 a, ez_i128 b) {
    ez_i128 r;
    r.lo = a.lo + b.lo;
    r.hi = a.hi + b.hi + (r.lo < a.lo ? 1 : 0);
    return r;
}

static inline ez_i128 ez_i128_sub(ez_i128 a, ez_i128 b) {
    ez_i128 r;
    r.lo = a.lo - b.lo;
    r.hi = a.hi - b.hi - (a.lo < b.lo ? 1 : 0);
    return r;
}

static inline ez_i128 ez_i128_neg(ez_i128 a) {
    ez_i128 r;
    r.lo = ~a.lo + 1;
    r.hi = ~a.hi + (r.lo == 0 ? 1 : 0);
    return r;
}

static inline ez_i128 ez_i128_mul(ez_i128 a, ez_i128 b) {
    /* Schoolbook multiplication on 64-bit halves */
    uint64_t a_lo = a.lo, b_lo = b.lo;
    uint64_t a_hi = (uint64_t)a.hi, b_hi = (uint64_t)b.hi;

    /* Split each 64-bit value into 32-bit halves for overflow-safe multiply */
    uint64_t a0 = a_lo & 0xFFFFFFFF, a1 = a_lo >> 32;
    uint64_t b0 = b_lo & 0xFFFFFFFF, b1 = b_lo >> 32;

    uint64_t p00 = a0 * b0;
    uint64_t p01 = a0 * b1;
    uint64_t p10 = a1 * b0;
    uint64_t p11 = a1 * b1;

    uint64_t mid = p01 + (p00 >> 32);
    uint64_t carry = ((mid & 0xFFFFFFFF) + p10) >> 32;

    ez_i128 r;
    r.lo = a_lo * b_lo;
    r.hi = (int64_t)(p11 + (mid >> 32) + carry + a_lo * b_hi + a_hi * b_lo);
    return r;
}

/* Division helper: unsigned 128-bit divide */
static inline void ez_u128_divmod(ez_u128 a, ez_u128 b, ez_u128 *q, ez_u128 *rem) {
    if (b.hi == 0 && b.lo == 0) {
        /* Division by zero — return zero */
        *q = EZ_U128_ZERO;
        *rem = EZ_U128_ZERO;
        return;
    }
    if (a.hi == 0 && b.hi == 0) {
        /* Both fit in 64 bits */
        q->hi = 0; q->lo = a.lo / b.lo;
        rem->hi = 0; rem->lo = a.lo % b.lo;
        return;
    }
    /* Binary long division */
    ez_u128 quotient = EZ_U128_ZERO;
    ez_u128 remainder = EZ_U128_ZERO;
    for (int i = 127; i >= 0; i--) {
        /* remainder <<= 1 */
        remainder.hi = (remainder.hi << 1) | (remainder.lo >> 63);
        remainder.lo <<= 1;
        /* Get bit i of a */
        if (i >= 64) {
            remainder.lo |= (a.hi >> (i - 64)) & 1;
        } else {
            remainder.lo |= (a.lo >> i) & 1;
        }
        /* if remainder >= b */
        if (remainder.hi > b.hi || (remainder.hi == b.hi && remainder.lo >= b.lo)) {
            /* remainder -= b */
            uint64_t borrow = (remainder.lo < b.lo) ? 1 : 0;
            remainder.lo -= b.lo;
            remainder.hi -= b.hi + borrow;
            /* Set bit i of quotient */
            if (i >= 64) {
                quotient.hi |= (uint64_t)1 << (i - 64);
            } else {
                quotient.lo |= (uint64_t)1 << i;
            }
        }
    }
    *q = quotient;
    *rem = remainder;
}

static inline ez_i128 ez_i128_div(ez_i128 a, ez_i128 b) {
    if (b.lo == 0 && b.hi == 0) return EZ_I128_ZERO;
    bool neg = false;
    ez_u128 ua, ub;
    if (a.hi < 0) { ez_i128 t = ez_i128_neg(a); ua.lo = t.lo; ua.hi = (uint64_t)t.hi; neg = !neg; }
    else { ua.lo = a.lo; ua.hi = (uint64_t)a.hi; }
    if (b.hi < 0) { ez_i128 t = ez_i128_neg(b); ub.lo = t.lo; ub.hi = (uint64_t)t.hi; neg = !neg; }
    else { ub.lo = b.lo; ub.hi = (uint64_t)b.hi; }
    ez_u128 q, rem;
    ez_u128_divmod(ua, ub, &q, &rem);
    ez_i128 result;
    result.lo = q.lo; result.hi = (int64_t)q.hi;
    return neg ? ez_i128_neg(result) : result;
}

static inline ez_i128 ez_i128_mod(ez_i128 a, ez_i128 b) {
    if (b.lo == 0 && b.hi == 0) return EZ_I128_ZERO;
    bool neg_a = (a.hi < 0);
    ez_u128 ua, ub;
    if (neg_a) { ez_i128 t = ez_i128_neg(a); ua.lo = t.lo; ua.hi = (uint64_t)t.hi; }
    else { ua.lo = a.lo; ua.hi = (uint64_t)a.hi; }
    if (b.hi < 0) { ez_i128 t = ez_i128_neg(b); ub.lo = t.lo; ub.hi = (uint64_t)t.hi; }
    else { ub.lo = b.lo; ub.hi = (uint64_t)b.hi; }
    ez_u128 q, rem;
    ez_u128_divmod(ua, ub, &q, &rem);
    ez_i128 result;
    result.lo = rem.lo; result.hi = (int64_t)rem.hi;
    return neg_a ? ez_i128_neg(result) : result;
}

/* --- i128 Comparison --- */

static inline bool ez_i128_eq(ez_i128 a, ez_i128 b) { return a.lo == b.lo && a.hi == b.hi; }
static inline bool ez_i128_ne(ez_i128 a, ez_i128 b) { return a.lo != b.lo || a.hi != b.hi; }
static inline bool ez_i128_lt(ez_i128 a, ez_i128 b) {
    return a.hi < b.hi || (a.hi == b.hi && a.lo < b.lo);
}
static inline bool ez_i128_gt(ez_i128 a, ez_i128 b) { return ez_i128_lt(b, a); }
static inline bool ez_i128_le(ez_i128 a, ez_i128 b) { return !ez_i128_gt(a, b); }
static inline bool ez_i128_ge(ez_i128 a, ez_i128 b) { return !ez_i128_lt(a, b); }

/* --- u128 Arithmetic --- */

static inline ez_u128 ez_u128_add(ez_u128 a, ez_u128 b) {
    ez_u128 r;
    r.lo = a.lo + b.lo;
    r.hi = a.hi + b.hi + (r.lo < a.lo ? 1 : 0);
    return r;
}

static inline ez_u128 ez_u128_sub(ez_u128 a, ez_u128 b) {
    ez_u128 r;
    r.lo = a.lo - b.lo;
    r.hi = a.hi - b.hi - (a.lo < b.lo ? 1 : 0);
    return r;
}

static inline ez_u128 ez_u128_mul(ez_u128 a, ez_u128 b) {
    uint64_t a0 = a.lo & 0xFFFFFFFF, a1 = a.lo >> 32;
    uint64_t b0 = b.lo & 0xFFFFFFFF, b1 = b.lo >> 32;

    uint64_t p00 = a0 * b0;
    uint64_t p01 = a0 * b1;
    uint64_t p10 = a1 * b0;
    uint64_t p11 = a1 * b1;

    uint64_t mid = p01 + (p00 >> 32);
    uint64_t carry = ((mid & 0xFFFFFFFF) + p10) >> 32;

    ez_u128 r;
    r.lo = a.lo * b.lo;
    r.hi = p11 + (mid >> 32) + carry + a.lo * b.hi + a.hi * b.lo;
    return r;
}

static inline ez_u128 ez_u128_div(ez_u128 a, ez_u128 b) {
    ez_u128 q, rem;
    ez_u128_divmod(a, b, &q, &rem);
    return q;
}

static inline ez_u128 ez_u128_mod(ez_u128 a, ez_u128 b) {
    ez_u128 q, rem;
    ez_u128_divmod(a, b, &q, &rem);
    return rem;
}

/* --- u128 Comparison --- */

static inline bool ez_u128_eq(ez_u128 a, ez_u128 b) { return a.lo == b.lo && a.hi == b.hi; }
static inline bool ez_u128_ne(ez_u128 a, ez_u128 b) { return a.lo != b.lo || a.hi != b.hi; }
static inline bool ez_u128_lt(ez_u128 a, ez_u128 b) {
    return a.hi < b.hi || (a.hi == b.hi && a.lo < b.lo);
}
static inline bool ez_u128_gt(ez_u128 a, ez_u128 b) { return ez_u128_lt(b, a); }
static inline bool ez_u128_le(ez_u128 a, ez_u128 b) { return !ez_u128_gt(a, b); }
static inline bool ez_u128_ge(ez_u128 a, ez_u128 b) { return !ez_u128_lt(a, b); }

/* --- i256 Arithmetic --- */

static inline ez_i256 ez_i256_add(ez_i256 a, ez_i256 b) {
    ez_i256 r;
    uint64_t carry = 0;
    for (int i = 0; i < 4; i++) {
        uint64_t sum = a.w[i] + b.w[i] + carry;
        carry = (sum < a.w[i] || (carry && sum == a.w[i])) ? 1 : 0;
        r.w[i] = sum;
    }
    return r;
}

static inline ez_i256 ez_i256_sub(ez_i256 a, ez_i256 b) {
    ez_i256 r;
    uint64_t borrow = 0;
    for (int i = 0; i < 4; i++) {
        uint64_t diff = a.w[i] - b.w[i] - borrow;
        borrow = (a.w[i] < b.w[i] + borrow) ? 1 : 0;
        r.w[i] = diff;
    }
    return r;
}

static inline ez_i256 ez_i256_neg(ez_i256 a) {
    ez_i256 r;
    uint64_t carry = 1;
    for (int i = 0; i < 4; i++) {
        uint64_t sum = ~a.w[i] + carry;
        carry = (sum < ~a.w[i]) ? 1 : 0;
        r.w[i] = sum;
    }
    return r;
}

static inline ez_i256 ez_i256_mul(ez_i256 a, ez_i256 b) {
    /* Schoolbook multiplication — only need lower 256 bits */
    ez_i256 r = EZ_I256_ZERO;
    for (int i = 0; i < 4; i++) {
        uint64_t carry = 0;
        for (int j = 0; j < 4 - i; j++) {
            /* Multiply a.w[j] * b.w[i] and add to r.w[i+j] */
            uint64_t a_lo = a.w[j] & 0xFFFFFFFF, a_hi = a.w[j] >> 32;
            uint64_t b_lo = b.w[i] & 0xFFFFFFFF, b_hi = b.w[i] >> 32;
            uint64_t p00 = a_lo * b_lo;
            uint64_t p01 = a_lo * b_hi;
            uint64_t p10 = a_hi * b_lo;
            uint64_t p11 = a_hi * b_hi;
            uint64_t mid = p01 + (p00 >> 32);
            uint64_t lo = (p00 & 0xFFFFFFFF) | ((mid & 0xFFFFFFFF) << 32) + (p10 << 32);

            /* Simpler approach: just use the truncating product */
            lo = a.w[j] * b.w[i];
            uint64_t hi_part = p11 + (mid >> 32) + (((mid & 0xFFFFFFFF) + p10) >> 32);

            uint64_t old = r.w[i + j];
            r.w[i + j] = old + lo + carry;
            carry = hi_part + (r.w[i + j] < old + lo ? 1 : 0);
            if (old + lo < old) carry++;
        }
    }
    return r;
}

/* 256-bit unsigned divmod (binary long division) */
static inline void ez_u256_divmod(ez_u256 a, ez_u256 b, ez_u256 *q, ez_u256 *rem) {
    int b_zero = (b.w[0] == 0 && b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0);
    if (b_zero) { *q = EZ_U256_ZERO; *rem = EZ_U256_ZERO; return; }
    if (a.w[1] == 0 && a.w[2] == 0 && a.w[3] == 0 &&
        b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0) {
        *q = EZ_U256_ZERO; q->w[0] = a.w[0] / b.w[0];
        *rem = EZ_U256_ZERO; rem->w[0] = a.w[0] % b.w[0];
        return;
    }
    ez_u256 quotient = EZ_U256_ZERO;
    ez_u256 remainder = EZ_U256_ZERO;
    for (int i = 255; i >= 0; i--) {
        /* remainder <<= 1 */
        remainder.w[3] = (remainder.w[3] << 1) | (remainder.w[2] >> 63);
        remainder.w[2] = (remainder.w[2] << 1) | (remainder.w[1] >> 63);
        remainder.w[1] = (remainder.w[1] << 1) | (remainder.w[0] >> 63);
        remainder.w[0] <<= 1;
        /* Get bit i of a */
        int word = i / 64;
        int bit = i % 64;
        remainder.w[0] |= (a.w[word] >> bit) & 1;
        /* if remainder >= b */
        bool ge = false;
        for (int k = 3; k >= 0; k--) {
            if (remainder.w[k] > b.w[k]) { ge = true; break; }
            if (remainder.w[k] < b.w[k]) break;
            if (k == 0) ge = true;
        }
        if (ge) {
            /* remainder -= b */
            uint64_t borrow = 0;
            for (int k = 0; k < 4; k++) {
                uint64_t diff = remainder.w[k] - b.w[k] - borrow;
                borrow = (remainder.w[k] < b.w[k] + borrow) ? 1 : 0;
                remainder.w[k] = diff;
            }
            quotient.w[word] |= (uint64_t)1 << bit;
        }
    }
    *q = quotient;
    *rem = remainder;
}

static inline bool ez_i256_is_neg(ez_i256 a) { return (int64_t)a.w[3] < 0; }

static inline ez_i256 ez_i256_div(ez_i256 a, ez_i256 b) {
    int b_zero = (b.w[0] == 0 && b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0);
    if (b_zero) return EZ_I256_ZERO;
    bool neg = false;
    ez_u256 ua, ub;
    if (ez_i256_is_neg(a)) { ez_i256 t = ez_i256_neg(a); memcpy(&ua, &t, sizeof(ua)); neg = !neg; }
    else { memcpy(&ua, &a, sizeof(ua)); }
    if (ez_i256_is_neg(b)) { ez_i256 t = ez_i256_neg(b); memcpy(&ub, &t, sizeof(ub)); neg = !neg; }
    else { memcpy(&ub, &b, sizeof(ub)); }
    ez_u256 q, rem;
    ez_u256_divmod(ua, ub, &q, &rem);
    ez_i256 result; memcpy(&result, &q, sizeof(result));
    return neg ? ez_i256_neg(result) : result;
}

static inline ez_i256 ez_i256_mod(ez_i256 a, ez_i256 b) {
    int b_zero = (b.w[0] == 0 && b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0);
    if (b_zero) return EZ_I256_ZERO;
    bool neg_a = ez_i256_is_neg(a);
    ez_u256 ua, ub;
    if (neg_a) { ez_i256 t = ez_i256_neg(a); memcpy(&ua, &t, sizeof(ua)); }
    else { memcpy(&ua, &a, sizeof(ua)); }
    if (ez_i256_is_neg(b)) { ez_i256 t = ez_i256_neg(b); memcpy(&ub, &t, sizeof(ub)); }
    else { memcpy(&ub, &b, sizeof(ub)); }
    ez_u256 q, rem;
    ez_u256_divmod(ua, ub, &q, &rem);
    ez_i256 result; memcpy(&result, &rem, sizeof(result));
    return neg_a ? ez_i256_neg(result) : result;
}

/* --- i256 Comparison --- */

static inline bool ez_i256_eq(ez_i256 a, ez_i256 b) {
    return a.w[0] == b.w[0] && a.w[1] == b.w[1] && a.w[2] == b.w[2] && a.w[3] == b.w[3];
}
static inline bool ez_i256_ne(ez_i256 a, ez_i256 b) { return !ez_i256_eq(a, b); }
static inline bool ez_i256_lt(ez_i256 a, ez_i256 b) {
    if ((int64_t)a.w[3] != (int64_t)b.w[3]) return (int64_t)a.w[3] < (int64_t)b.w[3];
    if (a.w[2] != b.w[2]) return a.w[2] < b.w[2];
    if (a.w[1] != b.w[1]) return a.w[1] < b.w[1];
    return a.w[0] < b.w[0];
}
static inline bool ez_i256_gt(ez_i256 a, ez_i256 b) { return ez_i256_lt(b, a); }
static inline bool ez_i256_le(ez_i256 a, ez_i256 b) { return !ez_i256_gt(a, b); }
static inline bool ez_i256_ge(ez_i256 a, ez_i256 b) { return !ez_i256_lt(a, b); }

/* --- u256 Arithmetic --- */

static inline ez_u256 ez_u256_add(ez_u256 a, ez_u256 b) {
    ez_u256 r;
    uint64_t carry = 0;
    for (int i = 0; i < 4; i++) {
        uint64_t sum = a.w[i] + b.w[i] + carry;
        carry = (sum < a.w[i] || (carry && sum == a.w[i])) ? 1 : 0;
        r.w[i] = sum;
    }
    return r;
}

static inline ez_u256 ez_u256_sub(ez_u256 a, ez_u256 b) {
    ez_u256 r;
    uint64_t borrow = 0;
    for (int i = 0; i < 4; i++) {
        uint64_t diff = a.w[i] - b.w[i] - borrow;
        borrow = (a.w[i] < b.w[i] + borrow) ? 1 : 0;
        r.w[i] = diff;
    }
    return r;
}

static inline ez_u256 ez_u256_mul(ez_u256 a, ez_u256 b) {
    ez_u256 r = EZ_U256_ZERO;
    for (int i = 0; i < 4; i++) {
        uint64_t carry = 0;
        for (int j = 0; j < 4 - i; j++) {
            uint64_t a_lo = a.w[j] & 0xFFFFFFFF, a_hi = a.w[j] >> 32;
            uint64_t b_lo = b.w[i] & 0xFFFFFFFF, b_hi = b.w[i] >> 32;
            uint64_t p00 = a_lo * b_lo;
            uint64_t p01 = a_lo * b_hi;
            uint64_t p10 = a_hi * b_lo;
            uint64_t p11 = a_hi * b_hi;
            uint64_t mid = p01 + (p00 >> 32);
            uint64_t lo = a.w[j] * b.w[i];
            uint64_t hi_part = p11 + (mid >> 32) + (((mid & 0xFFFFFFFF) + p10) >> 32);

            uint64_t old = r.w[i + j];
            r.w[i + j] = old + lo + carry;
            carry = hi_part + (r.w[i + j] < old + lo ? 1 : 0);
            if (old + lo < old) carry++;
        }
    }
    return r;
}

static inline ez_u256 ez_u256_div(ez_u256 a, ez_u256 b) {
    ez_u256 q, rem;
    ez_u256_divmod(a, b, &q, &rem);
    return q;
}

static inline ez_u256 ez_u256_mod(ez_u256 a, ez_u256 b) {
    ez_u256 q, rem;
    ez_u256_divmod(a, b, &q, &rem);
    return rem;
}

/* --- u256 Comparison --- */

static inline bool ez_u256_eq(ez_u256 a, ez_u256 b) {
    return a.w[0] == b.w[0] && a.w[1] == b.w[1] && a.w[2] == b.w[2] && a.w[3] == b.w[3];
}
static inline bool ez_u256_ne(ez_u256 a, ez_u256 b) { return !ez_u256_eq(a, b); }
static inline bool ez_u256_lt(ez_u256 a, ez_u256 b) {
    if (a.w[3] != b.w[3]) return a.w[3] < b.w[3];
    if (a.w[2] != b.w[2]) return a.w[2] < b.w[2];
    if (a.w[1] != b.w[1]) return a.w[1] < b.w[1];
    return a.w[0] < b.w[0];
}
static inline bool ez_u256_gt(ez_u256 a, ez_u256 b) { return ez_u256_lt(b, a); }
static inline bool ez_u256_le(ez_u256 a, ez_u256 b) { return !ez_u256_gt(a, b); }
static inline bool ez_u256_ge(ez_u256 a, ez_u256 b) { return !ez_u256_lt(a, b); }

/* --- Decimal String Constructors --- */

static inline ez_u128 ez_u128_from_decimal(const char *s) {
    ez_u128 result = EZ_U128_ZERO;
    ez_u128 ten = {10, 0};
    while (*s) {
        if (*s == '_') { s++; continue; }
        result = ez_u128_mul(result, ten);
        ez_u128 digit = {(uint64_t)(*s - '0'), 0};
        result = ez_u128_add(result, digit);
        s++;
    }
    return result;
}

static inline ez_i128 ez_i128_from_decimal(const char *s) {
    bool neg = false;
    if (*s == '-') { neg = true; s++; }
    ez_u128 u = ez_u128_from_decimal(s);
    ez_i128 r;
    r.lo = u.lo;
    r.hi = (int64_t)u.hi;
    return neg ? ez_i128_neg(r) : r;
}

static inline ez_u256 ez_u256_from_decimal(const char *s) {
    ez_u256 result = EZ_U256_ZERO;
    ez_u256 ten = EZ_U256_ZERO;
    ten.w[0] = 10;
    while (*s) {
        if (*s == '_') { s++; continue; }
        result = ez_u256_mul(result, ten);
        ez_u256 digit = EZ_U256_ZERO;
        digit.w[0] = (uint64_t)(*s - '0');
        result = ez_u256_add(result, digit);
        s++;
    }
    return result;
}

static inline ez_i256 ez_i256_from_decimal(const char *s) {
    bool neg = false;
    if (*s == '-') { neg = true; s++; }
    ez_u256 u = ez_u256_from_decimal(s);
    ez_i256 r;
    memcpy(&r, &u, sizeof(r));
    return neg ? ez_i256_neg(r) : r;
}

/* --- Overflow-Checked Arithmetic --- */

static inline ez_i128 ez_i128_add_checked(ez_i128 a, ez_i128 b, const char *file, int line) {
    ez_i128 r = ez_i128_add(a, b);
    if ((a.hi >= 0 && b.hi >= 0 && r.hi < 0) || (a.hi < 0 && b.hi < 0 && r.hi >= 0)) {
        ez_panic_code("P0021", "i128 addition result is too large; value exceeds the range of i128");
    }
    return r;
}

static inline ez_i128 ez_i128_sub_checked(ez_i128 a, ez_i128 b, const char *file, int line) {
    ez_i128 r = ez_i128_sub(a, b);
    if ((a.hi >= 0 && b.hi < 0 && r.hi < 0) || (a.hi < 0 && b.hi >= 0 && r.hi >= 0)) {
        ez_panic_code("P0022", "i128 subtraction result is too large; value exceeds the range of i128");
    }
    return r;
}

static inline ez_i128 ez_i128_mul_checked(ez_i128 a, ez_i128 b, const char *file, int line) {
    ez_i128 r = ez_i128_mul(a, b);
    bool a_zero = (a.hi == 0 && a.lo == 0);
    bool b_zero = (b.hi == 0 && b.lo == 0);
    if (!a_zero && !b_zero) {
        ez_i128 check = ez_i128_div(r, b);
        if (!ez_i128_eq(check, a)) {
            ez_panic_code("P0023", "i128 multiplication result is too large; value exceeds the range of i128");
        }
    }
    return r;
}

static inline ez_u128 ez_u128_add_checked(ez_u128 a, ez_u128 b, const char *file, int line) {
    ez_u128 r = ez_u128_add(a, b);
    if (ez_u128_lt(r, a)) {
        ez_panic_code("P0024", "u128 addition result is too large; value exceeds the range of u128");
    }
    return r;
}

static inline ez_u128 ez_u128_sub_checked(ez_u128 a, ez_u128 b, const char *file, int line) {
    if (ez_u128_lt(a, b)) {
        ez_panic_code("P0025", "u128 subtraction result is negative, but u128 cannot hold negative values");
    }
    return ez_u128_sub(a, b);
}

static inline ez_u128 ez_u128_mul_checked(ez_u128 a, ez_u128 b, const char *file, int line) {
    ez_u128 r = ez_u128_mul(a, b);
    bool a_zero = (a.hi == 0 && a.lo == 0);
    bool b_zero = (b.hi == 0 && b.lo == 0);
    if (!a_zero && !b_zero) {
        ez_u128 check = ez_u128_div(r, b);
        if (!ez_u128_eq(check, a)) {
            ez_panic_code("P0026", "u128 multiplication result is too large; value exceeds the range of u128");
        }
    }
    return r;
}

static inline ez_i256 ez_i256_add_checked(ez_i256 a, ez_i256 b, const char *file, int line) {
    ez_i256 r = ez_i256_add(a, b);
    bool a_neg = ez_i256_is_neg(a);
    bool b_neg = ez_i256_is_neg(b);
    bool r_neg = ez_i256_is_neg(r);
    if ((!a_neg && !b_neg && r_neg) || (a_neg && b_neg && !r_neg)) {
        ez_panic_code("P0027", "i256 addition result is too large; value exceeds the range of i256");
    }
    return r;
}

static inline ez_i256 ez_i256_sub_checked(ez_i256 a, ez_i256 b, const char *file, int line) {
    ez_i256 r = ez_i256_sub(a, b);
    bool a_neg = ez_i256_is_neg(a);
    bool b_neg = ez_i256_is_neg(b);
    bool r_neg = ez_i256_is_neg(r);
    if ((!a_neg && b_neg && r_neg) || (a_neg && !b_neg && !r_neg)) {
        ez_panic_code("P0028", "i256 subtraction result is too large; value exceeds the range of i256");
    }
    return r;
}

static inline ez_i256 ez_i256_mul_checked(ez_i256 a, ez_i256 b, const char *file, int line) {
    ez_i256 r = ez_i256_mul(a, b);
    bool a_zero = (a.w[0] == 0 && a.w[1] == 0 && a.w[2] == 0 && a.w[3] == 0);
    bool b_zero = (b.w[0] == 0 && b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0);
    if (!a_zero && !b_zero) {
        ez_i256 check = ez_i256_div(r, b);
        if (!ez_i256_eq(check, a)) {
            ez_panic_code("P0029", "i256 multiplication result is too large; value exceeds the range of i256");
        }
    }
    return r;
}

static inline ez_u256 ez_u256_add_checked(ez_u256 a, ez_u256 b, const char *file, int line) {
    ez_u256 r = ez_u256_add(a, b);
    if (ez_u256_lt(r, a)) {
        ez_panic_code("P0030", "u256 addition result is too large; value exceeds the range of u256");
    }
    return r;
}

static inline ez_u256 ez_u256_sub_checked(ez_u256 a, ez_u256 b, const char *file, int line) {
    if (ez_u256_lt(a, b)) {
        ez_panic_code("P0031", "u256 subtraction result is negative, but u256 cannot hold negative values");
    }
    return ez_u256_sub(a, b);
}

static inline ez_u256 ez_u256_mul_checked(ez_u256 a, ez_u256 b, const char *file, int line) {
    ez_u256 r = ez_u256_mul(a, b);
    bool a_zero = (a.w[0] == 0 && a.w[1] == 0 && a.w[2] == 0 && a.w[3] == 0);
    bool b_zero = (b.w[0] == 0 && b.w[1] == 0 && b.w[2] == 0 && b.w[3] == 0);
    if (!a_zero && !b_zero) {
        ez_u256 check = ez_u256_div(r, b);
        if (!ez_u256_eq(check, a)) {
            ez_panic_code("P0032", "u256 multiplication result is too large; value exceeds the range of u256");
        }
    }
    return r;
}

/* --- Printing (to_string) --- */

/* Helper: convert unsigned 128-bit to decimal string */
static inline EzString ez_u128_to_string(EzArena *arena, ez_u128 val) {
    if (val.hi == 0) {
        char buf[U128_MAX_DIGITS];
        snprintf(buf, sizeof(buf), "%" PRIu64, val.lo);
        size_t len = strlen(buf);
        char *s = (char *)ez_arena_alloc(arena, len + 1);
        memcpy(s, buf, len + 1);
        return (EzString){ s, (int32_t)len };
    }
    /* For large values, extract digits by repeated division */
    char buf[I256_MAX_DIGITS];
    int pos = I256_MAX_DIGITS - 1;
    buf[pos] = '\0';
    ez_u128 ten = { 10, 0 };
    ez_u128 v = val;
    while (v.hi != 0 || v.lo != 0) {
        ez_u128 q, rem;
        ez_u128_divmod(v, ten, &q, &rem);
        buf[--pos] = '0' + (char)rem.lo;
        v = q;
    }
    if (pos == I256_MAX_DIGITS - 1) buf[--pos] = '0';
    size_t len = (I256_MAX_DIGITS - 1) - pos;
    char *s = (char *)ez_arena_alloc(arena, len + 1);
    memcpy(s, buf + pos, len + 1);
    return (EzString){ s, (int32_t)len };
}

static inline EzString ez_i128_to_string(EzArena *arena, ez_i128 val) {
    if (val.hi < 0) {
        ez_i128 neg = ez_i128_neg(val);
        ez_u128 uval;
        uval.lo = neg.lo; uval.hi = (uint64_t)neg.hi;
        EzString digits = ez_u128_to_string(arena, uval);
        char *s = (char *)ez_arena_alloc(arena, digits.len + 2);
        s[0] = '-';
        memcpy(s + 1, digits.data, digits.len + 1);
        return (EzString){ s, digits.len + 1 };
    }
    ez_u128 uval;
    uval.lo = val.lo; uval.hi = (uint64_t)val.hi;
    return ez_u128_to_string(arena, uval);
}

/* Helper: convert unsigned 256-bit to decimal string */
static inline EzString ez_u256_to_string(EzArena *arena, ez_u256 val) {
    if (val.w[1] == 0 && val.w[2] == 0 && val.w[3] == 0) {
        char buf[U128_MAX_DIGITS];
        snprintf(buf, sizeof(buf), "%" PRIu64, val.w[0]);
        size_t len = strlen(buf);
        char *s = (char *)ez_arena_alloc(arena, len + 1);
        memcpy(s, buf, len + 1);
        return (EzString){ s, (int32_t)len };
    }
    char buf[U256_MAX_DIGITS];
    int pos = U256_MAX_DIGITS - 1;
    buf[pos] = '\0';
    ez_u256 ten = EZ_U256_ZERO;
    ten.w[0] = 10;
    ez_u256 v = val;
    while (v.w[0] != 0 || v.w[1] != 0 || v.w[2] != 0 || v.w[3] != 0) {
        ez_u256 q, rem;
        ez_u256_divmod(v, ten, &q, &rem);
        buf[--pos] = '0' + (char)rem.w[0];
        v = q;
    }
    if (pos == U256_MAX_DIGITS - 1) buf[--pos] = '0';
    size_t len = (U256_MAX_DIGITS - 1) - pos;
    char *s = (char *)ez_arena_alloc(arena, len + 1);
    memcpy(s, buf + pos, len + 1);
    return (EzString){ s, (int32_t)len };
}

static inline EzString ez_i256_to_string(EzArena *arena, ez_i256 val) {
    if (ez_i256_is_neg(val)) {
        ez_i256 neg = ez_i256_neg(val);
        ez_u256 uval; memcpy(&uval, &neg, sizeof(uval));
        EzString digits = ez_u256_to_string(arena, uval);
        char *s = (char *)ez_arena_alloc(arena, digits.len + 2);
        s[0] = '-';
        memcpy(s + 1, digits.data, digits.len + 1);
        return (EzString){ s, digits.len + 1 };
    }
    ez_u256 uval; memcpy(&uval, &val, sizeof(uval));
    return ez_u256_to_string(arena, uval);
}

#endif /* EZ_BIGINT_H */
