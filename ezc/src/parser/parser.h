/*
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_PARSER_H
#define EZC_PARSER_H

#include "ast.h"
#include "../lexer/lexer.h"
#include "../util/arena.h"
#include "../util/error.h"

#define GRAY_MAX_PARSE_DEPTH 256

typedef struct {
    Lexer *lexer;
    Arena *arena;
    Token cur_token;
    Token peek_token;
    const char *file;
    DiagnosticList *diag;
    int depth;
    bool no_struct_literal; /* suppress struct literal parsing (RHS of in/not_in) */
    bool in_interp;         /* true when parsing a ${...} sub-expression */

    /* Known struct names collected by a pre-scan so the parser can
     * distinguish struct literals (Point{x:1}) from identifiers
     * followed by blocks without relying on capitalization heuristics. */
    const char **struct_names;
    int struct_name_count;
    int struct_name_cap;
} Parser;

Parser *parser_create(Arena *arena, Lexer *lexer, const char *file, DiagnosticList *diag);
AstNode *parser_parse_program(Parser *p);

#endif
