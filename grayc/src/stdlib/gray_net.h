/*
 * gray_net.h - @net module for EZ
 *
 * Low-level TCP/UDP networking primitives.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_NET_H
#define GRAY_NET_H

#include "../runtime/gray_runtime.h"
#include "gray_io.h" /* EzResult_string */

/* Opaque socket handle */
typedef struct {
    int fd;
} EzSocket;

/* TCP client */
EzSocket gray_net_dial(EzArena *arena, EzString host, int64_t port);
void gray_net_close(EzSocket sock);

/* Send/receive raw data */
int64_t gray_net_send(EzSocket sock, EzString data);
EzString gray_net_recv(EzArena *arena, EzSocket sock, int64_t max_bytes);

/* TCP server */
EzSocket gray_net_listen(EzArena *arena, int64_t port);
EzSocket gray_net_listen_host(EzArena *arena, EzString host, int64_t port);
EzSocket gray_net_accept(EzArena *arena, EzSocket listener);

/* Socket options */
void gray_net_set_timeout(EzSocket sock, int64_t milliseconds);

/* DNS resolution */
EzString gray_net_resolve(EzArena *arena, EzString hostname);

/* _result variants */
typedef struct { EzSocket v0; EzError *v1; } EzResult_socket;

EzResult_socket gray_net_dial_result(EzArena *arena, EzString host, int64_t port);
EzResult_socket gray_net_listen_result(EzArena *arena, int64_t port);
EzResult_socket gray_net_listen_host_result(EzArena *arena, EzString host, int64_t port);
EzResult_socket gray_net_accept_result(EzArena *arena, EzSocket listener);
EzResult_int gray_net_send_result(EzArena *arena, EzSocket sock, EzString data);
EzResult_string gray_net_recv_result(EzArena *arena, EzSocket sock, int64_t max_bytes);
EzResult_string gray_net_resolve_result(EzArena *arena, EzString hostname);

#endif
