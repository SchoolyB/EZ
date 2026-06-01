/*
 * ez_server.c - @server module implementation
 *
 * HTTP server using POSIX sockets with thread-per-connection.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_server.h"
#include "ez_net.h"
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pthread.h>
#include <sys/socket.h>

#define EZ_SERVER_BUF_SIZE          65536
#define EZ_SERVER_REQUEST_ARENA     (64 * 1024)
#define EZ_SERVER_CORS_BUF          512
#define EZ_ROUTER_INITIAL_CAP       16
#define EZ_HTTP_METHOD_BUF          16
#define EZ_HTTP_PATH_BUF_SERVER     2048

/* Global arena for server allocations */
static EzArena *server_arena = NULL;

static EzArena *get_server_arena(void) {
    if (!server_arena) {
        server_arena = ez_arena_create(EZ_DEFAULT_ARENA_SIZE);
    }
    return server_arena;
}

EzRouter ez_server_router(void) {
    EzRouter r;
    r.count = 0;
    r.capacity = EZ_ROUTER_INITIAL_CAP;
    r.routes = malloc(sizeof(EzRoute) * r.capacity);
    r.cors_origin = NULL;
    r.middlewares = NULL;
    r.mw_count = 0;
    r.mw_capacity = 0;
    return r;
}

void ez_server_route(EzRouter *r, EzString method, EzString pattern,
                     EzResponse (*handler)(EzRequest)) {
    if (r->count >= r->capacity) {
        r->capacity *= 2;
        void *tmp = realloc(r->routes, sizeof(EzRoute) * r->capacity);
        if (!tmp) {
            fprintf(stderr, "ez: out of memory\n");
            exit(1);
        }
        r->routes = tmp;
    }
    EzRoute *route = &r->routes[r->count++];

    /* Null-terminate method and pattern */
    EzArena *a = get_server_arena();
    char *m = ez_arena_alloc(a, method.len + 1);
    memcpy(m, method.data, method.len);
    m[method.len] = '\0';
    route->method = m;

    char *p = ez_arena_alloc(a, pattern.len + 1);
    memcpy(p, pattern.data, pattern.len);
    p[pattern.len] = '\0';
    route->pattern = p;

    route->handler = handler;
}

void ez_server_cors(EzRouter *r, EzString origin) {
    EzArena *a = get_server_arena();
    char *o = ez_arena_alloc(a, origin.len + 1);
    memcpy(o, origin.data, origin.len);
    o[origin.len] = '\0';
    r->cors_origin = o;
}

void ez_server_use(EzRouter *r, EzMiddleware fn) {
    if (r->mw_count >= r->mw_capacity) {
        r->mw_capacity = r->mw_capacity == 0 ? 8 : r->mw_capacity * 2;
        r->middlewares = realloc(r->middlewares, sizeof(EzMiddleware) * r->mw_capacity);
    }
    r->middlewares[r->mw_count++] = fn;
}

/* Check if a route pattern matches a path, extracting params */
static bool match_route(const char *pattern, const char *path,
                        EzArena *arena, EzMap *params) {
    const char *pp = pattern;
    const char *rp = path;

    while (*pp && *rp) {
        if (*pp == ':') {
            /* Path parameter — extract name and value */
            pp++; /* skip : */
            const char *name_start = pp;
            while (*pp && *pp != '/') pp++;
            int32_t name_len = (int32_t)(pp - name_start);

            const char *val_start = rp;
            while (*rp && *rp != '/') rp++;
            int32_t val_len = (int32_t)(rp - val_start);

            EzString key = ez_string_new(arena, name_start, name_len);
            EzString val = ez_string_new(arena, val_start, val_len);
            ez_map_set(arena, params, &key, &val);
        } else {
            if (*pp != *rp) return false;
            pp++;
            rp++;
        }
    }

    /* Both must be fully consumed (or both at trailing /) */
    if (*pp == '\0' && *rp == '\0') return true;
    if (*pp == '\0' && *rp == '/' && *(rp+1) == '\0') return true;
    if (*rp == '\0' && *pp == '/' && *(pp+1) == '\0') return true;
    return false;
}

/* Parse HTTP request from raw data */
static bool parse_request(EzArena *arena, const char *data, int data_len,
                          EzRequest *req) {
    if (data_len < 10) return false;

    /* Parse request line: METHOD /path HTTP/1.1 */
    const char *sp1 = memchr(data, ' ', data_len);
    if (!sp1) return false;
    req->method = ez_string_new(arena, data, (int32_t)(sp1 - data));

    const char *path_start = sp1 + 1;
    const char *sp2 = memchr(path_start, ' ', data_len - (path_start - data));
    if (!sp2) return false;

    /* Split path and query string */
    const char *qmark = memchr(path_start, '?', sp2 - path_start);
    if (qmark) {
        req->path = ez_string_new(arena, path_start, (int32_t)(qmark - path_start));
        /* Parse query params */
        const char *qs = qmark + 1;
        int32_t qs_len = (int32_t)(sp2 - qs);
        /* Simple key=value&key2=value2 parser */
        const char *cursor = qs;
        const char *end = qs + qs_len;
        while (cursor < end) {
            const char *eq = memchr(cursor, '=', end - cursor);
            if (!eq) break;
            const char *amp = memchr(eq, '&', end - eq);
            if (!amp) amp = end;

            EzString key = ez_string_new(arena, cursor, (int32_t)(eq - cursor));
            EzString val = ez_string_new(arena, eq + 1, (int32_t)(amp - eq - 1));
            ez_map_set(arena, &req->query, &key, &val);

            cursor = amp + 1;
        }
    } else {
        req->path = ez_string_new(arena, path_start, (int32_t)(sp2 - path_start));
    }

    /* Parse headers */
    const char *hdr_start = strstr(data, "\r\n");
    const char *body_sep = strstr(data, "\r\n\r\n");
    if (hdr_start) hdr_start += 2;

    while (hdr_start && hdr_start < body_sep) {
        const char *eol = strstr(hdr_start, "\r\n");
        if (!eol) break;

        const char *colon = memchr(hdr_start, ':', eol - hdr_start);
        if (colon) {
            int32_t klen = (int32_t)(colon - hdr_start);
            const char *vstart = colon + 1;
            while (*vstart == ' ') vstart++;
            int32_t vlen = (int32_t)(eol - vstart);

            EzString key = ez_string_new(arena, hdr_start, klen);
            EzString val = ez_string_new(arena, vstart, vlen);
            ez_map_set(arena, &req->headers, &key, &val);
        }
        hdr_start = eol + 2;
    }

    /* Body */
    if (body_sep) {
        const char *bstart = body_sep + 4;
        int32_t blen = (int32_t)(data_len - (bstart - data));
        if (blen > 0) {
            req->body = ez_string_new(arena, bstart, blen);
        }
    }

    return true;
}

/* Connection handler — runs in its own thread */
typedef struct {
    int client_fd;
    EzRouter *router;
} ConnCtx;

static void *handle_connection(void *arg) {
    ConnCtx *ctx = (ConnCtx *)arg;
    EzArena *arena = ez_arena_create(EZ_SERVER_REQUEST_ARENA); /* 64KB per request */

    /* Receive request data */
    char buf[EZ_SERVER_BUF_SIZE];
    ssize_t n = recv(ctx->client_fd, buf, sizeof(buf) - 1, 0);
    if (n <= 0) {
        close(ctx->client_fd);
        free(ctx);
        ez_arena_destroy(arena, __FILE__, __LINE__);
        free(arena);
        return NULL;
    }
    buf[n] = '\0';

    /* Parse request */
    EzRequest req;
    req.body = (EzString){"", 0};
    req.query = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 8);
    req.headers = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 16);
    req.params = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 8);

    EzResponse resp;
    resp.status = 404;
    resp.body = ez_string_new(arena, "Not Found", sizeof("Not Found") - 1);
    resp.content_type = ez_string_new(arena, "text/plain", sizeof("text/plain") - 1);

    if (parse_request(arena, buf, (int)n, &req)) {
        /* Reject oversized method or path before copying into fixed stack buffers */
        if (req.method.len >= EZ_HTTP_METHOD_BUF || req.path.len >= EZ_HTTP_PATH_BUF_SERVER) {
            const char *err = "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\nConnection: close\r\n\r\n";
            send(ctx->client_fd, err, strlen(err), 0);
            close(ctx->client_fd);
            free(ctx);
            ez_arena_destroy(arena, __FILE__, __LINE__);
            free(arena);
            return NULL;
        }

        /* Find matching route */
        char method_buf[EZ_HTTP_METHOD_BUF], path_buf[EZ_HTTP_PATH_BUF_SERVER];
        memcpy(method_buf, req.method.data, req.method.len);
        method_buf[req.method.len] = '\0';
        memcpy(path_buf, req.path.data, req.path.len);
        path_buf[req.path.len] = '\0';

        for (int i = 0; i < ctx->router->count; i++) {
            EzRoute *route = &ctx->router->routes[i];
            if (strcmp(route->method, method_buf) == 0 &&
                match_route(route->pattern, path_buf, arena, &req.params)) {
                resp = route->handler(req);
                break;
            }
        }

        /* Run middleware */
        for (int i = 0; i < ctx->router->mw_count; i++) {
            ctx->router->middlewares[i](&req, &resp);
        }
    }

    /* Build HTTP response with optional CORS headers */
    char cors_hdrs[EZ_SERVER_CORS_BUF];
    cors_hdrs[0] = '\0';
    if (ctx->router->cors_origin) {
        snprintf(cors_hdrs, sizeof(cors_hdrs),
            "Access-Control-Allow-Origin: %s\r\n"
            "Access-Control-Allow-Methods: GET, POST, PUT, DELETE, PATCH, OPTIONS\r\n"
            "Access-Control-Allow-Headers: Content-Type, Authorization\r\n",
            ctx->router->cors_origin);
    }

    char resp_buf[EZ_SERVER_BUF_SIZE];
    int resp_len = snprintf(resp_buf, sizeof(resp_buf),
        "HTTP/1.1 %d OK\r\n"
        "Content-Type: %.*s\r\n"
        "Content-Length: %d\r\n"
        "%s"
        "Connection: close\r\n"
        "\r\n"
        "%.*s",
        (int)resp.status,
        (int)resp.content_type.len, resp.content_type.data,
        (int)resp.body.len,
        cors_hdrs,
        (int)resp.body.len, resp.body.data);

    send(ctx->client_fd, resp_buf, resp_len, 0);
    close(ctx->client_fd);
    free(ctx);
    ez_arena_destroy(arena, __FILE__, __LINE__);
    free(arena);
    return NULL;
}

void ez_server_listen(int64_t port, EzRouter *r) {
    EzArena *arena = get_server_arena();
    EzSocket listener = ez_net_listen(arena, port);
    if (listener.fd < 0) {
        fprintf(stderr, "server: failed to listen on port %d\n", (int)port);
        return;
    }

    printf("EZ server listening on port %d\n", (int)port);
    fflush(stdout);

    while (1) {
        EzSocket client = ez_net_accept(arena, listener);
        if (client.fd < 0) continue;

        ConnCtx *ctx = malloc(sizeof(ConnCtx));
        if (!ctx) { close(client.fd); continue; }
        ctx->client_fd = client.fd;
        ctx->router = r;

        pthread_t thread;
        if (pthread_create(&thread, NULL, handle_connection, ctx) != 0) {
            fprintf(stderr, "server: failed to create thread\n");
            free(ctx);
            close(client.fd);
            continue;
        }
        pthread_detach(thread);
    }
}

/* Response builders */
EzResponse ez_server_text(int64_t status, EzString body) {
    return (EzResponse){status, body, ez_string_lit("text/plain")};
}

EzResponse ez_server_json(int64_t status, EzString body) {
    return (EzResponse){status, body, ez_string_lit("application/json")};
}

EzResponse ez_server_html(int64_t status, EzString body) {
    return (EzResponse){status, body, ez_string_lit("text/html")};
}

EzResponse ez_server_redirect(int64_t status, EzString location) {
    /* For redirect, body contains Location header value */
    return (EzResponse){status, location, ez_string_lit("text/plain")};
}
