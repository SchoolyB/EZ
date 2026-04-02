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
#include "../typechecker/typechecker.h"

typedef struct {
    Buf output;
    int indent;
    bool has_mem;       /* Whether @mem was imported */
    bool has_fmt;       /* Whether @fmt was imported */
    const char *file;

    /* Track declared type names for codegen */
    const char **enum_names;
    int enum_count;
    int enum_cap;

    /* Current function context (for multi-return, ensure) */
    AstNode *current_func;

    /* All function declarations (for mutable param lookup at call sites) */
    AstNode **all_funcs;
    int func_count;
    int func_cap;

    /* Type table from type checker (for type-aware codegen) */
    TypeTable *type_table;

    /* Current var decl context (for context-aware call emission) */
    const char *current_var_name;
    const char *current_var_type;

    /* Ref variables (transparent references from ref()) */
    const char **ref_vars;
    int ref_var_count;
    int ref_var_cap;

    /* Track declared bigint variable types (name → type_name) */
    const char **bigint_var_names;
    const char **bigint_var_types;
    int bigint_var_count;
    int bigint_var_cap;

    /* Struct declarations for composite printing */
    AstNode **struct_decls;
    int struct_decl_count;
    int struct_decl_cap;

    /* Modules brought into scope via 'using' or 'import and use' */
    const char **using_modules;
    int using_module_count;
    int using_module_cap;

    /* Import alias → module name mapping */
    const char **alias_names;
    const char **alias_modules;
    int alias_count;
    int alias_cap;
} CodeGen;

CodeGen codegen_create(const char *file);
void codegen_generate(CodeGen *cg, AstNode *program);
const char *codegen_result(CodeGen *cg);
void codegen_destroy(CodeGen *cg);

#endif
