/*
 * ez_channels.c - @channels module implementation
 *
 * Bounded, blocking channels built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_channels.h"
#include <pthread.h>
#include <stdlib.h>

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

EzChannel ez_channels_open(int64_t capacity) {
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

void ez_channels_send(EzChannel handle, int64_t value) {
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

int64_t ez_channels_receive(EzChannel handle) {
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

bool ez_channels_try_send(EzChannel handle, int64_t value) {
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch || ch->closed) return false;

    pthread_mutex_lock(&ch->mutex);
    if (ch->count == ch->capacity || ch->closed) {
        pthread_mutex_unlock(&ch->mutex);
        return false;
    }
    ch->buffer[ch->tail] = value;
    ch->tail = (ch->tail + 1) % ch->capacity;
    ch->count++;
    pthread_cond_signal(&ch->not_empty);
    pthread_mutex_unlock(&ch->mutex);
    return true;
}

EzChannelTryRecv ez_channels_try_receive(EzChannel handle) {
    EzChannelTryRecv result = {0, false};
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch) return result;

    pthread_mutex_lock(&ch->mutex);
    if (ch->count == 0) {
        pthread_mutex_unlock(&ch->mutex);
        return result;
    }
    result.v0 = ch->buffer[ch->head];
    ch->head = (ch->head + 1) % ch->capacity;
    ch->count--;
    result.v1 = true;
    pthread_cond_signal(&ch->not_full);
    pthread_mutex_unlock(&ch->mutex);
    return result;
}

void ez_channels_close(EzChannel handle) {
    EzChannelInternal *ch = (EzChannelInternal *)handle._internal;
    if (!ch) return;

    pthread_mutex_lock(&ch->mutex);
    ch->closed = true;
    pthread_cond_broadcast(&ch->not_full);
    pthread_cond_broadcast(&ch->not_empty);
    pthread_mutex_unlock(&ch->mutex);
}
