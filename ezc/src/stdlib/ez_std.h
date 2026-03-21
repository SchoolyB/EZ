/*
 * ez_std.h - @std module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STD_H
#define EZ_STD_H

#include "../runtime/ez_runtime.h"

/* println(value) - print a string followed by newline */
void ez_std_println_str(EzString s);
void ez_std_println_int(int64_t v);
void ez_std_println_float(double v);
void ez_std_println_bool(bool v);

/* print(value) - print without newline */
void ez_std_print_str(EzString s);
void ez_std_print_int(int64_t v);

#endif
