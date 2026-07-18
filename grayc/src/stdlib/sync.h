/*
 * sync.h — Public interface for the sync stdlib module.
 * Declares mutex type and lock/unlock/try_lock operations built
 * on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_SYNC_H
#define GRAY_SYNC_H

#include "../runtime/runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* pthread_mutex_t* */
} GrayMutex;

GrayMutex gray_sync_mutex(void);
void gray_sync_lock(GrayMutex m);
void gray_sync_unlock(GrayMutex m);
bool gray_sync_try_lock(GrayMutex m);
void gray_sync_destroy(GrayMutex m);

#endif
