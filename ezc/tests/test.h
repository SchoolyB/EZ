/*
 * test.h - Minimal C test framework
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_TEST_H
#define EZC_TEST_H

#include <stdio.h>
#include <string.h>
#include <stdlib.h>

static int _test_pass = 0;
static int _test_fail = 0;
static int _test_total = 0;
static int _test_failed_this = 0;

#define ASSERT(cond) do { \
    if (!(cond)) { \
        fprintf(stderr, "  FAIL %s:%d: %s\n", __FILE__, __LINE__, #cond); \
        _test_failed_this = 1; \
        return; \
    } \
} while(0)

#define ASSERT_EQ(a, b) do { \
    if ((a) != (b)) { \
        fprintf(stderr, "  FAIL %s:%d: %s != %s (%lld != %lld)\n", \
            __FILE__, __LINE__, #a, #b, (long long)(a), (long long)(b)); \
        _test_failed_this = 1; \
        return; \
    } \
} while(0)

#define ASSERT_STR_EQ(a, b) do { \
    if (strcmp((a), (b)) != 0) { \
        fprintf(stderr, "  FAIL %s:%d: \"%s\" != \"%s\"\n", \
            __FILE__, __LINE__, (a), (b)); \
        _test_failed_this = 1; \
        return; \
    } \
} while(0)

#define ASSERT_NOT_NULL(ptr) do { \
    if ((ptr) == NULL) { \
        fprintf(stderr, "  FAIL %s:%d: %s is NULL\n", __FILE__, __LINE__, #ptr); \
        _test_failed_this = 1; \
        return; \
    } \
} while(0)

#define RUN_TEST(func) do { \
    _test_total++; \
    _test_failed_this = 0; \
    func(); \
    if (_test_failed_this) { \
        _test_fail++; \
    } else { \
        printf("  PASS %s\n", #func); \
        _test_pass++; \
    } \
} while(0)

#define PRINT_RESULTS() do { \
    printf("\n%d passed, %d failed (%d total)\n\n", \
        _test_pass, _test_fail, _test_total); \
} while(0)

#endif
