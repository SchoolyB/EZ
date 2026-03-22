/*
 * lexer.c - Lexical analysis for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "lexer.h"
#include <string.h>
#include <stdbool.h>
#include <ctype.h>
#include <stdio.h>

static void read_char(Lexer *l) {
    if (l->read_position >= l->input_len) {
        l->ch = 0;
    } else {
        l->ch = l->input[l->read_position];
    }
    l->position = l->read_position;
    l->read_position++;
    l->column++;
    if (l->ch == '\n') {
        l->line++;
        l->column = 0;
    }
}

static char peek_char(Lexer *l) {
    if (l->read_position >= l->input_len) return 0;
    return l->input[l->read_position];
}

static void skip_whitespace(Lexer *l) {
    while (l->ch == ' ' || l->ch == '\t' || l->ch == '\r' || l->ch == '\n') {
        read_char(l);
    }
}

static void skip_line_comment(Lexer *l) {
    /* Skip until end of line */
    while (l->ch != '\n' && l->ch != 0) {
        read_char(l);
    }
}

static void skip_block_comment(Lexer *l) {
    /* Skip past opening slash-star */
    read_char(l); /* skip * */
    while (l->ch != 0) {
        if (l->ch == '*' && peek_char(l) == '/') {
            read_char(l); /* skip * */
            read_char(l); /* skip / */
            return;
        }
        read_char(l);
    }
}

static void skip_whitespace_and_comments(Lexer *l) {
    for (;;) {
        skip_whitespace(l);
        if (l->ch == '/' && peek_char(l) == '/') {
            read_char(l); /* skip first / */
            read_char(l); /* skip second / */
            skip_line_comment(l);
        } else if (l->ch == '/' && peek_char(l) == '*') {
            read_char(l); /* skip / */
            skip_block_comment(l);
        } else {
            break;
        }
    }
}

static const char *read_identifier(Lexer *l) {
    int start = l->position;
    while (isalpha(l->ch) || l->ch == '_' || isdigit(l->ch)) {
        read_char(l);
    }
    return arena_strndup(l->arena, l->input + start, l->position - start);
}

static const char *read_number(Lexer *l, TokenType *type) {
    int start = l->position;
    *type = TOK_INT;

    while (isdigit(l->ch) || l->ch == '_') {
        read_char(l);
    }

    if (l->ch == '.' && isdigit(peek_char(l))) {
        *type = TOK_FLOAT;
        read_char(l); /* skip . */
        while (isdigit(l->ch) || l->ch == '_') {
            read_char(l);
        }
    }

    return arena_strndup(l->arena, l->input + start, l->position - start);
}

static const char *read_string(Lexer *l) {
    read_char(l); /* skip opening " */
    int start = l->position;

    while (l->ch != '"' && l->ch != 0) {
        if (l->ch == '\\') {
            read_char(l); /* skip escape char */
        }
        read_char(l);
    }

    const char *str = arena_strndup(l->arena, l->input + start, l->position - start);
    if (l->ch == '"') {
        read_char(l); /* skip closing " */
    }
    return str;
}

static const char *read_raw_string(Lexer *l) {
    read_char(l); /* skip opening ` */
    int start = l->position;

    while (l->ch != '`' && l->ch != 0) {
        read_char(l);
    }

    const char *str = arena_strndup(l->arena, l->input + start, l->position - start);
    if (l->ch == '`') {
        read_char(l); /* skip closing ` */
    }
    return str;
}

static const char *read_char_literal(Lexer *l) {
    read_char(l); /* skip opening ' */
    int start = l->position;

    if (l->ch == '\\') {
        read_char(l); /* skip backslash */
        read_char(l); /* skip escaped char */
    } else {
        read_char(l); /* skip the char */
    }

    const char *str = arena_strndup(l->arena, l->input + start, l->position - start);
    if (l->ch == '\'') {
        read_char(l); /* skip closing ' */
    }
    return str;
}

static Token make_token(TokenType type, const char *literal, int line, int col) {
    Token t;
    t.type = type;
    t.literal = literal;
    t.line = line;
    t.column = col;
    return t;
}

static int match_ahead(Lexer *l, const char *s, int len) {
    if (l->position + len > l->input_len) return 0;
    return strncmp(l->input + l->position, s, len) == 0;
}

Lexer *lexer_create(Arena *arena, const char *input, const char *file) {
    Lexer *l = arena_alloc(arena, sizeof(Lexer));
    l->input = input;
    l->input_len = (int)strlen(input);
    l->position = 0;
    l->read_position = 0;
    l->ch = 0;
    l->line = 1;
    l->column = 0;
    l->file = file;
    l->arena = arena;
    read_char(l);
    return l;
}

Token lexer_next_token(Lexer *l) {
    Token tok;
    skip_whitespace_and_comments(l);

    tok.line = l->line;
    tok.column = l->column;

    switch (l->ch) {
    case 0:
        tok = make_token(TOK_EOF, "", l->line, l->column);
        return tok;

    case '=':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_EQ, "==", tok.line, tok.column);
        } else {
            tok = make_token(TOK_ASSIGN, "=", tok.line, tok.column);
        }
        break;

    case '+':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_PLUS_ASSIGN, "+=", tok.line, tok.column);
        } else if (peek_char(l) == '+') {
            read_char(l);
            tok = make_token(TOK_INCREMENT, "++", tok.line, tok.column);
        } else {
            tok = make_token(TOK_PLUS, "+", tok.line, tok.column);
        }
        break;

    case '-':
        if (peek_char(l) == '>') {
            read_char(l);
            tok = make_token(TOK_ARROW, "->", tok.line, tok.column);
        } else if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_MINUS_ASSIGN, "-=", tok.line, tok.column);
        } else if (peek_char(l) == '-') {
            read_char(l);
            tok = make_token(TOK_DECREMENT, "--", tok.line, tok.column);
        } else {
            tok = make_token(TOK_MINUS, "-", tok.line, tok.column);
        }
        break;

    case '!':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_NOT_EQ, "!=", tok.line, tok.column);
        } else {
            tok = make_token(TOK_BANG, "!", tok.line, tok.column);
        }
        break;

    case '*':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_ASTERISK_ASSIGN, "*=", tok.line, tok.column);
        } else if (peek_char(l) == '*') {
            read_char(l);
            tok = make_token(TOK_POWER, "**", tok.line, tok.column);
        } else {
            tok = make_token(TOK_ASTERISK, "*", tok.line, tok.column);
        }
        break;

    case '/':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_SLASH_ASSIGN, "/=", tok.line, tok.column);
        } else {
            tok = make_token(TOK_SLASH, "/", tok.line, tok.column);
        }
        break;

    case '%':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_PERCENT_ASSIGN, "%=", tok.line, tok.column);
        } else {
            tok = make_token(TOK_PERCENT, "%", tok.line, tok.column);
        }
        break;

    case '<':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_LT_EQ, "<=", tok.line, tok.column);
        } else {
            tok = make_token(TOK_LT, "<", tok.line, tok.column);
        }
        break;

    case '>':
        if (peek_char(l) == '=') {
            read_char(l);
            tok = make_token(TOK_GT_EQ, ">=", tok.line, tok.column);
        } else {
            tok = make_token(TOK_GT, ">", tok.line, tok.column);
        }
        break;

    case '&':
        if (peek_char(l) == '&') {
            read_char(l);
            tok = make_token(TOK_AND, "&&", tok.line, tok.column);
        } else {
            tok = make_token(TOK_AMPERSAND, "&", tok.line, tok.column);
        }
        break;

    case '|':
        if (peek_char(l) == '|') {
            read_char(l);
            tok = make_token(TOK_OR, "||", tok.line, tok.column);
        } else {
            tok = make_token(TOK_ILLEGAL, "|", tok.line, tok.column);
        }
        break;

    case ',': tok = make_token(TOK_COMMA, ",", tok.line, tok.column); break;
    case ':': tok = make_token(TOK_COLON, ":", tok.line, tok.column); break;
    case ';': tok = make_token(TOK_SEMICOLON, ";", tok.line, tok.column); break;
    case '(': tok = make_token(TOK_LPAREN, "(", tok.line, tok.column); break;
    case ')': tok = make_token(TOK_RPAREN, ")", tok.line, tok.column); break;
    case '{': tok = make_token(TOK_LBRACE, "{", tok.line, tok.column); break;
    case '}': tok = make_token(TOK_RBRACE, "}", tok.line, tok.column); break;
    case '[': tok = make_token(TOK_LBRACKET, "[", tok.line, tok.column); break;
    case ']': tok = make_token(TOK_RBRACKET, "]", tok.line, tok.column); break;
    case '.': tok = make_token(TOK_DOT, ".", tok.line, tok.column); break;
    case '@': tok = make_token(TOK_AT, "@", tok.line, tok.column); break;
    case '^': tok = make_token(TOK_CARET, "^", tok.line, tok.column); break;

    case '#':
        if (match_ahead(l, "#suppress", 9)) {
            tok = make_token(TOK_SUPPRESS, "#suppress", tok.line, tok.column);
            for (int i = 0; i < 8; i++) read_char(l);
        } else if (match_ahead(l, "#strict", 7)) {
            tok = make_token(TOK_STRICT, "#strict", tok.line, tok.column);
            for (int i = 0; i < 6; i++) read_char(l);
        } else if (match_ahead(l, "#flags", 6)) {
            tok = make_token(TOK_FLAGS, "#flags", tok.line, tok.column);
            for (int i = 0; i < 5; i++) read_char(l);
        } else if (match_ahead(l, "#enum", 5)) {
            tok = make_token(TOK_ENUM_ATTR, "#enum", tok.line, tok.column);
            for (int i = 0; i < 4; i++) read_char(l);
        } else if (match_ahead(l, "#doc", 4)) {
            tok = make_token(TOK_DOC, "#doc", tok.line, tok.column);
            for (int i = 0; i < 3; i++) read_char(l);
        } else {
            tok = make_token(TOK_ILLEGAL, "#", tok.line, tok.column);
        }
        break;

    case '"':
        tok.literal = read_string(l);
        tok.type = TOK_STRING;
        return tok;

    case '`':
        tok.literal = read_raw_string(l);
        tok.type = TOK_RAW_STRING;
        return tok;

    case '\'':
        tok.literal = read_char_literal(l);
        tok.type = TOK_CHAR;
        return tok;

    default:
        if (isalpha(l->ch) || l->ch == '_') {
            tok.literal = read_identifier(l);
            tok.type = token_lookup_ident(tok.literal);
            return tok;
        } else if (isdigit(l->ch)) {
            TokenType num_type;
            tok.literal = read_number(l, &num_type);
            tok.type = num_type;
            return tok;
        } else {
            char buf[2] = {l->ch, 0};
            tok = make_token(TOK_ILLEGAL, arena_strdup(l->arena, buf), tok.line, tok.column);
        }
        break;
    }

    read_char(l);
    return tok;
}
