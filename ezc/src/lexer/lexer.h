/*
 * lexer.h - Lexical analysis for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_LEXER_H
#define EZC_LEXER_H

#include "token.h"
#include "../util/arena.h"

typedef struct {
    const char *input;
    int input_len;
    int position;
    int read_position;
    char ch;
    int line;
    int column;
    const char *file;
    Arena *arena;   /* For allocating token literals */
} Lexer;

Lexer *lexer_create(Arena *arena, const char *input, const char *file);
Token lexer_next_token(Lexer *l);

#endif
