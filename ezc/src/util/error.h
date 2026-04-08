/*
 * error.h - Compiler diagnostic system
 *
 * Provides structured error/warning reporting with source context,
 * span underlines, colored output, and help text.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_ERROR_H
#define EZC_ERROR_H

#include <stdbool.h>

typedef enum {
    SEV_ERROR,
    SEV_WARNING,
    SEV_NOTE,
} Severity;

typedef struct {
    Severity severity;
    const char *code;       /* "E3018", "W2001", etc. */
    const char *message;    /* Primary error message */
    const char *file;
    int line;               /* 1-indexed */
    int column;             /* 1-indexed start column */
    int end_column;         /* 1-indexed end column (0 = same as column) */
    const char *source_line;/* The actual source line (optional, will be read if NULL) */
    const char *help;       /* Optional help/suggestion text */
} Diagnostic;

typedef struct {
    Diagnostic *items;
    int count;
    int cap;

    /* Source file cache for reading lines */
    const char *cached_file;
    const char *cached_source;

    /* Options */
    bool use_color;

    /* Warning suppression (-q / --quiet) */
    bool suppress_all_warnings;
    const char **suppressed_codes;
    int suppressed_count;
} DiagnosticList;

/* Create/destroy */
DiagnosticList *diag_create(void);
void diag_destroy(DiagnosticList *dl);

/* Add diagnostics */
void diag_error(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col, int end_col);

void diag_error_help(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col, int end_col, const char *help);

void diag_warning(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col, int end_col);

void diag_warning_help(DiagnosticList *dl, const char *code, const char *message,
    const char *file, int line, int col, int end_col, const char *help);

void diag_note(DiagnosticList *dl, const char *message,
    const char *file, int line, int col, int end_col);

/* Set the source text for a file (avoids re-reading from disk) */
void diag_set_source(DiagnosticList *dl, const char *file, const char *source);

/* Query */
bool diag_has_errors(DiagnosticList *dl);
int diag_error_count(DiagnosticList *dl);
int diag_warning_count(DiagnosticList *dl);

/* Render all diagnostics to stderr */
void diag_print_all(DiagnosticList *dl);

/* Print summary line: "ezc: 2 errors, 1 warning" */
void diag_print_summary(DiagnosticList *dl);

#endif
