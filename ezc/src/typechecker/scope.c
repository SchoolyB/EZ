/*
 * scope.c - Lexical scope and symbol table
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "scope.h"
#include <stdlib.h>
#include <string.h>

#define EZ_SCOPE_INITIAL_CAP 8

Scope *scope_create(Scope *parent) {
    Scope *s = calloc(1, sizeof(Scope));
    s->parent = parent;
    return s;
}

void scope_define(Scope *s, const char *name, EzType *type, bool mutable) {
    /* Check for redefinition in current scope */
    for (int i = 0; i < s->count; i++) {
        if (strcmp(s->symbols[i].name, name) == 0) {
            /* Update existing */
            s->symbols[i].type = type;
            s->symbols[i].mutable = mutable;
            return;
        }
    }

    if (s->count >= s->cap) {
        s->cap = s->cap ? s->cap * 2 : EZ_SCOPE_INITIAL_CAP;
        s->symbols = realloc(s->symbols, sizeof(Symbol) * s->cap);
    }

    Symbol *sym = &s->symbols[s->count++];
    memset(sym, 0, sizeof(Symbol));
    sym->name = name;
    sym->type = type;
    sym->mutable = mutable;
}

Symbol *scope_lookup(Scope *s, const char *name) {
    for (Scope *cur = s; cur; cur = cur->parent) {
        Symbol *sym = scope_lookup_local(cur, name);
        if (sym) return sym;
    }
    return NULL;
}

Symbol *scope_lookup_local(Scope *s, const char *name) {
    for (int i = 0; i < s->count; i++) {
        if (strcmp(s->symbols[i].name, name) == 0) {
            return &s->symbols[i];
        }
    }
    return NULL;
}
