/*
 * ez_io.h - @io module for EZ (file I/O)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_IO_H
#define EZ_IO_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* File reading */

/*@man read_file
 *@module io
 *@group File Reading
 *@sig read_file(path string) -> (string, Error)
 *@desc Reads the entire file at path and returns its contents as a string. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut content string = io.read_file("data.txt")
 *   mut content, err = io.read_file("data.txt")
 *@end
 */
EzString ez_io_read_file(EzArena *arena, EzString path);

/*@man read_bytes
 *@module io
 *@group File Reading
 *@sig read_bytes(path string) -> ([byte], Error)
 *@desc Reads the entire file at path and returns its contents as a byte array. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut data [byte] = io.read_bytes("image.png")
 *@end
 */
EzArray  ez_io_read_bytes(EzArena *arena, EzString path);

/*@man read_lines
 *@module io
 *@group File Reading
 *@sig read_lines(path string) -> ([string], Error)
 *@desc Reads the file at path and returns its contents split into lines. CR/LF and LF line endings are stripped. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut lines [string] = io.read_lines("data.txt")
 *   for_each line in lines {
 *       println(line)
 *   }
 *@end
 */
EzArray  ez_io_read_lines(EzArena *arena, EzString path);

/*@man file_exists
 *@module io
 *@group File Operations
 *@sig file_exists(path string) -> bool
 *@desc Returns true if a file or directory exists at path.
 *@example
 *   import @io
 *   if io.file_exists("config.txt") {
 *       println("found")
 *   }
 *@end
 */
bool ez_io_file_exists(EzString path);

/*@man is_file
 *@module io
 *@group File Operations
 *@sig is_file(path string) -> bool
 *@desc Returns true if path exists and is a regular file (not a directory or special file).
 *@example
 *   import @io
 *   println(io.is_file("data.txt"))
 *@end
 */
bool ez_io_is_file(EzString path);

/*@man is_directory
 *@module io
 *@group File Operations
 *@sig is_directory(path string) -> bool
 *@desc Returns true if path exists and is a directory.
 *@example
 *   import @io
 *   println(io.is_directory("/tmp"))
 *@end
 */
bool ez_io_is_directory(EzString path);

/*@man file_size
 *@module io
 *@group File Operations
 *@sig file_size(path string) -> (int, Error)
 *@desc Returns the size of the file at path in bytes. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   println(io.file_size("data.txt"))
 *@end
 */
int64_t ez_io_file_size(EzString path);

/* File writing */

/*@man write_file
 *@module io
 *@group File Writing
 *@sig write_file(path string, content string) -> (bool, Error)
 *@desc Writes content to the file at path, creating it if it does not exist or overwriting it if it does. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.write_file("out.txt", "hello\n")
 *@end
 */
bool ez_io_write_file(EzString path, EzString content);

/*@man append_file
 *@module io
 *@group File Writing
 *@sig append_file(path string, content string) -> (bool, Error)
 *@desc Appends content to the end of the file at path. Creates the file if it does not exist. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.append_file("log.txt", "new entry\n")
 *@end
 */
bool ez_io_append_file(EzString path, EzString content);

/* File operations */

/*@man delete_file
 *@module io
 *@group File Operations
 *@sig delete_file(path string) -> (bool, Error)
 *@desc Deletes the file at path. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.delete_file("tmp.txt")
 *@end
 */
bool ez_io_delete_file(EzString path);

/*@man rename_file
 *@module io
 *@group File Operations
 *@sig rename_file(old_path string, new_path string) -> (bool, Error)
 *@desc Renames (or moves) the file at old_path to new_path. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.rename_file("old.txt", "new.txt")
 *@end
 */
bool ez_io_rename_file(EzString old_path, EzString new_path);

/*@man copy_file
 *@module io
 *@group File Operations
 *@sig copy_file(src string, dst string) -> (bool, Error)
 *@desc Copies the file at src to dst. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.copy_file("src.txt", "dst.txt")
 *@end
 */
bool ez_io_copy_file(EzString src, EzString dst);

/*@man move_file
 *@module io
 *@group File Operations
 *@sig move_file(src string, dst string) -> (bool, Error)
 *@desc Moves the file at src to dst. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.move_file("tmp/file.txt", "final/file.txt")
 *@end
 */
bool ez_io_move_file(EzString src, EzString dst);

/* Directory operations */

/*@man list_dir
 *@module io
 *@group Directory Operations
 *@sig list_dir(path string) -> ([string], Error)
 *@desc Returns the names of all entries in the directory at path. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut entries [string] = io.list_dir(".")
 *   for_each e in entries {
 *       println(e)
 *   }
 *@end
 */
EzArray ez_io_list_dir(EzArena *arena, EzString path);

/*@man make_dir
 *@module io
 *@group Directory Operations
 *@sig make_dir(path string) -> (bool, Error)
 *@desc Creates the directory at path. Fails if the parent directory does not exist. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.make_dir("output")
 *@end
 */
bool ez_io_make_dir(EzString path);

/*@man make_dir_all
 *@module io
 *@group Directory Operations
 *@sig make_dir_all(path string) -> (bool, Error)
 *@desc Creates the directory at path, including all missing parent directories. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.make_dir_all("a/b/c")
 *@end
 */
bool ez_io_make_dir_all(EzString path);

/*@man remove_dir
 *@module io
 *@group Directory Operations
 *@sig remove_dir(path string) -> (bool, Error)
 *@desc Removes the empty directory at path. Fails if the directory is not empty. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.remove_dir("empty_dir")
 *@end
 */
bool ez_io_remove_dir(EzString path);

/*@man remove_dir_all
 *@module io
 *@group Directory Operations
 *@sig remove_dir_all(path string) -> (bool, Error)
 *@desc Removes the directory at path and all of its contents recursively. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   io.remove_dir_all("build")
 *@end
 */
bool ez_io_remove_dir_all(EzString path);

/*@man walk
 *@module io
 *@group Directory Operations
 *@sig walk(path string) -> ([string], Error)
 *@desc Recursively lists all files under path. Returns full paths relative to path. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut files [string] = io.walk("src")
 *   for_each f in files {
 *       println(f)
 *   }
 *@end
 */
EzArray ez_io_walk(EzArena *arena, EzString path);

/*@man glob
 *@module io
 *@group Directory Operations
 *@sig glob(pattern string) -> ([string], Error)
 *@desc Returns all file paths matching the glob pattern. Returns an empty array if there are no matches. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @io
 *   mut files [string] = io.glob("src/\*.ez")
 *@end
 */
EzArray ez_io_glob(EzArena *arena, EzString pattern);

/* Path manipulation */

/*@man path_join
 *@module io
 *@group Path
 *@sig path_join(parts [string]) -> string
 *@desc Joins path segments into a single path. An absolute segment replaces everything accumulated so far.
 *@example
 *   import @io
 *   println(io.path_join(["/home", "user", "docs"]))
 *@end
 */
EzString ez_io_path_join(EzArena *arena, EzArray parts);

/*@man dirname
 *@module io
 *@group Path
 *@sig dirname(path string) -> string
 *@desc Returns the parent directory component of path.
 *@example
 *   import @io
 *   println(io.dirname("/home/user/file.txt"))
 *@end
 */
EzString ez_io_dirname(EzArena *arena, EzString path);

/*@man basename
 *@module io
 *@group Path
 *@sig basename(path string) -> string
 *@desc Returns the filename component of path, without any leading directory.
 *@example
 *   import @io
 *   println(io.basename("/home/user/file.txt"))
 *@end
 */
EzString ez_io_basename(EzArena *arena, EzString path);

/*@man extension
 *@module io
 *@group Path
 *@sig extension(path string) -> string
 *@desc Returns the file extension of path including the leading dot (e.g. ".txt"). Returns an empty string if there is no extension.
 *@example
 *   import @io
 *   println(io.extension("report.pdf"))
 *@end
 */
EzString ez_io_extension(EzArena *arena, EzString path);

/*@man is_absolute
 *@module io
 *@group Path
 *@sig is_absolute(path string) -> bool
 *@desc Returns true if path is an absolute path.
 *@example
 *   import @io
 *   println(io.is_absolute("/etc/hosts"))
 *   println(io.is_absolute("relative/path"))
 *@end
 */
bool ez_io_is_absolute(EzString path);

/*@man normalize
 *@module io
 *@group Path
 *@sig normalize(path string) -> string
 *@desc Cleans and normalizes path by resolving ".", "..", and redundant separators.
 *@example
 *   import @io
 *   println(io.normalize("a/b/../c"))
 *@end
 */
EzString ez_io_normalize(EzArena *arena, EzString path);

/* Tuple-returning versions for (value, Error) pattern */
typedef struct { EzString v0; EzError *v1; } EzResult_string;
#ifndef EZRESULT_BOOL_DEFINED
#define EZRESULT_BOOL_DEFINED
typedef struct { bool v0; EzError *v1; } EzResult_bool;
#endif
typedef struct { EzArray v0; EzError *v1; } EzResult_array;

EzResult_string ez_io_read_file_result(EzArena *arena, EzString path);
EzResult_array  ez_io_read_bytes_result(EzArena *arena, EzString path);
EzResult_array  ez_io_read_lines_result(EzArena *arena, EzString path);
EzResult_int    ez_io_file_size_result(EzArena *arena, EzString path);
EzResult_bool ez_io_write_file_result(EzArena *arena, EzString path, EzString content);
EzResult_bool ez_io_delete_file_result(EzArena *arena, EzString path);
EzResult_bool ez_io_append_file_result(EzArena *arena, EzString path, EzString content);
EzResult_bool ez_io_rename_file_result(EzArena *arena, EzString old_path, EzString new_path);
EzResult_bool ez_io_copy_file_result(EzArena *arena, EzString src, EzString dst);
EzResult_bool ez_io_move_file_result(EzArena *arena, EzString src, EzString dst);
EzResult_array ez_io_glob_result(EzArena *arena, EzString pattern);
EzResult_array ez_io_list_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_make_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_make_dir_all_result(EzArena *arena, EzString path);
EzResult_bool ez_io_remove_dir_result(EzArena *arena, EzString path);
EzResult_bool ez_io_remove_dir_all_result(EzArena *arena, EzString path);
EzResult_array ez_io_walk_result(EzArena *arena, EzString path);

/*@man O_RDONLY
 *@module io
 *@group Constants
 *@kind const
 *@sig 0
 *@desc Open for reading only.
 *@end
 */

/*@man O_WRONLY
 *@module io
 *@group Constants
 *@kind const
 *@sig 1
 *@desc Open for writing only.
 *@end
 */

/*@man O_RDWR
 *@module io
 *@group Constants
 *@kind const
 *@sig 2
 *@desc Open for reading and writing.
 *@end
 */

#endif
