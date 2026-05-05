/*
 * xalloc.h - allocation wrappers that abort on OOM
 *
 * Compiler-internal allocations are not user-recoverable; on failure
 * we print to stderr and exit. Mirrors the pattern already inlined in
 * buf.c and arena.c.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_XALLOC_H
#define EZC_XALLOC_H

#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>

static inline void *xrealloc(void *ptr, size_t size) {
    void *p = realloc(ptr, size);
    if (!p) {
        fprintf(stderr, "ezc: out of memory\n");
        exit(1);
    }
    return p;
}

static inline void *xmalloc(size_t size) {
    void *p = malloc(size);
    if (!p) {
        fprintf(stderr, "ezc: out of memory\n");
        exit(1);
    }
    return p;
}

#endif
