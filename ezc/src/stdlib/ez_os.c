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
#include <sys/wait.h>
#include <errno.h>

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

EzOsExecResult ez_os_exec(EzArena *arena, EzString cmd, EzArray args) {
    EzOsExecResult fail = {0, ez_string_lit(""), false};

    /* Build null-terminated argv: argv[0] = cmd, argv[1..n] = args, argv[n+1] = NULL */
    int argc = 1 + args.len;
    char **argv = ez_arena_alloc(arena, sizeof(char *) * (size_t)(argc + 1));
    argv[0] = (char *)cmd.data;
    for (int i = 0; i < args.len; i++) {
        EzString s = EZ_ARRAY_GET(args, EzString, i);
        argv[1 + i] = (char *)s.data;
    }
    argv[argc] = NULL;

    int stderr_pipe[2];
    if (pipe(stderr_pipe) < 0) return fail;

    pid_t pid = fork();
    if (pid < 0) {
        close(stderr_pipe[0]);
        close(stderr_pipe[1]);
        return fail;
    }

    if (pid == 0) {
        /* Child: redirect stderr into pipe, stdout stays connected to terminal */
        close(stderr_pipe[0]);
        dup2(stderr_pipe[1], STDERR_FILENO);
        close(stderr_pipe[1]);
        execvp(cmd.data, argv);
        /* execvp failed */
        _exit(127);
    }

    /* Parent: read stderr from pipe */
    close(stderr_pipe[1]);

    char buf[4096];
    size_t total = 0;
    size_t cap = sizeof(buf);
    char *collected = ez_arena_alloc(arena, cap);

    ssize_t n;
    while ((n = read(stderr_pipe[0], buf, sizeof(buf))) > 0) {
        if (total + (size_t)n > cap) {
            size_t new_cap = cap * 2 + (size_t)n;
            char *grown = ez_arena_alloc(arena, new_cap);
            memcpy(grown, collected, total);
            collected = grown;
            cap = new_cap;
        }
        memcpy(collected + total, buf, (size_t)n);
        total += (size_t)n;
    }
    close(stderr_pipe[0]);

    int status = 0;
    waitpid(pid, &status, 0);

    int exit_code = 0;
    if (WIFEXITED(status)) {
        exit_code = WEXITSTATUS(status);
    } else if (WIFSIGNALED(status)) {
        exit_code = 128 + WTERMSIG(status);
    }

    /* exit_code 127 means execvp failed (command not found / bad path) */
    if (exit_code == 127) return fail;

    EzString stderr_str = ez_string_new(arena, collected, (int32_t)total);
    EzOsExecResult r = {(int64_t)exit_code, stderr_str, true};
    return r;
}

