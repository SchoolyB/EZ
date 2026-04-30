/*
 * scope.h - Lexical scope and symbol table for type checking
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_SCOPE_H
#define EZC_SCOPE_H

#include "types.h"

typedef struct {
    const char *name;
    EzType *type;
    const char *declared_type; /* original declared type name (e.g., "uint", "i8") */
    bool mutable;
    bool is_ref;         /* true if created via ref() — transparent reference */
    bool used;           /* true if variable was read */
    int def_line;        /* line where variable was defined */
    int def_column;      /* column where variable was defined */
    EzType **ret_types;  /* for multi-return temps: all return types */
    int ret_count;       /* number of return types */
    const char *func_ref_name; /* for func-typed vars: name of the referenced
                                  function, used for call-site arity/type
                                  validation (NULL if not assigned from a
                                  static func ref) */
    /* For [func] array vars initialised with a literal of func refs
     *per-element referenced function names, length == the
     * literal's count. NULL entries mark elements that aren't a
     * static func ref. Used by call-site type inference on
     * arr[const](...) expressions so struct return types survive the
     * trip through a type-erased void* array. */
    const char **func_array_refs;
    int func_array_ref_count;
} Symbol;

typedef struct Scope {
    struct Scope *parent;
    Symbol *symbols;
    int count;
    int cap;
} Scope;

Scope *scope_create(Scope *parent);
void scope_define(Scope *s, const char *name, EzType *type, bool mutable);
Symbol *scope_lookup(Scope *s, const char *name);
Symbol *scope_lookup_local(Scope *s, const char *name);

#endif
