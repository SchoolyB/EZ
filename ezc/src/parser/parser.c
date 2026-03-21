/*
 * parser.c - Recursive descent parser for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "parser.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>

/* Operator precedence levels */
typedef enum {
    PREC_LOWEST,
    PREC_OR,            /* || */
    PREC_AND,           /* && */
    PREC_EQUALS,        /* == != */
    PREC_LESSGREATER,   /* > < >= <= */
    PREC_MEMBERSHIP,    /* in, not_in */
    PREC_SUM,           /* + - */
    PREC_PRODUCT,       /* * / % */
    PREC_POWER,         /* ** */
    PREC_PREFIX,        /* -x !x &x */
    PREC_CALL,          /* f(x) */
    PREC_INDEX,         /* a[i] a.b */
    PREC_POSTFIX,       /* x++ x-- */
} Precedence;

/* Forward declarations */
static AstNode *parse_statement(Parser *p);
static AstNode *parse_expression(Parser *p, Precedence prec);
static AstNode *parse_block_statement(Parser *p);

/* --- Helpers --- */

static void parser_error(Parser *p, const char *fmt, ...) {
    char buf[512];
    va_list args;
    va_start(args, fmt);
    vsnprintf(buf, sizeof(buf), fmt, args);
    va_end(args);

    if (p->error_count >= p->error_cap) {
        p->error_cap = p->error_cap ? p->error_cap * 2 : 16;
        p->errors = realloc(p->errors, sizeof(char *) * p->error_cap);
    }
    p->errors[p->error_count++] = arena_strdup(p->arena, buf);
}

static void next_token(Parser *p) {
    p->cur_token = p->peek_token;
    p->peek_token = lexer_next_token(p->lexer);
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
    parser_error(p, "%s:%d:%d: expected '%s', got '%s'",
        p->file, p->peek_token.line, p->peek_token.column,
        token_type_name(t), token_type_name(p->peek_token.type));
    return false;
}

static Precedence token_precedence(TokenType t) {
    switch (t) {
    case TOK_OR:             return PREC_OR;
    case TOK_AND:            return PREC_AND;
    case TOK_EQ:
    case TOK_NOT_EQ:         return PREC_EQUALS;
    case TOK_LT:
    case TOK_GT:
    case TOK_LT_EQ:
    case TOK_GT_EQ:          return PREC_LESSGREATER;
    case TOK_IN:
    case TOK_NOT_IN:         return PREC_MEMBERSHIP;
    case TOK_PLUS:
    case TOK_MINUS:          return PREC_SUM;
    case TOK_ASTERISK:
    case TOK_SLASH:
    case TOK_PERCENT:        return PREC_PRODUCT;
    case TOK_POWER:          return PREC_POWER;
    case TOK_LPAREN:         return PREC_CALL;
    case TOK_LBRACKET:       return PREC_INDEX;
    case TOK_DOT:            return PREC_INDEX;
    case TOK_INCREMENT:
    case TOK_DECREMENT:      return PREC_POSTFIX;
    default:                 return PREC_LOWEST;
    }
}

/* --- Expression Parsing --- */

static AstNode *parse_identifier(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_LABEL, p->cur_token);
    node->data.label.value = p->cur_token.literal;
    return node;
}

static AstNode *parse_int_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_INT_VALUE, p->cur_token);
    /* Parse integer, ignoring underscores */
    int64_t val = 0;
    const char *s = p->cur_token.literal;
    while (*s) {
        if (*s != '_') {
            val = val * 10 + (*s - '0');
        }
        s++;
    }
    node->data.int_value.value = val;
    node->data.int_value.literal = p->cur_token.literal;
    return node;
}

static AstNode *parse_float_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FLOAT_VALUE, p->cur_token);
    node->data.float_value.value = atof(p->cur_token.literal);
    return node;
}

static AstNode *parse_string_literal(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_STRING_VALUE, p->cur_token);
    node->data.string_value.value = p->cur_token.literal;
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
    next_token(p);
    AstNode *expr = parse_expression(p, PREC_LOWEST);
    if (!expect_peek(p, TOK_RPAREN)) return NULL;
    return expr;
}

/* Parse prefix expression (the "nud" in Pratt parsing) */
static AstNode *parse_prefix(Parser *p) {
    switch (p->cur_token.type) {
    case TOK_IDENT:     return parse_identifier(p);
    case TOK_INT:       return parse_int_literal(p);
    case TOK_FLOAT:     return parse_float_literal(p);
    case TOK_STRING:    return parse_string_literal(p);
    case TOK_TRUE:
    case TOK_FALSE:     return parse_bool_literal(p);
    case TOK_NIL:       return parse_nil_literal(p);
    case TOK_MINUS:
    case TOK_BANG:
    case TOK_AMPERSAND: return parse_prefix_expression(p);
    case TOK_LPAREN:    return parse_grouped_expression(p);
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
        parser_error(p, "%s:%d:%d: unexpected token '%s'",
            p->file, p->cur_token.line, p->cur_token.column,
            token_type_name(p->cur_token.type));
        return NULL;
    }
}

static AstNode *parse_infix_expression(Parser *p, AstNode *left) {
    AstNode *node = ast_alloc(p->arena, NODE_INFIX_EXPR, p->cur_token);
    node->data.infix.left = left;
    node->data.infix.op = p->cur_token.literal;
    Precedence prec = token_precedence(p->cur_token.type);
    next_token(p);
    node->data.infix.right = parse_expression(p, prec);
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
    AstNode *node = ast_alloc(p->arena, NODE_CALL_EXPR, p->cur_token);
    node->data.call.function = function;
    node->data.call.args = parse_expression_list(p, TOK_RPAREN, &node->data.call.arg_count);
    return node;
}

static AstNode *parse_member_expression(Parser *p, AstNode *object) {
    next_token(p); /* skip the identifier after dot */
    AstNode *node = ast_alloc(p->arena, NODE_MEMBER_EXPR, p->cur_token);
    node->data.member.object = object;
    node->data.member.member = p->cur_token.literal;
    return node;
}

static AstNode *parse_index_expression(Parser *p, AstNode *left) {
    AstNode *node = ast_alloc(p->arena, NODE_INDEX_EXPR, p->cur_token);
    node->data.index_expr.left = left;
    next_token(p);
    node->data.index_expr.index = parse_expression(p, PREC_LOWEST);
    if (!expect_peek(p, TOK_RBRACKET)) return NULL;
    return node;
}

static AstNode *parse_postfix_expression(Parser *p, AstNode *left) {
    AstNode *node = ast_alloc(p->arena, NODE_POSTFIX_EXPR, p->cur_token);
    node->data.postfix.left = left;
    node->data.postfix.op = p->cur_token.literal;
    return node;
}

/* Parse infix expression (the "led" in Pratt parsing) */
static AstNode *parse_infix(Parser *p, AstNode *left) {
    switch (p->cur_token.type) {
    case TOK_PLUS: case TOK_MINUS: case TOK_ASTERISK: case TOK_SLASH:
    case TOK_PERCENT: case TOK_POWER:
    case TOK_EQ: case TOK_NOT_EQ: case TOK_LT: case TOK_GT:
    case TOK_LT_EQ: case TOK_GT_EQ:
    case TOK_AND: case TOK_OR:
    case TOK_IN: case TOK_NOT_IN:
        return parse_infix_expression(p, left);
    case TOK_LPAREN:
        return parse_call_expression(p, left);
    case TOK_DOT:
        return parse_member_expression(p, left);
    case TOK_LBRACKET:
        return parse_index_expression(p, left);
    case TOK_INCREMENT:
    case TOK_DECREMENT:
        return parse_postfix_expression(p, left);
    default:
        return left;
    }
}

static AstNode *parse_expression(Parser *p, Precedence prec) {
    AstNode *left = parse_prefix(p);
    if (!left) return NULL;

    while (!peek_token_is(p, TOK_EOF) && prec < token_precedence(p->peek_token.type)) {
        next_token(p);
        left = parse_infix(p, left);
        if (!left) return NULL;
    }

    return left;
}

/* --- Statement Parsing --- */

static AstNode *parse_expression_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_EXPR_STMT, p->cur_token);
    node->data.expr_stmt.expr = parse_expression(p, PREC_LOWEST);
    return node;
}

static AstNode *parse_var_declaration(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_VAR_DECL, p->cur_token);
    node->data.var_decl.mutable = (p->cur_token.type == TOK_TEMP);

    if (!expect_peek(p, TOK_IDENT)) return NULL;
    node->data.var_decl.name = p->cur_token.literal;

    /* Optional type annotation */
    node->data.var_decl.type_name = NULL;
    if (peek_token_is(p, TOK_IDENT) || peek_token_is(p, TOK_LBRACKET)) {
        next_token(p);
        node->data.var_decl.type_name = p->cur_token.literal;
    }

    /* = value */
    if (peek_token_is(p, TOK_ASSIGN)) {
        next_token(p); /* skip = */
        next_token(p);
        node->data.var_decl.value = parse_expression(p, PREC_LOWEST);
    }

    return node;
}

static AstNode *parse_return_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_RETURN_STMT, p->cur_token);
    next_token(p);

    int cap = 4;
    int count = 0;
    AstNode **values = arena_alloc(p->arena, sizeof(AstNode *) * cap);

    if (!cur_token_is(p, TOK_RBRACE) && !cur_token_is(p, TOK_EOF)) {
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
        }
        next_token(p);
    }

    return node;
}

static AstNode *parse_func_declaration(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FUNC_DECL, p->cur_token);

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

            /* Type name follows (unless next param or closing paren) */
            if (peek_token_is(p, TOK_IDENT)) {
                next_token(p);
                param->type_name = p->cur_token.literal;
            }

            node->data.func_decl.param_count++;

            if (peek_token_is(p, TOK_COMMA)) {
                next_token(p); /* skip comma */
                next_token(p);
            }
        } while (!cur_token_is(p, TOK_RPAREN) && !peek_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF));
    }

    if (!expect_peek(p, TOK_RPAREN)) return NULL;

    /* Return type(s) */
    node->data.func_decl.return_type_count = 0;
    node->data.func_decl.return_types = NULL;

    if (peek_token_is(p, TOK_ARROW)) {
        next_token(p); /* skip -> */
        next_token(p);

        int ret_cap = 4;
        node->data.func_decl.return_types = arena_alloc(p->arena, sizeof(const char *) * ret_cap);

        if (cur_token_is(p, TOK_LPAREN)) {
            /* Multiple return types: -> (int, string) */
            next_token(p);
            while (!cur_token_is(p, TOK_RPAREN) && !cur_token_is(p, TOK_EOF)) {
                node->data.func_decl.return_types[node->data.func_decl.return_type_count++] =
                    p->cur_token.literal;
                if (peek_token_is(p, TOK_COMMA)) {
                    next_token(p);
                }
                next_token(p);
            }
        } else {
            /* Single return type */
            node->data.func_decl.return_types[0] = p->cur_token.literal;
            node->data.func_decl.return_type_count = 1;
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

        if (cur_token_is(p, TOK_AT)) {
            item->is_stdlib = true;
            next_token(p);
            item->module = p->cur_token.literal;
            item->alias = p->cur_token.literal;
        } else if (cur_token_is(p, TOK_STRING)) {
            item->is_stdlib = false;
            item->path = p->cur_token.literal;
        } else if (cur_token_is(p, TOK_AMPERSAND)) {
            /* import & use syntax */
            node->data.import_stmt.auto_use = true;
            continue;
        }

        node->data.import_stmt.count++;

        if (node->data.import_stmt.count >= cap) {
            cap *= 2;
            ImportItem *new_items = arena_alloc(p->arena, sizeof(ImportItem) * cap);
            memcpy(new_items, node->data.import_stmt.items,
                sizeof(ImportItem) * node->data.import_stmt.count);
            node->data.import_stmt.items = new_items;
        }
    } while (peek_token_is(p, TOK_COMMA));

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

static AstNode *parse_ensure_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_ENSURE_STMT, p->cur_token);
    next_token(p);
    node->data.ensure_stmt.expr = parse_expression(p, PREC_LOWEST);
    return node;
}

static AstNode *parse_for_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FOR_STMT, p->cur_token);

    if (!expect_peek(p, TOK_IDENT)) return NULL;
    node->data.for_stmt.var_name = p->cur_token.literal;

    /* Optional type annotation */
    node->data.for_stmt.var_type = NULL;

    if (!expect_peek(p, TOK_IN)) return NULL;

    next_token(p);
    node->data.for_stmt.iterable = parse_expression(p, PREC_LOWEST);

    if (!expect_peek(p, TOK_LBRACE)) return NULL;
    node->data.for_stmt.body = parse_block_statement(p);

    return node;
}

static AstNode *parse_for_each_statement(Parser *p) {
    AstNode *node = ast_alloc(p->arena, NODE_FOR_EACH_STMT, p->cur_token);

    next_token(p);
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
    node->data.for_each.collection = parse_expression(p, PREC_LOWEST);

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

static bool is_assignment_op(TokenType t) {
    return t == TOK_ASSIGN || t == TOK_PLUS_ASSIGN || t == TOK_MINUS_ASSIGN ||
           t == TOK_ASTERISK_ASSIGN || t == TOK_SLASH_ASSIGN || t == TOK_PERCENT_ASSIGN;
}

static AstNode *parse_statement(Parser *p) {
    switch (p->cur_token.type) {
    case TOK_TEMP:
    case TOK_CONST:
        return parse_var_declaration(p);
    case TOK_DO:
        return parse_func_declaration(p);
    case TOK_RETURN:
        return parse_return_statement(p);
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
    case TOK_ENSURE:
        return parse_ensure_statement(p);
    default: {
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

Parser *parser_create(Arena *arena, Lexer *lexer, const char *file) {
    Parser *p = arena_alloc(arena, sizeof(Parser));
    p->lexer = lexer;
    p->arena = arena;
    p->file = file;
    p->errors = NULL;
    p->error_count = 0;
    p->error_cap = 0;

    /* Read two tokens to fill cur and peek */
    next_token(p);
    next_token(p);

    return p;
}

AstNode *parser_parse_program(Parser *p) {
    Token tok = {TOK_EOF, "", 0, 0};
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
        }
        next_token(p);
    }

    return program;
}

bool parser_has_errors(Parser *p) {
    return p->error_count > 0;
}

void parser_print_errors(Parser *p) {
    for (int i = 0; i < p->error_count; i++) {
        fprintf(stderr, "error: %s\n", p->errors[i]);
    }
}
