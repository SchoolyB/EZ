/*
 * ez_strconv.c - strconv module for EZ (string/type conversions)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_strconv.h"
#include <ctype.h>
#include <errno.h>
#include <inttypes.h>
#include <strings.h>

#define STRCONV_BUF_SIZE 64

/* --- Panicking conversions --- */

int64_t ez_strconv_to_int(EzString s, int base) {
    if (base < 2 || base > 36) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_int: invalid base %d (must be 2-36)\n", base);
        exit(1);
    }
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_int: cannot convert \"%s\" to int (base %d)\n", buf, base);
        exit(1);
    }
    char *end = NULL;
    errno = 0;
    int64_t result = strtoll(buf, &end, base);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_int: cannot convert \"%s\" to int (base %d)\n", buf, base);
        exit(1);
    }
    return result;
}

uint64_t ez_strconv_to_uint(EzString s, int base) {
    if (base < 2 || base > 36) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_uint: invalid base %d (must be 2-36)\n", base);
        exit(1);
    }
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_uint: cannot convert \"%s\" to uint (base %d)\n", buf, base);
        exit(1);
    }
    /* Reject negative numbers */
    for (int i = 0; i < len; i++) {
        if (buf[i] == '-') {
            fflush(stdout);
            fprintf(stderr, "panic: strconv.to_uint: cannot convert \"%s\" to uint (negative)\n", buf);
            exit(1);
        }
        if (!isspace((unsigned char)buf[i])) break;
    }
    char *end = NULL;
    errno = 0;
    uint64_t result = strtoull(buf, &end, base);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_uint: cannot convert \"%s\" to uint (base %d)\n", buf, base);
        exit(1);
    }
    return result;
}

double ez_strconv_to_float(EzString s) {
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_float: cannot convert \"%s\" to float\n", buf);
        exit(1);
    }
    char *end = NULL;
    errno = 0;
    double result = strtod(buf, &end);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        fflush(stdout);
        fprintf(stderr, "panic: strconv.to_float: cannot convert \"%s\" to float\n", buf);
        exit(1);
    }
    return result;
}

bool ez_strconv_to_bool(EzString s) {
    if (s.len == 4 && strncasecmp(s.data, "true", 4) == 0) {
        return true;
    }
    if (s.len == 5 && strncasecmp(s.data, "false", 5) == 0) {
        return false;
    }
    fflush(stdout);
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    fprintf(stderr, "panic: strconv.to_bool: cannot convert \"%s\" to bool\n", buf);
    exit(1);
}

/* --- Fallible conversions (result versions) --- */

EzResult_int ez_strconv_to_int_result(EzString s, int base) {
    if (base < 2 || base > 36) {
        EzString msg = ez_string_lit("invalid base for integer conversion (must be 2-36)");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_int){0, err};
    }
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        EzString msg = ez_string_lit("cannot convert string to int");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_int){0, err};
    }
    char *end = NULL;
    errno = 0;
    int64_t result = strtoll(buf, &end, base);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        EzString msg = ez_string_lit("cannot convert string to int");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_int){0, err};
    }
    return (EzResult_int){result, NULL};
}

EzResult_uint ez_strconv_to_uint_result(EzString s, int base) {
    if (base < 2 || base > 36) {
        EzString msg = ez_string_lit("invalid base for integer conversion (must be 2-36)");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_uint){0, err};
    }
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        EzString msg = ez_string_lit("cannot convert string to uint");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_uint){0, err};
    }
    /* Reject negative numbers */
    for (int i = 0; i < len; i++) {
        if (buf[i] == '-') {
            EzString msg = ez_string_lit("cannot convert negative string to uint");
            EzError *err = ez_error_new(ez_default_arena, msg);
            return (EzResult_uint){0, err};
        }
        if (!isspace((unsigned char)buf[i])) break;
    }
    char *end = NULL;
    errno = 0;
    uint64_t result = strtoull(buf, &end, base);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        EzString msg = ez_string_lit("cannot convert string to uint");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_uint){0, err};
    }
    return (EzResult_uint){result, NULL};
}

EzResult_float ez_strconv_to_float_result(EzString s) {
    char buf[STRCONV_BUF_SIZE];
    int len = s.len < (int32_t)sizeof(buf) - 1 ? s.len : (int32_t)sizeof(buf) - 1;
    memcpy(buf, s.data, (size_t)len);
    buf[len] = '\0';
    if (len > 0 && isspace((unsigned char)buf[0])) {
        EzString msg = ez_string_lit("cannot convert string to float");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_float){0.0, err};
    }
    char *end = NULL;
    errno = 0;
    double result = strtod(buf, &end);
    if (end == buf || *end != '\0' || errno == ERANGE) {
        EzString msg = ez_string_lit("cannot convert string to float");
        EzError *err = ez_error_new(ez_default_arena, msg);
        return (EzResult_float){0.0, err};
    }
    return (EzResult_float){result, NULL};
}

EzResult_bool ez_strconv_to_bool_result(EzString s) {
    if (s.len == 4 && strncasecmp(s.data, "true", 4) == 0) {
        return (EzResult_bool){true, NULL};
    }
    if (s.len == 5 && strncasecmp(s.data, "false", 5) == 0) {
        return (EzResult_bool){false, NULL};
    }
    EzString msg = ez_string_lit("cannot convert string to bool");
    EzError *err = ez_error_new(ez_default_arena, msg);
    return (EzResult_bool){false, err};
}

/* --- Type to string conversions --- */

EzString ez_strconv_from_int(EzArena *arena, int64_t n) {
    char buf[STRCONV_BUF_SIZE];
    int len = snprintf(buf, sizeof(buf), "%" PRId64, n);
    char *data = (char *)ez_arena_alloc(arena, (size_t)len + 1);
    memcpy(data, buf, (size_t)len + 1);
    return (EzString){data, (int32_t)len};
}

EzString ez_strconv_from_uint(EzArena *arena, uint64_t n) {
    char buf[STRCONV_BUF_SIZE];
    int len = snprintf(buf, sizeof(buf), "%" PRIu64, n);
    char *data = (char *)ez_arena_alloc(arena, (size_t)len + 1);
    memcpy(data, buf, (size_t)len + 1);
    return (EzString){data, (int32_t)len};
}

EzString ez_strconv_from_float(EzArena *arena, double f) {
    char buf[STRCONV_BUF_SIZE];
    int len = snprintf(buf, sizeof(buf), "%g", f);
    char *data = (char *)ez_arena_alloc(arena, (size_t)len + 1);
    memcpy(data, buf, (size_t)len + 1);
    return (EzString){data, (int32_t)len};
}

EzString ez_strconv_from_bool(bool b) {
    if (b) return ez_string_lit("true");
    return ez_string_lit("false");
}

/* --- Query functions --- */

bool ez_strconv_is_numeric(EzString s) {
    if (s.len == 0) return false;
    int start = 0;
    if (s.data[0] == '-' || s.data[0] == '+') {
        start = 1;
        if (s.len == 1) return false;
    }
    bool has_dot = false;
    bool has_digit = false;
    for (int i = start; i < s.len; i++) {
        if (s.data[i] == '.') {
            if (has_dot) return false;
            has_dot = true;
        } else if (isdigit((unsigned char)s.data[i])) {
            has_digit = true;
        } else {
            return false;
        }
    }
    return has_digit;
}

bool ez_strconv_is_integer(EzString s) {
    if (s.len == 0) return false;
    int start = 0;
    if (s.data[0] == '-' || s.data[0] == '+') {
        start = 1;
        if (s.len == 1) return false;
    }
    for (int i = start; i < s.len; i++) {
        if (!isdigit((unsigned char)s.data[i])) return false;
    }
    return true;
}
