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

/* --- Type Table --- */

TypeTable *typetable_create(void) {
    TypeTable *tt = calloc(1, sizeof(TypeTable));
    return tt;
}

void typetable_set(TypeTable *tt, AstNode *node, EzType *type) {
    /* Check if already set */
    for (int i = 0; i < tt->count; i++) {
        if (tt->nodes[i] == node) {
            tt->types[i] = type;
            return;
        }
    }
    if (tt->count >= tt->cap) {
        tt->cap = tt->cap ? tt->cap * 2 : 64;
        tt->nodes = realloc(tt->nodes, sizeof(AstNode *) * tt->cap);
        tt->types = realloc(tt->types, sizeof(EzType *) * tt->cap);
    }
    tt->nodes[tt->count] = node;
    tt->types[tt->count] = type;
    tt->count++;
}

EzType *typetable_get(TypeTable *tt, AstNode *node) {
    if (!tt) return NULL;
    for (int i = 0; i < tt->count; i++) {
        if (tt->nodes[i] == node) return tt->types[i];
    }
    return NULL;
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
}

static FuncSig *find_func(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->func_count; i++) {
        if (strcmp(tc->funcs[i].name, name) == 0)
            return &tc->funcs[i];
    }
    return NULL;
}

/* --- Enum helpers --- */

static void register_enum(TypeChecker *tc, const char *name) {
    if (tc->enum_count >= tc->enum_cap) {
        tc->enum_cap = tc->enum_cap ? tc->enum_cap * 2 : 8;
        tc->enum_names = realloc(tc->enum_names, sizeof(const char *) * tc->enum_cap);
    }
    tc->enum_names[tc->enum_count++] = name;
}

static bool is_enum_name(TypeChecker *tc, const char *name) {
    for (int i = 0; i < tc->enum_count; i++) {
        if (strcmp(tc->enum_names[i], name) == 0) return true;
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
        Symbol *sym = scope_lookup(tc->current_scope, node->data.label.value);
        if (sym) {
            result = sym->type;
        }
        break;
    }

    case NODE_PREFIX_EXPR: {
        EzType *right = resolve_expr(tc, node->data.prefix.right);
        if (strcmp(node->data.prefix.op, "!") == 0) {
            result = &TYPE_BOOL;
        } else if (strcmp(node->data.prefix.op, "-") == 0) {
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
        if (strcmp(node->data.postfix.op, "^") == 0 && left_t->kind == TK_POINTER) {
            /* Dereference: ^T^ → T */
            result = type_from_name(left_t->element_type);
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
            if (strcmp(mod, "mem") == 0) {
                if (strcmp(mfn, "arena") == 0) {
                    result = &TYPE_UNKNOWN; /* arena pointer — opaque */
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
                    strcmp(mfn, "ends_with") == 0 || strcmp(mfn, "is_empty") == 0) {
                    result = &TYPE_BOOL;
                } else if (strcmp(mfn, "index") == 0 || strcmp(mfn, "count") == 0 ||
                           strcmp(mfn, "to_int") == 0) {
                    result = &TYPE_INT;
                } else if (strcmp(mfn, "to_float") == 0) {
                    result = &TYPE_FLOAT;
                } else if (strcmp(mfn, "split") == 0) {
                    result = type_array("string");
                } else {
                    result = &TYPE_STRING;
                }
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
            } else {
                result = &TYPE_VOID;
            }
            break;
        }

        if (fn_name) {
            /* Check built-in functions first */
            if ((strcmp(fn_name, "addr") == 0 || strcmp(fn_name, "ref") == 0) && node->data.call.arg_count == 1) {
                EzType *arg_t = resolve_expr(tc, node->data.call.args[0]);
                result = type_pointer(type_name(arg_t));
            } else if (strcmp(fn_name, "len") == 0 || strcmp(fn_name, "to_int") == 0) {
                result = &TYPE_INT;
            } else if (strcmp(fn_name, "to_float") == 0) {
                result = &TYPE_FLOAT;
            } else if (strcmp(fn_name, "to_string") == 0 || strcmp(fn_name, "typeof") == 0 ||
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
            } else {
                FuncSig *sig = find_func(tc, fn_name);
                if (sig && sig->return_count > 0) {
                    result = sig->return_types[0];
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
                result = &TYPE_INT;
                break;
            }

            /* Otherwise it's a struct field or multi-return access */
            Symbol *sym = scope_lookup(tc->current_scope, obj_name);
            if (sym && sym->type->kind == TK_STRUCT) {
                result = struct_field_type(tc, sym->type->name, member);
            } else if (sym && member[0] == 'v' && member[1] >= '0' && member[1] <= '9') {
                /* Multi-return .v0/.v1 access — .v0 returns the primary type */
                if (member[1] == '0') {
                    result = sym->type;
                } else {
                    result = type_from_name("Error");
                }
            } else if (sym && sym->type->kind == TK_POINTER) {
                /* Pointer auto-deref field access */
                result = struct_field_type(tc, sym->type->element_type, member);
            }
        }
        break;
    }

    case NODE_INDEX_EXPR: {
        EzType *left = resolve_expr(tc, node->data.index_expr.left);
        resolve_expr(tc, node->data.index_expr.index);
        if (left->kind == TK_ARRAY && left->element_type) {
            result = type_from_name(left->element_type);
        } else if (left->kind == TK_MAP && left->value_type) {
            result = type_from_name(left->value_type);
        } else if (left->kind == TK_STRING) {
            result = &TYPE_CHAR;
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

    case NODE_STRUCT_VALUE:
        /* Resolve field value types */
        for (int i = 0; i < node->data.struct_value.count; i++) {
            resolve_expr(tc, node->data.struct_value.field_values[i]);
        }
        result = type_struct(node->data.struct_value.name);
        break;

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

    default:
        break;
    }

    typetable_set(tc->type_table, node, result);
    return result;
}

/* --- Statement checking --- */

static void check_statement(TypeChecker *tc, AstNode *node);

static void check_block(TypeChecker *tc, AstNode *node) {
    if (!node || node->kind != NODE_BLOCK_STMT) return;
    for (int i = 0; i < node->data.block.count; i++) {
        check_statement(tc, node->data.block.stmts[i]);
    }
}

static void check_statement(TypeChecker *tc, AstNode *node) {
    if (!node) return;

    switch (node->kind) {
    case NODE_VAR_DECL: {
        EzType *declared = node->data.var_decl.type_name
            ? type_from_name(node->data.var_decl.type_name)
            : &TYPE_UNKNOWN;

        if (node->data.var_decl.value) {
            EzType *value_type = resolve_expr(tc, node->data.var_decl.value);
            /* If no declared type, infer from value */
            if (declared->kind == TK_UNKNOWN) {
                declared = value_type;
            }
        }

        if (strcmp(node->data.var_decl.name, "_") != 0) {
            scope_define(tc->current_scope, node->data.var_decl.name,
                declared, node->data.var_decl.mutable);

            /* Mark as transparent ref if assigned from ref() */
            if (node->data.var_decl.value &&
                node->data.var_decl.value->kind == NODE_CALL_EXPR) {
                AstNode *fn = node->data.var_decl.value->data.call.function;
                if (fn->kind == NODE_LABEL && strcmp(fn->data.label.value, "ref") == 0) {
                    Symbol *sym = scope_lookup_local(tc->current_scope,
                        node->data.var_decl.name);
                    if (sym) sym->is_ref = true;
                }
            }
        }
        break;
    }

    case NODE_ASSIGN_STMT:
        resolve_expr(tc, node->data.assign.target);
        resolve_expr(tc, node->data.assign.value);
        break;

    case NODE_RETURN_STMT:
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            resolve_expr(tc, node->data.return_stmt.values[i]);
        }
        break;

    case NODE_EXPR_STMT:
        resolve_expr(tc, node->data.expr_stmt.expr);
        break;

    case NODE_BLOCK_STMT:
        /* Inline blocks (from multi-var expansion) share parent scope.
         * Only control flow blocks (if, for, etc.) create new scopes,
         * and those are handled by their own cases. */
        check_block(tc, node);
        break;

    case NODE_IF_STMT:
        resolve_expr(tc, node->data.if_stmt.condition);
        check_block(tc, node->data.if_stmt.consequence);
        if (node->data.if_stmt.alternative) {
            check_statement(tc, node->data.if_stmt.alternative);
        }
        break;

    case NODE_FOR_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;
        scope_define(loop_scope, node->data.for_stmt.var_name, &TYPE_INT, false);
        resolve_expr(tc, node->data.for_stmt.iterable);
        check_block(tc, node->data.for_stmt.body);
        tc->current_scope = outer;
        break;
    }

    case NODE_FOR_EACH_STMT: {
        Scope *loop_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = loop_scope;

        /* Resolve collection type to determine element type */
        EzType *coll_t = resolve_expr(tc, node->data.for_each.collection);
        EzType *elem_t = &TYPE_UNKNOWN;
        if (coll_t->kind == TK_ARRAY && coll_t->element_type) {
            elem_t = type_from_name(coll_t->element_type);
        }

        /* Define iteration variables */
        if (node->data.for_each.index_name) {
            scope_define(loop_scope, node->data.for_each.index_name, &TYPE_INT, false);
        }
        scope_define(loop_scope, node->data.for_each.var_name, elem_t, false);

        check_block(tc, node->data.for_each.body);
        tc->current_scope = outer;
        break;
    }

    case NODE_WHILE_STMT:
        resolve_expr(tc, node->data.while_stmt.condition);
        check_block(tc, node->data.while_stmt.body);
        break;

    case NODE_LOOP_STMT:
        check_block(tc, node->data.loop_stmt.body);
        break;

    case NODE_FUNC_DECL: {
        Scope *func_scope = scope_create(tc->current_scope);
        Scope *outer = tc->current_scope;
        tc->current_scope = func_scope;

        /* Define parameters in function scope */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            Param *p = &node->data.func_decl.params[i];
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

        check_block(tc, node->data.func_decl.body);
        tc->current_scope = outer;
        break;
    }

    case NODE_ENSURE_STMT:
        resolve_expr(tc, node->data.ensure_stmt.expr);
        break;

    case NODE_WHEN_STMT:
        resolve_expr(tc, node->data.when_stmt.value);
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            for (int j = 0; j < node->data.when_stmt.cases[i].value_count; j++) {
                resolve_expr(tc, node->data.when_stmt.cases[i].values[j]);
            }
            check_block(tc, node->data.when_stmt.cases[i].body);
        }
        if (node->data.when_stmt.default_body) {
            check_block(tc, node->data.when_stmt.default_body);
        }
        break;

    default:
        break;
    }
}

/* --- Registration pass --- */

static void register_declarations(TypeChecker *tc, AstNode *program) {
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];

        if (stmt->kind == NODE_STRUCT_DECL) {
            int fc = stmt->data.struct_decl.field_count;
            const char **fnames = malloc(sizeof(const char *) * fc);
            EzType **ftypes = malloc(sizeof(EzType *) * fc);
            for (int j = 0; j < fc; j++) {
                fnames[j] = stmt->data.struct_decl.fields[j].name;
                ftypes[j] = type_from_name(stmt->data.struct_decl.fields[j].type_name);
            }
            register_struct(tc, stmt->data.struct_decl.name, fnames, ftypes, fc);
        }

        if (stmt->kind == NODE_ENUM_DECL) {
            register_enum(tc, stmt->data.enum_decl.name);
        }

        if (stmt->kind == NODE_FUNC_DECL) {
            int pc = stmt->data.func_decl.param_count;
            EzType **ptypes = malloc(sizeof(EzType *) * (pc ? pc : 1));
            for (int j = 0; j < pc; j++) {
                ptypes[j] = type_from_name(stmt->data.func_decl.params[j].type_name);
            }

            int rc = stmt->data.func_decl.return_type_count;
            EzType **rtypes = malloc(sizeof(EzType *) * (rc ? rc : 1));
            for (int j = 0; j < rc; j++) {
                rtypes[j] = type_from_name(stmt->data.func_decl.return_types[j]);
            }

            register_func(tc, stmt->data.func_decl.name, ptypes, pc, rtypes, rc);
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

    /* Pass 1: register all type/function declarations */
    register_declarations(tc, program);

    /* Pass 2: check all statements */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        check_statement(tc, program->data.program.stmts[i]);
    }
}

TypeTable *typechecker_get_table(TypeChecker *tc) {
    return tc->type_table;
}
