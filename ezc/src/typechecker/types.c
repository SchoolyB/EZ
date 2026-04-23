/*
 * types.c - Type representation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "types.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* Built-in type singletons */
EzType TYPE_VOID    = {TK_VOID,   "void",   NULL, NULL, NULL, NULL};
EzType TYPE_INT     = {TK_INT,    "int",    NULL, NULL, NULL, NULL};
EzType TYPE_UINT    = {TK_UINT,   "uint",   NULL, NULL, NULL, NULL};
EzType TYPE_FLOAT   = {TK_FLOAT,  "float",  NULL, NULL, NULL, NULL};
EzType TYPE_BOOL    = {TK_BOOL,   "bool",   NULL, NULL, NULL, NULL};
EzType TYPE_CHAR    = {TK_CHAR,   "char",   NULL, NULL, NULL, NULL};
EzType TYPE_BYTE    = {TK_BYTE,   "byte",   NULL, NULL, NULL, NULL};
EzType TYPE_STRING  = {TK_STRING, "string", NULL, NULL, NULL, NULL};
EzType TYPE_NIL     = {TK_NIL,    "nil",    NULL, NULL, NULL, NULL};
EzType TYPE_UNKNOWN = {TK_UNKNOWN,"unknown",NULL, NULL, NULL, NULL};

#define TYPE_POOL_CAPACITY 1024

/* Pool for dynamically created types (leak on exit, fine for a compiler) */
static EzType type_pool[TYPE_POOL_CAPACITY];
static int type_pool_count = 0;

EzType *type_alloc(void) {
    if (type_pool_count >= TYPE_POOL_CAPACITY) {
        fprintf(stderr, "error: type pool exhausted (%d types) — please report this bug\n", TYPE_POOL_CAPACITY);
        exit(1);
    }
    return &type_pool[type_pool_count++];
}

EzType *type_array(const char *elem_type) {
    EzType *t = type_alloc();
    t->kind = TK_ARRAY;
    t->element_type = elem_type;
    t->name = elem_type; /* simplified */
    return t;
}

EzType *type_struct(const char *name) {
    EzType *t = type_alloc();
    t->kind = TK_STRUCT;
    t->name = name;
    return t;
}

EzType *type_enum(const char *name) {
    EzType *t = type_alloc();
    t->kind = TK_ENUM;
    t->name = name;
    return t;
}

EzType *type_pointer(const char *pointee_type) {
    EzType *t = type_alloc();
    t->kind = TK_POINTER;
    t->element_type = pointee_type; /* reuse element_type for pointee */
    t->name = pointee_type;
    return t;
}

/* Find the index of the matching ')' for the '(' at start. Tracks nesting
 * so that nested func(...) types (e.g. func(func(int)->int)->bool) work.
 * Returns -1 if not balanced. */
static int find_matching_paren(const char *s, int start) {
    int depth = 0;
    for (int i = start; s[i]; i++) {
        if (s[i] == '(') depth++;
        else if (s[i] == ')') {
            depth--;
            if (depth == 0) return i;
        }
    }
    return -1;
}

/* Split src[start..end-1] on commas at top-level depth, dup each segment.
 * Sets *out_count and allocates *out_arr (caller frees the array but the
 * dup'd strings live with the type pool — they're leaked at exit). */
static void split_top_commas(const char *src, int start, int end,
                              int *out_count, char ***out_arr) {
    *out_count = 0;
    *out_arr = NULL;
    if (end <= start) return;
    int cap = 4;
    char **arr = (char **)malloc(sizeof(char *) * (size_t)cap);
    int n = 0;
    int depth = 0;
    int seg_start = start;
    for (int i = start; i <= end; i++) {
        char c = (i < end) ? src[i] : ',';
        if (c == '(' || c == '[') depth++;
        else if (c == ')' || c == ']') depth--;
        if (depth == 0 && (c == ',' || i == end)) {
            int seg_len = i - seg_start;
            char *seg = (char *)malloc((size_t)seg_len + 1);
            memcpy(seg, src + seg_start, (size_t)seg_len);
            seg[seg_len] = '\0';
            if (n == cap) {
                cap *= 2;
                arr = (char **)realloc(arr, sizeof(char *) * (size_t)cap);
            }
            arr[n++] = seg;
            seg_start = i + 1;
        }
    }
    *out_count = n;
    *out_arr = arr;
}

/* Parse "func(p1,&p2,...)->R" or "func(...)->()" or "func(...)->(R1,R2)"
 * into an EzFuncSig. Returns NULL if the string isn't well-formed. */
static EzFuncSig *parse_func_sig(const char *name) {
    if (strncmp(name, "func(", 5) != 0) return NULL;
    int rparen = find_matching_paren(name, 4);
    if (rparen < 0) return NULL;
    EzFuncSig *sig = (EzFuncSig *)malloc(sizeof(EzFuncSig));
    sig->param_count = 0;
    sig->param_types = NULL;
    sig->param_mutable = NULL;
    sig->return_count = 0;
    sig->return_types = NULL;

    /* Params */
    char **raw_params = NULL;
    int raw_count = 0;
    split_top_commas(name, 5, rparen, &raw_count, &raw_params);
    sig->param_count = raw_count;
    if (raw_count > 0) {
        sig->param_types = (const char **)malloc(sizeof(char *) * (size_t)raw_count);
        sig->param_mutable = (bool *)malloc(sizeof(bool) * (size_t)raw_count);
        for (int i = 0; i < raw_count; i++) {
            const char *p = raw_params[i];
            if (p[0] == '&') {
                sig->param_mutable[i] = true;
                sig->param_types[i] = strdup(p + 1);
            } else {
                sig->param_mutable[i] = false;
                sig->param_types[i] = strdup(p);
            }
            free(raw_params[i]);
        }
    }
    free(raw_params);

    /* Return */
    const char *after = name + rparen + 1;
    if (after[0] == '-' && after[1] == '>') {
        const char *ret = after + 2;
        if (ret[0] == '(') {
            int ret_len = (int)strlen(ret);
            if (ret[ret_len - 1] != ')') return sig; /* malformed; keep params */
            char **rets = NULL;
            int rcount = 0;
            split_top_commas(ret, 1, ret_len - 1, &rcount, &rets);
            sig->return_count = rcount;
            if (rcount > 0) {
                sig->return_types = (const char **)malloc(sizeof(char *) * (size_t)rcount);
                for (int i = 0; i < rcount; i++) {
                    sig->return_types[i] = rets[i];
                }
            }
            free(rets);
        } else if (strcmp(ret, "void") != 0) {
            sig->return_count = 1;
            sig->return_types = (const char **)malloc(sizeof(char *));
            sig->return_types[0] = strdup(ret);
        }
    }
    return sig;
}

bool type_is_numeric(EzType *t) {
    return t->kind == TK_INT || t->kind == TK_UINT || t->kind == TK_FLOAT ||
           t->kind == TK_CHAR || t->kind == TK_BYTE;
}

bool type_is_integer(EzType *t) {
    return t->kind == TK_INT || t->kind == TK_UINT ||
           t->kind == TK_CHAR || t->kind == TK_BYTE;
}

bool type_eq(EzType *a, EzType *b) {
    if (a->kind != b->kind) return false;
    if (a->kind == TK_STRUCT || a->kind == TK_ENUM) {
        return strcmp(a->name, b->name) == 0;
    }
    if (a->kind == TK_ARRAY) {
        return strcmp(a->element_type, b->element_type) == 0;
    }
    if (a->kind == TK_FUNCTION) {
        /* Canonical encoded form lives in name; signatures equal iff the
         * encoded strings match (parser writes a canonical form). */
        return a->name && b->name && strcmp(a->name, b->name) == 0;
    }
    return true;
}

const char *type_name(EzType *t) {
    if (!t) return "unknown";
    return t->name;
}

EzType *type_from_name(const char *name) {
    if (!name) return &TYPE_UNKNOWN;

    if (strcmp(name, "int") == 0)    return &TYPE_INT;
    if (strcmp(name, "uint") == 0)   return &TYPE_UINT;
    if (strcmp(name, "float") == 0)  return &TYPE_FLOAT;

    /* Sized integer and float types: same kind as parent but preserve name */
    if (strcmp(name, "i8") == 0 || strcmp(name, "i16") == 0 ||
        strcmp(name, "i32") == 0 || strcmp(name, "i64") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_INT;
        t->name = name;
        return t;
    }
    if (strcmp(name, "i128") == 0 || strcmp(name, "i256") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_INT;
        t->name = name;
        return t;
    }
    if (strcmp(name, "u8") == 0 || strcmp(name, "u16") == 0 ||
        strcmp(name, "u32") == 0 || strcmp(name, "u64") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_UINT;
        t->name = name;
        return t;
    }
    if (strcmp(name, "u128") == 0 || strcmp(name, "u256") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_UINT;
        t->name = name;
        return t;
    }
    if (strcmp(name, "f32") == 0 || strcmp(name, "f64") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_FLOAT;
        t->name = name;
        return t;
    }
    if (strcmp(name, "bool") == 0)   return &TYPE_BOOL;
    if (strcmp(name, "char") == 0)   return &TYPE_CHAR;
    if (strcmp(name, "byte") == 0)   return &TYPE_BYTE;
    if (strcmp(name, "string") == 0) return &TYPE_STRING;
    if (strcmp(name, "void") == 0)   return &TYPE_VOID;
    if (strcmp(name, "nil") == 0)    return &TYPE_NIL;
    /* Bare "func" is no longer a valid type — the parser rejects it before
     * reaching here. Defensive fallback: model as TK_UNKNOWN so downstream
     * code doesn't crash if it slips through. */
    if (strcmp(name, "func") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_UNKNOWN;
        t->name = "func";
        return t;
    }
    /* Typed function reference: "func(p1,&p2)->R" */
    if (strncmp(name, "func(", 5) == 0) {
        EzType *t = type_alloc();
        t->kind = TK_FUNCTION;
        t->name = name; /* canonical encoded form is the name */
        t->func_sig = parse_func_sig(name);
        return t;
    }
    if (strcmp(name, "Error") == 0) {
        EzType *t = type_alloc();
        t->kind = TK_ERROR;
        t->name = "Error";
        return t;
    }

    /* Pointer type: ^int, ^Person, etc. */
    if (name[0] == '^') {
        return type_pointer(name + 1);
    }

    /* Array type: [int], [string], [int,3], etc. */
    if (name[0] == '[') {
        size_t len = strlen(name);
        if (len > 2 && name[len - 1] == ']') {
            char *elem = malloc(len - 1);
            if (!elem) return &TYPE_UNKNOWN;
            memcpy(elem, name + 1, len - 2);
            elem[len - 2] = '\0';
            /* Strip ",N" suffix for fixed-size arrays like [string,3] */
            char *comma = strchr(elem, ',');
            if (comma) *comma = '\0';
            return type_array(elem);
        }
    }

    /* Map type: map[K:V] */
    if (strncmp(name, "map[", 4) == 0) {
        EzType *t = type_alloc();
        t->kind = TK_MAP;
        t->name = name;
        /* Parse key:value types from "map[string:int]" */
        const char *start = name + 4;
        const char *colon = strchr(start, ':');
        if (colon) {
            size_t klen = (size_t)(colon - start);
            char *key = malloc(klen + 1);
            if (!key) return &TYPE_UNKNOWN;
            memcpy(key, start, klen);
            key[klen] = '\0';
            t->key_type = key;

            const char *vstart = colon + 1;
            size_t vlen = strlen(vstart);
            if (vlen > 0 && vstart[vlen - 1] == ']') vlen--;
            char *val = malloc(vlen + 1);
            if (!val) { free(key); return &TYPE_UNKNOWN; }
            memcpy(val, vstart, vlen);
            val[vlen] = '\0';
            t->value_type = val;
        }
        return t;
    }

    /* Qualified type: module.Type → convert to module_Type */
    const char *dot = strchr(name, '.');
    if (dot) {
        const char *base = dot + 1;
        if (base[0] >= 'A' && base[0] <= 'Z') {
            /* Build prefixed name: mod.Type → mod_Type */
            size_t prefix_len = (size_t)(dot - name);
            size_t base_len = strlen(base);
            char *prefixed = malloc(prefix_len + 1 + base_len + 1);
            memcpy(prefixed, name, prefix_len);
            prefixed[prefix_len] = '_';
            memcpy(prefixed + prefix_len + 1, base, base_len + 1);
            return type_struct(prefixed);
        }
    }

    /* Uppercase = struct type, or module-prefixed: mod_Name */
    if (name[0] >= 'A' && name[0] <= 'Z') {
        return type_struct(name);
    }
    {
        const char *us = strchr(name, '_');
        if (us && us[1] >= 'A' && us[1] <= 'Z') {
            return type_struct(name);
        }
    }

    return &TYPE_UNKNOWN;
}
