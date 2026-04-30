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
