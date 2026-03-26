/*
 * ez_server.h - @server module for EZC
 *
 * HTTP server with routing, path params, and handler functions.
 * Uses @net for sockets and @threads for concurrency.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_SERVER_H
#define EZ_SERVER_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_map.h"

/* Request struct passed to handlers */
typedef struct {
    EzString method;
    EzString path;
    EzString body;
    EzMap query;       /* map[string:string] */
    EzMap headers;     /* map[string:string] */
    EzMap params;      /* map[string:string] — path params from :id patterns */
} EzRequest;

/* Response struct returned by handlers */
typedef struct {
    int64_t status;
    EzString body;
    EzString content_type;
} EzResponse;

/* Route entry */
typedef struct {
    const char *method;
    const char *pattern;
    EzResponse (*handler)(EzRequest);
} EzRoute;

/* Router — holds registered routes */
typedef struct {
    EzRoute *routes;
    int count;
    int capacity;
} EzRouter;

/* Create a new router */
EzRouter ez_server_router(void);

/* Register a route: server.route(router, method, pattern, handler) */
void ez_server_route(EzRouter *r, EzString method, EzString pattern,
                     EzResponse (*handler)(EzRequest));

/* Start listening — blocks until killed */
void ez_server_listen(int64_t port, EzRouter *r);

/* Response builders */
EzResponse ez_server_text(int64_t status, EzString body);
EzResponse ez_server_json(int64_t status, EzString body);
EzResponse ez_server_html(int64_t status, EzString body);
EzResponse ez_server_redirect(int64_t status, EzString location);

#endif
