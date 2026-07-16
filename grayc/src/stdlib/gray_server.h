/*
 * gray_server.h - @server module for EZ
 *
 * HTTP server with routing, path params, and handler functions.
 * Uses @net for sockets and @threads for concurrency.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_SERVER_H
#define GRAY_SERVER_H

#include "../runtime/gray_runtime.h"
#include "../runtime/gray_map.h"

/*@man HttpRequest
 *@module server
 *@group Types
 *@kind type
 *@field method string
 *@field path string
 *@field body string
 *@field query map[string:string]
 *@field headers map[string:string]
 *@field params map[string:string]
 *@desc The request object passed to every handler function.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       mut id = req.params["id"]
 *       return server.json(200, id)
 *   }
 *@end
 */
/* Request struct passed to handlers */
typedef struct {
    EzString method;
    EzString path;
    EzString body;
    EzMap query;       /* map[string:string] */
    EzMap headers;     /* map[string:string] */
    EzMap params;      /* map[string:string] — path params from :id patterns */
} EzRequest;

/*@man HttpResponse
 *@module server
 *@group Types
 *@kind type
 *@field status int
 *@field body string
 *@field content_type string
 *@desc The response object returned by handler functions. Build using server.text(), server.json(), server.html(), or server.redirect() rather than constructing directly.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       return server.text(200, "hello")
 *   }
 *@end
 */
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

/* Middleware function pointer */
typedef void (*EzMiddleware)(EzRequest *req, EzResponse *resp);

/*@man Router
 *@module server
 *@group Types
 *@kind type
 *@desc The router object returned by add_router(). Pass it to add_route(), cors(), use(), and listen(). Its internal fields are not user-accessible.
 *@example
 *   import @server
 *   mut r = server.add_router()
 *   server.add_route(r, "GET", "/", ()home)
 *   server.listen(r, 8080)
 *@end
 */
/* Router — holds registered routes */
typedef struct {
    EzRoute *routes;
    int count;
    int capacity;
    const char *cors_origin;    /* NULL = no CORS */
    EzMiddleware *middlewares;
    int mw_count;
    int mw_capacity;
} EzRouter;

/*@man add_router
 *@module server
 *@group Routing
 *@sig add_router() -> Router
 *@desc Creates and returns a new router. Pass the router to add_route(), cors(), use(), and listen().
 *@example
 *   import @server
 *   mut r = server.add_router()
 *@end
 */
/* Create a new router */
EzRouter gray_server_router(void);

/*@man add_route
 *@module server
 *@group Routing
 *@sig add_route(router Router, method string, path string, handler func)
 *@desc Registers a route on the router. method is an HTTP verb ("GET", "POST", etc.). path may contain :param segments for dynamic matching. handler is a function that takes HttpRequest and returns HttpResponse.
 *@example
 *   import @server
 *   do home(req HttpRequest) -> HttpResponse {
 *       return server.text(200, "hello")
 *   }
 *   mut r = server.add_router()
 *   server.add_route(r, "GET", "/", ()home)
 *@end
 */
/* Register a route: server.route(router, method, pattern, handler) */
void gray_server_route(EzRouter *r, EzString method, EzString pattern,
                     EzResponse (*handler)(EzRequest));

/*@man listen
 *@module server
 *@group Routing
 *@sig listen(router Router, port int)
 *@desc Starts the HTTP server on the given port. Blocks until the process is killed.
 *@example
 *   import @server
 *   mut r = server.add_router()
 *   server.add_route(r, "GET", "/", ()home)
 *   server.listen(r, 8080)
 *@end
 */
/* Start listening — blocks until killed */
void gray_server_listen(int64_t port, EzRouter *r);
void gray_server_listen_host(int64_t port, EzString host, EzRouter *r);

/*@man cors
 *@module server
 *@group Routing
 *@sig cors(router Router, origin string)
 *@desc Enables CORS on the router for the given origin. Use "*" to allow all origins.
 *@example
 *   import @server
 *   mut r = server.add_router()
 *   server.cors(r, "*")
 *@end
 */
/* Enable CORS with the given origin (e.g. "*" or "http://example.com") */
void gray_server_cors(EzRouter *r, EzString origin);

/*@man use
 *@module server
 *@group Routing
 *@sig use(router Router, middleware func)
 *@desc Registers a middleware function on the router. Middleware runs before each handler and can inspect or modify the request and response.
 *@example
 *   import @server
 *   mut r = server.add_router()
 *   server.use(r, ()my_logger)
 *@end
 */
/* Register a middleware function */
void gray_server_use(EzRouter *r, EzMiddleware fn);

/*@man text
 *@module server
 *@group Response Builders
 *@sig text(status int, body string) -> HttpResponse
 *@desc Returns an HttpResponse with Content-Type: text/plain and the given status code and body.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       return server.text(200, "hello")
 *   }
 *@end
 */
/* Response builders */
EzResponse gray_server_text(int64_t status, EzString body);

/*@man json
 *@module server
 *@group Response Builders
 *@sig json(status int, body string) -> HttpResponse
 *@desc Returns an HttpResponse with Content-Type: application/json and the given status code and body.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       return server.json(200, "{\"ok\": true}")
 *   }
 *@end
 */
EzResponse gray_server_json(int64_t status, EzString body);

/*@man html
 *@module server
 *@group Response Builders
 *@sig html(status int, body string) -> HttpResponse
 *@desc Returns an HttpResponse with Content-Type: text/html and the given status code and body.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       return server.html(200, "<h1>Hello</h1>")
 *   }
 *@end
 */
EzResponse gray_server_html(int64_t status, EzString body);

/*@man redirect
 *@module server
 *@group Response Builders
 *@sig redirect(status int, url string) -> HttpResponse
 *@desc Returns an HttpResponse that redirects to url. Use 301 for permanent or 302 for temporary redirects.
 *@example
 *   import @server
 *   do handler(req HttpRequest) -> HttpResponse {
 *       return server.redirect(302, "/new-path")
 *   }
 *@end
 */
EzResponse gray_server_redirect(int64_t status, EzString location);

#endif
