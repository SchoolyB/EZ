/*
 * typechecker.h - Type checking pass for the EZ language
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_TYPECHECKER_H
#define EZC_TYPECHECKER_H

#include "types.h"
#include "scope.h"
#include "../parser/ast.h"
#include "../util/error.h"

/* Type annotation table: maps AST node pointers to resolved types */
typedef struct {
    AstNode **nodes;
    EzType **types;
    int count;
    int cap;
} TypeTable;

typedef struct {
    /* Struct field info for field type lookup */
    const char *struct_name;
    const char **field_names;
    EzType **field_types;
    int field_count;
} StructInfo;

typedef struct {
    /* Function signature for call type checking */
    const char *name;
    EzType **param_types;
    int param_count;
    EzType **return_types;
    int return_count;
} FuncSig;

typedef struct {
    DiagnosticList *diag;
    Scope *current_scope;
    TypeTable *type_table;
    const char *file;

    /* Registered struct types */
    StructInfo *structs;
    int struct_count;
    int struct_cap;

    /* Registered function signatures */
    FuncSig *funcs;
    int func_count;
    int func_cap;

    /* Registered enum names */
    const char **enum_names;
    int enum_count;
    int enum_cap;
} TypeChecker;

/* Create and run the type checker */
TypeChecker *typechecker_create(DiagnosticList *diag, const char *file);
void typechecker_check(TypeChecker *tc, AstNode *program);

/* Query the type table (used by codegen) */
EzType *typetable_get(TypeTable *tt, AstNode *node);
void typetable_set(TypeTable *tt, AstNode *node, EzType *type);

/* Get the type table from the checker */
TypeTable *typechecker_get_table(TypeChecker *tc);

#endif
