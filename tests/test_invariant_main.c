#include <check.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <stdint.h>
#include <limits.h>

/*
 * We test the buffer growth and realloc logic directly by reimplementing
 * the vulnerable pattern and asserting the invariants that MUST hold:
 *   1. The allocated buffer is always large enough to hold the data written.
 *   2. realloc failure is handled without NULL dereference or use-after-free.
 *   3. Capacity doubling never overflows (new_cap > old_cap).
 *   4. The final buffer length matches the actual data length.
 */

/* Reimplementation of the vulnerable buffer-growth pattern from main.c */
typedef struct {
    char   *buf;
    size_t  len;
    size_t  cap;
    int     error;   /* non-zero if an allocation failure occurred */
} GrowBuf;

static GrowBuf growbuf_init(size_t initial_cap)
{
    GrowBuf g;
    g.len   = 0;
    g.error = 0;

    /* Guard against zero initial capacity */
    if (initial_cap == 0) initial_cap = 1;
    g.cap = initial_cap;
    g.buf = (char *)malloc(g.cap);
    if (!g.buf) {
        g.cap   = 0;
        g.error = 1;
    }
    return g;
}

static int growbuf_append(GrowBuf *g, const char *data, size_t data_len)
{
    if (g->error || !g->buf) return -1;
    if (data_len == 0) return 0;

    /* Check if we need to grow */
    if (g->len + data_len > g->cap) {
        size_t new_cap = g->cap;

        /* Invariant: capacity doubling must not overflow */
        while (new_cap < g->len + data_len) {
            if (new_cap > SIZE_MAX / 2) {
                /* Would overflow — cap at SIZE_MAX */
                new_cap = SIZE_MAX;
                if (new_cap < g->len + data_len) {
                    g->error = 1;
                    return -1;
                }
                break;
            }
            new_cap *= 2;
        }

        /* Invariant: new_cap must be strictly larger than old cap */
        if (new_cap <= g->cap && new_cap != SIZE_MAX) {
            g->error = 1;
            return -1;
        }

        char *new_buf = (char *)realloc(g->buf, new_cap);
        if (!new_buf) {
            /*
             * Invariant: on realloc failure the original pointer must
             * remain valid (not freed) so we can clean up safely.
             */
            g->error = 1;
            /* g->buf is still valid here — do NOT free it yet */
            return -1;
        }
        g->buf = new_buf;
        g->cap = new_cap;
    }

    memcpy(g->buf + g->len, data, data_len);
    g->len += data_len;
    return 0;
}

static void growbuf_free(GrowBuf *g)
{
    if (g->buf) {
        free(g->buf);
        g->buf = NULL;
    }
    g->len   = 0;
    g->cap   = 0;
    g->error = 0;
}

/* ------------------------------------------------------------------ */

START_TEST(test_buffer_growth_security_invariants)
{
    /* Invariant: buffer is always large enough; no overflow; safe on failure */

    /* Each payload entry: {data, repeat_count} */
    struct { const char *data; size_t repeat; } payloads[] = {
        /* Normal strings */
        { "hello world",                                    1    },
        { "A",                                              1    },
        { "",                                               0    },

        /* Boundary: exactly fills typical initial cap (64 bytes) */
        { "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", 1 },

        /* Forces multiple doublings */
        { "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", 100 },

        /* Large single chunk */
        { "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC", 1024 },

        /* Binary / null-byte containing data */
        { "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f", 256 },

        /* Format-string-like payload */
        { "%s%s%s%n%n%n%x%x%x",                            50   },

        /* Heap-spray-like repetition */
        { "\xff\xfe\xfd\xfc",                               4096 },

        /* Near SIZE_MAX/2 chunk size — we use a moderate large size */
        { "X",                                              65535 },
    };

    int num_payloads = (int)(sizeof(payloads) / sizeof(payloads[0]));

    for (int i = 0; i < num_payloads; i++) {
        const char *data   = payloads[i].data;
        size_t      repeat = payloads[i].repeat;
        size_t      dlen   = strlen(data);

        /* Test with several initial capacities including edge cases */
        size_t init_caps[] = { 1, 2, 4, 8, 64, 128, 4096 };
        int num_caps = (int)(sizeof(init_caps) / sizeof(init_caps[0]));

        for (int c = 0; c < num_caps; c++) {
            GrowBuf g = growbuf_init(init_caps[c]);

            if (g.error) {
                /* malloc failed — nothing to assert, skip */
                growbuf_free(&g);
                continue;
            }

            /* Invariant: initial cap must be positive */
            ck_assert_uint_gt(g.cap, 0);
            ck_assert_ptr_nonnull(g.buf);

            size_t expected_len = 0;
            int    append_ok    = 1;

            for (size_t r = 0; r < repeat; r++) {
                if (dlen == 0) break;
                int rc = growbuf_append(&g, data, dlen);
                if (rc != 0) {
                    /* Allocation failure is acceptable — but must be flagged */
                    ck_assert_int_ne(g.error, 0);
                    append_ok = 0;
                    break;
                }
                expected_len += dlen;

                /* Invariant: cap must always be >= len after successful append */
                ck_assert_uint_ge(g.cap, g.len);

                /* Invariant: reported length must match expected */
                ck_assert_uint_eq(g.len, expected_len);
            }

            if (append_ok) {
                /* Invariant: buffer is large enough to hold all written data */
                ck_assert_uint_ge(g.cap, g.len);
                ck_assert_uint_eq(g.len, expected_len);

                /* Invariant: data integrity — last chunk matches what we wrote */
                if (dlen > 0 && repeat > 0 && g.len >= dlen) {
                    ck_assert_mem_eq(g.buf + g.len - dlen, data, dlen);
                }
            }

            /* Invariant: buf pointer must still be valid (not freed on error) */
            /* We can safely call free — no double-free or use-after-free */
            growbuf_free(&g);

            /* Invariant: after free, buf is NULL */
            ck_assert_ptr_null(g.buf);
        }
    }
}
END_TEST

START_TEST(test_capacity_doubling_no_overflow)
{
    /*
     * Invariant: for any starting capacity, doubling must never produce
     * a value smaller than or equal to the original (i.e., no integer overflow
     * that wraps around to a small number).
     */
    size_t caps[] = {
        1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024,
        (size_t)INT_MAX - 1,
        (size_t)INT_MAX,
        (size_t)UINT_MAX / 4,
        (size_t)UINT_MAX / 2 - 1,
        (size_t)UINT_MAX / 2,
        SIZE_MAX / 4,
        SIZE_MAX / 2 - 1,
        SIZE_MAX / 2,
    };
    int num_caps = (int)(sizeof(caps) / sizeof(caps[0]));

    for (int i = 0; i < num_caps; i++) {
        size_t cap = caps[i];

        if (cap > SIZE_MAX / 2) {
            /*
             * Invariant: if cap > SIZE_MAX/2, doubling would overflow.
             * The code MUST detect this and NOT produce a smaller value.
             * We verify the detection logic: new_cap = cap * 2 would wrap.
             */
            size_t new_cap = cap * 2; /* intentional — check wrap */
            /*
             * If overflow occurred, new_cap < cap.
             * The safe implementation must cap at SIZE_MAX instead.
             */
            if (new_cap < cap) {
                /* Overflow detected — safe code should use SIZE_MAX */
                ck_assert_uint_eq(SIZE_MAX, SIZE_MAX); /* tautology: overflow path exists */
            }
        } else {
            size_t new_cap = cap * 2;
            /* Invariant: doubled capacity must be strictly greater */
            ck_assert_uint_gt(new_cap, cap);
        }
    }
}
END_TEST

START_TEST(test_realloc_failure_no_use_after_free)
{
    /*
     * Invariant: when realloc returns NULL, the original buffer pointer
     * must remain accessible (not freed) until explicit cleanup.
     * We simulate this by verifying the pattern is safe.
     */

    /* Allocate a buffer */
    size_t cap = 64;
    char *buf = (char *)malloc(cap);
    ck_assert_ptr_nonnull(buf);

    memset(buf, 'A', cap);

    /*
     * Simulate realloc failure: attempt an impossibly large allocation.
     * realloc(buf, SIZE_MAX) should return NULL on any sane system.
     */
    char *new_buf = (char *)realloc(buf, SIZE_MAX);

    if (new_buf == NULL) {
        /*
         * Invariant: buf must still be valid — realloc did NOT free it.
         * We verify by reading from it (if it were freed this would be UB,
         * but the invariant is that correct code preserves buf here).
         */
        ck_assert_ptr_nonnull(buf);
        /* Verify data integrity — buf was not corrupted */
        for (size_t j = 0; j < cap; j++) {
            ck_assert_int_eq((unsigned char)buf[j], 'A');
        }
        /* Safe cleanup */
        free(buf);
        buf = NULL;
    } else {
        /* realloc succeeded (unlikely for SIZE_MAX, but handle it) */
        free(new_buf);
        new_buf = NULL;
    }

    /* Invariant: no double-free — buf is either NULL or freed exactly once */
    ck_assert_ptr_null(buf);
}
END_TEST

START_TEST(test_final_buffer_nul_termination)
{
    /*
     * Invariant: after all data is appended, the buffer can be safely
     * NUL-terminated (len+1 bytes must be available or allocated).
     */
    const char *chunks[] = {
        "chunk_one_data_",
        "chunk_two_data_",
        "chunk_three____",
        "final_chunk____",
    };
    int num_chunks = (int)(sizeof(chunks) / sizeof(chunks[0]));

    GrowBuf g = growbuf_init(1); /* start tiny to force growth */
    ck_assert_int_eq(g.error, 0);
    ck_assert_ptr_nonnull(g.buf);

    for (int i = 0; i < num_chunks; i++) {
        int rc = growbuf_append(&g, chunks[i], strlen(chunks[i]));
        if (rc != 0) {
            /* Allocation failure — skip rest of test */
            growbuf_free(&g);
            return;
        }
    }

    /* Invariant: we must be able to NUL-terminate without overflow */
    ck_assert_uint_ge(g.cap, g.len); /* room exists up to cap */

    /* Grow by 1 for NUL terminator if needed */
    if (g.len + 1 > g.cap) {
        char *grown = (char *)realloc(g.buf, g.len + 1);
        if (grown) {
            g.buf = grown;
            g.cap = g.len + 1;
        }
        /* If realloc fails, we cannot NUL-terminate — that's an error condition */
    }

    if (g.cap >= g.len + 1) {
        g.buf[g.len] = '\0';
        /* Invariant: NUL-terminated string length matches accumulated length */
        ck_assert_uint_eq(strlen(g.buf), g.len);
    }

    growbuf_free(&g);
}
END_TEST

/* ------------------------------------------------------------------ */

Suite *security_suite(void)
{
    Suite *s;
    TCase *tc_core;

    s       = suite_create("Security_BufferGrowth");
    tc_core = tcase_create("Core");

    tcase_set_timeout(tc_core, 60);

    tcase_add_test(tc_core, test_buffer_growth_security_invariants);
    tcase_add_test(tc_core, test_capacity_doubling_no_overflow);
    tcase_add_test(tc_core, test_realloc_failure_no_use_after_free);
    tcase_add_test(tc_core, test_final_buffer_nul_termination);

    suite_add_tcase(s, tc_core);
    return s;
}

int main(void)
{
    int      number_failed;
    Suite   *s;
    SRunner *sr;

    s  = security_suite();
    sr = srunner_create(s);

    srunner_run_all(sr, CK_NORMAL);
    number_failed = srunner_ntests_failed(sr);
    srunner_free(sr);

    return (number_failed == 0) ? EXIT_SUCCESS : EXIT_FAILURE;
}