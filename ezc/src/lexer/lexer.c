/*
 * lexer.c - Lexical analysis for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "lexer.h"
#include <string.h>
#include <stdbool.h>
#include <stdint.h>
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
    /* EOF reached — unclosed comment */
    l->error_code = "E1003";
    l->error_msg = "unclosed multi-line comment";
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

    /* Check for 0x, 0o, 0b prefixes */
    if (l->ch == '0') {
        char next = peek_char(l);
        if (next == 'x' || next == 'X') {
            read_char(l); read_char(l);
            int dstart = l->position;
            while (isxdigit(l->ch) || l->ch == '_') read_char(l);
            if (l->position == dstart) {
                l->error_code = "E1010";
                l->error_msg = "invalid number format: '0x' must be followed by hex digits (0-9, a-f)";
            }
            return arena_strndup(l->arena, l->input + start, l->position - start);
        }
        if (next == 'o' || next == 'O') {
            read_char(l); read_char(l);
            int dstart = l->position;
            while ((l->ch >= '0' && l->ch <= '7') || l->ch == '_') read_char(l);
            if (l->position == dstart) {
                l->error_code = "E1010";
                l->error_msg = "invalid number format: '0o' must be followed by octal digits (0-7)";
            }
            return arena_strndup(l->arena, l->input + start, l->position - start);
        }
        if (next == 'b' || next == 'B') {
            read_char(l); read_char(l);
            int dstart = l->position;
            while (l->ch == '0' || l->ch == '1' || l->ch == '_') read_char(l);
            if (l->position == dstart) {
                l->error_code = "E1010";
                l->error_msg = "invalid number format: '0b' must be followed by binary digits (0-1)";
            }
            return arena_strndup(l->arena, l->input + start, l->position - start);
        }
    }

    while (isdigit(l->ch) || l->ch == '_') {
        read_char(l);
    }

    if (l->ch == '.') {
        char next = peek_char(l);
        if (isdigit(next) || next == '_' || next == 0 || next == '\n' ||
            next == ' ' || next == ')' || next == '}' || next == ',' ||
            next == ';') {
            /* Consume decimal point — validation below will catch errors */
            *type = TOK_FLOAT;
            read_char(l);
            while (isdigit(l->ch) || l->ch == '_') {
                read_char(l);
            }
        }
    }

    const char *num = arena_strndup(l->arena, l->input + start, l->position - start);

    /* Validate number literal format */
    if (num && !l->error_code) {
        const char *s = num;
        /* Skip 0x/0o/0b prefix */
        if (s[0] == '0' && (s[1] == 'x' || s[1] == 'X' || s[1] == 'o' || s[1] == 'O' || s[1] == 'b' || s[1] == 'B'))
            s += 2;
        int len = (int)strlen(s);
        if (len > 0) {
            /* E1013: trailing underscore */
            if (s[len-1] == '_') { l->error_code = "E1013"; l->error_msg = "number cannot end with underscore"; }
            /* E1011: consecutive underscores */
            for (int i = 0; s[i] && s[i+1]; i++) {
                if (s[i] == '_' && s[i+1] == '_') { l->error_code = "E1011"; l->error_msg = "number cannot have consecutive underscores"; break; }
            }
            /* E1014/E1015: underscore adjacent to decimal point */
            for (int i = 0; s[i]; i++) {
                if (s[i] == '.') {
                    if (i > 0 && s[i-1] == '_') { l->error_code = "E1014"; l->error_msg = "underscore cannot appear before decimal point"; }
                    if (s[i+1] == '_') { l->error_code = "E1015"; l->error_msg = "underscore cannot appear after decimal point"; }
                    /* E1016: trailing decimal */
                    if (!s[i+1] || (!isdigit(s[i+1]) && s[i+1] != '_')) { l->error_code = "E1016"; l->error_msg = "number cannot end with decimal point"; }
                    break;
                }
            }
        }
    }

    return num;
}

static const char *read_string(Lexer *l) {
    read_char(l); /* skip opening " */
    int start = l->position;
    int brace_depth = 0;

    while (l->ch != 0) {
        if (l->ch == '\\') {
            read_char(l); /* move to escape char */
            /* Validate escape sequence */
            if (l->ch == 'x') {
                /* Hex escape: \xNN — exactly two hex digits */
                read_char(l);
                if (!isxdigit(l->ch)) {
                    l->error_code = "E1006";
                    l->error_msg = "invalid hex escape sequence — \\x must be followed by exactly two hex digits";
                    continue;
                }
                read_char(l);
                if (!isxdigit(l->ch)) {
                    l->error_code = "E1006";
                    l->error_msg = "invalid hex escape sequence — \\x must be followed by exactly two hex digits";
                    continue;
                }
                read_char(l);
                continue;
            } else if (l->ch != 'n' && l->ch != 't' && l->ch != 'r' && l->ch != '\\' &&
                l->ch != '"' && l->ch != '\'' && l->ch != '0' &&
                l->ch != 'a' && l->ch != 'b' && l->ch != 'f' && l->ch != 'v' &&
                l->ch != '$' && l->ch != 0) {
                l->error_code = "E1006";
                l->error_msg = "invalid escape sequence in string";
            }
            read_char(l);
            continue;
        }
        if (l->ch == '$' && peek_char(l) == '{') {
            brace_depth++;
            read_char(l); /* skip $ */
            read_char(l); /* skip { */
            continue;
        }
        if (l->ch == '{' && brace_depth > 0) {
            brace_depth++;
            read_char(l);
            continue;
        }
        if (l->ch == '}' && brace_depth > 0) {
            brace_depth--;
            read_char(l);
            continue;
        }
        if (l->ch == '"' && brace_depth == 0) {
            break; /* end of string */
        }
        read_char(l);
    }

    const char *str = arena_strndup(l->arena, l->input + start, l->position - start);
    if (l->ch == '"') {
        read_char(l); /* skip closing " */
    } else {
        l->unterminated_string = true;
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
    } else {
        l->error_code = "E1017";
        l->error_msg = "unclosed raw string literal";
    }
    return str;
}

static const char *read_char_literal(Lexer *l) {
    read_char(l); /* skip opening ' */
    int start = l->position;

    if (l->ch == '\\') {
        read_char(l); /* move to escape char */
        /* E1007: validate escape in char literal */
        if (l->ch != 'n' && l->ch != 't' && l->ch != 'r' && l->ch != '\\' &&
            l->ch != '\'' && l->ch != '0' && l->ch != 'x' && l->ch != 0) {
            l->error_code = "E1007";
            l->error_msg = "invalid escape sequence in character literal";
        }
        read_char(l); /* skip escaped char */
    } else {
        read_char(l); /* skip the char */
    }

    /* Check for multi-character char literal */
    if (l->ch != '\'' && l->ch != 0 && !l->error_code) {
        /* Consume remaining characters until closing quote or end */
        while (l->ch != '\'' && l->ch != 0 && l->ch != '\n') {
            read_char(l);
        }
        l->error_code = "E1018";
        l->error_msg = "char literal must contain exactly one character — use a string for multiple characters";
    }

    const char *str = arena_strndup(l->arena, l->input + start, l->position - start);
    if (l->ch == '\'') {
        read_char(l); /* skip closing ' */
    } else if (!l->error_code) {
        l->error_code = "E1005";
        l->error_msg = "unclosed character literal";
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
    size_t raw_len = strlen(input);
    if (raw_len > (size_t)INT32_MAX) raw_len = (size_t)INT32_MAX;
    l->input_len = (int)raw_len;
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
    l->error_code = NULL;
    skip_whitespace_and_comments(l);

    /* Check for lexer errors from comment/whitespace skipping */
    if (l->error_code) {
        tok = make_token(TOK_ILLEGAL, l->error_msg, l->line, l->column);
        return tok;
    }

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
        } else if (peek_char(l) == 'i' && l->read_position + 1 < l->input_len &&
                   l->input[l->read_position + 1] == 'n' &&
                   (l->read_position + 2 >= l->input_len ||
                    !isalpha(l->input[l->read_position + 2]))) {
            /* !in → NOT_IN */
            read_char(l); /* skip i */
            read_char(l); /* skip n */
            tok = make_token(TOK_NOT_IN, "!in", tok.line, tok.column);
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
            l->error_code = "E1020";
            l->error_msg = "unexpected character '|' — use '||' for logical OR";
            tok = make_token(TOK_ILLEGAL, l->error_msg, tok.line, tok.column);
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
            l->error_code = "E1019";
            l->error_msg = "unexpected character '#' — use '//' for comments, or '#strict', '#flags', '#enum', '#doc' for attributes";
            tok = make_token(TOK_ILLEGAL, l->error_msg, tok.line, tok.column);
        }
        break;

    case '"':
        l->unterminated_string = false;
        tok.literal = read_string(l);
        if (l->unterminated_string) {
            l->error_code = "E1021";
            l->error_msg = "string literal was never closed — add a closing double quote";
            tok.type = TOK_ILLEGAL;
            tok.literal = l->error_msg;
        } else if (l->error_code) {
            tok.type = TOK_ILLEGAL;
            tok.literal = l->error_msg;
        } else {
            tok.type = TOK_STRING;
        }
        return tok;

    case '`':
        tok.literal = read_raw_string(l);
        tok.type = l->error_code ? TOK_ILLEGAL : TOK_RAW_STRING;
        if (l->error_code) tok.literal = l->error_msg;
        return tok;

    case '\'':
        l->error_code = NULL;
        tok.literal = read_char_literal(l);
        tok.type = l->error_code ? TOK_ILLEGAL : TOK_CHAR;
        if (l->error_code) tok.literal = l->error_msg;
        return tok;

    default:
        if (isalpha(l->ch) || l->ch == '_') {
            tok.literal = read_identifier(l);
            tok.type = token_lookup_ident(tok.literal);
            return tok;
        } else if (isdigit(l->ch)) {
            TokenType num_type;
            tok.literal = read_number(l, &num_type);
            if (l->error_code) {
                tok.type = TOK_ILLEGAL;
                tok.literal = l->error_msg;
            } else {
                tok.type = num_type;
            }
            return tok;
        } else {
            char msg[64];
            snprintf(msg, sizeof(msg), "unexpected character '%c'", l->ch);
            l->error_code = "E1022";
            l->error_msg = arena_strdup(l->arena, msg);
            tok = make_token(TOK_ILLEGAL, l->error_msg, tok.line, tok.column);
        }
        break;
    }

    read_char(l);
    return tok;
}
