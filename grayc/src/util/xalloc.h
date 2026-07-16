/*
 * xalloc.h — Allocation wrappers (xmalloc, xcalloc, xrealloc) that abort
 * on out-of-memory, used throughout the compiler for non-recoverable allocs.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAYC_XALLOC_H
#define GRAYC_XALLOC_H

#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>

static inline void *xrealloc(void *ptr, size_t size) {
    void *p = realloc(ptr, size);
    if (!p) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    return p;
}

static inline void *xmalloc(size_t size) {
    void *p = malloc(size);
    if (!p) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    return p;
}

static inline void *xcalloc(size_t nmemb, size_t size) {
    void *p = calloc(nmemb, size);
    if (!p) {
        fprintf(stderr, "grayc: out of memory\n");
        exit(1);
    }
    return p;
}

#endif
