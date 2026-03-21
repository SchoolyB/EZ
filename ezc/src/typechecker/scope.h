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
    bool mutable;
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
