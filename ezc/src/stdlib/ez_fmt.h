/*
 * ez_fmt.h - @fmt module for EZC (formatted output)
 *
 * EZC-only module for C-style formatted I/O.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_FMT_H
#define EZ_FMT_H

#include "../runtime/ez_runtime.h"
#include <stdio.h>

/*
 * fmt.printf and fmt.sprintf are handled directly by the codegen —
 * the compiler translates EZ format strings to C printf format strings
 * at compile time, so no runtime function is needed for those.
 *
 * These helpers are for operations that need runtime support.
 */

/* fmt.eprintln(value) — print to stderr with newline */
void ez_fmt_eprintln_str(EzString s);
void ez_fmt_eprintln_int(int64_t v);
void ez_fmt_eprintln_uint(uint64_t v);
void ez_fmt_eprintln_float(double v);
void ez_fmt_eprintln_bool(bool v);

/* fmt.eprint(value) — print to stderr without newline */
void ez_fmt_eprint_str(EzString s);

/* fmt.sprint(arena, values...) — concatenate values into a string */
/* Handled by codegen via ez_string_format */

/* fmt.format(template, ...) — like sprintf but without explicit arena */
/* Handled by codegen via ez_string_format with ez_default_arena */

/* fmt.pad_left / pad_right / center — string padding */
EzString ez_fmt_pad_left(EzArena *arena, EzString s, int64_t width, char ch);
EzString ez_fmt_pad_right(EzArena *arena, EzString s, int64_t width, char ch);
EzString ez_fmt_center(EzArena *arena, EzString s, int64_t width, char ch);

/* Number formatting */
EzString ez_fmt_int_to_hex(EzArena *arena, int64_t n);
EzString ez_fmt_int_to_binary(EzArena *arena, int64_t n);
EzString ez_fmt_int_to_octal(EzArena *arena, int64_t n);
EzString ez_fmt_float_fixed(EzArena *arena, double f, int64_t decimals);
EzString ez_fmt_float_sci(EzArena *arena, double f);

#endif
