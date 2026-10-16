/*
 * atomic.h — Low-level atomic operation declarations for the
 * Grayscale runtime. Backed by hand-written assembly (ARM64 and
 * x86_64) providing lock-free 64-bit, 32-bit, and 8-bit primitives.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_ATOMIC_H
#define GRAY_ATOMIC_H

#include <stdint.h>
#include <stdbool.h>

/* TinyCC has no __atomic_* builtins of its own. Where it ships stdatomic.h,
 * that header supplies the __atomic_*_n forms and the __ATOMIC_* orders the
 * runtime is written against. */
#if defined(__TINYC__)
#  if __has_include(<stdatomic.h>)
#include <stdatomic.h>
/* ...except the exchange, which stdatomic.h leaves out. */
#define __atomic_exchange_n(pointer, value, order) \
    ({ __typeof__(*(pointer)) exchange_new_ = (value), exchange_old_; \
       __atomic_exchange((pointer), &exchange_new_, &exchange_old_, (order)); \
       exchange_old_; })
#  else
/* Older TinyCC snapshots (Ubuntu's 0.9.27) have neither stdatomic.h nor the
 * builtins, so every read-modify-write runs under one process-wide lock. A
 * plain load of an aligned word is already atomic on the targets TinyCC
 * supports, which keeps the array-iteration check free of the lock. */
#define __ATOMIC_RELAXED 0
#define __ATOMIC_CONSUME 1
#define __ATOMIC_ACQUIRE 2
#define __ATOMIC_RELEASE 3
#define __ATOMIC_ACQ_REL 4
#define __ATOMIC_SEQ_CST 5

void gray_fallback_atomic_lock(void);
void gray_fallback_atomic_unlock(void);

#define __atomic_load_n(pointer, order) \
    (*(volatile __typeof__(*(pointer)) *)(pointer))

#define GRAY_FALLBACK_ATOMIC_UPDATE(pointer, new_value_expression, operand) \
    ({ __typeof__(*(pointer)) fallback_operand_ = (operand), fallback_old_; \
       gray_fallback_atomic_lock(); \
       fallback_old_ = *(volatile __typeof__(*(pointer)) *)(pointer); \
       *(volatile __typeof__(*(pointer)) *)(pointer) = (new_value_expression); \
       gray_fallback_atomic_unlock(); \
       fallback_old_; })

#define __atomic_store_n(pointer, value, order) \
    ((void)GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_operand_, value))
#define __atomic_exchange_n(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_operand_, value)
#define __atomic_fetch_add(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_old_ + fallback_operand_, value)
#define __atomic_fetch_sub(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_old_ - fallback_operand_, value)
#define __atomic_fetch_and(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_old_ & fallback_operand_, value)
#define __atomic_fetch_or(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_old_ | fallback_operand_, value)
#define __atomic_fetch_xor(pointer, value, order) \
    GRAY_FALLBACK_ATOMIC_UPDATE(pointer, fallback_old_ ^ fallback_operand_, value)

#define __atomic_compare_exchange_n(pointer, expected, desired, weak, success_order, failure_order) \
    ({ __typeof__(*(pointer)) fallback_desired_ = (desired); \
       bool fallback_swapped_; \
       gray_fallback_atomic_lock(); \
       fallback_swapped_ = *(volatile __typeof__(*(pointer)) *)(pointer) == *(expected); \
       if (fallback_swapped_) *(volatile __typeof__(*(pointer)) *)(pointer) = fallback_desired_; \
       else *(expected) = *(volatile __typeof__(*(pointer)) *)(pointer); \
       gray_fallback_atomic_unlock(); \
       fallback_swapped_; })

#define __atomic_thread_fence(order) \
    ((void)({ gray_fallback_atomic_lock(); gray_fallback_atomic_unlock(); 0; }))
#  endif
#endif

// ── 64-bit Atomics ──────────────────────────────────────────────

int64_t gray_atomic_load(int64_t *target);
void    gray_atomic_store(int64_t *target, int64_t value);
int64_t gray_atomic_add(int64_t *target, int64_t value);
int64_t gray_atomic_sub(int64_t *target, int64_t value);
int64_t gray_atomic_exchange(int64_t *target, int64_t value);
bool    gray_atomic_cas(int64_t *target, int64_t expected, int64_t desired);
int64_t gray_atomic_and(int64_t *target, int64_t value);
int64_t gray_atomic_or(int64_t *target, int64_t value);
int64_t gray_atomic_xor(int64_t *target, int64_t value);

// ── 32-bit Atomics ──────────────────────────────────────────────

int32_t gray_atomic_load32(int32_t *target);
void    gray_atomic_store32(int32_t *target, int32_t value);
int32_t gray_atomic_add32(int32_t *target, int32_t value);
int32_t gray_atomic_sub32(int32_t *target, int32_t value);
int32_t gray_atomic_exchange32(int32_t *target, int32_t value);
bool    gray_atomic_cas32(int32_t *target, int32_t expected, int32_t desired);

// ── 8-bit Atomics ───────────────────────────────────────────────

uint8_t gray_atomic_load8(uint8_t *target);
void    gray_atomic_store8(uint8_t *target, uint8_t value);
uint8_t gray_atomic_exchange8(uint8_t *target, uint8_t value);
bool    gray_atomic_cas8(uint8_t *target, uint8_t expected, uint8_t desired);

// ── Spinlock ────────────────────────────────────────────────────

void    gray_spin_lock(int32_t *lock);
bool    gray_spin_trylock(int32_t *lock);
void    gray_spin_unlock(int32_t *lock);

// ── Memory Barrier ──────────────────────────────────────────────

void    gray_atomic_fence(void);

#endif // GRAY_ATOMIC_H
