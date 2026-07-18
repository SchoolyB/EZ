/*
 * fmt.c — Grayscale source formatter implementation. Lexes the source to
 * build a per-line indentation depth table based on brace nesting, then
 * re-emits each original line with corrected leading whitespace.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "fmt.h"
#include "../lexer/lexer.h"
#include "../lexer/token.h"
#include "../util/arena.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <ctype.h>

#define FMT_INDENT_WIDTH 4
#define FMT_ARENA_SIZE   (1024 * 1024)  /* 1 MB — enough for any source file */

/* Count the number of newlines in src to find the maximum line number. */
static int count_lines(const char *src) {
    int n = 1;
    for (const char *p = src; *p; p++) {
        if (*p == '\n') n++;
    }
    return n;
}

/*
 * Build a depth table: depth_at[line] = indentation depth at the START of
 * that line (1-indexed).  Lines that contain no tokens (blank lines or
 * comment-only lines) inherit the depth of the nearest preceding token line.
 *
 * TOK_RBRACE decrements the depth BEFORE the line is recorded, so a closing
 * brace line gets the outer depth.  TOK_LBRACE increments the depth AFTER
 * the line is recorded, so the opening-brace line itself stays at the current
 * depth and the body lines inside get depth+1.
 */
static int *build_depth_table(const char *src, const char *filename, int max_line) {
    int *table = calloc(max_line + 2, sizeof(int));
    if (!table) return NULL;

    /* -1 = not yet assigned by a token */
    for (int i = 0; i <= max_line + 1; i++) table[i] = -1;

    Arena *arena = arena_create(FMT_ARENA_SIZE);
    if (!arena) { free(table); return NULL; }

    Lexer *lexer = lexer_create(arena, src, filename);
    if (!lexer) { arena_destroy(arena); free(table); return NULL; }

    int depth = 0;
    Token token;
    while ((token = lexer_next_token(lexer)).type != TOK_EOF) {
        if (token.type == TOK_ILLEGAL) break;
        if (token.type == TOK_NEWLINE) continue;

        int line = token.line;
        if (line < 1 || line > max_line) continue;

        /* Closing brace: dedent first, then record */
        if (token.type == TOK_RBRACE) {
            if (depth > 0) depth--;
        }

        /* Record depth for this line if not yet set */
        if (table[line] < 0) {
            table[line] = depth;
        }

        /* Opening brace: indent for subsequent lines */
        if (token.type == TOK_LBRACE) {
            depth++;
        }
    }

    arena_destroy(arena);

    /* Forward-fill gaps: blank lines and comment-only lines inherit the
     * depth of the most recent token-bearing line. */
    int current = 0;
    for (int i = 1; i <= max_line; i++) {
        if (table[i] >= 0) {
            current = table[i];
        } else {
            table[i] = current;
        }
    }

    return table;
}

/*
 * Returns true if `line` (NUL-terminated, with leading whitespace already
 * stripped) is inside a raw string that started on an earlier line.
 * We track this by counting unpaired backticks across the whole source up to
 * the given line number.  An odd count means we are inside a raw string.
 */
static bool line_is_in_raw_string(const char *src, int target_line) {
    int line = 1;
    bool in_raw = false;
    bool in_str = false;  /* regular double-quoted string */
    const char *p = src;
    while (*p && line < target_line) {
        if (*p == '\n') { line++; p++; continue; }
        if (in_raw) {
            if (*p == '`') in_raw = false;
            p++; continue;
        }
        if (in_str) {
            if (*p == '\\' && *(p + 1)) { p += 2; continue; } /* skip escape */
            if (*p == '"')  { in_str = false; }
            p++; continue;
        }
        if (*p == '`') { in_raw = true; p++; continue; }
        if (*p == '"') { in_str = true; p++; continue; }
        p++;
    }
    return in_raw;
}

int gray_fmt_source(const char *src, const char *filename, FILE *out) {
    int max_line = count_lines(src);
    int *depth_table = build_depth_table(src, filename, max_line);
    if (!depth_table) return 1;

    /* Walk the source line by line, re-indenting each one. */
    const char *p = src;
    int line_num = 1;

    while (*p) {
        /* Find end of this line */
        const char *line_start = p;
        while (*p && *p != '\n') p++;
        int line_len = (int)(p - line_start);

        /* Advance past newline for next iteration */
        if (*p == '\n') p++;

        /* Skip leading whitespace to get the trimmed content */
        const char *content = line_start;
        while (content < line_start + line_len && (*content == ' ' || *content == '\t'))
            content++;
        int content_len = (int)((line_start + line_len) - content);

        if (content_len == 0) {
            /* Blank line: emit as-is (no indentation) */
            fputc('\n', out);
        } else if (line_is_in_raw_string(src, line_num)) {
            /* Inside a raw string: preserve the original line verbatim */
            fwrite(line_start, 1, line_len, out);
            fputc('\n', out);
        } else {
            /* Emit corrected indentation + original content */
            int depth = (line_num <= max_line) ? depth_table[line_num] : 0;
            if (depth < 0) depth = 0;
            for (int i = 0; i < depth * FMT_INDENT_WIDTH; i++) fputc(' ', out);
            fwrite(content, 1, content_len, out);
            fputc('\n', out);
        }

        line_num++;
    }

    /* Ensure file ends with exactly one newline (already handled above if
     * the source ended with \n; if it didn't, the last fputc above adds one). */

    free(depth_table);
    return 0;
}
