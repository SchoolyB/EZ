# atomic.s — x86_64 lock-free atomic primitives for the Grayscale runtime.
# Provides load, store, add, sub, CAS, exchange, bitwise, fence, and spinlock operations
# at 64-bit, 32-bit, and 8-bit widths using sequential consistency (mfence/lock-prefixed instructions).
#
# Author:  Marshall A Burns (@SchoolyB)
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.
#
# Calling convention: System V AMD64 ABI
#   arg1 = %rdi  (pointer to atomic value)
#   arg2 = %rsi  (value / expected)
#   arg3 = %rdx  (desired, for CAS)
#   return = %rax

.text

# ─────────────────────────────────────────────
# int64_t gray_atomic_load(int64_t *ptr)
# ─────────────────────────────────────────────
.globl _gray_atomic_load
.globl gray_atomic_load
_gray_atomic_load:
gray_atomic_load:
    mfence
    movq    (%rdi), %rax
    mfence
    ret

# ─────────────────────────────────────────────
# void gray_atomic_store(int64_t *ptr, int64_t val)
# ─────────────────────────────────────────────
.globl _gray_atomic_store
.globl gray_atomic_store
_gray_atomic_store:
gray_atomic_store:
    mfence
    movq    %rsi, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_add(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_add
.globl gray_atomic_add
_gray_atomic_add:
gray_atomic_add:
    movq    %rsi, %rax
    lock xaddq %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_sub(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_sub
.globl gray_atomic_sub
_gray_atomic_sub:
gray_atomic_sub:
    negq    %rsi
    movq    %rsi, %rax
    lock xaddq %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# bool gray_atomic_cas(int64_t *ptr, int64_t expected, int64_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _gray_atomic_cas
.globl gray_atomic_cas
_gray_atomic_cas:
gray_atomic_cas:
    movq    %rsi, %rax          # rax = expected
    lock cmpxchgq %rdx, (%rdi)  # if (*ptr == expected) *ptr = desired
    sete    %al                  # al = 1 if swap succeeded
    movzbq  %al, %rax            # zero-extend to 64-bit
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_exchange(int64_t *ptr, int64_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _gray_atomic_exchange
.globl gray_atomic_exchange
_gray_atomic_exchange:
gray_atomic_exchange:
    movq    %rsi, %rax
    xchgq   %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_and(int64_t *ptr, int64_t val)
# Atomic bitwise AND. Returns previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_and
.globl gray_atomic_and
_gray_atomic_and:
gray_atomic_and:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    andq    %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_or(int64_t *ptr, int64_t val)
# Atomic bitwise OR. Returns previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_or
.globl gray_atomic_or
_gray_atomic_or:
gray_atomic_or:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    orq     %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# int64_t gray_atomic_xor(int64_t *ptr, int64_t val)
# Atomic bitwise XOR. Returns previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_xor
.globl gray_atomic_xor
_gray_atomic_xor:
gray_atomic_xor:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    xorq    %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# void gray_atomic_fence(void)
# Full memory barrier.
# ─────────────────────────────────────────────
.globl _gray_atomic_fence
.globl gray_atomic_fence
_gray_atomic_fence:
gray_atomic_fence:
    mfence
    ret

# =============================================================================
# 32-bit Atomic Operations
# =============================================================================

# ─────────────────────────────────────────────
# int32_t gray_atomic_load32(int32_t *ptr)
# ─────────────────────────────────────────────
.globl _gray_atomic_load32
.globl gray_atomic_load32
_gray_atomic_load32:
gray_atomic_load32:
    mfence
    movl    (%rdi), %eax
    mfence
    ret

# ─────────────────────────────────────────────
# void gray_atomic_store32(int32_t *ptr, int32_t val)
# ─────────────────────────────────────────────
.globl _gray_atomic_store32
.globl gray_atomic_store32
_gray_atomic_store32:
gray_atomic_store32:
    mfence
    movl    %esi, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# int32_t gray_atomic_add32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_add32
.globl gray_atomic_add32
_gray_atomic_add32:
gray_atomic_add32:
    movl    %esi, %eax
    lock xaddl %eax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int32_t gray_atomic_sub32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _gray_atomic_sub32
.globl gray_atomic_sub32
_gray_atomic_sub32:
gray_atomic_sub32:
    negl    %esi
    movl    %esi, %eax
    lock xaddl %eax, (%rdi)
    ret

# ─────────────────────────────────────────────
# bool gray_atomic_cas32(int32_t *ptr, int32_t expected, int32_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _gray_atomic_cas32
.globl gray_atomic_cas32
_gray_atomic_cas32:
gray_atomic_cas32:
    movl    %esi, %eax
    lock cmpxchgl %edx, (%rdi)
    sete    %al
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# int32_t gray_atomic_exchange32(int32_t *ptr, int32_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _gray_atomic_exchange32
.globl gray_atomic_exchange32
_gray_atomic_exchange32:
gray_atomic_exchange32:
    movl    %esi, %eax
    xchgl   %eax, (%rdi)
    ret

# =============================================================================
# 8-bit Atomic Operations
# =============================================================================

# ─────────────────────────────────────────────
# uint8_t gray_atomic_load8(uint8_t *ptr)
# ─────────────────────────────────────────────
.globl _gray_atomic_load8
.globl gray_atomic_load8
_gray_atomic_load8:
gray_atomic_load8:
    mfence
    movzbl  (%rdi), %eax
    mfence
    ret

# ─────────────────────────────────────────────
# void gray_atomic_store8(uint8_t *ptr, uint8_t val)
# ─────────────────────────────────────────────
.globl _gray_atomic_store8
.globl gray_atomic_store8
_gray_atomic_store8:
gray_atomic_store8:
    mfence
    movb    %sil, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# bool gray_atomic_cas8(uint8_t *ptr, uint8_t expected, uint8_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _gray_atomic_cas8
.globl gray_atomic_cas8
_gray_atomic_cas8:
gray_atomic_cas8:
    movb    %sil, %al
    lock cmpxchgb %dl, (%rdi)
    sete    %al
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# uint8_t gray_atomic_exchange8(uint8_t *ptr, uint8_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _gray_atomic_exchange8
.globl gray_atomic_exchange8
_gray_atomic_exchange8:
gray_atomic_exchange8:
    movb    %sil, %al
    xchgb   %al, (%rdi)
    movzbl  %al, %eax
    ret

# =============================================================================
# Spinlock Primitives
# =============================================================================

# ─────────────────────────────────────────────
# void gray_spin_lock(int32_t *lock)
# Acquire spinlock. Spins until lock is obtained.
# ─────────────────────────────────────────────
.globl _gray_spin_lock
.globl gray_spin_lock
_gray_spin_lock:
gray_spin_lock:
1:
    movl    $1, %eax
    xchgl   %eax, (%rdi)        # atomically swap 1 into lock
    testl   %eax, %eax           # was it 0 (unlocked)?
    jnz     2f                   # if not, spin
    ret
2:
    pause                        # hint: we're spinning
    cmpl    $0, (%rdi)           # test-and-test-and-set: check without locking bus
    jne     2b
    jmp     1b

# ─────────────────────────────────────────────
# bool gray_spin_trylock(int32_t *lock)
# Try to acquire spinlock. Returns 1 if acquired, 0 if not.
# ─────────────────────────────────────────────
.globl _gray_spin_trylock
.globl gray_spin_trylock
_gray_spin_trylock:
gray_spin_trylock:
    movl    $1, %eax
    xchgl   %eax, (%rdi)        # swap 1 in, get old value
    testl   %eax, %eax           # was it 0 (unlocked)?
    sete    %al                  # al = 1 if we got the lock
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# void gray_spin_unlock(int32_t *lock)
# Release spinlock.
# ─────────────────────────────────────────────
.globl _gray_spin_unlock
.globl gray_spin_unlock
_gray_spin_unlock:
gray_spin_unlock:
    mfence
    movl    $0, (%rdi)
    ret
