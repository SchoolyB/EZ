/*
 * ez_std.h - @std module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STD_H
#define EZ_STD_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "../runtime/ez_map.h"

/* println(value) - print followed by newline */
void ez_std_println_str(EzString s);
void ez_std_println_int(int64_t v);
void ez_std_println_float(double v);
void ez_std_println_bool(bool v);

/* print(value) - print without newline */
void ez_std_print_str(EzString s);
void ez_std_print_int(int64_t v);

/* eprintln(value) - print to stderr with newline */
void ez_std_eprintln_str(EzString s);
void ez_std_eprintln_int(int64_t v);

/* eprint(value) - print to stderr without newline */
void ez_std_eprint_str(EzString s);

/* input() - read line from stdin */
EzString ez_std_input(EzArena *arena);

/* assert(condition, message) */
void ez_std_assert(bool condition, EzString message, const char *file, int line);

/* panic(message) */
void ez_std_panic_msg(EzString message);

/* exit(code) */
void ez_std_exit(int64_t code);

/* copy(value) - handled by codegen (deep copy) */

/* sleep */
void ez_std_sleep_seconds(int64_t seconds);
void ez_std_sleep_milliseconds(int64_t ms);
void ez_std_sleep_nanoseconds(int64_t ns);

/* to_string */
EzString ez_std_to_string_int(EzArena *arena, int64_t v);
EzString ez_std_to_string_float(EzArena *arena, double v);
EzString ez_std_to_string_bool(EzArena *arena, bool v);

/* Format float for interpolation — always shows decimal point */
EzString ez_std_format_float(EzArena *arena, double v);

/* Composite type to string */
EzString ez_std_array_to_string(EzArena *arena, EzArray *arr, int elem_kind);
EzString ez_std_map_to_string(EzArena *arena, EzMap *m, int val_kind);

#endif
