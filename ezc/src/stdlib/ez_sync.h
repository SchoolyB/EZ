/*
 * ez_sync.h - @sync module for EZC
 *
 * Synchronization primitives: mutex lock/unlock.
 * Built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_SYNC_H
#define EZ_SYNC_H

#include "../runtime/ez_runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* pthread_mutex_t* */
} EzMutex;

EzMutex ez_sync_mutex(void);
void ez_sync_lock(EzMutex m);
void ez_sync_unlock(EzMutex m);
bool ez_sync_try_lock(EzMutex m);
void ez_sync_destroy(EzMutex m);

#endif
