/*
 * gray_sync.c — Implementation of the sync stdlib module.
 * Provides mutex creation, lock, unlock, and try_lock operations
 * built on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_sync.h"
#include <pthread.h>
#include <stdlib.h>

GrayMutex gray_sync_mutex(void) {
    pthread_mutex_t *m = malloc(sizeof(pthread_mutex_t));
    pthread_mutex_init(m, NULL);
    GrayMutex result;
    result._internal = m;
    return result;
}

void gray_sync_lock(GrayMutex m) {
    if (m._internal) {
        pthread_mutex_lock((pthread_mutex_t *)m._internal);
    }
}

void gray_sync_unlock(GrayMutex m) {
    if (m._internal) {
        pthread_mutex_unlock((pthread_mutex_t *)m._internal);
    }
}

bool gray_sync_try_lock(GrayMutex m) {
    if (m._internal) {
        return pthread_mutex_trylock((pthread_mutex_t *)m._internal) == 0;
    }
    return false;
}

void gray_sync_destroy(GrayMutex m) {
    if (m._internal) {
        pthread_mutex_destroy((pthread_mutex_t *)m._internal);
        free(m._internal);
    }
}
