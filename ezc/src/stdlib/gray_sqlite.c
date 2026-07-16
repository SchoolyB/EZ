/*
 * gray_sqlite.c - @sqlite module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_sqlite.h"
#include "../vendor/sqlite3.h"
#include <string.h>
#include <stdio.h>

EzSqlite *gray_sqlite_open(EzArena *arena, EzString path) {
    EzSqlite *db = (EzSqlite *)gray_arena_alloc(arena, sizeof(EzSqlite));
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

void gray_sqlite_close(EzSqlite *db) {
    if (db && db->handle) {
        sqlite3_close((sqlite3 *)db->handle);
        db->handle = NULL;
    }
}

bool gray_sqlite_exec(EzSqlite *db, EzString sql) {
    if (!db || !db->handle) return false;
    char *err = NULL;
    int rc = sqlite3_exec((sqlite3 *)db->handle, sql.data, NULL, NULL, &err);
    if (err) sqlite3_free(err);
    return rc == SQLITE_OK;
}

EzArray gray_sqlite_query(EzArena *arena, EzSqlite *db, EzString sql) {
    EzArray rows = gray_array_new(arena, sizeof(EzMap), 8);
    if (!db || !db->handle) return rows;

    sqlite3_stmt *stmt = NULL;
    int rc = sqlite3_prepare_v2((sqlite3 *)db->handle, sql.data, sql.len, &stmt, NULL);
    if (rc != SQLITE_OK || !stmt) return rows;

    int col_count = sqlite3_column_count(stmt);

    while (sqlite3_step(stmt) == SQLITE_ROW) {
        EzMap row = gray_map_new(arena, sizeof(EzString), sizeof(EzString), col_count * 2);
        for (int i = 0; i < col_count; i++) {
            const char *col_name = sqlite3_column_name(stmt, i);
            EzString key = gray_string_new(arena, col_name, (int32_t)strlen(col_name));

            const char *val_text = (const char *)sqlite3_column_text(stmt, i);
            EzString val;
            if (val_text) {
                val = gray_string_new(arena, val_text, (int32_t)strlen(val_text));
            } else {
                val = gray_string_lit("");
            }
            gray_map_set(arena, &row, &key, &val);
        }
        gray_array_push(arena, &rows, &row);
    }

    sqlite3_finalize(stmt);
    return rows;
}

/* _result variants */

EzResult_sqlite gray_sqlite_open_result(EzArena *arena, EzString path) {
    EzResult_sqlite r;
    r.v0 = gray_sqlite_open(arena, path);
    if (!r.v0 || !r.v0->handle) {
        if (!r.v0) r.v0 = (EzSqlite *)gray_arena_alloc(arena, sizeof(EzSqlite));
        r.v0->handle = NULL;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot open database '%s'", path.data));
    } else {
        r.v1 = NULL;
    }
    return r;
}

EzResult_bool gray_sqlite_exec_result(EzArena *arena, EzSqlite *db, EzString sql) {
    EzResult_bool r;
    if (!db || !db->handle) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "database handle is nil"));
        return r;
    }
    char *err = NULL;
    int rc = sqlite3_exec((sqlite3 *)db->handle, sql.data, NULL, NULL, &err);
    if (rc != SQLITE_OK) {
        EzString msg = err ? gray_string_format(arena, "exec failed: %s", err)
                           : gray_string_format(arena, "exec failed (code %d)", rc);
        if (err) sqlite3_free(err);
        r.v0 = false;
        r.v1 = gray_error_new(arena, msg);
    } else {
        if (err) sqlite3_free(err);
        r.v0 = true;
        r.v1 = NULL;
    }
    return r;
}

EzResult_array gray_sqlite_query_result(EzArena *arena, EzSqlite *db, EzString sql) {
    EzResult_array r;
    if (!db || !db->handle) {
        r.v0 = gray_array_new(arena, sizeof(EzMap), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena, "database handle is nil"));
        return r;
    }
    sqlite3_stmt *stmt = NULL;
    int rc = sqlite3_prepare_v2((sqlite3 *)db->handle, sql.data, sql.len, &stmt, NULL);
    if (rc != SQLITE_OK || !stmt) {
        r.v0 = gray_array_new(arena, sizeof(EzMap), 0);
        const char *errmsg = sqlite3_errmsg((sqlite3 *)db->handle);
        r.v1 = gray_error_new(arena, gray_string_format(arena, "query failed: %s", errmsg ? errmsg : "unknown error"));
        return r;
    }
    sqlite3_finalize(stmt);
    r.v0 = gray_sqlite_query(arena, db, sql);
    r.v1 = NULL;
    return r;
}
