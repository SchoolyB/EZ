/*
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "parser.h"
#include "../util/constants.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <ctype.h>

#define MAX_MULTI_VARS 16
#define MAX_SHARED_RETURNS 16
#define EZ_FLOAT_LIT_BUF    128
#define EZ_TMP_NAME_BUF     32
#define EZ_FIELD_NAME_BUF   8

/* Operator precedence levels */
typedef enum {
    PREC_LOWEST,
    PREC_OR,            /* || */
    PREC_AND,           /* && */
    PREC_EQUALS,        /* == != */
    PREC_BITWISE,       /* bit_and, bit_or, bit_xor — above == so a bit_and b == c → (a bit_and b) == c */
    PREC_LESSGREATER,   /* > < >= <= */
    PREC_MEMBERSHIP,    /* in, not_in */
    PREC_SHIFT,         /* bit_shift_left, bit_shift_right */
    PREC_SUM,           /* + - */
    PREC_PRODUCT,       /* * / % */
    PREC_PREFIX,        /* -x !x bit_not x */
    PREC_CALL,          /* f(x) */
    PREC_INDEX,         /* a[i] a.b */
    PREC_POSTFIX,       /* x++ x-- */
} Precedence;

/* Forward declarations */
static AstNode *parse_statement(Parser *p);
static AstNode *parse_expression(Parser *p, Precedence prec);
static AstNode *parse_block_statement(Parser *p);
static AstNode *parse_struct_literal(Parser *p, const char *name);
static AstNode *maybe_apply_or_return(Parser *p, AstNode *var_decl);

/* --- Struct name pre-scan --- */

/* Lightweight lexer scan to collect struct (and enum) names before the
 * main parse.  This lets the parser disambiguate `Name{` (struct literal)
 * from an identifier followed by a block without relying on uppercase
 * heuristics.  The scan looks for `const <IDENT> struct` and
 * `const <IDENT> enum` token triples. */
static void prescan_struct_names(Parser *p) {
    /* Save lexer state */
    int saved_pos  = p->lexer->position;
    int saved_rpos = p->lexer->read_position;
    char saved_ch  = p->lexer->ch;
    int saved_line = p->lexer->line;
    int saved_col  = p->lexer->column;

    /* Reset lexer to start of input */
    p->lexer->position = 0;
    p->lexer->read_position = 0;
    p->lexer->line = 1;
    p->lexer->column = 0;
    p->lexer->ch = p->lexer->input[0];
    if (p->lexer->input[0])
        p->lexer->read_position = 1;

    Token prev2 = {TOK_EOF, "", 0, 0, NULL, false};
    Token prev1 = {TOK_EOF, "", 0, 0, NULL, false};
    for (;;) {
        Token tok = lexer_next_token(p->lexer);
        if (tok.type == TOK_EOF) break;
        /* Match: const <IDENT> struct  OR  const <IDENT> enum */
        if (prev2.type == TOK_CONST && prev1.type == TOK_IDENT &&
            (tok.type == TOK_STRUCT || tok.type == TOK_ENUM)) {
            /* Add prev1.literal to struct_names */
            if (p->struct_name_count >= p->struct_name_cap) {
                int new_cap = p->struct_name_cap * 2;
                const char **new_names = arena_alloc(p->arena,
                    sizeof(const char *) * new_cap);
                memcpy(new_names, p->struct_names,
                    sizeof(const char *) * p->struct_name_count);
                p->struct_names = new_names;
                p->struct_name_cap = new_cap;
            }
            p->struct_names[p->struct_name_count++] =
                arena_strdup(p->arena, prev1.literal);
        }
        prev2 = prev1;
        prev1 = tok;
    }

    /* Restore lexer state */
    p->lexer->position = saved_pos;
    p->lexer->read_position = saved_rpos;
    p->lexer->ch = saved_ch;
    p->lexer->line = saved_line;
    p->lexer->column = saved_col;
}

static bool parser_is_struct_name(Parser *p, const char *name) {
    for (int i = 0; i < p->struct_name_count; i++) {
        if (strcmp(p->struct_names[i], name) == 0) return true;
    }
    return false;
}

/* --- Helpers --- */

static void next_token(Parser *p) {
    p->cur_token = p->peek_token;
    p->peek_token = lexer_next_token(p->lexer);
    /* Surface lexer errors (E1xxx). The lexer does not call diag_error()
     * directly — it sets error_code/error_msg on itself and returns
     * TOK_ILLEGAL. We detect that here and emit the diagnostic so the
     * lexer stays free of diagnostic dependencies. After emitting, clear
     * error_code so the same error is not reported twice. */
    if (p->peek_token.type == TOK_ILLEGAL && p->lexer->error_code) {
        diag_error_msg(p->diag, p->lexer->error_code,
            arena_strdup(p->arena, p->lexer->error_msg),
            p->file, p->peek_token.line, p->peek_token.column, 0);
        p->lexer->error_code = NULL;
    }
}

static bool cur_token_is(Parser *p, TokenType t) {
    return p->cur_token.type == t;
}

static bool peek_token_is(Parser *p, TokenType t) {
    return p->peek_token.type == t;
}

static bool expect_peek(Parser *p, TokenType t) {
    if (peek_token_is(p, t)) {
        next_token(p);
        return true;
    }
    char buf[EZ_MSG_BUF_SIZE];
    snprintf(buf, sizeof(buf), "expected '%s', got '%s'",
        token_type_name(t), token_type_name(p->peek_token.type));
    /* Point at current token (where the expected token should be), not the peek token */
    diag_error_msg(p->diag, "E2001", arena_strdup(p->arena, buf),
        p->file, p->cur_token.line, p->cur_token.column, 0);
    return false;
}

/* Check if a token type is a keyword (reserved word) */
static bool is_keyword_token(TokenType t) {
    switch (t) {
    case TOK_MUT: case TOK_CONST: case TOK_DO: case TOK_RETURN:
    case TOK_IF: case TOK_OR_KW: case TOK_OTHERWISE:
    case TOK_FOR: case TOK_FOR_EACH: case TOK_AS_LONG_AS:
    case TOK_LOOP: case TOK_BREAK: case TOK_CONTINUE:
    case TOK_IN: case TOK_NOT_IN: case TOK_RANGE:
    case TOK_IMPORT: case TOK_USING: case TOK_STRUCT: case TOK_ENUM:
    case TOK_NIL: case TOK_NEW: case TOK_TRUE: case TOK_FALSE:
    case TOK_ENSURE: case TOK_OR_RETURN: case TOK_WHEN:
    case TOK_MODULE: case TOK_PRIVATE:
        return true;
    default:
        return false;
    }
}

/* Synchronize parser after an error; skip to a safe point.
 * Advances past the current line and stops at the next statement boundary. */
static void synchronize(Parser *p) {
    int error_line = p->cur_token.line;
    /* First, skip past the current line to avoid re-parsing the same error */
    while (!cur_token_is(p, TOK_EOF) && p->cur_token.line == error_line) {
        next_token(p);
    }
    /* Then find the next statement-starting token */
    while (!cur_token_is(p, TOK_EOF)) {
        switch (p->cur_token.type) {
        case TOK_DO: case TOK_MUT: case TOK_CONST:
        case TOK_RETURN: case TOK_IF: case TOK_FOR:
        case TOK_FOR_EACH: case TOK_AS_LONG_AS: case TOK_LOOP:
        case TOK_WHEN: case TOK_IMPORT: case TOK_USING:
        case TOK_BREAK: case TOK_CONTINUE:
        case TOK_RBRACE:
        case TOK_IDENT:
            return;
        default:
            next_token(p);
        }
    }
}

static Precedence token_precedence(TokenType t) {
    switch (t) {
    case TOK_OR:              return PREC_OR;
    case TOK_AND:             return PREC_AND;
    case TOK_EQ:
    case TOK_NOT_EQ:          return PREC_EQUALS;
    case TOK_BIT_AND:
    case TOK_BIT_OR:
    case TOK_BIT_XOR:         return PREC_BITWISE;
    case TOK_LT:
    case TOK_GT:
    case TOK_LT_EQ:
    case TOK_GT_EQ:           return PREC_LESSGREATER;
    case TOK_IN:
    case TOK_NOT_IN:          return PREC_MEMBERSHIP;
    case TOK_BIT_SHIFT_LEFT:
    case TOK_BIT_SHIFT_RIGHT: return PREC_SHIFT;
    case TOK_PLUS:
    case TOK_MINUS:           return PREC_SUM;
    case TOK_ASTERISK:
    case TOK_SLASH:
    case TOK_PERCENT:         return PREC_PRODUCT;
    case TOK_LPAREN:          return PREC_CALL;
    case TOK_LBRACKET:        return PREC_INDEX;
    case TOK_DOT:             return PREC_INDEX;
    case TOK_INCREMENT:
    case TOK_DECREMENT:
    case TOK_CARET:           return PREC_POSTFIX;
    default:                  return PREC_LOWEST;
    }
}

/* --- Expression Parsing --- */

/* Read a type name: simple (int, Person) or qualified (models.Task).
 * Assumes current token is the first identifier. Returns arena-allocated string. */
/* Wildcard type placeholder for generics. Stored as the
 * string "?" in the same slot as any other type name so the rest of
 * the compiler can carry it through unchanged until typechecker
 * instantiation (slice 2) replaces it with a concrete type. */
static bool type_string_has_wildcard(const char *tn) {
    if (!tn) return false;
    for (const char *c = tn; *c; c++) {
        if (*c == '?') return true;
    }
    return false;
}

static const char *read_type_name(Parser *p) {
    /* Wildcard type placeholder: `?` in a type position */
    if (cur_token_is(p, TOK_QUESTION)) {
        return "?";
    }
    const char *name = p->cur_token.literal;
    if (peek_token_is(p, TOK_DOT)) {
        next_token(p); /* skip . */
        next_token(p); /* qualified part */
        size_t nlen = strlen(name), qlen = strlen(p->cur_token.literal);
        size_t len = nlen + qlen + 2;
        char *qualified = arena_alloc(p->arena, len);
        /* Use underscore for module-qualified types: mod.Type → mod_Type */
        snprintf(qualified, len, "%s_%s", name, p->cur_token.literal);
        return qualified;
    }
    return name;
}

/* Parse a complex type annotation.
 * Precondition: parser is ON the first token of the type ([, ^, map, or IDENT).
 * Postcondition: returns the type string, parser on the last token of the type.
 * Returns NULL on parse error (diagnostic already emitted). */
static const char *parse_complex_type(Parser *p) {
    if (cur_token_is(p, TOK_QUESTION)) {
        /* Bare wildcard type: ? */
        return "?";
    }
    if (cur_token_is(p, TOK_LBRACKET)) {
        /* Array type: [int], [int,3], [[int]], [[[int]]], etc. */
        next_token(p); /* element type or nested [ */
        if (cur_token_is(p, TOK_LBRACKET)) {
            /* Nested array type: count depth of brackets */
            int depth = 1;
            while (cur_token_is(p, TOK_LBRACKET)) {
                depth++;
                next_token(p);
            }
            const char *inner = read_type_name(p);
            for (int d = 0; d < depth; d++) {
                if (!expect_peek(p, TOK_RBRACKET)) return NULL;
            }
            size_t ts_len = strlen(inner) + (size_t)depth * 2 + 1;
            char *type_str = arena_alloc(p->arena, ts_len);
            int pos = 0;
            for (int d = 0; d < depth; d++) type_str[pos++] = '[';
            memcpy(type_str + pos, inner, strlen(inner));
            pos += (int)strlen(inner);
            for (int d = 0; d < depth; d++) type_str[pos++] = ']';
            type_str[pos] = '\0';
            return type_str;
        } else if (cur_token_is(p, TOK_CARET)) {
            /* Array of pointers: [^Type] */
            next_token(p); /* skip ^ to type name */
            const char *pointee = read_type_name(p);
            if (!expect_peek(p, TOK_RBRACKET)) return NULL;
            size_t ts_len = strlen(pointee) + 4;
            char *type_str = arena_alloc(p->arena, ts_len);
            snprintf(type_str, ts_len, "[^%s]", pointee);
            return type_str;
        } else if (cur_token_is(p, TOK_IDENT) && strcmp(p->cur_token.literal, "map") == 0 &&
                   peek_token_is(p, TOK_LBRACKET)) {
            /* Array of maps: [map[K:V]] */
            const char *elem = parse_complex_type(p);
            if (!elem) return NULL;
            if (!expect_peek(p, TOK_RBRACKET)) return NULL;
            size_t ts_len = strlen(elem) + 3;
            char *type_str = arena_alloc(p->arena, ts_len);
            snprintf(type_str, ts_len, "[%s]", elem);
            return type_str;
        } else if (cur_token_is(p, TOK_IDENT) && strcmp(p->cur_token.literal, "func") == 0 &&
                   peek_token_is(p, TOK_LPAREN)) {
            /* Arrays of typed func signatures are not supported. */
            diag_error_code(p->diag, "E2082", p->file, p->cur_token.line, p->cur_token.column, 0);
            return NULL;
        } else {
            const char *elem = read_type_name(p);
            if (peek_token_is(p, TOK_COMMA)) {
                /* Fixed-size array: [int, 3] */
                next_token(p); /* skip , */
                next_token(p); /* size */
                if (!cur_token_is(p, TOK_INT)) {
                    diag_error_code(p->diag, "E2025", p->file, p->cur_token.line, p->cur_token.column, 0);
                }
                const char *sz = p->cur_token.literal;
                if (!expect_peek(p, TOK_RBRACKET)) return NULL;
                size_t elen = strlen(elem), szlen = strlen(sz);
                size_t ts_len = elen + szlen + 4;
                char *type_str = arena_alloc(p->arena, ts_len);
                snprintf(type_str, ts_len, "[%s,%s]", elem, sz);
                return type_str;
            } else {
                /* Dynamic array: [int] */
                if (!expect_peek(p, TOK_RBRACKET)) return NULL;
                size_t ts_len = strlen(elem) + 3;
                char *type_str = arena_alloc(p->arena, ts_len);
                snprintf(type_str, ts_len, "[%s]", elem);
                return type_str;
            }
        }
    } else if (cur_token_is(p, TOK_CARET)) {
        /* Pointer type: ^T; recurse to support ^^T, ^^^T, etc. */
        next_token(p);
        const char *pointee = parse_complex_type(p);
        if (!pointee) return NULL;
        size_t ts_len = strlen(pointee) + 2;
        char *type_str = arena_alloc(p->arena, ts_len);
        snprintf(type_str, ts_len, "^%s", pointee);
        return type_str;
    } else if (cur_token_is(p, TOK_IDENT) && strcmp(p->cur_token.literal, "map") == 0 &&
               peek_token_is(p, TOK_LBRACKET)) {
        /* Map type: map[K:V]; V is parsed recursively to support nesting */
        next_token(p); /* skip [ */
        next_token(p); /* key type */
        const char *key_type = p->cur_token.literal;
        if (!expect_peek(p, TOK_COLON)) return NULL;
        next_token(p); /* value type */
        const char *val_type = parse_complex_type(p);
        if (!val_type) return NULL;
        if (!expect_peek(p, TOK_RBRACKET)) return NULL;
        /* "map[" + key + ":" + val + "]" + '\0' = klen + vlen + 7 */
        size_t klen = strlen(key_type), vlen = strlen(val_type);
        size_t ts_len = klen + vlen + 7;
        char *type_str = arena_alloc(p->arena, ts_len);
        snprintf(type_str, ts_len, "map[%s:%s]", key_type, val_type);
        return type_str;
    } else if (cur_token_is(p, TOK_IDENT) && strcmp(p->cur_token.literal, "func") == 0) {
        /* Typed function reference: func(P1, P2, ...) [-> R | -> (R1, R2, ...)]
         * Encoded as a flat string: "func(p1,p2,...)->ret" so the existing
         * type-string plumbing can carry it. `&` on a param is preserved. */
        if (!peek_token_is(p, TOK_LPAREN)) {
            /* Bare 'func' without a signature — valid as an untyped func reference
             * (e.g. map[string:func], [func], struct fields).  Just return "func". */
            return "func";
        }
        next_token(p); /* consume ( */
        /* Build params buffer */
        char params[EZ_MSG_BUF_LARGE] = {0};
        size_t plen = 0;
        if (!peek_token_is(p, TOK_RPAREN)) {
            next_token(p); /* first param */
            for (;;) {
                bool mut_p = false;
                if (cur_token_is(p, TOK_AMPERSAND)) {
                    mut_p = true;
                    next_token(p);
                }
                const char *pt = parse_complex_type(p);
                if (!pt) return NULL;
                int n = snprintf(params + plen, sizeof(params) - plen,
                    "%s%s%s", plen ? "," : "", mut_p ? "&" : "", pt);
                if (n < 0 || (size_t)n >= sizeof(params) - plen) return NULL;
                plen += (size_t)n;
                if (!peek_token_is(p, TOK_COMMA)) break;
                next_token(p); /* , */
                next_token(p); /* next param */
            }
        }
        if (!expect_peek(p, TOK_RPAREN)) return NULL;
        /* Optional -> R | -> (R1, R2, ...).
         * Absence of -> means "no return value" (the canonical encoding
         * just omits the suffix; there is no user-facing 'void' type). */
        char ret[EZ_MSG_BUF_SIZE] = {0};
        bool has_return = false;
        if (peek_token_is(p, TOK_ARROW)) {
            next_token(p); /* -> */
            next_token(p); /* first return type or ( */
            has_return = true;
            if (cur_token_is(p, TOK_LPAREN)) {
                size_t rlen = 0;
                ret[rlen++] = '(';
                next_token(p); /* first return type */
                for (;;) {
                    const char *rt = parse_complex_type(p);
                    if (!rt) return NULL;
                    if (rt && strcmp(rt, "void") == 0) {
                        diag_error_code(p->diag, "E3068", p->file, p->cur_token.line, p->cur_token.column, 0);
                        return NULL;
                    }
                    int n = snprintf(ret + rlen, sizeof(ret) - rlen, "%s%s",
                        rlen > 1 ? "," : "", rt);
                    if (n < 0 || (size_t)n >= sizeof(ret) - rlen) return NULL;
                    rlen += (size_t)n;
                    if (!peek_token_is(p, TOK_COMMA)) break;
                    next_token(p); /* , */
                    next_token(p); /* next return type */
                }
                if (!expect_peek(p, TOK_RPAREN)) return NULL;
                if (rlen + 2 >= sizeof(ret)) return NULL;
                ret[rlen++] = ')';
                ret[rlen] = '\0';
            } else {
                const char *rt = parse_complex_type(p);
                if (!rt) return NULL;
                if (strcmp(rt, "void") == 0) {
                    diag_error_code(p->diag, "E3068", p->file, p->cur_token.line, p->cur_token.column, 0);
                    return NULL;
                }
                snprintf(ret, sizeof(ret), "%s", rt);
            }
        }
        size_t ts_len = 5 /* "func(" */ + plen + 5 /* ")->\0" + slack */ + strlen(ret) + 1;
        char *type_str = arena_alloc(p->arena, ts_len);
        if (has_return) {
            snprintf(type_str, ts_len, "func(%s)->%s", params, ret);
        } else {
            snprintf(type_str, ts_len, "func(%s)", params);
        }
        return type_str;
    } else {
        /* Plain type name (possibly qualified: module.Type) */
        return read_type_name(p);
    }
}

static AstNode *parse_identifier(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
    node->data.label.value = p->cur_token.literal;
    return node;
}

static AstNode *parse_int_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_INT_VALUE, p->cur_token);
    const char *s = p->cur_token.literal;
    /* Accumulate as uint64_t so the full UINT64_MAX range is representable
     * and overflow detection works the same for every base. */
    uint64_t uval = 0;
    bool overflow_u64 = false;

    if (s[0] == '0' && (s[1] == 'x' || s[1] == 'X')) {
        s += 2;
        while (*s) {
            if (*s == '_') { s++; continue; }
            unsigned d = 0;
            if (*s >= '0' && *s <= '9') d = (unsigned)(*s - '0');
            else if (*s >= 'a' && *s <= 'f') d = (unsigned)(*s - 'a' + 10);
            else if (*s >= 'A' && *s <= 'F') d = (unsigned)(*s - 'A' + 10);
            if (uval > (UINT64_MAX >> 4)) overflow_u64 = true;
            uval = uval * 16 + d;
            s++;
        }
    } else if (s[0] == '0' && (s[1] == 'o' || s[1] == 'O')) {
        s += 2;
        while (*s) {
            if (*s == '_') { s++; continue; }
            unsigned d = (unsigned)(*s - '0');
            if (uval > (UINT64_MAX >> 3)) overflow_u64 = true;
            uval = uval * 8 + d;
            s++;
        }
    } else if (s[0] == '0' && (s[1] == 'b' || s[1] == 'B')) {
        s += 2;
        while (*s) {
            if (*s == '_') { s++; continue; }
            unsigned d = (unsigned)(*s - '0');
            if (uval > (UINT64_MAX >> 1)) overflow_u64 = true;
            uval = uval * 2 + d;
            s++;
        }
    } else {
        while (*s) {
            if (*s != '_') {
                unsigned d = (unsigned)(*s - '0');
                if (uval > (UINT64_MAX - d) / 10) overflow_u64 = true;
                uval = uval * 10 + d;
            }
            s++;
        }
    }

    node->data.int_value.value = (int64_t)uval;
    node->data.int_value.literal = p->cur_token.literal;
    node->data.int_value.overflow = overflow_u64 || uval > (uint64_t)INT64_MAX;
    node->data.int_value.overflow_u64 = overflow_u64;
    return node;
}

static AstNode *parse_float_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FLOAT_VALUE, p->cur_token);
    /* Strip underscores before parsing; atof stops at _ */
    const char *lit = p->cur_token.literal;
    if (strchr(lit, '_')) {
        char buf[EZ_FLOAT_LIT_BUF];
        int j = 0;
        for (int i = 0; lit[i] && j < (int)sizeof(buf) - 1; i++) {
            if (lit[i] != '_') buf[j++] = lit[i];
        }
        buf[j] = '\0';
        node->data.float_value.value = atof(buf);
    } else {
        node->data.float_value.value = atof(lit);
    }
    return node;
}

static bool has_interpolation(const char *s) {
    for (int i = 0; s[i]; i++) {
        if (s[i] == '$' && s[i + 1] == '{') return true;
        if (s[i] == '\\') i++;
    }
    return false;
}

static AstNode *parse_interpolated_string(Parser *p, const char *raw) {
    AstNode *node = ast_alloc(p->arena, NODE_INTERPOLATED_STRING, p->cur_token);

    int cap = 8;
    int count = 0;
    AstNode **parts = arena_alloc(p->arena, sizeof(AstNode *) * cap);

    const char *s = raw;
    const char *seg_start = s;

    while (*s) {
        if (*s == '\\' && *(s + 1)) {
            s += 2;
            continue;
        }
        if (*s == '$' && *(s + 1) == '{') {
            /* Emit the text segment before ${ */
            if (s > seg_start) {
                if (count >= cap) {
                    cap *= 2;
                    AstNode **new_parts = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                    memcpy(new_parts, parts, sizeof(AstNode *) * count);
                    parts = new_parts;
                }
                AstNode *text = ast_alloc(p->arena, NODE_STRING_VALUE, p->cur_token);
                text->data.string_value.value = arena_strndup(p->arena, seg_start, s - seg_start);
                parts[count++] = text;
            }

            /* Find matching } and parse the expression inside */
            s += 2; /* skip ${ */
            const char *expr_start = s;
            int brace_depth = 1;
            while (*s && brace_depth > 0) {
                if (*s == '{') brace_depth++;
                else if (*s == '}') brace_depth--;
                if (brace_depth > 0) s++;
            }

            /* Parse the expression text */
            if (count >= cap) {
                cap *= 2;
                AstNode **new_parts = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                memcpy(new_parts, parts, sizeof(AstNode *) * count);
                parts = new_parts;
            }

            char *expr_text = arena_strndup(p->arena, expr_start, s - expr_start);
            /* reject '${}' (and whitespace-only '${ }') before
             * spinning up a sub-parser. Otherwise the sub-parser hits
             * EOF on an empty input and reports E2002 with the stale
             * file-start position it was initialized to, which is
             * actively misleading. */
            bool is_empty = true;
            for (const char *c = expr_text; *c; c++) {
                if (*c != ' ' && *c != '\t' && *c != '\n' && *c != '\r') {
                    is_empty = false;
                    break;
                }
            }
            if (is_empty) {
                diag_error_code(p->diag, "E2071", p->file, p->cur_token.line, p->cur_token.column, 0);
            } else {
                Lexer *expr_lexer = lexer_create(p->arena, expr_text, p->file);
                /* Offset the sub-lexer to the real source position of this
                 * ${...} expression so diagnostics point at the right line
                 * and column instead of always reporting 1:N. expr_start
                 * points to the first char of the expression (past "${"),
                 * so (expr_start - raw) is its byte offset from the opening
                 * quote. Count any newlines in the string before this point
                 * to handle multi-line strings correctly. */
                {
                    int line_off = 0;
                    int col_from_nl = 0;
                    for (const char *cp = raw; cp < expr_start; cp++) {
                        if (*cp == '\n') { line_off++; col_from_nl = 0; }
                        else col_from_nl++;
                    }
                    expr_lexer->line = p->cur_token.line + line_off;
                    if (line_off > 0)
                        expr_lexer->column = col_from_nl + 1;
                    else
                        expr_lexer->column = p->cur_token.column + 1 + (int)(expr_start - raw);
                }
                Parser *expr_parser = parser_create(p->arena, expr_lexer, p->file, p->diag);
                /* Inherit struct names from parent so interpolated expressions
                 * can recognize struct literals. */
                expr_parser->struct_names = p->struct_names;
                expr_parser->struct_name_count = p->struct_name_count;
                expr_parser->struct_name_cap = p->struct_name_cap;
                AstNode *expr = parse_expression(expr_parser, PREC_LOWEST);
                if (expr) parts[count++] = expr;
            }

            if (*s == '}') s++;
            seg_start = s;
        } else {
            s++;
        }
    }

    /* Remaining text segment */
    if (s > seg_start) {
        if (count >= cap) {
            cap *= 2;
            AstNode **new_parts = arena_alloc(p->arena, sizeof(AstNode *) * cap);
            memcpy(new_parts, parts, sizeof(AstNode *) * count);
            parts = new_parts;
        }
        AstNode *text = ast_alloc(p->arena, NODE_STRING_VALUE, p->cur_token);
        text->data.string_value.value = arena_strndup(p->arena, seg_start, s - seg_start);
        parts[count++] = text;
    }

    node->data.interpolated_string.parts = parts;
    node->data.interpolated_string.part_count = count;
    return node;
}

static AstNode *parse_string_literal(Parser *p) {
    const char *raw = p->cur_token.literal;
    bool is_raw = (p->cur_token.type == TOK_RAW_STRING);
    if (!is_raw && has_interpolation(raw)) {
        return parse_interpolated_string(p, raw);
    }
    if (!is_raw) {
        /* E2057: check for bare $identifier (missing braces) */
        for (int i = 0; raw[i]; i++) {
            if (raw[i] == '\\') { i++; continue; }
            if (raw[i] == '$' && raw[i + 1] != '{' && raw[i + 1] != '\0' &&
                (isalpha((unsigned char)raw[i + 1]) || raw[i + 1] == '_')) {
                diag_error_code(p->diag, "E2057", p->file, p->cur_token.line, p->cur_token.column, 0);
                break;
            }
        }
    }
    AstNode *node = ast_alloc(p->arena, NODE_STRING_VALUE, p->cur_token);
    node->data.string_value.value = raw;
    node->data.string_value.is_raw = is_raw;
    return node;
}

static AstNode *parse_bool_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_BOOL_VALUE, p->cur_token);
    node->data.bool_value.value = (p->cur_token.type == TOK_TRUE);
    return node;
}

static AstNode *parse_nil_literal(Parser *p) {
    return ast_alloc(p->arena, NODE_NIL_VALUE, p->cur_token);
}

static AstNode *parse_prefix_expression(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_PREFIX_EXPR, p->cur_token);
    node->data.prefix.op = p->cur_token.literal;
    next_token(p);
    node->data.prefix.right = parse_expression(p, PREC_PREFIX);
    return node;
}

static AstNode *parse_grouped_expression(Parser *p) {
    /* Check for function reference: ()func_name or ()Type.func */
    if (peek_token_is(p, TOK_RPAREN)) {
        Token ref_tok = p->cur_token;
        next_token(p); /* consume ) */
        next_token(p); /* move to identifier */

        if (p->cur_token.type != TOK_IDENT) {
            return NULL;
        }

        /* Parse the function name; may be qualified with dots */
        AstNode *func_expr = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
        func_expr->data.label.value = p->cur_token.literal;

        while (peek_token_is(p, TOK_DOT)) {
            next_token(p); /* consume . */
            next_token(p); /* move to member */
            AstNode *member = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
            member->data.member.object = func_expr;
            member->data.member.member = p->cur_token.literal;
            func_expr = member;
        }

        AstNode *ref = ast_alloc(p->arena, NODE_FUNC_REF, ref_tok);
        ref->data.func_ref.function = func_expr;
        return ref;
    }

    next_token(p);
    AstNode *expr = parse_expression(p, PREC_LOWEST);
    if (!expect_peek(p, TOK_RPAREN)) return NULL;
    return expr;
}

/* Parse prefix expression (the "nud" in Pratt parsing) */
static AstNode *parse_prefix(Parser *p) {
    switch (p->cur_token.type) {
    case TOK_IDENT:
        /* Check for module-qualified struct literal: mod.Name{ ... } */
        if (peek_token_is(p, TOK_DOT)) {
            const char *mod = p->cur_token.literal;
            if (mod[0] >= 'a' && mod[0] <= 'z') {
                /* Save state for lookahead */
                Token saved_cur = p->cur_token;
                Token saved_peek = p->peek_token;
                int saved_pos = p->lexer->position;
                int saved_rpos = p->lexer->read_position;
                char saved_ch = p->lexer->ch;
                int saved_line = p->lexer->line;
                int saved_col = p->lexer->column;

                next_token(p); /* consume . */
                next_token(p); /* move to potential type name */

                if (p->cur_token.type == TOK_IDENT &&
                    peek_token_is(p, TOK_LBRACE) && !p->no_struct_literal &&
                    (parser_is_struct_name(p, p->cur_token.literal) ||
                     (p->cur_token.literal[0] >= 'A' && p->cur_token.literal[0] <= 'Z'))) {
                    /* mod.Name{; module-qualified struct literal */
                    char *prefixed = arena_alloc(p->arena, EZ_MSG_BUF_SIZE);
                    snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod, p->cur_token.literal);
                    next_token(p); /* move to { */
                    return parse_struct_literal(p, prefixed);
                }

                /* Not a struct literal; restore state */
                p->cur_token = saved_cur;
                p->peek_token = saved_peek;
                p->lexer->position = saved_pos;
                p->lexer->read_position = saved_rpos;
                p->lexer->ch = saved_ch;
                p->lexer->line = saved_line;
                p->lexer->column = saved_col;
            }
        }
        /* Check for struct literal: Name{ ... }
         * Use the prescan list first (exact match), fall back to uppercase
         * heuristic for imported struct names not in the current file. */
        if (peek_token_is(p, TOK_LBRACE) && !p->no_struct_literal) {
            const char *name = p->cur_token.literal;
            if (parser_is_struct_name(p, name) ||
                (name[0] >= 'A' && name[0] <= 'Z')) {
                next_token(p); /* move to { */
                return parse_struct_literal(p, name);
            }
        }
        return parse_identifier(p);
    case TOK_INT:       return parse_int_literal(p);
    case TOK_FLOAT:     return parse_float_literal(p);
    case TOK_STRING:
    case TOK_RAW_STRING: return parse_string_literal(p);
    case TOK_TRUE:
    case TOK_FALSE:     return parse_bool_literal(p);
    case TOK_NIL:       return parse_nil_literal(p);
    case TOK_CHAR: {
        AstNode *node = ast_alloc(p->arena, NODE_CHAR_VALUE, p->cur_token);
        node->data.char_value.value = p->cur_token.literal[0];
        if (p->cur_token.literal[0] == '\\' && p->cur_token.literal[1]) {
            switch (p->cur_token.literal[1]) {
            case 'n': node->data.char_value.value = '\n'; break;
            case 't': node->data.char_value.value = '\t'; break;
            case 'r': node->data.char_value.value = '\r'; break;
            case '\\': node->data.char_value.value = '\\'; break;
            case '\'': node->data.char_value.value = '\''; break;
            case '0': node->data.char_value.value = '\0'; break;
            default: node->data.char_value.value = p->cur_token.literal[1]; break;
            }
        }
        return node;
    }
    case TOK_MINUS:
    case TOK_BANG:
    case TOK_BIT_NOT: return parse_prefix_expression(p);
    case TOK_AMPERSAND: {
        diag_error_code(p->diag, "E2072",
            p->file, p->cur_token.line, p->cur_token.column, 0);
        return parse_prefix_expression(p);
    }
    case TOK_LPAREN:    return parse_grouped_expression(p);
    case TOK_LBRACE: {
        /* Could be array literal {1, 2, 3} or map literal {"k": v, ...}
         * Detect map by checking for colon after first expression */
        Token brace_tok = p->cur_token;
        next_token(p); /* skip { */

        /* Empty: {}; treat as empty array (context-dependent) */
        if (cur_token_is(p, TOK_RBRACE)) {
            AstNode *node = ast_alloc(p->arena, NODE_ARRAY_VALUE, brace_tok);
            node->data.array_value.count = 0;
            node->data.array_value.elements = NULL;
            return node;
        }

        /* Empty map: {:} */
        if (cur_token_is(p, TOK_COLON) && peek_token_is(p, TOK_RBRACE)) {
            next_token(p); /* skip } */
            AstNode *node = ast_alloc(p->arena, NODE_MAP_VALUE, brace_tok);
            node->data.map_value.count = 0;
            node->data.map_value.keys = NULL;
            node->data.map_value.values = NULL;
            return node;
        }

        /* Leading comma: {, 1, 2} — reuse the trailing-comma diagnostic. */
        if (cur_token_is(p, TOK_COMMA)) {
            diag_error_code(p->diag, "E2017",
                p->file, p->cur_token.line, p->cur_token.column, 0);
            next_token(p); /* skip the stray , and keep parsing */
        }

        /* Parse first expression */
        AstNode *first = parse_expression(p, PREC_LOWEST);

        if (peek_token_is(p, TOK_COLON)) {
            /* Map literal: {"key": value, ...} */
            AstNode *node = ast_alloc(p->arena, NODE_MAP_VALUE, brace_tok);
            int cap = 8;
            node->data.map_value.count = 0;
            node->data.map_value.keys = arena_alloc(p->arena, sizeof(AstNode *) * cap);
            node->data.map_value.values = arena_alloc(p->arena, sizeof(AstNode *) * cap);

            /* First key already parsed */
            node->data.map_value.keys[0] = first;
            next_token(p); /* skip : */
            next_token(p);
            node->data.map_value.values[0] = parse_expression(p, PREC_LOWEST);
            node->data.map_value.count = 1;

            while (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip , */
                next_token(p);
                if (cur_token_is(p, TOK_RBRACE)) break;
                if (node->data.map_value.count >= cap) {
                    cap *= 2;
                    AstNode **nk = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                    AstNode **nv = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                    memcpy(nk, node->data.map_value.keys, sizeof(AstNode *) * node->data.map_value.count);
                    memcpy(nv, node->data.map_value.values, sizeof(AstNode *) * node->data.map_value.count);
                    node->data.map_value.keys = nk;
                    node->data.map_value.values = nv;
                }
                node->data.map_value.keys[node->data.map_value.count] =
                    parse_expression(p, PREC_LOWEST);
                if (!expect_peek(p, TOK_COLON)) return NULL;
                next_token(p);
                node->data.map_value.values[node->data.map_value.count] =
                    parse_expression(p, PREC_LOWEST);
                node->data.map_value.count++;
            }
            if (peek_token_is(p, TOK_RBRACE)) next_token(p);
            return node;
        }

        /* Array literal: {expr, expr, ...} */
        AstNode *node = ast_alloc(p->arena, NODE_ARRAY_VALUE, brace_tok);
        int cap = 8;
        node->data.array_value.count = 0;
        node->data.array_value.elements = arena_alloc(p->arena, sizeof(AstNode *) * cap);
        node->data.array_value.elements[node->data.array_value.count++] = first;

        while (peek_token_is(p, TOK_COMMA)) {
            next_token(p); /* skip , */
            next_token(p);
            if (cur_token_is(p, TOK_RBRACE)) {
                diag_error_code(p->diag, "E2017",
                    p->file, p->cur_token.line, p->cur_token.column, 0);
                break;
            }
            if (node->data.array_value.count >= cap) {
                cap *= 2;
                AstNode **new_elems = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                memcpy(new_elems, node->data.array_value.elements,
                    sizeof(AstNode *) * node->data.array_value.count);
                node->data.array_value.elements = new_elems;
            }
            node->data.array_value.elements[node->data.array_value.count++] =
                parse_expression(p, PREC_LOWEST);
        }
        if (peek_token_is(p, TOK_RBRACE)) next_token(p);
        return node;
    }
    case TOK_CAST: {
        /* cast(value, type) */
        AstNode *node = ast_alloc(p->arena, NODE_CAST_EXPR, p->cur_token);
        node->data.cast.is_array = false;
        node->data.cast.element_type = NULL;
        if (!expect_peek(p, TOK_LPAREN)) return NULL;
        next_token(p);
        node->data.cast.value = parse_expression(p, PREC_LOWEST);
        if (!expect_peek(p, TOK_COMMA)) return NULL;
        next_token(p);
        if (cur_token_is(p, TOK_LBRACKET)) {
            /* Array type: cast(value, [type]) */
            next_token(p); /* element type */
            const char *elem = p->cur_token.literal;
            if (!expect_peek(p, TOK_RBRACKET)) return NULL;
            size_t ts_len = strlen(elem) + 3;
            char *type_str = arena_alloc(p->arena, ts_len);
            snprintf(type_str, ts_len, "[%s]", elem);
            node->data.cast.target_type = type_str;
            node->data.cast.is_array = true;
            node->data.cast.element_type = elem;
        } else {
            node->data.cast.target_type = p->cur_token.literal;
        }
        if (!expect_peek(p, TOK_RPAREN)) return NULL;
        return node;
    }
    case TOK_NEW: {
        /* new(Type); allocate zeroed value on default arena */
        AstNode *node = ast_alloc(p->arena, NODE_NEW_EXPR, p->cur_token);
        if (!expect_peek(p, TOK_LPAREN)) return NULL;
        next_token(p);
        node->data.new_expr.type_name = read_type_name(p);
        if (type_string_has_wildcard(node->data.new_expr.type_name)) {
            diag_error_msg(p->diag, "E2070",
                arena_strdup(p->arena,
                    "wildcard type '?' cannot be used with new(); new() requires a concrete type"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
        }
        if (!expect_peek(p, TOK_RPAREN)) return NULL;
        return node;
    }
    case TOK_RANGE: {
        /* range(end) or range(start, end) or range(start, end, step) */
        AstNode *node = ast_alloc(p->arena, NODE_RANGE_EXPR, p->cur_token);
        node->data.range_expr.start = NULL;
        node->data.range_expr.end = NULL;
        node->data.range_expr.step = NULL;
        if (!expect_peek(p, TOK_LPAREN)) return NULL;
        next_token(p);
        AstNode *first = parse_expression(p, PREC_LOWEST);
        if (peek_token_is(p, TOK_COMMA)) {
            node->data.range_expr.start = first;
            next_token(p); /* skip comma */
            next_token(p);
            node->data.range_expr.end = parse_expression(p, PREC_LOWEST);
            if (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma */
                next_token(p);
                node->data.range_expr.step = parse_expression(p, PREC_LOWEST);
            }
        } else {
            /* range(end) - start defaults to 0 */
            node->data.range_expr.end = first;
        }
        if (!expect_peek(p, TOK_RPAREN)) return NULL;
        return node;
    }
    default:
    {
        /* Skip generic error for ILLEGAL tokens; the lexer already emitted a specific diagnostic */
        if (p->cur_token.type != TOK_ILLEGAL) {
            char buf[EZ_MSG_BUF_SIZE];
            snprintf(buf, sizeof(buf), "unexpected token '%s'",
                token_type_name(p->cur_token.type));
            diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, buf),
                p->file, p->cur_token.line, p->cur_token.column, 0);
        }
    }
        return NULL;
    }
}

static AstNode *parse_infix_expression(Parser *p, AstNode *left) {
    AstNode *node = ast_alloc(p->arena, NODE_INFIX_EXPR, p->cur_token);
    node->data.infix.left = left;
    node->data.infix.op = p->cur_token.literal;
    Precedence prec = token_precedence(p->cur_token.type);
    bool is_membership = (p->cur_token.type == TOK_IN || p->cur_token.type == TOK_NOT_IN);
    /* Suppress struct-literal parsing on the RHS of comparisons and
     * membership operators.  In `if x < Foo {`, the `{` is the
     * if-block, not the start of a struct literal `Foo{ ... }`. */
    bool suppress_struct_lit = is_membership ||
        p->cur_token.type == TOK_LT || p->cur_token.type == TOK_GT ||
        p->cur_token.type == TOK_LT_EQ || p->cur_token.type == TOK_GT_EQ ||
        p->cur_token.type == TOK_EQ || p->cur_token.type == TOK_NOT_EQ;
    next_token(p);
    if (suppress_struct_lit) p->no_struct_literal = true;
    node->data.infix.right = parse_expression(p, prec);
    if (suppress_struct_lit) p->no_struct_literal = false;
    return node;
}

static AstNode **parse_expression_list(Parser *p, TokenType end, int *count) {
    *count = 0;
    int cap = 4;
    AstNode **list = arena_alloc(p->arena, sizeof(AstNode *) * cap);

    if (peek_token_is(p, end)) {
        next_token(p);
        return list;
    }

    next_token(p);
    list[(*count)++] = parse_expression(p, PREC_LOWEST);

    while (peek_token_is(p, TOK_COMMA)) {
        next_token(p); /* skip comma */
        next_token(p);
        if (*count >= cap) {
            cap *= 2;
            AstNode **new_list = arena_alloc(p->arena, sizeof(AstNode *) * cap);
            memcpy(new_list, list, sizeof(AstNode *) * (*count));
            list = new_list;
        }
        list[(*count)++] = parse_expression(p, PREC_LOWEST);
    }

    if (!expect_peek(p, end)) return NULL;
    return list;
}

static AstNode *parse_call_expression(Parser *p, AstNode *function) {
    if (p->cur_token.preceded_by_ws) {
        diag_error_code(p->diag, "E2073", p->file, p->cur_token.line, p->cur_token.column, 0);
        /* Still parse the argument list so we consume the closing ')'
         * and don't cascade into E2001/E2002 on the unrelated tokens. */
    }
    AstNode *node = ast_alloc(p->arena, NODE_CALL_EXPR, p->cur_token);
    node->data.call.function = function;
    node->data.call.args = parse_expression_list(p, TOK_RPAREN, &node->data.call.arg_count);
    return node;
}

static AstNode *parse_member_expression(Parser *p, AstNode *object) {
    if (p->cur_token.preceded_by_ws) {
        diag_error_code(p->diag, "E2074", p->file, p->cur_token.line, p->cur_token.column, 0);
        /* Continue parsing the member so we swallow the identifier after '.'
         * and don't cascade into unrelated diagnostics. */
    }
    next_token(p); /* skip the identifier after dot */
    AstNode *node = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
    node->data.member.object = object;
    node->data.member.member = p->cur_token.literal;
    return node;
}

static AstNode *parse_index_expression(Parser *p, AstNode *left) {
    if (p->cur_token.preceded_by_ws) {
        diag_error_code(p->diag, "E2075", p->file, p->cur_token.line, p->cur_token.column, 0);
        /* Continue parsing so we consume the closing ']'. */
    }
    AstNode *node = ast_alloc(p->arena, NODE_INDEX_EXPR, p->cur_token);
    node->data.index_expr.left = left;
    next_token(p);
    node->data.index_expr.index = parse_expression(p, PREC_LOWEST);
    if (!expect_peek(p, TOK_RBRACKET)) return NULL;
    return node;
}

static AstNode *parse_postfix_expression(Parser *p, AstNode *left) {
    if (p->cur_token.preceded_by_ws) {
        diag_error_code(p->diag, "E2076", p->file, p->cur_token.line, p->cur_token.column, 0);
    }
    AstNode *node = ast_alloc(p->arena, NODE_POSTFIX_EXPR, p->cur_token);
    node->data.postfix.left = left;
    node->data.postfix.op = p->cur_token.literal;
    return node;
}

/* Parse infix expression (the "led" in Pratt parsing) */
static AstNode *parse_infix(Parser *p, AstNode *left) {
    switch (p->cur_token.type) {
    case TOK_PLUS: case TOK_MINUS: case TOK_ASTERISK: case TOK_SLASH:
    case TOK_PERCENT:
    case TOK_EQ: case TOK_NOT_EQ: case TOK_LT: case TOK_GT:
    case TOK_LT_EQ: case TOK_GT_EQ:
    case TOK_AND: case TOK_OR:
    case TOK_IN: case TOK_NOT_IN:
    case TOK_BIT_AND: case TOK_BIT_OR: case TOK_BIT_XOR:
    case TOK_BIT_SHIFT_LEFT: case TOK_BIT_SHIFT_RIGHT:
        return parse_infix_expression(p, left);
    case TOK_LPAREN:
        return parse_call_expression(p, left);
    case TOK_DOT:
        return parse_member_expression(p, left);
    case TOK_LBRACKET:
        return parse_index_expression(p, left);
    case TOK_INCREMENT:
    case TOK_DECREMENT:
    case TOK_CARET:
        return parse_postfix_expression(p, left);
    default:
        return left;
    }
}

static AstNode *parse_expression(Parser *p, Precedence prec) {
    p->depth++;
    if (p->depth > EZ_MAX_PARSE_DEPTH) {
        diag_error_msg(p->diag, "E2001",
            strdup("expression is nested too deeply; maximum depth is 256"),
            p->file, p->cur_token.line, p->cur_token.column, 0);
        p->depth--;
        return NULL;
    }

    AstNode *left = parse_prefix(p);
    if (!left) { p->depth--; return NULL; }

    while (!peek_token_is(p, TOK_EOF) && prec < token_precedence(p->peek_token.type)) {
        next_token(p);
        left = parse_infix(p, left);
        if (!left) { p->depth--; return NULL; }
    }

    p->depth--;
    return left;
}

/* --- Statement Parsing --- */

/* If peek is or_return, consume it and desugar
 *
 *     <var_decl with value = expr> or_return
 *
 * into a block:
 *
 *     mut _tmp = expr
 *     if _tmp.v1 != nil { return _tmp.v1 }
 *     <var_decl with value = _tmp.v0>
 *
 * Returns the desugared block, or NULL if no or_return was present
 * (caller keeps the original var_decl untouched). */
static AstNode *maybe_apply_or_return(Parser *p, AstNode *var_decl) {
    if (!peek_token_is(p, TOK_OR_RETURN)) return NULL;
    next_token(p); /* consume or_return */

    /* Check for custom fallback values on the same line as the or_return token.
     * e.g. `mut val = risky(fail) or_return -99` → return -99, _tmp.v1
     * The caller provides all non-error return slots; _tmp.v1 is appended. */
    int or_return_line = p->cur_token.line;
    AstNode *fallback_buf[16];
    int fallback_count = 0;
    if (p->peek_token.type != TOK_EOF &&
        p->peek_token.type != TOK_SEMICOLON &&
        p->peek_token.type != TOK_RBRACE &&
        p->peek_token.line == or_return_line) {
        next_token(p); /* advance to first fallback token */
        while (fallback_count < 16) {
            fallback_buf[fallback_count++] = parse_expression(p, PREC_LOWEST);
            if (!peek_token_is(p, TOK_COMMA)) break;
            next_token(p); /* skip comma */
            next_token(p); /* advance to next fallback token */
        }
    }

    static int or_return_counter = 0;
    char *tmp_name = arena_alloc(p->arena, EZ_TMP_NAME_BUF);
    snprintf(tmp_name, EZ_TMP_NAME_BUF, "_ez_or%d", or_return_counter++);

    AstNode *block = ast_alloc(p->arena, NODE_BLOCK_STMT, p->cur_token);
    block->data.block.cap = 4;
    block->data.block.count = 0;
    block->data.block.stmts = arena_alloc(p->arena, sizeof(AstNode *) * block->data.block.cap);

    /* _tmp = expr */
    AstNode *tmp_decl = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
    tmp_decl->data.var_decl.mutable = true;
    tmp_decl->data.var_decl.name = tmp_name;
    tmp_decl->data.var_decl.type_name = NULL;
    tmp_decl->data.var_decl.value = var_decl->data.var_decl.value;
    block->data.block.stmts[block->data.block.count++] = tmp_decl;

    /* if (_tmp.v1 != nil) { return _tmp.v1 } */
    AstNode *if_stmt = ast_alloc(p->arena, NODE_IF_STMT, p->cur_token);
    AstNode *err_access = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
    AstNode *tmp_label = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
    tmp_label->data.label.value = tmp_name;
    err_access->data.member.object = tmp_label;
    err_access->data.member.member = "v1";
    AstNode *nil_val = ast_alloc(p->arena, NODE_NIL_VALUE, p->cur_token);
    AstNode *cond = ast_alloc(p->arena, NODE_INFIX_EXPR, p->cur_token);
    cond->data.infix.left = err_access;
    cond->data.infix.op = "!=";
    cond->data.infix.right = nil_val;
    if_stmt->data.if_stmt.condition = cond;

    AstNode *ret_block = ast_alloc(p->arena, NODE_BLOCK_STMT, p->cur_token);
    ret_block->data.block.cap = 1;
    ret_block->data.block.count = 0;
    ret_block->data.block.stmts = arena_alloc(p->arena, sizeof(AstNode *));
    AstNode *ret_stmt = ast_alloc(p->arena, NODE_RETURN_STMT, p->cur_token);
    /* Build _tmp.v1 node (the propagated error value) */
    AstNode *err_access2 = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
    AstNode *tmp_label2 = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
    tmp_label2->data.label.value = tmp_name;
    err_access2->data.member.object = tmp_label2;
    err_access2->data.member.member = "v1";
    if (fallback_count > 0) {
        /* Custom fallback: return fallback_vals..., _tmp.v1 */
        int total = fallback_count + 1;
        ret_stmt->data.return_stmt.values = arena_alloc(p->arena, sizeof(AstNode *) * total);
        for (int i = 0; i < fallback_count; i++)
            ret_stmt->data.return_stmt.values[i] = fallback_buf[i];
        ret_stmt->data.return_stmt.values[fallback_count] = err_access2;
        ret_stmt->data.return_stmt.count = total;
    } else {
        /* Bare or_return: propagate just the error; codegen fills {0} for other slots */
        ret_stmt->data.return_stmt.values = arena_alloc(p->arena, sizeof(AstNode *));
        ret_stmt->data.return_stmt.values[0] = err_access2;
        ret_stmt->data.return_stmt.count = 1;
    }
    ret_block->data.block.stmts[ret_block->data.block.count++] = ret_stmt;
    if_stmt->data.if_stmt.consequence = ret_block;
    if_stmt->data.if_stmt.alternative = NULL;
    block->data.block.stmts[block->data.block.count++] = if_stmt;

    /* x = _tmp.v0 */
    AstNode *var = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
    var->data.var_decl.mutable = var_decl->data.var_decl.mutable;
    var->data.var_decl.name = var_decl->data.var_decl.name;
    var->data.var_decl.type_name = var_decl->data.var_decl.type_name;
    AstNode *val_access = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
    AstNode *tmp_label3 = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
    tmp_label3->data.label.value = tmp_name;
    val_access->data.member.object = tmp_label3;
    val_access->data.member.member = "v0";
    var->data.var_decl.value = val_access;
    block->data.block.stmts[block->data.block.count++] = var;

    return block;
}

/* Bare throwaway: `_ = expr` (no mut/const keyword) at statement position.
 * Desugars to a var_decl(name="_", mutable=true), which the typechecker
 * and codegen already special-case to skip symbol creation and emit
 * `(void)(expr);`. or_return is supported via the shared helper. */
/* E5012: the throwaway '_' is only meaningful when the RHS is a
 * function call. Literal/identifier/arithmetic RHSes have no return
 * value to discard and no side effect to run, so `_ = 32` etc. are
 * dead code with a misleading name. Checked at parse time (before
 * or_return desugaring) so the user-written RHS, not the rewritten
 * member-access, is what gets validated. */
static void check_throwaway_target(Parser *p, AstNode *value) {
    if (!value || value->kind == NODE_CALL_EXPR) return;
    diag_error_code(p->diag, "E5012",
        p->file, value->token.line, value->token.column, 0);
}

static AstNode *parse_discard_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
    node->data.var_decl.mutable = true;
    node->data.var_decl.name = "_";
    node->data.var_decl.type_name = NULL;
    node->data.var_decl.value = NULL;

    next_token(p); /* consume = */
    next_token(p); /* move to RHS */
    node->data.var_decl.value = parse_expression(p, PREC_LOWEST);
    if (!node->data.var_decl.value) return NULL;

    check_throwaway_target(p, node->data.var_decl.value);

    AstNode *desugared = maybe_apply_or_return(p, node);
    if (desugared) return desugared;
    return node;
}

static AstNode *parse_var_declaration(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
    node->data.var_decl.mutable = (p->cur_token.type == TOK_MUT);

    if (peek_token_is(p, TOK_IDENT) || peek_token_is(p, TOK_BLANK)) {
        next_token(p);
    } else if (is_keyword_token(p->peek_token.type)) {
        char msg[EZ_MSG_BUF_SIZE];
        snprintf(msg, sizeof(msg),
            "'%s' is a reserved keyword and cannot be used as a variable name",
            p->peek_token.literal);
        diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, msg),
            p->file, p->peek_token.line, p->peek_token.column, 0);
        synchronize(p);
        return NULL;
    } else {
        expect_peek(p, TOK_IDENT); /* will error */
        return NULL;
    }
    node->data.var_decl.name = p->cur_token.literal;

    /* Optional type annotation. TOK_QUESTION is included so a bare
     * wildcard `?` in a var_decl flows through parse_complex_type and
     * lands on the existing E2070 diagnostic below; without it, the
     * token falls through to the generic "unexpected token" fallback
     * and the user gets no hint about why `?` isn't allowed here
     * (). */
    /* E2079: reject 'nil' as a type annotation. nil is a value per the
     * language, not a type; consume the token to avoid a cascading
     * "nil is an unexpected expression statement" diagnostic. */
    if (peek_token_is(p, TOK_NIL)) {
        next_token(p); /* consume nil */
        diag_error_code(p->diag, "E2079",
            p->file, p->cur_token.line, p->cur_token.column, 0);
    }

    node->data.var_decl.type_name = NULL;
    if (peek_token_is(p, TOK_IDENT) || peek_token_is(p, TOK_CARET) || peek_token_is(p, TOK_LBRACKET) ||
        peek_token_is(p, TOK_STRUCT) || peek_token_is(p, TOK_ENUM) ||
        peek_token_is(p, TOK_QUESTION)) {
        next_token(p);
        node->data.var_decl.type_name = parse_complex_type(p);
        if (!node->data.var_decl.type_name) return NULL;
        /* E2070: wildcard `?` only allowed in function signatures */
        if (type_string_has_wildcard(node->data.var_decl.type_name)) {
            diag_error_msg(p->diag, "E2070",
                arena_strdup(p->arena,
                    "wildcard type '?' is only allowed in function parameter and return types; not in variable declarations"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
        }
        /* E2068: mut <name> struct/enum; should be const */
        if (node->data.var_decl.mutable &&
            (strcmp(node->data.var_decl.type_name, "struct") == 0 ||
             strcmp(node->data.var_decl.type_name, "enum") == 0)) {
            diag_error_codef(p->diag, "E2068", p->file, node->token.line, node->token.column, 0, node->data.var_decl.type_name);
            return NULL;
        }
    }

    /* Check for multi-var declaration: temp x int, y int = expr OR temp _, _ = expr */
    if (peek_token_is(p, TOK_COMMA)) {
            /* Collect all variable names and types */
            const char *names[MAX_MULTI_VARS];
            const char *types[MAX_MULTI_VARS];
            int var_count = 0;
            names[var_count] = node->data.var_decl.name;
            types[var_count] = node->data.var_decl.type_name;
            var_count++;

            while (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma */
                next_token(p); /* name (IDENT or _) */
                names[var_count] = p->cur_token.literal;
                if (cur_token_is(p, TOK_BLANK)) names[var_count] = "_";
                if (peek_token_is(p, TOK_IDENT)) {
                    next_token(p); /* type */
                    types[var_count] = p->cur_token.literal;
                } else {
                    types[var_count] = NULL;
                }
                var_count++;
                if (var_count > MAX_MULTI_VARS) {
                    diag_error_codef(p->diag, "E2062", p->file, p->cur_token.line, p->cur_token.column, 0, MAX_MULTI_VARS);
                    return NULL;
                }
            }

            /* Expect = expr */
            if (!expect_peek(p, TOK_ASSIGN)) return NULL;
            next_token(p);
            AstNode *value = parse_expression(p, PREC_LOWEST);

            /* Generate unique temp name */
            static int multi_var_counter = 0;
            char *tmp_name = arena_alloc(p->arena, EZ_TMP_NAME_BUF);
            snprintf(tmp_name, EZ_TMP_NAME_BUF, "_ez_tmp%d", multi_var_counter++);

            /* Create a block with: __auto_type _tmp = expr; type x = _tmp.v0; ... */
            AstNode *block = ast_alloc(p->arena, NODE_BLOCK_STMT, p->cur_token);
            block->data.block.cap = var_count + 1;
            block->data.block.count = 0;
            block->data.block.stmts = arena_alloc(p->arena, sizeof(AstNode *) * block->data.block.cap);

            /* temp _tmp = value */
            AstNode *tmp_decl = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
            tmp_decl->data.var_decl.mutable = true;
            tmp_decl->data.var_decl.name = tmp_name;
            tmp_decl->data.var_decl.type_name = NULL;
            tmp_decl->data.var_decl.value = value;
            block->data.block.stmts[block->data.block.count++] = tmp_decl;

            /* Individual declarations: type x = _tmp.v0 */
            for (int i = 0; i < var_count; i++) {
                AstNode *vd = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
                vd->data.var_decl.mutable = node->data.var_decl.mutable;
                vd->data.var_decl.name = names[i];
                vd->data.var_decl.type_name = types[i];
                /* Value: _ez_tmp.vN */
                AstNode *member = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
                AstNode *label = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
                label->data.label.value = tmp_name;
                member->data.member.object = label;
                char *field = arena_alloc(p->arena, EZ_FIELD_NAME_BUF);
                snprintf(field, EZ_FIELD_NAME_BUF, "v%d", i);
                member->data.member.member = field;
                vd->data.var_decl.value = member;
                block->data.block.stmts[block->data.block.count++] = vd;
            }

            return block;
    }

    /* = value */
    if (peek_token_is(p, TOK_ASSIGN)) {
        next_token(p); /* skip = */
        next_token(p);
        node->data.var_decl.value = parse_expression(p, PREC_LOWEST);

        if (node->data.var_decl.value &&
            strcmp(node->data.var_decl.name, "_") == 0) {
            check_throwaway_target(p, node->data.var_decl.value);
        }

        AstNode *desugared = maybe_apply_or_return(p, node);
        if (desugared) return desugared;
    }

    return node;
}

static AstNode *parse_return_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_RETURN_STMT, p->cur_token);

    int cap = 4;
    int count = 0;
    AstNode **values = arena_alloc(p->arena, sizeof(AstNode *) * cap);

    /* Check if there's a value to return (peek, don't consume) */
    if (!peek_token_is(p, TOK_RBRACE) && !peek_token_is(p, TOK_EOF)) {
        next_token(p);
        values[count++] = parse_expression(p, PREC_LOWEST);

        while (peek_token_is(p, TOK_COMMA)) {
            next_token(p); /* skip comma */
            next_token(p);
            if (count >= cap) {
                cap *= 2;
                AstNode **new_vals = arena_alloc(p->arena, sizeof(AstNode *) * cap);
                memcpy(new_vals, values, sizeof(AstNode *) * count);
                values = new_vals;
            }
            values[count++] = parse_expression(p, PREC_LOWEST);
        }
    }

    node->data.return_stmt.values = values;
    node->data.return_stmt.count = count;
    return node;
}

static AstNode *parse_block_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_BLOCK_STMT, p->cur_token);
    node->data.block.count = 0;
    node->data.block.cap = 8;
    node->data.block.stmts = arena_alloc(p->arena, sizeof(AstNode *) * node->data.block.cap);

    next_token(p); /* skip { */

    while (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
        AstNode *stmt = parse_statement(p);
        if (stmt) {
            if (node->data.block.count >= node->data.block.cap) {
                node->data.block.cap *= 2;
                AstNode **new_stmts = arena_alloc(p->arena, sizeof(AstNode *) * node->data.block.cap);
                memcpy(new_stmts, node->data.block.stmts, sizeof(AstNode *) * node->data.block.count);
                node->data.block.stmts = new_stmts;
            }
            node->data.block.stmts[node->data.block.count++] = stmt;
        } else {
            /* Error recovery: skip to next statement boundary */
            synchronize(p);
            if (cur_token_is(p, TOK_RBRACE)) break;
            continue;
        }
        next_token(p);
    }

    return node;
}

static AstNode *parse_func_declaration(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FUNC_DECL, p->cur_token);

    if (is_keyword_token(p->peek_token.type)) {
        char msg[EZ_MSG_BUF_SIZE];
        snprintf(msg, sizeof(msg),
            "'%s' is a reserved keyword and cannot be used as a function name",
            p->peek_token.literal);
        diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, msg),
            p->file, p->peek_token.line, p->peek_token.column, 0);
        synchronize(p);
        return NULL;
    }
    if (!expect_peek(p, TOK_IDENT)) return NULL;
    node->data.func_decl.name = p->cur_token.literal;

    /* Parameters */
    if (!expect_peek(p, TOK_LPAREN)) return NULL;

    int param_cap = 4;
    node->data.func_decl.param_count = 0;
    node->data.func_decl.params = arena_alloc(p->arena, sizeof(Param) * param_cap);

    if (!peek_token_is(p, TOK_RPAREN)) {
        next_token(p);
        do {
            if (node->data.func_decl.param_count >= param_cap) {
                param_cap *= 2;
                Param *new_params = arena_alloc(p->arena, sizeof(Param) * param_cap);
                memcpy(new_params, node->data.func_decl.params,
                    sizeof(Param) * node->data.func_decl.param_count);
                node->data.func_decl.params = new_params;
            }

            Param *param = &node->data.func_decl.params[node->data.func_decl.param_count];
            memset(param, 0, sizeof(Param));

            /* Check for mutable parameter (&) */
            if (cur_token_is(p, TOK_AMPERSAND)) {
                param->mutable = true;
                next_token(p);
            }

            param->name = p->cur_token.literal;

            /* Check for reserved names as parameters */
            if (p->cur_token.type != TOK_IDENT && p->cur_token.type != TOK_BLANK) {
                /* Keyword used as parameter name */
                char buf[EZ_MSG_BUF_SIZE];
                snprintf(buf, sizeof(buf),
                    "'%s' is a keyword and cannot be used as a parameter name",
                    param->name);
                diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, buf),
                    p->file, p->cur_token.line, p->cur_token.column, 0);
            } else if (p->cur_token.type == TOK_IDENT) {
                /* Check for builtin function/type names */
                static const char *reserved[] = {
                    /* types */
                    "int", "uint", "float", "string", "bool", "char", "byte", "void",
                    "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64",
                    "f32", "f64", "i128", "u128", "i256", "u256",
                    /* builtin functions */
                    "println", "print", "eprintln", "eprint", "input",
                    "len", "type_of", "size_of", "copy", "ref", "addr", "error",
                    "exit", "panic", "assert", "sleep_s", "sleep_ms", "sleep_ns",
                    NULL
                };
                for (int ri = 0; reserved[ri]; ri++) {
                    if (strcmp(param->name, reserved[ri]) == 0) {
                        char buf[EZ_MSG_BUF_SIZE];
                        snprintf(buf, sizeof(buf),
                            "'%s' is a built-in name and cannot be used as a parameter name",
                            param->name);
                        diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, buf),
                            p->file, p->cur_token.line, p->cur_token.column, 0);
                        break;
                    }
                }
            }

            /* Type name follows (unless next param or closing paren) */
            if (peek_token_is(p, TOK_IDENT) || peek_token_is(p, TOK_CARET) ||
                peek_token_is(p, TOK_LBRACKET) || peek_token_is(p, TOK_QUESTION)) {
                next_token(p);
                param->type_name = parse_complex_type(p);
                if (!param->type_name) return NULL;
            } else if (peek_token_is(p, TOK_AMPERSAND)) {
                /* Common mistake: `name &type` instead of `&name type`.
                 * Without this, the loop has no token to consume and
                 * spins until killed externally (#bug-report). */
                diag_error_codef(p->diag, "E3069", p->file, p->peek_token.line, p->peek_token.column, 0, param->name, "type");
                return NULL;
            }

            /* Check for default value: param type = expr */
            if (peek_token_is(p, TOK_ASSIGN)) {
                next_token(p); /* skip = */
                next_token(p);
                param->default_value = parse_expression(p, PREC_LOWEST);
            }

            node->data.func_decl.param_count++;

            if (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma */
                next_token(p);
            } else if (!peek_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_RPAREN) &&
                       !cur_token_is(p, TOK_EOF)) {
                /* Forward-progress guard: any unexpected token between
                 * params that isn't ',' or ')' would otherwise loop
                 * forever. Surface it as a parse error and bail. */
                char buf[EZ_MSG_BUF_SIZE];
                snprintf(buf, sizeof(buf),
                    "unexpected token '%s' in parameter list; expected ',' or ')'",
                    p->peek_token.literal ? p->peek_token.literal : "?");
                diag_error_msg(p->diag, "E2001", arena_strdup(p->arena, buf),
                    p->file, p->peek_token.line, p->peek_token.column, 0);
                return NULL;
            }
        } while (!cur_token_is(p, TOK_RPAREN) && !peek_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF));
    }

    if (!expect_peek(p, TOK_RPAREN)) return NULL;

    /* Backfill grouped param types (a, b int → both get int) and check for missing types */
    for (int i = node->data.func_decl.param_count - 1; i >= 0; i--) {
        Param *p_i = &node->data.func_decl.params[i];
        if (!p_i->type_name && i + 1 < node->data.func_decl.param_count) {
            p_i->type_name = node->data.func_decl.params[i + 1].type_name;
        }
        if (!p_i->type_name && !p_i->default_value) {
            char buf[EZ_MSG_BUF_SIZE];
            snprintf(buf, sizeof(buf),
                "parameter '%s' is missing a type; every parameter must have a type (e.g., %s int)",
                p_i->name, p_i->name);
            diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, buf),
                p->file, node->token.line, node->token.column, 0);
        }
    }

    /* Return type(s) */
    node->data.func_decl.return_type_count = 0;
    node->data.func_decl.return_types = NULL;
    node->data.func_decl.return_names = NULL;

    if (peek_token_is(p, TOK_ARROW)) {
        next_token(p); /* skip -> */
        next_token(p);

        /* E2002: missing return type after -> */
        if (cur_token_is(p, TOK_LBRACE)) {
            diag_error_msg(p->diag, "E2002",
                arena_strdup(p->arena, "expected return type after '->', got '{'; either specify a type or remove the '->'"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            /* Parse body to avoid cascading errors */
            node->data.func_decl.body = parse_block_statement(p);
            return node;
        }

        /* E2079: reject 'nil' as a return type. For a function that
         * returns nothing, the user should omit the '-> ...' clause. */
        if (cur_token_is(p, TOK_NIL)) {
            diag_error_code(p->diag, "E2079",
                p->file, p->cur_token.line, p->cur_token.column, 0);
            /* Skip the nil and parse the body so the error doesn't
             * cascade into a codegen crash on an unknown C type. */
            next_token(p);
            if (!cur_token_is(p, TOK_LBRACE)) {
                /* Malformed; bail with what we have. */
                return node;
            }
            node->data.func_decl.body = parse_block_statement(p);
            return node;
        }

        int ret_cap = 16;
        node->data.func_decl.return_types = arena_alloc(p->arena, sizeof(const char *) * ret_cap);
        node->data.func_decl.return_names = arena_alloc(p->arena, sizeof(const char *) * ret_cap);
        memset(node->data.func_decl.return_names, 0, sizeof(const char *) * ret_cap);

        if (cur_token_is(p, TOK_LPAREN)) {
            /* Multiple/named return types:
             *   -> (int, string)        plain types
             *   -> (x int, y int)       named returns
             *   -> (x, y int)           shared type
             *
             * Disambiguation: if the identifier is a known type name,
             * it's a plain type list, not names.
             */
            next_token(p);
            while (!cur_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF)) {
                /* Check if current ident is a type name (not a variable name) */
                bool is_type = false;
                if (cur_token_is(p, TOK_IDENT)) {
                    const char *lit = p->cur_token.literal;
                    is_type = (strcmp(lit, "int") == 0 || strcmp(lit, "uint") == 0 ||
                        strcmp(lit, "i8") == 0 || strcmp(lit, "i16") == 0 ||
                        strcmp(lit, "i32") == 0 || strcmp(lit, "i64") == 0 ||
                        strcmp(lit, "i128") == 0 || strcmp(lit, "i256") == 0 ||
                        strcmp(lit, "u8") == 0 || strcmp(lit, "u16") == 0 ||
                        strcmp(lit, "u32") == 0 || strcmp(lit, "u64") == 0 ||
                        strcmp(lit, "u128") == 0 || strcmp(lit, "u256") == 0 ||
                        strcmp(lit, "float") == 0 || strcmp(lit, "f32") == 0 ||
                        strcmp(lit, "f64") == 0 || strcmp(lit, "string") == 0 ||
                        strcmp(lit, "bool") == 0 || strcmp(lit, "char") == 0 ||
                        strcmp(lit, "byte") == 0 ||
                        (strcmp(lit, "map") == 0 && peek_token_is(p, TOK_LBRACKET)) ||
                        (strcmp(lit, "func") == 0 && peek_token_is(p, TOK_LPAREN)) ||
                        parser_is_struct_name(p, lit) ||
                        (lit[0] >= 'A' && lit[0] <= 'Z')); /* struct/enum types (fallback for imports) */
                }

                /* map[K:V] and func(...) are complex types, not named returns */
                bool is_complex_type_start = is_type && cur_token_is(p, TOK_IDENT) &&
                    ((strcmp(p->cur_token.literal, "map") == 0 && peek_token_is(p, TOK_LBRACKET)) ||
                     (strcmp(p->cur_token.literal, "func") == 0 && peek_token_is(p, TOK_LPAREN)));

                if (cur_token_is(p, TOK_IDENT) && !is_complex_type_start &&
                    (peek_token_is(p, TOK_IDENT) || peek_token_is(p, TOK_QUESTION) ||
                     peek_token_is(p, TOK_LBRACKET) || peek_token_is(p, TOK_CARET)) &&
                    (!is_type || peek_token_is(p, TOK_IDENT) ||
                     peek_token_is(p, TOK_CARET) || peek_token_is(p, TOK_LBRACKET) ||
                     peek_token_is(p, TOK_QUESTION))) {
                    /* Named return: name type; store both (: accept
                     * TOK_QUESTION, TOK_LBRACKET, TOK_CARET as type-start
                     * tokens so `(first ?, items [int], ptr ^T)` work) */
                    const char *ret_name = p->cur_token.literal;
                    next_token(p);
                    int idx = node->data.func_decl.return_type_count;
                    if (idx >= ret_cap) {
                        diag_error_code(p->diag, "E2060", p->file, p->cur_token.line, p->cur_token.column, 0);
                        return NULL;
                    }
                    node->data.func_decl.return_names[idx] = ret_name;
                    node->data.func_decl.return_types[idx] = parse_complex_type(p);
                    node->data.func_decl.return_type_count++;
                } else if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_COMMA) && !is_type) {
                    /* Shared type: (x, y int); collect names, assign same type */
                    const char *names[MAX_SHARED_RETURNS];
                    int shared = 0;
                    names[shared++] = p->cur_token.literal;
                    while (peek_token_is(p, TOK_COMMA)) {
                        next_token(p); /* skip comma */
                        next_token(p); /* next name */
                        names[shared++] = p->cur_token.literal;
                        if (shared >= MAX_SHARED_RETURNS) break;
                        if (!peek_token_is(p, TOK_COMMA)) break;
                    }
                    /* cur is last name, peek should be the shared type */
                    if (peek_token_is(p, TOK_IDENT)) {
                        next_token(p);
                        for (int s = 0; s < shared; s++) {
                            int idx = node->data.func_decl.return_type_count;
                            if (idx >= ret_cap) {
                                diag_error_code(p->diag, "E2060", p->file, p->cur_token.line, p->cur_token.column, 0);
                                return NULL;
                            }
                            node->data.func_decl.return_names[idx] = names[s];
                            node->data.func_decl.return_types[idx] = read_type_name(p);
                            node->data.func_decl.return_type_count++;
                        }
                    }
                } else {
                    /* Plain type (no name) — use parse_complex_type to
                     * handle array, map, and pointer return types like
                     * [string], map[K:V], ^T, not just simple idents. */
                    int idx = node->data.func_decl.return_type_count;
                    if (idx >= ret_cap) {
                        diag_error_code(p->diag, "E2060", p->file, p->cur_token.line, p->cur_token.column, 0);
                        return NULL;
                    }
                    node->data.func_decl.return_names[idx] = NULL;
                    node->data.func_decl.return_types[idx] = parse_complex_type(p);
                    node->data.func_decl.return_type_count++;
                }
                if (peek_token_is(p, TOK_COMMA)) {
                    next_token(p);
                }
                next_token(p);
            }
        } else {
            /* Single return type (array, pointer, map, or plain) */
            node->data.func_decl.return_types[0] = parse_complex_type(p);
            if (!node->data.func_decl.return_types[0]) return NULL;
            node->data.func_decl.return_type_count = 1;

            /* E2081: catch `-> Foo^` — '^' after a type name is a dereference
             * operator, not a type modifier; the correct form is `-> ^Foo`. */
            if (peek_token_is(p, TOK_CARET)) {
                const char *type_name = node->data.func_decl.return_types[0];
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "'^' is a dereference operator, not a type modifier; "
                    "for a pointer return type write '^%s', not '%s^'",
                    type_name, type_name);
                next_token(p); /* consume the '^' so we can point at it */
                diag_error_msg(p->diag, "E2081", arena_strdup(p->arena, msg),
                    p->file, p->cur_token.line, p->cur_token.column, 0);
                /* Recover: parse the body to avoid cascading errors. */
                if (peek_token_is(p, TOK_LBRACE)) {
                    next_token(p);
                    node->data.func_decl.body = parse_block_statement(p);
                }
                return node;
            }
        }
    }

    /* Body */
    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.func_decl.body = parse_block_statement(p);

    return node;
}

static AstNode *parse_import_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_IMPORT_STMT, p->cur_token);

    int cap = 4;
    node->data.import_stmt.count = 0;
    node->data.import_stmt.items = arena_alloc(p->arena, sizeof(ImportItem) * cap);
    node->data.import_stmt.auto_use = false;

    do {
        next_token(p);
        ImportItem *item = &node->data.import_stmt.items[node->data.import_stmt.count];
        memset(item, 0, sizeof(ImportItem));

        if (cur_token_is(p, TOK_IDENT) && strcmp(p->cur_token.literal, "and") == 0) {
            /* import and use syntax; consume 'and' then 'use' */
            node->data.import_stmt.auto_use = true;
            next_token(p); /* consume 'use' */
            next_token(p); /* advance to alias or @module */
        }

        /* Check for C interop import: import c"header.h" */
        if (cur_token_is(p, TOK_IDENT) &&
            strcmp(p->cur_token.literal, "c") == 0 &&
            peek_token_is(p, TOK_STRING)) {
            next_token(p); /* consume 'c', now on string */
            item->is_c_import = true;
            item->is_stdlib = false;
            item->path = p->cur_token.literal;
            item->alias = "c";
            item->module = "c";
            /* Validate path: only [A-Za-z0-9./_+-] permitted to prevent injection */
            for (const char *q = item->path; *q; q++) {
                unsigned char c = (unsigned char)*q;
                bool ok = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                          (c >= '0' && c <= '9') ||
                          c == '/' || c == '.' || c == '_' || c == '-' || c == '+';
                if (!ok) {
                    diag_error(p->diag, "E2080",
                        arena_strdup(p->arena, "invalid character in C header path; only [A-Za-z0-9./_+-] are permitted"),
                        p->file, p->cur_token.line, p->cur_token.column, 0);
                    break;
                }
            }
            goto import_item_done;
        }

        /* Check for alias: identifier followed by @ or string */
        if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_AT)) {
            if (strcmp(p->cur_token.literal, "c") == 0) {
                diag_error_msg(p->diag, "E2002",
                    strdup("'c' is reserved for C interop; choose a different alias"),
                    p->file, p->cur_token.line, p->cur_token.column, 0);
            }
            item->alias = p->cur_token.literal;
            next_token(p); /* consume alias, now on @ */
        } else if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_STRING)) {
            item->alias = p->cur_token.literal;
            next_token(p); /* consume alias, now on string */
        }

        if (cur_token_is(p, TOK_AT)) {
            item->is_stdlib = true;
            next_token(p);
            item->module = p->cur_token.literal;
            if (!item->alias) item->alias = p->cur_token.literal;
        } else if (cur_token_is(p, TOK_STRING)) {
            item->is_stdlib = false;
            item->path = p->cur_token.literal;
            /* Derive module name from filename/directory if no alias */
            if (!item->alias) {
                const char *slash = strrchr(item->path, '/');
                const char *base = slash ? slash + 1 : item->path;
                size_t blen = strlen(base);
                if (blen > 3 && strcmp(base + blen - 3, ".ez") == 0) {
                    /* Strip .ez extension: "helpers.ez" → "helpers" */
                    char *mod = arena_alloc(p->arena, blen - 2);
                    memcpy(mod, base, blen - 3);
                    mod[blen - 3] = '\0';
                    item->alias = mod;
                    item->module = mod;
                } else if (blen > 0) {
                    /* No .ez extension: use last path component as module name */
                    char *mod = arena_alloc(p->arena, blen + 1);
                    memcpy(mod, base, blen);
                    mod[blen] = '\0';
                    item->alias = mod;
                    item->module = mod;
                }
            }
            /* Reject 'c' as a module name; reserved for C interop */
            if (item->alias && strcmp(item->alias, "c") == 0) {
                diag_error_msg(p->diag, "E2002",
                    strdup("'c' is reserved for C interop; rename the file or use an alias (e.g., import myc\"./c.ez\")"),
                    p->file, p->cur_token.line, p->cur_token.column, 0);
            }
        } else if (cur_token_is(p, TOK_IDENT)) {
            char buf[EZ_MSG_BUF_SIZE];
            snprintf(buf, sizeof(buf),
                "expected @module or \"path\" after import, got '%s'",
                p->cur_token.literal);
            diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, buf),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            return node;
        }

        import_item_done:
        node->data.import_stmt.count++;

        if (node->data.import_stmt.count >= cap) {
            cap *= 2;
            ImportItem *new_items = arena_alloc(p->arena, sizeof(ImportItem) * cap);
            memcpy(new_items, node->data.import_stmt.items,
                sizeof(ImportItem) * node->data.import_stmt.count);
            node->data.import_stmt.items = new_items;
        }
    } while (peek_token_is(p, TOK_COMMA) && (next_token(p), 1));

    return node;
}

static AstNode *parse_using_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_USING_STMT, p->cur_token);

    int cap = 4;
    node->data.using_stmt.count = 0;
    node->data.using_stmt.modules = arena_alloc(p->arena, sizeof(const char *) * cap);

    do {
        next_token(p);
        node->data.using_stmt.modules[node->data.using_stmt.count++] = p->cur_token.literal;
    } while (peek_token_is(p, TOK_COMMA) && (next_token(p), 1));

    return node;
}

static AstNode *parse_if_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_IF_STMT, p->cur_token);

    next_token(p);
    node->data.if_stmt.condition = parse_expression(p, PREC_LOWEST);

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.if_stmt.consequence = parse_block_statement(p);

    node->data.if_stmt.alternative = NULL;

    if (peek_token_is(p, TOK_OR_KW)) {
        next_token(p); /* skip 'or' */
        /* 'or' acts like 'else if' */
        node->data.if_stmt.alternative = parse_if_statement(p);
    } else if (peek_token_is(p, TOK_OTHERWISE)) {
        next_token(p); /* skip 'otherwise' */
        if (!expect_peek(p, TOK_LBRACE)) return NULL;
        node->data.if_stmt.alternative = parse_block_statement(p);
    }

    return node;
}

static AstNode *parse_struct_declaration(Parser *p) {
    /* cur_token is the struct name (IDENT), already consumed by caller */
    AstNode *node = ast_alloc(p->arena, NODE_STRUCT_DECL, p->cur_token);
    node->data.struct_decl.name = p->cur_token.literal;

    next_token(p); /* skip 'struct' keyword */
    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    int brace_line = p->cur_token.line;
    next_token(p); /* skip { */

    /* Reject inline struct declarations; fields must be on separate lines */
    if (p->cur_token.line == brace_line && !cur_token_is(p, TOK_RBRACE)) {
        diag_error_msg(p->diag, "E2002",
            strdup("struct fields must be on separate lines; inline struct declarations are not allowed"),
            p->file, p->cur_token.line, p->cur_token.column, 0);
    }

    int field_cap = 8;
    int func_cap = 4;
    node->data.struct_decl.field_count = 0;
    node->data.struct_decl.fields = arena_alloc(p->arena, sizeof(StructField) * field_cap);
    node->data.struct_decl.func_count = 0;
    node->data.struct_decl.funcs = arena_alloc(p->arena, sizeof(StructFunc) * func_cap);

    while (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
        /* : skip #doc attributes on struct functions. Consume
         * the attribute + any parenthesised args, then continue so
         * the next token (do/private do) is handled normally. */
        if (cur_token_is(p, TOK_DOC)) {
            if (peek_token_is(p, TOK_LPAREN)) {
                next_token(p);
                while (!cur_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF))
                    next_token(p);
            }
            next_token(p);
            continue;
        }
        /* Check for struct-namespaced function: do func() or private do func() */
        if (cur_token_is(p, TOK_DO)) {
            AstNode *fn = parse_func_declaration(p);
            if (fn) {
                if (node->data.struct_decl.func_count >= func_cap) {
                    func_cap *= 2;
                    StructFunc *new_funcs = arena_alloc(p->arena, sizeof(StructFunc) * func_cap);
                    memcpy(new_funcs, node->data.struct_decl.funcs,
                        sizeof(StructFunc) * node->data.struct_decl.func_count);
                    node->data.struct_decl.funcs = new_funcs;
                }
                node->data.struct_decl.funcs[node->data.struct_decl.func_count++].func_decl = fn;
            }
            next_token(p);
            continue;
        }
        if (cur_token_is(p, TOK_PRIVATE) && peek_token_is(p, TOK_DO)) {
            next_token(p); /* consume 'private' */
            AstNode *fn = parse_func_declaration(p);
            if (fn) {
                fn->data.func_decl.is_private = true;
                if (node->data.struct_decl.func_count >= func_cap) {
                    func_cap *= 2;
                    StructFunc *new_funcs = arena_alloc(p->arena, sizeof(StructFunc) * func_cap);
                    memcpy(new_funcs, node->data.struct_decl.funcs,
                        sizeof(StructFunc) * node->data.struct_decl.func_count);
                    node->data.struct_decl.funcs = new_funcs;
                }
                int idx = node->data.struct_decl.func_count++;
                node->data.struct_decl.funcs[idx].func_decl = fn;
                node->data.struct_decl.funcs[idx].is_private = true;
            }
            next_token(p);
            continue;
        }

        /* E2058: nested struct/enum declaration */
        if (cur_token_is(p, TOK_CONST)) {
            diag_error_codef(p->diag, "E2058",
                p->file, p->cur_token.line, p->cur_token.column, 0,
                "struct", node->data.struct_decl.name);
            /* Skip to the end of the nested declaration to avoid cascading errors */
            int depth = 0;
            while (!cur_token_is(p, TOK_EOF)) {
                if (cur_token_is(p, TOK_LBRACE)) depth++;
                if (cur_token_is(p, TOK_RBRACE)) {
                    if (depth <= 1) { next_token(p); break; }
                    depth--;
                }
                next_token(p);
            }
            continue;
        }

        if (node->data.struct_decl.field_count >= field_cap) {
            field_cap *= 2;
            StructField *new_fields = arena_alloc(p->arena, sizeof(StructField) * field_cap);
            memcpy(new_fields, node->data.struct_decl.fields,
                sizeof(StructField) * node->data.struct_decl.field_count);
            node->data.struct_decl.fields = new_fields;
        }

        /* E2070 ( follow-up): wildcard `?` in field-name position
         * used to slip past the struct-field guard; the existing check
         * further down only inspects the type slot; and embed '?' in
         * the generated C struct identifier, where clang rejected it
         * with a raw C error. Catch it here before reading the name. */
        if (cur_token_is(p, TOK_QUESTION)) {
            diag_error_msg(p->diag, "E2070",
                arena_strdup(p->arena,
                    "wildcard type '?' is not allowed as a struct field name; only in function parameter and return types"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            next_token(p); /* skip the '?' */
            /* Skip the trailing type token (if any) so we don't cascade. */
            if (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
                parse_complex_type(p);
                next_token(p);
            }
            continue;
        }

        /* Collect one or more comma-separated field names, then read the
         * shared type and backfill (mirrors the parameter grouping logic).
         * Example: `x, y, z float` → three fields, all typed float.       */
        int group_start = node->data.struct_decl.field_count;
        for (;;) {
            if (node->data.struct_decl.field_count >= field_cap) {
                field_cap *= 2;
                StructField *new_f = arena_alloc(p->arena, sizeof(StructField) * field_cap);
                memcpy(new_f, node->data.struct_decl.fields,
                    sizeof(StructField) * node->data.struct_decl.field_count);
                node->data.struct_decl.fields = new_f;
            }
            StructField *field = &node->data.struct_decl.fields[node->data.struct_decl.field_count];
            field->name = p->cur_token.literal;
            field->type_name = NULL;
            node->data.struct_decl.field_count++;
            next_token(p);
            if (cur_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma, loop for next name */
            } else {
                break;
            }
        }
        /* Current token is now the type; parse it and backfill all names in this group */
        const char *type_name = parse_complex_type(p);
        if (!type_name) return NULL;
        /* : `?` is now allowed in struct fields for generic structs.
         * The E2070 rejection was removed; the typechecker validates
         * binding consistency at each struct literal usage site. */
        for (int i = group_start; i < node->data.struct_decl.field_count; i++) {
            node->data.struct_decl.fields[i].type_name = type_name;
        }
        next_token(p);

        /* Skip optional trailing comma after a field type */
        if (cur_token_is(p, TOK_COMMA)) next_token(p);

        /* Reject semicolons */
        if (cur_token_is(p, TOK_SEMICOLON)) {
            diag_error_msg(p->diag, "E2069",
                strdup("semicolons are not used; put each struct field on its own line"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            next_token(p);
        }
    }

    return node;
}

static AstNode *parse_enum_declaration(Parser *p) {
    /* cur_token is the enum name (IDENT), already consumed by caller */
    AstNode *node = ast_alloc(p->arena, NODE_ENUM_DECL, p->cur_token);
    node->data.enum_decl.name = p->cur_token.literal;
    node->data.enum_decl.is_flags = false;

    next_token(p); /* skip 'enum' keyword */
    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    int enum_brace_line = p->cur_token.line;
    next_token(p); /* skip { */

    /* Reject inline enum declarations; variants must be on separate lines */
    if (p->cur_token.line == enum_brace_line && !cur_token_is(p, TOK_RBRACE)) {
        diag_error_msg(p->diag, "E2002",
            strdup("enum variants must be on separate lines; inline enum declarations are not allowed"),
            p->file, p->cur_token.line, p->cur_token.column, 0);
    }

    int val_cap = 8;
    node->data.enum_decl.value_count = 0;
    node->data.enum_decl.values = arena_alloc(p->arena, sizeof(EnumVal) * val_cap);

    while (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
        if (node->data.enum_decl.value_count >= val_cap) {
            val_cap *= 2;
            EnumVal *new_vals = arena_alloc(p->arena, sizeof(EnumVal) * val_cap);
            memcpy(new_vals, node->data.enum_decl.values,
                sizeof(EnumVal) * node->data.enum_decl.value_count);
            node->data.enum_decl.values = new_vals;
        }

        /* E2058: nested struct/enum declaration */
        if (cur_token_is(p, TOK_CONST)) {
            diag_error_codef(p->diag, "E2058",
                p->file, p->cur_token.line, p->cur_token.column, 0,
                "enum", node->data.enum_decl.name);
            /* Skip to the end of the nested declaration to avoid cascading errors */
            int depth = 0;
            while (!cur_token_is(p, TOK_EOF)) {
                if (cur_token_is(p, TOK_LBRACE)) depth++;
                if (cur_token_is(p, TOK_RBRACE)) {
                    if (depth <= 1) { next_token(p); break; }
                    depth--;
                }
                next_token(p);
            }
            continue;
        }

        /* E2070 ( follow-up): wildcard `?` in variant-name position
         * used to slip past the parser and embed '?' in the generated C
         * enum identifier, where clang rejected it with a raw C error.
         * Catch it here before reading the variant name. */
        if (cur_token_is(p, TOK_QUESTION)) {
            diag_error_msg(p->diag, "E2070",
                arena_strdup(p->arena,
                    "wildcard type '?' is not allowed in enum declarations; only in function parameter and return types"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            next_token(p); /* skip the '?' */
            /* Skip an optional trailing ',' so we don't cascade into the
             * next variant with a stale cur_token. */
            if (cur_token_is(p, TOK_COMMA)) next_token(p);
            continue;
        }

        EnumVal *ev = &node->data.enum_decl.values[node->data.enum_decl.value_count];
        ev->name = p->cur_token.literal;
        ev->value = NULL;

        /* Check for explicit value: VALUE = expr */
        if (peek_token_is(p, TOK_ASSIGN)) {
            next_token(p); /* skip = */
            next_token(p);
            ev->value = parse_expression(p, PREC_LOWEST);
        }

        node->data.enum_decl.value_count++;
        /* Skip optional trailing comma */
        if (peek_token_is(p, TOK_COMMA)) {
            next_token(p);
        }
        next_token(p);

        /* Reject semicolons */
        if (cur_token_is(p, TOK_SEMICOLON)) {
            diag_error_msg(p->diag, "E2069",
                strdup("semicolons are not used; put each enum variant on its own line"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
            next_token(p);
        }
    }

    return node;
}

/* Parse struct literal: StructName{field: value, ...} */
static AstNode *parse_struct_literal(Parser *p, const char *name) {
    AstNode *node = ast_alloc(p->arena, NODE_STRUCT_VALUE, p->cur_token);
    node->data.struct_value.name = name;

    int cap = 8;
    node->data.struct_value.count = 0;
    node->data.struct_value.field_names = arena_alloc(p->arena, sizeof(const char *) * cap);
    node->data.struct_value.field_values = arena_alloc(p->arena, sizeof(AstNode *) * cap);

    next_token(p); /* skip { */

    while (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
        if (node->data.struct_value.count >= cap) {
            cap *= 2;
            const char **new_names = arena_alloc(p->arena, sizeof(const char *) * cap);
            AstNode **new_vals = arena_alloc(p->arena, sizeof(AstNode *) * cap);
            memcpy(new_names, node->data.struct_value.field_names,
                sizeof(const char *) * node->data.struct_value.count);
            memcpy(new_vals, node->data.struct_value.field_values,
                sizeof(AstNode *) * node->data.struct_value.count);
            node->data.struct_value.field_names = new_names;
            node->data.struct_value.field_values = new_vals;
        }

        node->data.struct_value.field_names[node->data.struct_value.count] = p->cur_token.literal;

        if (!expect_peek(p, TOK_COLON)) return NULL;
        next_token(p);
        node->data.struct_value.field_values[node->data.struct_value.count] =
            parse_expression(p, PREC_LOWEST);
        node->data.struct_value.count++;

        if (peek_token_is(p, TOK_COMMA)) {
            next_token(p);
        }
        next_token(p);
    }

    return node;
}

static AstNode *parse_ensure_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_ENSURE_STMT, p->cur_token);
    next_token(p);
    node->data.ensure_stmt.expr = parse_expression(p, PREC_LOWEST);
    return node;
}

static AstNode *parse_for_statement(Parser *p) {
    Token for_tok = p->cur_token;

    /* Optional parentheses: for (i in range(...)) */
    bool has_parens = peek_token_is(p, TOK_LPAREN);
    if (has_parens) next_token(p);

    if (peek_token_is(p, TOK_IDENT)) {
        next_token(p);  /* advance: cur_token = IDENT */
        if (peek_token_is(p, TOK_IN)) {
            /* --- iteration form: for x in collection { } --- */
            AstNode *node = ast_alloc(p->arena, NODE_FOR_STMT, for_tok);
            node->data.for_stmt.var_name = p->cur_token.literal;
            node->data.for_stmt.var_type = NULL;
            next_token(p);  /* consume IN */
            next_token(p);  /* advance to iterable start */
            node->data.for_stmt.iterable = parse_expression(p, PREC_LOWEST);
            if (has_parens && peek_token_is(p, TOK_RPAREN)) next_token(p);
            if (!expect_peek(p, TOK_LBRACE)) return NULL;
            node->data.for_stmt.body = parse_block_statement(p);
            return node;
        }
        /* else: cur_token = IDENT, fall through to while-style */
    } else {
        next_token(p);  /* advance to condition start token */
    }

    /* --- while-style: for condition { } --- */
    AstNode *wnode = ast_alloc(p->arena, NODE_WHILE_STMT, for_tok);
    wnode->data.while_stmt.condition = parse_expression(p, PREC_LOWEST);
    if (has_parens && peek_token_is(p, TOK_RPAREN)) next_token(p);
    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    wnode->data.while_stmt.body = parse_block_statement(p);
    return wnode;
}

static AstNode *parse_for_each_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FOR_EACH_STMT, p->cur_token);

    next_token(p);

    /* Optional parentheses: for_each (val in arr) {} */
    bool has_paren = cur_token_is(p, TOK_LPAREN);
    if (has_paren) next_token(p);

    node->data.for_each.index_name = NULL;
    node->data.for_each.var_name = p->cur_token.literal;

    /* Check for index, value pattern: for_each i, item in collection */
    if (peek_token_is(p, TOK_COMMA)) {
        node->data.for_each.index_name = node->data.for_each.var_name;
        next_token(p); /* skip comma */
        next_token(p);
        node->data.for_each.var_name = p->cur_token.literal;
    }

    if (!expect_peek(p, TOK_IN)) return NULL;

    next_token(p);
    /* Parse collection carefully; prevent struct literal parser from consuming
     * the block-opening { after identifiers or module.Name expressions */
    if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_LBRACE)) {
        /* Bare identifier followed by {; parse as label only */
        node->data.for_each.collection = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
        node->data.for_each.collection->data.label.value = p->cur_token.literal;
    } else if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_DOT)) {
        /* Module-qualified name: mod.Name; parse as member expr manually,
         * don't use parse_expression which would trigger struct literal parsing */
        AstNode *obj = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
        obj->data.label.value = p->cur_token.literal;
        next_token(p); /* consume . */
        next_token(p); /* move to member */
        AstNode *member = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
        member->data.member.object = obj;
        member->data.member.member = p->cur_token.literal;
        node->data.for_each.collection = member;
    } else {
        node->data.for_each.collection = parse_expression(p, PREC_LOWEST);
    }

    if (has_paren) {
        if (!expect_peek(p, TOK_RPAREN)) return NULL;
    }

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.for_each.body = parse_block_statement(p);

    return node;
}

static AstNode *parse_while_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_WHILE_STMT, p->cur_token);

    next_token(p);
    node->data.while_stmt.condition = parse_expression(p, PREC_LOWEST);

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.while_stmt.body = parse_block_statement(p);

    return node;
}

static AstNode *parse_loop_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_LOOP_STMT, p->cur_token);

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.loop_stmt.body = parse_block_statement(p);

    return node;
}

static AstNode *parse_when_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_WHEN_STMT, p->cur_token);

    next_token(p);
    p->no_struct_literal = true;
    node->data.when_stmt.value = parse_expression(p, PREC_LOWEST);
    p->no_struct_literal = false;
    node->data.when_stmt.is_strict = false;
    node->data.when_stmt.default_body = NULL;

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    next_token(p); /* skip { */

    int case_cap = 8;
    node->data.when_stmt.case_count = 0;
    node->data.when_stmt.cases = arena_alloc(p->arena, sizeof(WhenCase) * case_cap);

    while (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
        if (cur_token_is(p, TOK_IS)) {
            if (node->data.when_stmt.case_count >= case_cap) {
                case_cap *= 2;
                WhenCase *new_cases = arena_alloc(p->arena, sizeof(WhenCase) * case_cap);
                memcpy(new_cases, node->data.when_stmt.cases,
                    sizeof(WhenCase) * node->data.when_stmt.case_count);
                node->data.when_stmt.cases = new_cases;
            }

            WhenCase *wc = &node->data.when_stmt.cases[node->data.when_stmt.case_count];
            memset(wc, 0, sizeof(WhenCase));

            int val_cap = 4;
            wc->value_count = 0;
            wc->values = arena_alloc(p->arena, sizeof(AstNode *) * val_cap);
            wc->is_range = false;

            /* Parse case values: is 1, 2, 3 { } */
            next_token(p);
            if (cur_token_is(p, TOK_RANGE)) {
                wc->is_range = true;
            }
            if (wc->value_count < val_cap) {
                wc->values[wc->value_count++] = parse_expression(p, PREC_LOWEST);
            }
            while (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma */
                next_token(p); /* next value */
                if (wc->value_count >= val_cap) {
                    val_cap *= 2;
                    AstNode **new_vals = arena_alloc(p->arena, sizeof(AstNode *) * val_cap);
                    memcpy(new_vals, wc->values, sizeof(AstNode *) * wc->value_count);
                    wc->values = new_vals;
                }
                if (cur_token_is(p, TOK_RANGE)) {
                    wc->is_range = true;
                }
                wc->values[wc->value_count++] = parse_expression(p, PREC_LOWEST);
            }

            if (!expect_peek(p, TOK_LBRACE)) return NULL;
            wc->body = parse_block_statement(p);
            node->data.when_stmt.case_count++;

        } else if (cur_token_is(p, TOK_DEFAULT)) {
            if (!expect_peek(p, TOK_LBRACE)) return NULL;
            node->data.when_stmt.default_body = parse_block_statement(p);
        }

        next_token(p);
    }

    /* E2059: empty when block */
    if (node->data.when_stmt.case_count == 0 && node->data.when_stmt.default_body == NULL) {
        diag_error_code(p->diag, "E2059", p->file, node->token.line, node->token.column, 0);
    }

    return node;
}

static bool is_assignment_op(TokenType t) {
    return t == TOK_ASSIGN || t == TOK_PLUS_ASSIGN || t == TOK_MINUS_ASSIGN ||
           t == TOK_ASTERISK_ASSIGN || t == TOK_SLASH_ASSIGN || t == TOK_PERCENT_ASSIGN;
}

static AstNode *parse_statement(Parser *p) {
    switch (p->cur_token.type) {
    case TOK_PRIVATE: {
        /* private do / private const / private mut; consume and set flag */
        next_token(p);
        AstNode *stmt = parse_statement(p);
        if (stmt) {
            if (stmt->kind == NODE_FUNC_DECL) {
                stmt->data.func_decl.is_private = true;
            } else if (stmt->kind == NODE_VAR_DECL) {
                stmt->data.var_decl.is_private = true;
            }
        }
        return stmt;
    }
    case TOK_MUT:
    case TOK_CONST:
        /* Check for keyword used as name: const for struct / mut for int */
        if (is_keyword_token(p->peek_token.type)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "'%s' is a reserved keyword and cannot be used as a name",
                p->peek_token.literal);
            diag_error_msg(p->diag, "E2002", arena_strdup(p->arena, msg),
                p->file, p->peek_token.line, p->peek_token.column, 0);
            synchronize(p);
            return NULL;
        }
        /* Check if this is a struct or enum declaration: const Name struct { */
        if (p->cur_token.type == TOK_CONST && peek_token_is(p, TOK_IDENT)) {
            /* Save full parser state for lookahead */
            Token saved_cur = p->cur_token;
            Token saved_peek = p->peek_token;
            int saved_pos = p->lexer->position;
            int saved_rpos = p->lexer->read_position;
            char saved_ch = p->lexer->ch;
            int saved_line = p->lexer->line;
            int saved_col = p->lexer->column;

            next_token(p); /* now on IDENT (name) */
            if (peek_token_is(p, TOK_STRUCT)) {
                return parse_struct_declaration(p);
            }
            if (peek_token_is(p, TOK_ENUM)) {
                return parse_enum_declaration(p);
            }
            /* Not struct/enum; restore full state */
            p->cur_token = saved_cur;
            p->peek_token = saved_peek;
            p->lexer->position = saved_pos;
            p->lexer->read_position = saved_rpos;
            p->lexer->ch = saved_ch;
            p->lexer->line = saved_line;
            p->lexer->column = saved_col;
        }
        return parse_var_declaration(p);
    case TOK_DO:
        return parse_func_declaration(p);
    case TOK_RETURN:
        return parse_return_statement(p);
    case TOK_MODULE:
        diag_error_code(p->diag, "E2061", p->file, p->cur_token.line, p->cur_token.column, 0);
        next_token(p); /* consume module name */
        return NULL;
    case TOK_IMPORT:
        return parse_import_statement(p);
    case TOK_USING:
        return parse_using_statement(p);
    case TOK_IF:
        return parse_if_statement(p);
    case TOK_FOR:
        return parse_for_statement(p);
    case TOK_FOR_EACH:
        return parse_for_each_statement(p);
    case TOK_AS_LONG_AS:
        return parse_while_statement(p);
    case TOK_LOOP:
        return parse_loop_statement(p);
    case TOK_BREAK:
        return ast_alloc(p->arena, NODE_BREAK_STMT, p->cur_token);
    case TOK_CONTINUE:
        return ast_alloc(p->arena, NODE_CONTINUE_STMT, p->cur_token);
    case TOK_WHEN:
        return parse_when_statement(p);
    case TOK_STRICT:
        /* #strict; applies to the next when statement */
        next_token(p);
        if (cur_token_is(p, TOK_WHEN)) {
            AstNode *ws = parse_when_statement(p);
            if (ws) ws->data.when_stmt.is_strict = true;
            return ws;
        }
        /* If not followed by when, just skip */
        return parse_statement(p);
    case TOK_FLAGS: {
        /* #flags; applies to the next enum declaration */
        next_token(p); /* skip #flags */
        AstNode *stmt = parse_statement(p);
        if (stmt && stmt->kind == NODE_ENUM_DECL) {
            stmt->data.enum_decl.is_flags = true;
        }
        return stmt;
    }
    case TOK_JSON_ATTR: {
        /* #json; applies to the next struct declaration () */
        next_token(p);
        AstNode *stmt = parse_statement(p);
        if (stmt && stmt->kind == NODE_STRUCT_DECL) {
            stmt->data.struct_decl.is_json = true;
        } else {
            diag_error_msg(p->diag, "E2002",
                strdup("#json attribute can only be applied to struct declarations"),
                p->file, p->cur_token.line, p->cur_token.column, 0);
        }
        return stmt;
    }
    case TOK_SUPPRESS:
        diag_error_msg(p->diag, "E2002",
            strdup("#suppress is no longer supported; use 'ez file.ez -q W1001' to suppress warnings from the command line"),
            p->file, p->cur_token.line, p->cur_token.column, 0);
        /* Consume the attribute and its args to avoid cascading errors */
        if (peek_token_is(p, TOK_LPAREN)) {
            next_token(p);
            while (!cur_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF)) {
                next_token(p);
            }
        }
        next_token(p);
        return parse_statement(p);
    case TOK_DOC:
        /* Skip #doc attribute tokens; consume args if present */
        if (peek_token_is(p, TOK_LPAREN)) {
            next_token(p);
            while (!cur_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF)) {
                next_token(p);
            }
        }
        next_token(p);
        return parse_statement(p);
    case TOK_ENSURE:
        return parse_ensure_statement(p);
    case TOK_BLANK:
        /* Bare throwaway: `_ = expr` discards the RHS without creating
         * a symbol. Any other use of `_` at statement position (bare
         * `_`, `_ +=`, `_, x = ...`) is invalid. */
        if (peek_token_is(p, TOK_ASSIGN)) {
            return parse_discard_statement(p);
        }
        diag_error_msg(p->diag, "E2002",
            strdup("unexpected token '_'; the throwaway '_' is only valid as the entire left-hand side of an assignment"),
            p->file, p->cur_token.line, p->cur_token.column, 0);
        synchronize(p);
        return NULL;
    default: {
        /* IDENT IDENT at statement position means the user tried to
         * declare a variable but typed the wrong leading keyword
         * (e.g. `cont myVar = 10`). No valid EZ statement has this
         * shape, so we can confidently redirect them at 'const' or
         * 'mut'. Consume the botched header and keep parsing. */
        if (cur_token_is(p, TOK_IDENT) && peek_token_is(p, TOK_IDENT)) {
            diag_error_codef(p->diag, "E2078",
                p->file, p->cur_token.line, p->cur_token.column, 0,
                p->peek_token.literal, p->peek_token.literal);
            synchronize(p);
            return NULL;
        }
        /* Could be assignment or expression statement */
        AstNode *expr = parse_expression(p, PREC_LOWEST);
        if (!expr) return NULL;

        /* Check for assignment */
        if (is_assignment_op(p->peek_token.type)) {
            next_token(p);
            AstNode *node = ast_alloc(p->arena, NODE_ASSIGN_STMT, p->cur_token);
            node->data.assign.target = expr;
            node->data.assign.op = p->cur_token.literal;
            next_token(p);
            node->data.assign.value = parse_expression(p, PREC_LOWEST);
            return node;
        }

        AstNode *node = ast_alloc(p->arena, NODE_EXPR_STMT, p->cur_token);
        node->data.expr_stmt.expr = expr;
        return node;
    }
    }
}

/* --- Public API --- */

Parser *parser_create(Arena *arena, Lexer *lexer, const char *file, DiagnosticList *diag) {
    Parser *p = arena_alloc(arena, sizeof(Parser));
    p->lexer = lexer;
    p->arena = arena;
    p->file = file;
    p->diag = diag;
    p->depth = 0;
    p->no_struct_literal = false;
    p->struct_name_count = 0;
    p->struct_name_cap = 16;
    p->struct_names = arena_alloc(arena, sizeof(const char *) * 16);

    /* Pre-scan for struct/enum names before the main parse so we can
     * disambiguate struct literals without capitalization heuristics. */
    prescan_struct_names(p);

    /* Read two tokens to fill cur and peek */
    next_token(p);
    next_token(p);

    return p;
}

AstNode *parser_parse_program(Parser *p) {
    Token tok = {TOK_EOF, "", 0, 0, NULL, false};
    AstNode *program = ast_alloc(p->arena, NODE_PROGRAM, tok);
    program->data.program.module_decl = NULL;
    program->data.program.using_stmts = NULL;
    program->data.program.using_count = 0;
    program->data.program.stmt_count = 0;
    program->data.program.stmt_cap = 16;
    program->data.program.stmts = arena_alloc(p->arena,
        sizeof(AstNode *) * program->data.program.stmt_cap);

    while (!cur_token_is(p, TOK_EOF)) {
        AstNode *stmt = parse_statement(p);
        if (stmt) {
            if (program->data.program.stmt_count >= program->data.program.stmt_cap) {
                program->data.program.stmt_cap *= 2;
                AstNode **new_stmts = arena_alloc(p->arena,
                    sizeof(AstNode *) * program->data.program.stmt_cap);
                memcpy(new_stmts, program->data.program.stmts,
                    sizeof(AstNode *) * program->data.program.stmt_count);
                program->data.program.stmts = new_stmts;
            }
            program->data.program.stmts[program->data.program.stmt_count++] = stmt;
        } else {
            /* Error recovery: skip to next statement boundary */
            synchronize(p);
            continue;
        }
        next_token(p);
    }

    return program;
}
