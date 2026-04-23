/*
 * ez_csv.h - @csv module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_CSV_H
#define EZ_CSV_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "ez_io.h" /* EzResult_bool, EzResult_array */

/* Parse CSV string into array of arrays of strings */
EzArray ez_csv_parse(EzArena *arena, EzString csv_string);

/* Convert array of arrays of strings to CSV string */
EzString ez_csv_stringify(EzArena *arena, EzArray *data);

/* Read CSV file */
EzArray ez_csv_read(EzArena *arena, EzString path);

/* Write CSV to file */
bool ez_csv_write(EzArena *arena, EzString path, EzArray *data);

/* _result variants */
EzResult_array ez_csv_read_result(EzArena *arena, EzString path);
EzResult_bool ez_csv_write_result(EzArena *arena, EzString path, EzArray *data);

#endif
