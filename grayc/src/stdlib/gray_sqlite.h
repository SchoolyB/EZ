/*
 * gray_sqlite.h — Public interface for the sqlite stdlib module.
 * Declares database handle types, open/close, query execution, and
 * row retrieval functions backed by the SQLite3 amalgamation.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_SQLITE_H
#define GRAY_SQLITE_H

#include "../runtime/gray_runtime.h"
#include "../runtime/gray_array.h"
#include "../runtime/gray_map.h"
#include "gray_io.h" /* GrayResult_bool, GrayResult_array */

/*@man open
 *@module sqlite
 *@group Connection
 *@sig open(path string) -> (Database, Error)
 *@desc Opens or creates a SQLite database file at path. Returns a database handle used for all subsequent operations. Always use destructuring — single-variable assignment is a compile error. Relative paths resolve from the working directory where the binary is executed, not the source file location.
 *@example
 *   import @sqlite
 *   mut db, err = sqlite.open("myapp.db")
 *   if err != nil { println("failed: ${err}") }
 *@end
 */

/*@man close
 *@module sqlite
 *@group Connection
 *@sig close(db Database)
 *@desc Closes the database connection. No return value.
 *@example
 *   import @sqlite
 *   mut db, _ = sqlite.open("myapp.db")
 *   sqlite.close(db)
 *@end
 */

/*@man exec
 *@module sqlite
 *@group Queries
 *@sig exec(db Database, sql string) -> (bool, Error)
 *@desc Executes a SQL statement that does not return rows (CREATE, INSERT, UPDATE, DELETE, etc.). Returns true on success. Always use destructuring — single-variable assignment is a compile error. SQL strings are passed as-is; parameterized queries with placeholders are not supported.
 *@example
 *   import @sqlite
 *   mut db, _ = sqlite.open("myapp.db")
 *   mut ok, err = sqlite.exec(db, "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
 *   if err != nil { println("exec failed: ${err}") }
 *   mut ok, _ = sqlite.exec(db, "INSERT INTO users (name) VALUES ('Alice')")
 *@end
 */

/*@man query
 *@module sqlite
 *@group Queries
 *@sig query(db Database, sql string) -> ([map[string:string]], Error)
 *@desc Executes a SQL SELECT statement and returns the result rows. Each row is a map of column name to string value. Always use destructuring — single-variable assignment is a compile error. SQL strings are passed as-is; parameterized queries with placeholders are not supported.
 *@example
 *   import @sqlite
 *   mut db, _ = sqlite.open("myapp.db")
 *   mut rows, err = sqlite.query(db, "SELECT * FROM users")
 *   if err != nil { println("query failed: ${err}") }
 *   for_each row in rows {
 *       println(row)
 *   }
 *@end
 */

typedef struct {
    void *handle; /* sqlite3* */
} GraySqlite;

/* sqlite.open(path) — open or create database */
GraySqlite *gray_sqlite_open(GrayArena *arena, GrayString path);

/* sqlite.close(db) — close database */
void gray_sqlite_close(GraySqlite *db);

/* sqlite.exec(db, sql) — execute statement, return success */
bool gray_sqlite_exec(GraySqlite *db, GrayString sql);

/* sqlite.query(db, sql) — execute query, return array of maps */
GrayArray gray_sqlite_query(GrayArena *arena, GraySqlite *db, GrayString sql);

/* _result variants */
typedef struct { GraySqlite *v0; GrayError *v1; } GrayResult_sqlite;

GrayResult_sqlite gray_sqlite_open_result(GrayArena *arena, GrayString path);
GrayResult_bool gray_sqlite_exec_result(GrayArena *arena, GraySqlite *db, GrayString sql);
GrayResult_array gray_sqlite_query_result(GrayArena *arena, GraySqlite *db, GrayString sql);

#endif
