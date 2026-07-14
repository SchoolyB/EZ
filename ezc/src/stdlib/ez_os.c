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
#include <signal.h>
#include <sys/select.h>
#include <sys/wait.h>
#include <errno.h>

#define EZ_HOSTNAME_BUF 256
#define EZ_EXEC_OUTPUT_MAX (64 * 1024 * 1024) /* 64 MiB per stream */

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
    // EzOsExecResult fail = {0, ez_string_lit(""), ez_string_lit(""), false};
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

    int stdout_pipe[2], stderr_pipe[2];
    if (pipe(stdout_pipe) < 0) return fail;
    if (pipe(stderr_pipe) < 0) {
        close(stdout_pipe[0]); close(stdout_pipe[1]);
        return fail;
    }

    pid_t pid = fork();
    if (pid < 0) {
        close(stdout_pipe[0]); close(stdout_pipe[1]);
        close(stderr_pipe[0]); close(stderr_pipe[1]);
        return fail;
    }

    if (pid == 0) {
        /* Child: redirect stdout and stderr into pipes */
        close(stdout_pipe[0]);
        close(stderr_pipe[0]);
        dup2(stdout_pipe[1], STDOUT_FILENO);
        dup2(stderr_pipe[1], STDERR_FILENO);
        close(stdout_pipe[1]);
        close(stderr_pipe[1]);
        execvp(cmd.data, argv);
        /* execvp failed */
        _exit(127);
    }

    /* Parent: close write ends, then read stdout and stderr concurrently
     * using select() to avoid deadlock when one pipe's buffer fills. */
    close(stdout_pipe[1]);
    close(stderr_pipe[1]);

    char buf[4096];
    size_t out_total = 0, err_total = 0;
    size_t out_cap = sizeof(buf), err_cap = sizeof(buf);
    char *out_buf = ez_arena_alloc(arena, out_cap);
    char *err_buf = ez_arena_alloc(arena, err_cap);
    int out_fd = stdout_pipe[0];
    int err_fd = stderr_pipe[0];
    bool out_done = false, err_done = false;
    bool truncated = false;

    while (!out_done || !err_done) {
        fd_set fds;
        FD_ZERO(&fds);
        if (!out_done) FD_SET(out_fd, &fds);
        if (!err_done) FD_SET(err_fd, &fds);
        int maxfd = (out_fd > err_fd ? out_fd : err_fd) + 1;
        if (select(maxfd, &fds, NULL, NULL, NULL) < 0) break;

        if (!out_done && FD_ISSET(out_fd, &fds)) {
            ssize_t n = read(out_fd, buf, sizeof(buf));
            if (n <= 0) {
                out_done = true;
            } else if (out_total + (size_t)n > EZ_EXEC_OUTPUT_MAX) {
                kill(pid, SIGKILL);
                out_done = true;
                err_done = true;
                truncated = true;
            } else {
                if (out_total + (size_t)n > out_cap) {
                    size_t new_cap = out_cap * 2 + (size_t)n;
                    char *grown = ez_arena_alloc(arena, new_cap);
                    memcpy(grown, out_buf, out_total);
                    out_buf = grown;
                    out_cap = new_cap;
                }
                memcpy(out_buf + out_total, buf, (size_t)n);
                out_total += (size_t)n;
            }
        }

        if (!err_done && FD_ISSET(err_fd, &fds)) {
            ssize_t n = read(err_fd, buf, sizeof(buf));
            if (n <= 0) {
                err_done = true;
            } else if (err_total + (size_t)n > EZ_EXEC_OUTPUT_MAX) {
                kill(pid, SIGKILL);
                out_done = true;
                err_done = true;
                truncated = true;
            } else {
                if (err_total + (size_t)n > err_cap) {
                    size_t new_cap = err_cap * 2 + (size_t)n;
                    char *grown = ez_arena_alloc(arena, new_cap);
                    memcpy(grown, err_buf, err_total);
                    err_buf = grown;
                    err_cap = new_cap;
                }
                memcpy(err_buf + err_total, buf, (size_t)n);
                err_total += (size_t)n;
            }
        }
    }

    close(out_fd);
    close(err_fd);

    int status = 0;
    waitpid(pid, &status, 0);

    int exit_code = 0;
    if (WIFEXITED(status)) {
        exit_code = WEXITSTATUS(status);
    } else if (WIFSIGNALED(status)) {
        exit_code = 128 + WTERMSIG(status);
    }

    /* exit_code 127 means execvp failed (command not found / bad path) */
    if (exit_code == 127 || truncated) return fail;

    EzString stdout_str = ez_string_new(arena, out_buf, (int32_t)out_total);
    EzString stderr_str = ez_string_new(arena, err_buf, (int32_t)err_total);
    // EzOsExecResult r = {(int64_t)exit_code, stdout_str, stderr_str, true};
    EzOsExecResult r = {(int64_t)exit_code, stderr_str, true};
    return r;
}

