/*
 * gray_sync.c - @sync module implementation
 *
 * Synchronization primitives built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_sync.h"
#include <pthread.h>
#include <stdlib.h>

EzMutex gray_sync_mutex(void) {
    pthread_mutex_t *m = malloc(sizeof(pthread_mutex_t));
    pthread_mutex_init(m, NULL);
    EzMutex result;
    result._internal = m;
    return result;
}

void gray_sync_lock(EzMutex m) {
    if (m._internal) {
        pthread_mutex_lock((pthread_mutex_t *)m._internal);
    }
}

void gray_sync_unlock(EzMutex m) {
    if (m._internal) {
        pthread_mutex_unlock((pthread_mutex_t *)m._internal);
    }
}

bool gray_sync_try_lock(EzMutex m) {
    if (m._internal) {
        return pthread_mutex_trylock((pthread_mutex_t *)m._internal) == 0;
    }
    return false;
}

void gray_sync_destroy(EzMutex m) {
    if (m._internal) {
        pthread_mutex_destroy((pthread_mutex_t *)m._internal);
        free(m._internal);
    }
}
