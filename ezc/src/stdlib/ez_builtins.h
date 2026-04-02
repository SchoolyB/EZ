/*
 * ez_builtins.h - Built-in functions that require no import
 *
 * These are always available without importing any module.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_BUILTINS_H
#define EZ_BUILTINS_H

#include "../runtime/ez_runtime.h"

/* input() -> string */
EzString ez_builtin_input(EzArena *arena);

/* assert(condition, message) */
void ez_builtin_assert(bool condition, EzString message, const char *file, int line);

/* panic(message) */
void ez_builtin_panic_msg(EzString message);

/* exit(code) */
void ez_builtin_exit(int64_t code);

/* sleep */
void ez_builtin_sleep_s(int64_t seconds);
void ez_builtin_sleep_ms(int64_t ms);
void ez_builtin_sleep_ns(int64_t ns);

#endif
