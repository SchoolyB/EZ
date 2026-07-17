/*
 * gray_http.c — Implementation of the http stdlib module.
 * Minimal HTTP/1.1 client using raw POSIX sockets with support for
 * GET, POST, PUT, DELETE, and custom headers. No TLS.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_http.h"
#include "gray_net.h"
#include <string.h>
#include <stdio.h>
#include <stdlib.h>

#define GRAY_HTTP_DEFAULT_PORT    80
#define GRAY_HTTP_TIMEOUT_MS      10000
#define GRAY_HTTP_URL_BUF         4096
#define GRAY_HTTP_HOST_BUF        256
#define GRAY_HTTP_PATH_BUF        2048
#define GRAY_HTTP_HDR_BUF         4096
#define GRAY_HTTP_RESP_BUF        1048576
#define GRAY_HTTP_MIN_RESP_LEN    12

/* Helper: null-terminate an GrayString */
static const char *http_cstr(GrayString s, char *buf, size_t bufsz) {
    size_t len = (size_t)s.len < bufsz - 1 ? (size_t)s.len : bufsz - 1;
    memcpy(buf, s.data, len);
    buf[len] = '\0';
    return buf;
}

/* Parse a URL into host, port, and path components */
static bool parse_url(const char *url, char *host, size_t host_sz,
                      int *port, char *path, size_t path_sz) {
    /* Require http:// or https:// scheme */
    const char *p = url;
    if (strncmp(p, "http://", 7) == 0) {
        p += 7;
    } else if (strncmp(p, "https://", 8) == 0) {
        return false;
    } else {
        return false;
    }

    /* Reject empty host or host containing characters that would break HTTP headers */
    if (*p == '\0' || *p == '/' || *p == ':') return false;
    for (const char *c = p; *c && *c != '/' && *c != ':'; c++) {
        if (*c == ' ' || *c == '\t' || *c == '\r' || *c == '\n') return false;
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
        *port = GRAY_HTTP_DEFAULT_PORT;
    }

    /* Path — reject CR/LF to prevent header injection via request line */
    if (slash) {
        size_t plen = strlen(slash);
        if (plen >= path_sz) plen = path_sz - 1;
        memcpy(path, slash, plen);
        path[plen] = '\0';
        for (size_t i = 0; i < plen; i++) {
            if (path[i] == '\r' || path[i] == '\n') return false;
        }
    } else {
        path[0] = '/';
        path[1] = '\0';
    }

    return host[0] != '\0';
}

/* Parse HTTP response: extract status code, headers, body */
static GrayHttpResponse parse_response(GrayArena *arena, const char *data, int data_len) {
    GrayHttpResponse resp;
    resp.status = 0;
    resp.body = (GrayString){"", 0};
    resp.headers = gray_map_new(arena, sizeof(GrayString), sizeof(GrayString), 16);

    if (data_len < GRAY_HTTP_MIN_RESP_LEN) return resp;

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

            GrayString key = gray_string_new(arena, line, klen);
            GrayString val = gray_string_new(arena, vstart, vlen);
            gray_map_set(arena, &resp.headers, &key, &val);
        }
        line = eol + 1;
    }

    /* Body */
    if (body_start) {
        int32_t body_len = (int32_t)(data_len - (int)(body_start - data));
        if (body_len > 0) {
            resp.body = gray_string_new(arena, body_start, body_len);
        }
    }

    return resp;
}

/* Core HTTP request function */
static GrayHttpResponse do_request(GrayArena *arena, const char *method,
                                  GrayString url, GrayString body, GrayMap *custom_headers) {
    GrayHttpResponse err_resp;
    err_resp.status = 0;
    err_resp.body = (GrayString){"", 0};
    err_resp.headers = gray_map_new(arena, sizeof(GrayString), sizeof(GrayString), 4);

    char url_buf[GRAY_HTTP_URL_BUF];
    http_cstr(url, url_buf, sizeof(url_buf));

    if (strncmp(url_buf, "https://", 8) == 0) {
        const char *detail = "https:// is not supported; use http://";
        err_resp.body = gray_string_new(arena, detail, (int32_t)strlen(detail));
        return err_resp;
    }

    char host[GRAY_HTTP_HOST_BUF], path[GRAY_HTTP_PATH_BUF];
    int port;
    if (!parse_url(url_buf, host, sizeof(host), &port, path, sizeof(path))) {
        const char *detail = "invalid URL: expected http:// scheme";
        err_resp.body = gray_string_new(arena, detail, (int32_t)strlen(detail));
        return err_resp;
    }

    /* Connect */
    GrayString host_str = gray_string_new(arena, host, (int32_t)strlen(host));
    GraySocket sock = gray_net_dial(arena, host_str, port);
    if (sock.fd < 0) {
        err_resp.body = gray_string_new(arena, "connection failed", 17);
        return err_resp;
    }

    /* Set 10s timeout */
    gray_net_set_timeout(sock, GRAY_HTTP_TIMEOUT_MS);

    /* Build headers — body is sent separately to avoid truncation */
    char hdr[GRAY_HTTP_HDR_BUF];
    int hdr_len;

    if (body.data && body.len > 0) {
        hdr_len = snprintf(hdr, sizeof(hdr),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Content-Length: %d\r\n"
            "Content-Type: application/x-www-form-urlencoded\r\n"
            "Connection: close\r\n",
            method, path, host, (int)body.len);
    } else {
        hdr_len = snprintf(hdr, sizeof(hdr),
            "%s %s HTTP/1.1\r\n"
            "Host: %s\r\n"
            "Connection: close\r\n",
            method, path, host);
    }

    /* Append custom headers */
    if (custom_headers && custom_headers->count > 0) {
        for (int32_t i = 0; i < custom_headers->order_len; i++) {
            int32_t slot = custom_headers->order[i];
            if (slot < 0) continue;
            GrayString *k = (GrayString *)gray_map_key_at(custom_headers, slot);
            GrayString *v = (GrayString *)gray_map_value_at(custom_headers, slot);
            if (!k || !v) continue;
            int n = snprintf(hdr + hdr_len, sizeof(hdr) - (size_t)hdr_len,
                "%.*s: %.*s\r\n", (int)k->len, k->data, (int)v->len, v->data);
            if (n > 0) hdr_len += n;
        }
    }

    /* Terminate headers */
    if ((size_t)hdr_len + 2 < sizeof(hdr)) {
        hdr[hdr_len++] = '\r';
        hdr[hdr_len++] = '\n';
    }

    if (hdr_len <= 0 || (size_t)hdr_len >= sizeof(hdr)) {
        gray_net_close(sock);
        const char *detail = "request URL or path too long";
        err_resp.body = gray_string_new(arena, detail, (int32_t)strlen(detail));
        return err_resp;
    }

    GrayString hdr_str = {hdr, (int32_t)hdr_len};
    gray_net_send(sock, hdr_str);
    if (body.data && body.len > 0) {
        gray_net_send(sock, body);
    }

    /* Receive response (up to 1MB) — heap-allocated to avoid stack overflow */
    char *resp_buf = malloc(GRAY_HTTP_RESP_BUF);
    if (!resp_buf) {
        gray_net_close(sock);
        return err_resp;
    }
    int total = 0;
    while (total < GRAY_HTTP_RESP_BUF - 1) {
        GrayString chunk = gray_net_recv(arena, sock, GRAY_HTTP_RESP_BUF - total);
        if (chunk.len <= 0) break;
        memcpy(resp_buf + total, chunk.data, (size_t)chunk.len);
        total += chunk.len;
    }
    resp_buf[total] = '\0';

    gray_net_close(sock);

    GrayHttpResponse result = parse_response(arena, resp_buf, total);
    free(resp_buf);
    return result;
}

GrayHttpResponse gray_http_get(GrayArena *arena, GrayString url, GrayMap *headers) {
    return do_request(arena, "GET", url, (GrayString){"", 0}, headers);
}

GrayHttpResponse gray_http_post(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return do_request(arena, "POST", url, body, headers);
}

GrayHttpResponse gray_http_put(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return do_request(arena, "PUT", url, body, headers);
}

GrayHttpResponse gray_http_delete(GrayArena *arena, GrayString url, GrayMap *headers) {
    return do_request(arena, "DELETE", url, (GrayString){"", 0}, headers);
}

GrayHttpResponse gray_http_head(GrayArena *arena, GrayString url, GrayMap *headers) {
    return do_request(arena, "HEAD", url, (GrayString){"", 0}, headers);
}

GrayHttpResponse gray_http_patch(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return do_request(arena, "PATCH", url, body, headers);
}

/* _result variants — status==0 indicates connection/request failure */

static GrayResult_http http_result(GrayArena *arena, GrayHttpResponse resp, const char *method, GrayString url) {
    GrayResult_http r;
    r.v0 = resp;
    if (resp.status == 0) {
        r.v1 = gray_error_new(arena, gray_string_format(arena, "HTTP %s failed: %.*s",
            method, resp.body.len, resp.body.data));
    } else {
        r.v1 = NULL;
    }
    return r;
}

GrayResult_http gray_http_get_result(GrayArena *arena, GrayString url, GrayMap *headers) {
    return http_result(arena, gray_http_get(arena, url, headers), "GET", url);
}

GrayResult_http gray_http_post_result(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return http_result(arena, gray_http_post(arena, url, body, headers), "POST", url);
}

GrayResult_http gray_http_put_result(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return http_result(arena, gray_http_put(arena, url, body, headers), "PUT", url);
}

GrayResult_http gray_http_delete_result(GrayArena *arena, GrayString url, GrayMap *headers) {
    return http_result(arena, gray_http_delete(arena, url, headers), "DELETE", url);
}

GrayResult_http gray_http_head_result(GrayArena *arena, GrayString url, GrayMap *headers) {
    return http_result(arena, gray_http_head(arena, url, headers), "HEAD", url);
}

GrayResult_http gray_http_patch_result(GrayArena *arena, GrayString url, GrayString body, GrayMap *headers) {
    return http_result(arena, gray_http_patch(arena, url, body, headers), "PATCH", url);
}
