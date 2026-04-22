/*
 * ez_io.c - @io module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_io.h"
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>
#include <errno.h>
#include <glob.h>

/* ---- Path manipulation (pure, no I/O) ---- */

EzString ez_io_path_join(EzArena *arena, EzString a, EzString b) {
    if (a.len == 0) return b;
    if (b.len == 0) return a;
    /* Absolute second argument replaces the first (Python/Rust convention) */
    if (b.data[0] == '/' || b.data[0] == '\\') return b;
    bool a_has_sep = (a.data[a.len - 1] == '/' || a.data[a.len - 1] == '\\');
    bool b_has_sep = (b.data[0] == '/' || b.data[0] == '\\');
    if (a_has_sep && b_has_sep) {
        return ez_string_format(arena, "%.*s%.*s", a.len, a.data, b.len - 1, b.data + 1);
    } else if (!a_has_sep && !b_has_sep) {
        return ez_string_format(arena, "%.*s/%.*s", a.len, a.data, b.len, b.data);
    } else {
        return ez_string_format(arena, "%.*s%.*s", a.len, a.data, b.len, b.data);
    }
}

EzString ez_io_dirname(EzArena *arena, EzString path) {
    if (path.len == 0) return ez_string_lit(".");
    /* Strip trailing slashes (keep at least 1 char so "/" stays "/") */
    int eff = path.len;
    while (eff > 1 && (path.data[eff - 1] == '/' || path.data[eff - 1] == '\\')) eff--;
    /* Find last separator in the stripped range */
    int last_sep = -1;
    for (int i = eff - 1; i >= 0; i--) {
        if (path.data[i] == '/' || path.data[i] == '\\') {
            last_sep = i;
            break;
        }
    }
    if (last_sep < 0) return ez_string_lit(".");
    /* Collapse leading separator: dirname("/foo") -> "/" */
    if (last_sep == 0) return ez_string_lit("/");
    char *buf = ez_arena_alloc(arena, (size_t)last_sep + 1);
    memcpy(buf, path.data, (size_t)last_sep);
    buf[last_sep] = '\0';
    return (EzString){ buf, (int32_t)last_sep };
}

EzString ez_io_basename(EzArena *arena, EzString path) {
    (void)arena;
    if (path.len == 0) return ez_string_lit(".");
    int end = path.len;
    while (end > 0 && (path.data[end - 1] == '/' || path.data[end - 1] == '\\')) end--;
    if (end == 0) return ez_string_lit("/");
    int start = end;
    while (start > 0 && path.data[start - 1] != '/' && path.data[start - 1] != '\\') start--;
    int32_t len = (int32_t)(end - start);
    char *buf = ez_arena_alloc(arena, (size_t)len + 1);
    memcpy(buf, path.data + start, (size_t)len);
    buf[len] = '\0';
    return (EzString){ buf, len };
}

EzString ez_io_extension(EzArena *arena, EzString path) {
    (void)arena;
    int last_sep = -1;
    for (int i = path.len - 1; i >= 0; i--) {
        if (path.data[i] == '/' || path.data[i] == '\\') { last_sep = i; break; }
    }
    int search_start = last_sep + 1;
    int dot_pos = -1;
    for (int i = path.len - 1; i >= search_start; i--) {
        if (path.data[i] == '.') { dot_pos = i; break; }
    }
    if (dot_pos < 0 || dot_pos == path.len - 1) return ez_string_lit("");
    /* Dotfiles: leading dot with no other dot is part of the name, not an extension */
    if (dot_pos == search_start) return ez_string_lit("");
    int32_t len = (int32_t)(path.len - dot_pos);
    char *buf = ez_arena_alloc(arena, (size_t)len + 1);
    memcpy(buf, path.data + dot_pos, (size_t)len);
    buf[len] = '\0';
    return (EzString){ buf, len };
}

bool ez_io_is_absolute(EzString path) {
    return path.len > 0 && path.data[0] == '/';
}

EzString ez_io_normalize(EzArena *arena, EzString path) {
    if (path.len == 0) return ez_string_lit(".");
    char *buf = ez_arena_alloc(arena, (size_t)path.len + 1);
    /* Copy input, converting backslashes to forward slashes */
    for (int i = 0; i < path.len; i++) {
        buf[i] = (path.data[i] == '\\') ? '/' : path.data[i];
    }
    buf[path.len] = '\0';
    bool absolute = (buf[0] == '/');

    /* Split into segments */
    char *segments[256];
    int seg_count = 0;
    char *tok = buf;
    while (*tok) {
        while (*tok == '/') tok++;
        if (*tok == '\0') break;
        char *seg_start = tok;
        while (*tok && *tok != '/') tok++;
        if (*tok) { *tok = '\0'; tok++; }
        if (strcmp(seg_start, ".") == 0) continue;
        if (strcmp(seg_start, "..") == 0) {
            if (seg_count > 0 && strcmp(segments[seg_count - 1], "..") != 0) {
                seg_count--;
            } else if (!absolute) {
                segments[seg_count++] = seg_start;
            }
        } else {
            if (seg_count < 256) segments[seg_count++] = seg_start;
        }
    }

    /* Rebuild */
    char *out = ez_arena_alloc(arena, (size_t)path.len + 2);
    int pos = 0;
    if (absolute) out[pos++] = '/';
    for (int i = 0; i < seg_count; i++) {
        if (i > 0) out[pos++] = '/';
        int slen = (int)strlen(segments[i]);
        memcpy(out + pos, segments[i], (size_t)slen);
        pos += slen;
    }
    if (pos == 0) {
        out[0] = '.';
        pos = 1;
    }
    out[pos] = '\0';
    return (EzString){ out, (int32_t)pos };
}

/* ---- Existing file operations ---- */

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
    struct stat st;
    if (stat(path.data, &st) == 0 && S_ISDIR(st.st_mode)) {
        fflush(stdout);
        fprintf(stderr, "panic: io.delete_file() cannot delete a directory — use io.delete_dir() for directories\n");
        exit(1);
    }
    return unlink(path.data) == 0;
}

bool ez_io_rename_file(EzString old_path, EzString new_path) {
    return rename(old_path.data, new_path.data) == 0;
}

/* ---- New file operations ---- */

bool ez_io_copy_file(EzString src, EzString dst) {
    FILE *in = fopen(src.data, "rb");
    if (!in) return false;
    FILE *out = fopen(dst.data, "wb");
    if (!out) { fclose(in); return false; }
    char buf[8192];
    size_t n;
    bool ok = true;
    while ((n = fread(buf, 1, sizeof(buf), in)) > 0) {
        if (fwrite(buf, 1, n, out) != n) { ok = false; break; }
    }
    fclose(in);
    fclose(out);
    return ok;
}

bool ez_io_move_file(EzString src, EzString dst) {
    if (rename(src.data, dst.data) == 0) return true;
    if (!ez_io_copy_file(src, dst)) return false;
    unlink(src.data);
    return true;
}

/* ---- Directory operations ---- */

EzArray ez_io_list_dir(EzArena *arena, EzString path) {
    EzArray arr = ez_array_new(arena, (int32_t)sizeof(EzString), 16);
    DIR *d = opendir(path.data);
    if (!d) return arr;
    struct dirent *ent;
    while ((ent = readdir(d)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) continue;
        EzString name = ez_string_format(arena, "%s", ent->d_name);
        ez_array_push(arena, &arr, &name);
    }
    closedir(d);
    return arr;
}

bool ez_io_make_dir(EzString path) {
    return mkdir(path.data, 0755) == 0;
}

bool ez_io_make_dir_all(EzString path) {
    if (path.len == 0) return false;
    char buf[4096];
    if ((size_t)path.len >= sizeof(buf)) return false;
    memcpy(buf, path.data, (size_t)path.len);
    buf[path.len] = '\0';
    for (char *p = buf + 1; *p; p++) {
        if (*p == '/' || *p == '\\') {
            *p = '\0';
            mkdir(buf, 0755);
            *p = '/';
        }
    }
    return mkdir(buf, 0755) == 0 || errno == EEXIST;
}

bool ez_io_remove_dir(EzString path) {
    return rmdir(path.data) == 0;
}

static bool remove_dir_recursive(const char *path) {
    DIR *d = opendir(path);
    if (!d) return false;
    struct dirent *ent;
    bool ok = true;
    while ((ent = readdir(d)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) continue;
        char child[4096];
        snprintf(child, sizeof(child), "%s/%s", path, ent->d_name);
        struct stat st;
        if (stat(child, &st) != 0) { ok = false; continue; }
        if (S_ISDIR(st.st_mode)) {
            if (!remove_dir_recursive(child)) ok = false;
        } else {
            if (unlink(child) != 0) ok = false;
        }
    }
    closedir(d);
    if (rmdir(path) != 0) ok = false;
    return ok;
}

bool ez_io_remove_dir_all(EzString path) {
    return remove_dir_recursive(path.data);
}

static void walk_recursive(EzArena *arena, const char *base, const char *rel, EzArray *out) {
    char full[4096];
    if (rel[0] == '\0') {
        snprintf(full, sizeof(full), "%s", base);
    } else {
        snprintf(full, sizeof(full), "%s/%s", base, rel);
    }
    DIR *d = opendir(full);
    if (!d) return;
    struct dirent *ent;
    while ((ent = readdir(d)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) continue;
        EzString child_rel;
        if (rel[0] == '\0') {
            child_rel = ez_string_format(arena, "%s", ent->d_name);
        } else {
            child_rel = ez_string_format(arena, "%s/%s", rel, ent->d_name);
        }
        ez_array_push(arena, out, &child_rel);
        char child_full[4096];
        snprintf(child_full, sizeof(child_full), "%s/%s", full, ent->d_name);
        struct stat st;
        if (stat(child_full, &st) == 0 && S_ISDIR(st.st_mode)) {
            walk_recursive(arena, base, child_rel.data, out);
        }
    }
    closedir(d);
}

EzArray ez_io_walk(EzArena *arena, EzString path) {
    EzArray arr = ez_array_new(arena, (int32_t)sizeof(EzString), 32);
    walk_recursive(arena, path.data, "", &arr);
    return arr;
}

EzArray ez_io_glob(EzArena *arena, EzString pattern) {
    EzArray arr = ez_array_new(arena, (int32_t)sizeof(EzString), 16);
    glob_t gl;
    if (glob(pattern.data, GLOB_NOSORT, NULL, &gl) == 0) {
        for (size_t i = 0; i < gl.gl_pathc; i++) {
            EzString s = ez_string_format(arena, "%s", gl.gl_pathv[i]);
            ez_array_push(arena, &arr, &s);
        }
        globfree(&gl);
    }
    return arr;
}

/* ---- Tuple-returning (fallible) versions ---- */

EzResult_string ez_io_read_file_result(EzArena *arena, EzString path) {
    EzResult_string r;
    FILE *f = fopen(path.data, "rb");
    if (!f) {
        r.v0 = ez_string_lit("");
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot read '%s'", path.data));
        return r;
    }
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);
    char *buf = ez_arena_alloc(arena, (size_t)size + 1);
    size_t read = fread(buf, 1, (size_t)size, f);
    buf[read] = '\0';
    fclose(f);
    r.v0 = (EzString){ buf, (int32_t)read };
    r.v1 = NULL;
    return r;
}

EzResult_bool ez_io_write_file_result(EzArena *arena, EzString path, EzString content) {
    EzResult_bool r;
    FILE *f = fopen(path.data, "wb");
    if (!f) {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot write '%s'", path.data));
        return r;
    }
    fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    r.v0 = true;
    r.v1 = NULL;
    return r;
}

EzResult_bool ez_io_delete_file_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    struct stat st;
    if (stat(path.data, &st) == 0 && S_ISDIR(st.st_mode)) {
        fflush(stdout);
        fprintf(stderr, "panic: io.delete_file() cannot delete a directory — use io.delete_dir() for directories\n");
        exit(1);
    }
    if (unlink(path.data) == 0) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot delete '%s'", path.data));
    }
    return r;
}

EzResult_bool ez_io_copy_file_result(EzArena *arena, EzString src, EzString dst) {
    EzResult_bool r;
    if (ez_io_copy_file(src, dst)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot copy '%s' to '%s'", src.data, dst.data));
    }
    return r;
}

EzResult_bool ez_io_move_file_result(EzArena *arena, EzString src, EzString dst) {
    EzResult_bool r;
    if (ez_io_move_file(src, dst)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot move '%s' to '%s'", src.data, dst.data));
    }
    return r;
}

EzResult_array ez_io_list_dir_result(EzArena *arena, EzString path) {
    EzResult_array r;
    DIR *d = opendir(path.data);
    if (!d) {
        r.v0 = ez_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot list directory '%s'", path.data));
        return r;
    }
    closedir(d);
    r.v0 = ez_io_list_dir(arena, path);
    r.v1 = NULL;
    return r;
}

EzResult_bool ez_io_make_dir_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (ez_io_make_dir(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot create directory '%s'", path.data));
    }
    return r;
}

EzResult_bool ez_io_make_dir_all_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (ez_io_make_dir_all(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot create directories '%s'", path.data));
    }
    return r;
}

EzResult_bool ez_io_remove_dir_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (ez_io_remove_dir(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot remove directory '%s'", path.data));
    }
    return r;
}

EzResult_bool ez_io_remove_dir_all_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (ez_io_remove_dir_all(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot recursively remove '%s'", path.data));
    }
    return r;
}

EzResult_array ez_io_walk_result(EzArena *arena, EzString path) {
    EzResult_array r;
    DIR *d = opendir(path.data);
    if (!d) {
        r.v0 = ez_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = ez_error_new(arena, ez_string_format(arena, "cannot walk directory '%s'", path.data));
        return r;
    }
    closedir(d);
    r.v0 = ez_io_walk(arena, path);
    r.v1 = NULL;
    return r;
}
