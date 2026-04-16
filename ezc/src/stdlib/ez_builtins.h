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
#include "../runtime/ez_array.h"
#include "../runtime/ez_map.h"

/* println(value) - print with newline */
void ez_builtin_println_str(EzString s);
void ez_builtin_println_int(int64_t v);
void ez_builtin_println_float(double v);
void ez_builtin_println_bool(bool v);
void ez_builtin_println_char(int32_t c);
void ez_builtin_println_addr(uintptr_t v);

/* print(value) - print without newline */
void ez_builtin_print_str(EzString s);
void ez_builtin_print_int(int64_t v);
void ez_builtin_print_float(double v);
void ez_builtin_print_bool(bool v);
void ez_builtin_print_char(int32_t c);
void ez_builtin_print_addr(uintptr_t v);

/* eprintln(value) - print to stderr with newline */
void ez_builtin_eprintln_str(EzString s);
void ez_builtin_eprintln_int(int64_t v);
void ez_builtin_eprintln_char(int32_t c);
void ez_builtin_eprintln_addr(uintptr_t v);

/* eprint(value) - print to stderr without newline */
void ez_builtin_eprint_str(EzString s);
void ez_builtin_eprint_char(int32_t c);
void ez_builtin_eprint_addr(uintptr_t v);

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

/* to_string */
EzString ez_builtin_to_string_int(EzArena *arena, int64_t v);
EzString ez_builtin_to_string_float(EzArena *arena, double v);
EzString ez_builtin_to_string_bool(EzArena *arena, bool v);

/* from_string */
int64_t ez_builtin_string_to_int(EzString s);
double ez_builtin_string_to_float(EzString s);

/* format float for interpolation */
EzString ez_builtin_format_float(EzArena *arena, double v);

/* composite to_string */
EzString ez_builtin_array_to_string(EzArena *arena, EzArray *arr, int elem_kind);
EzString ez_builtin_map_to_string(EzArena *arena, EzMap *m, int val_kind);

/* to_char / char_count — Unicode codepoint access */
int32_t ez_builtin_to_char(EzString s, int64_t index, const char *file, int line);
int64_t ez_builtin_char_count(EzString s);

#endif
