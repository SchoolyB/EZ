/*
 * ez_net.h - @net module for EZ
 *
 * Low-level TCP/UDP networking primitives.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_NET_H
#define EZ_NET_H

#include "../runtime/ez_runtime.h"
#include "ez_io.h" /* EzResult_string */

/* Opaque socket handle */
typedef struct {
    int fd;
} EzSocket;

/* TCP client */
EzSocket ez_net_dial(EzArena *arena, EzString host, int64_t port);
void ez_net_close(EzSocket sock);

/* Send/receive raw data */
int64_t ez_net_send(EzSocket sock, EzString data);
EzString ez_net_recv(EzArena *arena, EzSocket sock, int64_t max_bytes);

/* TCP server */
EzSocket ez_net_listen(EzArena *arena, int64_t port);
EzSocket ez_net_listen_host(EzArena *arena, EzString host, int64_t port);
EzSocket ez_net_accept(EzArena *arena, EzSocket listener);

/* Socket options */
void ez_net_set_timeout(EzSocket sock, int64_t milliseconds);

/* DNS resolution */
EzString ez_net_resolve(EzArena *arena, EzString hostname);

/* _result variants */
typedef struct { EzSocket v0; EzError *v1; } EzResult_socket;

EzResult_socket ez_net_dial_result(EzArena *arena, EzString host, int64_t port);
EzResult_socket ez_net_listen_result(EzArena *arena, int64_t port);
EzResult_socket ez_net_listen_host_result(EzArena *arena, EzString host, int64_t port);
EzResult_socket ez_net_accept_result(EzArena *arena, EzSocket listener);
EzResult_int ez_net_send_result(EzArena *arena, EzSocket sock, EzString data);
EzResult_string ez_net_recv_result(EzArena *arena, EzSocket sock, int64_t max_bytes);
EzResult_string ez_net_resolve_result(EzArena *arena, EzString hostname);

#endif
