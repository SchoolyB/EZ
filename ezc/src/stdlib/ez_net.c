/*
 * ez_net.c - @net module implementation
 *
 * TCP/UDP networking using POSIX sockets.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_net.h"
#include <string.h>
#include <stdio.h>
#include <unistd.h>
#include <errno.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <sys/time.h>

#define EZ_NET_HOST_BUF         256
#define EZ_NET_PORT_BUF         16
#define EZ_NET_MAX_RECV_BUF     1048576
#define EZ_NET_LISTEN_BACKLOG   128

/* Helper: null-terminate an EzString */
static const char *net_cstr(EzString s, char *buf, size_t bufsz) {
    size_t len = (size_t)s.len < bufsz - 1 ? (size_t)s.len : bufsz - 1;
    memcpy(buf, s.data, len);
    buf[len] = '\0';
    return buf;
}

EzSocket ez_net_dial(EzArena *arena, EzString host, int64_t port) {
    (void)arena;
    EzSocket sock = {-1};

    char host_buf[EZ_NET_HOST_BUF];
    net_cstr(host, host_buf, sizeof(host_buf));

    /* Resolve hostname */
    struct addrinfo hints, *res;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;

    char port_str[EZ_NET_PORT_BUF];
    snprintf(port_str, sizeof(port_str), "%d", (int)port);

    if (getaddrinfo(host_buf, port_str, &hints, &res) != 0) {
        return sock;
    }

    int fd = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
    if (fd < 0) {
        freeaddrinfo(res);
        return sock;
    }

    if (connect(fd, res->ai_addr, res->ai_addrlen) != 0) {
        close(fd);
        freeaddrinfo(res);
        return sock;
    }

    freeaddrinfo(res);
    sock.fd = fd;
    return sock;
}

void ez_net_close(EzSocket sock) {
    if (sock.fd >= 0) {
        close(sock.fd);
    }
}

int64_t ez_net_send(EzSocket sock, EzString data) {
    if (sock.fd < 0 || !data.data) return -1;
    ssize_t sent = send(sock.fd, data.data, (size_t)data.len, 0);
    return (int64_t)sent;
}

EzString ez_net_recv(EzArena *arena, EzSocket sock, int64_t max_bytes) {
    if (sock.fd < 0 || max_bytes <= 0) return (EzString){"", 0};

    size_t bufsz = (size_t)max_bytes;
    if (bufsz > EZ_NET_MAX_RECV_BUF) bufsz = EZ_NET_MAX_RECV_BUF; /* cap at 1MB */
    char *buf = ez_arena_alloc(arena, bufsz);

    ssize_t n = recv(sock.fd, buf, bufsz, 0);
    if (n <= 0) return (EzString){"", 0};

    return (EzString){buf, (int32_t)n};
}

EzSocket ez_net_listen(EzArena *arena, int64_t port) {
    (void)arena;
    EzSocket sock = {-1};

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) return sock;

    /* Allow port reuse */
    int opt = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons((uint16_t)port);

    if (bind(fd, (struct sockaddr *)&addr, sizeof(addr)) != 0) {
        close(fd);
        return sock;
    }

    if (listen(fd, EZ_NET_LISTEN_BACKLOG) != 0) {
        close(fd);
        return sock;
    }

    sock.fd = fd;
    return sock;
}

EzSocket ez_net_accept(EzArena *arena, EzSocket listener) {
    (void)arena;
    EzSocket sock = {-1};
    if (listener.fd < 0) return sock;

    struct sockaddr_in client_addr;
    socklen_t client_len = sizeof(client_addr);
    int fd = accept(listener.fd, (struct sockaddr *)&client_addr, &client_len);
    if (fd < 0) return sock;

    sock.fd = fd;
    return sock;
}

void ez_net_set_timeout(EzSocket sock, int64_t milliseconds) {
    if (sock.fd < 0) return;
    struct timeval tv;
    tv.tv_sec = milliseconds / 1000;
    tv.tv_usec = (milliseconds % 1000) * 1000;
    setsockopt(sock.fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    setsockopt(sock.fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
}

EzString ez_net_resolve(EzArena *arena, EzString hostname) {
    char host_buf[EZ_NET_HOST_BUF];
    net_cstr(hostname, host_buf, sizeof(host_buf));

    struct addrinfo hints, *res;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;

    if (getaddrinfo(host_buf, NULL, &hints, &res) != 0) {
        return (EzString){"", 0};
    }

    struct sockaddr_in *addr = (struct sockaddr_in *)res->ai_addr;
    char ip_buf[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &addr->sin_addr, ip_buf, sizeof(ip_buf));

    EzString result = ez_string_new(arena, ip_buf, (int32_t)strlen(ip_buf));
    freeaddrinfo(res);
    return result;
}

/* _result variants */

EzResult_socket ez_net_dial_result(EzArena *arena, EzString host, int64_t port) {
    EzResult_socket r;
    r.v0 = ez_net_dial(arena, host, port);
    if (r.v0.fd < 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot connect to '%.*s:%lld'",
            host.len, host.data, (long long)port));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzSocket ez_net_listen_host(EzArena *arena, EzString host, int64_t port) {
    (void)arena;
    EzSocket sock = {-1};

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) return sock;

    int opt = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    char host_buf[EZ_NET_HOST_BUF];
    net_cstr(host, host_buf, sizeof(host_buf));

    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons((uint16_t)port);

    if (inet_pton(AF_INET, host_buf, &addr.sin_addr) != 1) {
        close(fd);
        return sock;
    }

    if (bind(fd, (struct sockaddr *)&addr, sizeof(addr)) != 0) {
        close(fd);
        return sock;
    }

    if (listen(fd, EZ_NET_LISTEN_BACKLOG) != 0) {
        close(fd);
        return sock;
    }

    sock.fd = fd;
    return sock;
}

EzResult_socket ez_net_listen_result(EzArena *arena, int64_t port) {
    EzResult_socket r;
    r.v0 = ez_net_listen(arena, port);
    if (r.v0.fd < 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot listen on port %lld",
            (long long)port));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_socket ez_net_listen_host_result(EzArena *arena, EzString host, int64_t port) {
    EzResult_socket r;
    r.v0 = ez_net_listen_host(arena, host, port);
    if (r.v0.fd < 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot listen on %.*s:%lld",
            host.len, host.data, (long long)port));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_socket ez_net_accept_result(EzArena *arena, EzSocket listener) {
    EzResult_socket r;
    r.v0 = ez_net_accept(arena, listener);
    if (r.v0.fd < 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "accept failed on fd %d", listener.fd));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_int ez_net_send_result(EzArena *arena, EzSocket sock, EzString data) {
    EzResult_int r;
    r.v0 = ez_net_send(sock, data);
    if (r.v0 < 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "send failed on fd %d", sock.fd));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_string ez_net_recv_result(EzArena *arena, EzSocket sock, int64_t max_bytes) {
    EzResult_string r;
    r.v0 = ez_net_recv(arena, sock, max_bytes);
    if (r.v0.len == 0 && sock.fd >= 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "recv returned no data on fd %d", sock.fd));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_string ez_net_resolve_result(EzArena *arena, EzString hostname) {
    EzResult_string r;
    r.v0 = ez_net_resolve(arena, hostname);
    if (r.v0.len == 0) {
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot resolve '%.*s'",
            hostname.len, hostname.data));
    } else {
        r.v1 = NULL;
    }
    return r;
}
