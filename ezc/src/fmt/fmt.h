/*
 * fmt.h - EZ source formatter
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_FMT_H
#define EZC_FMT_H

#include <stdio.h>

/*
 * ez_fmt_source:
 *   Format the EZ source in `src` (NUL-terminated) and write the result to
 *   `out`. Returns 0 on success, non-zero if the source could not be lexed.
 *
 *   Strategy: lex the source to build a per-line indentation depth table,
 *   then re-emit each original source line with corrected leading whitespace.
 *   All content (comments, string literals, operators) is preserved verbatim —
 *   only the leading indentation of each line is touched.
 */
int ez_fmt_source(const char *src, const char *filename, FILE *out);

#endif
