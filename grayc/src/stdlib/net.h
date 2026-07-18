/*
 * net.h — Public interface for the net stdlib module.
 * Declares low-level TCP/UDP networking primitives for connect,
 * listen, accept, send, and receive operations.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_NET_H
#define GRAY_NET_H

#include "../runtime/runtime.h"
#include "io.h" /* GrayResult_string */

/* Opaque socket handle */
typedef struct {
    int fd;
} GraySocket;

/* TCP client */
GraySocket gray_net_dial(GrayArena *arena, GrayString host, int64_t port);
void gray_net_close(GraySocket sock);

/* Send/receive raw data */
int64_t gray_net_send(GraySocket sock, GrayString data);
GrayString gray_net_recv(GrayArena *arena, GraySocket sock, int64_t max_bytes);

/* TCP server */
GraySocket gray_net_listen(GrayArena *arena, int64_t port);
GraySocket gray_net_listen_host(GrayArena *arena, GrayString host, int64_t port);
GraySocket gray_net_accept(GrayArena *arena, GraySocket listener);

/* Socket options */
void gray_net_set_timeout(GraySocket sock, int64_t milliseconds);

/* DNS resolution */
GrayString gray_net_resolve(GrayArena *arena, GrayString hostname);

/* _result variants */
typedef struct { GraySocket v0; GrayError *v1; } GrayResult_socket;

GrayResult_socket gray_net_dial_result(GrayArena *arena, GrayString host, int64_t port);
GrayResult_socket gray_net_listen_result(GrayArena *arena, int64_t port);
GrayResult_socket gray_net_listen_host_result(GrayArena *arena, GrayString host, int64_t port);
GrayResult_socket gray_net_accept_result(GrayArena *arena, GraySocket listener);
GrayResult_int gray_net_send_result(GrayArena *arena, GraySocket sock, GrayString data);
GrayResult_string gray_net_recv_result(GrayArena *arena, GraySocket sock, int64_t max_bytes);
GrayResult_string gray_net_resolve_result(GrayArena *arena, GrayString hostname);

#endif
