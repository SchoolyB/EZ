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

// ── 64-bit Atomics ──────────────────────────────────────────────

int64_t gray_atomic_load(int64_t *ptr);
void    gray_atomic_store(int64_t *ptr, int64_t val);
int64_t gray_atomic_add(int64_t *ptr, int64_t val);
int64_t gray_atomic_sub(int64_t *ptr, int64_t val);
int64_t gray_atomic_exchange(int64_t *ptr, int64_t val);
bool    gray_atomic_cas(int64_t *ptr, int64_t expected, int64_t desired);
int64_t gray_atomic_and(int64_t *ptr, int64_t val);
int64_t gray_atomic_or(int64_t *ptr, int64_t val);
int64_t gray_atomic_xor(int64_t *ptr, int64_t val);

// ── 32-bit Atomics ──────────────────────────────────────────────

int32_t gray_atomic_load32(int32_t *ptr);
void    gray_atomic_store32(int32_t *ptr, int32_t val);
int32_t gray_atomic_add32(int32_t *ptr, int32_t val);
int32_t gray_atomic_sub32(int32_t *ptr, int32_t val);
int32_t gray_atomic_exchange32(int32_t *ptr, int32_t val);
bool    gray_atomic_cas32(int32_t *ptr, int32_t expected, int32_t desired);

// ── 8-bit Atomics ───────────────────────────────────────────────

uint8_t gray_atomic_load8(uint8_t *ptr);
void    gray_atomic_store8(uint8_t *ptr, uint8_t val);
uint8_t gray_atomic_exchange8(uint8_t *ptr, uint8_t val);
bool    gray_atomic_cas8(uint8_t *ptr, uint8_t expected, uint8_t desired);

// ── Spinlock ────────────────────────────────────────────────────

void    gray_spin_lock(int32_t *lock);
bool    gray_spin_trylock(int32_t *lock);
void    gray_spin_unlock(int32_t *lock);

// ── Memory Barrier ──────────────────────────────────────────────

void    gray_atomic_fence(void);

#endif // GRAY_ATOMIC_H
