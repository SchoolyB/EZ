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
bool ez_io_rename(EzString old_path, EzString new_path);

/* Aliases used by interpreter: remove = delete_file, exists = file_exists */
#define ez_io_remove ez_io_delete_file
#define ez_io_exists ez_io_file_exists

#endif
