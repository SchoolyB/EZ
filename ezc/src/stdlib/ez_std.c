/*
 * ez_std.c - @std module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_std.h"
#include <stdio.h>
#include <inttypes.h>

void ez_std_println_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
    putchar('\n');
}

void ez_std_println_int(int64_t v) {
    printf("%" PRId64 "\n", v);
}

void ez_std_println_float(double v) {
    printf("%g\n", v);
}

void ez_std_println_bool(bool v) {
    printf("%s\n", v ? "true" : "false");
}

void ez_std_print_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stdout);
}

void ez_std_print_int(int64_t v) {
    printf("%" PRId64, v);
}
