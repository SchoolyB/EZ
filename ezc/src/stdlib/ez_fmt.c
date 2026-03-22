/*
 * ez_fmt.c - @fmt module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_fmt.h"
#include <inttypes.h>

void ez_fmt_eprintln_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
    fputc('\n', stderr);
}

void ez_fmt_eprintln_int(int64_t v) {
    fprintf(stderr, "%" PRId64 "\n", v);
}

void ez_fmt_eprintln_float(double v) {
    fprintf(stderr, "%g\n", v);
}

void ez_fmt_eprintln_bool(bool v) {
    fprintf(stderr, "%s\n", v ? "true" : "false");
}

void ez_fmt_eprint_str(EzString s) {
    fwrite(s.data, 1, (size_t)s.len, stderr);
}
