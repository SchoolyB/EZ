/*
 * gray_atomic.c - @atomic module implementation
 *
 * Thin wrappers around the runtime assembly-backed atomic primitives.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_atomic_mod.h"
#include <stdlib.h>

/* 64-bit atomics */

int64_t gray_atomic_mod_load(int64_t *ptr) { return gray_atomic_load(ptr); }
void    gray_atomic_mod_store(int64_t *ptr, int64_t val) { gray_atomic_store(ptr, val); }
int64_t gray_atomic_mod_add(int64_t *ptr, int64_t val) { return gray_atomic_add(ptr, val); }
int64_t gray_atomic_mod_sub(int64_t *ptr, int64_t val) { return gray_atomic_sub(ptr, val); }
int64_t gray_atomic_mod_exchange(int64_t *ptr, int64_t val) { return gray_atomic_exchange(ptr, val); }
bool    gray_atomic_mod_cas(int64_t *ptr, int64_t expected, int64_t desired) { return gray_atomic_cas(ptr, expected, desired); }
int64_t gray_atomic_mod_and(int64_t *ptr, int64_t val) { return gray_atomic_and(ptr, val); }
int64_t gray_atomic_mod_or(int64_t *ptr, int64_t val) { return gray_atomic_or(ptr, val); }
int64_t gray_atomic_mod_xor(int64_t *ptr, int64_t val) { return gray_atomic_xor(ptr, val); }

/* Spinlock */

EzSpinLock gray_atomic_mod_spinlock(void) {
    int32_t *lock = malloc(sizeof(int32_t));
    *lock = 0;
    EzSpinLock result;
    result._internal = lock;
    return result;
}

void gray_atomic_mod_spinlock_destroy(EzSpinLock lk) {
    if (lk._internal) {
        free(lk._internal);
    }
}

void gray_atomic_mod_spin_lock(EzSpinLock lk) {
    if (lk._internal) {
        gray_spin_lock((int32_t *)lk._internal);
    }
}

bool gray_atomic_mod_spin_trylock(EzSpinLock lk) {
    if (lk._internal) {
        return gray_spin_trylock((int32_t *)lk._internal);
    }
    return false;
}

void gray_atomic_mod_spin_unlock(EzSpinLock lk) {
    if (lk._internal) {
        gray_spin_unlock((int32_t *)lk._internal);
    }
}

/* Memory barrier */

void gray_atomic_mod_fence(void) { gray_atomic_fence(); }
