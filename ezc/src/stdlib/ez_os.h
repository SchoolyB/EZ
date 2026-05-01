/*
 * ez_os.h - @os module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_OS_H
#define EZ_OS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* os.args() — return command-line arguments as [string] */
EzArray ez_os_args(EzArena *arena);

/* os.get_env(name) — get environment variable */
EzString ez_os_get_env(EzArena *arena, EzString name);

/* os.set_env(name, value) */
void ez_os_set_env(EzString name, EzString value);

/* os.cwd() — current working directory */
EzString ez_os_cwd(EzArena *arena);

/* os.hostname() */
EzString ez_os_hostname(EzArena *arena);

/* os.current_os() — returns MAC_OS=0, LINUX=1, WINDOWS=2, OTHER=3 */
int64_t ez_os_current_os(void);

/* os.arch() — "arm64", "x86_64", etc. */
EzString ez_os_arch(void);

/* os.pid() */
int64_t ez_os_pid(void);

/* Store argc/argv from main for os.args() */
void ez_os_init(int argc, char **argv);

#endif
