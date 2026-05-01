/*
 * ez_channels.h - @channels module for EZ
 *
 * Bounded, blocking channels for message passing between threads.
 * Built on POSIX pthreads.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_CHANNELS_H
#define EZ_CHANNELS_H

#include "../runtime/ez_runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* EzChannelInternal* */
} EzChannel;

/* Create a buffered channel with given capacity. */
EzChannel ez_channels_open(int64_t capacity);

/* Send a value into the channel. Blocks if full. */
void ez_channels_send(EzChannel ch, int64_t value);

/* Receive a value from the channel. Blocks if empty. */
int64_t ez_channels_receive(EzChannel ch);

/* Close a channel. */
void ez_channels_close(EzChannel ch);

#endif
