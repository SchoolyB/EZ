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

/*@man args
 *@module os
 *@group System
 *@sig args() -> [string]
 *@desc Returns the command-line arguments passed to the program as an array of strings. The first element is the program name.
 *@example
 *   import @os
 *   mut argv = os.args()
 *   println(argv[0])
 *@end
 */
/* os.args() — return command-line arguments as [string] */
EzArray ez_os_args(EzArena *arena);

/*@man get_env
 *@module os
 *@group Environment
 *@sig get_env(name string) -> string
 *@desc Returns the value of the environment variable name. Returns an empty string if the variable is not set.
 *@example
 *   import @os
 *   mut home = os.get_env("HOME")
 *   println(home)
 *@end
 */
/* os.get_env(name) — get environment variable */
EzString ez_os_get_env(EzArena *arena, EzString name);

/*@man set_env
 *@module os
 *@group Environment
 *@sig set_env(name string, value string)
 *@desc Sets the environment variable name to value for the current process.
 *@example
 *   import @os
 *   os.set_env("MY_VAR", "hello")
 *   println(os.get_env("MY_VAR"))
 *@end
 */
/* os.set_env(name, value) */
void ez_os_set_env(EzString name, EzString value);

/*@man current_dir
 *@module os
 *@group System
 *@sig current_dir() -> string
 *@desc Returns the current working directory of the process.
 *@example
 *   import @os
 *   mut cwd = os.current_dir()
 *   println(cwd)
 *@end
 */
/* os.current_dir() — current working directory */
EzString ez_os_cwd(EzArena *arena);

/*@man hostname
 *@module os
 *@group System
 *@sig hostname() -> string
 *@desc Returns the hostname of the machine.
 *@example
 *   import @os
 *   println(os.hostname())
 *@end
 */
/* os.hostname() */
EzString ez_os_hostname(EzArena *arena);

/*@man current_os
 *@module os
 *@group System
 *@sig current_os() -> int
 *@desc Returns an integer identifying the current operating system. Compare against the module constants: MAC_OS (0), LINUX (1), WINDOWS (2), OTHER (3).
 *@example
 *   import @os
 *   if os.current_os() == os.LINUX {
 *       println("running on Linux")
 *   }
 *@end
 */
/* os.current_os() — returns MAC_OS=0, LINUX=1, WINDOWS=2, OTHER=3 */
int64_t ez_os_current_os(void);

/*@man arch
 *@module os
 *@group System
 *@sig arch() -> string
 *@desc Returns the CPU architecture of the current machine, such as "arm64" or "x86_64".
 *@example
 *   import @os
 *   println(os.arch())
 *@end
 */
/* os.arch() — "arm64", "x86_64", etc. */
EzString ez_os_arch(void);

/*@man pid
 *@module os
 *@group System
 *@sig pid() -> int
 *@desc Returns the process ID of the current process.
 *@example
 *   import @os
 *   println(os.pid())
 *@end
 */
/* os.pid() */
int64_t ez_os_pid(void);

/* Store argc/argv from main for os.args() */
void ez_os_init(int argc, char **argv);

#endif
