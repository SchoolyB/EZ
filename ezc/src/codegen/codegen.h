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
    Buf global_init;    /* Deferred initialization for file-scope arrays */
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

    /* #1521: loop scoping depth — when > 0, codegen is inside a
     * scoped loop and container mutations need escape-copy logic. */
    int loop_scope_depth;

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

    /* All imported module names (for module detection in member expressions) */
    const char **imported_modules;
    int imported_module_count;
    int imported_module_cap;

    /* C interop headers from import c"header.h" */
    const char **c_headers;
    int c_header_count;
    int c_header_cap;
    bool has_c_imports;

    /* Active wildcard binding (#1443). Set while emitting a specialised
     * instantiation of a generic function so type-string lookups can
     * substitute "?" with a concrete type name, and so the mangled
     * function name can be appended at call sites. NULL outside a
     * generic instantiation. */
    const char *wildcard_binding;

    /* Side channel for typed-func call-through: when the callee is a
     * variable typed func(...), the cast emitter stashes the parsed
     * signature here so the arg-emission loop can pick up &-mutability
     * even when target_func (the AST decl) is unknown. Reset to NULL
     * after each call. */
    void *pending_call_typed_sig;
} CodeGen;

CodeGen codegen_create(const char *file);
void codegen_generate(CodeGen *cg, AstNode *program);
const char *codegen_result(CodeGen *cg);
void codegen_destroy(CodeGen *cg);

#endif
