/*
 * ez_io.h - @io module for EZC (file I/O)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_IO_H
#define EZ_IO_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

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
bool ez_io_copy_file(EzString src, EzString dst);
bool ez_io_move_file(EzString src, EzString dst);

/* Directory operations */
EzArray ez_io_list_dir(EzArena *arena, EzString path);
bool ez_io_make_dir(EzString path);
bool ez_io_make_dir_all(EzString path);
bool ez_io_remove_dir(EzString path);
bool ez_io_remove_dir_all(EzString path);
EzArray ez_io_walk(EzArena *arena, EzString path);
EzArray ez_io_glob(EzArena *arena, EzString pattern);

/* Path manipulation */
EzString ez_io_path_join(EzArena *arena, EzString a, EzString b);
EzString ez_io_dirname(EzArena *arena, EzString path);
EzString ez_io_basename(EzArena *arena, EzString path);
EzString ez_io_extension(EzArena *arena, EzString path);
bool ez_io_is_absolute(EzString path);
EzString ez_io_normalize(EzArena *arena, EzString path);

/* Tuple-returning versions for (value, Error) pattern */
typedef struct { EzString v0; EzError *v1; } EzResult_string;
typedef struct { bool v0; EzError *v1; } EzResult_bool;
typedef struct { EzArray v0; EzError *v1; } EzResult_array;

EzResult_string ez_io_read_file_result(EzArena *arena, EzString path);
EzResult_bool ez_io_write_file_result(EzArena *arena, EzString path, EzString content);
EzResult_bool ez_io_delete_file_result(EzArena *arena, EzString path);
EzResult_bool ez_io_append_file_result(EzArena *arena, EzString path, EzString content);
EzResult_bool ez_io_rename_file_result(EzArena *arena, EzString old_path, EzString new_path);
EzResult_bool ez_io_copy_file_result(EzArena *arena, EzString src, EzString dst);
EzResult_bool ez_io_move_file_result(EzArena *arena, EzString src, EzString dst);
EzResult_array ez_io_list_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_make_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_make_dir_all_result(EzArena *arena, EzString path);
EzResult_bool ez_io_remove_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_remove_dir_all_result(EzArena *arena, EzString path);
EzResult_array ez_io_walk_result(EzArena *arena, EzString path);

#endif
