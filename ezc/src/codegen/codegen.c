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
#include <ctype.h>

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
    if (strcmp(type_name, "i128") == 0)   return "ez_i128";
    if (strcmp(type_name, "u128") == 0)   return "ez_u128";
    if (strcmp(type_name, "i256") == 0)   return "ez_i256";
    if (strcmp(type_name, "u256") == 0)   return "ez_u256";
    if (strcmp(type_name, "float") == 0)  return "double";
    if (strcmp(type_name, "f32") == 0)    return "float";
    if (strcmp(type_name, "f64") == 0)    return "double";
    if (strcmp(type_name, "bool") == 0)   return "bool";
    if (strcmp(type_name, "char") == 0)   return "int32_t";
    if (strcmp(type_name, "byte") == 0)   return "uint8_t";
    if (strcmp(type_name, "string") == 0) return "EzString";
    if (strcmp(type_name, "Error") == 0) return "EzError *";
    if (strcmp(type_name, "func") == 0)  return "void *"; /* generic fn ptr — cast at call site */

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
static bool is_ref_var(CodeGen *cg, const char *name) {
    for (int i = 0; i < cg->ref_var_count; i++) {
        if (strcmp(cg->ref_vars[i], name) == 0) return true;
    }
    return false;
}

static void register_ref_var(CodeGen *cg, const char *name) {
    if (cg->ref_var_count >= cg->ref_var_cap) {
        cg->ref_var_cap = cg->ref_var_cap ? cg->ref_var_cap * 2 : 8;
        cg->ref_vars = realloc(cg->ref_vars, sizeof(const char *) * cg->ref_var_cap);
    }
    cg->ref_vars[cg->ref_var_count++] = name;
}

/* Check if a type name is a wide integer type (i128/u128/i256/u256) */
static bool is_bigint_type(const char *tn) {
    if (!tn) return false;
    return strcmp(tn, "i128") == 0 || strcmp(tn, "u128") == 0 ||
           strcmp(tn, "i256") == 0 || strcmp(tn, "u256") == 0;
}

/* Register a bigint variable's declared type name */
static void register_bigint_var(CodeGen *cg, const char *name, const char *type_name) {
    if (cg->bigint_var_count >= cg->bigint_var_cap) {
        cg->bigint_var_cap = cg->bigint_var_cap ? cg->bigint_var_cap * 2 : 8;
        cg->bigint_var_names = realloc(cg->bigint_var_names, sizeof(const char *) * cg->bigint_var_cap);
        cg->bigint_var_types = realloc(cg->bigint_var_types, sizeof(const char *) * cg->bigint_var_cap);
    }
    cg->bigint_var_names[cg->bigint_var_count] = name;
    cg->bigint_var_types[cg->bigint_var_count] = type_name;
    cg->bigint_var_count++;
}

/* Look up a variable's bigint type name, or NULL if not bigint */
static const char *lookup_bigint_var(CodeGen *cg, const char *name) {
    for (int i = cg->bigint_var_count - 1; i >= 0; i--) {
        if (strcmp(cg->bigint_var_names[i], name) == 0) return cg->bigint_var_types[i];
    }
    return NULL;
}

/* Get the bigint type prefix for a given type name (e.g., "i128" → "ez_i128") */
static const char *bigint_prefix(const char *tn) {
    if (strcmp(tn, "i128") == 0) return "ez_i128";
    if (strcmp(tn, "u128") == 0) return "ez_u128";
    if (strcmp(tn, "i256") == 0) return "ez_i256";
    if (strcmp(tn, "u256") == 0) return "ez_u256";
    return NULL;
}

/* Resolve the bigint type name for an expression (checks labels against tracked vars) */
static const char *resolve_bigint_type(CodeGen *cg, AstNode *node) {
    if (!node) return NULL;
    if (node->kind == NODE_LABEL) {
        return lookup_bigint_var(cg, node->data.label.value);
    }
    /* If the node is a call to a bigint cast function */
    if (node->kind == NODE_CALL_EXPR && node->data.call.function->kind == NODE_LABEL) {
        const char *fn = node->data.call.function->data.label.value;
        if (is_bigint_type(fn)) return fn;
    }
    /* If this is an infix expression, check left operand */
    if (node->kind == NODE_INFIX_EXPR) {
        const char *lt = resolve_bigint_type(cg, node->data.infix.left);
        if (lt) return lt;
        return resolve_bigint_type(cg, node->data.infix.right);
    }
    return NULL;
}

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
    case NODE_LABEL: {
        const char *name = node->data.label.value;
        /* Check for known stdlib constants (unambiguous names) */
        if (is_mutable_param(cg, name)) {
            emitf(cg, "(*%s)", name);
        } else if (is_ref_var(cg, name)) {
            emitf(cg, "(*%s)", name);
        } else {
            emit(cg, name);
        }
        break;
    }

    case NODE_INT_VALUE:
        emitf(cg, "%lld", (long long)node->data.int_value.value);
        break;

    case NODE_FLOAT_VALUE: {
        /* Emit float with enough precision, ensuring a decimal point so C
         * treats it as double (e.g. 1.0 must emit "1.0", not "1") */
        char fbuf[64];
        snprintf(fbuf, sizeof(fbuf), "%.17g", node->data.float_value.value);
        if (!strchr(fbuf, '.') && !strchr(fbuf, 'e')) {
            size_t flen = strlen(fbuf);
            fbuf[flen] = '.';
            fbuf[flen+1] = '0';
            fbuf[flen+2] = '\0';
        }
        emit(cg, fbuf);
        break;
    }

    case NODE_STRING_VALUE: {
        /* Emit string literal, breaking hex escapes to prevent C's greedy \x parsing.
         * "A\x42C" → "A\x42" "C" (C string concatenation) */
        const char *s = node->data.string_value.value;
        /* Use macro form for file-scope compatibility */
        emit(cg, (cg->indent == 0) ? "EZ_STRING_LIT(\"" : "ez_string_lit(\"");
        while (*s) {
            if (s[0] == '\\' && s[1] == 'x' && isxdigit(s[2])) {
                /* Emit \xNN then break the string if followed by a hex digit */
                buf_append_char(&cg->output, s[0]); /* \ */
                buf_append_char(&cg->output, s[1]); /* x */
                buf_append_char(&cg->output, s[2]); /* first hex */
                s += 3;
                if (isxdigit(*s)) {
                    buf_append_char(&cg->output, *s); /* second hex */
                    s++;
                }
                if (isxdigit(*s)) {
                    /* Next char is also hex — break the string */
                    emit(cg, "\" \"");
                }
            } else {
                buf_append_char(&cg->output, *s);
                s++;
            }
        }
        emit(cg, "\")");
        break;
    }

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
                    else tk = TK_INT; /* default integer kind */
                }

                switch (tk) {
                case TK_STRING: emit(cg, "%s"); break;
                case TK_FLOAT:  emit(cg, "%s"); break; /* uses ez_std_format_float */
                case TK_BOOL:   emit(cg, "%s"); break;
                case TK_CHAR:   emit(cg, "%c"); break;
                case TK_ARRAY:  emit(cg, "%s"); break;
                case TK_MAP:    emit(cg, "%s"); break;
                case TK_ERROR:  emit(cg, "%s"); break;
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
                else tk = TK_INT; /* default integer kind */
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
                emit(cg, "ez_std_format_float(ez_default_arena, ");
                emit_expression(cg, part);
                emit(cg, ").data");
                break;
            case TK_CHAR:
                emit(cg, "(char)(");
                emit_expression(cg, part);
                emit(cg, ")");
                break;
            case TK_ARRAY: {
                /* Determine element kind: 0=int, 1=float, 2=string, 3=bool */
                int ek = 0;
                if (t && t->element_type) {
                    EzType *et = type_from_name(t->element_type);
                    if (et->kind == TK_FLOAT) ek = 1;
                    else if (et->kind == TK_STRING) ek = 2;
                    else if (et->kind == TK_BOOL) ek = 3;
                }
                emitf(cg, "ez_std_array_to_string(ez_default_arena, &");
                emit_expression(cg, part);
                emitf(cg, ", %d).data", ek);
                break;
            }
            case TK_MAP: {
                /* Determine value kind: 0=int, 1=float, 2=string, 3=bool */
                int vk = 0;
                if (t && t->value_type) {
                    EzType *vt = type_from_name(t->value_type);
                    if (vt->kind == TK_FLOAT) vk = 1;
                    else if (vt->kind == TK_STRING) vk = 2;
                    else if (vt->kind == TK_BOOL) vk = 3;
                }
                emitf(cg, "ez_std_map_to_string(ez_default_arena, &");
                emit_expression(cg, part);
                emitf(cg, ", %d).data", vk);
                break;
            }
            case TK_ERROR:
                /* Print error message */
                emit_expression(cg, part);
                emit(cg, " ? ");
                emit_expression(cg, part);
                emit(cg, "->message.data : \"nil\"");
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
            /* Empty array — check type table for element type */
            EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
            const char *elem_sz = "int64_t";
            if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
                EzType *et = type_from_name(arr_t->element_type);
                if (et->kind == TK_FLOAT) elem_sz = "double";
                else if (et->kind == TK_BOOL) elem_sz = "bool";
                else if (et->kind == TK_STRING) elem_sz = "EzString";
                else if (et->kind == TK_STRUCT) elem_sz = ez_type_to_c_cg(cg, arr_t->element_type);
            }
            emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", elem_sz);
            break;
        }

        /* Check if this is a nested array (elements are arrays) */
        if (node->data.array_value.elements[0]->kind == NODE_ARRAY_VALUE) {
            /* Nested array: each element is an EzArray */
            emitf(cg, "ez_array_from(ez_default_arena, (EzArray[]){");
            for (int i = 0; i < count; i++) {
                if (i > 0) emit(cg, ", ");
                emit_expression(cg, node->data.array_value.elements[i]);
            }
            emitf(cg, "}, sizeof(EzArray), %d)", count);
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
        case TK_STRUCT: c_type = ez_type_to_c_cg(cg, elem_t->name); break;
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
            if (kt && (kt->kind == TK_INT || kt->kind == TK_UINT)) c_key_type = "int64_t";
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
        /* For negation of int literals that are already negative (e.g. parser
         * stored -9223372036854775808 as the literal), emit directly.
         * Special case INT64_MIN to avoid C literal overflow warning. */
        if (strcmp(node->data.prefix.op, "-") == 0 &&
            node->data.prefix.right->kind == NODE_INT_VALUE &&
            node->data.prefix.right->data.int_value.value < 0) {
            int64_t v = node->data.prefix.right->data.int_value.value;
            if (v == INT64_MIN) {
                emit(cg, "(-9223372036854775807LL - 1)");
            } else {
                emitf(cg, "(%lldLL)", (long long)v);
            }
            break;
        }
        /* Bigint negation */
        if (strcmp(node->data.prefix.op, "-") == 0) {
            const char *bi_type = resolve_bigint_type(cg, node->data.prefix.right);
            if (bi_type && (strcmp(bi_type, "i128") == 0 || strcmp(bi_type, "i256") == 0)) {
                emitf(cg, "%s_neg(", bigint_prefix(bi_type));
                emit_expression(cg, node->data.prefix.right);
                emit(cg, ")");
                break;
            }
        }
        emit(cg, "(");
        emit(cg, node->data.prefix.op);
        if (strcmp(node->data.prefix.op, "-") == 0 &&
            node->data.prefix.right->kind == NODE_INT_VALUE) {
            emit(cg, " ");
        }
        emit_expression(cg, node->data.prefix.right);
        emit(cg, ")");
        break;

    case NODE_INFIX_EXPR: {
        const char *op = node->data.infix.op;

        /* Check if either operand is a string — need special handling */
        EzType *lt = cg->type_table ? typetable_get(cg->type_table, node->data.infix.left) : NULL;
        EzType *rt = cg->type_table ? typetable_get(cg->type_table, node->data.infix.right) : NULL;
        bool left_is_str = (lt && lt->kind == TK_STRING) || node->data.infix.left->kind == NODE_STRING_VALUE;
        bool right_is_str = (rt && rt->kind == TK_STRING) || node->data.infix.right->kind == NODE_STRING_VALUE;

        if ((left_is_str || right_is_str) && strcmp(op, "+") == 0) {
            /* String concatenation */
            emit(cg, "ez_string_concat(ez_default_arena, ");
            emit_expression(cg, node->data.infix.left);
            emit(cg, ", ");
            emit_expression(cg, node->data.infix.right);
            emit(cg, ")");
            break;
        }
        if ((left_is_str || right_is_str) && strcmp(op, "==") == 0) {
            emit(cg, "ez_string_eq(");
            emit_expression(cg, node->data.infix.left);
            emit(cg, ", ");
            emit_expression(cg, node->data.infix.right);
            emit(cg, ")");
            break;
        }
        if ((left_is_str || right_is_str) && strcmp(op, "!=") == 0) {
            emit(cg, "!ez_string_eq(");
            emit_expression(cg, node->data.infix.left);
            emit(cg, ", ");
            emit_expression(cg, node->data.infix.right);
            emit(cg, ")");
            break;
        }

        /* in / not_in — array or range membership check */
        if (strcmp(op, "in") == 0 || strcmp(op, "not_in") == 0 || strcmp(op, "!in") == 0) {
            bool negated = (op[0] == 'n' || op[0] == '!');

            /* Check if right side is a range expression: x in range(a, b) */
            if (node->data.infix.right->kind == NODE_RANGE_EXPR) {
                AstNode *r = node->data.infix.right;
                if (negated) emit(cg, "!(");
                emit(cg, "(");
                emit_expression(cg, node->data.infix.left);
                emit(cg, " >= ");
                if (r->data.range_expr.start) {
                    emit_expression(cg, r->data.range_expr.start);
                } else {
                    emit(cg, "0");
                }
                emit(cg, " && ");
                emit_expression(cg, node->data.infix.left);
                emit(cg, " < ");
                emit_expression(cg, r->data.range_expr.end);
                /* Step check: value must be at a step interval from start */
                if (r->data.range_expr.step) {
                    emit(cg, " && (");
                    emit_expression(cg, node->data.infix.left);
                    emit(cg, " - ");
                    if (r->data.range_expr.start) {
                        emit_expression(cg, r->data.range_expr.start);
                    } else {
                        emit(cg, "0");
                    }
                    emit(cg, ") % ");
                    emit_expression(cg, r->data.range_expr.step);
                    emit(cg, " == 0");
                }
                emit(cg, ")");
                if (negated) emit(cg, ")");
                break;
            }

            /* Map or array membership */
            EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.infix.right) : NULL;
            /* Map membership: key in map → ez_maps_has_key */
            if (arr_t && arr_t->kind == TK_MAP) {
                if (negated) emit(cg, "!");
                emit(cg, "({ __auto_type _ik = ");
                emit_expression(cg, node->data.infix.left);
                emit(cg, "; ez_maps_has_key(&");
                emit_expression(cg, node->data.infix.right);
                emit(cg, ", &_ik); })");
                break;
            }
            if (negated) emit(cg, "!");
            if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
                strcmp(arr_t->element_type, "string") == 0) {
                emit(cg, "ez_arrays_contains_str(&");
                emit_expression(cg, node->data.infix.right);
                emit(cg, ", ");
                emit_expression(cg, node->data.infix.left);
                emit(cg, ")");
            } else {
                emit(cg, "ez_arrays_contains_int(&");
                emit_expression(cg, node->data.infix.right);
                emit(cg, ", ");
                emit_expression(cg, node->data.infix.left);
                emit(cg, ")");
            }
            break;
        }

        /* Bigint infix — emit function calls instead of C operators.
         * Must come before overflow-check and div-by-zero handlers since
         * bigint types share TK_INT/TK_UINT kind. */
        {
            const char *bi_type = resolve_bigint_type(cg, node->data.infix.left);
            if (!bi_type) bi_type = resolve_bigint_type(cg, node->data.infix.right);
            if (bi_type) {
                const char *pfx = bigint_prefix(bi_type);
                const char *fn_op = NULL;
                if (strcmp(op, "+") == 0) fn_op = "add";
                else if (strcmp(op, "-") == 0) fn_op = "sub";
                else if (strcmp(op, "*") == 0) fn_op = "mul";
                else if (strcmp(op, "/") == 0) fn_op = "div";
                else if (strcmp(op, "%") == 0) fn_op = "mod";
                else if (strcmp(op, "==") == 0) fn_op = "eq";
                else if (strcmp(op, "!=") == 0) fn_op = "ne";
                else if (strcmp(op, "<") == 0) fn_op = "lt";
                else if (strcmp(op, ">") == 0) fn_op = "gt";
                else if (strcmp(op, "<=") == 0) fn_op = "le";
                else if (strcmp(op, ">=") == 0) fn_op = "ge";
                if (fn_op) {
                    emitf(cg, "%s_%s(", pfx, fn_op);
                    emit_expression(cg, node->data.infix.left);
                    emit(cg, ", ");
                    emit_expression(cg, node->data.infix.right);
                    emit(cg, ")");
                    break;
                }
            }
        }

        /* Runtime division/modulo by zero check (skip for floats — IEEE 754 Inf) */
        if (strcmp(op, "/") == 0 || strcmp(op, "%") == 0) {
            bool is_float_div = (lt && lt->kind == TK_FLOAT) || (rt && rt->kind == TK_FLOAT);
            if (is_float_div) {
                /* Float division: let IEEE 754 handle it (Inf, -Inf, NaN) */
            } else {
                emit(cg, "({ __auto_type _dv = ");
                emit_expression(cg, node->data.infix.right);
                emitf(cg, "; if (!_dv) { fflush(stdout); ez_panic(__FILE__, %d, \"division by zero\"); } ",
                    node->token.line);
                emit_expression(cg, node->data.infix.left);
                emitf(cg, " %s _dv; })", op);
                break;
            }
        }

        /* Overflow-checked integer arithmetic for +, -, * */
        {
            bool left_is_int = (lt && (lt->kind == TK_INT || lt->kind == TK_UINT || lt->kind == TK_BYTE || lt->kind == TK_CHAR));
            bool right_is_int = (rt && (rt->kind == TK_INT || rt->kind == TK_UINT || rt->kind == TK_BYTE || rt->kind == TK_CHAR));
            bool left_is_float = (lt && lt->kind == TK_FLOAT);
            bool right_is_float = (rt && rt->kind == TK_FLOAT);
            bool is_arith = (strcmp(op, "+") == 0 || strcmp(op, "-") == 0 || strcmp(op, "*") == 0);

            if (is_arith && left_is_int && right_is_int && !left_is_float && !right_is_float) {
                /* Use unsigned-safe variants when either operand is unsigned */
                bool is_unsigned = (lt && lt->kind == TK_UINT) || (rt && rt->kind == TK_UINT);
                const char *fn = NULL;
                if (is_unsigned) {
                    if (strcmp(op, "+") == 0) fn = "ez_uadd_check";
                    else if (strcmp(op, "-") == 0) fn = "ez_usub_check";
                    else if (strcmp(op, "*") == 0) fn = "ez_umul_check";
                } else {
                    if (strcmp(op, "+") == 0) fn = "ez_add_check";
                    else if (strcmp(op, "-") == 0) fn = "ez_sub_check";
                    else if (strcmp(op, "*") == 0) fn = "ez_mul_check";
                }
                if (fn) {
                    emitf(cg, "%s(", fn);
                    emit_expression(cg, node->data.infix.left);
                    emit(cg, ", ");
                    emit_expression(cg, node->data.infix.right);
                    emitf(cg, ", __FILE__, %d)", node->token.line);
                    break;
                }
            }
        }

        /* Normal infix — always wrap sub-infix expressions in parens to
         * preserve the precedence the parser established via the AST shape.
         * Without this, (x + y) * z would emit as x + y * z. */
        bool l_infix = (node->data.infix.left->kind == NODE_INFIX_EXPR);
        bool r_infix = (node->data.infix.right->kind == NODE_INFIX_EXPR);
        if (l_infix) emit(cg, "(");
        emit_expression(cg, node->data.infix.left);
        if (l_infix) emit(cg, ")");
        emitf(cg, " %s ", op);
        if (r_infix) emit(cg, "(");
        emit_expression(cg, node->data.infix.right);
        if (r_infix) emit(cg, ")");
        break;
    }

    case NODE_POSTFIX_EXPR:
        if (strcmp(node->data.postfix.op, "^") == 0) {
            /* Pointer dereference: p^ → (*p) with nil check */
            emit(cg, "({ __auto_type _dp = ");
            emit_expression(cg, node->data.postfix.left);
            emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
                "\"nil pointer dereference\"); } *_dp; })", node->token.line);
        } else if (strcmp(node->data.postfix.op, "++") == 0) {
            /* Overflow-checked increment */
            emit(cg, "(");
            emit_expression(cg, node->data.postfix.left);
            emitf(cg, " = ez_add_check(");
            emit_expression(cg, node->data.postfix.left);
            emitf(cg, ", 1, __FILE__, %d))", node->token.line);
        } else if (strcmp(node->data.postfix.op, "--") == 0) {
            /* Overflow-checked decrement */
            emit(cg, "(");
            emit_expression(cg, node->data.postfix.left);
            emitf(cg, " = ez_sub_check(");
            emit_expression(cg, node->data.postfix.left);
            emitf(cg, ", 1, __FILE__, %d))", node->token.line);
        } else {
            emit_expression(cg, node->data.postfix.left);
            emit(cg, node->data.postfix.op);
        }
        break;

    case NODE_FUNC_REF:
        /* ()func_name or ()Type.func — emit as C function pointer */
        if (node->data.func_ref.function->kind == NODE_LABEL) {
            emit(cg, "ez_fn_");
            emit(cg, node->data.func_ref.function->data.label.value);
        } else if (node->data.func_ref.function->kind == NODE_MEMBER_EXPR) {
            /* ()StructName.funcName → ez_fn_StructName_funcName */
            AstNode *mem = node->data.func_ref.function;
            if (mem->data.member.object->kind == NODE_LABEL) {
                emitf(cg, "ez_fn_%s_%s",
                    mem->data.member.object->data.label.value,
                    mem->data.member.member);
            } else {
                emit(cg, "ez_fn_");
                emit_expression(cg, node->data.func_ref.function);
            }
        } else {
            emit(cg, "ez_fn_");
            emit_expression(cg, node->data.func_ref.function);
        }
        break;

    case NODE_CALL_EXPR:
        emit_call_expression(cg, node);
        break;

    case NODE_MEMBER_EXPR:
        /* Check for module constants first */
        if (node->data.member.object->kind == NODE_LABEL) {
            const char *mod = node->data.member.object->data.label.value;
            const char *mem = node->data.member.member;

            /* @math constants */
            if (strcmp(mod, "math") == 0) {
                if (strcmp(mem, "PI") == 0)      { emit(cg, "3.14159265358979323846"); break; }
                if (strcmp(mem, "E") == 0)       { emit(cg, "2.71828182845904523536"); break; }
                if (strcmp(mem, "TAU") == 0)     { emit(cg, "6.28318530717958647692"); break; }
                if (strcmp(mem, "PHI") == 0)     { emit(cg, "1.61803398874989484820"); break; }
                if (strcmp(mem, "SQRT2") == 0)   { emit(cg, "1.41421356237309504880"); break; }
                if (strcmp(mem, "LN2") == 0)     { emit(cg, "0.69314718055994530942"); break; }
                if (strcmp(mem, "LN10") == 0)    { emit(cg, "2.30258509299404568402"); break; }
                if (strcmp(mem, "INF") == 0)     { emit(cg, "(1.0/0.0)"); break; }
                if (strcmp(mem, "NEG_INF") == 0) { emit(cg, "(-1.0/0.0)"); break; }
                if (strcmp(mem, "EPSILON") == 0) { emit(cg, "2.2204460492503131e-16"); break; }
            }

            /* @os constants */
            if (strcmp(mod, "os") == 0) {
                if (strcmp(mem, "MAC_OS") == 0)  { emit(cg, "0"); break; }
                if (strcmp(mem, "LINUX") == 0)   { emit(cg, "1"); break; }
                if (strcmp(mem, "WINDOWS") == 0) { emit(cg, "2"); break; }
                if (strcmp(mem, "OTHER") == 0)   { emit(cg, "3"); break; }
            }

            /* Check if this is an enum access: EnumName.VALUE */
            if (mod[0] >= 'A' && mod[0] <= 'Z') {
                emitf(cg, "EzEnum_%s_%s", mod, mem);
                break;
            }
        }
        /* Check if object is a pointer type — use -> instead of . */
        {
            EzType *obj_t = cg->type_table
                ? typetable_get(cg->type_table, node->data.member.object)
                : NULL;

            /* When accessing .v0 on a single-return value (not a multi-return struct),
             * just emit the value itself (e.g., from temp x, _ = single_return_func()) */
            /* When accessing .v0 on a single-return value (not a multi-return temp),
             * just emit the value itself. Skip this for _ez_tmp* variables which
             * are multi-return unpacking temps. */
            const char *mem_name = node->data.member.member;
            if (mem_name[0] == 'v' && mem_name[1] >= '0' && mem_name[1] <= '9' && mem_name[2] == '\0') {
                bool is_multi_temp = false;
                if (node->data.member.object->kind == NODE_LABEL) {
                    const char *oname = node->data.member.object->data.label.value;
                    if (strncmp(oname, "_ez_tmp", 7) == 0 ||
                        strncmp(oname, "_ez_or", 6) == 0) is_multi_temp = true;
                }
                if (!is_multi_temp && obj_t &&
                    (obj_t->kind == TK_INT || obj_t->kind == TK_UINT || obj_t->kind == TK_FLOAT ||
                     obj_t->kind == TK_BOOL || obj_t->kind == TK_STRING ||
                     obj_t->kind == TK_CHAR || obj_t->kind == TK_BYTE)) {
                    if (mem_name[1] == '0') {
                        emit_expression(cg, node->data.member.object);
                    } else {
                        emit(cg, "0 /* discarded */");
                    }
                    break;
                }
            }

            emit_expression(cg, node->data.member.object);
            /* Ref vars are already dereferenced by label emission — use . not -> */
            bool obj_is_ref = (node->data.member.object->kind == NODE_LABEL &&
                is_ref_var(cg, node->data.member.object->data.label.value));
            if (!obj_is_ref && obj_t && (obj_t->kind == TK_POINTER || obj_t->kind == TK_ERROR)) {
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
                else if (et->kind == TK_ARRAY) c_elem = "EzArray";
                else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, left_t->element_type);
            }
            emitf(cg, "EZ_ARRAY_GET(");
            emit_expression(cg, node->data.index_expr.left);
            emitf(cg, ", %s, ", c_elem);
            emit_expression(cg, node->data.index_expr.index);
            emit(cg, ")");
        } else if (left_t && left_t->kind == TK_MAP) {
            /* Map key access — use temp to handle rvalue keys like literals */
            const char *c_key = "EzString";
            const char *c_val = "int64_t";
            if (left_t->key_type) {
                EzType *kt = type_from_name(left_t->key_type);
                if (kt->kind == TK_INT || kt->kind == TK_UINT) c_key = "int64_t";
            }
            if (left_t->value_type) {
                EzType *vt = type_from_name(left_t->value_type);
                if (vt->kind == TK_FLOAT) c_val = "double";
                else if (vt->kind == TK_STRING) c_val = "EzString";
                else if (vt->kind == TK_BOOL) c_val = "bool";
            }
            emitf(cg, "({ %s _mk = ", c_key);
            emit_expression(cg, node->data.index_expr.index);
            emitf(cg, "; void *_mv = ez_map_get(&");
            emit_expression(cg, node->data.index_expr.left);
            emitf(cg, ", &_mk); if (!_mv) { fflush(stdout); ez_panic(__FILE__, %d, \"key not found in map\"); } ",
                node->token.line);
            emitf(cg, "*(%s *)_mv; })", c_val);
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
        emitf(cg, "0 /* ezc: unhandled expression kind %d at %s:%d */",
            node->kind, cg->file, node->token.line);
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

/* --- Stdlib call emission helpers --- */

static const char *resolve_print_suffix(CodeGen *cg, AstNode *arg) {
    /* addr() calls always print in hex format */
    if (arg->kind == NODE_CALL_EXPR && arg->data.call.function->kind == NODE_LABEL &&
        strcmp(arg->data.call.function->data.label.value, "addr") == 0) return "_addr";
    EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
    if (t && t->kind != TK_UNKNOWN) {
        switch (t->kind) {
        case TK_STRING: return "_str";
        case TK_FLOAT:  return "_float";
        case TK_BOOL:   return "_bool";
        case TK_CHAR:   return "_char";
        default:        return "_int";
        }
    }
    if (arg->kind == NODE_STRING_VALUE || arg->kind == NODE_INTERPOLATED_STRING) return "_str";
    if (arg->kind == NODE_FLOAT_VALUE) return "_float";
    if (arg->kind == NODE_BOOL_VALUE) return "_bool";
    if (arg->kind == NODE_CHAR_VALUE) return "_char";
    /* For call expressions, check the return type of the called function */
    if (arg->kind == NODE_CALL_EXPR && arg->data.call.function->kind == NODE_MEMBER_EXPR) {
        AstNode *fn = arg->data.call.function;
        if (fn->data.member.object->kind == NODE_LABEL) {
            const char *obj = fn->data.member.object->data.label.value;
            const char *mem = fn->data.member.member;
            /* Check if it's a known stdlib module function that returns string */
            if ((strcmp(obj, "strings") == 0) ||
                (strcmp(obj, "encoding") == 0) ||
                (strcmp(obj, "crypto") == 0) ||
                (strcmp(obj, "uuid") == 0)) return "_str";
            /* Check if it's a struct-namespaced function */
            if (codegen_is_enum(cg, obj) || (obj[0] >= 'A' && obj[0] <= 'Z')) {
                /* Look up forward declaration to determine return type */
                /* For now, check if the generated C declaration returns EzString */
                (void)mem; /* TODO: proper return type lookup */
            }
        }
    }
    if (arg->kind == NODE_CALL_EXPR && arg->data.call.function->kind == NODE_LABEL) {
        const char *fn = arg->data.call.function->data.label.value;
        if (strcmp(fn, "input") == 0 || strcmp(fn, "type_of") == 0) return "_str";
        if (strcmp(fn, "addr") == 0) return "_addr";
    }
    return "_int";
}

static void emit_to_string(CodeGen *cg, AstNode *arg) {
    /* Bigint to_string */
    const char *bi_type = resolve_bigint_type(cg, arg);
    if (bi_type) {
        emitf(cg, "%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
        emit_expression(cg, arg);
        emit(cg, ")");
        return;
    }
    EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
    if (at && at->kind == TK_FLOAT)
        emit(cg, "ez_std_to_string_float(ez_default_arena, ");
    else if (at && at->kind == TK_BOOL)
        emit(cg, "ez_std_to_string_bool(ez_default_arena, ");
    else
        emit(cg, "ez_std_to_string_int(ez_default_arena, ");
    emit_expression(cg, arg);
    emit(cg, ")");
}

static void emit_fmt_args(CodeGen *cg, AstNode *node, int start_idx) {
    for (int i = start_idx; i < node->data.call.arg_count; i++) {
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
}

/* --- Builtin call handler (no-module functions) --- */

static bool emit_builtin_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "println") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "putchar('\\n')");
        } else {
            AstNode *arg = node->data.call.args[0];
            const char *bi_type = resolve_bigint_type(cg, arg);
            if (bi_type) {
                emitf(cg, "ez_std_println_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
                emit_expression(cg, arg);
                emit(cg, "))");
            } else {
                emitf(cg, "ez_std_println%s(", resolve_print_suffix(cg, arg));
                emit_expression(cg, arg);
                emit(cg, ")");
            }
        }
        return true;
    }

    if (strcmp(func, "len") == 0 && node->data.call.arg_count == 1) {
        AstNode *arg = node->data.call.args[0];
        EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (t && t->kind == TK_MAP) {
            emit(cg, "(int64_t)(");
            emit_expression(cg, arg);
            emit(cg, ").count");
        } else {
            emit(cg, "(int64_t)(");
            emit_expression(cg, arg);
            emit(cg, ").len");
        }
        return true;
    }

    if (strcmp(func, "type_of") == 0 && node->data.call.arg_count == 1) {
        AstNode *arg = node->data.call.args[0];
        /* Bigint type_of: return the exact type name */
        const char *bi_type = resolve_bigint_type(cg, arg);
        if (bi_type) {
            emitf(cg, "ez_string_lit(\"%s\")", bi_type);
            return true;
        }
        EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        /* Range expression: type_of(range(0, 5)) → "Range<int>" */
        if (arg->kind == NODE_RANGE_EXPR ||
            (arg->kind == NODE_CALL_EXPR && arg->data.call.function->kind == NODE_LABEL &&
             strcmp(arg->data.call.function->data.label.value, "range") == 0)) {
            emitf(cg, "ez_string_lit(\"Range<int>\")");
            return true;
        }
        /* Enum member access: type_of(Color.RED) → "Color" */
        if (arg->kind == NODE_MEMBER_EXPR && arg->data.member.object->kind == NODE_LABEL &&
            t && (t->kind == TK_INT || t->kind == TK_UINT || t->kind == TK_STRING)) {
            const char *obj_name = arg->data.member.object->data.label.value;
            if (obj_name[0] >= 'A' && obj_name[0] <= 'Z' &&
                strcmp(obj_name, "std") != 0 && strcmp(obj_name, "math") != 0 &&
                strcmp(obj_name, "os") != 0) {
                emitf(cg, "ez_string_lit(\"%s\")", obj_name);
                return true;
            }
        }
        if (t && t->kind == TK_ARRAY && t->element_type) {
            emitf(cg, "ez_string_lit(\"[%s]\")", t->element_type);
        } else if (t && t->kind == TK_MAP) {
            const char *kt = t->key_type ? t->key_type : "unknown";
            const char *vt = t->value_type ? t->value_type : "unknown";
            emitf(cg, "ez_string_lit(\"map[%s:%s]\")", kt, vt);
        } else if (t && t->kind == TK_POINTER && t->element_type) {
            emitf(cg, "ez_string_lit(\"^%s\")", t->element_type);
        } else {
            const char *tn = t ? type_name(t) : "unknown";
            emitf(cg, "ez_string_lit(\"%s\")", tn);
        }
        return true;
    }

    if (strcmp(func, "size_of") == 0 && node->data.call.arg_count == 1) {
        AstNode *type_arg = node->data.call.args[0];
        if (type_arg->kind == NODE_LABEL) {
            emitf(cg, "(int64_t)sizeof(%s)", ez_type_to_c_cg(cg, type_arg->data.label.value));
        } else {
            emit(cg, "0");
        }
        return true;
    }

    if (strcmp(func, "addr") == 0 && node->data.call.arg_count == 1) {
        /* addr() returns the raw numeric memory address */
        emit(cg, "(uintptr_t)&");
        emit_expression(cg, node->data.call.args[0]);
        return true;
    }

    if (strcmp(func, "ref") == 0 && node->data.call.arg_count == 1) {
        /* Check if argument is a function name — emit as function pointer */
        if (node->data.call.args[0]->kind == NODE_LABEL) {
            const char *arg_name = node->data.call.args[0]->data.label.value;
            AstNode *target = find_func(cg, arg_name);
            if (target) {
                /* Function reference: emit ez_fn_name (function pointer) */
                emitf(cg, "ez_fn_%s", arg_name);
                return true;
            }
        }
        /* Variable reference: emit &var */
        emit(cg, "&");
        emit_expression(cg, node->data.call.args[0]);
        return true;
    }

    if (strcmp(func, "exit") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_std_exit(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "panic") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_std_panic_msg(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
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
        return true;
    }

    if (strcmp(func, "error") == 0 && node->data.call.arg_count >= 1) {
        emit(cg, "ez_error_new(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "input") == 0) {
        emit(cg, "ez_std_input(ez_default_arena)");
        return true;
    }


    if (strcmp(func, "eprintln") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "fputc('\\n', stderr)");
        } else {
            AstNode *arg = node->data.call.args[0];
            const char *bi_type = resolve_bigint_type(cg, arg);
            if (bi_type) {
                emitf(cg, "ez_std_eprintln_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
                emit_expression(cg, arg);
                emit(cg, "))");
            } else {
                emitf(cg, "ez_std_eprintln%s(", resolve_print_suffix(cg, arg));
                emit_expression(cg, arg);
                emit(cg, ")");
            }
        }
        return true;
    }

    if (strcmp(func, "eprint") == 0 && node->data.call.arg_count > 0) {
        emit(cg, "ez_std_eprint_str(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sleep_seconds") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_std_sleep_seconds(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sleep_milliseconds") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_std_sleep_milliseconds(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sleep_nanoseconds") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_std_sleep_nanoseconds(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    /* Bigint conversion functions: i128(), u128(), i256(), u256() */
    if (node->data.call.arg_count == 1 && is_bigint_type(func)) {
        AstNode *carg = node->data.call.args[0];
        const char *src_bi = resolve_bigint_type(cg, carg);
        const char *pfx = bigint_prefix(func);
        if (src_bi) {
            /* Bigint→bigint cast */
            emitf(cg, "%s_from_%s(", pfx, src_bi);
            emit_expression(cg, carg);
            emit(cg, ")");
        } else {
            /* Scalar→bigint: e.g., ez_i128_from_i64(x) */
            const char *from_suffix = (strcmp(func, "u128") == 0 || strcmp(func, "u256") == 0) ? "u64" : "i64";
            emitf(cg, "%s_from_%s(", pfx, from_suffix);
            emit_expression(cg, carg);
            emit(cg, ")");
        }
        return true;
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
        else if (strcmp(func, "char") == 0) cast_type = "int32_t";
        else if (strcmp(func, "byte") == 0) cast_type = "uint8_t";
        if (cast_type) {
            AstNode *carg = node->data.call.args[0];
            /* Bigint→scalar: e.g., int(x128) → ez_i128_to_i64(x128) */
            const char *src_bi = resolve_bigint_type(cg, carg);
            if (src_bi) {
                const char *src_pfx = bigint_prefix(src_bi);
                const char *to_suffix = (strcmp(src_bi, "u128") == 0 || strcmp(src_bi, "u256") == 0) ? "u64" : "i64";
                emitf(cg, "((%s)%s_to_%s(", cast_type, src_pfx, to_suffix);
                emit_expression(cg, carg);
                emit(cg, "))");
                return true;
            }
            /* Use overflow-safe conversion for float→int casts */
            bool use_safe_float = false;
            if ((strcmp(func, "int") == 0 || strcmp(func, "i64") == 0 ||
                 strcmp(func, "i32") == 0 || strcmp(func, "i16") == 0 ||
                 strcmp(func, "i8") == 0) &&
                (carg->kind == NODE_FLOAT_VALUE || carg->kind == NODE_LABEL)) {
                use_safe_float = true;
            }
            if (use_safe_float) {
                emitf(cg, "ez_float_to_int((double)(");
                emit_expression(cg, carg);
                emitf(cg, "), __FILE__, __LINE__)");
            } else {
                emitf(cg, "((%s)(", cast_type);
                emit_expression(cg, carg);
                emit(cg, "))");
            }
            return true;
        }
        /* string() conversion */
        if (strcmp(func, "string") == 0) {
            emit_to_string(cg, node->data.call.args[0]);
            return true;
        }
    }

    if (strcmp(func, "copy") == 0 && node->data.call.arg_count == 1) {
        AstNode *arg = node->data.call.args[0];
        EzType *at = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (at && at->kind == TK_ARRAY) {
            emit(cg, "ez_array_copy(ez_default_arena, &");
            emit_expression(cg, arg);
            emit(cg, ")");
        } else {
            emit_expression(cg, arg);
        }
        return true;
    }

    if (strcmp(func, "print") == 0 && node->data.call.arg_count > 0) {
        AstNode *arg = node->data.call.args[0];
        const char *bi_type = resolve_bigint_type(cg, arg);
        if (bi_type) {
            emitf(cg, "ez_std_print_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
            emit_expression(cg, arg);
            emit(cg, "))");
        } else {
            const char *suffix = "_int";
            if (arg->kind == NODE_STRING_VALUE) suffix = "_str";
            emitf(cg, "ez_std_print%s(", suffix);
            emit_expression(cg, arg);
            emit(cg, ")");
        }
        return true;
    }

    return false;
}

/* --- @mem module --- */

static bool emit_mem_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "arena") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_mem_arena(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "destroy") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_mem_destroy(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "reset") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_mem_reset(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "usage") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_mem_usage(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "copy") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_mem_copy(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "zero") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_mem_zero(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "set") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_mem_set(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "make") == 0 && node->data.call.arg_count == 2) {
        AstNode *arena_arg = node->data.call.args[0];
        AstNode *type_arg = node->data.call.args[1];
        const char *tn = "int64_t";
        if (type_arg->kind == NODE_LABEL) {
            tn = ez_type_to_c_cg(cg, type_arg->data.label.value);
        }
        emitf(cg, "(%s *)ez_arena_alloc(", tn);
        emit_expression(cg, arena_arg);
        emitf(cg, ", sizeof(%s))", tn);
        return true;
    }
    if (strcmp(func, "alloc") == 0 && node->data.call.arg_count == 2) {
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
            emit(cg, "({ __auto_type _v = ");
            emit_expression(cg, value_arg);
            emit(cg, "; __auto_type _p = ez_arena_alloc(");
            emit_expression(cg, arena_arg);
            emit(cg, ", sizeof(_v)); *(__typeof__(_v) *)_p = _v; _v; })");
        }
        return true;
    }
    return false;
}

/* --- @math module --- */

static bool emit_math_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "random") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "ez_math_random_float(0.0, 1.0)");
        } else if (node->data.call.arg_count == 1) {
            emit(cg, "ez_math_random_int(0, ");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
        } else {
            emit(cg, "ez_math_random_int(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
        }
        return true;
    }
    if (strcmp(func, "abs") == 0 && node->data.call.arg_count == 1) {
        EzType *at = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        emitf(cg, "ez_math_abs_%s(", (at && at->kind == TK_FLOAT) ? "float" : "int");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "neg") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "(-(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, "))");
        return true;
    }
    /* Generic: math.func(args...) → ez_math_func(args...) */
    emitf(cg, "ez_math_%s(", func);
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @maps module --- */

static bool emit_maps_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "keys") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_maps_keys(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "values") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_maps_values(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "has_key") == 0) {
        emit(cg, "({ __auto_type _hk = ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_maps_has_key(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_hk); })");
        return true;
    }
    if (strcmp(func, "remove") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "({ __auto_type _rk = ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_map_remove(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_rk); })");
        return true;
    }
    if (strcmp(func, "clear") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_map_clear(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @time module --- */

static bool emit_time_call(CodeGen *cg, AstNode *node, const char *func) {
    bool needs_arena = (strcmp(func, "format") == 0 || strcmp(func, "iso") == 0 ||
        strcmp(func, "date") == 0 || strcmp(func, "clock") == 0);
    emitf(cg, "ez_time_%s(", func);
    if (needs_arena) emit(cg, "ez_default_arena, ");
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @uuid module --- */

static bool emit_uuid_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "generate") == 0) {
        emit(cg, "ez_uuid_generate(ez_default_arena)"); return true;
    }
    if (strcmp(func, "generate_compact") == 0) {
        emit(cg, "ez_uuid_generate_compact(ez_default_arena)"); return true;
    }
    if (strcmp(func, "is_valid") == 0) {
        emit(cg, "ez_uuid_is_valid(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")"); return true;
    }
    return false;
}

/* --- @regex module --- */

static bool emit_regex_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "match") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_regex_match(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "find") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_regex_find(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "find_all") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_regex_find_all(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "replace") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_regex_replace(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "split") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_regex_split(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @server module --- */

static bool emit_server_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "router") == 0) {
        emit(cg, "ez_server_router()");
        return true;
    }
    if (strcmp(func, "route") == 0 && node->data.call.arg_count == 4) {
        emit(cg, "ez_server_route(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ", (EzResponse (*)(EzRequest))");
        emit_expression(cg, node->data.call.args[3]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "listen") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_server_listen(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "text") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_server_text(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "json") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_server_json(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "html") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_server_html(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "redirect") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_server_redirect(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @http module --- */

static bool emit_http_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "get") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_http_get(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "post") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_http_post(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "put") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_http_put(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "delete") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_http_delete(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "head") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_http_head(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @net module --- */

static bool emit_net_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "dial") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_net_dial(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "close") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_net_close(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "send") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_net_send(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "recv") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_net_recv(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "listen") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_net_listen(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "accept") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_net_accept(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "set_timeout") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_net_set_timeout(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "resolve") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_net_resolve(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @encoding module --- */

static bool emit_encoding_call(CodeGen *cg, AstNode *node, const char *func) {
    emitf(cg, "ez_encoding_%s(ez_default_arena, ", func);
    emit_expression(cg, node->data.call.args[0]);
    emit(cg, ")");
    return true;
}

/* --- @crypto module --- */

static bool emit_crypto_call(CodeGen *cg, AstNode *node, const char *func) {
    emitf(cg, "ez_crypto_%s(ez_default_arena, ", func);
    emit_expression(cg, node->data.call.args[0]);
    emit(cg, ")");
    return true;
}

/* --- @bytes module --- */

static bool emit_bytes_call(CodeGen *cg, AstNode *node, const char *func) {
    bool needs_arena = (strcmp(func, "from_string") == 0 || strcmp(func, "from_hex") == 0 ||
        strcmp(func, "from_base64") == 0);
    bool needs_arena_ptr = (strcmp(func, "to_string") == 0 || strcmp(func, "to_hex") == 0 ||
        strcmp(func, "to_base64") == 0);
    emitf(cg, "ez_bytes_%s(", func);
    if (needs_arena || needs_arena_ptr) emit(cg, "ez_default_arena, ");
    if (needs_arena_ptr) {
        emit(cg, "&");
        emit_expression(cg, node->data.call.args[0]);
    } else {
        emit_expression(cg, node->data.call.args[0]);
    }
    emit(cg, ")");
    return true;
}

/* --- @binary module --- */

static bool emit_binary_call(CodeGen *cg, AstNode *node, const char *func) {
    bool is_encode = (strncmp(func, "encode", 6) == 0);
    bool is_decode = (strncmp(func, "decode", 6) == 0);
    /* Append _le for default little-endian if no endian suffix present */
    bool has_endian = strstr(func, "_le") || strstr(func, "_be");
    if (has_endian || strcmp(func, "encode_u8") == 0 || strcmp(func, "decode_u8") == 0) {
        emitf(cg, "ez_binary_%s(", func);
    } else if (is_encode || is_decode) {
        emitf(cg, "ez_binary_%s_le(", func);
    } else {
        emitf(cg, "ez_binary_%s(", func);
    }
    if (is_encode) emit(cg, "ez_default_arena, ");
    if (is_decode) emit(cg, "&");
    emit_expression(cg, node->data.call.args[0]);
    emit(cg, ")");
    return true;
}

/* --- @csv module --- */

static bool emit_csv_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "parse") == 0 || strcmp(func, "read") == 0) {
        emitf(cg, "ez_csv_%s(ez_default_arena, ", func);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "stringify") == 0) {
        emit(cg, "({ EzArray _csv_a = ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, "; ez_csv_stringify(ez_default_arena, &_csv_a); })");
        return true;
    }
    if (strcmp(func, "write") == 0) {
        emit(cg, "({ EzArray _csv_a = ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_csv_write(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_csv_a); })");
        return true;
    }
    return false;
}

/* --- @json module --- */

static bool emit_json_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "encode") == 0) {
        emit(cg, "ez_json_encode_map(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "decode") == 0) {
        emit(cg, "ez_json_decode(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "is_valid") == 0) {
        emit(cg, "ez_json_is_valid(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "pretty") == 0) {
        emit(cg, "ez_json_pretty_map(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @sqlite module --- */

static bool emit_sqlite_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "open") == 0) {
        emit(cg, "ez_sqlite_open(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "close") == 0) {
        emit(cg, "ez_sqlite_close(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "exec") == 0) {
        emit(cg, "ez_sqlite_exec(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "query") == 0) {
        emit(cg, "ez_sqlite_query(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @random module --- */

static bool emit_random_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "float") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "ez_random_float_unit()");
        } else if (node->data.call.arg_count == 2) {
            emit(cg, "ez_random_float_range(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
        }
        return true;
    }
    if (strcmp(func, "int") == 0) {
        if (node->data.call.arg_count == 1) {
            emit(cg, "ez_random_int_max(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
        } else if (node->data.call.arg_count == 2) {
            emit(cg, "ez_random_int_range(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
        }
        return true;
    }
    if (strcmp(func, "bool") == 0) { emit(cg, "ez_random_bool()"); return true; }
    if (strcmp(func, "byte") == 0) { emit(cg, "ez_random_byte()"); return true; }
    if (strcmp(func, "char") == 0) {
        if (node->data.call.arg_count == 2) {
            emit(cg, "ez_random_char_range(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
        } else {
            emit(cg, "ez_random_char()");
        }
        return true;
    }
    if (strcmp(func, "shuffle") == 0) {
        emit(cg, "ez_random_shuffle(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "sample") == 0) {
        emit(cg, "ez_random_sample(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "choice") == 0) {
        emit(cg, "({ int32_t _ri = ez_random_int_max(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ".len); *(__auto_type *)ez_array_get_ptr(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", _ri, __FILE__, __LINE__); })");
        return true;
    }
    return false;
}

/* --- @arrays module --- */

static bool emit_arrays_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "append") == 0 && node->data.call.arg_count == 2) {
        EzType *val_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[1]) : NULL;
        const char *c_elem = "__auto_type";
        if (val_t) {
            switch (val_t->kind) {
            case TK_INT: c_elem = "int64_t"; break;
            case TK_UINT: c_elem = "uint64_t"; break;
            case TK_FLOAT: c_elem = "double"; break;
            case TK_BOOL: c_elem = "bool"; break;
            case TK_STRING: c_elem = "EzString"; break;
            default: break;
            }
            if (val_t->kind == TK_STRUCT) {
                c_elem = ez_type_to_c_cg(cg, val_t->name);
            }
        }
        emitf(cg, "{ %s _av = ", c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_arrays_append(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_av); }");
        return true;
    }
    if (strcmp(func, "remove_at") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_arrays_remove_at(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "clear") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_arrays_clear(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "sort") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_arrays_sort(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "sort_desc") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_arrays_sort_desc(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "is_empty") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_arrays_is_empty(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "contains") == 0 && node->data.call.arg_count == 2) {
        EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
            strcmp(arr_t->element_type, "string") == 0) {
            emit(cg, "ez_arrays_contains_str(&");
        } else {
            emit(cg, "ez_arrays_contains_int(&");
        }
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "index_of") == 0 && node->data.call.arg_count == 2) {
        EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type &&
            strcmp(arr_t->element_type, "string") == 0) {
            emit(cg, "ez_arrays_index_of_str(&");
        } else {
            emit(cg, "ez_arrays_index_of_int(&");
        }
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    /* Generic: arrays.func(&arr, ...) */
    bool needs_arena = (strcmp(func, "reverse") == 0 || strcmp(func, "slice") == 0 ||
        strcmp(func, "concat") == 0);
    emitf(cg, "ez_arrays_%s(", func);
    if (needs_arena) emit(cg, "ez_default_arena, ");
    emit(cg, "&");
    emit_expression(cg, node->data.call.args[0]);
    for (int i = 1; i < node->data.call.arg_count; i++) {
        emit(cg, ", ");
        /* concat needs & on all array args */
        if (strcmp(func, "concat") == 0) emit(cg, "&");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @os module --- */

static bool emit_os_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "args") == 0) {
        emit(cg, "ez_os_args(ez_default_arena)"); return true;
    }
    if (strcmp(func, "get_env") == 0) {
        emit(cg, "ez_os_get_env(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "cwd") == 0) {
        emit(cg, "ez_os_cwd(ez_default_arena)"); return true;
    }
    if (strcmp(func, "hostname") == 0) {
        emit(cg, "ez_os_hostname(ez_default_arena)"); return true;
    }
    if (strcmp(func, "current_os") == 0) { emit(cg, "ez_os_current_os()"); return true; }
    if (strcmp(func, "arch") == 0) { emit(cg, "ez_os_arch()"); return true; }
    if (strcmp(func, "pid") == 0) { emit(cg, "ez_os_pid()"); return true; }
    if (strcmp(func, "exit") == 0) {
        emit(cg, "ez_os_exit(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "set_env") == 0) {
        emit(cg, "ez_os_set_env(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @io module --- */

static bool emit_io_call(CodeGen *cg, AstNode *node, const char *func) {
    bool is_fallible = (strcmp(func, "read_file") == 0 ||
        strcmp(func, "write_file") == 0 ||
        strcmp(func, "delete_file") == 0);
    if (is_fallible) {
        emitf(cg, "ez_io_%s_result(ez_default_arena, ", func);
        for (int i = 0; i < node->data.call.arg_count; i++) {
            if (i > 0) emit(cg, ", ");
            emit_expression(cg, node->data.call.args[i]);
        }
        emit(cg, ")");
        return true;
    }
    /* Non-fallible functions */
    emitf(cg, "ez_io_%s(", func);
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @strings module --- */

static bool emit_strings_call(CodeGen *cg, AstNode *node, const char *func) {
    bool needs_arena = (strcmp(func, "upper") == 0 || strcmp(func, "lower") == 0 ||
        strcmp(func, "trim") == 0 || strcmp(func, "trim_left") == 0 ||
        strcmp(func, "trim_right") == 0 || strcmp(func, "replace") == 0 ||
        strcmp(func, "repeat") == 0 || strcmp(func, "reverse") == 0 ||
        strcmp(func, "slice") == 0 || strcmp(func, "split") == 0 ||
        strcmp(func, "join") == 0);

    emitf(cg, "ez_strings_%s(", func);
    if (needs_arena) {
        emit(cg, "ez_default_arena, ");
    }
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @fmt module --- */

static bool emit_fmt_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "printf") == 0 && node->data.call.arg_count >= 1) {
        emit(cg, "printf(");
        AstNode *fmt_arg = node->data.call.args[0];
        if (fmt_arg->kind == NODE_STRING_VALUE) {
            emitf(cg, "\"%s\"", fmt_arg->data.string_value.value);
        } else {
            emit_expression(cg, fmt_arg);
            emit(cg, ".data");
        }
        emit_fmt_args(cg, node, 1);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sprintf") == 0 && node->data.call.arg_count >= 2) {
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
        emit_fmt_args(cg, node, 2);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "eprintln") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "fputc('\\n', stderr)");
        } else {
            AstNode *arg = node->data.call.args[0];
            emitf(cg, "ez_fmt_eprintln%s(", resolve_print_suffix(cg, arg));
            emit_expression(cg, arg);
            emit(cg, ")");
        }
        return true;
    }

    if (strcmp(func, "eprint") == 0 && node->data.call.arg_count > 0) {
        emit(cg, "ez_fmt_eprint_str(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

static bool emit_threads_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "spawn") == 0 && node->data.call.arg_count >= 1) {
        /* threads.spawn(()func) or threads.spawn(()func, arg) */
        if (node->data.call.arg_count == 1) {
            emit(cg, "ez_threads_spawn(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
        } else {
            emit(cg, "ez_threads_spawn_arg(");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
        }
        return true;
    }
    if (strcmp(func, "join") == 0) {
        emit(cg, "ez_threads_join(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "sleep_ms") == 0) {
        emit(cg, "ez_threads_sleep_ms(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "id") == 0) {
        emit(cg, "ez_threads_id()");
        return true;
    }
    if (strcmp(func, "mutex") == 0) {
        emit(cg, "ez_threads_mutex()");
        return true;
    }
    if (strcmp(func, "lock") == 0) {
        emit(cg, "ez_threads_lock(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "unlock") == 0) {
        emit(cg, "ez_threads_unlock(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "channel") == 0) {
        emit(cg, "ez_threads_channel(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "send") == 0) {
        emit(cg, "ez_threads_send(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "recv") == 0) {
        emit(cg, "ez_threads_recv(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "close") == 0) {
        emit(cg, "ez_threads_channel_close(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- Main call dispatcher --- */

static void emit_call_expression(CodeGen *cg, AstNode *node) {
    const char *module = NULL;
    const char *func = NULL;

    if (is_stdlib_call(node, &module, &func)) {
        /* No-module builtins (println, len, type_of, etc.) */
        /* Also handle std.println() — std module functions are builtins */
        if ((!module || strcmp(module, "std") == 0) && emit_builtin_call(cg, node, func)) return;

        /* Module dispatch table */
        typedef bool (*ModuleHandler)(CodeGen *, AstNode *, const char *);
        static const struct { const char *name; ModuleHandler handler; } modules[] = {
            {"mem",      emit_mem_call},
            {"math",     emit_math_call},
            {"maps",     emit_maps_call},
            {"time",     emit_time_call},
            {"uuid",     emit_uuid_call},
            {"encoding", emit_encoding_call},
            {"crypto",   emit_crypto_call},
            {"bytes",    emit_bytes_call},
            {"binary",   emit_binary_call},
            {"csv",      emit_csv_call},
            {"json",     emit_json_call},
            {"sqlite",   emit_sqlite_call},
            {"random",   emit_random_call},
            {"threads",  emit_threads_call},
            {"arrays",   emit_arrays_call},
            {"os",       emit_os_call},
            {"io",       emit_io_call},
            {"strings",  emit_strings_call},
            {"fmt",      emit_fmt_call},
            {"regex",    emit_regex_call},
            {"net",      emit_net_call},
            {"http",     emit_http_call},
            {"server",   emit_server_call},
        };
        if (module) {
            for (int i = 0; i < (int)(sizeof(modules) / sizeof(modules[0])); i++) {
                if (strcmp(module, modules[i].name) == 0) {
                    if (modules[i].handler(cg, node, func)) return;
                    break;
                }
            }
        }
    }

    /* Check for struct-namespaced function call: StructName.func() */
    if (node->data.call.function->kind == NODE_MEMBER_EXPR) {
        AstNode *obj = node->data.call.function->data.member.object;
        const char *member = node->data.call.function->data.member.member;
        if (obj->kind == NODE_LABEL) {
            const char *struct_name = obj->data.label.value;
            /* Try to find as a struct-namespaced function: StructName_func */
            char ns_name[256];
            snprintf(ns_name, sizeof(ns_name), "%s_%s", struct_name, member);
            AstNode *ns_func = find_func(cg, ns_name);
            if (ns_func) {
                emitf(cg, "ez_fn_%s_%s(", struct_name, member);
                for (int i = 0; i < node->data.call.arg_count; i++) {
                    if (i > 0) emit(cg, ", ");
                    emit_expression(cg, node->data.call.args[i]);
                }
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

    /* Look up function to check if it's a known function or a variable (function pointer) */
    AstNode *target_func = fn_name ? find_func(cg, fn_name) : NULL;

    if (fn_name && target_func) {
        /* Known function — use ez_fn_ prefix */
        emit(cg, "ez_fn_");
        emit(cg, fn_name);
    } else if (fn_name) {
        /* Not a known function — variable holding a function pointer (void *).
         * Cast to appropriate function pointer type based on arg count. */
        int nargs = node->data.call.arg_count;
        emit(cg, "((int64_t (*)(");
        for (int i = 0; i < nargs; i++) {
            if (i > 0) emit(cg, ", ");
            emit(cg, "int64_t");
        }
        if (nargs == 0) emit(cg, "void");
        emit(cg, "))");
        emit(cg, fn_name);
        emit(cg, ")");
    } else {
        emit_expression(cg, node->data.call.function);
    }

    /* Determine total args: provided + defaults */
    int total_args = node->data.call.arg_count;
    int param_count = target_func ? target_func->data.func_decl.param_count : 0;
    if (total_args < param_count) total_args = param_count;

    emit(cg, "(");
    for (int i = 0; i < total_args; i++) {
        if (i > 0) emit(cg, ", ");

        if (i < node->data.call.arg_count) {
            /* Provided argument */
            bool needs_addr = false;
            if (target_func && i < param_count) {
                needs_addr = target_func->data.func_decl.params[i].mutable;
            }
            if (needs_addr && node->data.call.args[i]->kind == NODE_LABEL) {
                const char *var_name = node->data.call.args[i]->data.label.value;
                if (is_mutable_param(cg, var_name)) {
                    /* Already a pointer — pass through */
                    emit(cg, var_name);
                } else {
                    emitf(cg, "&%s", var_name);
                }
            } else if (needs_addr && node->data.call.args[i]->kind == NODE_INDEX_EXPR) {
                /* Mutable param on array element: pass pointer to element */
                AstNode *idx_node = node->data.call.args[i];
                emit(cg, "(int64_t *)ez_array_get_ptr(&");
                emit_expression(cg, idx_node->data.index_expr.left);
                emit(cg, ", ");
                emit_expression(cg, idx_node->data.index_expr.index);
                emit(cg, ", __FILE__, __LINE__)");
            } else if (needs_addr && node->data.call.args[i]->kind == NODE_MEMBER_EXPR) {
                /* Mutable param on struct field: pass address of field */
                emit(cg, "&");
                emit_expression(cg, node->data.call.args[i]);
            } else {
                emit_expression(cg, node->data.call.args[i]);
            }
        } else if (target_func && i < param_count &&
                   target_func->data.func_decl.params[i].default_value) {
            /* Default value */
            emit_expression(cg, target_func->data.func_decl.params[i].default_value);
        } else {
            /* No arg and no default — emit zero */
            emit(cg, "0");
        }
    }
    emit(cg, ")");
}

/* --- Statement Emission --- */

static const char *extract_array_elem_type(const char *type_name) {
    if (!type_name || type_name[0] != '[') return NULL;
    static char buf[128];
    size_t len = strlen(type_name);
    if (len < 3) return NULL;
    /* Dynamic array "[int]" -> "int" */
    /* Fixed-size "[int,3]" -> "int" (strip size) */
    /* Nested "[[int]]" -> "[int]" */
    const char *start = type_name + 1;
    const char *end = type_name + len - 1;
    /* Find the comma for fixed-size, or just strip brackets */
    for (size_t i = 1; i < len - 1; i++) {
        if (type_name[i] == ',') {
            size_t elen = i - 1;
            memcpy(buf, start, elen);
            buf[elen] = '\0';
            return buf;
        }
    }
    size_t elen = (size_t)(end - start);
    memcpy(buf, start, elen);
    buf[elen] = '\0';
    return buf;
}

/* Extract size from fixed-size array type "[int,3]" -> 3, returns 0 if dynamic */
static int extract_array_size(const char *type_name) {
    if (!type_name || type_name[0] != '[') return 0;
    const char *comma = strchr(type_name, ',');
    if (!comma) return 0;
    return atoi(comma + 1);
}

/* Check if type is a nested array "[[...]]" */
static bool is_nested_array_type(const char *type_name) {
    return type_name && type_name[0] == '[' && type_name[1] == '[';
}

static void emit_var_declaration(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    const char *type_name = node->data.var_decl.type_name;
    const char *elem_type = extract_array_elem_type(type_name);

    if (elem_type) {
        int fixed_size = extract_array_size(type_name);
        if (fixed_size > 0) {
            /* Fixed-size array: use EzArray but initialized with exact capacity */
            emitf(cg, "EzArray %s = ", node->data.var_decl.name);
            if (node->data.var_decl.value) {
                emit_expression(cg, node->data.var_decl.value);
            } else {
                const char *c_elem_type = ez_type_to_c_cg(cg, elem_type);
                emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), %d)", c_elem_type, fixed_size);
            }
            emit(cg, ";\n");
            return;
        }

        if (is_nested_array_type(type_name)) {
            /* Nested array [[int]]: EzArray of EzArrays */
            emitf(cg, "EzArray %s = ", node->data.var_decl.name);
            if (node->data.var_decl.value) {
                emit_expression(cg, node->data.var_decl.value);
            } else {
                emitf(cg, "ez_array_new(ez_default_arena, sizeof(EzArray), 4)");
            }
            emit(cg, ";\n");
            return;
        }

        /* Dynamic array: use EzArray */
        const char *c_elem_type = ez_type_to_c_cg(cg, elem_type);
        emitf(cg, "EzArray %s = ", node->data.var_decl.name);
        if (node->data.var_decl.value &&
            node->data.var_decl.value->kind == NODE_ARRAY_VALUE &&
            node->data.var_decl.value->data.array_value.count == 0) {
            /* Empty array literal with type annotation — use correct elem size */
            emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", c_elem_type);
        } else if (node->data.var_decl.value) {
            emit_expression(cg, node->data.var_decl.value);
        } else {
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
            if (kt->kind == TK_INT || kt->kind == TK_UINT) c_kt = "int64_t";
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

    /* Register bigint variable for type tracking */
    if (type_name && is_bigint_type(type_name)) {
        register_bigint_var(cg, node->data.var_decl.name, type_name);
    }

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
        } else if (val->kind == NODE_ARRAY_VALUE) {
            c_type = "EzArray";
        } else if (val->kind == NODE_MAP_VALUE) {
            c_type = "EzMap";
        } else if (val->kind == NODE_STRUCT_VALUE) {
            c_type = ez_type_to_c_cg(cg, val->data.struct_value.name);
        } else if (val->kind == NODE_INFIX_EXPR) {
            /* Check type table for infix result type */
            EzType *infix_t = cg->type_table ? typetable_get(cg->type_table, val) : NULL;
            if (infix_t && infix_t->kind == TK_STRING) {
                c_type = "EzString";
            } else if (infix_t && infix_t->kind == TK_FLOAT) {
                c_type = "double";
            } else if (infix_t && infix_t->kind == TK_BOOL) {
                c_type = "bool";
            } else {
                c_type = "__auto_type";
            }
        } else if (val->kind == NODE_CALL_EXPR || val->kind == NODE_NEW_EXPR ||
                   val->kind == NODE_MEMBER_EXPR || val->kind == NODE_INDEX_EXPR) {
            /* Use __auto_type for function calls, new(), member access, and index
             * (needed for multi-var unpacking and nested array element extraction) */
            c_type = "__auto_type";
        } else if (val->kind == NODE_FUNC_REF) {
            /* Function reference — use __auto_type to capture the pointer type */
            c_type = "__auto_type";
        }
    }

    /* Detect ref() assignment — register as transparent reference (but not for function refs) */
    if (node->data.var_decl.value && node->data.var_decl.value->kind == NODE_CALL_EXPR) {
        AstNode *fn = node->data.var_decl.value->data.call.function;
        if (fn->kind == NODE_LABEL && strcmp(fn->data.label.value, "ref") == 0) {
            /* Only register as ref if the argument is a variable, not a function */
            if (node->data.var_decl.value->data.call.arg_count == 1) {
                AstNode *arg = node->data.var_decl.value->data.call.args[0];
                if (arg->kind == NODE_LABEL && !find_func(cg, arg->data.label.value)) {
                    register_ref_var(cg, node->data.var_decl.name);
                }
            }
        }
    }

    if (!node->data.var_decl.mutable) {
        emit(cg, "const ");
    }

    emitf(cg, "%s %s", c_type, node->data.var_decl.name);

    if (node->data.var_decl.value) {
        emit(cg, " = ");
        /* Bigint literal zero: emit zero constant instead of plain 0 */
        if (type_name && is_bigint_type(type_name) &&
            node->data.var_decl.value->kind == NODE_INT_VALUE &&
            node->data.var_decl.value->data.int_value.value == 0) {
            if (strcmp(type_name, "i128") == 0) emit(cg, "EZ_I128_ZERO");
            else if (strcmp(type_name, "u128") == 0) emit(cg, "EZ_U128_ZERO");
            else if (strcmp(type_name, "i256") == 0) emit(cg, "EZ_I256_ZERO");
            else if (strcmp(type_name, "u256") == 0) emit(cg, "EZ_U256_ZERO");
        } else if (type_name && is_bigint_type(type_name) &&
                   node->data.var_decl.value->kind == NODE_INT_VALUE) {
            /* Non-zero integer literal: use constructor */
            int64_t v = node->data.var_decl.value->data.int_value.value;
            const char *pfx = bigint_prefix(type_name);
            const char *from_suffix = (strcmp(type_name, "u128") == 0 || strcmp(type_name, "u256") == 0) ? "u64" : "i64";
            emitf(cg, "%s_from_%s(%lldLL)", pfx, from_suffix, (long long)v);
        } else {
            emit_expression(cg, node->data.var_decl.value);
        }
    } else {
        /* Zero-initialize when no value is provided */
        if (strcmp(c_type, "int64_t") == 0) emit(cg, " = 0");
        else if (strcmp(c_type, "double") == 0) emit(cg, " = 0.0");
        else if (strcmp(c_type, "bool") == 0) emit(cg, " = false");
        else if (strcmp(c_type, "EzString") == 0) emit(cg, " = (EzString){\"\", 0}");
        else if (strcmp(c_type, "EzArray") == 0) emit(cg, " = (EzArray){0}");
        else if (strcmp(c_type, "EzMap") == 0) emit(cg, " = (EzMap){0}");
        else if (strcmp(c_type, "ez_i128") == 0) emit(cg, " = EZ_I128_ZERO");
        else if (strcmp(c_type, "ez_u128") == 0) emit(cg, " = EZ_U128_ZERO");
        else if (strcmp(c_type, "ez_i256") == 0) emit(cg, " = EZ_I256_ZERO");
        else if (strcmp(c_type, "ez_u256") == 0) emit(cg, " = EZ_U256_ZERO");
        else emit(cg, " = {0}");
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
            const char *c_key = "EzString";
            if (left_t->key_type) {
                EzType *kt = type_from_name(left_t->key_type);
                if (kt->kind == TK_INT || kt->kind == TK_UINT) c_key = "int64_t";
            }
            emitf(cg, "{ %s _mk = ", c_key);
            emit_expression(cg, node->data.assign.target->data.index_expr.index);
            emitf(cg, "; %s _mv = ", c_val);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; ez_map_set(ez_default_arena, &");
            emit_expression(cg, left);
            emit(cg, ", &_mk, &_mv); }\n");
            return;
        }
    }

    /* Pointer dereference assignment: p^ = value → nil check + *p = value */
    if (node->data.assign.target->kind == NODE_POSTFIX_EXPR &&
        strcmp(node->data.assign.target->data.postfix.op, "^") == 0) {
        emit(cg, "{ __auto_type _dp = ");
        emit_expression(cg, node->data.assign.target->data.postfix.left);
        emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
            "\"nil pointer dereference\"); } *_dp", node->token.line);
        emitf(cg, " %s ", node->data.assign.op);
        emit_expression(cg, node->data.assign.value);
        emit(cg, "; }\n");
        return;
    }
    /* Pointer deref field assignment: p^.field = value → nil check + p->field = value */
    if (node->data.assign.target->kind == NODE_MEMBER_EXPR &&
        node->data.assign.target->data.member.object->kind == NODE_POSTFIX_EXPR &&
        strcmp(node->data.assign.target->data.member.object->data.postfix.op, "^") == 0) {
        AstNode *ptr = node->data.assign.target->data.member.object->data.postfix.left;
        const char *field = node->data.assign.target->data.member.member;
        emit(cg, "{ __auto_type _dp = ");
        emit_expression(cg, ptr);
        emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
            "\"nil pointer dereference\"); } _dp->%s", node->token.line, field);
        emitf(cg, " %s ", node->data.assign.op);
        emit_expression(cg, node->data.assign.value);
        emit(cg, "; }\n");
        return;
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

    /* Stack depth guard */
    emit_indent(cg);
    emit(cg, "ez_exit_func();\n");
    emit_indent(cg);

    if (node->data.return_stmt.count > 1 && cg->current_func) {
        emitf(cg, "return (EzMulti_%s){", cg->current_func->data.func_decl.name);
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            if (i > 0) emit(cg, ", ");
            emit_expression(cg, node->data.return_stmt.values[i]);
        }
        emit(cg, "};\n");
    } else if (node->data.return_stmt.count == 1 && cg->current_func &&
               cg->current_func->data.func_decl.return_type_count > 1) {
        /* Single value returned from multi-return function (e.g., or_return error propagation).
         * Pad with zeros for leading values, put the error last. */
        int rc = cg->current_func->data.func_decl.return_type_count;
        emitf(cg, "return (EzMulti_%s){", cg->current_func->data.func_decl.name);
        for (int i = 0; i < rc - 1; i++) {
            emit(cg, "{0}, ");
        }
        emit_expression(cg, node->data.return_stmt.values[0]);
        emit(cg, "};\n");
    } else if (cg->current_func &&
               cg->current_func->data.func_decl.return_type_count == 0) {
        /* Void function — return without value (e.g. or_return in main) */
        emit(cg, "return;\n");
        return;
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
        emitf(cg, "/* ezc: non-range for loop not yet supported */\n");
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

    /* Register bigint parameters for type tracking */
    for (int i = 0; i < node->data.func_decl.param_count; i++) {
        Param *param = &node->data.func_decl.params[i];
        if (param->type_name && is_bigint_type(param->type_name)) {
            register_bigint_var(cg, param->name, param->type_name);
        }
    }

    /* Emit named return variable declarations (zero-initialized) */
    if (node->data.func_decl.return_names) {
        for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
            if (node->data.func_decl.return_names[i]) {
                const char *c_t = ez_type_to_c_cg(cg, node->data.func_decl.return_types[i]);
                emit_indent(cg);
                if (strcmp(c_t, "EzString") == 0) {
                    emitf(cg, "EzString %s = ez_string_lit(\"\");\n",
                        node->data.func_decl.return_names[i]);
                } else if (strncmp(c_t, "EzStruct_", 9) == 0 || strncmp(c_t, "EzEnum_", 7) == 0) {
                    emitf(cg, "%s %s = {0};\n", c_t,
                        node->data.func_decl.return_names[i]);
                } else {
                    emitf(cg, "%s %s = 0;\n", c_t,
                        node->data.func_decl.return_names[i]);
                }
            }
        }
    }

    if (node->data.func_decl.body) {
        /* Stack depth guard */
        emit_indent(cg);
        emitf(cg, "ez_enter_func(__FILE__, %d);\n", node->token.line);
        emit_block(cg, node->data.func_decl.body);
        /* Emit ensure cleanup at end of function (for implicit returns) */
        emit_ensure_cleanup(cg);
        emit_indent(cg);
        emit(cg, "ez_exit_func();\n");
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

        if (coll_t && coll_t->kind == TK_MAP) {
            /* for_each on map — iterate occupied slots with internal counter */
            static int map_iter_counter = 0;
            char mi_name[32];
            snprintf(mi_name, sizeof(mi_name), "_ez_mi%d", map_iter_counter++);

            const char *c_key = "EzString";
            const char *c_val = "int64_t";
            if (coll_t->key_type) {
                EzType *kt = type_from_name(coll_t->key_type);
                if (kt->kind == TK_INT || kt->kind == TK_UINT) c_key = "int64_t";
            }
            if (coll_t->value_type) {
                EzType *vt = type_from_name(coll_t->value_type);
                if (vt->kind == TK_FLOAT) c_val = "double";
                else if (vt->kind == TK_STRING) c_val = "EzString";
                else if (vt->kind == TK_BOOL) c_val = "bool";
            }

            /* Iterate in insertion order using the order array */
            char slot_name[32];
            snprintf(slot_name, sizeof(slot_name), "_ez_sl%d", map_iter_counter - 1);
            emitf(cg, "for (int32_t %s = 0; %s < ", mi_name, mi_name);
            emit_expression(cg, coll);
            emitf(cg, ".order_len; %s++) {\n", mi_name);
            cg->indent++;
            emit_indent(cg);
            emitf(cg, "int32_t %s = ", slot_name);
            emit_expression(cg, coll);
            emitf(cg, ".order[%s];\n", mi_name);

            if (node->data.for_each.index_name) {
                /* Two-var form: for_each k, v in map */
                if (strcmp(node->data.for_each.index_name, "_") != 0) {
                    emit_indent(cg);
                    emitf(cg, "%s %s = *(%s *)ez_map_key_at(&",
                        c_key, node->data.for_each.index_name, c_key);
                    emit_expression(cg, coll);
                    emitf(cg, ", %s);\n", slot_name);
                }
                if (strcmp(node->data.for_each.var_name, "_") != 0) {
                    emit_indent(cg);
                    emitf(cg, "%s %s = *(%s *)ez_map_value_at(&",
                        c_val, node->data.for_each.var_name, c_val);
                    emit_expression(cg, coll);
                    emitf(cg, ", %s);\n", slot_name);
                }
            } else {
                /* One-var form: for_each key in map (keys only) */
                emit_indent(cg);
                emitf(cg, "%s %s = *(%s *)ez_map_key_at(&",
                    c_key, node->data.for_each.var_name, c_key);
                emit_expression(cg, coll);
                emitf(cg, ", %s);\n", slot_name);
            }
        } else if (coll_t && coll_t->kind == TK_STRING) {
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
                else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, coll_t->element_type);
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
        EzType *when_val_t = cg->type_table ? typetable_get(cg->type_table, val) : NULL;
        bool when_is_string = (when_val_t && when_val_t->kind == TK_STRING);
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
                    if (r->data.range_expr.step) {
                        emit(cg, " && (");
                        emit_expression(cg, val);
                        emit(cg, " - ");
                        emit_expression(cg, r->data.range_expr.start);
                        emit(cg, ") % ");
                        emit_expression(cg, r->data.range_expr.step);
                        emit(cg, " == 0");
                    }
                    emit(cg, ")");
                } else if (when_is_string) {
                    emit(cg, "ez_string_eq(");
                    emit_expression(cg, val);
                    emit(cg, ", ");
                    emit_expression(cg, wc->values[j]);
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
    case NODE_MODULE_DECL:
        /* Module declarations are informational only */
        break;
    case NODE_IMPORT_STMT:
        /* Imports are handled during the preamble scan */
        break;
    case NODE_USING_STMT:
        /* Using is handled during the preamble scan */
        break;
    default:
        emit_indent(cg);
        emitf(cg, "/* ezc: unhandled statement kind %d at %s:%d */\n",
            node->kind, cg->file, node->token.line);
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
    cg.ref_vars = NULL;
    cg.ref_var_count = 0;
    cg.ref_var_cap = 0;
    cg.bigint_var_names = NULL;
    cg.bigint_var_types = NULL;
    cg.bigint_var_count = 0;
    cg.bigint_var_cap = 0;
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
    emit(cg, "#include \"ez_math.h\"\n");
    emit(cg, "#include \"ez_strings.h\"\n");
    emit(cg, "#include \"ez_io.h\"\n");
    emit(cg, "#include \"ez_maps.h\"\n");
    emit(cg, "#include \"ez_os.h\"\n");
    emit(cg, "#include \"ez_arrays.h\"\n");
    emit(cg, "#include \"ez_random.h\"\n");
    emit(cg, "#include \"ez_time.h\"\n");
    emit(cg, "#include \"ez_uuid.h\"\n");
    emit(cg, "#include \"ez_encoding.h\"\n");
    emit(cg, "#include \"ez_crypto.h\"\n");
    emit(cg, "#include \"ez_bytes.h\"\n");
    emit(cg, "#include \"ez_binary.h\"\n");
    emit(cg, "#include \"ez_csv.h\"\n");
    emit(cg, "#include \"ez_json.h\"\n");
    emit(cg, "#include \"ez_sqlite.h\"\n");
    emit(cg, "#include \"ez_threads.h\"\n");
    emit(cg, "#include \"ez_regex.h\"\n");
    emit(cg, "#include \"ez_net.h\"\n");
    emit(cg, "#include \"ez_http.h\"\n");
    emit(cg, "#include \"ez_server.h\"\n");
    emit(cg, "#include \"ez_bigint.h\"\n");
    emit(cg, "\n");

    /* Emit enum type definitions FIRST (before structs, since structs may reference enums) */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_ENUM_DECL) {
            /* Register enum name */
            if (cg->enum_count >= cg->enum_cap) {
                cg->enum_cap = cg->enum_cap ? cg->enum_cap * 2 : 8;
                cg->enum_names = realloc(cg->enum_names, sizeof(const char *) * cg->enum_cap);
            }
            cg->enum_names[cg->enum_count++] = stmt->data.enum_decl.name;

            /* Check if this is a string enum */
            bool is_string_enum = false;
            if (stmt->data.enum_decl.base_type &&
                strcmp(stmt->data.enum_decl.base_type, "string") == 0) {
                is_string_enum = true;
            } else {
                for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                    if (stmt->data.enum_decl.values[j].value &&
                        stmt->data.enum_decl.values[j].value->kind == NODE_STRING_VALUE) {
                        is_string_enum = true;
                        break;
                    }
                }
            }

            if (is_string_enum) {
                emitf(cg, "typedef EzString EzEnum_%s;\n", stmt->data.enum_decl.name);
                for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                    EnumVal *ev = &stmt->data.enum_decl.values[j];
                    const char *str_val = ev->name;
                    if (ev->value && ev->value->kind == NODE_STRING_VALUE) {
                        str_val = ev->value->data.string_value.value;
                    }
                    emitf(cg, "#define EzEnum_%s_%s ((EzString){ \"%s\", %d })\n",
                        stmt->data.enum_decl.name, ev->name,
                        str_val, (int)strlen(str_val));
                }
                emit(cg, "\n");
            } else {
                bool is_flags = stmt->data.enum_decl.is_flags;
                emitf(cg, "typedef enum {\n");
                for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                    EnumVal *ev = &stmt->data.enum_decl.values[j];
                    emitf(cg, "    EzEnum_%s_%s", stmt->data.enum_decl.name, ev->name);
                    if (ev->value) {
                        emit(cg, " = ");
                        emit_expression(cg, ev->value);
                    } else if (is_flags) {
                        emitf(cg, " = %d", 1 << j);
                    } else {
                        emitf(cg, " = %d", j);
                    }
                    emit(cg, ",\n");
                }
                emitf(cg, "} EzEnum_%s;\n\n", stmt->data.enum_decl.name);
            }
        }
    }

    /* Collect struct declarations and emit in dependency order (topological sort).
     * Structs that reference other structs as value fields must come after them. */
    {
        int struct_count = 0;
        AstNode *structs[256];
        for (int i = 0; i < program->data.program.stmt_count; i++) {
            if (program->data.program.stmts[i]->kind == NODE_STRUCT_DECL &&
                struct_count < 256) {
                structs[struct_count++] = program->data.program.stmts[i];
            }
        }

        /* Simple topological sort: repeatedly emit structs with no unresolved deps */
        bool emitted[256] = {false};
        int emit_count = 0;
        for (int pass = 0; pass < struct_count && emit_count < struct_count; pass++) {
            for (int i = 0; i < struct_count; i++) {
                if (emitted[i]) continue;
                AstNode *s = structs[i];
                bool deps_met = true;
                for (int j = 0; j < s->data.struct_decl.field_count; j++) {
                    const char *ft = s->data.struct_decl.fields[j].type_name;
                    if (!ft) continue;
                    /* Check if this field type is another user struct */
                    for (int k = 0; k < struct_count; k++) {
                        if (k != i && !emitted[k] &&
                            strcmp(structs[k]->data.struct_decl.name, ft) == 0) {
                            deps_met = false;
                            break;
                        }
                    }
                    if (!deps_met) break;
                }
                if (deps_met) {
                    emitted[i] = true;
                    emit_count++;
                    emitf(cg, "typedef struct {\n");
                    for (int j = 0; j < s->data.struct_decl.field_count; j++) {
                        StructField *f = &s->data.struct_decl.fields[j];
                        emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg, f->type_name), f->name);
                    }
                    emitf(cg, "} EzStruct_%s;\n\n", s->data.struct_decl.name);
                }
            }
        }
        /* If any structs couldn't be emitted (circular deps), emit them anyway */
        for (int i = 0; i < struct_count; i++) {
            if (!emitted[i]) {
                AstNode *s = structs[i];
                emitf(cg, "typedef struct {\n");
                for (int j = 0; j < s->data.struct_decl.field_count; j++) {
                    StructField *f = &s->data.struct_decl.fields[j];
                    emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg, f->type_name), f->name);
                }
                emitf(cg, "} EzStruct_%s;\n\n", s->data.struct_decl.name);
            }
        }
    }

    /* (Enum typedefs already emitted above, before struct definitions) */

    /* Collect all function declarations (including struct-namespaced) */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL) {
            if (cg->func_count >= cg->func_cap) {
                cg->func_cap = cg->func_cap ? cg->func_cap * 2 : 16;
                cg->all_funcs = realloc(cg->all_funcs, sizeof(AstNode *) * cg->func_cap);
            }
            cg->all_funcs[cg->func_count++] = stmt;
        }
        /* Collect struct-namespaced functions with prefixed names */
        if (stmt->kind == NODE_STRUCT_DECL) {
            for (int j = 0; j < stmt->data.struct_decl.func_count; j++) {
                AstNode *fn = stmt->data.struct_decl.funcs[j].func_decl;
                if (fn && fn->kind == NODE_FUNC_DECL) {
                    const char *sn = stmt->data.struct_decl.name;
                    const char *fn_name = fn->data.func_decl.name;
                    size_t ns_len = strlen(sn) + 1 + strlen(fn_name) + 1;
                    char *ns_name = malloc(ns_len);
                    snprintf(ns_name, ns_len, "%s_%s", sn, fn_name);
                    fn->data.func_decl.name = ns_name;

                    if (cg->func_count >= cg->func_cap) {
                        cg->func_cap = cg->func_cap ? cg->func_cap * 2 : 16;
                        cg->all_funcs = realloc(cg->all_funcs, sizeof(AstNode *) * cg->func_cap);
                    }
                    cg->all_funcs[cg->func_count++] = fn;
                }
            }
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

    /* Emit forward declarations for all functions (including struct-namespaced) */
    for (int i = 0; i < cg->func_count; i++) {
        AstNode *stmt = cg->all_funcs[i];
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

    /* Emit function definitions (top-level statements) */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_IMPORT_STMT && stmt->kind != NODE_USING_STMT) {
            emit_statement(cg, stmt);
        }
    }

    /* Emit struct-namespaced function definitions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_STRUCT_DECL) {
            for (int j = 0; j < stmt->data.struct_decl.func_count; j++) {
                AstNode *fn = stmt->data.struct_decl.funcs[j].func_decl;
                if (fn && fn->kind == NODE_FUNC_DECL) {
                    emit_statement(cg, fn);
                }
            }
        }
    }

    /* Emit C main() */
    emit(cg, "int main(int argc, char **argv) {\n");
    emit(cg, "    (void)argc; (void)argv;\n");
    emit(cg, "    ez_runtime_init();\n");
    emit(cg, "    ez_os_init(argc, argv);\n");
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
