/*
 * threads.h — Public interface for the threads stdlib module.
 * Declares thread spawning, joining, detaching, ID queries, and
 * yield functions built on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_THREADS_H
#define GRAY_THREADS_H

#include "../runtime/runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* points to an internal {pthread_t, alive flag} */
} GrayThread;

/* Spawn a new thread running the given function.
 * The function pointer must match: void (*fn)(void) or void (*fn)(int64_t)
 * Returns a thread handle for joining. */
GrayThread gray_threads_spawn(void (*fn)(void));
GrayThread gray_threads_spawn_arg(void (*fn)(int64_t), int64_t arg);

/* Wait for a thread to finish. Frees the underlying handle. */
void gray_threads_join(GrayThread t);

/* Release ownership; the thread runs to completion independently and its
 * underlying handle is freed by pthread itself. After detach() the
 * GrayThread must not be joined or queried for is_alive(). */
void gray_threads_detach(GrayThread t);

/* True while the thread's entry function has not returned. Safe to call
 * before join() (and meaningless after). Not valid after detach(). */
bool gray_threads_is_alive(GrayThread t);

/* Get current thread id (for debugging). Identical to `current()`; both
 * names exist so callers can pick whichever reads better at the call
 * site. */
int64_t gray_threads_id(void);
int64_t gray_threads_current(void);

/* Hint the scheduler to run another runnable thread. */
void gray_threads_yield(void);

/* Sleep the current thread for `ms` milliseconds. */
void gray_threads_sleep(int64_t ms);

/* Number of live threads currently spawned through this module. Excludes
 * the main thread and any non-Grayscale threads in the process. */
int64_t gray_threads_thread_count(void);

#endif
