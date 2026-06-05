/*
 * ez_sqlite.h - @sqlite module for EZ
 *
 * Backed by the SQLite3 amalgamation.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_SQLITE_H
#define EZ_SQLITE_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "../runtime/ez_map.h"
#include "ez_io.h" /* EzResult_bool, EzResult_array */

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
} EzSqlite;

/* sqlite.open(path) — open or create database */
EzSqlite *ez_sqlite_open(EzArena *arena, EzString path);

/* sqlite.close(db) — close database */
void ez_sqlite_close(EzSqlite *db);

/* sqlite.exec(db, sql) — execute statement, return success */
bool ez_sqlite_exec(EzSqlite *db, EzString sql);

/* sqlite.query(db, sql) — execute query, return array of maps */
EzArray ez_sqlite_query(EzArena *arena, EzSqlite *db, EzString sql);

/* _result variants */
typedef struct { EzSqlite *v0; EzError *v1; } EzResult_sqlite;

EzResult_sqlite ez_sqlite_open_result(EzArena *arena, EzString path);
EzResult_bool ez_sqlite_exec_result(EzArena *arena, EzSqlite *db, EzString sql);
EzResult_array ez_sqlite_query_result(EzArena *arena, EzSqlite *db, EzString sql);

#endif
