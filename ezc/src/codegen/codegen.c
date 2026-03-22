/*
 * codegen.c - C code generator for the EZ language
 *
 * Walks the AST and emits C source code.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "codegen.h"
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>

/* Forward declarations */
static void emit_statement(CodeGen *cg, AstNode *node);
static void emit_expression(CodeGen *cg, AstNode *node);
static void emit_call_expression(CodeGen *cg, AstNode *node);
static bool codegen_is_enum(CodeGen *cg, const char *name);

/* --- Helpers --- */

static void emit(CodeGen *cg, const char *s) {
    buf_append(&cg->output, s);
}

static void emitf(CodeGen *cg, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);

    int needed = vsnprintf(NULL, 0, fmt, args);
    va_end(args);

    if (needed < 0) return;

    char *tmp = malloc((size_t)needed + 1);
    va_start(args, fmt);
    vsnprintf(tmp, (size_t)needed + 1, fmt, args);
    va_end(args);

    buf_append(&cg->output, tmp);
    free(tmp);
}

static void emit_indent(CodeGen *cg) {
    buf_append_indent(&cg->output, cg->indent);
}

/* Map EZ type name to C type */
static const char *ez_type_to_c_cg(CodeGen *cg, const char *type_name) {
    if (!type_name) return "int64_t";

    if (strcmp(type_name, "int") == 0)    return "int64_t";
    if (strcmp(type_name, "uint") == 0)   return "uint64_t";
    if (strcmp(type_name, "i8") == 0)     return "int8_t";
    if (strcmp(type_name, "i16") == 0)    return "int16_t";
    if (strcmp(type_name, "i32") == 0)    return "int32_t";
    if (strcmp(type_name, "i64") == 0)    return "int64_t";
    if (strcmp(type_name, "u8") == 0)     return "uint8_t";
    if (strcmp(type_name, "u16") == 0)    return "uint16_t";
    if (strcmp(type_name, "u32") == 0)    return "uint32_t";
    if (strcmp(type_name, "u64") == 0)    return "uint64_t";
    if (strcmp(type_name, "i128") == 0)   return "__int128";
    if (strcmp(type_name, "u128") == 0)   return "unsigned __int128";
    if (strcmp(type_name, "float") == 0)  return "double";
    if (strcmp(type_name, "f32") == 0)    return "float";
    if (strcmp(type_name, "f64") == 0)    return "double";
    if (strcmp(type_name, "bool") == 0)   return "bool";
    if (strcmp(type_name, "char") == 0)   return "int32_t";
    if (strcmp(type_name, "byte") == 0)   return "uint8_t";
    if (strcmp(type_name, "string") == 0) return "EzString";

    /* Pointer type: ^T — use C pointer */
    if (type_name[0] == '^') {
        static char ptrbuf[256];
        const char *pointee = ez_type_to_c_cg(cg, type_name + 1);
        snprintf(ptrbuf, sizeof(ptrbuf), "%s *", pointee);
        return ptrbuf;
    }

    /* Array type: [T] — use EzArray */
    if (type_name[0] == '[') {
        return "EzArray";
    }

    /* Map type: map[K:V] — use EzMap */
    if (strncmp(type_name, "map[", 4) == 0) {
        return "EzMap";
    }

    /* If starts with uppercase, it's a user-defined type */
    if (type_name[0] >= 'A' && type_name[0] <= 'Z') {
        static char buf[256];
        if (cg && codegen_is_enum(cg, type_name)) {
            snprintf(buf, sizeof(buf), "EzEnum_%s", type_name);
        } else {
            snprintf(buf, sizeof(buf), "EzStruct_%s", type_name);
        }
        return buf;
    }

    return type_name;
}

/* Check if a variable name is a mutable parameter in the current function */
static bool is_mutable_param(CodeGen *cg, const char *name) {
    if (!cg->current_func) return false;
    for (int i = 0; i < cg->current_func->data.func_decl.param_count; i++) {
        Param *p = &cg->current_func->data.func_decl.params[i];
        if (p->mutable && strcmp(p->name, name) == 0) return true;
    }
    return false;
}

/* Find a function declaration by name */
static AstNode *find_func(CodeGen *cg, const char *name) {
    for (int i = 0; i < cg->func_count; i++) {
        if (strcmp(cg->all_funcs[i]->data.func_decl.name, name) == 0) {
            return cg->all_funcs[i];
        }
    }
    return NULL;
}

/* --- Expression Emission --- */

static void emit_expression(CodeGen *cg, AstNode *node) {
    if (!node) return;

    switch (node->kind) {
    case NODE_LABEL:
        if (is_mutable_param(cg, node->data.label.value)) {
            emitf(cg, "(*%s)", node->data.label.value);
        } else {
            emit(cg, node->data.label.value);
        }
        break;

    case NODE_INT_VALUE:
        emitf(cg, "%lld", (long long)node->data.int_value.value);
        break;

    case NODE_FLOAT_VALUE:
        emitf(cg, "%g", node->data.float_value.value);
        break;

    case NODE_STRING_VALUE:
        emitf(cg, "ez_string_lit(\"%s\")", node->data.string_value.value);
        break;

    case NODE_BOOL_VALUE:
        emit(cg, node->data.bool_value.value ? "true" : "false");
        break;

    case NODE_CHAR_VALUE: {
        char c = node->data.char_value.value;
        if (c == '\n') emit(cg, "'\\n'");
        else if (c == '\t') emit(cg, "'\\t'");
        else if (c == '\r') emit(cg, "'\\r'");
        else if (c == '\\') emit(cg, "'\\\\'");
        else if (c == '\'') emit(cg, "'\\''");
        else if (c == '\0') emit(cg, "'\\0'");
        else emitf(cg, "'%c'", c);
        break;
    }

    case NODE_NIL_VALUE:
        emit(cg, "NULL");
        break;

    case NODE_INTERPOLATED_STRING: {
        emit(cg, "ez_string_format(ez_default_arena, \"");
        /* First pass: emit format string */
        for (int i = 0; i < node->data.interpolated_string.part_count; i++) {
            AstNode *part = node->data.interpolated_string.parts[i];
            if (part->kind == NODE_STRING_VALUE) {
                const char *s = part->data.string_value.value;
                while (*s) {
                    if (*s == '%') buf_append(&cg->output, "%%");
                    else buf_append_char(&cg->output, *s);
                    s++;
                }
            } else {
                /* Use type table to determine format specifier */
                EzType *t = cg->type_table ? typetable_get(cg->type_table, part) : NULL;
                TypeKind tk = t ? t->kind : TK_UNKNOWN;

                /* Fall back to AST-based inference if no type info */
                if (tk == TK_UNKNOWN) {
                    if (part->kind == NODE_FLOAT_VALUE) tk = TK_FLOAT;
                    else if (part->kind == NODE_BOOL_VALUE) tk = TK_BOOL;
                    else if (part->kind == NODE_STRING_VALUE) tk = TK_STRING;
                    else tk = TK_INT;
                }

                switch (tk) {
                case TK_STRING: emit(cg, "%s"); break;
                case TK_FLOAT:  emit(cg, "%g"); break;
                case TK_BOOL:   emit(cg, "%s"); break;
                case TK_CHAR:   emit(cg, "%c"); break;
                case TK_ARRAY:  emit(cg, "%s"); break;
                case TK_MAP:    emit(cg, "%s"); break;
                case TK_ENUM:   emit(cg, "%lld"); break;
                default:        emit(cg, "%lld"); break;
                }
            }
        }
        emit(cg, "\"");
        /* Second pass: emit arguments */
        for (int i = 0; i < node->data.interpolated_string.part_count; i++) {
            AstNode *part = node->data.interpolated_string.parts[i];
            if (part->kind == NODE_STRING_VALUE) continue;
            emit(cg, ", ");

            EzType *t = cg->type_table ? typetable_get(cg->type_table, part) : NULL;
            TypeKind tk = t ? t->kind : TK_UNKNOWN;
            if (tk == TK_UNKNOWN) {
                if (part->kind == NODE_FLOAT_VALUE) tk = TK_FLOAT;
                else if (part->kind == NODE_BOOL_VALUE) tk = TK_BOOL;
                else if (part->kind == NODE_STRING_VALUE) tk = TK_STRING;
                else tk = TK_INT;
            }

            switch (tk) {
            case TK_STRING:
                emit_expression(cg, part);
                emit(cg, ".data");
                break;
            case TK_BOOL:
                emit(cg, "(");
                emit_expression(cg, part);
                emit(cg, ") ? \"true\" : \"false\"");
                break;
            case TK_FLOAT:
                emit(cg, "(double)(");
                emit_expression(cg, part);
                emit(cg, ")");
                break;
            case TK_CHAR:
                emit(cg, "(char)(");
                emit_expression(cg, part);
                emit(cg, ")");
                break;
            case TK_ARRAY:
                emit(cg, "\"[...]\"");
                break;
            case TK_MAP:
                emit(cg, "\"{...}\"");
                break;
            default:
                emit(cg, "(long long)(");
                emit_expression(cg, part);
                emit(cg, ")");
                break;
            }
        }
        emit(cg, ")");
        break;
    }

    case NODE_ARRAY_VALUE: {
        /* Array literal: emit as EzArray using ez_array_from */
        int count = node->data.array_value.count;
        if (count == 0) {
            emit(cg, "ez_array_new(ez_default_arena, sizeof(int64_t), 4)");
            break;
        }
        /* Determine element type from first element */
        EzType *elem_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.array_value.elements[0])
            : NULL;
        TypeKind tk = elem_t ? elem_t->kind : TK_INT;

        const char *c_type;
        switch (tk) {
        case TK_FLOAT:  c_type = "double"; break;
        case TK_BOOL:   c_type = "bool"; break;
        case TK_STRING: c_type = "EzString"; break;
        default:        c_type = "int64_t"; break;
        }

        emitf(cg, "ez_array_from(ez_default_arena, (%s[]){", c_type);
        for (int i = 0; i < count; i++) {
            if (i > 0) emit(cg, ", ");
            emit_expression(cg, node->data.array_value.elements[i]);
        }
        emitf(cg, "}, sizeof(%s), %d)", c_type, count);
        break;
    }

    case NODE_MAP_VALUE: {
        /* Map literal: emit inline construction with ez_map_set calls.
         * We need a temp variable, so wrap in a GCC statement expression. */
        int count = node->data.map_value.count;

        /* Determine key/value C types from first pair */
        const char *c_key_type = "EzString";
        const char *c_val_type = "int64_t";
        if (count > 0) {
            EzType *kt = cg->type_table ? typetable_get(cg->type_table, node->data.map_value.keys[0]) : NULL;
            EzType *vt = cg->type_table ? typetable_get(cg->type_table, node->data.map_value.values[0]) : NULL;
            if (kt && kt->kind == TK_INT) c_key_type = "int64_t";
            if (vt && vt->kind == TK_FLOAT) c_val_type = "double";
            else if (vt && vt->kind == TK_STRING) c_val_type = "EzString";
            else if (vt && vt->kind == TK_BOOL) c_val_type = "bool";
        }

        /* Use GCC statement expression: ({ EzMap m = ...; ez_map_set(...); m; }) */
        static int map_lit_counter = 0;
        emitf(cg, "({ EzMap _ml%d = ez_map_new(ez_default_arena, sizeof(%s), sizeof(%s), %d); ",
            map_lit_counter, c_key_type, c_val_type, count > 4 ? count * 2 : 8);
        for (int i = 0; i < count; i++) {
            emitf(cg, "{ %s _mk = ", c_key_type);
            emit_expression(cg, node->data.map_value.keys[i]);
            emitf(cg, "; %s _mv = ", c_val_type);
            emit_expression(cg, node->data.map_value.values[i]);
            emitf(cg, "; ez_map_set(ez_default_arena, &_ml%d, &_mk, &_mv); } ", map_lit_counter);
        }
        emitf(cg, "_ml%d; })", map_lit_counter);
        map_lit_counter++;
        break;
    }

    case NODE_STRUCT_VALUE:
        /* Struct literal: (EzStruct_Name){.field = value, ...} */
        emitf(cg, "(EzStruct_%s){", node->data.struct_value.name);
        for (int i = 0; i < node->data.struct_value.count; i++) {
            if (i > 0) emit(cg, ", ");
            emitf(cg, ".%s = ", node->data.struct_value.field_names[i]);
            emit_expression(cg, node->data.struct_value.field_values[i]);
        }
        emit(cg, "}");
        break;

    case NODE_PREFIX_EXPR:
        emit(cg, "(");
        emit(cg, node->data.prefix.op);
        emit_expression(cg, node->data.prefix.right);
        emit(cg, ")");
        break;

    case NODE_INFIX_EXPR: {
        /* Only wrap in parens if the parent is also an expression.
         * Skip parens for top-level comparisons to avoid ((x == y)) warnings. */
        const char *op = node->data.infix.op;
        bool needs_parens = true;
        /* Check if both children are simple (no further nesting needed) */
        NodeKind lk = node->data.infix.left->kind;
        NodeKind rk = node->data.infix.right->kind;
        bool l_simple = (lk == NODE_LABEL || lk == NODE_INT_VALUE ||
                        lk == NODE_FLOAT_VALUE || lk == NODE_MEMBER_EXPR ||
                        lk == NODE_CALL_EXPR || lk == NODE_BOOL_VALUE);
        bool r_simple = (rk == NODE_LABEL || rk == NODE_INT_VALUE ||
                        rk == NODE_FLOAT_VALUE || rk == NODE_MEMBER_EXPR ||
                        rk == NODE_CALL_EXPR || rk == NODE_BOOL_VALUE);
        if (l_simple && r_simple) {
            needs_parens = false;
        }
        if (needs_parens) emit(cg, "(");
        emit_expression(cg, node->data.infix.left);
        emitf(cg, " %s ", op);
        emit_expression(cg, node->data.infix.right);
        if (needs_parens) emit(cg, ")");
        break;
    }

    case NODE_POSTFIX_EXPR:
        if (strcmp(node->data.postfix.op, "^") == 0) {
            /* Pointer dereference: p^ → (*p) */
            emit(cg, "(*");
            emit_expression(cg, node->data.postfix.left);
            emit(cg, ")");
        } else {
            emit_expression(cg, node->data.postfix.left);
            emit(cg, node->data.postfix.op);
        }
        break;

    case NODE_CALL_EXPR:
        emit_call_expression(cg, node);
        break;

    case NODE_MEMBER_EXPR:
        /* Check if this is an enum access: EnumName.VALUE */
        if (node->data.member.object->kind == NODE_LABEL) {
            const char *obj_name = node->data.member.object->data.label.value;
            if (obj_name[0] >= 'A' && obj_name[0] <= 'Z') {
                /* Could be enum access — emit as EzEnum_Name_VALUE */
                emitf(cg, "EzEnum_%s_%s", obj_name, node->data.member.member);
                break;
            }
        }
        /* Check if object is a pointer type — use -> instead of . */
        {
            EzType *obj_t = cg->type_table
                ? typetable_get(cg->type_table, node->data.member.object)
                : NULL;
            emit_expression(cg, node->data.member.object);
            if (obj_t && obj_t->kind == TK_POINTER) {
                emitf(cg, "->%s", node->data.member.member);
            } else {
                emitf(cg, ".%s", node->data.member.member);
            }
        }
        break;

    case NODE_INDEX_EXPR: {
        /* Check if left side is an array (EzArray) or string */
        EzType *left_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.index_expr.left)
            : NULL;
        if (left_t && left_t->kind == TK_ARRAY) {
            /* Determine element C type */
            const char *c_elem = "int64_t";
            if (left_t->element_type) {
                EzType *et = type_from_name(left_t->element_type);
                if (et->kind == TK_FLOAT) c_elem = "double";
                else if (et->kind == TK_BOOL) c_elem = "bool";
                else if (et->kind == TK_STRING) c_elem = "EzString";
                else if (et->kind == TK_CHAR) c_elem = "int32_t";
                else if (et->kind == TK_BYTE) c_elem = "uint8_t";
            }
            emitf(cg, "EZ_ARRAY_GET(");
            emit_expression(cg, node->data.index_expr.left);
            emitf(cg, ", %s, ", c_elem);
            emit_expression(cg, node->data.index_expr.index);
            emit(cg, ")");
        } else if (left_t && left_t->kind == TK_MAP) {
            /* Map key access: *(type *)ez_map_get(&m, &key) */
            const char *c_val = "int64_t";
            if (left_t->value_type) {
                EzType *vt = type_from_name(left_t->value_type);
                if (vt->kind == TK_FLOAT) c_val = "double";
                else if (vt->kind == TK_STRING) c_val = "EzString";
                else if (vt->kind == TK_BOOL) c_val = "bool";
            }
            emitf(cg, "(*(%s *)ez_map_get(&", c_val);
            emit_expression(cg, node->data.index_expr.left);
            emit(cg, ", &(");
            emit_expression(cg, node->data.index_expr.index);
            emit(cg, ")))");
        } else {
            /* String indexing or fallback */
            emit_expression(cg, node->data.index_expr.left);
            emit(cg, "[");
            emit_expression(cg, node->data.index_expr.index);
            emit(cg, "]");
        }
        break;
    }

    case NODE_CAST_EXPR:
        /* cast(value, type) → (C_type)(value) */
        emitf(cg, "((%s)(", ez_type_to_c_cg(cg, node->data.cast.target_type));
        emit_expression(cg, node->data.cast.value);
        emit(cg, "))");
        break;

    case NODE_NEW_EXPR: {
        /* new(Type) → zeroed allocation on default arena, returns pointer */
        const char *c_type = ez_type_to_c_cg(cg, node->data.new_expr.type_name);
        emitf(cg, "((%s *)ez_arena_alloc(ez_default_arena, sizeof(%s)))", c_type, c_type);
        break;
    }

    default:
        emitf(cg, "/* TODO: expression kind %d */", node->kind);
        break;
    }
}

/* Check if a call is a stdlib call like std.println or just println (with using) */
static bool is_stdlib_call(AstNode *node, const char **module, const char **func) {
    if (node->data.call.function->kind == NODE_MEMBER_EXPR) {
        AstNode *obj = node->data.call.function->data.member.object;
        if (obj->kind == NODE_LABEL) {
            *module = obj->data.label.value;
            *func = node->data.call.function->data.member.member;
            return true;
        }
    } else if (node->data.call.function->kind == NODE_LABEL) {
        /* Direct call like println() via using */
        *module = NULL;
        *func = node->data.call.function->data.label.value;
        return true;
    }
    return false;
}

static void emit_call_expression(CodeGen *cg, AstNode *node) {
    const char *module = NULL;
    const char *func = NULL;

    if (is_stdlib_call(node, &module, &func)) {
        /* Handle known stdlib functions */
        if (strcmp(func, "println") == 0) {
            if (node->data.call.arg_count == 0) {
                emit(cg, "putchar('\\n')");
            } else {
                AstNode *arg = node->data.call.args[0];
                const char *suffix = "_int"; /* default */
                /* Use type table if available */
                EzType *arg_type = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                if (arg_type) {
                    switch (arg_type->kind) {
                    case TK_STRING: suffix = "_str"; break;
                    case TK_FLOAT:  suffix = "_float"; break;
                    case TK_BOOL:   suffix = "_bool"; break;
                    default: suffix = "_int"; break;
                    }
                } else {
                    /* Fallback to AST-based inference */
                    if (arg->kind == NODE_STRING_VALUE ||
                        arg->kind == NODE_INTERPOLATED_STRING) suffix = "_str";
                    else if (arg->kind == NODE_FLOAT_VALUE) suffix = "_float";
                    else if (arg->kind == NODE_BOOL_VALUE) suffix = "_bool";
                }
                emitf(cg, "ez_std_println%s(", suffix);
                emit_expression(cg, arg);
                emit(cg, ")");
            }
            return;
        }

        if (strcmp(func, "len") == 0 && node->data.call.arg_count == 1) {
            AstNode *arg = node->data.call.args[0];
            EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            if (t && t->kind == TK_STRING) {
                emit(cg, "(int64_t)(");
                emit_expression(cg, arg);
                emit(cg, ").len");
            } else if (t && t->kind == TK_ARRAY) {
                emit(cg, "(int64_t)(");
                emit_expression(cg, arg);
                emit(cg, ").len");
            } else if (t && t->kind == TK_MAP) {
                emit(cg, "(int64_t)(");
                emit_expression(cg, arg);
                emit(cg, ").count");
            } else {
                /* Default: try .len for EzArray */
                emit(cg, "(int64_t)(");
                emit_expression(cg, arg);
                emit(cg, ").len");
            }
            return;
        }

        if (strcmp(func, "typeof") == 0 && node->data.call.arg_count == 1) {
            AstNode *arg = node->data.call.args[0];
            EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            const char *tn = t ? type_name(t) : "unknown";
            emitf(cg, "ez_string_lit(\"%s\")", tn);
            return;
        }

        if (strcmp(func, "to_int") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "(int64_t)(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "to_float") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "(double)(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "addr") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "&");
            emit_expression(cg, node->data.call.args[0]);
            return;
        }

        if (strcmp(func, "exit") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_exit(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "panic") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_panic_msg(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "assert") == 0 && node->data.call.arg_count >= 1) {
            emit(cg, "ez_std_assert(");
            emit_expression(cg, node->data.call.args[0]);
            if (node->data.call.arg_count >= 2) {
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[1]);
            } else {
                emit(cg, ", ez_string_lit(\"assertion failed\")");
            }
            emitf(cg, ", \"%s\", %d)", cg->file, node->token.line);
            return;
        }

        if (strcmp(func, "error") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_error(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "input") == 0) {
            emit(cg, "ez_std_input(ez_default_arena)");
            return;
        }

        if (strcmp(func, "read_int") == 0) {
            /* read_int() → reads line from stdin, converts to int */
            emit(cg, "({ EzString _s = ez_std_input(ez_default_arena); atoll(_s.data); })");
            return;
        }

        if (strcmp(func, "eprintln") == 0) {
            if (node->data.call.arg_count == 0) {
                emit(cg, "fputc('\\n', stderr)");
            } else {
                AstNode *arg = node->data.call.args[0];
                EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                const char *suffix = "_int";
                if (at && at->kind == TK_STRING) suffix = "_str";
                else if (arg->kind == NODE_STRING_VALUE || arg->kind == NODE_INTERPOLATED_STRING) suffix = "_str";
                emitf(cg, "ez_std_eprintln%s(", suffix);
                emit_expression(cg, arg);
                emit(cg, ")");
            }
            return;
        }

        if (strcmp(func, "eprint") == 0 && node->data.call.arg_count > 0) {
            emit(cg, "ez_std_eprint_str(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "to_string") == 0 && node->data.call.arg_count == 1) {
            AstNode *arg = node->data.call.args[0];
            EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            if (at && at->kind == TK_FLOAT) {
                emit(cg, "ez_std_to_string_float(ez_default_arena, ");
            } else if (at && at->kind == TK_BOOL) {
                emit(cg, "ez_std_to_string_bool(ez_default_arena, ");
            } else {
                emit(cg, "ez_std_to_string_int(ez_default_arena, ");
            }
            emit_expression(cg, arg);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "to_bool") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "(bool)(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "sleep_seconds") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_sleep_seconds(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "sleep_milliseconds") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_sleep_milliseconds(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        if (strcmp(func, "sleep_nanoseconds") == 0 && node->data.call.arg_count == 1) {
            emit(cg, "ez_std_sleep_nanoseconds(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return;
        }

        /* Sized type conversion functions: i8(), u16(), f32(), etc. */
        if (node->data.call.arg_count == 1) {
            const char *cast_type = NULL;
            if (strcmp(func, "i8") == 0) cast_type = "int8_t";
            else if (strcmp(func, "i16") == 0) cast_type = "int16_t";
            else if (strcmp(func, "i32") == 0) cast_type = "int32_t";
            else if (strcmp(func, "i64") == 0) cast_type = "int64_t";
            else if (strcmp(func, "u8") == 0) cast_type = "uint8_t";
            else if (strcmp(func, "u16") == 0) cast_type = "uint16_t";
            else if (strcmp(func, "u32") == 0) cast_type = "uint32_t";
            else if (strcmp(func, "u64") == 0) cast_type = "uint64_t";
            else if (strcmp(func, "f32") == 0) cast_type = "float";
            else if (strcmp(func, "f64") == 0) cast_type = "double";
            else if (strcmp(func, "int") == 0) cast_type = "int64_t";
            else if (strcmp(func, "uint") == 0) cast_type = "uint64_t";
            else if (strcmp(func, "float") == 0) cast_type = "double";
            else if (strcmp(func, "string") == 0) cast_type = NULL; /* handled below */
            else if (strcmp(func, "char") == 0) cast_type = "int32_t";
            else if (strcmp(func, "byte") == 0) cast_type = "uint8_t";
            if (cast_type) {
                emitf(cg, "((%s)(", cast_type);
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, "))");
                return;
            }
            /* string() conversion */
            if (strcmp(func, "string") == 0) {
                AstNode *arg = node->data.call.args[0];
                EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                if (at && at->kind == TK_FLOAT) {
                    emit(cg, "ez_std_to_string_float(ez_default_arena, ");
                } else if (at && at->kind == TK_BOOL) {
                    emit(cg, "ez_std_to_string_bool(ez_default_arena, ");
                } else {
                    emit(cg, "ez_std_to_string_int(ez_default_arena, ");
                }
                emit_expression(cg, arg);
                emit(cg, ")");
                return;
            }
        }

        if (strcmp(func, "copy") == 0 && node->data.call.arg_count == 1) {
            /* Deep copy — for arrays use ez_array_copy, otherwise value copy */
            AstNode *arg = node->data.call.args[0];
            EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            if (at && at->kind == TK_ARRAY) {
                emit(cg, "ez_array_copy(ez_default_arena, &");
                emit_expression(cg, arg);
                emit(cg, ")");
            } else {
                /* Value types — just copy the value */
                emit_expression(cg, arg);
            }
            return;
        }

        if (strcmp(func, "print") == 0 && node->data.call.arg_count > 0) {
            AstNode *arg = node->data.call.args[0];
            const char *suffix = "_int";
            if (arg->kind == NODE_STRING_VALUE) suffix = "_str";
            emitf(cg, "ez_std_print%s(", suffix);
            emit_expression(cg, arg);
            emit(cg, ")");
            return;
        }

        /* @mem module functions */
        if (module && strcmp(module, "mem") == 0) {
            if (strcmp(func, "arena") == 0 && node->data.call.arg_count == 1) {
                emit(cg, "ez_mem_arena(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "destroy") == 0 && node->data.call.arg_count == 1) {
                emit(cg, "ez_mem_destroy(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "reset") == 0 && node->data.call.arg_count == 1) {
                emit(cg, "ez_mem_reset(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "usage") == 0 && node->data.call.arg_count == 1) {
                emit(cg, "ez_mem_usage(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "copy") == 0 && node->data.call.arg_count == 3) {
                emit(cg, "ez_mem_copy(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[1]);
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[2]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "zero") == 0 && node->data.call.arg_count == 2) {
                emit(cg, "ez_mem_zero(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[1]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "set") == 0 && node->data.call.arg_count == 3) {
                emit(cg, "ez_mem_set(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[1]);
                emit(cg, ", ");
                emit_expression(cg, node->data.call.args[2]);
                emit(cg, ")");
                return;
            }
            if (strcmp(func, "size_of") == 0 && node->data.call.arg_count == 1) {
                /* mem.size_of(Type) → sizeof(C_type) */
                AstNode *type_arg = node->data.call.args[0];
                if (type_arg->kind == NODE_LABEL) {
                    emitf(cg, "(int64_t)sizeof(%s)", ez_type_to_c_cg(cg, type_arg->data.label.value));
                } else {
                    emit(cg, "0");
                }
                return;
            }
            if (strcmp(func, "new") == 0 && node->data.call.arg_count == 2) {
                /* mem.new(arena, Type) — allocate zeroed T, return ^T */
                AstNode *arena_arg = node->data.call.args[0];
                AstNode *type_arg = node->data.call.args[1];
                /* Type arg should be a label (type name) */
                const char *type_name = "int64_t";
                if (type_arg->kind == NODE_LABEL) {
                    type_name = ez_type_to_c_cg(cg, type_arg->data.label.value);
                }
                emitf(cg, "(%s *)ez_arena_alloc(", type_name);
                emit_expression(cg, arena_arg);
                emitf(cg, ", sizeof(%s))", type_name);
                return;
            }
            if (strcmp(func, "alloc") == 0 && node->data.call.arg_count == 2) {
                /* mem.alloc(arena, value) — allocate value on specific arena.
                 * For strings: ez_string_new(arena, src.data, src.len)
                 * For arrays: emit array literal with arena instead of ez_default_arena
                 * For other: ez_arena_alloc + memcpy */
                AstNode *arena_arg = node->data.call.args[0];
                AstNode *value_arg = node->data.call.args[1];
                EzType *vt = cg->type_table ? typetable_get(cg->type_table, value_arg) : NULL;

                if (vt && vt->kind == TK_STRING) {
                    emit(cg, "({ EzString _s = ");
                    emit_expression(cg, value_arg);
                    emit(cg, "; ez_string_new(");
                    emit_expression(cg, arena_arg);
                    emit(cg, ", _s.data, _s.len); })");
                } else if (value_arg->kind == NODE_ARRAY_VALUE) {
                    /* Emit array literal with custom arena */
                    int count = value_arg->data.array_value.count;
                    EzType *elem_t = (count > 0 && cg->type_table)
                        ? typetable_get(cg->type_table, value_arg->data.array_value.elements[0])
                        : NULL;
                    const char *c_type = "int64_t";
                    if (elem_t) {
                        if (elem_t->kind == TK_FLOAT) c_type = "double";
                        else if (elem_t->kind == TK_STRING) c_type = "EzString";
                        else if (elem_t->kind == TK_BOOL) c_type = "bool";
                    }
                    emitf(cg, "ez_array_from(");
                    emit_expression(cg, arena_arg);
                    emitf(cg, ", (%s[]){", c_type);
                    for (int i = 0; i < count; i++) {
                        if (i > 0) emit(cg, ", ");
                        emit_expression(cg, value_arg->data.array_value.elements[i]);
                    }
                    emitf(cg, "}, sizeof(%s), %d)", c_type, count);
                } else {
                    /* Generic: copy value onto arena */
                    emit(cg, "({ __auto_type _v = ");
                    emit_expression(cg, value_arg);
                    emit(cg, "; __auto_type _p = ez_arena_alloc(");
                    emit_expression(cg, arena_arg);
                    emit(cg, ", sizeof(_v)); *(__typeof__(_v) *)_p = _v; _v; })");
                }
                return;
            }
        }

        /* @fmt module functions */
        if (module && strcmp(module, "fmt") == 0) {
            if (strcmp(func, "printf") == 0 && node->data.call.arg_count >= 1) {
                /* fmt.printf(format, args...) → printf(format, args...) */
                emit(cg, "printf(");
                /* First arg is the format string — emit as C string */
                AstNode *fmt_arg = node->data.call.args[0];
                if (fmt_arg->kind == NODE_STRING_VALUE) {
                    emitf(cg, "\"%s\"", fmt_arg->data.string_value.value);
                } else {
                    emit_expression(cg, fmt_arg);
                    emit(cg, ".data");
                }
                for (int i = 1; i < node->data.call.arg_count; i++) {
                    emit(cg, ", ");
                    AstNode *arg = node->data.call.args[i];
                    EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                    if (at && at->kind == TK_STRING) {
                        emit_expression(cg, arg);
                        emit(cg, ".data");
                    } else if (at && at->kind == TK_BOOL) {
                        emit_expression(cg, arg);
                        emit(cg, " ? \"true\" : \"false\"");
                    } else {
                        emit_expression(cg, arg);
                    }
                }
                emit(cg, ")");
                return;
            }

            if (strcmp(func, "sprintf") == 0 && node->data.call.arg_count >= 2) {
                /* fmt.sprintf(arena, format, args...) → ez_string_format(arena, format, args...) */
                emit(cg, "ez_string_format(");
                emit_expression(cg, node->data.call.args[0]); /* arena */
                emit(cg, ", ");
                AstNode *fmt_arg = node->data.call.args[1];
                if (fmt_arg->kind == NODE_STRING_VALUE) {
                    emitf(cg, "\"%s\"", fmt_arg->data.string_value.value);
                } else {
                    emit_expression(cg, fmt_arg);
                    emit(cg, ".data");
                }
                for (int i = 2; i < node->data.call.arg_count; i++) {
                    emit(cg, ", ");
                    AstNode *arg = node->data.call.args[i];
                    EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                    if (at && at->kind == TK_STRING) {
                        emit_expression(cg, arg);
                        emit(cg, ".data");
                    } else {
                        emit_expression(cg, arg);
                    }
                }
                emit(cg, ")");
                return;
            }

            if (strcmp(func, "eprintln") == 0) {
                if (node->data.call.arg_count == 0) {
                    emit(cg, "fputc('\\n', stderr)");
                } else {
                    AstNode *arg = node->data.call.args[0];
                    EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
                    const char *suffix = "_int";
                    if (at) {
                        switch (at->kind) {
                        case TK_STRING: suffix = "_str"; break;
                        case TK_FLOAT: suffix = "_float"; break;
                        case TK_BOOL: suffix = "_bool"; break;
                        default: break;
                        }
                    } else if (arg->kind == NODE_STRING_VALUE || arg->kind == NODE_INTERPOLATED_STRING) {
                        suffix = "_str";
                    }
                    emitf(cg, "ez_fmt_eprintln%s(", suffix);
                    emit_expression(cg, arg);
                    emit(cg, ")");
                }
                return;
            }

            if (strcmp(func, "eprint") == 0 && node->data.call.arg_count > 0) {
                emit(cg, "ez_fmt_eprint_str(");
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return;
            }
        }
    }

    /* Generic function call */
    const char *fn_name = NULL;
    if (node->data.call.function->kind == NODE_LABEL) {
        fn_name = node->data.call.function->data.label.value;
    }

    emit(cg, "ez_fn_");
    if (fn_name) {
        emit(cg, fn_name);
    } else {
        emit_expression(cg, node->data.call.function);
    }

    /* Look up function to check for mutable params */
    AstNode *target_func = fn_name ? find_func(cg, fn_name) : NULL;

    emit(cg, "(");
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        /* Pass address for mutable parameters */
        bool needs_addr = false;
        if (target_func && i < target_func->data.func_decl.param_count) {
            needs_addr = target_func->data.func_decl.params[i].mutable;
        }
        if (needs_addr && node->data.call.args[i]->kind == NODE_LABEL) {
            emitf(cg, "&%s", node->data.call.args[i]->data.label.value);
        } else {
            emit_expression(cg, node->data.call.args[i]);
        }
    }
    emit(cg, ")");
}

/* --- Statement Emission --- */

static const char *extract_array_elem_type(const char *type_name) {
    if (!type_name || type_name[0] != '[') return NULL;
    /* Extract element type from "[int]" -> "int" */
    static char buf[128];
    size_t len = strlen(type_name);
    if (len < 3) return NULL;
    memcpy(buf, type_name + 1, len - 2);
    buf[len - 2] = '\0';
    return buf;
}

static void emit_var_declaration(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    const char *type_name = node->data.var_decl.type_name;
    const char *elem_type = extract_array_elem_type(type_name);

    if (elem_type) {
        /* Array declaration: use EzArray */
        emitf(cg, "EzArray %s = ", node->data.var_decl.name);
        if (node->data.var_decl.value) {
            emit_expression(cg, node->data.var_decl.value);
        } else {
            const char *c_elem_type = ez_type_to_c_cg(cg, elem_type);
            emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", c_elem_type);
        }
        emit(cg, ";\n");
        return;
    }

    /* Map type: map[K:V] */
    if (type_name && strncmp(type_name, "map[", 4) == 0) {
        /* Parse K:V from type string to determine C types */
        EzType *mt = type_from_name(type_name);
        const char *c_kt = "EzString";
        const char *c_vt = "int64_t";
        if (mt && mt->key_type) {
            EzType *kt = type_from_name(mt->key_type);
            if (kt->kind == TK_INT) c_kt = "int64_t";
        }
        if (mt && mt->value_type) {
            EzType *vt = type_from_name(mt->value_type);
            if (vt->kind == TK_FLOAT) c_vt = "double";
            else if (vt->kind == TK_STRING) c_vt = "EzString";
            else if (vt->kind == TK_BOOL) c_vt = "bool";
        }

        emitf(cg, "EzMap %s = ", node->data.var_decl.name);
        if (node->data.var_decl.value &&
            node->data.var_decl.value->kind == NODE_MAP_VALUE) {
            emit_expression(cg, node->data.var_decl.value);
        } else {
            /* Empty map or no initializer */
            emitf(cg, "ez_map_new(ez_default_arena, sizeof(%s), sizeof(%s), 8)", c_kt, c_vt);
        }
        emit(cg, ";\n");
        return;
    }

    const char *c_type = ez_type_to_c_cg(cg, type_name);

    /* Skip blank identifiers (_) */
    if (strcmp(node->data.var_decl.name, "_") == 0) {
        if (node->data.var_decl.value) {
            emit_indent(cg);
            emit(cg, "(void)(");
            emit_expression(cg, node->data.var_decl.value);
            emit(cg, ");\n");
        }
        return;
    }

    /* If no type annotation, try to infer from value */
    if (!type_name && node->data.var_decl.value) {
        AstNode *val = node->data.var_decl.value;
        if (val->kind == NODE_STRING_VALUE || val->kind == NODE_INTERPOLATED_STRING) {
            c_type = "EzString";
        } else if (val->kind == NODE_FLOAT_VALUE) {
            c_type = "double";
        } else if (val->kind == NODE_BOOL_VALUE) {
            c_type = "bool";
        } else if (val->kind == NODE_CALL_EXPR || val->kind == NODE_NEW_EXPR) {
            /* Use __auto_type for function calls and new() to infer types */
            c_type = "__auto_type";
        }
    }

    if (!node->data.var_decl.mutable) {
        emit(cg, "const ");
    }

    emitf(cg, "%s %s", c_type, node->data.var_decl.name);

    if (node->data.var_decl.value) {
        emit(cg, " = ");
        emit_expression(cg, node->data.var_decl.value);
    }

    emit(cg, ";\n");
}

static void emit_assign_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    /* Check for array index assignment: arr[i] = value */
    if (node->data.assign.target->kind == NODE_INDEX_EXPR) {
        AstNode *left = node->data.assign.target->data.index_expr.left;
        EzType *left_t = cg->type_table ? typetable_get(cg->type_table, left) : NULL;
        if (left_t && left_t->kind == TK_ARRAY) {
            const char *c_elem = "int64_t";
            if (left_t->element_type) {
                EzType *et = type_from_name(left_t->element_type);
                if (et->kind == TK_FLOAT) c_elem = "double";
                else if (et->kind == TK_BOOL) c_elem = "bool";
                else if (et->kind == TK_STRING) c_elem = "EzString";
            }
            emitf(cg, "EZ_ARRAY_SET(");
            emit_expression(cg, left);
            emitf(cg, ", %s, ", c_elem);
            emit_expression(cg, node->data.assign.target->data.index_expr.index);
            emit(cg, ", ");
            emit_expression(cg, node->data.assign.value);
            emit(cg, ");\n");
            return;
        }
        if (left_t && left_t->kind == TK_MAP) {
            /* Map key assignment: ez_map_set(arena, &m, &key, &value) */
            const char *c_val = "int64_t";
            if (left_t->value_type) {
                EzType *vt = type_from_name(left_t->value_type);
                if (vt->kind == TK_FLOAT) c_val = "double";
                else if (vt->kind == TK_STRING) c_val = "EzString";
                else if (vt->kind == TK_BOOL) c_val = "bool";
            }
            emitf(cg, "{ %s _mv = ", c_val);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; ez_map_set(ez_default_arena, &");
            emit_expression(cg, left);
            emit(cg, ", &(");
            emit_expression(cg, node->data.assign.target->data.index_expr.index);
            emit(cg, "), &_mv); }\n");
            return;
        }
    }

    /* Default assignment */
    emit_expression(cg, node->data.assign.target);
    emitf(cg, " %s ", node->data.assign.op);
    emit_expression(cg, node->data.assign.value);
    emit(cg, ";\n");
}

/* Collect ensure statements from a block */
static void collect_ensures(AstNode *block, AstNode **ensures, int *count, int cap) {
    if (!block || block->kind != NODE_BLOCK_STMT) return;
    for (int i = 0; i < block->data.block.count; i++) {
        AstNode *stmt = block->data.block.stmts[i];
        if (stmt->kind == NODE_ENSURE_STMT && *count < cap) {
            ensures[(*count)++] = stmt;
        }
    }
}

/* Emit ensure cleanup calls in LIFO order */
static void emit_ensure_cleanup(CodeGen *cg) {
    if (!cg->current_func || !cg->current_func->data.func_decl.body) return;

    AstNode *ensures[32];
    int ensure_count = 0;
    collect_ensures(cg->current_func->data.func_decl.body, ensures, &ensure_count, 32);

    /* Emit in reverse (LIFO) order */
    for (int i = ensure_count - 1; i >= 0; i--) {
        emit_indent(cg);
        emit_expression(cg, ensures[i]->data.ensure_stmt.expr);
        emit(cg, ";\n");
    }
}

static void emit_return_statement(CodeGen *cg, AstNode *node) {
    /* Emit ensure cleanup before return */
    emit_ensure_cleanup(cg);

    emit_indent(cg);

    if (node->data.return_stmt.count > 1 && cg->current_func) {
        emitf(cg, "return (EzMulti_%s){", cg->current_func->data.func_decl.name);
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            if (i > 0) emit(cg, ", ");
            emit_expression(cg, node->data.return_stmt.values[i]);
        }
        emit(cg, "};\n");
    } else {
        emit(cg, "return");
        if (node->data.return_stmt.count > 0) {
            emit(cg, " ");
            emit_expression(cg, node->data.return_stmt.values[0]);
        }
        emit(cg, ";\n");
    }
}

static void emit_block(CodeGen *cg, AstNode *node) {
    for (int i = 0; i < node->data.block.count; i++) {
        emit_statement(cg, node->data.block.stmts[i]);
    }
}

static void emit_if_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit(cg, "if (");
    emit_expression(cg, node->data.if_stmt.condition);
    emit(cg, ") {\n");

    cg->indent++;
    emit_block(cg, node->data.if_stmt.consequence);
    cg->indent--;

    if (node->data.if_stmt.alternative) {
        if (node->data.if_stmt.alternative->kind == NODE_IF_STMT) {
            /* or (else if) - emit inline */
            emit_indent(cg);
            emit(cg, "} else if (");
            emit_expression(cg, node->data.if_stmt.alternative->data.if_stmt.condition);
            emit(cg, ") {\n");
            cg->indent++;
            emit_block(cg, node->data.if_stmt.alternative->data.if_stmt.consequence);
            cg->indent--;
            /* Recurse for further chains */
            AstNode *alt = node->data.if_stmt.alternative->data.if_stmt.alternative;
            while (alt) {
                if (alt->kind == NODE_IF_STMT) {
                    emit_indent(cg);
                    emit(cg, "} else if (");
                    emit_expression(cg, alt->data.if_stmt.condition);
                    emit(cg, ") {\n");
                    cg->indent++;
                    emit_block(cg, alt->data.if_stmt.consequence);
                    cg->indent--;
                    alt = alt->data.if_stmt.alternative;
                } else {
                    /* otherwise (else) block */
                    emit_indent(cg);
                    emit(cg, "} else {\n");
                    cg->indent++;
                    emit_block(cg, alt);
                    cg->indent--;
                    break;
                }
            }
            emit_indent(cg);
            emit(cg, "}\n");
        } else {
            /* otherwise (else) block */
            emit_indent(cg);
            emit(cg, "} else {\n");
            cg->indent++;
            emit_block(cg, node->data.if_stmt.alternative);
            cg->indent--;
            emit_indent(cg);
            emit(cg, "}\n");
        }
    } else {
        emit_indent(cg);
        emit(cg, "}\n");
    }
}

static void emit_for_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    AstNode *iter = node->data.for_stmt.iterable;
    if (iter && iter->kind == NODE_RANGE_EXPR) {
        /* for i in range(start, end) or range(start, end, step) */
        const char *var = node->data.for_stmt.var_name;

        if (iter->data.range_expr.start) {
            /* range(start, end) or range(start, end, step) */
            emitf(cg, "for (int64_t %s = ", var);
            emit_expression(cg, iter->data.range_expr.start);
            emitf(cg, "; %s < ", var);
            emit_expression(cg, iter->data.range_expr.end);
            emitf(cg, "; %s", var);
            if (iter->data.range_expr.step) {
                emit(cg, " += ");
                emit_expression(cg, iter->data.range_expr.step);
            } else {
                emit(cg, "++");
            }
        } else {
            /* range(end) - start at 0 */
            emitf(cg, "for (int64_t %s = 0; %s < ", var, var);
            emit_expression(cg, iter->data.range_expr.end);
            emitf(cg, "; %s++", var);
        }

        emit(cg, ") {\n");
    } else {
        /* Generic for - fallback */
        emitf(cg, "/* TODO: non-range for loop */\n");
        emit_indent(cg);
        emit(cg, "{\n");
    }

    cg->indent++;
    emit_block(cg, node->data.for_stmt.body);
    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
}

static void emit_while_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit(cg, "while (");
    emit_expression(cg, node->data.while_stmt.condition);
    emit(cg, ") {\n");

    cg->indent++;
    emit_block(cg, node->data.while_stmt.body);
    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
}

static void emit_loop_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit(cg, "for (;;) {\n");

    cg->indent++;
    emit_block(cg, node->data.loop_stmt.body);
    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
}

/* Build a multi-return type name like EzMulti_add */
static void emit_multi_return_typedef(CodeGen *cg, AstNode *node) {
    emitf(cg, "typedef struct {\n");
    for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
        emitf(cg, "    %s v%d;\n",
            ez_type_to_c_cg(cg, node->data.func_decl.return_types[i]), i);
    }
    emitf(cg, "} EzMulti_%s;\n\n", node->data.func_decl.name);
}

static const char *func_return_type(CodeGen *cg, AstNode *node) {
    if (node->data.func_decl.return_type_count == 0) return "void";
    if (node->data.func_decl.return_type_count == 1) {
        return ez_type_to_c_cg(cg, node->data.func_decl.return_types[0]);
    }
    static char buf[256];
    snprintf(buf, sizeof(buf), "EzMulti_%s", node->data.func_decl.name);
    return buf;
}

static void emit_func_declaration(CodeGen *cg, AstNode *node, bool is_main) {
    /* Return type */
    if (is_main) {
        emit(cg, "static void ez_fn_main(void)");
    } else {
        emitf(cg, "static %s ", func_return_type(cg, node));
        emitf(cg, "ez_fn_%s(", node->data.func_decl.name);

        /* Parameters */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            if (i > 0) emit(cg, ", ");
            Param *param = &node->data.func_decl.params[i];
            if (param->mutable) {
                emitf(cg, "%s *%s", ez_type_to_c_cg(cg,param->type_name), param->name);
            } else {
                emitf(cg, "%s %s", ez_type_to_c_cg(cg,param->type_name), param->name);
            }
        }

        if (node->data.func_decl.param_count == 0) {
            emit(cg, "void");
        }
        emit(cg, ")");
    }

    emit(cg, " {\n");
    cg->indent++;
    AstNode *prev_func = cg->current_func;
    cg->current_func = node;

    /* Emit named return variable declarations */
    if (node->data.func_decl.return_names) {
        for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
            if (node->data.func_decl.return_names[i]) {
                emit_indent(cg);
                emitf(cg, "%s %s;\n",
                    ez_type_to_c_cg(cg, node->data.func_decl.return_types[i]),
                    node->data.func_decl.return_names[i]);
            }
        }
    }

    if (node->data.func_decl.body) {
        emit_block(cg, node->data.func_decl.body);
        /* Emit ensure cleanup at end of function (for implicit returns) */
        emit_ensure_cleanup(cg);
    }
    cg->current_func = prev_func;
    cg->indent--;
    emit(cg, "}\n\n");
}

static void emit_expression_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit_expression(cg, node->data.expr_stmt.expr);
    emit(cg, ";\n");
}

static void emit_statement(CodeGen *cg, AstNode *node) {
    if (!node) return;

    switch (node->kind) {
    case NODE_VAR_DECL:
        emit_var_declaration(cg, node);
        break;
    case NODE_ASSIGN_STMT:
        emit_assign_statement(cg, node);
        break;
    case NODE_RETURN_STMT:
        emit_return_statement(cg, node);
        break;
    case NODE_EXPR_STMT:
        emit_expression_statement(cg, node);
        break;
    case NODE_IF_STMT:
        emit_if_statement(cg, node);
        break;
    case NODE_FOR_STMT:
        emit_for_statement(cg, node);
        break;
    case NODE_FOR_EACH_STMT: {
        emit_indent(cg);
        AstNode *coll = node->data.for_each.collection;
        EzType *coll_t = cg->type_table ? typetable_get(cg->type_table, coll) : NULL;

        const char *idx_name = node->data.for_each.index_name;
        if (!idx_name) idx_name = "_ez_idx";

        if (coll_t && coll_t->kind == TK_STRING) {
            /* for_each ch in "string" → iterate characters */
            emitf(cg, "{ EzString _ez_str = ");
            emit_expression(cg, coll);
            emit(cg, ";\n");
            emit_indent(cg);
            emitf(cg, "for (int32_t %s = 0; %s < _ez_str.len; %s++) {\n", idx_name, idx_name, idx_name);
            cg->indent++;
            emit_indent(cg);
            emitf(cg, "int32_t %s = _ez_str.data[%s];\n", node->data.for_each.var_name, idx_name);
        } else {
            /* for_each item in array → iterate EzArray */
            const char *c_elem = "int64_t";
            if (coll_t && coll_t->kind == TK_ARRAY && coll_t->element_type) {
                EzType *et = type_from_name(coll_t->element_type);
                if (et->kind == TK_FLOAT) c_elem = "double";
                else if (et->kind == TK_BOOL) c_elem = "bool";
                else if (et->kind == TK_STRING) c_elem = "EzString";
            }

            emitf(cg, "for (int32_t %s = 0; %s < ", idx_name, idx_name);
            emit_expression(cg, coll);
            emitf(cg, ".len; %s++) {\n", idx_name);
            cg->indent++;
            emit_indent(cg);
            emitf(cg, "%s %s = EZ_ARRAY_GET(", c_elem, node->data.for_each.var_name);
            emit_expression(cg, coll);
            emitf(cg, ", %s, %s);\n", c_elem, idx_name);
        }

        emit_block(cg, node->data.for_each.body);
        cg->indent--;
        emit_indent(cg);
        emit(cg, "}\n");
        /* Close extra scope for string iteration */
        if (coll_t && coll_t->kind == TK_STRING) {
            emit_indent(cg);
            emit(cg, "}\n");
        }
        break;
    }
    case NODE_WHILE_STMT:
        emit_while_statement(cg, node);
        break;
    case NODE_LOOP_STMT:
        emit_loop_statement(cg, node);
        break;
    case NODE_BREAK_STMT:
        emit_indent(cg);
        emit(cg, "break;\n");
        break;
    case NODE_CONTINUE_STMT:
        emit_indent(cg);
        emit(cg, "continue;\n");
        break;
    case NODE_WHEN_STMT: {
        /* Emit as if-else chain for now (switch requires constant values) */
        AstNode *val = node->data.when_stmt.value;
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            WhenCase *wc = &node->data.when_stmt.cases[i];
            emit_indent(cg);
            if (i == 0) {
                emit(cg, "if (");
            } else {
                emit(cg, "} else if (");
            }
            for (int j = 0; j < wc->value_count; j++) {
                if (j > 0) emit(cg, " || ");
                if (wc->is_range && wc->values[j]->kind == NODE_RANGE_EXPR) {
                    AstNode *r = wc->values[j];
                    emit(cg, "(");
                    emit_expression(cg, val);
                    emit(cg, " >= ");
                    emit_expression(cg, r->data.range_expr.start);
                    emit(cg, " && ");
                    emit_expression(cg, val);
                    emit(cg, " < ");
                    emit_expression(cg, r->data.range_expr.end);
                    emit(cg, ")");
                } else {
                    emit_expression(cg, val);
                    emit(cg, " == ");
                    emit_expression(cg, wc->values[j]);
                }
            }
            emit(cg, ") {\n");
            cg->indent++;
            emit_block(cg, wc->body);
            cg->indent--;
        }
        if (node->data.when_stmt.default_body) {
            emit_indent(cg);
            if (node->data.when_stmt.case_count > 0) {
                emit(cg, "} else {\n");
            } else {
                emit(cg, "{\n");
            }
            cg->indent++;
            emit_block(cg, node->data.when_stmt.default_body);
            cg->indent--;
        }
        emit_indent(cg);
        emit(cg, "}\n");
        break;
    }
    case NODE_FUNC_DECL:
        emit_func_declaration(cg, node,
            strcmp(node->data.func_decl.name, "main") == 0);
        break;
    case NODE_BLOCK_STMT:
        /* Inline block (e.g., from multi-var declaration expansion) */
        emit_block(cg, node);
        break;
    case NODE_ENSURE_STMT:
        /* Ensure is collected and emitted at return/function-exit */
        break;
    case NODE_STRUCT_DECL:
        /* Struct declarations are emitted in the preamble */
        break;
    case NODE_ENUM_DECL:
        /* Enum declarations are emitted in the preamble */
        break;
    case NODE_IMPORT_STMT:
        /* Imports are handled during the preamble scan */
        break;
    case NODE_USING_STMT:
        /* Using is handled during the preamble scan */
        break;
    default:
        emit_indent(cg);
        emitf(cg, "/* TODO: statement kind %d */\n", node->kind);
        break;
    }
}

/* --- Public API --- */

static bool codegen_is_enum(CodeGen *cg, const char *name) {
    for (int i = 0; i < cg->enum_count; i++) {
        if (strcmp(cg->enum_names[i], name) == 0) return true;
    }
    return false;
}

CodeGen codegen_create(const char *file) {
    CodeGen cg;
    cg.output = buf_create(4096);
    cg.indent = 0;
    cg.has_std = false;
    cg.has_mem = false;
    cg.has_fmt = false;
    cg.file = file;
    cg.enum_names = NULL;
    cg.enum_count = 0;
    cg.enum_cap = 0;
    cg.current_func = NULL;
    cg.all_funcs = NULL;
    cg.func_count = 0;
    cg.func_cap = 0;
    cg.type_table = NULL;
    return cg;
}

void codegen_generate(CodeGen *cg, AstNode *program) {
    if (program->kind != NODE_PROGRAM) return;

    /* First pass: scan for imports */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_IMPORT_STMT) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->is_stdlib && item->module) {
                    if (strcmp(item->module, "std") == 0) cg->has_std = true;
                    if (strcmp(item->module, "mem") == 0) cg->has_mem = true;
                    if (strcmp(item->module, "fmt") == 0) cg->has_fmt = true;
                }
            }
        }
    }

    /* Emit preamble */
    emit(cg, "/* Generated by EZC */\n");
    emit(cg, "#include \"ez_runtime.h\"\n");
    emit(cg, "#include \"ez_array.h\"\n");
    emit(cg, "#include \"ez_map.h\"\n");
    if (cg->has_std) {
        emit(cg, "#include \"ez_std.h\"\n");
    }
    if (cg->has_mem) {
        emit(cg, "#include \"ez_mem.h\"\n");
    }
    if (cg->has_fmt) {
        emit(cg, "#include \"ez_fmt.h\"\n");
    }
    emit(cg, "\n");

    /* Emit struct type definitions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_STRUCT_DECL) {
            emitf(cg, "typedef struct {\n");
            for (int j = 0; j < stmt->data.struct_decl.field_count; j++) {
                StructField *f = &stmt->data.struct_decl.fields[j];
                emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg,f->type_name), f->name);
            }
            emitf(cg, "} EzStruct_%s;\n\n", stmt->data.struct_decl.name);
        }
    }

    /* Collect and emit enum type definitions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_ENUM_DECL) {
            /* Register enum name */
            if (cg->enum_count >= cg->enum_cap) {
                cg->enum_cap = cg->enum_cap ? cg->enum_cap * 2 : 8;
                const char **new_names = realloc(cg->enum_names, sizeof(const char *) * cg->enum_cap);
                cg->enum_names = new_names;
            }
            cg->enum_names[cg->enum_count++] = stmt->data.enum_decl.name;
            emitf(cg, "typedef enum {\n");
            for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                EnumVal *ev = &stmt->data.enum_decl.values[j];
                emitf(cg, "    EzEnum_%s_%s", stmt->data.enum_decl.name, ev->name);
                if (ev->value) {
                    emit(cg, " = ");
                    emit_expression(cg, ev->value);
                } else {
                    emitf(cg, " = %d", j);
                }
                emit(cg, ",\n");
            }
            emitf(cg, "} EzEnum_%s;\n\n", stmt->data.enum_decl.name);
        }
    }

    /* Collect all function declarations */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL) {
            if (cg->func_count >= cg->func_cap) {
                cg->func_cap = cg->func_cap ? cg->func_cap * 2 : 16;
                cg->all_funcs = realloc(cg->all_funcs, sizeof(AstNode *) * cg->func_cap);
            }
            cg->all_funcs[cg->func_count++] = stmt;
        }
    }

    /* Emit multi-return type definitions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL &&
            stmt->data.func_decl.return_type_count > 1) {
            emit_multi_return_typedef(cg, stmt);
        }
    }

    /* Emit forward declarations for user functions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL) {
            if (strcmp(stmt->data.func_decl.name, "main") == 0) {
                emit(cg, "static void ez_fn_main(void);\n");
            } else {
                emitf(cg, "static %s ", func_return_type(cg, stmt));
                emitf(cg, "ez_fn_%s(", stmt->data.func_decl.name);
                for (int j = 0; j < stmt->data.func_decl.param_count; j++) {
                    if (j > 0) emit(cg, ", ");
                    Param *param = &stmt->data.func_decl.params[j];
                    if (param->mutable) {
                        emitf(cg, "%s *", ez_type_to_c_cg(cg,param->type_name));
                    } else {
                        emit(cg, ez_type_to_c_cg(cg,param->type_name));
                    }
                }
                if (stmt->data.func_decl.param_count == 0) {
                    emit(cg, "void");
                }
                emit(cg, ");\n");
            }
        }
    }
    emit(cg, "\n");

    /* Emit function definitions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_IMPORT_STMT && stmt->kind != NODE_USING_STMT) {
            emit_statement(cg, stmt);
        }
    }

    /* Emit C main() */
    emit(cg, "int main(int argc, char **argv) {\n");
    emit(cg, "    (void)argc; (void)argv;\n");
    emit(cg, "    ez_runtime_init();\n");
    emit(cg, "    ez_fn_main();\n");
    emit(cg, "    ez_runtime_shutdown();\n");
    emit(cg, "    return 0;\n");
    emit(cg, "}\n");
}

const char *codegen_result(CodeGen *cg) {
    return buf_cstr(&cg->output);
}

void codegen_destroy(CodeGen *cg) {
    buf_destroy(&cg->output);
}
