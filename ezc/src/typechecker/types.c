/*
 * types.c - Type representation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "types.h"
#include <string.h>
#include <stdlib.h>

/* Built-in type singletons */
EzType TYPE_VOID    = {TK_VOID,   "void",   NULL, NULL, NULL};
EzType TYPE_INT     = {TK_INT,    "int",    NULL, NULL, NULL};
EzType TYPE_FLOAT   = {TK_FLOAT,  "float",  NULL, NULL, NULL};
EzType TYPE_BOOL    = {TK_BOOL,   "bool",   NULL, NULL, NULL};
EzType TYPE_CHAR    = {TK_CHAR,   "char",   NULL, NULL, NULL};
EzType TYPE_BYTE    = {TK_BYTE,   "byte",   NULL, NULL, NULL};
EzType TYPE_STRING  = {TK_STRING, "string", NULL, NULL, NULL};
EzType TYPE_NIL     = {TK_NIL,    "nil",    NULL, NULL, NULL};
EzType TYPE_UNKNOWN = {TK_UNKNOWN,"unknown",NULL, NULL, NULL};

/* Pool for dynamically created types (leak on exit, fine for a compiler) */
static EzType type_pool[256];
static int type_pool_count = 0;

EzType *type_alloc(void) {
    if (type_pool_count >= 256) return &TYPE_UNKNOWN;
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

bool type_is_numeric(EzType *t) {
    return t->kind == TK_INT || t->kind == TK_FLOAT ||
           t->kind == TK_CHAR || t->kind == TK_BYTE;
}

bool type_is_integer(EzType *t) {
    return t->kind == TK_INT || t->kind == TK_CHAR || t->kind == TK_BYTE;
}

bool type_eq(EzType *a, EzType *b) {
    if (a->kind != b->kind) return false;
    if (a->kind == TK_STRUCT || a->kind == TK_ENUM) {
        return strcmp(a->name, b->name) == 0;
    }
    if (a->kind == TK_ARRAY) {
        return strcmp(a->element_type, b->element_type) == 0;
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
    if (strcmp(name, "uint") == 0)   return &TYPE_INT;
    if (strcmp(name, "i8") == 0)     return &TYPE_INT;
    if (strcmp(name, "i16") == 0)    return &TYPE_INT;
    if (strcmp(name, "i32") == 0)    return &TYPE_INT;
    if (strcmp(name, "i64") == 0)    return &TYPE_INT;
    if (strcmp(name, "i128") == 0)   return &TYPE_INT;
    if (strcmp(name, "u8") == 0)     return &TYPE_INT;
    if (strcmp(name, "u16") == 0)    return &TYPE_INT;
    if (strcmp(name, "u32") == 0)    return &TYPE_INT;
    if (strcmp(name, "u64") == 0)    return &TYPE_INT;
    if (strcmp(name, "u128") == 0)   return &TYPE_INT;
    if (strcmp(name, "float") == 0)  return &TYPE_FLOAT;
    if (strcmp(name, "f32") == 0)    return &TYPE_FLOAT;
    if (strcmp(name, "f64") == 0)    return &TYPE_FLOAT;
    if (strcmp(name, "bool") == 0)   return &TYPE_BOOL;
    if (strcmp(name, "char") == 0)   return &TYPE_CHAR;
    if (strcmp(name, "byte") == 0)   return &TYPE_BYTE;
    if (strcmp(name, "string") == 0) return &TYPE_STRING;
    if (strcmp(name, "void") == 0)   return &TYPE_VOID;
    if (strcmp(name, "nil") == 0)    return &TYPE_NIL;

    /* Array type: [int], [string], etc. */
    if (name[0] == '[') {
        size_t len = strlen(name);
        if (len > 2 && name[len - 1] == ']') {
            char *elem = malloc(len - 1);
            memcpy(elem, name + 1, len - 2);
            elem[len - 2] = '\0';
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
            memcpy(key, start, klen);
            key[klen] = '\0';
            t->key_type = key;

            const char *vstart = colon + 1;
            size_t vlen = strlen(vstart);
            if (vlen > 0 && vstart[vlen - 1] == ']') vlen--;
            char *val = malloc(vlen + 1);
            memcpy(val, vstart, vlen);
            val[vlen] = '\0';
            t->value_type = val;
        }
        return t;
    }

    /* Uppercase = struct type */
    if (name[0] >= 'A' && name[0] <= 'Z') {
        return type_struct(name);
    }

    return &TYPE_UNKNOWN;
}
