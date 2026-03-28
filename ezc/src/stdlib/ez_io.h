/*
 * ez_io.h - @io module for EZC (file I/O)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_IO_H
#define EZ_IO_H

#include "../runtime/ez_runtime.h"

/* File reading */
EzString ez_io_read_file(EzArena *arena, EzString path);
bool ez_io_file_exists(EzString path);
bool ez_io_is_file(EzString path);
bool ez_io_is_directory(EzString path);
int64_t ez_io_file_size(EzString path);

/* File writing */
bool ez_io_write_file(EzString path, EzString content);
bool ez_io_append_file(EzString path, EzString content);

/* File operations */
bool ez_io_delete_file(EzString path);
bool ez_io_rename_file(EzString old_path, EzString new_path);

/* Tuple-returning versions for (value, Error) pattern */
typedef struct { EzString v0; EzError *v1; } EzResult_string;
typedef struct { bool v0; EzError *v1; } EzResult_bool;

EzResult_string ez_io_read_file_result(EzArena *arena, EzString path);
EzResult_bool ez_io_write_file_result(EzArena *arena, EzString path, EzString content);
EzResult_bool ez_io_delete_file_result(EzArena *arena, EzString path);

#endif
