/*
 * ast.c - AST node construction helpers
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ast.h"
#include "../util/arena.h"
#include <string.h>

AstNode *ast_alloc(Arena *a, NodeKind kind, Token tok) {
    AstNode *node = arena_alloc(a, sizeof(AstNode));
    memset(node, 0, sizeof(AstNode));
    node->kind = kind;
    node->token = tok;
    return node;
}
