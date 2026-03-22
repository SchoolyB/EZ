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
void ez_fmt_eprintln_float(double v);
void ez_fmt_eprintln_bool(bool v);

/* fmt.eprint(value) — print to stderr without newline */
void ez_fmt_eprint_str(EzString s);

/* fmt.sprint(arena, values...) — concatenate values into a string */
/* Handled by codegen via ez_string_format */

#endif
