/*
 * ez_threads.c - @threads module implementation
 *
 * Built on POSIX pthreads. See @sync for mutexes, @channels for
 * message passing.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_threads.h"
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
} EzThreadInternal;

static _Atomic int64_t ez_threads_live_count = 0;

/* Wrapper for void(*)(void) thread functions */
typedef struct {
    void (*fn)(void);
    EzThreadInternal *state;
} ThreadArg0;

static void *thread_entry_0(void *arg) {
    ThreadArg0 *ta = (ThreadArg0 *)arg;
    EzThreadInternal *st = ta->state;
    ez_default_arena = ez_arena_create(EZ_DEFAULT_ARENA_SIZE);
    ta->fn();
    ez_arena_destroy(ez_default_arena, __FILE__, __LINE__);
    free(ez_default_arena);
    ez_default_arena = NULL;
    atomic_fetch_sub(&ez_threads_live_count, 1);
    atomic_store(&st->alive, 0);
    /* If detached, the state struct's lifetime ends here too. */
    if (atomic_load(&st->detached)) free(st);
    free(ta);
    return NULL;
}

EzThread ez_threads_spawn(void (*fn)(void)) {
    EzThreadInternal *st = malloc(sizeof(EzThreadInternal));
    /* alive=1 set before pthread_create so is_alive() is true immediately
     * after spawn() returns; otherwise callers race the scheduler. The
     * thread wrapper clears it on exit. */
    atomic_store(&st->alive, 1);
    atomic_store(&st->detached, 0);
    atomic_fetch_add(&ez_threads_live_count, 1);
    ThreadArg0 *ta = malloc(sizeof(ThreadArg0));
    ta->fn = fn;
    ta->state = st;
    pthread_create(&st->pt, NULL, thread_entry_0, ta);
    EzThread t;
    t._internal = st;
    return t;
}

/* Wrapper for void(*)(int64_t) thread functions */
typedef struct {
    void (*fn)(int64_t);
    int64_t arg;
    EzThreadInternal *state;
} ThreadArg1;

static void *thread_entry_1(void *arg) {
    ThreadArg1 *ta = (ThreadArg1 *)arg;
    EzThreadInternal *st = ta->state;
    ez_default_arena = ez_arena_create(EZ_DEFAULT_ARENA_SIZE);
    ta->fn(ta->arg);
    ez_arena_destroy(ez_default_arena, __FILE__, __LINE__);
    free(ez_default_arena);
    ez_default_arena = NULL;
    atomic_fetch_sub(&ez_threads_live_count, 1);
    atomic_store(&st->alive, 0);
    if (atomic_load(&st->detached)) free(st);
    free(ta);
    return NULL;
}

EzThread ez_threads_spawn_arg(void (*fn)(int64_t), int64_t arg) {
    EzThreadInternal *st = malloc(sizeof(EzThreadInternal));
    atomic_store(&st->alive, 1);
    atomic_store(&st->detached, 0);
    atomic_fetch_add(&ez_threads_live_count, 1);
    ThreadArg1 *ta = malloc(sizeof(ThreadArg1));
    ta->fn = fn;
    ta->arg = arg;
    ta->state = st;
    pthread_create(&st->pt, NULL, thread_entry_1, ta);
    EzThread t;
    t._internal = st;
    return t;
}

void ez_threads_join(EzThread t) {
    if (!t._internal) return;
    EzThreadInternal *st = (EzThreadInternal *)t._internal;
    pthread_join(st->pt, NULL);
    free(st);
}

void ez_threads_detach(EzThread t) {
    if (!t._internal) return;
    EzThreadInternal *st = (EzThreadInternal *)t._internal;
    atomic_store(&st->detached, 1);
    /* If the thread already exited before detach() was called, the entry
     * function already saw detached=0 and won't free the struct — free
     * it here. Otherwise the entry function frees it once it finishes. */
    if (!atomic_load(&st->alive)) {
        /* Race-safe: pthread_detach is idempotent for a finished thread,
         * and we only free in the branch where alive was already 0. */
        pthread_detach(st->pt);
        free(st);
    } else {
        pthread_detach(st->pt);
    }
}

bool ez_threads_is_alive(EzThread t) {
    if (!t._internal) return false;
    EzThreadInternal *st = (EzThreadInternal *)t._internal;
    return atomic_load(&st->alive) != 0;
}

int64_t ez_threads_id(void) {
    return (int64_t)(uintptr_t)pthread_self();
}

int64_t ez_threads_current(void) {
    return ez_threads_id();
}

void ez_threads_yield(void) {
    sched_yield();
}

void ez_threads_sleep(int64_t ms) {
    if (ms < 0) ms = 0;
    struct timespec req;
    req.tv_sec  = (time_t)(ms / 1000);
    req.tv_nsec = (long)((ms % 1000) * 1000000L);
    /* Restart on signal interruption — the user asked for a sleep, not a
     * sleep-or-signal. */
    while (nanosleep(&req, &req) == -1 && errno == EINTR) {}
}

int64_t ez_threads_thread_count(void) {
    return atomic_load(&ez_threads_live_count);
}
