/*
 * ezc_wasm.c - WASM entry point for the EZ compiler
 *
 * Compiles EZ source to C code entirely in-memory.
 * No file I/O, no system() calls, no POSIX dependencies.
 *
 * Exported functions (called from JavaScript):
 *   ezc_compile(source) -> C code or error string
 *   ezc_check(source)   -> error string or "ok"
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include <stdio.h>
#include <string.h>

#include "../ezc/src/util/arena.h"
#include "../ezc/src/util/error.h"
#include "../ezc/src/lexer/lexer.h"
#include "../ezc/src/parser/parser.h"
#include "../ezc/src/typechecker/typechecker.h"
#include "../ezc/src/codegen/codegen.h"

#ifdef __EMSCRIPTEN__
#include <emscripten.h>
#define EXPORT EMSCRIPTEN_KEEPALIVE
#else
#define EXPORT
#endif

/* Static buffers for returning results to JS */
static char result_buf[1024 * 1024]; /* 1MB output buffer */
static char diag_buf[64 * 1024];
static int diag_pos = 0;

static void diag_to_string(DiagnosticList *dl) {
    diag_pos = 0;
    diag_buf[0] = '\0';

    for (int i = 0; i < dl->count; i++) {
        Diagnostic *d = &dl->items[i];
        const char *sev = d->severity == SEV_ERROR ? "error" :
                          d->severity == SEV_WARNING ? "warning" : "note";
        int n = snprintf(diag_buf + diag_pos, sizeof(diag_buf) - diag_pos,
            "%s[%s]: %s (line %d)\n",
            sev, d->code ? d->code : "", d->message ? d->message : "", d->line);
        if (n > 0 && diag_pos + n < (int)sizeof(diag_buf)) diag_pos += n;
    }
}

EXPORT
const char *ezc_compile(const char *source) {
    if (!source || !*source) return "error: empty source";

    Arena *arena = arena_create(256 * 1024);
    DiagnosticList *diag = diag_create();
    diag_set_source(diag, "<playground>", source);

    /* Parse */
    Lexer *lex = lexer_create(arena, source, "<playground>");
    Parser *parser = parser_create(arena, lex, "<playground>", diag);
    AstNode *program = parser_parse_program(parser);

    if (diag_has_errors(diag)) {
        diag_to_string(diag);
        strncpy(result_buf, diag_buf, sizeof(result_buf) - 1);
        diag_destroy(diag);
        arena_destroy(arena);
        return result_buf;
    }

    /* Type check */
    TypeChecker *tc = typechecker_create(diag, "<playground>");
    typechecker_check(tc, program);

    if (diag_has_errors(diag)) {
        diag_to_string(diag);
        strncpy(result_buf, diag_buf, sizeof(result_buf) - 1);
        diag_destroy(diag);
        arena_destroy(arena);
        return result_buf;
    }

    /* Codegen */
    TypeTable *type_table = typechecker_get_table(tc);
    CodeGen cg = codegen_create("<playground>");
    cg.type_table = type_table;
    codegen_generate(&cg, program);

    const char *c_code = codegen_result(&cg);
    if (c_code) {
        strncpy(result_buf, c_code, sizeof(result_buf) - 1);
        result_buf[sizeof(result_buf) - 1] = '\0';
    } else {
        strcpy(result_buf, "error: codegen produced no output");
    }

    codegen_destroy(&cg);
    diag_destroy(diag);
    arena_destroy(arena);
    return result_buf;
}

EXPORT
const char *ezc_check(const char *source) {
    if (!source || !*source) return "ok";

    Arena *arena = arena_create(256 * 1024);
    DiagnosticList *diag = diag_create();
    diag_set_source(diag, "<playground>", source);

    Lexer *lex = lexer_create(arena, source, "<playground>");
    Parser *parser = parser_create(arena, lex, "<playground>", diag);
    AstNode *program = parser_parse_program(parser);

    if (!diag_has_errors(diag)) {
        TypeChecker *tc = typechecker_create(diag, "<playground>");
        typechecker_check(tc, program);
    }

    if (diag_has_errors(diag) || diag_warning_count(diag) > 0) {
        diag_to_string(diag);
        strncpy(result_buf, diag_buf, sizeof(result_buf) - 1);
        diag_destroy(diag);
        arena_destroy(arena);
        return result_buf;
    }

    diag_destroy(diag);
    arena_destroy(arena);
    return "ok";
}

#ifndef __EMSCRIPTEN__
int main(int argc, char **argv) {
    if (argc < 2) {
        fprintf(stderr, "usage: ezc_wasm <source-string>\n");
        return 1;
    }
    const char *result = ezc_compile(argv[1]);
    printf("%s\n", result);
    return 0;
}
#endif
