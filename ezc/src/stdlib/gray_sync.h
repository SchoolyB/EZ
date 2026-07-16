/*
 * gray_sync.h - @sync module for EZ
 *
 * Synchronization primitives: mutex lock/unlock.
 * Built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_SYNC_H
#define GRAY_SYNC_H

#include "../runtime/gray_runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* pthread_mutex_t* */
} EzMutex;

EzMutex gray_sync_mutex(void);
void gray_sync_lock(EzMutex m);
void gray_sync_unlock(EzMutex m);
bool gray_sync_try_lock(EzMutex m);
void gray_sync_destroy(EzMutex m);

#endif
