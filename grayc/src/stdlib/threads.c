/*
 * threads.c — Implementation of the threads stdlib module.
 * Provides thread spawning, joining, detaching, ID queries, and
 * yield, all built on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "threads.h"
#include <pthread.h>
#include <sched.h>
#include <stdlib.h>
#include <stdatomic.h>
#include <time.h>
#include <errno.h>

typedef struct {
    pthread_t pt;
    _Atomic int alive;       /* 1 between entry and exit, 0 otherwise */
    _Atomic int detached;    /* 1 once detach() has been called */
} GrayThreadInternal;

static _Atomic int64_t gray_threads_live_count = 0;

/* Unified thread wrapper: fn0 is set for no-arg spawns, fn1 for one-arg. */
typedef struct {
    void (*fn0)(void);
    void (*fn1)(int64_t);
    int64_t arg;
    GrayThreadInternal *state;
} ThreadArg;

static void *thread_entry(void *raw) {
    ThreadArg *ta = (ThreadArg *)raw;
    GrayThreadInternal *state = ta->state;
    gray_default_arena = gray_arena_create(GRAY_DEFAULT_ARENA_SIZE);
    if (ta->fn1)
        ta->fn1(ta->arg);
    else
        ta->fn0();
    gray_arena_destroy(gray_default_arena, __FILE__, __LINE__);
    free(gray_default_arena);
    gray_default_arena = NULL;
    atomic_fetch_sub(&gray_threads_live_count, 1);
    atomic_store(&state->alive, 0);
    /* If detached, the state struct's lifetime ends here too. */
    if (atomic_load(&state->detached)) free(state);
    free(ta);
    return NULL;
}

static GrayThread spawn_thread(ThreadArg *ta) {
    GrayThreadInternal *state = ta->state;
    /* alive=1 set before pthread_create so is_alive() is true immediately
     * after spawn() returns; otherwise callers race the scheduler. The
     * thread wrapper clears it on exit. */
    atomic_store(&state->alive, 1);
    atomic_store(&state->detached, 0);
    atomic_fetch_add(&gray_threads_live_count, 1);
    pthread_create(&state->pt, NULL, thread_entry, ta);
    GrayThread t;
    t._internal = state;
    return t;
}

GrayThread gray_threads_spawn(void (*fn)(void)) {
    ThreadArg *ta = malloc(sizeof(ThreadArg));
    ta->fn0 = fn;
    ta->fn1 = NULL;
    ta->arg = 0;
    ta->state = malloc(sizeof(GrayThreadInternal));
    return spawn_thread(ta);
}

GrayThread gray_threads_spawn_arg(void (*fn)(int64_t), int64_t arg) {
    ThreadArg *ta = malloc(sizeof(ThreadArg));
    ta->fn0 = NULL;
    ta->fn1 = fn;
    ta->arg = arg;
    ta->state = malloc(sizeof(GrayThreadInternal));
    return spawn_thread(ta);
}

void gray_threads_join(GrayThread t) {
    if (!t._internal) return;
    GrayThreadInternal *state = (GrayThreadInternal *)t._internal;
    pthread_join(state->pt, NULL);
    free(state);
}

void gray_threads_detach(GrayThread t) {
    if (!t._internal) return;
    GrayThreadInternal *state = (GrayThreadInternal *)t._internal;
    atomic_store(&state->detached, 1);
    /* If the thread already exited before detach() was called, the entry
     * function already saw detached=0 and won't free the struct — free
     * it here. Otherwise the entry function frees it once it finishes. */
    if (!atomic_load(&state->alive)) {
        /* Race-safe: pthread_detach is idempotent for a finished thread,
         * and we only free in the branch where alive was already 0. */
        pthread_detach(state->pt);
        free(state);
    } else {
        pthread_detach(state->pt);
    }
}

bool gray_threads_is_alive(GrayThread t) {
    if (!t._internal) return false;
    GrayThreadInternal *state = (GrayThreadInternal *)t._internal;
    return atomic_load(&state->alive) != 0;
}

int64_t gray_threads_id(void) {
    return (int64_t)(uintptr_t)pthread_self();
}

int64_t gray_threads_current(void) {
    return gray_threads_id();
}

void gray_threads_yield(void) {
    sched_yield();
}

void gray_threads_sleep(int64_t ms) {
    if (ms < 0) ms = 0;
    struct timespec req;
    req.tv_sec  = (time_t)(ms / 1000);
    req.tv_nsec = (long)((ms % 1000) * 1000000L);
    /* Restart on signal interruption — the user asked for a sleep, not a
     * sleep-or-signal. */
    while (nanosleep(&req, &req) == -1 && errno == EINTR) {}
}

int64_t gray_threads_thread_count(void) {
    return atomic_load(&gray_threads_live_count);
}
