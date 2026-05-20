/*
 * ez_threads.h - @threads module for EZ
 *
 * Thread spawning, joining, detaching, and introspection.
 * Built on POSIX pthreads. See @sync for mutexes, @channels for
 * message passing.
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
    void *_internal; /* points to an internal {pthread_t, alive flag} */
} EzThread;

/* Spawn a new thread running the given function.
 * The function pointer must match: void (*fn)(void) or void (*fn)(int64_t)
 * Returns a thread handle for joining. */
EzThread ez_threads_spawn(void (*fn)(void));
EzThread ez_threads_spawn_arg(void (*fn)(int64_t), int64_t arg);

/* Wait for a thread to finish. Frees the underlying handle. */
void ez_threads_join(EzThread t);

/* Release ownership; the thread runs to completion independently and its
 * underlying handle is freed by pthread itself. After detach() the
 * EzThread must not be joined or queried for is_alive(). */
void ez_threads_detach(EzThread t);

/* True while the thread's entry function has not returned. Safe to call
 * before join() (and meaningless after). Not valid after detach(). */
bool ez_threads_is_alive(EzThread t);

/* Get current thread id (for debugging). Identical to `current()`; both
 * names exist so callers can pick whichever reads better at the call
 * site. */
int64_t ez_threads_id(void);
int64_t ez_threads_current(void);

/* Hint the scheduler to run another runnable thread. */
void ez_threads_yield(void);

/* Sleep the current thread for `ms` milliseconds. */
void ez_threads_sleep(int64_t ms);

/* Number of live threads currently spawned through this module. Excludes
 * the main thread and any non-EZ threads in the process. */
int64_t ez_threads_thread_count(void);

#endif
