/*
 * ez_threads.c - @threads module implementation
 *
 * Thread spawning and joining built on POSIX pthreads.
 * See @sync for mutexes, @channels for message passing.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_threads.h"
#include <pthread.h>
#include <stdlib.h>

/* Wrapper for void(*)(void) thread functions */
typedef struct {
    void (*fn)(void);
} ThreadArg0;

static void *thread_entry_0(void *arg) {
    ThreadArg0 *ta = (ThreadArg0 *)arg;
    ta->fn();
    free(ta);
    return NULL;
}

EzThread ez_threads_spawn(void (*fn)(void)) {
    pthread_t *pt = malloc(sizeof(pthread_t));
    ThreadArg0 *ta = malloc(sizeof(ThreadArg0));
    ta->fn = fn;
    pthread_create(pt, NULL, thread_entry_0, ta);
    EzThread t;
    t._internal = pt;
    return t;
}

/* Wrapper for void(*)(int64_t) thread functions */
typedef struct {
    void (*fn)(int64_t);
    int64_t arg;
} ThreadArg1;

static void *thread_entry_1(void *arg) {
    ThreadArg1 *ta = (ThreadArg1 *)arg;
    ta->fn(ta->arg);
    free(ta);
    return NULL;
}

EzThread ez_threads_spawn_arg(void (*fn)(int64_t), int64_t arg) {
    pthread_t *pt = malloc(sizeof(pthread_t));
    ThreadArg1 *ta = malloc(sizeof(ThreadArg1));
    ta->fn = fn;
    ta->arg = arg;
    pthread_create(pt, NULL, thread_entry_1, ta);
    EzThread t;
    t._internal = pt;
    return t;
}

void ez_threads_join(EzThread t) {
    if (t._internal) {
        pthread_t *pt = (pthread_t *)t._internal;
        pthread_join(*pt, NULL);
        free(pt);
    }
}

int64_t ez_threads_id(void) {
    return (int64_t)(uintptr_t)pthread_self();
}
