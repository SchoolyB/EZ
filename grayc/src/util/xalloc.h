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

/* Read an entire seekable file into a malloc'd NUL-terminated string.
 * Returns NULL on open failure or OOM. */
static inline char *read_file_to_string(const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) return NULL;

    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *buf = malloc((size_t)size + 1);
    if (!buf) { fclose(f); return NULL; }

    size_t n = fread(buf, 1, (size_t)size, f);
    buf[n] = '\0';
    fclose(f);
    return buf;
}

#endif
