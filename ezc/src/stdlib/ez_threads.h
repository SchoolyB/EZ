/*
 * ez_threads.h - @threads module for EZC
 *
 * Provides threading primitives: spawn, join, mutex, and channels.
 * Built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_THREADS_H
#define EZ_THREADS_H

#include "../runtime/ez_runtime.h"
#include <stdint.h>
#include <stdbool.h>

/* ---- Thread Handle ---- */

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

/* ---- Mutex ---- */

typedef struct {
    void *_internal; /* pthread_mutex_t* */
} EzMutex;

EzMutex ez_threads_mutex(void);
void ez_threads_lock(EzMutex m);
void ez_threads_unlock(EzMutex m);
void ez_threads_mutex_destroy(EzMutex m);

/* ---- Channel (typed as int64_t for simplicity) ---- */

typedef struct {
    void *_internal; /* EzChannelInternal* */
} EzChannel;

/* Create a buffered channel with given capacity. */
EzChannel ez_threads_channel(int64_t capacity);

/* Send a value into the channel. Blocks if full. */
void ez_threads_send(EzChannel ch, int64_t value);

/* Receive a value from the channel. Blocks if empty. */
int64_t ez_threads_recv(EzChannel ch);

/* Close a channel and free resources. */
void ez_threads_channel_close(EzChannel ch);

#endif
