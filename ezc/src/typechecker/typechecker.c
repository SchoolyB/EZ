/*
 *
 * Walks the AST, resolves expression types, checks type correctness,
 * and builds a type table that the codegen can query.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "typechecker.h"
#include "../util/constants.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <ctype.h>

#define EZ_MAX_STRUCT_DEPTH 32

/* Helper: get the source file from an AST node's token, falling back to tc->file.
 * Imported nodes carry their original file path in token.file; main-file nodes have NULL. */
#define NODE_FILE(tc, n) ((n)->token.file ? (n)->token.file : (tc)->file)

/* --- Type Table (open-addressing hash, pointer keys) --- */

static uint32_t hash_ptr(const void *ptr) {
    uintptr_t v = (uintptr_t)ptr;
    /* Fibonacci hashing; good distribution for pointer alignment */
    v = ((v >> 4) ^ v) * 0x9E3779B9U;
    return (uint32_t)(v ^ (v >> 16));
}

TypeTable *typetable_create(void) {
    TypeTable *tt = calloc(1, sizeof(TypeTable));
    tt->cap = TYPETABLE_INIT_CAP;
    tt->nodes = calloc((size_t)tt->cap, sizeof(AstNode *));
    tt->types = calloc((size_t)tt->cap, sizeof(EzType *));
    return tt;
}

static void typetable_grow(TypeTable *tt) {
    int old_cap = tt->cap;
    AstNode **old_nodes = tt->nodes;
    EzType **old_types = tt->types;

    tt->cap = old_cap * 2;
    tt->nodes = calloc((size_t)tt->cap, sizeof(AstNode *));
    tt->types = calloc((size_t)tt->cap, sizeof(EzType *));
    tt->count = 0;

    for (int i = 0; i < old_cap; i++) {
        if (old_nodes[i]) {
            typetable_set(tt, old_nodes[i], old_types[i]);
        }
    }
    free(old_nodes);
    free(old_types);
}

void typetable_set(TypeTable *tt, AstNode *node, EzType *type) {
    /* Grow at 70% load factor */
    if (tt->count * 10 >= tt->cap * 7) {
        typetable_grow(tt);
    }

    uint32_t mask = (uint32_t)(tt->cap - 1);
    uint32_t idx = hash_ptr(node) & mask;

    for (;;) {
        if (!tt->nodes[idx]) {
            /* Empty slot; insert */
            tt->nodes[idx] = node;
            tt->types[idx] = type;
            tt->count++;
            return;
        }
        if (tt->nodes[idx] == node) {
            /* Already present; update */
            tt->types[idx] = type;
            return;
        }
        idx = (idx + 1) & mask;
    }
}

EzType *typetable_get(TypeTable *tt, AstNode *node) {
    if (!tt || !tt->nodes) return NULL;

    uint32_t mask = (uint32_t)(tt->cap - 1);
    uint32_t idx = hash_ptr(node) & mask;

    for (;;) {
        if (!tt->nodes[idx]) return NULL;
        if (tt->nodes[idx] == node) return tt->types[idx];
        idx = (idx + 1) & mask;
    }
}

/* --- Reserved type name check --- */
static bool is_reserved_type_name(const char *name) {
    return strcmp(name, "int") == 0 || strcmp(name, "uint") == 0 ||
           strcmp(name, "float") == 0 || strcmp(name, "string") == 0 ||
           strcmp(name, "bool") == 0 || strcmp(name, "char") == 0 ||
           strcmp(name, "byte") == 0 || strcmp(name, "void") == 0 ||
           strcmp(name, "Error") == 0 || strcmp(name, "nil") == 0;
}

/* True if expr is an lvalue (something with a stable address): a variable,
 * a field of an lvalue, an index into an lvalue, or a pointer dereference.
 * Used by addr() and ref() to reject literals, call results, arithmetic
 * expressions, etc. — none of which have an address to take. */
static bool is_lvalue_expr(AstNode *e) {
    if (!e) return false;
    switch (e->kind) {
    case NODE_LABEL:        return true;
    case NODE_MEMBER_EXPR:  return is_lvalue_expr(e->data.member.object);
    case NODE_INDEX_EXPR:   return is_lvalue_expr(e->data.index_expr.left);
    case NODE_POSTFIX_EXPR:
        /* p^ (dereference) is an lvalue */
        return e->data.postfix.op &&
               strcmp(e->data.postfix.op, "^") == 0;
    default:                return false;
    }
}

/* --- Struct info helpers --- */

static void register_struct(TypeChecker *tc, const char *name,
    const char **field_names, EzType **field_types, int field_count) {
    if (tc->struct_count >= tc->struct_cap) {
        tc->struct_cap = tc->struct_cap ? tc->struct_cap * 2 : 8;
        tc->structs = realloc(tc->structs, sizeof(StructInfo) * tc->struct_cap);
    }
    StructInfo *si = &tc->structs[tc->struct_count++];
    si->struct_name = name;
    si->field_names = field_names;
    si->field_types = field_types;
    si->field_count = field_count;
}

/* Resolve an import alias to the actual module name */
static const char *tc_resolve_alias(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->alias_count; i++) {
        if (strcmp(tc->alias_names[i], name) == 0) return tc->alias_modules[i];
    }
    return name;
}

static AstNode *find_struct_in_program(AstNode *program, const char *name);

static bool is_struct_name(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->struct_count; i++) {
        if (strcmp(tc->structs[i].struct_name, name) == 0) return true;
    }
    return false;
}

static StructInfo *find_struct(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->struct_count; i++) {
        if (strcmp(tc->structs[i].struct_name, name) == 0)
            return &tc->structs[i];
    }
    return NULL;
}

static EzType *struct_field_type(TypeChecker *tc, const char *struct_name, const char *field) {
    StructInfo *si = find_struct(tc, struct_name);
    /* : for mangled generic struct names (Pair__int), fall back
     * to the base name (Pair) since fields are registered there.
     * When the field type is "?", substitute the concrete binding
     * extracted from the mangled suffix. */
    const char *generic_binding = NULL;
    if (!si && struct_name) {
        const char *dunder = strstr(struct_name, "__");
        if (dunder) {
            char base[EZ_MSG_BUF_SIZE];
            size_t n = (size_t)(dunder - struct_name);
            if (n < sizeof(base)) {
                memcpy(base, struct_name, n);
                base[n] = '\0';
                si = find_struct(tc, base);
                generic_binding = dunder + 2;
            }
        }
    }
    if (!si) return &TYPE_UNKNOWN;
    for (int i = 0; i < si->field_count; i++) {
        if (strcmp(si->field_names[i], field) == 0) {
            /* : if the field type is ? (registered as TK_UNKNOWN)
             * and we have a generic binding from the mangled name,
             * substitute to the concrete type. Check the raw decl
             * type_name since the resolved EzType lost the "?" marker. */
            if (generic_binding && si->field_types[i]->kind == TK_UNKNOWN) {
                /* Find the raw struct decl to check the field type_name */
                if (tc->program) {
                    AstNode *decl = find_struct_in_program(tc->program,
                        struct_name); /* try mangled first */
                    if (!decl) {
                        /* Extract base name and try again */
                        const char *dd = strstr(struct_name, "__");
                        if (dd) {
                            char bname[EZ_MSG_BUF_SIZE];
                            size_t bn = (size_t)(dd - struct_name);
                            if (bn < sizeof(bname)) {
                                memcpy(bname, struct_name, bn);
                                bname[bn] = '\0';
                                decl = find_struct_in_program(tc->program, bname);
                            }
                        }
                    }
                    if (decl) {
                        for (int fi = 0; fi < decl->data.struct_decl.field_count; fi++) {
                            if (strcmp(decl->data.struct_decl.fields[fi].name, field) == 0 &&
                                decl->data.struct_decl.fields[fi].type_name &&
                                strchr(decl->data.struct_decl.fields[fi].type_name, '?')) {
                                return type_from_name(generic_binding);
                            }
                        }
                    }
                }
            }
            return si->field_types[i];
        }
    }
    return &TYPE_UNKNOWN;
}

/* --- Function signature helpers --- */

static bool type_name_has_wildcard(const char *tn) {
    if (!tn) return false;
    for (const char *c = tn; *c; c++) {
        if (*c == '?') return true;
    }
    return false;
}

static void register_func(TypeChecker *tc, const char *name,
    EzType **param_types, int param_count,
    EzType **return_types, int return_count) {
    if (tc->func_count >= tc->func_cap) {
        tc->func_cap = tc->func_cap ? tc->func_cap * 2 : 16;
        tc->funcs = realloc(tc->funcs, sizeof(FuncSig) * tc->func_cap);
    }
    FuncSig *fs = &tc->funcs[tc->func_count++];
    fs->name = name;
    fs->param_types = param_types;
    fs->param_count = param_count;
    fs->return_types = return_types;
    fs->return_count = return_count;
    fs->used = false;
    fs->def_line = 0;
    fs->is_private = false;
    fs->is_generic = false;
    fs->decl = NULL;
    fs->instantiations = NULL;
    fs->instantiation_calls = NULL;
    fs->instantiation_count = 0;
    fs->instantiation_cap = 0;
}

/* Substitute '?' with `concrete` in a type string and return a heap copy.
 * Returns a strdup of the original if no wildcard is present. */
static char *substitute_wildcard(const char *src, const char *concrete) {
    if (!src) return NULL;
    if (!type_name_has_wildcard(src)) return strdup(src);
    size_t cl = strlen(concrete);
    size_t len = 0;
    for (const char *c = src; *c; c++) len += (*c == '?') ? cl : 1;
    char *out = malloc(len + 1);
    char *w = out;
    for (const char *c = src; *c; c++) {
        if (*c == '?') { memcpy(w, concrete, cl); w += cl; }
        else *w++ = *c;
    }
    *w = '\0';
    return out;
}

/* Derive the "concrete type that a wildcard parameter binds to", given
 * the parameter's source type string (containing '?') and the argument's
 * resolved EzType. Examples:
 *   param "?"             + arg int                -> "int"
 *   param "[?]"            + arg [string]           -> "string"
 *   param "map[string:?]"  + arg map[string:int]    -> "int"
 *   param "map[?:int]"     + arg map[string:int]    -> "string"
 * Returns NULL if the shape doesn't match (e.g. param "[?]" with a
 * non-array arg, or param "map[int:?]" with a map whose key type is
 * not int). Caller owns the returned string. */
static char *bind_wildcard(const char *param_tn, EzType *arg_t) {
    if (!param_tn || !arg_t) return NULL;
    if (strcmp(param_tn, "?") == 0) {
        return strdup(type_name(arg_t));
    }
    if (param_tn[0] == '[' && arg_t->kind == TK_ARRAY && arg_t->element_type) {
        /* [?] vs concrete array; bind to the element type */
        const char *inside = param_tn + 1;
        /* Only handle the simple "[?]" case for this slice. Nested or
         * sized forms (`[[?]]`, `[?,3]`) fall through to a literal
         * match below. */
        if (strncmp(inside, "?]", 2) == 0) {
            return strdup(arg_t->element_type);
        }
    }
    /* map[K:V] with '?' in either slot (). The array path above
     * only handles "[?]"; the equivalent TK_MAP path here was missing,
     * so every generic map helper silently fell through to NULL. */
    if (strncmp(param_tn, "map[", 4) == 0 && arg_t->kind == TK_MAP &&
        arg_t->key_type && arg_t->value_type) {
        /* Split the param's key/value slots at the top-level ':'. */
        const char *start = param_tn + 4;
        const char *colon = strchr(start, ':');
        if (!colon) return NULL;
        size_t klen = (size_t)(colon - start);
        const char *vstart = colon + 1;
        size_t vlen = strlen(vstart);
        if (vlen == 0 || vstart[vlen - 1] != ']') return NULL;
        vlen--; /* drop trailing ']' */

        bool k_wild = (klen == 1 && start[0] == '?');
        bool v_wild = (vlen == 1 && vstart[0] == '?');

        /* If the key slot is concrete, it must match the argument's key
         * type exactly; otherwise the map types genuinely disagree and
         * unification should fail. Same for the value slot. */
        if (!k_wild) {
            if (strlen(arg_t->key_type) != klen ||
                memcmp(arg_t->key_type, start, klen) != 0) {
                return NULL;
            }
        }
        if (!v_wild) {
            if (strlen(arg_t->value_type) != vlen ||
                memcmp(arg_t->value_type, vstart, vlen) != 0) {
                return NULL;
            }
        }

        if (k_wild && v_wild) {
            /* Both slots are '?'. The generic sig carries a single
             * wildcard binding, so we can only accept this when the
             * argument's key and value types agree (the one binding
             * satisfies both slots). */
            if (strcmp(arg_t->key_type, arg_t->value_type) != 0) return NULL;
            return strdup(arg_t->key_type);
        }
        if (k_wild) return strdup(arg_t->key_type);
        if (v_wild) return strdup(arg_t->value_type);
    }
    return NULL;
}

/* Mark a just-registered FuncSig as generic if any of the declared
 * parameter or return type strings on `decl` contains a '?'. Stores
 * `decl` on the sig so later call-site instantiation can walk the
 * original type_name strings for substitution. */
static void finalize_generic_sig(FuncSig *fs, AstNode *decl) {
    fs->decl = decl;
    if (!decl || decl->kind != NODE_FUNC_DECL) return;
    for (int i = 0; i < decl->data.func_decl.param_count; i++) {
        if (type_name_has_wildcard(decl->data.func_decl.params[i].type_name)) {
            fs->is_generic = true;
            return;
        }
    }
    for (int i = 0; i < decl->data.func_decl.return_type_count; i++) {
        if (type_name_has_wildcard(decl->data.func_decl.return_types[i])) {
            fs->is_generic = true;
            return;
        }
    }
}

/* Record a concrete instantiation of a generic function. Returns true
 * if this is a new instantiation (i.e. codegen hasn't seen it yet),
 * false if the same concrete binding was already recorded. Also
 * mirrors the entry onto the source AST func_decl so codegen can
 * enumerate instantiations without needing the FuncSig table. */
static bool record_instantiation(FuncSig *fs, const char *concrete,
                                  AstNode *call_site) {
    if (!fs || !concrete) return false;
    /* : reject "unknown" as a concrete binding. This comes from
     * the main-pass walk of a generic body where the inner call's
     * arguments are still `?` (TK_UNKNOWN). The real bindings are
     * recorded during the slice-4 re-check pass once the outer
     * function's parameters are rebound to concrete types. */
    if (strcmp(concrete, "unknown") == 0) return false;
    for (int i = 0; i < fs->instantiation_count; i++) {
        if (strcmp(fs->instantiations[i], concrete) == 0) return false;
    }
    if (fs->instantiation_count >= fs->instantiation_cap) {
        fs->instantiation_cap = fs->instantiation_cap ? fs->instantiation_cap * 2 : 4;
        fs->instantiations = realloc(fs->instantiations,
            sizeof(const char *) * fs->instantiation_cap);
        fs->instantiation_calls = realloc(fs->instantiation_calls,
            sizeof(AstNode *) * fs->instantiation_cap);
    }
    char *stored = strdup(concrete);
    fs->instantiations[fs->instantiation_count] = stored;
    fs->instantiation_calls[fs->instantiation_count] = call_site;
    fs->instantiation_count++;

    if (fs->decl && fs->decl->kind == NODE_FUNC_DECL) {
        int n = fs->decl->data.func_decl.instantiation_count;
        fs->decl->data.func_decl.instantiations = realloc(
            (void *)fs->decl->data.func_decl.instantiations,
            sizeof(const char *) * (size_t)(n + 1));
        fs->decl->data.func_decl.instantiations[n] = stored;
        fs->decl->data.func_decl.instantiation_count = n + 1;
    }
    return true;
}

static FuncSig *find_func(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->func_count; i++) {
        if (strcmp(tc->funcs[i].name, name) == 0)
            return &tc->funcs[i];
    }
    return NULL;
}

/* Fallible stdlib function lookup table.
 * Each entry maps a (module, function) pair to a flag indicating it supports
 * multi-var (value, Error) destructuring via a _result variant. */
typedef struct {
    const char *mod;
    const char *fn;
} FallibleEntry;

static const FallibleEntry fallible_stdlib[] = {
    /* io */
    {"io", "read_file"}, {"io", "write_file"}, {"io", "delete_file"},
    {"io", "copy_file"}, {"io", "move_file"}, {"io", "list_dir"},
    {"io", "make_dir"}, {"io", "make_dir_all"}, {"io", "remove_dir"},
    {"io", "remove_dir_all"}, {"io", "walk"}, {"io", "append_file"},
    {"io", "rename_file"},
    /* sqlite */
    {"sqlite", "open"}, {"sqlite", "exec"}, {"sqlite", "query"},
    /* net */
    {"net", "connect"}, {"net", "listen"}, {"net", "accept"},
    {"net", "send"}, {"net", "receive"}, {"net", "resolve"},
    /* http */
    {"http", "get"}, {"http", "post"}, {"http", "put"},
    {"http", "delete"}, {"http", "head"}, {"http", "patch"},
    /* csv */
    {"csv", "read_file"}, {"csv", "write_file"},
    /* json */
    {"json", "decode"},
    /* regex */
    {"regex", "find"}, {"regex", "find_all"},
    {"regex", "replace"}, {"regex", "split"},
};

static bool tc_is_fallible_stdlib(const char *mod, const char *fn) {
    for (int i = 0; i < (int)(sizeof(fallible_stdlib) / sizeof(fallible_stdlib[0])); i++) {
        if (strcmp(mod, fallible_stdlib[i].mod) == 0 &&
            strcmp(fn, fallible_stdlib[i].fn) == 0) return true;
    }
    return false;
}

/* Return the primary (non-error) type for a fallible stdlib function.
 * This mirrors the type registration blocks so that multi-var synthesis
 * doesn't depend on sym->type being resolved first. */
typedef enum {
    FT_BOOL, FT_INT, FT_STRING,
    FT_ARRAY_STRING, FT_ARRAY_MAP,
    FT_STRUCT_DATABASE, FT_STRUCT_SOCKET, FT_STRUCT_LISTENER,
    FT_STRUCT_HTTP_RESPONSE, FT_STRUCT_MAP,
} FallibleType;

typedef struct {
    const char *mod;
    const char *fn;
    FallibleType type;
} FallibleTypeEntry;

static const FallibleTypeEntry fallible_type_table[] = {
    /* io */
    {"io", "read_file", FT_STRING},
    {"io", "list_dir", FT_ARRAY_STRING}, {"io", "walk", FT_ARRAY_STRING},
    {"io", "write_file", FT_BOOL}, {"io", "delete_file", FT_BOOL},
    {"io", "copy_file", FT_BOOL}, {"io", "move_file", FT_BOOL},
    {"io", "make_dir", FT_BOOL}, {"io", "make_dir_all", FT_BOOL},
    {"io", "remove_dir", FT_BOOL}, {"io", "remove_dir_all", FT_BOOL},
    {"io", "append_file", FT_BOOL}, {"io", "rename_file", FT_BOOL},
    /* sqlite */
    {"sqlite", "open", FT_STRUCT_DATABASE},
    {"sqlite", "query", FT_ARRAY_MAP},
    {"sqlite", "exec", FT_BOOL},
    /* net */
    {"net", "connect", FT_STRUCT_SOCKET}, {"net", "accept", FT_STRUCT_SOCKET},
    {"net", "listen", FT_STRUCT_LISTENER},
    {"net", "send", FT_INT},
    {"net", "receive", FT_STRING}, {"net", "resolve", FT_STRING},
    /* http */
    {"http", "get", FT_STRUCT_HTTP_RESPONSE}, {"http", "post", FT_STRUCT_HTTP_RESPONSE},
    {"http", "put", FT_STRUCT_HTTP_RESPONSE}, {"http", "delete", FT_STRUCT_HTTP_RESPONSE},
    {"http", "head", FT_STRUCT_HTTP_RESPONSE}, {"http", "patch", FT_STRUCT_HTTP_RESPONSE},
    /* csv */
    {"csv", "read_file", FT_ARRAY_STRING}, {"csv", "write_file", FT_BOOL},
    /* json */
    {"json", "decode", FT_STRUCT_MAP},
    /* regex */
    {"regex", "find", FT_STRING}, {"regex", "replace", FT_STRING},
    {"regex", "find_all", FT_ARRAY_STRING}, {"regex", "split", FT_ARRAY_STRING},
};

static EzType *tc_get_fallible_stdlib_type(const char *mod, const char *fn) {
    for (int i = 0; i < (int)(sizeof(fallible_type_table) / sizeof(fallible_type_table[0])); i++) {
        if (strcmp(mod, fallible_type_table[i].mod) == 0 &&
            strcmp(fn, fallible_type_table[i].fn) == 0) {
            switch (fallible_type_table[i].type) {
            case FT_BOOL:                return &TYPE_BOOL;
            case FT_INT:                 return &TYPE_INT;
            case FT_STRING:              return &TYPE_STRING;
            case FT_ARRAY_STRING:        return type_array("string");
            case FT_ARRAY_MAP:           return type_array("map");
            case FT_STRUCT_DATABASE:     return type_struct("Database");
            case FT_STRUCT_SOCKET:       return type_struct("Socket");
            case FT_STRUCT_LISTENER:     return type_struct("Listener");
            case FT_STRUCT_HTTP_RESPONSE:return type_struct("HttpResponse");
            case FT_STRUCT_MAP:          return type_struct("Map");
            }
        }
    }
    return NULL;
}

/* --- "Did you mean?" suggestion helper --- */

static int levenshtein(const char *a, const char *b) {
    int la = (int)strlen(a), lb = (int)strlen(b);
    if (la == 0) return lb;
    if (lb == 0) return la;
    /* Use single-row DP */
    int *row = malloc(sizeof(int) * (lb + 1));
    for (int j = 0; j <= lb; j++) row[j] = j;
    for (int i = 1; i <= la; i++) {
        int prev = row[0];
        row[0] = i;
        for (int j = 1; j <= lb; j++) {
            int cost = (a[i-1] == b[j-1]) ? 0 : 1;
            int del = row[j] + 1;
            int ins = row[j-1] + 1;
            int sub = prev + cost;
            prev = row[j];
            row[j] = del < ins ? (del < sub ? del : sub) : (ins < sub ? ins : sub);
        }
    }
    int result = row[lb];
    free(row);
    return result;
}

/* Find closest matching name from scope variables, functions, and builtins */
static const char *suggest_name(TypeChecker *tc, const char *name) {
    const char *best = NULL;
    int best_dist = 3; /* max distance for suggestions */

    /* Check scope variables */
    for (Scope *s = tc->current_scope; s; s = s->parent) {
        for (int i = 0; i < s->count; i++) {
            int d = levenshtein(name, s->symbols[i].name);
            if (d > 0 && d < best_dist) {
                best_dist = d;
                best = s->symbols[i].name;
            }
        }
    }

    /* Check registered functions */
    for (int i = 0; i < tc->func_count; i++) {
        int d = levenshtein(name, tc->funcs[i].name);
        if (d > 0 && d < best_dist) {
            best_dist = d;
            best = tc->funcs[i].name;
        }
    }

    return best;
}

/* --- Builtin name check --- */

static bool is_valid_module(const char *name);

static bool tc_is_imported_module(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->import_count; i++) {
        if (strcmp(tc->imported_modules[i], name) == 0) return true;
    }
    return false;
}

static bool tc_is_stdlib_import(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->import_count; i++) {
        if (strcmp(tc->imported_modules[i], name) == 0)
            return tc->import_is_stdlib[i];
    }
    return false;
}

/* If type_name is module-prefixed (e.g. "T_Query"), mark that module used.
 * The parser rewrites "T.Query" to "T_Query", so we split on the first
 * underscore and check whether the prefix is a known import. */
static void tc_mark_type_module_used(TypeChecker *tc, const char *type_name) {
    if (!type_name) return;
    const char *us = strchr(type_name, '_');
    if (!us || us == type_name) return;
    size_t prefix_len = (size_t)(us - type_name);
    for (int mi = 0; mi < tc->import_count; mi++) {
        if (strlen(tc->imported_modules[mi]) == prefix_len &&
            strncmp(tc->imported_modules[mi], type_name, prefix_len) == 0) {
            tc->import_used[mi] = true;
            return;
        }
    }
}

static bool tc_is_builtin(const char *name) {
    static const char *builtins[] = {
        "println", "print", "eprintln", "eprint", "input",
        "len", "type_of", "size_of", "copy", "new", "ref", "addr", "error",
        "int", "uint", "float", "string", "char", "byte", "bool",
        "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "f32", "f64",
        "i128", "i256", "u128", "u256",
        "exit", "panic", "assert", "range", "cast",
        "sleep_s", "sleep_ms", "sleep_ns", "c_string",
        "to_char", "char_count",
        NULL
    };
    for (int i = 0; builtins[i]; i++) {
        if (strcmp(name, builtins[i]) == 0) return true;
    }
    return false;
}

/* : stdlib constants reachable via `import and use` / `using`. */
typedef struct { const char *name; const char *mod; TypeKind ret; } UsingConst;
static const UsingConst _using_consts[] = {
    {"PI","math",TK_FLOAT},{"E","math",TK_FLOAT},{"TAU","math",TK_FLOAT},
    {"PHI","math",TK_FLOAT},{"SQRT2","math",TK_FLOAT},{"LN2","math",TK_FLOAT},
    {"LN10","math",TK_FLOAT},{"INF","math",TK_FLOAT},{"NEG_INF","math",TK_FLOAT},
    {"EPSILON","math",TK_FLOAT},
    {"MAC_OS","os",TK_INT},{"LINUX","os",TK_INT},{"WINDOWS","os",TK_INT},{"OTHER","os",TK_INT},
    {"READ_ONLY","io",TK_INT},{"WRITE_ONLY","io",TK_INT},{"READ_WRITE","io",TK_INT},
    {NULL,NULL,TK_UNKNOWN}
};

static bool tc_is_using_constant(TypeChecker *tc, const char *name) {
    for (int ui = 0; ui < tc->using_module_count; ui++) {
        const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
        for (int ci = 0; _using_consts[ci].name; ci++) {
            if (strcmp(name, _using_consts[ci].name) == 0 &&
                strcmp(real_mod, _using_consts[ci].mod) == 0) {
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], tc->using_modules[ui]) == 0 ||
                        strcmp(tc->imported_modules[mi], real_mod) == 0) {
                        tc->import_used[mi] = true;
                        break;
                    }
                }
                return true;
            }
        }
    }
    return false;
}

static EzType *tc_resolve_using_constant_type(TypeChecker *tc, const char *name) {
    for (int ui = 0; ui < tc->using_module_count; ui++) {
        const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
        for (int ci = 0; _using_consts[ci].name; ci++) {
            if (strcmp(name, _using_consts[ci].name) == 0 &&
                strcmp(real_mod, _using_consts[ci].mod) == 0) {
                switch (_using_consts[ci].ret) {
                case TK_FLOAT: return &TYPE_FLOAT;
                case TK_INT:   return &TYPE_INT;
                case TK_STRING: return &TYPE_STRING;
                default:       return &TYPE_UNKNOWN;
                }
            }
        }
    }
    return &TYPE_UNKNOWN;
}

/* --- Enum helpers --- */

static void register_enum(TypeChecker *tc, const char *name, bool is_string,
    const char **values, int value_count) {
    if (tc->enum_count >= tc->enum_cap) {
        tc->enum_cap = tc->enum_cap ? tc->enum_cap * 2 : 8;
        tc->enum_names = realloc(tc->enum_names, sizeof(const char *) * tc->enum_cap);
        tc->enum_is_string = realloc(tc->enum_is_string, sizeof(bool) * tc->enum_cap);
        tc->enum_values = realloc(tc->enum_values, sizeof(const char **) * tc->enum_cap);
        tc->enum_value_counts = realloc(tc->enum_value_counts, sizeof(int) * tc->enum_cap);
    }
    tc->enum_names[tc->enum_count] = name;
    tc->enum_is_string[tc->enum_count] = is_string;
    tc->enum_values[tc->enum_count] = values;
    tc->enum_value_counts[tc->enum_count] = value_count;
    tc->enum_count++;
}

static bool is_enum_name(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->enum_count; i++) {
        if (strcmp(tc->enum_names[i], name) == 0) return true;
    }
    return false;
}

/* Resolve a type name, returning TK_ENUM for known enum names instead of
 * the default TK_STRUCT that type_from_name() produces for uppercase names. */
static EzType *tc_type_from_name(TypeChecker *tc, const char *name) {
    if (name && is_enum_name(tc, name)) return type_enum(name);
    EzType *t = type_from_name(name);
    /* : try prefixed type names from using-modules so bare
     * "Point" resolves to "shapes_Point" when shapes is using'd.
     * type_from_name returns TK_STRUCT for any capitalized name
     * even if the struct isn't registered, so check is_struct_name
     * to see if the bare name actually exists before giving up. */
    if (name && name[0] >= 'A' && name[0] <= 'Z' &&
        !is_struct_name(tc, name) && !is_enum_name(tc, name)) {
        for (int ui = 0; ui < tc->using_module_count; ui++) {
            const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
            char prefixed[EZ_MSG_BUF_SIZE];
            snprintf(prefixed, sizeof(prefixed), "%s_%s", real_mod, name);
            if (is_enum_name(tc, prefixed)) return type_enum(prefixed);
            if (is_struct_name(tc, prefixed)) return type_struct(prefixed);
        }
        /* : after registration completes, reject uppercase names that
         * aren't registered as structs or enums. During registration we must
         * allow forward references, so only enforce this in later passes.
         * Exempt built-in types that are mapped directly in codegen without
         * struct registration (e.g. Error → EzError*). */
        if (!tc->registering && strcmp(name, "Error") != 0) {
            return &TYPE_UNKNOWN;
        }
    }
    return t;
}

/* --- Type signedness helpers --- */

/* Check if a TypeKind is any integer type (signed or unsigned) */
static bool is_int_kind(TypeKind k) {
    return k == TK_INT || k == TK_UINT || k == TK_BYTE;
}

static bool is_unsigned_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "uint") == 0 || strcmp(tn, "u8") == 0 ||
           strcmp(tn, "u16") == 0 || strcmp(tn, "u32") == 0 ||
           strcmp(tn, "u64") == 0 || strcmp(tn, "u128") == 0 ||
           strcmp(tn, "u256") == 0 || strcmp(tn, "byte") == 0;
}

static bool is_signed_int_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "int") == 0 || strcmp(tn, "i8") == 0 ||
           strcmp(tn, "i16") == 0 || strcmp(tn, "i32") == 0 ||
           strcmp(tn, "i64") == 0 || strcmp(tn, "i128") == 0 ||
           strcmp(tn, "i256") == 0;
}

/* --- Literal value extraction --- */

/* Try to extract a compile-time integer value from a literal expression.
 * Handles: NODE_INT_VALUE, PREFIX(-) NODE_INT_VALUE, and simple constant
 * binary expressions on literals. Returns true if a value was extracted. */
static bool try_get_literal_int(AstNode *node, int64_t *out) {
    if (!node) return false;
    if (node->kind == NODE_INT_VALUE) {
        *out = node->data.int_value.value;
        return true;
    }
    if (node->kind == NODE_PREFIX_EXPR && strcmp(node->data.prefix.op, "-") == 0 &&
        node->data.prefix.right && node->data.prefix.right->kind == NODE_INT_VALUE) {
        *out = -node->data.prefix.right->data.int_value.value;
        return true;
    }
    /* Simple constant folding for literal +, -, * */
    if (node->kind == NODE_INFIX_EXPR) {
        int64_t lv, rv;
        if (try_get_literal_int(node->data.infix.left, &lv) &&
            try_get_literal_int(node->data.infix.right, &rv)) {
            if (strcmp(node->data.infix.op, "+") == 0) { *out = lv + rv; return true; }
            if (strcmp(node->data.infix.op, "-") == 0) { *out = lv - rv; return true; }
            if (strcmp(node->data.infix.op, "*") == 0) { *out = lv * rv; return true; }
            if (strcmp(node->data.infix.op, "/") == 0 && rv != 0) { *out = lv / rv; return true; }
        }
    }
    return false;
}

/* Check if a literal integer value fits in the declared sized type.
 * Returns true if an error was emitted.
 *
 * value carries the signed bit pattern as parsed. For u64/uint, max_val is
 * UINT64_MAX which doesn't fit in int64_t; we compare via the unsigned
 * bit pattern in that case. */
static bool check_integer_range(DiagnosticList *diag, const char *file,
    int line, int col, const char *type_name_str, int64_t value) {
    int64_t min_val = 0, max_val = 0;
    uint64_t umax_val = 0;
    bool is_unsigned = false;
    bool is_u64 = false;

    if (strcmp(type_name_str, "i8") == 0)        { min_val = -128; max_val = 127; }
    else if (strcmp(type_name_str, "i16") == 0)   { min_val = -32768; max_val = 32767; }
    else if (strcmp(type_name_str, "i32") == 0)   { min_val = -2147483648LL; max_val = 2147483647; }
    else if (strcmp(type_name_str, "u8") == 0)    { min_val = 0; max_val = 255; is_unsigned = true; }
    else if (strcmp(type_name_str, "u16") == 0)   { min_val = 0; max_val = 65535; is_unsigned = true; }
    else if (strcmp(type_name_str, "u32") == 0)   { min_val = 0; max_val = 4294967295LL; is_unsigned = true; }
    else if (strcmp(type_name_str, "u64") == 0)   { umax_val = UINT64_MAX; is_unsigned = true; is_u64 = true; }
    else if (strcmp(type_name_str, "uint") == 0)  { umax_val = UINT64_MAX; is_unsigned = true; is_u64 = true; }
    else if (strcmp(type_name_str, "byte") == 0)  { min_val = 0; max_val = 255; is_unsigned = true; }
    else return false; /* not a range-checked type */

    bool out_of_range;
    if (is_u64) {
        /* u64/uint: any non-negative bit pattern fits (0..UINT64_MAX).
         * value < 0 means the literal carried a `-` sign. */
        out_of_range = value < 0;
    } else {
        out_of_range = value < min_val || value > max_val;
    }

    if (out_of_range) {
        char msg[EZ_MSG_BUF_SIZE];
        if (is_unsigned && value < 0) {
            if (is_u64) {
                snprintf(msg, sizeof(msg),
                    "value %lld is out of range for type '%s'; unsigned types cannot hold negative values (valid range: 0 to %llu)",
                    (long long)value, type_name_str, (unsigned long long)umax_val);
            } else {
                snprintf(msg, sizeof(msg),
                    "value %lld is out of range for type '%s'; unsigned types cannot hold negative values (valid range: %lld to %lld)",
                    (long long)value, type_name_str, (long long)min_val, (long long)max_val);
            }
        } else {
            snprintf(msg, sizeof(msg),
                "value %lld is out of range for type '%s' (valid range: %lld to %lld)",
                (long long)value, type_name_str, (long long)min_val, (long long)max_val);
        }
        diag_error_msg(diag, "E3036", strdup(msg), file, line, col, 0);
        return true;
    }
    return false;
}

/* --- Expression type resolution --- */

static EzType *resolve_expr(TypeChecker *tc, AstNode *node);

/* : shared void-expression guard. Emits E3038 at `expr` when `t`
 * is TK_VOID. `context` is a short phrase describing what the
 * position wants ("println argument", "map value", "binary operand",
 * etc.). If `expr` is a direct call to a named function, the error
 * quotes the function name; otherwise it falls back to a generic
 * "void expression" wording. Caller-suppliable context keeps each
 * diagnostic site self-describing without a zillion format strings. */
static AstNode *find_struct_in_program(AstNode *program, const char *name);

static void reject_void_in_context(TypeChecker *tc, AstNode *expr,
                                    EzType *t, const char *context) {
    if (!t || t->kind != TK_VOID || !expr) return;
    char msg[EZ_MSG_BUF_SIZE];
    if (expr->kind == NODE_CALL_EXPR && expr->data.call.function &&
        expr->data.call.function->kind == NODE_LABEL) {
        snprintf(msg, sizeof(msg),
            "cannot use void function '%s' as %s; the function does not return a value",
            expr->data.call.function->data.label.value, context);
    } else {
        snprintf(msg, sizeof(msg),
            "cannot use void expression as %s; the expression does not produce a value",
            context);
    }
    diag_error_msg(tc->diag, "E3038", strdup(msg),
        NODE_FILE(tc, expr), expr->token.line, expr->token.column, 0);
}

/* : emit E4005 at a stdlib call site where the function name
 * isn't recognized. Shared between every module dispatch branch that
 * has a fallthrough "unknown function" else. Without this, typing
 * `strings.totally_fake_function()` silently types as `string` (or
 * whatever the module's default-else set) and cascades into a
 * misleading downstream "cannot assign string to int" diagnostic,
 * hiding the real bug. */
static void emit_unknown_stdlib_fn(TypeChecker *tc, const char *mod,
                                    const char *mfn, AstNode *node) {
    diag_error_codef(tc->diag, "E4005", NODE_FILE(tc, node), node->token.line, node->token.column, 0, mod, mfn);
}

static EzType *resolve_expr(TypeChecker *tc, AstNode *node) {
    if (!node) return &TYPE_UNKNOWN;

    /* Memoize: if we already resolved this node, return the cached type.
     * This prevents duplicate diagnostics when the same subtree is walked
     * multiple times (e.g. builtin call args resolved by both the general
     * call path and the builtin-specific path).
     *
     * : skip the cache for call expressions during the re-check
     * pass (suppress_typetable_writes is true). The re-check walks
     * generic bodies with concrete parameter bindings; inner calls to
     * other generic functions need to re-run the dispatch to record
     * their concrete instantiations. Without this bypass, the cached
     * TK_UNKNOWN from the main pass short-circuits the resolution and
     * the inner function's binding never gets recorded. */
    EzType *cached = typetable_get(tc->type_table, node);
    /* : bypass the cache entirely during the re-check pass
     * (suppress_typetable_writes). The re-check walks generic bodies
     * with concrete param bindings; stale TK_UNKNOWN entries from the
     * main pass prevent inner generic calls from resolving their
     * concrete bindings. */
    if (cached && !tc->suppress_typetable_writes)
        return cached;

    EzType *result = &TYPE_UNKNOWN;

    switch (node->kind) {
    case NODE_INT_VALUE:
        result = &TYPE_INT;
        break;

    case NODE_FLOAT_VALUE:
        result = &TYPE_FLOAT;
        break;

    case NODE_STRING_VALUE:
        result = &TYPE_STRING;
        break;

    case NODE_INTERPOLATED_STRING:
        /* Resolve types of all interpolation parts */
        for (int i = 0; i < node->data.interpolated_string.part_count; i++) {
            AstNode *part = node->data.interpolated_string.parts[i];
            EzType *pt = resolve_expr(tc, part);
            /* Only check non-literal parts (the ${expr} expressions) */
            if (part->kind == NODE_STRING_VALUE || !pt) continue;
            /* Interpolation expressions are re-lexed by a sub-lexer on
             * the extracted ${...} text, so part tokens have positions
             * relative to that sub-stream; not the original file.
             * Always anchor diagnostics at the outer string literal's
             * location instead. */
            int line = node->token.line;
            int col = node->token.column;
            bool is_func_type = (pt->kind == TK_FUNCTION ||
                                   (pt->name && strcmp(pt->name, "func") == 0));
            if (pt->kind == TK_VOID) {
                diag_error_msg(tc->diag, "E3041",
                    strdup("cannot interpolate void expression; the function does not return a value"),
                    NODE_FILE(tc, node), line, col, 0);
            } else if (pt->kind == TK_STRUCT ||
                       pt->kind == TK_POINTER ||
                       is_func_type) {
                /* : interpolation codegen only handles scalars,
                 * strings, arrays, and maps. Structs, pointers, and
                 * func references fall through to a `%lld` + long-long
                 * cast in the generated C, which clang rejects. Catch
                 * it here with a targeted E3041 instead of leaking a
                 * raw C error, and nudge the user at the workaround
                 * (interpolate individual fields for structs). */
                char msg[EZ_MSG_BUF_SIZE];
                if (pt->kind == TK_STRUCT && pt->name) {
                    snprintf(msg, sizeof(msg),
                        "cannot interpolate struct value of type '%s'; format fields individually (e.g. \"${v.field}\")",
                        pt->name);
                } else if (pt->kind == TK_POINTER) {
                    snprintf(msg, sizeof(msg),
                        "cannot interpolate pointer value; dereference with ^ or format the pointee explicitly");
                } else {
                    snprintf(msg, sizeof(msg),
                        "cannot interpolate function reference; call the function or format its result");
                }
                diag_error_msg(tc->diag, "E3041", strdup(msg),
                    NODE_FILE(tc, node), line, col, 0);
            }
        }
        result = &TYPE_STRING;
        break;

    case NODE_BOOL_VALUE:
        result = &TYPE_BOOL;
        break;

    case NODE_CHAR_VALUE:
        result = &TYPE_CHAR;
        break;

    case NODE_NIL_VALUE:
        result = &TYPE_NIL;
        break;

    case NODE_LABEL: {
        const char *name = node->data.label.value;

        /* Type names used as values are caught downstream; they won't match
         * any variable in scope, and functions like new(), mem.make(), and casts
         * legitimately take type names as arguments. */

        Symbol *sym = scope_lookup(tc->current_scope, name);
        /* Try using-module-prefixed name if not found */
        if (!sym) {
            for (int ui = 0; ui < tc->using_module_count; ui++) {
                const char *umod = tc_resolve_alias(tc, tc->using_modules[ui]);
                char prefixed[EZ_MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", umod, name);
                sym = scope_lookup(tc->current_scope, prefixed);
                if (sym) {
                    /* Rewrite the label to the prefixed name so codegen finds it */
                    node->data.label.value = strdup(prefixed);
                    /* Mark module as used */
                    for (int mi = 0; mi < tc->import_count; mi++) {
                        if (strcmp(tc->imported_modules[mi], tc->using_modules[ui]) == 0 ||
                            strcmp(tc->imported_modules[mi], umod) == 0) {
                            tc->import_used[mi] = true;
                            break;
                        }
                    }
                    break;
                }
            }
        }
        if (sym) {
            sym->used = true;
            result = sym->type;
            /* Transparent ref: unwrap pointer to expose underlying type.
             * The codegen auto-derefs ref vars, so the typechecker must
             * see the dereferenced type for indexing, comparison, etc. */
            if (sym->is_ref && result->kind == TK_POINTER && result->element_type) {
                result = type_from_name(result->element_type);
            }
        } else if (find_func(tc, name)) {
            /* Bare function name used as a value (). Call sites
             * inspect node->data.call.function directly and never
             * recurse through resolve_expr for a LABEL callee, and
             * NODE_FUNC_REF reads its inner label without going
             * through here either; so any NODE_LABEL that resolves
             * to a function at this point is an unwrapped reference
             * in a value position and should be rejected. */
            FuncSig *fs = find_func(tc, name);
            if (fs) fs->used = true;
            diag_error_codef(tc->diag, "E3031", NODE_FILE(tc, node), node->token.line, node->token.column, 0, name, name, name);
        } else if (tc_is_using_constant(tc, name)) {
            result = tc_resolve_using_constant_type(tc, name);
        } else if (!is_enum_name(tc, name) &&
                   !tc_is_builtin(name) && !is_struct_name(tc, name) &&
                   !tc_is_imported_module(tc, name)) {
            /* Check if it looks like a number with a leading underscore */
            if (name[0] == '_' && name[1] >= '0' && name[1] <= '9') {
                diag_error_codef(tc->diag, "E1012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, name + 1);
            } else {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg), "undefined variable '%s'", name);
                /* Check if the name matches a named return value */
                bool is_named_return = false;
                const char *nr_type = NULL;
                if (tc->current_has_named_returns && tc->current_return_names) {
                    for (int i = 0; i < tc->current_return_count; i++) {
                        if (tc->current_return_names[i] &&
                            strcmp(tc->current_return_names[i], name) == 0) {
                            is_named_return = true;
                            nr_type = tc->current_return_type_names[i];
                            break;
                        }
                    }
                }
                if (is_named_return) {
                    char help[EZ_MSG_BUF_LARGE];
                    snprintf(help, sizeof(help),
                        "'%s' is a named return value in the function signature — did you forget to declare 'mut %s %s' at function scope?",
                        name, name, nr_type ? nr_type : "?");
                    diag_error_help(tc->diag, "E4001", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0, strdup(help));
                } else {
                    const char *suggestion = suggest_name(tc, name);
                    if (suggestion) {
                        char help[EZ_MSG_BUF_SIZE];
                        snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                        diag_error_help(tc->diag, "E4001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0, strdup(help));
                    } else {
                        diag_error_msg(tc->diag, "E4001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
            }
        }
        break;
    }

    case NODE_PREFIX_EXPR: {
        EzType *right = resolve_expr(tc, node->data.prefix.right);
        if (strcmp(node->data.prefix.op, "!") == 0) {
            result = &TYPE_BOOL;
        } else if (strcmp(node->data.prefix.op, "-") == 0) {
            if (right->kind != TK_UNKNOWN && !type_is_numeric(right)) {
                diag_error_codef(tc->diag, "E3007", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_name(right));
            }
            result = right;
        } else {
            result = right;
        }
        break;
    }

    case NODE_INFIX_EXPR: {
        EzType *left = resolve_expr(tc, node->data.infix.left);
        EzType *right = resolve_expr(tc, node->data.infix.right);
        const char *op = node->data.infix.op;

        /* : track whether any op-specific check has rejected the
         * expression so the final result can be collapsed to TK_UNKNOWN
         * instead of one operand's type. Otherwise `mut x int = true +
         * 1` fires E3002 at the '+' and then cascades into E3001 "can't
         * assign bool to int" at the var_decl, where the bool came from
         * the left operand rather than a real result type. */
        bool infix_errored = false;

        /* : void operands never make sense in any binary
         * operator. Check both sides at the kind level before any
         * op-specific diagnostic runs, so `1 + nothing()` and
         * `nothing() == x` report a clean E3038 instead of
         * cascading through the op-specific type checks or leaking
         * to clang. */
        if (left && left->kind == TK_VOID) infix_errored = true;
        if (right && right->kind == TK_VOID) infix_errored = true;
        reject_void_in_context(tc, node->data.infix.left, left, "binary operand");
        reject_void_in_context(tc, node->data.infix.right, right, "binary operand");

        /* String ordering operators not supported; use strings.compare() */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) &&
            (strcmp(op, "<") == 0 || strcmp(op, ">") == 0 ||
             strcmp(op, "<=") == 0 || strcmp(op, ">=") == 0)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "cannot use '%s' on strings; use strings.compare() instead", op);
            diag_error_msg(tc->diag, "E3002", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            result = &TYPE_BOOL;
            break;
        }

        /* E3002: modulo on float */
        if (strcmp(op, "%") == 0 &&
            (left->kind == TK_FLOAT || right->kind == TK_FLOAT)) {
            diag_error_msg(tc->diag, "E3002",
                strdup("modulo (%) only works on integers, not floats"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3002: literal divide/modulo by zero (). Catches the
         * statically-detectable case where the RHS is an integer or
         * float literal zero (including a prefix -0). Runtime checks
         * still cover the dynamic case. */
        if (strcmp(op, "/") == 0 || strcmp(op, "%") == 0) {
            AstNode *r = node->data.infix.right;
            bool is_zero = false;
            int64_t iv;
            if (try_get_literal_int(r, &iv) && iv == 0) {
                is_zero = true;
            } else if (r && r->kind == NODE_FLOAT_VALUE &&
                       r->data.float_value.value == 0.0) {
                is_zero = true;
            } else if (r && r->kind == NODE_PREFIX_EXPR &&
                       strcmp(r->data.prefix.op, "-") == 0 &&
                       r->data.prefix.right &&
                       r->data.prefix.right->kind == NODE_FLOAT_VALUE &&
                       r->data.prefix.right->data.float_value.value == 0.0) {
                is_zero = true;
            }
            if (is_zero) {
                char msg[EZ_TYPE_NAME_MAX];
                snprintf(msg, sizeof(msg),
                    "%s by zero; dividing by a literal zero is always invalid",
                    strcmp(op, "%") == 0 ? "modulo" : "division");
                diag_error_msg(tc->diag, "E3002", strdup(msg),
                    NODE_FILE(tc, r), r->token.line, r->token.column, 0);
                infix_errored = true;
            }
        }

        /* E3002: bool used in arithmetic (e.g., 1 + true) */
        if ((left->kind == TK_BOOL || right->kind == TK_BOOL) &&
            (strcmp(op, "+") == 0 || strcmp(op, "-") == 0 ||
             strcmp(op, "*") == 0 || strcmp(op, "/") == 0 ||
             strcmp(op, "%") == 0) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "invalid operands: cannot use '%s' with %s and %s",
                op, type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3002", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3002 (): nil in any operator other than equality. `nil
         * == x` and `nil != x` are valid against nullable types (the
         * existing comparison path already validates that); every
         * other operator on nil is nonsense. Catches both
         * `println(nil + 1)` (which leaked straight to clang) and
         * `mut x int = nil + 1` (which was caught by the downstream
         * nil-assignment check with a confusing message). */
        if ((left->kind == TK_NIL || right->kind == TK_NIL) &&
            strcmp(op, "==") != 0 && strcmp(op, "!=") != 0) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "cannot use nil with operator '%s'; nil is only valid for == / != against nullable types (Error, pointers)",
                op);
            diag_error_msg(tc->diag, "E3002", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* String + string: reject with helpful message */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) && strcmp(op, "+") == 0) {
            diag_error_code(tc->diag, "E3048", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* Arithmetic on strings (-, *, /, %, etc.) */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) &&
            strcmp(op, "+") != 0 && strcmp(op, "==") != 0 && strcmp(op, "!=") != 0 &&
            strcmp(op, "in") != 0 && strcmp(op, "not_in") != 0 && strcmp(op, "!in") != 0) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "cannot use '%s' on string type", op);
            diag_error_msg(tc->diag, "E3002", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3078: pointer arithmetic is not supported. The spec disallows
         * it (no contiguous-buffer guarantee on '^T'), and without this
         * check 'p + 1' silently emitted C pointer math and produced
         * garbage. Equality (== / !=) against nil or another pointer is
         * still allowed. */
        if ((left->kind == TK_POINTER || right->kind == TK_POINTER) &&
            (strcmp(op, "+") == 0 || strcmp(op, "-") == 0 ||
             strcmp(op, "*") == 0 || strcmp(op, "/") == 0 || strcmp(op, "%") == 0)) {
            diag_error_code(tc->diag, "E3078",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3032: different enum types in comparison */
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0) &&
            node->data.infix.left->kind == NODE_MEMBER_EXPR &&
            node->data.infix.right->kind == NODE_MEMBER_EXPR &&
            node->data.infix.left->data.member.object->kind == NODE_LABEL &&
            node->data.infix.right->data.member.object->kind == NODE_LABEL) {
            const char *lname = node->data.infix.left->data.member.object->data.label.value;
            const char *rname = node->data.infix.right->data.member.object->data.label.value;
            if (is_enum_name(tc, lname) && is_enum_name(tc, rname) &&
                strcmp(lname, rname) != 0) {
                diag_error_codef(tc->diag, "E3032", NODE_FILE(tc, node), node->token.line, node->token.column, 0, lname, rname);
            }
        }

        /* E3049: arithmetic on enum values */
        if (strcmp(op, "+") == 0 || strcmp(op, "-") == 0 ||
            strcmp(op, "*") == 0 || strcmp(op, "/") == 0 || strcmp(op, "%") == 0) {
            bool left_is_enum = node->data.infix.left->kind == NODE_MEMBER_EXPR &&
                node->data.infix.left->data.member.object->kind == NODE_LABEL &&
                is_enum_name(tc, node->data.infix.left->data.member.object->data.label.value);
            bool right_is_enum = node->data.infix.right->kind == NODE_MEMBER_EXPR &&
                node->data.infix.right->data.member.object->kind == NODE_LABEL &&
                is_enum_name(tc, node->data.infix.right->data.member.object->data.label.value);
            if (left_is_enum || right_is_enum) {
                diag_error_codef(tc->diag, "E3049", NODE_FILE(tc, node), node->token.line, node->token.column, 0, op);
            }
        }

        /* Comparison of incompatible types (e.g., int == string) */
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN &&
            left->kind != right->kind && left->kind != TK_NIL && right->kind != TK_NIL &&
            !(is_int_kind(left->kind) && is_int_kind(right->kind)) &&
            !(is_int_kind(left->kind) && right->kind == TK_ENUM) &&
            !(left->kind == TK_ENUM && is_int_kind(right->kind)) &&
            !(left->kind == TK_STRUCT && is_int_kind(right->kind)) &&
            !(is_int_kind(left->kind) && right->kind == TK_STRUCT) &&
            !(is_int_kind(left->kind) && right->kind == TK_BOOL) &&
            !(left->kind == TK_BOOL && is_int_kind(right->kind))) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "cannot compare %s with %s", type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3001", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        /* E3074: arrays cannot be compared with == / != directly. The C
         * backend has no structural-equality operator on aggregate types,
         * so this used to slip through to clang. Point users at
         * arrays.is_equal. */
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0 ||
             strcmp(op, "<") == 0 || strcmp(op, "<=") == 0 ||
             strcmp(op, ">") == 0 || strcmp(op, ">=") == 0) &&
            left->kind == TK_ARRAY && right->kind == TK_ARRAY) {
            diag_error_code(tc->diag, "E3074",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0 ||
             strcmp(op, "<") == 0 || strcmp(op, "<=") == 0 ||
             strcmp(op, ">") == 0 || strcmp(op, ">=") == 0) &&
            left->kind == TK_MAP && right->kind == TK_MAP) {
            diag_error_code(tc->diag, "E3076",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0 ||
             strcmp(op, "<") == 0 || strcmp(op, "<=") == 0 ||
             strcmp(op, ">") == 0 || strcmp(op, ">=") == 0) &&
            left->kind == TK_STRUCT && right->kind == TK_STRUCT) {
            diag_error_code(tc->diag, "E3077",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3085: validate type compatibility for in/not_in/!in */
        if ((strcmp(op, "in") == 0 || strcmp(op, "not_in") == 0 || strcmp(op, "!in") == 0) &&
            !infix_errored && left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            bool mismatch = false;
            const char *left_tn = type_name(left);
            const char *right_tn = type_name(right);
            if (right->kind == TK_ARRAY && right->element_type) {
                EzType *elem = type_from_name(right->element_type);
                if (elem->kind != TK_UNKNOWN && left->kind != elem->kind &&
                    !(is_int_kind(left->kind) && is_int_kind(elem->kind)) &&
                    !(left->name && strcmp(left->name, right->element_type) == 0)) {
                    mismatch = true;
                }
            } else if (right->kind == TK_MAP && right->key_type) {
                EzType *key = type_from_name(right->key_type);
                if (key->kind != TK_UNKNOWN && left->kind != key->kind &&
                    !(is_int_kind(left->kind) && is_int_kind(key->kind)) &&
                    !(left->name && strcmp(left->name, right->key_type) == 0)) {
                    mismatch = true;
                }
            } else if (right->kind == TK_STRING) {
                /* char in string and string in string are valid */
                if (left->kind != TK_CHAR && left->kind != TK_STRING) {
                    mismatch = true;
                }
            }
            if (mismatch) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "'in' operator type mismatch: cannot check if '%s' is in '%s'",
                    left_tn, right_tn);
                diag_error_msg(tc->diag, "E3085", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                infix_errored = true;
            }
        }

        if (strcmp(op, "==") == 0 || strcmp(op, "!=") == 0 ||
            strcmp(op, "<") == 0 || strcmp(op, ">") == 0 ||
            strcmp(op, "<=") == 0 || strcmp(op, ">=") == 0 ||
            strcmp(op, "&&") == 0 || strcmp(op, "||") == 0 ||
            strcmp(op, "in") == 0 || strcmp(op, "not_in") == 0 || strcmp(op, "!in") == 0) {
            result = &TYPE_BOOL;
        } else if (left->kind == TK_FLOAT || right->kind == TK_FLOAT) {
            result = &TYPE_FLOAT;
        } else if (left->kind == TK_STRING && strcmp(op, "+") == 0) {
            result = &TYPE_STRING;
        } else {
            result = left;
        }
        /* : if any op-level check rejected the expression, drop
         * result to TK_UNKNOWN so downstream var_decl / return / etc.
         * checks that already skip TK_UNKNOWN don't cascade a second
         * diagnostic off one of the (invalid) operand types. */
        if (infix_errored) result = &TYPE_UNKNOWN;
        break;
    }

    case NODE_POSTFIX_EXPR: {
        EzType *left_t = resolve_expr(tc, node->data.postfix.left);
        if (strcmp(node->data.postfix.op, "^") == 0) {
            if (left_t->kind == TK_POINTER) {
                /* Dereference: ^T^ → T */
                result = type_from_name(left_t->element_type);
            } else if (left_t->kind != TK_UNKNOWN) {
                diag_error_codef(tc->diag, "E3016", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_name(left_t));
                result = left_t;
            } else {
                result = left_t;
            }
        } else if (strcmp(node->data.postfix.op, "++") == 0 ||
                   strcmp(node->data.postfix.op, "--") == 0) {
            /* E5015: ++ and -- require a variable, not a literal */
            if (node->data.postfix.left->kind != NODE_LABEL &&
                node->data.postfix.left->kind != NODE_INDEX_EXPR &&
                node->data.postfix.left->kind != NODE_MEMBER_EXPR) {
                diag_error_msg(tc->diag, "E5015",
                    strdup("++ and -- require a variable; you cannot increment a literal or expression"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* ++ and -- only valid on mutable numeric types */
            if (node->data.postfix.left->kind == NODE_LABEL) {
                Symbol *sym = scope_lookup(tc->current_scope, node->data.postfix.left->data.label.value);
                if (sym && !sym->mutable) {
                    diag_error_codef(tc->diag, "E3005", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.postfix.left->data.label.value);
                }
            }
            if (left_t->kind != TK_UNKNOWN && !type_is_integer(left_t)) {
                diag_error_codef(tc->diag, "E5023", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.postfix.op, type_name(left_t));
            }
            result = left_t;
        } else {
            result = left_t;
        }
        break;
    }

    case NODE_CALL_EXPR: {
        /* Resolve argument types first. Skip the argument of ref()
         * when it's a bare function name; the ref() builtin handler
         * below resolves it specially, and the general resolve_expr
         * path would fire E3031 on the bare name ( follow-up). */
        bool is_ref_call = (node->data.call.function &&
            node->data.call.function->kind == NODE_LABEL &&
            strcmp(node->data.call.function->data.label.value, "ref") == 0);
        for (int i = 0; i < node->data.call.arg_count; i++) {
            if (is_ref_call && node->data.call.args[i]->kind == NODE_LABEL &&
                find_func(tc, node->data.call.args[i]->data.label.value)) {
                continue;
            }
            resolve_expr(tc, node->data.call.args[i]);
        }

        /* Resolve function return type */
        AstNode *fn = node->data.call.function;
        const char *fn_name = NULL;

        /* For non-LABEL/non-MEMBER callees (e.g. arr[i]() on a [func] array)
         * the specific branches below won't traverse the function expression,
         * leaving its subtree untyped and breaking codegen paths that rely on
         * typetable lookups. Resolve it here so those subtrees get populated. */
        if (fn && fn->kind != NODE_LABEL && fn->kind != NODE_MEMBER_EXPR) {
            resolve_expr(tc, fn);
        }

        /* [func] array + constant index + literal-of-func-refs origin:
         * recover the original referenced function's return type so it
         * survives the trip through void* storage (). */
        if (fn && fn->kind == NODE_INDEX_EXPR &&
            fn->data.index_expr.left->kind == NODE_LABEL &&
            fn->data.index_expr.index->kind == NODE_INT_VALUE) {
            const char *arr_name = fn->data.index_expr.left->data.label.value;
            Symbol *arr_sym = scope_lookup(tc->current_scope, arr_name);
            int64_t idx_v = fn->data.index_expr.index->data.int_value.value;
            if (arr_sym && arr_sym->func_array_refs &&
                idx_v >= 0 && idx_v < arr_sym->func_array_ref_count) {
                const char *ref_name = arr_sym->func_array_refs[idx_v];
                if (ref_name) {
                    FuncSig *rsig = find_func(tc, ref_name);
                    if (rsig && rsig->return_count > 0) {
                        result = rsig->return_types[0];
                        typetable_set(tc->type_table, node, result);
                        return result;
                    }
                }
            }
        }

        /* Fallback for [func(...)->T] arrays: parse the return type from
         * the array's typed element type. Covers dynamically appended
         * refs where func_array_refs isn't populated (). */
        if (fn && fn->kind == NODE_INDEX_EXPR &&
            fn->data.index_expr.left->kind == NODE_LABEL) {
            const char *arr_name = fn->data.index_expr.left->data.label.value;
            Symbol *arr_sym = scope_lookup(tc->current_scope, arr_name);
            if (arr_sym && arr_sym->type && arr_sym->type->kind == TK_ARRAY &&
                arr_sym->type->element_type &&
                strncmp(arr_sym->type->element_type, "func(", 5) == 0) {
                EzType *elem_t = type_from_name(arr_sym->type->element_type);
                if (elem_t && elem_t->func_sig &&
                    elem_t->func_sig->return_count > 0 &&
                    elem_t->func_sig->return_types[0]) {
                    result = type_from_name(elem_t->func_sig->return_types[0]);
                    typetable_set(tc->type_table, node, result);
                    return result;
                }
            }
        }

        if (fn->kind == NODE_LABEL) {
            fn_name = fn->data.label.value;
        } else if (fn->kind == NODE_MEMBER_EXPR &&
                   fn->data.member.object->kind == NODE_MEMBER_EXPR &&
                   fn->data.member.object->data.member.object->kind == NODE_LABEL) {
            /* mod.Struct.func() triple chain: geometry.Vec2.create() */
            const char *mod_name = fn->data.member.object->data.member.object->data.label.value;
            const char *struct_name = fn->data.member.object->data.member.member;
            const char *func_name = fn->data.member.member;
            /* Mark module as used */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], mod_name) == 0) {
                    tc->import_used[mi] = true;
                    break;
                }
            }
            /* Look up mod_Struct_func */
            char prefixed[EZ_MSG_BUF_SIZE];
            snprintf(prefixed, sizeof(prefixed), "%s_%s_%s", mod_name, struct_name, func_name);
            FuncSig *sig = find_func(tc, prefixed);
            if (sig) {
                sig->used = true;
                result = sig->return_count > 0 ? sig->return_types[0] : &TYPE_VOID;
                /* Check argument count */
                if (node->data.call.arg_count != sig->param_count) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "function '%s.%s.%s' expects %d argument(s), got %d",
                        mod_name, struct_name, func_name,
                        sig->param_count, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                /* E3027: mutable param checks for triple-namespaced calls.
                 * Struct functions live inside NODE_STRUCT_DECL, so scan
                 * struct declarations in imported modules. */
                int check_count = node->data.call.arg_count < sig->param_count
                    ? node->data.call.arg_count : sig->param_count;
                for (int ai = 0; ai < check_count; ai++) {
                    resolve_expr(tc, node->data.call.args[ai]);
                    AstNode *arg = node->data.call.args[ai];
                    AstNode *found_decl = NULL;
                    for (int fi = 0; fi < tc->program->data.program.stmt_count && !found_decl; fi++) {
                        AstNode *s = tc->program->data.program.stmts[fi];
                        if (s->kind == NODE_STRUCT_DECL &&
                            strcmp(s->data.struct_decl.name, struct_name) == 0) {
                            for (int sfi = 0; sfi < s->data.struct_decl.func_count; sfi++) {
                                AstNode *sf = s->data.struct_decl.funcs[sfi].func_decl;
                                if (sf && sf->kind == NODE_FUNC_DECL &&
                                    strcmp(sf->data.func_decl.name, func_name) == 0 &&
                                    ai < sf->data.func_decl.param_count &&
                                    sf->data.func_decl.params[ai].mutable) {
                                    found_decl = sf;
                                    break;
                                }
                            }
                        }
                    }
                    if (found_decl) {
                        if (arg->kind == NODE_LABEL) {
                            Symbol *arg_sym = scope_lookup(tc->current_scope,
                                arg->data.label.value);
                            if (arg_sym && !arg_sym->mutable) {
                                char emsg[256];
                                snprintf(emsg, sizeof(emsg),
                                    "cannot pass constant '%s' to mutable parameter '%s' of '%s.%s.%s'",
                                    arg_sym->name, found_decl->data.func_decl.params[ai].name,
                                    mod_name, struct_name, func_name);
                                diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                    NODE_FILE(tc, arg), arg->token.line,
                                    arg->token.column, 0);
                            }
                        } else if (arg->kind != NODE_MEMBER_EXPR &&
                                   arg->kind != NODE_INDEX_EXPR &&
                                   arg->kind != NODE_PREFIX_EXPR) {
                            char emsg[256];
                            snprintf(emsg, sizeof(emsg),
                                "cannot pass a literal or expression to mutable parameter '%s' of '%s.%s.%s'; expected a mutable variable",
                                found_decl->data.func_decl.params[ai].name,
                                mod_name, struct_name, func_name);
                            diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                NODE_FILE(tc, arg), arg->token.line,
                                arg->token.column, 0);
                        }
                    }
                }
            } else {
                result = &TYPE_VOID;
            }
        } else if (fn->kind == NODE_MEMBER_EXPR && fn->data.member.object->kind == NODE_LABEL) {
            const char *mod_raw = fn->data.member.object->data.label.value;
            const char *mod = tc_resolve_alias(tc, mod_raw);
            const char *mfn = fn->data.member.member;
            /* Check that module is actually imported */
            bool mod_imported = false;
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], mod_raw) == 0 ||
                    strcmp(tc->imported_modules[mi], mod) == 0) {
                    tc->import_used[mi] = true;
                    mod_imported = true;
                    break;
                }
            }
            if (!mod_imported && is_valid_module(mod)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "module '%s' is not imported; add 'import @%s' at the top of the file",
                    mod, mod);
                diag_error_msg(tc->diag, "E4001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* c.func() without import c"..."; but only if "c" isn't a
             * local variable. A variable named `c` with a struct type
             * should fall through to the struct-method dispatch, not
             * be treated as the C interop module (). */
            if (!mod_imported && strcmp(mod, "c") == 0 &&
                !scope_lookup(tc->current_scope, "c")) {
                diag_error_msg(tc->diag, "E4001",
                    strdup("C interop requires a C header import; add import c\"header.h\" at the top of the file"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                result = &TYPE_UNKNOWN;
                break;
            }
            /* C interop: c.func(); skip type checking but reject bigints */
            if (strcmp(mod, "c") == 0 && mod_imported) {
                /* Mark all C imports as used */
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], "c") == 0)
                        tc->import_used[mi] = true;
                }
                /* Validate arguments; reject types that don't translate to C */
                for (int ai = 0; ai < node->data.call.arg_count; ai++) {
                    EzType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                    if (!arg_t || arg_t->kind == TK_UNKNOWN) continue;
                    /* Reject bigint types */
                    if (arg_t->name &&
                        (strcmp(arg_t->name, "i128") == 0 || strcmp(arg_t->name, "i256") == 0 ||
                         strcmp(arg_t->name, "u128") == 0 || strcmp(arg_t->name, "u256") == 0)) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "cannot pass %s to a C function; C has no 128/256-bit integer types",
                            arg_t->name);
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                    /* Reject EZ-specific composite types */
                    if (arg_t->kind == TK_ARRAY || arg_t->kind == TK_MAP) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "cannot pass %s to a C function; use individual elements instead",
                            arg_t->kind == TK_ARRAY ? "an array" : "a map");
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                    /* Reject EZ structs (registered in typechecker) */
                    if (arg_t->kind == TK_STRUCT && arg_t->name &&
                        is_struct_name(tc, arg_t->name)) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "cannot pass struct '%s' to a C function; pass individual fields instead",
                            arg_t->name);
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                }
                result = &TYPE_UNKNOWN;
                break;
            }
            /* Skip stdlib dispatch if this module is a user import, not stdlib.
             * User modules with the same name (e.g., import "./server.ez") must
             * fall through to the user-module handler below. */
            if (!tc_is_stdlib_import(tc, mod_raw)) {
                goto user_module_dispatch;
            }
            if (strcmp(mod, "mem") == 0) {
                if (strcmp(mfn, "arena") == 0) {
                    result = type_struct("Arena"); /* arena pointer; opaque */
                } else if (strcmp(mfn, "usage") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "init") == 0 && node->data.call.arg_count == 2) {
                    /* mem.init(arena, Type) returns ^Type */
                    AstNode *type_arg = node->data.call.args[1];
                    if (type_arg->kind == NODE_LABEL) {
                        result = type_pointer(type_arg->data.label.value);
                    } else {
                        result = &TYPE_UNKNOWN;
                    }
                } else if (strcmp(mfn, "alloc") == 0 && node->data.call.arg_count == 2) {
                    /* alloc returns the same type as its second argument */
                    result = resolve_expr(tc, node->data.call.args[1]);
                } else if (strcmp(mfn, "make") == 0) {
                    result = &TYPE_UNKNOWN;
                } else if (strcmp(mfn, "free") == 0 || strcmp(mfn, "reset") == 0 ||
                           strcmp(mfn, "destroy") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "maps") == 0) {
                if (strcmp(mfn, "get_keys") == 0) {
                    if (node->data.call.arg_count > 0) {
                        EzType *map_t = resolve_expr(tc, node->data.call.args[0]);
                        result = type_array(map_t && map_t->key_type ? map_t->key_type : "string");
                    } else result = type_array("string");
                } else if (strcmp(mfn, "get_values") == 0) {
                    if (node->data.call.arg_count > 0) {
                        EzType *map_t = resolve_expr(tc, node->data.call.args[0]);
                        result = type_array(map_t && map_t->value_type ? map_t->value_type : "string");
                    } else result = type_array("string");
                } else if (strcmp(mfn, "has_key") == 0 || strcmp(mfn, "is_empty") == 0 ||
                           strcmp(mfn, "contains_value") == 0 ||
                           strcmp(mfn, "is_equal") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "merge") == 0) {
                    if (node->data.call.arg_count > 0) {
                        result = resolve_expr(tc, node->data.call.args[0]);
                    } else result = &TYPE_UNKNOWN;
                } else if (strcmp(mfn, "get_or_default") == 0) {
                    if (node->data.call.arg_count >= 3) {
                        result = resolve_expr(tc, node->data.call.args[2]);
                    } else result = &TYPE_UNKNOWN;
                } else if (strcmp(mfn, "remove_key") == 0 || strcmp(mfn, "clear") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E12001: maps functions require map argument */
                if (node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    EzType *arg0_t = typetable_get(tc->type_table, arg0);
                    if (arg0_t && arg0_t->kind == TK_ARRAY) {
                        diag_error_codef(tc->diag, "E12001", NODE_FILE(tc, arg0), arg0->token.line, arg0->token.column, 0, mfn);
                    }
                }
                /* maps.is_equal(a, b): both args must be maps with the same
                 * key/value types, and key/value types must be primitive or
                 * string. Composite element types would require recursive
                 * deep-equality which is out of scope here. */
                if (strcmp(mfn, "is_equal") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *a0 = node->data.call.args[0];
                    AstNode *a1 = node->data.call.args[1];
                    EzType *t0 = typetable_get(tc->type_table, a0);
                    EzType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_MAP && t1->kind == TK_MAP) {
                        bool key_match = t0->key_type && t1->key_type &&
                            strcmp(t0->key_type, t1->key_type) == 0;
                        bool val_match = t0->value_type && t1->value_type &&
                            strcmp(t0->value_type, t1->value_type) == 0;
                        if (!key_match || !val_match) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "type mismatch: cannot compare map[%s:%s] with map[%s:%s]",
                                t0->key_type ? t0->key_type : "?",
                                t0->value_type ? t0->value_type : "?",
                                t1->key_type ? t1->key_type : "?",
                                t1->value_type ? t1->value_type : "?");
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                        }
                        const char *bad_member = NULL;
                        if (t0->value_type) {
                            EzType *vt = type_from_name(t0->value_type);
                            if (vt->kind == TK_ARRAY || vt->kind == TK_MAP || vt->kind == TK_STRUCT)
                                bad_member = t0->value_type;
                        }
                        if (bad_member) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "maps.is_equal does not support maps with %s values; only primitive and string element types are supported",
                                bad_member);
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0);
                        }
                    }
                }
                /* E5007: mutating map functions on const map */
                if ((strcmp(mfn, "clear") == 0 || strcmp(mfn, "remove_key") == 0) &&
                    node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    if (arg0->kind == NODE_LABEL) {
                        Symbol *sym = scope_lookup(tc->current_scope, arg0->data.label.value);
                        if (sym && !sym->mutable) {
                            diag_error_codef(tc->diag, "E5007",
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                                "map", arg0->data.label.value);
                        }
                    }
                }
            } else if (strcmp(mod, "io") == 0) {
                /* Fallible I/O: the type checker returns the primary value type.
                 * The codegen emits _result() versions that return (T, Error) tuples.
                 * The .v0 access gets __auto_type in C, but the EZ type system
                 * sees it as the value type for interpolation purposes. */
                if (strcmp(mfn, "read_file") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "write_file") == 0 ||
                    strcmp(mfn, "delete_file") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "file_exists") == 0 ||
                           strcmp(mfn, "is_file") == 0 || strcmp(mfn, "is_directory") == 0 ||
                           strcmp(mfn, "append_file") == 0 || strcmp(mfn, "rename_file") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "file_size") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "list_dir") == 0 || strcmp(mfn, "walk") == 0 ||
                           strcmp(mfn, "glob") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "make_dir") == 0 || strcmp(mfn, "make_dir_all") == 0 ||
                           strcmp(mfn, "remove_dir") == 0 || strcmp(mfn, "remove_dir_all") == 0 ||
                           strcmp(mfn, "copy_file") == 0 || strcmp(mfn, "move_file") == 0 ||
                           strcmp(mfn, "is_absolute") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "path_join") == 0 || strcmp(mfn, "dirname") == 0 ||
                           strcmp(mfn, "basename") == 0 || strcmp(mfn, "extension") == 0 ||
                           strcmp(mfn, "normalize") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "strings") == 0) {
                if (strcmp(mfn, "contains") == 0 || strcmp(mfn, "starts_with") == 0 ||
                    strcmp(mfn, "ends_with") == 0 || strcmp(mfn, "is_empty") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index_of") == 0 ||
                           strcmp(mfn, "count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "split") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "to_upper") == 0 || strcmp(mfn, "to_lower") == 0 ||
                           strcmp(mfn, "trim") == 0 || strcmp(mfn, "trim_left") == 0 ||
                           strcmp(mfn, "trim_right") == 0 || strcmp(mfn, "replace") == 0 ||
                           strcmp(mfn, "repeat") == 0 || strcmp(mfn, "reverse") == 0 ||
                           strcmp(mfn, "slice") == 0 || strcmp(mfn, "join") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E7004: strings.repeat() second arg must be integer */
                if (strcmp(mfn, "repeat") == 0 && node->data.call.arg_count >= 2) {
                    EzType *count_t = typetable_get(tc->type_table, node->data.call.args[1]);
                    if (count_t && count_t->kind == TK_FLOAT) {
                        diag_error_msg(tc->diag, "E7004",
                            strdup("strings.repeat() count must be an integer, not a float"),
                            NODE_FILE(tc, node->data.call.args[1]), node->data.call.args[1]->token.line,
                            node->data.call.args[1]->token.column, 0);
                    }
                }
                /* E7004: strings.slice() bounds must be integers */
                if (strcmp(mfn, "slice") == 0 && node->data.call.arg_count >= 3) {
                    for (int si = 1; si <= 2 && si < node->data.call.arg_count; si++) {
                        EzType *bt = typetable_get(tc->type_table, node->data.call.args[si]);
                        if (bt && bt->kind == TK_FLOAT) {
                            diag_error_msg(tc->diag, "E7004",
                                strdup("strings.slice() bounds must be integers, not floats"),
                                NODE_FILE(tc, node->data.call.args[si]), node->data.call.args[si]->token.line,
                                node->data.call.args[si]->token.column, 0);
                        }
                    }
                }
            } else if (strcmp(mod, "time") == 0) {
                if (strcmp(mfn, "format") == 0 || strcmp(mfn, "to_iso") == 0 ||
                    strcmp(mfn, "date") == 0 || strcmp(mfn, "to_clock") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "now") == 0 || strcmp(mfn, "now_ms") == 0 ||
                           strcmp(mfn, "now_ns") == 0 || strcmp(mfn, "tick") == 0 ||
                           strcmp(mfn, "elapsed_ms") == 0 ||
                           strcmp(mfn, "year") == 0 || strcmp(mfn, "month") == 0 ||
                           strcmp(mfn, "day") == 0 || strcmp(mfn, "hour") == 0 ||
                           strcmp(mfn, "minute") == 0 || strcmp(mfn, "second") == 0 ||
                           strcmp(mfn, "weekday") == 0) {
                    result = &TYPE_INT;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "uuid") == 0) {
                if (strcmp(mfn, "is_valid") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "generate") == 0 || strcmp(mfn, "generate_hyphenated") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "encoding") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(mod, "crypto") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(mod, "bytes") == 0) {
                if (strcmp(mfn, "from_string") == 0 || strcmp(mfn, "from_hex") == 0 ||
                    strcmp(mfn, "from_base64") == 0) {
                    result = type_array("byte");
                } else if (strcmp(mfn, "to_string") == 0 || strcmp(mfn, "to_hex") == 0 ||
                           strcmp(mfn, "to_base64") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "binary") == 0) {
                if (strncmp(mfn, "encode", 6) == 0) result = type_array("byte");
                else if (strncmp(mfn, "decode_f", 8) == 0) result = &TYPE_FLOAT;
                else result = &TYPE_INT;
            } else if (strcmp(mod, "csv") == 0) {
                if (strcmp(mfn, "decode") == 0 || strcmp(mfn, "parse") == 0 ||
                    strcmp(mfn, "read_file") == 0 || strcmp(mfn, "headers") == 0) {
                    result = type_array("array");
                } else if (strcmp(mfn, "write_file") == 0) {
                    result = &TYPE_BOOL;
                    /* E5026: second arg must be an array, not a string */
                    if (node->data.call.arg_count >= 2) {
                        EzType *arg2_type = resolve_expr(tc, node->data.call.args[1]);
                        if (arg2_type && arg2_type->kind == TK_STRING) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "csv.%s() expects an array as the second argument, got string",
                                mfn);
                            diag_error_msg(tc->diag, "E5026", strdup(msg),
                                NODE_FILE(tc, node), node->data.call.args[1]->token.line,
                                node->data.call.args[1]->token.column, 0);
                        }
                    }
                } else if (strcmp(mfn, "format") == 0 || strcmp(mfn, "encode") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "json") == 0) {
                if (strcmp(mfn, "decode") == 0) result = type_from_name("map[string:string]");
                else if (strcmp(mfn, "is_valid") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "parse") == 0) {
                    /* : json.parse() return type depends on
                     * assignment context. Start as UNKNOWN; the
                     * var_decl handler pushes the declared struct
                     * type onto the call node via 's mechanism. */
                    result = &TYPE_UNKNOWN;
                } else if (strcmp(mfn, "encode") == 0 || strcmp(mfn, "stringify") == 0 ||
                           strcmp(mfn, "format") == 0 || strcmp(mfn, "pretty_print") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "sqlite") == 0) {
                if (strcmp(mfn, "open") == 0) result = &TYPE_UNKNOWN; /* opaque handle */
                else if (strcmp(mfn, "exec") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "query") == 0) result = type_array("map");
                else if (strcmp(mfn, "close") == 0) result = &TYPE_VOID;
                else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "random") == 0) {
                if (strcmp(mfn, "rand_float") == 0) result = &TYPE_FLOAT;
                else if (strcmp(mfn, "rand_int") == 0) result = &TYPE_INT;
                else if (strcmp(mfn, "rand_bool") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "rand_byte") == 0) result = &TYPE_BYTE;
                else if (strcmp(mfn, "rand_char") == 0) result = &TYPE_CHAR;
                else if (strcmp(mfn, "shuffle") == 0 || strcmp(mfn, "sample") == 0) {
                    /* Preserve input array's element type */
                    if (node->data.call.arg_count > 0) {
                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type)
                            result = type_array(arr_t->element_type);
                        else
                            result = type_array("int");
                    } else {
                        result = type_array("int");
                    }
                } else if (strcmp(mfn, "choice") == 0) {
                    /* Return element type of the array argument */
                    if (node->data.call.arg_count > 0) {
                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type)
                            result = type_from_name(arr_t->element_type);
                        else
                            result = &TYPE_INT;
                    } else {
                        result = &TYPE_INT;
                    }
                } else if (strcmp(mfn, "random_hex") == 0) result = &TYPE_STRING;
                else if (strcmp(mfn, "seed") == 0) result = &TYPE_VOID;
                else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "arrays") == 0) {
                if (strcmp(mfn, "is_empty") == 0 || strcmp(mfn, "contains") == 0 ||
                    strcmp(mfn, "is_equal") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index_of") == 0 || strcmp(mfn, "get_sum") == 0 ||
                           strcmp(mfn, "get_min") == 0 || strcmp(mfn, "get_max") == 0 ||
                           strcmp(mfn, "count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "reverse") == 0 || strcmp(mfn, "slice") == 0 ||
                           strcmp(mfn, "concat") == 0 || strcmp(mfn, "deduplicate") == 0) {
                    /* Preserve input array element type */
                    if (node->data.call.arg_count > 0) {
                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arr_t && arr_t->element_type) ? type_array(arr_t->element_type) : type_array("int");
                    } else {
                        result = type_array("int");
                    }
                } else if (strcmp(mfn, "split_every") == 0 || strcmp(mfn, "pair") == 0) {
                    result = type_array("[int]"); /* nested array */
                } else if (strcmp(mfn, "flatten") == 0) {
                    if (node->data.call.arg_count > 0) {
                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arr_t && arr_t->element_type) ? type_array(arr_t->element_type) : type_array("int");
                    } else {
                        result = type_array("int");
                    }
                } else if (strcmp(mfn, "get_first") == 0 || strcmp(mfn, "get_last") == 0 ||
                           strcmp(mfn, "remove_last") == 0 || strcmp(mfn, "remove_first") == 0) {
                    if (node->data.call.arg_count > 0) {
                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arr_t && arr_t->element_type) ? type_from_name(arr_t->element_type) : &TYPE_INT;
                    } else {
                        result = &TYPE_INT;
                    }
                } else if (strcmp(mfn, "append") == 0 || strcmp(mfn, "prepend") == 0 ||
                           strcmp(mfn, "insert") == 0 || strcmp(mfn, "insert_at") == 0 ||
                           strcmp(mfn, "remove") == 0 || strcmp(mfn, "remove_at") == 0 ||
                           strcmp(mfn, "fill") == 0 || strcmp(mfn, "set") == 0 ||
                           strcmp(mfn, "sort_asc") == 0 || strcmp(mfn, "sort_desc") == 0 ||
                           strcmp(mfn, "clear") == 0 || strcmp(mfn, "sum") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E5007: mutating array functions on const array */
                if ((strcmp(mfn, "append") == 0 || strcmp(mfn, "insert") == 0 ||
                     strcmp(mfn, "insert_at") == 0 ||
                     strcmp(mfn, "remove") == 0 || strcmp(mfn, "remove_at") == 0 ||
                     strcmp(mfn, "remove_last") == 0 ||
                     strcmp(mfn, "remove_first") == 0 || strcmp(mfn, "prepend") == 0 ||
                     strcmp(mfn, "fill") == 0 || strcmp(mfn, "set") == 0 ||
                     strcmp(mfn, "sort_asc") == 0 || strcmp(mfn, "sort_desc") == 0 ||
                     strcmp(mfn, "clear") == 0) &&
                    node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    if (arg0->kind == NODE_LABEL) {
                        Symbol *sym = scope_lookup(tc->current_scope, arg0->data.label.value);
                        if (sym && !sym->mutable) {
                            diag_error_codef(tc->diag, "E5007",
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                                "array", arg0->data.label.value);
                        }
                    }
                }
                /* E3001: arrays.append/prepend/insert_at element type mismatch */
                {
                    AstNode *val_node = NULL;
                    const char *op_name = NULL;
                    if ((strcmp(mfn, "append") == 0 || strcmp(mfn, "prepend") == 0) &&
                        node->data.call.arg_count >= 2) {
                        val_node = node->data.call.args[1];
                        op_name = mfn;
                    } else if (strcmp(mfn, "insert_at") == 0 && node->data.call.arg_count >= 3) {
                        val_node = node->data.call.args[2];
                        op_name = "insert_at";
                    }
                    if (val_node && op_name) {
                        AstNode *arr_arg = node->data.call.args[0];
                        EzType *arr_t = typetable_get(tc->type_table, arr_arg);
                        if (!arr_t) arr_t = resolve_expr(tc, arr_arg);
                        EzType *val_t = resolve_expr(tc, val_node);
                        if (arr_t && arr_t->kind != TK_ARRAY && arr_t->kind != TK_UNKNOWN) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "arrays.%s() expects an array as the first argument, got %s",
                                op_name, type_name(arr_t));
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, arr_arg), arr_arg->token.line, arr_arg->token.column, 0);
                        } else if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
                            val_t && val_t->kind != TK_UNKNOWN) {
                            EzType *elem_t = type_from_name(arr_t->element_type);
                            if (elem_t->kind != TK_UNKNOWN && elem_t->kind != val_t->kind &&
                                !(is_int_kind(elem_t->kind) && is_int_kind(val_t->kind))) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "type mismatch in arrays.%s(); cannot add %s to array of %s",
                                    op_name, type_name(val_t), arr_t->element_type);
                                diag_error_msg(tc->diag, "E3001", strdup(msg),
                                    NODE_FILE(tc, val_node), val_node->token.line, val_node->token.column, 0);
                            }
                        }
                    }
                }
                /* E9002: arrays.sum/min/max require numeric array */
                if ((strcmp(mfn, "sum") == 0 || strcmp(mfn, "min") == 0 ||
                     strcmp(mfn, "max") == 0) && node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    EzType *arr_t = typetable_get(tc->type_table, arg0);
                    if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
                        EzType *elem_t = type_from_name(arr_t->element_type);
                        if (elem_t->kind == TK_STRING || elem_t->kind == TK_BOOL) {
                            diag_error_codef(tc->diag, "E9002", NODE_FILE(tc, arg0), arg0->token.line, arg0->token.column, 0, mfn, arr_t->element_type);
                        }
                    }
                }
                /* E3001: arrays.concat element type mismatch */
                if (strcmp(mfn, "concat") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *a0 = node->data.call.args[0];
                    AstNode *a1 = node->data.call.args[1];
                    EzType *t0 = typetable_get(tc->type_table, a0);
                    EzType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_ARRAY && t1->kind == TK_ARRAY &&
                        t0->element_type && t1->element_type &&
                        strcmp(t0->element_type, t1->element_type) != 0) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "type mismatch: cannot concat array of %s with array of %s",
                            t0->element_type, t1->element_type);
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                    }
                }
                if (strcmp(mfn, "is_equal") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *a0 = node->data.call.args[0];
                    AstNode *a1 = node->data.call.args[1];
                    EzType *t0 = typetable_get(tc->type_table, a0);
                    EzType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_ARRAY && t1->kind == TK_ARRAY &&
                        t0->element_type && t1->element_type &&
                        strcmp(t0->element_type, t1->element_type) != 0) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "type mismatch: cannot compare array of %s with array of %s",
                            t0->element_type, t1->element_type);
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                    }
                    /* Composite element types (nested arrays, maps, structs) cannot be
                     * compared with the primitive memcmp path; reject for now. */
                    if (t0 && t0->kind == TK_ARRAY && t0->element_type) {
                        EzType *et = type_from_name(t0->element_type);
                        if (et->kind == TK_ARRAY || et->kind == TK_MAP || et->kind == TK_STRUCT) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "arrays.is_equal does not support arrays of %s; only primitive and string element types are supported",
                                t0->element_type);
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0);
                        }
                    }
                }
            } else if (strcmp(mod, "os") == 0) {
                if (strcmp(mfn, "args") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "get_env") == 0 ||
                           strcmp(mfn, "current_dir") == 0 || strcmp(mfn, "hostname") == 0 ||
                           strcmp(mfn, "arch") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "current_os") == 0 || strcmp(mfn, "pid") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "set_env") == 0 ||
                           strcmp(mfn, "exit") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "math") == 0) {
                /* Math functions return types */
                if (strcmp(mfn, "is_prime") == 0 || strcmp(mfn, "is_even") == 0 ||
                    strcmp(mfn, "is_odd") == 0 ||
                    strcmp(mfn, "is_nan") == 0 || strcmp(mfn, "is_finite") == 0 ||
                    strcmp(mfn, "is_infinite") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "abs") == 0 || strcmp(mfn, "neg") == 0 ||
                           strcmp(mfn, "min") == 0 || strcmp(mfn, "max") == 0 ||
                           strcmp(mfn, "clamp") == 0) {
                    /* Return type matches argument type; float in, float out */
                    if (node->data.call.arg_count > 0) {
                        EzType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arg_t && arg_t->kind == TK_FLOAT) ? &TYPE_FLOAT : &TYPE_INT;
                    } else {
                        result = &TYPE_INT;
                    }
                } else if (strcmp(mfn, "sign") == 0 || strcmp(mfn, "factorial") == 0 ||
                           strcmp(mfn, "gcd") == 0 || strcmp(mfn, "lcm") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "sqrt") == 0 || strcmp(mfn, "pow") == 0 ||
                           strcmp(mfn, "exp") == 0 || strcmp(mfn, "exp2") == 0 ||
                           strcmp(mfn, "log") == 0 ||
                           strcmp(mfn, "log2") == 0 || strcmp(mfn, "log10") == 0 ||
                           strcmp(mfn, "log_base") == 0 ||
                           strcmp(mfn, "sin") == 0 || strcmp(mfn, "cos") == 0 ||
                           strcmp(mfn, "tan") == 0 || strcmp(mfn, "asin") == 0 ||
                           strcmp(mfn, "acos") == 0 || strcmp(mfn, "atan") == 0 ||
                           strcmp(mfn, "atan2") == 0 || strcmp(mfn, "sinh") == 0 ||
                           strcmp(mfn, "cosh") == 0 || strcmp(mfn, "tanh") == 0 ||
                           strcmp(mfn, "floor") == 0 || strcmp(mfn, "ceil") == 0 ||
                           strcmp(mfn, "round") == 0 || strcmp(mfn, "trunc") == 0 ||
                           strcmp(mfn, "cbrt") == 0 || strcmp(mfn, "hypot") == 0 ||
                           strcmp(mfn, "deg_to_rad") == 0 || strcmp(mfn, "rad_to_deg") == 0 ||
                           strcmp(mfn, "lerp") == 0 || strcmp(mfn, "distance") == 0 ||
                           strcmp(mfn, "mod") == 0 || strcmp(mfn, "pi") == 0 ||
                           strcmp(mfn, "e") == 0 || strcmp(mfn, "tau") == 0) {
                    result = &TYPE_FLOAT;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "threads") == 0) {
                if (strcmp(mfn, "spawn") == 0) {
                    /* Validate first arg is a function reference */
                    if (node->data.call.arg_count >= 1) {
                        AstNode *arg0 = node->data.call.args[0];
                        if (arg0->kind != NODE_FUNC_REF &&
                            !(arg0->kind == NODE_CALL_EXPR &&
                              arg0->data.call.function->kind == NODE_LABEL &&
                              strcmp(arg0->data.call.function->data.label.value, "ref") == 0)) {
                            diag_error_msg(tc->diag, "E7006",
                                strdup("threads.spawn() requires a function reference; use ()func_name or ref(func_name)"),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                    result = type_struct("Thread"); /* EzThread; opaque */
                } else if (strcmp(mfn, "get_id") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "join") == 0 || strcmp(mfn, "sleep_s") == 0 ||
                           strcmp(mfn, "sleep_ms") == 0 || strcmp(mfn, "sleep_ns") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "sync") == 0) {
                if (strcmp(mfn, "mutex") == 0) {
                    result = type_struct("Mutex"); /* EzMutex; opaque */
                } else if (strcmp(mfn, "try_lock") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "lock") == 0 || strcmp(mfn, "unlock") == 0 ||
                           strcmp(mfn, "destroy") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "atomic") == 0) {
                if (strcmp(mfn, "load") == 0 || strcmp(mfn, "add") == 0 ||
                    strcmp(mfn, "sub") == 0 || strcmp(mfn, "exchange") == 0 ||
                    strcmp(mfn, "and") == 0 || strcmp(mfn, "or") == 0 ||
                    strcmp(mfn, "xor") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "cas") == 0 || strcmp(mfn, "spin_trylock") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "spinlock") == 0) {
                    result = type_struct("SpinLock"); /* EzSpinLock; opaque */
                } else if (strcmp(mfn, "store") == 0 || strcmp(mfn, "fence") == 0 ||
                           strcmp(mfn, "spin_lock") == 0 ||
                           strcmp(mfn, "spin_unlock") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "channels") == 0) {
                if (strcmp(mfn, "open") == 0) {
                    result = type_struct("Channel"); /* EzChannel; opaque */
                } else if (strcmp(mfn, "receive") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "send") == 0 || strcmp(mfn, "close") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "server") == 0) {
                if (strcmp(mfn, "add_router") == 0) {
                    result = type_struct("Router"); /* EzRouter; opaque */
                } else if (strcmp(mfn, "text") == 0 || strcmp(mfn, "json") == 0 ||
                           strcmp(mfn, "html") == 0 || strcmp(mfn, "redirect") == 0) {
                    result = type_struct("HttpResponse"); /* EzResponse */
                } else if (strcmp(mfn, "add_route") == 0 || strcmp(mfn, "listen") == 0 ||
                           strcmp(mfn, "cors") == 0 || strcmp(mfn, "use") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "http") == 0) {
                /* All http methods return EzHttpResponse struct
                 * with .status (int), .body (string), .headers (map) */
                result = type_struct("HttpResponse"); /* opaque struct; member access via __auto_type */
            } else if (strcmp(mod, "net") == 0) {
                if (strcmp(mfn, "listen") == 0) {
                    result = type_struct("Listener"); /* EzSocket; opaque */
                } else if (strcmp(mfn, "connect") == 0 || strcmp(mfn, "accept") == 0) {
                    result = type_struct("Socket"); /* EzSocket; opaque */
                } else if (strcmp(mfn, "send") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "receive") == 0 || strcmp(mfn, "resolve") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "close") == 0 || strcmp(mfn, "set_timeout") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E5026: functions that take a socket/listener as first arg */
                if (strcmp(mfn, "send") == 0 || strcmp(mfn, "receive") == 0 ||
                    strcmp(mfn, "close") == 0 || strcmp(mfn, "set_timeout") == 0 ||
                    strcmp(mfn, "accept") == 0) {
                    if (node->data.call.arg_count >= 1) {
                        EzType *arg1_type = resolve_expr(tc, node->data.call.args[0]);
                        if (arg1_type && arg1_type->kind != TK_STRUCT) {
                            const char *expected = strcmp(mfn, "accept") == 0
                                ? "Listener" : "Socket";
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "net.%s() expects a %s as the first argument, got %s",
                                mfn, expected,
                                arg1_type->name ? arg1_type->name : "non-struct type");
                            diag_error_msg(tc->diag, "E5026", strdup(msg),
                                NODE_FILE(tc, node), node->data.call.args[0]->token.line,
                                node->data.call.args[0]->token.column, 0);
                        }
                    }
                }
            } else if (strcmp(mod, "regex") == 0) {
                if (strcmp(mfn, "is_match") == 0 || strcmp(mfn, "is_valid") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "find") == 0 || strcmp(mfn, "replace") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "find_all") == 0 || strcmp(mfn, "split") == 0) {
                    result = type_array("string");
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "mem") == 0) {
                if (strcmp(mfn, "make") == 0 && node->data.call.arg_count == 2) {
                    AstNode *type_arg = node->data.call.args[1];
                    if (type_arg->kind == NODE_LABEL) {
                        result = type_pointer(type_arg->data.label.value);
                    } else {
                        result = &TYPE_UNKNOWN;
                    }
                } else if (strcmp(mfn, "alloc") == 0 && node->data.call.arg_count == 2) {
                    EzType *val_t = resolve_expr(tc, node->data.call.args[1]);
                    result = type_pointer(type_name(val_t));
                } else if (strcmp(mfn, "arena") == 0) {
                    result = type_struct("Arena");
                } else if (strcmp(mfn, "usage") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "free") == 0 || strcmp(mfn, "reset") == 0 ||
                           strcmp(mfn, "destroy") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "sqlite") == 0) {
                if (strcmp(mfn, "open") == 0) {
                    result = type_struct("Database");
                } else if (strcmp(mfn, "exec") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "query") == 0) {
                    result = type_array("map");
                } else if (strcmp(mfn, "close") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "fmt") == 0) {
                if (strcmp(mfn, "sprintf") == 0 ||
                    strcmp(mfn, "format") == 0 ||
                    strcmp(mfn, "pad_left") == 0 ||
                    strcmp(mfn, "pad_right") == 0 ||
                    strcmp(mfn, "center") == 0 ||
                    strcmp(mfn, "int_to_hex") == 0 ||
                    strcmp(mfn, "int_to_binary") == 0 ||
                    strcmp(mfn, "int_to_octal") == 0 ||
                    strcmp(mfn, "float_fixed") == 0 ||
                    strcmp(mfn, "float_sci") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "printf") == 0 || strcmp(mfn, "println") == 0 ||
                           strcmp(mfn, "eprintln") == 0 || strcmp(mfn, "eprint") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* Validate that non-format args are primitive types */
                for (int ai = 1; ai < node->data.call.arg_count; ai++) {
                    EzType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                    if (arg_t && (arg_t->kind == TK_STRUCT || arg_t->kind == TK_ARRAY ||
                                  arg_t->kind == TK_MAP || arg_t->kind == TK_POINTER)) {
                        /* Build a readable type name */
                        char tn[EZ_TYPE_NAME_MAX];
                        if (arg_t->kind == TK_ARRAY && arg_t->element_type)
                            snprintf(tn, sizeof(tn), "[%s]", arg_t->element_type);
                        else if (arg_t->kind == TK_MAP)
                            snprintf(tn, sizeof(tn), "map[%s:%s]",
                                arg_t->key_type ? arg_t->key_type : "?",
                                arg_t->value_type ? arg_t->value_type : "?");
                        else if (arg_t->kind == TK_POINTER && arg_t->element_type)
                            snprintf(tn, sizeof(tn), "^%s", arg_t->element_type);
                        else
                            snprintf(tn, sizeof(tn), "%s", type_name(arg_t));
                        diag_error_codef(tc->diag, "E3017", NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0, mfn, tn);
                    }
                }
            } else
            user_module_dispatch:
            if (is_struct_name(tc, mod)) {
                /* Struct-namespaced function call: Type.func(); look up return type */
                char prefixed[EZ_MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                FuncSig *sig = find_func(tc, prefixed);
                if (sig) {
                    sig->used = true;
                    result = sig->return_count > 0 ? sig->return_types[0] : &TYPE_VOID;
                    /* E5008: check argument count */
                    if (node->data.call.arg_count != sig->param_count) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "function '%s.%s' expects %d argument(s), got %d",
                            mod, mfn, sig->param_count, node->data.call.arg_count);
                        diag_error_msg(tc->diag, "E5008", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* Check argument types */
                    int check_count = node->data.call.arg_count < sig->param_count
                        ? node->data.call.arg_count : sig->param_count;
                    for (int ai = 0; ai < check_count; ai++) {
                        EzType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                        EzType *param_t = sig->param_types[ai];
                        if (arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                            arg_t->kind != param_t->kind &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                            !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                            !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                            !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s.%s': expected %s, got %s",
                                ai + 1, mod, mfn, type_name(param_t), type_name(arg_t));
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Enum-to-enum: kinds both TK_ENUM but different names */
                        if (arg_t->kind == TK_ENUM && param_t->kind == TK_ENUM &&
                            arg_t->name && param_t->name &&
                            strcmp(arg_t->name, param_t->name) != 0) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s.%s': expected enum '%s', got enum '%s'",
                                ai + 1, mod, mfn, param_t->name, arg_t->name);
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* E3027: non-lvalue or const passed to mutable (&) param.
                         * Struct functions live inside NODE_STRUCT_DECL, not as
                         * top-level stmts, so scan struct declarations. */
                        {
                            AstNode *arg = node->data.call.args[ai];
                            AstNode *found_decl = NULL;
                            for (int fi = 0; fi < tc->program->data.program.stmt_count && !found_decl; fi++) {
                                AstNode *s = tc->program->data.program.stmts[fi];
                                if (s->kind == NODE_STRUCT_DECL &&
                                    strcmp(s->data.struct_decl.name, mod) == 0) {
                                    for (int sfi = 0; sfi < s->data.struct_decl.func_count; sfi++) {
                                        AstNode *sf = s->data.struct_decl.funcs[sfi].func_decl;
                                        if (sf && sf->kind == NODE_FUNC_DECL &&
                                            strcmp(sf->data.func_decl.name, mfn) == 0 &&
                                            ai < sf->data.func_decl.param_count &&
                                            sf->data.func_decl.params[ai].mutable) {
                                            found_decl = sf;
                                            break;
                                        }
                                    }
                                }
                            }
                            if (found_decl) {
                                if (arg->kind == NODE_LABEL) {
                                    Symbol *arg_sym = scope_lookup(tc->current_scope,
                                        arg->data.label.value);
                                    if (arg_sym && !arg_sym->mutable) {
                                        char emsg[256];
                                        snprintf(emsg, sizeof(emsg),
                                            "cannot pass constant '%s' to mutable parameter '%s' of '%s.%s'",
                                            arg_sym->name, found_decl->data.func_decl.params[ai].name, mod, mfn);
                                        diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                            NODE_FILE(tc, arg), arg->token.line,
                                            arg->token.column, 0);
                                    }
                                } else if (arg->kind != NODE_MEMBER_EXPR &&
                                           arg->kind != NODE_INDEX_EXPR &&
                                           arg->kind != NODE_PREFIX_EXPR) {
                                    char emsg[256];
                                    snprintf(emsg, sizeof(emsg),
                                        "cannot pass a literal or expression to mutable parameter '%s' of '%s.%s'; expected a mutable variable",
                                        found_decl->data.func_decl.params[ai].name, mod, mfn);
                                    diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                        NODE_FILE(tc, arg), arg->token.line,
                                        arg->token.column, 0);
                                }
                            }
                        }
                    }
                } else {
                    result = &TYPE_VOID;
                }
            } else {
                /* Try user-defined module: look up <module>_<func> in function registry */
                char prefixed[EZ_MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                FuncSig *sig = find_func(tc, prefixed);
                if (sig && sig->is_private) {
                    diag_error_codef(tc->diag, "E4015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, mfn);
                } else if (sig) {
                    sig->used = true;
                    if (sig->return_count > 0) {
                        result = sig->return_types[0];
                    } else {
                        result = &TYPE_VOID;
                    }
                } else {
                    /* Check if 'mod' is a variable with a struct type —
                     * user wrote instance.func() instead of Type.func(). */
                    Symbol *sym = scope_lookup(tc->current_scope, mod_raw);
                    if (sym && sym->type && (sym->type->kind == TK_STRUCT ||
                        sym->type->kind == TK_POINTER)) {
                        const char *sname = sym->type->kind == TK_POINTER
                            ? sym->type->element_type : sym->type->name;
                        /* : check if `mfn` is a data field of type func
                         * before trying struct-function dispatch. A func-typed
                         * field should be called as a function pointer, not
                         * mistaken for a struct method. The bare "func" type
                         * was deprecated; modern typed function refs are
                         * encoded as "func(...)->R" with kind TK_FUNCTION. */
                        EzType *field_t = struct_field_type(tc, sname, mfn);
                        if (field_t && field_t->kind == TK_FUNCTION) {
                            /* This is a func-typed data field. Accept the call
                             *; codegen will emit it as a function-pointer
                             * call through the field access. Resolve the
                             * return type from the encoded signature so
                             * downstream uses get a typed value. */
                            sym->used = true;
                            result = (field_t->func_sig &&
                                      field_t->func_sig->return_count > 0 &&
                                      field_t->func_sig->return_types &&
                                      field_t->func_sig->return_types[0])
                                ? type_from_name(field_t->func_sig->return_types[0])
                                : &TYPE_UNKNOWN;
                            break;
                        }
                        char sfn[256];
                        snprintf(sfn, sizeof(sfn), "%s_%s", sname, mfn);
                        FuncSig *ssig = find_func(tc, sfn);
                        /* Auto-dispatch instance.method() → Type.method(instance)
                         * whenever the struct function takes the struct (or a
                         * pointer to it) as its first parameter. This covers
                         * both `do bar(self Foo)` and `do bar(&self Foo)`, and
                         * lets users call methods on instances without
                         * having to write the type name at every call site.
                         * Factory-style functions (e.g. `do make(x int) -> Foo`)
                         * whose first param isn't a Foo continue to require
                         * explicit `Foo.make(...)` since there's no instance
                         * to bind. */
                        bool is_self_method = false;
                        if (ssig && ssig->decl && ssig->decl->kind == NODE_FUNC_DECL &&
                            ssig->decl->data.func_decl.param_count > 0) {
                            const char *p0_tn = ssig->decl->data.func_decl.params[0].type_name;
                            if (p0_tn) {
                                if (strcmp(p0_tn, sname) == 0) {
                                    is_self_method = true;
                                } else if (p0_tn[0] == '^' && strcmp(p0_tn + 1, sname) == 0) {
                                    is_self_method = true;
                                }
                            }
                        }
                        if (is_self_method) {
                            /* Rewrite the call AST: change the member-expr
                             * object from the instance label to the type
                             * name, and prepend the instance as arg[0]. */
                            fn->data.member.object->data.label.value = strdup(sname);
                            int orig_count = node->data.call.arg_count;
                            AstNode **new_args = malloc(sizeof(AstNode *) * (orig_count + 1));
                            AstNode *self_arg = calloc(1, sizeof(AstNode));
                            self_arg->kind = NODE_LABEL;
                            self_arg->token = node->token;
                            self_arg->data.label.value = strdup(mod_raw);
                            new_args[0] = self_arg;
                            for (int ai = 0; ai < orig_count; ai++) {
                                new_args[ai + 1] = node->data.call.args[ai];
                            }
                            node->data.call.args = new_args;
                            node->data.call.arg_count = orig_count + 1;
                            /* Mark the function used and resolve return type */
                            ssig->used = true;
                            sym->used = true;
                            if (ssig->return_count > 0) {
                                result = ssig->return_types[0];
                            } else {
                                result = &TYPE_VOID;
                            }
                            /* E5008: validate argument count (after AST rewrite
                             * which prepended self as arg[0]). Both arg_count
                             * and param_count include self, so they compare
                             * directly. Display counts subtract 1 to hide self. */
                            {
                                int min_params = ssig->param_count;
                                if (ssig->decl && ssig->decl->kind == NODE_FUNC_DECL) {
                                    min_params = 0;
                                    for (int pi = 0; pi < ssig->decl->data.func_decl.param_count; pi++) {
                                        if (!ssig->decl->data.func_decl.params[pi].default_value)
                                            min_params++;
                                    }
                                }
                                if (node->data.call.arg_count < min_params ||
                                    node->data.call.arg_count > ssig->param_count) {
                                    int display_got = node->data.call.arg_count - 1;
                                    int display_max = ssig->param_count - 1;
                                    int display_min = min_params - 1;
                                    char emsg[256];
                                    if (display_min == display_max) {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d argument(s), got %d",
                                            sname, mfn, display_max, display_got);
                                    } else {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d-%d argument(s), got %d",
                                            sname, mfn, display_min, display_max, display_got);
                                    }
                                    diag_error_msg(tc->diag, "E5008", strdup(emsg),
                                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                                }
                            }
                            /* Validate argument types (after AST rewrite) */
                            {
                                int check_count = node->data.call.arg_count < ssig->param_count
                                    ? node->data.call.arg_count : ssig->param_count;
                                for (int ai = 0; ai < check_count; ai++) {
                                    EzType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                                    EzType *param_t = ssig->param_types[ai];
                                    if (arg_t && param_t &&
                                        arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                                        arg_t->kind != param_t->kind &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                                        !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                                        !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                                        !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                                        char amsg[256];
                                        snprintf(amsg, sizeof(amsg),
                                            "argument %d of '%s.%s': expected %s, got %s",
                                            ai + 1, sname, mfn, type_name(param_t), type_name(arg_t));
                                        diag_error_msg(tc->diag, "E3001", strdup(amsg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                }
                            }
                        } else {
                            diag_error_codef(tc->diag, "E3042", NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0, sname, mfn, mod_raw, mfn);
                            if (ssig && ssig->return_count > 0) {
                                result = ssig->return_types[0];
                            } else {
                                result = &TYPE_VOID;
                            }
                        }
                    } else {
                        result = &TYPE_VOID;
                    }
                }
            }
            break;
        }

        /* Member call on expression result: foo().bar() */
        if (fn->kind == NODE_MEMBER_EXPR && fn->data.member.object->kind != NODE_LABEL) {
            EzType *obj_t = resolve_expr(tc, fn->data.member.object);
            if (obj_t && obj_t->kind != TK_STRUCT && obj_t->kind != TK_POINTER &&
                obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_VOID) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type '%s' has no functions; only structs support function calls with dot syntax",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", strdup(msg),
                    NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0);
            } else if (obj_t && (obj_t->kind == TK_STRUCT || obj_t->kind == TK_POINTER)) {
                /* E3075: chaining struct function calls (calling one struct
                 * function on the result of another) isn't supported.
                 * Assigning the intermediate result to a variable keeps
                 * each call site readable and avoids the AST-rewrite
                 * gymnastics that fluent-interface chaining would require. */
                diag_error_code(tc->diag, "E3075",
                    NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0);
            }
            result = &TYPE_UNKNOWN;
            break;
        }

        if (fn_name) {
            if (tc_is_builtin(fn_name)) {
            }
            /* Check built-in functions first */
            if (strcmp(fn_name, "addr") == 0 && node->data.call.arg_count == 1) {
                AstNode *arg = node->data.call.args[0];
                /* addr() requires an lvalue. is_lvalue_expr recurses into
                 * member/index chains so 'addr(some_call().field)' is
                 * rejected at typecheck instead of leaking an
                 * '&(rvalue)' to clang. */
                if (!is_lvalue_expr(arg)) {
                    diag_error_msg(tc->diag, "E3012",
                        strdup("addr() requires a variable, field, or index expression; cannot take address of a literal or expression"),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                EzType *arg_t = resolve_expr(tc, arg);
                result = type_pointer(type_name(arg_t));
            } else if (strcmp(fn_name, "ref") == 0 && node->data.call.arg_count == 1) {
                AstNode *arg = node->data.call.args[0];
                /* ref(func_name) returns func type; resolve the
                 * function lookup BEFORE calling resolve_expr on the
                 * label, otherwise the E3031 "bare function name as
                 * value" check fires and rejects a legitimate
                 * use ( follow-up). */
                if (arg->kind == NODE_LABEL &&
                    find_func(tc, arg->data.label.value)) {
                    FuncSig *rfs = find_func(tc, arg->data.label.value);
                    if (rfs) rfs->used = true;
                    result = type_from_name("func");
                } else {
                    /* Same lvalue requirement as addr(): reject literals,
                     * call results, arithmetic expressions — anything
                     * without a stable address. Without this, ref(42) and
                     * ref(some_call()) leaked '&42' / '&(rvalue)' to clang
                     * and produced opaque generated-C errors. */
                    if (!is_lvalue_expr(arg)) {
                        diag_error_msg(tc->diag, "E3012",
                            strdup("ref() requires a variable, field, or index expression; cannot take a reference to a literal, call result, or expression"),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* Build a pointer type that preserves the full source type.
                     * For arrays, type_name returns the element type ("int"),
                     * so reconstruct the full name ("[int]"). */
                    EzType *arg_t = resolve_expr(tc, arg);
                    const char *pointee_name = type_name(arg_t);
                    if (arg_t->kind == TK_ARRAY) {
                        char buf[256];
                        snprintf(buf, sizeof(buf), "[%s]", arg_t->element_type);
                        pointee_name = strdup(buf);
                    } else if (arg_t->kind == TK_MAP) {
                        pointee_name = strdup(arg_t->name);
                    }
                    result = type_pointer(pointee_name);
                }
            } else if (strcmp(fn_name, "len") == 0) {
                /* E7015 (): len() only works on string / array / map.
                 * Codegen blindly emits '.len' on the receiver, which
                 * works for the runtime's EzArray / EzMap / EzString
                 * structs but bombs with a raw clang "no member 'len'"
                 * on anything else; structs, enums, primitives,
                 * pointers, func refs, errors. Validate the argument
                 * type here instead of letting it leak. */
                if (node->data.call.arg_count != 1) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "len() expects 1 argument, got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    reject_void_in_context(tc, node->data.call.args[0], at,
                        "len() argument");
                    if (at && at->kind != TK_UNKNOWN && at->kind != TK_VOID &&
                        at->kind != TK_STRING && at->kind != TK_ARRAY &&
                        at->kind != TK_MAP) {
                        diag_error_codef(tc->diag, "E7015", NODE_FILE(tc, node->data.call.args[0]), node->data.call.args[0]->token.line,
                            node->data.call.args[0]->token.column, 0, type_name(at));
                    }
                }
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "type_of") == 0) {
                /* E3084: type_of() with a type name instead of a value */
                if (node->data.call.arg_count > 0) {
                    AstNode *arg = node->data.call.args[0];
                    if (arg->kind == NODE_LABEL) {
                        const char *aname = arg->data.label.value;
                        Symbol *sym = scope_lookup(tc->current_scope, aname);
                        if (!sym && (is_struct_name(tc, aname) || is_enum_name(tc, aname))) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "type_of() expects a value, not a type name '%s'; use type_of(instance) instead",
                                aname);
                            diag_error_msg(tc->diag, "E3084", strdup(msg),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                }
                /* E3038: type_of() on void function result */
                if (node->data.call.arg_count > 0) {
                    EzType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                    if (arg_t->kind == TK_VOID) {
                        diag_error_msg(tc->diag, "E3038",
                            strdup("cannot use type_of() on a void function; the function does not return a value"),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "size_of") == 0) {
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "to_char") == 0) {
                if (node->data.call.arg_count != 2) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "to_char() expects 2 arguments (string, index), got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    EzType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    EzType *arg1 = resolve_expr(tc, node->data.call.args[1]);
                    if (arg0->kind != TK_STRING) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "to_char() first argument must be a string, got %s",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3043", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (arg1->kind != TK_INT && arg1->kind != TK_UINT) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "to_char() second argument must be int or uint, got %s",
                            type_name(arg1));
                        diag_error_msg(tc->diag, "E3043", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_CHAR;
            } else if (strcmp(fn_name, "char_count") == 0) {
                if (node->data.call.arg_count != 1) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "char_count() expects 1 argument (string), got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    EzType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    if (arg0->kind != TK_STRING) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "char_count() argument must be a string, got %s",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3043", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "c_string") == 0) {
                if (node->data.call.arg_count >= 1) {
                    EzType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    if (arg0 && arg0->kind != TK_POINTER && arg0->kind != TK_UNKNOWN) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "c_string() requires a raw C pointer; '%s' is not a pointer type. "
                            "c_string() is only valid with values from C interop (import c\"header.h\")",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3083", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "input") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "error") == 0) {
                result = type_from_name("Error");
            } else if (strcmp(fn_name, "println") == 0 || strcmp(fn_name, "eprintln") == 0) {
                /* println/eprintln accept 0 or 1 arguments */
                if (node->data.call.arg_count > 1) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "%s() expects 0 or 1 argument(s), got %d",
                        fn_name, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (node->data.call.arg_count >= 1) {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    char ctx[EZ_TYPE_NAME_MAX];
                    snprintf(ctx, sizeof(ctx), "%s() argument", fn_name);
                    reject_void_in_context(tc, node->data.call.args[0], at, ctx);
                }
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "print") == 0 || strcmp(fn_name, "eprint") == 0) {
                /* print/eprint accept exactly 1 argument */
                if (node->data.call.arg_count != 1) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "%s() expects 1 argument, got %d",
                        fn_name, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (node->data.call.arg_count >= 1) {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    char ctx[EZ_TYPE_NAME_MAX];
                    snprintf(ctx, sizeof(ctx), "%s() argument", fn_name);
                    reject_void_in_context(tc, node->data.call.args[0], at, ctx);
                }
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "exit") == 0 || strcmp(fn_name, "panic") == 0 ||
                       strcmp(fn_name, "assert") == 0 ||
                       strcmp(fn_name, "sleep_s") == 0 ||
                       strcmp(fn_name, "sleep_ms") == 0 ||
                       strcmp(fn_name, "sleep_ns") == 0) {
                /* Validate argument types for these builtins */
                if (strcmp(fn_name, "exit") == 0 && node->data.call.arg_count >= 1) {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && !is_int_kind(at->kind)) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "exit() expects an integer argument, got '%s'", type_name(at));
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                } else if (strcmp(fn_name, "panic") == 0 && node->data.call.arg_count >= 1) {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && at->kind != TK_STRING) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "panic() expects a string argument, got '%s'", type_name(at));
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                } else if (strcmp(fn_name, "assert") == 0 && node->data.call.arg_count >= 1) {
                    EzType *cond_t = resolve_expr(tc, node->data.call.args[0]);
                    if (cond_t->kind != TK_UNKNOWN && cond_t->kind != TK_BOOL) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "assert() condition must be a bool, got '%s'", type_name(cond_t));
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (node->data.call.arg_count >= 2) {
                        EzType *msg_t = resolve_expr(tc, node->data.call.args[1]);
                        if (msg_t->kind != TK_UNKNOWN && msg_t->kind != TK_STRING) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "assert() message must be a string, got '%s'", type_name(msg_t));
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                } else if ((strcmp(fn_name, "sleep_s") == 0 || strcmp(fn_name, "sleep_ms") == 0 ||
                            strcmp(fn_name, "sleep_ns") == 0) && node->data.call.arg_count >= 1) {
                    EzType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && !is_int_kind(at->kind)) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "%s() expects an integer argument, got '%s'", fn_name, type_name(at));
                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "copy") == 0 && node->data.call.arg_count == 1) {
                result = resolve_expr(tc, node->data.call.args[0]);
            } else if (strcmp(fn_name, "char") == 0 && node->data.call.arg_count == 1) {
                /* E7014: char() with negative integer */
                int64_t lit_val;
                if (try_get_literal_int(node->data.call.args[0], &lit_val) && lit_val < 0) {
                    diag_error_codef(tc->diag, "E7014", NODE_FILE(tc, node), node->token.line, node->token.column, 0, (long long)lit_val);
                }
                result = &TYPE_CHAR;
            } else if ((strcmp(fn_name, "int") == 0 ||
                        strcmp(fn_name, "i128") == 0 ||
                        strcmp(fn_name, "i256") == 0 ||
                        strcmp(fn_name, "u128") == 0 ||
                        strcmp(fn_name, "u256") == 0 || strcmp(fn_name, "uint") == 0 ||
                        strcmp(fn_name, "byte") == 0) &&
                       node->data.call.arg_count == 1) {
                /* E3043: validate source type is convertible to numeric */
                EzType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                    src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "cannot convert %s to %s; only numeric types, strings, and bools can be converted",
                        type_name(src_t), fn_name);
                    diag_error_msg(tc->diag, "E3043", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (is_unsigned_type(fn_name))
                    result = &TYPE_UINT;
                else
                    result = &TYPE_INT;
            } else if (strcmp(fn_name, "string") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "float") == 0 && node->data.call.arg_count == 1) {
                /* E3043: validate source type is convertible to float */
                EzType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                    src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER ||
                    src_t->kind == TK_BOOL) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "cannot convert %s to float; only numeric types and strings can be converted",
                        type_name(src_t));
                    diag_error_msg(tc->diag, "E3043", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                result = &TYPE_FLOAT;
            } else if (strcmp(fn_name, "bool") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_BOOL;
            } else {
                FuncSig *sig = find_func(tc, fn_name);
                if (sig) {
                    sig->used = true;
                    /* : if this bare name is a using-module alias,
                     * also mark the prefixed sig + import as used so
                     * W1002/W1003 don't fire on the source. */
                    for (int ui = 0; ui < tc->using_module_count; ui++) {
                        const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                        char pfx[256];
                        snprintf(pfx, sizeof(pfx), "%s_%s", real_mod, fn_name);
                        FuncSig *psig = find_func(tc, pfx);
                        if (psig) {
                            psig->used = true;
                            for (int mi = 0; mi < tc->import_count; mi++) {
                                if (strcmp(tc->imported_modules[mi], tc->using_modules[ui]) == 0 ||
                                    strcmp(tc->imported_modules[mi], real_mod) == 0) {
                                    tc->import_used[mi] = true;
                                    break;
                                }
                            }
                            break;
                        }
                    }
                    /* Check argument count; account for default parameters */
                    int min_args = sig->param_count;
                    /* Find the AST func decl to count defaults */
                    for (int fi = 0; fi < tc->program->data.program.stmt_count; fi++) {
                        AstNode *s = tc->program->data.program.stmts[fi];
                        if (s->kind == NODE_FUNC_DECL &&
                            strcmp(s->data.func_decl.name, fn_name) == 0) {
                            min_args = 0;
                            for (int pi = 0; pi < s->data.func_decl.param_count; pi++) {
                                if (!s->data.func_decl.params[pi].default_value) {
                                    min_args++;
                                }
                            }
                            break;
                        }
                    }
                    if (node->data.call.arg_count < min_args ||
                        node->data.call.arg_count > sig->param_count) {
                        char msg[EZ_MSG_BUF_SIZE];
                        if (min_args == sig->param_count) {
                            snprintf(msg, sizeof(msg),
                                "function '%s' expects %d argument(s), got %d",
                                fn_name, sig->param_count, node->data.call.arg_count);
                        } else {
                            snprintf(msg, sizeof(msg),
                                "function '%s' expects %d-%d argument(s), got %d",
                                fn_name, min_args, sig->param_count, node->data.call.arg_count);
                        }
                        diag_error_msg(tc->diag, "E5008", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* Generic (wildcard) dispatch: unify each '?' parameter
                     * against the corresponding argument to derive a single
                     * concrete binding T, record the instantiation, and
                     * substitute T into the return type. Skip the normal
                     * per-arg check below since '?' would otherwise collapse
                     * to TK_UNKNOWN and produce no useful errors. */
                    char *generic_binding = NULL;
                    EzType *generic_return_t = NULL;
                    bool is_generic_call = sig->is_generic && sig->decl &&
                        sig->decl->kind == NODE_FUNC_DECL;
                    if (is_generic_call) {
                        int cc = node->data.call.arg_count < sig->decl->data.func_decl.param_count
                            ? node->data.call.arg_count : sig->decl->data.func_decl.param_count;
                        for (int ai = 0; ai < cc; ai++) {
                            const char *ptn = sig->decl->data.func_decl.params[ai].type_name;
                            EzType *at = resolve_expr(tc, node->data.call.args[ai]);
                            if (!type_name_has_wildcard(ptn)) continue;
                            char *bound = bind_wildcard(ptn, at);
                            if (!bound) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "cannot infer wildcard type '%s' from argument %d of '%s' (got %s)",
                                    ptn, ai + 1, fn_name, type_name(at));
                                diag_error_msg(tc->diag, "E3001", strdup(msg),
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                                continue;
                            }
                            if (!generic_binding) {
                                generic_binding = bound;
                            } else if (strcmp(generic_binding, bound) != 0) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "wildcard type conflict in '%s': '?' was bound to %s, but argument %d is %s",
                                    fn_name, generic_binding, ai + 1, bound);
                                diag_error_msg(tc->diag, "E3001", strdup(msg),
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                                free(bound);
                            } else {
                                free(bound);
                            }
                        }
                        if (generic_binding) {
                            record_instantiation(sig, generic_binding, node);
                            if (sig->decl->data.func_decl.return_type_count > 0) {
                                char *rt_str = substitute_wildcard(
                                    sig->decl->data.func_decl.return_types[0],
                                    generic_binding);
                                generic_return_t = type_from_name(rt_str);
                                /* rt_str is owned by type_from_name on the
                                 * heap path; leak is fine at compile time. */
                            } else {
                                generic_return_t = &TYPE_VOID;
                            }
                        }
                    }

                    /* Check argument types */
                    int check_count = node->data.call.arg_count < sig->param_count
                        ? node->data.call.arg_count : sig->param_count;
                    for (int ai = 0; ai < check_count; ai++) {
                        EzType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                        EzType *param_t = sig->param_types[ai];
                        if (is_generic_call) {
                            /* Generic branch already handled unification;
                             * suppress the scalar param/arg comparison which
                             * would compare against TK_UNKNOWN. */
                            continue;
                        }
                        if (arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                            arg_t->kind != param_t->kind &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                            !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                            !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                            !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s': expected %s, got %s",
                                ai + 1, fn_name, type_name(param_t), type_name(arg_t));
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Enum-to-enum: kinds both TK_ENUM but different names */
                        if (arg_t->kind == TK_ENUM && param_t->kind == TK_ENUM &&
                            arg_t->name && param_t->name &&
                            strcmp(arg_t->name, param_t->name) != 0) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s': expected enum '%s', got enum '%s'",
                                ai + 1, fn_name, param_t->name, arg_t->name);
                            diag_error_msg(tc->diag, "E3001", strdup(msg),
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* E3066: typed-func signatures must match exactly */
                        if (arg_t->kind == TK_FUNCTION && param_t->kind == TK_FUNCTION &&
                            arg_t->name && param_t->name &&
                            strcmp(arg_t->name, param_t->name) != 0) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s': expected %s, got %s",
                                ai + 1, fn_name, param_t->name, arg_t->name);
                            diag_error_msg(tc->diag, "E3066", strdup(msg),
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* E3027: non-lvalue or const passed to mutable (&) param */
                        {
                            AstNode *arg = node->data.call.args[ai];
                            /* Find the func decl to check if this param is mutable */
                            for (int fi = 0; fi < tc->program->data.program.stmt_count; fi++) {
                                AstNode *s = tc->program->data.program.stmts[fi];
                                if (s->kind != NODE_FUNC_DECL ||
                                    strcmp(s->data.func_decl.name, fn_name) != 0 ||
                                    ai >= s->data.func_decl.param_count ||
                                    !s->data.func_decl.params[ai].mutable)
                                    continue;
                                /* Param is mutable; check argument is a valid lvalue */
                                if (arg->kind == NODE_LABEL) {
                                    Symbol *arg_sym = scope_lookup(tc->current_scope,
                                        arg->data.label.value);
                                    if (arg_sym && !arg_sym->mutable) {
                                        char msg[EZ_MSG_BUF_SIZE];
                                        snprintf(msg, sizeof(msg),
                                            "cannot pass constant '%s' to mutable parameter '%s' of '%s'",
                                            arg_sym->name, s->data.func_decl.params[ai].name, fn_name);
                                        diag_error_msg(tc->diag, "E3027", strdup(msg),
                                            NODE_FILE(tc, arg), arg->token.line,
                                            arg->token.column, 0);
                                    }
                                } else if (arg->kind != NODE_MEMBER_EXPR &&
                                           arg->kind != NODE_INDEX_EXPR &&
                                           arg->kind != NODE_PREFIX_EXPR) {
                                    /* Literal, call result, or other non-lvalue expression */
                                    char msg[EZ_MSG_BUF_SIZE];
                                    snprintf(msg, sizeof(msg),
                                        "cannot pass a literal or expression to mutable parameter '%s' of '%s'; expected a mutable variable",
                                        s->data.func_decl.params[ai].name, fn_name);
                                    diag_error_msg(tc->diag, "E3027", strdup(msg),
                                        NODE_FILE(tc, arg), arg->token.line,
                                        arg->token.column, 0);
                                }
                                break;
                            }
                        }
                    }
                    if (is_generic_call && generic_return_t) {
                        result = generic_return_t;
                    } else if (sig->return_count > 0) {
                        result = sig->return_types[0];
                    } else {
                        result = &TYPE_VOID;
                    }
                } else {
                    /* Check if it's a variable holding a function reference */
                    Symbol *fn_sym = scope_lookup(tc->current_scope, fn_name);
                    bool is_typed_func = fn_sym && fn_sym->type && fn_sym->type->kind == TK_FUNCTION;
                    bool is_bare_func = fn_sym && fn_sym->type && type_name(fn_sym->type) &&
                                        strcmp(type_name(fn_sym->type), "func") == 0;
                    if (is_typed_func || is_bare_func) {
                        fn_sym->used = true;
                        /* Record the variable's type on the call.function
                         * label node so codegen can pick up the typed-func
                         * signature for cast emission. */
                        if (!tc->suppress_typetable_writes) {
                            typetable_set(tc->type_table, node->data.call.function, fn_sym->type);
                        }
                        result = &TYPE_UNKNOWN; /* callable func ref; return type unknown */
                        /* If we know which static function this var holds,
                         * validate arity + argument types at compile time
                         * and propagate the real return type. */
                        FuncSig *ref_sig = fn_sym->func_ref_name
                            ? find_func(tc, fn_sym->func_ref_name) : NULL;
                        if (ref_sig) {
                            /* : compute min arity by counting
                             * params without default values. */
                            int min_arity = ref_sig->param_count;
                            if (ref_sig->decl && ref_sig->decl->kind == NODE_FUNC_DECL) {
                                min_arity = 0;
                                for (int pi = 0; pi < ref_sig->decl->data.func_decl.param_count; pi++) {
                                    if (!ref_sig->decl->data.func_decl.params[pi].default_value)
                                        min_arity++;
                                }
                            }
                            int ac = node->data.call.arg_count;
                            if (ac < min_arity || ac > ref_sig->param_count) {
                                char msg[EZ_MSG_BUF_SIZE];
                                if (min_arity == ref_sig->param_count) {
                                    snprintf(msg, sizeof(msg),
                                        "function '%s' expects %d argument(s), got %d",
                                        ref_sig->name, ref_sig->param_count, ac);
                                } else {
                                    snprintf(msg, sizeof(msg),
                                        "function '%s' expects %d to %d argument(s), got %d",
                                        ref_sig->name, min_arity, ref_sig->param_count, ac);
                                }
                                diag_error_msg(tc->diag, "E5008", strdup(msg),
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            } else {
                                for (int ai = 0; ai < ref_sig->param_count; ai++) {
                                    EzType *at = resolve_expr(tc, node->data.call.args[ai]);
                                    EzType *pt = ref_sig->param_types[ai];
                                    if (at && pt && at->kind != TK_UNKNOWN &&
                                        pt->kind != TK_UNKNOWN &&
                                        at->kind != pt->kind &&
                                        !(is_int_kind(at->kind) && is_int_kind(pt->kind)) &&
                                        !(pt->kind == TK_FLOAT && is_int_kind(at->kind))) {
                                        char msg[EZ_MSG_BUF_SIZE];
                                        snprintf(msg, sizeof(msg),
                                            "argument %d of '%s': expected %s, got %s",
                                            ai + 1, ref_sig->name,
                                            type_name(pt), type_name(at));
                                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                    /* Enum-to-enum: different enum types */
                                    if (at && pt && at->kind == TK_ENUM && pt->kind == TK_ENUM &&
                                        at->name && pt->name &&
                                        strcmp(at->name, pt->name) != 0) {
                                        char msg[EZ_MSG_BUF_SIZE];
                                        snprintf(msg, sizeof(msg),
                                            "argument %d of '%s': expected enum '%s', got enum '%s'",
                                            ai + 1, ref_sig->name, pt->name, at->name);
                                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                    /* E3027: non-lvalue or const passed to mutable (&) param */
                                    {
                                        AstNode *arg = node->data.call.args[ai];
                                        const char *ref_name = fn_sym->func_ref_name;
                                        for (int fi = 0; fi < tc->program->data.program.stmt_count; fi++) {
                                            AstNode *s = tc->program->data.program.stmts[fi];
                                            if (s->kind != NODE_FUNC_DECL ||
                                                strcmp(s->data.func_decl.name, ref_name) != 0 ||
                                                ai >= s->data.func_decl.param_count ||
                                                !s->data.func_decl.params[ai].mutable)
                                                continue;
                                            if (arg->kind == NODE_LABEL) {
                                                Symbol *arg_sym = scope_lookup(tc->current_scope,
                                                    arg->data.label.value);
                                                if (arg_sym && !arg_sym->mutable) {
                                                    char emsg[256];
                                                    snprintf(emsg, sizeof(emsg),
                                                        "cannot pass constant '%s' to mutable parameter '%s' of '%s'",
                                                        arg_sym->name, s->data.func_decl.params[ai].name,
                                                        ref_sig->name);
                                                    diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                                        NODE_FILE(tc, arg), arg->token.line,
                                                        arg->token.column, 0);
                                                }
                                            } else if (arg->kind != NODE_MEMBER_EXPR &&
                                                       arg->kind != NODE_INDEX_EXPR &&
                                                       arg->kind != NODE_PREFIX_EXPR) {
                                                char emsg[256];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass a literal or expression to mutable parameter '%s' of '%s'; expected a mutable variable",
                                                    s->data.func_decl.params[ai].name, ref_sig->name);
                                                diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                                    NODE_FILE(tc, arg), arg->token.line,
                                                    arg->token.column, 0);
                                            }
                                            break;
                                        }
                                    }
                                }
                            }
                            if (ref_sig->return_count > 0)
                                result = ref_sig->return_types[0];
                            else
                                result = &TYPE_VOID;
                        } else if (is_typed_func && fn_sym->type->func_sig) {
                            /* No source FuncSig; validate against the typed-
                             * func signature from the variable's annotated type
                             * (e.g. callback parameters: do f(g func(int)->int)). */
                            EzFuncSig *sig = fn_sym->type->func_sig;
                            int ac = node->data.call.arg_count;
                            if (ac != sig->param_count) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "function reference '%s' expects %d argument(s), got %d",
                                    fn_name, sig->param_count, ac);
                                diag_error_msg(tc->diag, "E5008", strdup(msg),
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            } else {
                                for (int ai = 0; ai < sig->param_count; ai++) {
                                    AstNode *arg = node->data.call.args[ai];
                                    EzType *at = resolve_expr(tc, arg);
                                    EzType *pt = sig->param_types[ai] ? type_from_name(sig->param_types[ai]) : NULL;
                                    if (at && pt && at->kind != TK_UNKNOWN && pt->kind != TK_UNKNOWN &&
                                        at->kind != pt->kind &&
                                        !(is_int_kind(at->kind) && is_int_kind(pt->kind)) &&
                                        !(pt->kind == TK_FLOAT && is_int_kind(at->kind))) {
                                        char msg[EZ_MSG_BUF_SIZE];
                                        snprintf(msg, sizeof(msg),
                                            "argument %d of '%s': expected %s, got %s",
                                            ai + 1, fn_name,
                                            type_name(pt), type_name(at));
                                        diag_error_msg(tc->diag, "E3001", strdup(msg),
                                            NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0);
                                    }
                                    /* E3027/E3067: `&` param requires an lvalue */
                                    if (sig->param_mutable[ai]) {
                                        bool is_lvalue = (arg->kind == NODE_LABEL ||
                                                          arg->kind == NODE_MEMBER_EXPR ||
                                                          arg->kind == NODE_INDEX_EXPR);
                                        if (!is_lvalue) {
                                            diag_error_codef(tc->diag, "E3067", NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0, ai + 1, fn_name);
                                        } else if (arg->kind == NODE_LABEL) {
                                            Symbol *as = scope_lookup(tc->current_scope, arg->data.label.value);
                                            if (as && !as->mutable) {
                                                char emsg[256];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass constant '%s' to '&' parameter %d of '%s'",
                                                    as->name, ai + 1, fn_name);
                                                diag_error_msg(tc->diag, "E3027", strdup(emsg),
                                                    NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0);
                                            }
                                        }
                                    }
                                }
                            }
                            if (sig->return_count > 0 && sig->return_types[0]) {
                                result = type_from_name(sig->return_types[0]);
                            } else {
                                result = &TYPE_VOID;
                            }
                        }
                    } else if (fn_sym && fn_sym->type) {
                        /* Variable exists but is not a function */
                        diag_error_codef(tc->diag, "E3015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, fn_name, type_name(fn_sym->type));
                    } else if (!tc_is_builtin(fn_name)) {
                        /* Check if it's a function from a 'using' module */
                        bool found_in_using = false;
                        /* Map function names to module + return type */
                        static const struct { const char *func; const char *mod; TypeKind ret; } using_funcs[] = {
                            /* @strings */
                            {"to_upper","strings",TK_STRING},{"to_lower","strings",TK_STRING},
                            {"trim","strings",TK_STRING},{"trim_left","strings",TK_STRING},
                            {"trim_right","strings",TK_STRING},{"replace","strings",TK_STRING},
                            {"repeat","strings",TK_STRING},{"reverse","strings",TK_STRING},
                            {"slice","strings",TK_STRING},{"join","strings",TK_STRING},
                            {"contains","strings",TK_BOOL},{"starts_with","strings",TK_BOOL},
                            {"ends_with","strings",TK_BOOL},{"is_empty","strings",TK_BOOL},
                            {"index_of","strings",TK_INT},{"count","strings",TK_INT},
                            {"split","strings",TK_ARRAY},
                            /* @math (arg-dependent abs/neg/min/max/clamp handled by special case below) */
                            {"sign","math",TK_INT},{"factorial","math",TK_INT},{"gcd","math",TK_INT},
                            {"lcm","math",TK_INT},
                            {"floor","math",TK_FLOAT},{"ceil","math",TK_FLOAT},{"round","math",TK_FLOAT},
                            {"trunc","math",TK_FLOAT},
                            {"pow","math",TK_FLOAT},{"sqrt","math",TK_FLOAT},{"cbrt","math",TK_FLOAT},
                            {"hypot","math",TK_FLOAT},{"exp","math",TK_FLOAT},{"exp2","math",TK_FLOAT},
                            {"log","math",TK_FLOAT},{"log2","math",TK_FLOAT},{"log10","math",TK_FLOAT},
                            {"log_base","math",TK_FLOAT},{"sin","math",TK_FLOAT},{"cos","math",TK_FLOAT},
                            {"tan","math",TK_FLOAT},{"asin","math",TK_FLOAT},{"acos","math",TK_FLOAT},
                            {"atan","math",TK_FLOAT},{"atan2","math",TK_FLOAT},
                            {"sinh","math",TK_FLOAT},{"cosh","math",TK_FLOAT},{"tanh","math",TK_FLOAT},
                            {"deg_to_rad","math",TK_FLOAT},{"rad_to_deg","math",TK_FLOAT},
                            {"lerp","math",TK_FLOAT},{"distance","math",TK_FLOAT},
                            {"is_prime","math",TK_BOOL},{"is_even","math",TK_BOOL},
                            {"is_odd","math",TK_BOOL},{"is_infinite","math",TK_BOOL},
                            {"is_nan","math",TK_BOOL},{"is_finite","math",TK_BOOL},
                            /* @arrays (arg-dependent get_first/get_last/etc handled by inline dispatch) */
                            {"append","arrays",TK_VOID},{"insert_at","arrays",TK_VOID},
                            {"prepend","arrays",TK_VOID},{"fill","arrays",TK_VOID},
                            {"remove_at","arrays",TK_VOID},{"sort_asc","arrays",TK_VOID},
                            {"sort_desc","arrays",TK_VOID},{"clear","arrays",TK_VOID},
                            {"concat","arrays",TK_ARRAY},{"deduplicate","arrays",TK_ARRAY},
                            {"flatten","arrays",TK_ARRAY},{"reverse","arrays",TK_ARRAY},
                            {"slice","arrays",TK_ARRAY},{"split_every","arrays",TK_ARRAY},
                            {"pair","arrays",TK_ARRAY},
                            {"get_sum","arrays",TK_INT},{"get_min","arrays",TK_INT},
                            {"get_max","arrays",TK_INT},{"count","arrays",TK_INT},
                            {"index_of","arrays",TK_INT},
                            {"is_empty","arrays",TK_BOOL},{"contains","arrays",TK_BOOL},
                            {"is_equal","arrays",TK_BOOL},
                            /* @maps (arg-dependent get_keys/get_values handled by special case below) */
                            {"has_key","maps",TK_BOOL},{"is_empty","maps",TK_BOOL},
                            {"contains_value","maps",TK_BOOL},{"remove_key","maps",TK_VOID},
                            {"clear","maps",TK_VOID},{"is_equal","maps",TK_BOOL},
                            /* @random (arg-dependent choice/shuffle/sample handled by special case below) */
                            {"rand_float","random",TK_FLOAT},{"rand_int","random",TK_INT},
                            {"rand_bool","random",TK_BOOL},{"rand_byte","random",TK_INT},
                            {"rand_char","random",TK_INT},{"random_hex","random",TK_STRING},
                            {"seed","random",TK_VOID},
                            /* @encoding */
                            {"base64_encode","encoding",TK_STRING},{"base64_decode","encoding",TK_STRING},
                            {"hex_encode","encoding",TK_STRING},{"hex_decode","encoding",TK_STRING},
                            {"url_encode","encoding",TK_STRING},{"url_decode","encoding",TK_STRING},
                            /* @crypto */
                            {"sha256","crypto",TK_STRING},{"md5","crypto",TK_STRING},
                            {"random_hex","crypto",TK_STRING},
                            /* @regex */
                            {"is_match","regex",TK_BOOL},{"is_valid","regex",TK_BOOL},
                            {"find","regex",TK_STRING},{"replace","regex",TK_STRING},
                            {"find_all","regex",TK_ARRAY},{"split","regex",TK_ARRAY},
                            /* @json */
                            {"is_valid","json",TK_BOOL},{"decode","json",TK_MAP},
                            {"encode","json",TK_STRING},{"stringify","json",TK_STRING},
                            {"parse","json",TK_UNKNOWN},{"pretty","json",TK_STRING},
                            /* @io */
                            {"read_file","io",TK_STRING},{"write_file","io",TK_BOOL},
                            {"append_file","io",TK_BOOL},{"delete_file","io",TK_BOOL},
                            {"rename_file","io",TK_BOOL},{"file_exists","io",TK_BOOL},
                            {"is_file","io",TK_BOOL},{"is_directory","io",TK_BOOL},
                            {"file_size","io",TK_INT},{"glob","io",TK_ARRAY},
                            {"list_dir","io",TK_ARRAY},{"walk","io",TK_ARRAY},
                            {"make_dir","io",TK_BOOL},{"make_dir_all","io",TK_BOOL},
                            {"remove_dir","io",TK_BOOL},{"remove_dir_all","io",TK_BOOL},
                            {"copy_file","io",TK_BOOL},{"move_file","io",TK_BOOL},
                            {"is_absolute","io",TK_BOOL},
                            {"path_join","io",TK_STRING},{"dirname","io",TK_STRING},
                            {"basename","io",TK_STRING},{"extension","io",TK_STRING},
                            {"normalize","io",TK_STRING},
                            /* @os */
                            {"args","os",TK_ARRAY},{"get_env","os",TK_STRING},
                            {"set_env","os",TK_VOID},{"current_dir","os",TK_STRING},
                            {"hostname","os",TK_STRING},{"arch","os",TK_STRING},
                            {"current_os","os",TK_INT},{"pid","os",TK_INT},
                            {"exit","os",TK_VOID},
                            /* @time */
                            {"now","time",TK_INT},{"now_ms","time",TK_INT},{"now_ns","time",TK_INT},
                            {"tick","time",TK_INT},{"elapsed_ms","time",TK_INT},
                            {"year","time",TK_INT},{"month","time",TK_INT},{"day","time",TK_INT},
                            {"hour","time",TK_INT},{"minute","time",TK_INT},{"second","time",TK_INT},
                            {"weekday","time",TK_INT},
                            {"format","time",TK_STRING},{"to_iso","time",TK_STRING},
                            {"date","time",TK_STRING},{"to_clock","time",TK_STRING},
                            /* @uuid */
                            {"generate_hyphenated","uuid",TK_STRING},{"generate","uuid",TK_STRING},
                            {"is_valid","uuid",TK_BOOL},
                            /* @bytes */
                            {"from_string","bytes",TK_ARRAY},{"from_hex","bytes",TK_ARRAY},
                            {"from_base64","bytes",TK_ARRAY},
                            {"to_string","bytes",TK_STRING},{"to_hex","bytes",TK_STRING},
                            {"to_base64","bytes",TK_STRING},
                            /* @binary */
                            {"encode_i8","binary",TK_ARRAY},{"encode_u8","binary",TK_ARRAY},
                            {"encode_i16_le","binary",TK_ARRAY},{"encode_i16_be","binary",TK_ARRAY},
                            {"encode_u16_le","binary",TK_ARRAY},{"encode_u16_be","binary",TK_ARRAY},
                            {"encode_i32_le","binary",TK_ARRAY},{"encode_i32_be","binary",TK_ARRAY},
                            {"encode_u32_le","binary",TK_ARRAY},{"encode_u32_be","binary",TK_ARRAY},
                            {"encode_i64_le","binary",TK_ARRAY},{"encode_i64_be","binary",TK_ARRAY},
                            {"encode_u64_le","binary",TK_ARRAY},{"encode_u64_be","binary",TK_ARRAY},
                            {"encode_i128_le","binary",TK_ARRAY},{"encode_i128_be","binary",TK_ARRAY},
                            {"encode_u128_le","binary",TK_ARRAY},{"encode_u128_be","binary",TK_ARRAY},
                            {"encode_i256_le","binary",TK_ARRAY},{"encode_i256_be","binary",TK_ARRAY},
                            {"encode_u256_le","binary",TK_ARRAY},{"encode_u256_be","binary",TK_ARRAY},
                            {"encode_f32_le","binary",TK_ARRAY},{"encode_f32_be","binary",TK_ARRAY},
                            {"encode_f64_le","binary",TK_ARRAY},{"encode_f64_be","binary",TK_ARRAY},
                            {"decode_i8","binary",TK_INT},{"decode_u8","binary",TK_INT},
                            {"decode_i16_le","binary",TK_INT},{"decode_i16_be","binary",TK_INT},
                            {"decode_u16_le","binary",TK_INT},{"decode_u16_be","binary",TK_INT},
                            {"decode_i32_le","binary",TK_INT},{"decode_i32_be","binary",TK_INT},
                            {"decode_u32_le","binary",TK_INT},{"decode_u32_be","binary",TK_INT},
                            {"decode_i64_le","binary",TK_INT},{"decode_i64_be","binary",TK_INT},
                            {"decode_u64_le","binary",TK_INT},{"decode_u64_be","binary",TK_INT},
                            {"decode_i128_le","binary",TK_INT},{"decode_i128_be","binary",TK_INT},
                            {"decode_u128_le","binary",TK_INT},{"decode_u128_be","binary",TK_INT},
                            {"decode_i256_le","binary",TK_INT},{"decode_i256_be","binary",TK_INT},
                            {"decode_u256_le","binary",TK_INT},{"decode_u256_be","binary",TK_INT},
                            {"decode_f32_le","binary",TK_FLOAT},{"decode_f32_be","binary",TK_FLOAT},
                            {"decode_f64_le","binary",TK_FLOAT},{"decode_f64_be","binary",TK_FLOAT},
                            /* @csv */
                            {"parse","csv",TK_ARRAY},{"read_file","csv",TK_ARRAY},
                            {"headers","csv",TK_ARRAY},
                            {"write","csv",TK_BOOL},{"write_file","csv",TK_BOOL},
                            {"format","csv",TK_STRING},{"encode","csv",TK_STRING},
                            /* @sqlite */
                            {"open","sqlite",TK_UNKNOWN},{"close","sqlite",TK_VOID},
                            {"exec","sqlite",TK_BOOL},{"query","sqlite",TK_ARRAY},
                            /* @threads */
                            {"spawn","threads",TK_UNKNOWN},{"join","threads",TK_VOID},
                            {"get_id","threads",TK_INT},
                            /* @sync */
                            {"mutex","sync",TK_UNKNOWN},{"lock","sync",TK_VOID},
                            {"unlock","sync",TK_VOID},{"try_lock","sync",TK_VOID},
                            {"destroy","sync",TK_VOID},
                            /* @atomic */
                            {"load","atomic",TK_INT},{"store","atomic",TK_VOID},
                            {"add","atomic",TK_INT},{"sub","atomic",TK_INT},
                            {"exchange","atomic",TK_INT},{"cas","atomic",TK_BOOL},
                            {"and","atomic",TK_INT},{"or","atomic",TK_INT},{"xor","atomic",TK_INT},
                            {"spinlock","atomic",TK_UNKNOWN},{"spin_lock","atomic",TK_VOID},
                            {"spin_trylock","atomic",TK_BOOL},{"spin_unlock","atomic",TK_VOID},
                            {"fence","atomic",TK_VOID},
                            /* @channels */
                            {"open","channels",TK_UNKNOWN},{"send","channels",TK_VOID},
                            {"receive","channels",TK_INT},{"close","channels",TK_VOID},
                            /* @server */
                            {"add_router","server",TK_UNKNOWN},{"add_route","server",TK_VOID},
                            {"listen","server",TK_VOID},{"cors","server",TK_VOID},
                            {"use","server",TK_VOID},{"text","server",TK_UNKNOWN},
                            {"json","server",TK_UNKNOWN},{"html","server",TK_UNKNOWN},
                            {"redirect","server",TK_UNKNOWN},{"parse_json","server",TK_UNKNOWN},
                            /* @http */
                            {"get","http",TK_UNKNOWN},{"post","http",TK_UNKNOWN},
                            {"put","http",TK_UNKNOWN},{"delete","http",TK_UNKNOWN},
                            {"head","http",TK_UNKNOWN},{"patch","http",TK_UNKNOWN},
                            {"request","http",TK_UNKNOWN},{"json_body","http",TK_STRING},
                            /* @net */
                            {"listen","net",TK_UNKNOWN},{"connect","net",TK_UNKNOWN},
                            {"accept","net",TK_UNKNOWN},{"send","net",TK_INT},
                            {"receive","net",TK_STRING},{"resolve","net",TK_STRING},
                            {"close","net",TK_VOID},
                            /* @fmt */
                            {"sprintf","fmt",TK_STRING},{"format","fmt",TK_STRING},
                            {"pad_left","fmt",TK_STRING},{"pad_right","fmt",TK_STRING},
                            {"center","fmt",TK_STRING},{"int_to_hex","fmt",TK_STRING},
                            {"int_to_binary","fmt",TK_STRING},{"int_to_octal","fmt",TK_STRING},
                            {"float_fixed","fmt",TK_STRING},{"float_sci","fmt",TK_STRING},
                            {"printf","fmt",TK_VOID},
                            /* @mem */
                            {"arena","mem",TK_UNKNOWN},{"usage","mem",TK_INT},
                            {"free","mem",TK_VOID},{"reset","mem",TK_VOID},
                            {"destroy","mem",TK_VOID},
                            {NULL,NULL,TK_UNKNOWN}
                        };
                        /* Check for math functions whose return type depends on argument */
                        if (!found_in_using && (strcmp(fn_name, "abs") == 0 || strcmp(fn_name, "neg") == 0 ||
                            strcmp(fn_name, "min") == 0 || strcmp(fn_name, "max") == 0 ||
                            strcmp(fn_name, "clamp") == 0)) {
                            for (int ui = 0; ui < tc->using_module_count; ui++) {
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "math") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        EzType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                                        result = (arg_t && arg_t->kind == TK_FLOAT) ? &TYPE_FLOAT : &TYPE_INT;
                                    } else {
                                        result = &TYPE_INT;
                                    }
                                    break;
                                }
                            }
                        }
                        /* Random functions whose return type depends on argument */
                        if (!found_in_using && (strcmp(fn_name, "choice") == 0 ||
                            strcmp(fn_name, "shuffle") == 0 || strcmp(fn_name, "sample") == 0)) {
                            for (int ui = 0; ui < tc->using_module_count; ui++) {
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "random") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        EzType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                                        if (strcmp(fn_name, "choice") == 0) {
                                            result = (arr_t && arr_t->element_type) ? type_from_name(arr_t->element_type) : &TYPE_INT;
                                        } else {
                                            result = (arr_t && arr_t->element_type) ? type_array(arr_t->element_type) : type_array("int");
                                        }
                                    } else {
                                        result = (strcmp(fn_name, "choice") == 0) ? &TYPE_INT : type_array("int");
                                    }
                                    break;
                                }
                            }
                        }
                        /* Maps functions whose return type depends on map key/value types */
                        if (!found_in_using && (strcmp(fn_name, "get_keys") == 0 || strcmp(fn_name, "get_values") == 0)) {
                            for (int ui = 0; ui < tc->using_module_count; ui++) {
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "maps") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        EzType *map_t = resolve_expr(tc, node->data.call.args[0]);
                                        if (strcmp(fn_name, "get_keys") == 0)
                                            result = type_array(map_t && map_t->key_type ? map_t->key_type : "string");
                                        else
                                            result = type_array(map_t && map_t->value_type ? map_t->value_type : "string");
                                    } else {
                                        result = type_array("string");
                                    }
                                    break;
                                }
                            }
                        }
                        for (int ui = 0; ui < tc->using_module_count && !found_in_using; ui++) {
                            const char *umod = tc->using_modules[ui];
                            /* Resolve alias to actual module name */
                            const char *real_mod = tc_resolve_alias(tc, umod);
                            /* 1) Try hardcoded stdlib table */
                            for (int fi = 0; using_funcs[fi].func; fi++) {
                                if (strcmp(fn_name, using_funcs[fi].func) == 0 &&
                                    strcmp(real_mod, using_funcs[fi].mod) == 0) {
                                    found_in_using = true;
                                    switch (using_funcs[fi].ret) {
                                    case TK_STRING: result = &TYPE_STRING; break;
                                    case TK_FLOAT:  result = &TYPE_FLOAT; break;
                                    case TK_BOOL:   result = &TYPE_BOOL; break;
                                    case TK_INT:    result = &TYPE_INT; break;
                                    case TK_VOID:   result = &TYPE_VOID; break;
                                    default:        result = &TYPE_UNKNOWN; break;
                                    }
                                    break;
                                }
                            }
                            /* 2) Try user-defined module: look up <module>_<func> */
                            if (!found_in_using) {
                                char prefixed[EZ_MSG_BUF_SIZE];
                                snprintf(prefixed, sizeof(prefixed), "%s_%s", real_mod, fn_name);
                                FuncSig *sig = find_func(tc, prefixed);
                                if (sig) {
                                    if (sig->is_private) {
                                        diag_error_codef(tc->diag, "E4015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, fn_name);
                                        found_in_using = true;
                                    } else {
                                        found_in_using = true;
                                        if (sig->return_count > 0) {
                                            result = sig->return_types[0];
                                        } else {
                                            result = &TYPE_VOID;
                                        }
                                        sig->used = true;
                                    }
                                }
                            }
                            if (found_in_using) {
                                /* Mark module as used */
                                for (int mi = 0; mi < tc->import_count; mi++) {
                                    if (strcmp(tc->imported_modules[mi], umod) == 0 ||
                                        strcmp(tc->imported_modules[mi], real_mod) == 0) {
                                        tc->import_used[mi] = true;
                                        break;
                                    }
                                }
                            }
                        }
                        if (found_in_using) {
                            /* Type already set above */
                        } else {
                            /* Check if it's a variable holding a function reference */
                            Symbol *fn_sym = scope_lookup(tc->current_scope, fn_name);
                            if (fn_sym && fn_sym->type && strcmp(type_name(fn_sym->type), "func") == 0) {
                                fn_sym->used = true;
                                result = &TYPE_UNKNOWN;
                                /* Arity/type validation for func refs is done
                                 * at the earlier func-var branch; avoid
                                 * re-emitting the same diagnostic here. */
                            } else {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg), "undefined function '%s'", fn_name);
                                const char *suggestion = suggest_name(tc, fn_name);
                                /* Point at the function name, not the ( */
                                AstNode *fn_node = node->data.call.function;
                                int el = fn_node ? fn_node->token.line : node->token.line;
                                int ec = fn_node ? fn_node->token.column : node->token.column;
                                if (suggestion) {
                                    char help[EZ_MSG_BUF_SIZE];
                                    snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                                    diag_error_help(tc->diag, "E4002", strdup(msg),
                                        NODE_FILE(tc, node), el, ec, 0, strdup(help));
                                } else {
                                    diag_error_msg(tc->diag, "E4002", strdup(msg),
                                        NODE_FILE(tc, node), el, ec, 0);
                                }
                            }
                        }
                    }
                }
            }
        }
        break;
    }

    case NODE_MEMBER_EXPR: {
        /* Resolve object type first (sets type table entry for the object) */
        AstNode *obj = node->data.member.object;
        const char *member = node->data.member.member;

        /* Handle mod.Enum.VALUE or mod.Struct.field triple chain */
        if (obj->kind == NODE_MEMBER_EXPR &&
            obj->data.member.object->kind == NODE_LABEL) {
            const char *mod_name = obj->data.member.object->data.label.value;
            const char *type_name = obj->data.member.member;
            /* Build prefixed type name: mod_Type */
            char prefixed_type[256];
            snprintf(prefixed_type, sizeof(prefixed_type), "%s_%s", mod_name, type_name);
            /* Check if it's a module-qualified enum access */
            if (is_enum_name(tc, prefixed_type)) {
                bool is_str_enum = false;
                for (int ei = 0; ei < tc->enum_count; ei++) {
                    if (strcmp(tc->enum_names[ei], prefixed_type) == 0) {
                        is_str_enum = tc->enum_is_string[ei];
                        break;
                    }
                }
                /* Mark module as used */
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], mod_name) == 0) {
                        tc->import_used[mi] = true;
                        break;
                    }
                }
                result = is_str_enum ? &TYPE_STRING : &TYPE_INT;
                break;
            }
        }

        resolve_expr(tc, obj);

        if (obj->kind == NODE_LABEL) {
            const char *obj_name = obj->data.label.value;

            /* Mark module as used (for member access like math.PI) */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], obj_name) == 0) {
                    tc->import_used[mi] = true;
                    break;
                }
            }

            /* Check for module constants */
            if (strcmp(obj_name, "math") == 0) {
                result = &TYPE_FLOAT; /* PI, E, TAU, etc. */
                break;
            }
            if (strcmp(obj_name, "os") == 0) {
                result = &TYPE_INT; /* MAC_OS, LINUX, etc. */
                break;
            }

            /* Check if it's an enum access: Color.RED */
            if (is_enum_name(tc, obj_name)) {
                /* Check if this is a string enum and validate member exists */
                bool is_str_enum = false;
                bool member_found = false;
                for (int ei = 0; ei < tc->enum_count; ei++) {
                    if (strcmp(tc->enum_names[ei], obj_name) == 0) {
                        is_str_enum = tc->enum_is_string[ei];
                        for (int vi = 0; vi < tc->enum_value_counts[ei]; vi++) {
                            if (strcmp(tc->enum_values[ei][vi], member) == 0) {
                                member_found = true;
                                break;
                            }
                        }
                        break;
                    }
                }
                if (!member_found) {
                    diag_error_codef(tc->diag, "E3047", NODE_FILE(tc, node), node->token.line, node->token.column, 0, obj_name, member);
                }
                result = is_str_enum ? &TYPE_STRING : type_enum(obj_name);
                break;
            }

            /* C interop constant access: c.EOF, c.NULL, etc. */
            if (strcmp(obj_name, "c") == 0 && tc_is_imported_module(tc, "c")) {
                result = &TYPE_UNKNOWN;
                break;
            }

            /* Check for user-module constant access: mod.CONST */
            if (tc_is_imported_module(tc, obj_name)) {
                char prefixed[EZ_MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", obj_name, member);
                Symbol *mod_sym = scope_lookup(tc->current_scope, prefixed);
                if (mod_sym) {
                    mod_sym->used = true;
                    result = mod_sym->type;
                    /* Mark module as used */
                    for (int mi = 0; mi < tc->import_count; mi++) {
                        if (strcmp(tc->imported_modules[mi], obj_name) == 0) {
                            tc->import_used[mi] = true;
                            break;
                        }
                    }
                    break;
                }
            }

            /* Otherwise it's a struct field or multi-return access */
            Symbol *sym = scope_lookup(tc->current_scope, obj_name);
            /* Multi-return .v0/.v1 access takes priority over struct field
             * lookup when the symbol has ret_types set. Without this, stdlib
             * functions returning struct types (Socket, HttpResponse, etc.)
             * would enter struct_field_type() which fails on .v0/.v1. */
            if (sym && sym->ret_types && member[0] == 'v' && member[1] >= '0' && member[1] <= '9') {
                int idx = member[1] - '0';
                if (idx < sym->ret_count) {
                    result = sym->ret_types[idx];
                } else {
                    diag_error_codef(tc->diag, "E3006", NODE_FILE(tc, node), node->token.line, node->token.column, 0, sym->ret_count, idx + 1);
                    result = &TYPE_UNKNOWN;
                }
            } else if (sym && sym->type->kind == TK_STRUCT) {
                result = struct_field_type(tc, sym->type->name, member);
                /* : func-typed fields resolve as TK_UNKNOWN (name="func")
                 * via type_from_name because "func" maps to TK_UNKNOWN. Don't
                 * emit E3010 "no field" when the field genuinely exists; it's
                 * just a func field. */
                bool is_func_field = (result->kind == TK_UNKNOWN && result->name &&
                                      strcmp(result->name, "func") == 0);
                if (result->kind == TK_UNKNOWN && !is_func_field && member[0] != 'v') {
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0, sym->type->name, member);
                }
            } else if (sym && member[0] == 'v' && member[1] >= '0' && member[1] <= '9') {
                /* Multi-return .v0/.v1/.v2 access; use stored return types */
                int idx = member[1] - '0';
                if (sym->ret_types && idx < sym->ret_count) {
                    result = sym->ret_types[idx];
                } else if (idx == 0) {
                    result = sym->type;
                } else if (sym->ret_types && idx >= sym->ret_count) {
                    /* More variables than the function returns */
                    diag_error_codef(tc->diag, "E3006", NODE_FILE(tc, node), node->token.line, node->token.column, 0, sym->ret_count, idx + 1);
                    result = &TYPE_UNKNOWN;
                } else if (!sym->ret_types && idx > 0) {
                    /* Single-return function used in multi-variable assignment */
                    diag_error_msg(tc->diag, "E3006",
                        strdup("too many variables; the function returns only 1 value"),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_UNKNOWN;
                } else {
                    result = type_from_name("Error"); /* fallback for (T, Error) pattern */
                }
            } else if (sym && sym->type->kind == TK_POINTER) {
                /* Pointer auto-deref field access */
                result = struct_field_type(tc, sym->type->element_type, member);
                if (result->kind == TK_UNKNOWN && member[0] != 'v') {
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0, sym->type->element_type, member);
                }
            } else if (sym && sym->type->kind != TK_UNKNOWN &&
                       sym->type->kind != TK_STRUCT && sym->type->kind != TK_ENUM &&
                       sym->type->kind != TK_POINTER &&
                       !(member[0] == 'v' && member[1] >= '0' && member[1] <= '9')) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type '%s' has no fields; only structs support field access",
                    type_name(sym->type));
                diag_error_msg(tc->diag, "E3013", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Struct-namespaced function or enum access: Type.func() / Type.MEMBER */
            if (!sym && is_struct_name(tc, obj_name)) {
                /* Check if member is a field; can't access fields on the type itself */
                EzType *ft = struct_field_type(tc, obj_name, member);
                if (ft && ft->kind != TK_UNKNOWN) {
                    diag_error_codef(tc->diag, "E3044", NODE_FILE(tc, node), node->token.line, node->token.column, 0, member, obj_name);
                }
                result = &TYPE_UNKNOWN;
            }
        } else if (obj->kind == NODE_MEMBER_EXPR) {
            /* Check for module-qualified enum: lib.Color.RED */
            if (obj->data.member.object->kind == NODE_LABEL) {
                const char *mod = obj->data.member.object->data.label.value;
                const char *type_n = obj->data.member.member;
                if (mod[0] >= 'a' && mod[0] <= 'z' &&
                    type_n[0] >= 'A' && type_n[0] <= 'Z') {
                    char prefixed[EZ_MSG_BUF_SIZE];
                    snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, type_n);
                    if (is_enum_name(tc, prefixed)) {
                        result = &TYPE_INT;
                        /* Mark module as used */
                        for (int mi = 0; mi < tc->import_count; mi++) {
                            if (strcmp(tc->imported_modules[mi], mod) == 0) {
                                tc->import_used[mi] = true;
                                break;
                            }
                        }
                        break;
                    }
                }
            }
            /* Nested member access: a.b.c; resolve a.b first, then look up .c */
            EzType *obj_t = typetable_get(tc->type_table, obj);
            if (!obj_t) obj_t = resolve_expr(tc, obj);
            if (obj_t && obj_t->kind == TK_STRUCT) {
                result = struct_field_type(tc, obj_t->name, member);
            } else if (obj_t && obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_STRUCT) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type '%s' has no fields; only structs support field access",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else {
            /* Object is an expression (e.g. foo().bar); resolve its type */
            EzType *obj_t = resolve_expr(tc, obj);
            if (obj_t && obj_t->kind == TK_STRUCT) {
                result = struct_field_type(tc, obj_t->name, member);
            } else if (obj_t && obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_VOID) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type '%s' has no fields or functions; only structs support member access",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        break;
    }

    case NODE_INDEX_EXPR: {
        EzType *left = resolve_expr(tc, node->data.index_expr.left);
        EzType *idx_t = resolve_expr(tc, node->data.index_expr.index);
        /* E3003: array index must be integer */
        if (left->kind == TK_ARRAY && idx_t->kind != TK_UNKNOWN &&
            !is_int_kind(idx_t->kind) && idx_t->kind != TK_BYTE) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "array index must be an integer, got %s", type_name(idx_t));
            diag_error_msg(tc->diag, "E3003", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Reject negative literal index at compile time. Applies to
         * arrays and strings (); the analogous runtime panic
         * fires for both, and there's no reason to wait until then
         * when the index is a literal '-N'. */
        if ((left->kind == TK_ARRAY || left->kind == TK_STRING) &&
            node->data.index_expr.index->kind == NODE_PREFIX_EXPR &&
            strcmp(node->data.index_expr.index->data.prefix.op, "-") == 0 &&
            node->data.index_expr.index->data.prefix.right->kind == NODE_INT_VALUE) {
            const char *what = left->kind == TK_STRING ? "string" : "array";
            char msg[EZ_TYPE_NAME_MAX];
            snprintf(msg, sizeof(msg), "%s index cannot be negative", what);
            diag_error_msg(tc->diag, "E3003", strdup(msg),
                NODE_FILE(tc, node->data.index_expr.index), node->data.index_expr.index->token.line,
                node->data.index_expr.index->token.column, 0);
        }
        if (left->kind == TK_ARRAY && left->element_type) {
            result = type_from_name(left->element_type);
        } else if (left->kind == TK_MAP && left->value_type) {
            result = type_from_name(left->value_type);
            /* Check map key type matches. Enum keys are int-backed, so accept
             * int expressions (and enum members, which resolve as int) when
             * the declared key is a user enum name. */
            if (left->key_type && idx_t->kind != TK_UNKNOWN) {
                EzType *key_t = tc_type_from_name(tc, left->key_type);
                bool declared_is_enum = is_enum_name(tc, left->key_type);
                bool compatible =
                    key_t->kind == TK_UNKNOWN ||
                    key_t->kind == idx_t->kind ||
                    (is_int_kind(key_t->kind) && is_int_kind(idx_t->kind)) ||
                    (declared_is_enum && is_int_kind(idx_t->kind));
                if (!compatible) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "map key type mismatch: expected '%s', got '%s'",
                        left->key_type, type_name(idx_t));
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        } else if (left->kind == TK_STRING) {
            result = &TYPE_CHAR;
        } else if (left->kind != TK_UNKNOWN) {
            diag_error_codef(tc->diag, "E3008", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_name(left));
        }
        break;
    }

    case NODE_ARRAY_VALUE:
        if (node->data.array_value.count > 0) {
            EzType *first = resolve_expr(tc, node->data.array_value.elements[0]);
            result = type_array(type_name(first));
            /* Validate all elements have the same type */
            for (int i = 1; i < node->data.array_value.count; i++) {
                EzType *ei = resolve_expr(tc, node->data.array_value.elements[i]);
                if (!ei || ei->kind == TK_UNKNOWN || !first || first->kind == TK_UNKNOWN)
                    continue;
                bool compatible = (ei->kind == first->kind) ||
                    (is_int_kind(ei->kind) && is_int_kind(first->kind)) ||
                    (ei->kind == TK_BYTE && is_int_kind(first->kind)) ||
                    (is_int_kind(ei->kind) && first->kind == TK_BYTE);
                if (!compatible) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "array elements must all be the same type; element %d is '%s' but the array is '%s'",
                        i, type_name(ei), type_name(first));
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node->data.array_value.elements[i]), node->data.array_value.elements[i]->token.line,
                        node->data.array_value.elements[i]->token.column, 0);
                    break;
                }
            }
        }
        break;

    case NODE_MAP_VALUE: {
        /* Resolve key and value types */
        for (int i = 0; i < node->data.map_value.count; i++) {
            EzType *kt = resolve_expr(tc, node->data.map_value.keys[i]);
            EzType *vt = resolve_expr(tc, node->data.map_value.values[i]);
            /* : void can't be a map key or value. */
            reject_void_in_context(tc, node->data.map_value.keys[i], kt, "map key");
            reject_void_in_context(tc, node->data.map_value.values[i], vt, "map value");
        }
        /* E12006: Check for duplicate keys in map literal */
        for (int i = 0; i < node->data.map_value.count; i++) {
            AstNode *ki = node->data.map_value.keys[i];
            for (int j = i + 1; j < node->data.map_value.count; j++) {
                AstNode *kj = node->data.map_value.keys[j];
                bool dup = false;
                if (ki->kind == NODE_STRING_VALUE && kj->kind == NODE_STRING_VALUE &&
                    strcmp(ki->data.string_value.value, kj->data.string_value.value) == 0) {
                    dup = true;
                } else if (ki->kind == NODE_INT_VALUE && kj->kind == NODE_INT_VALUE &&
                    ki->data.int_value.value == kj->data.int_value.value) {
                    dup = true;
                }
                if (dup) {
                    diag_error_code(tc->diag, "E12006", NODE_FILE(tc, kj), kj->token.line, kj->token.column, 0);
                }
            }
        }
        EzType *t = type_alloc();
        t->kind = TK_MAP;
        t->name = "map";
        if (node->data.map_value.count > 0) {
            EzType *kt = typetable_get(tc->type_table, node->data.map_value.keys[0]);
            EzType *vt = typetable_get(tc->type_table, node->data.map_value.values[0]);
            t->key_type = kt ? type_name(kt) : "unknown";
            t->value_type = vt ? type_name(vt) : "unknown";
        }
        result = t;
        break;
    }

    case NODE_STRUCT_VALUE: {
        const char *sname = node->data.struct_value.name;
        tc_mark_type_module_used(tc, sname);
        StructInfo *si = find_struct(tc, sname);
        /* E4016: reject undefined/unimported struct types in struct literals */
        if (!si && !is_struct_name(tc, sname)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "undefined type '%s'; check the spelling or import the module that defines it",
                sname);
            diag_error_msg(tc->diag, "E4016", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            result = &TYPE_UNKNOWN;
            break;
        }
        /* E2015: check for duplicate field names in struct literal */
        for (int i = 0; i < node->data.struct_value.count; i++) {
            if (!node->data.struct_value.field_names[i]) continue;
            for (int j = 0; j < i; j++) {
                if (!node->data.struct_value.field_names[j]) continue;
                if (strcmp(node->data.struct_value.field_names[j],
                           node->data.struct_value.field_names[i]) == 0) {
                    diag_error_codef(tc->diag, "E2015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.struct_value.field_names[i]);
                    break;
                }
            }
        }
        for (int i = 0; i < node->data.struct_value.count; i++) {
            EzType *val_t = resolve_expr(tc, node->data.struct_value.field_values[i]);
            /* Validate field exists */
            if (si && node->data.struct_value.field_names[i]) {
                const char *fname = node->data.struct_value.field_names[i];
                bool found = false;
                EzType *expected_t = NULL;
                for (int j = 0; j < si->field_count; j++) {
                    if (strcmp(si->field_names[j], fname) == 0) {
                        found = true;
                        expected_t = si->field_types[j];
                        break;
                    }
                }
                if (!found) {
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0, sname, fname);
                } else if (expected_t && val_t->kind != TK_UNKNOWN &&
                           expected_t->kind != TK_UNKNOWN &&
                           expected_t->kind != val_t->kind &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_ENUM) &&
                           !(expected_t->kind == TK_ENUM && is_int_kind(val_t->kind)) &&
                           !(expected_t->kind == TK_STRUCT && is_int_kind(val_t->kind)) &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_STRUCT) &&
                           !(expected_t->kind == TK_FLOAT && is_int_kind(val_t->kind)) &&
                           /* nil is a valid value for pointer and Error fields */
                           !(val_t->kind == TK_NIL &&
                             (expected_t->kind == TK_POINTER || expected_t->kind == TK_ERROR))) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "field '%s' of struct '%s': expected %s, got %s",
                        fname, sname, type_name(expected_t), type_name(val_t));
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }
        /* : for generic structs, infer the wildcard binding from
         * the field values and record the instantiation on the struct
         * decl so codegen can emit per-binding typedefs. */
        AstNode *sdecl = find_struct_in_program(tc->program, sname);
        if (sdecl && sdecl->data.struct_decl.is_generic) {
            const char *binding = NULL;
            for (int i = 0; i < node->data.struct_value.count; i++) {
                const char *fname = node->data.struct_value.field_names[i];
                if (!fname) continue;
                /* Find the field's declared type in the struct decl */
                for (int j = 0; j < sdecl->data.struct_decl.field_count; j++) {
                    if (strcmp(sdecl->data.struct_decl.fields[j].name, fname) == 0 &&
                        sdecl->data.struct_decl.fields[j].type_name &&
                        strcmp(sdecl->data.struct_decl.fields[j].type_name, "?") == 0) {
                        EzType *val_t = typetable_get(tc->type_table,
                            node->data.struct_value.field_values[i]);
                        if (!val_t) val_t = resolve_expr(tc, node->data.struct_value.field_values[i]);
                        if (val_t && val_t->kind != TK_UNKNOWN) {
                            const char *concrete = type_name(val_t);
                            if (!binding) {
                                binding = concrete;
                            } else if (strcmp(binding, concrete) != 0) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "wildcard type conflict in struct '%s': '?' was bound to %s, but field '%s' is %s",
                                    sname, binding, fname, concrete);
                                diag_error_msg(tc->diag, "E3001", strdup(msg),
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            }
                        }
                        break;
                    }
                }
            }
            if (binding) {
                node->data.struct_value.wildcard_binding = strdup(binding);
                /* Record instantiation on the struct decl */
                bool already = false;
                for (int ii = 0; ii < sdecl->data.struct_decl.instantiation_count; ii++) {
                    if (strcmp(sdecl->data.struct_decl.instantiations[ii], binding) == 0) {
                        already = true;
                        break;
                    }
                }
                if (!already) {
                    int n = sdecl->data.struct_decl.instantiation_count;
                    sdecl->data.struct_decl.instantiations = realloc(
                        (void *)sdecl->data.struct_decl.instantiations,
                        sizeof(const char *) * (size_t)(n + 1));
                    sdecl->data.struct_decl.instantiations[n] = strdup(binding);
                    sdecl->data.struct_decl.instantiation_count = n + 1;
                }
                /* Return mangled struct type */
                char mangled[256];
                size_t pos = snprintf(mangled, sizeof(mangled), "%s__", sname);
                for (const char *c = binding; *c && pos < sizeof(mangled) - 1; c++) {
                    mangled[pos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
                }
                mangled[pos] = '\0';
                result = type_struct(strdup(mangled));
            } else {
                result = type_struct(sname);
            }
        } else {
            result = type_struct(sname);
        }
        break;
    }

    case NODE_RANGE_EXPR: {
        /* Validate range arguments are integer types */
        AstNode *parts[] = { node->data.range_expr.start,
                             node->data.range_expr.end,
                             node->data.range_expr.step };
        const char *labels[] = { "start", "end", "step" };
        for (int ri = 0; ri < 3; ri++) {
            if (!parts[ri]) continue;
            EzType *pt = resolve_expr(tc, parts[ri]);
            if (pt->kind != TK_UNKNOWN && !is_int_kind(pt->kind)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "range() %s argument must be an integer type, got '%s'",
                    labels[ri], type_name(pt));
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), parts[ri]->token.line,
                    parts[ri]->token.column, 0);
            }
        }
        EzType *rt = type_alloc();
        rt->kind = TK_INT;
        rt->name = "Range<int>";
        result = rt;
        break;
    }

    case NODE_CAST_EXPR: {
        EzType *src_t = resolve_expr(tc, node->data.cast.value);
        EzType *dst_t = type_from_name(node->data.cast.target_type);
        /* Allowlist-based cast validation */
        if (src_t && src_t->kind != TK_UNKNOWN && dst_t && dst_t->kind != TK_UNKNOWN) {
            bool allowed = false;
            /* Same kind is always allowed (identity cast) */
            if (src_t->kind == dst_t->kind && src_t->kind != TK_ARRAY &&
                src_t->kind != TK_MAP && src_t->kind != TK_STRUCT)
                allowed = true;
            /* Numeric <-> Numeric (int, uint, float, char, byte, sized types) */
            if (type_is_numeric(src_t) && type_is_numeric(dst_t))
                allowed = true;
            /* Bool <-> Numeric */
            if ((src_t->kind == TK_BOOL && type_is_numeric(dst_t)) ||
                (type_is_numeric(src_t) && dst_t->kind == TK_BOOL))
                allowed = true;
            /* String -> int, uint, float (parsing) */
            if (src_t->kind == TK_STRING &&
                (dst_t->kind == TK_INT || dst_t->kind == TK_UINT || dst_t->kind == TK_FLOAT))
                allowed = true;
            /* Numeric/Bool -> String (stringification) */
            if ((type_is_numeric(src_t) || src_t->kind == TK_BOOL) && dst_t->kind == TK_STRING)
                allowed = true;
            /* String -> String (identity) */
            if (src_t->kind == TK_STRING && dst_t->kind == TK_STRING)
                allowed = true;
            /* Enum -> int/uint (enums are int-backed) */
            if (src_t->kind == TK_ENUM &&
                (dst_t->kind == TK_INT || dst_t->kind == TK_UINT))
                allowed = true;
            /* int/uint -> Enum (explicit reinterpretation) */
            if ((src_t->kind == TK_INT || src_t->kind == TK_UINT) &&
                dst_t->kind == TK_ENUM)
                allowed = true;
            /* Array -> Array: only when both element types are numeric */
            if (src_t->kind == TK_ARRAY && dst_t->kind == TK_ARRAY) {
                if (src_t->element_type && dst_t->element_type) {
                    EzType *src_elem = type_from_name(src_t->element_type);
                    EzType *dst_elem = type_from_name(dst_t->element_type);
                    if (type_is_numeric(src_elem) && type_is_numeric(dst_elem))
                        allowed = true;
                } else {
                    allowed = true;
                }
            }
            if (!allowed) {
                char tn[EZ_TYPE_NAME_MAX];
                if (src_t->kind == TK_ARRAY && src_t->element_type)
                    snprintf(tn, sizeof(tn), "[%s]", src_t->element_type);
                else
                    snprintf(tn, sizeof(tn), "%s", type_name(src_t));
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "cannot cast '%s' to '%s'",
                    tn, node->data.cast.target_type);
                diag_error_msg(tc->diag, "E3043", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        result = dst_t;
        break;
    }

    case NODE_NEW_EXPR:
        tc_mark_type_module_used(tc, node->data.new_expr.type_name);
        if (!is_struct_name(tc, node->data.new_expr.type_name)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "new() requires a struct type, but '%s' is not a struct",
                node->data.new_expr.type_name);
            diag_error_msg(tc->diag, "E3041", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        result = type_pointer(node->data.new_expr.type_name);
        break;

    case NODE_FUNC_REF: {
        /* Validate that the referenced function exists */
        const char *ref_name = NULL;
        if (node->data.func_ref.function->kind == NODE_LABEL) {
            ref_name = node->data.func_ref.function->data.label.value;
        } else if (node->data.func_ref.function->kind == NODE_MEMBER_EXPR) {
            /* Struct.func → lookup as Struct_func */
            AstNode *obj = node->data.func_ref.function->data.member.object;
            const char *member = node->data.func_ref.function->data.member.member;
            if (obj->kind == NODE_LABEL) {
                char *prefixed = malloc(strlen(obj->data.label.value) + strlen(member) + 2);
                sprintf(prefixed, "%s_%s", obj->data.label.value, member);
                ref_name = prefixed;
            }
        }
        FuncSig *ref_sig = ref_name ? find_func(tc, ref_name) : NULL;
        if (ref_sig) {
            ref_sig->used = true;
        } else if (ref_name && !tc_is_builtin(ref_name)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg), "undefined function '%s' in function reference", ref_name);
            const char *suggestion = suggest_name(tc, ref_name);
            if (suggestion) {
                char help[EZ_MSG_BUF_SIZE];
                snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                diag_error_help(tc->diag, "E4002", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0, strdup(help));
            } else {
                diag_error_msg(tc->diag, "E4002", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* Build a typed-func type from the referenced function's signature.
         * Canonical encoding: "func(p1,&p2,...)" with no "->R" suffix when
         * the function returns nothing; "func(...)->R" for a single return;
         * "func(...)->(R1,R2)" for multi-return. */
        if (ref_sig) {
            char buf[512];
            size_t bufsz = sizeof(buf);
            int n = snprintf(buf, bufsz, "func(");
            if ((size_t)n >= bufsz) n = (int)bufsz - 1;
            for (int i = 0; i < ref_sig->param_count && (size_t)n < bufsz - 1; i++) {
                bool mut_p = (ref_sig->decl && ref_sig->decl->kind == NODE_FUNC_DECL &&
                              i < ref_sig->decl->data.func_decl.param_count &&
                              ref_sig->decl->data.func_decl.params[i].mutable);
                const char *ptn = (ref_sig->param_types[i] && ref_sig->param_types[i]->name)
                    ? ref_sig->param_types[i]->name : "int";
                int w = snprintf(buf + n, bufsz - (size_t)n, "%s%s%s",
                    i ? "," : "", mut_p ? "&" : "", ptn);
                if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                else { n = (int)bufsz - 1; break; }
            }
            if ((size_t)n < bufsz - 1) {
                int w = snprintf(buf + n, bufsz - (size_t)n, ")");
                if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                else n = (int)bufsz - 1;
            }
            if (ref_sig->return_count == 1 && ref_sig->return_types[0] &&
                ref_sig->return_types[0]->name &&
                strcmp(ref_sig->return_types[0]->name, "void") != 0 &&
                (size_t)n < bufsz - 1) {
                int w = snprintf(buf + n, bufsz - (size_t)n, "->%s",
                    ref_sig->return_types[0]->name);
                if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                else n = (int)bufsz - 1;
            } else if (ref_sig->return_count > 1 && (size_t)n < bufsz - 1) {
                int w = snprintf(buf + n, bufsz - (size_t)n, "->(");
                if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                else n = (int)bufsz - 1;
                for (int i = 0; i < ref_sig->return_count && (size_t)n < bufsz - 1; i++) {
                    const char *rn = (ref_sig->return_types[i] && ref_sig->return_types[i]->name)
                        ? ref_sig->return_types[i]->name : "int";
                    w = snprintf(buf + n, bufsz - (size_t)n, "%s%s",
                        i ? "," : "", rn);
                    if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                    else { n = (int)bufsz - 1; break; }
                }
                if ((size_t)n < bufsz - 1) {
                    w = snprintf(buf + n, bufsz - (size_t)n, ")");
                    if (w > 0 && (size_t)w < bufsz - (size_t)n) n += w;
                    else n = (int)bufsz - 1;
                }
            }
            char *encoded = strdup(buf);
            result = type_from_name(encoded);
        } else {
            /* Unknown function: fall back to the legacy bare-func type
             * so downstream "is this callable" checks don't crash. */
            result = type_from_name("func");
        }
        break;
    }

    default:
        break;
    }

    if (!tc->suppress_typetable_writes) {
        typetable_set(tc->type_table, node, result);
    }
    return result;
}

/* Check if a name uses a reserved prefix that would collide with generated C.
 * Skip names starting with _ez_ as those are compiler-generated temporaries. */
static void check_reserved_name(TypeChecker *tc, const char *name, const char *file, int line, int col) {
    if (!name) return;
    /* Skip compiler-generated temps (_ez_tmp, _ez_or, _ez_idx, etc.) */
    if (strncmp(name, "_ez_", 4) == 0) return;
    if (strncmp(name, "ez_", 3) == 0 || strncmp(name, "Ez", 2) == 0) {
        diag_error_codef(tc->diag, "E4006", file, line, col, 0, name);
    }
}

/* --- Statement checking --- */

static void check_statement(TypeChecker *tc, AstNode *node);

/* Check if ALL paths through a block end in a return statement */
static bool block_has_return(AstNode *node); /* forward declaration */

static bool all_paths_return(AstNode *node) {
    if (!node) return false;
    if (node->kind == NODE_RETURN_STMT) return true;
    if (node->kind == NODE_BLOCK_STMT) {
        /* Last statement must be a return or all-paths-return construct */
        if (node->data.block.count == 0) return false;
        for (int i = 0; i < node->data.block.count; i++) {
            if (node->data.block.stmts[i]->kind == NODE_RETURN_STMT) return true;
        }
        AstNode *last = node->data.block.stmts[node->data.block.count - 1];
        return all_paths_return(last);
    }
    if (node->kind == NODE_IF_STMT) {
        /* Both branches must return */
        if (!node->data.if_stmt.alternative) return false;
        return all_paths_return(node->data.if_stmt.consequence) &&
               all_paths_return(node->data.if_stmt.alternative);
    }
    if (node->kind == NODE_WHEN_STMT) {
        if (!node->data.when_stmt.default_body) return false;
        if (!all_paths_return(node->data.when_stmt.default_body)) return false;
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            if (!all_paths_return(node->data.when_stmt.cases[i].body)) return false;
        }
        return true;
    }
    /* loop { ... return ... }; an infinite loop with a return always returns */
    if (node->kind == NODE_LOOP_STMT) {
        return block_has_return(node->data.loop_stmt.body);
    }
    return false;
}

/* Recursively check if an AST node or its children contain a return statement */
static bool block_has_return(AstNode *node) {
    if (!node) return false;
    if (node->kind == NODE_RETURN_STMT) return true;
    if (node->kind == NODE_BLOCK_STMT) {
        for (int i = 0; i < node->data.block.count; i++) {
            if (block_has_return(node->data.block.stmts[i])) return true;
        }
    }
    if (node->kind == NODE_IF_STMT) {
        if (block_has_return(node->data.if_stmt.consequence)) return true;
        if (block_has_return(node->data.if_stmt.alternative)) return true;
    }
    if (node->kind == NODE_WHEN_STMT) {
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            if (block_has_return(node->data.when_stmt.cases[i].body)) return true;
        }
        if (block_has_return(node->data.when_stmt.default_body)) return true;
    }
    /* loop { ... return ... }; loop always executes, so a return inside counts */
    if (node->kind == NODE_LOOP_STMT) {
        if (block_has_return(node->data.loop_stmt.body)) return true;
    }
    /* for/for_each/as_long_as may also contain returns */
    if (node->kind == NODE_FOR_STMT) {
        if (block_has_return(node->data.for_stmt.body)) return true;
    }
    if (node->kind == NODE_FOR_EACH_STMT) {
        if (block_has_return(node->data.for_each.body)) return true;
    }
    if (node->kind == NODE_WHILE_STMT) {
        if (block_has_return(node->data.while_stmt.body)) return true;
    }
    return false;
}

/* E3070: ensure must appear at the top level of a function body. The
 * codegen ensure-cleanup pass only walks the function-body block, so
 * an ensure inside a for/while/if/etc. body would silently never fire.
 * Rather than introducing closure-capture semantics for nested ensures,
 * reject them at the type-check stage with a clear error. */
static void check_no_nested_ensure(TypeChecker *tc, AstNode *node, bool nested) {
    if (!node) return;
    if (node->kind == NODE_ENSURE_STMT && nested) {
        diag_error_code(tc->diag, "E3070", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        return;
    }
    if (node->kind == NODE_BLOCK_STMT) {
        for (int i = 0; i < node->data.block.count; i++) {
            check_no_nested_ensure(tc, node->data.block.stmts[i], nested);
        }
        return;
    }
    switch (node->kind) {
    case NODE_IF_STMT:
        check_no_nested_ensure(tc, node->data.if_stmt.consequence, true);
        check_no_nested_ensure(tc, node->data.if_stmt.alternative, true);
        break;
    case NODE_FOR_STMT:
        check_no_nested_ensure(tc, node->data.for_stmt.body, true);
        break;
    case NODE_FOR_EACH_STMT:
        check_no_nested_ensure(tc, node->data.for_each.body, true);
        break;
    case NODE_WHILE_STMT:
        check_no_nested_ensure(tc, node->data.while_stmt.body, true);
        break;
    case NODE_LOOP_STMT:
        check_no_nested_ensure(tc, node->data.loop_stmt.body, true);
        break;
    case NODE_WHEN_STMT:
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            check_no_nested_ensure(tc, node->data.when_stmt.cases[i].body, true);
        }
        check_no_nested_ensure(tc, node->data.when_stmt.default_body, true);
        break;
    default:
        break;
    }
}

static void check_block(TypeChecker *tc, AstNode *node) {
    if (!node || node->kind != NODE_BLOCK_STMT) return;
    bool seen_return = false;
    for (int i = 0; i < node->data.block.count; i++) {
        AstNode *stmt = node->data.block.stmts[i];
        if (seen_return && stmt) {
            diag_warning_code(tc->diag, "W2003", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            break; /* only warn once per block */
        }
        check_statement(tc, stmt);
        if (stmt && stmt->kind == NODE_RETURN_STMT) {
            seen_return = true;
        }
    }
}

static void check_statement(TypeChecker *tc, AstNode *node) {
    if (!node) return;

    /* E2056: executable statements not allowed at file scope */
    if (tc->func_depth == 0) {
        bool is_executable = (node->kind == NODE_ASSIGN_STMT ||
                              node->kind == NODE_IF_STMT ||
                              node->kind == NODE_FOR_STMT ||
                              node->kind == NODE_FOR_EACH_STMT ||
                              node->kind == NODE_WHILE_STMT ||
                              node->kind == NODE_LOOP_STMT ||
                              node->kind == NODE_EXPR_STMT ||
                              node->kind == NODE_RETURN_STMT ||
                              node->kind == NODE_BREAK_STMT ||
                              node->kind == NODE_CONTINUE_STMT ||
                              node->kind == NODE_WHEN_STMT);
        if (is_executable) {
            diag_error_msg(tc->diag, "E2056",
                strdup("executable statements are not allowed at file scope; put this inside a function"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
    }

    switch (node->kind) {
    case NODE_VAR_DECL: {
        /* E3038: void cannot be used as variable type */
        if (node->data.var_decl.type_name && strcmp(node->data.var_decl.type_name, "void") == 0) {
            diag_error_msg(tc->diag, "E3038",
                strdup("'void' cannot be used as a variable type"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E3038: void in array/map types.
         * "void" is legal as a typed-func return type (encoded as
         * "func(...)->void"), so skip the strstr check for those. */
        if (node->data.var_decl.type_name) {
            const char *tn = node->data.var_decl.type_name;
            bool is_typed_func = strncmp(tn, "func(", 5) == 0 ||
                                 strncmp(tn, "[func(", 6) == 0;
            if (!is_typed_func && strstr(tn, "void") != NULL && strcmp(tn, "void") != 0) {
                diag_error_msg(tc->diag, "E3038",
                    strdup("'void' cannot be used as an element type in arrays or maps"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* E3034: 'any' type is reserved */
        if (node->data.var_decl.type_name && strcmp(node->data.var_decl.type_name, "any") == 0) {
            diag_error_code(tc->diag, "E3034", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E2038: reserved type name as variable name */
        if (node->data.var_decl.name[0] != '_' &&
            is_reserved_type_name(node->data.var_decl.name)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "'%s' is a reserved type name and cannot be used as a variable name",
                node->data.var_decl.name);
            diag_error_msg(tc->diag, "E2038", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E3045: or_return on non-error-returning function */
        if (strncmp(node->data.var_decl.name, "_ez_or", 6) == 0 &&
            node->data.var_decl.value && node->data.var_decl.value->kind == NODE_CALL_EXPR) {
            AstNode *call_fn = node->data.var_decl.value->data.call.function;
            const char *call_name = NULL;
            if (call_fn->kind == NODE_LABEL) call_name = call_fn->data.label.value;
            else if (call_fn->kind == NODE_MEMBER_EXPR && call_fn->data.member.object->kind == NODE_LABEL) {
                /* module.func() or Type.func(); construct prefixed name */
                static char prefixed[EZ_MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s",
                    call_fn->data.member.object->data.label.value, call_fn->data.member.member);
                call_name = prefixed;
            }
            if (call_name) {
                FuncSig *sig = find_func(tc, call_name);
                if (sig && (sig->return_count < 2 ||
                    sig->return_types[sig->return_count - 1]->kind != TK_ERROR)) {
                    char display[256];
                    if (call_fn->kind == NODE_MEMBER_EXPR)
                        snprintf(display, sizeof(display), "%s.%s",
                            call_fn->data.member.object->data.label.value,
                            call_fn->data.member.member);
                    else
                        snprintf(display, sizeof(display), "%s", call_name);
                    diag_error_codef(tc->diag, "E3045", NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0, display);
                }
            }
        }
        /* E3059: maps cannot be declared const */
        if (!node->data.var_decl.mutable && node->data.var_decl.type_name &&
            strncmp(node->data.var_decl.type_name, "map[", 4) == 0) {
            diag_error_code(tc->diag, "E3059", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* const must have a value */
        if (!node->data.var_decl.mutable && !node->data.var_decl.value) {
            diag_error_codef(tc->diag, "E2011", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name);
        }
        /* Check for type keyword used as value: mut x = int */
        if (node->data.var_decl.value && node->data.var_decl.value->kind == NODE_LABEL) {
            const char *vname = node->data.var_decl.value->data.label.value;
            if (is_reserved_type_name(vname)) {
                diag_error_codef(tc->diag, "E3011", NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                    node->data.var_decl.value->token.column, 0, vname, vname);
            }
        }

        /* E3050/E3051: array/map literals require explicit type annotations */
        if (!node->data.var_decl.type_name && node->data.var_decl.value &&
            strncmp(node->data.var_decl.name, "_ez_tmp", 7) != 0 &&
            strncmp(node->data.var_decl.name, "_ez_or", 6) != 0) {
            if (node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                diag_error_code(tc->diag, "E3050",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            } else if (node->data.var_decl.value->kind == NODE_MAP_VALUE) {
                diag_error_code(tc->diag, "E3051",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }

        /* E3054: mutable array with fixed size */
        /* E3055: const array without fixed size */
        if (node->data.var_decl.type_name && node->data.var_decl.type_name[0] == '[') {
            const char *tn = node->data.var_decl.type_name;
            /* Top-level comma only; commas inside (), [], or func sigs are
             * part of the element type, not the [T,N] size separator. */
            const char *size_comma = NULL;
            int depth = 0;
            for (const char *c = tn; *c; c++) {
                if (*c == '(' || *c == '[') depth++;
                else if (*c == ')' || *c == ']') depth--;
                else if (*c == ',' && depth == 1) { size_comma = c; break; }
            }
            bool has_size = size_comma != NULL;
            if (node->data.var_decl.mutable && has_size) {
                diag_error_codef(tc->diag, "E3054", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name,
                    (int)(size_comma - tn), tn);
            } else if (!node->data.var_decl.mutable && !has_size) {
                diag_error_codef(tc->diag, "E3055", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name,
                    (int)(strlen(tn) - 2), tn + 1,
                    node->data.var_decl.value && node->data.var_decl.value->kind == NODE_ARRAY_VALUE
                        ? node->data.var_decl.value->data.array_value.count : 0);
            }
        }

        /* E2002: private not allowed inside functions */
        if (node->data.var_decl.is_private && tc->func_depth > 0) {
            diag_error_msg(tc->diag, "E2002",
                strdup("'private' cannot be used inside a function; it only applies to top-level declarations"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        EzType *declared = node->data.var_decl.type_name
            ? tc_type_from_name(tc, node->data.var_decl.type_name)
            : &TYPE_UNKNOWN;
        /* E4016: explicitly annotated type name that doesn't exist */
        if (node->data.var_decl.type_name && declared->kind == TK_UNKNOWN &&
            node->data.var_decl.type_name[0] >= 'A' && node->data.var_decl.type_name[0] <= 'Z') {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "undefined type '%s'; check the spelling or import the module that defines it",
                node->data.var_decl.type_name);
            diag_error_msg(tc->diag, "E4016", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        tc_mark_type_module_used(tc, node->data.var_decl.type_name);

        /* E3057: reject composite types as map keys before downstream checks
         * produce misleading cascades (e.g. struct-literal-in-index-position
         * tripping "no field 'y'"). Enums are allowed; they're int-backed
         * and hash fine. */
        if (declared->kind == TK_MAP && declared->key_type) {
            const char *kt = declared->key_type;
            EzType *key_resolved = type_from_name(kt);
            const char *bad = NULL;
            if (key_resolved->kind == TK_STRUCT && !is_enum_name(tc, kt))
                bad = "struct";
            else if (key_resolved->kind == TK_ARRAY) bad = "array";
            else if (key_resolved->kind == TK_MAP) bad = "map";
            else if (key_resolved->kind == TK_POINTER) bad = "pointer";
            if (bad) {
                diag_error_codef(tc->diag, "E3057", NODE_FILE(tc, node), node->token.line, node->token.column, 0, kt);
            }
        }

        if (node->data.var_decl.value) {
            EzType *value_type = resolve_expr(tc, node->data.var_decl.value);
            /* : when a func-pointer call returns TK_UNKNOWN but
             * the assignment target has a concrete declared type,
             * push the declared type onto the call node's typetable
             * entry so codegen can derive the correct function-pointer
             * return cast instead of defaulting to int64_t. */
            if (value_type->kind == TK_UNKNOWN && declared->kind != TK_UNKNOWN &&
                declared->kind != TK_VOID &&
                node->data.var_decl.value->kind == NODE_CALL_EXPR) {
                typetable_set(tc->type_table, node->data.var_decl.value, declared);
                value_type = declared;
            }
            /* E3038: cannot assign void function result */
            if (value_type->kind == TK_VOID) {
                diag_error_msg(tc->diag, "E3038",
                    strdup("cannot assign the result of a void function to a variable"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Check for multi-return to single variable
             * (skip if this is part of a multi-var expansion; the value will be a .v0 access) */
            if (node->data.var_decl.value->kind == NODE_CALL_EXPR &&
                node->data.var_decl.value->data.call.function->kind == NODE_LABEL &&
                strncmp(node->data.var_decl.name, "_ez_tmp", 7) != 0 &&
                strncmp(node->data.var_decl.name, "_ez_or", 6) != 0) {
                const char *call_name = node->data.var_decl.value->data.call.function->data.label.value;
                FuncSig *sig = find_func(tc, call_name);
                if (sig && sig->return_count > 1) {
                    diag_error_codef(tc->diag, "E3040", NODE_FILE(tc, node), node->token.line, node->token.column, 0, call_name, sig->return_count, call_name);
                }
            }
            /* Reject nil on non-nullable types */
            if (value_type->kind == TK_NIL && declared->kind != TK_UNKNOWN &&
                declared->kind != TK_ERROR && declared->kind != TK_POINTER &&
                declared->kind != TK_NIL) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "cannot assign nil to '%s'; only Error and pointer types are nullable",
                    type_name(declared));
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Reject bare 'mut x = nil' with no type context */
            if (value_type->kind == TK_NIL && declared->kind == TK_UNKNOWN) {
                diag_error_msg(tc->diag, "E3001",
                    strdup("cannot infer type from nil; add a type annotation (e.g., mut x Error = nil)"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3066: typed-func variable assigned a function reference with a
             * different signature. Both sides are TK_FUNCTION; the canonical
             * encoded names (e.g. "func(int)->int") must match exactly. */
            if (declared->kind == TK_FUNCTION && value_type->kind == TK_FUNCTION &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "cannot assign %s to variable of type %s",
                    value_type->name, declared->name);
                diag_error_msg(tc->diag, "E3066", strdup(msg),
                    NODE_FILE(tc, node->data.var_decl.value),
                    node->data.var_decl.value->token.line,
                    node->data.var_decl.value->token.column, 0);
            }
            /* E3001 (): \`mut f func = expr\` requires expr to be a
             * function reference. The declared type "func" round-trips as
             * TK_UNKNOWN via type_from_name, so the generic mismatch check
             * below can't see it; it would fall through to "declared is
             * unknown, infer from value" and silently adopt whatever type
             * the call site returned. Catch it explicitly here. */
            if (node->data.var_decl.type_name &&
                strcmp(node->data.var_decl.type_name, "func") == 0 &&
                node->data.var_decl.value) {
                AstNode *v = node->data.var_decl.value;
                bool value_is_func =
                    v->kind == NODE_FUNC_REF ||
                    value_type->kind == TK_NIL ||
                    (value_type->name && strcmp(value_type->name, "func") == 0);
                if (!value_is_func) {
                    char msg[EZ_MSG_BUF_SIZE];
                    /* If the initializer is a direct call, point the user
                     * at the reference form of the same name; that's
                     * overwhelmingly what they meant. */
                    if (v->kind == NODE_CALL_EXPR &&
                        v->data.call.function &&
                        v->data.call.function->kind == NODE_LABEL) {
                        const char *called = v->data.call.function->data.label.value;
                        snprintf(msg, sizeof(msg),
                            "cannot assign %s to 'func'; to store a reference to '%s', use '()%s' (not '%s()')",
                            type_name(value_type), called, called, called);
                    } else {
                        snprintf(msg, sizeof(msg),
                            "cannot assign %s to 'func'; func variables hold function references (e.g. '()name')",
                            type_name(value_type));
                    }
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* If no declared type, infer from value */
            if (declared->kind == TK_UNKNOWN) {
                declared = value_type;
            } else if (value_type->kind != TK_UNKNOWN &&
                       value_type->kind != TK_VOID &&
                       declared->kind != value_type->kind &&
                       value_type->kind != TK_NIL &&
                       /* Skip mismatch between int/uint (handled by E3019) */
                       !(is_int_kind(declared->kind) && is_int_kind(value_type->kind)) &&
                       /* Allow enum → int (enums are int-backed) but not
                        * the reverse; int literals / variables can't be
                        * assigned to enum variables (). */
                       !(is_int_kind(declared->kind) && value_type->kind == TK_ENUM) &&
                       /* Skip mismatch on multi-var expansion (.v0/.v1 access) */
                       !(node->data.var_decl.value &&
                         node->data.var_decl.value->kind == NODE_MEMBER_EXPR &&
                         node->data.var_decl.value->data.member.member[0] == 'v') &&
                       /* Skip mismatch when assigning ref var to ^T pointer */
                       !(declared->kind == TK_POINTER && node->data.var_decl.value &&
                         node->data.var_decl.value->kind == NODE_LABEL &&
                         scope_lookup(tc->current_scope, node->data.var_decl.value->data.label.value) &&
                         scope_lookup(tc->current_scope, node->data.var_decl.value->data.label.value)->is_ref) &&
                       /* Skip mismatch when assigning pointer (addr) to uint */
                       !(is_int_kind(declared->kind) && value_type->kind == TK_POINTER) &&
                       /* Skip mismatch when assigning pointer (addr) to ^T */
                       !(declared->kind == TK_POINTER && value_type->kind == TK_POINTER) &&
                       /* : implicit int→float coercion when target is float */
                       !(declared->kind == TK_FLOAT && is_int_kind(value_type->kind))) {
                /* Type mismatch */
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign %s to %s",
                    type_name(value_type), type_name(declared));
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Struct-to-struct name mismatch (both TK_STRUCT but different names).
             * : skip when one name is a module-prefixed alias of the
             * other (e.g. "Point" vs "shapes_Point" via import and use). */
            bool struct_alias_match = false;
            if (declared->kind == TK_STRUCT && value_type->kind == TK_STRUCT &&
                declared->name && value_type->name) {
                const char *d = declared->name;
                const char *v = value_type->name;
                const char *d_us = strrchr(d, '_');
                const char *v_us = strrchr(v, '_');
                if (d_us && strcmp(d_us + 1, v) == 0) struct_alias_match = true;
                if (v_us && strcmp(v_us + 1, d) == 0) struct_alias_match = true;
            }
            if (declared->kind == TK_STRUCT && value_type->kind == TK_STRUCT &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0 &&
                !struct_alias_match) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign '%s' to '%s'",
                    value_type->name, declared->name);
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Enum-to-enum name mismatch (both TK_ENUM but different enum types) */
            if (declared->kind == TK_ENUM && value_type->kind == TK_ENUM &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign enum '%s' to enum '%s'",
                    value_type->name, declared->name);
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3046: literal that exceeds the destination type's range.
             *   overflow_u64 = true  : exceeds UINT64_MAX, never fits a non-bigint
             *   overflow     = true  : exceeds INT64_MAX but fits in UINT64_MAX,
             *                         OK for uint/u64/bigint, error otherwise */
            if (node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_INT_VALUE &&
                node->data.var_decl.value->data.int_value.overflow) {
                const char *tn = node->data.var_decl.type_name;
                bool is_bigint = tn && (strcmp(tn, "i128") == 0 || strcmp(tn, "u128") == 0 ||
                                        strcmp(tn, "i256") == 0 || strcmp(tn, "u256") == 0);
                bool is_u64_like = tn && (strcmp(tn, "u64") == 0 || strcmp(tn, "uint") == 0);
                bool exceeds_u64 = node->data.var_decl.value->data.int_value.overflow_u64;
                if (exceeds_u64 && !is_bigint) {
                    diag_error_msg(tc->diag, "E3046",
                        strdup("integer literal overflows 64-bit integer; max value is 18446744073709551615"),
                        NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0);
                } else if (!exceeds_u64 && !is_bigint && !is_u64_like) {
                    diag_error_msg(tc->diag, "E3046",
                        strdup("integer literal overflows 64-bit integer; max value is 9223372036854775807"),
                        NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0);
                }
            }
            /* E3036: Check literal value fits in sized integer type (skip overflowed literals) */
            if (node->data.var_decl.type_name && node->data.var_decl.value) {
                bool val_overflowed = (node->data.var_decl.value->kind == NODE_INT_VALUE &&
                    node->data.var_decl.value->data.int_value.overflow);
                int64_t lit_val;
                if (!val_overflowed && try_get_literal_int(node->data.var_decl.value, &lit_val)) {
                    check_integer_range(tc->diag, NODE_FILE(tc, node),
                        node->token.line, node->token.column,
                        node->data.var_decl.type_name, lit_val);
                }
                /* E3001 (): assigning an array literal `{}` to a map
                 * variable falls through the normal type check because the
                 * literal has no elements to derive a concrete element type
                 * from, and codegen then emits ez_array_new which the C
                 * compiler rejects. Point the user at the empty-map form
                 * `{:}` before the rest of the var_decl check runs. */
                const char *tn = node->data.var_decl.type_name;
                if (strncmp(tn, "map[", 4) == 0 &&
                    node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                    char msg[EZ_MSG_BUF_SIZE];
                    if (node->data.var_decl.value->data.array_value.count == 0) {
                        snprintf(msg, sizeof(msg),
                            "cannot assign array literal '{}' to '%s'; use '{:}' for an empty map",
                            tn);
                    } else {
                        snprintf(msg, sizeof(msg),
                            "cannot assign array literal to '%s'; map literals use '{key: value, ...}' syntax",
                            tn);
                    }
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node->data.var_decl.value),
                        node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0);
                }
                /* E3026/E3036: Check array literal elements fit in sized element type */
                if (tn[0] == '[' && node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                    /* Extract element type name from "[byte]", "[i8]", "[u8, 3]", etc. */
                    char elem_type[EZ_TYPE_NAME_MAX] = {0};
                    const char *start = tn + 1;
                    const char *end = strchr(start, ']');
                    const char *comma = strchr(start, ',');
                    if (end) {
                        int elen = (int)((comma && comma < end ? comma : end) - start);
                        if (elen > 0 && elen < (int)sizeof(elem_type)) {
                            /* trim whitespace */
                            while (elen > 0 && start[elen-1] == ' ') elen--;
                            memcpy(elem_type, start, (size_t)elen);
                            elem_type[elen] = '\0';
                        }
                    }
                    if (elem_type[0]) {
                        AstNode *arr = node->data.var_decl.value;
                        bool elem_is_u64_like = (strcmp(elem_type, "uint") == 0 || strcmp(elem_type, "u64") == 0);
                        for (int ei = 0; ei < arr->data.array_value.count; ei++) {
                            AstNode *el = arr->data.array_value.elements[ei];
                            bool el_overflowed = (el->kind == NODE_INT_VALUE && el->data.int_value.overflow);
                            bool el_overflowed_u64 = (el->kind == NODE_INT_VALUE && el->data.int_value.overflow_u64);
                            /* Element exceeds UINT64_MAX entirely; always an error. */
                            if (el_overflowed_u64) {
                                diag_error_msg(tc->diag, "E3046",
                                    strdup("integer literal overflows 64-bit integer; max value is 18446744073709551615"),
                                    NODE_FILE(tc, el), el->token.line, el->token.column, 0);
                                continue;
                            }
                            /* Element exceeds INT64_MAX but fits UINT64_MAX —
                             * fine for u64/uint elements, error for narrower
                             * signed/unsigned and for int. */
                            if (el_overflowed) {
                                if (!elem_is_u64_like) {
                                    diag_error_msg(tc->diag, "E3046",
                                        strdup("integer literal overflows 64-bit integer; max value is 9223372036854775807"),
                                        NODE_FILE(tc, el), el->token.line, el->token.column, 0);
                                }
                                continue;
                            }
                            int64_t ev;
                            if (try_get_literal_int(el, &ev)) {
                                check_integer_range(tc->diag, NODE_FILE(tc, el),
                                    el->token.line, el->token.column,
                                    elem_type, ev);
                            }
                        }
                    }
                    /* E3053: element type mismatch in array initializer */
                    if (elem_type[0]) {
                        EzType *expected_et = type_from_name(elem_type);
                        AstNode *arr = node->data.var_decl.value;
                        for (int ei = 0; ei < arr->data.array_value.count; ei++) {
                            EzType *actual_et = resolve_expr(tc, arr->data.array_value.elements[ei]);
                            if (actual_et && actual_et->kind != TK_UNKNOWN &&
                                expected_et && expected_et->kind != TK_UNKNOWN &&
                                actual_et->kind != expected_et->kind) {
                                /* Allow compatible integer kinds + enum↔int */
                                bool compatible = false;
                                if ((expected_et->kind == TK_INT || expected_et->kind == TK_UINT ||
                                     expected_et->kind == TK_BYTE) &&
                                    (actual_et->kind == TK_INT || actual_et->kind == TK_UINT ||
                                     actual_et->kind == TK_BYTE)) {
                                    compatible = true;
                                }
                                if (is_int_kind(expected_et->kind) && actual_et->kind == TK_ENUM)
                                    compatible = true;
                                if (expected_et->kind == TK_ENUM && is_int_kind(actual_et->kind))
                                    compatible = true;
                                if (expected_et->kind == TK_ENUM && actual_et->kind == TK_ENUM)
                                    compatible = true;
                                /* : int→float coercion in array initializer */
                                if (expected_et->kind == TK_FLOAT && is_int_kind(actual_et->kind))
                                    compatible = true;
                                if (!compatible) {
                                    diag_error_codef(tc->diag, "E3053", NODE_FILE(tc, arr->data.array_value.elements[ei]),
                                        arr->data.array_value.elements[ei]->token.line,
                                        arr->data.array_value.elements[ei]->token.column, 0, expected_et->name, actual_et->name);
                                }
                            }
                        }
                    }
                    /* W3003/E3052: fixed-size array initialization count checks */
                    if (comma && comma < end) {
                        int fixed_size = atoi(comma + 1);
                        AstNode *arr = node->data.var_decl.value;
                        if (fixed_size > 0 && arr->data.array_value.count > fixed_size) {
                            diag_error_codef(tc->diag, "E3052", NODE_FILE(tc, node), node->token.line, node->token.column, 0, fixed_size, arr->data.array_value.count);
                        }
                        if (fixed_size > 0 && arr->data.array_value.count < fixed_size) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "fixed-size array [%s, %d] initialized with only %d of %d elements; remaining will be zero-valued",
                                elem_type[0] ? elem_type : "?", fixed_size,
                                arr->data.array_value.count, fixed_size);
                            diag_warning_msg(tc->diag, "W3003", strdup(msg),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                }
                /* : map literal key/value type mismatch. Parallel
                 * to the array E3053 check above; walks NODE_MAP_VALUE
                 * pairs and rejects entries whose key or value type
                 * doesn't match the declared map's K/V. The void case is
                 * already caught in NODE_MAP_VALUE's
                 * reject_void_in_context path (); this block
                 * covers the non-void-but-wrong-type leak. */
                if (strncmp(tn, "map[", 4) == 0 &&
                    node->data.var_decl.value->kind == NODE_MAP_VALUE) {
                    const char *mstart = tn + 4;
                    const char *mcolon = strchr(mstart, ':');
                    const char *mend = strrchr(tn, ']');
                    if (mcolon && mend && mend > mcolon) {
                        char key_tn[EZ_TYPE_NAME_MAX] = {0};
                        char val_tn[EZ_TYPE_NAME_MAX] = {0};
                        size_t klen = (size_t)(mcolon - mstart);
                        size_t vlen = (size_t)(mend - mcolon - 1);
                        if (klen > 0 && klen < sizeof(key_tn) &&
                            vlen > 0 && vlen < sizeof(val_tn)) {
                            memcpy(key_tn, mstart, klen);
                            key_tn[klen] = '\0';
                            memcpy(val_tn, mcolon + 1, vlen);
                            val_tn[vlen] = '\0';
                            EzType *expected_k = tc_type_from_name(tc, key_tn);
                            EzType *expected_v = tc_type_from_name(tc, val_tn);
                            AstNode *mv = node->data.var_decl.value;
                            for (int mi = 0; mi < mv->data.map_value.count; mi++) {
                                AstNode *kn = mv->data.map_value.keys[mi];
                                AstNode *vn = mv->data.map_value.values[mi];
                                EzType *kt = resolve_expr(tc, kn);
                                EzType *vt = resolve_expr(tc, vn);
                                if (kt && kt->kind != TK_UNKNOWN && kt->kind != TK_VOID &&
                                    expected_k && expected_k->kind != TK_UNKNOWN &&
                                    kt->kind != expected_k->kind &&
                                    !(is_int_kind(expected_k->kind) && is_int_kind(kt->kind)) &&
                                    !(is_int_kind(expected_k->kind) && kt->kind == TK_ENUM) &&
                                    !(expected_k->kind == TK_ENUM && is_int_kind(kt->kind)) &&
                                    !(expected_k->kind == TK_ENUM && kt->kind == TK_ENUM) &&
                                    !(expected_k->kind == TK_FLOAT && is_int_kind(kt->kind))) {
                                    char msg[EZ_MSG_BUF_SIZE];
                                    snprintf(msg, sizeof(msg),
                                        "type mismatch in map literal key; expected '%s', got '%s'",
                                        expected_k->name, kt->name);
                                    diag_error_msg(tc->diag, "E3053", strdup(msg),
                                        NODE_FILE(tc, kn), kn->token.line, kn->token.column, 0);
                                }
                                if (vt && vt->kind != TK_UNKNOWN && vt->kind != TK_VOID &&
                                    expected_v && expected_v->kind != TK_UNKNOWN &&
                                    vt->kind != expected_v->kind &&
                                    !(is_int_kind(expected_v->kind) && is_int_kind(vt->kind)) &&
                                    !(is_int_kind(expected_v->kind) && vt->kind == TK_ENUM) &&
                                    !(expected_v->kind == TK_ENUM && is_int_kind(vt->kind)) &&
                                    !(expected_v->kind == TK_ENUM && vt->kind == TK_ENUM) &&
                                    !(expected_v->kind == TK_POINTER && vt->kind == TK_POINTER) &&
                                    !(expected_v->kind == TK_FLOAT && is_int_kind(vt->kind))) {
                                    char msg[EZ_MSG_BUF_SIZE];
                                    snprintf(msg, sizeof(msg),
                                        "type mismatch in map literal value; expected '%s', got '%s'",
                                        expected_v->name, vt->name);
                                    diag_error_msg(tc->diag, "E3053", strdup(msg),
                                        NODE_FILE(tc, vn), vn->token.line, vn->token.column, 0);
                                }
                            }
                        }
                    }
                }
            }
            /* E3019: signed-to-unsigned assignment from variable */
            if (node->data.var_decl.type_name &&
                is_unsigned_type(node->data.var_decl.type_name) &&
                node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_LABEL) {
                const char *src_name = node->data.var_decl.value->data.label.value;
                Symbol *src_sym = scope_lookup(tc->current_scope, src_name);
                if (src_sym && src_sym->declared_type &&
                    is_signed_int_type(src_sym->declared_type)) {
                    diag_error_codef(tc->diag, "E3019", NODE_FILE(tc, node), node->token.line, node->token.column, 0, src_sym->declared_type, node->data.var_decl.type_name);
                }
            }
        }

        /* E3062 (): handle types (channels, mutexes, threads)
         * cannot be declared const; every meaningful operation on
         * them mutates internal state, so const is a semantic lie.
         * Same class as the E3059 map check above. */
        if (!node->data.var_decl.mutable && declared->kind == TK_STRUCT && declared->name) {
            const char *dn = declared->name;
            const char *handle_label = NULL;
            if (strcmp(dn, "Channel") == 0) handle_label = "channel";
            else if (strcmp(dn, "Mutex") == 0) handle_label = "mutex";
            else if (strcmp(dn, "Thread") == 0) handle_label = "thread handle";
            if (handle_label) {
                diag_error_codef(tc->diag, "E3062", NODE_FILE(tc, node), node->token.line, node->token.column, 0, handle_label, handle_label);
            }
        }

        /* W1005: typed blank identifier; _ with explicit type annotation */
        if (strcmp(node->data.var_decl.name, "_") == 0 && node->data.var_decl.type_name) {
            diag_warning_code(tc->diag, "W1005", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        if (strcmp(node->data.var_decl.name, "_") != 0) {
            /* Check for reserved prefix */
            check_reserved_name(tc, node->data.var_decl.name,
                NODE_FILE(tc, node), node->token.line, node->token.column);
            /* Check for redeclaration in same scope */
            Symbol *existing = scope_lookup_local(tc->current_scope,
                node->data.var_decl.name);
            if (existing) {
                diag_error_codef(tc->diag, "E4003", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name, existing->def_line);
            }
            /* W2002/W2007: check if variable shadows outer scope */
            if (!existing && tc->current_scope->parent) {
                Symbol *outer_sym = scope_lookup(tc->current_scope->parent,
                    node->data.var_decl.name);
                if (outer_sym && outer_sym->def_line > 0) {
                    /* Check if it's a global (file-scope) variable */
                    Scope *outer_scope = tc->current_scope->parent;
                    while (outer_scope->parent) {
                        Symbol *s = scope_lookup_local(outer_scope, node->data.var_decl.name);
                        if (s) break;
                        outer_scope = outer_scope->parent;
                    }
                    bool is_global = (outer_scope->parent == NULL);
                    char msg[EZ_MSG_BUF_SIZE];
                    if (is_global) {
                        snprintf(msg, sizeof(msg),
                            "variable '%s' shadows a global constant or variable declared on line %d",
                            node->data.var_decl.name, outer_sym->def_line);
                        diag_warning_msg(tc->diag, "W2007", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    } else {
                        snprintf(msg, sizeof(msg),
                            "variable '%s' shadows a variable declared on line %d",
                            node->data.var_decl.name, outer_sym->def_line);
                        diag_warning_msg(tc->diag, "W2002", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
            }
            /* E4012: shadows a type */
            if (is_struct_name(tc, node->data.var_decl.name) ||
                is_enum_name(tc, node->data.var_decl.name)) {
                diag_error_codef(tc->diag, "E4012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name);
            }
            /* E4013: shadows a function */
            if (find_func(tc, node->data.var_decl.name)) {
                diag_error_codef(tc->diag, "E4013", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name);
            }
            /* E4014: shadows an imported module */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], node->data.var_decl.name) == 0) {
                    diag_error_codef(tc->diag, "E4014", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.var_decl.name);
                    break;
                }
            }
            if (declared->kind == TK_UNKNOWN &&
                !(declared->name && strcmp(declared->name, "func") == 0) &&
                !(node->data.var_decl.value && node->data.var_decl.value->kind == NODE_CALL_EXPR)) {
                /* Don't register variables with unresolved types; an error
                   (E3050, E3051, etc.) has already been emitted upstream.
                   Skipping scope_define prevents confusing cascading errors.
                   Exceptions: func refs and func ref calls (return type unknown). */
                break;
            }
            scope_define(tc->current_scope, node->data.var_decl.name,
                declared, node->data.var_decl.mutable);
            /* Store definition location and declared type for unused/signedness warnings */
            Symbol *def_sym = scope_lookup_local(tc->current_scope,
                node->data.var_decl.name);
            if (def_sym) {
                def_sym->declared_type = node->data.var_decl.type_name;
                def_sym->def_line = node->token.line;
                def_sym->def_column = node->token.column;
            }

            /* Mark as transparent ref if assigned from ref() */
            if (node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_CALL_EXPR) {
                AstNode *fn = node->data.var_decl.value->data.call.function;
                if (fn->kind == NODE_LABEL && strcmp(fn->data.label.value, "ref") == 0) {
                    Symbol *sym = scope_lookup_local(tc->current_scope,
                        node->data.var_decl.name);
                    if (sym) sym->is_ref = true;
                    /* E3079: a mutable reference to a const source is a
                     * contradiction — the source promised immutability and
                     * the reference would let writes through. Allow:
                     *   const r = ref(const_var)   (read-only view)
                     *   const r = ref(mut_var)     (read-only view of mutable)
                     *   mut r   = ref(mut_var)     (full mutable alias)
                     * Reject:
                     *   mut r   = ref(const_var)
                     */
                    if (node->data.var_decl.mutable &&
                        node->data.var_decl.value->data.call.arg_count == 1) {
                        AstNode *src = node->data.var_decl.value->data.call.args[0];
                        if (src->kind == NODE_LABEL) {
                            Symbol *src_sym = scope_lookup(tc->current_scope,
                                src->data.label.value);
                            if (src_sym && !src_sym->mutable &&
                                !find_func(tc, src->data.label.value)) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "cannot take a mutable reference to const variable '%s'; declare '%s' as const, or copy() the value to get an independent mutable instance",
                                    src->data.label.value,
                                    node->data.var_decl.name);
                                diag_error_msg(tc->diag, "E3079", strdup(msg),
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            }
                        }
                    }
                }
                /* Store multi-return types for temp variables from calls.
                 * For generic functions, substitute the wildcard binding
                 * so destructured slots get concrete types instead of
                 * TK_UNKNOWN; without this, `mut a, b = pair(42)` where
                 * pair returns (?, ?) leaves the temp's slot types
                 * unknown, the unannotated LHS vars never declare, and
                 * subsequent uses error as undefined. */
                if (fn->kind == NODE_LABEL) {
                    FuncSig *sig = find_func(tc, fn->data.label.value);
                    if (sig && sig->return_count > 1) {
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym) {
                            EzType **slots = sig->return_types;
                            int slot_count = sig->return_count;
                            if (sig->is_generic && sig->decl &&
                                sig->decl->kind == NODE_FUNC_DECL) {
                                /* Bind '?' from the call's args, then
                                 * substitute into each return slot. */
                                AstNode *call = node->data.var_decl.value;
                                AstNode *decl = sig->decl;
                                char *binding = NULL;
                                int cc = call->data.call.arg_count <
                                         decl->data.func_decl.param_count
                                    ? call->data.call.arg_count
                                    : decl->data.func_decl.param_count;
                                for (int ai = 0; ai < cc && !binding; ai++) {
                                    const char *ptn =
                                        decl->data.func_decl.params[ai].type_name;
                                    if (!ptn || !type_name_has_wildcard(ptn)) continue;
                                    EzType *at = resolve_expr(tc, call->data.call.args[ai]);
                                    binding = bind_wildcard(ptn, at);
                                }
                                if (binding) {
                                    int rc = decl->data.func_decl.return_type_count;
                                    EzType **subbed = malloc(sizeof(EzType *) * (size_t)rc);
                                    for (int ri = 0; ri < rc; ri++) {
                                        char *sub = substitute_wildcard(
                                            decl->data.func_decl.return_types[ri], binding);
                                        subbed[ri] = sub ? type_from_name(sub) : &TYPE_UNKNOWN;
                                    }
                                    free(binding);
                                    slots = subbed;
                                    slot_count = rc;
                                }
                            }
                            sym->ret_types = slots;
                            sym->ret_count = slot_count;
                        }
                    }
                }
                /* Stdlib module calls (mod.func); synthesize (T, Error) return
                 * types for fallible functions so multi-var destructuring works. */
                if (fn->kind == NODE_MEMBER_EXPR &&
                    fn->data.member.object->kind == NODE_LABEL) {
                    const char *mod = fn->data.member.object->data.label.value;
                    const char *mfn = fn->data.member.member;
                    if (tc_is_fallible_stdlib(mod, mfn)) {
                        EzType *primary = tc_get_fallible_stdlib_type(mod, mfn);
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym && primary) {
                            EzType **rt = malloc(sizeof(EzType *) * 2);
                            rt[0] = primary;
                            rt[1] = type_from_name("Error");
                            sym->ret_types = rt;
                            sym->ret_count = 2;
                        }
                    }
                }
            }
            /* Track referenced function for func-typed vars so calls through
             * them can be arity/type-checked at compile time. */
            if (node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_FUNC_REF) {
                AstNode *fref = node->data.var_decl.value->data.func_ref.function;
                const char *rname = NULL;
                if (fref->kind == NODE_LABEL) {
                    rname = fref->data.label.value;
                } else if (fref->kind == NODE_MEMBER_EXPR &&
                           fref->data.member.object->kind == NODE_LABEL) {
                    char *prefixed = malloc(
                        strlen(fref->data.member.object->data.label.value) +
                        strlen(fref->data.member.member) + 2);
                    sprintf(prefixed, "%s_%s",
                        fref->data.member.object->data.label.value,
                        fref->data.member.member);
                    rname = prefixed;
                }
                if (rname) {
                    Symbol *sym = scope_lookup_local(tc->current_scope,
                        node->data.var_decl.name);
                    if (sym) sym->func_ref_name = rname;
                }
            }
            /* Per-element tracking for [func] arrays initialised with a
             * literal of func refs (). Preserves each element's
             * originating function name so constant-index calls can
             * recover the real return type (e.g. struct returns) that
             * would otherwise be erased by the void* storage. */
            if (node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_ARRAY_VALUE &&
                node->data.var_decl.type_name &&
                (strcmp(node->data.var_decl.type_name, "[func]") == 0 ||
                 strncmp(node->data.var_decl.type_name, "[func(", 6) == 0)) {
                AstNode *lit = node->data.var_decl.value;
                int n = lit->data.array_value.count;
                Symbol *sym = scope_lookup_local(tc->current_scope,
                    node->data.var_decl.name);
                if (sym && n > 0) {
                    sym->func_array_refs = calloc((size_t)n, sizeof(const char *));
                    sym->func_array_ref_count = n;
                    for (int ei = 0; ei < n; ei++) {
                        AstNode *el = lit->data.array_value.elements[ei];
                        if (!el || el->kind != NODE_FUNC_REF) continue;
                        AstNode *fref = el->data.func_ref.function;
                        if (fref->kind == NODE_LABEL) {
                            sym->func_array_refs[ei] = fref->data.label.value;
                        } else if (fref->kind == NODE_MEMBER_EXPR &&
                                   fref->data.member.object->kind == NODE_LABEL) {
                            size_t plen =
                                strlen(fref->data.member.object->data.label.value) +
                                strlen(fref->data.member.member) + 2;
                            char *pref = malloc(plen);
                            snprintf(pref, plen, "%s_%s",
                                fref->data.member.object->data.label.value,
                                fref->data.member.member);
                            sym->func_array_refs[ei] = pref;
                        }
                    }
                }
            }
        }
        break;
    }

    case NODE_ASSIGN_STMT: {
        EzType *target_t = resolve_expr(tc, node->data.assign.target);
        EzType *value_t = resolve_expr(tc, node->data.assign.value);

        /* E3078: compound arithmetic assigns on pointer variables
         * (p += 1, p -= 2, etc.) are pointer arithmetic and not
         * supported. Plain `=` (and the existing `p = nil` /
         * `p = addr(...)` paths) stay valid. */
        const char *aop = node->data.assign.op;
        if (aop && (strcmp(aop, "+=") == 0 || strcmp(aop, "-=") == 0 ||
                    strcmp(aop, "*=") == 0 || strcmp(aop, "/=") == 0 ||
                    strcmp(aop, "%=") == 0) &&
            target_t && target_t->kind == TK_POINTER) {
            diag_error_code(tc->diag, "E3078",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        /* E5025: lvalue validation; reject assignment to non-lvalue targets */
        AstNode *target = node->data.assign.target;
        if (target->kind != NODE_LABEL &&
            target->kind != NODE_MEMBER_EXPR &&
            target->kind != NODE_INDEX_EXPR &&
            target->kind != NODE_PREFIX_EXPR &&
            target->kind != NODE_POSTFIX_EXPR) {
            diag_error_msg(tc->diag, "E5025",
                strdup("cannot assign to this expression; left side of '=' must be a variable, field, or index"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        /* Check for assignment to const variable (direct, index, or field) */
        const char *const_name = NULL;
        if (target->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.label.value);
            if (sym && !sym->mutable) const_name = target->data.label.value;
        } else if (target->kind == NODE_INDEX_EXPR && target->data.index_expr.left->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.index_expr.left->data.label.value);
            if (sym && !sym->mutable) const_name = target->data.index_expr.left->data.label.value;
        } else if (target->kind == NODE_MEMBER_EXPR && target->data.member.object->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.member.object->data.label.value);
            if (sym && !sym->mutable) const_name = target->data.member.object->data.label.value;
        }
        if (const_name) {
            diag_error_codef(tc->diag, "E3005", NODE_FILE(tc, node), node->token.line, node->token.column, 0, const_name);
        }
        /* E3036 (): range check on reassignment; the var_decl path
         * already catches out-of-range literals at declaration, but
         * reassignment (`x = 300` where x is u8) was unchecked. */
        if (target->kind == NODE_LABEL && node->data.assign.value) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.label.value);
            if (sym && sym->declared_type) {
                int64_t lit_val;
                if (try_get_literal_int(node->data.assign.value, &lit_val)) {
                    check_integer_range(tc->diag, NODE_FILE(tc, node),
                        node->data.assign.value->token.line,
                        node->data.assign.value->token.column,
                        sym->declared_type, lit_val);
                }
            }
        }
        /* E3036 (): range check on struct field assignment. */
        if (target->kind == NODE_MEMBER_EXPR &&
            target->data.member.object->kind == NODE_LABEL &&
            node->data.assign.value) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.member.object->data.label.value);
            if (sym && sym->type && sym->type->kind == TK_STRUCT) {
                EzType *field_t = struct_field_type(tc, sym->type->name, target->data.member.member);
                if (field_t && field_t->name) {
                    int64_t lit_val;
                    if (try_get_literal_int(node->data.assign.value, &lit_val)) {
                        check_integer_range(tc->diag, NODE_FILE(tc, node),
                            node->data.assign.value->token.line,
                            node->data.assign.value->token.column,
                            field_t->name, lit_val);
                    }
                }
            }
        }
        /* Also handle dereferenced pointer field: p^.field = value */
        if (target->kind == NODE_MEMBER_EXPR &&
            target->data.member.object->kind == NODE_POSTFIX_EXPR &&
            target->data.member.object->data.postfix.left->kind == NODE_LABEL &&
            node->data.assign.value) {
            Symbol *sym = scope_lookup(tc->current_scope,
                target->data.member.object->data.postfix.left->data.label.value);
            if (sym && sym->type && sym->type->kind == TK_POINTER && sym->type->element_type) {
                EzType *field_t = struct_field_type(tc, sym->type->element_type, target->data.member.member);
                if (field_t && field_t->name) {
                    int64_t lit_val;
                    if (try_get_literal_int(node->data.assign.value, &lit_val)) {
                        check_integer_range(tc->diag, NODE_FILE(tc, node),
                            node->data.assign.value->token.line,
                            node->data.assign.value->token.column,
                            field_t->name, lit_val);
                    }
                }
            }
        }
        /* Check type mismatch on assignment (only for direct variable targets) */
        if (target->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.label.value);
            if (sym && sym->type->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                target_t->kind != TK_UNKNOWN &&
                target_t->kind != value_t->kind &&
                !(is_int_kind(target_t->kind) && is_int_kind(value_t->kind)) &&
                !(target_t->kind == TK_ENUM && is_int_kind(value_t->kind)) &&
                !(is_int_kind(target_t->kind) && value_t->kind == TK_ENUM) &&
                !(target_t->kind == TK_STRUCT && is_int_kind(value_t->kind)) &&
                !(target_t->kind == TK_POINTER && node->data.assign.value->kind == NODE_LABEL &&
                  scope_lookup(tc->current_scope, node->data.assign.value->data.label.value) &&
                  scope_lookup(tc->current_scope, node->data.assign.value->data.label.value)->is_ref) &&
                /* : implicit int→float coercion */
                !(target_t->kind == TK_FLOAT && is_int_kind(value_t->kind)) &&
                /* nil is a valid value for pointer and Error variables */
                !(value_t->kind == TK_NIL &&
                  (target_t->kind == TK_POINTER || target_t->kind == TK_ERROR))) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign %s to %s variable '%s'",
                    type_name(value_t), type_name(target_t), target->data.label.value);
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* Check type mismatch on struct field assignment */
        if (target->kind == NODE_MEMBER_EXPR && target->data.member.object->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.member.object->data.label.value);
            if (sym && sym->type->kind == TK_STRUCT) {
                EzType *field_t = struct_field_type(tc, sym->type->name, target->data.member.member);
                if (field_t->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                    field_t->kind != value_t->kind &&
                    !(is_int_kind(field_t->kind) && is_int_kind(value_t->kind)) &&
                    !(is_int_kind(field_t->kind) && value_t->kind == TK_ENUM) &&
                    !(field_t->kind == TK_FLOAT && is_int_kind(value_t->kind)) &&
                    /* nil is a valid value for pointer and Error fields */
                    !(value_t->kind == TK_NIL &&
                      (field_t->kind == TK_POINTER || field_t->kind == TK_ERROR))) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "type mismatch: cannot assign %s to %s field '%s'",
                        type_name(value_t), type_name(field_t), target->data.member.member);
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }
        /* : reject addr() of local assigned to outer-scope variable,
         * and warn on cross-scope pointer assignments. */
        if (target->kind == NODE_LABEL && node->data.assign.value &&
            node->data.assign.value->kind == NODE_CALL_EXPR &&
            node->data.assign.value->data.call.function->kind == NODE_LABEL &&
            strcmp(node->data.assign.value->data.call.function->data.label.value, "addr") == 0 &&
            node->data.assign.value->data.call.arg_count == 1 &&
            node->data.assign.value->data.call.args[0]->kind == NODE_LABEL) {
            const char *ptr_name = target->data.label.value;
            const char *addr_var = node->data.assign.value->data.call.args[0]->data.label.value;
            Symbol *ptr_sym_local = scope_lookup_local(tc->current_scope, ptr_name);
            Symbol *addr_sym_local = scope_lookup_local(tc->current_scope, addr_var);
            if (!ptr_sym_local && scope_lookup(tc->current_scope, ptr_name) &&
                addr_sym_local) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "pointer '%s' may reference memory from a scope that has ended; "
                    "'%s' is a local variable in an inner scope",
                    ptr_name, addr_var);
                diag_warning_msg(tc->diag, "W3004", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        break;
    }

    case NODE_RETURN_STMT:
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            resolve_expr(tc, node->data.return_stmt.values[i]);
        }
        /* main() exits when control reaches the closing brace; an
         * explicit `return` is not allowed. Without this check, codegen
         * emits `ez_scope_restore(_, _scope_mark)` referencing a
         * variable that main never declares, and the C compile fails. */
        if (tc->current_func_is_main) {
            diag_error_msg(tc->diag, "E3073",
                strdup("'return' is not allowed in main(); main exits when control reaches the closing brace"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            break;
        }
        /* : reject addr() of local variable in return; the
         * local's memory is freed when the function returns. */
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            AstNode *rv = node->data.return_stmt.values[i];
            if (rv->kind == NODE_CALL_EXPR &&
                rv->data.call.function->kind == NODE_LABEL &&
                strcmp(rv->data.call.function->data.label.value, "addr") == 0 &&
                rv->data.call.arg_count == 1 &&
                rv->data.call.args[0]->kind == NODE_LABEL) {
                const char *var_name = rv->data.call.args[0]->data.label.value;
                Symbol *sym = scope_lookup(tc->current_scope, var_name);
                if (sym) {
                    diag_error_codef(tc->diag, "E3063", NODE_FILE(tc, node), rv->token.line, rv->token.column, 0, var_name, var_name);
                }
            }
        }
        /* E3071: `return nil` from a function whose return type contains
         * '?' is unsound; nil isn't a value for every binding (int,
         * string, etc.). The codegen would otherwise emit `NULL` and let
         * clang reject the result as an int/struct conversion error.
         * Allow nil in non-primary return slots (e.g. (?, Error)).
         * Skip during the per-instantiation re-check so we only emit once. */
        if (!tc->suppress_typetable_writes &&
            tc->current_return_count > 0 && node->data.return_stmt.count > 0) {
            int n = node->data.return_stmt.count;
            int slots = n < tc->current_return_count ? n : tc->current_return_count;
            for (int i = 0; i < slots; i++) {
                AstNode *rv = node->data.return_stmt.values[i];
                if (rv->kind != NODE_NIL_VALUE) continue;
                const char *tn = (i == 0 && tc->current_return_type_names)
                    ? tc->current_return_type_names[i] : NULL;
                if (tn && type_name_has_wildcard(tn)) {
                    diag_error_code(tc->diag, "E3071", NODE_FILE(tc, rv), rv->token.line, rv->token.column, 0);
                }
            }
        }

        /* E3072: `return nil` from a function returning a non-nullable type
         * (struct, int, string, array, etc.). nil is only valid for pointer
         * and error return types. */
        if (tc->current_return_count > 0 && node->data.return_stmt.count > 0) {
            AstNode *rv = node->data.return_stmt.values[0];
            if (rv->kind == NODE_NIL_VALUE) {
                EzType *expected = tc->current_return_types[0];
                if (expected && expected->kind != TK_POINTER &&
                    expected->kind != TK_ERROR && expected->kind != TK_UNKNOWN &&
                    expected->kind != TK_NIL && expected->kind != TK_VOID) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "cannot return 'nil' from a function that returns '%s'; nil is only valid for pointer and error types",
                        type_name(expected));
                    diag_error_msg(tc->diag, "E3072", strdup(msg),
                        NODE_FILE(tc, rv), rv->token.line, rv->token.column, 0);
                }
            }
        }

        /* Check return type matches function signature */
        if (tc->current_return_count == 0 && node->data.return_stmt.count > 0) {
            /* Returning a value from a void function; suppress when
             * we've rewritten main()'s declared return type to void
             * after E4008 (). */
            if (!tc->current_main_return_suppressed) {
                diag_error_msg(tc->diag, "E3006", strdup("cannot return a value from a void function"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count == 0 &&
                   !tc->current_has_named_returns) {
            /* Bare return in non-void function (without named returns) */
            diag_error_msg(tc->diag, "E3006",
                strdup("missing return value; function expects a return value"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count != tc->current_return_count) {
            /* E3013: wrong number of return values (skip or_return synthetic returns
             * which have count=1 but the function expects more; that's handled by codegen) */
            bool is_or_return_synthetic = false;
            if (node->data.return_stmt.count == 1 &&
                node->data.return_stmt.values[0]->kind == NODE_MEMBER_EXPR) {
                AstNode *obj = node->data.return_stmt.values[0]->data.member.object;
                if (obj->kind == NODE_LABEL && strncmp(obj->data.label.value, "_ez_or", 6) == 0) {
                    is_or_return_synthetic = true;
                }
            }
            if (!is_or_return_synthetic) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "function expects %d return value(s), got %d",
                    tc->current_return_count, node->data.return_stmt.count);
                diag_error_msg(tc->diag, "E3013", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count == tc->current_return_count) {
            /* Check first return value type (skip for or_return synthetic returns) */
            EzType *ret_t = resolve_expr(tc, node->data.return_stmt.values[0]);
            EzType *expected = tc->current_return_types[0];
            /* : same push as var_decl; when a func-pointer call
             * is the return value and the function's declared return
             * type is concrete, push it onto the call node so codegen
             * uses the right function-pointer return cast. */
            if (ret_t->kind == TK_UNKNOWN && expected->kind != TK_UNKNOWN &&
                expected->kind != TK_VOID &&
                node->data.return_stmt.values[0]->kind == NODE_CALL_EXPR) {
                typetable_set(tc->type_table, node->data.return_stmt.values[0], expected);
                ret_t = expected;
            }
            if (ret_t->kind != TK_UNKNOWN && expected->kind != TK_UNKNOWN &&
                ret_t->kind != expected->kind && ret_t->kind != TK_NIL &&
                /* int/uint are compatible at kind level (E5024 checks signedness) */
                !(is_int_kind(expected->kind) && is_int_kind(ret_t->kind)) &&
                /* Allow enum → int return (enums are int-backed) but not
                 * the reverse; returning an int from a function declared
                 * to return an enum is a type error (). */
                !(is_int_kind(expected->kind) && ret_t->kind == TK_ENUM) &&
                !(expected->kind == TK_STRUCT && is_int_kind(ret_t->kind)) &&
                !(is_int_kind(expected->kind) && ret_t->kind == TK_STRUCT) &&
                !(expected->kind == TK_FLOAT && is_int_kind(ret_t->kind))) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "return type mismatch: expected %s, got %s",
                    type_name(expected), type_name(ret_t));
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Struct-to-struct return name mismatch */
            if (ret_t->kind == TK_STRUCT && expected->kind == TK_STRUCT &&
                ret_t->name && expected->name &&
                strcmp(ret_t->name, expected->name) != 0) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "return type mismatch: expected '%s', got '%s'",
                    expected->name, ret_t->name);
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* : pointer depth mismatch (e.g. returning ^^int
             * from a function declared -> ^int). Both sides are
             * TK_POINTER so the kind check above passes, but the
             * element_type strings differ ("int" vs "^int"). */
            if (ret_t->kind == TK_POINTER && expected->kind == TK_POINTER &&
                ret_t->element_type && expected->element_type &&
                strcmp(ret_t->element_type, expected->element_type) != 0) {
                /* Build human-readable pointer type strings */
                char exp_str[EZ_TYPE_NAME_MAX], got_str[EZ_TYPE_NAME_MAX];
                snprintf(exp_str, sizeof(exp_str), "^%s", expected->element_type);
                snprintf(got_str, sizeof(got_str), "^%s", ret_t->element_type);
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "return type mismatch: expected '%s', got '%s'",
                    exp_str, got_str);
                diag_error_msg(tc->diag, "E3001", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E5024: signed-to-unsigned return type mismatch */
            if (tc->current_return_type_names && tc->current_return_type_names[0] &&
                is_unsigned_type(tc->current_return_type_names[0]) &&
                node->data.return_stmt.values[0]->kind == NODE_LABEL) {
                const char *src_name = node->data.return_stmt.values[0]->data.label.value;
                Symbol *src_sym = scope_lookup(tc->current_scope, src_name);
                if (src_sym && src_sym->declared_type &&
                    is_signed_int_type(src_sym->declared_type)) {
                    diag_error_codef(tc->diag, "E5024", NODE_FILE(tc, node), node->token.line, node->token.column, 0, src_sym->declared_type, tc->current_return_type_names[0]);
                }
            }
            /* E3073: named return variable must be the value returned */
            if (tc->current_has_named_returns && tc->current_return_names) {
                for (int i = 0; i < node->data.return_stmt.count && i < tc->current_return_count; i++) {
                    if (!tc->current_return_names[i]) continue;
                    AstNode *rv = node->data.return_stmt.values[i];
                    bool is_named_var = (rv->kind == NODE_LABEL &&
                        strcmp(rv->data.label.value, tc->current_return_names[i]) == 0);
                    if (!is_named_var) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "function must return named variable '%s', not a different expression",
                            tc->current_return_names[i]);
                        diag_error_msg(tc->diag, "E3080", strdup(msg),
                            NODE_FILE(tc, rv), rv->token.line, rv->token.column, 0);
                    }
                }
            }
        }
        break;

    case NODE_EXPR_STMT: {
        EzType *expr_t = resolve_expr(tc, node->data.expr_stmt.expr);
        /* E3081: bare function name used as statement without call */
        AstNode *expr = node->data.expr_stmt.expr;
        if (expr && expr->kind == NODE_LABEL) {
            const char *name = expr->data.label.value;
            if (tc_is_builtin(name) || find_func(tc, name)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "function '%s' used as a statement without being called; did you mean '%s()'?",
                    name, name);
                diag_error_msg(tc->diag, "E3081", strdup(msg),
                    NODE_FILE(tc, expr), expr->token.line, expr->token.column, 0);
            }
        }
        if (expr && expr->kind == NODE_CALL_EXPR && expr_t &&
            expr_t->kind != TK_VOID && expr_t->kind != TK_UNKNOWN) {
            AstNode *fn = expr->data.call.function;
            const char *fn_name = NULL;
            if (fn->kind == NODE_LABEL) fn_name = fn->data.label.value;
            /* Don't warn for known side-effect functions */
            bool is_side_effect = fn_name && (
                strcmp(fn_name, "println") == 0 || strcmp(fn_name, "print") == 0 ||
                strcmp(fn_name, "eprintln") == 0 || strcmp(fn_name, "eprint") == 0 ||
                strcmp(fn_name, "panic") == 0 || strcmp(fn_name, "assert") == 0 ||
                strcmp(fn_name, "exit") == 0 || strcmp(fn_name, "sleep_s") == 0 ||
                strcmp(fn_name, "sleep_ms") == 0 || strcmp(fn_name, "sleep_ns") == 0);
            /* For member expression calls, check if the return type is void —
             * only warn about non-void return values being discarded */
            if (fn->kind == NODE_MEMBER_EXPR) {
                /* expr_t is already the resolved return type from resolve_expr above.
                 * If it's void or unknown, this is a side-effect call; no warning needed. */
                if (expr_t->kind == TK_VOID || expr_t->kind == TK_UNKNOWN) {
                    is_side_effect = true;
                } else {
                    /* Build display name for the error message */
                    const char *obj_name = NULL;
                    const char *mem_name = NULL;
                    if (fn->data.member.object->kind == NODE_LABEL) {
                        obj_name = fn->data.member.object->data.label.value;
                        mem_name = fn->data.member.member;
                    }
                    if (obj_name && mem_name && !is_side_effect) {
                        char full[256];
                        snprintf(full, sizeof(full), "%s.%s()", obj_name, mem_name);
                        diag_error_codef(tc->diag, "E5011",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                            full);
                    }
                    is_side_effect = true; /* already handled */
                }
            }
            if (!is_side_effect && fn_name) {
                char full[256];
                snprintf(full, sizeof(full), "%s()", fn_name);
                diag_error_codef(tc->diag, "E5011",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    full);
            }
        }
        /* : double-free detection for mem.destroy() */
        if (expr && expr->kind == NODE_CALL_EXPR &&
            expr->data.call.function->kind == NODE_MEMBER_EXPR) {
            AstNode *obj = expr->data.call.function->data.member.object;
            const char *mem_fn = expr->data.call.function->data.member.member;
            if (obj->kind == NODE_LABEL && strcmp(mem_fn, "destroy") == 0 &&
                strcmp(obj->data.label.value, "mem") == 0 &&
                expr->data.call.arg_count == 1 &&
                expr->data.call.args[0]->kind == NODE_LABEL) {
                const char *arena_name = expr->data.call.args[0]->data.label.value;
                bool already_destroyed = false;
                for (int di = 0; di < tc->destroyed_arena_count; di++) {
                    if (strcmp(tc->destroyed_arenas[di], arena_name) == 0) {
                        already_destroyed = true;
                        break;
                    }
                }
                if (already_destroyed) {
                    diag_error_codef(tc->diag, "E3064",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        "mem.destroy", arena_name, arena_name);
                } else {
                    if (tc->destroyed_arena_count >= tc->destroyed_arena_cap) {
                        tc->destroyed_arena_cap = tc->destroyed_arena_cap ? tc->destroyed_arena_cap * 2 : 8;
                        tc->destroyed_arenas = realloc(tc->destroyed_arenas,
                            (size_t)tc->destroyed_arena_cap * sizeof(const char *));
                    }
                    tc->destroyed_arenas[tc->destroyed_arena_count++] = arena_name;
                }
            }
        }
        /* Also catch bare destroy() via 'using mem' */
        if (expr && expr->kind == NODE_CALL_EXPR &&
            expr->data.call.function->kind == NODE_LABEL &&
            strcmp(expr->data.call.function->data.label.value, "destroy") == 0 &&
            expr->data.call.arg_count == 1 &&
            expr->data.call.args[0]->kind == NODE_LABEL) {
            bool is_mem_using = false;
            for (int ui = 0; ui < tc->using_module_count; ui++) {
                if (strcmp(tc->using_modules[ui], "mem") == 0) { is_mem_using = true; break; }
            }
            if (is_mem_using) {
                const char *arena_name = expr->data.call.args[0]->data.label.value;
                bool already_destroyed = false;
                for (int di = 0; di < tc->destroyed_arena_count; di++) {
                    if (strcmp(tc->destroyed_arenas[di], arena_name) == 0) {
                        already_destroyed = true;
                        break;
                    }
                }
                if (already_destroyed) {
                    diag_error_codef(tc->diag, "E3064",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        "destroy", arena_name, arena_name);
                } else {
                    if (tc->destroyed_arena_count >= tc->destroyed_arena_cap) {
                        tc->destroyed_arena_cap = tc->destroyed_arena_cap ? tc->destroyed_arena_cap * 2 : 8;
                        tc->destroyed_arenas = realloc(tc->destroyed_arenas,
                            (size_t)tc->destroyed_arena_cap * sizeof(const char *));
                    }
                    tc->destroyed_arenas[tc->destroyed_arena_count++] = arena_name;
                }
            }
        }
        break;
    }

    case NODE_BLOCK_STMT:
        /* Inline blocks (from multi-var expansion) share parent scope.
         * Only control flow blocks (if, for, etc.) create new scopes,
         * and those are handled by their own cases. */
        check_block(tc, node);
        break;

    case NODE_IF_STMT: {
        EzType *cond_t = resolve_expr(tc, node->data.if_stmt.condition);
        /* E3038 (): void function call as condition. The same check
         * already exists for variable assignment and arithmetic; wire
         * it up for control-flow conditions too. The 'or' branch of an
         * if chain is parsed as a nested NODE_IF_STMT, so this one spot
         * covers 'if' and every subsequent 'or'. */
        if (cond_t && cond_t->kind == TK_VOID) {
            AstNode *c = node->data.if_stmt.condition;
            char msg[EZ_MSG_BUF_SIZE];
            if (c && c->kind == NODE_CALL_EXPR && c->data.call.function &&
                c->data.call.function->kind == NODE_LABEL) {
                snprintf(msg, sizeof(msg),
                    "cannot use void function '%s' as condition; 'if' requires a bool expression",
                    c->data.call.function->data.label.value);
            } else {
                snprintf(msg, sizeof(msg),
                    "cannot use void expression as condition; 'if' requires a bool expression");
            }
            diag_error_msg(tc->diag, "E3038", strdup(msg),
                NODE_FILE(tc, c), c->token.line, c->token.column, 0);
        }
        Scope *if_outer = tc->current_scope;
        tc->current_scope = scope_create(if_outer);
        check_block(tc, node->data.if_stmt.consequence);
        tc->current_scope = if_outer;
        if (node->data.if_stmt.alternative) {
            tc->current_scope = scope_create(if_outer);
            check_statement(tc, node->data.if_stmt.alternative);
            tc->current_scope = if_outer;
        }
        break;
    }

    case NODE_FOR_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;
        scope_define(loop_scope, node->data.for_stmt.var_name, &TYPE_INT, false);
        resolve_expr(tc, node->data.for_stmt.iterable);
        /* E9005: check range bounds when both are literals */
        if (node->data.for_stmt.iterable &&
            node->data.for_stmt.iterable->kind == NODE_RANGE_EXPR) {
            AstNode *r = node->data.for_stmt.iterable;
            if (r->data.range_expr.start && r->data.range_expr.end &&
                r->data.range_expr.start->kind == NODE_INT_VALUE &&
                r->data.range_expr.end->kind == NODE_INT_VALUE) {
                int64_t start_val = r->data.range_expr.start->data.int_value.value;
                int64_t end_val = r->data.range_expr.end->data.int_value.value;
                bool has_neg_step = r->data.range_expr.step &&
                    r->data.range_expr.step->kind == NODE_INT_VALUE &&
                    r->data.range_expr.step->data.int_value.value < 0;
                bool has_neg_prefix = r->data.range_expr.step &&
                    r->data.range_expr.step->kind == NODE_PREFIX_EXPR &&
                    strcmp(r->data.range_expr.step->data.prefix.op, "-") == 0;
                bool negative_step = has_neg_step || has_neg_prefix;
                bool invalid = negative_step ? (start_val < end_val) : (start_val >= end_val);
                if (invalid) {
                    char msg[EZ_MSG_BUF_SIZE];
                    if (negative_step) {
                        snprintf(msg, sizeof(msg),
                            "invalid range: start (%lld) must be greater than or equal to end (%lld) for negative step",
                            (long long)start_val, (long long)end_val);
                    } else {
                        snprintf(msg, sizeof(msg),
                            "invalid range: start (%lld) must be less than end (%lld)",
                            (long long)start_val, (long long)end_val);
                    }
                    diag_error_msg(tc->diag, "E9005", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }
        tc->loop_depth++;
        check_block(tc, node->data.for_stmt.body);
        tc->loop_depth--;
        tc->current_scope = outer;
        break;
    }

    case NODE_FOR_EACH_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;

        /* Resolve collection type to determine element type */
        EzType *coll_t = resolve_expr(tc, node->data.for_each.collection);

        /* Check that collection is iterable */
        if (coll_t->kind != TK_UNKNOWN && coll_t->kind != TK_ARRAY &&
            coll_t->kind != TK_MAP && coll_t->kind != TK_STRING) {
            diag_error_codef(tc->diag, "E3009", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_name(coll_t));
        }

        if (coll_t->kind == TK_MAP) {
            /* Map iteration: for_each k, v in map OR for_each key in map */
            EzType *key_t = coll_t->key_type ? type_from_name(coll_t->key_type) : &TYPE_STRING;
            EzType *val_t = coll_t->value_type ? type_from_name(coll_t->value_type) : &TYPE_UNKNOWN;
            if (node->data.for_each.index_name) {
                /* Two-var: index_name = key, var_name = value */
                scope_define(loop_scope, node->data.for_each.index_name, key_t, false);
                scope_define(loop_scope, node->data.for_each.var_name, val_t, false);
            } else {
                /* One-var: var_name = key */
                scope_define(loop_scope, node->data.for_each.var_name, key_t, false);
            }
        } else {
            /* Array/string iteration */
            EzType *elem_t = &TYPE_UNKNOWN;
            if (coll_t->kind == TK_ARRAY && coll_t->element_type) {
                elem_t = type_from_name(coll_t->element_type);
            } else if (coll_t->kind == TK_STRING) {
                elem_t = &TYPE_CHAR;
            }
            if (node->data.for_each.index_name) {
                scope_define(loop_scope, node->data.for_each.index_name, &TYPE_INT, false);
            }
            scope_define(loop_scope, node->data.for_each.var_name, elem_t, false);
        }

        tc->loop_depth++;
        check_block(tc, node->data.for_each.body);
        tc->loop_depth--;
        tc->current_scope = outer;
        break;
    }

    case NODE_WHILE_STMT: {
        EzType *wh_cond_t = resolve_expr(tc, node->data.while_stmt.condition);
        /* E3038 (): void function call as 'as_long_as' condition. */
        if (wh_cond_t && wh_cond_t->kind == TK_VOID) {
            AstNode *c = node->data.while_stmt.condition;
            char msg[EZ_MSG_BUF_SIZE];
            if (c && c->kind == NODE_CALL_EXPR && c->data.call.function &&
                c->data.call.function->kind == NODE_LABEL) {
                snprintf(msg, sizeof(msg),
                    "cannot use void function '%s' as condition; 'as_long_as' requires a bool expression",
                    c->data.call.function->data.label.value);
            } else {
                snprintf(msg, sizeof(msg),
                    "cannot use void expression as condition; 'as_long_as' requires a bool expression");
            }
            diag_error_msg(tc->diag, "E3038", strdup(msg),
                NODE_FILE(tc, c), c->token.line, c->token.column, 0);
        }
        tc->loop_depth++;
        check_block(tc, node->data.while_stmt.body);
        tc->loop_depth--;
        break;
    }

    case NODE_LOOP_STMT:
        tc->loop_depth++;
        check_block(tc, node->data.loop_stmt.body);
        tc->loop_depth--;
        break;

    case NODE_BREAK_STMT:
    case NODE_CONTINUE_STMT:
        if (tc->loop_depth == 0) {
            diag_error_code(tc->diag, "E2050",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        break;

    case NODE_FUNC_DECL: {
        /* E2038: reserved type name as function name */
        if (is_reserved_type_name(node->data.func_decl.name)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "'%s' is a reserved type name and cannot be used as a function name",
                node->data.func_decl.name);
            diag_error_msg(tc->diag, "E2038", strdup(msg),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Check for nested function declarations */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2051", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.func_decl.name);
        }

        Scope *func_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = func_scope;
        tc->func_depth++;
        tc->destroyed_arena_count = 0;

        /* Define parameters in function scope, check for duplicates */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            Param *p = &node->data.func_decl.params[i];
            /* E2038: reserved type name as parameter name */
            if (is_reserved_type_name(p->name)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "'%s' is a reserved type name and cannot be used as a parameter name",
                    p->name);
                diag_error_msg(tc->diag, "E2038", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Check for duplicate parameter name */
            for (int j = 0; j < i; j++) {
                if (strcmp(node->data.func_decl.params[j].name, p->name) == 0) {
                    diag_error_codef(tc->diag, "E2012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, p->name);
                    break;
                }
            }
            /* E2039: required param after param with default value */
            if (i > 0 && !p->default_value) {
                bool prev_has_default = false;
                for (int k = 0; k < i; k++) {
                    if (node->data.func_decl.params[k].default_value) {
                        prev_has_default = true;
                        break;
                    }
                }
                if (prev_has_default) {
                    diag_error_codef(tc->diag, "E2039", NODE_FILE(tc, node), node->token.line, node->token.column, 0, p->name);
                }
            }
            EzType *ptype = p->type_name ? tc_type_from_name(tc, p->type_name) : &TYPE_UNKNOWN;
            /* E4016: undefined parameter type */
            if (p->type_name && ptype->kind == TK_UNKNOWN &&
                p->type_name[0] >= 'A' && p->type_name[0] <= 'Z') {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "undefined type '%s'; check the spelling or import the module that defines it",
                    p->type_name);
                diag_error_msg(tc->diag, "E4016", strdup(msg),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3001: validate default value type matches parameter type */
            if (p->default_value && p->type_name) {
                EzType *def_t = resolve_expr(tc, p->default_value);
                if (def_t->kind != TK_UNKNOWN && ptype->kind != TK_UNKNOWN &&
                    def_t->kind != ptype->kind &&
                    !(is_int_kind(def_t->kind) && is_int_kind(ptype->kind)) &&
                    !(def_t->kind == TK_NIL)) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "default value for parameter '%s' has wrong type; expected %s, got %s",
                        p->name, p->type_name, type_name(def_t));
                    diag_error_msg(tc->diag, "E3001", strdup(msg),
                        NODE_FILE(tc, p->default_value), p->default_value->token.line, p->default_value->token.column, 0);
                }
            }
            scope_define(func_scope, p->name, ptype, p->mutable);
        }

        /* E3060: wildcard in return type but no wildcard in any parameter.
         * Suppress when every wildcard return is in a named position — E3082
         * handles that case with a more specific message. */
        {
            bool ret_has_wc = false;
            bool all_wc_named = true;
            bool param_has_wc = false;
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                if (type_name_has_wildcard(node->data.func_decl.return_types[i])) {
                    ret_has_wc = true;
                    if (!node->data.func_decl.return_names ||
                        !node->data.func_decl.return_names[i]) {
                        all_wc_named = false;
                    }
                }
            }
            if (ret_has_wc && !all_wc_named) {
                for (int i = 0; i < node->data.func_decl.param_count; i++) {
                    if (type_name_has_wildcard(node->data.func_decl.params[i].type_name)) {
                        param_has_wc = true;
                        break;
                    }
                }
                if (!param_has_wc) {
                    diag_error_code(tc->diag, "E3060", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }

        /* Define named return variables in function scope */
        if (node->data.func_decl.return_names) {
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                if (node->data.func_decl.return_names[i]) {
                    const char *rn = node->data.func_decl.return_names[i];
                    /* E2063: duplicate named return value */
                    for (int j = 0; j < i; j++) {
                        if (node->data.func_decl.return_names[j] &&
                            strcmp(node->data.func_decl.return_names[j], rn) == 0) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "duplicate named return value '%s'", rn);
                            diag_error_msg(tc->diag, "E2063", strdup(msg),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            break;
                        }
                    }
                    /* E3082: wildcard type '?' in named return position */
                    if (i < node->data.func_decl.return_type_count &&
                        node->data.func_decl.return_types[i] &&
                        strcmp(node->data.func_decl.return_types[i], "?") == 0) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "wildcard type '?' cannot be used in named return value '%s'; use an unnamed return instead (e.g. -> (?, int))",
                            rn);
                        diag_error_msg(tc->diag, "E3082", strdup(msg),
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* E2063: named return collides with parameter */
                    for (int j = 0; j < node->data.func_decl.param_count; j++) {
                        if (strcmp(node->data.func_decl.params[j].name, rn) == 0) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "named return value '%s' conflicts with parameter '%s'",
                                rn, rn);
                            diag_error_msg(tc->diag, "E2063", strdup(msg),
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            break;
                        }
                    }
                }
            }
        }

        /* Save using-module count so function-scoped `using` doesn't
         * leak to subsequent functions (). */
        int prev_using_count = tc->using_module_count;

        /* Track current function return types for return statement checking */
        EzType **prev_ret = tc->current_return_types;
        const char **prev_ret_names = tc->current_return_type_names;
        int prev_ret_count = tc->current_return_count;
        bool prev_named = tc->current_has_named_returns;

        /* Detect named return values */
        const char **prev_return_names = tc->current_return_names;
        tc->current_has_named_returns = false;
        tc->current_return_names = node->data.func_decl.return_names;
        if (node->data.func_decl.return_names) {
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                if (node->data.func_decl.return_names[i]) {
                    tc->current_has_named_returns = true;
                    break;
                }
            }
        }

        if (node->data.func_decl.return_type_count > 0) {
            tc->current_return_types = malloc(sizeof(EzType *) * node->data.func_decl.return_type_count);
            tc->current_return_type_names = malloc(sizeof(const char *) * node->data.func_decl.return_type_count);
            tc->current_return_count = node->data.func_decl.return_type_count;
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                tc->current_return_types[i] = tc_type_from_name(tc, node->data.func_decl.return_types[i]);
                tc->current_return_type_names[i] = node->data.func_decl.return_types[i];
                /* E4016: undefined return type */
                const char *rtn = node->data.func_decl.return_types[i];
                if (rtn && tc->current_return_types[i]->kind == TK_UNKNOWN &&
                    rtn[0] >= 'A' && rtn[0] <= 'Z') {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "undefined type '%s'; check the spelling or import the module that defines it",
                        rtn);
                    diag_error_msg(tc->diag, "E4016", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        } else {
            tc->current_return_types = NULL;
            tc->current_return_type_names = NULL;
            tc->current_return_count = 0;
        }

        /* : main() is always void, and E4008 was already emitted
         * by register_declarations when the user attached a return
         * type. Treat main's effective return type as void for the
         * body walk so downstream "must return a value" (E3024),
         * "cannot return a value from a void function" (E3006), and
         * return-type-mismatch cascades don't fire on top of the
         * E4008 the user is already going to fix. The suppression
         * flag lets individual checks distinguish "real void
         * function" from "main that tried to declare a return type
         * but got rewritten to void"; for the latter, we want
         * silence, not a different cascade. */
        bool main_return_coerced = false;
        if (strcmp(node->data.func_decl.name, "main") == 0 &&
            tc->current_return_count > 0) {
            free(tc->current_return_types);
            free((void *)tc->current_return_type_names);
            tc->current_return_types = NULL;
            tc->current_return_type_names = NULL;
            tc->current_return_count = 0;
            main_return_coerced = true;
        }
        bool saved_main_suppressed = tc->current_main_return_suppressed;
        tc->current_main_return_suppressed = main_return_coerced;
        bool saved_is_main = tc->current_func_is_main;
        tc->current_func_is_main =
            (strcmp(node->data.func_decl.name, "main") == 0);

        check_block(tc, node->data.func_decl.body);

        /* E3070: ensure must be at the function body's top level. */
        check_no_nested_ensure(tc, node->data.func_decl.body, false);

        /* Check for missing return in non-void function (simple: check last statement) */
        if (tc->current_return_count > 0 && node->data.func_decl.body &&
            node->data.func_decl.body->kind == NODE_BLOCK_STMT) {
            AstNode *body = node->data.func_decl.body;
            bool has_return = false;
            /* Recursively check if any statement in the body is a return */
            for (int i = 0; i < body->data.block.count; i++) {
                if (block_has_return(body->data.block.stmts[i])) {
                    has_return = true;
                    break;
                }
            }
            /* Also check named returns (if return names are set, implicit return is OK) */
            bool has_named_returns = false;
            if (node->data.func_decl.return_names) {
                for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                    if (node->data.func_decl.return_names[i]) {
                        has_named_returns = true;
                        break;
                    }
                }
            }
            if (!has_return && !has_named_returns) {
                diag_error_codef(tc->diag, "E3024", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.func_decl.name);
            } else if (has_return && !has_named_returns &&
                       !all_paths_return(node->data.func_decl.body)) {
                diag_error_codef(tc->diag, "E3035", NODE_FILE(tc, node), node->token.line, node->token.column, 0, node->data.func_decl.name);
            }
        }

        /* W2011: named return value declared but no matching variable in body */
        if (node->data.func_decl.return_names) {
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                const char *rn = node->data.func_decl.return_names[i];
                if (!rn) continue;
                Symbol *sym = scope_lookup_local(func_scope, rn);
                if (!sym) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "named return value '%s' is declared in the signature but no matching variable exists in the function body",
                        rn);
                    diag_warning_msg(tc->diag, "W2011", strdup(msg),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }

        /* Warn about unused variables in this function scope */
        for (int si = 0; si < func_scope->count; si++) {
            Symbol *s = &func_scope->symbols[si];
            if (!s->used && s->name[0] != '_' && s->def_line > 0) {
                /* Skip function parameters (they have def_line == 0 or from param list) */
                bool is_param = false;
                for (int pi = 0; pi < node->data.func_decl.param_count; pi++) {
                    if (strcmp(node->data.func_decl.params[pi].name, s->name) == 0) {
                        is_param = true;
                        break;
                    }
                }
                if (!is_param) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "variable '%s' is declared but never used", s->name);
                    diag_warning_msg(tc->diag, "W1001", strdup(msg),
                        tc->file, s->def_line, s->def_column, 0);
                }
            }
        }

        if (tc->current_return_types) free(tc->current_return_types);
        if (tc->current_return_type_names) free(tc->current_return_type_names);
        tc->current_return_types = prev_ret;
        tc->current_return_type_names = prev_ret_names;
        tc->current_return_count = prev_ret_count;
        tc->current_has_named_returns = prev_named;
        tc->current_return_names = prev_return_names;
        tc->current_main_return_suppressed = saved_main_suppressed;
        tc->current_func_is_main = saved_is_main;
        tc->using_module_count = prev_using_count;
        tc->func_depth--;
        tc->current_scope = outer;
        break;
    }

    case NODE_IMPORT_STMT:
        if (tc->func_depth > 0) {
            diag_error_code(tc->diag, "E2036", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        break;

    case NODE_USING_STMT:
        /* Function-scoped using: add modules to the using list so
         * bare-name resolution works for the rest of this scope. */
        for (int j = 0; j < node->data.using_stmt.count; j++) {
            if (tc->using_module_count >= tc->using_module_cap) {
                tc->using_module_cap = tc->using_module_cap ? tc->using_module_cap * 2 : 8;
                tc->using_modules = realloc(tc->using_modules,
                    sizeof(const char *) * (size_t)tc->using_module_cap);
            }
            tc->using_modules[tc->using_module_count++] = node->data.using_stmt.modules[j];
        }
        break;

    case NODE_ENSURE_STMT:
        resolve_expr(tc, node->data.ensure_stmt.expr);
        if (node->data.ensure_stmt.expr &&
            node->data.ensure_stmt.expr->kind != NODE_CALL_EXPR) {
            diag_error_code(tc->diag, "E3039", NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        break;

    case NODE_STRUCT_DECL:
        /* E2053: struct inside function */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2053",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "struct", node->data.struct_decl.name);
        }
        /* Type-check struct-namespaced function bodies */
        for (int i = 0; i < node->data.struct_decl.func_count; i++) {
            AstNode *fn = node->data.struct_decl.funcs[i].func_decl;
            if (fn && fn->kind == NODE_FUNC_DECL) {
                check_statement(tc, fn);
            }
        }
        break;

    case NODE_ENUM_DECL:
        /* E2053: enum inside function */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2053",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "enum", node->data.enum_decl.name);
        }
        break;

    case NODE_WHEN_STMT: {
        EzType *when_t = resolve_expr(tc, node->data.when_stmt.value);
        /* W2012: float subjects use bit-equality, which is rarely what the
         * user wants given 0.1 + 0.2 != 0.3. */
        if (when_t && when_t->kind == TK_FLOAT) {
            AstNode *subj = node->data.when_stmt.value;
            diag_warning_code(tc->diag, "W2012", NODE_FILE(tc, subj), subj->token.line, subj->token.column, 0);
        }
        /* E2043: check for duplicate case values, E3001: check type match */
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            for (int j = 0; j < node->data.when_stmt.cases[i].value_count; j++) {
                AstNode *val_i = node->data.when_stmt.cases[i].values[j];
                EzType *case_t = resolve_expr(tc, val_i);
                /* Check case value type matches scrutinee; skip range exprs and unknowns */
                if (when_t && case_t &&
                    when_t->kind != TK_UNKNOWN && case_t->kind != TK_UNKNOWN &&
                    val_i->kind != NODE_RANGE_EXPR &&
                    !(val_i->kind == NODE_CALL_EXPR && val_i->data.call.function->kind == NODE_LABEL &&
                      strcmp(val_i->data.call.function->data.label.value, "range") == 0)) {
                    bool compat = (when_t->kind == case_t->kind) ||
                        (is_int_kind(when_t->kind) && is_int_kind(case_t->kind)) ||
                        (is_int_kind(when_t->kind) && case_t->kind == TK_ENUM) ||
                        (when_t->kind == TK_ENUM && is_int_kind(case_t->kind)) ||
                        (is_int_kind(when_t->kind) && case_t->kind == TK_BYTE) ||
                        (when_t->kind == TK_BYTE && is_int_kind(case_t->kind));
                    if (!compat) {
                        diag_error_codef(tc->diag, "E3018", NODE_FILE(tc, val_i), val_i->token.line, val_i->token.column, 0, type_name(when_t), type_name(case_t));
                    }
                }
                /* Compare against all previous case values */
                for (int pi = 0; pi < i; pi++) {
                    for (int pj = 0; pj < node->data.when_stmt.cases[pi].value_count; pj++) {
                        AstNode *val_p = node->data.when_stmt.cases[pi].values[pj];
                        bool dup = false;
                        if (val_i->kind == NODE_INT_VALUE && val_p->kind == NODE_INT_VALUE &&
                            val_i->data.int_value.value == val_p->data.int_value.value) dup = true;
                        if (val_i->kind == NODE_STRING_VALUE && val_p->kind == NODE_STRING_VALUE &&
                            strcmp(val_i->data.string_value.value, val_p->data.string_value.value) == 0) dup = true;
                        if (dup) {
                            diag_error_code(tc->diag, "E2043", NODE_FILE(tc, val_i), val_i->token.line, val_i->token.column, 0);
                        }
                    }
                }
            }
            Scope *case_outer = tc->current_scope;
            tc->current_scope = scope_create(case_outer);
            check_block(tc, node->data.when_stmt.cases[i].body);
            tc->current_scope = case_outer;
        }
        if (node->data.when_stmt.default_body) {
            Scope *def_outer = tc->current_scope;
            tc->current_scope = scope_create(def_outer);
            check_block(tc, node->data.when_stmt.default_body);
            tc->current_scope = def_outer;
        }
        /* #strict exhaustiveness check for enum types */
        if (node->data.when_stmt.is_strict && !node->data.when_stmt.default_body) {
            /* Infer the enum name from case values (e.g., Color.RED → "Color") */
            const char *enum_name = NULL;
            for (int ci = 0; ci < node->data.when_stmt.case_count && !enum_name; ci++) {
                for (int cj = 0; cj < node->data.when_stmt.cases[ci].value_count && !enum_name; cj++) {
                    AstNode *cv = node->data.when_stmt.cases[ci].values[cj];
                    if (cv->kind == NODE_MEMBER_EXPR &&
                        cv->data.member.object->kind == NODE_LABEL) {
                        const char *name = cv->data.member.object->data.label.value;
                        if (is_enum_name(tc, name)) enum_name = name;
                    }
                }
            }
            if (enum_name) {
                /* Find the enum's variants */
                int enum_idx = -1;
                for (int ei = 0; ei < tc->enum_count; ei++) {
                    if (strcmp(tc->enum_names[ei], enum_name) == 0) {
                        enum_idx = ei;
                        break;
                    }
                }
                if (enum_idx >= 0) {
                    int variant_count = tc->enum_value_counts[enum_idx];
                    const char **variants = tc->enum_values[enum_idx];
                    /* Collect covered variants from case branches */
                    for (int vi = 0; vi < variant_count; vi++) {
                        bool covered = false;
                        for (int ci = 0; ci < node->data.when_stmt.case_count && !covered; ci++) {
                            for (int cj = 0; cj < node->data.when_stmt.cases[ci].value_count && !covered; cj++) {
                                AstNode *cv = node->data.when_stmt.cases[ci].values[cj];
                                /* Match Enum.VARIANT pattern */
                                if (cv->kind == NODE_MEMBER_EXPR &&
                                    cv->data.member.object->kind == NODE_LABEL &&
                                    strcmp(cv->data.member.member, variants[vi]) == 0) {
                                    covered = true;
                                }
                                /* Match bare integer literal (for auto-increment enums) */
                                if (cv->kind == NODE_INT_VALUE &&
                                    cv->data.int_value.value == vi) {
                                    covered = true;
                                }
                            }
                        }
                        if (!covered) {
                            diag_error_codef(tc->diag, "E3056", NODE_FILE(tc, node), node->token.line, node->token.column, 0, enum_name, variants[vi]);
                        }
                    }
                }
            } else {
                /* #strict on non-enum: just warn that it has no effect without default */
                diag_error_msg(tc->diag, "E3056",
                    strdup("#strict when on a non-enum type requires a default branch to be exhaustive"),
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        break;
    }

    default:
        break;
    }
}

/* --- Registration pass --- */

/* Known stdlib module names */
static bool is_valid_module(const char *name) {
    static const char *modules[] = {
        "math", "strings", "arrays", "maps", "io", "os", "time",
        "random", "json", "csv", "encoding", "crypto", "uuid", "bytes",
        "binary", "fmt", "http", "server", "regex", "net", "threads",
        "sync", "atomic", "channels", "mem", "sqlite", "errors", "db", NULL
    };
    for (int i = 0; modules[i]; i++) {
        if (strcmp(name, modules[i]) == 0) return true;
    }
    return false;
}

/* : look up a struct declaration in the program by name. Returns
 * NULL if no struct with the given name exists. Used by the by-value
 * recursion detector below. */
static AstNode *find_struct_in_program(AstNode *program, const char *name) {
    if (!program || !name) return NULL;
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *s = program->data.program.stmts[i];
        if (s && s->kind == NODE_STRUCT_DECL && s->data.struct_decl.name &&
            strcmp(s->data.struct_decl.name, name) == 0) {
            return s;
        }
    }
    return NULL;
}

/* : depth-first walk of a struct's by-value field graph, testing
 * whether `target` appears transitively. Pointer fields (^T), arrays
 * ([T]), and maps (map[K:V]) are treated as heap-indirected; they
 * break the chain since the inner type lives behind a fat-pointer
 * header, not inline. `visited` is a stack of struct names on the
 * current DFS path used to short-circuit cycles that don't touch
 * `target` directly. */
static bool struct_contains_by_value(AstNode *program, AstNode *decl,
                                      const char *target,
                                      const char **visited, int *visited_count,
                                      int visited_cap) {
    if (!decl || decl->kind != NODE_STRUCT_DECL) return false;
    for (int i = 0; i < *visited_count; i++) {
        if (strcmp(visited[i], decl->data.struct_decl.name) == 0) return false;
    }
    if (*visited_count >= visited_cap) return false;
    visited[(*visited_count)++] = decl->data.struct_decl.name;

    for (int i = 0; i < decl->data.struct_decl.field_count; i++) {
        const char *ftn = decl->data.struct_decl.fields[i].type_name;
        if (!ftn || !*ftn) continue;
        /* Pointer, array, map; heap-indirected, size doesn't propagate. */
        if (ftn[0] == '^' || ftn[0] == '[' || strncmp(ftn, "map[", 4) == 0) continue;
        if (strcmp(ftn, target) == 0) {
            (*visited_count)--;
            return true;
        }
        AstNode *child = find_struct_in_program(program, ftn);
        if (child && struct_contains_by_value(program, child, target,
                                               visited, visited_count, visited_cap)) {
            (*visited_count)--;
            return true;
        }
    }
    (*visited_count)--;
    return false;
}

static void register_declarations(TypeChecker *tc, AstNode *program) {
    tc->registering = true;
    /* Validate and record imports */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_IMPORT_STMT) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->is_stdlib && item->module && !is_valid_module(item->module)) {
                    diag_error_codef(tc->diag, "E6001", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, item->module);
                }
                /* Record import for unused-import tracking.
                 * Store the ALIAS as the import name (so using/dot notation uses the alias). */
                if (item->module || item->alias) {
                    if (tc->import_count >= tc->import_cap) {
                        tc->import_cap = tc->import_cap ? tc->import_cap * 2 : 16;
                        tc->imported_modules = realloc(tc->imported_modules, sizeof(const char *) * tc->import_cap);
                        tc->import_lines = realloc(tc->import_lines, sizeof(int) * tc->import_cap);
                        tc->import_used = realloc(tc->import_used, sizeof(bool) * tc->import_cap);
                        tc->import_is_stdlib = realloc(tc->import_is_stdlib, sizeof(bool) * tc->import_cap);
                    }
                    tc->imported_modules[tc->import_count] = item->alias ? item->alias : item->module;
                    tc->import_lines[tc->import_count] = stmt->token.line;
                    tc->import_used[tc->import_count] = item->is_c_import; /* C imports are always "used" */
                    tc->import_is_stdlib[tc->import_count] = item->is_stdlib;
                    tc->import_count++;
                }
                /* Track alias → module mapping */
                if (item->alias && item->module && strcmp(item->alias, item->module) != 0) {
                    if (tc->alias_count >= tc->alias_cap) {
                        tc->alias_cap = tc->alias_cap ? tc->alias_cap * 2 : 8;
                        tc->alias_names = realloc(tc->alias_names, sizeof(const char *) * tc->alias_cap);
                        tc->alias_modules = realloc(tc->alias_modules, sizeof(const char *) * tc->alias_cap);
                    }
                    tc->alias_names[tc->alias_count] = item->alias;
                    tc->alias_modules[tc->alias_count] = item->module;
                    tc->alias_count++;
                }
            }
        }
    }

    /* Pass 2a: Register enums BEFORE structs so that struct field types
     * referencing enums resolve correctly via tc_type_from_name(). Without
     * this ordering guarantee, a struct field typed as an enum (e.g. `color
     * Color`) resolves as TK_STRUCT instead of TK_ENUM when the struct
     * appears before the enum in the merged AST (common with imports). */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_ENUM_DECL) continue;

        /* E2038: reserved name for enums */
        const char *en = stmt->data.enum_decl.name;
        if (is_reserved_type_name(en)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg), "'%s' is a reserved type name and cannot be used as an enum name", en);
            diag_error_msg(tc->diag, "E2038", strdup(msg), NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        /* E2016: empty enum */
        if (stmt->data.enum_decl.value_count == 0) {
            diag_error_codef(tc->diag, "E2016", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.enum_decl.name);
        }
        /* E3033: check for duplicate enum values */
        for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
            if (!stmt->data.enum_decl.values[j].value) continue;
            for (int k = 0; k < j; k++) {
                if (!stmt->data.enum_decl.values[k].value) continue;
                if (stmt->data.enum_decl.values[j].value->kind == NODE_INT_VALUE &&
                    stmt->data.enum_decl.values[k].value->kind == NODE_INT_VALUE &&
                    stmt->data.enum_decl.values[j].value->data.int_value.value ==
                    stmt->data.enum_decl.values[k].value->data.int_value.value) {
                    diag_error_codef(tc->diag, "E3033", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.enum_decl.name,
                        stmt->data.enum_decl.values[k].name,
                        stmt->data.enum_decl.values[j].name);
                    break;
                }
            }
        }
        /* E2014: check for duplicate enum variant names */
        /* E2065: check variant name vs enum type name */
        for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
            if (strcmp(stmt->data.enum_decl.values[j].name,
                       stmt->data.enum_decl.name) == 0) {
                diag_error_codef(tc->diag, "E2065", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.enum_decl.values[j].name,
                    stmt->data.enum_decl.name);
            }
            for (int k = 0; k < j; k++) {
                if (strcmp(stmt->data.enum_decl.values[k].name,
                          stmt->data.enum_decl.values[j].name) == 0) {
                    diag_error_codef(tc->diag, "E2014", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.enum_decl.values[j].name,
                        stmt->data.enum_decl.name);
                    break;
                }
            }
        }
        /* Detect string enum (auto-detect from values) */
        bool is_str = false;
        for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
            if (stmt->data.enum_decl.values[j].value &&
                stmt->data.enum_decl.values[j].value->kind == NODE_STRING_VALUE) {
                is_str = true;
                break;
            }
        }
        /* E4007: duplicate enum name */
        if (is_enum_name(tc, stmt->data.enum_decl.name) ||
            is_struct_name(tc, stmt->data.enum_decl.name)) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "a type named '%s' is already declared",
                stmt->data.enum_decl.name);
            diag_error_msg(tc->diag, "E4007", strdup(msg),
                NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        int vc = stmt->data.enum_decl.value_count;
        const char **vnames = malloc(sizeof(const char *) * (vc ? vc : 1));
        for (int j = 0; j < vc; j++) {
            vnames[j] = stmt->data.enum_decl.values[j].name;
        }
        register_enum(tc, stmt->data.enum_decl.name, is_str, vnames, vc);
    }

    /* Pass 2b: Register structs and functions (enums already registered above) */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];

        if (stmt->kind == NODE_STRUCT_DECL) {
            /* E2067: empty struct */
            if (stmt->data.struct_decl.field_count == 0) {
                diag_error_codef(tc->diag, "E2067", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.struct_decl.name);
            }
            int fc = stmt->data.struct_decl.field_count;
            const char **fnames = malloc(sizeof(const char *) * (fc ? fc : 1));
            EzType **ftypes = malloc(sizeof(EzType *) * (fc ? fc : 1));
            if (!fnames || !ftypes) { free(fnames); free(ftypes); continue; }
            for (int j = 0; j < fc; j++) {
                fnames[j] = stmt->data.struct_decl.fields[j].name;
                ftypes[j] = tc_type_from_name(tc, stmt->data.struct_decl.fields[j].type_name);
                /* E3038: void field type */
                if (stmt->data.struct_decl.fields[j].type_name &&
                    strcmp(stmt->data.struct_decl.fields[j].type_name, "void") == 0) {
                    char msg[EZ_MSG_BUF_SIZE];
                    snprintf(msg, sizeof(msg),
                        "'void' cannot be used as a struct field type (field '%s')",
                        fnames[j]);
                    diag_error_msg(tc->diag, "E3038", strdup(msg),
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
                /* E2066: field name matches struct type name */
                if (strcmp(fnames[j], stmt->data.struct_decl.name) == 0) {
                    diag_error_codef(tc->diag, "E2066", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, fnames[j], stmt->data.struct_decl.name);
                }
                /* E3061 (): struct field cannot be the enclosing
                 * struct by value; that produces an infinite-size
                 * type. Direct self-reference or transitive cycles
                 * through other by-value struct fields both count.
                 * Pointer/array/map fields are fine because they're
                 * heap-indirected and have a finite header size. */
                const char *ftn = stmt->data.struct_decl.fields[j].type_name;
                if (ftn && *ftn && ftn[0] != '^' && ftn[0] != '[' &&
                    strncmp(ftn, "map[", 4) != 0) {
                    bool is_cycle = false;
                    const char *self_name = stmt->data.struct_decl.name;
                    if (strcmp(ftn, self_name) == 0) {
                        is_cycle = true;
                    } else {
                        AstNode *child = find_struct_in_program(program, ftn);
                        if (child) {
                            const char *visited[EZ_MAX_STRUCT_DEPTH];
                            int vc = 0;
                            is_cycle = struct_contains_by_value(
                                program, child, self_name, visited, &vc, 32);
                        }
                    }
                    if (is_cycle) {
                        char msg[EZ_MSG_BUF_SIZE];
                        if (strcmp(ftn, self_name) == 0) {
                            snprintf(msg, sizeof(msg),
                                "struct '%s' cannot contain itself by value; use a pointer field '^%s' for recursive types",
                                self_name, self_name);
                        } else {
                            snprintf(msg, sizeof(msg),
                                "struct '%s' cannot contain itself by value through '%s'; break the cycle with a pointer field '^%s'",
                                self_name, ftn, ftn);
                        }
                        diag_error_msg(tc->diag, "E3061", strdup(msg),
                            NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                    }
                }
                /* Check for duplicate field names */
                for (int k = 0; k < j; k++) {
                    if (strcmp(fnames[k], fnames[j]) == 0) {
                        diag_error_codef(tc->diag, "E2013", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, fnames[j], stmt->data.struct_decl.name);
                        break;
                    }
                }
            }
            /* E2037/E2038: reserved name check for structs */
            const char *sn = stmt->data.struct_decl.name;
            if (is_reserved_type_name(sn)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg), "'%s' is a reserved type name and cannot be used as a struct name", sn);
                diag_error_msg(tc->diag, "E2037", strdup(msg), NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E4007: duplicate struct name */
            if (is_struct_name(tc, stmt->data.struct_decl.name) ||
                is_enum_name(tc, stmt->data.struct_decl.name)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "a type named '%s' is already declared",
                    stmt->data.struct_decl.name);
                diag_error_msg(tc->diag, "E4007", strdup(msg),
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            register_struct(tc, stmt->data.struct_decl.name, fnames, ftypes, fc);

            /* : detect generic structs (any field with ? in type) */
            stmt->data.struct_decl.is_generic = false;
            stmt->data.struct_decl.instantiations = NULL;
            stmt->data.struct_decl.instantiation_count = 0;
            for (int j = 0; j < fc; j++) {
                if (stmt->data.struct_decl.fields[j].type_name &&
                    strchr(stmt->data.struct_decl.fields[j].type_name, '?')) {
                    stmt->data.struct_decl.is_generic = true;
                    break;
                }
            }

            /* Register struct-namespaced functions as StructName_funcName */
            for (int j = 0; j < stmt->data.struct_decl.func_count; j++) {
                AstNode *fn = stmt->data.struct_decl.funcs[j].func_decl;
                if (!fn || fn->kind != NODE_FUNC_DECL) continue;
                /* E2037: check for duplicate function names in struct */
                for (int k = 0; k < j; k++) {
                    AstNode *prev = stmt->data.struct_decl.funcs[k].func_decl;
                    if (prev && prev->kind == NODE_FUNC_DECL &&
                        strcmp(prev->data.func_decl.name, fn->data.func_decl.name) == 0) {
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "duplicate function '%s' in struct '%s'",
                            fn->data.func_decl.name, stmt->data.struct_decl.name);
                        diag_error_msg(tc->diag, "E2037", strdup(msg),
                            NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0);
                        break;
                    }
                }
                /* E2064: function name conflicts with field name */
                for (int k = 0; k < fc; k++) {
                    if (strcmp(fnames[k], fn->data.func_decl.name) == 0) {
                        diag_error_codef(tc->diag, "E2064", NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0, fn->data.func_decl.name, fnames[k],
                            stmt->data.struct_decl.name);
                        break;
                    }
                }
                int pc = fn->data.func_decl.param_count;
                EzType **ptypes = malloc(sizeof(EzType *) * (pc ? pc : 1));
                if (!ptypes) continue;
                for (int k = 0; k < pc; k++) {
                    ptypes[k] = tc_type_from_name(tc, fn->data.func_decl.params[k].type_name);
                }
                int rc = fn->data.func_decl.return_type_count;
                EzType **rtypes = malloc(sizeof(EzType *) * (rc ? rc : 1));
                if (!rtypes) { free(ptypes); continue; }
                for (int k = 0; k < rc; k++) {
                    rtypes[k] = tc_type_from_name(tc, fn->data.func_decl.return_types[k]);
                }
                /* Register with prefixed name: StructName_funcName */
                char *prefixed = malloc(strlen(stmt->data.struct_decl.name) +
                    strlen(fn->data.func_decl.name) + 2);
                sprintf(prefixed, "%s_%s", stmt->data.struct_decl.name, fn->data.func_decl.name);
                register_func(tc, prefixed, ptypes, pc, rtypes, rc);
                tc->funcs[tc->func_count - 1].is_private = fn->data.func_decl.is_private;
                finalize_generic_sig(&tc->funcs[tc->func_count - 1], fn);
            }
        }

        /* Enums already processed in pass 2a above */

        if (stmt->kind == NODE_FUNC_DECL) {
            int pc = stmt->data.func_decl.param_count;
            EzType **ptypes = malloc(sizeof(EzType *) * (pc ? pc : 1));
            if (!ptypes) continue;
            for (int j = 0; j < pc; j++) {
                ptypes[j] = tc_type_from_name(tc, stmt->data.func_decl.params[j].type_name);
                tc_mark_type_module_used(tc, stmt->data.func_decl.params[j].type_name);
            }

            int rc = stmt->data.func_decl.return_type_count;
            EzType **rtypes = malloc(sizeof(EzType *) * (rc ? rc : 1));
            if (!rtypes) { free(ptypes); continue; }
            for (int j = 0; j < rc; j++) {
                rtypes[j] = tc_type_from_name(tc, stmt->data.func_decl.return_types[j]);
                tc_mark_type_module_used(tc, stmt->data.func_decl.return_types[j]);
            }

            /* E4008: main() cannot have parameters or return types */
            if (strcmp(stmt->data.func_decl.name, "main") == 0) {
                if (pc > 0) {
                    diag_error_msg(tc->diag, "E4008",
                        strdup("'main' function cannot have parameters; main() takes no arguments"),
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
                if (rc > 0) {
                    diag_error_msg(tc->diag, "E4008",
                        strdup("'main' function cannot have a return type; main() always returns void"),
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
            }
            /* Check for reserved prefix */
            check_reserved_name(tc, stmt->data.func_decl.name,
                NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column);
            /* Check for duplicate function names */
            if (find_func(tc, stmt->data.func_decl.name)) {
                diag_error_codef(tc->diag, "E4004", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, stmt->data.func_decl.name);
            }
            /* E4007: function name conflicts with a type */
            if (is_struct_name(tc, stmt->data.func_decl.name) ||
                is_enum_name(tc, stmt->data.func_decl.name)) {
                char msg[EZ_MSG_BUF_SIZE];
                snprintf(msg, sizeof(msg),
                    "function '%s' conflicts with a type of the same name",
                    stmt->data.func_decl.name);
                diag_error_msg(tc->diag, "E4007", strdup(msg),
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            register_func(tc, stmt->data.func_decl.name, ptypes, pc, rtypes, rc);
            tc->funcs[tc->func_count - 1].is_private = stmt->data.func_decl.is_private;
            /* Store line for unused function warning */
            tc->funcs[tc->func_count - 1].def_line = stmt->token.line;
            finalize_generic_sig(&tc->funcs[tc->func_count - 1], stmt);
            /* Also register unprefixed name for internal cross-references.
             * If name is "lib_foo", also register "foo" so calls within imported
             * function bodies can resolve unprefixed names.
             * Skip "main" — it's the entry point and must not collide. */
            const char *fn = stmt->data.func_decl.name;
            const char *underscore = strchr(fn, '_');
            if (underscore && underscore != fn) {
                const char *unprefixed = underscore + 1;
                if (strcmp(unprefixed, "main") != 0) {
                    /* Only register if the prefix matches a known import */
                    for (int ii = 0; ii < tc->import_count; ii++) {
                        size_t mod_len = strlen(tc->imported_modules[ii]);
                        if (strncmp(fn, tc->imported_modules[ii], mod_len) == 0 &&
                            fn[mod_len] == '_') {
                            register_func(tc, unprefixed, ptypes, pc, rtypes, rc);
                            tc->funcs[tc->func_count - 1].is_private = stmt->data.func_decl.is_private;
                            tc->funcs[tc->func_count - 1].def_line = 0; /* suppress unused warning */
                            finalize_generic_sig(&tc->funcs[tc->func_count - 1], stmt);
                            break;
                        }
                    }
                }
            }
        }
    }
    tc->registering = false;
}

/* --- Public API --- */

TypeChecker *typechecker_create(DiagnosticList *diag, const char *file) {
    TypeChecker *tc = calloc(1, sizeof(TypeChecker));
    tc->diag = diag;
    tc->file = file;
    tc->current_scope = scope_create(NULL);
    tc->type_table = typetable_create();
    return tc;
}

void typechecker_check(TypeChecker *tc, AstNode *program) {
    if (!program || program->kind != NODE_PROGRAM) return;

    tc->program = program;

    /* Pass 1: register all type/function declarations */
    register_declarations(tc, program);

    /* Register stdlib struct types scoped to their module imports */
    if (tc_is_imported_module(tc, "server") || tc_is_imported_module(tc, "http")) {
        static const char *resp_fields[] = {"status", "body", "headers"};
        static EzType *resp_types[3];
        resp_types[0] = &TYPE_INT;
        resp_types[1] = &TYPE_STRING;
        resp_types[2] = type_from_name("map[string:string]");
        register_struct(tc, "HttpResponse", resp_fields, resp_types, 3);
    }

    if (tc_is_imported_module(tc, "server")) {
        static const char *req_fields[] = {"method", "path", "body", "query", "headers", "params"};
        static EzType *req_types[6];
        req_types[0] = &TYPE_STRING;
        req_types[1] = &TYPE_STRING;
        req_types[2] = &TYPE_STRING;
        req_types[3] = type_from_name("map[string:string]");
        req_types[4] = type_from_name("map[string:string]");
        req_types[5] = type_from_name("map[string:string]");
        register_struct(tc, "HttpRequest", req_fields, req_types, 6);
    }

    /* Collect 'using' and 'import and use' module names */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_USING_STMT) {
            for (int j = 0; j < stmt->data.using_stmt.count; j++) {
                /* E2010: check that the module was imported BEFORE this using statement */
                const char *umod = stmt->data.using_stmt.modules[j];
                bool imported_before = false;
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], umod) == 0 &&
                        tc->import_lines[mi] < stmt->token.line) {
                        imported_before = true;
                        break;
                    }
                }
                if (!imported_before) {
                    diag_error_codef(tc->diag, "E2010", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, umod, umod);
                }
                if (tc->using_module_count >= tc->using_module_cap) {
                    tc->using_module_cap = tc->using_module_cap ? tc->using_module_cap * 2 : 8;
                    tc->using_modules = realloc(tc->using_modules,
                        sizeof(const char *) * (size_t)tc->using_module_cap);
                }
                tc->using_modules[tc->using_module_count++] = stmt->data.using_stmt.modules[j];
                /* Mark the module as used */
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], stmt->data.using_stmt.modules[j]) == 0) {
                        tc->import_used[mi] = true;
                        break;
                    }
                }
            }
        }
        if (stmt->kind == NODE_IMPORT_STMT && stmt->data.import_stmt.auto_use) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->module) {
                    if (tc->using_module_count >= tc->using_module_cap) {
                        tc->using_module_cap = tc->using_module_cap ? tc->using_module_cap * 2 : 8;
                        tc->using_modules = realloc(tc->using_modules,
                            sizeof(const char *) * (size_t)tc->using_module_cap);
                    }
                    tc->using_modules[tc->using_module_count++] = item->module;
                }
            }
        }
    }

    /* Register unprefixed aliases for struct/enum types from 'import and use' modules */
    for (int ui = 0; ui < tc->using_module_count; ui++) {
        const char *umod = tc->using_modules[ui];
        size_t umod_len = strlen(umod);
        char prefix[EZ_TYPE_NAME_MAX];
        snprintf(prefix, sizeof(prefix), "%s_", umod);
        size_t prefix_len = umod_len + 1;
        /* Check structs */
        for (int si = 0; si < tc->struct_count; si++) {
            const char *sn = tc->structs[si].struct_name;
            if (strncmp(sn, prefix, prefix_len) == 0) {
                const char *unprefixed = sn + prefix_len;
                if (!is_struct_name(tc, unprefixed)) {
                    register_struct(tc, unprefixed,
                        tc->structs[si].field_names,
                        tc->structs[si].field_types,
                        tc->structs[si].field_count);
                }
            }
        }
        /* Check enums */
        for (int ei = 0; ei < tc->enum_count; ei++) {
            const char *en = tc->enum_names[ei];
            if (strncmp(en, prefix, prefix_len) == 0) {
                const char *unprefixed = en + prefix_len;
                if (!is_enum_name(tc, unprefixed)) {
                    register_enum(tc, unprefixed, tc->enum_is_string[ei],
                        tc->enum_values[ei], tc->enum_value_counts[ei]);
                }
            }
        }
        /* Register unprefixed aliases for struct-namespaced functions.
         * e.g., testmod_Hero_attack → Hero_attack so Hero.attack() works unprefixed */
        int current_func_count = tc->func_count;
        for (int fi = 0; fi < current_func_count; fi++) {
            const char *fn = tc->funcs[fi].name;
            if (strncmp(fn, prefix, prefix_len) == 0) {
                const char *unprefixed = fn + prefix_len;
                /* Only register if it's a struct-namespaced function (StructName_func)
                 * and not already registered */
                const char *inner_us = strchr(unprefixed, '_');
                if (inner_us && unprefixed[0] >= 'A' && unprefixed[0] <= 'Z' &&
                    !find_func(tc, unprefixed)) {
                    register_func(tc, unprefixed,
                        tc->funcs[fi].param_types,
                        tc->funcs[fi].param_count,
                        tc->funcs[fi].return_types,
                        tc->funcs[fi].return_count);
                    tc->funcs[tc->func_count - 1].def_line = 0;
                }
            }
        }
    }

    /* Pass 2: check all statements */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        check_statement(tc, program->data.program.stmts[i]);
    }

    /* Pass 3: re-check generic function bodies per instantiation
     * ( slice 4). The main pass walked bodies with '?' as
     * TK_UNKNOWN, so operations the concrete type doesn't support
     * (e.g. `a + b` on strings) slipped through. For every recorded
     * instantiation, rebind the parameters to concrete types and
     * re-run check_block on the body. If any new errors fire, emit
     * a companion diagnostic at the originating call site so the
     * user sees "I asked for this specialisation, and here's what
     * broke inside it". */
    for (int fi = 0; fi < tc->func_count; fi++) {
        FuncSig *fs = &tc->funcs[fi];
        if (!fs->is_generic || !fs->decl ||
            fs->decl->kind != NODE_FUNC_DECL ||
            fs->instantiation_count == 0) continue;

        AstNode *decl = fs->decl;
        for (int ii = 0; ii < fs->instantiation_count; ii++) {
            const char *concrete = fs->instantiations[ii];
            AstNode *call_site = fs->instantiation_calls[ii];

            Scope *inst_scope = scope_create(tc->current_scope);
            Scope *outer_scope = tc->current_scope;
            tc->current_scope = inst_scope;
            tc->func_depth++;

            for (int pi = 0; pi < decl->data.func_decl.param_count; pi++) {
                Param *p = &decl->data.func_decl.params[pi];
                char *sub = substitute_wildcard(p->type_name, concrete);
                EzType *pt = sub ? type_from_name(sub) : &TYPE_UNKNOWN;
                scope_define(inst_scope, p->name, pt, p->mutable);
                /* `sub` leaks on purpose; type_from_name stores the
                 * name pointer for array/map kinds and we need it
                 * alive for the duration of the re-check. Compile-
                 * time allocation; short-lived process. */
            }

            EzType **prev_ret = tc->current_return_types;
            const char **prev_ret_names = tc->current_return_type_names;
            int prev_ret_count = tc->current_return_count;
            bool prev_named = tc->current_has_named_returns;
            const char **prev_return_names = tc->current_return_names;

            int rc = decl->data.func_decl.return_type_count;
            EzType **ret_types = NULL;
            const char **ret_names = NULL;
            if (rc > 0) {
                ret_types = malloc(sizeof(EzType *) * (size_t)rc);
                ret_names = malloc(sizeof(const char *) * (size_t)rc);
                for (int ri = 0; ri < rc; ri++) {
                    char *sub = substitute_wildcard(
                        decl->data.func_decl.return_types[ri], concrete);
                    ret_types[ri] = sub ? type_from_name(sub) : &TYPE_UNKNOWN;
                    ret_names[ri] = decl->data.func_decl.return_types[ri];
                }
            }
            tc->current_return_types = ret_types;
            tc->current_return_type_names = ret_names;
            tc->current_return_count = rc;
            tc->current_return_names = decl->data.func_decl.return_names;

            /* Detect and define named return variables in instantiation scope */
            tc->current_has_named_returns = false;
            if (decl->data.func_decl.return_names) {
                for (int ri = 0; ri < rc; ri++) {
                    if (decl->data.func_decl.return_names[ri]) {
                        tc->current_has_named_returns = true;
                    }
                }
            }

            int errs_before = diag_error_count(tc->diag);
            tc->suppress_typetable_writes = true;
            if (decl->data.func_decl.body) {
                check_block(tc, decl->data.func_decl.body);
            }
            tc->suppress_typetable_writes = false;
            int errs_after = diag_error_count(tc->diag);

            tc->current_return_types = prev_ret;
            tc->current_return_type_names = prev_ret_names;
            tc->current_return_count = prev_ret_count;
            tc->current_has_named_returns = prev_named;
            tc->current_return_names = prev_return_names;
            tc->current_scope = outer_scope;
            tc->func_depth--;
            free(ret_types);
            free(ret_names);

            if (errs_after > errs_before && call_site) {
                diag_error_codef(tc->diag, "E3058", NODE_FILE(tc, call_site), call_site->token.line, call_site->token.column, 0, fs->name, concrete);
            }
        }
    }

    /* Verify main() exists */
    if (!find_func(tc, "main")) {
        /* Point at the last statement or line 1 if empty */
        int err_line = 1;
        if (program->data.program.stmt_count > 0) {
            AstNode *last = program->data.program.stmts[program->data.program.stmt_count - 1];
            if (last) err_line = last->token.line;
        }
        diag_error_msg(tc->diag, "E4005",
            strdup("program has no main() function; every program needs 'do main() { }'"),
            tc->file, err_line, 1, 0);
    }

    /* Warn about unused imports */
    for (int i = 0; i < tc->import_count; i++) {
        if (!tc->import_used[i]) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "module '%s' is imported but never used; remove the import or use the module",
                tc->imported_modules[i]);
            diag_warning_msg(tc->diag, "W1002", strdup(msg),
                tc->file, tc->import_lines[i], 1, 0);
        }
    }

    /* Warn about unused functions (skip main and struct-namespaced) */
    for (int i = 0; i < tc->func_count; i++) {
        FuncSig *fs = &tc->funcs[i];
        if (!fs->used && fs->def_line > 0 &&
            strcmp(fs->name, "main") != 0 &&
            !fs->is_private &&
            !(fs->name[0] >= 'A' && fs->name[0] <= 'Z' && strchr(fs->name, '_'))) {
            char msg[EZ_MSG_BUF_SIZE];
            snprintf(msg, sizeof(msg),
                "function '%s' is declared but never called", fs->name);
            diag_warning_msg(tc->diag, "W1003", strdup(msg),
                tc->file, fs->def_line, 1, 0);
        }
    }
}

TypeTable *typechecker_get_table(TypeChecker *tc) {
    return tc->type_table;
}
