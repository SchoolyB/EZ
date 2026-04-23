/*
 * types.h - Type representation for the EZ type system
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZC_TYPES_H
#define EZC_TYPES_H

#include <stdbool.h>

typedef enum {
    TK_VOID,
    TK_INT,
    TK_UINT,    /* unsigned integer types: uint, u8, u16, u32, u64, u128 */
    TK_FLOAT,
    TK_BOOL,
    TK_CHAR,
    TK_BYTE,
    TK_STRING,
    TK_ARRAY,
    TK_MAP,
    TK_STRUCT,
    TK_ENUM,
    TK_POINTER,
    TK_ERROR,
    TK_FUNCTION,
    TK_NIL,
    TK_UNKNOWN,
} TypeKind;

/* Signature payload for TK_FUNCTION (typed function references).
 * param_types[i] is the EZ type-name string for parameter i (no `&` prefix);
 * param_mutable[i] tracks the `&` flag separately. return_count==0 means
 * void. The strings are owned elsewhere (parser arena). */
typedef struct {
    int param_count;
    const char **param_types;
    bool *param_mutable;
    int return_count;
    const char **return_types;
} EzFuncSig;

typedef struct EzType {
    TypeKind kind;
    const char *name;           /* "int", "string", "Person", "[int]", etc. */
    const char *element_type;   /* For arrays: element type name */
    const char *key_type;       /* For maps: key type name */
    const char *value_type;     /* For maps: value type name */
    EzFuncSig *func_sig;        /* For TK_FUNCTION: parsed signature */
} EzType;

/* Built-in type singletons */
extern EzType TYPE_VOID;
extern EzType TYPE_INT;
extern EzType TYPE_UINT;
extern EzType TYPE_FLOAT;
extern EzType TYPE_BOOL;
extern EzType TYPE_CHAR;
extern EzType TYPE_BYTE;
extern EzType TYPE_STRING;
extern EzType TYPE_NIL;
extern EzType TYPE_UNKNOWN;

/* Type constructors */
EzType *type_array(const char *elem_type);
EzType *type_struct(const char *name);
EzType *type_enum(const char *name);
EzType *type_pointer(const char *pointee_type);

/* Allocate a new type (from internal pool) */
EzType *type_alloc(void);

/* Type queries */
bool type_is_numeric(EzType *t);
bool type_is_integer(EzType *t);
bool type_eq(EzType *a, EzType *b);
const char *type_name(EzType *t);

/* Resolve a type name string to an EzType */
EzType *type_from_name(const char *name);

#endif
