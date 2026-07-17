/*
 * gray_channels.h — Public interface for the channels stdlib module.
 * Bounded, blocking channels for message passing between threads,
 * built on POSIX pthreads.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_CHANNELS_H
#define GRAY_CHANNELS_H

#include "../runtime/gray_runtime.h"
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    void *_internal; /* GrayChannelInternal* */
} GrayChannel;

/* Create a buffered channel with given capacity. */
GrayChannel gray_channels_open(int64_t capacity);

/* Send a value into the channel. Blocks if full. */
void gray_channels_send(GrayChannel ch, int64_t value);

/* Receive a value from the channel. Blocks if empty. */
int64_t gray_channels_receive(GrayChannel ch);

/* Close a channel. */
void gray_channels_close(GrayChannel ch);

/*@man
 *@module channels
 *@function try_send
 *@brief Attempt to send a value into a channel without blocking.
 *@param ch Channel - The channel to send to.
 *@param value int - The value to send.
 *@returns bool - true if the value was sent, false if the channel is full.
 *@end
 */
bool gray_channels_try_send(GrayChannel ch, int64_t value);

/*@man
 *@module channels
 *@function try_receive
 *@brief Attempt to receive a value from a channel without blocking.
 *@param ch Channel - The channel to receive from.
 *@returns (int, bool) - The value and true if one was available, or (0, false) if empty.
 *@end
 */
typedef struct { int64_t v0; bool v1; } GrayChannelTryRecv;
GrayChannelTryRecv gray_channels_try_receive(GrayChannel ch);

#endif
