/*
 * sync.c — Implementation of the sync stdlib module.
 * Provides mutex creation, lock, unlock, and try_lock operations
 * built on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "sync.h"
#include <pthread.h>
#include <stdlib.h>

GrayMutex gray_sync_mutex(void) {
    pthread_mutex_t *handle = malloc(sizeof(pthread_mutex_t));
    pthread_mutex_init(handle, NULL);
    GrayMutex result;
    result._internal = handle;
    return result;
}

/* A zero-value mutex gives no mutual exclusion, so using one is a bug the
 * program must hear about rather than a silent no-op. */
static pthread_mutex_t *live_mutex(GrayMutex mutex, const char *function_name) {
    if (!mutex._internal) gray_panic_code("P0168", "sync.%s on a mutex that was never created with sync.mutex() or was already destroyed", function_name);
    return (pthread_mutex_t *)mutex._internal;
}

void gray_sync_lock(GrayMutex mutex) {
    pthread_mutex_lock(live_mutex(mutex, "lock"));
}

void gray_sync_unlock(GrayMutex mutex) {
    pthread_mutex_unlock(live_mutex(mutex, "unlock"));
}

bool gray_sync_try_lock(GrayMutex mutex) {
    return pthread_mutex_trylock(live_mutex(mutex, "try_lock")) == 0;
}

void gray_sync_destroy(GrayMutex *mutex) {
    pthread_mutex_t *handle = live_mutex(*mutex, "destroy");
    pthread_mutex_destroy(handle);
    free(handle);
    mutex->_internal = NULL;
}
