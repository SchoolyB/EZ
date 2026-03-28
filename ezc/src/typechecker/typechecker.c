/*
 * typechecker.c - Type checking pass for the EZ language
 *
 * Walks the AST, resolves expression types, checks type correctness,
 * and builds a type table that the codegen can query.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "typechecker.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/* --- Type Table (open-addressing hash, pointer keys) --- */

static uint32_t hash_ptr(const void *ptr) {
    uintptr_t v = (uintptr_t)ptr;
    /* Fibonacci hashing — good distribution for pointer alignment */
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
            /* Empty slot — insert */
            tt->nodes[idx] = node;
            tt->types[idx] = type;
            tt->count++;
            return;
        }
        if (tt->nodes[idx] == node) {
            /* Already present — update */
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
    if (!si) return &TYPE_UNKNOWN;
    for (int i = 0; i < si->field_count; i++) {
        if (strcmp(si->field_names[i], field) == 0)
            return si->field_types[i];
    }
    return &TYPE_UNKNOWN;
}

/* --- Function signature helpers --- */

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
}

static FuncSig *find_func(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->func_count; i++) {
        if (strcmp(tc->funcs[i].name, name) == 0)
            return &tc->funcs[i];
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

static bool tc_is_imported_module(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->import_count; i++) {
        if (strcmp(tc->imported_modules[i], name) == 0) return true;
    }
    return false;
}

static bool tc_is_builtin(const char *name) {
    static const char *builtins[] = {
        "println", "print", "eprintln", "eprint", "input",
        "len", "typeof", "copy", "new", "ref", "error",
        "int", "uint", "float", "string", "char", "byte", "bool",
        "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64",
        "i128", "u128", "f32", "f64",
        "exit", "panic", "assert", "range", "cast",
        "append", "read_int", "to_int", "to_float", "to_string", "to_bool",
        "addr", "sleep_seconds", "sleep_milliseconds", "sleep_nanoseconds",
        NULL
    };
    for (int i = 0; builtins[i]; i++) {
        if (strcmp(name, builtins[i]) == 0) return true;
    }
    return false;
}

/* --- Enum helpers --- */

static void register_enum(TypeChecker *tc, const char *name, bool is_string) {
    if (tc->enum_count >= tc->enum_cap) {
        tc->enum_cap = tc->enum_cap ? tc->enum_cap * 2 : 8;
        tc->enum_names = realloc(tc->enum_names, sizeof(const char *) * tc->enum_cap);
        tc->enum_is_string = realloc(tc->enum_is_string, sizeof(bool) * tc->enum_cap);
    }
    tc->enum_names[tc->enum_count] = name;
    tc->enum_is_string[tc->enum_count] = is_string;
    tc->enum_count++;
}

static bool is_enum_name(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->enum_count; i++) {
        if (strcmp(tc->enum_names[i], name) == 0) return true;
    }
    return false;
}

/* --- Type signedness helpers --- */

/* Check if a TypeKind is any integer type (signed or unsigned) */
static bool is_int_kind(TypeKind k) {
    return k == TK_INT || k == TK_UINT;
}

static bool is_unsigned_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "uint") == 0 || strcmp(tn, "u8") == 0 ||
           strcmp(tn, "u16") == 0 || strcmp(tn, "u32") == 0 ||
           strcmp(tn, "u64") == 0 || strcmp(tn, "u128") == 0 ||
           strcmp(tn, "byte") == 0;
}

static bool is_signed_int_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "int") == 0 || strcmp(tn, "i8") == 0 ||
           strcmp(tn, "i16") == 0 || strcmp(tn, "i32") == 0 ||
           strcmp(tn, "i64") == 0 || strcmp(tn, "i128") == 0;
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
 * Returns true if an error was emitted. */
static bool check_integer_range(DiagnosticList *diag, const char *file,
    int line, int col, const char *type_name_str, int64_t value) {
    int64_t min_val = 0, max_val = 0;
    bool is_unsigned = false;

    if (strcmp(type_name_str, "i8") == 0)        { min_val = -128; max_val = 127; }
    else if (strcmp(type_name_str, "i16") == 0)   { min_val = -32768; max_val = 32767; }
    else if (strcmp(type_name_str, "i32") == 0)   { min_val = -2147483648LL; max_val = 2147483647; }
    else if (strcmp(type_name_str, "u8") == 0)    { min_val = 0; max_val = 255; is_unsigned = true; }
    else if (strcmp(type_name_str, "u16") == 0)   { min_val = 0; max_val = 65535; is_unsigned = true; }
    else if (strcmp(type_name_str, "u32") == 0)   { min_val = 0; max_val = 4294967295LL; is_unsigned = true; }
    else if (strcmp(type_name_str, "u64") == 0)   { min_val = 0; max_val = INT64_MAX; is_unsigned = true; }
    else if (strcmp(type_name_str, "uint") == 0)  { min_val = 0; max_val = INT64_MAX; is_unsigned = true; }
    else if (strcmp(type_name_str, "byte") == 0)  { min_val = 0; max_val = 255; is_unsigned = true; }
    else return false; /* not a range-checked type */

    if (value < min_val || value > max_val) {
        char msg[256];
        if (is_unsigned && value < 0) {
            snprintf(msg, sizeof(msg),
                "value %lld is out of range for type '%s' — unsigned types cannot hold negative values (valid range: %lld to %lld)",
                (long long)value, type_name_str, (long long)min_val, (long long)max_val);
        } else {
            snprintf(msg, sizeof(msg),
                "value %lld is out of range for type '%s' (valid range: %lld to %lld)",
                (long long)value, type_name_str, (long long)min_val, (long long)max_val);
        }
        diag_error(diag, "E3036", strdup(msg), file, line, col, 0);
        return true;
    }
    return false;
}

/* --- Expression type resolution --- */

static EzType *resolve_expr(TypeChecker *tc, AstNode *node);

static EzType *resolve_expr(TypeChecker *tc, AstNode *node) {
    if (!node) return &TYPE_UNKNOWN;

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
            resolve_expr(tc, node->data.interpolated_string.parts[i]);
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

        /* Type names used as values are caught downstream — they won't match
         * any variable in scope, and functions like new(), mem.make(), and casts
         * legitimately take type names as arguments. */

        Symbol *sym = scope_lookup(tc->current_scope, name);
        if (sym) {
            sym->used = true;
            result = sym->type;
        } else if (!is_enum_name(tc, name) && !find_func(tc, name) &&
                   !tc_is_builtin(name) && !is_struct_name(tc, name) &&
                   !tc_is_imported_module(tc, name)) {
            char msg[256];
            snprintf(msg, sizeof(msg), "undefined variable '%s'", name);
            const char *suggestion = suggest_name(tc, name);
            if (suggestion) {
                char help[256];
                snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                diag_error_help(tc->diag, "E4001", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0, strdup(help));
            } else {
                diag_error(tc->diag, "E4001", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "cannot negate type '%s' — only numeric types support negation",
                    type_name(right));
                diag_error(tc->diag, "E3007", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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

        /* String ordering operators not supported — use strings.compare() */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) &&
            (strcmp(op, "<") == 0 || strcmp(op, ">") == 0 ||
             strcmp(op, "<=") == 0 || strcmp(op, ">=") == 0)) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot use '%s' on strings; use strings.compare() instead", op);
            diag_error(tc->diag, "E3002", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
            result = &TYPE_BOOL;
            break;
        }

        /* String + non-string: only string + string is valid for concat */
        if (strcmp(op, "+") == 0 &&
            ((left->kind == TK_STRING && right->kind != TK_STRING && right->kind != TK_UNKNOWN) ||
             (right->kind == TK_STRING && left->kind != TK_STRING && left->kind != TK_UNKNOWN))) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot add %s and %s — use to_string() to convert", type_name(left), type_name(right));
            diag_error(tc->diag, "E3002", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }

        /* E3002: modulo on float */
        if (strcmp(op, "%") == 0 &&
            (left->kind == TK_FLOAT || right->kind == TK_FLOAT)) {
            diag_error(tc->diag, "E3002",
                strdup("modulo (%) only works on integers, not floats"),
                tc->file, node->token.line, node->token.column, 0);
        }

        /* E3002: bool used in arithmetic (e.g., 1 + true) */
        if ((left->kind == TK_BOOL || right->kind == TK_BOOL) &&
            (strcmp(op, "+") == 0 || strcmp(op, "-") == 0 ||
             strcmp(op, "*") == 0 || strcmp(op, "/") == 0 ||
             strcmp(op, "%") == 0) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "invalid operands: cannot use '%s' with %s and %s",
                op, type_name(left), type_name(right));
            diag_error(tc->diag, "E3002", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }

        /* Arithmetic on strings (other than + for concat) */
        if ((left->kind == TK_STRING || right->kind == TK_STRING) &&
            strcmp(op, "+") != 0 && strcmp(op, "==") != 0 && strcmp(op, "!=") != 0 &&
            strcmp(op, "in") != 0 && strcmp(op, "not_in") != 0 && strcmp(op, "!in") != 0) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot use '%s' on string type", op);
            diag_error(tc->diag, "E3002", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
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
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "cannot compare enum '%s' with enum '%s' — different enum types are never equal",
                    lname, rname);
                diag_error(tc->diag, "E3032", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
        }

        /* Comparison of incompatible types (e.g., int == string) */
        if ((strcmp(op, "==") == 0 || strcmp(op, "!=") == 0) &&
            left->kind != TK_UNKNOWN && right->kind != TK_UNKNOWN &&
            left->kind != right->kind && left->kind != TK_NIL && right->kind != TK_NIL &&
            !(is_int_kind(left->kind) && right->kind == TK_ENUM) &&
            !(left->kind == TK_ENUM && is_int_kind(right->kind)) &&
            !(left->kind == TK_STRUCT && is_int_kind(right->kind)) &&
            !(is_int_kind(left->kind) && right->kind == TK_STRUCT) &&
            !(is_int_kind(left->kind) && right->kind == TK_BOOL) &&
            !(left->kind == TK_BOOL && is_int_kind(right->kind))) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot compare %s with %s", type_name(left), type_name(right));
            diag_error(tc->diag, "E3001", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }

        if (strcmp(op, "==") == 0 || strcmp(op, "!=") == 0 ||
            strcmp(op, "<") == 0 || strcmp(op, ">") == 0 ||
            strcmp(op, "<=") == 0 || strcmp(op, ">=") == 0 ||
            strcmp(op, "&&") == 0 || strcmp(op, "||") == 0 ||
            strcmp(op, "in") == 0 || strcmp(op, "not_in") == 0) {
            result = &TYPE_BOOL;
        } else if (left->kind == TK_FLOAT || right->kind == TK_FLOAT) {
            result = &TYPE_FLOAT;
        } else if (left->kind == TK_STRING && strcmp(op, "+") == 0) {
            result = &TYPE_STRING;
        } else {
            result = left;
        }
        break;
    }

    case NODE_POSTFIX_EXPR: {
        EzType *left_t = resolve_expr(tc, node->data.postfix.left);
        if (strcmp(node->data.postfix.op, "^") == 0) {
            if (left_t->kind == TK_POINTER) {
                /* Dereference: ^T^ → T */
                result = type_from_name(left_t->element_type);
            } else if (left_t->kind != TK_UNKNOWN) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "cannot dereference non-pointer type '%s' — only ^T types can use ^",
                    type_name(left_t));
                diag_error(tc->diag, "E3016", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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
                diag_error(tc->diag, "E5015",
                    strdup("++ and -- require a variable — you cannot increment a literal or expression"),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* ++ and -- only valid on mutable numeric types */
            if (node->data.postfix.left->kind == NODE_LABEL) {
                Symbol *sym = scope_lookup(tc->current_scope, node->data.postfix.left->data.label.value);
                if (sym && !sym->mutable) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "cannot modify constant '%s' — declare with 'mut' to make it mutable",
                        node->data.postfix.left->data.label.value);
                    diag_error(tc->diag, "E3005", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
            if (left_t->kind != TK_UNKNOWN && !type_is_integer(left_t)) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "cannot use '%s' on type '%s' — only integer types support increment/decrement",
                    node->data.postfix.op, type_name(left_t));
                diag_error(tc->diag, "E5023", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            result = left_t;
        } else {
            result = left_t;
        }
        break;
    }

    case NODE_CALL_EXPR: {
        /* Resolve argument types first */
        for (int i = 0; i < node->data.call.arg_count; i++) {
            resolve_expr(tc, node->data.call.args[i]);
        }

        /* Resolve function return type */
        AstNode *fn = node->data.call.function;
        const char *fn_name = NULL;

        if (fn->kind == NODE_LABEL) {
            fn_name = fn->data.label.value;
        } else if (fn->kind == NODE_MEMBER_EXPR && fn->data.member.object->kind == NODE_LABEL) {
            const char *mod = fn->data.member.object->data.label.value;
            const char *mfn = fn->data.member.member;
            /* Mark module as used */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], mod) == 0) {
                    tc->import_used[mi] = true;
                    break;
                }
            }
            if (strcmp(mod, "mem") == 0) {
                if (strcmp(mfn, "arena") == 0) {
                    result = type_struct("Arena"); /* arena pointer — opaque */
                } else if (strcmp(mfn, "usage") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "new") == 0 && node->data.call.arg_count == 2) {
                    /* mem.new(arena, Type) returns ^Type */
                    AstNode *type_arg = node->data.call.args[1];
                    if (type_arg->kind == NODE_LABEL) {
                        result = type_pointer(type_arg->data.label.value);
                    } else {
                        result = &TYPE_UNKNOWN;
                    }
                } else if (strcmp(mfn, "make") == 0 && node->data.call.arg_count == 2) {
                    /* mem.make(arena, Type) returns ^Type */
                    AstNode *type_arg = node->data.call.args[1];
                    if (type_arg->kind == NODE_LABEL) {
                        result = type_pointer(type_arg->data.label.value);
                    } else {
                        result = &TYPE_UNKNOWN;
                    }
                } else if (strcmp(mfn, "alloc") == 0 && node->data.call.arg_count == 2) {
                    /* alloc returns the type of its second argument */
                    result = resolve_expr(tc, node->data.call.args[1]);
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "maps") == 0) {
                if (strcmp(mfn, "keys") == 0 || strcmp(mfn, "values") == 0) {
                    result = type_array("string"); /* approximate */
                } else if (strcmp(mfn, "has_key") == 0 || strcmp(mfn, "contains") == 0) {
                    result = &TYPE_BOOL;
                } else {
                    result = &TYPE_VOID;
                }
                /* E12001: maps functions require map argument */
                if (node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    EzType *arg0_t = typetable_get(tc->type_table, arg0);
                    if (arg0_t && arg0_t->kind == TK_ARRAY) {
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "maps.%s() requires a map argument, got an array", mfn);
                        diag_error(tc->diag, "E12001", strdup(msg),
                            tc->file, arg0->token.line, arg0->token.column, 0);
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
                    strcmp(mfn, "delete_file") == 0 || strcmp(mfn, "remove") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "file_exists") == 0 || strcmp(mfn, "exists") == 0 ||
                           strcmp(mfn, "is_file") == 0 || strcmp(mfn, "is_directory") == 0 ||
                           strcmp(mfn, "append_file") == 0 || strcmp(mfn, "rename") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "file_size") == 0) {
                    result = &TYPE_INT;
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "strings") == 0) {
                if (strcmp(mfn, "contains") == 0 || strcmp(mfn, "starts_with") == 0 ||
                    strcmp(mfn, "ends_with") == 0 || strcmp(mfn, "is_empty") == 0 ||
                    strcmp(mfn, "to_bool") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index") == 0 || strcmp(mfn, "last_index") == 0 ||
                           strcmp(mfn, "count") == 0 || strcmp(mfn, "to_int") == 0 ||
                           strcmp(mfn, "len") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "to_float") == 0) {
                    result = &TYPE_FLOAT;
                } else if (strcmp(mfn, "split") == 0 || strcmp(mfn, "chars") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "to_bytes") == 0) {
                    result = type_array("byte");
                } else {
                    result = &TYPE_STRING;
                }
                /* E7004: strings.repeat() second arg must be integer */
                if (strcmp(mfn, "repeat") == 0 && node->data.call.arg_count >= 2) {
                    EzType *count_t = typetable_get(tc->type_table, node->data.call.args[1]);
                    if (count_t && count_t->kind == TK_FLOAT) {
                        diag_error(tc->diag, "E7004",
                            strdup("strings.repeat() count must be an integer, not a float"),
                            tc->file, node->data.call.args[1]->token.line,
                            node->data.call.args[1]->token.column, 0);
                    }
                }
                /* E7004: strings.slice() bounds must be integers */
                if (strcmp(mfn, "slice") == 0 && node->data.call.arg_count >= 3) {
                    for (int si = 1; si <= 2 && si < node->data.call.arg_count; si++) {
                        EzType *bt = typetable_get(tc->type_table, node->data.call.args[si]);
                        if (bt && bt->kind == TK_FLOAT) {
                            diag_error(tc->diag, "E7004",
                                strdup("strings.slice() bounds must be integers, not floats"),
                                tc->file, node->data.call.args[si]->token.line,
                                node->data.call.args[si]->token.column, 0);
                        }
                    }
                }
            } else if (strcmp(mod, "time") == 0) {
                if (strcmp(mfn, "format") == 0 || strcmp(mfn, "iso") == 0 ||
                    strcmp(mfn, "date") == 0 || strcmp(mfn, "clock") == 0) {
                    result = &TYPE_STRING;
                } else {
                    result = &TYPE_INT;
                }
            } else if (strcmp(mod, "uuid") == 0) {
                if (strcmp(mfn, "is_valid") == 0) result = &TYPE_BOOL;
                else result = &TYPE_STRING;
            } else if (strcmp(mod, "encoding") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(mod, "crypto") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(mod, "bytes") == 0) {
                if (strcmp(mfn, "from_string") == 0 || strcmp(mfn, "from_hex") == 0 ||
                    strcmp(mfn, "from_base64") == 0) {
                    result = type_array("byte");
                } else {
                    result = &TYPE_STRING;
                }
            } else if (strcmp(mod, "binary") == 0) {
                if (strncmp(mfn, "encode", 6) == 0) result = type_array("byte");
                else if (strncmp(mfn, "decode_f", 8) == 0) result = &TYPE_FLOAT;
                else result = &TYPE_INT;
            } else if (strcmp(mod, "csv") == 0) {
                if (strcmp(mfn, "parse") == 0 || strcmp(mfn, "read") == 0) {
                    result = type_array("array");
                } else if (strcmp(mfn, "write") == 0) {
                    result = &TYPE_BOOL;
                } else {
                    result = &TYPE_STRING;
                }
            } else if (strcmp(mod, "json") == 0) {
                if (strcmp(mfn, "decode") == 0) result = type_from_name("map[string:string]");
                else if (strcmp(mfn, "is_valid") == 0) result = &TYPE_BOOL;
                else result = &TYPE_STRING;
            } else if (strcmp(mod, "sqlite") == 0) {
                if (strcmp(mfn, "open") == 0) result = &TYPE_UNKNOWN; /* opaque handle */
                else if (strcmp(mfn, "exec") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "query") == 0) result = type_array("map");
                else result = &TYPE_VOID;
            } else if (strcmp(mod, "random") == 0) {
                if (strcmp(mfn, "float") == 0) result = &TYPE_FLOAT;
                else if (strcmp(mfn, "int") == 0) result = &TYPE_INT;
                else if (strcmp(mfn, "bool") == 0) result = &TYPE_BOOL;
                else if (strcmp(mfn, "byte") == 0) result = &TYPE_BYTE;
                else if (strcmp(mfn, "char") == 0) result = &TYPE_CHAR;
                else if (strcmp(mfn, "shuffle") == 0 || strcmp(mfn, "sample") == 0) {
                    result = type_array("int");
                } else result = &TYPE_UNKNOWN;
            } else if (strcmp(mod, "arrays") == 0) {
                if (strcmp(mfn, "is_empty") == 0 || strcmp(mfn, "contains") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index_of") == 0 || strcmp(mfn, "sum") == 0 ||
                           strcmp(mfn, "min") == 0 || strcmp(mfn, "max") == 0 ||
                           strcmp(mfn, "count") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "reverse") == 0 || strcmp(mfn, "slice") == 0 ||
                           strcmp(mfn, "concat") == 0) {
                    result = type_array("int"); /* approximate */
                } else {
                    result = &TYPE_VOID;
                }
                /* E5007: arrays.append/insert/remove/pop on const array */
                if ((strcmp(mfn, "append") == 0 || strcmp(mfn, "insert") == 0 ||
                     strcmp(mfn, "remove") == 0 || strcmp(mfn, "pop") == 0 ||
                     strcmp(mfn, "sort") == 0 || strcmp(mfn, "clear") == 0) &&
                    node->data.call.arg_count > 0) {
                    AstNode *arg0 = node->data.call.args[0];
                    if (arg0->kind == NODE_LABEL) {
                        Symbol *sym = scope_lookup(tc->current_scope, arg0->data.label.value);
                        if (sym && !sym->mutable) {
                            char msg[256];
                            snprintf(msg, sizeof(msg),
                                "cannot modify immutable array '%s' — declare with 'mut' to allow modification",
                                arg0->data.label.value);
                            diag_error(tc->diag, "E5007", strdup(msg),
                                tc->file, node->token.line, node->token.column, 0);
                        }
                    }
                }
                /* E3001: arrays.append element type mismatch */
                if (strcmp(mfn, "append") == 0 && node->data.call.arg_count >= 2) {
                    AstNode *arg0 = node->data.call.args[0];
                    AstNode *arg1 = node->data.call.args[1];
                    EzType *arr_t = typetable_get(tc->type_table, arg0);
                    EzType *val_t = typetable_get(tc->type_table, arg1);
                    if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
                        val_t && val_t->kind != TK_UNKNOWN) {
                        EzType *elem_t = type_from_name(arr_t->element_type);
                        if (elem_t->kind != TK_UNKNOWN && elem_t->kind != val_t->kind) {
                            char msg[256];
                            snprintf(msg, sizeof(msg),
                                "type mismatch: cannot append %s to array of %s",
                                type_name(val_t), arr_t->element_type);
                            diag_error(tc->diag, "E3001", strdup(msg),
                                tc->file, arg1->token.line, arg1->token.column, 0);
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
                            char msg[256];
                            snprintf(msg, sizeof(msg),
                                "arrays.%s() requires a numeric array, got array of %s",
                                mfn, arr_t->element_type);
                            diag_error(tc->diag, "E9002", strdup(msg),
                                tc->file, arg0->token.line, arg0->token.column, 0);
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
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "type mismatch: cannot concat array of %s with array of %s",
                            t0->element_type, t1->element_type);
                        diag_error(tc->diag, "E3001", strdup(msg),
                            tc->file, a1->token.line, a1->token.column, 0);
                    }
                }
            } else if (strcmp(mod, "os") == 0) {
                if (strcmp(mfn, "args") == 0) {
                    result = type_array("string");
                } else if (strcmp(mfn, "get_env") == 0 || strcmp(mfn, "getenv") == 0 ||
                           strcmp(mfn, "cwd") == 0 || strcmp(mfn, "hostname") == 0 ||
                           strcmp(mfn, "arch") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "current_os") == 0 || strcmp(mfn, "pid") == 0) {
                    result = &TYPE_INT;
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "math") == 0) {
                /* Math functions return types */
                if (strcmp(mfn, "is_prime") == 0 || strcmp(mfn, "is_even") == 0 ||
                    strcmp(mfn, "is_odd") == 0 || strcmp(mfn, "is_inf") == 0 ||
                    strcmp(mfn, "is_nan") == 0 || strcmp(mfn, "is_finite") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "abs") == 0 || strcmp(mfn, "min") == 0 ||
                           strcmp(mfn, "max") == 0 || strcmp(mfn, "clamp") == 0 ||
                           strcmp(mfn, "sign") == 0 || strcmp(mfn, "factorial") == 0 ||
                           strcmp(mfn, "gcd") == 0 || strcmp(mfn, "lcm") == 0 ||
                           strcmp(mfn, "random") == 0) {
                    result = &TYPE_INT;
                } else {
                    result = &TYPE_FLOAT;
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
                            diag_error(tc->diag, "E7006",
                                strdup("threads.spawn() requires a function reference — use ()func_name or ref(func_name)"),
                                tc->file, node->token.line, node->token.column, 0);
                        }
                    }
                    result = type_struct("Thread"); /* EzThread — opaque */
                } else if (strcmp(mfn, "recv") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "id") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "mutex") == 0) {
                    result = type_struct("Mutex"); /* EzMutex — opaque */
                } else if (strcmp(mfn, "channel") == 0) {
                    result = type_struct("Channel"); /* EzChannel — opaque */
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "server") == 0) {
                if (strcmp(mfn, "router") == 0) {
                    result = type_struct("Router"); /* EzRouter — opaque */
                } else if (strcmp(mfn, "text") == 0 || strcmp(mfn, "json") == 0 ||
                           strcmp(mfn, "html") == 0 || strcmp(mfn, "redirect") == 0) {
                    result = type_struct("Http_Response"); /* EzResponse */
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "http") == 0) {
                /* All http methods return EzHttpResponse struct
                 * with .status (int), .body (string), .headers (map) */
                result = type_struct("Http_Response"); /* opaque struct — member access via __auto_type */
            } else if (strcmp(mod, "net") == 0) {
                if (strcmp(mfn, "listen") == 0) {
                    result = type_struct("Listener"); /* EzSocket — opaque */
                } else if (strcmp(mfn, "dial") == 0 || strcmp(mfn, "accept") == 0) {
                    result = type_struct("Socket"); /* EzSocket — opaque */
                } else if (strcmp(mfn, "send") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "recv") == 0 || strcmp(mfn, "resolve") == 0) {
                    result = &TYPE_STRING;
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "regex") == 0) {
                if (strcmp(mfn, "match") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "find") == 0 || strcmp(mfn, "replace") == 0) {
                    result = &TYPE_STRING;
                } else if (strcmp(mfn, "find_all") == 0 || strcmp(mfn, "split") == 0) {
                    result = type_array("string");
                } else {
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
                } else if (strcmp(mfn, "arena") == 0) {
                    result = type_struct("Arena");
                } else if (strcmp(mfn, "usage") == 0) {
                    result = &TYPE_INT;
                } else {
                    result = &TYPE_VOID;
                }
            } else if (strcmp(mod, "sqlite") == 0) {
                if (strcmp(mfn, "open") == 0) {
                    result = type_struct("Database");
                } else if (strcmp(mfn, "exec") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "query") == 0) {
                    result = type_array("map");
                } else {
                    result = &TYPE_VOID;
                }
            } else if (is_struct_name(tc, mod)) {
                /* Struct-namespaced function call: Type.func() — look up return type */
                char prefixed[256];
                snprintf(prefixed, sizeof(prefixed), "%s_%s", mod, mfn);
                FuncSig *sig = find_func(tc, prefixed);
                if (sig && sig->return_count > 0) {
                    result = sig->return_types[0];
                } else {
                    result = &TYPE_VOID;
                }
            } else {
                result = &TYPE_VOID;
            }
            break;
        }

        if (fn_name) {
            /* If calling a builtin, mark @std as used */
            if (tc_is_builtin(fn_name)) {
                for (int mi = 0; mi < tc->import_count; mi++) {
                    if (strcmp(tc->imported_modules[mi], "std") == 0) {
                        tc->import_used[mi] = true;
                        break;
                    }
                }
            }
            /* Check built-in functions first */
            if ((strcmp(fn_name, "addr") == 0 || strcmp(fn_name, "ref") == 0) && node->data.call.arg_count == 1) {
                AstNode *arg = node->data.call.args[0];
                /* addr() requires an lvalue (variable, field, index) */
                if (strcmp(fn_name, "addr") == 0 &&
                    arg->kind != NODE_LABEL && arg->kind != NODE_MEMBER_EXPR &&
                    arg->kind != NODE_INDEX_EXPR) {
                    diag_error(tc->diag, "E3012",
                        strdup("addr() requires a variable, field, or index expression — cannot take address of a literal or expression"),
                        tc->file, node->token.line, node->token.column, 0);
                }
                EzType *arg_t = resolve_expr(tc, arg);
                /* ref(func_name) returns func type, ref(var) returns pointer */
                if (strcmp(fn_name, "ref") == 0 && arg->kind == NODE_LABEL &&
                    find_func(tc, arg->data.label.value)) {
                    result = type_from_name("func");
                } else {
                    result = type_pointer(type_name(arg_t));
                }
            } else if (strcmp(fn_name, "len") == 0 || strcmp(fn_name, "to_int") == 0) {
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "to_float") == 0) {
                result = &TYPE_FLOAT;
            } else if (strcmp(fn_name, "typeof") == 0) {
                /* E3038: typeof() on void function result */
                if (node->data.call.arg_count > 0) {
                    EzType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                    if (arg_t->kind == TK_VOID) {
                        diag_error(tc->diag, "E3038",
                            strdup("cannot use typeof() on a void function — the function does not return a value"),
                            tc->file, node->token.line, node->token.column, 0);
                    }
                }
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "to_string") == 0 ||
                       strcmp(fn_name, "input") == 0) {
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "to_bool") == 0) {
                result = &TYPE_BOOL;
            } else if (strcmp(fn_name, "error") == 0) {
                result = type_from_name("Error");
            } else if (strcmp(fn_name, "exit") == 0 || strcmp(fn_name, "panic") == 0 ||
                       strcmp(fn_name, "assert") == 0 || strcmp(fn_name, "eprintln") == 0 ||
                       strcmp(fn_name, "eprint") == 0 ||
                       strcmp(fn_name, "sleep_seconds") == 0 ||
                       strcmp(fn_name, "sleep_milliseconds") == 0 ||
                       strcmp(fn_name, "sleep_nanoseconds") == 0) {
                result = &TYPE_VOID;
            } else if (strcmp(fn_name, "copy") == 0 && node->data.call.arg_count == 1) {
                result = resolve_expr(tc, node->data.call.args[0]);
            } else if (strcmp(fn_name, "read_int") == 0) {
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "char") == 0 && node->data.call.arg_count == 1) {
                /* E7014: char() with negative integer */
                int64_t lit_val;
                if (try_get_literal_int(node->data.call.args[0], &lit_val) && lit_val < 0) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "cannot convert %lld to char — value must be a valid Unicode code point (0 or greater)",
                        (long long)lit_val);
                    diag_error(tc->diag, "E7014", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
                result = &TYPE_CHAR;
            } else if ((strcmp(fn_name, "int") == 0 || strcmp(fn_name, "i8") == 0 ||
                        strcmp(fn_name, "i16") == 0 || strcmp(fn_name, "i32") == 0 ||
                        strcmp(fn_name, "i64") == 0 || strcmp(fn_name, "u8") == 0 ||
                        strcmp(fn_name, "u16") == 0 || strcmp(fn_name, "u32") == 0 ||
                        strcmp(fn_name, "u64") == 0 || strcmp(fn_name, "uint") == 0 ||
                        strcmp(fn_name, "byte") == 0) &&
                       node->data.call.arg_count == 1) {
                /* Range check for sized type conversion with literal */
                int64_t lit_val;
                if (try_get_literal_int(node->data.call.args[0], &lit_val)) {
                    check_integer_range(tc->diag, tc->file,
                        node->token.line, node->token.column,
                        fn_name, lit_val);
                }
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "string") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_STRING;
            } else if (strcmp(fn_name, "float") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_FLOAT;
            } else if (strcmp(fn_name, "bool") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_BOOL;
            } else if (strcmp(fn_name, "byte") == 0 && node->data.call.arg_count == 1) {
                result = &TYPE_BYTE;
            } else {
                FuncSig *sig = find_func(tc, fn_name);
                if (sig) {
                    sig->used = true;
                    /* Check argument count — account for default parameters */
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
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "function '%s' expects %d argument(s), got %d",
                            fn_name, sig->param_count, node->data.call.arg_count);
                        diag_error(tc->diag, "E5008", strdup(msg),
                            tc->file, node->token.line, node->token.column, 0);
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
                            char msg[256];
                            snprintf(msg, sizeof(msg),
                                "argument %d of '%s': expected %s, got %s",
                                ai + 1, fn_name, type_name(param_t), type_name(arg_t));
                            diag_error(tc->diag, "E3001", strdup(msg),
                                tc->file, node->data.call.args[ai]->token.line,
                                node->data.call.args[ai]->token.column, 0);
                        }
                        /* E3027: const variable passed to mutable param */
                        if (node->data.call.args[ai]->kind == NODE_LABEL) {
                            Symbol *arg_sym = scope_lookup(tc->current_scope,
                                node->data.call.args[ai]->data.label.value);
                            /* Find the func decl to check param mutability */
                            for (int fi = 0; fi < tc->program->data.program.stmt_count; fi++) {
                                AstNode *s = tc->program->data.program.stmts[fi];
                                if (s->kind == NODE_FUNC_DECL &&
                                    strcmp(s->data.func_decl.name, fn_name) == 0 &&
                                    ai < s->data.func_decl.param_count &&
                                    s->data.func_decl.params[ai].mutable &&
                                    arg_sym && !arg_sym->mutable) {
                                    char msg[256];
                                    snprintf(msg, sizeof(msg),
                                        "cannot pass constant '%s' to mutable parameter '%s' of '%s'",
                                        arg_sym->name, s->data.func_decl.params[ai].name, fn_name);
                                    diag_error(tc->diag, "E3027", strdup(msg),
                                        tc->file, node->data.call.args[ai]->token.line,
                                        node->data.call.args[ai]->token.column, 0);
                                    break;
                                }
                            }
                        }
                    }
                    if (sig->return_count > 0) {
                        result = sig->return_types[0];
                    } else {
                        result = &TYPE_VOID;
                    }
                } else {
                    /* Check if it's a variable holding a function reference */
                    Symbol *fn_sym = scope_lookup(tc->current_scope, fn_name);
                    if (fn_sym && fn_sym->type && strcmp(type_name(fn_sym->type), "func") == 0) {
                        result = &TYPE_UNKNOWN; /* callable func ref — return type unknown */
                    } else if (fn_sym && fn_sym->type) {
                        /* Variable exists but is not a function */
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "'%s' is a %s, not a function — it cannot be called",
                            fn_name, type_name(fn_sym->type));
                        diag_error(tc->diag, "E3015", strdup(msg),
                            tc->file, node->token.line, node->token.column, 0);
                    } else if (!tc_is_builtin(fn_name)) {
                        char msg[256];
                        snprintf(msg, sizeof(msg), "undefined function '%s'", fn_name);
                        const char *suggestion = suggest_name(tc, fn_name);
                        if (suggestion) {
                            char help[256];
                            snprintf(help, sizeof(help), "did you mean '%s'?", suggestion);
                            diag_error_help(tc->diag, "E4002", strdup(msg),
                                tc->file, node->token.line, node->token.column, 0, strdup(help));
                        } else {
                            diag_error(tc->diag, "E4002", strdup(msg),
                                tc->file, node->token.line, node->token.column, 0);
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
            if (strcmp(obj_name, "std") == 0) {
                result = &TYPE_INT; /* EXIT_SUCCESS, EXIT_FAILURE */
                break;
            }
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
                /* Check if this is a string enum */
                bool is_str_enum = false;
                for (int ei = 0; ei < tc->enum_count; ei++) {
                    if (strcmp(tc->enum_names[ei], obj_name) == 0) {
                        is_str_enum = tc->enum_is_string[ei];
                        break;
                    }
                }
                result = is_str_enum ? &TYPE_STRING : &TYPE_INT;
                break;
            }

            /* Otherwise it's a struct field or multi-return access */
            Symbol *sym = scope_lookup(tc->current_scope, obj_name);
            if (sym && sym->type->kind == TK_STRUCT) {
                result = struct_field_type(tc, sym->type->name, member);
                if (result->kind == TK_UNKNOWN && member[0] != 'v') {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "struct '%s' has no field '%s'", sym->type->name, member);
                    diag_error(tc->diag, "E3010", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            } else if (sym && member[0] == 'v' && member[1] >= '0' && member[1] <= '9') {
                /* Multi-return .v0/.v1/.v2 access — use stored return types */
                int idx = member[1] - '0';
                if (sym->ret_types && idx < sym->ret_count) {
                    result = sym->ret_types[idx];
                } else if (idx == 0) {
                    result = sym->type;
                } else {
                    result = type_from_name("Error"); /* fallback for (T, Error) pattern */
                }
            } else if (sym && sym->type->kind == TK_POINTER) {
                /* Pointer auto-deref field access */
                result = struct_field_type(tc, sym->type->element_type, member);
            } else if (sym && sym->type->kind != TK_UNKNOWN &&
                       sym->type->kind != TK_STRUCT && sym->type->kind != TK_ENUM &&
                       sym->type->kind != TK_POINTER &&
                       !(member[0] == 'v' && member[1] >= '0' && member[1] <= '9')) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "type '%s' has no fields — only structs support field access",
                    type_name(sym->type));
                diag_error(tc->diag, "E3013", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* Struct-namespaced function or enum access: Type.func() / Type.MEMBER */
            if (!sym && is_struct_name(tc, obj_name)) {
                result = &TYPE_UNKNOWN;
            }
        } else if (obj->kind == NODE_MEMBER_EXPR) {
            /* Nested member access: a.b.c — resolve a.b first, then look up .c */
            EzType *obj_t = typetable_get(tc->type_table, obj);
            if (!obj_t) obj_t = resolve_expr(tc, obj);
            if (obj_t && obj_t->kind == TK_STRUCT) {
                result = struct_field_type(tc, obj_t->name, member);
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
            char msg[256];
            snprintf(msg, sizeof(msg),
                "array index must be an integer, got %s", type_name(idx_t));
            diag_error(tc->diag, "E3003", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        if (left->kind == TK_ARRAY && left->element_type) {
            result = type_from_name(left->element_type);
        } else if (left->kind == TK_MAP && left->value_type) {
            result = type_from_name(left->value_type);
        } else if (left->kind == TK_STRING) {
            result = &TYPE_CHAR;
        } else if (left->kind != TK_UNKNOWN) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "type '%s' does not support indexing — only arrays, maps, and strings can be indexed",
                type_name(left));
            diag_error(tc->diag, "E3008", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        break;
    }

    case NODE_ARRAY_VALUE:
        if (node->data.array_value.count > 0) {
            EzType *elem = resolve_expr(tc, node->data.array_value.elements[0]);
            result = type_array(type_name(elem));
        }
        break;

    case NODE_MAP_VALUE: {
        /* Resolve key and value types */
        for (int i = 0; i < node->data.map_value.count; i++) {
            resolve_expr(tc, node->data.map_value.keys[i]);
            resolve_expr(tc, node->data.map_value.values[i]);
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
                    char msg[256];
                    snprintf(msg, sizeof(msg), "duplicate key in map literal");
                    diag_error(tc->diag, "E12006", strdup(msg),
                        tc->file, kj->token.line, kj->token.column, 0);
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
        StructInfo *si = find_struct(tc, sname);
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
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "struct '%s' has no field '%s'", sname, fname);
                    diag_error(tc->diag, "E3010", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                } else if (expected_t && val_t->kind != TK_UNKNOWN &&
                           expected_t->kind != TK_UNKNOWN &&
                           expected_t->kind != val_t->kind &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_ENUM) &&
                           !(expected_t->kind == TK_ENUM && is_int_kind(val_t->kind)) &&
                           !(expected_t->kind == TK_STRUCT && is_int_kind(val_t->kind)) &&
                           !(is_int_kind(expected_t->kind) && val_t->kind == TK_STRUCT) &&
                           !(expected_t->kind == TK_FLOAT && is_int_kind(val_t->kind))) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "field '%s' of struct '%s': expected %s, got %s",
                        fname, sname, type_name(expected_t), type_name(val_t));
                    diag_error(tc->diag, "E3001", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
        }
        result = type_struct(sname);
        break;
    }

    case NODE_RANGE_EXPR:
        result = &TYPE_INT;
        break;

    case NODE_CAST_EXPR:
        resolve_expr(tc, node->data.cast.value);
        result = type_from_name(node->data.cast.target_type);
        break;

    case NODE_NEW_EXPR:
        result = type_pointer(node->data.new_expr.type_name);
        break;

    case NODE_FUNC_REF:
        result = type_from_name("func");
        break;

    default:
        break;
    }

    typetable_set(tc->type_table, node, result);
    return result;
}

/* Check if a name uses a reserved prefix that would collide with generated C.
 * Skip names starting with _ez_ as those are compiler-generated temporaries. */
static void check_reserved_name(TypeChecker *tc, const char *name, int line, int col) {
    if (!name) return;
    /* Skip compiler-generated temps (_ez_tmp, _ez_or, _ez_idx, etc.) */
    if (strncmp(name, "_ez_", 4) == 0) return;
    if (strncmp(name, "ez_", 3) == 0 || strncmp(name, "Ez", 2) == 0) {
        char msg[256];
        snprintf(msg, sizeof(msg),
            "name '%s' uses reserved prefix (ez_, _ez_, Ez) — these are reserved for the compiler",
            name);
        diag_error(tc->diag, "E4006", strdup(msg), tc->file, line, col, 0);
    }
}

/* --- Statement checking --- */

static void check_statement(TypeChecker *tc, AstNode *node);

/* Check if ALL paths through a block end in a return statement */
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
    return false;
}

static void check_block(TypeChecker *tc, AstNode *node) {
    if (!node || node->kind != NODE_BLOCK_STMT) return;
    for (int i = 0; i < node->data.block.count; i++) {
        check_statement(tc, node->data.block.stmts[i]);
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
            diag_error(tc->diag, "E2056",
                strdup("executable statements are not allowed at file scope — put this inside a function"),
                tc->file, node->token.line, node->token.column, 0);
        }
    }

    switch (node->kind) {
    case NODE_VAR_DECL: {
        /* E3038: void cannot be used as variable type */
        if (node->data.var_decl.type_name && strcmp(node->data.var_decl.type_name, "void") == 0) {
            diag_error(tc->diag, "E3038",
                strdup("'void' cannot be used as a variable type"),
                tc->file, node->token.line, node->token.column, 0);
        }
        /* E3038: void in array/map types */
        if (node->data.var_decl.type_name) {
            const char *tn = node->data.var_decl.type_name;
            if (strstr(tn, "void") != NULL && strcmp(tn, "void") != 0) {
                diag_error(tc->diag, "E3038",
                    strdup("'void' cannot be used as an element type in arrays or maps"),
                    tc->file, node->token.line, node->token.column, 0);
            }
        }
        /* E3034: 'any' type is reserved */
        if (node->data.var_decl.type_name && strcmp(node->data.var_decl.type_name, "any") == 0) {
            diag_error(tc->diag, "E3034",
                strdup("'any' type is reserved for internal use and cannot be used in declarations"),
                tc->file, node->token.line, node->token.column, 0);
        }
        /* const must have a value */
        if (!node->data.var_decl.mutable && !node->data.var_decl.value) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "constant '%s' must have a value — add = followed by a value",
                node->data.var_decl.name);
            diag_error(tc->diag, "E2011", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        /* Check for type keyword used as value: mut x = int */
        if (node->data.var_decl.value && node->data.var_decl.value->kind == NODE_LABEL) {
            const char *vname = node->data.var_decl.value->data.label.value;
            if (strcmp(vname, "int") == 0 || strcmp(vname, "uint") == 0 ||
                strcmp(vname, "float") == 0 || strcmp(vname, "string") == 0 ||
                strcmp(vname, "bool") == 0 || strcmp(vname, "char") == 0 ||
                strcmp(vname, "byte") == 0 || strcmp(vname, "void") == 0) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "'%s' is a type, not a value — did you mean to declare a type? (e.g., mut x %s = ...)",
                    vname, vname);
                diag_error(tc->diag, "E3011", strdup(msg),
                    tc->file, node->data.var_decl.value->token.line,
                    node->data.var_decl.value->token.column, 0);
            }
        }

        EzType *declared = node->data.var_decl.type_name
            ? type_from_name(node->data.var_decl.type_name)
            : &TYPE_UNKNOWN;

        if (node->data.var_decl.value) {
            EzType *value_type = resolve_expr(tc, node->data.var_decl.value);
            /* E3038: cannot assign void function result */
            if (value_type->kind == TK_VOID) {
                diag_error(tc->diag, "E3038",
                    strdup("cannot assign the result of a void function to a variable"),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* Check for multi-return to single variable
             * (skip if this is part of a multi-var expansion — the value will be a .v0 access) */
            if (node->data.var_decl.value->kind == NODE_CALL_EXPR &&
                node->data.var_decl.value->data.call.function->kind == NODE_LABEL &&
                strncmp(node->data.var_decl.name, "_ez_tmp", 7) != 0 &&
                strncmp(node->data.var_decl.name, "_ez_or", 6) != 0) {
                const char *call_name = node->data.var_decl.value->data.call.function->data.label.value;
                FuncSig *sig = find_func(tc, call_name);
                if (sig && sig->return_count > 1) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "'%s' returns %d values — use mut a, b = %s() to capture all of them",
                        call_name, sig->return_count, call_name);
                    diag_error(tc->diag, "E3040", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
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
                       /* Skip mismatch on enum assignment (enum resolves as int) */
                       !(declared->kind == TK_STRUCT && is_int_kind(value_type->kind) &&
                         is_enum_name(tc, node->data.var_decl.type_name)) &&
                       /* Skip mismatch on multi-var expansion (.v0/.v1 access) */
                       !(node->data.var_decl.value &&
                         node->data.var_decl.value->kind == NODE_MEMBER_EXPR &&
                         node->data.var_decl.value->data.member.member[0] == 'v')) {
                /* Type mismatch */
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign %s to %s",
                    type_name(value_type), type_name(declared));
                diag_error(tc->diag, "E3001", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* E3036: Check literal value fits in sized integer type */
            if (node->data.var_decl.type_name && node->data.var_decl.value) {
                int64_t lit_val;
                if (try_get_literal_int(node->data.var_decl.value, &lit_val)) {
                    check_integer_range(tc->diag, tc->file,
                        node->token.line, node->token.column,
                        node->data.var_decl.type_name, lit_val);
                }
                /* E3026/E3036: Check array literal elements fit in sized element type */
                const char *tn = node->data.var_decl.type_name;
                if (tn[0] == '[' && node->data.var_decl.value->kind == NODE_ARRAY_VALUE) {
                    /* Extract element type name from "[byte]", "[i8]", "[u8, 3]", etc. */
                    char elem_type[64] = {0};
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
                        for (int ei = 0; ei < arr->data.array_value.count; ei++) {
                            int64_t ev;
                            if (try_get_literal_int(arr->data.array_value.elements[ei], &ev)) {
                                check_integer_range(tc->diag, tc->file,
                                    arr->data.array_value.elements[ei]->token.line,
                                    arr->data.array_value.elements[ei]->token.column,
                                    elem_type, ev);
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
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "cannot assign signed type '%s' to unsigned type '%s' — value may be negative",
                        src_sym->declared_type, node->data.var_decl.type_name);
                    diag_error(tc->diag, "E3019", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
        }

        if (strcmp(node->data.var_decl.name, "_") != 0) {
            /* Check for reserved prefix */
            check_reserved_name(tc, node->data.var_decl.name,
                node->token.line, node->token.column);
            /* Check for redeclaration in same scope */
            Symbol *existing = scope_lookup_local(tc->current_scope,
                node->data.var_decl.name);
            if (existing) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "variable '%s' already declared in this scope (line %d)",
                    node->data.var_decl.name, existing->def_line);
                diag_error(tc->diag, "E4003", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* W2002: check if variable shadows outer scope */
            if (!existing && tc->current_scope->parent) {
                Symbol *outer_sym = scope_lookup(tc->current_scope->parent,
                    node->data.var_decl.name);
                if (outer_sym && outer_sym->def_line > 0) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "variable '%s' shadows a variable declared on line %d",
                        node->data.var_decl.name, outer_sym->def_line);
                    diag_warning(tc->diag, "W2002", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
            /* E4012: shadows a type */
            if (is_struct_name(tc, node->data.var_decl.name) ||
                is_enum_name(tc, node->data.var_decl.name)) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "variable '%s' shadows a type definition with the same name",
                    node->data.var_decl.name);
                diag_error(tc->diag, "E4012", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* E4013: shadows a function */
            if (find_func(tc, node->data.var_decl.name)) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "variable '%s' shadows a function with the same name",
                    node->data.var_decl.name);
                diag_error(tc->diag, "E4013", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* E4014: shadows an imported module */
            for (int mi = 0; mi < tc->import_count; mi++) {
                if (strcmp(tc->imported_modules[mi], node->data.var_decl.name) == 0) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "variable '%s' shadows an imported module with the same name",
                        node->data.var_decl.name);
                    diag_error(tc->diag, "E4014", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                    break;
                }
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
                }
                /* Store multi-return types for temp variables from calls */
                if (fn->kind == NODE_LABEL) {
                    FuncSig *sig = find_func(tc, fn->data.label.value);
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
        }
        break;
    }

    case NODE_ASSIGN_STMT: {
        EzType *target_t = resolve_expr(tc, node->data.assign.target);
        EzType *value_t = resolve_expr(tc, node->data.assign.value);

        /* Check for assignment to const variable (direct, index, or field) */
        AstNode *target = node->data.assign.target;
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
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot modify constant '%s' — declare with 'mut' to make it mutable",
                const_name);
            diag_error(tc->diag, "E3005", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        /* Check type mismatch on assignment (only for direct variable targets) */
        if (target->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.label.value);
            if (sym && sym->type->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                target_t->kind != TK_UNKNOWN &&
                target_t->kind != value_t->kind &&
                !(target_t->kind == TK_ENUM && is_int_kind(value_t->kind)) &&
                !(is_int_kind(target_t->kind) && value_t->kind == TK_ENUM) &&
                !(target_t->kind == TK_STRUCT && is_int_kind(value_t->kind))) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "type mismatch: cannot assign %s to %s variable '%s'",
                    type_name(value_t), type_name(target_t), target->data.label.value);
                diag_error(tc->diag, "E3001", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
        }
        /* Check type mismatch on struct field assignment */
        if (target->kind == NODE_MEMBER_EXPR && target->data.member.object->kind == NODE_LABEL) {
            Symbol *sym = scope_lookup(tc->current_scope, target->data.member.object->data.label.value);
            if (sym && sym->type->kind == TK_STRUCT) {
                EzType *field_t = struct_field_type(tc, sym->type->name, target->data.member.member);
                if (field_t->kind != TK_UNKNOWN && value_t->kind != TK_UNKNOWN &&
                    field_t->kind != value_t->kind &&
                    !(is_int_kind(field_t->kind) && value_t->kind == TK_ENUM) &&
                    !(field_t->kind == TK_FLOAT && is_int_kind(value_t->kind))) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "type mismatch: cannot assign %s to %s field '%s'",
                        type_name(value_t), type_name(field_t), target->data.member.member);
                    diag_error(tc->diag, "E3001", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
        }
        break;
    }

    case NODE_RETURN_STMT:
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            resolve_expr(tc, node->data.return_stmt.values[i]);
        }
        /* Check return type matches function signature */
        if (tc->current_return_count == 0 && node->data.return_stmt.count > 0) {
            /* Returning a value from a void function */
            diag_error(tc->diag, "E3006", strdup("cannot return a value from a void function"),
                tc->file, node->token.line, node->token.column, 0);
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count == 0) {
            /* Bare return in non-void function */
            diag_error(tc->diag, "E3006",
                strdup("missing return value — function expects a return value"),
                tc->file, node->token.line, node->token.column, 0);
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count != tc->current_return_count &&
                   node->data.return_stmt.count < tc->current_return_count) {
            /* E3013: wrong number of return values (skip or_return synthetic returns
             * which have count=1 but the function expects more — that's handled by codegen) */
            /* Only error if user explicitly returns fewer values */
            bool is_or_return_synthetic = false;
            if (node->data.return_stmt.count == 1 &&
                node->data.return_stmt.values[0]->kind == NODE_MEMBER_EXPR) {
                AstNode *obj = node->data.return_stmt.values[0]->data.member.object;
                if (obj->kind == NODE_LABEL && strncmp(obj->data.label.value, "_ez_or", 6) == 0) {
                    is_or_return_synthetic = true;
                }
            }
            if (!is_or_return_synthetic) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "function expects %d return value(s), got %d",
                    tc->current_return_count, node->data.return_stmt.count);
                diag_error(tc->diag, "E3013", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
        } else if (tc->current_return_count > 0 && node->data.return_stmt.count > 0 &&
                   node->data.return_stmt.count == tc->current_return_count) {
            /* Check first return value type (skip for or_return synthetic returns) */
            EzType *ret_t = resolve_expr(tc, node->data.return_stmt.values[0]);
            EzType *expected = tc->current_return_types[0];
            if (ret_t->kind != TK_UNKNOWN && expected->kind != TK_UNKNOWN &&
                ret_t->kind != expected->kind && ret_t->kind != TK_NIL &&
                /* int/uint are compatible at kind level (E5024 checks signedness) */
                !(is_int_kind(expected->kind) && is_int_kind(ret_t->kind)) &&
                !(is_int_kind(expected->kind) && ret_t->kind == TK_ENUM) &&
                !(expected->kind == TK_ENUM && is_int_kind(ret_t->kind)) &&
                !(expected->kind == TK_STRUCT && is_int_kind(ret_t->kind)) &&
                !(is_int_kind(expected->kind) && ret_t->kind == TK_STRUCT) &&
                !(expected->kind == TK_FLOAT && is_int_kind(ret_t->kind))) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "return type mismatch: expected %s, got %s",
                    type_name(expected), type_name(ret_t));
                diag_error(tc->diag, "E3001", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            }
            /* E5024: signed-to-unsigned return type mismatch */
            if (tc->current_return_type_names && tc->current_return_type_names[0] &&
                is_unsigned_type(tc->current_return_type_names[0]) &&
                node->data.return_stmt.values[0]->kind == NODE_LABEL) {
                const char *src_name = node->data.return_stmt.values[0]->data.label.value;
                Symbol *src_sym = scope_lookup(tc->current_scope, src_name);
                if (src_sym && src_sym->declared_type &&
                    is_signed_int_type(src_sym->declared_type)) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "return type mismatch: cannot return signed '%s' as unsigned '%s'",
                        src_sym->declared_type, tc->current_return_type_names[0]);
                    diag_error(tc->diag, "E5024", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
        }
        break;

    case NODE_EXPR_STMT: {
        EzType *expr_t = resolve_expr(tc, node->data.expr_stmt.expr);
        /* E5011: warn about unused return value from function call */
        AstNode *expr = node->data.expr_stmt.expr;
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
                strcmp(fn_name, "exit") == 0 || strcmp(fn_name, "sleep_seconds") == 0 ||
                strcmp(fn_name, "sleep_milliseconds") == 0 || strcmp(fn_name, "sleep_nanoseconds") == 0);
            /* Don't warn for module calls that are commonly used for side effects */
            if (fn->kind == NODE_MEMBER_EXPR) is_side_effect = true;
            if (!is_side_effect && fn_name) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "return value of '%s()' is not used — assign it to a variable or use '_' to discard",
                    fn_name);
                diag_error(tc->diag, "E5011", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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
        resolve_expr(tc, node->data.if_stmt.condition);
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
                r->data.range_expr.end->kind == NODE_INT_VALUE &&
                r->data.range_expr.start->data.int_value.value >=
                r->data.range_expr.end->data.int_value.value) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "invalid range: start (%lld) must be less than end (%lld)",
                    (long long)r->data.range_expr.start->data.int_value.value,
                    (long long)r->data.range_expr.end->data.int_value.value);
                diag_error(tc->diag, "E9005", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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
            char msg[256];
            snprintf(msg, sizeof(msg),
                "cannot iterate over type '%s' — for_each requires an array, map, or string",
                type_name(coll_t));
            diag_error(tc->diag, "E3009", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
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

    case NODE_WHILE_STMT:
        resolve_expr(tc, node->data.while_stmt.condition);
        tc->loop_depth++;
        check_block(tc, node->data.while_stmt.body);
        tc->loop_depth--;
        break;

    case NODE_LOOP_STMT:
        tc->loop_depth++;
        check_block(tc, node->data.loop_stmt.body);
        tc->loop_depth--;
        break;

    case NODE_BREAK_STMT:
    case NODE_CONTINUE_STMT:
        if (tc->loop_depth == 0) {
            const char *kw = (node->kind == NODE_BREAK_STMT) ? "break" : "continue";
            char msg[128];
            snprintf(msg, sizeof(msg), "'%s' can only be used inside a loop", kw);
            diag_error(tc->diag, "E2050", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        break;

    case NODE_FUNC_DECL: {
        /* Check for nested function declarations */
        if (tc->func_depth > 0) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "nested function declarations are not allowed — define '%s' at the top level",
                node->data.func_decl.name);
            diag_error(tc->diag, "E2051", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }

        Scope *func_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = func_scope;
        tc->func_depth++;

        /* Define parameters in function scope, check for duplicates */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            Param *p = &node->data.func_decl.params[i];
            /* Check for duplicate parameter name */
            for (int j = 0; j < i; j++) {
                if (strcmp(node->data.func_decl.params[j].name, p->name) == 0) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "duplicate parameter name '%s'", p->name);
                    diag_error(tc->diag, "E2012", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
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
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "required parameter '%s' cannot come after a parameter with a default value",
                        p->name);
                    diag_error(tc->diag, "E2039", strdup(msg),
                        tc->file, node->token.line, node->token.column, 0);
                }
            }
            EzType *ptype = p->type_name ? type_from_name(p->type_name) : &TYPE_UNKNOWN;
            scope_define(func_scope, p->name, ptype, p->mutable);
        }

        /* Define named return variables in function scope */
        if (node->data.func_decl.return_names) {
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                if (node->data.func_decl.return_names[i]) {
                    EzType *rt = type_from_name(node->data.func_decl.return_types[i]);
                    scope_define(func_scope, node->data.func_decl.return_names[i], rt, true);
                }
            }
        }

        /* Track current function return types for return statement checking */
        EzType **prev_ret = tc->current_return_types;
        const char **prev_ret_names = tc->current_return_type_names;
        int prev_ret_count = tc->current_return_count;
        if (node->data.func_decl.return_type_count > 0) {
            tc->current_return_types = malloc(sizeof(EzType *) * node->data.func_decl.return_type_count);
            tc->current_return_type_names = malloc(sizeof(const char *) * node->data.func_decl.return_type_count);
            tc->current_return_count = node->data.func_decl.return_type_count;
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                tc->current_return_types[i] = type_from_name(node->data.func_decl.return_types[i]);
                tc->current_return_type_names[i] = node->data.func_decl.return_types[i];
            }
        } else {
            tc->current_return_types = NULL;
            tc->current_return_type_names = NULL;
            tc->current_return_count = 0;
        }

        check_block(tc, node->data.func_decl.body);

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
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "function '%s' must return a value but has no return statement",
                    node->data.func_decl.name);
                diag_error(tc->diag, "E3024", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
            } else if (has_return && !has_named_returns &&
                       !all_paths_return(node->data.func_decl.body)) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "not all code paths in '%s' return a value",
                    node->data.func_decl.name);
                diag_error(tc->diag, "E3035", strdup(msg),
                    tc->file, node->token.line, node->token.column, 0);
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
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "variable '%s' is declared but never used", s->name);
                    diag_warning(tc->diag, "W1001", strdup(msg),
                        tc->file, s->def_line, s->def_column, 0);
                }
            }
        }

        if (tc->current_return_types) free(tc->current_return_types);
        if (tc->current_return_type_names) free(tc->current_return_type_names);
        tc->current_return_types = prev_ret;
        tc->current_return_type_names = prev_ret_names;
        tc->current_return_count = prev_ret_count;
        tc->func_depth--;
        tc->current_scope = outer;
        break;
    }

    case NODE_IMPORT_STMT:
        if (tc->func_depth > 0) {
            diag_error(tc->diag, "E2036",
                strdup("imports must be at the top of the file, not inside a function"),
                tc->file, node->token.line, node->token.column, 0);
        }
        break;

    case NODE_ENSURE_STMT:
        resolve_expr(tc, node->data.ensure_stmt.expr);
        if (node->data.ensure_stmt.expr &&
            node->data.ensure_stmt.expr->kind != NODE_CALL_EXPR) {
            diag_error(tc->diag, "E3039",
                strdup("ensure expects a function call — for example: ensure close(file)"),
                tc->file, node->token.line, node->token.column, 0);
        }
        break;

    case NODE_STRUCT_DECL:
        /* E2053: struct inside function */
        if (tc->func_depth > 0) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "struct '%s' must be defined at the top level, not inside a function",
                node->data.struct_decl.name);
            diag_error(tc->diag, "E2053", strdup(msg),
                tc->file, node->token.line, node->token.column, 0);
        }
        /* Type-check struct-namespaced function bodies */
        for (int i = 0; i < node->data.struct_decl.func_count; i++) {
            AstNode *fn = node->data.struct_decl.funcs[i].func_decl;
            if (fn && fn->kind == NODE_FUNC_DECL) {
                check_statement(tc, fn);
            }
        }
        break;

    case NODE_WHEN_STMT: {
        resolve_expr(tc, node->data.when_stmt.value);
        /* E2043: check for duplicate case values */
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            for (int j = 0; j < node->data.when_stmt.cases[i].value_count; j++) {
                AstNode *val_i = node->data.when_stmt.cases[i].values[j];
                resolve_expr(tc, val_i);
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
                            char msg[256];
                            snprintf(msg, sizeof(msg), "duplicate case value in when statement");
                            diag_error(tc->diag, "E2043", strdup(msg),
                                tc->file, val_i->token.line, val_i->token.column, 0);
                        }
                    }
                }
            }
            check_block(tc, node->data.when_stmt.cases[i].body);
        }
        if (node->data.when_stmt.default_body) {
            check_block(tc, node->data.when_stmt.default_body);
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
        "std", "math", "strings", "arrays", "maps", "io", "os", "time",
        "random", "json", "csv", "encoding", "crypto", "uuid", "bytes",
        "binary", "fmt", "http", "server", "regex", "net", "threads",
        "mem", "sqlite", "errors", "db", NULL
    };
    for (int i = 0; modules[i]; i++) {
        if (strcmp(name, modules[i]) == 0) return true;
    }
    return false;
}

static void register_declarations(TypeChecker *tc, AstNode *program) {
    /* Validate and record imports */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_IMPORT_STMT) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->is_stdlib && item->module && !is_valid_module(item->module)) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "unknown module '@%s'", item->module);
                    diag_error(tc->diag, "E6001", strdup(msg),
                        tc->file, stmt->token.line, stmt->token.column, 0);
                }
                /* Record import for unused-import tracking */
                if (item->is_stdlib && item->module) {
                    if (tc->import_count >= tc->import_cap) {
                        tc->import_cap = tc->import_cap ? tc->import_cap * 2 : 16;
                        tc->imported_modules = realloc(tc->imported_modules, sizeof(const char *) * tc->import_cap);
                        tc->import_lines = realloc(tc->import_lines, sizeof(int) * tc->import_cap);
                        tc->import_used = realloc(tc->import_used, sizeof(bool) * tc->import_cap);
                    }
                    tc->imported_modules[tc->import_count] = item->module;
                    tc->import_lines[tc->import_count] = stmt->token.line;
                    tc->import_used[tc->import_count] = false;
                    tc->import_count++;
                }
            }
        }
    }

    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];

        if (stmt->kind == NODE_STRUCT_DECL) {
            int fc = stmt->data.struct_decl.field_count;
            const char **fnames = malloc(sizeof(const char *) * (fc ? fc : 1));
            EzType **ftypes = malloc(sizeof(EzType *) * (fc ? fc : 1));
            if (!fnames || !ftypes) { free(fnames); free(ftypes); continue; }
            for (int j = 0; j < fc; j++) {
                fnames[j] = stmt->data.struct_decl.fields[j].name;
                ftypes[j] = type_from_name(stmt->data.struct_decl.fields[j].type_name);
                /* E3038: void field type */
                if (stmt->data.struct_decl.fields[j].type_name &&
                    strcmp(stmt->data.struct_decl.fields[j].type_name, "void") == 0) {
                    char msg[256];
                    snprintf(msg, sizeof(msg),
                        "'void' cannot be used as a struct field type (field '%s')",
                        fnames[j]);
                    diag_error(tc->diag, "E3038", strdup(msg),
                        tc->file, stmt->token.line, stmt->token.column, 0);
                }
                /* Check for duplicate field names */
                for (int k = 0; k < j; k++) {
                    if (strcmp(fnames[k], fnames[j]) == 0) {
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "duplicate field name '%s' in struct '%s'",
                            fnames[j], stmt->data.struct_decl.name);
                        diag_error(tc->diag, "E2013", strdup(msg),
                            tc->file, stmt->token.line, stmt->token.column, 0);
                        break;
                    }
                }
            }
            /* E2037/E2038: reserved name check for structs */
            const char *sn = stmt->data.struct_decl.name;
            if (strcmp(sn, "int") == 0 || strcmp(sn, "uint") == 0 ||
                strcmp(sn, "float") == 0 || strcmp(sn, "string") == 0 ||
                strcmp(sn, "bool") == 0 || strcmp(sn, "char") == 0 ||
                strcmp(sn, "byte") == 0 || strcmp(sn, "void") == 0 ||
                strcmp(sn, "Error") == 0 || strcmp(sn, "nil") == 0) {
                char msg[256];
                snprintf(msg, sizeof(msg), "'%s' is a reserved type name and cannot be used as a struct name", sn);
                diag_error(tc->diag, "E2037", strdup(msg), tc->file, stmt->token.line, stmt->token.column, 0);
            }
            register_struct(tc, stmt->data.struct_decl.name, fnames, ftypes, fc);

            /* Register struct-namespaced functions as StructName_funcName */
            for (int j = 0; j < stmt->data.struct_decl.func_count; j++) {
                AstNode *fn = stmt->data.struct_decl.funcs[j].func_decl;
                if (!fn || fn->kind != NODE_FUNC_DECL) continue;
                /* E2037: check for duplicate function names in struct */
                for (int k = 0; k < j; k++) {
                    AstNode *prev = stmt->data.struct_decl.funcs[k].func_decl;
                    if (prev && prev->kind == NODE_FUNC_DECL &&
                        strcmp(prev->data.func_decl.name, fn->data.func_decl.name) == 0) {
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "duplicate function '%s' in struct '%s'",
                            fn->data.func_decl.name, stmt->data.struct_decl.name);
                        diag_error(tc->diag, "E2037", strdup(msg),
                            tc->file, fn->token.line, fn->token.column, 0);
                        break;
                    }
                }
                int pc = fn->data.func_decl.param_count;
                EzType **ptypes = malloc(sizeof(EzType *) * (pc ? pc : 1));
                if (!ptypes) continue;
                for (int k = 0; k < pc; k++) {
                    ptypes[k] = type_from_name(fn->data.func_decl.params[k].type_name);
                }
                int rc = fn->data.func_decl.return_type_count;
                EzType **rtypes = malloc(sizeof(EzType *) * (rc ? rc : 1));
                if (!rtypes) { free(ptypes); continue; }
                for (int k = 0; k < rc; k++) {
                    rtypes[k] = type_from_name(fn->data.func_decl.return_types[k]);
                }
                /* Register with prefixed name: StructName_funcName */
                char *prefixed = malloc(strlen(stmt->data.struct_decl.name) +
                    strlen(fn->data.func_decl.name) + 2);
                sprintf(prefixed, "%s_%s", stmt->data.struct_decl.name, fn->data.func_decl.name);
                register_func(tc, prefixed, ptypes, pc, rtypes, rc);
            }
        }

        if (stmt->kind == NODE_ENUM_DECL) {
            /* E2038: reserved name for enums */
            const char *en = stmt->data.enum_decl.name;
            if (strcmp(en, "int") == 0 || strcmp(en, "uint") == 0 ||
                strcmp(en, "float") == 0 || strcmp(en, "string") == 0 ||
                strcmp(en, "bool") == 0 || strcmp(en, "char") == 0 ||
                strcmp(en, "byte") == 0 || strcmp(en, "void") == 0 ||
                strcmp(en, "Error") == 0 || strcmp(en, "nil") == 0) {
                char msg[256];
                snprintf(msg, sizeof(msg), "'%s' is a reserved type name and cannot be used as an enum name", en);
                diag_error(tc->diag, "E2038", strdup(msg), tc->file, stmt->token.line, stmt->token.column, 0);
            }
            /* E2016: empty enum */
            if (stmt->data.enum_decl.value_count == 0) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "enum '%s' has no values — an enum must have at least one value",
                    stmt->data.enum_decl.name);
                diag_error(tc->diag, "E2016", strdup(msg),
                    tc->file, stmt->token.line, stmt->token.column, 0);
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
                        char msg[256];
                        snprintf(msg, sizeof(msg),
                            "duplicate value in enum '%s': '%s' and '%s' both have the same value",
                            stmt->data.enum_decl.name,
                            stmt->data.enum_decl.values[k].name,
                            stmt->data.enum_decl.values[j].name);
                        diag_error(tc->diag, "E3033", strdup(msg),
                            tc->file, stmt->token.line, stmt->token.column, 0);
                        break;
                    }
                }
            }
            /* Detect string enum: has explicit base_type "string" or string literal values */
            bool is_str = false;
            if (stmt->data.enum_decl.base_type &&
                strcmp(stmt->data.enum_decl.base_type, "string") == 0) {
                is_str = true;
            } else {
                for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                    if (stmt->data.enum_decl.values[j].value &&
                        stmt->data.enum_decl.values[j].value->kind == NODE_STRING_VALUE) {
                        is_str = true;
                        break;
                    }
                }
            }
            register_enum(tc, stmt->data.enum_decl.name, is_str);
        }

        if (stmt->kind == NODE_FUNC_DECL) {
            int pc = stmt->data.func_decl.param_count;
            EzType **ptypes = malloc(sizeof(EzType *) * (pc ? pc : 1));
            if (!ptypes) continue;
            for (int j = 0; j < pc; j++) {
                ptypes[j] = type_from_name(stmt->data.func_decl.params[j].type_name);
            }

            int rc = stmt->data.func_decl.return_type_count;
            EzType **rtypes = malloc(sizeof(EzType *) * (rc ? rc : 1));
            if (!rtypes) { free(ptypes); continue; }
            for (int j = 0; j < rc; j++) {
                rtypes[j] = type_from_name(stmt->data.func_decl.return_types[j]);
            }

            /* Check for reserved prefix */
            check_reserved_name(tc, stmt->data.func_decl.name,
                stmt->token.line, stmt->token.column);
            /* Check for duplicate function names */
            if (find_func(tc, stmt->data.func_decl.name)) {
                char msg[256];
                snprintf(msg, sizeof(msg),
                    "function '%s' already declared", stmt->data.func_decl.name);
                diag_error(tc->diag, "E4004", strdup(msg),
                    tc->file, stmt->token.line, stmt->token.column, 0);
            }
            register_func(tc, stmt->data.func_decl.name, ptypes, pc, rtypes, rc);
            /* Store line for unused function warning */
            tc->funcs[tc->func_count - 1].def_line = stmt->token.line;
        }
    }
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

    /* Pass 2: check all statements */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        check_statement(tc, program->data.program.stmts[i]);
    }

    /* Verify main() exists */
    if (!find_func(tc, "main")) {
        diag_error(tc->diag, "E4005",
            strdup("program has no main() function — every EZ program needs 'do main() { }'"),
            tc->file, 1, 1, 0);
    }

    /* Warn about unused imports */
    for (int i = 0; i < tc->import_count; i++) {
        if (!tc->import_used[i]) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "module '%s' is imported but never used — remove the import or use the module",
                tc->imported_modules[i]);
            diag_warning(tc->diag, "W2001", strdup(msg),
                tc->file, tc->import_lines[i], 1, 0);
        }
    }

    /* Warn about unused functions (skip main and struct-namespaced) */
    for (int i = 0; i < tc->func_count; i++) {
        FuncSig *fs = &tc->funcs[i];
        if (!fs->used && fs->def_line > 0 &&
            strcmp(fs->name, "main") != 0 &&
            !(fs->name[0] >= 'A' && fs->name[0] <= 'Z' && strchr(fs->name, '_'))) {
            char msg[256];
            snprintf(msg, sizeof(msg),
                "function '%s' is declared but never called", fs->name);
            diag_warning(tc->diag, "W1003", strdup(msg),
                tc->file, fs->def_line, 1, 0);
        }
    }
}

TypeTable *typechecker_get_table(TypeChecker *tc) {
    return tc->type_table;
}
