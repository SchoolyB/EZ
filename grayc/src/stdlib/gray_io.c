/*
 * gray_io.c - @io module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "gray_io.h"
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>
#include <errno.h>
#include <glob.h>

#define GRAY_IO_PATH_BUF          4096
#define GRAY_IO_COPY_BUF          8192
#define GRAY_IO_MAX_SEGMENTS      256
#define GRAY_IO_DIR_MODE          0755
#define GRAY_IO_FILE_MODE         0644
#define GRAY_IO_WALK_INITIAL_CAP  32

/* ---- Path manipulation (pure, no I/O) ---- */

EzString gray_io_path_join(EzArena *arena, EzArray parts) {
    if (parts.len == 0) return gray_string_lit(".");
    EzString result = GRAY_ARRAY_GET(parts, EzString, 0);
    for (int32_t i = 1; i < parts.len; i++) {
        EzString seg = GRAY_ARRAY_GET(parts, EzString, i);
        if (seg.len == 0) continue;
        /* Absolute segment replaces accumulated path */
        if (seg.data[0] == '/' || seg.data[0] == '\\') {
            result = seg;
            continue;
        }
        bool a_sep = result.len > 0 &&
            (result.data[result.len - 1] == '/' || result.data[result.len - 1] == '\\');
        if (a_sep)
            result = gray_string_format(arena, "%.*s%.*s", result.len, result.data, seg.len, seg.data);
        else
            result = gray_string_format(arena, "%.*s/%.*s", result.len, result.data, seg.len, seg.data);
    }
    return result;
}

EzString gray_io_dirname(EzArena *arena, EzString path) {
    if (path.len == 0) return gray_string_lit(".");
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
    if (last_sep < 0) return gray_string_lit(".");
    /* Collapse leading separator: dirname("/foo") -> "/" */
    if (last_sep == 0) return gray_string_lit("/");
    char *buf = gray_arena_alloc(arena, (size_t)last_sep + 1);
    memcpy(buf, path.data, (size_t)last_sep);
    buf[last_sep] = '\0';
    return (EzString){ buf, (int32_t)last_sep };
}

EzString gray_io_basename(EzArena *arena, EzString path) {
    (void)arena;
    if (path.len == 0) return gray_string_lit(".");
    int end = path.len;
    while (end > 0 && (path.data[end - 1] == '/' || path.data[end - 1] == '\\')) end--;
    if (end == 0) return gray_string_lit("/");
    int start = end;
    while (start > 0 && path.data[start - 1] != '/' && path.data[start - 1] != '\\') start--;
    int32_t len = (int32_t)(end - start);
    char *buf = gray_arena_alloc(arena, (size_t)len + 1);
    memcpy(buf, path.data + start, (size_t)len);
    buf[len] = '\0';
    return (EzString){ buf, len };
}

EzString gray_io_extension(EzArena *arena, EzString path) {
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
    if (dot_pos < 0 || dot_pos == path.len - 1) return gray_string_lit("");
    /* Dotfiles: leading dot with no other dot is part of the name, not an extension */
    if (dot_pos == search_start) return gray_string_lit("");
    int32_t len = (int32_t)(path.len - dot_pos);
    char *buf = gray_arena_alloc(arena, (size_t)len + 1);
    memcpy(buf, path.data + dot_pos, (size_t)len);
    buf[len] = '\0';
    return (EzString){ buf, len };
}

bool gray_io_is_absolute(EzString path) {
    return path.len > 0 && path.data[0] == '/';
}

EzString gray_io_normalize(EzArena *arena, EzString path) {
    if (path.len == 0) return gray_string_lit(".");
    char *buf = gray_arena_alloc(arena, (size_t)path.len + 1);
    /* Copy input, converting backslashes to forward slashes */
    for (int i = 0; i < path.len; i++) {
        buf[i] = (path.data[i] == '\\') ? '/' : path.data[i];
    }
    buf[path.len] = '\0';
    bool absolute = (buf[0] == '/');

    /* Split into segments */
    char *segments[GRAY_IO_MAX_SEGMENTS];
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
            if (seg_count < GRAY_IO_MAX_SEGMENTS) segments[seg_count++] = seg_start;
        }
    }

    /* Rebuild */
    char *out = gray_arena_alloc(arena, (size_t)path.len + 2);
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

EzString gray_io_read_file(EzArena *arena, EzString path) {
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0086", "io.read_file() cannot read a directory; use io.list_dir() or io.walk() to list directory contents");
    FILE *f = fopen(path.data, "rb");
    if (!f) return gray_string_lit("");

    /* Try to size the buffer from the file. Falls back to streaming
     * when the file isn't seekable (pipes, FIFOs, /dev/stdin, /proc,
     * sockets). On those, ftell returns -1, which used to wrap to
     * (size_t)-1 + 1 = 0 and let fread write past the arena slot. */
    long size = -1;
    if (fseek(f, 0, SEEK_END) == 0) {
        size = ftell(f);
        if (size >= 0) (void)fseek(f, 0, SEEK_SET);
    }

    if (size >= 0 && size <= INT32_MAX) {
        char *buf = gray_arena_alloc(arena, (size_t)size + 1);
        size_t read = fread(buf, 1, (size_t)size, f);
        buf[read] = '\0';
        fclose(f);
        return (EzString){ buf, (int32_t)read };
    }

    /* Streaming fallback for non-seekable inputs. */
    clearerr(f);
    size_t cap = 4096;
    size_t len = 0;
    char *buf = gray_arena_alloc(arena, cap);
    for (;;) {
        if (len == cap) {
            if (cap > (size_t)INT32_MAX / 2) {
                fclose(f);
                gray_panic_code("P0053", "io.read_file: input exceeds maximum string length");
            }
            size_t new_cap = cap * 2;
            char *new_buf = gray_arena_alloc(arena, new_cap);
            memcpy(new_buf, buf, len);
            buf = new_buf;
            cap = new_cap;
        }
        size_t got = fread(buf + len, 1, cap - len, f);
        if (got == 0) break;
        len += got;
    }
    if (len == cap) {
        char *grow = gray_arena_alloc(arena, len + 1);
        memcpy(grow, buf, len);
        buf = grow;
    }
    buf[len] = '\0';
    fclose(f);
    return (EzString){ buf, (int32_t)len };
}

EzArray gray_io_read_bytes(EzArena *arena, EzString path) {
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0086", "io.read_bytes() cannot read a directory");
    FILE *f = fopen(path.data, "rb");
    EzArray arr = gray_array_new(arena, (int32_t)sizeof(uint8_t), 0);
    if (!f) return arr;
    uint8_t buf[4096];
    size_t n;
    while ((n = fread(buf, 1, sizeof(buf), f)) > 0) {
        for (size_t i = 0; i < n; i++)
            gray_array_push(arena, &arr, &buf[i]);
    }
    fclose(f);
    return arr;
}

EzArray gray_io_read_lines(EzArena *arena, EzString path) {
    EzArray arr = gray_array_new(arena, (int32_t)sizeof(EzString), 16);
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0086", "io.read_lines() cannot read a directory");
    EzString content = gray_io_read_file(arena, path);
    if (content.len == 0) return arr;
    const char *p = content.data;
    const char *end = content.data + content.len;
    while (p < end) {
        const char *nl = p;
        while (nl < end && *nl != '\n') nl++;
        int32_t len = (int32_t)(nl - p);
        /* Strip trailing \r for Windows line endings */
        if (len > 0 && p[len - 1] == '\r') len--;
        char *linebuf = gray_arena_alloc(arena, (size_t)len + 1);
        memcpy(linebuf, p, (size_t)len);
        linebuf[len] = '\0';
        EzString line = { linebuf, len };
        gray_array_push(arena, &arr, &line);
        p = nl + 1;
    }
    return arr;
}

bool gray_io_file_exists(EzString path) {
    return access(path.data, F_OK) == 0;
}

bool gray_io_is_file(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return false;
    return S_ISREG(st.st_mode);
}

bool gray_io_is_directory(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return false;
    return S_ISDIR(st.st_mode);
}

int64_t gray_io_file_size(EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0) return -1;
    return (int64_t)st.st_size;
}

EzResult_int gray_io_file_size_result(EzArena *arena, EzString path) {
    struct stat st;
    if (stat(path.data, &st) != 0)
        return (EzResult_int){-1, gray_error_new(arena,
            gray_string_format(arena, "cannot stat '%s'", path.data))};
    return (EzResult_int){(int64_t)st.st_size, NULL};
}

bool gray_io_write_file(EzString path, EzString content) {
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0087", "io.write_file() cannot write to a directory");
    FILE *f = fopen(path.data, "wb");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    return written == (size_t)content.len;
}

bool gray_io_append_file(EzString path, EzString content) {
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0088", "io.append_file() cannot append to a directory");
    FILE *f = fopen(path.data, "ab");
    if (!f) return false;
    size_t written = fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    return written == (size_t)content.len;
}

bool gray_io_delete_file(EzString path) {
    struct stat st;
    if (stat(path.data, &st) == 0 && S_ISDIR(st.st_mode))
        gray_panic_code("P0077", "io.delete_file() cannot delete a directory; use io.remove_dir() for directories");
    return unlink(path.data) == 0;
}

bool gray_io_rename_file(EzString old_path, EzString new_path) {
    return rename(old_path.data, new_path.data) == 0;
}

/* ---- New file operations ---- */

bool gray_io_copy_file(EzString src, EzString dst) {
    struct stat _st;
    if (stat(src.data, &_st) == 0 && S_ISDIR(_st.st_mode))
        gray_panic_code("P0089", "io.copy_file() cannot copy a directory; use io.walk() to enumerate files and copy them individually");
    FILE *in = fopen(src.data, "rb");
    if (!in) return false;
    int out_fd = open(dst.data, O_WRONLY | O_CREAT | O_TRUNC, GRAY_IO_FILE_MODE);
    if (out_fd < 0) { fclose(in); return false; }
    FILE *out = fdopen(out_fd, "wb");
    if (!out) { close(out_fd); fclose(in); return false; }
    char buf[GRAY_IO_COPY_BUF];
    size_t n;
    bool ok = true;
    while ((n = fread(buf, 1, sizeof(buf), in)) > 0) {
        if (fwrite(buf, 1, n, out) != n) { ok = false; break; }
    }
    fclose(in);
    fclose(out);
    return ok;
}

bool gray_io_move_file(EzString src, EzString dst) {
    if (rename(src.data, dst.data) == 0) return true;
    if (!gray_io_copy_file(src, dst)) return false;
    unlink(src.data);
    return true;
}

/* ---- Directory operations ---- */

EzArray gray_io_list_dir(EzArena *arena, EzString path) {
    EzArray arr = gray_array_new(arena, (int32_t)sizeof(EzString), 16);
    DIR *d = opendir(path.data);
    if (!d) return arr;
    struct dirent *ent;
    while ((ent = readdir(d)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) continue;
        EzString name = gray_string_format(arena, "%s", ent->d_name);
        gray_array_push(arena, &arr, &name);
    }
    closedir(d);
    return arr;
}

bool gray_io_make_dir(EzString path) {
    return mkdir(path.data, GRAY_IO_DIR_MODE) == 0;
}

bool gray_io_make_dir_all(EzString path) {
    if (path.len == 0) return false;
    char buf[GRAY_IO_PATH_BUF];
    if ((size_t)path.len >= sizeof(buf)) return false;
    memcpy(buf, path.data, (size_t)path.len);
    buf[path.len] = '\0';
    for (char *p = buf + 1; *p; p++) {
        if (*p == '/' || *p == '\\') {
            *p = '\0';
            mkdir(buf, GRAY_IO_DIR_MODE);
            *p = '/';
        }
    }
    return mkdir(buf, GRAY_IO_DIR_MODE) == 0 || errno == EEXIST;
}

bool gray_io_remove_dir(EzString path) {
    return rmdir(path.data) == 0;
}

static bool remove_dir_recursive(const char *path) {
    DIR *d = opendir(path);
    if (!d) return false;
    struct dirent *ent;
    bool ok = true;
    while ((ent = readdir(d)) != NULL) {
        if (strcmp(ent->d_name, ".") == 0 || strcmp(ent->d_name, "..") == 0) continue;
        char child[GRAY_IO_PATH_BUF];
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

bool gray_io_remove_dir_all(EzString path) {
    return remove_dir_recursive(path.data);
}

static void walk_recursive(EzArena *arena, const char *base, const char *rel, EzArray *out) {
    char full[GRAY_IO_PATH_BUF];
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
            child_rel = gray_string_format(arena, "%s", ent->d_name);
        } else {
            child_rel = gray_string_format(arena, "%s/%s", rel, ent->d_name);
        }
        gray_array_push(arena, out, &child_rel);
        char child_full[GRAY_IO_PATH_BUF];
        snprintf(child_full, sizeof(child_full), "%s/%s", full, ent->d_name);
        struct stat st;
        if (stat(child_full, &st) == 0 && S_ISDIR(st.st_mode)) {
            walk_recursive(arena, base, child_rel.data, out);
        }
    }
    closedir(d);
}

EzArray gray_io_walk(EzArena *arena, EzString path) {
    EzArray arr = gray_array_new(arena, (int32_t)sizeof(EzString), GRAY_IO_WALK_INITIAL_CAP);
    walk_recursive(arena, path.data, "", &arr);
    return arr;
}

EzArray gray_io_glob(EzArena *arena, EzString pattern) {
    EzArray arr = gray_array_new(arena, (int32_t)sizeof(EzString), 16);
    glob_t gl;
    if (glob(pattern.data, GLOB_NOSORT, NULL, &gl) == 0) {
        for (size_t i = 0; i < gl.gl_pathc; i++) {
            EzString s = gray_string_format(arena, "%s", gl.gl_pathv[i]);
            gray_array_push(arena, &arr, &s);
        }
        globfree(&gl);
    }
    return arr;
}

/* ---- Tuple-returning (fallible) versions ---- */

EzResult_string gray_io_read_file_result(EzArena *arena, EzString path) {
    EzResult_string r;
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = gray_string_lit("");
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot read '%s': is a directory", path.data));
        return r;
    }
    FILE *f = fopen(path.data, "rb");
    if (!f) {
        r.v0 = gray_string_lit("");
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot read '%s'", path.data));
        return r;
    }
    long size = -1;
    if (fseek(f, 0, SEEK_END) == 0) {
        size = ftell(f);
        if (size >= 0) (void)fseek(f, 0, SEEK_SET);
    }
    if (size >= 0 && size <= INT32_MAX) {
        char *buf = gray_arena_alloc(arena, (size_t)size + 1);
        size_t bytes = fread(buf, 1, (size_t)size, f);
        buf[bytes] = '\0';
        fclose(f);
        r.v0 = (EzString){ buf, (int32_t)bytes };
        r.v1 = NULL;
        return r;
    }
    if (size > INT32_MAX) {
        fclose(f);
        r.v0 = gray_string_lit("");
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot read '%s': file exceeds maximum string length", path.data));
        return r;
    }
    /* Streaming fallback for non-seekable inputs (pipes, /dev/stdin, etc.) */
    clearerr(f);
    size_t cap = 4096;
    size_t len = 0;
    char *buf = gray_arena_alloc(arena, cap);
    for (;;) {
        if (len == cap) {
            if (cap > (size_t)INT32_MAX / 2) {
                fclose(f);
                gray_panic_code("P0053", "io.read_file: input exceeds maximum string length");
            }
            size_t new_cap = cap * 2;
            char *new_buf = gray_arena_alloc(arena, new_cap);
            memcpy(new_buf, buf, len);
            buf = new_buf;
            cap = new_cap;
        }
        size_t got = fread(buf + len, 1, cap - len, f);
        if (got == 0) break;
        len += got;
    }
    if (len == cap) {
        char *grow = gray_arena_alloc(arena, len + 1);
        memcpy(grow, buf, len);
        buf = grow;
    }
    buf[len] = '\0';
    fclose(f);
    r.v0 = (EzString){ buf, (int32_t)len };
    r.v1 = NULL;
    return r;
}

EzResult_bool gray_io_write_file_result(EzArena *arena, EzString path, EzString content) {
    EzResult_bool r;
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot write '%s': is a directory", path.data));
        return r;
    }
    FILE *f = fopen(path.data, "wb");
    if (!f) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot write '%s'", path.data));
        return r;
    }
    fwrite(content.data, 1, (size_t)content.len, f);
    fclose(f);
    r.v0 = true;
    r.v1 = NULL;
    return r;
}

EzResult_bool gray_io_delete_file_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    struct stat st;
    if (stat(path.data, &st) == 0 && S_ISDIR(st.st_mode)) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot delete '%s': is a directory; use io.remove_dir() for directories", path.data));
        return r;
    }
    if (unlink(path.data) == 0) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot delete '%s'", path.data));
    }
    return r;
}

EzResult_bool gray_io_append_file_result(EzArena *arena, EzString path, EzString content) {
    EzResult_bool r;
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot append to '%s': is a directory", path.data));
        return r;
    }
    if (gray_io_append_file(path, content)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot append to '%s'", path.data));
    }
    return r;
}

EzResult_bool gray_io_rename_file_result(EzArena *arena, EzString old_path, EzString new_path) {
    EzResult_bool r;
    if (gray_io_rename_file(old_path, new_path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot rename '%s' to '%s'", old_path.data, new_path.data));
    }
    return r;
}

EzResult_bool gray_io_copy_file_result(EzArena *arena, EzString src, EzString dst) {
    EzResult_bool r;
    struct stat _st;
    if (stat(src.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot copy '%s': is a directory", src.data));
        return r;
    }
    if (gray_io_copy_file(src, dst)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot copy '%s' to '%s'", src.data, dst.data));
    }
    return r;
}

EzResult_bool gray_io_move_file_result(EzArena *arena, EzString src, EzString dst) {
    EzResult_bool r;
    if (gray_io_move_file(src, dst)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot move '%s' to '%s'", src.data, dst.data));
    }
    return r;
}

EzResult_array gray_io_list_dir_result(EzArena *arena, EzString path) {
    EzResult_array r;
    DIR *d = opendir(path.data);
    if (!d) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot list directory '%s'", path.data));
        return r;
    }
    closedir(d);
    r.v0 = gray_io_list_dir(arena, path);
    r.v1 = NULL;
    return r;
}

EzResult_bool gray_io_make_dir_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (gray_io_make_dir(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot create directory '%s'", path.data));
    }
    return r;
}

EzResult_bool gray_io_make_dir_all_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (gray_io_make_dir_all(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot create directories '%s'", path.data));
    }
    return r;
}

EzResult_bool gray_io_remove_dir_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (gray_io_remove_dir(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot remove directory '%s'", path.data));
    }
    return r;
}

EzResult_bool gray_io_remove_dir_all_result(EzArena *arena, EzString path) {
    EzResult_bool r;
    if (gray_io_remove_dir_all(path)) {
        r.v0 = true;
        r.v1 = NULL;
    } else {
        r.v0 = false;
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot recursively remove '%s'", path.data));
    }
    return r;
}

EzResult_array gray_io_walk_result(EzArena *arena, EzString path) {
    EzResult_array r;
    DIR *d = opendir(path.data);
    if (!d) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot walk directory '%s'", path.data));
        return r;
    }
    closedir(d);
    r.v0 = gray_io_walk(arena, path);
    r.v1 = NULL;
    return r;
}

EzResult_array gray_io_read_bytes_result(EzArena *arena, EzString path) {
    EzResult_array r;
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(uint8_t), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot read '%s': is a directory", path.data));
        return r;
    }
    FILE *f = fopen(path.data, "rb");
    if (!f) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(uint8_t), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena, "cannot read '%s'", path.data));
        return r;
    }
    r.v0 = gray_array_new(arena, (int32_t)sizeof(uint8_t), 0);
    uint8_t buf[4096];
    size_t n;
    while ((n = fread(buf, 1, sizeof(buf), f)) > 0) {
        for (size_t i = 0; i < n; i++)
            gray_array_push(arena, &r.v0, &buf[i]);
    }
    fclose(f);
    r.v1 = NULL;
    return r;
}

EzResult_array gray_io_read_lines_result(EzArena *arena, EzString path) {
    EzResult_array r;
    struct stat _st;
    if (stat(path.data, &_st) == 0 && S_ISDIR(_st.st_mode)) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "cannot read '%s': is a directory", path.data));
        return r;
    }
    EzResult_string fr = gray_io_read_file_result(arena, path);
    if (fr.v1 != NULL) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = fr.v1;
        return r;
    }
    r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 16);
    EzString content = fr.v0;
    const char *p = content.data;
    const char *end = content.data + content.len;
    while (p < end) {
        const char *nl = p;
        while (nl < end && *nl != '\n') nl++;
        int32_t len = (int32_t)(nl - p);
        if (len > 0 && p[len - 1] == '\r') len--;
        char *linebuf = gray_arena_alloc(arena, (size_t)len + 1);
        memcpy(linebuf, p, (size_t)len);
        linebuf[len] = '\0';
        EzString line = { linebuf, len };
        gray_array_push(arena, &r.v0, &line);
        p = nl + 1;
    }
    r.v1 = NULL;
    return r;
}

EzResult_array gray_io_glob_result(EzArena *arena, EzString pattern) {
    EzResult_array r;
    glob_t gl;
    int rc = glob(pattern.data, GLOB_NOSORT, NULL, &gl);
    if (rc != 0 && rc != GLOB_NOMATCH) {
        r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), 0);
        r.v1 = gray_error_new(arena, gray_string_format(arena,
            "glob pattern failed: '%s'", pattern.data));
        return r;
    }
    r.v0 = gray_array_new(arena, (int32_t)sizeof(EzString), (int32_t)gl.gl_pathc);
    for (size_t i = 0; i < gl.gl_pathc; i++) {
        EzString s = gray_string_format(arena, "%s", gl.gl_pathv[i]);
        gray_array_push(arena, &r.v0, &s);
    }
    globfree(&gl);
    r.v1 = NULL;
    return r;
}
