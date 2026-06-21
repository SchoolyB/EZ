/*
 * scope.c - Lexical scope and symbol table
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "scope.h"
#include "../util/xalloc.h"
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#define EZ_SCOPE_INITIAL_CAP 8

static uint32_t scope_str_hash(const char *s) {
    uint32_t h = 5381u;
    for (const unsigned char *p = (const unsigned char *)s; *p; p++)
        h = h * 33u ^ (uint32_t)*p;
    return h;
}

static void scope_hash_insert(Scope *s, int idx) {
    uint32_t mask = (uint32_t)(s->hash_cap - 1);
    uint32_t h = scope_str_hash(s->symbols[idx].name) & mask;
    for (;;) {
        if (!s->hash[h].name) {
            s->hash[h].name = s->symbols[idx].name;
            s->hash[h].idx  = idx;
            return;
        }
        h = (h + 1) & mask;
    }
}

static void scope_hash_rebuild(Scope *s, int new_cap) {
    free(s->hash);
    s->hash = xcalloc((size_t)new_cap, sizeof(ScopeHashEntry));
    s->hash_cap = new_cap;
    for (int i = 0; i < s->count; i++)
        scope_hash_insert(s, i);
}

Scope *scope_create(Scope *parent) {
    Scope *s = xmalloc(sizeof(Scope));
    memset(s, 0, sizeof(Scope));
    s->parent = parent;
    return s;
}

void scope_define(Scope *s, const char *name, EzType *type, bool mutable) {
    /* Check for redefinition in current scope */
    Symbol *existing = scope_lookup_local(s, name);
    if (existing) {
        existing->type = type;
        existing->mutable = mutable;
        return;
    }

    if (s->count >= s->cap) {
        s->cap = s->cap ? s->cap * 2 : EZ_SCOPE_INITIAL_CAP;
        s->symbols = xrealloc(s->symbols, sizeof(Symbol) * s->cap);
    }

    /* Grow hash table before adding the new symbol so it always has room. */
    int new_count = s->count + 1;
    if (!s->hash || new_count * 2 > s->hash_cap) {
        int new_hash_cap = s->hash_cap ? s->hash_cap * 2 : 16;
        while (new_count * 2 > new_hash_cap) new_hash_cap *= 2;
        scope_hash_rebuild(s, new_hash_cap);
    }

    Symbol *sym = &s->symbols[s->count++];
    memset(sym, 0, sizeof(Symbol));
    sym->name = name;
    sym->type = type;
    sym->mutable = mutable;
    scope_hash_insert(s, s->count - 1);
}

Symbol *scope_lookup(Scope *s, const char *name) {
    for (Scope *cur = s; cur; cur = cur->parent) {
        Symbol *sym = scope_lookup_local(cur, name);
        if (sym) return sym;
    }
    return NULL;
}

Symbol *scope_lookup_local(Scope *s, const char *name) {
    if (s->hash) {
        uint32_t mask = (uint32_t)(s->hash_cap - 1);
        uint32_t h = scope_str_hash(name) & mask;
        for (;;) {
            ScopeHashEntry *e = &s->hash[h];
            if (!e->name) return NULL;
            if (strcmp(e->name, name) == 0) return &s->symbols[e->idx];
            h = (h + 1) & mask;
        }
    }
    /* Fallback for scopes with no symbols defined yet */
    for (int i = 0; i < s->count; i++) {
        if (strcmp(s->symbols[i].name, name) == 0)
            return &s->symbols[i];
    }
    return NULL;
}
