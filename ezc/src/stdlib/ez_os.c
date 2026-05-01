/*
 * ez_os.c - @os module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_os.h"
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <limits.h>

#define EZ_HOSTNAME_BUF 256

#ifndef PATH_MAX
#define PATH_MAX 4096
#endif

static int _os_argc = 0;
static char **_os_argv = NULL;

void ez_os_init(int argc, char **argv) {
    _os_argc = argc;
    _os_argv = argv;
}

EzArray ez_os_args(EzArena *arena) {
    EzArray arr = ez_array_new(arena, sizeof(EzString), _os_argc > 0 ? _os_argc : 1);
    for (int i = 0; i < _os_argc; i++) {
        EzString s = ez_string_new(arena, _os_argv[i], (int32_t)strlen(_os_argv[i]));
        ez_array_push(arena, &arr, &s);
    }
    return arr;
}

EzString ez_os_get_env(EzArena *arena, EzString name) {
    const char *val = getenv(name.data);
    if (!val) return ez_string_lit("");
    return ez_string_new(arena, val, (int32_t)strlen(val));
}

void ez_os_set_env(EzString name, EzString value) {
    setenv(name.data, value.data, 1);
}

EzString ez_os_cwd(EzArena *arena) {
    char buf[PATH_MAX];
    if (getcwd(buf, sizeof(buf))) {
        return ez_string_new(arena, buf, (int32_t)strlen(buf));
    }
    return ez_string_lit("");
}

EzString ez_os_hostname(EzArena *arena) {
    char buf[EZ_HOSTNAME_BUF];
    if (gethostname(buf, sizeof(buf)) == 0) {
        return ez_string_new(arena, buf, (int32_t)strlen(buf));
    }
    return ez_string_lit("");
}

int64_t ez_os_current_os(void) {
#ifdef __APPLE__
    return 0; /* MAC_OS */
#elif defined(__linux__)
    return 1; /* LINUX */
#elif defined(_WIN32)
    return 2; /* WINDOWS */
#else
    return 3; /* OTHER */
#endif
}

EzString ez_os_arch(void) {
#if defined(__aarch64__) || defined(__arm64__)
    return ez_string_lit("arm64");
#elif defined(__x86_64__) || defined(__amd64__)
    return ez_string_lit("x86_64");
#elif defined(__i386__)
    return ez_string_lit("x86");
#elif defined(__arm__)
    return ez_string_lit("arm");
#else
    return ez_string_lit("unknown");
#endif
}

int64_t ez_os_pid(void) {
    return (int64_t)getpid();
}

