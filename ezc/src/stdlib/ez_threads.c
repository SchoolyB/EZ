/*
 * ez_threads.c - @threads module implementation
 *
 * Built on POSIX pthreads. Provides spawn, join, mutex, and channels.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_threads.h"
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

/* ========================================================================== */
/* Thread spawn/join                                                          */
/* ========================================================================== */

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

void ez_threads_sleep_ms(int64_t ms) {
    if (ms <= 0) return;
    struct timespec ts;
    ts.tv_sec = ms / 1000;
    ts.tv_nsec = (ms % 1000) * 1000000L;
    nanosleep(&ts, NULL);
}

int64_t ez_threads_id(void) {
    /* pthread_self() returns pthread_t which varies by platform.
     * Cast to int64_t for a unique-ish thread identifier. */
    return (int64_t)(uintptr_t)pthread_self();
}

/* ========================================================================== */
/* Mutex                                                                      */
/* ========================================================================== */

EzMutex ez_threads_mutex(void) {
    pthread_mutex_t *m = malloc(sizeof(pthread_mutex_t));
    pthread_mutex_init(m, NULL);
    EzMutex result;
    result._internal = m;
    return result;
}

void ez_threads_lock(EzMutex m) {
    if (m._internal) {
        pthread_mutex_lock((pthread_mutex_t *)m._internal);
    }
}

void ez_threads_unlock(EzMutex m) {
    if (m._internal) {
        pthread_mutex_unlock((pthread_mutex_t *)m._internal);
    }
}

void ez_threads_mutex_destroy(EzMutex m) {
    if (m._internal) {
        pthread_mutex_destroy((pthread_mutex_t *)m._internal);
        free(m._internal);
    }
}

/* ========================================================================== */
/* Channel (bounded, blocking, int64_t values)                                */
/* ========================================================================== */

typedef struct {
    int64_t *buffer;
    int64_t capacity;
    int64_t count;
    int64_t head;
    int64_t tail;
    pthread_mutex_t mutex;
    pthread_cond_t not_full;
    pthread_cond_t not_empty;
    bool closed;
} EzChannelInternal;

EzChannel ez_threads_channel(int64_t capacity) {
    if (capacity < 1) capacity = 1;
    EzChannelInternal *ch = malloc(sizeof(EzChannelInternal));
    ch->buffer = malloc(sizeof(int64_t) * (size_t)capacity);
    ch->capacity = capacity;
    ch->count = 0;
    ch->head = 0;
    ch->tail = 0;
    ch->closed = false;
    pthread_mutex_init(&ch->mutex, NULL);
    pthread_cond_init(&ch->not_full, NULL);
    pthread_cond_init(&ch->not_empty, NULL);
    EzChannel result;
    result._internal = ch;
    return result;
}

void ez_threads_send(EzChannel handle, int64_t value) {
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch || ch->closed) return;

    pthread_mutex_lock(&ch->mutex);
    while (ch->count == ch->capacity && !ch->closed) {
        pthread_cond_wait(&ch->not_full, &ch->mutex);
    }
    if (ch->closed) {
        pthread_mutex_unlock(&ch->mutex);
        return;
    }
    ch->buffer[ch->tail] = value;
    ch->tail = (ch->tail + 1) % ch->capacity;
    ch->count++;
    pthread_cond_signal(&ch->not_empty);
    pthread_mutex_unlock(&ch->mutex);
}

int64_t ez_threads_recv(EzChannel handle) {
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch) return 0;

    pthread_mutex_lock(&ch->mutex);
    while (ch->count == 0 && !ch->closed) {
        pthread_cond_wait(&ch->not_empty, &ch->mutex);
    }
    if (ch->count == 0 && ch->closed) {
        pthread_mutex_unlock(&ch->mutex);
        return 0;
    }
    int64_t value = ch->buffer[ch->head];
    ch->head = (ch->head + 1) % ch->capacity;
    ch->count--;
    pthread_cond_signal(&ch->not_full);
    pthread_mutex_unlock(&ch->mutex);
    return value;
}

void ez_threads_channel_close(EzChannel handle) {
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch) return;

    pthread_mutex_lock(&ch->mutex);
    ch->closed = true;
    pthread_cond_broadcast(&ch->not_full);
    pthread_cond_broadcast(&ch->not_empty);
    pthread_mutex_unlock(&ch->mutex);

    /* Note: don't free here — receivers may still be reading.
     * In a real implementation you'd ref-count or use a finalizer. */
}
