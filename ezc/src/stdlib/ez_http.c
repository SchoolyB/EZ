/*
 * ez_http.c - @http module implementation
 *
 * Minimal HTTP/1.1 client using raw POSIX sockets.
 * No TLS support in this initial version — HTTP only.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_http.h"
#include "ez_net.h"
#include <string.h>
#include <stdio.h>
#include <stdlib.h>

/* Helper: null-terminate an EzString */
static const char *http_cstr(EzString s, char *buf, size_t bufsz) {
    size_t len = (size_t)s.len < bufsz - 1 ? (size_t)s.len : bufsz - 1;
    memcpy(buf, s.data, len);
    buf[len] = '\0';
    return buf;
}

/* Parse a URL into host, port, and path components */
static bool parse_url(const char *url, char *host, size_t host_sz,
                      int *port, char *path, size_t path_sz) {
    /* Skip http:// prefix */
    const char *p = url;
    if (strncmp(p, "http://", 7) == 0) {
        p += 7;
    } else if (strncmp(p, "https://", 8) == 0) {
        /* HTTPS not supported yet */
        p += 8;
    }

    /* Extract host[:port] */
    const char *slash = strchr(p, '/');
    const char *colon = strchr(p, ':');

    if (colon && (!slash || colon < slash)) {
        /* host:port */
        size_t hlen = (size_t)(colon - p);
        if (hlen >= host_sz) hlen = host_sz - 1;
        memcpy(host, p, hlen);
        host[hlen] = '\0';
        *port = atoi(colon + 1);
    } else {
        /* host only */
        size_t hlen = slash ? (size_t)(slash - p) : strlen(p);
        if (hlen >= host_sz) hlen = host_sz - 1;
        memcpy(host, p, hlen);
        host[hlen] = '\0';
        *port = 80;
    }

    /* Path */
    if (slash) {
        size_t plen = strlen(slash);
        if (plen >= path_sz) plen = path_sz - 1;
        memcpy(path, slash, plen);
        path[plen] = '\0';
    } else {
        path[0] = '/';
        path[1] = '\0';
    }

    return host[0] != '\0';
}

/* Parse HTTP response: extract status code, headers, body */
static EzHttpResponse parse_response(EzArena *arena, const char *data, int data_len) {
    EzHttpResponse resp;
    resp.status = 0;
    resp.body = (EzString){"", 0};
    resp.headers = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 16);

    if (data_len < 12) return resp;

    /* Parse status line: HTTP/1.1 200 OK */
    if (strncmp(data, "HTTP/", 5) == 0) {
        const char *sp = strchr(data, ' ');
        if (sp) resp.status = atoi(sp + 1);
    }

    /* Find header/body separator */
    const char *body_start = NULL;
    const char *hdr_end = strstr(data, "\r\n\r\n");
    if (hdr_end) {
        body_start = hdr_end + 4;
    } else {
        hdr_end = strstr(data, "\n\n");
        if (hdr_end) body_start = hdr_end + 2;
    }

    /* Parse headers */
    const char *line = strchr(data, '\n');
    if (line) line++;
    while (line && line < hdr_end) {
        const char *eol = strchr(line, '\n');
        if (!eol || eol > hdr_end) break;

        const char *colon = memchr(line, ':', (size_t)(eol - line));
        if (colon) {
            int32_t klen = (int32_t)(colon - line);
            const char *vstart = colon + 1;
            while (*vstart == ' ') vstart++;
            int32_t vlen = (int32_t)(eol - vstart);
            if (vlen > 0 && vstart[vlen - 1] == '\r') vlen--;

            EzString key = ez_string_new(arena, line, klen);
            EzString val = ez_string_new(arena, vstart, vlen);
            ez_map_set(arena, &resp.headers, &key, &val);
        }
        line = eol + 1;
    }

    /* Body */
    if (body_start) {
        int32_t body_len = (int32_t)(data_len - (int)(body_start - data));
        if (body_len > 0) {
            resp.body = ez_string_new(arena, body_start, body_len);
        }
    }

    return resp;
}

/* Core HTTP request function */
static EzHttpResponse do_request(EzArena *arena, const char *method,
                                  EzString url, EzString body) {
    EzHttpResponse err_resp;
    err_resp.status = 0;
    err_resp.body = (EzString){"", 0};
    err_resp.headers = ez_map_new(arena, sizeof(EzString), sizeof(EzString), 4);

    char url_buf[4096];
    http_cstr(url, url_buf, sizeof(url_buf));

    char host[256], path[2048];
    int port;
    if (!parse_url(url_buf, host, sizeof(host), &port, path, sizeof(path))) {
        err_resp.body = ez_string_new(arena, "invalid URL", 11);
        return err_resp;
    }

    /* Connect */
    EzString host_str = ez_string_new(arena, host, (int32_t)strlen(host));
    EzSocket sock = ez_net_dial(arena, host_str, port);
    if (sock.fd < 0) {
        err_resp.body = ez_string_new(arena, "connection failed", 17);
        return err_resp;
    }

    /* Set 10s timeout */
    ez_net_set_timeout(sock, 10000);

    /* Build request */
    char req[8192];
    int req_len;

    if (body.data && body.len > 0) {
        req_len = snprintf(req, sizeof(req),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Content-Length: %d\r\n"
            "Content-Type: application/x-www-form-urlencoded\r\n"
            "Connection: close\r\n"
            "\r\n"
            "%.*s",
            method, path, host, (int)body.len, (int)body.len, body.data);
    } else {
        req_len = snprintf(req, sizeof(req),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Connection: close\r\n"
            "\r\n",
            method, path, host);
    }

    /* Send */
    EzString req_str = {req, (int32_t)req_len};
    ez_net_send(sock, req_str);

    /* Receive response (up to 1MB) */
    char resp_buf[1048576];
    int total = 0;
    while (total < (int)sizeof(resp_buf) - 1) {
        EzString chunk = ez_net_recv(arena, sock, sizeof(resp_buf) - total);
        if (chunk.len <= 0) break;
        memcpy(resp_buf + total, chunk.data, (size_t)chunk.len);
        total += chunk.len;
    }
    resp_buf[total] = '\0';

    ez_net_close(sock);

    return parse_response(arena, resp_buf, total);
}

EzHttpResponse ez_http_get(EzArena *arena, EzString url) {
    return do_request(arena, "GET", url, (EzString){"", 0});
}

EzHttpResponse ez_http_post(EzArena *arena, EzString url, EzString body) {
    return do_request(arena, "POST", url, body);
}

EzHttpResponse ez_http_put(EzArena *arena, EzString url, EzString body) {
    return do_request(arena, "PUT", url, body);
}

EzHttpResponse ez_http_delete(EzArena *arena, EzString url) {
    return do_request(arena, "DELETE", url, (EzString){"", 0});
}

EzHttpResponse ez_http_head(EzArena *arena, EzString url) {
    return do_request(arena, "HEAD", url, (EzString){"", 0});
}

EzHttpResponse ez_http_patch(EzArena *arena, EzString url, EzString body) {
    return do_request(arena, "PATCH", url, body);
}
