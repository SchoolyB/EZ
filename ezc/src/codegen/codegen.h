/*
 * codegen.h - C code generator for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_CODEGEN_H
#define EZC_CODEGEN_H

#include "../parser/ast.h"
#include "../util/buf.h"

typedef struct {
    Buf output;
    int indent;
    bool has_std;       /* Whether @std was imported */
    const char *file;

    /* Track declared type names for codegen */
    const char **enum_names;
    int enum_count;
    int enum_cap;
} CodeGen;

CodeGen codegen_create(const char *file);
void codegen_generate(CodeGen *cg, AstNode *program);
const char *codegen_result(CodeGen *cg);
void codegen_destroy(CodeGen *cg);

#endif
