/*
 * ez_threads.h - @threads module for EZC
 *
 * Thread spawning and joining. Built on POSIX pthreads.
 * See @sync for mutexes, @channels for message passing.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_THREADS_H
#define EZ_THREADS_H

#include "../runtime/ez_runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* pthread_t stored as opaque pointer */
} EzThread;

/* Spawn a new thread running the given function.
 * The function pointer must match: void (*fn)(void) or void (*fn)(int64_t)
 * Returns a thread handle for joining. */
EzThread ez_threads_spawn(void (*fn)(void));
EzThread ez_threads_spawn_arg(void (*fn)(int64_t), int64_t arg);

/* Wait for a thread to finish. */
void ez_threads_join(EzThread t);

/* Get current thread ID (for debugging). */
int64_t ez_threads_id(void);

#endif
