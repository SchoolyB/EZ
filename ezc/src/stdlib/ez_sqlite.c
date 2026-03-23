/*
 * ez_sqlite.c - @sqlite module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_sqlite.h"
#include "../vendor/sqlite3.h"
#include <string.h>
#include <stdio.h>

EzSqlite *ez_sqlite_open(EzArena *arena, EzString path) {
    EzSqlite *db = (EzSqlite *)ez_arena_alloc(arena, sizeof(EzSqlite));
    sqlite3 *handle = NULL;
    int rc = sqlite3_open(path.data, &handle);
    if (rc != SQLITE_OK) {
        if (handle) sqlite3_close(handle);
        db->handle = NULL;
        return db;
    }
    db->handle = handle;
    return db;
}

void ez_sqlite_close(EzSqlite *db) {
    if (db && db->handle) {
        sqlite3_close((sqlite3 *)db->handle);
        db->handle = NULL;
    }
}

bool ez_sqlite_exec(EzSqlite *db, EzString sql) {
    if (!db || !db->handle) return false;
    char *err = NULL;
    int rc = sqlite3_exec((sqlite3 *)db->handle, sql.data, NULL, NULL, &err);
    if (err) sqlite3_free(err);
    return rc == SQLITE_OK;
}

EzArray ez_sqlite_query(EzArena *arena, EzSqlite *db, EzString sql) {
    EzArray rows = ez_array_new(arena, sizeof(EzMap), 8);
    if (!db || !db->handle) return rows;

    sqlite3_stmt *stmt = NULL;
    int rc = sqlite3_prepare_v2((sqlite3 *)db->handle, sql.data, sql.len, &stmt, NULL);
    if (rc != SQLITE_OK || !stmt) return rows;

    int col_count = sqlite3_column_count(stmt);

    while (sqlite3_step(stmt) == SQLITE_ROW) {
        EzMap row = ez_map_new(arena, sizeof(EzString), sizeof(EzString), col_count * 2);
        for (int i = 0; i < col_count; i++) {
            const char *col_name = sqlite3_column_name(stmt, i);
            EzString key = ez_string_new(arena, col_name, (int32_t)strlen(col_name));

            const char *val_text = (const char *)sqlite3_column_text(stmt, i);
            EzString val;
            if (val_text) {
                val = ez_string_new(arena, val_text, (int32_t)strlen(val_text));
            } else {
                val = ez_string_lit("");
            }
            ez_map_set(arena, &row, &key, &val);
        }
        ez_array_push(arena, &rows, &row);
    }

    sqlite3_finalize(stmt);
    return rows;
}
