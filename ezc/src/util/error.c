/*
 * error.c - Compiler diagnostic system
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "error.h"
#include "error_codes.h"
#include "constants.h"
#include "xalloc.h"
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#define GRAY_MAX_ERRORS_DISPLAYED 20
#define GRAY_DIAG_INITIAL_CAP     16
#define GRAY_DIAG_FORMAT_BUF      1024

/* --- ANSI color codes --- */

#define COL_RESET   "\033[0m"
#define COL_BOLD    "\033[1m"
#define COL_RED     "\033[31m"
#define COL_YELLOW  "\033[33m"
#define COL_CYAN    "\033[36m"
#define COL_BLUE    "\033[34m"
#define COL_WHITE   "\033[37m"

static const char *col(DiagnosticList *dl, const char *code) {
    return dl->use_color ? code : "";
}

/* --- Source line reading --- */

static const char *read_source_line_indexed(const char * const *offsets,
    int line_count, int line_num) {
    if (!offsets || line_num < 1 || line_num > line_count) return NULL;

    const char *line_start = offsets[line_num - 1];
    const char *line_end = line_start;
    while (*line_end && *line_end != '\n') line_end++;

    static char buf[GRAY_SOURCE_LINE_MAX];
    int len = (int)(line_end - line_start);
    if (len >= (int)sizeof(buf)) len = (int)sizeof(buf) - 1;
    memcpy(buf, line_start, len);
    buf[len] = '\0';
    return buf;
}

static void build_line_index(DiagSourceSlot *slot) {
    const char *source = slot->source;
    int count = 1;
    for (const char *p = source; *p; p++) {
        if (*p == '\n') count++;
    }
    slot->line_offsets = xmalloc(sizeof(const char *) * (size_t)count);
    slot->line_offsets[0] = source;
    slot->line_count = 1;
    for (const char *p = source; *p && slot->line_count < count; p++) {
        if (*p == '\n') slot->line_offsets[slot->line_count++] = p + 1;
    }
}

static const char *read_file_to_string(const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) return NULL;

    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *buf = malloc((size_t)size + 1);
    if (!buf) { fclose(f); return NULL; }

    size_t read = fread(buf, 1, (size_t)size, f);
    buf[read] = '\0';
    fclose(f);
    return buf;
}

/* --- DiagnosticList management --- */

DiagnosticList *diag_create(void) {
    DiagnosticList *dl = xmalloc(sizeof(DiagnosticList));
    memset(dl, 0, sizeof(DiagnosticList));
    dl->use_color = isatty(STDERR_FILENO);
    return dl;
}

void diag_destroy(DiagnosticList *dl) {
    free(dl->items);
    for (int i = 0; i < DIAG_FILE_CACHE_SIZE; i++) {
        if (dl->file_cache[i].owned) free((void *)dl->file_cache[i].source);
        free(dl->file_cache[i].line_offsets);
    }
    free(dl);
}

static void diag_add(DiagnosticList *dl, Severity sev, const char *code,
    const char *message, const char *file, int line, int col_start,
    int end_col, const char *help) {

    /* Cap errors at 20 to avoid flooding output */
    if (sev == SEV_ERROR && dl->error_count >= GRAY_MAX_ERRORS_DISPLAYED) return;

    if (dl->count >= dl->cap) {
        dl->cap = dl->cap ? dl->cap * 2 : GRAY_DIAG_INITIAL_CAP;
        dl->items = xrealloc(dl->items, sizeof(Diagnostic) * dl->cap);
    }

    Diagnostic *d = &dl->items[dl->count++];
    d->severity = sev;
    d->code = code;
    d->message = message;
    d->file = file;
    d->line = line;
    d->column = col_start;
    d->end_column = end_col;
    d->source_line = NULL;
    d->help = help;

    if (sev == SEV_ERROR) dl->error_count++;
    else if (sev == SEV_WARNING) dl->warning_count++;
}

void diag_error(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_ERROR, code, message, file, line, col_start, end_col, NULL);
}

void diag_error_help(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col, const char *help) {
    diag_add(dl, SEV_ERROR, code, message, file, line, col_start, end_col, help);
}

void diag_warning(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_WARNING, code, message, file, line, col_start, end_col, NULL);
}

void diag_warning_help(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col, const char *help) {
    diag_add(dl, SEV_WARNING, code, message, file, line, col_start, end_col, help);
}

void diag_note(DiagnosticList *dl, const char *message,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_NOTE, NULL, message, file, line, col_start, end_col, NULL);
}

static const char *lookup_or_placeholder(const char *code) {
    const char *msg = gray_error_message(code);
    return msg ? msg : "<unknown error code>";
}

void diag_error_code(DiagnosticList *dl, const char *code,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_ERROR, code, lookup_or_placeholder(code),
        file, line, col_start, end_col, NULL);
}

void diag_error_code_help(DiagnosticList *dl, const char *code,
    const char *file, int line, int col_start, int end_col, const char *help) {
    diag_add(dl, SEV_ERROR, code, lookup_or_placeholder(code),
        file, line, col_start, end_col, help);
}

void diag_warning_code(DiagnosticList *dl, const char *code,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_WARNING, code, lookup_or_placeholder(code),
        file, line, col_start, end_col, NULL);
}

void diag_error_msg(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_ERROR, code, message, file, line, col_start, end_col, NULL);
}

void diag_warning_msg(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col_start, int end_col) {
    diag_add(dl, SEV_WARNING, code, message, file, line, col_start, end_col, NULL);
}

/* strdup is used so the formatted buffer outlives this stack frame;
 * the diagnostic list stores the pointer by reference. The tiny leak
 * per error is acceptable for a short-lived compiler invocation. */
static void emit_codef(DiagnosticList *dl, Severity sev, const char *code,
    const char *file, int line, int col_start, int end_col, va_list ap) {
    const char *tmpl = lookup_or_placeholder(code);
    char buf[GRAY_DIAG_FORMAT_BUF];
    vsnprintf(buf, sizeof(buf), tmpl, ap);
    diag_add(dl, sev, code, strdup(buf), file, line, col_start, end_col, NULL);
}

void diag_error_codef(DiagnosticList *dl, const char *code,
    const char *file, int line, int col_start, int end_col, ...) {
    va_list ap;
    va_start(ap, end_col);
    emit_codef(dl, SEV_ERROR, code, file, line, col_start, end_col, ap);
    va_end(ap);
}

void diag_warning_codef(DiagnosticList *dl, const char *code,
    const char *file, int line, int col_start, int end_col, ...) {
    va_list ap;
    va_start(ap, end_col);
    emit_codef(dl, SEV_WARNING, code, file, line, col_start, end_col, ap);
    va_end(ap);
}

void diag_set_source(DiagnosticList *dl, const char *file, const char *source) {
    /* Slot 0 is the primary entry-file slot. Source is caller-owned — never freed here. */
    DiagSourceSlot *s = &dl->file_cache[0];
    free(s->line_offsets);
    s->path = file;
    s->source = source;
    s->owned = false;
    s->line_offsets = NULL;
    s->line_count = 0;
    s->last_use = 0;
    if (source) build_line_index(s);
}

bool diag_has_errors(DiagnosticList *dl) {
    return diag_error_count(dl) > 0;
}

int diag_error_count(DiagnosticList *dl) {
    return dl->error_count;
}

int diag_warning_count(DiagnosticList *dl) {
    return dl->warning_count;
}

/* --- Rendering --- */

static void print_diagnostic(DiagnosticList *dl, Diagnostic *d) {
    const char *sev_str;
    const char *sev_color;

    switch (d->severity) {
    case SEV_ERROR:
        sev_str = "error";
        sev_color = COL_RED;
        break;
    case SEV_WARNING:
        sev_str = "warning";
        sev_color = COL_YELLOW;
        break;
    case SEV_NOTE:
        sev_str = "note";
        sev_color = COL_CYAN;
        break;
    }

    /* Line 1: severity[code]: message */
    fprintf(stderr, "%s%s%s%s", col(dl, COL_BOLD), col(dl, sev_color), sev_str, col(dl, COL_RESET));
    if (d->code) {
        fprintf(stderr, "%s%s[%s]%s", col(dl, COL_BOLD), col(dl, sev_color), d->code, col(dl, COL_RESET));
    }
    fprintf(stderr, "%s: %s%s\n", col(dl, COL_BOLD), d->message, col(dl, COL_RESET));

    /* Line 2: --> file:line:column */
    if (d->file && d->line > 0) {
        fprintf(stderr, "  %s-->%s %s:%d:%d\n",
            col(dl, COL_BLUE), col(dl, COL_RESET),
            d->file, d->line, d->column);
    }

    /* Lines 3-4: source context with underline */
    const char *src_line = d->source_line;
    if (!src_line && d->file) {
        /* Search cache slots for this file */
        DiagSourceSlot *slot = NULL;
        for (int ci = 0; ci < DIAG_FILE_CACHE_SIZE; ci++) {
            DiagSourceSlot *s = &dl->file_cache[ci];
            if (s->source && s->path && strcmp(s->path, d->file) == 0) {
                slot = s;
                break;
            }
        }
        if (!slot) {
            /* Cache miss: read from disk and store in the LRU secondary slot */
            const char *content = read_file_to_string(d->file);
            if (content) {
                int evict = 1;
                for (int ci = 2; ci < DIAG_FILE_CACHE_SIZE; ci++) {
                    if (dl->file_cache[ci].last_use < dl->file_cache[evict].last_use)
                        evict = ci;
                }
                DiagSourceSlot *s = &dl->file_cache[evict];
                if (s->owned) free((void *)s->source);
                free(s->line_offsets);
                s->path = d->file;
                s->source = content;
                s->owned = true;
                s->line_offsets = NULL;
                s->line_count = 0;
                build_line_index(s);
                slot = s;
            }
        }
        if (slot) {
            slot->last_use = ++dl->cache_clock;
            src_line = read_source_line_indexed(slot->line_offsets, slot->line_count, d->line);
        }
    }

    if (src_line && d->line > 0) {
        /* Line number gutter */
        fprintf(stderr, "   %s|%s\n", col(dl, COL_BLUE), col(dl, COL_RESET));
        fprintf(stderr, "%s%3d%s %s|%s %s\n",
            col(dl, COL_BLUE), d->line, col(dl, COL_RESET),
            col(dl, COL_BLUE), col(dl, COL_RESET),
            src_line);

        /* Underline */
        int start = d->column > 0 ? d->column : 1;
        int end = d->end_column > 0 ? d->end_column : start;
        int span_len = end - start + 1;
        if (span_len < 1) span_len = 1;

        fprintf(stderr, "   %s|%s ", col(dl, COL_BLUE), col(dl, COL_RESET));
        for (int i = 1; i < start; i++) {
            fputc(' ', stderr);
        }
        fprintf(stderr, "%s%s", col(dl, COL_BOLD), col(dl, sev_color));
        for (int i = 0; i < span_len; i++) {
            fputc('^', stderr);
        }
        fprintf(stderr, "%s\n", col(dl, COL_RESET));
    }

    /* Help text */
    if (d->help) {
        fprintf(stderr, "   %s|%s\n", col(dl, COL_BLUE), col(dl, COL_RESET));
        fprintf(stderr, "   %s=%s %shelp%s: %s\n",
            col(dl, COL_BLUE), col(dl, COL_RESET),
            col(dl, COL_CYAN), col(dl, COL_RESET),
            d->help);
    }

    fprintf(stderr, "\n");
}

static bool is_warning_suppressed(DiagnosticList *dl, Diagnostic *d) {
    if (d->severity != SEV_WARNING) return false;
    if (dl->suppress_all_warnings) return true;
    if (d->code) {
        for (int i = 0; i < dl->suppressed_count; i++) {
            if (strcmp(d->code, dl->suppressed_codes[i]) == 0) return true;
        }
    }
    return false;
}

void diag_print_all(DiagnosticList *dl) {
    for (int i = 0; i < dl->count; i++) {
        if (is_warning_suppressed(dl, &dl->items[i])) continue;
        print_diagnostic(dl, &dl->items[i]);
    }
}

void diag_print_summary(DiagnosticList *dl) {
    int errors = dl->error_count;

    /* visible_warnings excludes suppressed entries — checked at render time */
    int warnings = 0;
    for (int i = 0; i < dl->count; i++) {
        if (dl->items[i].severity == SEV_WARNING &&
            !is_warning_suppressed(dl, &dl->items[i]))
            warnings++;
    }

    if (errors == 0 && warnings == 0) return;

    fprintf(stderr, "%sez:%s ", col(dl, COL_BOLD), col(dl, COL_RESET));

    if (errors > 0) {
        fprintf(stderr, "%s%s%d error%s%s",
            col(dl, COL_BOLD), col(dl, COL_RED),
            errors, errors == 1 ? "" : "s",
            col(dl, COL_RESET));
    }
    if (errors > 0 && warnings > 0) {
        fprintf(stderr, ", ");
    }
    if (warnings > 0) {
        fprintf(stderr, "%s%s%d warning%s%s",
            col(dl, COL_BOLD), col(dl, COL_YELLOW),
            warnings, warnings == 1 ? "" : "s",
            col(dl, COL_RESET));
    }

    if (errors > 0) {
        fprintf(stderr, ". compilation failed.\n");
    } else {
        fprintf(stderr, "\n");
    }

    if (warnings > 0 && !dl->suppress_all_warnings && dl->suppressed_count == 0) {
        fprintf(stderr, "%shint:%s suppress warnings with -q <W1001,W1002,...> or -q 'all'\n",
            col(dl, COL_BOLD), col(dl, COL_RESET));
    }
}
