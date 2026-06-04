/*
 * ez_http.h - @http module for EZ
 *
 * HTTP client using raw sockets (no libcurl dependency).
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_HTTP_H
#define EZ_HTTP_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_map.h"

/*@man HttpResponse
 *@module http
 *@group Types
 *@kind type
 *@field status int
 *@field body string
 *@field headers map[string:string]
 *@desc The response object returned by all http request functions. Also available when the server module is imported.
 *@example
 *   import @http
 *   mut resp, err = http.get("http://example.com")
 *   println(resp.status)
 *   println(resp.body)
 *@end
 */
/* HTTP response: status code, body, headers */
typedef struct {
    int64_t status;
    EzString body;
    EzMap headers;
} EzHttpResponse;

/*@man get
 *@module http
 *@group Requests
 *@sig get(url string) -> (HttpResponse, Error)
 *@desc Sends an HTTP GET request to url and returns the response. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.get("http://example.com")
 *   println(resp.status)
 *   println(resp.body)
 *@end
 */

/*@man post
 *@module http
 *@group Requests
 *@sig post(url string, body string) -> (HttpResponse, Error)
 *@desc Sends an HTTP POST request to url with the given body. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.post("http://example.com/api", "{\"key\": \"value\"}")
 *   println(resp.status)
 *@end
 */

/*@man put
 *@module http
 *@group Requests
 *@sig put(url string, body string) -> (HttpResponse, Error)
 *@desc Sends an HTTP PUT request to url with the given body. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.put("http://example.com/api/1", "{\"name\": \"Alice\"}")
 *   println(resp.status)
 *@end
 */

/*@man patch
 *@module http
 *@group Requests
 *@sig patch(url string, body string) -> (HttpResponse, Error)
 *@desc Sends an HTTP PATCH request to url with the given body. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.patch("http://example.com/api/1", "{\"age\": 30}")
 *   println(resp.status)
 *@end
 */

/*@man delete
 *@module http
 *@group Requests
 *@sig delete(url string) -> (HttpResponse, Error)
 *@desc Sends an HTTP DELETE request to url. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.delete("http://example.com/api/1")
 *   println(resp.status)
 *@end
 */

/*@man head
 *@module http
 *@group Requests
 *@sig head(url string) -> (HttpResponse, Error)
 *@desc Sends an HTTP HEAD request to url. Returns only headers with no body. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @http
 *   mut resp, err = http.head("http://example.com")
 *   println(resp.status)
 *@end
 */

/* HTTP methods */
EzHttpResponse ez_http_get(EzArena *arena, EzString url);
EzHttpResponse ez_http_post(EzArena *arena, EzString url, EzString body);
EzHttpResponse ez_http_put(EzArena *arena, EzString url, EzString body);
EzHttpResponse ez_http_delete(EzArena *arena, EzString url);
EzHttpResponse ez_http_head(EzArena *arena, EzString url);
EzHttpResponse ez_http_patch(EzArena *arena, EzString url, EzString body);

/* _result variants */
typedef struct { EzHttpResponse v0; EzError *v1; } EzResult_http;

EzResult_http ez_http_get_result(EzArena *arena, EzString url);
EzResult_http ez_http_post_result(EzArena *arena, EzString url, EzString body);
EzResult_http ez_http_put_result(EzArena *arena, EzString url, EzString body);
EzResult_http ez_http_delete_result(EzArena *arena, EzString url);
EzResult_http ez_http_head_result(EzArena *arena, EzString url);
EzResult_http ez_http_patch_result(EzArena *arena, EzString url, EzString body);

#endif
