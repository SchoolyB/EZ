/*
 * parser.h - Recursive descent parser for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_PARSER_H
#define EZC_PARSER_H

#include "ast.h"
#include "../lexer/lexer.h"
#include "../util/arena.h"

typedef struct {
    Lexer *lexer;
    Arena *arena;
    Token cur_token;
    Token peek_token;
    const char *file;

    /* Error tracking */
    char **errors;
    int error_count;
    int error_cap;
} Parser;

Parser *parser_create(Arena *arena, Lexer *lexer, const char *file);
AstNode *parser_parse_program(Parser *p);
bool parser_has_errors(Parser *p);
void parser_print_errors(Parser *p);

#endif
