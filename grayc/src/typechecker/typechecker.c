/*
 * typechecker.c — Walks the AST to resolve expression types, enforce type
 * correctness, and build a type table that codegen can query at emission time.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "typechecker.h"
#include "../util/constants.h"
#include "../util/reserved.h"
#include "../util/xalloc.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdarg.h>
#include <ctype.h>

#define MAX_STRUCT_DEPTH 32

/* Helper: get the source file from an AST node's token, falling back to tc->file.
 * Imported nodes carry their original file path in token.file; main-file nodes have NULL. */
#define NODE_FILE(tc, n) ((n)->token.file ? (n)->token.file : (tc)->file)

/* Check if using_modules[ui] is accessible from the current file being checked.
 * A using-module entry is accessible only if it was declared in the same file
 * that is currently being validated (prevents transitive import type leaking). */
static inline bool using_module_accessible(TypeChecker *tc, int ui) {
    const char *using_file = tc->using_module_files ? tc->using_module_files[ui] : NULL;
    const char *check_file = tc->current_check_file;
    /* Both NULL — both from main file context */
    if (!using_file && !check_file) return true;
    /* Both set — compare paths */
    if (using_file && check_file && strcmp(using_file, check_file) == 0) return true;
    return false;
}

/* Helper: get the user-facing display name for a declaration.
 * Import merging prefixes names (e.g. foo → mod_foo) but errors should
 * show the original name the user wrote. */
#define FUNC_DISPLAY_NAME(n) ((n)->data.func_decl.original_name ? (n)->data.func_decl.original_name : (n)->data.func_decl.name)
#define VAR_DISPLAY_NAME(n)  ((n)->data.var_decl.original_name  ? (n)->data.var_decl.original_name  : (n)->data.var_decl.name)
#define STRUCT_DISPLAY_NAME(n) ((n)->data.struct_decl.original_name ? (n)->data.struct_decl.original_name : (n)->data.struct_decl.name)
#define ENUM_DISPLAY_NAME(n) ((n)->data.enum_decl.original_name ? (n)->data.enum_decl.original_name : (n)->data.enum_decl.name)

/* --- Type Table (open-addressing hash, pointer keys) --- */

static uint32_t hash_ptr(const void *ptr) {
    uintptr_t v = (uintptr_t)ptr;
    /* Fibonacci hashing; good distribution for pointer alignment */
    v = ((v >> 4) ^ v) * 0x9E3779B9U;
    return (uint32_t)(v ^ (v >> 16));
}

TypeTable *typetable_create(void) {
    TypeTable *tt = xcalloc(1, sizeof(TypeTable));
    tt->cap = TYPETABLE_INIT_CAP;
    tt->nodes = xcalloc((size_t)tt->cap, sizeof(AstNode *));
    tt->types = xcalloc((size_t)tt->cap, sizeof(GrayType *));
    return tt;
}

static void typetable_grow(TypeTable *tt) {
    int old_cap = tt->cap;
    AstNode **old_nodes = tt->nodes;
    GrayType **old_types = tt->types;

    tt->cap = old_cap * 2;
    tt->nodes = xcalloc((size_t)tt->cap, sizeof(AstNode *));
    tt->types = xcalloc((size_t)tt->cap, sizeof(GrayType *));
    tt->count = 0;

    for (int i = 0; i < old_cap; i++) {
        if (old_nodes[i]) {
            typetable_set(tt, old_nodes[i], old_types[i]);
        }
    }
    free(old_nodes);
    free(old_types);
}

void typetable_set(TypeTable *tt, AstNode *node, GrayType *type) {
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

GrayType *typetable_get(TypeTable *tt, AstNode *node) {
    if (!tt || !tt->nodes) return NULL;

    uint32_t mask = (uint32_t)(tt->cap - 1);
    uint32_t idx = hash_ptr(node) & mask;

    for (;;) {
        if (!tt->nodes[idx]) return NULL;
        if (tt->nodes[idx] == node) return tt->types[idx];
        idx = (idx + 1) & mask;
    }
}

/* Shared helpers for sorted-string-set lookups. */
static int strptr_cmp(const void *a, const void *b) {
    return strcmp(*(const char *const *)a, *(const char *const *)b);
}
static bool strset_contains(const char *const *sorted, int n, const char *name) {
    return bsearch(&name, sorted, (size_t)n, sizeof(const char *), strptr_cmp) != NULL;
}


/* Forward declaration — resolve_expr is defined later but needed by helper
 * routines and stdlib arg-type validation. */
static GrayType *resolve_expr(TypeChecker *tc, AstNode *node);

/* Return the user-facing display string for an operator TokenType.
 * Used in error messages that embed the operator name. */
static const char *op_to_str(TokenType op) {
    switch (op) {
    case TOK_PLUS: return "+";
    case TOK_MINUS: return "-";
    case TOK_ASTERISK: return "*";
    case TOK_SLASH: return "/";
    case TOK_PERCENT: return "%";
    case TOK_EQ: return "==";
    case TOK_NOT_EQ: return "!=";
    case TOK_LT: return "<";
    case TOK_GT: return ">";
    case TOK_LT_EQ: return "<=";
    case TOK_GT_EQ: return ">=";
    case TOK_AND: return "&&";
    case TOK_OR: return "||";
    case TOK_BANG: return "!";
    case TOK_BIT_AND: return "bit_and";
    case TOK_BIT_OR: return "bit_or";
    case TOK_BIT_XOR: return "bit_xor";
    case TOK_BIT_NOT: return "bit_not";
    case TOK_BIT_SHIFT_LEFT: return "bit_shift_left";
    case TOK_BIT_SHIFT_RIGHT: return "bit_shift_right";
    case TOK_INCREMENT: return "++";
    case TOK_DECREMENT: return "--";
    case TOK_CARET: return "^";
    case TOK_ASSIGN: return "=";
    case TOK_PLUS_ASSIGN: return "+=";
    case TOK_MINUS_ASSIGN: return "-=";
    case TOK_ASTERISK_ASSIGN: return "*=";
    case TOK_SLASH_ASSIGN: return "/=";
    case TOK_PERCENT_ASSIGN: return "%=";
    case TOK_IN: return "in";
    case TOK_NOT_IN: return "not_in";
    default: return "?";
    }
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
        return e->data.postfix.op == TOK_CARET;
    default:                return false;
    }
}

/* Walk an lvalue expression chain and return the root variable name.
 * Returns NULL if the chain passes through a pointer dereference (^),
 * because the pointed-to memory is independent of the pointer variable's
 * mutability. */
static const char *lvalue_root_name(AstNode *e) {
    if (!e) return NULL;
    switch (e->kind) {
    case NODE_LABEL:       return e->data.label.value;
    case NODE_MEMBER_EXPR: return lvalue_root_name(e->data.member.object);
    case NODE_INDEX_EXPR:  return lvalue_root_name(e->data.index_expr.left);
    case NODE_POSTFIX_EXPR:
        if (e->data.postfix.op == TOK_CARET)
            return NULL; /* pointer deref: memory is not the const variable */
        return NULL;
    default: return NULL;
    }
}

/* True if the access path contains a map index, e.g. ref(m["k"]) or
 * ref(m["k"].field) or ref(rows[i].cells["k"]). Map values relocate on
 * rehash so a pointer to one is unsafe. */
static bool path_contains_map_index(TypeChecker *tc, AstNode *e) {
    if (!e) return false;
    switch (e->kind) {
    case NODE_MEMBER_EXPR:
        return path_contains_map_index(tc, e->data.member.object);
    case NODE_INDEX_EXPR: {
        GrayType *left_t = resolve_expr(tc, e->data.index_expr.left);
        if (left_t && left_t->kind == TK_MAP) return true;
        return path_contains_map_index(tc, e->data.index_expr.left);
    }
    case NODE_POSTFIX_EXPR:
        return path_contains_map_index(tc, e->data.postfix.left);
    default:
        return false;
    }
}

/* --- Struct info helpers --- */

static void register_struct(TypeChecker *tc, const char *name,
    const char *display_name,
    const char **field_names, GrayType **field_types, int field_count) {
    if (tc->struct_count >= tc->struct_cap) {
        tc->struct_cap = tc->struct_cap ? tc->struct_cap * 2 : 8;
        tc->structs = xrealloc(tc->structs, sizeof(StructInfo) * tc->struct_cap);
    }
    tc->structs_sorted_built = false;
    StructInfo *si = &tc->structs[tc->struct_count++];
    si->struct_name = name;
    si->display_name = display_name ? display_name : name;
    si->field_names = field_names;
    si->field_types = field_types;
    si->field_count = field_count;
}

static int structinfo_name_cmp(const void *a, const void *b) {
    return strcmp((*(const StructInfo *const *)a)->struct_name,
                  (*(const StructInfo *const *)b)->struct_name);
}

static StructInfo *find_struct(TypeChecker *tc, const char *name) {
    if (tc->struct_count == 0) return NULL;
    if (!tc->structs_sorted_built) {
        tc->structs_sorted = xrealloc(tc->structs_sorted,
            sizeof(StructInfo *) * (size_t)tc->struct_count);
        for (int i = 0; i < tc->struct_count; i++)
            tc->structs_sorted[i] = &tc->structs[i];
        qsort(tc->structs_sorted, (size_t)tc->struct_count,
              sizeof(StructInfo *), structinfo_name_cmp);
        tc->structs_sorted_built = true;
    }
    StructInfo key = { .struct_name = name };
    StructInfo *key_ptr = &key;
    StructInfo **hit = bsearch(&key_ptr, tc->structs_sorted,
        (size_t)tc->struct_count, sizeof(StructInfo *), structinfo_name_cmp);
    return hit ? *hit : NULL;
}

static bool is_struct_name(TypeChecker *tc, const char *name) {
    return find_struct(tc, name) != NULL;
}

static int enum_name_str_cmp(const void *a, const void *b) {
    return strcmp(*(const char *const *)a, *(const char *const *)b);
}

static void enum_ensure_sorted(TypeChecker *tc) {
    if (tc->enum_names_sorted_built) return;
    tc->enum_names_sorted = xrealloc(tc->enum_names_sorted,
        sizeof(const char *) * (size_t)tc->enum_count);
    tc->enum_names_sorted_indices = xrealloc(tc->enum_names_sorted_indices,
        sizeof(int) * (size_t)tc->enum_count);
    for (int i = 0; i < tc->enum_count; i++)
        tc->enum_names_sorted[i] = tc->enum_names[i];
    qsort(tc->enum_names_sorted, (size_t)tc->enum_count,
          sizeof(const char *), enum_name_str_cmp);
    /* Build reverse mapping from sorted position to original index */
    for (int s = 0; s < tc->enum_count; s++) {
        for (int i = 0; i < tc->enum_count; i++) {
            if (tc->enum_names[i] == tc->enum_names_sorted[s]) {
                tc->enum_names_sorted_indices[s] = i;
                break;
            }
        }
    }
    tc->enum_names_sorted_built = true;
}

/* Returns the original index of the named enum via O(log n) bsearch, or -1. */
static int find_enum_index(TypeChecker *tc, const char *name) {
    if (tc->enum_count == 0) return -1;
    enum_ensure_sorted(tc);
    const char **hit = bsearch(&name, tc->enum_names_sorted,
        (size_t)tc->enum_count, sizeof(const char *), enum_name_str_cmp);
    if (!hit) return -1;
    return tc->enum_names_sorted_indices[hit - tc->enum_names_sorted];
}

static bool is_enum_name(TypeChecker *tc, const char *name) {
    if (tc->enum_count == 0) return false;
    enum_ensure_sorted(tc);
    return bsearch(&name, tc->enum_names_sorted, (size_t)tc->enum_count,
                   sizeof(const char *), enum_name_str_cmp) != NULL;
}

/* The name the programmer wrote for a struct, never the module-prefixed
 * lookup key. Diagnostics and namespace-collision checks must use this. */
static const char *struct_display_name(TypeChecker *tc, const char *name) {
    StructInfo *si = find_struct(tc, name);
    return si ? si->display_name : name;
}

/* As struct_display_name, for enums. */
static const char *enum_display_name(TypeChecker *tc, const char *name) {
    int i = find_enum_index(tc, name);
    if (i < 0) return name;
    return tc->enum_display_names[i] ? tc->enum_display_names[i] : tc->enum_names[i];
}

/* Compare two type names by their user-facing display names, so that
 * module-prefixed internal names (e.g. lib_Foo vs objects_Foo) match
 * when they refer to the same user-defined type. */
static bool tc_same_struct_type(TypeChecker *tc, const char *a, const char *b) {
    if (a == b || strcmp(a, b) == 0) return true;
    return strcmp(struct_display_name(tc, a), struct_display_name(tc, b)) == 0;
}
static bool tc_same_enum_type(TypeChecker *tc, const char *a, const char *b) {
    if (a == b || strcmp(a, b) == 0) return true;
    return strcmp(enum_display_name(tc, a), enum_display_name(tc, b)) == 0;
}
/* Compare array element type names accounting for module-prefixed struct
 * aliases (e.g. "Item" vs "utils_Item" should match). */
static bool tc_same_array_elem(TypeChecker *tc, const char *a, const char *b) {
    if (strcmp(a, b) == 0) return true;
    GrayType *at = type_from_name(a);
    GrayType *bt = type_from_name(b);
    if (at && bt && at->kind == TK_STRUCT && bt->kind == TK_STRUCT)
        return tc_same_struct_type(tc, a, b);
    if (at && bt && at->kind == TK_ENUM && bt->kind == TK_ENUM)
        return tc_same_enum_type(tc, a, b);
    return false;
}

/* Returns true if the named enum is string-backed. */
static bool tc_enum_is_string(TypeChecker *tc, const char *name) {
    int i = find_enum_index(tc, name);
    return i >= 0 && tc->enum_is_string[i];
}

/* Returns true if the named enum is a tagged enum (has payload variants). */
static bool tc_enum_is_tagged(TypeChecker *tc, const char *name) {
    int i = find_enum_index(tc, name);
    return i >= 0 && tc->enum_is_tagged[i];
}

/* A type name fit to print in a diagnostic — for struct/enum types this
 * is the user-facing name, never the module-prefixed lookup key. Composite
 * types (pointers, arrays, maps) recurse into their inner types so that
 * e.g. ^myutils_Thing displays as ^Thing. */
static const char *type_display_name(TypeChecker *tc, GrayType *t) {
    if (!t) return type_name(t);
    if (t->kind == TK_STRUCT && t->name) return struct_display_name(tc, t->name);
    if (t->kind == TK_ENUM && t->name) return enum_display_name(tc, t->name);
    if (t->kind == TK_POINTER && t->name) {
        const char *inner = struct_display_name(tc, t->name);
        if (inner == t->name) inner = enum_display_name(tc, t->name);
        if (inner != t->name) {
            static char ptr_bufs[4][TYPE_NAME_MAX];
            static int ptr_slot = 0;
            char *out = ptr_bufs[ptr_slot];
            ptr_slot = (ptr_slot + 1) & 3;
            snprintf(out, sizeof(ptr_bufs[0]), "^%s", inner);
            return out;
        }
    }
    if (t->kind == TK_ARRAY && t->element_type) {
        const char *inner = struct_display_name(tc, t->element_type);
        if (inner == t->element_type) inner = enum_display_name(tc, t->element_type);
        if (inner != t->element_type) {
            static char arr_bufs[4][TYPE_NAME_MAX];
            static int arr_slot = 0;
            char *out = arr_bufs[arr_slot];
            arr_slot = (arr_slot + 1) & 3;
            snprintf(out, sizeof(arr_bufs[0]), "[%s]", inner);
            return out;
        }
    }
    return type_name(t);
}

/* Resolve an import alias to the actual module name */
static const char *tc_resolve_alias(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->alias_count; i++) {
        if (strcmp(tc->alias_names[i], name) == 0) return tc->alias_modules[i];
    }
    return name;
}

/* Format a diagnostic message string into the arena (only called in error paths). */
static char *tc_fmt(TypeChecker *tc, const char *fmt, ...) {
    char buf[MSG_BUF_SIZE];
    va_list ap;
    va_start(ap, fmt);
    vsnprintf(buf, sizeof(buf), fmt, ap);
    va_end(ap);
    return arena_strdup(tc->arena, buf);
}

static AstNode *find_struct_in_program(AstNode *program, const char *name);

static GrayType *struct_field_type(TypeChecker *tc, const char *struct_name, const char *field) {
    StructInfo *si = find_struct(tc, struct_name);
    /* : for mangled generic struct names (Pair__int), fall back
     * to the base name (Pair) since fields are registered there.
     * When the field type is "?", substitute the concrete binding
     * extracted from the mangled suffix. */
    const char *generic_binding = NULL;
    if (!si && struct_name) {
        const char *dunder = strstr(struct_name, "__");
        if (dunder) {
            char base[MSG_BUF_SIZE];
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
             * type_name since the resolved GrayType lost the "?" marker. */
            if (generic_binding && si->field_types[i]->kind == TK_UNKNOWN) {
                /* Find the raw struct decl to check the field type_name */
                if (tc->program) {
                    AstNode *decl = find_struct_in_program(tc->program,
                        struct_name); /* try mangled first */
                    if (!decl) {
                        /* Extract base name and try again */
                        const char *dd = strstr(struct_name, "__");
                        if (dd) {
                            char bname[MSG_BUF_SIZE];
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
    GrayType **param_types, int param_count,
    GrayType **return_types, int return_count) {
    if (tc->func_count >= tc->func_cap) {
        tc->func_cap = tc->func_cap ? tc->func_cap * 2 : 16;
        tc->funcs = xrealloc(tc->funcs, sizeof(FuncSig) * tc->func_cap);
    }
    tc->funcs_sorted_built = false;
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
    char *out = xmalloc(len + 1);
    char *w = out;
    for (const char *c = src; *c; c++) {
        if (*c == '?') { memcpy(w, concrete, cl); w += cl; }
        else *w++ = *c;
    }
    *w = '\0';
    return out;
}

/* Find the top-level ':' inside a "map[K:V]" type string, skipping nested
 * brackets. Sets key_out/key_len and val_out/val_len on success. */
static bool parse_map_keyval(const char *tn,
                              const char **key_out, size_t *key_len,
                              const char **val_out, size_t *val_len) {
    if (strncmp(tn, "map[", 4) != 0) return false;
    const char *start = tn + 4;
    int depth = 0;
    const char *colon = NULL;
    for (const char *c = start; *c && !(depth == 0 && *c == ']'); c++) {
        if      (*c == '[')                   depth++;
        else if (*c == ']')                   depth--;
        else if (*c == ':' && depth == 0) { colon = c; break; }
    }
    if (!colon) return false;
    *key_out = start;
    *key_len = (size_t)(colon - start);
    *val_out = colon + 1;
    const char *end = tn + strlen(tn) - 1;
    if (*end != ']') return false;
    *val_len = (size_t)(end - *val_out);
    return true;
}

/* Recursive string-based wildcard unifier.
 * Matches param_tn (containing '?') against the concrete type string
 * arg_tn and returns a heap-allocated string for the type '?' binds to,
 * or NULL on shape mismatch. Recurses into arrays and maps so nested
 * composites like [[?]], [map[string:?]], and map[string:[?]] are handled.
 *
 * Examples:
 *   "?"               vs "int"            -> "int"
 *   "[?]"             vs "[string]"       -> "string"
 *   "[[?]]"           vs "[[int]]"        -> "int"
 *   "[map[string:?]]" vs "[map[string:int]]" -> "int"
 *   "map[string:[?]]" vs "map[string:[int]]" -> "int"
 *   "map[?:?]"        vs "map[int:int]"   -> "int"    (K==V required)
 */
static char *bind_wildcard_str(const char *param_tn, const char *arg_tn) {
    if (!param_tn || !arg_tn) return NULL;

    if (strcmp(param_tn, "?") == 0) {
        if (strcmp(arg_tn, "unknown") == 0) return NULL;
        return strdup(arg_tn);
    }

    size_t plen = strlen(param_tn);
    size_t alen = strlen(arg_tn);

    /* Array: both must start/end with brackets; recurse into element types */
    if (param_tn[0] == '[' && arg_tn[0] == '[') {
        if (plen < 3 || alen < 3) return NULL;
        if (param_tn[plen - 1] != ']' || arg_tn[alen - 1] != ']') return NULL;
        char *p_inner = strndup(param_tn + 1, plen - 2);
        char *a_inner = strndup(arg_tn + 1, alen - 2);
        char *result = bind_wildcard_str(p_inner, a_inner);
        free(p_inner);
        free(a_inner);
        return result;
    }

    /* Map: both must be map types; recurse into whichever slot carries '?' */
    if (strncmp(param_tn, "map[", 4) == 0 && strncmp(arg_tn, "map[", 4) == 0) {
        const char *pk, *pv, *ak, *av;
        size_t pkl, pvl, akl, avl;
        if (!parse_map_keyval(param_tn, &pk, &pkl, &pv, &pvl)) return NULL;
        if (!parse_map_keyval(arg_tn,   &ak, &akl, &av, &avl)) return NULL;

        bool pk_wild = false;
        for (size_t i = 0; i < pkl; i++) if (pk[i] == '?') { pk_wild = true; break; }
        bool pv_wild = false;
        for (size_t i = 0; i < pvl; i++) if (pv[i] == '?') { pv_wild = true; break; }

        if (!pk_wild && !pv_wild) return NULL;

        /* Concrete slots must match the argument's corresponding slot exactly */
        if (!pk_wild && (akl != pkl || memcmp(ak, pk, pkl) != 0)) return NULL;
        if (!pv_wild && (avl != pvl || memcmp(av, pv, pvl) != 0)) return NULL;

        if (pk_wild && pv_wild) {
            /* A single '?' binding must satisfy both slots: require arg K == V */
            if (akl != avl || memcmp(ak, av, akl) != 0) return NULL;
        }

        char *result;
        if (pk_wild) {
            char *p = strndup(pk, pkl);
            char *a = strndup(ak, akl);
            result = bind_wildcard_str(p, a);
            free(p); free(a);
        } else {
            char *p = strndup(pv, pvl);
            char *a = strndup(av, avl);
            result = bind_wildcard_str(p, a);
            free(p); free(a);
        }
        return result;
    }

    return NULL;
}

/* Derive the concrete type that '?' binds to given the parameter's type
 * string and the resolved argument GrayType. Delegates to bind_wildcard_str
 * so composite nesting (arrays-of-arrays, maps-of-arrays, etc.) is handled
 * recursively. Caller owns the returned string. */
static char *bind_wildcard(const char *param_tn, GrayType *arg_t) {
    if (!param_tn || !arg_t) return NULL;
    return bind_wildcard_str(param_tn, type_name(arg_t));
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
        fs->instantiations = xrealloc(fs->instantiations,
            sizeof(const char *) * fs->instantiation_cap);
        fs->instantiation_calls = xrealloc(fs->instantiation_calls,
            sizeof(AstNode *) * fs->instantiation_cap);
    }
    char *stored = strdup(concrete);
    fs->instantiations[fs->instantiation_count] = stored;
    fs->instantiation_calls[fs->instantiation_count] = call_site;
    fs->instantiation_count++;

    if (fs->decl && fs->decl->kind == NODE_FUNC_DECL) {
        int n = fs->decl->data.func_decl.instantiation_count;
        fs->decl->data.func_decl.instantiations = xrealloc(
            (void *)fs->decl->data.func_decl.instantiations,
            sizeof(const char *) * (size_t)(n + 1));
        fs->decl->data.func_decl.instantiations[n] = stored;
        fs->decl->data.func_decl.instantiation_count = n + 1;
    }
    return true;
}

static int funcsig_name_cmp(const void *a, const void *b) {
    return strcmp((*(const FuncSig *const *)a)->name,
                  (*(const FuncSig *const *)b)->name);
}

static FuncSig *find_func(TypeChecker *tc, const char *name) {
    if (tc->func_count == 0) return NULL;
    if (!tc->funcs_sorted_built) {
        tc->funcs_sorted = xrealloc(tc->funcs_sorted,
            sizeof(FuncSig *) * (size_t)tc->func_count);
        for (int i = 0; i < tc->func_count; i++)
            tc->funcs_sorted[i] = &tc->funcs[i];
        qsort(tc->funcs_sorted, (size_t)tc->func_count,
              sizeof(FuncSig *), funcsig_name_cmp);
        tc->funcs_sorted_built = true;
    }
    FuncSig key = { .name = name };
    FuncSig *key_ptr = &key;
    FuncSig **hit = bsearch(&key_ptr, tc->funcs_sorted,
        (size_t)tc->func_count, sizeof(FuncSig *), funcsig_name_cmp);
    return hit ? *hit : NULL;
}

/* The name the programmer wrote, never the module-prefixed internal one.
 * FuncSig.name is the flattened lookup key (e.g. "mod_size" for `size`
 * imported from module `mod`); the user-facing name lives on the decl.
 * Diagnostics and namespace-collision checks must use this, not .name. */
static const char *func_display_name(const FuncSig *fs) {
    if (fs && fs->decl && fs->decl->kind == NODE_FUNC_DECL &&
        fs->decl->data.func_decl.original_name) {
        return fs->decl->data.func_decl.original_name;
    }
    return fs ? fs->name : "";
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
    {"io", "read_file"}, {"io", "read_bytes"}, {"io", "read_lines"},
    {"io", "file_size"}, {"io", "write_file"}, {"io", "delete_file"},
    {"io", "copy_file"}, {"io", "move_file"}, {"io", "list_dir"},
    {"io", "make_dir"}, {"io", "make_dir_all"}, {"io", "remove_dir"},
    {"io", "remove_dir_all"}, {"io", "walk"}, {"io", "append_file"},
    {"io", "rename_file"}, {"io", "glob"},
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
    /* strconv */
    {"strconv", "to_int"}, {"strconv", "to_uint"},
    {"strconv", "to_float"}, {"strconv", "to_bool"},
};

static int fallible_entry_cmp(const void *a, const void *b) {
    const FallibleEntry *ea = *(const FallibleEntry *const *)a;
    const FallibleEntry *eb = *(const FallibleEntry *const *)b;
    int r = strcmp(ea->mod, eb->mod);
    return r != 0 ? r : strcmp(ea->fn, eb->fn);
}

#define FALLIBLE_STDLIB_N (int)(sizeof(fallible_stdlib) / sizeof(fallible_stdlib[0]))
static const FallibleEntry *fallible_stdlib_sorted[FALLIBLE_STDLIB_N];

/* Returns true if (mod, fn) is a fallible stdlib function.
 * Pass mod=NULL to check by fn name alone (any module). */
static bool tc_is_fallible_stdlib(const char *mod, const char *fn) {
    if (!mod) {
        for (int i = 0; i < FALLIBLE_STDLIB_N; i++) {
            if (strcmp(fn, fallible_stdlib[i].fn) == 0) return true;
        }
        return false;
    }
    FallibleEntry key = { .mod = mod, .fn = fn };
    const FallibleEntry *key_ptr = &key;
    return bsearch(&key_ptr, fallible_stdlib_sorted, FALLIBLE_STDLIB_N,
                   sizeof(const FallibleEntry *), fallible_entry_cmp) != NULL;
}

/* Return the primary (non-error) type for a fallible stdlib function.
 * This mirrors the type registration blocks so that multi-var synthesis
 * doesn't depend on sym->type being resolved first. */
typedef enum {
    FT_BOOL, FT_INT, FT_UINT, FT_FLOAT, FT_STRING,
    FT_ARRAY_STRING, FT_NESTED_ARRAY_STRING, FT_ARRAY_BYTE, FT_ARRAY_MAP,
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
    {"io", "read_bytes", FT_ARRAY_BYTE}, {"io", "read_lines", FT_ARRAY_STRING},
    {"io", "file_size", FT_INT}, {"io", "glob", FT_ARRAY_STRING},
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
    {"csv", "read_file", FT_NESTED_ARRAY_STRING}, {"csv", "write_file", FT_BOOL},
    /* json */
    {"json", "decode", FT_STRUCT_MAP},
    /* regex */
    {"regex", "find", FT_STRING}, {"regex", "replace", FT_STRING},
    {"regex", "find_all", FT_ARRAY_STRING}, {"regex", "split", FT_ARRAY_STRING},
    /* strconv */
    {"strconv", "to_int", FT_INT}, {"strconv", "to_uint", FT_UINT},
    {"strconv", "to_float", FT_FLOAT}, {"strconv", "to_bool", FT_BOOL},
};

static int fallible_type_entry_cmp(const void *a, const void *b) {
    const FallibleTypeEntry *ea = *(const FallibleTypeEntry *const *)a;
    const FallibleTypeEntry *eb = *(const FallibleTypeEntry *const *)b;
    int r = strcmp(ea->mod, eb->mod);
    return r != 0 ? r : strcmp(ea->fn, eb->fn);
}

#define FALLIBLE_TYPE_TABLE_N (int)(sizeof(fallible_type_table) / sizeof(fallible_type_table[0]))
static const FallibleTypeEntry *fallible_type_sorted[FALLIBLE_TYPE_TABLE_N];

static GrayType *tc_get_fallible_stdlib_type(const char *mod, const char *fn) {
    FallibleTypeEntry key = { .mod = mod, .fn = fn };
    const FallibleTypeEntry *key_ptr = &key;
    const FallibleTypeEntry **hit = bsearch(&key_ptr, fallible_type_sorted, FALLIBLE_TYPE_TABLE_N,
        sizeof(const FallibleTypeEntry *), fallible_type_entry_cmp);
    if (!hit) return NULL;
    switch ((*hit)->type) {
    case FT_BOOL:                return &TYPE_BOOL;
    case FT_INT:                 return &TYPE_INT;
    case FT_UINT:                return &TYPE_UINT;
    case FT_FLOAT:               return &TYPE_FLOAT;
    case FT_STRING:              return &TYPE_STRING;
    case FT_ARRAY_STRING:        return type_array("string");
    case FT_NESTED_ARRAY_STRING: return type_array("[string]");
    case FT_ARRAY_BYTE:          return type_array("byte");
    case FT_ARRAY_MAP:           return type_array("map");
    case FT_STRUCT_DATABASE:     return type_struct("Database");
    case FT_STRUCT_SOCKET:       return type_struct("Socket");
    case FT_STRUCT_LISTENER:     return type_struct("Listener");
    case FT_STRUCT_HTTP_RESPONSE:return type_struct("HttpResponse");
    case FT_STRUCT_MAP:          return type_from_name("map[string:string]");
    }
    return NULL;
}

/* Stdlib argument count validation table.
 * Each entry maps a (module, function) pair to the expected min/max arg count.
 * Functions with variable args use different min/max values. */
typedef struct {
    const char *mod;
    const char *fn;
    int min_args;
    int max_args;
} StdlibArgEntry;

static const StdlibArgEntry stdlib_arg_table[] = {
    /* io */
    {"io", "read_file", 1, 1}, {"io", "read_bytes", 1, 1}, {"io", "read_lines", 1, 1},
    {"io", "write_file", 2, 2}, {"io", "append_file", 2, 2},
    {"io", "file_exists", 1, 1}, {"io", "is_file", 1, 1},
    {"io", "is_directory", 1, 1}, {"io", "file_size", 1, 1},
    {"io", "delete_file", 1, 1},
    {"io", "rename_file", 2, 2}, {"io", "copy_file", 2, 2}, {"io", "move_file", 2, 2},
    {"io", "list_dir", 1, 1}, {"io", "walk", 1, 1}, {"io", "glob", 1, 1},
    {"io", "make_dir", 1, 1}, {"io", "make_dir_all", 1, 1},
    {"io", "remove_dir", 1, 1}, {"io", "remove_dir_all", 1, 1},
    {"io", "path_join", 1, 1}, {"io", "dirname", 1, 1},
    {"io", "basename", 1, 1}, {"io", "extension", 1, 1},
    {"io", "is_absolute", 1, 1}, {"io", "normalize", 1, 1},
    /* strings */
    {"strings", "to_upper", 1, 1}, {"strings", "to_lower", 1, 1},
    {"strings", "is_empty", 1, 1}, {"strings", "contains", 2, 2},
    {"strings", "starts_with", 2, 2}, {"strings", "ends_with", 2, 2},
    {"strings", "index_of", 2, 2}, {"strings", "last_index_of", 2, 2}, {"strings", "count", 2, 2},
    {"strings", "trim", 1, 1}, {"strings", "trim_left", 1, 1},
    {"strings", "trim_right", 1, 1},
    {"strings", "remove_prefix", 2, 2}, {"strings", "remove_suffix", 2, 2},
    {"strings", "replace", 3, 3},
    {"strings", "repeat", 2, 2}, {"strings", "reverse", 1, 1},
    {"strings", "split", 2, 2}, {"strings", "join", 2, 2},
    {"strings", "slice", 3, 3},
    {"strings", "char_at", 2, 2},
    {"strings", "to_chars", 1, 1}, {"strings", "from_chars", 1, 1},
    {"strings", "is_alpha", 1, 1}, {"strings", "is_digit", 1, 1},
    {"strings", "is_alnum", 1, 1}, {"strings", "is_whitespace", 1, 1},
    {"strings", "is_upper", 1, 1}, {"strings", "is_lower", 1, 1},
    /* json */
    {"json", "decode", 1, 1}, {"json", "encode", 1, 1},
    {"json", "stringify", 1, 1},
    {"json", "is_valid", 1, 1}, {"json", "pretty_print", 2, 2},
    {"json", "parse", 1, 1},
    /* crypto */
    {"crypto", "sha256", 1, 1}, {"crypto", "md5", 1, 1},
    {"crypto", "random_hex", 1, 1},
    /* encoding */
    {"encoding", "base64_encode", 1, 1}, {"encoding", "base64_decode", 1, 1},
    {"encoding", "hex_encode", 1, 1}, {"encoding", "hex_decode", 1, 1},
    {"encoding", "url_encode", 1, 1}, {"encoding", "url_decode", 1, 1},
    /* uuid */
    {"uuid", "generate", 0, 0}, {"uuid", "generate_hyphenated", 0, 0},
    {"uuid", "generate_compact", 1, 1},
    {"uuid", "is_valid", 1, 1}, {"uuid", "to_string", 1, 1},
    {"uuid", "generate_random", 0, 0}, {"uuid", "generate_time_ordered", 0, 0},
    {"uuid", "parse", 1, 1},
    /* regex */
    {"regex", "is_valid", 1, 1}, {"regex", "is_match", 2, 2},
    {"regex", "find", 2, 2}, {"regex", "find_all", 2, 2},
    {"regex", "replace", 3, 3}, {"regex", "split", 2, 2},
    /* csv */
    {"csv", "parse", 1, 1},
    {"csv", "encode", 1, 1},
    {"csv", "read_file", 1, 1}, {"csv", "write_file", 2, 2},
    {"csv", "headers", 1, 1},
    /* http */
    {"http", "get", 2, 2}, {"http", "delete", 2, 2}, {"http", "head", 2, 2},
    {"http", "post", 3, 3}, {"http", "put", 3, 3}, {"http", "patch", 3, 3},
    /* net */
    {"net", "connect", 2, 2}, {"net", "listen", 1, 2},
    {"net", "accept", 1, 1}, {"net", "send", 2, 2},
    {"net", "receive", 2, 2}, {"net", "close", 1, 1},
    {"net", "set_timeout", 2, 2}, {"net", "resolve", 1, 1},
    /* time */
    {"time", "now", 0, 0}, {"time", "now_ms", 0, 0}, {"time", "now_ns", 0, 0},
    {"time", "year", 1, 1}, {"time", "month", 1, 1}, {"time", "day", 1, 1},
    {"time", "hour", 1, 1}, {"time", "minute", 1, 1}, {"time", "second", 1, 1},
    {"time", "weekday", 1, 1}, {"time", "format", 2, 2},
    {"time", "to_iso", 1, 1}, {"time", "date", 1, 1}, {"time", "to_clock", 1, 1},
    {"time", "tick", 0, 0}, {"time", "elapsed_ms", 1, 1}, {"time", "diff", 2, 2},
    /* os */
    {"os", "get_env", 1, 1}, {"os", "set_env", 2, 2}, {"os", "unset_env", 1, 1},
    {"os", "args", 0, 0}, {"os", "current_dir", 0, 0},
    {"os", "hostname", 0, 0}, {"os", "pid", 0, 0},
    {"os", "current_os", 0, 0}, {"os", "arch", 0, 0},
    {"os", "exec", 2, 2},
    /* random (variable args) */
    {"random", "rand_float", 0, 2}, {"random", "rand_int", 1, 2},
    {"random", "rand_bool", 0, 0}, {"random", "rand_byte", 0, 0},
    {"random", "rand_char", 0, 2},
    {"random", "choice", 1, 1}, {"random", "shuffle", 1, 1},
    {"random", "sample", 2, 2}, {"random", "seed", 1, 1},
    /* sqlite */
    {"sqlite", "open", 1, 1}, {"sqlite", "close", 1, 1},
    {"sqlite", "exec", 2, 99}, {"sqlite", "query", 2, 99},
    /* arrays */
    {"arrays", "remove", 2, 2},
    {"arrays", "map", 2, 2}, {"arrays", "filter", 2, 2},
    {"arrays", "reduce", 3, 3},
    {"arrays", "any", 2, 2}, {"arrays", "all", 2, 2},
    /* bytes */
    {"bytes", "from_string", 1, 1}, {"bytes", "from_hex", 1, 1},
    {"bytes", "from_base64", 1, 1}, {"bytes", "to_string", 1, 1},
    {"bytes", "to_hex", 1, 1}, {"bytes", "to_base64", 1, 1},
    /* strconv */
    {"strconv", "to_int", 1, 2}, {"strconv", "to_uint", 1, 2},
    {"strconv", "to_float", 1, 1}, {"strconv", "to_bool", 1, 1},
    {"strconv", "from_int", 1, 1}, {"strconv", "from_uint", 1, 1},
    {"strconv", "from_float", 1, 1}, {"strconv", "from_bool", 1, 1},
    {"strconv", "is_numeric", 1, 1}, {"strconv", "is_integer", 1, 1},
    /* math */
    {"math", "abs", 1, 1}, {"math", "neg", 1, 1}, {"math", "sign", 1, 1},
    {"math", "min", 2, 2}, {"math", "max", 2, 2}, {"math", "clamp", 3, 3},
    {"math", "floor", 1, 1}, {"math", "ceil", 1, 1},
    {"math", "round", 1, 1}, {"math", "trunc", 1, 1},
    {"math", "pow", 2, 2}, {"math", "sqrt", 1, 1}, {"math", "cbrt", 1, 1},
    {"math", "hypot", 2, 2}, {"math", "exp", 1, 1}, {"math", "exp2", 1, 1},
    {"math", "log", 1, 1}, {"math", "log2", 1, 1},
    {"math", "log10", 1, 1}, {"math", "log_base", 2, 2},
    {"math", "sin", 1, 1}, {"math", "cos", 1, 1}, {"math", "tan", 1, 1},
    {"math", "asin", 1, 1}, {"math", "acos", 1, 1},
    {"math", "atan", 1, 1}, {"math", "atan2", 2, 2},
    {"math", "sinh", 1, 1}, {"math", "cosh", 1, 1}, {"math", "tanh", 1, 1},
    {"math", "deg_to_rad", 1, 1}, {"math", "rad_to_deg", 1, 1},
    {"math", "factorial", 1, 1}, {"math", "gcd", 2, 2}, {"math", "lcm", 2, 2},
    {"math", "is_prime", 1, 1}, {"math", "is_even", 1, 1}, {"math", "is_odd", 1, 1},
    {"math", "is_infinite", 1, 1}, {"math", "is_nan", 1, 1}, {"math", "is_finite", 1, 1},
    {"math", "lerp", 3, 3}, {"math", "distance", 4, 4},
    /* maps */
    {"maps", "get_keys", 1, 1}, {"maps", "get_values", 1, 1},
    {"maps", "has_key", 2, 2}, {"maps", "is_empty", 1, 1},
    {"maps", "contains_value", 2, 2}, {"maps", "is_equal", 2, 2},
    {"maps", "merge", 2, 2}, {"maps", "remove_key", 2, 2},
    {"maps", "clear", 1, 1}, {"maps", "get_or_default", 3, 3},
    /* mem */
    {"mem", "arena", 1, 1}, {"mem", "destroy", 1, 1},
    {"mem", "reset", 1, 1}, {"mem", "usage", 1, 1},
    {"mem", "raw_copy", 3, 3}, {"mem", "zero", 2, 2}, {"mem", "fill", 3, 3},
    {"mem", "init", 2, 2}, {"mem", "alloc", 2, 2},
    /* threads */
    {"threads", "spawn", 1, 2}, {"threads", "join", 1, 1},
    {"threads", "detach", 1, 1}, {"threads", "is_alive", 1, 1},
    {"threads", "get_id", 0, 0}, {"threads", "current", 0, 0},
    {"threads", "yield", 0, 0}, {"threads", "sleep", 1, 1},
    {"threads", "thread_count", 0, 0},
    /* sync */
    {"sync", "mutex", 0, 0}, {"sync", "lock", 1, 1},
    {"sync", "unlock", 1, 1}, {"sync", "try_lock", 1, 1},
    {"sync", "destroy", 1, 1},
    /* channels */
    {"channels", "open", 1, 1}, {"channels", "send", 2, 2},
    {"channels", "receive", 1, 1}, {"channels", "close", 1, 1},
    {"channels", "try_send", 2, 2}, {"channels", "try_receive", 1, 1},
    /* atomic */
    {"atomic", "load", 1, 1}, {"atomic", "store", 2, 2},
    {"atomic", "add", 2, 2}, {"atomic", "sub", 2, 2},
    {"atomic", "exchange", 2, 2}, {"atomic", "cas", 3, 3},
    {"atomic", "and", 2, 2}, {"atomic", "or", 2, 2}, {"atomic", "xor", 2, 2},
    {"atomic", "spinlock", 0, 0}, {"atomic", "spinlock_destroy", 1, 1},
    {"atomic", "spin_lock", 1, 1},
    {"atomic", "spin_trylock", 1, 1}, {"atomic", "spin_unlock", 1, 1},
    {"atomic", "fence", 0, 0},
    /* fmt */
    {"fmt", "sprintf", 1, 99}, {"fmt", "format", 1, 99},
    {"fmt", "printf", 1, 99}, {"fmt", "printfln", 1, 99},
    {"fmt", "eprintf", 1, 99}, {"fmt", "eprintfln", 1, 99},
    {"fmt", "sprintfln", 1, 99},
    {"fmt", "pad_left", 3, 3}, {"fmt", "pad_right", 3, 3}, {"fmt", "center", 3, 3},
    {"fmt", "int_to_hex", 1, 1}, {"fmt", "int_to_binary", 1, 1},
    {"fmt", "int_to_octal", 1, 1}, {"fmt", "float_fixed", 2, 2},
    {"fmt", "float_sci", 1, 1},
    /* server */
    {"server", "add_router", 0, 0}, {"server", "add_route", 4, 4},
    {"server", "listen", 2, 3}, {"server", "cors", 2, 2},
    {"server", "use", 2, 2}, {"server", "text", 2, 2},
    {"server", "json", 2, 2}, {"server", "html", 2, 2},
    {"server", "redirect", 2, 2},
};

static int stdlib_arg_entry_cmp(const void *a, const void *b) {
    const StdlibArgEntry *ea = *(const StdlibArgEntry *const *)a;
    const StdlibArgEntry *eb = *(const StdlibArgEntry *const *)b;
    int r = strcmp(ea->mod, eb->mod);
    return r != 0 ? r : strcmp(ea->fn, eb->fn);
}

#define STDLIB_ARG_TABLE_N (int)(sizeof(stdlib_arg_table) / sizeof(stdlib_arg_table[0]))
static const StdlibArgEntry *stdlib_arg_sorted[STDLIB_ARG_TABLE_N];

static void tc_check_stdlib_arg_count(TypeChecker *tc, const char *mod,
    const char *fn, AstNode *node)
{
    StdlibArgEntry key = { .mod = mod, .fn = fn };
    const StdlibArgEntry *key_ptr = &key;
    const StdlibArgEntry **hit = bsearch(&key_ptr, stdlib_arg_sorted, STDLIB_ARG_TABLE_N,
        sizeof(const StdlibArgEntry *), stdlib_arg_entry_cmp);
    if (!hit) return;

    const StdlibArgEntry *e = *hit;
    int nargs = node->data.call.arg_count;
    if (nargs < e->min_args || nargs > e->max_args) {
        char *msg = NULL;
        if (e->min_args == e->max_args) {
            msg = tc_fmt(tc,
                "function '%s.%s' expects %d argument(s), got %d",
                mod, fn, e->min_args, nargs);
        } else {
            msg = tc_fmt(tc,
                "function '%s.%s' expects %d to %d argument(s), got %d",
                mod, fn, e->min_args, e->max_args, nargs);
        }
        diag_error_msg(tc->diag, "E5008", msg,
            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
    }
}

/* Compile-time validation for strconv base parameter.
 * When the second arg to to_int/to_uint is a literal integer, verify it's
 * in the valid range [2, 36]. */
static void tc_check_strconv_base(TypeChecker *tc, const char *mod,
    const char *fn, AstNode *node)
{
    if (strcmp(mod, "strconv") != 0) return;
    if (strcmp(fn, "to_int") != 0 && strcmp(fn, "to_uint") != 0) return;
    if (node->data.call.arg_count < 2) return;
    AstNode *base_arg = node->data.call.args[1];
    if (base_arg->kind != NODE_INT_VALUE) return;
    int64_t base = base_arg->data.int_value.value;
    if (base < 2 || base > 36) {
        char *msg;
        msg = tc_fmt(tc,
            "invalid base %lld for strconv.%s; base must be between 2 and 36",
            (long long)base, fn);
        diag_error_msg(tc->diag, "E5009", msg,
            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
    }
}

/* Stdlib first-argument type validation.
 * Most stdlib modules have a dominant first-arg type (e.g. strings.* expects
 * a string, arrays.* expects an array). This table defines per-function
 * expected types so we can catch type mismatches before they leak to C.
 * ARG_ANY means no validation (the function accepts mixed types). */
typedef enum {
    ARG_STRING, ARG_INT, ARG_FLOAT, ARG_BOOL, ARG_ARRAY, ARG_MAP, ARG_ANY, ARG_NUMBER, ARG_CHAR, ARG_CHANNEL
} ExpectedArgKind;

typedef struct {
    const char *mod;
    const char *fn;
    int arg_index;        /* which argument to check (0-based) */
    ExpectedArgKind kind; /* expected type kind */
} StdlibArgTypeEntry;

static const StdlibArgTypeEntry stdlib_arg_type_table[] = {
    /* strings: first arg is always a string */
    {"strings", "to_upper", 0, ARG_STRING}, {"strings", "to_lower", 0, ARG_STRING},
    {"strings", "is_empty", 0, ARG_STRING}, {"strings", "contains", 0, ARG_STRING},
    {"strings", "starts_with", 0, ARG_STRING}, {"strings", "ends_with", 0, ARG_STRING},
    {"strings", "index_of", 0, ARG_STRING}, {"strings", "last_index_of", 0, ARG_STRING},
    {"strings", "count", 0, ARG_STRING},
    {"strings", "trim", 0, ARG_STRING}, {"strings", "trim_left", 0, ARG_STRING},
    {"strings", "trim_right", 0, ARG_STRING},
    {"strings", "remove_prefix", 0, ARG_STRING}, {"strings", "remove_prefix", 1, ARG_STRING},
    {"strings", "remove_suffix", 0, ARG_STRING}, {"strings", "remove_suffix", 1, ARG_STRING},
    {"strings", "replace", 0, ARG_STRING},
    {"strings", "repeat", 0, ARG_STRING}, {"strings", "reverse", 0, ARG_STRING},
    {"strings", "split", 0, ARG_STRING}, {"strings", "slice", 0, ARG_STRING},
    {"strings", "contains", 1, ARG_STRING}, {"strings", "starts_with", 1, ARG_STRING},
    {"strings", "ends_with", 1, ARG_STRING}, {"strings", "index_of", 1, ARG_STRING},
    {"strings", "last_index_of", 1, ARG_STRING},
    {"strings", "count", 1, ARG_STRING}, {"strings", "replace", 1, ARG_STRING},
    {"strings", "replace", 2, ARG_STRING}, {"strings", "split", 1, ARG_STRING},
    {"strings", "repeat", 1, ARG_INT},
    {"strings", "slice", 1, ARG_INT}, {"strings", "slice", 2, ARG_INT},
    {"strings", "join", 0, ARG_ARRAY}, {"strings", "join", 1, ARG_STRING},
    {"strings", "char_at", 0, ARG_STRING}, {"strings", "char_at", 1, ARG_INT},
    {"strings", "to_chars", 0, ARG_STRING}, {"strings", "from_chars", 0, ARG_ARRAY},
    {"strings", "is_alpha", 0, ARG_CHAR}, {"strings", "is_digit", 0, ARG_CHAR},
    {"strings", "is_alnum", 0, ARG_CHAR}, {"strings", "is_whitespace", 0, ARG_CHAR},
    {"strings", "is_upper", 0, ARG_CHAR}, {"strings", "is_lower", 0, ARG_CHAR},
    /* io: path args are strings */
    {"io", "read_file", 0, ARG_STRING}, {"io", "read_bytes", 0, ARG_STRING},
    {"io", "read_lines", 0, ARG_STRING}, {"io", "write_file", 0, ARG_STRING},
    {"io", "write_file", 1, ARG_STRING},
    {"io", "append_file", 0, ARG_STRING}, {"io", "append_file", 1, ARG_STRING},
    {"io", "file_exists", 0, ARG_STRING},
    {"io", "is_file", 0, ARG_STRING}, {"io", "is_directory", 0, ARG_STRING},
    {"io", "file_size", 0, ARG_STRING}, {"io", "delete_file", 0, ARG_STRING},
    {"io", "rename_file", 0, ARG_STRING}, {"io", "copy_file", 0, ARG_STRING},
    {"io", "move_file", 0, ARG_STRING}, {"io", "list_dir", 0, ARG_STRING},
    {"io", "walk", 0, ARG_STRING}, {"io", "glob", 0, ARG_STRING},
    {"io", "make_dir", 0, ARG_STRING}, {"io", "make_dir_all", 0, ARG_STRING},
    {"io", "remove_dir", 0, ARG_STRING}, {"io", "remove_dir_all", 0, ARG_STRING},
    {"io", "path_join", 0, ARG_ARRAY}, {"io", "dirname", 0, ARG_STRING},
    {"io", "basename", 0, ARG_STRING}, {"io", "extension", 0, ARG_STRING},
    {"io", "is_absolute", 0, ARG_STRING}, {"io", "normalize", 0, ARG_STRING},
    /* maps: first arg is a map */
    {"maps", "is_empty", 0, ARG_MAP}, {"maps", "has_key", 0, ARG_MAP},
    {"maps", "contains_value", 0, ARG_MAP}, {"maps", "get_keys", 0, ARG_MAP},
    {"maps", "get_values", 0, ARG_MAP}, {"maps", "get_or_default", 0, ARG_MAP},
    {"maps", "remove_key", 0, ARG_MAP}, {"maps", "clear", 0, ARG_MAP},
    {"maps", "merge", 0, ARG_MAP}, {"maps", "is_equal", 0, ARG_MAP},
    /* crypto: string data */
    {"crypto", "sha256", 0, ARG_STRING}, {"crypto", "md5", 0, ARG_STRING},
    {"crypto", "random_hex", 0, ARG_INT},
    /* encoding: string data */
    {"encoding", "base64_encode", 0, ARG_STRING}, {"encoding", "base64_decode", 0, ARG_STRING},
    {"encoding", "hex_encode", 0, ARG_STRING}, {"encoding", "hex_decode", 0, ARG_STRING},
    {"encoding", "url_encode", 0, ARG_STRING}, {"encoding", "url_decode", 0, ARG_STRING},
    /* regex: first arg (pattern) is string */
    {"regex", "is_valid", 0, ARG_STRING}, {"regex", "is_match", 0, ARG_STRING},
    {"regex", "find", 0, ARG_STRING}, {"regex", "find_all", 0, ARG_STRING},
    {"regex", "replace", 0, ARG_STRING}, {"regex", "split", 0, ARG_STRING},
    /* json */
    {"json", "decode", 0, ARG_STRING}, {"json", "is_valid", 0, ARG_STRING},
    {"json", "parse", 0, ARG_STRING},
    /* csv */
    {"csv", "parse", 0, ARG_STRING},
    {"csv", "read_file", 0, ARG_STRING}, {"csv", "write_file", 0, ARG_STRING},
    {"csv", "headers", 0, ARG_ARRAY},
    /* http: url is string, body is string, headers is map */
    {"http", "get", 0, ARG_STRING}, {"http", "get", 1, ARG_MAP},
    {"http", "delete", 0, ARG_STRING}, {"http", "delete", 1, ARG_MAP},
    {"http", "head", 0, ARG_STRING}, {"http", "head", 1, ARG_MAP},
    {"http", "post", 0, ARG_STRING}, {"http", "post", 1, ARG_STRING}, {"http", "post", 2, ARG_MAP},
    {"http", "put", 0, ARG_STRING}, {"http", "put", 1, ARG_STRING}, {"http", "put", 2, ARG_MAP},
    {"http", "patch", 0, ARG_STRING}, {"http", "patch", 1, ARG_STRING}, {"http", "patch", 2, ARG_MAP},
    /* bytes */
    {"bytes", "from_string", 0, ARG_STRING}, {"bytes", "from_hex", 0, ARG_STRING},
    {"bytes", "from_base64", 0, ARG_STRING}, {"bytes", "to_string", 0, ARG_ARRAY},
    {"bytes", "to_hex", 0, ARG_ARRAY}, {"bytes", "to_base64", 0, ARG_ARRAY},
    /* sqlite */
    {"sqlite", "open", 0, ARG_STRING},
    /* server: listen(router, port int) — catch non-int port before it reaches C */
    {"server", "listen", 1, ARG_INT},
    {"server", "listen", 2, ARG_STRING},
    {"net", "listen", 1, ARG_INT},
    {"server", "text", 1, ARG_STRING},
    {"server", "json", 1, ARG_STRING},
    {"server", "html", 1, ARG_STRING},
    {"server", "redirect", 1, ARG_STRING},
    /* channels: value arg must be int */
    {"channels", "send", 0, ARG_CHANNEL}, {"channels", "send", 1, ARG_INT},
    {"channels", "receive", 0, ARG_CHANNEL}, {"channels", "close", 0, ARG_CHANNEL},
    {"channels", "try_send", 0, ARG_CHANNEL}, {"channels", "try_send", 1, ARG_INT},
    {"channels", "try_receive", 0, ARG_CHANNEL},
    /* math: numeric argument required for all single-arg functions */
    {"math", "sqrt", 0, ARG_NUMBER}, {"math", "cbrt", 0, ARG_NUMBER},
    {"math", "log", 0, ARG_NUMBER}, {"math", "log2", 0, ARG_NUMBER},
    {"math", "log10", 0, ARG_NUMBER}, {"math", "exp", 0, ARG_NUMBER},
    {"math", "exp2", 0, ARG_NUMBER}, {"math", "floor", 0, ARG_NUMBER},
    {"math", "ceil", 0, ARG_NUMBER}, {"math", "round", 0, ARG_NUMBER},
    {"math", "trunc", 0, ARG_NUMBER}, {"math", "asin", 0, ARG_NUMBER},
    {"math", "acos", 0, ARG_NUMBER}, {"math", "atan", 0, ARG_NUMBER},
    {"math", "sin", 0, ARG_NUMBER}, {"math", "cos", 0, ARG_NUMBER},
    {"math", "tan", 0, ARG_NUMBER}, {"math", "abs", 0, ARG_NUMBER},
    {"math", "sign", 0, ARG_NUMBER}, {"math", "factorial", 0, ARG_INT},
    {"math", "is_prime", 0, ARG_INT}, {"math", "is_even", 0, ARG_INT},
    {"math", "is_odd", 0, ARG_INT}, {"math", "is_infinite", 0, ARG_NUMBER},
    {"math", "is_nan", 0, ARG_NUMBER}, {"math", "is_finite", 0, ARG_NUMBER},
    {"math", "deg_to_rad", 0, ARG_NUMBER}, {"math", "rad_to_deg", 0, ARG_NUMBER},
    {"math", "neg", 0, ARG_NUMBER},
    {"math", "min", 0, ARG_NUMBER}, {"math", "min", 1, ARG_NUMBER},
    {"math", "max", 0, ARG_NUMBER}, {"math", "max", 1, ARG_NUMBER},
    {"math", "clamp", 0, ARG_NUMBER}, {"math", "clamp", 1, ARG_NUMBER}, {"math", "clamp", 2, ARG_NUMBER},
    {"math", "pow", 0, ARG_NUMBER}, {"math", "pow", 1, ARG_NUMBER},
    {"math", "hypot", 0, ARG_NUMBER}, {"math", "hypot", 1, ARG_NUMBER},
    {"math", "log_base", 0, ARG_NUMBER}, {"math", "log_base", 1, ARG_NUMBER},
    {"math", "atan2", 0, ARG_NUMBER}, {"math", "atan2", 1, ARG_NUMBER},
    {"math", "sinh", 0, ARG_NUMBER}, {"math", "cosh", 0, ARG_NUMBER}, {"math", "tanh", 0, ARG_NUMBER},
    {"math", "gcd", 0, ARG_INT}, {"math", "gcd", 1, ARG_INT},
    {"math", "lcm", 0, ARG_INT}, {"math", "lcm", 1, ARG_INT},
    {"math", "lerp", 0, ARG_NUMBER}, {"math", "lerp", 1, ARG_NUMBER}, {"math", "lerp", 2, ARG_NUMBER},
    {"math", "distance", 0, ARG_NUMBER}, {"math", "distance", 1, ARG_NUMBER},
    {"math", "distance", 2, ARG_NUMBER}, {"math", "distance", 3, ARG_NUMBER},
};

static bool arg_kind_matches(ExpectedArgKind expected, GrayType *actual) {
    if (!actual || actual->kind == TK_UNKNOWN) return true; /* can't validate */
    switch (expected) {
    case ARG_STRING: return actual->kind == TK_STRING;
    case ARG_INT:    return actual->kind == TK_INT || actual->kind == TK_UINT ||
                            actual->kind == TK_BYTE;
    case ARG_FLOAT:  return actual->kind == TK_FLOAT;
    case ARG_BOOL:   return actual->kind == TK_BOOL;
    case ARG_ARRAY:  return actual->kind == TK_ARRAY;
    case ARG_MAP:    return actual->kind == TK_MAP;
    case ARG_ANY:    return true;
    case ARG_NUMBER: return actual->kind == TK_INT || actual->kind == TK_UINT ||
                            actual->kind == TK_BYTE || actual->kind == TK_FLOAT;
    case ARG_CHAR:   return actual->kind == TK_CHAR;
    case ARG_CHANNEL: return actual->kind == TK_STRUCT &&
                             actual->name && strcmp(actual->name, "Channel") == 0;
    }
    return true;
}

static const char *expected_kind_name(ExpectedArgKind kind) {
    switch (kind) {
    case ARG_STRING: return "string";
    case ARG_INT:    return "int";
    case ARG_FLOAT:  return "float";
    case ARG_BOOL:   return "bool";
    case ARG_ARRAY:  return "array";
    case ARG_MAP:    return "map";
    case ARG_ANY:    return "any";
    case ARG_NUMBER: return "number";
    case ARG_CHAR:   return "char";
    case ARG_CHANNEL: return "Channel";
    }
    return "unknown";
}

static int stdlib_argtype_entry_cmp(const void *a, const void *b) {
    const StdlibArgTypeEntry *ea = *(const StdlibArgTypeEntry *const *)a;
    const StdlibArgTypeEntry *eb = *(const StdlibArgTypeEntry *const *)b;
    int r = strcmp(ea->mod, eb->mod);
    return r != 0 ? r : strcmp(ea->fn, eb->fn);
}

#define STDLIB_ARG_TYPE_TABLE_N (int)(sizeof(stdlib_arg_type_table) / sizeof(stdlib_arg_type_table[0]))
static const StdlibArgTypeEntry *stdlib_argtype_sorted[STDLIB_ARG_TYPE_TABLE_N];

static void tc_check_stdlib_arg_types(TypeChecker *tc, const char *mod,
    const char *fn, AstNode *node)
{
    StdlibArgTypeEntry key = { .mod = mod, .fn = fn };
    const StdlibArgTypeEntry *key_ptr = &key;
    const StdlibArgTypeEntry **hit = bsearch(&key_ptr, stdlib_argtype_sorted, STDLIB_ARG_TYPE_TABLE_N,
        sizeof(const StdlibArgTypeEntry *), stdlib_argtype_entry_cmp);
    if (!hit) return;

    /* bsearch may land on any entry in the (mod, fn) group; walk back to first. */
    while (hit > stdlib_argtype_sorted && stdlib_argtype_entry_cmp(hit - 1, hit) == 0) hit--;

    /* Iterate the contiguous group of entries for this (mod, fn). */
    for (; hit < stdlib_argtype_sorted + STDLIB_ARG_TYPE_TABLE_N && stdlib_argtype_entry_cmp(hit, &key_ptr) == 0; hit++) {
        const StdlibArgTypeEntry *e = *hit;
        int idx = e->arg_index;
        if (idx < node->data.call.arg_count) {
            GrayType *arg_t = resolve_expr(tc, node->data.call.args[idx]);
            if (!arg_kind_matches(e->kind, arg_t)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "%s.%s() expects %s as argument %d, got '%s'",
                    mod, fn, expected_kind_name(e->kind), idx + 1, type_name(arg_t));
                diag_error_msg(tc->diag, "E5026", msg,
                    NODE_FILE(tc, node->data.call.args[idx]),
                    node->data.call.args[idx]->token.line,
                    node->data.call.args[idx]->token.column, 0);
            }
        }
    }
}

/* --- "Did you mean?" suggestion helper --- */

static int levenshtein(const char *a, const char *b) {
    int la = (int)strlen(a), lb = (int)strlen(b);
    if (la == 0) return lb;
    if (lb == 0) return la;
    int stack_row[256];
    int *row = lb < 256 ? stack_row : xmalloc(sizeof(int) * (lb + 1));
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
    if (row != stack_row) free(row);
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
    static const char *const builtins[] = {
        "addr", "assert", "bool", "byte", "c_string", "cast",
        "char", "char_count", "copy", "embed", "eprint", "eprintln",
        "error", "exit", "f32", "f64", "float", "here",
        "i128", "i16", "i256", "i32", "i64", "i8",
        "input", "int", "len", "new", "panic", "print", "println",
        "range", "ref", "size_of", "sleep_ms", "sleep_ns", "sleep_s",
        "string", "to_char", "type_of",
        "u128", "u16", "u256", "u32", "u64", "u8", "uint",
    };
    return strset_contains(builtins, (int)(sizeof(builtins)/sizeof(builtins[0])), name);
}

/* Check whether any arg_names entry is non-NULL (i.e. call has named args). */
static bool tc_has_named_args(AstNode *node) {
    if (!node->data.call.arg_names) return false;
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (node->data.call.arg_names[i]) return true;
    }
    return false;
}

/* Resolve named arguments: validate names, check ordering, and reorder
 * args in-place so downstream code sees normal positional args. */
static void tc_resolve_named_args(TypeChecker *tc, AstNode *node,
                                  AstNode *func_decl, const char *display_name) {
    if (!node->data.call.arg_names) return;

    const char **arg_names = node->data.call.arg_names;
    int arg_count = node->data.call.arg_count;

    /* Quick check: any named args at all? */
    bool any_named = false;
    for (int i = 0; i < arg_count; i++) {
        if (arg_names[i]) { any_named = true; break; }
    }
    if (!any_named) return;

    /* E5033: positional-before-named check */
    bool seen_named = false;
    for (int i = 0; i < arg_count; i++) {
        if (arg_names[i]) {
            seen_named = true;
        } else if (seen_named) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "positional argument after named argument in call to '%s'",
                display_name);
            diag_error_msg(tc->diag, "E5033", msg,
                NODE_FILE(tc, node), node->data.call.args[i]->token.line,
                node->data.call.args[i]->token.column, 0);
            return;
        }
    }

    if (!func_decl || func_decl->kind != NODE_FUNC_DECL) return;

    int param_count = func_decl->data.func_decl.param_count;
    Param *params = func_decl->data.func_decl.params;

    /* Build reordered args array sized to param_count. */
    size_t new_args_size = sizeof(AstNode *) * (size_t)(param_count > 0 ? param_count : 1);
    AstNode **new_args = arena_alloc(tc->arena, new_args_size);
    memset(new_args, 0, new_args_size);

    /* Copy positional args into their slots */
    int positional_count = 0;
    for (int i = 0; i < arg_count; i++) {
        if (!arg_names[i]) {
            if (i < param_count) {
                new_args[i] = node->data.call.args[i];
            }
            positional_count++;
        } else {
            break; /* positional args are contiguous at the front */
        }
    }

    /* Place named args by matching param names */
    for (int i = positional_count; i < arg_count; i++) {
        const char *name = arg_names[i];
        int slot = -1;
        for (int pi = 0; pi < param_count; pi++) {
            if (strcmp(params[pi].name, name) == 0) {
                slot = pi;
                break;
            }
        }
        if (slot < 0) {
            /* E5031: unknown parameter name */
            char *msg = NULL;
            msg = tc_fmt(tc,
                "unknown parameter name '%s' in call to '%s'",
                name, display_name);
            diag_error_msg(tc->diag, "E5031", msg,
                NODE_FILE(tc, node), node->data.call.args[i]->token.line,
                node->data.call.args[i]->token.column, 0);
            return;
        }
        if (new_args[slot]) {
            /* E5032: already filled by a positional arg */
            char *msg = NULL;
            msg = tc_fmt(tc,
                "parameter '%s' is already provided positionally (argument %d) in call to '%s'",
                name, slot + 1, display_name);
            diag_error_msg(tc->diag, "E5032", msg,
                NODE_FILE(tc, node), node->data.call.args[i]->token.line,
                node->data.call.args[i]->token.column, 0);
            return;
        }
        new_args[slot] = node->data.call.args[i];
    }

    /* Count how many slots are actually filled (positional + named).
     * Unfilled slots will be handled by the existing default-value logic. */
    int filled = 0;
    for (int i = param_count - 1; i >= 0; i--) {
        if (new_args[i]) { filled = i + 1; break; }
    }
    /* Ensure at least all positional + named are accounted for */
    if (filled < positional_count) filled = positional_count;

    /* Fill interior gaps with default value AST nodes from the function
     * declaration so codegen doesn't encounter NULL arg pointers. */
    for (int i = 0; i < filled; i++) {
        if (!new_args[i] && i < param_count && params[i].default_value) {
            new_args[i] = params[i].default_value;
        }
    }

    /* Replace the node's args in-place */
    node->data.call.args = new_args;
    node->data.call.arg_count = filled;
    node->data.call.arg_names = NULL; /* now positional */
}


/* stdlib functions reachable via `import and use` / `using`. */
typedef struct { const char *func; const char *mod; TypeKind ret; } UsingFunc;
static const UsingFunc _using_funcs[] = {
    /* strings */
    {"to_upper","strings",TK_STRING},{"to_lower","strings",TK_STRING},
    {"trim","strings",TK_STRING},{"trim_left","strings",TK_STRING},
    {"trim_right","strings",TK_STRING},
    {"remove_prefix","strings",TK_STRING},{"remove_suffix","strings",TK_STRING},
    {"replace","strings",TK_STRING},
    {"repeat","strings",TK_STRING},{"reverse","strings",TK_STRING},
    {"slice","strings",TK_STRING},{"join","strings",TK_STRING},
    {"contains","strings",TK_BOOL},{"starts_with","strings",TK_BOOL},
    {"ends_with","strings",TK_BOOL},{"is_empty","strings",TK_BOOL},
    {"is_alpha","strings",TK_BOOL},{"is_digit","strings",TK_BOOL},
    {"is_alnum","strings",TK_BOOL},{"is_whitespace","strings",TK_BOOL},
    {"is_upper","strings",TK_BOOL},{"is_lower","strings",TK_BOOL},
    {"char_at","strings",TK_CHAR},
    {"to_chars","strings",TK_ARRAY},{"from_chars","strings",TK_STRING},
    {"index_of","strings",TK_INT},{"last_index_of","strings",TK_INT},{"count","strings",TK_INT},
    {"split","strings",TK_ARRAY},
    {"is_alpha","strings",TK_BOOL},{"is_digit","strings",TK_BOOL},
    {"is_alnum","strings",TK_BOOL},{"is_whitespace","strings",TK_BOOL},
    {"is_upper","strings",TK_BOOL},{"is_lower","strings",TK_BOOL},
    /* math (arg-dependent abs/neg/min/max/clamp handled by special case) */
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
    /* arrays (arg-dependent get_first/get_last/etc handled by inline dispatch) */
    {"append","arrays",TK_VOID},{"insert_at","arrays",TK_VOID},
    {"prepend","arrays",TK_VOID},{"fill","arrays",TK_VOID},
    {"remove_at","arrays",TK_VOID},{"remove","arrays",TK_VOID},
    {"remove_first","arrays",TK_VOID},{"remove_last","arrays",TK_VOID},
    {"sort_asc","arrays",TK_VOID},
    {"sort_desc","arrays",TK_VOID},{"clear","arrays",TK_VOID},
    {"concat","arrays",TK_ARRAY},{"deduplicate","arrays",TK_ARRAY},
    {"flatten","arrays",TK_ARRAY},{"reverse","arrays",TK_ARRAY},
    {"slice","arrays",TK_ARRAY},{"split_every","arrays",TK_ARRAY},
    {"pair","arrays",TK_ARRAY},
    {"map","arrays",TK_ARRAY},{"filter","arrays",TK_ARRAY},
    {"reduce","arrays",TK_UNKNOWN},
    {"get_first","arrays",TK_UNKNOWN},{"get_last","arrays",TK_UNKNOWN},
    {"get_sum","arrays",TK_INT},{"get_min","arrays",TK_INT},
    {"get_max","arrays",TK_INT},{"count","arrays",TK_INT},
    {"index_of","arrays",TK_INT},
    {"is_empty","arrays",TK_BOOL},{"contains","arrays",TK_BOOL},
    {"is_equal","arrays",TK_BOOL},
    {"any","arrays",TK_BOOL},{"all","arrays",TK_BOOL},
    /* maps (arg-dependent get_keys/get_values handled by special case) */
    {"has_key","maps",TK_BOOL},{"is_empty","maps",TK_BOOL},
    {"contains_value","maps",TK_BOOL},{"remove_key","maps",TK_VOID},
    {"clear","maps",TK_VOID},{"is_equal","maps",TK_BOOL},
    {"merge","maps",TK_MAP},{"get_or_default","maps",TK_UNKNOWN},
    /* random (arg-dependent choice/shuffle/sample handled by special case) */
    {"rand_float","random",TK_FLOAT},{"rand_int","random",TK_INT},
    {"rand_bool","random",TK_BOOL},{"rand_byte","random",TK_INT},
    {"rand_char","random",TK_INT},
    {"seed","random",TK_VOID},
    /* encoding */
    {"base64_encode","encoding",TK_STRING},{"base64_decode","encoding",TK_STRING},
    {"hex_encode","encoding",TK_STRING},{"hex_decode","encoding",TK_STRING},
    {"url_encode","encoding",TK_STRING},{"url_decode","encoding",TK_STRING},
    /* crypto */
    {"sha256","crypto",TK_STRING},{"md5","crypto",TK_STRING},
    {"random_hex","crypto",TK_STRING},
    /* regex */
    {"is_match","regex",TK_BOOL},{"is_valid","regex",TK_BOOL},
    {"find","regex",TK_STRING},{"replace","regex",TK_STRING},
    {"find_all","regex",TK_ARRAY},{"split","regex",TK_ARRAY},
    /* json */
    {"is_valid","json",TK_BOOL},{"decode","json",TK_MAP},
    {"encode","json",TK_STRING},{"stringify","json",TK_STRING},
    {"parse","json",TK_UNKNOWN},{"pretty_print","json",TK_STRING},
    /* io */
    {"read_file","io",TK_STRING},{"read_bytes","io",TK_ARRAY},
    {"read_lines","io",TK_ARRAY},{"write_file","io",TK_BOOL},
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
    /* os */
    {"args","os",TK_ARRAY},{"get_env","os",TK_STRING},
    {"set_env","os",TK_VOID},{"unset_env","os",TK_VOID},{"current_dir","os",TK_STRING},
    {"hostname","os",TK_STRING},{"arch","os",TK_STRING},
    {"current_os","os",TK_INT},{"pid","os",TK_INT},
    {"exec","os",TK_BOOL},
    /* time */
    {"now","time",TK_INT},{"now_ms","time",TK_INT},{"now_ns","time",TK_INT},
    {"tick","time",TK_INT},{"elapsed_ms","time",TK_INT},{"diff","time",TK_INT},
    {"year","time",TK_INT},{"month","time",TK_INT},{"day","time",TK_INT},
    {"hour","time",TK_INT},{"minute","time",TK_INT},{"second","time",TK_INT},
    {"weekday","time",TK_INT},
    {"format","time",TK_STRING},{"to_iso","time",TK_STRING},
    {"date","time",TK_STRING},{"to_clock","time",TK_STRING},
    /* strconv */
    {"to_int","strconv",TK_INT},{"to_uint","strconv",TK_UINT},
    {"to_float","strconv",TK_FLOAT},{"to_bool","strconv",TK_BOOL},
    {"from_int","strconv",TK_STRING},{"from_uint","strconv",TK_STRING},
    {"from_float","strconv",TK_STRING},{"from_bool","strconv",TK_STRING},
    {"is_numeric","strconv",TK_BOOL},{"is_integer","strconv",TK_BOOL},
    /* uuid */
    {"generate_hyphenated","uuid",TK_UNKNOWN},{"generate","uuid",TK_UNKNOWN},
    {"generate_compact","uuid",TK_STRING},
    {"generate_random","uuid",TK_UNKNOWN},
    {"generate_time_ordered","uuid",TK_UNKNOWN},
    {"parse","uuid",TK_UNKNOWN},
    {"is_valid","uuid",TK_BOOL},{"to_string","uuid",TK_STRING},
    /* bytes */
    {"from_string","bytes",TK_ARRAY},{"from_hex","bytes",TK_ARRAY},
    {"from_base64","bytes",TK_ARRAY},
    {"to_string","bytes",TK_STRING},{"to_hex","bytes",TK_STRING},
    {"to_base64","bytes",TK_STRING},
    /* binary */
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
    /* csv */
    {"parse","csv",TK_ARRAY},{"read_file","csv",TK_ARRAY},
    {"headers","csv",TK_ARRAY},
    {"write","csv",TK_BOOL},{"write_file","csv",TK_BOOL},
    {"encode","csv",TK_STRING},
    /* sqlite */
    {"open","sqlite",TK_UNKNOWN},{"close","sqlite",TK_VOID},
    {"exec","sqlite",TK_BOOL},{"query","sqlite",TK_ARRAY},
    /* threads */
    {"spawn","threads",TK_UNKNOWN},{"spawn_arg","threads",TK_UNKNOWN},
    {"join","threads",TK_VOID},
    {"get_id","threads",TK_INT},
    {"detach","threads",TK_VOID},{"is_alive","threads",TK_BOOL},
    {"current","threads",TK_INT},{"yield","threads",TK_VOID},
    {"sleep","threads",TK_VOID},
    {"thread_count","threads",TK_INT},
    /* sync */
    {"mutex","sync",TK_UNKNOWN},{"lock","sync",TK_VOID},
    {"unlock","sync",TK_VOID},{"try_lock","sync",TK_BOOL},
    {"destroy","sync",TK_VOID},
    /* atomic */
    {"load","atomic",TK_INT},{"store","atomic",TK_VOID},
    {"add","atomic",TK_INT},{"sub","atomic",TK_INT},
    {"exchange","atomic",TK_INT},{"cas","atomic",TK_BOOL},
    {"and","atomic",TK_INT},{"or","atomic",TK_INT},{"xor","atomic",TK_INT},
    {"spinlock","atomic",TK_UNKNOWN},{"spinlock_destroy","atomic",TK_VOID},
    {"spin_lock","atomic",TK_VOID},
    {"spin_trylock","atomic",TK_BOOL},{"spin_unlock","atomic",TK_VOID},
    {"fence","atomic",TK_VOID},
    /* channels */
    {"open","channels",TK_UNKNOWN},{"send","channels",TK_VOID},
    {"receive","channels",TK_INT},{"close","channels",TK_VOID},
    {"try_send","channels",TK_BOOL},{"try_receive","channels",TK_INT},
    /* server */
    {"add_router","server",TK_UNKNOWN},{"add_route","server",TK_VOID},
    {"listen","server",TK_VOID},{"cors","server",TK_VOID},
    {"use","server",TK_VOID},{"text","server",TK_UNKNOWN},
    {"json","server",TK_UNKNOWN},{"html","server",TK_UNKNOWN},
    {"redirect","server",TK_UNKNOWN},
    /* http */
    {"get","http",TK_UNKNOWN},{"post","http",TK_UNKNOWN},
    {"put","http",TK_UNKNOWN},{"delete","http",TK_UNKNOWN},
    {"head","http",TK_UNKNOWN},{"patch","http",TK_UNKNOWN},
    /* net */
    {"listen","net",TK_UNKNOWN},{"connect","net",TK_UNKNOWN},
    {"accept","net",TK_UNKNOWN},{"send","net",TK_INT},
    {"receive","net",TK_STRING},{"resolve","net",TK_STRING},
    {"close","net",TK_VOID},{"set_timeout","net",TK_VOID},
    /* fmt */
    {"sprintf","fmt",TK_STRING},{"format","fmt",TK_STRING},
    {"pad_left","fmt",TK_STRING},{"pad_right","fmt",TK_STRING},
    {"center","fmt",TK_STRING},{"int_to_hex","fmt",TK_STRING},
    {"int_to_binary","fmt",TK_STRING},{"int_to_octal","fmt",TK_STRING},
    {"float_fixed","fmt",TK_STRING},{"float_sci","fmt",TK_STRING},
    {"printf","fmt",TK_VOID},
    {"printfln","fmt",TK_VOID},{"eprintf","fmt",TK_VOID},{"eprintfln","fmt",TK_VOID},
    {"sprintfln","fmt",TK_STRING},
    /* mem */
    {"arena","mem",TK_UNKNOWN},{"usage","mem",TK_INT},
    {"free","mem",TK_VOID},{"reset","mem",TK_VOID},
    {"destroy","mem",TK_VOID},
    {"init","mem",TK_UNKNOWN},{"alloc","mem",TK_UNKNOWN},
    {"make","mem",TK_UNKNOWN},
    {NULL,NULL,TK_UNKNOWN}
};

/* : stdlib constants reachable via `import and use` / `using`. */
typedef struct { const char *name; const char *mod; TypeKind ret; } UsingConst;
static const UsingConst _using_consts[] = {
    {"PI","math",TK_FLOAT},{"E","math",TK_FLOAT},{"TAU","math",TK_FLOAT},
    {"PHI","math",TK_FLOAT},{"SQRT2","math",TK_FLOAT},{"LN2","math",TK_FLOAT},
    {"LN10","math",TK_FLOAT},{"INF","math",TK_FLOAT},{"NEG_INF","math",TK_FLOAT},
    {"EPSILON","math",TK_FLOAT},
    {"MAC_OS","os",TK_INT},{"LINUX","os",TK_INT},{"WINDOWS","os",TK_INT},{"OTHER","os",TK_INT},
    {"O_RDONLY","io",TK_INT},{"O_WRONLY","io",TK_INT},{"O_RDWR","io",TK_INT},
    {"BASE_2","strconv",TK_INT},{"BASE_8","strconv",TK_INT},{"BASE_10","strconv",TK_INT},
    {"BASE_16","strconv",TK_INT},{"BASE_36","strconv",TK_INT},
    {"NIL_UUID","uuid",TK_STRUCT},
    {NULL,NULL,TK_UNKNOWN}
};

/* Find the index of a module name in tc->imported_modules[], or -1. */
static int tc_find_import_index(TypeChecker *tc, const char *mod) {
    const char *real = tc_resolve_alias(tc, mod);
    for (int mi = 0; mi < tc->import_count; mi++) {
        if (strcmp(tc->imported_modules[mi], mod) == 0 ||
            strcmp(tc->imported_modules[mi], real) == 0)
            return mi;
    }
    return -1;
}

/* Single-pass lookup: marks import used and returns the type (NULL = not found). */
static GrayType *tc_lookup_using_constant(TypeChecker *tc, const char *name) {
    for (int ui = 0; ui < tc->using_module_count; ui++) {
        if (!using_module_accessible(tc, ui)) continue;
        const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
        for (int ci = 0; _using_consts[ci].name; ci++) {
            if (strcmp(name, _using_consts[ci].name) == 0 &&
                strcmp(real_mod, _using_consts[ci].mod) == 0) {
                int mi = tc->using_module_import_indices[ui];
                if (mi >= 0) tc->import_used[mi] = true;
                switch (_using_consts[ci].ret) {
                case TK_FLOAT: return &TYPE_FLOAT;
                case TK_INT:   return &TYPE_INT;
                case TK_STRING: return &TYPE_STRING;
                default:       return &TYPE_UNKNOWN;
                }
            }
        }
    }
    return NULL;
}

/* --- Enum helpers --- */

static void register_enum(TypeChecker *tc, const char *name,
    const char *display_name, bool is_string,
    const char **values, int value_count,
    const char ***payload_types, int *payload_counts, bool is_tagged) {
    if (tc->enum_count >= tc->enum_cap) {
        tc->enum_cap = tc->enum_cap ? tc->enum_cap * 2 : 8;
        tc->enum_names = xrealloc(tc->enum_names, sizeof(const char *) * tc->enum_cap);
        tc->enum_display_names = xrealloc(tc->enum_display_names, sizeof(const char *) * tc->enum_cap);
        tc->enum_is_string = xrealloc(tc->enum_is_string, sizeof(bool) * tc->enum_cap);
        tc->enum_values = xrealloc(tc->enum_values, sizeof(const char **) * tc->enum_cap);
        tc->enum_value_counts = xrealloc(tc->enum_value_counts, sizeof(int) * tc->enum_cap);
        tc->enum_payload_types = xrealloc(tc->enum_payload_types, sizeof(const char ***) * tc->enum_cap);
        tc->enum_payload_counts = xrealloc(tc->enum_payload_counts, sizeof(int *) * tc->enum_cap);
        tc->enum_is_tagged = xrealloc(tc->enum_is_tagged, sizeof(bool) * tc->enum_cap);
    }
    tc->enum_names_sorted_built = false;
    tc->enum_names[tc->enum_count] = name;
    tc->enum_display_names[tc->enum_count] = display_name ? display_name : name;
    tc->enum_is_string[tc->enum_count] = is_string;
    tc->enum_values[tc->enum_count] = values;
    tc->enum_value_counts[tc->enum_count] = value_count;
    tc->enum_payload_types[tc->enum_count] = payload_types;
    tc->enum_payload_counts[tc->enum_count] = payload_counts;
    tc->enum_is_tagged[tc->enum_count] = is_tagged;
    tc->enum_count++;
}

/* Resolve a type name, returning TK_ENUM for known enum names instead of
 * the default TK_STRUCT that type_from_name() produces for uppercase names. */
static GrayType *tc_type_from_name(TypeChecker *tc, const char *name) {
    if (name && is_enum_name(tc, name)) return type_enum(name);
    GrayType *t = type_from_name(name);
    /* : try prefixed type names from using-modules so bare
     * "Point" resolves to "shapes_Point" when shapes is using'd.
     * type_from_name returns TK_STRUCT for any capitalized name
     * even if the struct isn't registered, so check is_struct_name
     * to see if the bare name actually exists before giving up. */
    if (name && name[0] >= 'A' && name[0] <= 'Z' &&
        !is_struct_name(tc, name) && !is_enum_name(tc, name)) {
        for (int ui = 0; ui < tc->using_module_count; ui++) {
            if (!using_module_accessible(tc, ui)) continue;
            const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
            char prefixed[MSG_BUF_SIZE];
            snprintf(prefixed, sizeof(prefixed), "%s_%s", real_mod, name);
            if (is_enum_name(tc, prefixed)) return type_enum(prefixed);
            if (is_struct_name(tc, prefixed)) return type_struct(prefixed);
        }
        /* : after registration completes, reject uppercase names that
         * aren't registered as structs or enums. During registration we must
         * allow forward references, so only enforce this in later passes.
         * Exempt built-in types that are mapped directly in codegen without
         * struct registration (e.g. Error → GrayError*). */
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

/* Rank for named integer types; 0 = not a named integer type.
 * Used to detect narrowing (declared rank < value rank). */
static int int_name_rank(const char *n) {
    if (!n) return 0;
    if (strcmp(n, "i8")   == 0 || strcmp(n, "u8")   == 0 || strcmp(n, "byte") == 0) return 1;
    if (strcmp(n, "i16")  == 0 || strcmp(n, "u16")  == 0) return 2;
    if (strcmp(n, "i32")  == 0 || strcmp(n, "u32")  == 0) return 3;
    if (strcmp(n, "i64")  == 0 || strcmp(n, "u64")  == 0 ||
        strcmp(n, "int")  == 0 || strcmp(n, "uint") == 0) return 4;
    if (strcmp(n, "i128") == 0 || strcmp(n, "u128") == 0) return 5;
    if (strcmp(n, "i256") == 0 || strcmp(n, "u256") == 0) return 6;
    return 0;
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

static bool is_bigint_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "i128") == 0 || strcmp(tn, "u128") == 0 ||
           strcmp(tn, "i256") == 0 || strcmp(tn, "u256") == 0;
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
    if (node->kind == NODE_PREFIX_EXPR && node->data.prefix.op == TOK_MINUS &&
        node->data.prefix.right && node->data.prefix.right->kind == NODE_INT_VALUE) {
        *out = -node->data.prefix.right->data.int_value.value;
        return true;
    }
    /* Simple constant folding for literal +, -, * */
    if (node->kind == NODE_INFIX_EXPR) {
        int64_t lv, rv;
        if (try_get_literal_int(node->data.infix.left, &lv) &&
            try_get_literal_int(node->data.infix.right, &rv)) {
            if (node->data.infix.op == TOK_PLUS) { *out = lv + rv; return true; }
            if (node->data.infix.op == TOK_MINUS) { *out = lv - rv; return true; }
            if (node->data.infix.op == TOK_ASTERISK) { *out = lv * rv; return true; }
            if (node->data.infix.op == TOK_SLASH && rv != 0) { *out = lv / rv; return true; }
        }
    }
    return false;
}

/* Register a file-scope const integer value for later constant folding. */
static void tc_register_const_int(TypeChecker *tc, const char *name, int64_t value) {
    if (tc->const_int_count >= tc->const_int_cap) {
        tc->const_int_cap = tc->const_int_cap ? tc->const_int_cap * 2 : 8;
        tc->const_int_names = xrealloc(tc->const_int_names,
            sizeof(const char *) * (size_t)tc->const_int_cap);
        tc->const_int_values = xrealloc(tc->const_int_values,
            sizeof(int64_t) * (size_t)tc->const_int_cap);
    }
    tc->const_int_names[tc->const_int_count] = name;
    tc->const_int_values[tc->const_int_count] = value;
    tc->const_int_count++;
}

/* Resolve a non-numeric array size identifier in a fixed-size array type
 * string like "[int,SIZE]".  If the size field is already numeric this is
 * a no-op.  Otherwise the name is looked up in const_int_names/values and
 * the type string on the var_decl node is rewritten to its numeric form
 * so that downstream code (E3052/W3003, codegen extract_array_size) sees
 * only numeric size strings.
 *
 * Emits E3125 if the identifier is not a known const int.
 * Emits E3126 if the resolved value is <= 0. */
static void tc_resolve_array_size(TypeChecker *tc, AstNode *node) {
    const char *tn = node->data.var_decl.type_name;
    /* Find the top-level comma separating element type from size. */
    const char *size_comma = NULL;
    int depth = 0;
    for (const char *c = tn; *c; c++) {
        if (*c == '(' || *c == '[') depth++;
        else if (*c == ')' || *c == ']') depth--;
        else if (*c == ',' && depth == 1) { size_comma = c; break; }
    }
    if (!size_comma) return; /* no size field */

    /* Extract the size substring: after comma, before closing ']' */
    const char *size_start = size_comma + 1;
    const char *rbracket = strrchr(tn, ']');
    if (!rbracket || rbracket <= size_start) return;
    size_t sz_len = (size_t)(rbracket - size_start);
    char size_buf[256];
    if (sz_len >= sizeof(size_buf)) return;
    memcpy(size_buf, size_start, sz_len);
    size_buf[sz_len] = '\0';

    /* If already numeric, nothing to resolve. */
    char *endp = NULL;
    long val = strtol(size_buf, &endp, 10);
    if (endp && *endp == '\0') {
        (void)val;
        return;
    }

    /* Look up the identifier in the const int table. */
    bool found = false;
    int64_t resolved = 0;
    for (int i = 0; i < tc->const_int_count; i++) {
        if (strcmp(tc->const_int_names[i], size_buf) == 0) {
            resolved = tc->const_int_values[i];
            found = true;
            break;
        }
    }
    if (!found) {
        char *msg = tc_fmt(tc,
            "'%s' is not a compile-time integer constant; array size must be a const int/uint value",
            size_buf);
        diag_error_msg(tc->diag, "E3125", msg,
            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        return;
    }
    if (resolved <= 0) {
        char *msg = tc_fmt(tc,
            "array size must be greater than zero; '%s' resolves to %d",
            size_buf, (int)resolved);
        diag_error_msg(tc->diag, "E3126", msg,
            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        return;
    }

    /* Rewrite the type string with the resolved numeric value.
     * e.g. "[int,SIZE]" → "[int,5]" */
    size_t prefix_len = (size_t)(size_comma + 1 - tn);
    char num_buf[32];
    int num_len = snprintf(num_buf, sizeof(num_buf), "%d", (int)resolved);
    size_t new_len = prefix_len + (size_t)num_len + 2; /* +1 for ']' +1 for '\0' */
    char *new_tn = arena_alloc(tc->arena, new_len);
    memcpy(new_tn, tn, prefix_len);
    memcpy(new_tn + prefix_len, num_buf, (size_t)num_len);
    new_tn[prefix_len + (size_t)num_len] = ']';
    new_tn[prefix_len + (size_t)num_len + 1] = '\0';
    node->data.var_decl.type_name = new_tn;
}

/* Try to evaluate node as a compile-time integer constant.
 * Handles integer literals, negated literals, label references to known
 * file-scope const ints, and infix arithmetic.
 *
 * Returns true if the expression folded to *out with no overflow.
 * Returns false if:
 *   - any operand is not a known constant (*overflowed unchanged), or
 *   - arithmetic overflowed int64 (*overflowed set to true).
 */
static bool tc_fold_const_int(TypeChecker *tc, AstNode *node,
                               int64_t *out, bool *overflowed) {
    if (!node) return false;
    if (node->kind == NODE_INT_VALUE) {
        /* overflow_u64 means the literal exceeds uint64 range entirely;
         * overflow alone only means it exceeds int64 range (still valid
         * for unsigned Grayscale types).  Only treat overflow_u64 as a hard
         * failure here since we fold using the raw int64 bit pattern. */
        if (node->data.int_value.overflow_u64) { *overflowed = true; return false; }
        *out = node->data.int_value.value;
        return true;
    }
    if (node->kind == NODE_PREFIX_EXPR && node->data.prefix.op == TOK_MINUS &&
        node->data.prefix.right && node->data.prefix.right->kind == NODE_INT_VALUE) {
        if (node->data.prefix.right->data.int_value.overflow_u64) { *overflowed = true; return false; }
        *out = -node->data.prefix.right->data.int_value.value;
        return true;
    }
    if (node->kind == NODE_LABEL) {
        const char *name = node->data.label.value;
        for (int i = 0; i < tc->const_int_count; i++) {
            if (strcmp(tc->const_int_names[i], name) == 0) {
                *out = tc->const_int_values[i];
                return true;
            }
        }
        return false;
    }
    if (node->kind == NODE_INFIX_EXPR) {
        int64_t lv, rv;
        bool l_ov = false, r_ov = false;
        bool l_ok = tc_fold_const_int(tc, node->data.infix.left, &lv, &l_ov);
        bool r_ok = tc_fold_const_int(tc, node->data.infix.right, &rv, &r_ov);
        if (!l_ok || !r_ok) {
            if (l_ov || r_ov) *overflowed = true;
            return false;
        }
        TokenType op = node->data.infix.op;
        int64_t result;
        if (op == TOK_PLUS) {
            if (__builtin_add_overflow(lv, rv, &result)) { *overflowed = true; return false; }
            *out = result; return true;
        }
        if (op == TOK_MINUS) {
            if (__builtin_sub_overflow(lv, rv, &result)) { *overflowed = true; return false; }
            *out = result; return true;
        }
        if (op == TOK_ASTERISK) {
            if (__builtin_mul_overflow(lv, rv, &result)) { *overflowed = true; return false; }
            *out = result; return true;
        }
        if (op == TOK_SLASH && rv != 0) { *out = lv / rv; return true; }
        if (op == TOK_PERCENT && rv != 0) { *out = lv % rv; return true; }
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
        char msg[MSG_BUF_SIZE];
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

/* : shared void-expression guard. Emits E3038 at `expr` when `t`
 * is TK_VOID. `context` is a short phrase describing what the
 * position wants ("println argument", "map value", "binary operand",
 * etc.). If `expr` is a direct call to a named function, the error
 * quotes the function name; otherwise it falls back to a generic
 * "void expression" wording. Caller-suppliable context keeps each
 * diagnostic site self-describing without a zillion format strings. */
static AstNode *find_struct_in_program(AstNode *program, const char *name);

static void reject_void_in_context(TypeChecker *tc, AstNode *expr,
                                    GrayType *t, const char *context) {
    if (!t || t->kind != TK_VOID || !expr) return;
    char *msg = NULL;
    if (expr->kind == NODE_CALL_EXPR && expr->data.call.function &&
        expr->data.call.function->kind == NODE_LABEL) {
        msg = tc_fmt(tc,
            "cannot use void function '%s' as %s; the function does not return a value",
            expr->data.call.function->data.label.value, context);
    } else {
        msg = tc_fmt(tc,
            "cannot use void expression as %s; the expression does not produce a value",
            context);
    }
    diag_error_msg(tc->diag, "E3038", msg,
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

/* Resolve .VARIANT implicit enum selector using expected type context.
 * Sets node->data.implicit_enum.resolved_enum on success. */
static GrayType *resolve_implicit_enum(TypeChecker *tc, AstNode *node) {
    const char *variant = node->data.implicit_enum.variant;
    GrayType *expected = tc->expected_type;

    /* No type context — emit E3110 */
    if (!expected || expected->kind != TK_ENUM || !expected->name) {
        diag_error_codef(tc->diag, "E3110",
            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
            variant, variant);
        return &TYPE_UNKNOWN;
    }

    const char *enum_name = expected->name;

    /* Validate the variant exists in the enum */
    bool found = false;
    for (int ei = 0; ei < tc->enum_count; ei++) {
        if (strcmp(tc->enum_names[ei], enum_name) == 0) {
            for (int vi = 0; vi < tc->enum_value_counts[ei]; vi++) {
                if (strcmp(tc->enum_values[ei][vi], variant) == 0) {
                    found = true;
                    break;
                }
            }
            break;
        }
    }

    if (!found) {
        diag_error_codef(tc->diag, "E3047",
            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
            enum_name, variant);
        return &TYPE_UNKNOWN;
    }

    /* Resolve: write the enum name onto the node for codegen */
    node->data.implicit_enum.resolved_enum = enum_name;
    return type_enum(enum_name);
}

static GrayType *resolve_expr(TypeChecker *tc, AstNode *node) {
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
    GrayType *cached = typetable_get(tc->type_table, node);
    /* : bypass the cache entirely during the re-check pass
     * (suppress_typetable_writes). The re-check walks generic bodies
     * with concrete param bindings; stale TK_UNKNOWN entries from the
     * main pass prevent inner generic calls from resolving their
     * concrete bindings. */
    if (cached && !tc->suppress_typetable_writes)
        return cached;

    GrayType *result = &TYPE_UNKNOWN;

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
            GrayType *pt = resolve_expr(tc, part);
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
                    "cannot interpolate void expression; the function does not return a value",
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
                char *msg = NULL;
                if (pt->kind == TK_STRUCT && pt->name) {
                    msg = tc_fmt(tc,
                        "cannot interpolate struct value of type '%s'; format fields individually (e.g. \"${v.field}\")",
                        type_display_name(tc, pt));
                } else if (pt->kind == TK_POINTER) {
                    msg = tc_fmt(tc,
                        "cannot interpolate pointer value; dereference with ^ or format the pointee explicitly");
                } else {
                    msg = tc_fmt(tc,
                        "cannot interpolate function reference; call the function or format its result");
                }
                diag_error_msg(tc->diag, "E3041", msg,
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

        /* Type parameter name (e.g. T) — resolve as unknown during main
         * pass, or as the concrete binding during re-check.
         * Also handle "?" which is the rewritten form of T. */
        if (tc->type_param_name &&
            (strcmp(name, tc->type_param_name) == 0 ||
             strcmp(name, "?") == 0)) {
            if (tc->type_param_binding) {
                result = type_from_name(tc->type_param_binding);
            } else {
                result = &TYPE_UNKNOWN;
            }
            break;
        }

        /* Type names used as values are caught downstream; they won't match
         * any variable in scope, and functions like new(), mem.make(), and casts
         * legitimately take type names as arguments. */

        Symbol *sym = scope_lookup(tc->current_scope, name);
        /* Try using-module-prefixed name if not found */
        if (!sym) {
            for (int ui = 0; ui < tc->using_module_count; ui++) {
                if (!using_module_accessible(tc, ui)) continue;
                const char *umod = tc_resolve_alias(tc, tc->using_modules[ui]);
                char prefixed[MSG_BUF_SIZE];
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
        } else if (tc_lookup_using_constant(tc, name)) {
            result = tc_lookup_using_constant(tc, name);
        } else if (tc_is_builtin(name)) {
            GrayType *bt = type_from_name(name);
            if (bt != &TYPE_UNKNOWN) {
                /* Builtin type name (int, i128, float, etc.) used as a
                 * bare label — valid as an argument to size_of(Type). */
                result = bt;
            } else {
                /* Bare builtin function name used as a value (e.g. `input`
                 * instead of `input()`). */
                diag_error_codef(tc->diag, "E3031", NODE_FILE(tc, node),
                    node->token.line, node->token.column, 0, name, name, name);
            }
        } else if (!is_enum_name(tc, name) &&
                   !is_struct_name(tc, name) &&
                   !tc_is_imported_module(tc, name)) {
            /* Check if it looks like a number with a leading underscore */
            if (name[0] == '_' && name[1] >= '0' && name[1] <= '9') {
                diag_error_codef(tc->diag, "E1012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, name + 1);
            } else {
                char *msg = NULL;
                msg = tc_fmt(tc, "undefined variable '%s'", name);
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
                    char help[MSG_BUF_LARGE];
                    snprintf(help, sizeof(help),
                        "'%s' is a named return value in the function signature. Did you forget to declare 'mut %s %s' at function scope?",
                        name, name, nr_type ? nr_type : "?");
                    diag_error_help(tc->diag, "E4001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0, arena_strdup(tc->arena, help));
                } else {
                    const char *suggestion = suggest_name(tc, name);
                    if (suggestion) {
                        char help[MSG_BUF_SIZE];
                        snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                        diag_error_help(tc->diag, "E4001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0, arena_strdup(tc->arena, help));
                    } else {
                        diag_error_msg(tc->diag, "E4001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
            }
        }
        break;
    }

    case NODE_PREFIX_EXPR: {
        GrayType *right = resolve_expr(tc, node->data.prefix.right);
        if (node->data.prefix.op == TOK_BANG) {
            if (right->kind != TK_BOOL && right->kind != TK_UNKNOWN) {
                diag_error_codef(tc->diag, "E3090", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, right));
            }
            result = &TYPE_BOOL;
        } else if (node->data.prefix.op == TOK_MINUS) {
            if (right->kind != TK_UNKNOWN && !type_is_numeric(right)) {
                diag_error_codef(tc->diag, "E3007", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, right));
            } else if (right->kind != TK_UNKNOWN && right->name && is_unsigned_type(right->name)) {
                diag_error_codef(tc->diag, "E3096", NODE_FILE(tc, node), node->token.line, node->token.column, 0, right->name);
            }
            result = right;
        } else if (node->data.prefix.op == TOK_BIT_NOT) {
            /* E3090: bit_not requires an integer operand */
            if (right->kind != TK_UNKNOWN && !is_int_kind(right->kind) && right->kind != TK_CHAR) {
                diag_error_codef(tc->diag, "E8002", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, right));
            }
            result = right;
        } else {
            result = right;
        }
        break;
    }

    case NODE_INFIX_EXPR: {
        TokenType op = node->data.infix.op;
        GrayType *left = resolve_expr(tc, node->data.infix.left);
        /* For == and !=, set expected_type so .VARIANT on the RHS
         * can resolve against the LHS enum type. */
        GrayType *saved_infix_expected = tc->expected_type;
        if ((op == TOK_EQ || op == TOK_NOT_EQ) &&
            left && left->kind == TK_ENUM && left->name)
            tc->expected_type = left;
        GrayType *right = resolve_expr(tc, node->data.infix.right);
        tc->expected_type = saved_infix_expected;

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
            (op == TOK_LT || op == TOK_GT ||
             op == TOK_LT_EQ || op == TOK_GT_EQ)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot use '%s' on strings; use strings.compare() instead", op_to_str(op));
            diag_error_msg(tc->diag, "E3002", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            result = &TYPE_BOOL;
            break;
        }

        /* E3002: modulo on float */
        if (op == TOK_PERCENT &&
            (left->kind == TK_FLOAT || right->kind == TK_FLOAT)) {
            diag_error_msg(tc->diag, "E3002",
                "modulo (%) only works on integers, not floats",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3002: literal divide/modulo by zero (). Catches the
         * statically-detectable case where the RHS is an integer or
         * float literal zero (including a prefix -0). Runtime checks
         * still cover the dynamic case. */
        if (op == TOK_SLASH || op == TOK_PERCENT) {
            AstNode *r = node->data.infix.right;
            bool is_zero = false;
            int64_t iv;
            if (try_get_literal_int(r, &iv) && iv == 0) {
                is_zero = true;
            } else if (r && r->kind == NODE_FLOAT_VALUE &&
                       r->data.float_value.value == 0.0) {
                is_zero = true;
            } else if (r && r->kind == NODE_PREFIX_EXPR &&
                       r->data.prefix.op == TOK_MINUS &&
                       r->data.prefix.right &&
                       r->data.prefix.right->kind == NODE_FLOAT_VALUE &&
                       r->data.prefix.right->data.float_value.value == 0.0) {
                is_zero = true;
            }
            if (is_zero) {
                char *msg;
                msg = tc_fmt(tc,
                    "%s by zero; dividing by a literal zero is always invalid",
                    op == TOK_PERCENT ? "modulo" : "division");
                diag_error_msg(tc->diag, "E3002", msg,
                    NODE_FILE(tc, r), r->token.line, r->token.column, 0);
                infix_errored = true;
            }
        }

        /* E3002: bool used in arithmetic (e.g., 1 + true) */
        if ((left->kind == TK_BOOL || right->kind == TK_BOOL) &&
            (op == TOK_PLUS || op == TOK_MINUS ||
             op == TOK_ASTERISK || op == TOK_SLASH ||
             op == TOK_PERCENT) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "invalid operands: cannot use '%s' with %s and %s",
                op_to_str(op), type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3002", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3002: incompatible bigint operands. Bigint arithmetic and comparison
         * functions (gray_i128_add_checked, gray_i128_eq, etc.) only accept their
         * own struct type. The only cross-type bigint combinations that codegen
         * can handle are i256+i128 and u256+u128 (the narrower type is widened).
         * Everything else — mixed signedness or the wrong direction — leaks a
         * C type error and must be caught here. */
        if (!infix_errored &&
            (op == TOK_PLUS || op == TOK_MINUS ||
             op == TOK_ASTERISK || op == TOK_SLASH || op == TOK_PERCENT ||
             op == TOK_EQ || op == TOK_NOT_EQ ||
             op == TOK_LT || op == TOK_GT ||
             op == TOK_LT_EQ || op == TOK_GT_EQ) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN &&
            left->name && right->name &&
            is_bigint_type(left->name) && is_bigint_type(right->name) &&
            strcmp(left->name, right->name) != 0 &&
            !(strcmp(left->name, "i256") == 0 && strcmp(right->name, "i128") == 0) &&
            !(strcmp(left->name, "u256") == 0 && strcmp(right->name, "u128") == 0)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "invalid operands: cannot use '%s' with %s and %s; bigint types must match",
                op_to_str(op), type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3002", msg,
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
            op != TOK_EQ && op != TOK_NOT_EQ) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot use nil with operator '%s'; nil is only valid for == / != against nullable types (Error, pointers)",
                op_to_str(op));
            diag_error_msg(tc->diag, "E3002", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3092: nil compared to a non-nullable type */
        if (!infix_errored && (op == TOK_EQ || op == TOK_NOT_EQ)) {
            GrayType *non_nil = NULL;
            if (left->kind == TK_NIL && right->kind != TK_UNKNOWN && right->kind != TK_NIL &&
                right->kind != TK_POINTER && right->kind != TK_ERROR)
                non_nil = right;
            else if (right->kind == TK_NIL && left->kind != TK_UNKNOWN && left->kind != TK_NIL &&
                left->kind != TK_POINTER && left->kind != TK_ERROR)
                non_nil = left;
            if (non_nil) {
                diag_error_codef(tc->diag, "E3092", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    type_display_name(tc, non_nil));
                infix_errored = true;
            }
        }

        /* String + string: reject with helpful message */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) && op == TOK_PLUS) {
            diag_error_code_help(tc->diag, "E3048", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "use string interpolation \"${a}${b}\" or fmt.format() to combine strings");
            infix_errored = true;
        }

        /* Arithmetic on strings (-, *, /, %, etc.) */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) &&
            op != TOK_PLUS && op != TOK_EQ && op != TOK_NOT_EQ &&
            op != TOK_IN && op != TOK_NOT_IN) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot use '%s' on string type", op_to_str(op));
            diag_error_msg(tc->diag, "E3002", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3093: arithmetic on map, array, or struct */
        if (!infix_errored &&
            (op == TOK_PLUS || op == TOK_MINUS ||
             op == TOK_ASTERISK || op == TOK_SLASH || op == TOK_PERCENT)) {
            GrayType *bad = NULL;
            if (left->kind == TK_MAP || left->kind == TK_ARRAY || left->kind == TK_STRUCT)
                bad = left;
            else if (right->kind == TK_MAP || right->kind == TK_ARRAY || right->kind == TK_STRUCT)
                bad = right;
            if (bad && bad->kind != TK_UNKNOWN) {
                diag_error_codef(tc->diag, "E3093", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    op_to_str(op), type_display_name(tc, bad));
                infix_errored = true;
            }
        }

        /* E3078: pointer arithmetic is not supported. The spec disallows
         * it (no contiguous-buffer guarantee on '^T'), and without this
         * check 'p + 1' silently emitted C pointer math and produced
         * garbage. Equality (== / !=) against nil or another pointer is
         * still allowed. */
        if ((left->kind == TK_POINTER || right->kind == TK_POINTER) &&
            (op == TOK_PLUS || op == TOK_MINUS ||
             op == TOK_ASTERISK || op == TOK_SLASH || op == TOK_PERCENT)) {
            diag_error_code(tc->diag, "E3078",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3120: pointer ordering comparisons are not supported. STANDARD.md
         * §5.2.2 allows only == and != on pointers; <, >, <=, >= silently
         * fell through to C where cross-allocation ordering is undefined. */
        if (!infix_errored &&
            (left->kind == TK_POINTER || right->kind == TK_POINTER) &&
            (op == TOK_LT || op == TOK_GT ||
             op == TOK_LT_EQ || op == TOK_GT_EQ)) {
            diag_error_code(tc->diag, "E3120",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            infix_errored = true;
        }

        /* E3032: different enum types in comparison — catches both
         * direct enum literals (Color.RED == Dir.NORTH) and variables
         * of different enum types (c == d where c:Color, d:Dir). */
        if (!infix_errored &&
            (op == TOK_EQ || op == TOK_NOT_EQ) &&
            left->kind == TK_ENUM && right->kind == TK_ENUM &&
            left->name && right->name &&
            strcmp(left->name, right->name) != 0) {
            diag_error_codef(tc->diag, "E3032", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                enum_display_name(tc, left->name), enum_display_name(tc, right->name));
            infix_errored = true;
        }

        /* E3124: == / != on tagged enums. Tagged enums are emitted as C
         * structs (union + tag field) and cannot be compared with ==.
         * Reject at the Grayscale level before C is ever invoked. */
        if (!infix_errored &&
            (op == TOK_EQ || op == TOK_NOT_EQ) &&
            left->kind == TK_ENUM && right->kind == TK_ENUM &&
            left->name) {
            int eidx = -1;
            for (int ei = 0; ei < tc->enum_count; ei++)
                if (strcmp(tc->enum_names[ei], left->name) == 0) { eidx = ei; break; }
            if (eidx >= 0 && tc->enum_is_tagged[eidx]) {
                diag_error_codef(tc->diag, "E3124",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    op_to_str(op), enum_display_name(tc, left->name));
                infix_errored = true;
            }
        }

        /* E3049: arithmetic and ordering on enum values — catch both
         * direct enum literals (Color.RED + 1) and variables of enum
         * type (c + 1).  Enums only support == and != comparison.
         * Exception: ordering comparisons (< > <= >=) are allowed when
         * one side is an integer variable (user has explicitly unboxed
         * the enum value into an int for numeric comparison). */
        if (!infix_errored &&
            (op == TOK_PLUS || op == TOK_MINUS ||
             op == TOK_ASTERISK || op == TOK_SLASH || op == TOK_PERCENT ||
             op == TOK_LT || op == TOK_GT ||
             op == TOK_LT_EQ || op == TOK_GT_EQ)) {
            bool left_is_enum = (left && left->kind == TK_ENUM);
            bool right_is_enum = (right && right->kind == TK_ENUM);
            bool is_ordering = (op == TOK_LT || op == TOK_GT ||
                                op == TOK_LT_EQ || op == TOK_GT_EQ);
            bool has_int_side = (left && left->kind == TK_INT) ||
                                (right && right->kind == TK_INT);
            if ((left_is_enum || right_is_enum) &&
                !(is_ordering && has_int_side)) {
                diag_error_codef(tc->diag, "E3049", NODE_FILE(tc, node), node->token.line, node->token.column, 0, op_to_str(op));
                infix_errored = true;
            }
        }

        /* Enum vs integer comparison: reject with a specific diagnostic. */
        if ((op == TOK_EQ || op == TOK_NOT_EQ) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            if (left->kind == TK_ENUM && is_int_kind(right->kind)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot compare enum '%s' with %s; use an enum variant like '%s.VARIANT'",
                    type_display_name(tc, left), type_name(right), type_display_name(tc, left));
                diag_error_msg(tc->diag, "E3117", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            if (is_int_kind(left->kind) && right->kind == TK_ENUM) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot compare %s with enum '%s'; use an enum variant like '%s.VARIANT'",
                    type_name(left), type_display_name(tc, right), type_display_name(tc, right));
                diag_error_msg(tc->diag, "E3117", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* Comparison of incompatible types (e.g., int == string) */
        if ((op == TOK_EQ || op == TOK_NOT_EQ) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN &&
            left->kind != right->kind && left->kind != TK_NIL && right->kind != TK_NIL &&
            !(is_int_kind(left->kind) && is_int_kind(right->kind)) &&
            !(left->kind == TK_STRUCT && is_int_kind(right->kind)) &&
            !(is_int_kind(left->kind) && right->kind == TK_STRUCT) &&
            !(is_int_kind(left->kind) && right->kind == TK_BOOL) &&
            !(left->kind == TK_BOOL && is_int_kind(right->kind)) &&
            !(left->kind == TK_ENUM && is_int_kind(right->kind)) &&
            !(is_int_kind(left->kind) && right->kind == TK_ENUM) &&
            /* String enums can be compared with string literals */
            !(left->kind == TK_ENUM && right->kind == TK_STRING && tc_enum_is_string(tc, left->name)) &&
            !(left->kind == TK_STRING && right->kind == TK_ENUM && tc_enum_is_string(tc, right->name))) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot compare %s with %s", type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3001", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        /* Pointer-to-pointer: pointee types differ in == / != comparison */
        if ((op == TOK_EQ || op == TOK_NOT_EQ) &&
            left->kind == TK_POINTER && right->kind == TK_POINTER &&
            left->name && right->name &&
            strcmp(left->name, right->name) != 0 &&
            strcmp(left->name, "nil") != 0 && strcmp(right->name, "nil") != 0) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot compare %s with %s", type_name(left), type_name(right));
            diag_error_msg(tc->diag, "E3001", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E3074: arrays cannot be compared with == / != directly. The C
         * backend has no structural-equality operator on aggregate types,
         * so this used to slip through to clang. Point users at
         * arrays.is_equal. */
        if ((op == TOK_EQ || op == TOK_NOT_EQ ||
             op == TOK_LT || op == TOK_LT_EQ ||
             op == TOK_GT || op == TOK_GT_EQ) &&
            left->kind == TK_ARRAY && right->kind == TK_ARRAY) {
            diag_error_code_help(tc->diag, "E3074",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "use arrays.is_equal(a, b) to compare arrays element-by-element");
            infix_errored = true;
        }
        if ((op == TOK_EQ || op == TOK_NOT_EQ ||
             op == TOK_LT || op == TOK_LT_EQ ||
             op == TOK_GT || op == TOK_GT_EQ) &&
            left->kind == TK_MAP && right->kind == TK_MAP) {
            diag_error_code_help(tc->diag, "E3076",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "use maps.is_equal(a, b) to compare maps for equality");
            infix_errored = true;
        }
        if ((op == TOK_EQ || op == TOK_NOT_EQ ||
             op == TOK_LT || op == TOK_LT_EQ ||
             op == TOK_GT || op == TOK_GT_EQ) &&
            left->kind == TK_STRUCT && right->kind == TK_STRUCT) {
            diag_error_code_help(tc->diag, "E3077",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "compare individual fields instead, e.g. a.x == b.x");
            infix_errored = true;
        }

        /* E3085: validate type compatibility for in/not_in/!in */
        if ((op == TOK_IN || op == TOK_NOT_IN) &&
            !infix_errored && left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            bool mismatch = false;
            const char *left_tn = type_name(left);
            const char *right_tn = type_name(right);
            if (right->kind == TK_ARRAY && right->element_type) {
                GrayType *elem = type_from_name(right->element_type);
                if (elem->kind != TK_UNKNOWN && left->kind != elem->kind &&
                    !(is_int_kind(left->kind) && is_int_kind(elem->kind)) &&
                    !(left->name && strcmp(left->name, right->element_type) == 0)) {
                    mismatch = true;
                }
            } else if (right->kind == TK_MAP && right->key_type) {
                GrayType *key = type_from_name(right->key_type);
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
            } else if (right->name && strcmp(right->name, "Range<int>") == 0) {
                /* range() produces Range<int>; only integer types can be checked */
                if (!is_int_kind(left->kind)) {
                    mismatch = true;
                }
            } else {
                /* RHS is not a valid target for 'in' */
                diag_error_codef(tc->diag, "E3095", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    right_tn);
                infix_errored = true;
            }
            if (mismatch) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'in' operator type mismatch: cannot check if '%s' is in '%s'",
                    left_tn, right_tn);
                diag_error_msg(tc->diag, "E3085", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                infix_errored = true;
            }
        }

        /* E3089: binary bitwise operators require integer operands */
        if ((op == TOK_BIT_AND || op == TOK_BIT_OR ||
             op == TOK_BIT_XOR || op == TOK_BIT_SHIFT_LEFT ||
             op == TOK_BIT_SHIFT_RIGHT) &&
            !infix_errored && left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            bool left_ok  = is_int_kind(left->kind)  || left->kind  == TK_CHAR;
            bool right_ok = is_int_kind(right->kind) || right->kind == TK_CHAR;
            if (!left_ok || !right_ok) {
                diag_error_codef(tc->diag, "E8001",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    op_to_str(op), type_display_name(tc, left), type_display_name(tc, right));
                infix_errored = true;
            } else {
                result = left;
            }
        }

        if (op == TOK_EQ || op == TOK_NOT_EQ ||
            op == TOK_LT || op == TOK_GT ||
            op == TOK_LT_EQ || op == TOK_GT_EQ ||
            op == TOK_AND || op == TOK_OR ||
            op == TOK_IN || op == TOK_NOT_IN) {
            result = &TYPE_BOOL;
        } else if (left->kind == TK_FLOAT || right->kind == TK_FLOAT) {
            result = &TYPE_FLOAT;
        } else if (left->kind == TK_STRING && op == TOK_PLUS) {
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
        GrayType *left_t = resolve_expr(tc, node->data.postfix.left);
        if (node->data.postfix.op == TOK_CARET) {
            if (left_t->kind == TK_POINTER) {
                /* Dereference: ^T^ → T */
                result = tc_type_from_name(tc, left_t->element_type);
            } else if (left_t->kind != TK_UNKNOWN) {
                diag_error_codef(tc->diag, "E3016", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, left_t));
                result = left_t;
            } else {
                result = left_t;
            }
        } else if (node->data.postfix.op == TOK_INCREMENT ||
                   node->data.postfix.op == TOK_DECREMENT) {
            /* E5015: ++ and -- require a variable, not a literal */
            if (node->data.postfix.left->kind != NODE_LABEL &&
                node->data.postfix.left->kind != NODE_INDEX_EXPR &&
                node->data.postfix.left->kind != NODE_MEMBER_EXPR) {
                diag_error_msg(tc->diag, "E5015",
                    "++ and -- require a variable; you cannot increment a literal or expression",
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
                diag_error_codef(tc->diag, "E5023", NODE_FILE(tc, node), node->token.line, node->token.column, 0, op_to_str(node->data.postfix.op), type_display_name(tc, left_t));
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
            /* Skip implicit enum nodes; they need expected_type context
             * from the function signature, which is resolved later. */
            if (node->data.call.args[i]->kind == NODE_IMPLICIT_ENUM)
                continue;
            resolve_expr(tc, node->data.call.args[i]);

            /* E3040: a multi-return call cannot appear in single-value
             * argument position. Caller must destructure first. */
            AstNode *arg = node->data.call.args[i];
            if (arg->kind == NODE_CALL_EXPR) {
                AstNode *afn = arg->data.call.function;
                FuncSig *asig = NULL;
                const char *aname = NULL;
                if (afn && afn->kind == NODE_LABEL) {
                    aname = afn->data.label.value;
                    asig = find_func(tc, aname);
                } else if (afn && afn->kind == NODE_MEMBER_EXPR &&
                           afn->data.member.object->kind == NODE_LABEL) {
                    const char *mod_raw = afn->data.member.object->data.label.value;
                    const char *mod = tc_resolve_alias(tc, mod_raw);
                    aname = afn->data.member.member;
                    char prefixed[MSG_BUF_SIZE];
                    snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, aname);
                    asig = find_func(tc, prefixed);
                }
                if (asig && asig->return_count > 1) {
                    diag_error_codef(tc->diag, "E3040",
                        NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0,
                        aname, asig->return_count, aname);
                }
            }
        }

        /* E3097: addr() of inner-scope variable passed alongside an outer-scope
         * argument — the classic escape pattern, e.g. arrays.append(outer, addr(x))
         * where x is loop/if/while-local.  We fire only when another argument in
         * the same call comes from an outer scope (scope_lookup_local returns null
         * for it) — that indicates the address is being stored in something that
         * outlives the inner variable's arena. */
        for (int i = 0; i < node->data.call.arg_count; i++) {
            AstNode *arg = node->data.call.args[i];
            if (arg->kind == NODE_CALL_EXPR &&
                arg->data.call.function &&
                arg->data.call.function->kind == NODE_LABEL &&
                strcmp(arg->data.call.function->data.label.value, "addr") == 0 &&
                arg->data.call.arg_count == 1 &&
                arg->data.call.args[0]->kind == NODE_LABEL) {
                const char *addr_var = arg->data.call.args[0]->data.label.value;
                if (!scope_lookup_local(tc->current_scope, addr_var)) continue;
                /* Check if any other argument is an outer-scope variable */
                bool has_outer_arg = false;
                for (int j = 0; j < node->data.call.arg_count; j++) {
                    if (j == i) continue;
                    AstNode *other = node->data.call.args[j];
                    if (other->kind == NODE_LABEL) {
                        const char *oname = other->data.label.value;
                        if (!scope_lookup_local(tc->current_scope, oname) &&
                            scope_lookup(tc->current_scope, oname)) {
                            has_outer_arg = true;
                            break;
                        }
                    }
                }
                if (has_outer_arg) {
                    diag_error_codef(tc->diag, "E3097",
                        NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0,
                        addr_var, addr_var);
                }
            }
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

        /* E5030: chained call on a function-call result — e.g. get_fn()(5).
         * A call result is never directly callable in Grayscale; func references
         * must be created with ()func_name or ref(func_name) first. */
        if (fn && fn->kind == NODE_CALL_EXPR) {
            AstNode *inner_fn = fn->data.call.function;
            const char *inner_name = NULL;
            char inner_buf[MSG_BUF_SIZE];
            if (inner_fn && inner_fn->kind == NODE_LABEL) {
                inner_name = inner_fn->data.label.value;
            } else if (inner_fn && inner_fn->kind == NODE_MEMBER_EXPR &&
                       inner_fn->data.member.object->kind == NODE_LABEL) {
                snprintf(inner_buf, sizeof(inner_buf), "%s.%s",
                    inner_fn->data.member.object->data.label.value,
                    inner_fn->data.member.member);
                inner_name = inner_buf;
            }
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot call the return value of '%s' directly; func references must be created with '()func_name' or 'ref(func_name)' before calling",
                inner_name ? inner_name : "function");
            diag_error_msg(tc->diag, "E5030", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            result = &TYPE_UNKNOWN;
            break;
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
                GrayType *elem_t = type_from_name(arr_sym->type->element_type);
                if (elem_t && elem_t->func_sig &&
                    elem_t->func_sig->return_count > 0 &&
                    elem_t->func_sig->return_types[0]) {
                    result = type_from_name(elem_t->func_sig->return_types[0]);
                    typetable_set(tc->type_table, node, result);
                    return result;
                }
            }
        }

        /* Tagged enum construction via implicit selector: .Circle(3.14) */
        if (fn->kind == NODE_IMPLICIT_ENUM) {
            const char *vname = fn->data.implicit_enum.variant;
            /* Resolve enum from expected_type */
            GrayType *et = tc->expected_type;
            if (et && et->kind == TK_ENUM && et->name) {
                fn->data.implicit_enum.resolved_enum = et->name;
                int eidx = -1;
                for (int ei = 0; ei < tc->enum_count; ei++) {
                    if (strcmp(tc->enum_names[ei], et->name) == 0) { eidx = ei; break; }
                }
                if (eidx >= 0) {
                    int vidx = -1;
                    for (int vi = 0; vi < tc->enum_value_counts[eidx]; vi++) {
                        if (strcmp(tc->enum_values[eidx][vi], vname) == 0) { vidx = vi; break; }
                    }
                    if (vidx < 0) {
                        diag_error_codef(tc->diag, "E3047", NODE_FILE(tc, node), node->token.line, node->token.column, 0, et->name, vname);
                    } else if (tc->enum_is_tagged[eidx]) {
                        int expected_pc = tc->enum_payload_counts[eidx][vidx];
                        int got_ac = node->data.call.arg_count;
                        if (expected_pc == 0 && got_ac > 0) {
                            diag_error_codef(tc->diag, "E3114", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vname, tc->enum_display_names[eidx]);
                        } else if (expected_pc != got_ac) {
                            diag_error_codef(tc->diag, "E3113", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vname, tc->enum_display_names[eidx], expected_pc, got_ac);
                        } else {
                            for (int ai = 0; ai < got_ac; ai++) {
                                GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                                GrayType *exp_t = tc_type_from_name(tc, tc->enum_payload_types[eidx][vidx][ai]);
                                if (arg_t && exp_t &&
                                    arg_t->kind != TK_UNKNOWN && exp_t->kind != TK_UNKNOWN &&
                                    arg_t->kind != exp_t->kind &&
                                    !(is_int_kind(arg_t->kind) && is_int_kind(exp_t->kind))) {
                                    diag_error_codef(tc->diag, "E3001", NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line, node->data.call.args[ai]->token.column, 0);
                                }
                            }
                        }
                    } else {
                        diag_error_codef(tc->diag, "E3115", NODE_FILE(tc, node), node->token.line, node->token.column, 0, tc->enum_display_names[eidx], vname);
                    }
                }
                result = type_enum(et->name);
            } else {
                diag_error_codef(tc->diag, "E3110", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vname, vname);
                result = &TYPE_UNKNOWN;
            }
            break;
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
            char prefixed[MSG_BUF_SIZE];
            snprintf(prefixed, sizeof(prefixed), "%s_%s_%s", mod_name, struct_name, func_name);
            FuncSig *sig = find_func(tc, prefixed);
            if (sig) {
                sig->used = true;
                result = sig->return_count > 0 ? sig->return_types[0] : &TYPE_VOID;
                /* Check argument count */
                if (node->data.call.arg_count != sig->param_count) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "function '%s.%s.%s' expects %d argument(s), got %d",
                        mod_name, struct_name, func_name,
                        sig->param_count, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
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
                                char emsg[MSG_BUF_SIZE];
                                snprintf(emsg, sizeof(emsg),
                                    "cannot pass constant '%s' to mutable parameter '%s' of '%s.%s.%s'",
                                    arg_sym->name, found_decl->data.func_decl.params[ai].name,
                                    mod_name, struct_name, func_name);
                                diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                    NODE_FILE(tc, arg), arg->token.line,
                                    arg->token.column, 0);
                            }
                        } else if (arg->kind == NODE_MEMBER_EXPR &&
                                   arg->data.member.object->kind == NODE_LABEL &&
                                   is_enum_name(tc, arg->data.member.object->data.label.value)) {
                            char emsg[MSG_BUF_SIZE];
                            snprintf(emsg, sizeof(emsg),
                                "cannot pass enum constant to mutable parameter '%s' of '%s.%s.%s'; expected a mutable variable",
                                found_decl->data.func_decl.params[ai].name,
                                mod_name, struct_name, func_name);
                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                NODE_FILE(tc, arg), arg->token.line,
                                arg->token.column, 0);
                        } else if (arg->kind != NODE_MEMBER_EXPR &&
                                   arg->kind != NODE_INDEX_EXPR &&
                                   arg->kind != NODE_PREFIX_EXPR) {
                            char emsg[MSG_BUF_SIZE];
                            snprintf(emsg, sizeof(emsg),
                                "cannot pass a literal or expression to mutable parameter '%s' of '%s.%s.%s'; expected a mutable variable",
                                found_decl->data.func_decl.params[ai].name,
                                mod_name, struct_name, func_name);
                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                NODE_FILE(tc, arg), arg->token.line,
                                arg->token.column, 0);
                        }
                    }
                }
            } else {
                result = &TYPE_VOID;
            }
        } else if (fn->kind == NODE_MEMBER_EXPR && fn->data.member.object->kind == NODE_LABEL &&
                   is_enum_name(tc, fn->data.member.object->data.label.value)) {
            /* Tagged enum construction: Shape.Circle(3.14) */
            const char *ename = fn->data.member.object->data.label.value;
            const char *vname = fn->data.member.member;
            int eidx = -1;
            for (int ei = 0; ei < tc->enum_count; ei++) {
                if (strcmp(tc->enum_names[ei], ename) == 0) { eidx = ei; break; }
            }
            if (eidx >= 0) {
                /* Find variant index */
                int vidx = -1;
                for (int vi = 0; vi < tc->enum_value_counts[eidx]; vi++) {
                    if (strcmp(tc->enum_values[eidx][vi], vname) == 0) { vidx = vi; break; }
                }
                if (vidx < 0) {
                    diag_error_codef(tc->diag, "E3047", NODE_FILE(tc, node), node->token.line, node->token.column, 0, ename, vname);
                    result = &TYPE_UNKNOWN;
                    break;
                }
                if (!tc->enum_is_tagged[eidx]) {
                    /* E3115: plain enum, can't call */
                    const char *dname = tc->enum_display_names[eidx];
                    diag_error_codef(tc->diag, "E3115", NODE_FILE(tc, node), node->token.line, node->token.column, 0, dname, vname);
                    result = type_enum(ename);
                    break;
                }
                int expected_pc = tc->enum_payload_counts[eidx][vidx];
                int got_ac = node->data.call.arg_count;
                if (expected_pc == 0 && got_ac > 0) {
                    diag_error_codef(tc->diag, "E3114", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vname, tc->enum_display_names[eidx]);
                } else if (expected_pc != got_ac) {
                    diag_error_codef(tc->diag, "E3113", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vname, tc->enum_display_names[eidx], expected_pc, got_ac);
                } else {
                    /* Validate each arg type against payload type */
                    for (int ai = 0; ai < got_ac; ai++) {
                        GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                        GrayType *expected_t = tc_type_from_name(tc, tc->enum_payload_types[eidx][vidx][ai]);
                        if (arg_t && expected_t &&
                            arg_t->kind != TK_UNKNOWN && expected_t->kind != TK_UNKNOWN &&
                            arg_t->kind != expected_t->kind &&
                            !(is_int_kind(arg_t->kind) && is_int_kind(expected_t->kind))) {
                            diag_error_codef(tc->diag, "E3001", NODE_FILE(tc, node->data.call.args[ai]),
                                node->data.call.args[ai]->token.line, node->data.call.args[ai]->token.column, 0);
                        }
                    }
                }
                result = type_enum(ename);
            } else {
                result = &TYPE_UNKNOWN;
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
            if (!mod_imported && is_stdlib_module_name(mod)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "module '%s' is not imported; add 'import @%s' at the top of the file",
                    mod, mod);
                diag_error_msg(tc->diag, "E4001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* c.func() without import c"..."; but only if "c" isn't a
             * local variable. A variable named `c` with a struct type
             * should fall through to the struct function dispatch, not
             * be treated as the C interop module (). */
            if (!mod_imported && strcmp(mod, "c") == 0 &&
                !scope_lookup(tc->current_scope, "c")) {
                diag_error_msg(tc->diag, "E4001",
                    "C interop requires a C header import; add import c\"header.h\" at the top of the file",
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
                    GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                    if (!arg_t || arg_t->kind == TK_UNKNOWN) continue;
                    /* Reject bigint types */
                    if (arg_t->name &&
                        (strcmp(arg_t->name, "i128") == 0 || strcmp(arg_t->name, "i256") == 0 ||
                         strcmp(arg_t->name, "u128") == 0 || strcmp(arg_t->name, "u256") == 0)) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "cannot pass %s to a C function; C has no 128/256-bit integer types",
                            arg_t->name);
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                    /* Reject Grayscale-specific composite types */
                    if (arg_t->kind == TK_ARRAY || arg_t->kind == TK_MAP) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "cannot pass %s to a C function; use individual elements instead",
                            arg_t->kind == TK_ARRAY ? "an array" : "a map");
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                    /* Reject Grayscale structs (registered in typechecker) */
                    if (arg_t->kind == TK_STRUCT && arg_t->name &&
                        is_struct_name(tc, arg_t->name)) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "cannot pass struct '%s' to a C function; pass individual fields instead",
                            type_display_name(tc, arg_t));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0);
                    }
                }
                result = &TYPE_UNKNOWN;
                break;
            }
            /* Skip stdlib dispatch if this module is a user import, not stdlib.
             * User modules with the same name (e.g., import "./server.gray") must
             * fall through to the user-module handler below. */
            if (!tc_is_stdlib_import(tc, mod_raw)) {
                goto user_module_dispatch;
            }
            /* E5034: named arguments are not supported for stdlib functions */
            if (tc_has_named_args(node)) {
                char fname[MSG_BUF_SIZE];
                snprintf(fname, sizeof(fname), "%s.%s", mod, mfn);
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "named arguments are not supported for builtin function '%s'",
                    fname);
                diag_error_msg(tc->diag, "E5034", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Validate argument count for stdlib function calls */
            tc_check_stdlib_arg_count(tc, mod, mfn, node);
            tc_check_stdlib_arg_types(tc, mod, mfn, node);
            tc_check_strconv_base(tc, mod, mfn, node);
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
                        GrayType *map_t = resolve_expr(tc, node->data.call.args[0]);
                        result = type_array(map_t && map_t->key_type ? map_t->key_type : "string");
                    } else result = type_array("string");
                } else if (strcmp(mfn, "get_values") == 0) {
                    if (node->data.call.arg_count > 0) {
                        GrayType *map_t = resolve_expr(tc, node->data.call.args[0]);
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
                    GrayType *arg0_t = typetable_get(tc->type_table, arg0);
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
                    GrayType *t0 = typetable_get(tc->type_table, a0);
                    GrayType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_MAP && t1->kind == TK_MAP) {
                        bool key_match = t0->key_type && t1->key_type &&
                            strcmp(t0->key_type, t1->key_type) == 0;
                        bool val_match = t0->value_type && t1->value_type &&
                            strcmp(t0->value_type, t1->value_type) == 0;
                        if (!key_match || !val_match) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "type mismatch: cannot compare map[%s:%s] with map[%s:%s]",
                                t0->key_type ? t0->key_type : "?",
                                t0->value_type ? t0->value_type : "?",
                                t1->key_type ? t1->key_type : "?",
                                t1->value_type ? t1->value_type : "?");
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                        }
                        const char *bad_member = NULL;
                        if (t0->value_type) {
                            GrayType *vt = type_from_name(t0->value_type);
                            if (vt->kind == TK_ARRAY || vt->kind == TK_MAP || vt->kind == TK_STRUCT)
                                bad_member = t0->value_type;
                        }
                        if (bad_member) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "maps.is_equal does not support maps with %s values; only primitive and string element types are supported",
                                bad_member);
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0);
                        }
                    }
                }
                if (strcmp(mfn, "contains_value") == 0 && node->data.call.arg_count >= 1) {
                    AstNode *a0 = node->data.call.args[0];
                    GrayType *t0 = typetable_get(tc->type_table, a0);
                    if (t0 && t0->kind == TK_MAP && t0->value_type) {
                        GrayType *vt = type_from_name(t0->value_type);
                        if (vt->kind == TK_ARRAY || vt->kind == TK_MAP || vt->kind == TK_STRUCT) {
                            diag_error_codef(tc->diag, "E12007",
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0,
                                t0->value_type);
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
                 * The .v0 access gets __auto_type in C, but the Grayscale type system
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
                } else if (strcmp(mfn, "read_bytes") == 0) {
                    result = type_array("byte");
                } else if (strcmp(mfn, "read_lines") == 0) {
                    result = type_array("string");
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
            } else if (strcmp(mod, "strconv") == 0) {
                if (strcmp(mfn, "to_int") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "to_uint") == 0) {
                    result = &TYPE_UINT;
                } else if (strcmp(mfn, "to_float") == 0) {
                    result = &TYPE_FLOAT;
                } else if (strcmp(mfn, "to_bool") == 0 ||
                           strcmp(mfn, "is_numeric") == 0 ||
                           strcmp(mfn, "is_integer") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "from_int") == 0 ||
                           strcmp(mfn, "from_uint") == 0 ||
                           strcmp(mfn, "from_float") == 0 ||
                           strcmp(mfn, "from_bool") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "strings") == 0) {
                if (strcmp(mfn, "contains") == 0 || strcmp(mfn, "starts_with") == 0 ||
                    strcmp(mfn, "ends_with") == 0 || strcmp(mfn, "is_empty") == 0 ||
                    strcmp(mfn, "is_alpha") == 0 || strcmp(mfn, "is_digit") == 0 ||
                    strcmp(mfn, "is_alnum") == 0 || strcmp(mfn, "is_whitespace") == 0 ||
                    strcmp(mfn, "is_upper") == 0 || strcmp(mfn, "is_lower") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "char_at") == 0) {
                    result = &TYPE_CHAR;
                } else if (strcmp(mfn, "to_chars") == 0) {
                    result = type_array("char");
                } else if (strcmp(mfn, "from_chars") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "index_of") == 0 ||
                           strcmp(mfn, "last_index_of") == 0 ||
                           strcmp(mfn, "count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "split") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "to_upper") == 0 || strcmp(mfn, "to_lower") == 0 ||
                           strcmp(mfn, "trim") == 0 || strcmp(mfn, "trim_left") == 0 ||
                           strcmp(mfn, "trim_right") == 0 ||
                           strcmp(mfn, "remove_prefix") == 0 || strcmp(mfn, "remove_suffix") == 0 ||
                           strcmp(mfn, "replace") == 0 ||
                           strcmp(mfn, "repeat") == 0 || strcmp(mfn, "reverse") == 0 ||
                           strcmp(mfn, "slice") == 0 || strcmp(mfn, "join") == 0) {
                    result = &TYPE_STRING;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E7004: strings.repeat() second arg must be integer */
                if (strcmp(mfn, "repeat") == 0 && node->data.call.arg_count >= 2) {
                    GrayType *count_t = typetable_get(tc->type_table, node->data.call.args[1]);
                    if (count_t && count_t->kind == TK_FLOAT) {
                        diag_error_msg(tc->diag, "E7004",
                            "strings.repeat() count must be an integer, not a float",
                            NODE_FILE(tc, node->data.call.args[1]), node->data.call.args[1]->token.line,
                            node->data.call.args[1]->token.column, 0);
                    }
                }
                /* E7004: strings.slice() bounds must be integers */
                if (strcmp(mfn, "slice") == 0 && node->data.call.arg_count >= 3) {
                    for (int si = 1; si <= 2 && si < node->data.call.arg_count; si++) {
                        GrayType *bt = typetable_get(tc->type_table, node->data.call.args[si]);
                        if (bt && bt->kind == TK_FLOAT) {
                            diag_error_msg(tc->diag, "E7004",
                                "strings.slice() bounds must be integers, not floats",
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
                           strcmp(mfn, "elapsed_ms") == 0 || strcmp(mfn, "diff") == 0 ||
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
                else if (strcmp(mfn, "generate_compact") == 0 ||
                         strcmp(mfn, "to_string") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "generate") == 0 ||
                         strcmp(mfn, "generate_hyphenated") == 0 ||
                         strcmp(mfn, "generate_random") == 0 ||
                         strcmp(mfn, "generate_time_ordered") == 0 ||
                         strcmp(mfn, "parse") == 0) {
                    result = type_struct("UUID");
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
                if (strcmp(mfn, "parse") == 0 ||
                    strcmp(mfn, "read_file") == 0) {
                    result = type_array("[string]"); /* [[string]] */
                } else if (strcmp(mfn, "headers") == 0) {
                    result = type_array("string"); /* [string] */
                } else if (strcmp(mfn, "write_file") == 0) {
                    result = &TYPE_BOOL;
                    /* E5026: second arg must be an array, not a string */
                    if (node->data.call.arg_count >= 2) {
                        GrayType *arg2_type = resolve_expr(tc, node->data.call.args[1]);
                        if (arg2_type && arg2_type->kind == TK_STRING) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "csv.%s() expects an array as the second argument, got string",
                                mfn);
                            diag_error_msg(tc->diag, "E5026", msg,
                                NODE_FILE(tc, node), node->data.call.args[1]->token.line,
                                node->data.call.args[1]->token.column, 0);
                        }
                    }
                } else if (strcmp(mfn, "encode") == 0) {
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
                           strcmp(mfn, "pretty_print") == 0) {
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
                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
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
                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type)
                            result = type_from_name(arr_t->element_type);
                        else
                            result = &TYPE_INT;
                    } else {
                        result = &TYPE_INT;
                    }
                } else if (strcmp(mfn, "seed") == 0) result = &TYPE_VOID;
                else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "arrays") == 0) {
                if (strcmp(mfn, "is_empty") == 0 || strcmp(mfn, "contains") == 0 ||
                    strcmp(mfn, "is_equal") == 0 ||
                    strcmp(mfn, "any") == 0 || strcmp(mfn, "all") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index_of") == 0 || strcmp(mfn, "get_sum") == 0 ||
                           strcmp(mfn, "get_min") == 0 || strcmp(mfn, "get_max") == 0 ||
                           strcmp(mfn, "count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "reverse") == 0 || strcmp(mfn, "slice") == 0 ||
                           strcmp(mfn, "concat") == 0 || strcmp(mfn, "deduplicate") == 0 ||
                           strcmp(mfn, "map") == 0 || strcmp(mfn, "filter") == 0) {
                    /* Preserve input array element type */
                    if (node->data.call.arg_count > 0) {
                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arr_t && arr_t->element_type) ? type_array(arr_t->element_type) : type_array("int");
                    } else {
                        result = type_array("int");
                    }
                } else if (strcmp(mfn, "split_every") == 0 || strcmp(mfn, "pair") == 0) {
                    result = type_array("[int]"); /* nested array */
                } else if (strcmp(mfn, "flatten") == 0) {
                    if (node->data.call.arg_count > 0) {
                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        if (arr_t && arr_t->element_type) {
                            /* arr_t is [[T]]; element_type is "[T]". Unwrap one level. */
                            GrayType *inner = type_from_name(arr_t->element_type);
                            if (inner && inner->kind == TK_ARRAY && inner->element_type)
                                result = type_array(inner->element_type);
                            else
                                result = type_array(arr_t->element_type);
                        } else {
                            result = type_array("int");
                        }
                    } else {
                        result = type_array("int");
                    }
                } else if (strcmp(mfn, "get_first") == 0 || strcmp(mfn, "get_last") == 0 ||
                           strcmp(mfn, "remove_last") == 0 || strcmp(mfn, "remove_first") == 0 ||
                           strcmp(mfn, "reduce") == 0) {
                    if (node->data.call.arg_count > 0) {
                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
                        result = (arr_t && arr_t->element_type) ? type_from_name(arr_t->element_type) : &TYPE_INT;
                    } else {
                        result = &TYPE_INT;
                    }
                } else if (strcmp(mfn, "append") == 0 || strcmp(mfn, "prepend") == 0 ||
                           strcmp(mfn, "insert_at") == 0 ||
                           strcmp(mfn, "remove") == 0 || strcmp(mfn, "remove_at") == 0 ||
                           strcmp(mfn, "fill") == 0 ||
                           strcmp(mfn, "sort_asc") == 0 || strcmp(mfn, "sort_desc") == 0 ||
                           strcmp(mfn, "clear") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* E5007: mutating array functions on const array */
                if ((strcmp(mfn, "append") == 0 || strcmp(mfn, "insert_at") == 0 ||
                     strcmp(mfn, "remove") == 0 || strcmp(mfn, "remove_at") == 0 ||
                     strcmp(mfn, "remove_last") == 0 ||
                     strcmp(mfn, "remove_first") == 0 || strcmp(mfn, "prepend") == 0 ||
                     strcmp(mfn, "fill") == 0 ||
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
                        GrayType *arr_t = typetable_get(tc->type_table, arr_arg);
                        if (!arr_t) arr_t = resolve_expr(tc, arr_arg);
                        GrayType *val_t = resolve_expr(tc, val_node);
                        if (arr_t && arr_t->kind != TK_ARRAY && arr_t->kind != TK_UNKNOWN) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "arrays.%s() expects an array as the first argument, got %s",
                                op_name, type_name(arr_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, arr_arg), arr_arg->token.line, arr_arg->token.column, 0);
                        } else if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
                            val_t && val_t->kind != TK_UNKNOWN) {
                            GrayType *elem_t = type_from_name(arr_t->element_type);
                            if (elem_t->kind != TK_UNKNOWN && elem_t->kind != val_t->kind &&
                                !(is_int_kind(elem_t->kind) && is_int_kind(val_t->kind))) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "type mismatch in arrays.%s(); cannot add %s to array of %s",
                                    op_name, type_name(val_t), arr_t->element_type);
                                diag_error_msg(tc->diag, "E3001", msg,
                                    NODE_FILE(tc, val_node), val_node->token.line, val_node->token.column, 0);
                            }
                        }
                    }
                }
                /* E3001: arrays.remove_at/insert_at index must be int */
                if ((strcmp(mfn, "remove_at") == 0 && node->data.call.arg_count >= 2) ||
                    (strcmp(mfn, "insert_at") == 0 && node->data.call.arg_count >= 2)) {
                    AstNode *idx_node = node->data.call.args[1];
                    GrayType *idx_t = resolve_expr(tc, idx_node);
                    if (idx_t && idx_t->kind != TK_UNKNOWN && !is_int_kind(idx_t->kind)) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "arrays.%s() expects an int index, got %s",
                            mfn, type_name(idx_t));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, idx_node), idx_node->token.line, idx_node->token.column, 0);
                    }
                }
                /* E9002: arrays.sum/min/max require numeric array */
                if ((strcmp(mfn, "sum") == 0 || strcmp(mfn, "min") == 0 ||
                     strcmp(mfn, "max") == 0) && node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    GrayType *arr_t = typetable_get(tc->type_table, arg0);
                    if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
                        GrayType *elem_t = type_from_name(arr_t->element_type);
                        if (elem_t->kind == TK_STRING || elem_t->kind == TK_BOOL) {
                            diag_error_codef(tc->diag, "E9002", NODE_FILE(tc, arg0), arg0->token.line, arg0->token.column, 0, mfn, arr_t->element_type);
                        }
                    }
                }
                /* E3001: arrays.concat element type mismatch */
                if (strcmp(mfn, "concat") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *a0 = node->data.call.args[0];
                    AstNode *a1 = node->data.call.args[1];
                    GrayType *t0 = typetable_get(tc->type_table, a0);
                    GrayType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_ARRAY && t1->kind == TK_ARRAY &&
                        t0->element_type && t1->element_type &&
                        strcmp(t0->element_type, t1->element_type) != 0) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "type mismatch: cannot concat array of %s with array of %s",
                            t0->element_type, t1->element_type);
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                    }
                }
                if (strcmp(mfn, "is_equal") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *a0 = node->data.call.args[0];
                    AstNode *a1 = node->data.call.args[1];
                    GrayType *t0 = typetable_get(tc->type_table, a0);
                    GrayType *t1 = typetable_get(tc->type_table, a1);
                    if (t0 && t1 && t0->kind == TK_ARRAY && t1->kind == TK_ARRAY &&
                        t0->element_type && t1->element_type &&
                        strcmp(t0->element_type, t1->element_type) != 0) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "type mismatch: cannot compare array of %s with array of %s",
                            t0->element_type, t1->element_type);
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, a1), a1->token.line, a1->token.column, 0);
                    }
                    /* Composite element types (nested arrays, maps, structs) cannot be
                     * compared with the primitive memcmp path; reject for now. */
                    if (t0 && t0->kind == TK_ARRAY && t0->element_type) {
                        GrayType *et = type_from_name(t0->element_type);
                        if (et->kind == TK_ARRAY || et->kind == TK_MAP || et->kind == TK_STRUCT) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "arrays.is_equal does not support arrays of %s; only primitive and string element types are supported",
                                t0->element_type);
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0);
                        }
                    }
                }
                if (strcmp(mfn, "contains") == 0 && node->data.call.arg_count >= 1) {
                    AstNode *a0 = node->data.call.args[0];
                    GrayType *t0 = typetable_get(tc->type_table, a0);
                    if (t0 && t0->kind == TK_ARRAY && t0->element_type) {
                        GrayType *et = type_from_name(t0->element_type);
                        if (et->kind == TK_ARRAY || et->kind == TK_MAP || et->kind == TK_STRUCT) {
                            diag_error_codef(tc->diag, "E9006",
                                NODE_FILE(tc, a0), a0->token.line, a0->token.column, 0,
                                t0->element_type);
                        }
                    }
                }
                /* E9003/E9004: map/filter/reduce callback validation */
                if ((strcmp(mfn, "map") == 0 || strcmp(mfn, "filter") == 0 ||
                     strcmp(mfn, "reduce") == 0 ||
                     strcmp(mfn, "any") == 0 || strcmp(mfn, "all") == 0) && node->data.call.arg_count >= 2) {
                    /* Determine which arg is the callback */
                    int cb_idx = (strcmp(mfn, "reduce") == 0) ? 2 : 1;
                    if (cb_idx < node->data.call.arg_count) {
                        AstNode *cb_arg = node->data.call.args[cb_idx];
                        if (cb_arg->kind != NODE_FUNC_REF &&
                            !(cb_arg->kind == NODE_CALL_EXPR &&
                              cb_arg->data.call.function->kind == NODE_LABEL &&
                              strcmp(cb_arg->data.call.function->data.label.value, "ref") == 0)) {
                            diag_error_codef(tc->diag, "E9003",
                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                mfn);
                        } else {
                            /* Validate callback signature */
                            const char *ref_name = NULL;
                            if (cb_arg->kind == NODE_FUNC_REF &&
                                cb_arg->data.func_ref.function->kind == NODE_LABEL) {
                                ref_name = cb_arg->data.func_ref.function->data.label.value;
                            }
                            if (ref_name) {
                                FuncSig *cb_fs = find_func(tc, ref_name);
                                if (cb_fs) {
                                    if (strcmp(mfn, "map") == 0) {
                                        if (cb_fs->param_count != 1) {
                                            char *msg = NULL;
                                            msg = tc_fmt(tc,
                                                "map callback must take 1 parameter, got %d",
                                                cb_fs->param_count);
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, msg);
                                        } else if (cb_fs->return_count < 1) {
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, "map callback must return a value");
                                        }
                                    } else if (strcmp(mfn, "filter") == 0 ||
                                               strcmp(mfn, "any") == 0 ||
                                               strcmp(mfn, "all") == 0) {
                                        if (cb_fs->param_count != 1) {
                                            char *msg = NULL;
                                            msg = tc_fmt(tc,
                                                "%s callback must take 1 parameter, got %d",
                                                mfn, cb_fs->param_count);
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, msg);
                                        } else if (cb_fs->return_count < 1 ||
                                                   cb_fs->return_types[0]->kind != TK_BOOL) {
                                            char *msg = NULL;
                                            msg = tc_fmt(tc,
                                                "%s callback must return bool", mfn);
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, msg);
                                        }
                                    } else { /* reduce */
                                        if (cb_fs->param_count != 2) {
                                            char *msg = NULL;
                                            msg = tc_fmt(tc,
                                                "reduce callback must take 2 parameters, got %d",
                                                cb_fs->param_count);
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, msg);
                                        } else if (cb_fs->return_count < 1) {
                                            diag_error_codef(tc->diag, "E9004",
                                                NODE_FILE(tc, cb_arg), cb_arg->token.line, cb_arg->token.column, 0,
                                                mfn, "reduce callback must return a value");
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                /* All arrays.* functions expect an array as the first argument */
                if (node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    GrayType *arg0_t = resolve_expr(tc, arg0);
                    if (arg0_t && arg0_t->kind != TK_ARRAY && arg0_t->kind != TK_UNKNOWN) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "arrays.%s() expects an array as the first argument, got '%s'",
                            mfn, type_name(arg0_t));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, arg0), arg0->token.line, arg0->token.column, 0);
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
                } else if (strcmp(mfn, "set_env") == 0 || strcmp(mfn, "unset_env") == 0) {
                    result = &TYPE_VOID;
                } else if (strcmp(mfn, "exec") == 0) {
                    result = &TYPE_BOOL;
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
                        GrayType *arg_t = resolve_expr(tc, node->data.call.args[0]);
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
                           strcmp(mfn, "lerp") == 0 || strcmp(mfn, "distance") == 0) {
                    result = &TYPE_FLOAT;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "threads") == 0) {
                if (strcmp(mfn, "spawn") == 0 || strcmp(mfn, "spawn_arg") == 0) {
                    /* Validate first arg is a function reference */
                    if (node->data.call.arg_count >= 1) {
                        AstNode *arg0 = node->data.call.args[0];
                        if (arg0->kind != NODE_FUNC_REF &&
                            !(arg0->kind == NODE_CALL_EXPR &&
                              arg0->data.call.function->kind == NODE_LABEL &&
                              strcmp(arg0->data.call.function->data.label.value, "ref") == 0)) {
                            diag_error_msg(tc->diag, "E7006",
                                "threads.spawn() requires a function reference; use ()func_name or ref(func_name)",
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                    result = type_struct("Thread"); /* GrayThread; opaque */
                } else if (strcmp(mfn, "get_id") == 0 ||
                           strcmp(mfn, "current") == 0 ||
                           strcmp(mfn, "thread_count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "is_alive") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "join") == 0 || strcmp(mfn, "detach") == 0 ||
                           strcmp(mfn, "yield") == 0 ||
                           strcmp(mfn, "sleep") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "sync") == 0) {
                if (strcmp(mfn, "mutex") == 0) {
                    result = type_struct("Mutex"); /* GrayMutex; opaque */
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
                    result = type_struct("SpinLock"); /* GraySpinLock; opaque */
                } else if (strcmp(mfn, "store") == 0 || strcmp(mfn, "fence") == 0 ||
                           strcmp(mfn, "spin_lock") == 0 ||
                           strcmp(mfn, "spin_unlock") == 0 ||
                           strcmp(mfn, "spinlock_destroy") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "channels") == 0) {
                if (strcmp(mfn, "open") == 0) {
                    result = type_struct("Channel"); /* GrayChannel; opaque */
                } else if (strcmp(mfn, "receive") == 0 || strcmp(mfn, "try_receive") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "send") == 0 || strcmp(mfn, "close") == 0) {
                    result = &TYPE_VOID;
                } else if (strcmp(mfn, "try_send") == 0) {
                    result = &TYPE_BOOL;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "server") == 0) {
                if (strcmp(mfn, "add_router") == 0) {
                    result = type_struct("Router"); /* GrayRouter; opaque */
                } else if (strcmp(mfn, "text") == 0 || strcmp(mfn, "json") == 0 ||
                           strcmp(mfn, "html") == 0 || strcmp(mfn, "redirect") == 0) {
                    result = type_struct("HttpResponse"); /* GrayResponse */
                } else if (strcmp(mfn, "add_route") == 0 || strcmp(mfn, "listen") == 0 ||
                           strcmp(mfn, "cors") == 0 || strcmp(mfn, "use") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(mod, "http") == 0) {
                /* All http methods return GrayHttpResponse struct
                 * with .status (int), .body (string), .headers (map) */
                result = type_struct("HttpResponse"); /* opaque struct; member access via __auto_type */
            } else if (strcmp(mod, "net") == 0) {
                if (strcmp(mfn, "listen") == 0) {
                    result = type_struct("Listener"); /* GraySocket; opaque */
                } else if (strcmp(mfn, "connect") == 0 || strcmp(mfn, "accept") == 0) {
                    result = type_struct("Socket"); /* GraySocket; opaque */
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
                        GrayType *arg1_type = resolve_expr(tc, node->data.call.args[0]);
                        if (arg1_type && arg1_type->kind != TK_STRUCT) {
                            const char *expected = strcmp(mfn, "accept") == 0
                                ? "Listener" : "Socket";
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "net.%s() expects a %s as the first argument, got %s",
                                mfn, expected,
                                arg1_type->name ? arg1_type->name : "non-struct type");
                            diag_error_msg(tc->diag, "E5026", msg,
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
                    GrayType *val_t = resolve_expr(tc, node->data.call.args[1]);
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
                    strcmp(mfn, "sprintfln") == 0 ||
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
                } else if (strcmp(mfn, "printf") == 0 ||
                           strcmp(mfn, "printfln") == 0 ||
                           strcmp(mfn, "eprintf") == 0 ||
                           strcmp(mfn, "eprintfln") == 0) {
                    result = &TYPE_VOID;
                } else {
                    emit_unknown_stdlib_fn(tc, mod, mfn, node);
                    result = &TYPE_UNKNOWN;
                }
                /* Validate printf/sprintf/format: literal format string + directive types */
                {
                    bool is_fmt_fn = strcmp(mfn, "printf") == 0 ||
                                     strcmp(mfn, "printfln") == 0 ||
                                     strcmp(mfn, "eprintf") == 0 ||
                                     strcmp(mfn, "eprintfln") == 0 ||
                                     strcmp(mfn, "sprintf") == 0 ||
                                     strcmp(mfn, "sprintfln") == 0 ||
                                     strcmp(mfn, "format") == 0;
                    if (is_fmt_fn && node->data.call.arg_count >= 1) {
                        AstNode *fmt_arg = node->data.call.args[0];
                        if (fmt_arg->kind != NODE_STRING_VALUE) {
                            diag_error_codef(tc->diag, "E3086",
                                NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                fmt_arg->token.column, 0, mfn);
                        } else {
                            /* Walk format string, validate each directive against arg type */
                            const char *fstr = fmt_arg->data.string_value.value;
                            const char *p = fstr;
                            int di = 1;
                            int num_directives = 0;
                            while (*p) {
                                if (*p != '%') { p++; continue; }
                                p++;
                                if (!*p) {
                                    /* Dangling % at end of format string */
                                    diag_error_codef(tc->diag, "E3106",
                                        NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                        fmt_arg->token.column, 0, mfn);
                                    break;
                                }
                                if (*p == '%') { p++; continue; }
                                if (*p == 'n') {
                                    diag_error_codef(tc->diag, "E3087",
                                        NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                        fmt_arg->token.column, 0);
                                    break;
                                }
                                /* Skip flags, width, precision, length modifier */
                                while (*p == '-' || *p == '+' || *p == ' ' || *p == '0' || *p == '#') p++;
                                while (*p >= '0' && *p <= '9') p++;
                                if (*p == '.') { p++; while (*p >= '0' && *p <= '9') p++; }
                                if (*p == 'h') { p++; if (*p == 'h') p++; }
                                else if (*p == 'l') { p++; if (*p == 'l') p++; }
                                else if (*p == 'L') p++;
                                char spec = *p ? *p++ : 0;
                                if (!spec) {
                                    diag_error_codef(tc->diag, "E3106",
                                        NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                        fmt_arg->token.column, 0, mfn);
                                    break;
                                }
                                num_directives++;
                                /* Reject unknown format directives */
                                bool known = false;
                                switch (spec) {
                                case 'd': case 'i': case 'u':
                                case 'x': case 'X': case 'o':
                                case 'f': case 'g': case 'e': case 'G': case 'E':
                                case 's': case 'c': case 'b':
                                    known = true;
                                    break;
                                default:
                                    known = false;
                                    break;
                                }
                                if (!known) {
                                    diag_error_codef(tc->diag, "E3105",
                                        NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                        fmt_arg->token.column, 0, mfn, spec);
                                    di++;
                                    continue;
                                }
                                if (di >= node->data.call.arg_count) { di++; continue; }
                                AstNode *darg = node->data.call.args[di];
                                GrayType *dt = resolve_expr(tc, darg);
                                di++;
                                if (!dt) continue;
                                const char *expected = NULL;
                                bool ok = false;
                                switch (spec) {
                                case 'd': case 'i':
                                    expected = "int or char";
                                    ok = dt->kind == TK_INT || dt->kind == TK_CHAR || dt->kind == TK_BYTE;
                                    break;
                                case 'u':
                                    expected = "uint";
                                    ok = dt->kind == TK_UINT || dt->kind == TK_BYTE;
                                    break;
                                case 'x': case 'X': case 'o':
                                    expected = "int or uint";
                                    ok = dt->kind == TK_INT || dt->kind == TK_UINT || dt->kind == TK_BYTE;
                                    break;
                                case 'f': case 'g': case 'e': case 'G': case 'E':
                                    expected = "float";
                                    ok = dt->kind == TK_FLOAT;
                                    break;
                                case 's':
                                    expected = "string";
                                    ok = dt->kind == TK_STRING;
                                    break;
                                case 'c':
                                    expected = "char";
                                    ok = dt->kind == TK_CHAR || dt->kind == TK_INT;
                                    break;
                                default:
                                    ok = true;
                                    break;
                                }
                                if (!ok && expected) {
                                    char spec_str[2] = { spec, '\0' };
                                    diag_error_codef(tc->diag, "E3088",
                                        NODE_FILE(tc, darg), darg->token.line,
                                        darg->token.column, 0,
                                        mfn, spec_str, expected, di - 1,
                                        type_name(dt));
                                }
                            }
                            /* Check argument count vs directive count */
                            int num_args = node->data.call.arg_count - 1;
                            if (num_args < num_directives) {
                                diag_error_codef(tc->diag, "E3107",
                                    NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                    fmt_arg->token.column, 0,
                                    mfn, num_directives, num_args);
                            } else if (num_args > num_directives) {
                                diag_error_codef(tc->diag, "E3108",
                                    NODE_FILE(tc, fmt_arg), fmt_arg->token.line,
                                    fmt_arg->token.column, 0,
                                    mfn, num_directives, num_args);
                            }
                        }
                    }
                }
                /* Validate that non-format args are primitive types */
                for (int ai = 1; ai < node->data.call.arg_count; ai++) {
                    GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                    if (arg_t && (arg_t->kind == TK_STRUCT || arg_t->kind == TK_ARRAY ||
                                  arg_t->kind == TK_MAP || arg_t->kind == TK_POINTER)) {
                        /* Build a readable type name */
                        char tn[TYPE_NAME_MAX];
                        if (arg_t->kind == TK_ARRAY && arg_t->element_type)
                            snprintf(tn, sizeof(tn), "[%s]", arg_t->element_type);
                        else if (arg_t->kind == TK_MAP)
                            snprintf(tn, sizeof(tn), "map[%s:%s]",
                                arg_t->key_type ? arg_t->key_type : "?",
                                arg_t->value_type ? arg_t->value_type : "?");
                        else if (arg_t->kind == TK_POINTER && arg_t->element_type)
                            snprintf(tn, sizeof(tn), "^%s", arg_t->element_type);
                        else {
                            strncpy(tn, type_name(arg_t), sizeof(tn) - 1);
                            tn[sizeof(tn) - 1] = '\0';
                        }
                        diag_error_codef(tc->diag, "E3017", NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                            node->data.call.args[ai]->token.column, 0, mfn, tn);
                    }
                }
            } else
            user_module_dispatch:
            if (is_struct_name(tc, mod)) {
                /* Struct-namespaced function call: Type.func(); look up return type */
                const char *display_mod = struct_display_name(tc, mod);
                char prefixed[MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                FuncSig *sig = find_func(tc, prefixed);
                if (sig) {
                    sig->used = true;
                    /* Resolve named arguments for struct static calls */
                    if (sig->decl) {
                        char display[MSG_BUF_SIZE];
                        snprintf(display, sizeof(display), "%s.%s", display_mod, mfn);
                        tc_resolve_named_args(tc, node, sig->decl, display);
                    }
                    /* E4017: private struct function called from outside the struct */
                    if (sig->is_private &&
                        !(tc->current_struct_name && strcmp(tc->current_struct_name, mod) == 0)) {
                        diag_error_codef(tc->diag, "E4017", NODE_FILE(tc, node),
                            node->token.line, node->token.column, 0, display_mod, mfn);
                    }
                    /* Wildcard (generic) struct function: record instantiation
                     * and substitute the concrete return type. */
                    if (sig->is_generic && sig->decl &&
                        sig->decl->kind == NODE_FUNC_DECL) {
                        char *binding = NULL;
                        int cc = node->data.call.arg_count < sig->decl->data.func_decl.param_count
                            ? node->data.call.arg_count : sig->decl->data.func_decl.param_count;
                        for (int ai = 0; ai < cc && !binding; ai++) {
                            const char *ptn = sig->decl->data.func_decl.params[ai].type_name;
                            GrayType *at = resolve_expr(tc, node->data.call.args[ai]);
                            if (!type_name_has_wildcard(ptn)) continue;
                            binding = bind_wildcard(ptn, at);
                        }
                        if (binding) {
                            record_instantiation(sig, binding, node);
                            if (sig->decl->data.func_decl.return_type_count > 0) {
                                char *rt = substitute_wildcard(
                                    sig->decl->data.func_decl.return_types[0], binding);
                                result = type_from_name(rt);
                            } else {
                                result = &TYPE_VOID;
                            }
                            free(binding);
                        } else {
                            result = sig->return_count > 0 ? sig->return_types[0] : &TYPE_VOID;
                        }
                    } else {
                        result = sig->return_count > 0 ? sig->return_types[0] : &TYPE_VOID;
                    }
                    /* E5008: check argument count, accounting for default params */
                    {
                        int min_params = sig->param_count;
                        if (sig->decl && sig->decl->kind == NODE_FUNC_DECL) {
                            min_params = 0;
                            for (int pi = 0; pi < sig->decl->data.func_decl.param_count; pi++) {
                                if (!sig->decl->data.func_decl.params[pi].default_value)
                                    min_params++;
                            }
                        }
                        if (node->data.call.arg_count < min_params ||
                            node->data.call.arg_count > sig->param_count) {
                            char *msg = NULL;
                            if (min_params == sig->param_count) {
                                msg = tc_fmt(tc,
                                    "function '%s.%s' expects %d argument(s), got %d",
                                    display_mod, mfn, sig->param_count, node->data.call.arg_count);
                            } else {
                                msg = tc_fmt(tc,
                                    "function '%s.%s' expects %d-%d argument(s), got %d",
                                    display_mod, mfn, min_params, sig->param_count, node->data.call.arg_count);
                            }
                            diag_error_msg(tc->diag, "E5008", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                    /* Check argument types */
                    int check_count = node->data.call.arg_count < sig->param_count
                        ? node->data.call.arg_count : sig->param_count;
                    for (int ai = 0; ai < check_count; ai++) {
                        GrayType *param_t = sig->param_types[ai];
                        /* Set expected_type for implicit enum resolution */
                        GrayType *saved_expected_m = tc->expected_type;
                        if (param_t && param_t->kind == TK_ENUM && param_t->name)
                            tc->expected_type = param_t;
                        GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                        tc->expected_type = saved_expected_m;
                        if (arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                            arg_t->kind != param_t->kind &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                            !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                            !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                            !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s.%s': expected %s, got %s",
                                ai + 1, display_mod, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Enum-to-enum: kinds both TK_ENUM but different names */
                        if (arg_t->kind == TK_ENUM && param_t->kind == TK_ENUM &&
                            arg_t->name && param_t->name &&
                            !tc_same_enum_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s.%s': expected enum '%s', got enum '%s'",
                                ai + 1, display_mod, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Struct-to-struct: kinds both TK_STRUCT but different names */
                        if (arg_t->kind == TK_STRUCT && param_t->kind == TK_STRUCT &&
                            arg_t->name && param_t->name &&
                            !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s.%s': expected struct '%s', got struct '%s'",
                                ai + 1, display_mod, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Pointer-to-pointer: pointee types differ */
                        if (arg_t->kind == TK_POINTER && param_t->kind == TK_POINTER &&
                            arg_t->name && param_t->name &&
                            !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s.%s': expected '%s', got '%s'",
                                ai + 1, display_mod, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
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
                                        char emsg[MSG_BUF_SIZE];
                                        snprintf(emsg, sizeof(emsg),
                                            "cannot pass constant '%s' to mutable parameter '%s' of '%s.%s'",
                                            arg_sym->name, found_decl->data.func_decl.params[ai].name, display_mod, mfn);
                                        diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                            NODE_FILE(tc, arg), arg->token.line,
                                            arg->token.column, 0);
                                    }
                                } else if (arg->kind == NODE_MEMBER_EXPR &&
                                           arg->data.member.object->kind == NODE_LABEL &&
                                           is_enum_name(tc, arg->data.member.object->data.label.value)) {
                                    char emsg[MSG_BUF_SIZE];
                                    snprintf(emsg, sizeof(emsg),
                                        "cannot pass enum constant to mutable parameter '%s' of '%s.%s'; expected a mutable variable",
                                        found_decl->data.func_decl.params[ai].name, display_mod, mfn);
                                    diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                        NODE_FILE(tc, arg), arg->token.line,
                                        arg->token.column, 0);
                                } else if (arg->kind != NODE_MEMBER_EXPR &&
                                           arg->kind != NODE_INDEX_EXPR &&
                                           arg->kind != NODE_PREFIX_EXPR) {
                                    char emsg[MSG_BUF_SIZE];
                                    snprintf(emsg, sizeof(emsg),
                                        "cannot pass a literal or expression to mutable parameter '%s' of '%s.%s'; expected a mutable variable",
                                        found_decl->data.func_decl.params[ai].name, display_mod, mfn);
                                    diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                        NODE_FILE(tc, arg), arg->token.line,
                                        arg->token.column, 0);
                                }
                            }
                        }
                    }
                } else {
                    /* E4018: struct has no function with this name */
                    diag_error_codef(tc->diag, "E4018", NODE_FILE(tc, node),
                        node->token.line, node->token.column, 0, display_mod, mfn);
                    result = &TYPE_VOID;
                }
            } else {
                /* Try user-defined module: look up <module>_<func> in function registry */
                char prefixed[MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                FuncSig *sig = find_func(tc, prefixed);
                if (sig && sig->is_private) {
                    diag_error_codef(tc->diag, "E4015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, mfn);
                } else if (sig) {
                    sig->used = true;
                    /* Resolve named arguments for user-module function calls */
                    if (sig->decl) {
                        char display[MSG_BUF_SIZE];
                        snprintf(display, sizeof(display), "%s.%s", mod, mfn);
                        tc_resolve_named_args(tc, node, sig->decl, display);
                    }
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
                        const char *display_sname = struct_display_name(tc, sname);
                        /* : check if `mfn` is a data field of type func
                         * before trying struct-function dispatch. A func-typed
                         * field should be called as a function pointer, not
                         * mistaken for a struct function. The bare "func" type
                         * was deprecated; modern typed function refs are
                         * encoded as "func(...)->R" with kind TK_FUNCTION. */
                        GrayType *field_t = struct_field_type(tc, sname, mfn);
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
                        char sfn[MSG_BUF_SIZE];
                        snprintf(sfn, sizeof(sfn), "%s_%s", sname, mfn);
                        FuncSig *ssig = find_func(tc, sfn);
                        /* Auto-dispatch instance.func() → Type.func(instance)
                         * whenever the struct function takes the struct (or a
                         * pointer to it) as its first parameter. This covers
                         * both `do bar(self Foo)` and `do bar(&self Foo)`, and
                         * lets users call struct functions on instances without
                         * having to write the type name at every call site.
                         * Factory-style functions (e.g. `do make(x int) -> Foo`)
                         * whose first param isn't a Foo continue to require
                         * explicit `Foo.make(...)` since there's no instance
                         * to bind. */
                        bool is_self_func = false;
                        if (ssig && ssig->decl && ssig->decl->kind == NODE_FUNC_DECL &&
                            ssig->decl->data.func_decl.param_count > 0) {
                            const char *p0_tn = ssig->decl->data.func_decl.params[0].type_name;
                            if (p0_tn) {
                                if (strcmp(p0_tn, sname) == 0) {
                                    is_self_func = true;
                                } else if (p0_tn[0] == '^' && strcmp(p0_tn + 1, sname) == 0) {
                                    is_self_func = true;
                                }
                            }
                        }
                        if (is_self_func) {
                            /* E4017: private struct function via instance dispatch */
                            if (ssig->is_private &&
                                !(tc->current_struct_name && strcmp(tc->current_struct_name, sname) == 0)) {
                                diag_error_codef(tc->diag, "E4017", NODE_FILE(tc, node),
                                    node->token.line, node->token.column, 0, display_sname, mfn);
                            }
                            /* Resolve named arguments before AST rewrite.
                             * User-visible params start at index 1 (after self),
                             * so build a temporary shifted decl view. Named
                             * args target param names excluding the self param. */
                            if (ssig->decl && ssig->decl->kind == NODE_FUNC_DECL &&
                                tc_has_named_args(node)) {
                                AstNode *sdecl = ssig->decl;
                                /* Create a temporary fake decl with params shifted
                                 * to skip the self parameter (param[0]). */
                                AstNode tmp_decl = *sdecl;
                                int orig_pc = sdecl->data.func_decl.param_count;
                                if (orig_pc > 1) {
                                    tmp_decl.data.func_decl.params = &sdecl->data.func_decl.params[1];
                                    tmp_decl.data.func_decl.param_count = orig_pc - 1;
                                } else {
                                    tmp_decl.data.func_decl.param_count = 0;
                                }
                                char display[MSG_BUF_SIZE];
                                snprintf(display, sizeof(display), "%s.%s", display_sname, mfn);
                                tc_resolve_named_args(tc, node, &tmp_decl, display);
                            }
                            /* Rewrite the call AST: change the member-expr
                             * object from the instance label to the type
                             * name, and prepend the instance as arg[0]. */
                            fn->data.member.object->data.label.value = strdup(sname);
                            int orig_count = node->data.call.arg_count;
                            AstNode **new_args = xmalloc(sizeof(AstNode *) * (orig_count + 1));
                            AstNode *self_arg = xcalloc(1, sizeof(AstNode));
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
                            if (ssig->is_generic && ssig->decl &&
                                ssig->decl->kind == NODE_FUNC_DECL) {
                                char *binding = NULL;
                                int cc = node->data.call.arg_count < ssig->decl->data.func_decl.param_count
                                    ? node->data.call.arg_count : ssig->decl->data.func_decl.param_count;
                                for (int ai = 0; ai < cc && !binding; ai++) {
                                    const char *ptn = ssig->decl->data.func_decl.params[ai].type_name;
                                    GrayType *at = resolve_expr(tc, node->data.call.args[ai]);
                                    if (!type_name_has_wildcard(ptn)) continue;
                                    binding = bind_wildcard(ptn, at);
                                }
                                if (binding) {
                                    record_instantiation(ssig, binding, node);
                                    if (ssig->decl->data.func_decl.return_type_count > 0) {
                                        char *rt = substitute_wildcard(
                                            ssig->decl->data.func_decl.return_types[0], binding);
                                        result = type_from_name(rt);
                                    } else {
                                        result = &TYPE_VOID;
                                    }
                                    free(binding);
                                } else {
                                    result = ssig->return_count > 0 ? ssig->return_types[0] : &TYPE_VOID;
                                }
                            } else {
                                result = ssig->return_count > 0 ? ssig->return_types[0] : &TYPE_VOID;
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
                                    char emsg[MSG_BUF_SIZE];
                                    if (display_min == display_max) {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d argument(s), got %d",
                                            display_sname, mfn, display_max, display_got);
                                    } else {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d-%d argument(s), got %d",
                                            display_sname, mfn, display_min, display_max, display_got);
                                    }
                                    diag_error_msg(tc->diag, "E5008", arena_strdup(tc->arena, emsg),
                                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                                }
                            }
                            /* Validate argument types (after AST rewrite) */
                            {
                                int check_count = node->data.call.arg_count < ssig->param_count
                                    ? node->data.call.arg_count : ssig->param_count;
                                for (int ai = 0; ai < check_count; ai++) {
                                    GrayType *param_t = ssig->param_types[ai];
                                    /* Set expected_type for implicit enum resolution */
                                    GrayType *saved_expected_s = tc->expected_type;
                                    if (param_t && param_t->kind == TK_ENUM && param_t->name)
                                        tc->expected_type = param_t;
                                    GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                                    tc->expected_type = saved_expected_s;
                                    if (arg_t && param_t &&
                                        arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                                        arg_t->kind != param_t->kind &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                                        !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                                        !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                                        !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                                        char amsg[MSG_BUF_SIZE];
                                        snprintf(amsg, sizeof(amsg),
                                            "argument %d of '%s.%s': expected %s, got %s",
                                            ai + 1, display_sname, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                        diag_error_msg(tc->diag, "E3001", arena_strdup(tc->arena, amsg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                    /* Struct-to-struct: kinds both TK_STRUCT but different names */
                                    if (arg_t && param_t &&
                                        arg_t->kind == TK_STRUCT && param_t->kind == TK_STRUCT &&
                                        arg_t->name && param_t->name &&
                                        !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                                        char smsg[MSG_BUF_SIZE];
                                        snprintf(smsg, sizeof(smsg),
                                            "argument %d of '%s.%s': expected struct '%s', got struct '%s'",
                                            ai + 1, display_sname, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                        diag_error_msg(tc->diag, "E3001", arena_strdup(tc->arena, smsg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                    /* Pointer-to-pointer: pointee types differ */
                                    if (arg_t && param_t &&
                                        arg_t->kind == TK_POINTER && param_t->kind == TK_POINTER &&
                                        arg_t->name && param_t->name &&
                                        !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                                        char pmsg[MSG_BUF_SIZE];
                                        snprintf(pmsg, sizeof(pmsg),
                                            "argument %d of '%s.%s': expected '%s', got '%s'",
                                            ai + 1, display_sname, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                        diag_error_msg(tc->diag, "E3001", arena_strdup(tc->arena, pmsg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                }
                            }
                            /* E3027: non-lvalue or const passed to mutable (&) param
                             * in instance dispatch. After the AST rewrite, arg[0]
                             * is self; user-visible args start at index 1. */
                            if (ssig->decl && ssig->decl->kind == NODE_FUNC_DECL) {
                                int pc = ssig->decl->data.func_decl.param_count;
                                int cc = node->data.call.arg_count < pc ? node->data.call.arg_count : pc;
                                for (int ai = 0; ai < cc; ai++) {
                                    if (!ssig->decl->data.func_decl.params[ai].mutable)
                                        continue;
                                    AstNode *arg = node->data.call.args[ai];
                                    if (ai == 0) {
                                        /* Self arg: synthetic NODE_LABEL with value mod_raw */
                                        Symbol *self_sym = scope_lookup(tc->current_scope, mod_raw);
                                        if (self_sym && !self_sym->mutable) {
                                            char emsg[MSG_BUF_SIZE];
                                            snprintf(emsg, sizeof(emsg),
                                                "cannot call '%s.%s' on constant '%s'; function requires a mutable ('&') self parameter",
                                                display_sname, mfn, self_sym->name);
                                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                NODE_FILE(tc, node), node->token.line,
                                                node->token.column, 0);
                                        }
                                    } else {
                                        /* User-visible args */
                                        if (arg->kind == NODE_LABEL) {
                                            Symbol *arg_sym = scope_lookup(tc->current_scope,
                                                arg->data.label.value);
                                            if (arg_sym && !arg_sym->mutable) {
                                                char emsg[MSG_BUF_SIZE];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass constant '%s' to mutable parameter '%s' of '%s.%s'",
                                                    arg_sym->name, ssig->decl->data.func_decl.params[ai].name,
                                                    display_sname, mfn);
                                                diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                    NODE_FILE(tc, arg), arg->token.line,
                                                    arg->token.column, 0);
                                            }
                                        } else if (arg->kind == NODE_MEMBER_EXPR &&
                                                   arg->data.member.object->kind == NODE_LABEL &&
                                                   is_enum_name(tc, arg->data.member.object->data.label.value)) {
                                            char emsg[MSG_BUF_SIZE];
                                            snprintf(emsg, sizeof(emsg),
                                                "cannot pass enum constant to mutable parameter '%s' of '%s.%s'; expected a mutable variable",
                                                ssig->decl->data.func_decl.params[ai].name, display_sname, mfn);
                                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                NODE_FILE(tc, arg), arg->token.line,
                                                arg->token.column, 0);
                                        } else if (arg->kind != NODE_MEMBER_EXPR &&
                                                   arg->kind != NODE_INDEX_EXPR &&
                                                   arg->kind != NODE_PREFIX_EXPR) {
                                            char emsg[MSG_BUF_SIZE];
                                            snprintf(emsg, sizeof(emsg),
                                                "cannot pass a literal or expression to mutable parameter '%s' of '%s.%s'; expected a mutable variable",
                                                ssig->decl->data.func_decl.params[ai].name, display_sname, mfn);
                                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                NODE_FILE(tc, arg), arg->token.line,
                                                arg->token.column, 0);
                                        }
                                    }
                                }
                            }
                        } else if (ssig) {
                            /* Non-self struct function called on an instance.
                             * Rewrite the AST so the member-expr object uses
                             * the struct type name instead of the instance
                             * name.  Do NOT prepend self as arg[0] — the
                             * function doesn't take a self parameter. This
                             * makes the call identical to Type.func() by the
                             * time codegen sees it. */
                            /* E4017: private struct function via instance dispatch */
                            if (ssig->is_private &&
                                !(tc->current_struct_name && strcmp(tc->current_struct_name, sname) == 0)) {
                                diag_error_codef(tc->diag, "E4017", NODE_FILE(tc, node),
                                    node->token.line, node->token.column, 0, display_sname, mfn);
                            }
                            /* Resolve named arguments if present */
                            if (ssig->decl && ssig->decl->kind == NODE_FUNC_DECL &&
                                tc_has_named_args(node)) {
                                char display[MSG_BUF_SIZE];
                                snprintf(display, sizeof(display), "%s.%s", display_sname, mfn);
                                tc_resolve_named_args(tc, node, ssig->decl, display);
                            }
                            fn->data.member.object->data.label.value = strdup(sname);
                            ssig->used = true;
                            sym->used = true;
                            result = ssig->return_count > 0 ? ssig->return_types[0] : &TYPE_VOID;
                            /* Validate argument count */
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
                                    char emsg[MSG_BUF_SIZE];
                                    if (min_params == ssig->param_count) {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d argument(s), got %d",
                                            display_sname, mfn, ssig->param_count, node->data.call.arg_count);
                                    } else {
                                        snprintf(emsg, sizeof(emsg),
                                            "function '%s.%s' expects %d-%d argument(s), got %d",
                                            display_sname, mfn, min_params, ssig->param_count, node->data.call.arg_count);
                                    }
                                    diag_error_msg(tc->diag, "E5008", arena_strdup(tc->arena, emsg),
                                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                                }
                            }
                            /* Validate argument types */
                            {
                                int check_count = node->data.call.arg_count < ssig->param_count
                                    ? node->data.call.arg_count : ssig->param_count;
                                for (int ai = 0; ai < check_count; ai++) {
                                    GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                                    GrayType *param_t = ssig->param_types[ai];
                                    if (arg_t && param_t &&
                                        arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                                        arg_t->kind != param_t->kind &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                                        !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                                        !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                                        !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                                        !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                                        char amsg[MSG_BUF_SIZE];
                                        snprintf(amsg, sizeof(amsg),
                                            "argument %d of '%s.%s': expected %s, got %s",
                                            ai + 1, display_sname, mfn, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                        diag_error_msg(tc->diag, "E3001", arena_strdup(tc->arena, amsg),
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                }
                            }
                        } else {
                            result = &TYPE_VOID;
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
            GrayType *obj_t = resolve_expr(tc, fn->data.member.object);
            if (obj_t && obj_t->kind != TK_STRUCT && obj_t->kind != TK_POINTER &&
                obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_VOID) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type '%s' does not support function calls via dot notation",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", msg,
                    NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0);
            } else if (obj_t && (obj_t->kind == TK_STRUCT || obj_t->kind == TK_POINTER)) {
                /* E3075: chaining struct function calls (calling one struct
                 * function on the result of another) isn't supported.
                 * Assigning the intermediate result to a variable keeps
                 * each call site readable and avoids the AST-rewrite
                 * gymnastics that fluent-interface chaining would require. */
                diag_error_code_help(tc->diag, "E3075",
                    NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0,
                    "assign the intermediate result to a variable, then call the next struct function on it");
            }
            result = &TYPE_UNKNOWN;
            break;
        }

        if (fn_name) {
            if (tc_is_builtin(fn_name)) {
                /* E5034: named arguments are not supported for builtins */
                if (tc_has_named_args(node)) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "named arguments are not supported for builtin function '%s'",
                        fn_name);
                    diag_error_msg(tc->diag, "E5034", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* Check built-in functions first */
            if (strcmp(fn_name, "addr") == 0 && node->data.call.arg_count == 1) {
                AstNode *arg = node->data.call.args[0];
                /* addr() requires an lvalue. is_lvalue_expr recurses into
                 * member/index chains so 'addr(some_call().field)' is
                 * rejected at typecheck instead of leaking an
                 * '&(rvalue)' to clang. */
                if (path_contains_map_index(tc, arg)) {
                    diag_error_msg(tc->diag, "E3013",
                        "addr() cannot take the address of a map index expression; map values may relocate on rehash",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (!is_lvalue_expr(arg)) {
                    diag_error_msg(tc->diag, "E3012",
                        "addr() requires a variable, field, or index expression; cannot take address of a literal or expression",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                /* E3122: addr() of a const variable would allow mutation
                 * through the resulting pointer, bypassing immutability. */
                const char *root = lvalue_root_name(arg);
                if (root) {
                    Symbol *sym = scope_lookup(tc->current_scope, root);
                    if (sym && !sym->mutable) {
                        diag_error_codef(tc->diag, "E3122", NODE_FILE(tc, node),
                            node->token.line, node->token.column, 0, root);
                    }
                }
                GrayType *arg_t = resolve_expr(tc, arg);
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
                    if (path_contains_map_index(tc, arg)) {
                        diag_error_msg(tc->diag, "E3013",
                            "ref() cannot take a reference to a map index expression; map values may relocate on rehash",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (!is_lvalue_expr(arg)) {
                        diag_error_msg(tc->diag, "E3012",
                            "ref() requires a variable, field, or index expression; cannot take a reference to a literal, call result, or expression",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* Build a pointer type that preserves the full source type.
                     * For arrays, type_name returns the element type ("int"),
                     * so reconstruct the full name ("[int]"). */
                    GrayType *arg_t = resolve_expr(tc, arg);
                    const char *pointee_name = type_name(arg_t);
                    if (arg_t->kind == TK_ARRAY) {
                        char buf[MSG_BUF_SIZE];
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
                 * works for the runtime's GrayArray / GrayMap / GrayString
                 * structs but bombs with a raw clang "no member 'len'"
                 * on anything else; structs, enums, primitives,
                 * pointers, func refs, errors. Validate the argument
                 * type here instead of letting it leak. */
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "len() expects 1 argument, got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
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
                /* E5008: type_of() requires exactly 1 argument */
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "type_of() expects 1 argument, got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_STRING;
                    break;
                }
                /* E3084: type_of() with a type name instead of a value */
                if (node->data.call.arg_count > 0) {
                    AstNode *arg = node->data.call.args[0];
                    if (arg->kind == NODE_LABEL) {
                        const char *aname = arg->data.label.value;
                        Symbol *sym = scope_lookup(tc->current_scope, aname);
                        if (!sym && (is_struct_name(tc, aname) || is_enum_name(tc, aname))) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "type_of() expects a value, not a type name '%s'; use type_of(instance) instead",
                                aname);
                            diag_error_msg(tc->diag, "E3084", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                }
                /* E3038: type_of() on void function result */
                if (node->data.call.arg_count > 0) {
                    GrayType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                    if (arg_t->kind == TK_VOID) {
                        diag_error_msg(tc->diag, "E3038",
                            "cannot use type_of() on a void function; the function does not return a value",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "size_of") == 0) {
                /* Rewrite size_of(T) → size_of(?) when T is a type param */
                if (node->data.call.arg_count == 1 &&
                    node->data.call.args[0]->kind == NODE_LABEL &&
                    tc->type_param_name &&
                    strcmp(node->data.call.args[0]->data.label.value,
                           tc->type_param_name) == 0) {
                    node->data.call.args[0]->data.label.value = "?";
                }
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "to_char") == 0) {
                if (node->data.call.arg_count != 2) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "to_char() expects 2 arguments (string, index), got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    GrayType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    GrayType *arg1 = resolve_expr(tc, node->data.call.args[1]);
                    if (arg0->kind != TK_STRING) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "to_char() first argument must be a string, got %s",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3043", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (arg1->kind != TK_INT && arg1->kind != TK_UINT) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "to_char() second argument must be int or uint, got %s",
                            type_name(arg1));
                        diag_error_msg(tc->diag, "E3043", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_CHAR;
            } else if (strcmp(fn_name, "char_count") == 0) {
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "char_count() expects 1 argument (string), got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    GrayType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    if (arg0->kind != TK_STRING) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "char_count() argument must be a string, got %s",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3043", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "c_string") == 0) {
                if (node->data.call.arg_count >= 1) {
                    GrayType *arg0 = resolve_expr(tc, node->data.call.args[0]);
                    if (arg0 && arg0->kind != TK_POINTER && arg0->kind != TK_UNKNOWN) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "c_string() requires a raw C pointer; '%s' is not a pointer type. "
                            "c_string() is only valid with values from C interop (import c\"header.h\")",
                            type_name(arg0));
                        diag_error_msg(tc->diag, "E3083", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "input") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "here") == 0) {
                if (node->data.call.arg_count != 0) {
                    diag_error_code(tc->diag, "E5014",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                result = type_struct("SourceLocation");
            } else if (strcmp(fn_name, "embed") == 0) {
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "embed() takes exactly 1 argument, got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    AstNode *arg = node->data.call.args[0];
                    if (arg->kind != NODE_STRING_VALUE) {
                        diag_error_code(tc->diag, "E5017",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    } else {
                        /* Resolve file path relative to source file directory */
                        const char *embed_path = arg->data.string_value.value;
                        char resolved[4096];
                        const char *src = NODE_FILE(tc, node);
                        if (embed_path[0] != '/' && src) {
                            const char *last_slash = strrchr(src, '/');
                            if (last_slash) {
                                snprintf(resolved, sizeof(resolved), "%.*s%s",
                                    (int)(last_slash - src + 1), src, embed_path);
                            } else {
                                snprintf(resolved, sizeof(resolved), "%s", embed_path);
                            }
                        } else {
                            snprintf(resolved, sizeof(resolved), "%s", embed_path);
                        }
                        /* Reject path traversal outside the source directory */
                        char real_embed[4096];
                        char real_src_dir[4096];
                        if (realpath(resolved, real_embed)) {
                            bool escaped = true;
                            if (src) {
                                const char *last_slash2 = strrchr(src, '/');
                                if (last_slash2) {
                                    char src_dir[4096];
                                    snprintf(src_dir, sizeof(src_dir), "%.*s",
                                        (int)(last_slash2 - src), src);
                                    if (realpath(src_dir, real_src_dir)) {
                                        size_t dir_len = strlen(real_src_dir);
                                        if (strncmp(real_embed, real_src_dir, dir_len) == 0 &&
                                            (real_embed[dir_len] == '/' || real_embed[dir_len] == '\0')) {
                                            escaped = false;
                                        }
                                    }
                                } else {
                                    /* source file has no directory component — cwd is the root */
                                    if (realpath(".", real_src_dir)) {
                                        size_t dir_len = strlen(real_src_dir);
                                        if (strncmp(real_embed, real_src_dir, dir_len) == 0 &&
                                            (real_embed[dir_len] == '/' || real_embed[dir_len] == '\0')) {
                                            escaped = false;
                                        }
                                    }
                                }
                            }
                            if (escaped) {
                                diag_error_code(tc->diag, "E5027",
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                                goto embed_done;
                            }
                        }
                        FILE *ef = fopen(resolved, "r");
                        if (!ef) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "embed() cannot open '%s': file not found or unreadable",
                                embed_path);
                            diag_error_msg(tc->diag, "E5018", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        } else {
                            fclose(ef);
                        }
                        embed_done:;
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "error") == 0) {
                result = type_from_name("Error");
            } else if (strcmp(fn_name, "println") == 0 || strcmp(fn_name, "eprintln") == 0) {
                /* println/eprintln accept 0 or 1 arguments */
                if (node->data.call.arg_count > 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "%s() expects 0 or 1 argument(s), got %d",
                        fn_name, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (node->data.call.arg_count >= 1) {
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind == TK_FUNCTION) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "cannot pass a func reference to %s(); func references are not printable values",
                            fn_name);
                        diag_error_msg(tc->diag, "E5028", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (at->kind == TK_ENUM && at->name && tc_enum_is_tagged(tc, at->name)) {
                        diag_error_codef(tc->diag, "E5038",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                            enum_display_name(tc, at->name), fn_name);
                    }
                    char ctx[TYPE_NAME_MAX];
                    snprintf(ctx, sizeof(ctx), "%s() argument", fn_name);
                    reject_void_in_context(tc, node->data.call.args[0], at, ctx);
                }
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "print") == 0 || strcmp(fn_name, "eprint") == 0) {
                /* print/eprint accept exactly 1 argument */
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "%s() expects 1 argument, got %d",
                        fn_name, node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (node->data.call.arg_count >= 1) {
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind == TK_FUNCTION) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "cannot pass a func reference to %s(); func references are not printable values",
                            fn_name);
                        diag_error_msg(tc->diag, "E5028", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (at->kind == TK_ENUM && at->name && tc_enum_is_tagged(tc, at->name)) {
                        diag_error_codef(tc->diag, "E5038",
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                            enum_display_name(tc, at->name), fn_name);
                    }
                    char ctx[TYPE_NAME_MAX];
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
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && !is_int_kind(at->kind)) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "exit() expects an integer argument, got '%s'", type_name(at));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                } else if (strcmp(fn_name, "panic") == 0 && node->data.call.arg_count >= 1) {
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && at->kind != TK_STRING) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "panic() expects a string argument, got '%s'", type_name(at));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                } else if (strcmp(fn_name, "assert") == 0 && node->data.call.arg_count >= 1) {
                    GrayType *cond_t = resolve_expr(tc, node->data.call.args[0]);
                    if (cond_t->kind != TK_UNKNOWN && cond_t->kind != TK_BOOL) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "assert() condition must be a bool, got '%s'", type_name(cond_t));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    if (node->data.call.arg_count >= 2) {
                        GrayType *msg_t = resolve_expr(tc, node->data.call.args[1]);
                        if (msg_t->kind != TK_UNKNOWN && msg_t->kind != TK_STRING) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "assert() message must be a string, got '%s'", type_display_name(tc, msg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
                } else if ((strcmp(fn_name, "sleep_s") == 0 || strcmp(fn_name, "sleep_ms") == 0 ||
                            strcmp(fn_name, "sleep_ns") == 0) && node->data.call.arg_count >= 1) {
                    GrayType *at = resolve_expr(tc, node->data.call.args[0]);
                    if (at->kind != TK_UNKNOWN && !is_int_kind(at->kind)) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "%s() expects an integer argument, got '%s'", fn_name, type_name(at));
                        diag_error_msg(tc->diag, "E3001", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "copy") == 0 && node->data.call.arg_count == 1) {
                result = resolve_expr(tc, node->data.call.args[0]);
                if (result->kind == TK_FUNCTION) {
                    diag_error_msg(tc->diag, "E5029",
                        arena_strdup(tc->arena, "copy() cannot be used on a func reference; func references are compile-time aliases, not copyable values"),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_UNKNOWN;
                } else if (result->kind == TK_POINTER) {
                    diag_error_code(tc->diag, "E5037",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_UNKNOWN;
                }
            } else if (strcmp(fn_name, "char") == 0) {
                /* E5008: char() requires exactly 1 argument */
                if (node->data.call.arg_count != 1) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "char() expects 1 argument, got %d",
                        node->data.call.arg_count);
                    diag_error_msg(tc->diag, "E5008", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_CHAR;
                    break;
                }
                /* E7014: char() with negative integer */
                int64_t lit_val;
                if (try_get_literal_int(node->data.call.args[0], &lit_val) && lit_val < 0) {
                    diag_error_codef(tc->diag, "E7014", NODE_FILE(tc, node), node->token.line, node->token.column, 0, (long long)lit_val);
                }
                /* E3001: char() with a string literal that is not exactly one character */
                GrayType *char_arg_t = resolve_expr(tc, node->data.call.args[0]);
                if (char_arg_t->kind == TK_STRING) {
                    AstNode *arg = node->data.call.args[0];
                    if (arg->kind == NODE_STRING_VALUE) {
                        int slen = (int)strlen(arg->data.string_value.value);
                        if (slen != 1) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "char() requires a single-character string; got a string of length %d",
                                slen);
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        }
                    }
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
                GrayType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                    src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot convert %s to %s; only numeric types, strings, and bools can be converted",
                        type_name(src_t), fn_name);
                    diag_error_help(tc->diag, "E3043", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        "only numeric, enum, and string conversions are supported");
                }
                if (strcmp(fn_name, "byte") == 0)
                    result = &TYPE_BYTE;
                else if (is_unsigned_type(fn_name))
                    result = &TYPE_UINT;
                else
                    result = &TYPE_INT;
            } else if (strcmp(fn_name, "string") == 0 && node->data.call.arg_count == 1) {
                /* E3043: validate source type is convertible to string */
                GrayType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_STRING) {
                    diag_error_msg(tc->diag, "E3043",
                        arena_strdup(tc->arena, "cannot convert string to string; value is already a string"),
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                           src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot convert %s to string; use string interpolation or access individual elements",
                        type_name(src_t));
                    diag_error_msg(tc->diag, "E3043", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "float") == 0 && node->data.call.arg_count == 1) {
                /* E3043: validate source type is convertible to float */
                GrayType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                    src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER ||
                    src_t->kind == TK_BOOL) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot convert %s to float; only numeric types and strings can be converted",
                        type_name(src_t));
                    diag_error_msg(tc->diag, "E3043", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                result = &TYPE_FLOAT;
            } else if (strcmp(fn_name, "bool") == 0 && node->data.call.arg_count == 1) {
                GrayType *src_t = resolve_expr(tc, node->data.call.args[0]);
                if (src_t->kind == TK_ARRAY || src_t->kind == TK_MAP ||
                    src_t->kind == TK_STRUCT || src_t->kind == TK_POINTER ||
                    src_t->kind == TK_STRING) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot convert %s to bool; only numeric types and bools can be converted",
                        type_name(src_t));
                    diag_error_msg(tc->diag, "E3043", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                result = &TYPE_BOOL;
            } else if ((strcmp(fn_name, "i8") == 0 || strcmp(fn_name, "i16") == 0 ||
                        strcmp(fn_name, "i32") == 0 || strcmp(fn_name, "i64") == 0 ||
                        strcmp(fn_name, "u8") == 0 || strcmp(fn_name, "u16") == 0 ||
                        strcmp(fn_name, "u32") == 0 || strcmp(fn_name, "u64") == 0 ||
                        strcmp(fn_name, "f32") == 0 || strcmp(fn_name, "f64") == 0)) {
                diag_error_codef(tc->diag, "E5036", NODE_FILE(tc, node),
                    node->token.line, node->token.column, 0, fn_name, fn_name);
                result = type_from_name(fn_name);
            } else {
                FuncSig *sig = find_func(tc, fn_name);
                if (sig) {
                    sig->used = true;
                    /* Use the user-facing name in error messages, never
                     * the module-prefixed internal key. */
                    fn_name = func_display_name(sig);
                    /* : if this bare name is a using-module alias,
                     * also mark the prefixed sig + import as used so
                     * W1002/W1003 don't fire on the source. */
                    for (int ui = 0; ui < tc->using_module_count; ui++) {
                        if (!using_module_accessible(tc, ui)) continue;
                        const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                        char pfx[MSG_BUF_SIZE];
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
                    /* Resolve named arguments before checking counts/types */
                    if (sig->decl) {
                        tc_resolve_named_args(tc, node, sig->decl, fn_name);
                    }
                    /* Check argument count; account for default parameters */
                    int min_args = sig->param_count;
                    if (sig->decl && sig->decl->kind == NODE_FUNC_DECL) {
                        min_args = 0;
                        for (int pi = 0; pi < sig->decl->data.func_decl.param_count; pi++) {
                            if (!sig->decl->data.func_decl.params[pi].default_value)
                                min_args++;
                        }
                    }
                    if (node->data.call.arg_count < min_args ||
                        node->data.call.arg_count > sig->param_count) {
                        char *msg = NULL;
                        if (min_args == sig->param_count) {
                            msg = tc_fmt(tc,
                                "function '%s' expects %d argument(s), got %d",
                                fn_name, sig->param_count, node->data.call.arg_count);
                        } else {
                            msg = tc_fmt(tc,
                                "function '%s' expects %d-%d argument(s), got %d",
                                fn_name, min_args, sig->param_count, node->data.call.arg_count);
                        }
                        diag_error_msg(tc->diag, "E5008", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* Generic (wildcard) dispatch: unify each '?' parameter
                     * against the corresponding argument to derive a single
                     * concrete binding T, record the instantiation, and
                     * substitute T into the return type. Skip the normal
                     * per-arg check below since '?' would otherwise collapse
                     * to TK_UNKNOWN and produce no useful errors. */
                    char *generic_binding = NULL;
                    GrayType *generic_return_t = NULL;
                    bool is_generic_call = sig->is_generic && sig->decl &&
                        sig->decl->kind == NODE_FUNC_DECL;
                    if (is_generic_call) {
                        int cc = node->data.call.arg_count < sig->decl->data.func_decl.param_count
                            ? node->data.call.arg_count : sig->decl->data.func_decl.param_count;
                        for (int ai = 0; ai < cc; ai++) {
                            const char *ptn = sig->decl->data.func_decl.params[ai].type_name;
                            /* Type parameter (<?>) — binding comes from the label
                             * (a struct name), not from resolve_expr. */
                            if (sig->decl->data.func_decl.params[ai].is_type_param) {
                                if (node->data.call.args[ai]->kind != NODE_LABEL) {
                                    diag_error_code(tc->diag, "E3128",
                                        NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0);
                                    continue;
                                }
                                const char *arg_label = node->data.call.args[ai]->data.label.value;
                                /* Must not be a variable in scope */
                                if (scope_lookup(tc->current_scope, arg_label)) {
                                    diag_error_code(tc->diag, "E3128",
                                        NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0);
                                    continue;
                                }
                                /* Must be a struct name */
                                if (!is_struct_name(tc, arg_label)) {
                                    diag_error_codef(tc->diag, "E3127",
                                        NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0,
                                        arg_label);
                                    continue;
                                }
                                if (!generic_binding) {
                                    generic_binding = (char *)arg_label;
                                }
                                continue;
                            }
                            GrayType *at = resolve_expr(tc, node->data.call.args[ai]);
                            if (!type_name_has_wildcard(ptn)) continue;
                            /* E3100: struct/enum type name passed as a value argument */
                            if (at->kind == TK_UNKNOWN &&
                                node->data.call.args[ai]->kind == NODE_LABEL) {
                                const char *arg_label = node->data.call.args[ai]->data.label.value;
                                if (!scope_lookup(tc->current_scope, arg_label)) {
                                    if (is_struct_name(tc, arg_label)) {
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "type name '%s' cannot be used as a value; to create an instance use 'new(%s)' or '%s{...}'",
                                            arg_label, arg_label, arg_label);
                                        diag_error_msg(tc->diag, "E3100", msg,
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                        continue;
                                    } else if (is_enum_name(tc, arg_label)) {
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "type name '%s' cannot be used as a value; use '%s.VARIANT' to access an enum value",
                                            arg_label, arg_label);
                                        diag_error_msg(tc->diag, "E3100", msg,
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                        continue;
                                    }
                                }
                            }
                            char *bound = bind_wildcard(ptn, at);
                            if (!bound) {
                                /* arg is TK_UNKNOWN: we are inside a generic
                                 * function body during the main pass and the
                                 * outer param hasn't been bound yet. The
                                 * re-check pass will validate with concrete
                                 * types — skip the false-positive here. */
                                if (at->kind == TK_UNKNOWN) continue;
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "cannot infer wildcard type '%s' from argument %d of '%s' (got %s)",
                                    ptn, ai + 1, fn_name, type_name(at));
                                diag_error_msg(tc->diag, "E3001", msg,
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                                continue;
                            }
                            if (!generic_binding) {
                                generic_binding = bound;
                            } else if (strcmp(generic_binding, bound) != 0) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "wildcard type conflict in '%s': '?' was bound to %s, but argument %d is %s",
                                    fn_name, generic_binding, ai + 1, bound);
                                diag_error_msg(tc->diag, "E3001", msg,
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
                        /* Skip type parameters — no value to type-check */
                        if (sig->decl && sig->decl->kind == NODE_FUNC_DECL &&
                            ai < sig->decl->data.func_decl.param_count &&
                            sig->decl->data.func_decl.params[ai].is_type_param)
                            continue;
                        GrayType *param_t = sig->param_types[ai];
                        /* Set expected_type for implicit enum resolution */
                        GrayType *saved_expected = tc->expected_type;
                        if (param_t && param_t->kind == TK_ENUM && param_t->name)
                            tc->expected_type = param_t;
                        GrayType *arg_t = resolve_expr(tc, node->data.call.args[ai]);
                        tc->expected_type = saved_expected;
                        if (is_generic_call) {
                            /* Generic branch already handled unification;
                             * suppress the scalar param/arg comparison which
                             * would compare against TK_UNKNOWN. */
                            continue;
                        }
                        /* E3100: struct/enum type name passed as a value argument */
                        if (arg_t->kind == TK_UNKNOWN &&
                            node->data.call.args[ai]->kind == NODE_LABEL &&
                            fn_name && strcmp(fn_name, "size_of") != 0) {
                            const char *arg_label = node->data.call.args[ai]->data.label.value;
                            if (!scope_lookup(tc->current_scope, arg_label)) {
                                if (is_struct_name(tc, arg_label)) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type name '%s' cannot be used as a value; to create an instance use 'new(%s)' or '%s{...}'",
                                        arg_label, arg_label, arg_label);
                                    diag_error_msg(tc->diag, "E3100", msg,
                                        NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0);
                                    continue;
                                } else if (is_enum_name(tc, arg_label)) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type name '%s' cannot be used as a value; use '%s.VARIANT' to access an enum value",
                                        arg_label, arg_label);
                                    diag_error_msg(tc->diag, "E3100", msg,
                                        NODE_FILE(tc, node->data.call.args[ai]),
                                        node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0);
                                    continue;
                                }
                            }
                        }
                        if (arg_t->kind != TK_UNKNOWN && param_t->kind != TK_UNKNOWN &&
                            arg_t->kind != param_t->kind &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_ENUM) &&
                            !(param_t->kind == TK_ENUM && is_int_kind(arg_t->kind)) &&
                            !(param_t->kind == TK_STRUCT && is_int_kind(arg_t->kind)) &&
                            !(is_int_kind(param_t->kind) && arg_t->kind == TK_BOOL) &&
                            !(param_t->kind == TK_FLOAT && is_int_kind(arg_t->kind))) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s': expected %s, got %s",
                                ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Enum-to-enum: kinds both TK_ENUM but different names */
                        if (arg_t->kind == TK_ENUM && param_t->kind == TK_ENUM &&
                            arg_t->name && param_t->name &&
                            !tc_same_enum_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s': expected enum '%s', got enum '%s'",
                                ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Struct-to-struct: kinds both TK_STRUCT but different names */
                        if (arg_t->kind == TK_STRUCT && param_t->kind == TK_STRUCT &&
                            arg_t->name && param_t->name &&
                            !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s': expected struct '%s', got struct '%s'",
                                ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Pointer-to-pointer: pointee types differ (e.g., addr(Color) to ^Point) */
                        if (arg_t->kind == TK_POINTER && param_t->kind == TK_POINTER &&
                            arg_t->name && param_t->name &&
                            !tc_same_struct_type(tc, arg_t->name, param_t->name)) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s': expected '%s', got '%s'",
                                ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3001", msg,
                                NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* Bigint narrowing in call argument: i128 arg to i64 param, etc. */
                        if (arg_t->name && param_t->name) {
                            int ar = int_name_rank(arg_t->name);
                            int pr = int_name_rank(param_t->name);
                            if (ar >= 5 && pr > 0 && pr < ar) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "argument %d of '%s': cannot implicitly narrow %s to %s; use %s() to convert explicitly",
                                    ai + 1, fn_name, arg_t->name, param_t->name, param_t->name);
                                diag_error_msg(tc->diag, "E3001", msg,
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                            }
                        }
                        /* Array element type mismatch */
                        if (arg_t->kind == TK_ARRAY && param_t->kind == TK_ARRAY &&
                            arg_t->element_type && param_t->element_type &&
                            !tc_same_array_elem(tc, arg_t->element_type, param_t->element_type)) {
                            GrayType *ae = type_from_name(arg_t->element_type);
                            GrayType *pe = type_from_name(param_t->element_type);
                            if (!(ae && pe && is_int_kind(ae->kind) && is_int_kind(pe->kind))) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "argument %d of '%s': expected '%s', got '%s'",
                                    ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                diag_error_msg(tc->diag, "E3001", msg,
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                            }
                        }
                        /* Map key/value type mismatch */
                        if (arg_t->kind == TK_MAP && param_t->kind == TK_MAP) {
                            bool key_mismatch = arg_t->key_type && param_t->key_type &&
                                strcmp(arg_t->key_type, param_t->key_type) != 0;
                            bool val_mismatch = arg_t->value_type && param_t->value_type &&
                                strcmp(arg_t->value_type, param_t->value_type) != 0;
                            if (key_mismatch || val_mismatch) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "argument %d of '%s': expected '%s', got '%s'",
                                    ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                                diag_error_msg(tc->diag, "E3001", msg,
                                    NODE_FILE(tc, node->data.call.args[ai]), node->data.call.args[ai]->token.line,
                                    node->data.call.args[ai]->token.column, 0);
                            }
                        }
                        /* E3066: typed-func signatures must match exactly */
                        if (arg_t->kind == TK_FUNCTION && param_t->kind == TK_FUNCTION &&
                            arg_t->name && param_t->name &&
                            strcmp(arg_t->name, param_t->name) != 0) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "argument %d of '%s': expected %s, got %s",
                                ai + 1, fn_name, type_display_name(tc, param_t), type_display_name(tc, arg_t));
                            diag_error_msg(tc->diag, "E3066", msg,
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
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "cannot pass constant '%s' to mutable parameter '%s' of '%s'",
                                            arg_sym->name, s->data.func_decl.params[ai].name, fn_name);
                                        diag_error_msg(tc->diag, "E3027", msg,
                                            NODE_FILE(tc, arg), arg->token.line,
                                            arg->token.column, 0);
                                    }
                                } else if (arg->kind == NODE_MEMBER_EXPR &&
                                           arg->data.member.object->kind == NODE_LABEL &&
                                           is_enum_name(tc, arg->data.member.object->data.label.value)) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "cannot pass enum constant to mutable parameter '%s' of '%s'; expected a mutable variable",
                                        s->data.func_decl.params[ai].name, fn_name);
                                    diag_error_msg(tc->diag, "E3027", msg,
                                        NODE_FILE(tc, arg), arg->token.line,
                                        arg->token.column, 0);
                                } else if (arg->kind != NODE_MEMBER_EXPR &&
                                           arg->kind != NODE_INDEX_EXPR &&
                                           arg->kind != NODE_PREFIX_EXPR) {
                                    /* Literal, call result, or other non-lvalue expression */
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "cannot pass a literal or expression to mutable parameter '%s' of '%s'; expected a mutable variable",
                                        s->data.func_decl.params[ai].name, fn_name);
                                    diag_error_msg(tc->diag, "E3027", msg,
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
                                char *msg = NULL;
                                if (min_arity == ref_sig->param_count) {
                                    msg = tc_fmt(tc,
                                        "function '%s' expects %d argument(s), got %d",
                                        func_display_name(ref_sig), ref_sig->param_count, ac);
                                } else {
                                    msg = tc_fmt(tc,
                                        "function '%s' expects %d to %d argument(s), got %d",
                                        func_display_name(ref_sig), min_arity, ref_sig->param_count, ac);
                                }
                                diag_error_msg(tc->diag, "E5008", msg,
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            } else {
                                for (int ai = 0; ai < ref_sig->param_count; ai++) {
                                    GrayType *at = resolve_expr(tc, node->data.call.args[ai]);
                                    GrayType *pt = ref_sig->param_types[ai];
                                    if (at && pt && at->kind != TK_UNKNOWN &&
                                        pt->kind != TK_UNKNOWN &&
                                        at->kind != pt->kind &&
                                        !(is_int_kind(at->kind) && is_int_kind(pt->kind)) &&
                                        !(pt->kind == TK_FLOAT && is_int_kind(at->kind))) {
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "argument %d of '%s': expected %s, got %s",
                                            ai + 1, func_display_name(ref_sig),
                                            type_display_name(tc, pt), type_display_name(tc, at));
                                        diag_error_msg(tc->diag, "E3001", msg,
                                            NODE_FILE(tc, node->data.call.args[ai]),
                                            node->data.call.args[ai]->token.line,
                                            node->data.call.args[ai]->token.column, 0);
                                    }
                                    /* Enum-to-enum: different enum types */
                                    if (at && pt && at->kind == TK_ENUM && pt->kind == TK_ENUM &&
                                        at->name && pt->name &&
                                        strcmp(at->name, pt->name) != 0) {
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "argument %d of '%s': expected enum '%s', got enum '%s'",
                                            ai + 1, func_display_name(ref_sig), enum_display_name(tc, pt->name), enum_display_name(tc, at->name));
                                        diag_error_msg(tc->diag, "E3001", msg,
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
                                                    char emsg[MSG_BUF_SIZE];
                                                    snprintf(emsg, sizeof(emsg),
                                                        "cannot pass constant '%s' to mutable parameter '%s' of '%s'",
                                                        arg_sym->name, s->data.func_decl.params[ai].name,
                                                        func_display_name(ref_sig));
                                                    diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                        NODE_FILE(tc, arg), arg->token.line,
                                                        arg->token.column, 0);
                                                }
                                            } else if (arg->kind == NODE_MEMBER_EXPR &&
                                                       arg->data.member.object->kind == NODE_LABEL &&
                                                       is_enum_name(tc, arg->data.member.object->data.label.value)) {
                                                char emsg[MSG_BUF_SIZE];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass enum constant to mutable parameter '%s' of '%s'; expected a mutable variable",
                                                    s->data.func_decl.params[ai].name, func_display_name(ref_sig));
                                                diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                    NODE_FILE(tc, arg), arg->token.line,
                                                    arg->token.column, 0);
                                            } else if (arg->kind != NODE_MEMBER_EXPR &&
                                                       arg->kind != NODE_INDEX_EXPR &&
                                                       arg->kind != NODE_PREFIX_EXPR) {
                                                char emsg[MSG_BUF_SIZE];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass a literal or expression to mutable parameter '%s' of '%s'; expected a mutable variable",
                                                    s->data.func_decl.params[ai].name, func_display_name(ref_sig));
                                                diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
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
                            GrayFuncSig *sig = fn_sym->type->func_sig;
                            int ac = node->data.call.arg_count;
                            if (ac != sig->param_count) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "function reference '%s' expects %d argument(s), got %d",
                                    fn_name, sig->param_count, ac);
                                diag_error_msg(tc->diag, "E5008", msg,
                                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            } else {
                                for (int ai = 0; ai < sig->param_count; ai++) {
                                    AstNode *arg = node->data.call.args[ai];
                                    GrayType *at = resolve_expr(tc, arg);
                                    GrayType *pt = sig->param_types[ai] ? type_from_name(sig->param_types[ai]) : NULL;
                                    if (at && pt && at->kind != TK_UNKNOWN && pt->kind != TK_UNKNOWN &&
                                        at->kind != pt->kind &&
                                        !(is_int_kind(at->kind) && is_int_kind(pt->kind)) &&
                                        !(pt->kind == TK_FLOAT && is_int_kind(at->kind))) {
                                        char *msg = NULL;
                                        msg = tc_fmt(tc,
                                            "argument %d of '%s': expected %s, got %s",
                                            ai + 1, fn_name,
                                            type_display_name(tc, pt), type_display_name(tc, at));
                                        diag_error_msg(tc->diag, "E3001", msg,
                                            NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0);
                                    }
                                    /* E3027/E3067: `&` param requires an lvalue */
                                    if (sig->param_mutable[ai]) {
                                        bool is_enum_const = (arg->kind == NODE_MEMBER_EXPR &&
                                                              arg->data.member.object->kind == NODE_LABEL &&
                                                              is_enum_name(tc, arg->data.member.object->data.label.value));
                                        bool is_lvalue = !is_enum_const &&
                                                         (arg->kind == NODE_LABEL ||
                                                          arg->kind == NODE_MEMBER_EXPR ||
                                                          arg->kind == NODE_INDEX_EXPR);
                                        if (is_enum_const) {
                                            char emsg[MSG_BUF_SIZE];
                                            snprintf(emsg, sizeof(emsg),
                                                "cannot pass enum constant to '&' parameter %d of '%s'; expected a mutable variable",
                                                ai + 1, fn_name);
                                            diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
                                                NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0);
                                        } else if (!is_lvalue) {
                                            diag_error_codef(tc->diag, "E3067", NODE_FILE(tc, arg), arg->token.line, arg->token.column, 0, ai + 1, fn_name);
                                        } else if (arg->kind == NODE_LABEL) {
                                            Symbol *as = scope_lookup(tc->current_scope, arg->data.label.value);
                                            if (as && !as->mutable) {
                                                char emsg[MSG_BUF_SIZE];
                                                snprintf(emsg, sizeof(emsg),
                                                    "cannot pass constant '%s' to '&' parameter %d of '%s'",
                                                    as->name, ai + 1, fn_name);
                                                diag_error_msg(tc->diag, "E3027", arena_strdup(tc->arena, emsg),
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
                        diag_error_codef(tc->diag, "E3015", NODE_FILE(tc, node), node->token.line, node->token.column, 0, fn_name, type_display_name(tc, fn_sym->type));
                    } else if (!tc_is_builtin(fn_name)) {
                        /* Check if it's a function from a 'using' module */
                        bool found_in_using = false;
                        /* Check for math functions whose return type depends on argument */
                        if (!found_in_using && (strcmp(fn_name, "abs") == 0 || strcmp(fn_name, "neg") == 0 ||
                            strcmp(fn_name, "min") == 0 || strcmp(fn_name, "max") == 0 ||
                            strcmp(fn_name, "clamp") == 0)) {
                            for (int ui = 0; ui < tc->using_module_count; ui++) {
                                if (!using_module_accessible(tc, ui)) continue;
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "math") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        GrayType *arg_t = resolve_expr(tc, node->data.call.args[0]);
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
                                if (!using_module_accessible(tc, ui)) continue;
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "random") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        GrayType *arr_t = resolve_expr(tc, node->data.call.args[0]);
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
                                if (!using_module_accessible(tc, ui)) continue;
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                if (strcmp(real_mod, "maps") == 0) {
                                    found_in_using = true;
                                    if (node->data.call.arg_count > 0) {
                                        GrayType *map_t = resolve_expr(tc, node->data.call.args[0]);
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
                            if (!using_module_accessible(tc, ui)) continue;
                            const char *umod = tc->using_modules[ui];
                            /* Resolve alias to actual module name */
                            const char *real_mod = tc_resolve_alias(tc, umod);
                            /* 1) Try hardcoded stdlib table */
                            for (int fi = 0; _using_funcs[fi].func; fi++) {
                                if (strcmp(fn_name, _using_funcs[fi].func) == 0 &&
                                    strcmp(real_mod, _using_funcs[fi].mod) == 0) {
                                    found_in_using = true;
                                    switch (_using_funcs[fi].ret) {
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
                                char prefixed[MSG_BUF_SIZE];
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
                                char *msg = NULL;
                                msg = tc_fmt(tc, "undefined function '%s'", fn_name);
                                const char *suggestion = suggest_name(tc, fn_name);
                                /* Point at the function name, not the ( */
                                AstNode *fn_node = node->data.call.function;
                                int el = fn_node ? fn_node->token.line : node->token.line;
                                int ec = fn_node ? fn_node->token.column : node->token.column;
                                if (suggestion) {
                                    char help[MSG_BUF_SIZE];
                                    snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                                    diag_error_help(tc->diag, "E4002", msg,
                                        NODE_FILE(tc, node), el, ec, 0, arena_strdup(tc->arena, help));
                                } else {
                                    diag_error_msg(tc->diag, "E4002", msg,
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
            char prefixed_type[MSG_BUF_SIZE];
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
            if (strcmp(obj_name, "io") == 0) {
                const char *mem = node->data.member.member;
                if (strcmp(mem, "O_RDONLY") == 0 || strcmp(mem, "O_WRONLY") == 0 ||
                    strcmp(mem, "O_RDWR") == 0) {
                    result = &TYPE_INT;
                    break;
                }
            }
            if (strcmp(obj_name, "strconv") == 0) {
                const char *mem = node->data.member.member;
                if (strcmp(mem, "BASE_2") == 0 || strcmp(mem, "BASE_8") == 0 ||
                    strcmp(mem, "BASE_10") == 0 || strcmp(mem, "BASE_16") == 0 ||
                    strcmp(mem, "BASE_36") == 0) {
                    result = &TYPE_INT;
                    break;
                }
            }
            if (strcmp(obj_name, "uuid") == 0) {
                const char *mem = node->data.member.member;
                if (strcmp(mem, "NIL_UUID") == 0) {
                    result = type_struct("UUID");
                    break;
                }
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
                result = type_enum(obj_name);
                break;
            }

            /* C interop constant access: c.EOF, c.NULL, etc. */
            if (strcmp(obj_name, "c") == 0 && tc_is_imported_module(tc, "c")) {
                result = &TYPE_UNKNOWN;
                break;
            }

            /* Check for user-module constant access: mod.CONST */
            if (tc_is_imported_module(tc, obj_name)) {
                char prefixed[MSG_BUF_SIZE];
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
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0, struct_display_name(tc, sym->type->name), member);
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
                        "too many variables; the function returns only 1 value",
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
            } else if (sym && sym->type->kind == TK_ERROR) {
                /* Error type has .message and .code string fields */
                if (strcmp(member, "message") == 0 || strcmp(member, "code") == 0) {
                    result = &TYPE_STRING;
                } else {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "Error has no field '%s'; available fields: message, code",
                        member);
                    diag_error_msg(tc->diag, "E3010", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    result = &TYPE_UNKNOWN;
                }
            } else if (sym && sym->type->kind != TK_UNKNOWN &&
                       sym->type->kind != TK_STRUCT && sym->type->kind != TK_ENUM &&
                       sym->type->kind != TK_POINTER &&
                       !(member[0] == 'v' && member[1] >= '0' && member[1] <= '9')) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type '%s' does not support access via dot notation",
                    type_name(sym->type));
                diag_error_msg(tc->diag, "E3013", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Struct-namespaced function or enum access: Type.func() / Type.MEMBER */
            if (!sym && is_struct_name(tc, obj_name)) {
                /* Check if member is a field; can't access fields on the type itself */
                GrayType *ft = struct_field_type(tc, obj_name, member);
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
                    char prefixed[MSG_BUF_SIZE];
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
            GrayType *obj_t = typetable_get(tc->type_table, obj);
            if (!obj_t) obj_t = resolve_expr(tc, obj);
            if (obj_t && obj_t->kind == TK_STRUCT) {
                result = struct_field_type(tc, obj_t->name, member);
                bool is_func_field = (result->kind == TK_UNKNOWN && result->name &&
                                      strcmp(result->name, "func") == 0);
                if (result->kind == TK_UNKNOWN && !is_func_field && member[0] != 'v') {
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        struct_display_name(tc, obj_t->name), member);
                }
            } else if (obj_t && obj_t->kind == TK_POINTER) {
                /* Auto-deref pointer field: a.next.val where a.next is ^Node */
                result = struct_field_type(tc, obj_t->element_type, member);
                if (result->kind == TK_UNKNOWN && member[0] != 'v') {
                    diag_error_codef(tc->diag, "E3010", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        obj_t->element_type, member);
                }
            } else if (obj_t && obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_STRUCT) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type '%s' does not support access via dot notation",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else {
            /* Object is an expression (e.g. foo().bar); resolve its type */
            GrayType *obj_t = resolve_expr(tc, obj);
            if (obj_t && obj_t->kind == TK_STRUCT) {
                result = struct_field_type(tc, obj_t->name, member);
            } else if (obj_t && obj_t->kind != TK_UNKNOWN && obj_t->kind != TK_VOID) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type '%s' does not support access via dot notation",
                    type_name(obj_t));
                diag_error_msg(tc->diag, "E3013", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        break;
    }

    case NODE_INDEX_EXPR: {
        GrayType *left = resolve_expr(tc, node->data.index_expr.left);
        GrayType *idx_t = resolve_expr(tc, node->data.index_expr.index);
        /* E3003: array index must be integer */
        if (left->kind == TK_ARRAY && idx_t->kind != TK_UNKNOWN &&
            !is_int_kind(idx_t->kind) && idx_t->kind != TK_BYTE) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "array index must be an integer, got %s", type_name(idx_t));
            diag_error_msg(tc->diag, "E3003", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Reject negative literal index at compile time. Applies to
         * arrays and strings (); the analogous runtime panic
         * fires for both, and there's no reason to wait until then
         * when the index is a literal '-N'. */
        if ((left->kind == TK_ARRAY || left->kind == TK_STRING) &&
            node->data.index_expr.index->kind == NODE_PREFIX_EXPR &&
            node->data.index_expr.index->data.prefix.op == TOK_MINUS &&
            node->data.index_expr.index->data.prefix.right->kind == NODE_INT_VALUE) {
            const char *what = left->kind == TK_STRING ? "string" : "array";
            char *msg;
            msg = tc_fmt(tc, "%s index cannot be negative", what);
            diag_error_msg(tc->diag, "E3003", msg,
                NODE_FILE(tc, node->data.index_expr.index), node->data.index_expr.index->token.line,
                node->data.index_expr.index->token.column, 0);
        }
        if (left->kind == TK_ARRAY && left->element_type) {
            result = tc_type_from_name(tc, left->element_type);
        } else if (left->kind == TK_MAP && left->value_type) {
            result = type_from_name(left->value_type);
            /* Check map key type matches. Enum keys are int-backed, so accept
             * int expressions (and enum members, which resolve as int) when
             * the declared key is a user enum name. */
            if (left->key_type && idx_t->kind != TK_UNKNOWN) {
                GrayType *key_t = tc_type_from_name(tc, left->key_type);
                bool declared_is_enum = is_enum_name(tc, left->key_type);
                bool compatible =
                    key_t->kind == TK_UNKNOWN ||
                    key_t->kind == idx_t->kind ||
                    (is_int_kind(key_t->kind) && is_int_kind(idx_t->kind)) ||
                    (declared_is_enum && is_int_kind(idx_t->kind));
                if (!compatible) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "map key type mismatch: expected '%s', got '%s'",
                        left->key_type, type_name(idx_t));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        } else if (left->kind == TK_STRING) {
            if (idx_t->kind != TK_UNKNOWN && !is_int_kind(idx_t->kind) && idx_t->kind != TK_BYTE) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "string index must be an integer, got %s", type_name(idx_t));
                diag_error_msg(tc->diag, "E3003", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            result = &TYPE_CHAR;
        } else if (left->kind != TK_UNKNOWN) {
            diag_error_codef(tc->diag, "E3008", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, left));
        }
        break;
    }

    case NODE_ARRAY_VALUE: {
        /* If expected_type is an array-of-enum, propagate element type for .VARIANT */
        GrayType *saved_arr_expected = tc->expected_type;
        if (tc->expected_type && tc->expected_type->kind == TK_ARRAY &&
            tc->expected_type->element_type) {
            GrayType *elem_t = tc_type_from_name(tc, tc->expected_type->element_type);
            if (elem_t && elem_t->kind == TK_ENUM && elem_t->name)
                tc->expected_type = elem_t;
        }
        if (node->data.array_value.count > 0) {
            GrayType *first = resolve_expr(tc, node->data.array_value.elements[0]);
            result = type_array(type_name(first));
            /* Validate all elements have the same type */
            for (int i = 1; i < node->data.array_value.count; i++) {
                GrayType *ei = resolve_expr(tc, node->data.array_value.elements[i]);
                if (!ei || ei->kind == TK_UNKNOWN || !first || first->kind == TK_UNKNOWN)
                    continue;
                bool compatible = (ei->kind == first->kind) ||
                    (is_int_kind(ei->kind) && is_int_kind(first->kind)) ||
                    (ei->kind == TK_BYTE && is_int_kind(first->kind)) ||
                    (is_int_kind(ei->kind) && first->kind == TK_BYTE);
                if (!compatible) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "array elements must all be the same type; element %d is '%s' but the array is '%s'",
                        i, type_name(ei), type_name(first));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node->data.array_value.elements[i]), node->data.array_value.elements[i]->token.line,
                        node->data.array_value.elements[i]->token.column, 0);
                    break;
                }
            }
        }
        tc->expected_type = saved_arr_expected;
        break;
    }

    case NODE_MAP_VALUE: {
        /* Resolve key and value types */
        for (int i = 0; i < node->data.map_value.count; i++) {
            GrayType *kt = resolve_expr(tc, node->data.map_value.keys[i]);
            GrayType *vt = resolve_expr(tc, node->data.map_value.values[i]);
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
        GrayType *t = type_alloc();
        t->kind = TK_MAP;
        t->name = strdup("map");
        if (node->data.map_value.count > 0) {
            GrayType *kt = typetable_get(tc->type_table, node->data.map_value.keys[0]);
            GrayType *vt = typetable_get(tc->type_table, node->data.map_value.values[0]);
            t->key_type = strdup(kt ? type_name(kt) : "unknown");
            t->value_type = strdup(vt ? type_name(vt) : "unknown");
        }
        result = t;
        break;
    }

    case NODE_STRUCT_VALUE: {
        const char *sname = node->data.struct_value.name;
        /* Type parameter: rewrite T → "?" so codegen can substitute */
        if (tc->type_param_name && strcmp(sname, tc->type_param_name) == 0) {
            node->data.struct_value.name = "?";
            sname = "?";
        }
        if (strcmp(sname, "?") == 0) {
            /* During re-check with a binding, validate with concrete struct */
            if (tc->type_param_binding) {
                sname = tc->type_param_binding;
            } else {
                /* Main pass — skip field validation, return unknown */
                for (int i = 0; i < node->data.struct_value.count; i++)
                    resolve_expr(tc, node->data.struct_value.field_values[i]);
                result = &TYPE_UNKNOWN;
                break;
            }
        }
        tc_mark_type_module_used(tc, sname);
        StructInfo *si = find_struct(tc, sname);
        /* E4016: reject undefined/unimported struct types in struct literals */
        if (!si && !is_struct_name(tc, sname)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "undefined type '%s'; check the spelling or import the module that defines it",
                sname);
            diag_error_msg(tc->diag, "E4016", msg,
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
            /* Look up expected field type for implicit enum resolution */
            GrayType *field_expected_t = NULL;
            if (si && node->data.struct_value.field_names[i]) {
                const char *fname_pre = node->data.struct_value.field_names[i];
                for (int j = 0; j < si->field_count; j++) {
                    if (strcmp(si->field_names[j], fname_pre) == 0) {
                        field_expected_t = si->field_types[j];
                        break;
                    }
                }
            }
            GrayType *saved_sv_expected = tc->expected_type;
            if (field_expected_t && field_expected_t->kind == TK_ENUM && field_expected_t->name)
                tc->expected_type = field_expected_t;
            GrayType *val_t = resolve_expr(tc, node->data.struct_value.field_values[i]);
            tc->expected_type = saved_sv_expected;
            /* Validate field exists */
            if (si && node->data.struct_value.field_names[i]) {
                const char *fname = node->data.struct_value.field_names[i];
                bool found = false;
                GrayType *expected_t = NULL;
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
                           /* kinds differ, OR both are pointers to different types,
                            * OR both are structs with different names */
                           (expected_t->kind != val_t->kind ||
                            (expected_t->kind == TK_POINTER &&
                             expected_t->name && val_t->name &&
                             strcmp(expected_t->name, val_t->name) != 0) ||
                            (expected_t->kind == TK_STRUCT && val_t->kind == TK_STRUCT &&
                             expected_t->name && val_t->name &&
                             strcmp(expected_t->name, val_t->name) != 0)) &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_ENUM) &&
                           !(expected_t->kind == TK_ENUM && is_int_kind(val_t->kind)) &&
                           !(expected_t->kind == TK_STRUCT && is_int_kind(val_t->kind)) &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_STRUCT) &&
                           !(expected_t->kind == TK_FLOAT && is_int_kind(val_t->kind)) &&
                           /* nil is a valid value for pointer and Error fields */
                           !(val_t->kind == TK_NIL &&
                             (expected_t->kind == TK_POINTER || expected_t->kind == TK_ERROR))) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "field '%s' of struct '%s': expected %s, got %s",
                        fname, struct_display_name(tc, sname), type_display_name(tc, expected_t), type_display_name(tc, val_t));
                    diag_error_msg(tc->diag, "E3001", msg,
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
                        GrayType *val_t = typetable_get(tc->type_table,
                            node->data.struct_value.field_values[i]);
                        if (!val_t) val_t = resolve_expr(tc, node->data.struct_value.field_values[i]);
                        if (val_t && val_t->kind != TK_UNKNOWN) {
                            const char *concrete = type_name(val_t);
                            if (!binding) {
                                binding = concrete;
                            } else if (strcmp(binding, concrete) != 0) {
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "wildcard type conflict in struct '%s': '?' was bound to %s, but field '%s' is %s",
                                    sname, binding, fname, concrete);
                                diag_error_msg(tc->diag, "E3001", msg,
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
                    sdecl->data.struct_decl.instantiations = xrealloc(
                        (void *)sdecl->data.struct_decl.instantiations,
                        sizeof(const char *) * (size_t)(n + 1));
                    sdecl->data.struct_decl.instantiations[n] = strdup(binding);
                    sdecl->data.struct_decl.instantiation_count = n + 1;
                }
                /* Return mangled struct type */
                char mangled[MSG_BUF_SIZE];
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
            GrayType *pt = resolve_expr(tc, parts[ri]);
            if (pt->kind != TK_UNKNOWN && !is_int_kind(pt->kind)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "range() %s argument must be an integer type, got '%s'",
                    labels[ri], type_name(pt));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), parts[ri]->token.line,
                    parts[ri]->token.column, 0);
            }
        }
        GrayType *rt = type_alloc();
        rt->kind = TK_INT;
        rt->name = strdup("Range<int>");
        result = rt;
        break;
    }

    case NODE_IMPLICIT_ENUM:
        result = resolve_implicit_enum(tc, node);
        break;

    case NODE_WHEN_PATTERN:
        /* Pattern nodes are validated in the NODE_WHEN_STMT handler */
        if (node->data.when_pattern.enum_name)
            result = type_enum(node->data.when_pattern.enum_name);
        else
            result = &TYPE_UNKNOWN;
        break;

    case NODE_CAST_EXPR: {
        GrayType *src_t = resolve_expr(tc, node->data.cast.value);
        GrayType *dst_t = type_from_name(node->data.cast.target_type);
        /* Resolve user-defined enum types that type_from_name() can't find */
        if (!dst_t && is_enum_name(tc, node->data.cast.target_type))
            dst_t = type_enum(node->data.cast.target_type);
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
            /* String -> numeric: NOT allowed via cast(); use dedicated parsing
             * functions (e.g. strings.parse_int()) instead. cast() is for
             * type reinterpretation, not fallible string parsing. */
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
                    GrayType *src_elem = type_from_name(src_t->element_type);
                    GrayType *dst_elem = type_from_name(dst_t->element_type);
                    if (type_is_numeric(src_elem) && type_is_numeric(dst_elem))
                        allowed = true;
                } else {
                    allowed = true;
                }
            }
            if (!allowed) {
                char tn[TYPE_NAME_MAX];
                if (src_t->kind == TK_ARRAY && src_t->element_type)
                    snprintf(tn, sizeof(tn), "[%s]", src_t->element_type);
                else {
                    strncpy(tn, type_name(src_t), sizeof(tn) - 1);
                    tn[sizeof(tn) - 1] = '\0';
                }
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot cast '%s' to '%s'",
                    tn, node->data.cast.target_type);
                diag_error_help(tc->diag, "E3043", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "only primitive-to-primitive casts are supported (e.g. cast(x, int), cast(x, string))");
            }
        }
        result = dst_t;
        break;
    }

    case NODE_NEW_EXPR: {
        const char *new_type = node->data.new_expr.type_name;
        /* Type parameter: rewrite T → "?" so codegen can substitute */
        if (tc->type_param_name && strcmp(new_type, tc->type_param_name) == 0) {
            node->data.new_expr.type_name = "?";
            new_type = "?";
        }
        if (strcmp(new_type, "?") == 0) {
            /* During re-check with a binding, validate the concrete type */
            if (tc->type_param_binding) {
                result = type_pointer(tc->type_param_binding);
            } else {
                result = type_pointer("?");
            }
        } else {
            tc_mark_type_module_used(tc, new_type);
            if (!is_struct_name(tc, new_type)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "new() requires a struct type, but '%s' is not a struct",
                    new_type);
                diag_error_msg(tc->diag, "E3041", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            result = type_pointer(new_type);
        }
        break;
    }

    case NODE_FUNC_REF: {
        /* Validate that the referenced function exists.
         * Builtin and stdlib functions cannot be used as function references. */
        const char *ref_name = NULL;
        const char *ref_struct_name = NULL;  /* struct name for privacy check */
        const char *ref_member_name = NULL;  /* member name for privacy check */
        if (node->data.func_ref.function->kind == NODE_LABEL) {
            const char *lname = node->data.func_ref.function->data.label.value;
            /* Surface 1: ()builtin_name — builtins are not first-class values */
            if (tc_is_builtin(lname)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot take a function reference to '%s'; builtin functions are not first-class values", lname);
                diag_error_msg(tc->diag, "E4019", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            } else {
                ref_name = lname;
            }
        } else if (node->data.func_ref.function->kind == NODE_MEMBER_EXPR) {
            AstNode *obj = node->data.func_ref.function->data.member.object;
            const char *member = node->data.func_ref.function->data.member.member;
            if (obj->kind == NODE_LABEL) {
                const char *mod_name = tc_resolve_alias(tc, obj->data.label.value);
                /* Surface 2: ()module.func — stdlib module functions are not first-class values */
                if (is_stdlib_module_name(mod_name)) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot take a function reference to '%s.%s'; stdlib functions are not first-class values",
                        obj->data.label.value, member);
                    diag_error_msg(tc->diag, "E4019", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                } else {
                    /* Struct.func → lookup as Struct_func */
                    ref_struct_name = obj->data.label.value;
                    ref_member_name = member;
                    char buf[MSG_BUF_SIZE];
                    snprintf(buf, sizeof(buf), "%s_%s", obj->data.label.value, member);
                    ref_name = arena_strdup(tc->arena, buf);
                }
            }
        }
        FuncSig *ref_sig = ref_name ? find_func(tc, ref_name) : NULL;
        if (ref_sig) {
            ref_sig->used = true;
            /* E4017: private struct function referenced from outside the struct */
            if (ref_sig->is_private && ref_struct_name &&
                !(tc->current_struct_name &&
                  strcmp(tc->current_struct_name, ref_struct_name) == 0)) {
                diag_error_codef(tc->diag, "E4017", NODE_FILE(tc, node),
                    node->token.line, node->token.column, 0,
                    ref_struct_name, ref_member_name);
            }
        } else if (ref_name) {
            /* Surface 3: using module; ()stdlib_func — check if ref_name is in an active using module */
            bool found_in_using = false;
            for (int ui = 0; ui < tc->using_module_count && !found_in_using; ui++) {
                if (!using_module_accessible(tc, ui)) continue;
                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                if (is_stdlib_module_name(real_mod)) {
                    for (int fi = 0; _using_funcs[fi].func; fi++) {
                        if (strcmp(ref_name, _using_funcs[fi].func) == 0 &&
                            strcmp(real_mod, _using_funcs[fi].mod) == 0) {
                            found_in_using = true;
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "cannot take a function reference to '%s'; stdlib functions are not first-class values",
                                ref_name);
                            diag_error_msg(tc->diag, "E4019", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            break;
                        }
                    }
                }
            }
            if (!found_in_using) {
                char *msg = NULL;
                msg = tc_fmt(tc, "undefined function '%s' in function reference", ref_name);
                const char *suggestion = suggest_name(tc, ref_name);
                if (suggestion) {
                    char help[MSG_BUF_SIZE];
                    snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                    diag_error_help(tc->diag, "E4002", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0, arena_strdup(tc->arena, help));
                } else {
                    diag_error_msg(tc->diag, "E4002", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
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
            /* Unknown function: fall back to bare-func type
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
 * Skip names starting with _gray_ as those are compiler-generated temporaries. */
static void check_reserved_name(TypeChecker *tc, const char *name, const char *file, int line, int col) {
    if (!name) return;
    /* Skip compiler-generated temps (_gray_tmp, _gray_or, _gray_idx, etc.) */
    if (strncmp(name, "_gray_", 6) == 0) return;
    if (strncmp(name, "gray_", 5) == 0 || strncmp(name, "Gray", 4) == 0) {
        diag_error_codef(tc->diag, "E4006", file, line, col, 0, name);
    }
}

/* --- Statement checking --- */

static void check_statement(TypeChecker *tc, AstNode *node);

/* Walk an expression tree looking for a function call anywhere inside.
 * Used by the file-scope initializer guard (E5013): runtime calls in
 * a top-level var_decl produce invalid C (free-standing init or
 * non-constant initializer), so the Grayscale side must reject them with
 * a real diagnostic before codegen runs. */
static bool expr_contains_call(AstNode *node) {
    if (!node) return false;
    switch (node->kind) {
    case NODE_CALL_EXPR:
        /* embed() is a compile-time builtin and is allowed at file scope */
        if (node->data.call.function &&
            node->data.call.function->kind == NODE_LABEL &&
            strcmp(node->data.call.function->data.label.value, "embed") == 0) {
            return false;
        }
        return true;
    case NODE_PREFIX_EXPR:
        return expr_contains_call(node->data.prefix.right);
    case NODE_INFIX_EXPR:
        return expr_contains_call(node->data.infix.left) ||
               expr_contains_call(node->data.infix.right);
    case NODE_POSTFIX_EXPR:
        return expr_contains_call(node->data.postfix.left);
    case NODE_INDEX_EXPR:
        return expr_contains_call(node->data.index_expr.left) ||
               expr_contains_call(node->data.index_expr.index);
    case NODE_MEMBER_EXPR:
        return expr_contains_call(node->data.member.object);
    case NODE_RANGE_EXPR:
        return expr_contains_call(node->data.range_expr.start) ||
               expr_contains_call(node->data.range_expr.end) ||
               expr_contains_call(node->data.range_expr.step);
    case NODE_CAST_EXPR:
        return expr_contains_call(node->data.cast.value);
    case NODE_ARRAY_VALUE:
        for (int i = 0; i < node->data.array_value.count; i++) {
            if (expr_contains_call(node->data.array_value.elements[i])) return true;
        }
        return false;
    case NODE_MAP_VALUE:
        for (int i = 0; i < node->data.map_value.count; i++) {
            if (expr_contains_call(node->data.map_value.keys[i])) return true;
            if (expr_contains_call(node->data.map_value.values[i])) return true;
        }
        return false;
    case NODE_STRUCT_VALUE:
        for (int i = 0; i < node->data.struct_value.count; i++) {
            if (expr_contains_call(node->data.struct_value.field_values[i])) return true;
        }
        return false;
    case NODE_INTERPOLATED_STRING:
        for (int i = 0; i < node->data.interpolated_string.part_count; i++) {
            if (expr_contains_call(node->data.interpolated_string.parts[i])) return true;
        }
        return false;
    default:
        return false;
    }
}

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
    /* E3006: multi-var destructuring with fewer variables than return values.
     * Desugared blocks look like: _gray_tmpN = call(); a = _gray_tmpN.v0; b = _gray_tmpN.v1; ...
     * If var_count < ret_count, trailing return values are silently lost. */
    if (node->data.block.count >= 2) {
        AstNode *first = node->data.block.stmts[0];
        if (first && first->kind == NODE_VAR_DECL &&
            strncmp(first->data.var_decl.name, "_gray_tmp", 9) == 0 &&
            first->data.var_decl.value &&
            first->data.var_decl.value->kind == NODE_CALL_EXPR) {
            Symbol *sym = scope_lookup_local(tc->current_scope, first->data.var_decl.name);
            if (sym && sym->ret_types && sym->ret_count > 0) {
                int var_count = node->data.block.count - 1;
                if (var_count < sym->ret_count) {
                    /* Extract function name for the error message */
                    AstNode *call_fn = first->data.var_decl.value->data.call.function;
                    const char *fn_name = "function";
                    if (call_fn->kind == NODE_LABEL) {
                        fn_name = call_fn->data.label.value;
                    } else if (call_fn->kind == NODE_MEMBER_EXPR &&
                               call_fn->data.member.member) {
                        fn_name = call_fn->data.member.member;
                    }
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "'%s' returns %d values but only %d variable(s) provided; "
                        "all return values must be handled (use '_' to discard unwanted values)",
                        fn_name, sym->ret_count, var_count);
                    diag_error_msg(tc->diag, "E3006", msg,
                        NODE_FILE(tc, first), first->token.line, first->token.column, 0);
                }
            }
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
                "executable statements are not allowed at file scope; put this inside a function",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
    }

    switch (node->kind) {
    case NODE_VAR_DECL: {
        /* E5013: file-scope initializers cannot contain function calls.
         * A runtime call as an initializer would either need a module-init
         * function (which Grayscale does not generate) or produce invalid C
         * (non-constant initializer / free-standing statement at file
         * scope). Catch it on the Grayscale side so the user sees a real
         * diagnostic instead of a clang error pointing at the generated C. */
        if (tc->func_depth == 0 && node->data.var_decl.value &&
            expr_contains_call(node->data.var_decl.value)) {
            diag_error_code(tc->diag, "E5013",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Track const integer values for constant folding in later
         * declarations (e.g. fixed-size array sizes).  Also detect overflow
         * in const arithmetic expressions: codegen emits runtime
         * overflow-check wrappers (gray_add_check etc.) which are not valid
         * as C file-scope initializers.  The typechecker must evaluate and
         * reject overflowing expressions before codegen runs.
         * E5039: constant expression overflows the declared integer type. */
        if (!node->data.var_decl.mutable &&
            node->data.var_decl.type_name && node->data.var_decl.value) {
            const char *tn = node->data.var_decl.type_name;
            /* Track integer types (signed and unsigned) in the const table.
             * Unsigned values that fit in int64_t are stored as-is; this
             * covers practical array-size use cases.  Full uint64 overflow
             * detection is left to a separate check; for now we just ensure
             * the codegen fix applies (in_const_decl suppresses the runtime
             * wrapper). */
            bool is_int_type =
                strcmp(tn, "int") == 0 || strcmp(tn, "i64") == 0 ||
                strcmp(tn, "i8") == 0 || strcmp(tn, "i16") == 0 || strcmp(tn, "i32") == 0 ||
                strcmp(tn, "i128") == 0 || strcmp(tn, "i256") == 0 ||
                strcmp(tn, "uint") == 0 || strcmp(tn, "u64") == 0 ||
                strcmp(tn, "u8") == 0 || strcmp(tn, "u16") == 0 || strcmp(tn, "u32") == 0 ||
                strcmp(tn, "u128") == 0 || strcmp(tn, "u256") == 0;
            if (is_int_type) {
                int64_t folded = 0;
                bool overflowed = false;
                bool ok = tc_fold_const_int(tc, node->data.var_decl.value, &folded, &overflowed);
                if (ok) {
                    /* Expression is a valid compile-time constant.  Register
                     * the value so later const declarations can reference it. */
                    tc_register_const_int(tc, node->data.var_decl.name, folded);
                } else if (overflowed) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "constant expression overflows type '%s'", tn);
                    diag_error_msg(tc->diag, "E5039",
                        msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }
        /* E3038: void cannot be used as variable type */
        if (node->data.var_decl.type_name && strcmp(node->data.var_decl.type_name, "void") == 0) {
            diag_error_msg(tc->diag, "E3038",
                "'void' cannot be used as a variable type",
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
                    "'void' cannot be used as an element type in arrays or maps",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* E3101: func reference variables must use 'const', not 'mut' */
        if (node->data.var_decl.mutable) {
            bool is_func_ref_value = node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_FUNC_REF;
            bool is_func_type = node->data.var_decl.type_name &&
                strncmp(node->data.var_decl.type_name, "func(", 5) == 0;
            if (is_func_ref_value || is_func_type) {
                diag_error_msg(tc->diag, "E3101",
                    "func reference variables must be declared with 'const', not 'mut'; func references are compile-time aliases",
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
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a reserved type name and cannot be used as a variable name",
                VAR_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E2038", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E5016: builtin function name as variable name */
        if (node->data.var_decl.name[0] != '_' &&
            is_reserved_builtin_func_name(node->data.var_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a builtin function and cannot be used as a variable name",
                VAR_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E5016", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E5035: stdlib module name as variable name */
        if (node->data.var_decl.name[0] != '_' &&
            is_stdlib_module_name(node->data.var_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a standard library module and cannot be used as a variable name",
                VAR_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E5035", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E3045: or_return on non-error-returning function */
        if (strncmp(node->data.var_decl.name, "_gray_or", 8) == 0 &&
            node->data.var_decl.value && node->data.var_decl.value->kind == NODE_CALL_EXPR) {
            AstNode *call_fn = node->data.var_decl.value->data.call.function;
            const char *call_name = NULL;
            if (call_fn->kind == NODE_LABEL) call_name = call_fn->data.label.value;
            else if (call_fn->kind == NODE_MEMBER_EXPR && call_fn->data.member.object->kind == NODE_LABEL) {
                /* module.func() or Type.func(); construct prefixed name */
                static char prefixed[MSG_BUF_SIZE];
                snprintf(prefixed, sizeof(prefixed), "%s_%s",
                    call_fn->data.member.object->data.label.value, call_fn->data.member.member);
                call_name = prefixed;
            }
            if (call_name) {
                FuncSig *sig = find_func(tc, call_name);
                if (sig && (sig->return_count < 2 ||
                    sig->return_types[sig->return_count - 1]->kind != TK_ERROR)) {
                    char display[MSG_BUF_SIZE];
                    if (call_fn->kind == NODE_MEMBER_EXPR)
                        snprintf(display, sizeof(display), "%s.%s",
                            call_fn->data.member.object->data.label.value,
                            call_fn->data.member.member);
                    else {
                        strncpy(display, call_name, sizeof(display) - 1);
                        display[sizeof(display) - 1] = '\0';
                    }
                    diag_error_codef(tc->diag, "E3045", NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0, display);
                }
            }
        }
        /* E3059: maps cannot be declared const */
        if (!node->data.var_decl.mutable && node->data.var_decl.type_name &&
            strncmp(node->data.var_decl.type_name, "map[", 4) == 0) {
            diag_error_code_help(tc->diag, "E3059", NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "change 'const' to 'mut'; use a struct for fixed key-value data");
        }
        /* const must have a value */
        if (!node->data.var_decl.mutable && !node->data.var_decl.value) {
            diag_error_codef(tc->diag, "E2011", NODE_FILE(tc, node), node->token.line, node->token.column, 0, VAR_DISPLAY_NAME(node));
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
            strncmp(node->data.var_decl.name, "_gray_tmp", 9) != 0 &&
            strncmp(node->data.var_decl.name, "_gray_or", 8) != 0) {
            if (node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                diag_error_code_help(tc->diag, "E3050",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "add a type annotation, e.g. mut x [int] = {1, 2, 3}");
            } else if (node->data.var_decl.value->kind == NODE_MAP_VALUE) {
                diag_error_code_help(tc->diag, "E3051",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "add a type annotation, e.g. mut x [string:int] = {\"a\": 1}");
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
            /* Resolve const identifier sizes (e.g. "[int,SIZE]" → "[int,5]")
             * before the mut/const checks so downstream code always sees
             * numeric type strings. */
            if (has_size) {
                tc_resolve_array_size(tc, node);
                /* Re-read type_name — tc_resolve_array_size may have rewritten it. */
                tn = node->data.var_decl.type_name;
            }
            if (node->data.var_decl.mutable && has_size) {
                char *msg = tc_fmt(tc, "mutable array '%s' cannot have a fixed size '%.*s'",
                    VAR_DISPLAY_NAME(node), (int)(size_comma - tn), tn);
                diag_error_help(tc->diag, "E3054", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "use 'const' for fixed-size arrays, or remove the size for a dynamic 'mut' array");
            } else if (!node->data.var_decl.mutable && !has_size) {
                char *msg = tc_fmt(tc, "const array '%s' of type [%.*s] must have a fixed size",
                    VAR_DISPLAY_NAME(node), (int)(strlen(tn) - 2), tn + 1);
                diag_error_help(tc->diag, "E3055", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "add a size: const name [type, N] = {...}, or use 'mut' for a dynamic array");
            }
        }

        /* E2002: private not allowed inside functions */
        if (node->data.var_decl.is_private && tc->func_depth > 0) {
            diag_error_msg(tc->diag, "E2002",
                "'private' cannot be used inside a function; it only applies to top-level declarations",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        GrayType *declared = node->data.var_decl.type_name
            ? tc_type_from_name(tc, node->data.var_decl.type_name)
            : &TYPE_UNKNOWN;
        /* E4016: explicitly annotated type name that doesn't exist */
        if (node->data.var_decl.type_name && declared->kind == TK_UNKNOWN &&
            node->data.var_decl.type_name[0] >= 'A' && node->data.var_decl.type_name[0] <= 'Z') {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "undefined type '%s'; check the spelling or import the module that defines it",
                node->data.var_decl.type_name);
            diag_error_msg(tc->diag, "E4016", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        tc_mark_type_module_used(tc, node->data.var_decl.type_name);

        /* E3057: reject composite types as map keys before downstream checks
         * produce misleading cascades (e.g. struct-literal-in-index-position
         * tripping "no field 'y'"). Enums are allowed; they're int-backed
         * and hash fine. */
        if (declared->kind == TK_MAP && declared->key_type) {
            const char *kt = declared->key_type;
            GrayType *key_resolved = type_from_name(kt);
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
            /* Set expected_type for implicit enum resolution (.VARIANT) */
            GrayType *saved_expected = tc->expected_type;
            if (declared->kind == TK_ENUM && declared->name)
                tc->expected_type = declared;
            else if (declared->kind == TK_ARRAY && declared->element_type) {
                GrayType *elem_t = tc_type_from_name(tc, declared->element_type);
                if (elem_t && elem_t->kind == TK_ENUM)
                    tc->expected_type = declared;
            }
            GrayType *value_type = resolve_expr(tc, node->data.var_decl.value);
            tc->expected_type = saved_expected;
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
                    "cannot assign the result of a void function to a variable",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3102: cannot assign a func-type return value to a variable.
             * Func references must be created with ()func_name or ref(func_name). */
            if (value_type->kind == TK_FUNCTION &&
                node->data.var_decl.value->kind == NODE_CALL_EXPR) {
                AstNode *call_fn = node->data.var_decl.value->data.call.function;
                const char *called = "function";
                char called_buf[MSG_BUF_SIZE];
                if (call_fn->kind == NODE_LABEL) {
                    called = call_fn->data.label.value;
                } else if (call_fn->kind == NODE_MEMBER_EXPR &&
                           call_fn->data.member.object->kind == NODE_LABEL) {
                    snprintf(called_buf, sizeof(called_buf), "%s.%s",
                        call_fn->data.member.object->data.label.value,
                        call_fn->data.member.member);
                    called = called_buf;
                }
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "function '%s' returns a func type; func references cannot be assigned from function return values. Use '()func_name' or 'ref(func_name)' to create a func reference",
                    called);
                diag_error_msg(tc->diag, "E3102", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Check for multi-return to single variable
             * (skip if this is part of a multi-var expansion; the value will be a .v0 access) */
            if (node->data.var_decl.value->kind == NODE_CALL_EXPR &&
                strncmp(node->data.var_decl.name, "_gray_tmp", 9) != 0 &&
                strncmp(node->data.var_decl.name, "_gray_or", 8) != 0) {
                AstNode *call_fn = node->data.var_decl.value->data.call.function;
                const char *call_name = NULL;
                const char *call_mod = NULL;
                FuncSig *sig = NULL;
                if (call_fn->kind == NODE_LABEL) {
                    call_name = call_fn->data.label.value;
                    sig = find_func(tc, call_name);
                } else if (call_fn->kind == NODE_MEMBER_EXPR &&
                           call_fn->data.member.object->kind == NODE_LABEL) {
                    const char *mod_raw = call_fn->data.member.object->data.label.value;
                    call_mod = tc_resolve_alias(tc, mod_raw);
                    const char *mfn = call_fn->data.member.member;
                    char prefixed[MSG_BUF_SIZE];
                    snprintf(prefixed, sizeof(prefixed), "%s_%s", call_mod, mfn);
                    sig = find_func(tc, prefixed);
                    call_name = mfn;
                }
                if (sig && sig->return_count > 1) {
                    diag_error_codef(tc->diag, "E3040", NODE_FILE(tc, node), node->token.line, node->token.column, 0, call_name, sig->return_count, call_name);
                } else if (call_name && !sig) {
                    bool is_fallible = tc_is_fallible_stdlib(call_mod, call_name);
                    if (is_fallible) {
                        diag_error_codef(tc->diag, "E3089", NODE_FILE(tc, node),
                            node->token.line, node->token.column, 0,
                            call_name, call_name, call_name);
                    } else if (call_mod && strcmp(call_mod, "channels") == 0 &&
                               strcmp(call_name, "try_receive") == 0) {
                        diag_error_codef(tc->diag, "E3040", NODE_FILE(tc, node),
                            node->token.line, node->token.column, 0,
                            call_name, 2, call_name);
                    }
                }
            }
            /* Reject nil on non-nullable types */
            if (value_type->kind == TK_NIL && declared->kind != TK_UNKNOWN &&
                declared->kind != TK_ERROR && declared->kind != TK_POINTER &&
                declared->kind != TK_NIL) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot assign nil to '%s'; only Error and pointer types are nullable",
                    type_name(declared));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Reject bare 'mut x = nil' with no type context */
            if (value_type->kind == TK_NIL && declared->kind == TK_UNKNOWN) {
                diag_error_msg(tc->diag, "E3001",
                    "cannot infer type from nil; add a type annotation (e.g., mut x Error = nil)",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3066: typed-func variable assigned a function reference with a
             * different signature. Both sides are TK_FUNCTION; the canonical
             * encoded names (e.g. "func(int)->int") must match exactly. */
            if (declared->kind == TK_FUNCTION && value_type->kind == TK_FUNCTION &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot assign %s to variable of type %s",
                    type_display_name(tc, value_type), type_display_name(tc, declared));
                diag_error_msg(tc->diag, "E3066", msg,
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
                    char *msg = NULL;
                    /* If the initializer is a direct call, point the user
                     * at the reference form of the same name; that's
                     * overwhelmingly what they meant. */
                    if (v->kind == NODE_CALL_EXPR &&
                        v->data.call.function &&
                        v->data.call.function->kind == NODE_LABEL) {
                        const char *called = v->data.call.function->data.label.value;
                        msg = tc_fmt(tc,
                            "cannot assign %s to 'func'; to store a reference to '%s', use '()%s' (not '%s()')",
                            type_name(value_type), called, called, called);
                    } else {
                        msg = tc_fmt(tc,
                            "cannot assign %s to 'func'; func variables hold function references (e.g. '()name')",
                            type_name(value_type));
                    }
                    diag_error_msg(tc->diag, "E3001", msg,
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
                       /* Skip mismatch between compatible int kinds (handled by E3019),
                        * but byte and u8/uint are distinct semantic types and must not
                        * be implicitly assigned to each other. */
                       !(is_int_kind(declared->kind) && is_int_kind(value_type->kind) &&
                         !((declared->kind == TK_BYTE && value_type->kind == TK_UINT) ||
                           (declared->kind == TK_UINT && value_type->kind == TK_BYTE))) &&
                       /* Allow enum → int (enums are int-backed) but not
                        * the reverse; int literals / variables can't be
                        * assigned to enum variables (). */
                       !(is_int_kind(declared->kind) && value_type->kind == TK_ENUM) &&
                       /* Allow string enum → string (string enums hold GrayString values) */
                       !(declared->kind == TK_STRING && value_type->kind == TK_ENUM &&
                         tc_enum_is_string(tc, value_type->name)) &&
                       /* Note: multi-var expansion (.v0/.v1 access) is intentionally
                        * NOT skipped here — type mismatch on reversed multi-return
                        * types must be caught. */
                       /* Skip mismatch when assigning ref var to ^T pointer */
                       !(declared->kind == TK_POINTER && node->data.var_decl.value &&
                         node->data.var_decl.value->kind == NODE_LABEL &&
                         scope_lookup(tc->current_scope, node->data.var_decl.value->data.label.value) &&
                         scope_lookup(tc->current_scope, node->data.var_decl.value->data.label.value)->is_ref) &&
                       /* Skip mismatch when assigning pointer (addr) to ^T */
                       !(declared->kind == TK_POINTER && value_type->kind == TK_POINTER) &&
                       /* : implicit int→float coercion when target is float */
                       !(declared->kind == TK_FLOAT && is_int_kind(value_type->kind))) {
                /* Type mismatch */
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot assign %s to %s",
                    type_display_name(tc, value_type), type_display_name(tc, declared));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Pointer-to-pointer: pointee types differ (e.g., ^int assigned from ^string).
             * The outer kind-mismatch guard above short-circuits when both sides are TK_POINTER,
             * so this separate check is required to catch it. Mirrors the call-site check. */
            if (declared && value_type &&
                declared->kind == TK_POINTER && value_type->kind == TK_POINTER &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot assign %s to %s",
                    type_display_name(tc, value_type), type_display_name(tc, declared));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Bigint narrowing: e.g. i128 → i64, u256 → int.  Both sides share
             * TK_INT/TK_UINT so the kind-equality guard above silently passes
             * them through.  Catch it here by comparing named ranks. */
            if (declared && value_type &&
                declared->name && value_type->name) {
                int dr = int_name_rank(declared->name);
                int vr = int_name_rank(value_type->name);
                if (dr > 0 && vr >= 5 && dr < vr) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "type mismatch: cannot implicitly narrow %s to %s; use %s() to convert explicitly",
                        value_type->name, declared->name, declared->name);
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
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
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot assign '%s' to '%s'",
                    type_display_name(tc, value_type), type_display_name(tc, declared));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Enum-to-enum name mismatch (both TK_ENUM but different enum types) */
            if (declared->kind == TK_ENUM && value_type->kind == TK_ENUM &&
                declared->name && value_type->name &&
                strcmp(declared->name, value_type->name) != 0) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot assign enum '%s' to enum '%s'",
                    type_display_name(tc, value_type), type_display_name(tc, declared));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Array element type mismatch (both TK_ARRAY but different element types) */
            if (declared->kind == TK_ARRAY && value_type->kind == TK_ARRAY &&
                declared->element_type && value_type->element_type &&
                !tc_same_array_elem(tc, declared->element_type, value_type->element_type)) {
                GrayType *decl_elem = type_from_name(declared->element_type);
                GrayType *val_elem  = type_from_name(value_type->element_type);
                /* Allow int-kind ↔ int-kind, int→float, and skip when either
                 * element type is opaque/unknown (e.g. generic stdlib returns) */
                /* Skip function-type arrays: signature strings differ by whitespace */
                bool elem_is_func = strncmp(declared->element_type, "func", 4) == 0;
                if (!elem_is_func && decl_elem && val_elem &&
                    decl_elem->kind != TK_UNKNOWN && val_elem->kind != TK_UNKNOWN &&
                    !(is_int_kind(decl_elem->kind) && is_int_kind(val_elem->kind)) &&
                    !(is_int_kind(decl_elem->kind) && val_elem->kind == TK_STRUCT) &&
                    !(decl_elem->kind == TK_FLOAT && is_int_kind(val_elem->kind))) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "type mismatch: cannot assign '%s' to '%s'",
                        type_display_name(tc, value_type), type_display_name(tc, declared));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* Map key/value type mismatch (both TK_MAP but different key or value types) */
            if (declared->kind == TK_MAP && value_type->kind == TK_MAP) {
                bool key_mismatch = declared->key_type && value_type->key_type &&
                    strcmp(declared->key_type, value_type->key_type) != 0;
                bool val_mismatch = declared->value_type && value_type->value_type &&
                    strcmp(declared->value_type, value_type->value_type) != 0;
                /* Suppress key/value mismatches caused by int-kind coercion or float coercion */
                if (key_mismatch) {
                    GrayType *dk = type_from_name(declared->key_type);
                    GrayType *vk = type_from_name(value_type->key_type);
                    if (dk && vk && ((is_int_kind(dk->kind) && is_int_kind(vk->kind)) ||
                                    (dk->kind == TK_FLOAT && vk->kind == TK_FLOAT) ||
                                    (dk->kind == TK_FLOAT && is_int_kind(vk->kind))))
                        key_mismatch = false;
                }
                if (val_mismatch) {
                    GrayType *dv = type_from_name(declared->value_type);
                    GrayType *vv = type_from_name(value_type->value_type);
                    bool val_is_literal = node->data.var_decl.value &&
                        node->data.var_decl.value->kind == NODE_MAP_VALUE;
                    if (dv && vv && ((is_int_kind(dv->kind) && is_int_kind(vv->kind)) ||
                                    /* int→float coercion only for map literals, not variables */
                                    (val_is_literal && dv->kind == TK_FLOAT && is_int_kind(vv->kind))))
                        val_mismatch = false;
                }
                if (key_mismatch || val_mismatch) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "type mismatch: cannot assign '%s' to '%s'",
                        type_display_name(tc, value_type), type_display_name(tc, declared));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
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
                        "integer literal overflows 64-bit integer; max value is 18446744073709551615",
                        NODE_FILE(tc, node->data.var_decl.value), node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0);
                } else if (!exceeds_u64 && !is_bigint && !is_u64_like) {
                    diag_error_msg(tc->diag, "E3046",
                        "integer literal overflows 64-bit integer; max value is 9223372036854775807",
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
                 * from, and codegen then emits gray_array_new which the C
                 * compiler rejects. Point the user at the empty-map form
                 * `{:}` before the rest of the var_decl check runs. */
                const char *tn = node->data.var_decl.type_name;
                if (strncmp(tn, "map[", 4) == 0 &&
                    node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                    char *msg = NULL;
                    if (node->data.var_decl.value->data.array_value.count == 0) {
                        msg = tc_fmt(tc,
                            "cannot assign array literal '{}' to '%s'; use '{:}' for an empty map",
                            tn);
                    } else {
                        msg = tc_fmt(tc,
                            "cannot assign array literal to '%s'; map literals use '{key: value, ...}' syntax",
                            tn);
                    }
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node->data.var_decl.value),
                        node->data.var_decl.value->token.line,
                        node->data.var_decl.value->token.column, 0);
                }
                /* E3026/E3036: Check array literal elements fit in sized element type */
                if (tn[0] == '[' && node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                    /* Extract element type name from "[byte]", "[i8]", "[u8, 3]", etc. */
                    char elem_type[TYPE_NAME_MAX] = {0};
                    const char *start = tn + 1;
                    /* Find the matching ']' for the outermost array bracket,
                     * skipping nested brackets (e.g. map[K:V], [T]). */
                    const char *end = NULL;
                    {
                        int depth = 0;
                        for (const char *p = start; *p; p++) {
                            if (*p == '[') depth++;
                            else if (*p == ']') {
                                if (depth == 0) { end = p; break; }
                                depth--;
                            }
                        }
                    }
                    /* Find the top-level comma (fixed-size separator), ignoring
                     * commas inside nested brackets. */
                    const char *comma = NULL;
                    {
                        int depth = 0;
                        for (const char *p = start; p < (end ? end : start + strlen(start)); p++) {
                            if (*p == '[') depth++;
                            else if (*p == ']') depth--;
                            else if (*p == ',' && depth == 0) { comma = p; break; }
                        }
                    }
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
                                    "integer literal overflows 64-bit integer; max value is 18446744073709551615",
                                    NODE_FILE(tc, el), el->token.line, el->token.column, 0);
                                continue;
                            }
                            /* Element exceeds INT64_MAX but fits UINT64_MAX —
                             * fine for u64/uint elements, error for narrower
                             * signed/unsigned and for int. */
                            if (el_overflowed) {
                                if (!elem_is_u64_like) {
                                    diag_error_msg(tc->diag, "E3046",
                                        "integer literal overflows 64-bit integer; max value is 9223372036854775807",
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
                        GrayType *expected_et = tc_type_from_name(tc, elem_type);
                        AstNode *arr = node->data.var_decl.value;
                        for (int ei = 0; ei < arr->data.array_value.count; ei++) {
                            GrayType *actual_et = resolve_expr(tc, arr->data.array_value.elements[ei]);
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
                            /* E3053: cross-enum mismatch — both are TK_ENUM but
                             * from different enum types (e.g. Color vs Dir).
                             * The kind-level check above passes since both are
                             * TK_ENUM, so we need a name-level comparison. */
                            if (actual_et && expected_et &&
                                actual_et->kind == TK_ENUM && expected_et->kind == TK_ENUM &&
                                actual_et->name && expected_et->name &&
                                strcmp(actual_et->name, expected_et->name) != 0) {
                                diag_error_codef(tc->diag, "E3053", NODE_FILE(tc, arr->data.array_value.elements[ei]),
                                    arr->data.array_value.elements[ei]->token.line,
                                    arr->data.array_value.elements[ei]->token.column, 0, expected_et->name, actual_et->name);
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
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "fixed-size array [%s, %d] initialized with only %d of %d elements; remaining will be zero-valued",
                                elem_type[0] ? elem_type : "?", fixed_size,
                                arr->data.array_value.count, fixed_size);
                            diag_warning_msg(tc->diag, "W3003", msg,
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
                        char key_tn[TYPE_NAME_MAX] = {0};
                        char val_tn[TYPE_NAME_MAX] = {0};
                        size_t klen = (size_t)(mcolon - mstart);
                        size_t vlen = (size_t)(mend - mcolon - 1);
                        if (klen > 0 && klen < sizeof(key_tn) &&
                            vlen > 0 && vlen < sizeof(val_tn)) {
                            memcpy(key_tn, mstart, klen);
                            key_tn[klen] = '\0';
                            memcpy(val_tn, mcolon + 1, vlen);
                            val_tn[vlen] = '\0';
                            GrayType *expected_k = tc_type_from_name(tc, key_tn);
                            GrayType *expected_v = tc_type_from_name(tc, val_tn);
                            AstNode *mv = node->data.var_decl.value;
                            for (int mi = 0; mi < mv->data.map_value.count; mi++) {
                                AstNode *kn = mv->data.map_value.keys[mi];
                                AstNode *vn = mv->data.map_value.values[mi];
                                GrayType *kt = resolve_expr(tc, kn);
                                GrayType *vt = resolve_expr(tc, vn);
                                if (kt && kt->kind != TK_UNKNOWN && kt->kind != TK_VOID &&
                                    expected_k && expected_k->kind != TK_UNKNOWN &&
                                    kt->kind != expected_k->kind &&
                                    !(is_int_kind(expected_k->kind) && is_int_kind(kt->kind)) &&
                                    !(is_int_kind(expected_k->kind) && kt->kind == TK_ENUM) &&
                                    !(expected_k->kind == TK_ENUM && is_int_kind(kt->kind)) &&
                                    !(expected_k->kind == TK_FLOAT && is_int_kind(kt->kind))) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type mismatch in map literal key; expected '%s', got '%s'",
                                        type_display_name(tc, expected_k), type_display_name(tc, kt));
                                    diag_error_msg(tc->diag, "E3053", msg,
                                        NODE_FILE(tc, kn), kn->token.line, kn->token.column, 0);
                                }
                                /* Enum-to-enum: key types both TK_ENUM but different names */
                                if (expected_k && kt &&
                                    expected_k->kind == TK_ENUM && kt->kind == TK_ENUM &&
                                    expected_k->name && kt->name &&
                                    !tc_same_enum_type(tc, expected_k->name, kt->name)) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type mismatch in map literal key; expected enum '%s', got enum '%s'",
                                        type_display_name(tc, expected_k), type_display_name(tc, kt));
                                    diag_error_msg(tc->diag, "E3053", msg,
                                        NODE_FILE(tc, kn), kn->token.line, kn->token.column, 0);
                                }
                                if (vt && vt->kind != TK_UNKNOWN && vt->kind != TK_VOID &&
                                    expected_v && expected_v->kind != TK_UNKNOWN &&
                                    vt->kind != expected_v->kind &&
                                    !(is_int_kind(expected_v->kind) && is_int_kind(vt->kind)) &&
                                    !(is_int_kind(expected_v->kind) && vt->kind == TK_ENUM) &&
                                    !(expected_v->kind == TK_ENUM && is_int_kind(vt->kind)) &&
                                    !(expected_v->kind == TK_POINTER && vt->kind == TK_POINTER) &&
                                    !(expected_v->kind == TK_FLOAT && is_int_kind(vt->kind))) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type mismatch in map literal value; expected '%s', got '%s'",
                                        type_display_name(tc, expected_v), type_display_name(tc, vt));
                                    diag_error_msg(tc->diag, "E3053", msg,
                                        NODE_FILE(tc, vn), vn->token.line, vn->token.column, 0);
                                }
                                /* Enum-to-enum: value types both TK_ENUM but different names */
                                if (expected_v && vt &&
                                    expected_v->kind == TK_ENUM && vt->kind == TK_ENUM &&
                                    expected_v->name && vt->name &&
                                    !tc_same_enum_type(tc, expected_v->name, vt->name)) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type mismatch in map literal value; expected enum '%s', got enum '%s'",
                                        type_display_name(tc, expected_v), type_display_name(tc, vt));
                                    diag_error_msg(tc->diag, "E3053", msg,
                                        NODE_FILE(tc, vn), vn->token.line, vn->token.column, 0);
                                }
                                /* Pointer-to-pointer: pointee types differ in map literal value */
                                if (expected_v && vt &&
                                    expected_v->kind == TK_POINTER && vt->kind == TK_POINTER &&
                                    expected_v->name && vt->name &&
                                    strcmp(expected_v->name, vt->name) != 0) {
                                    char *msg = NULL;
                                    msg = tc_fmt(tc,
                                        "type mismatch in map literal value; expected '%s', got '%s'",
                                        type_display_name(tc, expected_v), type_display_name(tc, vt));
                                    diag_error_msg(tc->diag, "E3053", msg,
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
            if (existing && existing->def_line != 0) {
                /* def_line == 0 means this was pre-registered in Pass 1.5
                 * to allow forward references between global constants;
                 * that is not a duplicate declaration. */
                diag_error_codef(tc->diag, "E4003", NODE_FILE(tc, node), node->token.line, node->token.column, 0, VAR_DISPLAY_NAME(node), existing->def_line);
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
                    char *msg = NULL;
                    if (is_global) {
                        msg = tc_fmt(tc,
                            "variable '%s' shadows a global constant or variable declared on line %d",
                            VAR_DISPLAY_NAME(node), outer_sym->def_line);
                        diag_warning_msg(tc->diag, "W2007", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    } else {
                        msg = tc_fmt(tc,
                            "variable '%s' shadows a variable declared on line %d",
                            VAR_DISPLAY_NAME(node), outer_sym->def_line);
                        diag_warning_msg(tc->diag, "W2002", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
            }
            /* E4012: shadows a type — only when the variable name matches a
             * type usable by that same name. is_struct_name/is_enum_name key
             * on the flattened (module-prefixed) name, so a legitimate local
             * like `mod_Foo` collides with the internal name of `mod.Foo`.
             * Compare display names so that artifact never fires. */
            {
                const char *vname = node->data.var_decl.name;
                const char *vdisplay = VAR_DISPLAY_NAME(node);
                bool shadows_type =
                    (is_struct_name(tc, vname) &&
                     strcmp(struct_display_name(tc, vname), vdisplay) == 0) ||
                    (is_enum_name(tc, vname) &&
                     strcmp(enum_display_name(tc, vname), vdisplay) == 0);
                if (shadows_type) {
                    diag_error_codef(tc->diag, "E4012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, vdisplay);
                }
            }
            /* E4013: shadows a function — only when the variable name
             * matches a function callable by that same name. find_func
             * keys on the flattened (module-prefixed) name, so a legitimate
             * local like `mod_size` collides with the internal name of
             * `mod.size`. Compare display names so that artifact never fires. */
            {
                FuncSig *shadowed = find_func(tc, node->data.var_decl.name);
                if (shadowed && strcmp(func_display_name(shadowed),
                                       VAR_DISPLAY_NAME(node)) == 0) {
                    diag_error_codef(tc->diag, "E4013", NODE_FILE(tc, node), node->token.line, node->token.column, 0, VAR_DISPLAY_NAME(node));
                }
            }
            /* E4014: shadows an imported module */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], node->data.var_decl.name) == 0) {
                    diag_error_codef(tc->diag, "E4014", NODE_FILE(tc, node), node->token.line, node->token.column, 0, VAR_DISPLAY_NAME(node));
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
                                char *msg = NULL;
                                msg = tc_fmt(tc,
                                    "cannot take a mutable reference to const variable '%s'; declare '%s' as const, or copy() the value to get an independent mutable instance",
                                    src->data.label.value,
                                    node->data.var_decl.name);
                                diag_error_msg(tc->diag, "E3079", msg,
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
                            GrayType **slots = sig->return_types;
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
                                    GrayType *at = resolve_expr(tc, call->data.call.args[ai]);
                                    binding = bind_wildcard(ptn, at);
                                }
                                if (binding) {
                                    int rc = decl->data.func_decl.return_type_count;
                                    GrayType **subbed = xmalloc(sizeof(GrayType *) * (size_t)rc);
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
                        GrayType *primary = tc_get_fallible_stdlib_type(mod, mfn);
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym && primary) {
                            GrayType **rt = xmalloc(sizeof(GrayType *) * 2);
                            rt[0] = primary;
                            rt[1] = type_from_name("Error");
                            sym->ret_types = rt;
                            sym->ret_count = 2;
                        }
                    }
                    /* os.exec returns (int, string, string, bool) — synthesize 4-type slots */
                    if (strcmp(mod, "os") == 0 && strcmp(mfn, "exec") == 0) {
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym) {
                            GrayType **rt = xmalloc(sizeof(GrayType *) * 4);
                            rt[0] = &TYPE_INT;
                            rt[1] = &TYPE_STRING;
                            rt[2] = &TYPE_STRING;
                            rt[3] = &TYPE_BOOL;
                            sym->ret_types = rt;
                            sym->ret_count = 4;
                        }
                    }
                    /* channels.try_receive returns (int, bool) */
                    if (strcmp(mod, "channels") == 0 && strcmp(mfn, "try_receive") == 0) {
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym) {
                            GrayType **rt = xmalloc(sizeof(GrayType *) * 2);
                            rt[0] = &TYPE_INT;
                            rt[1] = &TYPE_BOOL;
                            sym->ret_types = rt;
                            sym->ret_count = 2;
                        }
                    }
                }
                /* User-defined module calls (mod.func); look up the prefixed
                 * function signature and propagate multi-return types so
                 * destructuring like `mut a, b = mod.func()` works. */
                if (fn->kind == NODE_MEMBER_EXPR &&
                    fn->data.member.object->kind == NODE_LABEL) {
                    const char *mod_raw = fn->data.member.object->data.label.value;
                    const char *mod = tc_resolve_alias(tc, mod_raw);
                    const char *mfn = fn->data.member.member;
                    char prefixed[MSG_BUF_SIZE];
                    snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                    FuncSig *sig = find_func(tc, prefixed);
                    if (sig && sig->return_count > 1) {
                        Symbol *sym = scope_lookup_local(tc->current_scope,
                            node->data.var_decl.name);
                        if (sym) {
                            sym->ret_types = sig->return_types;
                            sym->ret_count = sig->return_count;
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
                    char buf[MSG_BUF_SIZE];
                    snprintf(buf, sizeof(buf), "%s_%s",
                        fref->data.member.object->data.label.value,
                        fref->data.member.member);
                    rname = arena_strdup(tc->arena, buf);
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
                    sym->func_array_refs = xcalloc((size_t)n, sizeof(const char *));
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
                            char *pref = xmalloc(plen);
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
        GrayType *target_t = resolve_expr(tc, node->data.assign.target);
        /* Set expected_type for implicit enum resolution (.VARIANT) */
        GrayType *saved_expected = tc->expected_type;
        if (target_t && target_t->kind == TK_ENUM && target_t->name)
            tc->expected_type = target_t;
        GrayType *value_t = resolve_expr(tc, node->data.assign.value);
        tc->expected_type = saved_expected;

        /* E3078: compound arithmetic assigns on pointer variables
         * (p += 1, p -= 2, etc.) are pointer arithmetic and not
         * supported. Plain `=` (and the existing `p = nil` /
         * `p = addr(...)` paths) stay valid. */
        TokenType aop = node->data.assign.op;
        if ((aop == TOK_PLUS_ASSIGN || aop == TOK_MINUS_ASSIGN ||
             aop == TOK_ASTERISK_ASSIGN || aop == TOK_SLASH_ASSIGN ||
             aop == TOK_PERCENT_ASSIGN) &&
            target_t && target_t->kind == TK_POINTER) {
            diag_error_code(tc->diag, "E3078",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }

        /* E6008: reject assignment to stdlib module constants (math.PI = x, etc.) */
        AstNode *target = node->data.assign.target;
        if (target->kind == NODE_MEMBER_EXPR &&
            target->data.member.object->kind == NODE_LABEL) {
            const char *obj = target->data.member.object->data.label.value;
            bool is_module = false;
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], obj) == 0) { is_module = true; break; }
            }
            if (is_module) {
                diag_error_codef(tc->diag, "E6008",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    obj, target->data.member.member);
            }
        }

        /* E5025: lvalue validation; reject assignment to non-lvalue targets */
        if (target->kind != NODE_LABEL &&
            target->kind != NODE_MEMBER_EXPR &&
            target->kind != NODE_INDEX_EXPR &&
            target->kind != NODE_PREFIX_EXPR &&
            target->kind != NODE_POSTFIX_EXPR) {
            diag_error_msg(tc->diag, "E5025",
                "cannot assign to this expression; left side of '=' must be a variable, field, or index",
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
            /* p.field on a pointer parameter auto-derefs to p^.field — the
             * pointer itself is not being modified, so don't flag it as const. */
            if (sym && !sym->mutable && !(sym->type && sym->type->kind == TK_POINTER))
                const_name = target->data.member.object->data.label.value;
        }
        if (const_name) {
            diag_error_codef(tc->diag, "E3005", NODE_FILE(tc, node), node->token.line, node->token.column, 0, const_name);
        }
        /* E3004: string index assignment is not supported; strings are immutable
         * sequences — individual characters cannot be modified by index.
         * This fires regardless of the assigned value's type. */
        if (target->kind == NODE_INDEX_EXPR) {
            GrayType *indexed_t = resolve_expr(tc, target->data.index_expr.left);
            if (indexed_t && indexed_t->kind == TK_STRING) {
                diag_error_code(tc->diag, "E3004",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }

        /* E3094: array index assignment type mismatch (arr[i] = wrong_type) */
        if (target->kind == NODE_INDEX_EXPR && node->data.assign.value) {
            GrayType *indexed_t = resolve_expr(tc, target->data.index_expr.left);
            if (indexed_t && indexed_t->kind == TK_ARRAY && indexed_t->element_type) {
                GrayType *elem_t = type_from_name(indexed_t->element_type);
                GrayType *val_t = resolve_expr(tc, node->data.assign.value);
                if (val_t && val_t->kind != TK_UNKNOWN && elem_t && elem_t->kind != TK_UNKNOWN &&
                    val_t->kind != elem_t->kind &&
                    !(is_int_kind(val_t->kind) && is_int_kind(elem_t->kind)) &&
                    !(val_t->kind == TK_ENUM   && is_int_kind(elem_t->kind)) &&
                    !(is_int_kind(val_t->kind) && elem_t->kind == TK_FLOAT)  &&
                    !(val_t->kind == TK_FLOAT  && is_int_kind(elem_t->kind)) &&
                    /* enum array: type_from_name returns TK_STRUCT for enum names */
                    !(val_t->kind == TK_ENUM && elem_t->kind == TK_STRUCT &&
                      indexed_t->element_type && is_enum_name(tc, indexed_t->element_type))) {
                    diag_error_codef(tc->diag, "E3094",
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        type_display_name(tc, val_t), type_display_name(tc, indexed_t));
                }
            }
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
                GrayType *field_t = struct_field_type(tc, sym->type->name, target->data.member.member);
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
                GrayType *field_t = struct_field_type(tc, sym->type->element_type, target->data.member.member);
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
        /* Reject integer assigned to enum variable */
        if (target->kind == NODE_LABEL && target_t->kind == TK_ENUM &&
            is_int_kind(value_t->kind)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "cannot assign %s to enum '%s'; use an enum variant like '%s.VARIANT'",
                type_name(value_t), type_display_name(tc, target_t), type_display_name(tc, target_t));
            diag_error_msg(tc->diag, "E3118", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Check type mismatch on assignment (only for direct variable targets) */
        if (target->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.label.value);
            if (sym && sym->type->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                target_t->kind != TK_UNKNOWN &&
                target_t->kind != value_t->kind &&
                !(is_int_kind(target_t->kind) && is_int_kind(value_t->kind) &&
                  !((target_t->kind == TK_BYTE && value_t->kind == TK_UINT) ||
                    (target_t->kind == TK_UINT && value_t->kind == TK_BYTE))) &&
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
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot assign %s to %s variable '%s'",
                    type_display_name(tc, value_t), type_display_name(tc, target_t), target->data.label.value);
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* Struct-to-struct name mismatch on direct variable assignment */
        if (target->kind == NODE_LABEL &&
            target_t->kind == TK_STRUCT && value_t->kind == TK_STRUCT &&
            target_t->name && value_t->name &&
            strcmp(target_t->name, value_t->name) != 0) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "type mismatch: cannot assign '%s' to '%s' variable '%s'",
                type_display_name(tc, value_t), type_display_name(tc, target_t),
                target->data.label.value);
            diag_error_msg(tc->diag, "E3001", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E3098: struct-to-struct name mismatch through pointer dereference: v3^ = v2^
         * The NODE_LABEL check above is bypassed when the target is a postfix
         * dereference. resolve_expr already strips the pointer layer, so
         * target_t and value_t are both TK_STRUCT — just compare names. */
        if (target->kind == NODE_POSTFIX_EXPR &&
            target->data.postfix.op == TOK_CARET &&
            target_t && value_t &&
            target_t->kind == TK_STRUCT && value_t->kind == TK_STRUCT &&
            target_t->name && value_t->name &&
            strcmp(target_t->name, value_t->name) != 0) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "type mismatch: cannot assign '%s' to '%s' through pointer dereference",
                type_display_name(tc, value_t), type_display_name(tc, target_t));
            diag_error_msg(tc->diag, "E3098", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Pointer-to-pointer: pointee types differ on reassignment (e.g., p = q where ^int ≠ ^string).
         * The outer kind-equality guard short-circuits, so a dedicated check is required. */
        if (target->kind == NODE_LABEL &&
            target_t && value_t &&
            target_t->kind == TK_POINTER && value_t->kind == TK_POINTER &&
            target_t->name && value_t->name &&
            strcmp(target_t->name, value_t->name) != 0) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "type mismatch: cannot assign %s to %s variable '%s'",
                type_display_name(tc, value_t), type_display_name(tc, target_t), target->data.label.value);
            diag_error_msg(tc->diag, "E3001", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Bigint narrowing on reassignment: i128 → i64, u256 → int, etc. */
        if (target->kind == NODE_LABEL &&
            target_t && value_t &&
            target_t->name && value_t->name) {
            int dr = int_name_rank(target_t->name);
            int vr = int_name_rank(value_t->name);
            if (dr > 0 && vr >= 5 && dr < vr) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "type mismatch: cannot implicitly narrow %s to %s variable '%s'; use %s() to convert explicitly",
                    value_t->name, target_t->name, target->data.label.value, target_t->name);
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* Check type mismatch on struct field assignment.
         * sym->type may be TK_STRUCT (by-value) or TK_POINTER (from new()),
         * in both cases sym->type->name is the pointee/struct name. */
        if (target->kind == NODE_MEMBER_EXPR && target->data.member.object->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.member.object->data.label.value);
            if (sym && (sym->type->kind == TK_STRUCT || sym->type->kind == TK_POINTER)) {
                GrayType *field_t = struct_field_type(tc, sym->type->name, target->data.member.member);
                if (field_t->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                    /* kinds differ, OR both are pointers/structs to different types */
                    (field_t->kind != value_t->kind ||
                     (field_t->kind == TK_POINTER &&
                      field_t->name && value_t->name &&
                      strcmp(field_t->name, value_t->name) != 0) ||
                     (field_t->kind == TK_STRUCT && value_t->kind == TK_STRUCT &&
                      field_t->name && value_t->name &&
                      strcmp(field_t->name, value_t->name) != 0)) &&
                    !(is_int_kind(field_t->kind) && is_int_kind(value_t->kind)) &&
                    !(is_int_kind(field_t->kind) && value_t->kind == TK_ENUM) &&
                    !(field_t->kind == TK_FLOAT && is_int_kind(value_t->kind)) &&
                    /* nil is a valid value for pointer and Error fields */
                    !(value_t->kind == TK_NIL &&
                      (field_t->kind == TK_POINTER || field_t->kind == TK_ERROR))) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "type mismatch: cannot assign %s to %s field '%s'",
                        type_display_name(tc, value_t), type_display_name(tc, field_t), target->data.member.member);
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                /* E3066: func signature mismatch on struct field assignment */
                if (field_t->kind == TK_FUNCTION && value_t->kind == TK_FUNCTION &&
                    field_t->name && value_t->name &&
                    strcmp(field_t->name, value_t->name) != 0) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot assign %s to field '%s' of type %s",
                        type_display_name(tc, value_t), target->data.member.member,
                        type_display_name(tc, field_t));
                    diag_error_msg(tc->diag, "E3066", msg,
                        NODE_FILE(tc, node->data.assign.value),
                        node->data.assign.value->token.line,
                        node->data.assign.value->token.column, 0);
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
                diag_error_codef(tc->diag, "E3097",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    ptr_name, addr_var);
            }
        }
        break;
    }

    case NODE_RETURN_STMT:
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            /* Set expected_type for implicit enum resolution in return values */
            GrayType *saved_ret_expected = tc->expected_type;
            if (i < tc->current_return_count &&
                tc->current_return_types[i] &&
                tc->current_return_types[i]->kind == TK_ENUM &&
                tc->current_return_types[i]->name)
                tc->expected_type = tc->current_return_types[i];
            resolve_expr(tc, node->data.return_stmt.values[i]);
            tc->expected_type = saved_ret_expected;
        }
        /* main() exits when control reaches the closing brace; an
         * explicit `return` is not allowed. Without this check, codegen
         * emits `gray_scope_restore(_, _scope_mark)` referencing a
         * variable that main never declares, and the C compile fails. */
        if (tc->current_func_is_main) {
            diag_error_help(tc->diag, "E3073",
                arena_strdup(tc->arena, "'return' is not allowed in main(); main exits when control reaches the closing brace"),
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "use exit(code) to terminate with a status code");
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
                GrayType *expected = tc->current_return_types[0];
                if (expected && expected->kind != TK_POINTER &&
                    expected->kind != TK_ERROR && expected->kind != TK_UNKNOWN &&
                    expected->kind != TK_NIL && expected->kind != TK_VOID) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "cannot return 'nil' from a function that returns '%s'; nil is only valid for pointer and error types",
                        type_name(expected));
                    diag_error_msg(tc->diag, "E3072", msg,
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
                diag_error_msg(tc->diag, "E3006", "cannot return a value from a void function",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count == 0 &&
                   !tc->current_has_named_returns) {
            /* Bare return in non-void function (without named returns) */
            diag_error_msg(tc->diag, "E3006",
                "missing return value; function expects a return value",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count != tc->current_return_count) {
            /* E3013: wrong number of return values (skip or_return synthetic returns
             * which have count=1 but the function expects more; that's handled by codegen) */
            bool is_or_return_synthetic = false;
            if (node->data.return_stmt.count == 1 &&
                node->data.return_stmt.values[0]->kind == NODE_MEMBER_EXPR) {
                AstNode *obj = node->data.return_stmt.values[0]->data.member.object;
                if (obj->kind == NODE_LABEL && strncmp(obj->data.label.value, "_gray_or", 8) == 0) {
                    is_or_return_synthetic = true;
                }
            }
            if (!is_or_return_synthetic) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "function expects %d return value(s), got %d",
                    tc->current_return_count, node->data.return_stmt.count);
                diag_error_msg(tc->diag, "E3013", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count == tc->current_return_count) {
            /* Check first return value type (skip for or_return synthetic returns) */
            GrayType *ret_t = resolve_expr(tc, node->data.return_stmt.values[0]);
            GrayType *expected = tc->current_return_types[0];
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
                !(expected->kind == TK_FLOAT && is_int_kind(ret_t->kind)) &&
                /* Allow string enum → string return */
                !(expected->kind == TK_STRING && ret_t->kind == TK_ENUM &&
                  tc_enum_is_string(tc, ret_t->name))) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "return type mismatch: expected %s, got %s",
                    type_display_name(tc, expected), type_display_name(tc, ret_t));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Struct-to-struct return name mismatch */
            if (ret_t->kind == TK_STRUCT && expected->kind == TK_STRUCT &&
                ret_t->name && expected->name &&
                strcmp(ret_t->name, expected->name) != 0) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "return type mismatch: expected '%s', got '%s'",
                    type_display_name(tc, expected), type_display_name(tc, ret_t));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Enum-to-enum return name mismatch */
            if (ret_t->kind == TK_ENUM && expected->kind == TK_ENUM &&
                ret_t->name && expected->name &&
                !tc_same_enum_type(tc, ret_t->name, expected->name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "return type mismatch: expected enum '%s', got enum '%s'",
                    type_display_name(tc, expected), type_display_name(tc, ret_t));
                diag_error_msg(tc->diag, "E3001", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E3066: func signature mismatch in return */
            if (ret_t->kind == TK_FUNCTION && expected->kind == TK_FUNCTION &&
                ret_t->name && expected->name &&
                strcmp(ret_t->name, expected->name) != 0) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "cannot return %s from function declared to return %s",
                    type_display_name(tc, ret_t), type_display_name(tc, expected));
                diag_error_msg(tc->diag, "E3066", msg,
                    NODE_FILE(tc, node->data.return_stmt.values[0]),
                    node->data.return_stmt.values[0]->token.line,
                    node->data.return_stmt.values[0]->token.column, 0);
            }
            /* Array element type mismatch in return */
            if (ret_t->kind == TK_ARRAY && expected->kind == TK_ARRAY &&
                ret_t->element_type && expected->element_type &&
                !tc_same_array_elem(tc, ret_t->element_type, expected->element_type)) {
                GrayType *re = type_from_name(ret_t->element_type);
                GrayType *ee = type_from_name(expected->element_type);
                if (!(re && ee && is_int_kind(re->kind) && is_int_kind(ee->kind))) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "return type mismatch: expected '%s', got '%s'",
                        type_display_name(tc, expected), type_display_name(tc, ret_t));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* Map key/value type mismatch in return */
            if (ret_t->kind == TK_MAP && expected->kind == TK_MAP) {
                bool key_mismatch = ret_t->key_type && expected->key_type &&
                    strcmp(ret_t->key_type, expected->key_type) != 0;
                bool val_mismatch = ret_t->value_type && expected->value_type &&
                    strcmp(ret_t->value_type, expected->value_type) != 0;
                if (key_mismatch || val_mismatch) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "return type mismatch: expected '%s', got '%s'",
                        type_display_name(tc, expected), type_display_name(tc, ret_t));
                    diag_error_msg(tc->diag, "E3001", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* : pointer depth mismatch (e.g. returning ^^int
             * from a function declared -> ^int). Both sides are
             * TK_POINTER so the kind check above passes, but the
             * element_type strings differ ("int" vs "^int"). */
            if (ret_t->kind == TK_POINTER && expected->kind == TK_POINTER &&
                ret_t->element_type && expected->element_type &&
                strcmp(ret_t->element_type, expected->element_type) != 0) {
                /* Build human-readable pointer type strings (strip module prefix) */
                const char *exp_inner = struct_display_name(tc, expected->element_type);
                if (exp_inner == expected->element_type) exp_inner = enum_display_name(tc, expected->element_type);
                const char *got_inner = struct_display_name(tc, ret_t->element_type);
                if (got_inner == ret_t->element_type) got_inner = enum_display_name(tc, ret_t->element_type);
                char exp_str[TYPE_NAME_MAX], got_str[TYPE_NAME_MAX];
                snprintf(exp_str, sizeof(exp_str), "^%s", exp_inner);
                snprintf(got_str, sizeof(got_str), "^%s", got_inner);
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "return type mismatch: expected '%s', got '%s'",
                    exp_str, got_str);
                diag_error_msg(tc->diag, "E3001", msg,
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
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "function must return named variable '%s', not a different expression",
                            tc->current_return_names[i]);
                        diag_error_msg(tc->diag, "E3080", msg,
                            NODE_FILE(tc, rv), rv->token.line, rv->token.column, 0);
                    }
                }
            }
        }
        break;

    case NODE_EXPR_STMT: {
        GrayType *expr_t = resolve_expr(tc, node->data.expr_stmt.expr);
        /* E3081: bare function name used as statement without call */
        AstNode *expr = node->data.expr_stmt.expr;
        if (expr && expr->kind == NODE_LABEL) {
            const char *name = expr->data.label.value;
            if (tc_is_builtin(name) || find_func(tc, name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "function '%s' used as a statement without being called; did you mean '%s()'?",
                    name, name);
                diag_error_msg(tc->diag, "E3081", msg,
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
                        char full[MSG_BUF_SIZE];
                        const char *display_obj = struct_display_name(tc, obj_name);
                        snprintf(full, sizeof(full), "%s.%s()", display_obj, mem_name);
                        char *msg = tc_fmt(tc, "return value of '%s' is not used", full);
                        diag_error_help(tc->diag, "E5011", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                            "assign the result to a variable, or use 'mut _ = ...' to discard it");
                    }
                    is_side_effect = true; /* already handled */
                }
            }
            if (!is_side_effect && fn_name) {
                char full[MSG_BUF_SIZE];
                snprintf(full, sizeof(full), "%s()", fn_name);
                char *msg = tc_fmt(tc, "return value of '%s' is not used", full);
                diag_error_help(tc->diag, "E5011", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                    "assign the result to a variable, or use 'mut _ = ...' to discard it");
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
                        tc->destroyed_arenas = xrealloc(tc->destroyed_arenas,
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
                if (!using_module_accessible(tc, ui)) continue;
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
                        tc->destroyed_arenas = xrealloc(tc->destroyed_arenas,
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
        GrayType *cond_t = resolve_expr(tc, node->data.if_stmt.condition);
        /* E3038 (): void function call as condition. The same check
         * already exists for variable assignment and arithmetic; wire
         * it up for control-flow conditions too. The 'or' branch of an
         * if chain is parsed as a nested NODE_IF_STMT, so this one spot
         * covers 'if' and every subsequent 'or'. */
        if (cond_t && cond_t->kind == TK_VOID) {
            AstNode *c = node->data.if_stmt.condition;
            char *msg = NULL;
            if (c && c->kind == NODE_CALL_EXPR && c->data.call.function &&
                c->data.call.function->kind == NODE_LABEL) {
                msg = tc_fmt(tc,
                    "cannot use void function '%s' as condition; 'if' requires a bool expression",
                    c->data.call.function->data.label.value);
            } else {
                msg = tc_fmt(tc,
                    "cannot use void expression as condition; 'if' requires a bool expression");
            }
            diag_error_msg(tc->diag, "E3038", msg,
                NODE_FILE(tc, c), c->token.line, c->token.column, 0);
        }
        if (cond_t && cond_t->kind != TK_UNKNOWN &&
            (cond_t->kind == TK_STRING || cond_t->kind == TK_ARRAY ||
             cond_t->kind == TK_MAP   || cond_t->kind == TK_STRUCT)) {
            AstNode *c = node->data.if_stmt.condition;
            diag_error_codef(tc->diag, "E3091", NODE_FILE(tc, c), c->token.line, c->token.column, 0,
                type_display_name(tc, cond_t));
        }
        Scope *if_outer = tc->current_scope;
        Scope *if_body = scope_create(if_outer);
        tc->current_scope = if_body;
        check_block(tc, node->data.if_stmt.consequence);
        tc->current_scope = if_outer;
        scope_destroy(if_body);
        if (node->data.if_stmt.alternative) {
            Scope *else_body = scope_create(if_outer);
            tc->current_scope = else_body;
            check_statement(tc, node->data.if_stmt.alternative);
            tc->current_scope = if_outer;
            scope_destroy(else_body);
        }
        break;
    }

    case NODE_FOR_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;
        scope_define(loop_scope, node->data.for_stmt.var_name, &TYPE_INT, false);
        resolve_expr(tc, node->data.for_stmt.iterable);
        /* E9005: check range bounds when both bounds and step direction are compile-time known */
        if (node->data.for_stmt.iterable &&
            node->data.for_stmt.iterable->kind == NODE_RANGE_EXPR) {
            AstNode *r = node->data.for_stmt.iterable;
            if (r->data.range_expr.start && r->data.range_expr.end &&
                r->data.range_expr.start->kind == NODE_INT_VALUE &&
                r->data.range_expr.end->kind == NODE_INT_VALUE) {
                /* Skip bounds check when step is a runtime variable — direction is unknown. */
                bool has_neg_step = r->data.range_expr.step &&
                    r->data.range_expr.step->kind == NODE_INT_VALUE &&
                    r->data.range_expr.step->data.int_value.value < 0;
                bool has_neg_prefix = r->data.range_expr.step &&
                    r->data.range_expr.step->kind == NODE_PREFIX_EXPR &&
                    r->data.range_expr.step->data.prefix.op == TOK_MINUS;
                bool step_direction_known = !r->data.range_expr.step ||
                    (r->data.range_expr.step->kind == NODE_INT_VALUE) ||
                    has_neg_prefix;
                if (step_direction_known) {
                    int64_t start_val = r->data.range_expr.start->data.int_value.value;
                    int64_t end_val = r->data.range_expr.end->data.int_value.value;
                    bool negative_step = has_neg_step || has_neg_prefix;
                    bool invalid = negative_step ? (start_val < end_val) : (start_val > end_val);
                    if (invalid) {
                        char *msg = NULL;
                        if (negative_step) {
                            msg = tc_fmt(tc,
                                "invalid range: start (%lld) must be greater than or equal to end (%lld) for negative step",
                                (long long)start_val, (long long)end_val);
                        } else {
                            msg = tc_fmt(tc,
                                "invalid range: start (%lld) must be less than or equal to end (%lld)",
                                (long long)start_val, (long long)end_val);
                        }
                        diag_error_msg(tc->diag, "E9005", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                }
            }
        }
        tc->loop_depth++;
        check_block(tc, node->data.for_stmt.body);
        tc->loop_depth--;
        tc->current_scope = outer;
        scope_destroy(loop_scope);
        break;
    }

    case NODE_FOR_EACH_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;

        /* W2002: check if for_each iterator/index variables shadow outer variables */
        {
            const char *var = node->data.for_each.var_name;
            if (var && strcmp(var, "_") != 0) {
                Symbol *outer_sym = scope_lookup(outer, var);
                if (outer_sym) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "for_each variable '%s' shadows a variable declared on line %d",
                        var, outer_sym->def_line);
                    diag_warning_msg(tc->diag, "W2002", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            const char *idx = node->data.for_each.index_name;
            if (idx && strcmp(idx, "_") != 0) {
                Symbol *outer_sym = scope_lookup(outer, idx);
                if (outer_sym) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "for_each index variable '%s' shadows a variable declared on line %d",
                        idx, outer_sym->def_line);
                    diag_warning_msg(tc->diag, "W2002", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
        }

        /* E3123: both index and value discarded — no collection access occurs */
        {
            const char *var = node->data.for_each.var_name;
            const char *idx = node->data.for_each.index_name;
            if (idx && strcmp(idx, "_") == 0 && var && strcmp(var, "_") == 0) {
                diag_error_codef(tc->diag, "E3123", NODE_FILE(tc, node),
                    node->token.line, node->token.column, 0);
                tc->current_scope = outer;
                scope_destroy(loop_scope);
                break;
            }
        }

        /* Resolve collection type to determine element type */
        GrayType *coll_t = resolve_expr(tc, node->data.for_each.collection);

        /* Check that collection is iterable */
        if (coll_t->kind != TK_UNKNOWN && coll_t->kind != TK_ARRAY &&
            coll_t->kind != TK_MAP && coll_t->kind != TK_STRING) {
            diag_error_codef(tc->diag, "E3009", NODE_FILE(tc, node), node->token.line, node->token.column, 0, type_display_name(tc, coll_t));
        }

        if (coll_t->kind == TK_MAP) {
            /* Map iteration: for_each k, v in map OR for_each key in map */
            GrayType *key_t = coll_t->key_type ? type_from_name(coll_t->key_type) : &TYPE_STRING;
            GrayType *val_t = coll_t->value_type ? type_from_name(coll_t->value_type) : &TYPE_UNKNOWN;
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
            GrayType *elem_t = &TYPE_UNKNOWN;
            if (coll_t->kind == TK_ARRAY && coll_t->element_type) {
                elem_t = tc_type_from_name(tc, coll_t->element_type);
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
        scope_destroy(loop_scope);
        break;
    }

    case NODE_WHILE_STMT: {
        GrayType *wh_cond_t = resolve_expr(tc, node->data.while_stmt.condition);
        /* E3038 (): void function call as 'as_long_as' condition. */
        if (wh_cond_t && wh_cond_t->kind == TK_VOID) {
            AstNode *c = node->data.while_stmt.condition;
            char *msg = NULL;
            if (c && c->kind == NODE_CALL_EXPR && c->data.call.function &&
                c->data.call.function->kind == NODE_LABEL) {
                msg = tc_fmt(tc,
                    "cannot use void function '%s' as condition; 'as_long_as' requires a bool expression",
                    c->data.call.function->data.label.value);
            } else {
                msg = tc_fmt(tc,
                    "cannot use void expression as condition; 'as_long_as' requires a bool expression");
            }
            diag_error_msg(tc->diag, "E3038", msg,
                NODE_FILE(tc, c), c->token.line, c->token.column, 0);
        }
        if (wh_cond_t && wh_cond_t->kind != TK_UNKNOWN &&
            (wh_cond_t->kind == TK_STRING || wh_cond_t->kind == TK_ARRAY ||
             wh_cond_t->kind == TK_MAP   || wh_cond_t->kind == TK_STRUCT)) {
            AstNode *c = node->data.while_stmt.condition;
            diag_error_codef(tc->diag, "E3091", NODE_FILE(tc, c), c->token.line, c->token.column, 0,
                type_display_name(tc, wh_cond_t));
        }
        /* E3129: empty loop body hangs forever at runtime */
        if (node->data.while_stmt.body &&
            node->data.while_stmt.body->data.block.count == 0) {
            diag_error_code(tc->diag, "E3129",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        tc->loop_depth++;
        check_block(tc, node->data.while_stmt.body);
        tc->loop_depth--;
        break;
    }

    case NODE_LOOP_STMT:
        /* E3129: empty loop body hangs forever at runtime */
        if (node->data.loop_stmt.body &&
            node->data.loop_stmt.body->data.block.count == 0) {
            diag_error_code(tc->diag, "E3129",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
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
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a reserved type name and cannot be used as a function name",
                FUNC_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E2038", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E5016: builtin function name redeclared */
        if (is_reserved_builtin_func_name(node->data.func_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a builtin function and cannot be redeclared",
                FUNC_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E5016", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* E5035: stdlib module name as function name */
        if (is_stdlib_module_name(node->data.func_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "'%s' is a standard library module and cannot be used as a function name",
                FUNC_DISPLAY_NAME(node));
            diag_error_msg(tc->diag, "E5035", msg,
                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
        }
        /* Check for nested function declarations */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2051", NODE_FILE(tc, node), node->token.line, node->token.column, 0, FUNC_DISPLAY_NAME(node));
        }

        Scope *func_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = func_scope;
        tc->func_depth++;
        tc->destroyed_arena_count = 0;

        /* Define parameters in function scope, check for duplicates */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            Param *p = &node->data.func_decl.params[i];
            /* Type parameter (<?>) — not a variable; just record the name
             * so the body can recognise T in type positions. */
            if (p->is_type_param) {
                tc->type_param_name = p->name;
                continue;
            }
            /* E2038: reserved type name as parameter name */
            if (is_reserved_type_name(p->name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a reserved type name and cannot be used as a parameter name",
                    p->name);
                diag_error_msg(tc->diag, "E2038", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E5016: builtin function name as parameter name */
            if (is_reserved_builtin_func_name(p->name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a builtin function and cannot be used as a parameter name",
                    p->name);
                diag_error_msg(tc->diag, "E5016", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* E5035: stdlib module name as parameter name */
            if (is_stdlib_module_name(p->name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a standard library module and cannot be used as a parameter name",
                    p->name);
                diag_error_msg(tc->diag, "E5035", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Check for duplicate parameter name */
            for (int j = 0; j < i; j++) {
                if (strcmp(node->data.func_decl.params[j].name, p->name) == 0) {
                    diag_error_codef(tc->diag, "E2012", NODE_FILE(tc, node), node->token.line, node->token.column, 0, p->name);
                    break;
                }
            }
            /* W2008: parameter shadows an enum variant name */
            for (int ei = 0; ei < tc->enum_count; ei++) {
                bool found_variant = false;
                for (int vi = 0; vi < tc->enum_value_counts[ei]; vi++) {
                    if (strcmp(tc->enum_values[ei][vi], p->name) == 0) {
                        const char *display = tc->enum_display_names[ei]
                            ? tc->enum_display_names[ei] : tc->enum_names[ei];
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "parameter '%s' shadows enum variant '%s.%s'",
                            p->name, display, p->name);
                        diag_warning(tc->diag, "W2008", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                        found_variant = true;
                        break;
                    }
                }
                if (found_variant) break;
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
                    char *msg = tc_fmt(tc, "required parameter '%s' follows a parameter with a default value", p->name);
                    diag_error_help(tc->diag, "E2039", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        "move all parameters with default values to the end of the parameter list");
                }
            }
            /* E3119: fixed-size array in function parameter */
            if (p->type_name && p->type_name[0] == '[') {
                const char *tn = p->type_name;
                const char *size_comma = NULL;
                int depth = 0;
                for (const char *c = tn; *c; c++) {
                    if (*c == '(' || *c == '[') depth++;
                    else if (*c == ')' || *c == ']') depth--;
                    else if (*c == ',' && depth == 1) { size_comma = c; break; }
                }
                if (size_comma) {
                    char elem[MSG_BUF_SIZE];
                    int elem_len = (int)(size_comma - tn - 1);
                    snprintf(elem, sizeof(elem), "%.*s", elem_len, tn + 1);
                    char *msg = tc_fmt(tc, "fixed-size array type '%s' is not allowed in function parameter '%s'; use [%s] instead",
                        tn, p->name, elem);
                    diag_error_help(tc->diag, "E3119", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                        "use a dynamic array type instead, e.g. [int] without a size");
                }
            }
            GrayType *ptype = p->type_name ? tc_type_from_name(tc, p->type_name) : &TYPE_UNKNOWN;
            /* E4016: undefined parameter type */
            if (p->type_name && ptype->kind == TK_UNKNOWN &&
                p->type_name[0] >= 'A' && p->type_name[0] <= 'Z') {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "undefined type '%s'; check the spelling or import the module that defines it",
                    p->type_name);
                diag_error_msg(tc->diag, "E4016", msg,
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
            /* Type inference: if no explicit type annotation, infer from default
             * value when it is an enum member access (e.g. t = Color.RED). */
            if (!p->type_name && p->default_value) {
                GrayType *inferred = resolve_expr(tc, p->default_value);
                if (inferred && inferred->kind == TK_ENUM) {
                    ptype = inferred;
                } else {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "parameter '%s' has no type annotation; omitting the type is only allowed when the default value is an enum member (e.g. %s = MyEnum.VALUE)",
                        p->name, p->name);
                    diag_error_msg(tc->diag, "E2002", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            /* E3001: validate default value type matches parameter type */
            if (p->default_value && p->type_name) {
                /* Set expected_type for implicit enum resolution in default param values */
                GrayType *saved_def_expected = tc->expected_type;
                if (ptype->kind == TK_ENUM && ptype->name)
                    tc->expected_type = ptype;
                GrayType *def_t = resolve_expr(tc, p->default_value);
                tc->expected_type = saved_def_expected;
                if (def_t->kind != TK_UNKNOWN && ptype->kind != TK_UNKNOWN &&
                    def_t->kind != ptype->kind &&
                    !(is_int_kind(def_t->kind) && is_int_kind(ptype->kind)) &&
                    !(def_t->kind == TK_NIL)) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "default value for parameter '%s' has wrong type; expected %s, got %s",
                        p->name, p->type_name, type_name(def_t));
                    diag_error_msg(tc->diag, "E3001", msg,
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
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "duplicate named return value '%s'", rn);
                            diag_error_msg(tc->diag, "E2063", msg,
                                NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                            break;
                        }
                    }
                    /* E3082: wildcard type '?' in named return position */
                    if (i < node->data.func_decl.return_type_count &&
                        node->data.func_decl.return_types[i] &&
                        strcmp(node->data.func_decl.return_types[i], "?") == 0) {
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "wildcard type '?' cannot be used in named return value '%s'; use an unnamed return instead (e.g. -> (?, int))",
                            rn);
                        diag_error_msg(tc->diag, "E3082", msg,
                            NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                    }
                    /* E2063: named return collides with parameter */
                    for (int j = 0; j < node->data.func_decl.param_count; j++) {
                        if (strcmp(node->data.func_decl.params[j].name, rn) == 0) {
                            char *msg = NULL;
                            msg = tc_fmt(tc,
                                "named return value '%s' conflicts with parameter '%s'",
                                rn, rn);
                            diag_error_msg(tc->diag, "E2063", msg,
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
        GrayType **prev_ret = tc->current_return_types;
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
            tc->current_return_types = xmalloc(sizeof(GrayType *) * node->data.func_decl.return_type_count);
            tc->current_return_type_names = xmalloc(sizeof(const char *) * node->data.func_decl.return_type_count);
            tc->current_return_count = node->data.func_decl.return_type_count;
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                tc->current_return_types[i] = tc_type_from_name(tc, node->data.func_decl.return_types[i]);
                tc->current_return_type_names[i] = node->data.func_decl.return_types[i];
                /* E4016: undefined return type */
                const char *rtn = node->data.func_decl.return_types[i];
                if (rtn && tc->current_return_types[i]->kind == TK_UNKNOWN &&
                    rtn[0] >= 'A' && rtn[0] <= 'Z') {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "undefined type '%s'; check the spelling or import the module that defines it",
                        rtn);
                    diag_error_msg(tc->diag, "E4016", msg,
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
                diag_error_codef(tc->diag, "E3024", NODE_FILE(tc, node), node->token.line, node->token.column, 0, FUNC_DISPLAY_NAME(node));
            } else if (has_return && !has_named_returns &&
                       !all_paths_return(node->data.func_decl.body)) {
                diag_error_codef(tc->diag, "E3035", NODE_FILE(tc, node), node->token.line, node->token.column, 0, FUNC_DISPLAY_NAME(node));
            }
        }

        /* W2011: named return value declared but no matching variable in body */
        if (node->data.func_decl.return_names) {
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                const char *rn = node->data.func_decl.return_names[i];
                if (!rn) continue;
                Symbol *sym = scope_lookup_local(func_scope, rn);
                if (!sym) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "named return value '%s' is declared in the signature but no matching variable exists in the function body",
                        rn);
                    diag_warning_msg(tc->diag, "W2011", msg,
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
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "variable '%s' is declared but never used", s->name);
                    diag_warning_msg(tc->diag, "W1001", msg,
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
        tc->type_param_name = NULL;
        tc->type_param_binding = NULL;
        tc->func_depth--;
        tc->current_scope = outer;
        scope_destroy(func_scope);
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
                tc->using_modules = xrealloc(tc->using_modules,
                    sizeof(const char *) * (size_t)tc->using_module_cap);
                tc->using_module_files = xrealloc(tc->using_module_files,
                    sizeof(const char *) * (size_t)tc->using_module_cap);
                tc->using_module_import_indices = xrealloc(tc->using_module_import_indices,
                    sizeof(int) * (size_t)tc->using_module_cap);
            }
            tc->using_module_files[tc->using_module_count] = node->token.file;
            tc->using_module_import_indices[tc->using_module_count] =
                tc_find_import_index(tc, node->data.using_stmt.modules[j]);
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

    case NODE_STRUCT_DECL: {
        /* E3099: struct name collides with a stdlib opaque type reserved by codegen.
         * These names map to internal C types (GrayRouter, GrayThread, etc.) before the
         * user-struct path, so any user struct with these names silently generates
         * invalid C with no Grayscale diagnostic. */
        const char *sname = STRUCT_DISPLAY_NAME(node);
        if (is_reserved_stdlib_struct_name(sname)) {
            diag_error_codef(tc->diag, "E3099",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0, sname);
        }
        /* E2053: struct inside function */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2053",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "struct", STRUCT_DISPLAY_NAME(node));
        }
        /* E3103/E3104: #json structs are data-only */
        if (node->data.struct_decl.is_json) {
            for (int fi = 0; fi < node->data.struct_decl.field_count; fi++) {
                const char *ftype = node->data.struct_decl.fields[fi].type_name;
                if (ftype && strncmp(ftype, "func", 4) == 0) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "#json struct '%s' cannot have func-typed field '%s'; func references have no JSON representation",
                        STRUCT_DISPLAY_NAME(node),
                        node->data.struct_decl.fields[fi].name);
                    diag_error_msg(tc->diag, "E3103", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
                if (node->data.struct_decl.fields[fi].default_value) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "#json struct '%s' cannot have default field values; field '%s' has a default",
                        STRUCT_DISPLAY_NAME(node),
                        node->data.struct_decl.fields[fi].name);
                    diag_error_msg(tc->diag, "E3109", msg,
                        NODE_FILE(tc, node), node->token.line, node->token.column, 0);
                }
            }
            for (int fi = 0; fi < node->data.struct_decl.func_count; fi++) {
                AstNode *fn = node->data.struct_decl.funcs[fi].func_decl;
                if (fn && fn->kind == NODE_FUNC_DECL) {
                    const char *fname = FUNC_DISPLAY_NAME(fn);
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "#json struct '%s' cannot declare functions; #json structs are data-only — move '%s' to a standalone function",
                        STRUCT_DISPLAY_NAME(node), fname);
                    diag_error_msg(tc->diag, "E3104", msg,
                        NODE_FILE(tc, node), fn->token.line, fn->token.column, 0);
                }
            }
        }
        /* Type-check struct-namespaced function bodies */
        tc->current_struct_name = node->data.struct_decl.name;
        for (int i = 0; i < node->data.struct_decl.func_count; i++) {
            AstNode *fn = node->data.struct_decl.funcs[i].func_decl;
            if (fn && fn->kind == NODE_FUNC_DECL) {
                check_statement(tc, fn);
            }
        }
        tc->current_struct_name = NULL;
        break;
    }

    case NODE_ENUM_DECL:
        /* E2053: enum inside function */
        if (tc->func_depth > 0) {
            diag_error_codef(tc->diag, "E2053",
                NODE_FILE(tc, node), node->token.line, node->token.column, 0,
                "enum", ENUM_DISPLAY_NAME(node));
        }
        break;

    case NODE_WHEN_STMT: {
        GrayType *when_t = resolve_expr(tc, node->data.when_stmt.value);
        /* W2012: float subjects use bit-equality, which is rarely what the
         * user wants given 0.1 + 0.2 != 0.3. */
        if (when_t && when_t->kind == TK_FLOAT) {
            AstNode *subj = node->data.when_stmt.value;
            diag_warning_code(tc->diag, "W2012", NODE_FILE(tc, subj), subj->token.line, subj->token.column, 0);
        }
        /* E3121: struct, array, map, and pointer types are not valid when subjects.
         * Null out when_t so subsequent case type checks are skipped. */
        if (when_t && (when_t->kind == TK_STRUCT || when_t->kind == TK_ARRAY ||
                       when_t->kind == TK_MAP || when_t->kind == TK_POINTER)) {
            AstNode *subj = node->data.when_stmt.value;
            diag_error_codef(tc->diag, "E3121", NODE_FILE(tc, subj), subj->token.line, subj->token.column, 0,
                type_display_name(tc, when_t));
            when_t = NULL;
        }
        /* E2043: check for duplicate case values, E3001: check type match */
        /* Set expected_type for implicit enum resolution in when/is branches */
        GrayType *saved_when_expected = tc->expected_type;
        if (when_t && when_t->kind == TK_ENUM && when_t->name)
            tc->expected_type = when_t;
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            for (int j = 0; j < node->data.when_stmt.cases[i].value_count; j++) {
                AstNode *val_i = node->data.when_stmt.cases[i].values[j];
                /* Handle NODE_WHEN_PATTERN: validate variant + binding count */
                if (val_i->kind == NODE_WHEN_PATTERN) {
                    const char *vname = val_i->data.when_pattern.variant;
                    const char *ename = NULL;
                    if (val_i->data.when_pattern.is_implicit) {
                        if (when_t && when_t->kind == TK_ENUM && when_t->name)
                            ename = when_t->name;
                    } else {
                        /* For explicit form Variant(x), resolve from scrutinee type */
                        if (when_t && when_t->kind == TK_ENUM && when_t->name)
                            ename = when_t->name;
                    }
                    if (ename) {
                        val_i->data.when_pattern.enum_name = ename;
                        int eidx = -1;
                        for (int ei = 0; ei < tc->enum_count; ei++) {
                            if (strcmp(tc->enum_names[ei], ename) == 0) { eidx = ei; break; }
                        }
                        if (eidx >= 0) {
                            int vidx = -1;
                            for (int vi = 0; vi < tc->enum_value_counts[eidx]; vi++) {
                                if (strcmp(tc->enum_values[eidx][vi], vname) == 0) { vidx = vi; break; }
                            }
                            if (vidx < 0) {
                                diag_error_codef(tc->diag, "E3047", NODE_FILE(tc, val_i), val_i->token.line, val_i->token.column, 0, ename, vname);
                            } else {
                                int expected_bc = tc->enum_payload_counts[eidx][vidx];
                                int got_bc = val_i->data.when_pattern.binding_count;
                                if (expected_bc != got_bc) {
                                    diag_error_codef(tc->diag, "E3116", NODE_FILE(tc, val_i), val_i->token.line, val_i->token.column, 0, vname, expected_bc, got_bc);
                                }
                            }
                        }
                    }
                    continue;
                }
                GrayType *case_t = resolve_expr(tc, val_i);
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
                        (when_t->kind == TK_BYTE && is_int_kind(case_t->kind)) ||
                        /* string subject with string-enum arm, or vice versa */
                        (when_t->kind == TK_STRING && case_t->kind == TK_ENUM &&
                         tc_enum_is_string(tc, case_t->name)) ||
                        (when_t->kind == TK_ENUM && case_t->kind == TK_STRING &&
                         tc_enum_is_string(tc, when_t->name));
                    if (!compat) {
                        diag_error_codef(tc->diag, "E3018", NODE_FILE(tc, val_i), val_i->token.line, val_i->token.column, 0, type_display_name(tc, when_t), type_display_name(tc, case_t));
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
            Scope *case_body = scope_create(case_outer);
            tc->current_scope = case_body;
            /* Introduce pattern bindings into case scope */
            for (int j = 0; j < node->data.when_stmt.cases[i].value_count; j++) {
                AstNode *val_i = node->data.when_stmt.cases[i].values[j];
                if (val_i->kind == NODE_WHEN_PATTERN && val_i->data.when_pattern.enum_name) {
                    const char *ename = val_i->data.when_pattern.enum_name;
                    const char *vname = val_i->data.when_pattern.variant;
                    int eidx = -1;
                    for (int ei = 0; ei < tc->enum_count; ei++) {
                        if (strcmp(tc->enum_names[ei], ename) == 0) { eidx = ei; break; }
                    }
                    if (eidx >= 0) {
                        int vidx = -1;
                        for (int vi = 0; vi < tc->enum_value_counts[eidx]; vi++) {
                            if (strcmp(tc->enum_values[eidx][vi], vname) == 0) { vidx = vi; break; }
                        }
                        if (vidx >= 0) {
                            int bc = val_i->data.when_pattern.binding_count;
                            int pc = tc->enum_payload_counts[eidx][vidx];
                            int limit = bc < pc ? bc : pc;
                            for (int bi = 0; bi < limit; bi++) {
                                GrayType *bt = tc_type_from_name(tc, tc->enum_payload_types[eidx][vidx][bi]);
                                scope_define(tc->current_scope, val_i->data.when_pattern.bindings[bi], bt, false);
                            }
                        }
                    }
                }
            }
            check_block(tc, node->data.when_stmt.cases[i].body);
            tc->current_scope = case_outer;
            scope_destroy(case_body);
        }
        tc->expected_type = saved_when_expected;
        if (node->data.when_stmt.default_body) {
            Scope *def_outer = tc->current_scope;
            Scope *def_body = scope_create(def_outer);
            tc->current_scope = def_body;
            check_block(tc, node->data.when_stmt.default_body);
            tc->current_scope = def_outer;
            scope_destroy(def_body);
            /* W3006: empty default branch */
            if (node->data.when_stmt.default_body->data.block.count == 0) {
                diag_warning(tc->diag, "W3006",
                    "empty default branch in when statement; unmatched values are silently ignored",
                    NODE_FILE(tc, node->data.when_stmt.default_body),
                    node->data.when_stmt.default_body->token.line,
                    node->data.when_stmt.default_body->token.column, 0);
            }
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
                    /* Also infer from resolved implicit enum */
                    if (cv->kind == NODE_IMPLICIT_ENUM &&
                        cv->data.implicit_enum.resolved_enum) {
                        enum_name = cv->data.implicit_enum.resolved_enum;
                    }
                    /* Infer from when pattern */
                    if (cv->kind == NODE_WHEN_PATTERN &&
                        cv->data.when_pattern.enum_name) {
                        enum_name = cv->data.when_pattern.enum_name;
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
                                /* Match .VARIANT (implicit enum selector) */
                                if (cv->kind == NODE_IMPLICIT_ENUM &&
                                    strcmp(cv->data.implicit_enum.variant, variants[vi]) == 0) {
                                    covered = true;
                                }
                                /* Match when pattern (destructuring) */
                                if (cv->kind == NODE_WHEN_PATTERN &&
                                    strcmp(cv->data.when_pattern.variant, variants[vi]) == 0) {
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
                    "#strict when on a non-enum type requires a default branch to be exhaustive",
                    NODE_FILE(tc, node), node->token.line, node->token.column, 0);
            }
        }
        /* W3005: when matches on enum values but has no #strict and no default */
        if (!node->data.when_stmt.is_strict && !node->data.when_stmt.default_body) {
            bool has_enum_case = false;
            for (int ci = 0; ci < node->data.when_stmt.case_count && !has_enum_case; ci++) {
                for (int cj = 0; cj < node->data.when_stmt.cases[ci].value_count && !has_enum_case; cj++) {
                    AstNode *cv = node->data.when_stmt.cases[ci].values[cj];
                    if (cv->kind == NODE_MEMBER_EXPR &&
                        cv->data.member.object->kind == NODE_LABEL) {
                        const char *name = cv->data.member.object->data.label.value;
                        if (is_enum_name(tc, name)) {
                            has_enum_case = true;
                        } else {
                            for (int ui = 0; ui < tc->using_module_count && !has_enum_case; ui++) {
                                if (!using_module_accessible(tc, ui)) continue;
                                const char *real_mod = tc_resolve_alias(tc, tc->using_modules[ui]);
                                char prefixed[MSG_BUF_SIZE];
                                snprintf(prefixed, sizeof(prefixed), "%s_%s", real_mod, name);
                                if (is_enum_name(tc, prefixed)) has_enum_case = true;
                            }
                        }
                    }
                }
            }
            if (has_enum_case) {
                diag_warning(tc->diag, "W3005",
                    "when statement matches on enum values without #strict and no default; exhaustiveness is not checked",
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


/* : returns true if any NODE_STRUCT_DECL in the program has the given
 * name. Used by pointer-field pointee validation to accept forward
 * references and self-recursion before the struct is registered. */
static bool struct_name_declared(AstNode *program, const char *name) {
    if (!program || !name) return false;
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *s = program->data.program.stmts[i];
        if (s && s->kind == NODE_STRUCT_DECL && s->data.struct_decl.name &&
            strcmp(s->data.struct_decl.name, name) == 0)
            return true;
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

/* Recursively validate that every named type inside a field's type string
 * (including array element types, map key/value types, pointer pointees, and
 * arbitrarily nested combinations) refers to a known type. Emits E4016 for
 * every unknown leaf type found. */
static void validate_field_type_recursive(TypeChecker *tc, AstNode *program,
                                          const char *type_name,
                                          const char *field_name,
                                          AstNode *stmt) {
    if (!type_name || !*type_name) return;

    /* Pointer: ^T → recurse on T */
    if (type_name[0] == '^') {
        const char *pointee = type_name + 1;
        while (*pointee == '^') pointee++;
        validate_field_type_recursive(tc, program, pointee, field_name, stmt);
        return;
    }

    /* Array: [T] or [T,N] → recurse on T */
    if (type_name[0] == '[') {
        size_t len = strlen(type_name);
        if (len > 2 && type_name[len - 1] == ']') {
            char elem[MSG_BUF_SIZE];
            size_t elem_len = len - 2;
            memcpy(elem, type_name + 1, elem_len);
            elem[elem_len] = '\0';
            char *comma = strchr(elem, ',');
            if (comma) *comma = '\0';
            validate_field_type_recursive(tc, program, elem, field_name, stmt);
        }
        return;
    }

    /* Map: map[K:V] → recurse on K and V separately */
    if (strncmp(type_name, "map[", 4) == 0) {
        size_t len = strlen(type_name);
        if (len > 5 && type_name[len - 1] == ']') {
            /* Extract inner "K:V" string */
            char inner[MSG_BUF_SIZE];
            size_t inner_len = len - 5; /* skip "map[" and "]" */
            memcpy(inner, type_name + 4, inner_len);
            inner[inner_len] = '\0';
            /* Find colon at depth 0 (handles nested [T] keys) */
            int depth = 0;
            int colon_pos = -1;
            for (int i = 0; inner[i]; i++) {
                if (inner[i] == '[') depth++;
                else if (inner[i] == ']') depth--;
                else if (inner[i] == ':' && depth == 0) { colon_pos = i; break; }
            }
            if (colon_pos >= 0) {
                char key_t[MSG_BUF_SIZE];
                memcpy(key_t, inner, (size_t)colon_pos);
                key_t[colon_pos] = '\0';
                const char *val_t = inner + colon_pos + 1;
                validate_field_type_recursive(tc, program, key_t, field_name, stmt);
                validate_field_type_recursive(tc, program, val_t, field_name, stmt);
            }
        }
        return;
    }

    /* Leaf: must be a known primitive, enum, struct, or wildcard '?' */
    if (type_name_has_wildcard(type_name)) return;
    GrayType *t = tc_type_from_name(tc, type_name);
    if (t && t->kind != TK_STRUCT && t->kind != TK_UNKNOWN) return;
    if (is_enum_name(tc, type_name)) return;
    if (is_struct_name(tc, type_name)) return;
    if (struct_name_declared(program, type_name)) return;

    /* Stdlib opaque struct types are registered after user structs; accept
     * them here so struct fields can reference them without false E4016. */
    static const char *stdlib_struct_types[] = {
        "Thread", "Mutex", "SpinLock", "Channel", "Socket", "Listener",
        "Database", "Router", "HttpRequest", "HttpResponse", "UUID",
        "Arena", "SourceLocation", NULL
    };
    for (int si = 0; stdlib_struct_types[si]; si++) {
        if (strcmp(type_name, stdlib_struct_types[si]) == 0) return;
    }

    char *msg = NULL;
    msg = tc_fmt(tc,
        "field '%s' references undefined type '%s'",
        field_name, type_name);
    diag_error_msg(tc->diag, "E4016", msg,
        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
}

static void register_declarations(TypeChecker *tc, AstNode *program) {
    tc->registering = true;
    /* Validate and record imports */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_IMPORT_STMT) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->is_stdlib && item->module && !is_stdlib_module_name(item->module)) {
                    diag_error_codef(tc->diag, "E6001", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, item->module);
                }
                /* Record import for unused-import tracking.
                 * Store the ALIAS as the import name (so using/dot notation uses the alias). */
                if (item->module || item->alias) {
                    if (tc->import_count >= tc->import_cap) {
                        tc->import_cap = tc->import_cap ? tc->import_cap * 2 : 16;
                        tc->imported_modules = xrealloc(tc->imported_modules, sizeof(const char *) * tc->import_cap);
                        tc->import_files = xrealloc(tc->import_files, sizeof(const char *) * tc->import_cap);
                        tc->import_lines = xrealloc(tc->import_lines, sizeof(int) * tc->import_cap);
                        tc->import_used = xrealloc(tc->import_used, sizeof(bool) * tc->import_cap);
                        tc->import_is_stdlib = xrealloc(tc->import_is_stdlib, sizeof(bool) * tc->import_cap);
                    }
                    tc->imported_modules[tc->import_count] = item->alias ? item->alias : item->module;
                    tc->import_files[tc->import_count] = stmt->token.file; /* NULL = main file */
                    tc->import_lines[tc->import_count] = stmt->token.line;
                    tc->import_used[tc->import_count] = item->is_c_import; /* C imports are always "used" */
                    tc->import_is_stdlib[tc->import_count] = item->is_stdlib;
                    tc->import_count++;
                }
                /* Track alias → module mapping */
                if (item->alias && item->module && strcmp(item->alias, item->module) != 0) {
                    if (tc->alias_count >= tc->alias_cap) {
                        tc->alias_cap = tc->alias_cap ? tc->alias_cap * 2 : 8;
                        tc->alias_names = xrealloc(tc->alias_names, sizeof(const char *) * tc->alias_cap);
                        tc->alias_modules = xrealloc(tc->alias_modules, sizeof(const char *) * tc->alias_cap);
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
        const char *en = ENUM_DISPLAY_NAME(stmt);
        if (is_reserved_type_name(stmt->data.enum_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc, "'%s' is a reserved type name and cannot be used as an enum name", en);
            diag_error_msg(tc->diag, "E2038", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        /* E5016: builtin function name as enum name */
        if (is_reserved_builtin_func_name(stmt->data.enum_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc, "'%s' is a builtin function and cannot be used as an enum name", en);
            diag_error_msg(tc->diag, "E5016", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        /* E5035: stdlib module name as enum name */
        if (is_stdlib_module_name(stmt->data.enum_decl.name)) {
            char *msg = NULL;
            msg = tc_fmt(tc, "'%s' is a standard library module and cannot be used as an enum name", en);
            diag_error_msg(tc->diag, "E5035", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        /* E2016: empty enum */
        if (stmt->data.enum_decl.value_count == 0) {
            diag_error_codef(tc->diag, "E2016", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, en);
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
                    diag_error_codef(tc->diag, "E3033", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, en,
                        stmt->data.enum_decl.values[k].name,
                        stmt->data.enum_decl.values[j].name);
                    break;
                }
            }
        }
        /* E2014: check for duplicate enum variant names */
        /* E2065: check variant name vs enum type name */
        for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
            const char *vname = stmt->data.enum_decl.values[j].name;
            if (strcmp(vname, stmt->data.enum_decl.name) == 0) {
                diag_error_codef(tc->diag, "E2065", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, vname,
                    en);
            }
            /* E2038: reserved type name as enum variant */
            if (is_reserved_type_name(vname)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a reserved type name and cannot be used as an enum variant name",
                    vname);
                diag_error_msg(tc->diag, "E2038", msg,
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E5016: builtin function name as enum variant */
            if (is_reserved_builtin_func_name(vname)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a builtin function and cannot be used as an enum variant name",
                    vname);
                diag_error_msg(tc->diag, "E5016", msg,
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E5035: stdlib module name as enum variant */
            if (is_stdlib_module_name(vname)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "'%s' is a standard library module and cannot be used as an enum variant name",
                    vname);
                diag_error_msg(tc->diag, "E5035", msg,
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            for (int k = 0; k < j; k++) {
                if (strcmp(stmt->data.enum_decl.values[k].name, vname) == 0) {
                    diag_error_codef(tc->diag, "E2014", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, vname,
                        en);
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
            char *msg = NULL;
            msg = tc_fmt(tc,
                "a type named '%s' is already declared",
                ENUM_DISPLAY_NAME(stmt));
            diag_error_msg(tc->diag, "E4007", msg,
                NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        int vc = stmt->data.enum_decl.value_count;
        const char **vnames = arena_alloc(tc->arena, sizeof(const char *) * (vc ? vc : 1));
        for (int j = 0; j < vc; j++) {
            vnames[j] = stmt->data.enum_decl.values[j].name;
        }
        /* Extract payload info for tagged enums */
        bool has_tagged = stmt->data.enum_decl.is_tagged;
        const char ***pt = NULL;
        int *pc = NULL;
        if (vc > 0) {
            pt = arena_alloc(tc->arena, sizeof(const char **) * vc);
            pc = arena_alloc(tc->arena, sizeof(int) * vc);
            for (int j = 0; j < vc; j++) {
                EnumVal *ev = &stmt->data.enum_decl.values[j];
                pc[j] = ev->payload_count;
                if (ev->payload_count > 0) {
                    pt[j] = arena_alloc(tc->arena, sizeof(const char *) * ev->payload_count);
                    for (int k = 0; k < ev->payload_count; k++) {
                        pt[j][k] = ev->payload_types[k];
                    }
                } else {
                    pt[j] = NULL;
                }
            }
        }
        /* E3111: string enum with payloads */
        if (is_str && has_tagged) {
            diag_error_code(tc->diag, "E3111", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        /* E3112: flags enum with payloads */
        if (stmt->data.enum_decl.is_flags && has_tagged) {
            diag_error_code(tc->diag, "E3112", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
        }
        register_enum(tc, stmt->data.enum_decl.name, ENUM_DISPLAY_NAME(stmt), is_str, vnames, vc, pt, pc, has_tagged);
    }

    /* Pass 2b: Register structs and functions (enums already registered above) */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        tc->current_check_file = stmt->token.file;

        if (stmt->kind == NODE_STRUCT_DECL) {
            /* E2067: empty struct */
            if (stmt->data.struct_decl.field_count == 0) {
                diag_error_codef(tc->diag, "E2067", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, STRUCT_DISPLAY_NAME(stmt));
            }
            int fc = stmt->data.struct_decl.field_count;
            const char **fnames = arena_alloc(tc->arena, sizeof(const char *) * (fc ? fc : 1));
            GrayType **ftypes = arena_alloc(tc->arena, sizeof(GrayType *) * (fc ? fc : 1));
            for (int j = 0; j < fc; j++) {
                fnames[j] = stmt->data.struct_decl.fields[j].name;
                ftypes[j] = tc_type_from_name(tc, stmt->data.struct_decl.fields[j].type_name);
                /* E3038: void field type */
                if (stmt->data.struct_decl.fields[j].type_name &&
                    strcmp(stmt->data.struct_decl.fields[j].type_name, "void") == 0) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "'void' cannot be used as a struct field type (field '%s')",
                        fnames[j]);
                    diag_error_msg(tc->diag, "E3038", msg,
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
                /* E2066: field name matches struct type name */
                if (strcmp(fnames[j], stmt->data.struct_decl.name) == 0) {
                    diag_error_codef(tc->diag, "E2066", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, fnames[j], STRUCT_DISPLAY_NAME(stmt));
                }
                /* E5016: builtin function name as struct field name */
                if (is_reserved_builtin_func_name(fnames[j])) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "'%s' is a builtin function and cannot be used as a struct field name",
                        fnames[j]);
                    diag_error_msg(tc->diag, "E5016", msg,
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
                /* E5035: stdlib module name as struct field name */
                if (is_stdlib_module_name(fnames[j])) {
                    char *msg = NULL;
                    msg = tc_fmt(tc,
                        "'%s' is a standard library module and cannot be used as a struct field name",
                        fnames[j]);
                    diag_error_msg(tc->diag, "E5035", msg,
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
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
                            const char *visited[MAX_STRUCT_DEPTH];
                            int vc = 0;
                            is_cycle = struct_contains_by_value(
                                program, child, self_name, visited, &vc, 32);
                        }
                    }
                    if (is_cycle) {
                        char *msg = NULL;
                        const char *display = STRUCT_DISPLAY_NAME(stmt);
                        if (strcmp(ftn, self_name) == 0) {
                            msg = tc_fmt(tc,
                                "struct '%s' cannot contain itself by value; use a pointer field '^%s' for recursive types",
                                display, display);
                        } else {
                            msg = tc_fmt(tc,
                                "struct '%s' cannot contain itself by value through '%s'; break the cycle with a pointer field '^%s'",
                                display, ftn, ftn);
                        }
                        diag_error_msg(tc->diag, "E3061", msg,
                            NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                    }
                }
                /* Validate all named types within the field type recursively.
                 * Covers ^T, [T], map[K:V], and arbitrary nesting thereof. */
                if (ftn && *ftn)
                    validate_field_type_recursive(tc, program, ftn, fnames[j], stmt);
                /* Check for duplicate field names */
                for (int k = 0; k < j; k++) {
                    if (strcmp(fnames[k], fnames[j]) == 0) {
                        diag_error_codef(tc->diag, "E2013", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, fnames[j], STRUCT_DISPLAY_NAME(stmt));
                        break;
                    }
                }
            }
            /* E2037/E2038: reserved name check for structs */
            const char *sn = STRUCT_DISPLAY_NAME(stmt);
            if (is_reserved_type_name(stmt->data.struct_decl.name)) {
                char *msg = NULL;
                msg = tc_fmt(tc, "'%s' is a reserved type name and cannot be used as a struct name", sn);
                diag_error_msg(tc->diag, "E2037", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E5016: builtin function name as struct name */
            if (is_reserved_builtin_func_name(stmt->data.struct_decl.name)) {
                char *msg = NULL;
                msg = tc_fmt(tc, "'%s' is a builtin function and cannot be used as a struct name", sn);
                diag_error_msg(tc->diag, "E5016", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E5035: stdlib module name as struct name */
            if (is_stdlib_module_name(stmt->data.struct_decl.name)) {
                char *msg = NULL;
                msg = tc_fmt(tc, "'%s' is a standard library module and cannot be used as a struct name", sn);
                diag_error_msg(tc->diag, "E5035", msg, NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            /* E4007: duplicate struct name */
            if (is_struct_name(tc, stmt->data.struct_decl.name) ||
                is_enum_name(tc, stmt->data.struct_decl.name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "a type named '%s' is already declared",
                    STRUCT_DISPLAY_NAME(stmt));
                diag_error_msg(tc->diag, "E4007", msg,
                    NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
            }
            register_struct(tc, stmt->data.struct_decl.name, STRUCT_DISPLAY_NAME(stmt), fnames, ftypes, fc);

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
                        char *msg = NULL;
                        msg = tc_fmt(tc,
                            "duplicate function '%s' in struct '%s'",
                            FUNC_DISPLAY_NAME(fn), STRUCT_DISPLAY_NAME(stmt));
                        diag_error_msg(tc->diag, "E2037", msg,
                            NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0);
                        break;
                    }
                }
                /* E2064: function name conflicts with field name */
                for (int k = 0; k < fc; k++) {
                    if (strcmp(fnames[k], fn->data.func_decl.name) == 0) {
                        diag_error_codef(tc->diag, "E2064", NODE_FILE(tc, fn), fn->token.line, fn->token.column, 0, FUNC_DISPLAY_NAME(fn), fnames[k],
                            STRUCT_DISPLAY_NAME(stmt));
                        break;
                    }
                }
                int pc = fn->data.func_decl.param_count;
                GrayType **ptypes = arena_alloc(tc->arena, sizeof(GrayType *) * (pc ? pc : 1));
                for (int k = 0; k < pc; k++) {
                    ptypes[k] = tc_type_from_name(tc, fn->data.func_decl.params[k].type_name);
                }
                int rc = fn->data.func_decl.return_type_count;
                GrayType **rtypes = arena_alloc(tc->arena, sizeof(GrayType *) * (rc ? rc : 1));
                for (int k = 0; k < rc; k++) {
                    rtypes[k] = tc_type_from_name(tc, fn->data.func_decl.return_types[k]);
                }
                /* Register with prefixed name: StructName_funcName */
                char buf[MSG_BUF_SIZE];
                snprintf(buf, sizeof(buf), "%s_%s", stmt->data.struct_decl.name, fn->data.func_decl.name);
                const char *prefixed = arena_strdup(tc->arena, buf);
                register_func(tc, prefixed, ptypes, pc, rtypes, rc);
                tc->funcs[tc->func_count - 1].is_private = fn->data.func_decl.is_private;
                finalize_generic_sig(&tc->funcs[tc->func_count - 1], fn);
            }
        }

        /* Enums already processed in pass 2a above */

        if (stmt->kind == NODE_FUNC_DECL) {
            int pc = stmt->data.func_decl.param_count;
            GrayType **ptypes = arena_alloc(tc->arena, sizeof(GrayType *) * (pc ? pc : 1));
            for (int j = 0; j < pc; j++) {
                ptypes[j] = tc_type_from_name(tc, stmt->data.func_decl.params[j].type_name);
                tc_mark_type_module_used(tc, stmt->data.func_decl.params[j].type_name);
            }

            int rc = stmt->data.func_decl.return_type_count;
            GrayType **rtypes = arena_alloc(tc->arena, sizeof(GrayType *) * (rc ? rc : 1));
            for (int j = 0; j < rc; j++) {
                rtypes[j] = tc_type_from_name(tc, stmt->data.func_decl.return_types[j]);
                tc_mark_type_module_used(tc, stmt->data.func_decl.return_types[j]);
            }

            /* E4008: main() cannot have parameters or return types */
            if (strcmp(stmt->data.func_decl.name, "main") == 0) {
                if (pc > 0) {
                    diag_error_msg(tc->diag, "E4008",
                        "'main' function cannot have parameters; main() takes no arguments",
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
                if (rc > 0) {
                    diag_error_msg(tc->diag, "E4008",
                        "'main' function cannot have a return type; main() always returns void",
                        NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0);
                }
            }
            /* Check for reserved prefix */
            check_reserved_name(tc, stmt->data.func_decl.name,
                NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column);
            /* Check for duplicate function names */
            if (find_func(tc, stmt->data.func_decl.name)) {
                diag_error_codef(tc->diag, "E4004", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, FUNC_DISPLAY_NAME(stmt));
            }
            /* E4007: function name conflicts with a type */
            if (is_struct_name(tc, stmt->data.func_decl.name) ||
                is_enum_name(tc, stmt->data.func_decl.name)) {
                char *msg = NULL;
                msg = tc_fmt(tc,
                    "function '%s' conflicts with a type of the same name",
                    FUNC_DISPLAY_NAME(stmt));
                diag_error_msg(tc->diag, "E4007", msg,
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

static void typetable_free(TypeTable *tt) {
    if (!tt) return;
    free(tt->nodes);
    free(tt->types);
    free(tt);
}

void typechecker_free(TypeChecker *tc) {
    if (!tc) return;

    for (int i = 0; i < tc->func_count; i++) {
        free(tc->funcs[i].instantiations);
        free(tc->funcs[i].instantiation_calls);
    }
    free(tc->funcs);
    free(tc->funcs_sorted);

    free(tc->structs);
    free(tc->structs_sorted);

    free(tc->enum_names);
    free(tc->enum_display_names);
    free(tc->enum_is_string);
    free(tc->enum_values);
    free(tc->enum_value_counts);
    free(tc->enum_payload_types);
    free(tc->enum_payload_counts);
    free(tc->enum_is_tagged);
    free(tc->enum_names_sorted);
    free(tc->enum_names_sorted_indices);

    free(tc->imported_modules);
    free(tc->import_files);
    free(tc->import_lines);
    free(tc->import_used);
    free(tc->import_is_stdlib);

    free(tc->using_modules);
    free(tc->using_module_files);
    free(tc->using_module_import_indices);
    free(tc->alias_names);
    free(tc->alias_modules);

    free(tc->destroyed_arenas);
    free(tc->const_int_names);
    free(tc->const_int_values);

    typetable_free(tc->type_table);
    arena_destroy(tc->arena);
    scope_destroy(tc->current_scope);
    type_pool_reset();

    free(tc);
}

TypeChecker *typechecker_create(DiagnosticList *diag, const char *file) {
    /* Build sorted stdlib lookup tables once before any type-check begins. */
    static bool tables_built = false;
    if (!tables_built) {
        for (int i = 0; i < STDLIB_ARG_TABLE_N; i++) stdlib_arg_sorted[i] = &stdlib_arg_table[i];
        qsort(stdlib_arg_sorted, STDLIB_ARG_TABLE_N, sizeof(const StdlibArgEntry *), stdlib_arg_entry_cmp);
        for (int i = 0; i < STDLIB_ARG_TYPE_TABLE_N; i++) stdlib_argtype_sorted[i] = &stdlib_arg_type_table[i];
        qsort(stdlib_argtype_sorted, STDLIB_ARG_TYPE_TABLE_N, sizeof(const StdlibArgTypeEntry *), stdlib_argtype_entry_cmp);
        for (int i = 0; i < FALLIBLE_STDLIB_N; i++) fallible_stdlib_sorted[i] = &fallible_stdlib[i];
        qsort(fallible_stdlib_sorted, FALLIBLE_STDLIB_N, sizeof(const FallibleEntry *), fallible_entry_cmp);
        for (int i = 0; i < FALLIBLE_TYPE_TABLE_N; i++) fallible_type_sorted[i] = &fallible_type_table[i];
        qsort(fallible_type_sorted, FALLIBLE_TYPE_TABLE_N, sizeof(const FallibleTypeEntry *), fallible_type_entry_cmp);
        tables_built = true;
    }

    TypeChecker *tc = xcalloc(1, sizeof(TypeChecker));
    tc->diag = diag;
    tc->file = file;
    tc->current_scope = scope_create(NULL);
    tc->type_table = typetable_create();
    tc->arena = arena_create(64 * 1024);
    return tc;
}

void typechecker_check(TypeChecker *tc, AstNode *program) {
    if (!program || program->kind != NODE_PROGRAM) return;

    tc->program = program;

    /* Pass 1: register all type/function declarations */
    register_declarations(tc, program);

    /* SourceLocation is always registered: the here() builtin returns it
     * and is available without any import. */
    {
        const char **fnames = arena_alloc(tc->arena, sizeof(const char *) * 3);
        GrayType **ftypes = arena_alloc(tc->arena, sizeof(GrayType *) * 3);
        fnames[0] = "file"; fnames[1] = "line"; fnames[2] = "column";
        ftypes[0] = &TYPE_STRING; ftypes[1] = &TYPE_INT; ftypes[2] = &TYPE_INT;
        register_struct(tc, "SourceLocation", "SourceLocation", fnames, ftypes, 3);
    }

    /* Register stdlib struct types scoped to their module imports */
    if (tc_is_imported_module(tc, "server") || tc_is_imported_module(tc, "http")) {
        const char **fnames = arena_alloc(tc->arena, sizeof(const char *) * 3);
        GrayType **ftypes = arena_alloc(tc->arena, sizeof(GrayType *) * 3);
        fnames[0] = "status"; fnames[1] = "body"; fnames[2] = "headers";
        ftypes[0] = &TYPE_INT; ftypes[1] = &TYPE_STRING; ftypes[2] = type_from_name("map[string:string]");
        register_struct(tc, "HttpResponse", "HttpResponse", fnames, ftypes, 3);
    }

    if (tc_is_imported_module(tc, "server")) {
        const char **fnames = arena_alloc(tc->arena, sizeof(const char *) * 6);
        GrayType **ftypes = arena_alloc(tc->arena, sizeof(GrayType *) * 6);
        fnames[0] = "method"; fnames[1] = "path"; fnames[2] = "body";
        fnames[3] = "query"; fnames[4] = "headers"; fnames[5] = "params";
        ftypes[0] = &TYPE_STRING; ftypes[1] = &TYPE_STRING; ftypes[2] = &TYPE_STRING;
        ftypes[3] = type_from_name("map[string:string]");
        ftypes[4] = type_from_name("map[string:string]");
        ftypes[5] = type_from_name("map[string:string]");
        register_struct(tc, "HttpRequest", "HttpRequest", fnames, ftypes, 6);
    }

    if (tc_is_imported_module(tc, "uuid")) {
        const char **fnames = arena_alloc(tc->arena, sizeof(const char *) * 1);
        GrayType **ftypes = arena_alloc(tc->arena, sizeof(GrayType *) * 1);
        fnames[0] = "value";
        ftypes[0] = &TYPE_STRING;
        register_struct(tc, "UUID", "UUID", fnames, ftypes, 1);
    }

    /* Collect 'using' and 'import and use' module names */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_USING_STMT) {
            for (int j = 0; j < stmt->data.using_stmt.count; j++) {
                /* E2010: check that the module was imported BEFORE this using statement.
                 * For using stmts from imported files, just verify the module exists
                 * as an import (line ordering can't be compared across files). */
                const char *umod = stmt->data.using_stmt.modules[j];
                bool imported_before = false;
                const char *using_src = stmt->token.file;
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], umod) != 0) continue;
                    const char *imp_src = tc->import_files[mi];
                    /* Same file: require import line < using line */
                    bool same_file = (!using_src && !imp_src) ||
                        (using_src && imp_src && strcmp(using_src, imp_src) == 0);
                    if (same_file && tc->import_lines[mi] < stmt->token.line) {
                        imported_before = true;
                        break;
                    }
                    /* Cross-file: the using is from an imported file whose own
                     * import was already injected — just check existence */
                    if (!same_file && using_src) {
                        imported_before = true;
                        break;
                    }
                }
                if (!imported_before) {
                    diag_error_codef(tc->diag, "E2010", NODE_FILE(tc, stmt), stmt->token.line, stmt->token.column, 0, umod, umod);
                }
                if (tc->using_module_count >= tc->using_module_cap) {
                    tc->using_module_cap = tc->using_module_cap ? tc->using_module_cap * 2 : 8;
                    tc->using_modules = xrealloc(tc->using_modules,
                        sizeof(const char *) * (size_t)tc->using_module_cap);
                    tc->using_module_files = xrealloc(tc->using_module_files,
                        sizeof(const char *) * (size_t)tc->using_module_cap);
                    tc->using_module_import_indices = xrealloc(tc->using_module_import_indices,
                        sizeof(int) * (size_t)tc->using_module_cap);
                }
                tc->using_module_files[tc->using_module_count] = stmt->token.file;
                {
                    int mi = tc_find_import_index(tc, stmt->data.using_stmt.modules[j]);
                    tc->using_module_import_indices[tc->using_module_count] = mi;
                    if (mi >= 0) tc->import_used[mi] = true;
                }
                tc->using_modules[tc->using_module_count++] = stmt->data.using_stmt.modules[j];
            }
        }
        if (stmt->kind == NODE_IMPORT_STMT && stmt->data.import_stmt.auto_use) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->module) {
                    if (tc->using_module_count >= tc->using_module_cap) {
                        tc->using_module_cap = tc->using_module_cap ? tc->using_module_cap * 2 : 8;
                        tc->using_modules = xrealloc(tc->using_modules,
                            sizeof(const char *) * (size_t)tc->using_module_cap);
                        tc->using_module_files = xrealloc(tc->using_module_files,
                            sizeof(const char *) * (size_t)tc->using_module_cap);
                        tc->using_module_import_indices = xrealloc(tc->using_module_import_indices,
                            sizeof(int) * (size_t)tc->using_module_cap);
                    }
                    tc->using_module_files[tc->using_module_count] = stmt->token.file;
                    tc->using_module_import_indices[tc->using_module_count] =
                        tc_find_import_index(tc, item->module);
                    tc->using_modules[tc->using_module_count++] = item->module;
                }
            }
        }
    }

    /* Register unprefixed aliases for struct/enum types from 'import and use' modules.
     * Only process using-modules declared in the main file — transitive imports
     * should not leak their unprefixed aliases into the main compilation unit. */
    for (int ui = 0; ui < tc->using_module_count; ui++) {
        const char *uf = tc->using_module_files ? tc->using_module_files[ui] : NULL;
        bool is_main = (!uf && !tc->file) || (uf && tc->file && strcmp(uf, tc->file) == 0);
        if (!is_main) continue;
        const char *umod = tc->using_modules[ui];
        size_t umod_len = strlen(umod);
        char prefix[TYPE_NAME_MAX];
        snprintf(prefix, sizeof(prefix), "%s_", umod);
        size_t prefix_len = umod_len + 1;
        /* Check structs */
        for (int si = 0; si < tc->struct_count; si++) {
            const char *sn = tc->structs[si].struct_name;
            if (strncmp(sn, prefix, prefix_len) == 0) {
                const char *unprefixed = sn + prefix_len;
                if (!is_struct_name(tc, unprefixed)) {
                    int fc = tc->structs[si].field_count;
                    const char **fn = arena_alloc(tc->arena, sizeof(const char *) * (fc ? fc : 1));
                    GrayType **ft = arena_alloc(tc->arena, sizeof(GrayType *) * (fc ? fc : 1));
                    memcpy(fn, tc->structs[si].field_names, sizeof(const char *) * fc);
                    memcpy(ft, tc->structs[si].field_types, sizeof(GrayType *) * fc);
                    register_struct(tc, unprefixed, unprefixed, fn, ft, fc);
                }
            }
        }
        /* Check enums */
        for (int ei = 0; ei < tc->enum_count; ei++) {
            const char *en = tc->enum_names[ei];
            if (strncmp(en, prefix, prefix_len) == 0) {
                const char *unprefixed = en + prefix_len;
                if (!is_enum_name(tc, unprefixed)) {
                    register_enum(tc, unprefixed, unprefixed, tc->enum_is_string[ei],
                        tc->enum_values[ei], tc->enum_value_counts[ei],
                        tc->enum_payload_types[ei], tc->enum_payload_counts[ei],
                        tc->enum_is_tagged[ei]);
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

    /* Pass 1.5: Pre-register all global constants and variables that have an
     * explicit type annotation. This allows global initializers to reference
     * any other global constant regardless of declaration order within the
     * same file or across imported files. The def_line is intentionally left
     * as 0 to mark these as pre-registered (not real duplicates); Pass 2
     * overwrites each entry with the fully resolved type and real def_line. */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_VAR_DECL) continue;
        if (!stmt->data.var_decl.type_name) continue; /* inferred type — skip */
        if (scope_lookup_local(tc->current_scope, stmt->data.var_decl.name)) continue;
        GrayType *pre_t = type_from_name(stmt->data.var_decl.type_name);
        if (!pre_t) continue;
        scope_define(tc->current_scope, stmt->data.var_decl.name,
            pre_t, stmt->data.var_decl.mutable);
        /* Leave def_line = 0: sentinel that lets the E4003 duplicate check
         * in check_statement distinguish pre-registration from real duplicates. */
    }

    /* Pass 2: check all statements */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        tc->current_check_file = program->data.program.stmts[i]->token.file;
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
        tc->current_check_file = decl->token.file;
        for (int ii = 0; ii < fs->instantiation_count; ii++) {
            const char *concrete = fs->instantiations[ii];
            AstNode *call_site = fs->instantiation_calls[ii];

            Scope *inst_scope = scope_create(tc->current_scope);
            Scope *outer_scope = tc->current_scope;
            tc->current_scope = inst_scope;
            tc->func_depth++;

            for (int pi = 0; pi < decl->data.func_decl.param_count; pi++) {
                Param *p = &decl->data.func_decl.params[pi];
                /* Type parameter — not a variable; set binding for body re-check */
                if (p->is_type_param) {
                    tc->type_param_name = p->name;
                    tc->type_param_binding = concrete;
                    continue;
                }
                char *sub = substitute_wildcard(p->type_name, concrete);
                GrayType *pt = sub ? type_from_name(sub) : &TYPE_UNKNOWN;
                scope_define(inst_scope, p->name, pt, p->mutable);
                /* `sub` leaks on purpose; type_from_name stores the
                 * name pointer for array/map kinds and we need it
                 * alive for the duration of the re-check. Compile-
                 * time allocation; short-lived process. */
            }

            GrayType **prev_ret = tc->current_return_types;
            const char **prev_ret_names = tc->current_return_type_names;
            int prev_ret_count = tc->current_return_count;
            bool prev_named = tc->current_has_named_returns;
            const char **prev_return_names = tc->current_return_names;

            int rc = decl->data.func_decl.return_type_count;
            GrayType **ret_types = NULL;
            const char **ret_names = NULL;
            if (rc > 0) {
                ret_types = xmalloc(sizeof(GrayType *) * (size_t)rc);
                ret_names = xmalloc(sizeof(const char *) * (size_t)rc);
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
            tc->type_param_name = NULL;
            tc->type_param_binding = NULL;
            tc->current_scope = outer_scope;
            tc->func_depth--;
            free(ret_types);
            free(ret_names);
            scope_destroy(inst_scope);

            if (errs_after > errs_before && call_site) {
                diag_error_codef(tc->diag, "E3058", NODE_FILE(tc, call_site), call_site->token.line, call_site->token.column, 0, func_display_name(fs), concrete);
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
            "program has no main() function; every program needs 'do main() { }'",
            tc->file, err_line, 1, 0);
    }

    /* Warn about unused imports (only for the main file; imports from
     * sub-files are the sub-file author's responsibility). */
    for (int i = 0; i < tc->import_count; i++) {
        if (tc->import_files[i] && tc->file &&
            strcmp(tc->import_files[i], tc->file) != 0) continue; /* skip sub-file imports */
        if (!tc->import_used[i]) {
            char *msg = NULL;
            msg = tc_fmt(tc,
                "module '%s' is imported but never used; remove the import or use the module",
                tc->imported_modules[i]);
            diag_warning_msg(tc->diag, "W1002", msg,
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
            const char *display = (fs->decl && fs->decl->kind == NODE_FUNC_DECL &&
                fs->decl->data.func_decl.original_name) ?
                fs->decl->data.func_decl.original_name : fs->name;
            char *msg = NULL;
            msg = tc_fmt(tc,
                "function '%s' is declared but never called", display);
            diag_warning_msg(tc->diag, "W1003", msg,
                tc->file, fs->def_line, 1, 0);
        }
    }
}

TypeTable *typechecker_get_table(TypeChecker *tc) {
    return tc->type_table;
}
