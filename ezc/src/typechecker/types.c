/*
 * types.c - Type representation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "types.h"
#include "../util/constants.h"
#include "../util/xalloc.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

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

#define TYPE_POOL_CAPACITY 4096

/* Pool for dynamically created types (leak on exit, fine for a compiler) */
static EzType type_pool[TYPE_POOL_CAPACITY];
static int type_pool_count = 0;

EzType *type_alloc(void) {
    if (type_pool_count >= TYPE_POOL_CAPACITY) {
        fprintf(stderr, "error: type pool exhausted (%d types); please report this bug\n", TYPE_POOL_CAPACITY);
        exit(1);
    }
    return &type_pool[type_pool_count++];
}

/* Hash index for O(1) pool_find. Must be a power of 2 and at least 2x
 * TYPE_POOL_CAPACITY to keep load factor below 50%. */
#define TYPE_HASH_CAP 8192

typedef struct {
    TypeKind kind;
    const char *name;  /* points into the pool entry's name — not a copy */
    EzType    *type;   /* NULL = empty slot */
} TypeHashEntry;

static TypeHashEntry type_hash_table[TYPE_HASH_CAP];

static uint32_t type_hash(TypeKind kind, const char *name) {
    uint32_t h = 5381u ^ ((uint32_t)kind * 2654435761u);
    for (const unsigned char *p = (const unsigned char *)name; *p; p++)
        h = h * 33u ^ (uint32_t)*p;
    return h;
}

/* Return an existing pool entry matching kind+name, or NULL if not found. */
static EzType *pool_find(TypeKind kind, const char *name) {
    if (!name) return NULL;
    uint32_t mask = TYPE_HASH_CAP - 1;
    uint32_t idx  = type_hash(kind, name) & mask;
    for (;;) {
        TypeHashEntry *e = &type_hash_table[idx];
        if (!e->type) return NULL;
        if (e->kind == kind && strcmp(e->name, name) == 0) return e->type;
        idx = (idx + 1) & mask;
    }
}

/* Insert a newly created type into the hash index. */
static void pool_insert(TypeKind kind, const char *name, EzType *t) {
    uint32_t mask = TYPE_HASH_CAP - 1;
    uint32_t idx  = type_hash(kind, name) & mask;
    for (;;) {
        TypeHashEntry *e = &type_hash_table[idx];
        if (!e->type) {
            e->kind = kind;
            e->name = name;
            e->type = t;
            return;
        }
        idx = (idx + 1) & mask;
    }
}

EzType *type_array(const char *elem_type) {
    EzType *existing = pool_find(TK_ARRAY, elem_type);
    if (existing) return existing;
    EzType *t = type_alloc();
    t->kind = TK_ARRAY;
    t->element_type = elem_type;
    t->name = elem_type; /* simplified */
    pool_insert(TK_ARRAY, t->name, t);
    return t;
}

EzType *type_struct(const char *name) {
    EzType *existing = pool_find(TK_STRUCT, name);
    if (existing) return existing;
    EzType *t = type_alloc();
    t->kind = TK_STRUCT;
    t->name = strdup(name);
    pool_insert(TK_STRUCT, t->name, t);
    return t;
}

EzType *type_enum(const char *name) {
    EzType *existing = pool_find(TK_ENUM, name);
    if (existing) return existing;
    EzType *t = type_alloc();
    t->kind = TK_ENUM;
    t->name = strdup(name);
    pool_insert(TK_ENUM, t->name, t);
    return t;
}

EzType *type_pointer(const char *pointee_type) {
    EzType *existing = pool_find(TK_POINTER, pointee_type);
    if (existing) return existing;
    EzType *t = type_alloc();
    t->kind = TK_POINTER;
    const char *dup = strdup(pointee_type);
    t->element_type = dup;
    t->name = dup;
    pool_insert(TK_POINTER, t->name, t);
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
    char **arr = (char **)xmalloc(sizeof(char *) * (size_t)cap);
    int n = 0;
    int depth = 0;
    int seg_start = start;
    for (int i = start; i <= end; i++) {
        char c = (i < end) ? src[i] : ',';
        if (c == '(' || c == '[') depth++;
        else if (c == ')' || c == ']') depth--;
        if (depth == 0 && (c == ',' || i == end)) {
            int seg_len = i - seg_start;
            char *seg = (char *)xmalloc((size_t)seg_len + 1);
            memcpy(seg, src + seg_start, (size_t)seg_len);
            seg[seg_len] = '\0';
            if (n == cap) {
                cap *= 2;
                arr = (char **)xrealloc(arr, sizeof(char *) * (size_t)cap);
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
    EzFuncSig *sig = (EzFuncSig *)xmalloc(sizeof(EzFuncSig));
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
        sig->param_types = (const char **)xmalloc(sizeof(char *) * (size_t)raw_count);
        sig->param_mutable = (bool *)xmalloc(sizeof(bool) * (size_t)raw_count);
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
                sig->return_types = (const char **)xmalloc(sizeof(char *) * (size_t)rcount);
                for (int i = 0; i < rcount; i++) {
                    sig->return_types[i] = rets[i];
                }
            }
            free(rets);
        } else if (strcmp(ret, "void") != 0) {
            sig->return_count = 1;
            sig->return_types = (const char **)xmalloc(sizeof(char *));
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
    if (a->kind == TK_POINTER) {
        return a->name && b->name && strcmp(a->name, b->name) == 0;
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
    /* Pointer types store the bare pointee in t->name (and t->element_type).
     * Render them with the leading '^' so error messages match source syntax.
     * A small ring of static buffers lets callers chain type_name() calls
     * inside one snprintf() without clobbering the previous result. */
    if (t->kind == TK_POINTER && t->name) {
        static char bufs[4][EZ_TYPE_NAME_MAX];
        static int slot = 0;
        char *out = bufs[slot];
        slot = (slot + 1) & 3;
        snprintf(out, sizeof(bufs[0]), "^%s", t->name);
        return out;
    }
    if (t->kind == TK_ARRAY && t->element_type) {
        static char bufs[4][EZ_TYPE_NAME_MAX];
        static int slot = 0;
        char *out = bufs[slot];
        slot = (slot + 1) & 3;
        snprintf(out, sizeof(bufs[0]), "[%s]", t->element_type);
        return out;
    }
    if (t->kind == TK_MAP && t->key_type && t->value_type) {
        static char bufs[4][EZ_TYPE_NAME_MAX];
        static int slot = 0;
        char *out = bufs[slot];
        slot = (slot + 1) & 3;
        snprintf(out, sizeof(bufs[0]), "map[%s:%s]", t->key_type, t->value_type);
        return out;
    }
    return t->name;
}

/* Builtin type-name dispatch table. MUST stay sorted lexicographically by
 * name (uppercase precedes lowercase in ASCII), validated by bsearch. */
typedef struct {
    const char *name;
    EzType *singleton;   /* non-NULL: return this singleton directly */
    int alloc_kind;      /* used when singleton is NULL: pool-alloc with this kind */
    const char *alloc_name; /* if non-NULL, set t->name to this literal instead of strdup(name) */
} BuiltinTypeEntry;

static BuiltinTypeEntry builtin_types[] = {
    { "Error",  NULL,         TK_ERROR,   "Error" },
    { "bool",   &TYPE_BOOL,   0, NULL },
    { "byte",   &TYPE_BYTE,   0, NULL },
    { "char",   &TYPE_CHAR,   0, NULL },
    { "f32",    NULL,         TK_FLOAT,   NULL },
    { "f64",    NULL,         TK_FLOAT,   NULL },
    { "float",  &TYPE_FLOAT,  0, NULL },
    { "func",   NULL,         TK_UNKNOWN, "func" },
    { "i128",   NULL,         TK_INT,     NULL },
    { "i16",    NULL,         TK_INT,     NULL },
    { "i256",   NULL,         TK_INT,     NULL },
    { "i32",    NULL,         TK_INT,     NULL },
    { "i64",    NULL,         TK_INT,     NULL },
    { "i8",     NULL,         TK_INT,     NULL },
    { "int",    &TYPE_INT,    0, NULL },
    { "nil",    &TYPE_NIL,    0, NULL },
    { "string", &TYPE_STRING, 0, NULL },
    { "u128",   NULL,         TK_UINT,    NULL },
    { "u16",    NULL,         TK_UINT,    NULL },
    { "u256",   NULL,         TK_UINT,    NULL },
    { "u32",    NULL,         TK_UINT,    NULL },
    { "u64",    NULL,         TK_UINT,    NULL },
    { "u8",     NULL,         TK_UINT,    NULL },
    { "uint",   &TYPE_UINT,   0, NULL },
    { "void",   &TYPE_VOID,   0, NULL },
};

#define BUILTIN_TYPES_COUNT (sizeof(builtin_types) / sizeof(builtin_types[0]))

static int builtin_type_cmp(const void *a, const void *b) {
    return strcmp(((const BuiltinTypeEntry *)a)->name,
                  ((const BuiltinTypeEntry *)b)->name);
}

EzType *type_from_name(const char *name) {
    if (!name) return &TYPE_UNKNOWN;

    /* Typed function reference: "func(p1,&p2)->R" — checked before bsearch
     * so "func(...)" doesn't get conflated with the bare "func" entry. */
    if (strncmp(name, "func(", 5) == 0) {
        EzType *existing = pool_find(TK_FUNCTION, name);
        if (existing) return existing;
        EzType *t = type_alloc();
        t->kind = TK_FUNCTION;
        t->name = strdup(name);
        t->func_sig = parse_func_sig(name);
        pool_insert(TK_FUNCTION, t->name, t);
        return t;
    }

    BuiltinTypeEntry key = { name, NULL, 0, NULL };
    BuiltinTypeEntry *hit = bsearch(&key, builtin_types, BUILTIN_TYPES_COUNT,
                                    sizeof(BuiltinTypeEntry), builtin_type_cmp);
    if (hit) {
        if (hit->singleton) return hit->singleton;
        const char *resolved_name = hit->alloc_name ? hit->alloc_name : name;
        EzType *existing = pool_find(hit->alloc_kind, resolved_name);
        if (existing) return existing;
        EzType *t = type_alloc();
        t->kind = hit->alloc_kind;
        t->name = hit->alloc_name ? hit->alloc_name : strdup(name);
        pool_insert(t->kind, t->name, t);
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
            char *elem = xmalloc(len - 1);
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
        EzType *existing = pool_find(TK_MAP, name);
        if (existing) return existing;
        EzType *t = type_alloc();
        t->kind = TK_MAP;
        t->name = strdup(name);
        /* Parse key:value types from "map[string:int]" */
        const char *start = name + 4;
        const char *colon = strchr(start, ':');
        if (colon) {
            size_t klen = (size_t)(colon - start);
            char *key_str = xmalloc(klen + 1);
            memcpy(key_str, start, klen);
            key_str[klen] = '\0';
            t->key_type = key_str;

            const char *vstart = colon + 1;
            size_t vlen = strlen(vstart);
            if (vlen > 0 && vstart[vlen - 1] == ']') vlen--;
            char *val = xmalloc(vlen + 1);
            memcpy(val, vstart, vlen);
            val[vlen] = '\0';
            t->value_type = val;
        }
        pool_insert(TK_MAP, t->name, t);
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
            char *prefixed = xmalloc(prefix_len + 1 + base_len + 1);
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
