/*
 * ez_atomic.h - @atomic module for EZ
 *
 * Lock-free atomic operations backed by hand-written assembly
 * (ARM64 and x86_64).
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_ATOMIC_MOD_H
#define EZ_ATOMIC_MOD_H

#include "../runtime/ez_atomic.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* int32_t* lock value */
} EzSpinLock;

/* 64-bit atomics */
int64_t ez_atomic_mod_load(int64_t *ptr);
void    ez_atomic_mod_store(int64_t *ptr, int64_t val);
int64_t ez_atomic_mod_add(int64_t *ptr, int64_t val);
int64_t ez_atomic_mod_sub(int64_t *ptr, int64_t val);
int64_t ez_atomic_mod_exchange(int64_t *ptr, int64_t val);
bool    ez_atomic_mod_cas(int64_t *ptr, int64_t expected, int64_t desired);
int64_t ez_atomic_mod_and(int64_t *ptr, int64_t val);
int64_t ez_atomic_mod_or(int64_t *ptr, int64_t val);
int64_t ez_atomic_mod_xor(int64_t *ptr, int64_t val);

/* Spinlock */
EzSpinLock ez_atomic_mod_spinlock(void);
void       ez_atomic_mod_spinlock_destroy(EzSpinLock lk);
void       ez_atomic_mod_spin_lock(EzSpinLock lk);
bool       ez_atomic_mod_spin_trylock(EzSpinLock lk);
void       ez_atomic_mod_spin_unlock(EzSpinLock lk);

/* Memory barrier */
void ez_atomic_mod_fence(void);

#endif
