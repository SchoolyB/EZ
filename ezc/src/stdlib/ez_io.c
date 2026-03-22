/*
 * ez_io.c - @io module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_io.h"
#include <stdio.h>
#include <sys/stat.h>
#include <unistd.h>

EzString ez_io_read_file(EzArena *arena, EzString path) {
    FILE *f = fopen(path.data, "rb");
    if (!f) return ez_string_lit("");
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);
    char *buf = ez_arena_alloc(arena, (size_t)size + 1);
    size_t read = fread(buf, 1, (size_t)size, f);
    buf[read] = '\0';
    fclose(f);
    EzString s = { buf, (int32_t)read };
    return s;
}

bool ez_io_file_exists(EzString path) {
    return access(path.data, F_OK) == 0;
}

bool ez_io_is_file(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return false;
    return S_ISREG(st.st_mode);
}

bool ez_io_is_directory(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return false;
    return S_ISDIR(st.st_mode);
}

int64_t ez_io_file_size(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return -1;
    return (int64_t)st.st_size;
}

bool ez_io_write_file(EzString path, EzString content) {
    FILE *f = fopen(path.data, "wb");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    return written == (size_t)content.len;
}

bool ez_io_append_file(EzString path, EzString content) {
    FILE *f = fopen(path.data, "ab");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    return written == (size_t)content.len;
}

bool ez_io_delete_file(EzString path) {
    return unlink(path.data) == 0;
}

bool ez_io_rename(EzString old_path, EzString new_path) {
    return rename(old_path.data, new_path.data) == 0;
}
