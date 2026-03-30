# EZ Atomic Operations — x86_64
# Lock-free primitives for systems programming.
# All operations use sequential consistency (full memory barriers)
# unless otherwise noted.
#
# Calling convention: System V AMD64 ABI
#   arg1 = %rdi  (pointer to atomic value)
#   arg2 = %rsi  (value / expected)
#   arg3 = %rdx  (desired, for CAS)
#   return = %rax

.text

# ─────────────────────────────────────────────
# int64_t ez_atomic_load(int64_t *ptr)
# ─────────────────────────────────────────────
.globl _ez_atomic_load
.globl ez_atomic_load
_ez_atomic_load:
ez_atomic_load:
    mfence
    movq    (%rdi), %rax
    mfence
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store(int64_t *ptr, int64_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store
.globl ez_atomic_store
_ez_atomic_store:
ez_atomic_store:
    mfence
    movq    %rsi, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_add(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_add
.globl ez_atomic_add
_ez_atomic_add:
ez_atomic_add:
    movq    %rsi, %rax
    lock xaddq %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_sub(int64_t *ptr, int64_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_sub
.globl ez_atomic_sub
_ez_atomic_sub:
ez_atomic_sub:
    negq    %rsi
    movq    %rsi, %rax
    lock xaddq %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas(int64_t *ptr, int64_t expected, int64_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas
.globl ez_atomic_cas
_ez_atomic_cas:
ez_atomic_cas:
    movq    %rsi, %rax          # rax = expected
    lock cmpxchgq %rdx, (%rdi)  # if (*ptr == expected) *ptr = desired
    sete    %al                  # al = 1 if swap succeeded
    movzbq  %al, %rax            # zero-extend to 64-bit
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_exchange(int64_t *ptr, int64_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange
.globl ez_atomic_exchange
_ez_atomic_exchange:
ez_atomic_exchange:
    movq    %rsi, %rax
    xchgq   %rax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_and(int64_t *ptr, int64_t val)
# Atomic bitwise AND. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_and
.globl ez_atomic_and
_ez_atomic_and:
ez_atomic_and:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    andq    %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_or(int64_t *ptr, int64_t val)
# Atomic bitwise OR. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_or
.globl ez_atomic_or
_ez_atomic_or:
ez_atomic_or:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    orq     %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# int64_t ez_atomic_xor(int64_t *ptr, int64_t val)
# Atomic bitwise XOR. Returns previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_xor
.globl ez_atomic_xor
_ez_atomic_xor:
ez_atomic_xor:
    movq    (%rdi), %rax
1:
    movq    %rax, %rcx
    xorq    %rsi, %rcx
    lock cmpxchgq %rcx, (%rdi)
    jne     1b
    ret

# ─────────────────────────────────────────────
# void ez_atomic_fence(void)
# Full memory barrier.
# ─────────────────────────────────────────────
.globl _ez_atomic_fence
.globl ez_atomic_fence
_ez_atomic_fence:
ez_atomic_fence:
    mfence
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
    mfence
    movl    (%rdi), %eax
    mfence
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store32(int32_t *ptr, int32_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store32
.globl ez_atomic_store32
_ez_atomic_store32:
ez_atomic_store32:
    mfence
    movl    %esi, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_add32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_add32
.globl ez_atomic_add32
_ez_atomic_add32:
ez_atomic_add32:
    movl    %esi, %eax
    lock xaddl %eax, (%rdi)
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_sub32(int32_t *ptr, int32_t val)
# Returns the previous value.
# ─────────────────────────────────────────────
.globl _ez_atomic_sub32
.globl ez_atomic_sub32
_ez_atomic_sub32:
ez_atomic_sub32:
    negl    %esi
    movl    %esi, %eax
    lock xaddl %eax, (%rdi)
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas32(int32_t *ptr, int32_t expected, int32_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas32
.globl ez_atomic_cas32
_ez_atomic_cas32:
ez_atomic_cas32:
    movl    %esi, %eax
    lock cmpxchgl %edx, (%rdi)
    sete    %al
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# int32_t ez_atomic_exchange32(int32_t *ptr, int32_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange32
.globl ez_atomic_exchange32
_ez_atomic_exchange32:
ez_atomic_exchange32:
    movl    %esi, %eax
    xchgl   %eax, (%rdi)
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
    mfence
    movzbl  (%rdi), %eax
    mfence
    ret

# ─────────────────────────────────────────────
# void ez_atomic_store8(uint8_t *ptr, uint8_t val)
# ─────────────────────────────────────────────
.globl _ez_atomic_store8
.globl ez_atomic_store8
_ez_atomic_store8:
ez_atomic_store8:
    mfence
    movb    %sil, (%rdi)
    mfence
    ret

# ─────────────────────────────────────────────
# bool ez_atomic_cas8(uint8_t *ptr, uint8_t expected, uint8_t desired)
# Compare-and-swap. Returns 1 on success, 0 on failure.
# ─────────────────────────────────────────────
.globl _ez_atomic_cas8
.globl ez_atomic_cas8
_ez_atomic_cas8:
ez_atomic_cas8:
    movb    %sil, %al
    lock cmpxchgb %dl, (%rdi)
    sete    %al
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# uint8_t ez_atomic_exchange8(uint8_t *ptr, uint8_t val)
# Atomically swap *ptr with val. Returns old value.
# ─────────────────────────────────────────────
.globl _ez_atomic_exchange8
.globl ez_atomic_exchange8
_ez_atomic_exchange8:
ez_atomic_exchange8:
    movb    %sil, %al
    xchgb   %al, (%rdi)
    movzbl  %al, %eax
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
# bool ez_spin_trylock(int32_t *lock)
# Try to acquire spinlock. Returns 1 if acquired, 0 if not.
# ─────────────────────────────────────────────
.globl _ez_spin_trylock
.globl ez_spin_trylock
_ez_spin_trylock:
ez_spin_trylock:
    movl    $1, %eax
    xchgl   %eax, (%rdi)        # swap 1 in, get old value
    testl   %eax, %eax           # was it 0 (unlocked)?
    sete    %al                  # al = 1 if we got the lock
    movzbl  %al, %eax
    ret

# ─────────────────────────────────────────────
# void ez_spin_unlock(int32_t *lock)
# Release spinlock.
# ─────────────────────────────────────────────
.globl _ez_spin_unlock
.globl ez_spin_unlock
_ez_spin_unlock:
ez_spin_unlock:
    mfence
    movl    $0, (%rdi)
    ret
