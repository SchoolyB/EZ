/*
 * ez_http.h - @http module for EZC
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

/* HTTP response: status code, body, headers */
typedef struct {
    int64_t status;
    EzString body;
    EzMap headers;
} EzHttpResponse;

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
