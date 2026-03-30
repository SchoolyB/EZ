# EZ Atomic Operations — ARM64 (AArch64)
# Lock-free primitives for systems programming.
# All operations use sequential consistency (dmb barriers)
# unless otherwise noted.
#
# Calling convention: AAPCS64
#   arg1 = x0  (pointer to atomic value)
#   arg2 = x1  (value / expected)
#   arg3 = x2  (desired, for CAS)
#   return = x0

.text

# ─────────────────────────────────────────────
# int64_t ez_atomic_load(int64_t *ptr)
# ─────────────────────────────────────────────
.globl _ez_atomic_load
.globl ez_atomic_load
_ez_atomic_load:
ez_atomic_load:
    dmb     ish
    ldr     x0, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store(int64_t *ptr, int64_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store
.globl ez_atomic_store
_ez_atomic_store:
ez_atomic_store:
    dmb     ish
    str     x1, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_add(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_add
.globl ez_atomic_add
_ez_atomic_add:
ez_atomic_add:
1:
    ldxr    x2, [x0]           // exclusive load old value
    add     x3, x2, x1         // new = old + val
    stlxr   w4, x3, [x0]       // exclusive store with release
    cbnz    w4, 1b              // retry if store failed
    dmb     ish
    mov     x0, x2              // return old value
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_sub(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_sub
.globl ez_atomic_sub
_ez_atomic_sub:
ez_atomic_sub:
1:
    ldxr    x2, [x0]           // exclusive load old value
    sub     x3, x2, x1         // new = old - val
    stlxr   w4, x3, [x0]       // exclusive store with release
    cbnz    w4, 1b              // retry if store failed
    dmb     ish
    mov     x0, x2              // return old value
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas(int64_t *ptr, int64_t expected, int64_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas
.globl ez_atomic_cas
_ez_atomic_cas:
ez_atomic_cas:
1:
    ldxr    x3, [x0]           // exclusive load current value
    cmp     x3, x1             // compare with expected
    b.ne    2f                  // if not equal, fail
    stlxr   w4, x2, [x0]       // try store desired
    cbnz    w4, 1b              // retry if exclusive store failed
    dmb     ish
    mov     x0, #1              // success
    ret
2:
    clrex                       // clear exclusive monitor
    mov     x0, #0              // failure
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_exchange(int64_t *ptr, int64_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange
.globl ez_atomic_exchange
_ez_atomic_exchange:
ez_atomic_exchange:
1:
    ldxr    x2, [x0]           // exclusive load old value
    stlxr   w3, x1, [x0]       // exclusive store new value
    cbnz    w3, 1b              // retry if store failed
    dmb     ish
    mov     x0, x2              // return old value
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_and(int64_t *ptr, int64_t val)
# Atomic bitwise AND. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_and
.globl ez_atomic_and
_ez_atomic_and:
ez_atomic_and:
1:
    ldxr    x2, [x0]
    and     x3, x2, x1
    stlxr   w4, x3, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     x0, x2
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_or(int64_t *ptr, int64_t val)
# Atomic bitwise OR. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_or
.globl ez_atomic_or
_ez_atomic_or:
ez_atomic_or:
1:
    ldxr    x2, [x0]
    orr     x3, x2, x1
    stlxr   w4, x3, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     x0, x2
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_xor(int64_t *ptr, int64_t val)
# Atomic bitwise XOR. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_xor
.globl ez_atomic_xor
_ez_atomic_xor:
ez_atomic_xor:
1:
    ldxr    x2, [x0]
    eor     x3, x2, x1
    stlxr   w4, x3, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     x0, x2
    ret

# ─────────────────────────────────────────────
# void ez_atomic_fence(void)
# Full memory barrier.
# ─────────────────────────────────────────────
.globl _ez_atomic_fence
.globl ez_atomic_fence
_ez_atomic_fence:
ez_atomic_fence:
    dmb     ish
    ret

# =============================================================================
# 32-bit Atomic Operations
# =============================================================================

# ─────────────────────────────────────────────
# int32_t ez_atomic_load32(int32_t *ptr)
# ─────────────────────────────────────────────
.globl _ez_atomic_load32
.globl ez_atomic_load32
_ez_atomic_load32:
ez_atomic_load32:
    dmb     ish
    ldr     w0, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store32(int32_t *ptr, int32_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store32
.globl ez_atomic_store32
_ez_atomic_store32:
ez_atomic_store32:
    dmb     ish
    str     w1, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_add32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_add32
.globl ez_atomic_add32
_ez_atomic_add32:
ez_atomic_add32:
1:
    ldxr    w2, [x0]
    add     w3, w2, w1
    stlxr   w4, w3, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     w0, w2
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_sub32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_sub32
.globl ez_atomic_sub32
_ez_atomic_sub32:
ez_atomic_sub32:
1:
    ldxr    w2, [x0]
    sub     w3, w2, w1
    stlxr   w4, w3, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     w0, w2
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas32(int32_t *ptr, int32_t expected, int32_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas32
.globl ez_atomic_cas32
_ez_atomic_cas32:
ez_atomic_cas32:
1:
    ldxr    w3, [x0]
    cmp     w3, w1
    b.ne    2f
    stlxr   w4, w2, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     w0, #1
    ret
2:
    clrex
    mov     w0, #0
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_exchange32(int32_t *ptr, int32_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange32
.globl ez_atomic_exchange32
_ez_atomic_exchange32:
ez_atomic_exchange32:
1:
    ldxr    w2, [x0]
    stlxr   w3, w1, [x0]
    cbnz    w3, 1b
    dmb     ish
    mov     w0, w2
    ret

# =============================================================================
# 8-bit Atomic Operations
# =============================================================================

# ─────────────────────────────────────────────
# uint8_t ez_atomic_load8(uint8_t *ptr)
# ─────────────────────────────────────────────
.globl _ez_atomic_load8
.globl ez_atomic_load8
_ez_atomic_load8:
ez_atomic_load8:
    dmb     ish
    ldrb    w0, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store8(uint8_t *ptr, uint8_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store8
.globl ez_atomic_store8
_ez_atomic_store8:
ez_atomic_store8:
    dmb     ish
    strb    w1, [x0]
    dmb     ish
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas8(uint8_t *ptr, uint8_t expected, uint8_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas8
.globl ez_atomic_cas8
_ez_atomic_cas8:
ez_atomic_cas8:
1:
    ldxrb   w3, [x0]
    cmp     w3, w1
    b.ne    2f
    stlxrb  w4, w2, [x0]
    cbnz    w4, 1b
    dmb     ish
    mov     w0, #1
    ret
2:
    clrex
    mov     w0, #0
    ret

# ─────────────────────────────────────────────
# uint8_t ez_atomic_exchange8(uint8_t *ptr, uint8_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange8
.globl ez_atomic_exchange8
_ez_atomic_exchange8:
ez_atomic_exchange8:
1:
    ldxrb   w2, [x0]
    stlxrb  w3, w1, [x0]
    cbnz    w3, 1b
    dmb     ish
    mov     w0, w2
    ret

# =============================================================================
# Spinlock Primitives
# =============================================================================

# ─────────────────────────────────────────────
# void ez_spin_lock(int32_t *lock)
# Acquire spinlock. Spins until lock is obtained.
# ─────────────────────────────────────────────
.globl _ez_spin_lock
.globl ez_spin_lock
_ez_spin_lock:
ez_spin_lock:
1:
    ldxr    w1, [x0]
    cbnz    w1, 2f              // if locked, spin
    mov     w2, #1
    stxr    w3, w2, [x0]
    cbnz    w3, 1b              // retry if store failed
    dmb     ish
    ret
2:
    yield                       // hint: we're spinning
    ldr     w1, [x0]           // test without exclusive (TTAS)
    cbnz    w1, 2b
    b       1b

# ─────────────────────────────────────────────
# bool ez_spin_trylock(int32_t *lock)
# Try to acquire spinlock. Returns 1 if acquired, 0 if not.
# ─────────────────────────────────────────────
.globl _ez_spin_trylock
.globl ez_spin_trylock
_ez_spin_trylock:
ez_spin_trylock:
    ldxr    w1, [x0]
    cbnz    w1, 1f              // already locked
    mov     w2, #1
    stxr    w3, w2, [x0]
    cbnz    w3, 1f              // store failed
    dmb     ish
    mov     w0, #1              // acquired
    ret
1:
    clrex
    mov     w0, #0              // not acquired
    ret

# ─────────────────────────────────────────────
# void ez_spin_unlock(int32_t *lock)
# Release spinlock.
# ─────────────────────────────────────────────
.globl _ez_spin_unlock
.globl ez_spin_unlock
_ez_spin_unlock:
ez_spin_unlock:
    dmb     ish
    str     wzr, [x0]
    ret
