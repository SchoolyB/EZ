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

/* Type annotation table: maps AST node pointers to resolved types.
 * Uses open-addressing hash table with pointer hashing for O(1) lookup. */
#define TYPETABLE_INIT_CAP 256

typedef struct {
    AstNode **nodes;    /* hash buckets — NULL = empty slot */
    EzType **types;     /* parallel type array */
    int count;          /* number of entries */
    int cap;            /* always a power of 2 */
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
    bool used;          /* true if function was called */
    int def_line;       /* line where function was declared */
    bool is_private;    /* true if declared with 'private' keyword */
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

    /* Program AST (for default param lookup) */
    AstNode *program;

    /* Registered enum names */
    const char **enum_names;
    bool *enum_is_string; /* parallel array: true if string enum */
    const char ***enum_values; /* parallel array: variant name arrays */
    int *enum_value_counts; /* parallel array: variant counts */
    int enum_count;
    int enum_cap;

    /* Control flow tracking */
    int loop_depth;               /* >0 means inside a loop */
    int func_depth;               /* >0 means inside a function body */
    EzType **current_return_types; /* expected return types of current function */
    const char **current_return_type_names; /* raw declared return type names */
    int current_return_count;
    bool current_has_named_returns; /* true if current function uses named return values */
    const char **current_return_names; /* named return variable names (NULL entries for unnamed) */

    /* Import tracking for unused import warnings */
    const char **imported_modules;
    int *import_lines;
    bool *import_used;
    bool *import_is_stdlib;
    int import_count;
    int import_cap;

    /* Modules brought into scope via 'using' or 'import and use' */
    const char **using_modules;
    int using_module_count;
    int using_module_cap;

    /* Import alias → module name mapping */
    const char **alias_names;
    const char **alias_modules;
    int alias_count;
    int alias_cap;
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
