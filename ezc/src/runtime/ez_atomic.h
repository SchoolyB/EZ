#ifndef EZ_ATOMIC_H
#define EZ_ATOMIC_H

#include <stdint.h>
#include <stdbool.h>

// ── 64-bit Atomics ──────────────────────────────────────────────

int64_t ez_atomic_load(int64_t *ptr);
void    ez_atomic_store(int64_t *ptr, int64_t val);
int64_t ez_atomic_add(int64_t *ptr, int64_t val);
int64_t ez_atomic_sub(int64_t *ptr, int64_t val);
int64_t ez_atomic_exchange(int64_t *ptr, int64_t val);
bool    ez_atomic_cas(int64_t *ptr, int64_t expected, int64_t desired);
int64_t ez_atomic_and(int64_t *ptr, int64_t val);
int64_t ez_atomic_or(int64_t *ptr, int64_t val);
int64_t ez_atomic_xor(int64_t *ptr, int64_t val);

// ── 32-bit Atomics ──────────────────────────────────────────────

int32_t ez_atomic_load32(int32_t *ptr);
void    ez_atomic_store32(int32_t *ptr, int32_t val);
int32_t ez_atomic_add32(int32_t *ptr, int32_t val);
int32_t ez_atomic_sub32(int32_t *ptr, int32_t val);
int32_t ez_atomic_exchange32(int32_t *ptr, int32_t val);
bool    ez_atomic_cas32(int32_t *ptr, int32_t expected, int32_t desired);

// ── 8-bit Atomics ───────────────────────────────────────────────

uint8_t ez_atomic_load8(uint8_t *ptr);
void    ez_atomic_store8(uint8_t *ptr, uint8_t val);
uint8_t ez_atomic_exchange8(uint8_t *ptr, uint8_t val);
bool    ez_atomic_cas8(uint8_t *ptr, uint8_t expected, uint8_t desired);

// ── Spinlock ────────────────────────────────────────────────────

void    ez_spin_lock(int32_t *lock);
bool    ez_spin_trylock(int32_t *lock);
void    ez_spin_unlock(int32_t *lock);

// ── Memory Barrier ──────────────────────────────────────────────

void    ez_atomic_fence(void);

#endif // EZ_ATOMIC_H
