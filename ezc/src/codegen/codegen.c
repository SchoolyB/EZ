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
static void emit_to_string(CodeGen *cg, AstNode *arg);

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

/* Internal compiler error — emit a clear message instead of segfaulting.
 * Used when a type lookup unexpectedly returns NULL. */
__attribute__((unused))
static void codegen_ice(const char *context, const char *file, int line) {
    fflush(stdout);
    fprintf(stderr, "internal compiler error: %s (at %s:%d)\n"
        "This is a bug in the EZ compiler. Please report it.\n",
        context, file ? file : "<unknown>", line);
    exit(1);
}

/* Check if a name collides with a C keyword and mangle it if so.
 * Uses a rotating pool of static buffers so multiple calls can appear
 * in one format string (up to 4 simultaneous uses). */
static bool is_c_keyword(const char *name) {
    static const char *keywords[] = {
        "auto", "break", "case", "char", "const", "continue", "default",
        "do", "double", "else", "enum", "extern", "float", "for", "goto",
        "if", "inline", "int", "long", "register", "restrict", "return",
        "short", "signed", "sizeof", "static", "struct", "switch",
        "typedef", "union", "unsigned", "void", "volatile", "while",
        "bool", "true", "false", "NULL",
        NULL
    };
    for (const char **kw = keywords; *kw; kw++) {
        if (strcmp(name, *kw) == 0) return true;
    }
    return false;
}

static const char *safe_name(const char *name) {
    if (!name || !is_c_keyword(name)) return name;
    static char bufs[4][256];
    static int idx = 0;
    int i = idx++ & 3;
    snprintf(bufs[i], sizeof(bufs[i]), "_ez_%s", name);
    return bufs[i];
}

/* Map EZ type name to C type */
/* Return a type string with any '?' replaced by the active wildcard
 * binding. Returns the original pointer if no binding is active or the
 * string has no wildcard. The substituted string lives in a small ring
 * of static buffers so a handful of nested calls can each keep their
 * result alive simultaneously. */
static const char *multi_base_name(const char *fn_name);
static const char *multi_ret_name(AstNode *func);
static const char *cg_effective_type_str(CodeGen *cg, const char *type_name) {
    if (!type_name || !cg || !cg->wildcard_binding) return type_name;
    if (!strchr(type_name, '?')) return type_name;
    size_t cl = strlen(cg->wildcard_binding);
    /* Count wildcards to compute exact output size */
    size_t need = 0;
    for (const char *q = type_name; *q; q++) {
        if (*q == '?') need += cl;
        else need += 1;
    }
    char *out = malloc(need + 1);
    char *w = out;
    for (const char *q = type_name; *q; q++) {
        if (*q == '?') {
            memcpy(w, cg->wildcard_binding, cl);
            w += cl;
        } else {
            *w++ = *q;
        }
    }
    *w = '\0';
    return out;
}

static const char *ez_type_to_c_cg(CodeGen *cg, const char *type_name) {
    if (!type_name) return "int64_t";

    /* Wildcard substitution (#1443): if '?' appears in the type
     * string while a generic instantiation is active, rewrite via
     * cg_effective_type_str and recurse through the normal mapping. */
    if (cg && cg->wildcard_binding && strchr(type_name, '?')) {
        const char *sub = cg_effective_type_str(cg, type_name);
        const char *saved = cg->wildcard_binding;
        cg->wildcard_binding = NULL;
        const char *resolved = ez_type_to_c_cg(cg, sub);
        cg->wildcard_binding = saved;
        return resolved;
    }

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
    if (strcmp(type_name, "Error") == 0 || strcmp(type_name, "error") == 0) return "EzError *";
    if (strcmp(type_name, "func") == 0)  return "void *"; /* generic fn ptr — cast at call site */

    /* Pointer type: ^T — use C pointer (ring buffer avoids aliasing on recursion) */
    if (type_name[0] == '^') {
        static char ptrbufs[4][256];
        static int ptridx = 0;
        char *buf = ptrbufs[ptridx++ & 3];
        const char *pointee = ez_type_to_c_cg(cg, type_name + 1);
        snprintf(buf, 256, "%s *", pointee);
        return buf;
    }

    /* Array type: [T] — use EzArray */
    if (type_name[0] == '[') {
        return "EzArray";
    }

    /* Map type: map[K:V] — use EzMap */
    if (strncmp(type_name, "map[", 4) == 0) {
        return "EzMap";
    }

    /* Qualified type name: module.Type → strip module prefix */
    const char *dot = strchr(type_name, '.');
    if (dot) {
        const char *base = dot + 1;
        static char buf[256];
        if (cg && codegen_is_enum(cg, base)) {
            snprintf(buf, sizeof(buf), "EzEnum_%s", base);
        } else {
            snprintf(buf, sizeof(buf), "EzStruct_%s", base);
        }
        return buf;
    }

    /* If starts with uppercase, it's a user-defined type */
    /* Also handle module-prefixed types: lib_Point, mod_Color */
    bool is_user_type = (type_name[0] >= 'A' && type_name[0] <= 'Z');
    if (!is_user_type) {
        const char *us = strchr(type_name, '_');
        if (us && us[1] >= 'A' && us[1] <= 'Z') is_user_type = true;
    }
    if (is_user_type) {
        static char buf[256];
        const char *resolved = type_name;
        /* Resolve unprefixed names from 'import and use' */
        if (cg && type_name[0] >= 'A' && type_name[0] <= 'Z' && !strchr(type_name, '_')) {
            /* Check if a prefixed version exists in struct declarations */
            for (int si = 0; si < cg->struct_decl_count; si++) {
                const char *dn = cg->struct_decls[si]->data.struct_decl.name;
                const char *us = strrchr(dn, '_');
                if (us && strcmp(us + 1, type_name) == 0) {
                    resolved = dn;
                    break;
                }
            }
            /* Check enums too */
            if (resolved == type_name) {
                for (int ei = 0; ei < cg->enum_count; ei++) {
                    const char *en = cg->enum_names[ei];
                    const char *us = strrchr(en, '_');
                    if (us && strcmp(us + 1, type_name) == 0) {
                        resolved = en;
                        break;
                    }
                }
            }
        }
        if (cg && codegen_is_enum(cg, resolved)) {
            snprintf(buf, sizeof(buf), "EzEnum_%s", resolved);
        } else {
            snprintf(buf, sizeof(buf), "EzStruct_%s", resolved);
        }
        return buf;
    }

    return type_name;
}

/* Resolve an EZ type to its C type for map key/value storage.
 * Uses ez_type_to_c_cg for struct/array/map types, hardcoded for primitives.
 * Routes the input through cg_effective_type_str so '?' inside a generic
 * instantiation resolves to the active wildcard binding (#1463). */
static const char *ez_map_elem_c_type(CodeGen *cg, const char *ez_tn) {
    if (!ez_tn) return "int64_t";
    ez_tn = cg_effective_type_str(cg, ez_tn);
    EzType *t = type_from_name(ez_tn);
    if (!t) return "int64_t";
    switch (t->kind) {
    case TK_FLOAT:   return "double";
    case TK_STRING:  return "EzString";
    case TK_BOOL:    return "bool";
    case TK_CHAR:    return "int32_t";
    case TK_BYTE:    return "uint8_t";
    case TK_ARRAY:   return "EzArray";
    case TK_MAP:     return "EzMap";
    case TK_STRUCT:  return ez_type_to_c_cg(cg, ez_tn);
    case TK_POINTER: return ez_type_to_c_cg(cg, ez_tn);
    default:         return "int64_t";
    }
}

/* --- Deep copy machinery (#1465, #1466) ---
 *
 * A value is "needs-deep-copy" iff reading one C-level copy of it
 * would share mutable backing storage with the source. That covers
 * arrays (EzArray header aliases data), maps (EzMap header aliases
 * keys/values/states/order), and any struct that transitively holds a
 * field of either. Pointers are deliberately left to alias — following
 * the pointee would surprise users, loop on cycles, and doesn't match
 * how any real language treats pointer copy.
 *
 * Three mutually-recursive emitters handle each collection kind, and
 * emit_value_deep_copy dispatches based on the EZ type string. All of
 * them take a `src_var` naming a C lvalue holding the source value,
 * and emit a single C expression (usually a GCC statement expression)
 * that evaluates to a fully independent copy. */

static AstNode *find_struct_decl(CodeGen *cg, const char *name);

static bool type_needs_deep_copy(CodeGen *cg, const char *ez_tn) {
    if (!ez_tn || !*ez_tn) return false;
    if (ez_tn[0] == '[') return true;
    if (strncmp(ez_tn, "map[", 4) == 0) return true;
    if (ez_tn[0] == '^') return false; /* pointers alias — see header comment */
    AstNode *sdecl = find_struct_decl(cg, ez_tn);
    if (!sdecl) return false;
    for (int i = 0; i < sdecl->data.struct_decl.field_count; i++) {
        const char *ft = sdecl->data.struct_decl.fields[i].type_name;
        if (type_needs_deep_copy(cg, ft)) return true;
    }
    return false;
}

/* Shared counter for unique temp names across all deep-copy emitters. */
static int deep_copy_tag_counter = 0;
static int next_dc_tag(void) { return deep_copy_tag_counter++; }

static void emit_value_deep_copy(CodeGen *cg, const char *ez_tn, const char *src_var);

static void emit_array_deep_copy(CodeGen *cg, const char *ez_tn, const char *src_var) {
    size_t len = ez_tn ? strlen(ez_tn) : 0;
    if (len < 3 || ez_tn[0] != '[' || ez_tn[len - 1] != ']') {
        emitf(cg, "ez_array_copy(ez_default_arena, &%s)", src_var);
        return;
    }

    /* Extract element type name from "[T]" (dropping any ",N" sized tail). */
    char elem_tn[256];
    size_t elen = len - 2;
    if (elen >= sizeof(elem_tn)) elen = sizeof(elem_tn) - 1;
    memcpy(elem_tn, ez_tn + 1, elen);
    elem_tn[elen] = '\0';
    char *comma = strchr(elem_tn, ',');
    if (comma) *comma = '\0';

    if (!type_needs_deep_copy(cg, elem_tn)) {
        /* Flat element type — the shallow bulk memcpy in ez_array_copy
         * is already correct. */
        emitf(cg, "ez_array_copy(ez_default_arena, &%s)", src_var);
        return;
    }

    /* Element needs its own deep copy. Allocate a fresh outer and walk
     * each slot, recursively deep-copying the element in place. */
    const char *c_elem = ez_type_to_c_cg(cg, elem_tn);
    int t = next_dc_tag();
    emitf(cg,
        "({ EzArray _ds%d = %s; "
        "EzArray _dd%d = ez_array_new(ez_default_arena, sizeof(%s), _ds%d.len); "
        "_dd%d.len = _ds%d.len; "
        "for (int32_t _di%d = 0; _di%d < _ds%d.len; _di%d++) { "
        "%s _de%d = ",
        t, src_var,
        t, c_elem, t,
        t, t,
        t, t, t, t,
        c_elem, t);

    char inner_var[192];
    snprintf(inner_var, sizeof(inner_var),
        "((%s *)_ds%d.data)[_di%d]", c_elem, t, t);
    emit_value_deep_copy(cg, elem_tn, inner_var);

    emitf(cg,
        "; ((%s *)_dd%d.data)[_di%d] = _de%d; "
        "} _dd%d; })",
        c_elem, t, t, t, t);
}

static void emit_map_deep_copy(CodeGen *cg, const char *ez_tn, const char *src_var) {
    /* Parse "map[K:V]" into its two slots. */
    if (!ez_tn || strncmp(ez_tn, "map[", 4) != 0) {
        emitf(cg, "ez_map_copy(ez_default_arena, &%s)", src_var);
        return;
    }
    size_t len = strlen(ez_tn);
    if (len < 7 || ez_tn[len - 1] != ']') {
        emitf(cg, "ez_map_copy(ez_default_arena, &%s)", src_var);
        return;
    }
    const char *start = ez_tn + 4;
    const char *colon = strchr(start, ':');
    if (!colon) {
        emitf(cg, "ez_map_copy(ez_default_arena, &%s)", src_var);
        return;
    }
    char key_tn[128];
    char val_tn[256];
    size_t klen = (size_t)(colon - start);
    if (klen >= sizeof(key_tn)) klen = sizeof(key_tn) - 1;
    memcpy(key_tn, start, klen);
    key_tn[klen] = '\0';
    size_t vlen = len - 4 - klen - 1 - 1; /* drop "map[", K, ":", "]" */
    if (vlen >= sizeof(val_tn)) vlen = sizeof(val_tn) - 1;
    memcpy(val_tn, colon + 1, vlen);
    val_tn[vlen] = '\0';

    if (!type_needs_deep_copy(cg, val_tn)) {
        /* Value type is flat — the existing runtime helper is correct.
         * Keys never need deep copying in our model. */
        emitf(cg, "ez_map_copy(ez_default_arena, &%s)", src_var);
        return;
    }

    /* Value type needs recursion. Iterate the source in insertion order,
     * deep-copy each value, and insert into a fresh map. */
    const char *c_key = ez_map_elem_c_type(cg, key_tn);
    const char *c_val = ez_map_elem_c_type(cg, val_tn);
    int t = next_dc_tag();
    emitf(cg,
        "({ EzMap _ms%d = %s; "
        "EzMap _md%d = ez_map_new(ez_default_arena, _ms%d.key_size, _ms%d.value_size, "
        "_ms%d.order_len > 4 ? _ms%d.order_len * 2 : 8); "
        "for (int32_t _mi%d = 0; _mi%d < _ms%d.order_len; _mi%d++) { "
        "int32_t _mslot%d = _ms%d.order[_mi%d]; "
        "%s _mk%d = *(%s *)ez_map_key_at(&_ms%d, _mslot%d); "
        "%s _mvs%d = *(%s *)ez_map_value_at(&_ms%d, _mslot%d); "
        "%s _mvd%d = ",
        t, src_var,
        t, t, t, t, t,
        t, t, t, t,
        t, t, t,
        c_key, t, c_key, t, t,
        c_val, t, c_val, t, t,
        c_val, t);

    char src_val_var[64];
    snprintf(src_val_var, sizeof(src_val_var), "_mvs%d", t);
    emit_value_deep_copy(cg, val_tn, src_val_var);

    emitf(cg,
        "; ez_map_set(ez_default_arena, &_md%d, &_mk%d, &_mvd%d); "
        "} _md%d; })",
        t, t, t, t);
}

static void emit_struct_deep_copy(CodeGen *cg, const char *struct_tn, const char *src_var) {
    AstNode *sdecl = find_struct_decl(cg, struct_tn);
    if (!sdecl) {
        /* No decl info — bitwise copy is the best we can do. */
        emitf(cg, "%s", src_var);
        return;
    }
    const char *c_struct = ez_type_to_c_cg(cg, struct_tn);
    int t = next_dc_tag();
    emitf(cg,
        "({ %s _ss%d = %s; %s _sd%d = _ss%d; ",
        c_struct, t, src_var, c_struct, t, t);
    for (int i = 0; i < sdecl->data.struct_decl.field_count; i++) {
        StructField *f = &sdecl->data.struct_decl.fields[i];
        if (!f->type_name || !f->name) continue;
        if (!type_needs_deep_copy(cg, f->type_name)) continue;
        char src_field[192];
        snprintf(src_field, sizeof(src_field), "_ss%d.%s", t, f->name);
        emitf(cg, "_sd%d.%s = ", t, f->name);
        emit_value_deep_copy(cg, f->type_name, src_field);
        emit(cg, "; ");
    }
    emitf(cg, "_sd%d; })", t);
}

static void emit_value_deep_copy(CodeGen *cg, const char *ez_tn, const char *src_var) {
    if (!type_needs_deep_copy(cg, ez_tn)) {
        /* Primitive / pointer / scalar struct — C value copy is correct. */
        emitf(cg, "%s", src_var);
        return;
    }
    if (ez_tn[0] == '[') {
        emit_array_deep_copy(cg, ez_tn, src_var);
        return;
    }
    if (strncmp(ez_tn, "map[", 4) == 0) {
        emit_map_deep_copy(cg, ez_tn, src_var);
        return;
    }
    /* Must be a struct that needs recursion. */
    emit_struct_deep_copy(cg, ez_tn, src_var);
}

/* Entry point used by the three sites that hold an AstNode for the
 * source array (copy() builtin, var_decl copy-on-assign, assignment
 * copy-on-assign). Evaluates the AstNode once into a temp, then hands
 * the temp name to emit_value_deep_copy with a reconstructed "[elem]"
 * type string. */
static void emit_deep_array_copy(CodeGen *cg, AstNode *src_node, const char *elem_type_name) {
    int t = next_dc_tag();
    emitf(cg, "({ EzArray _dtop%d = ", t);
    emit_expression(cg, src_node);
    emit(cg, "; ");
    char src_var[32];
    snprintf(src_var, sizeof(src_var), "_dtop%d", t);
    char full_tn[256];
    snprintf(full_tn, sizeof(full_tn), "[%s]", elem_type_name ? elem_type_name : "");
    emit_value_deep_copy(cg, full_tn, src_var);
    emit(cg, "; })");
}

/* Resolve an import alias to the actual module name, or return the name itself */
static const char *resolve_alias(CodeGen *cg, const char *name) {
    for (int i = 0; i < cg->alias_count; i++) {
        if (strcmp(cg->alias_names[i], name) == 0) return cg->alias_modules[i];
    }
    return name;
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
        const char *name = safe_name(node->data.label.value);
        const char *raw = node->data.label.value;
        /* #1519: bare stdlib constants from using-modules */
        static const struct { const char *n; const char *mod; const char *val; } _cg_consts[] = {
            {"PI","math","3.14159265358979323846"},{"E","math","2.71828182845904523536"},
            {"TAU","math","6.28318530717958647692"},{"PHI","math","1.61803398874989484820"},
            {"SQRT2","math","1.41421356237309504880"},{"LN2","math","0.69314718055994530942"},
            {"LN10","math","2.30258509299404568402"},{"INF","math","(1.0/0.0)"},
            {"NEG_INF","math","(-1.0/0.0)"},{"EPSILON","math","2.2204460492503131e-16"},
            {"MAC_OS","os","0"},{"LINUX","os","1"},{"WINDOWS","os","2"},{"OTHER","os","3"},
            {NULL,NULL,NULL}
        };
        bool emitted_const = false;
        for (int ui = 0; ui < cg->using_module_count && !emitted_const; ui++) {
            const char *real_mod = resolve_alias(cg, cg->using_modules[ui]);
            for (int ci = 0; _cg_consts[ci].n; ci++) {
                if (strcmp(raw, _cg_consts[ci].n) == 0 &&
                    strcmp(real_mod, _cg_consts[ci].mod) == 0) {
                    emit(cg, _cg_consts[ci].val);
                    emitted_const = true;
                    break;
                }
            }
        }
        if (emitted_const) break;
        if (is_mutable_param(cg, raw)) {
            emitf(cg, "(*%s)", name);
        } else if (is_ref_var(cg, raw)) {
            emitf(cg, "(*%s)", name);
        } else {
            emit(cg, name);
        }
        break;
    }

    case NODE_INT_VALUE:
        if (node->data.int_value.overflow) {
            /* Overflowed literal — check if we're in a bigint context */
            const char *bi_ctx = resolve_bigint_type(cg, node);
            if (!bi_ctx) {
                /* Check parent context: look for bigint var assignment */
                /* For non-bigint contexts, emit as ULL (valid for u64/uint) */
                emitf(cg, "%sULL", node->data.int_value.literal);
            } else {
                emitf(cg, "%s_from_decimal(\"%s\")", bigint_prefix(bi_ctx),
                    node->data.int_value.literal);
            }
        } else {
            emitf(cg, "%lld", (long long)node->data.int_value.value);
        }
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
        /* Check for null bytes — if present, use ez_string_lit_len with explicit length
         * since strlen() would truncate at the null */
        bool has_null = false;
        int str_len = 0;
        for (const char *p = s; *p; p++) {
            if (p[0] == '\\' && p[1] == 'x' && p[2] == '0' && p[3] == '0') {
                has_null = true;
                str_len++; /* \x00 = 1 byte */
                p += 3;
            } else if (p[0] == '\\' && p[1] == '0') {
                has_null = true;
                str_len++; /* \0 = 1 byte */
                p += 1;
            } else if (p[0] == '\\' && p[1]) {
                str_len++; /* other escape = 1 byte */
                p += 1;
            } else {
                str_len++;
            }
        }
        /* Use macro form for file-scope compatibility */
        if (has_null && cg->indent > 0) {
            emitf(cg, "ez_string_lit_len(\"");
        } else {
            emit(cg, (cg->indent == 0) ? "EZ_STRING_LIT(\"" : "ez_string_lit(\"");
        }
        if (node->data.string_value.is_raw) {
            /* Raw string — escape special characters for C output */
            while (*s) {
                if (*s == '\\') {
                    emit(cg, "\\\\");
                } else if (*s == '"') {
                    emit(cg, "\\\"");
                } else if (*s == '\n') {
                    emit(cg, "\\n");
                } else if (*s == '\r') {
                    emit(cg, "\\r");
                } else if (*s == '\t') {
                    emit(cg, "\\t");
                } else {
                    buf_append_char(&cg->output, *s);
                }
                s++;
            }
        } else {
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
        }
        if (has_null && cg->indent > 0) {
            emitf(cg, "\", %d)", str_len);
        } else {
            emit(cg, "\")");
        }
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

                /* Check for bigint types — format as %s (use to_string) */
                const char *bi_interp = resolve_bigint_type(cg, part);
                if (bi_interp) {
                    emit(cg, "%s");
                } else switch (tk) {
                case TK_STRING: emit(cg, "%s"); break;
                case TK_FLOAT:  emit(cg, "%s"); break; /* uses ez_builtin_format_float */
                case TK_BOOL:   emit(cg, "%s"); break;
                case TK_CHAR:   emit(cg, "%s"); break;
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

            /* Check for bigint types — use to_string */
            const char *bi_arg = resolve_bigint_type(cg, part);
            if (bi_arg) {
                emitf(cg, "%s_to_string(ez_default_arena, ", bigint_prefix(bi_arg));
                emit_expression(cg, part);
                emit(cg, ").data");
            } else switch (tk) {
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
                emit(cg, "ez_builtin_format_float(ez_default_arena, ");
                emit_expression(cg, part);
                emit(cg, ").data");
                break;
            case TK_CHAR:
                emit(cg, "ez_builtin_char_to_utf8(ez_default_arena, ");
                emit_expression(cg, part);
                emit(cg, ").data");
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
                emitf(cg, "ez_builtin_array_to_string(ez_default_arena, &");
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
                emitf(cg, "ez_builtin_map_to_string(ez_default_arena, &");
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
            /* Empty array — check type table for element type, falling
             * back to the var-decl context type if the node has none. */
            EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
            if ((!arr_t || arr_t->kind == TK_UNKNOWN) && cg->current_var_type && cg->current_var_type[0]) {
                arr_t = type_from_name(cg->current_var_type);
            }
            const char *elem_sz = "int64_t";
            if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
                const char *etype = arr_t->element_type;
                EzType *et = type_from_name(etype);
                if (et->kind == TK_FLOAT) elem_sz = "double";
                else if (et->kind == TK_BOOL) elem_sz = "bool";
                else if (et->kind == TK_STRING) elem_sz = "EzString";
                else if (et->kind == TK_ARRAY) elem_sz = "EzArray";
                else if (et->kind == TK_MAP) elem_sz = "EzMap";
                else if (et->kind == TK_STRUCT) elem_sz = ez_type_to_c_cg(cg, etype);
                else if (et->kind == TK_POINTER) elem_sz = ez_type_to_c_cg(cg, etype);
                else if (et->kind == TK_CHAR) elem_sz = "int32_t";
                else if (et->kind == TK_BYTE) elem_sz = "uint8_t";
                else if (strcmp(etype, "i128") == 0) elem_sz = "ez_i128";
                else if (strcmp(etype, "u128") == 0) elem_sz = "ez_u128";
                else if (strcmp(etype, "i256") == 0) elem_sz = "ez_i256";
                else if (strcmp(etype, "u256") == 0) elem_sz = "ez_u256";
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

        /* Array of maps: elements are map literals */
        if (node->data.array_value.elements[0]->kind == NODE_MAP_VALUE) {
            emitf(cg, "ez_array_from(ez_default_arena, (EzMap[]){");
            for (int i = 0; i < count; i++) {
                if (i > 0) emit(cg, ", ");
                emit_expression(cg, node->data.array_value.elements[i]);
            }
            emitf(cg, "}, sizeof(EzMap), %d)", count);
            break;
        }

        /* Determine element type — try bigint detection first, then type table */
        const char *bi_elem = resolve_bigint_type(cg, node->data.array_value.elements[0]);
        EzType *elem_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.array_value.elements[0])
            : NULL;
        if (!bi_elem && elem_t && elem_t->name && is_bigint_type(elem_t->name))
            bi_elem = elem_t->name;
        /* Also check var decl context for bigint element type */
        if (!bi_elem && cg->current_var_type && is_bigint_type(cg->current_var_type))
            bi_elem = cg->current_var_type;
        TypeKind tk = elem_t ? elem_t->kind : TK_INT;

        const char *c_type;
        /* Check for bigint types first */
        if (bi_elem) {
            c_type = bigint_prefix(bi_elem);
        } else if (elem_t && elem_t->name && strcmp(elem_t->name, "func") == 0) {
            /* Function reference elements: store as generic fn ptrs, cast at
             * call sites (mirrors ez_type_to_c_cg's handling of "func"). */
            c_type = "void *";
        } else if (cg->current_var_type &&
                   strcmp(cg->current_var_type, "[func]") == 0) {
            /* Declared as [func] but element inference missed it (e.g. empty
             * literal or heterogeneous func refs). */
            c_type = "void *";
        } else switch (tk) {
        case TK_FLOAT:  c_type = "double"; break;
        case TK_BOOL:   c_type = "bool"; break;
        case TK_STRING: c_type = "EzString"; break;
        case TK_STRUCT: c_type = ez_type_to_c_cg(cg, elem_t->name); break;
        case TK_POINTER: {
            const char *pointee = elem_t->element_type ? elem_t->element_type : "void";
            const char *c_pointee = ez_type_to_c_cg(cg, pointee);
            static char ptr_buf[256];
            snprintf(ptr_buf, sizeof(ptr_buf), "%s *", c_pointee);
            c_type = ptr_buf;
            break;
        }
        case TK_MAP:    c_type = "EzMap"; break;
        case TK_ARRAY:  c_type = "EzArray"; break;
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

        /* Determine key/value C types. Prefer the enclosing var/field declared
         * type when available — byte/char literals are typechecked as int, so
         * first-pair inference would miss the declared key type. */
        const char *c_key_type = "EzString";
        const char *c_val_type = "int64_t";
        EzType *decl_mt = (cg->current_var_type &&
                           strncmp(cg->current_var_type, "map[", 4) == 0)
            ? type_from_name(cg->current_var_type) : NULL;
        if (decl_mt && decl_mt->key_type)
            c_key_type = ez_map_elem_c_type(cg, decl_mt->key_type);
        if (decl_mt && decl_mt->value_type)
            c_val_type = ez_map_elem_c_type(cg, decl_mt->value_type);
        if (count > 0) {
            EzType *kt = cg->type_table ? typetable_get(cg->type_table, node->data.map_value.keys[0]) : NULL;
            EzType *vt = cg->type_table ? typetable_get(cg->type_table, node->data.map_value.values[0]) : NULL;
            if (!decl_mt && kt) c_key_type = ez_map_elem_c_type(cg, type_name(kt));
            if (!decl_mt && vt && vt->kind == TK_POINTER) {
                static char map_ptr_buf[256];
                const char *pointee = vt->element_type ? vt->element_type : "void";
                snprintf(map_ptr_buf, sizeof(map_ptr_buf), "%s *", ez_type_to_c_cg(cg, pointee));
                c_val_type = map_ptr_buf;
            } else if (!decl_mt && vt) {
                c_val_type = ez_map_elem_c_type(cg, type_name(vt));
            }
        }

        /* Use GCC statement expression: ({ EzMap m = ...; ez_map_set(...); m; })
         * Capture counter before emitting values — nested map literals will
         * re-enter this case and increment the counter, so each level gets
         * a unique temp name. */
        static int map_lit_counter = 0;
        int my_counter = map_lit_counter++;
        emitf(cg, "({ EzMap _ml%d = ez_map_new(ez_default_arena, sizeof(%s), sizeof(%s), %d); ",
            my_counter, c_key_type, c_val_type, count > 4 ? count * 2 : 8);

        /* For nested map values, propagate the inner type so inner literals
         * resolve their key/value C types correctly. */
        const char *inner_var_type = NULL;
        if (decl_mt && decl_mt->value_type &&
            strncmp(decl_mt->value_type, "map[", 4) == 0) {
            inner_var_type = decl_mt->value_type;
        }

        for (int i = 0; i < count; i++) {
            emitf(cg, "{ %s _mk = ", c_key_type);
            emit_expression(cg, node->data.map_value.keys[i]);
            emitf(cg, "; %s _mv = ", c_val_type);
            if (inner_var_type) {
                const char *saved = cg->current_var_type;
                cg->current_var_type = inner_var_type;
                emit_expression(cg, node->data.map_value.values[i]);
                cg->current_var_type = saved;
            } else {
                emit_expression(cg, node->data.map_value.values[i]);
            }
            emitf(cg, "; ez_map_set(ez_default_arena, &_ml%d, &_mk, &_mv); } ", my_counter);
        }
        emitf(cg, "_ml%d; })", my_counter);
        break;
    }

    case NODE_STRUCT_VALUE: {
        /* Struct literal: (EzStruct_Name){.field = value, ...} */
        /* Resolve unprefixed struct names from 'import and use' */
        const char *sname = node->data.struct_value.name;
        if (sname[0] >= 'A' && sname[0] <= 'Z') {
            bool found = false;
            for (int si = 0; si < cg->struct_decl_count; si++) {
                if (strcmp(cg->struct_decls[si]->data.struct_decl.name, sname) == 0) {
                    found = true;
                    break;
                }
            }
            if (!found) {
                /* Try prefixed names */
                for (int si = 0; si < cg->struct_decl_count; si++) {
                    const char *dn = cg->struct_decls[si]->data.struct_decl.name;
                    const char *us = strrchr(dn, '_');
                    if (us && strcmp(us + 1, sname) == 0) {
                        sname = dn;
                        break;
                    }
                }
            }
        }
        /* #1520: use mangled name for generic struct instantiations */
        if (node->data.struct_value.wildcard_binding) {
            const char *binding = node->data.struct_value.wildcard_binding;
            char mangled[256];
            size_t mpos = snprintf(mangled, sizeof(mangled), "%s__", sname);
            for (const char *c = binding; *c && mpos < sizeof(mangled) - 1; c++) {
                mangled[mpos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
            }
            mangled[mpos] = '\0';
            emitf(cg, "(EzStruct_%s){", mangled);
        } else {
            emitf(cg, "(EzStruct_%s){", sname);
        }
        for (int i = 0; i < node->data.struct_value.count; i++) {
            if (i > 0) emit(cg, ", ");
            emitf(cg, ".%s = ", safe_name(node->data.struct_value.field_names[i]));
            emit_expression(cg, node->data.struct_value.field_values[i]);
        }
        emit(cg, "}");
        break;
    }

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
        /* Overflow-checked negation for signed integer types — STANDARD §3.1.1
         * promises arithmetic panics rather than silent wrap on overflow. */
        if (strcmp(node->data.prefix.op, "-") == 0) {
            EzType *ot = cg->type_table ? typetable_get(cg->type_table, node->data.prefix.right) : NULL;
            if (ot && ot->kind == TK_INT) {
                const char *sn = ot->name;
                const char *smin = NULL, *smax = NULL;
                if (sn) {
                    if (strcmp(sn, "i8") == 0) { smin = "-128"; smax = "127"; }
                    else if (strcmp(sn, "i16") == 0) { smin = "-32768"; smax = "32767"; }
                    else if (strcmp(sn, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
                }
                if (smax) {
                    emit(cg, "ez_sized_neg_check(");
                    emit_expression(cg, node->data.prefix.right);
                    emitf(cg, ", %s, %s, \"%s\", __FILE__, %d)", smin, smax, sn, node->token.line);
                } else {
                    emit(cg, "ez_neg_check(");
                    emit_expression(cg, node->data.prefix.right);
                    emitf(cg, ", __FILE__, %d)", node->token.line);
                }
                break;
            }
        }
        emit(cg, "(");
        emit(cg, node->data.prefix.op);
        if (strcmp(node->data.prefix.op, "-") == 0 &&
            node->data.prefix.right->kind == NODE_INT_VALUE) {
            emit(cg, " ");
        }
        /* Wrap infix operands in parens so !(a && b) emits as (!(a && b)) not (!a && b) */
        bool wrap = (node->data.prefix.right->kind == NODE_INFIX_EXPR);
        if (wrap) emit(cg, "(");
        emit_expression(cg, node->data.prefix.right);
        if (wrap) emit(cg, ")");
        emit(cg, ")");
        break;

    case NODE_INFIX_EXPR: {
        const char *op = node->data.infix.op;

        /* Check if either operand is a string — need special handling */
        EzType *lt = cg->type_table ? typetable_get(cg->type_table, node->data.infix.left) : NULL;
        EzType *rt = cg->type_table ? typetable_get(cg->type_table, node->data.infix.right) : NULL;
        /* Inside a generic instantiation (#1443), operands that were
         * typed TK_UNKNOWN in the main pass (because they traced back
         * to a '?' parameter) should be treated as the active wildcard
         * binding so string/struct comparisons pick up the right path. */
        if (cg && cg->wildcard_binding) {
            static EzType wc_t_static;
            EzType *wc_t = type_from_name(cg->wildcard_binding);
            if (wc_t) { wc_t_static = *wc_t; wc_t = &wc_t_static; }
            if (!lt || lt->kind == TK_UNKNOWN) lt = wc_t;
            if (!rt || rt->kind == TK_UNKNOWN) rt = wc_t;
        }
        bool left_is_str = (lt && lt->kind == TK_STRING) || node->data.infix.left->kind == NODE_STRING_VALUE;
        bool right_is_str = (rt && rt->kind == TK_STRING) || node->data.infix.right->kind == NODE_STRING_VALUE;

        if ((left_is_str || right_is_str) && strcmp(op, "+") == 0) {
            /* String concatenation is rejected by the typechecker (E3048).
             * This path is unreachable but kept as a safety net. */
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
                emitf(cg, "({ %s _ik = ", ez_map_elem_c_type(cg, arr_t->key_type));
                emit_expression(cg, node->data.infix.left);
                emit(cg, "; ez_maps_has_key(&");
                emit_expression(cg, node->data.infix.right);
                emit(cg, ", &_ik); })");
                break;
            }
            if (negated) emit(cg, "!");
            {
                const char *contains_fn = "ez_arrays_contains_int";
                if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
                    if (strcmp(arr_t->element_type, "string") == 0)
                        contains_fn = "ez_arrays_contains_str";
                    else if (strcmp(arr_t->element_type, "float") == 0 ||
                             strcmp(arr_t->element_type, "f32") == 0 ||
                             strcmp(arr_t->element_type, "f64") == 0)
                        contains_fn = "ez_arrays_contains_float";
                }
                emitf(cg, "%s(&", contains_fn);
                emit_expression(cg, node->data.infix.right);
                emit(cg, ", ");
                emit_expression(cg, node->data.infix.left);
                emit(cg, ")");
            }
            break;
        }

        /* Helper: emit an operand in bigint context, wrapping int literals */
        #define EMIT_BIGINT_OPERAND(cg, operand, pfx, bi_type) do { \
            if ((operand)->kind == NODE_INT_VALUE) { \
                if ((operand)->data.int_value.overflow) { \
                    emitf((cg), "%s_from_decimal(\"%s\")", (pfx), (operand)->data.int_value.literal); \
                } else { \
                    const char *_sfx = (strcmp((bi_type), "u128") == 0 || strcmp((bi_type), "u256") == 0) ? "u64" : "i64"; \
                    emitf((cg), "%s_from_%s(%lldLL)", (pfx), _sfx, (long long)(operand)->data.int_value.value); \
                } \
            } else if ((operand)->kind == NODE_PREFIX_EXPR && \
                       strcmp((operand)->data.prefix.op, "-") == 0 && \
                       (operand)->data.prefix.right->kind == NODE_INT_VALUE) { \
                if ((operand)->data.prefix.right->data.int_value.overflow) { \
                    emitf((cg), "%s_from_decimal(\"-%s\")", (pfx), \
                        (operand)->data.prefix.right->data.int_value.literal); \
                } else { \
                    emitf((cg), "%s_from_i64(%lldLL)", (pfx), \
                        -(long long)(operand)->data.prefix.right->data.int_value.value); \
                } \
            } else { \
                emit_expression((cg), (operand)); \
            } \
        } while(0)

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
                    bool is_checked = (strcmp(fn_op, "add") == 0 || strcmp(fn_op, "sub") == 0 || strcmp(fn_op, "mul") == 0);
                    if (is_checked) {
                        emitf(cg, "%s_%s_checked(", pfx, fn_op);
                    } else {
                        emitf(cg, "%s_%s(", pfx, fn_op);
                    }
                    EMIT_BIGINT_OPERAND(cg, node->data.infix.left, pfx, bi_type);
                    emit(cg, ", ");
                    EMIT_BIGINT_OPERAND(cg, node->data.infix.right, pfx, bi_type);
                    if (is_checked) {
                        emitf(cg, ", __FILE__, %d)", node->token.line);
                    } else {
                        emit(cg, ")");
                    }
                    break;
                }
            }
        }

        /* Runtime division/modulo by zero check */
        if (strcmp(op, "/") == 0 || strcmp(op, "%") == 0) {
            bool is_float_div = (lt && lt->kind == TK_FLOAT) || (rt && rt->kind == TK_FLOAT);
            if (is_float_div) {
                /* Float division: check for zero (EZ panics, no IEEE 754 inf) */
                emit(cg, "({ double _dv = (double)");
                emit_expression(cg, node->data.infix.right);
                emitf(cg, "; if (_dv == 0.0) { fflush(stdout); ez_panic(__FILE__, %d, \"division by zero\"); } (double)",
                    node->token.line);
                emit_expression(cg, node->data.infix.left);
                emitf(cg, " %s _dv; })", op);
                break;
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
                /* Check for sized types that need bounds-checked arithmetic */
                const char *sized_name = (lt && lt->name) ? lt->name : ((rt && rt->name) ? rt->name : NULL);
                const char *sized_min = NULL, *sized_max = NULL;
                bool sized_unsigned = false;
                if (sized_name) {
                    if (strcmp(sized_name, "i8") == 0) { sized_min = "-128"; sized_max = "127"; }
                    else if (strcmp(sized_name, "i16") == 0) { sized_min = "-32768"; sized_max = "32767"; }
                    else if (strcmp(sized_name, "i32") == 0) { sized_min = "-2147483648LL"; sized_max = "2147483647LL"; }
                    else if (strcmp(sized_name, "u8") == 0 || strcmp(sized_name, "byte") == 0) { sized_unsigned = true; sized_max = "255"; }
                    else if (strcmp(sized_name, "u16") == 0) { sized_unsigned = true; sized_max = "65535"; }
                    else if (strcmp(sized_name, "u32") == 0) { sized_unsigned = true; sized_max = "4294967295ULL"; }
                }

                if (sized_max) {
                    /* Sized type — use bounds-checked arithmetic */
                    const char *op_fn = NULL;
                    if (sized_unsigned) {
                        if (strcmp(op, "+") == 0) op_fn = "ez_usized_add_check";
                        else if (strcmp(op, "-") == 0) op_fn = "ez_usized_sub_check";
                        else if (strcmp(op, "*") == 0) op_fn = "ez_usized_mul_check";
                    } else {
                        if (strcmp(op, "+") == 0) op_fn = "ez_sized_add_check";
                        else if (strcmp(op, "-") == 0) op_fn = "ez_sized_sub_check";
                        else if (strcmp(op, "*") == 0) op_fn = "ez_sized_mul_check";
                    }
                    if (op_fn) {
                        emitf(cg, "%s(", op_fn);
                        emit_expression(cg, node->data.infix.left);
                        emit(cg, ", ");
                        emit_expression(cg, node->data.infix.right);
                        if (sized_unsigned) {
                            emitf(cg, ", %s, \"%s\", __FILE__, %d)", sized_max, sized_name, node->token.line);
                        } else {
                            emitf(cg, ", %s, %s, \"%s\", __FILE__, %d)", sized_min, sized_max, sized_name, node->token.line);
                        }
                        break;
                    }
                }

                /* 64-bit overflow checks for int/uint/i64/u64 */
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
            /* Overflow-checked increment — sized types need bounds check */
            EzType *pt = cg->type_table ? typetable_get(cg->type_table, node->data.postfix.left) : NULL;
            const char *sn = (pt && pt->name) ? pt->name : NULL;
            const char *smin = NULL, *smax = NULL;
            bool su = false;
            if (sn) {
                if (strcmp(sn, "i8") == 0) { smin = "-128"; smax = "127"; }
                else if (strcmp(sn, "i16") == 0) { smin = "-32768"; smax = "32767"; }
                else if (strcmp(sn, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
                else if (strcmp(sn, "u8") == 0 || strcmp(sn, "byte") == 0) { su = true; smax = "255"; }
                else if (strcmp(sn, "u16") == 0) { su = true; smax = "65535"; }
                else if (strcmp(sn, "u32") == 0) { su = true; smax = "4294967295ULL"; }
            }
            emit(cg, "(");
            emit_expression(cg, node->data.postfix.left);
            if (smax) {
                if (su) {
                    emit(cg, " = ez_usized_add_check(");
                    emit_expression(cg, node->data.postfix.left);
                    emitf(cg, ", 1, %s, \"%s\", __FILE__, %d))", smax, sn, node->token.line);
                } else {
                    emit(cg, " = ez_sized_add_check(");
                    emit_expression(cg, node->data.postfix.left);
                    emitf(cg, ", 1, %s, %s, \"%s\", __FILE__, %d))", smin, smax, sn, node->token.line);
                }
            } else {
                emitf(cg, " = ez_add_check(");
                emit_expression(cg, node->data.postfix.left);
                emitf(cg, ", 1, __FILE__, %d))", node->token.line);
            }
        } else if (strcmp(node->data.postfix.op, "--") == 0) {
            /* Overflow-checked decrement — sized types need bounds check */
            EzType *pt = cg->type_table ? typetable_get(cg->type_table, node->data.postfix.left) : NULL;
            const char *sn = (pt && pt->name) ? pt->name : NULL;
            const char *smin = NULL, *smax = NULL;
            bool su = false;
            if (sn) {
                if (strcmp(sn, "i8") == 0) { smin = "-128"; smax = "127"; }
                else if (strcmp(sn, "i16") == 0) { smin = "-32768"; smax = "32767"; }
                else if (strcmp(sn, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
                else if (strcmp(sn, "u8") == 0 || strcmp(sn, "byte") == 0) { su = true; smax = "255"; }
                else if (strcmp(sn, "u16") == 0) { su = true; smax = "65535"; }
                else if (strcmp(sn, "u32") == 0) { su = true; smax = "4294967295ULL"; }
            }
            emit(cg, "(");
            emit_expression(cg, node->data.postfix.left);
            if (smax) {
                if (su) {
                    emit(cg, " = ez_usized_sub_check(");
                    emit_expression(cg, node->data.postfix.left);
                    emitf(cg, ", 1, %s, \"%s\", __FILE__, %d))", smax, sn, node->token.line);
                } else {
                    emit(cg, " = ez_sized_sub_check(");
                    emit_expression(cg, node->data.postfix.left);
                    emitf(cg, ", 1, %s, %s, \"%s\", __FILE__, %d))", smin, smax, sn, node->token.line);
                }
            } else {
                emitf(cg, " = ez_sub_check(");
                emit_expression(cg, node->data.postfix.left);
                emitf(cg, ", 1, __FILE__, %d))", node->token.line);
            }
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

            /* Check if this is an enum access: EnumName.VALUE or prefix_EnumName.VALUE */
            if (mod[0] >= 'A' && mod[0] <= 'Z') {
                /* Resolve unprefixed enum names from 'import and use' */
                const char *resolved_enum = mod;
                if (!codegen_is_enum(cg, mod)) {
                    for (int ei = 0; ei < cg->enum_count; ei++) {
                        const char *en = cg->enum_names[ei];
                        const char *us = strrchr(en, '_');
                        if (us && strcmp(us + 1, mod) == 0) {
                            resolved_enum = en;
                            break;
                        }
                    }
                }
                emitf(cg, "EzEnum_%s_%s", resolved_enum, mem);
                break;
            }
            /* Rewritten enum name from import: defs_Color.RED → EzEnum_defs_Color_RED */
            if (codegen_is_enum(cg, mod) && mem[0] >= 'A' && mem[0] <= 'Z') {
                emitf(cg, "EzEnum_%s_%s", mod, mem);
                break;
            }

            /* C interop constant access: c.EOF, c.NULL, c.EXIT_SUCCESS */
            if (strcmp(mod, "c") == 0 && cg->has_c_imports) {
                emitf(cg, "%s", mem);
                break;
            }

            /* User-module qualified constant/variable access: mod.NAME → mod_NAME
             * Only apply for known imported module names, not local variables */
            if (mod[0] >= 'a' && mod[0] <= 'z') {
                /* Check if mod is an imported module by looking for mod_ prefixed declarations */
                bool is_module = false;
                size_t cn_len = strlen(mod) + 1 + strlen(mem) + 1;
                char *check_name = malloc(cn_len);
                snprintf(check_name, cn_len, "%s_%s", mod, mem);
                /* Check functions, variables via find_func */
                if (find_func(cg, check_name)) is_module = true;
                /* Check if any function starts with mod_ prefix */
                if (!is_module) {
                    size_t pfx_len = strlen(mod) + 2;
                    char *prefix = malloc(pfx_len);
                    snprintf(prefix, pfx_len, "%s_", mod);
                    size_t plen = strlen(prefix);
                    for (int fi = 0; fi < cg->func_count; fi++) {
                        if (strncmp(cg->all_funcs[fi]->data.func_decl.name, prefix, plen) == 0) {
                            is_module = true;
                            break;
                        }
                    }
                }
                /* Check using_modules list */
                if (!is_module) {
                    for (int ui = 0; ui < cg->using_module_count; ui++) {
                        if (strcmp(cg->using_modules[ui], mod) == 0) {
                            is_module = true;
                            break;
                        }
                    }
                }
                /* Check alias mappings */
                if (!is_module) {
                    for (int ai = 0; ai < cg->alias_count; ai++) {
                        if (strcmp(cg->alias_names[ai], mod) == 0) {
                            is_module = true;
                            mod = cg->alias_modules[ai];
                            break;
                        }
                    }
                }
                /* Check imported module names list */
                if (!is_module) {
                    for (int ii = 0; ii < cg->imported_module_count; ii++) {
                        if (strcmp(cg->imported_modules[ii], mod) == 0) {
                            is_module = true;
                            break;
                        }
                    }
                }
                if (is_module) {
                    emitf(cg, "%s_%s", mod, mem);
                    break;
                }
            }
        }
        /* Module-qualified enum access: lib.Color.RED → EzEnum_lib_Color_RED */
        if (node->data.member.object->kind == NODE_MEMBER_EXPR) {
            AstNode *inner = node->data.member.object;
            if (inner->data.member.object->kind == NODE_LABEL) {
                const char *mod = inner->data.member.object->data.label.value;
                const char *type_name = inner->data.member.member;
                const char *value = node->data.member.member;
                /* Check if it's a module.EnumName.VALUE pattern */
                if (mod[0] >= 'a' && mod[0] <= 'z' &&
                    type_name[0] >= 'A' && type_name[0] <= 'Z' &&
                    value[0] >= 'A' && value[0] <= 'Z') {
                    emitf(cg, "EzEnum_%s_%s_%s", mod, type_name, value);
                    break;
                }
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

            /* Ref vars are already dereferenced by label emission — use . not -> */
            bool obj_is_ref = (node->data.member.object->kind == NODE_LABEL &&
                is_ref_var(cg, node->data.member.object->data.label.value));
            if (!obj_is_ref && obj_t && obj_t->kind == TK_POINTER) {
                /* Nil-guarded pointer field access */
                emit(cg, "({ __auto_type _dp = ");
                emit_expression(cg, node->data.member.object);
                emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
                    "\"nil pointer dereference\"); } _dp->%s; })",
                    node->token.line, safe_name(node->data.member.member));
            } else if (!obj_is_ref && obj_t && obj_t->kind == TK_ERROR) {
                emit_expression(cg, node->data.member.object);
                emitf(cg, "->%s", safe_name(node->data.member.member));
            } else {
                emit_expression(cg, node->data.member.object);
                emitf(cg, ".%s", safe_name(node->data.member.member));
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
            const char *elem_tn = cg_effective_type_str(cg, left_t->element_type);
            if (elem_tn && strcmp(elem_tn, "func") == 0) {
                c_elem = "void *";
            } else if (elem_tn) {
                EzType *et = type_from_name(elem_tn);
                if (et->kind == TK_FLOAT) c_elem = "double";
                else if (et->kind == TK_BOOL) c_elem = "bool";
                else if (et->kind == TK_STRING) c_elem = "EzString";
                else if (et->kind == TK_CHAR) c_elem = "int32_t";
                else if (et->kind == TK_BYTE) c_elem = "uint8_t";
                else if (et->kind == TK_ARRAY) c_elem = "EzArray";
                else if (et->kind == TK_MAP) c_elem = "EzMap";
                else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, elem_tn);
                else if (et->kind == TK_POINTER) {
                    static char idx_ptr_buf[256];
                    const char *pointee = et->element_type ? et->element_type : "void";
                    snprintf(idx_ptr_buf, sizeof(idx_ptr_buf), "%s *", ez_type_to_c_cg(cg, pointee));
                    c_elem = idx_ptr_buf;
                }
            }
            /* Check for bigint element types */
            if (left_t->element_type && is_bigint_type(left_t->element_type)) {
                c_elem = bigint_prefix(left_t->element_type);
            }
            /* If left is an rvalue (function call), store in temp first —
             * EZ_ARRAY_GET takes &arr which requires an lvalue */
            if (node->data.index_expr.left->kind == NODE_CALL_EXPR) {
                emitf(cg, "({ EzArray _ea = ");
                emit_expression(cg, node->data.index_expr.left);
                emitf(cg, "; EZ_ARRAY_GET(_ea, %s, ", c_elem);
                emit_expression(cg, node->data.index_expr.index);
                emit(cg, "); })");
            } else {
                emitf(cg, "EZ_ARRAY_GET(");
                emit_expression(cg, node->data.index_expr.left);
                emitf(cg, ", %s, ", c_elem);
                emit_expression(cg, node->data.index_expr.index);
                emit(cg, ")");
            }
        } else if (left_t && left_t->kind == TK_MAP) {
            /* Map key access — use temp to handle rvalue keys like literals */
            const char *c_key = "EzString";
            const char *c_val = "int64_t";
            if (left_t->key_type) c_key = ez_map_elem_c_type(cg, left_t->key_type);
            if (left_t->value_type) c_val = ez_map_elem_c_type(cg, left_t->value_type);
            /* When the left side is an rvalue (e.g. chained map access
             * like m["a"]["x"]), store it in a temp to make it addressable. */
            if (node->data.index_expr.left->kind == NODE_INDEX_EXPR ||
                node->data.index_expr.left->kind == NODE_CALL_EXPR) {
                emitf(cg, "({ EzMap _mt = ");
                emit_expression(cg, node->data.index_expr.left);
                emitf(cg, "; %s _mk = ", c_key);
                emit_expression(cg, node->data.index_expr.index);
                emitf(cg, "; void *_mv = ez_map_get(&_mt, &_mk); if (!_mv) { fflush(stdout); ez_panic(__FILE__, %d, \"key not found in map\"); } ",
                    node->token.line);
                emitf(cg, "*(%s *)_mv; })", c_val);
            } else {
                emitf(cg, "({ %s _mk = ", c_key);
                emit_expression(cg, node->data.index_expr.index);
                emitf(cg, "; void *_mv = ez_map_get(&");
                emit_expression(cg, node->data.index_expr.left);
                emitf(cg, ", &_mk); if (!_mv) { fflush(stdout); ez_panic(__FILE__, %d, \"key not found in map\"); } ",
                    node->token.line);
                emitf(cg, "*(%s *)_mv; })", c_val);
            }
        } else if (left_t && left_t->kind == TK_STRING) {
            /* String indexing with bounds check: s.data[i] */
            emitf(cg, "({ EzString _es = ");
            emit_expression(cg, node->data.index_expr.left);
            emitf(cg, "; int32_t _ei = (int32_t)(");
            emit_expression(cg, node->data.index_expr.index);
            emitf(cg, "); if (_ei < 0 || _ei >= _es.len) { fflush(stdout); ez_panic(__FILE__, %d, ",
                node->token.line);
            emit(cg, "\"string index %d out of bounds (length %d)\", _ei, _es.len); } ");
            emit(cg, "(int32_t)(unsigned char)_es.data[_ei]; })");
        } else {
            /* Fallback */
            emit_expression(cg, node->data.index_expr.left);
            emit(cg, "[");
            emit_expression(cg, node->data.index_expr.index);
            emit(cg, "]");
        }
        break;
    }

    case NODE_CAST_EXPR: {
        /* cast(value, type) — dispatch to conversion functions for non-trivial casts */
        const char *target = node->data.cast.target_type;
        AstNode *val = node->data.cast.value;
        EzType *val_t = cg->type_table ? typetable_get(cg->type_table, val) : NULL;
        TypeKind val_kind = val_t ? val_t->kind : TK_UNKNOWN;

        /* Infer kind from AST if type table has no info */
        if (val_kind == TK_UNKNOWN) {
            if (val->kind == NODE_STRING_VALUE || val->kind == NODE_INTERPOLATED_STRING)
                val_kind = TK_STRING;
            else if (val->kind == NODE_BOOL_VALUE) val_kind = TK_BOOL;
            else if (val->kind == NODE_FLOAT_VALUE) val_kind = TK_FLOAT;
            else if (val->kind == NODE_INT_VALUE) val_kind = TK_INT;
        }

        if (strcmp(target, "string") == 0) {
            /* any → string: use to_string functions */
            emit_to_string(cg, val);
        } else if ((strcmp(target, "int") == 0 || strcmp(target, "i64") == 0) && val_kind == TK_STRING) {
            /* string → int */
            emit(cg, "ez_builtin_string_to_int(");
            emit_expression(cg, val);
            emit(cg, ")");
        } else if ((strcmp(target, "float") == 0 || strcmp(target, "f64") == 0) && val_kind == TK_STRING) {
            /* string → float */
            emit(cg, "ez_builtin_string_to_float(");
            emit_expression(cg, val);
            emit(cg, ")");
        } else if ((strcmp(target, "int") == 0 || strcmp(target, "i64") == 0) && val_kind == TK_FLOAT) {
            /* float → int: overflow-safe */
            emit(cg, "ez_float_to_int((double)(");
            emit_expression(cg, val);
            emit(cg, "), __FILE__, __LINE__)");
        } else {
            /* Numeric casts: range-checked for narrowing, raw for widening */
            const char *smin = NULL, *smax = NULL;
            bool is_unsigned = false;
            if (strcmp(target, "i8") == 0) { smin = "-128"; smax = "127"; }
            else if (strcmp(target, "i16") == 0) { smin = "-32768"; smax = "32767"; }
            else if (strcmp(target, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
            else if (strcmp(target, "u8") == 0 || strcmp(target, "byte") == 0) { is_unsigned = true; smax = "255"; }
            else if (strcmp(target, "u16") == 0) { is_unsigned = true; smax = "65535"; }
            else if (strcmp(target, "u32") == 0) { is_unsigned = true; smax = "4294967295ULL"; }

            if (smax && is_unsigned) {
                emitf(cg, "(%s)ez_ucast_check(", ez_type_to_c_cg(cg, target));
                emit_expression(cg, val);
                emitf(cg, ", %s, \"%s\", __FILE__, %d)", smax, target, node->token.line);
            } else if (smax) {
                emitf(cg, "(%s)ez_cast_check(", ez_type_to_c_cg(cg, target));
                emit_expression(cg, val);
                emitf(cg, ", %s, %s, \"%s\", __FILE__, %d)", smin, smax, target, node->token.line);
            } else {
                emitf(cg, "((%s)(", ez_type_to_c_cg(cg, target));
                emit_expression(cg, val);
                emit(cg, "))");
            }
        }
        break;
    }

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
        emit(cg, "ez_builtin_to_string_float(ez_default_arena, ");
    else if (at && at->kind == TK_BOOL)
        emit(cg, "ez_builtin_to_string_bool(ez_default_arena, ");
    else
        emit(cg, "ez_builtin_to_string_int(ez_default_arena, ");
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

/* --- Composite type printing --- */

/* Global counter for unique print variable names */
static int _ez_print_uid = 0;

/* Look up a struct declaration by name */
static AstNode *find_struct_decl(CodeGen *cg, const char *name) {
    if (!name) return NULL;
    for (int i = 0; i < cg->struct_decl_count; i++) {
        if (strcmp(cg->struct_decls[i]->data.struct_decl.name, name) == 0) {
            return cg->struct_decls[i];
        }
    }
    return NULL;
}

/* Emit C statements that print the value of c_expr (of type t) to stream.
 * stream is "stdout" or "stderr". Handles all types recursively. */
static void emit_value_print(CodeGen *cg, const char *c_expr, EzType *t, const char *stream) {
    if (!t || t->kind == TK_UNKNOWN) {
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%lld\", (long long)(%s));\n", stream, c_expr);
        return;
    }

    switch (t->kind) {
    case TK_INT: case TK_UINT: case TK_BYTE: case TK_ENUM:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%lld\", (long long)(%s));\n", stream, c_expr);
        break;
    case TK_FLOAT:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%g\", (double)(%s));\n", stream, c_expr);
        break;
    case TK_STRING:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%.*s\", (int)(%s).len, (%s).data);\n",
               stream, c_expr, c_expr);
        break;
    case TK_BOOL:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%s\", (%s) ? \"true\" : \"false\");\n",
               stream, c_expr);
        break;
    case TK_CHAR:
        emit_indent(cg);
        emitf(cg, "{ EzString _cs = ez_builtin_char_to_utf8(ez_default_arena, %s); fwrite(_cs.data, 1, (size_t)_cs.len, %s); }\n", c_expr, stream);
        break;
    case TK_NIL:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"nil\");\n", stream);
        break;
    case TK_ARRAY: {
        int uid = _ez_print_uid++;
        const char *elem_tn = t->element_type ? t->element_type : "int";
        EzType *elem_t = type_from_name(elem_tn);
        char c_elem[128];
        snprintf(c_elem, sizeof(c_elem), "%s", ez_type_to_c_cg(cg, elem_tn));

        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"{\");\n", stream);
        emit_indent(cg);
        emitf(cg, "for (int32_t _ez_pi%d = 0; _ez_pi%d < (%s).len; _ez_pi%d++) {\n",
               uid, uid, c_expr, uid);
        cg->indent++;
        emit_indent(cg);
        emitf(cg, "if (_ez_pi%d > 0) fprintf(%s, \", \");\n", uid, stream);

        /* For composite element types, capture in temp var */
        char elem_expr[256];
        if (elem_t->kind == TK_STRUCT || elem_t->kind == TK_ARRAY ||
            elem_t->kind == TK_MAP || elem_t->kind == TK_POINTER) {
            int euid = _ez_print_uid++;
            emit_indent(cg);
            emitf(cg, "%s _ez_pv%d = EZ_ARRAY_GET((%s), %s, _ez_pi%d);\n",
                   c_elem, euid, c_expr, c_elem, uid);
            snprintf(elem_expr, sizeof(elem_expr), "_ez_pv%d", euid);
        } else {
            snprintf(elem_expr, sizeof(elem_expr),
                     "EZ_ARRAY_GET((%s), %s, _ez_pi%d)", c_expr, c_elem, uid);
        }

        emit_value_print(cg, elem_expr, elem_t, stream);

        cg->indent--;
        emit_indent(cg);
        emit(cg, "}\n");
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"}\");\n", stream);
        break;
    }
    case TK_MAP: {
        int uid = _ez_print_uid++;
        const char *key_tn = t->key_type ? t->key_type : "string";
        const char *val_tn = t->value_type ? t->value_type : "int";
        EzType *key_t = type_from_name(key_tn);
        EzType *val_t = type_from_name(val_tn);
        char c_key[128], c_val[128];
        snprintf(c_key, sizeof(c_key), "%s", ez_type_to_c_cg(cg, key_tn));
        snprintf(c_val, sizeof(c_val), "%s", ez_type_to_c_cg(cg, val_tn));

        char mi[32], sl[32];
        snprintf(mi, sizeof(mi), "_ez_mi%d", uid);
        snprintf(sl, sizeof(sl), "_ez_sl%d", uid);

        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"{\");\n", stream);
        emit_indent(cg);
        emitf(cg, "if ((%s).order_len == 0) fprintf(%s, \":\");\n", c_expr, stream);
        emit_indent(cg);
        emitf(cg, "for (int32_t %s = 0; %s < (%s).order_len; %s++) {\n",
               mi, mi, c_expr, mi);
        cg->indent++;
        emit_indent(cg);
        emitf(cg, "int32_t %s = (%s).order[%s];\n", sl, c_expr, mi);
        emit_indent(cg);
        emitf(cg, "if (%s > 0) fprintf(%s, \", \");\n", mi, stream);

        /* Print key */
        char key_expr[256];
        snprintf(key_expr, sizeof(key_expr),
                 "*(%s *)ez_map_key_at(&(%s), %s)", c_key, c_expr, sl);
        emit_value_print(cg, key_expr, key_t, stream);

        emit_indent(cg);
        emitf(cg, "fprintf(%s, \": \");\n", stream);

        /* Print value */
        char val_expr[256];
        snprintf(val_expr, sizeof(val_expr),
                 "*(%s *)ez_map_value_at(&(%s), %s)", c_val, c_expr, sl);
        emit_value_print(cg, val_expr, val_t, stream);

        cg->indent--;
        emit_indent(cg);
        emit(cg, "}\n");
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"}\");\n", stream);
        break;
    }
    case TK_STRUCT: {
        const char *struct_name = t->name;
        AstNode *sdecl = find_struct_decl(cg, struct_name);

        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%s{\");\n", stream, struct_name);

        if (sdecl) {
            for (int i = 0; i < sdecl->data.struct_decl.field_count; i++) {
                StructField *f = &sdecl->data.struct_decl.fields[i];
                if (i > 0) {
                    emit_indent(cg);
                    emitf(cg, "fprintf(%s, \", \");\n", stream);
                }
                emit_indent(cg);
                emitf(cg, "fprintf(%s, \"%s: \");\n", stream, f->name);

                char field_expr[256];
                snprintf(field_expr, sizeof(field_expr), "(%s).%s", c_expr, f->name);
                EzType *ft = type_from_name(f->type_name);
                emit_value_print(cg, field_expr, ft, stream);
            }
        }

        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"}\");\n", stream);
        break;
    }
    case TK_POINTER: {
        const char *pointee_tn = t->element_type ? t->element_type : "int";
        EzType *pointee_t = type_from_name(pointee_tn);

        emit_indent(cg);
        emitf(cg, "if ((%s) == NULL) {\n", c_expr);
        cg->indent++;
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"nil\");\n", stream);
        cg->indent--;
        emit_indent(cg);
        emit(cg, "} else {\n");
        cg->indent++;

        char deref_expr[256];
        snprintf(deref_expr, sizeof(deref_expr), "*(%s)", c_expr);
        emit_value_print(cg, deref_expr, pointee_t, stream);

        cg->indent--;
        emit_indent(cg);
        emit(cg, "}\n");
        break;
    }
    default:
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"%%lld\", (long long)(%s));\n", stream, c_expr);
        break;
    }
}

/* Try to emit a composite type print. Returns true if handled. */
static bool emit_composite_print(CodeGen *cg, AstNode *node,
                                  const char *stream, bool newline) {
    if (node->data.call.arg_count < 1) return false;

    AstNode *arg = node->data.call.args[0];
    EzType *t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
    if (!t) return false;
    if (t->kind != TK_STRUCT && t->kind != TK_ARRAY &&
        t->kind != TK_MAP && t->kind != TK_POINTER) return false;
    /* Enum types are stored as TK_STRUCT but should print as integers */
    if (t->kind == TK_STRUCT && t->name && codegen_is_enum(cg, t->name)) return false;

    /* Emit a block to scope temp variables */
    emit(cg, "{\n");
    cg->indent++;

    /* Capture expression in temp var to evaluate only once */
    int uid = _ez_print_uid++;
    char c_type[128];
    if (t->kind == TK_ARRAY) snprintf(c_type, sizeof(c_type), "EzArray");
    else if (t->kind == TK_MAP) snprintf(c_type, sizeof(c_type), "EzMap");
    else if (t->kind == TK_POINTER) {
        const char *pointee_tn = t->element_type ? t->element_type : "int";
        snprintf(c_type, sizeof(c_type), "%s *", ez_type_to_c_cg(cg, pointee_tn));
    }
    else snprintf(c_type, sizeof(c_type), "%s", ez_type_to_c_cg(cg, type_name(t)));

    emit_indent(cg);
    emitf(cg, "%s _ez_pv%d = ", c_type, uid);
    emit_expression(cg, arg);
    emit(cg, ";\n");

    char var[32];
    snprintf(var, sizeof(var), "_ez_pv%d", uid);

    emit_value_print(cg, var, t, stream);

    if (newline) {
        emit_indent(cg);
        emitf(cg, "fprintf(%s, \"\\n\");\n", stream);
    }

    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
    emit_indent(cg);
    emit(cg, "(void)0"); /* Absorb trailing ;\n from emit_expression_statement */

    return true;
}

/* --- Builtin call handler (no-module functions) --- */

static bool emit_builtin_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "println") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "putchar('\\n')");
        } else {
            if (emit_composite_print(cg, node, "stdout", true)) return true;
            AstNode *arg = node->data.call.args[0];
            /* Error type: print message or "nil" */
            EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            if (arg_t && arg_t->kind == TK_ERROR) {
                emit(cg, "ez_builtin_println_str(");
                emit_expression(cg, arg);
                emit(cg, " ? ");
                emit_expression(cg, arg);
                emit(cg, "->message : ez_string_lit(\"nil\"))");
            } else {
                const char *bi_type = resolve_bigint_type(cg, arg);
                if (bi_type) {
                    emitf(cg, "ez_builtin_println_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
                    emit_expression(cg, arg);
                    emit(cg, "))");
                } else {
                    emitf(cg, "ez_builtin_println%s(", resolve_print_suffix(cg, arg));
                    emit_expression(cg, arg);
                    emit(cg, ")");
                }
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
        /* addr() returns a pointer to the argument */
        emit(cg, "&");
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
        emit(cg, "ez_builtin_exit(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "panic") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_builtin_panic_msg(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "assert") == 0 && node->data.call.arg_count >= 1) {
        emit(cg, "ez_builtin_assert(");
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
        emit(cg, "ez_builtin_input(ez_default_arena)");
        return true;
    }


    if (strcmp(func, "eprintln") == 0) {
        if (node->data.call.arg_count == 0) {
            emit(cg, "fputc('\\n', stderr)");
        } else {
            if (emit_composite_print(cg, node, "stderr", true)) return true;
            AstNode *arg = node->data.call.args[0];
            EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
            if (arg_t && arg_t->kind == TK_ERROR) {
                emit(cg, "ez_builtin_eprintln_str(");
                emit_expression(cg, arg);
                emit(cg, " ? ");
                emit_expression(cg, arg);
                emit(cg, "->message : ez_string_lit(\"nil\"))");
            } else {
                const char *bi_type = resolve_bigint_type(cg, arg);
                if (bi_type) {
                    emitf(cg, "ez_builtin_eprintln_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
                    emit_expression(cg, arg);
                    emit(cg, "))");
                } else {
                    emitf(cg, "ez_builtin_eprintln%s(", resolve_print_suffix(cg, arg));
                    emit_expression(cg, arg);
                    emit(cg, ")");
                }
            }
        }
        return true;
    }

    if (strcmp(func, "eprint") == 0 && node->data.call.arg_count > 0) {
        if (emit_composite_print(cg, node, "stderr", false)) return true;
        AstNode *arg = node->data.call.args[0];
        EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (arg_t && arg_t->kind == TK_ERROR) {
            emit(cg, "ez_builtin_eprint_str(");
            emit_expression(cg, arg);
            emit(cg, " ? ");
            emit_expression(cg, arg);
            emit(cg, "->message : ez_string_lit(\"nil\"))");
        } else {
            emit(cg, "ez_builtin_eprint_str(");
            emit_expression(cg, arg);
            emit(cg, ")");
        }
        return true;
    }

    if (strcmp(func, "sleep_s") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_builtin_sleep_s(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sleep_ms") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_builtin_sleep_ms(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "sleep_ns") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_builtin_sleep_ns(");
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
        if (strcmp(func, "int") == 0) cast_type = "int64_t";
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
            /* String→numeric conversion */
            EzType *carg_t = cg->type_table ? typetable_get(cg->type_table, carg) : NULL;
            bool is_string_src = (carg->kind == NODE_STRING_VALUE || carg->kind == NODE_INTERPOLATED_STRING ||
                                  (carg_t && carg_t->kind == TK_STRING));
            if (is_string_src && (strcmp(func, "int") == 0 || strcmp(func, "uint") == 0)) {
                emit(cg, "ez_builtin_string_to_int(");
                emit_expression(cg, carg);
                emit(cg, ")");
            } else if (is_string_src && strcmp(func, "float") == 0) {
                emit(cg, "ez_builtin_string_to_float(");
                emit_expression(cg, carg);
                emit(cg, ")");
            } else if (strcmp(func, "int") == 0 &&
                (carg->kind == NODE_FLOAT_VALUE || (carg_t && carg_t->kind == TK_FLOAT))) {
                /* Use overflow-safe conversion for float→int */
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
        if (at && (at->kind == TK_ARRAY || at->kind == TK_MAP || at->kind == TK_STRUCT)) {
            /* Route every container kind through the unified deep-copy
             * emitter so nested collections, structs containing
             * collections, collections containing such structs, and any
             * transitive mix of those all come out fully independent
             * of the source (#1465, #1466). */
            int t = next_dc_tag();
            const char *c_type = (at->kind == TK_ARRAY) ? "EzArray"
                               : (at->kind == TK_MAP) ? "EzMap"
                               : ez_type_to_c_cg(cg, at->name);
            emitf(cg, "({ %s _cpy%d = ", c_type, t);
            emit_expression(cg, arg);
            emit(cg, "; ");
            char src_var[32];
            snprintf(src_var, sizeof(src_var), "_cpy%d", t);
            char full_tn[256];
            if (at->kind == TK_ARRAY) {
                snprintf(full_tn, sizeof(full_tn), "[%s]",
                    at->element_type ? at->element_type : "");
            } else if (at->kind == TK_MAP) {
                snprintf(full_tn, sizeof(full_tn), "map[%s:%s]",
                    at->key_type ? at->key_type : "",
                    at->value_type ? at->value_type : "");
            } else {
                snprintf(full_tn, sizeof(full_tn), "%s", at->name ? at->name : "");
            }
            emit_value_deep_copy(cg, full_tn, src_var);
            emit(cg, "; })");
        } else {
            emit_expression(cg, arg);
        }
        return true;
    }

    if (strcmp(func, "print") == 0 && node->data.call.arg_count > 0) {
        if (emit_composite_print(cg, node, "stdout", false)) return true;
        AstNode *arg = node->data.call.args[0];
        EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (arg_t && arg_t->kind == TK_ERROR) {
            emit(cg, "ez_builtin_print_str(");
            emit_expression(cg, arg);
            emit(cg, " ? ");
            emit_expression(cg, arg);
            emit(cg, "->message : ez_string_lit(\"nil\"))");
        } else {
            const char *bi_type = resolve_bigint_type(cg, arg);
            if (bi_type) {
                emitf(cg, "ez_builtin_print_str(%s_to_string(ez_default_arena, ", bigint_prefix(bi_type));
                emit_expression(cg, arg);
                emit(cg, "))");
            } else {
                emitf(cg, "ez_builtin_print%s(", resolve_print_suffix(cg, arg));
                emit_expression(cg, arg);
                emit(cg, ")");
            }
        }
        return true;
    }

    /* to_char(str, index) — extract Nth Unicode codepoint */
    if (strcmp(func, "to_char") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_builtin_to_char(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emitf(cg, ", \"%s\", %d)", cg->file, node->token.line);
        return true;
    }

    /* char_count(str) — return Unicode codepoint count */
    if (strcmp(func, "char_count") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_builtin_char_count(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    /* c_string(ptr) — convert C char* to EZ string */
    if (strcmp(func, "c_string") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_string_lit((const char *)");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
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
        emitf(cg, ", __FILE__, %d)", node->token.line);
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
    if (strcmp(func, "raw_copy") == 0 && node->data.call.arg_count == 3) {
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
    if (strcmp(func, "fill") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_mem_set(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "init") == 0 && node->data.call.arg_count == 2) {
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
            emit(cg, ", sizeof(_v)); *(__typeof__(_v) *)_p = _v; (__typeof__(_v) *)_p; })");
        }
        return true;
    }
    return false;
}

/* --- @math module --- */

static bool emit_math_call(CodeGen *cg, AstNode *node, const char *func) {
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
    if ((strcmp(func, "min") == 0 || strcmp(func, "max") == 0) && node->data.call.arg_count == 2) {
        EzType *at = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        emitf(cg, "ez_math_%s_%s(", func, (at && at->kind == TK_FLOAT) ? "float" : "int");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "clamp") == 0 && node->data.call.arg_count == 3) {
        EzType *at = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        emitf(cg, "ez_math_clamp_%s(", (at && at->kind == TK_FLOAT) ? "float" : "int");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
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
    if (strcmp(func, "get_keys") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_maps_get_keys(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "get_values") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_maps_get_values(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "has_key") == 0) {
        /* Key buffer must match the map's declared key storage type
         * (ez_map_elem_c_type), not whatever C type the argument expression
         * happens to have — otherwise the hash/memcmp compares the wrong
         * number of bytes. Issue #1430 follow-up. */
        const char *c_key = "int64_t";
        EzType *map_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        if (map_t && map_t->kind == TK_MAP && map_t->key_type)
            c_key = ez_map_elem_c_type(cg, map_t->key_type);
        emitf(cg, "({ %s _hk = ", c_key);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_maps_has_key(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_hk); })");
        return true;
    }
    if (strcmp(func, "remove_key") == 0 && node->data.call.arg_count == 2) {
        const char *c_key = "int64_t";
        EzType *map_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        if (map_t && map_t->kind == TK_MAP && map_t->key_type)
            c_key = ez_map_elem_c_type(cg, map_t->key_type);
        emitf(cg, "({ %s _rk = ", c_key);
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
    if (strcmp(func, "is_empty") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_maps_is_empty(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "merge") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_maps_merge(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "contains_value") == 0 && node->data.call.arg_count == 2) {
        /* Determine value type from map to ensure correct size */
        EzType *map_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *c_val_type = "int64_t";
        if (map_t && map_t->value_type) {
            EzType *vt = type_from_name(map_t->value_type);
            if (vt->kind == TK_FLOAT) c_val_type = "double";
            else if (vt->kind == TK_BOOL) c_val_type = "bool";
            else if (vt->kind == TK_STRING) c_val_type = "EzString";
        }
        emitf(cg, "({ %s _cv = ", c_val_type);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_maps_contains_value(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_cv); })");
        return true;
    }
    if (strcmp(func, "get_or_default") == 0 && node->data.call.arg_count == 3) {
        /* get_or_default(m, key, default) — lookup key, return default if missing */
        emit(cg, "({ __auto_type _gk = ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; void *_gv = ez_map_get(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_gk); _gv ? *(__typeof__(");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ") *)_gv : ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "; })");
        return true;
    }
    return false;
}

/* --- @time module --- */

static bool emit_time_call(CodeGen *cg, AstNode *node, const char *func) {
    bool needs_arena = (strcmp(func, "format") == 0 || strcmp(func, "to_iso") == 0 ||
        strcmp(func, "date") == 0 || strcmp(func, "to_time") == 0);
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
    if (strcmp(func, "generate_hyphenated") == 0) {
        emit(cg, "ez_uuid_generate(ez_default_arena)"); return true;
    }
    if (strcmp(func, "generate") == 0) {
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
    if (strcmp(func, "is_valid") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_regex_is_valid(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "is_match") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_regex_match(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "find") == 0 && node->data.call.arg_count == 2) {
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        emitf(cg, "ez_regex_find%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "find_all") == 0 && node->data.call.arg_count == 2) {
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        emitf(cg, "ez_regex_find_all%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "replace") == 0 && node->data.call.arg_count == 3) {
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        emitf(cg, "ez_regex_replace%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "split") == 0 && node->data.call.arg_count == 2) {
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        emitf(cg, "ez_regex_split%s(ez_default_arena, ", is_multi_var ? "_result" : "");
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
    if (strcmp(func, "add_router") == 0) {
        emit(cg, "ez_server_router()");
        return true;
    }
    if (strcmp(func, "add_route") == 0 && node->data.call.arg_count == 4) {
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
    bool is_multi_var = cg->current_var_name != NULL &&
        strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
    const char *sfx = is_multi_var ? "_result" : "";
    if (strcmp(func, "get") == 0 && node->data.call.arg_count == 1) {
        emitf(cg, "ez_http_get%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "post") == 0 && node->data.call.arg_count == 2) {
        emitf(cg, "ez_http_post%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "put") == 0 && node->data.call.arg_count == 2) {
        emitf(cg, "ez_http_put%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "delete") == 0 && node->data.call.arg_count == 1) {
        emitf(cg, "ez_http_delete%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "head") == 0 && node->data.call.arg_count == 1) {
        emitf(cg, "ez_http_head%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "patch") == 0 && node->data.call.arg_count == 2) {
        emitf(cg, "ez_http_patch%s(ez_default_arena, ", sfx);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @net module --- */

static bool emit_net_call(CodeGen *cg, AstNode *node, const char *func) {
    bool is_multi_var = cg->current_var_name != NULL &&
        strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
    if (strcmp(func, "connect") == 0 && node->data.call.arg_count == 2) {
        emitf(cg, "ez_net_dial%s(ez_default_arena, ", is_multi_var ? "_result" : "");
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
        if (is_multi_var) {
            emit(cg, "ez_net_send_result(ez_default_arena, ");
        } else {
            emit(cg, "ez_net_send(");
        }
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "receive") == 0 && node->data.call.arg_count == 2) {
        emitf(cg, "ez_net_recv%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "listen") == 0 && node->data.call.arg_count == 1) {
        emitf(cg, "ez_net_listen%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "accept") == 0 && node->data.call.arg_count == 1) {
        emitf(cg, "ez_net_accept%s(ez_default_arena, ", is_multi_var ? "_result" : "");
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
        emitf(cg, "ez_net_resolve%s(ez_default_arena, ", is_multi_var ? "_result" : "");
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
    if (has_endian || strcmp(func, "encode_u8") == 0 || strcmp(func, "decode_u8") == 0 ||
        strcmp(func, "encode_i8") == 0 || strcmp(func, "decode_i8") == 0) {
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
    bool is_multi_var = cg->current_var_name != NULL &&
        strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
    if (strcmp(func, "decode") == 0 || strcmp(func, "parse") == 0) {
        emit(cg, "ez_csv_parse(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "read_file") == 0) {
        emitf(cg, "ez_csv_read%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "encode") == 0) {
        emit(cg, "({ EzArray _csv_a = ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, "; ez_csv_stringify(ez_default_arena, &_csv_a); })");
        return true;
    }
    if (strcmp(func, "write_file") == 0) {
        if (is_multi_var) {
            emit(cg, "({ EzArray _csv_a = ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, "; ez_csv_write_result(ez_default_arena, ");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", &_csv_a); })");
        } else {
            emit(cg, "({ EzArray _csv_a = ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, "; ez_csv_write(ez_default_arena, ");
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", &_csv_a); })");
        }
        return true;
    }
    return false;
}

/* --- @json module --- */

static bool emit_json_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "encode") == 0) {
        AstNode *arg = node->data.call.args[0];
        EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (arg_t && arg_t->kind == TK_MAP) {
            /* Typed map: dispatch based on value type */
            if (arg_t->value_type && strcmp(arg_t->value_type, "int") == 0) {
                emit(cg, "ez_json_encode_map_int(ez_default_arena, &");
            } else if (arg_t->value_type && strcmp(arg_t->value_type, "float") == 0) {
                emit(cg, "ez_json_encode_map_float(ez_default_arena, &");
            } else if (arg_t->value_type && strcmp(arg_t->value_type, "bool") == 0) {
                emit(cg, "ez_json_encode_map_bool(ez_default_arena, &");
            } else {
                emit(cg, "ez_json_encode_map(ez_default_arena, &");
            }
            emit_expression(cg, arg);
            emit(cg, ")");
        } else if (arg_t && arg_t->kind == TK_ARRAY) {
            /* Typed array: dispatch based on element type */
            if (arg_t->element_type && strcmp(arg_t->element_type, "float") == 0) {
                emit(cg, "ez_json_encode_array_float(ez_default_arena, &");
            } else if (arg_t->element_type && strcmp(arg_t->element_type, "string") == 0) {
                emit(cg, "ez_json_encode_array_string(ez_default_arena, &");
            } else if (arg_t->element_type && strcmp(arg_t->element_type, "bool") == 0) {
                emit(cg, "ez_json_encode_array_bool(ez_default_arena, &");
            } else {
                emit(cg, "ez_json_encode_array_int(ez_default_arena, &");
            }
            emit_expression(cg, arg);
            emit(cg, ")");
        } else if (arg_t && arg_t->kind == TK_INT) {
            /* Int: format as JSON number string */
            emit(cg, "({ char _jbuf[32]; snprintf(_jbuf, sizeof(_jbuf), \"%\" PRId64, (int64_t)");
            emit_expression(cg, arg);
            emit(cg, "); ez_string_new(ez_default_arena, _jbuf, (int32_t)strlen(_jbuf)); })");
        } else if (arg_t && arg_t->kind == TK_FLOAT) {
            emit(cg, "({ char _jbuf[64]; snprintf(_jbuf, sizeof(_jbuf), \"%g\", (double)");
            emit_expression(cg, arg);
            emit(cg, "); ez_string_new(ez_default_arena, _jbuf, (int32_t)strlen(_jbuf)); })");
        } else if (arg_t && arg_t->kind == TK_BOOL) {
            emit(cg, "(");
            emit_expression(cg, arg);
            emit(cg, " ? ez_string_lit(\"true\") : ez_string_lit(\"false\"))");
        } else if (arg_t && arg_t->kind == TK_STRING) {
            /* String: wrap in quotes */
            emit(cg, "({ EzString _js = ");
            emit_expression(cg, arg);
            emit(cg, "; char *_jbuf = ez_arena_alloc(ez_default_arena, _js.len + 3); ");
            emit(cg, "_jbuf[0] = '\"'; memcpy(_jbuf+1, _js.data, _js.len); _jbuf[_js.len+1] = '\"'; _jbuf[_js.len+2] = '\\0'; ");
            emit(cg, "ez_string_new(ez_default_arena, _jbuf, _js.len + 2); })");
        } else {
            /* Fallback: store in temp to allow & */
            emit(cg, "({ __auto_type _jtmp = ");
            emit_expression(cg, arg);
            emit(cg, "; ez_json_encode_map(ez_default_arena, (EzMap *)&_jtmp); })");
        }
        return true;
    }
    if (strcmp(func, "decode") == 0) {
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        emitf(cg, "ez_json_decode%s(ez_default_arena, ", is_multi_var ? "_result" : "");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    /* #1496: json.parse() — dispatch to per-struct helper when the
     * call node's typetable entry is a #json struct (pushed by the
     * var_decl handler via #1507). Falls back to ez_json_decode for
     * the map-based path. */
    if (strcmp(func, "parse") == 0 && node->data.call.arg_count >= 1) {
        EzType *target_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
        if (target_t && target_t->kind == TK_STRUCT && target_t->name) {
            AstNode *sdecl = find_struct_decl(cg, target_t->name);
            if (sdecl && sdecl->data.struct_decl.is_json) {
                emitf(cg, "ez_json_parse_%s(ez_default_arena, ", target_t->name);
                emit_expression(cg, node->data.call.args[0]);
                emit(cg, ")");
                return true;
            }
        }
        /* Fallback: map-based decode */
        emit(cg, "ez_json_decode(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    /* #1496: json.stringify() — dispatch to per-struct helper when
     * the argument is a #json struct. */
    if (strcmp(func, "stringify") == 0 && node->data.call.arg_count >= 1) {
        AstNode *arg = node->data.call.args[0];
        EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, arg) : NULL;
        if (arg_t && arg_t->kind == TK_STRUCT && arg_t->name) {
            AstNode *sdecl = find_struct_decl(cg, arg_t->name);
            if (sdecl && sdecl->data.struct_decl.is_json) {
                emitf(cg, "ez_json_stringify_%s(ez_default_arena, ", arg_t->name);
                emit_expression(cg, arg);
                emit(cg, ")");
                return true;
            }
        }
        /* Fallback: encode as map */
        emit(cg, "({ __auto_type _jtmp = ");
        emit_expression(cg, arg);
        emit(cg, "; ez_json_encode_map(ez_default_arena, (EzMap *)&_jtmp); })");
        return true;
    }
    if (strcmp(func, "is_valid") == 0) {
        emit(cg, "ez_json_is_valid(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "pretty_print") == 0) {
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
    bool is_fallible = (strcmp(func, "open") == 0 || strcmp(func, "exec") == 0 ||
        strcmp(func, "query") == 0);
    bool is_multi_var = cg->current_var_name != NULL &&
        strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
    if (strcmp(func, "open") == 0) {
        emitf(cg, "ez_sqlite_open%s(ez_default_arena, ", (is_fallible && is_multi_var) ? "_result" : "");
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
        if (is_multi_var) {
            emit(cg, "ez_sqlite_exec_result(ez_default_arena, ");
        } else {
            emit(cg, "ez_sqlite_exec(");
        }
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "query") == 0) {
        emitf(cg, "ez_sqlite_query%s(ez_default_arena, ", is_multi_var ? "_result" : "");
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
    if (strcmp(func, "rand_float") == 0) {
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
    if (strcmp(func, "rand_int") == 0) {
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
    if (strcmp(func, "rand_bool") == 0) { emit(cg, "ez_random_bool()"); return true; }
    if (strcmp(func, "rand_byte") == 0) { emit(cg, "ez_random_byte()"); return true; }
    if (strcmp(func, "rand_char") == 0) {
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
        /* Determine element C type from the array's type info */
        const char *c_elem = "int64_t";
        EzType *arr_t = cg->type_table
            ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        if (arr_t && arr_t->kind == TK_ARRAY && arr_t->element_type) {
            EzType *et = type_from_name(arr_t->element_type);
            if (et->kind == TK_FLOAT) c_elem = "double";
            else if (et->kind == TK_BOOL) c_elem = "bool";
            else if (et->kind == TK_STRING) c_elem = "EzString";
            else if (et->kind == TK_CHAR) c_elem = "int32_t";
            else if (et->kind == TK_BYTE) c_elem = "uint8_t";
            else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, arr_t->element_type);
        }
        emit(cg, "({ int32_t _ri = ez_random_int_max(");
        emit_expression(cg, node->data.call.args[0]);
        emitf(cg, ".len); *(%s *)ez_array_get_ptr(&", c_elem);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", _ri, __FILE__, __LINE__); })");
        return true;
    }
    if (strcmp(func, "random_hex") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_random_hex(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @arrays module --- */

static bool emit_arrays_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "append") == 0 && node->data.call.arg_count == 2) {
        EzType *val_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[1]) : NULL;
        EzType *arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *elem_tn = (arr_t && arr_t->kind == TK_ARRAY) ? arr_t->element_type : NULL;
        bool elem_is_string = (val_t && val_t->kind == TK_STRING) ||
            (elem_tn && strcmp(elem_tn, "string") == 0);
        const char *c_elem = "__auto_type";
        if (val_t) {
            switch (val_t->kind) {
            case TK_INT: c_elem = "int64_t"; break;
            case TK_UINT: c_elem = "uint64_t"; break;
            case TK_FLOAT: c_elem = "double"; break;
            case TK_BOOL: c_elem = "bool"; break;
            case TK_STRING: c_elem = "EzString"; break;
            case TK_ARRAY: c_elem = "EzArray"; break;
            case TK_MAP: c_elem = "EzMap"; break;
            default: break;
            }
            if (val_t->kind == TK_STRUCT) {
                c_elem = ez_type_to_c_cg(cg, val_t->name);
            }
        } else if (elem_is_string) {
            c_elem = "EzString";
        } else if (elem_tn) {
            EzType *et = type_from_name(elem_tn);
            if (et->kind == TK_ARRAY) c_elem = "EzArray";
            else if (et->kind == TK_MAP) c_elem = "EzMap";
            else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, elem_tn);
        }
        const char *alloc_arena = cg->loop_scope_depth > 0 ? "_ez_outer_arena" : "ez_default_arena";
        emitf(cg, "{ %s _av = ", c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ");
        /* #1521: escape arena-allocated data to the outer arena when
         * inside a loop scope. Strings get a simple copy; arrays, maps,
         * and structs with embedded pointers need a full deep copy. */
        if (cg->loop_scope_depth > 0) {
            if (elem_is_string) {
                emitf(cg, "_av = ez_string_new(%s, _av.data, _av.len); ", alloc_arena);
            } else if (elem_tn && type_needs_deep_copy(cg, elem_tn)) {
                emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _av = ", alloc_arena);
                emit_value_deep_copy(cg, elem_tn, "_av");
                emit(cg, "; ez_default_arena = _esc; } ");
            }
        }
        emitf(cg, "ez_arrays_append(%s, &", alloc_arena);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_av); }");
        return true;
    }
    if (strcmp(func, "insert_at") == 0 && node->data.call.arg_count == 3) {
        EzType *val_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[2]) : NULL;
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
        const char *ia_arena = cg->loop_scope_depth > 0 ? "_ez_outer_arena" : "ez_default_arena";
        emitf(cg, "{ %s _iv = ", c_elem);
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "; ");
        EzType *ia_arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *ia_elem_tn = (ia_arr_t && ia_arr_t->kind == TK_ARRAY) ? ia_arr_t->element_type : NULL;
        bool ia_str = (val_t && val_t->kind == TK_STRING) ||
            (ia_elem_tn && strcmp(ia_elem_tn, "string") == 0);
        if (cg->loop_scope_depth > 0) {
            if (ia_str) {
                emitf(cg, "_iv = ez_string_new(%s, _iv.data, _iv.len); ", ia_arena);
            } else if (ia_elem_tn && type_needs_deep_copy(cg, ia_elem_tn)) {
                emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _iv = ", ia_arena);
                emit_value_deep_copy(cg, ia_elem_tn, "_iv");
                emit(cg, "; ez_default_arena = _esc; } ");
            }
        }
        emitf(cg, "ez_arrays_insert_at(%s, &", ia_arena);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", &_iv); }");
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
    if (strcmp(func, "sort_asc") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_arrays_sort_asc(&");
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
    /* prepend/fill need special value wrapping */
    if (strcmp(func, "prepend") == 0 && node->data.call.arg_count == 2) {
        const char *pp_arena = cg->loop_scope_depth > 0 ? "_ez_outer_arena" : "ez_default_arena";
        EzType *pp_arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *pp_elem_tn = (pp_arr_t && pp_arr_t->kind == TK_ARRAY) ? pp_arr_t->element_type : NULL;
        bool pp_str = pp_elem_tn && strcmp(pp_elem_tn, "string") == 0;
        const char *pp_c_elem = "int64_t";
        if (pp_elem_tn) {
            EzType *pet = type_from_name(pp_elem_tn);
            if (pet->kind == TK_FLOAT) pp_c_elem = "double";
            else if (pet->kind == TK_BOOL) pp_c_elem = "bool";
            else if (pet->kind == TK_STRING) pp_c_elem = "EzString";
            else if (pet->kind == TK_ARRAY) pp_c_elem = "EzArray";
            else if (pet->kind == TK_MAP) pp_c_elem = "EzMap";
            else if (pet->kind == TK_STRUCT) pp_c_elem = ez_type_to_c_cg(cg, pp_elem_tn);
        }
        emitf(cg, "{ %s _pv = ", pp_c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ");
        if (cg->loop_scope_depth > 0) {
            if (pp_str) {
                emitf(cg, "_pv = ez_string_new(%s, _pv.data, _pv.len); ", pp_arena);
            } else if (pp_elem_tn && type_needs_deep_copy(cg, pp_elem_tn)) {
                emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _pv = ", pp_arena);
                emit_value_deep_copy(cg, pp_elem_tn, "_pv");
                emit(cg, "; ez_default_arena = _esc; } ");
            }
        }
        emitf(cg, "ez_arrays_prepend(%s, &", pp_arena);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_pv); }");
        return true;
    }
    if (strcmp(func, "fill") == 0 && node->data.call.arg_count == 3) {
        EzType *fl_arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *fl_c_elem = "int64_t";
        if (fl_arr_t && fl_arr_t->kind == TK_ARRAY && fl_arr_t->element_type) {
            EzType *fet = type_from_name(fl_arr_t->element_type);
            if (fet->kind == TK_FLOAT) fl_c_elem = "double";
            else if (fet->kind == TK_BOOL) fl_c_elem = "bool";
            else if (fet->kind == TK_STRING) fl_c_elem = "EzString";
        }
        emitf(cg, "{ %s _fv = ", fl_c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_arrays_fill(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_fv, ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "); }");
        return true;
    }
    /* Generic: arrays.func(&arr, ...) or arrays.func(arena, &arr, ...) */
    bool needs_arena = (strcmp(func, "reverse") == 0 || strcmp(func, "slice") == 0 ||
        strcmp(func, "concat") == 0 || strcmp(func, "deduplicate") == 0 ||
        strcmp(func, "flatten") == 0 || strcmp(func, "split_every") == 0 ||
        strcmp(func, "pair") == 0);
    bool ref_args = (strcmp(func, "concat") == 0 || strcmp(func, "pair") == 0);
    emitf(cg, "ez_arrays_%s(", func);
    if (needs_arena) emit(cg, "ez_default_arena, ");
    emit(cg, "&");
    emit_expression(cg, node->data.call.args[0]);
    for (int i = 1; i < node->data.call.arg_count; i++) {
        emit(cg, ", ");
        if (ref_args) emit(cg, "&");
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
    if (strcmp(func, "current_dir") == 0) {
        emit(cg, "ez_os_cwd(ez_default_arena)"); return true;
    }
    if (strcmp(func, "hostname") == 0) {
        emit(cg, "ez_os_hostname(ez_default_arena)"); return true;
    }
    if (strcmp(func, "current_os") == 0) { emit(cg, "ez_os_current_os()"); return true; }
    if (strcmp(func, "arch") == 0) { emit(cg, "ez_os_arch()"); return true; }
    if (strcmp(func, "pid") == 0) { emit(cg, "ez_os_pid()"); return true; }
    if (strcmp(func, "set_env") == 0) {
        emit(cg, "ez_os_set_env(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if ((strcmp(func, "get_first") == 0 || strcmp(func, "get_last") == 0 ||
         strcmp(func, "remove_last") == 0 || strcmp(func, "remove_first") == 0) &&
        node->data.call.arg_count == 1) {
        emitf(cg, "ez_arrays_%s(&", func);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "prepend") == 0 && node->data.call.arg_count == 2) {
        const char *pp2_arena = cg->loop_scope_depth > 0 ? "_ez_outer_arena" : "ez_default_arena";
        EzType *pp2_arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *pp2_elem_tn = (pp2_arr_t && pp2_arr_t->kind == TK_ARRAY) ? pp2_arr_t->element_type : NULL;
        bool pp2_str = pp2_elem_tn && strcmp(pp2_elem_tn, "string") == 0;
        const char *pp2_c_elem = "int64_t";
        if (pp2_elem_tn) {
            EzType *pet2 = type_from_name(pp2_elem_tn);
            if (pet2->kind == TK_FLOAT) pp2_c_elem = "double";
            else if (pet2->kind == TK_BOOL) pp2_c_elem = "bool";
            else if (pet2->kind == TK_STRING) pp2_c_elem = "EzString";
            else if (pet2->kind == TK_ARRAY) pp2_c_elem = "EzArray";
            else if (pet2->kind == TK_MAP) pp2_c_elem = "EzMap";
            else if (pet2->kind == TK_STRUCT) pp2_c_elem = ez_type_to_c_cg(cg, pp2_elem_tn);
        }
        emitf(cg, "{ %s _pv = ", pp2_c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ");
        if (cg->loop_scope_depth > 0) {
            if (pp2_str) {
                emitf(cg, "_pv = ez_string_new(%s, _pv.data, _pv.len); ", pp2_arena);
            } else if (pp2_elem_tn && type_needs_deep_copy(cg, pp2_elem_tn)) {
                emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _pv = ", pp2_arena);
                emit_value_deep_copy(cg, pp2_elem_tn, "_pv");
                emit(cg, "; ez_default_arena = _esc; } ");
            }
        }
        emitf(cg, "ez_arrays_prepend(%s, &", pp2_arena);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_pv); }");
        return true;
    }
    if (strcmp(func, "fill") == 0 && node->data.call.arg_count == 3) {
        EzType *fl_arr_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[0]) : NULL;
        const char *fl_c_elem = "int64_t";
        if (fl_arr_t && fl_arr_t->kind == TK_ARRAY && fl_arr_t->element_type) {
            EzType *fet = type_from_name(fl_arr_t->element_type);
            if (fet->kind == TK_FLOAT) fl_c_elem = "double";
            else if (fet->kind == TK_BOOL) fl_c_elem = "bool";
            else if (fet->kind == TK_STRING) fl_c_elem = "EzString";
        }
        emitf(cg, "{ %s _fv = ", fl_c_elem);
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, "; ez_arrays_fill(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &_fv, ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "); }");
        return true;
    }
    if (strcmp(func, "count") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_arrays_count(&");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if ((strcmp(func, "deduplicate") == 0 || strcmp(func, "flatten") == 0) &&
        node->data.call.arg_count == 1) {
        emitf(cg, "ez_arrays_%s(ez_default_arena, &", func);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "split_every") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_arrays_split_every(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "pair") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_arrays_pair(ez_default_arena, &");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", &");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if ((strcmp(func, "get_sum") == 0 || strcmp(func, "get_min") == 0 ||
         strcmp(func, "get_max") == 0) && node->data.call.arg_count == 1) {
        emitf(cg, "ez_arrays_%s(&", func);
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @io module --- */

static bool emit_io_call(CodeGen *cg, AstNode *node, const char *func) {
    bool is_fallible = (strcmp(func, "read_file") == 0 ||
        strcmp(func, "write_file") == 0 ||
        strcmp(func, "delete_file") == 0 ||
        strcmp(func, "append_file") == 0 ||
        strcmp(func, "rename_file") == 0 ||
        strcmp(func, "copy_file") == 0 ||
        strcmp(func, "move_file") == 0 ||
        strcmp(func, "list_dir") == 0 ||
        strcmp(func, "make_dir") == 0 ||
        strcmp(func, "make_dir_all") == 0 ||
        strcmp(func, "remove_dir") == 0 ||
        strcmp(func, "remove_dir_all") == 0 ||
        strcmp(func, "walk") == 0);
    bool needs_arena = (strcmp(func, "read_file") == 0 ||
        strcmp(func, "list_dir") == 0 ||
        strcmp(func, "walk") == 0 ||
        strcmp(func, "glob") == 0 ||
        strcmp(func, "path_join") == 0 ||
        strcmp(func, "dirname") == 0 ||
        strcmp(func, "basename") == 0 ||
        strcmp(func, "extension") == 0 ||
        strcmp(func, "normalize") == 0);
    if (is_fallible) {
        /* Use non-result version when assigned to a single variable (typed or
         * inferred).  Use _result version only for multi-var destructuring
         * (temp vars prefixed with _ez_tmp). */
        bool is_multi_var = cg->current_var_name != NULL &&
            strncmp(cg->current_var_name, "_ez_tmp", 7) == 0;
        bool use_non_result = !is_multi_var;
        if (use_non_result) {
            if (needs_arena) {
                emitf(cg, "ez_io_%s(ez_default_arena, ", func);
            } else {
                emitf(cg, "ez_io_%s(", func);
            }
            for (int i = 0; i < node->data.call.arg_count; i++) {
                if (i > 0) emit(cg, ", ");
                emit_expression(cg, node->data.call.args[i]);
            }
            emit(cg, ")");
        } else {
            emitf(cg, "ez_io_%s_result(ez_default_arena, ", func);
            for (int i = 0; i < node->data.call.arg_count; i++) {
                if (i > 0) emit(cg, ", ");
                emit_expression(cg, node->data.call.args[i]);
            }
            emit(cg, ")");
        }
        return true;
    }
    /* Non-fallible functions */
    if (needs_arena) {
        emitf(cg, "ez_io_%s(ez_default_arena, ", func);
    } else {
        emitf(cg, "ez_io_%s(", func);
    }
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
    return true;
}

/* --- @strings module --- */

static bool emit_strings_call(CodeGen *cg, AstNode *node, const char *func) {
    bool needs_arena = (strcmp(func, "to_upper") == 0 || strcmp(func, "to_lower") == 0 ||
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

    if (strcmp(func, "format") == 0 && node->data.call.arg_count >= 1) {
        emit(cg, "ez_string_format(ez_default_arena, ");
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

    if (strcmp(func, "pad_left") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_fmt_pad_left(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", (char)(");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "))");
        return true;
    }

    if (strcmp(func, "pad_right") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_fmt_pad_right(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", (char)(");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "))");
        return true;
    }

    if (strcmp(func, "center") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_fmt_center(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", (char)(");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, "))");
        return true;
    }

    if (strcmp(func, "int_to_hex") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_fmt_int_to_hex(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "int_to_binary") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_fmt_int_to_binary(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "int_to_octal") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_fmt_int_to_octal(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "float_fixed") == 0 && node->data.call.arg_count == 2) {
        emit(cg, "ez_fmt_float_fixed(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }

    if (strcmp(func, "float_sci") == 0 && node->data.call.arg_count == 1) {
        emit(cg, "ez_fmt_float_sci(ez_default_arena, ");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }

    return false;
}

static bool emit_threads_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "spawn") == 0 && node->data.call.arg_count >= 1) {
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
    if (strcmp(func, "get_id") == 0) {
        emit(cg, "ez_threads_id()");
        return true;
    }
    return false;
}

/* --- @sync module --- */

static bool emit_sync_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "mutex") == 0) {
        emit(cg, "ez_sync_mutex()");
        return true;
    }
    if (strcmp(func, "lock") == 0) {
        emit(cg, "ez_sync_lock(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "unlock") == 0) {
        emit(cg, "ez_sync_unlock(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "destroy") == 0) {
        emit(cg, "ez_sync_destroy(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @atomic module --- */

static bool emit_atomic_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "spinlock") == 0) {
        emit(cg, "ez_atomic_mod_spinlock()");
        return true;
    }
    if (strcmp(func, "fence") == 0) {
        emit(cg, "ez_atomic_mod_fence()");
        return true;
    }
    /* Single-argument functions: load, spin_lock, spin_unlock, spin_trylock */
    if (node->data.call.arg_count == 1) {
        if (strcmp(func, "load") == 0 || strcmp(func, "spin_lock") == 0 ||
            strcmp(func, "spin_trylock") == 0 || strcmp(func, "spin_unlock") == 0) {
            emitf(cg, "ez_atomic_mod_%s(", func);
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ")");
            return true;
        }
    }
    /* Two-argument functions: store, add, sub, exchange, and, or, xor */
    if (node->data.call.arg_count == 2) {
        if (strcmp(func, "store") == 0 || strcmp(func, "add") == 0 ||
            strcmp(func, "sub") == 0 || strcmp(func, "exchange") == 0 ||
            strcmp(func, "and") == 0 || strcmp(func, "or") == 0 ||
            strcmp(func, "xor") == 0) {
            emitf(cg, "ez_atomic_mod_%s(", func);
            emit_expression(cg, node->data.call.args[0]);
            emit(cg, ", ");
            emit_expression(cg, node->data.call.args[1]);
            emit(cg, ")");
            return true;
        }
    }
    /* Three-argument: cas */
    if (strcmp(func, "cas") == 0 && node->data.call.arg_count == 3) {
        emit(cg, "ez_atomic_mod_cas(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[2]);
        emit(cg, ")");
        return true;
    }
    return false;
}

/* --- @channels module --- */

static bool emit_channels_call(CodeGen *cg, AstNode *node, const char *func) {
    if (strcmp(func, "open") == 0) {
        emit(cg, "ez_channels_open(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "send") == 0) {
        emit(cg, "ez_channels_send(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ", ");
        emit_expression(cg, node->data.call.args[1]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "receive") == 0) {
        emit(cg, "ez_channels_receive(");
        emit_expression(cg, node->data.call.args[0]);
        emit(cg, ")");
        return true;
    }
    if (strcmp(func, "close") == 0) {
        emit(cg, "ez_channels_close(");
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
        /* Resolve import aliases: io@std → io.println maps to std.println */
        if (module) module = resolve_alias(cg, module);

        /* No-module builtins (println, len, type_of, etc.) */
        /* Also handle std.println() — std module functions are builtins */
        if (!module && emit_builtin_call(cg, node, func)) return;

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
            {"sync",     emit_sync_call},
            {"atomic",   emit_atomic_call},
            {"channels", emit_channels_call},
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
        /* Unqualified call not handled by builtins — try 'using' modules.
         * We must verify the function name belongs to the module before calling
         * the handler, since some handlers emit code for any function name. */
        if (!module) {
            /* Map function names to their module. Only includes functions that
             * differ from builtins (std builtins are already handled above). */
            static const struct { const char *func; const char *mod; } func_to_mod[] = {
                /* @strings */
                {"to_upper","strings"},{"to_lower","strings"},{"trim","strings"},
                {"trim_left","strings"},{"trim_right","strings"},{"replace","strings"},
                {"repeat","strings"},{"reverse","strings"},{"slice","strings"},
                {"join","strings"},{"contains","strings"},{"starts_with","strings"},
                {"ends_with","strings"},{"is_empty","strings"},{"index_of","strings"},
                {"count","strings"},{"split","strings"},
                /* @math */
                {"abs","math"},{"neg","math"},{"sign","math"},{"min","math"},{"max","math"},
                {"clamp","math"},{"floor","math"},{"ceil","math"},{"round","math"},{"trunc","math"},
                {"pow","math"},{"sqrt","math"},{"cbrt","math"},{"hypot","math"},{"exp","math"},
                {"exp2","math"},{"log","math"},{"log2","math"},{"log10","math"},{"log_base","math"},
                {"sin","math"},{"cos","math"},{"tan","math"},{"asin","math"},{"acos","math"},
                {"atan","math"},{"atan2","math"},{"sinh","math"},{"cosh","math"},{"tanh","math"},
                {"deg_to_rad","math"},{"rad_to_deg","math"},{"factorial","math"},{"gcd","math"},
                {"lcm","math"},{"is_prime","math"},{"is_even","math"},{"is_odd","math"},
                {"is_infinite","math"},{"is_nan","math"},{"is_finite","math"},{"lerp","math"},
                {"distance","math"},
                /* @arrays */
                {"append","arrays"},{"insert_at","arrays"},{"prepend","arrays"},
                {"remove_at","arrays"},{"sort_asc","arrays"},{"sort_desc","arrays"},
                {"concat","arrays"},{"get_sum","arrays"},{"get_min","arrays"},{"get_max","arrays"},
                {"clear","arrays"},{"get_first","arrays"},{"get_last","arrays"},
                {"remove_last","arrays"},{"remove_first","arrays"},
                {"fill","arrays"},{"deduplicate","arrays"},{"flatten","arrays"},
                {"reverse","arrays"},{"slice","arrays"},
                {"split_every","arrays"},{"pair","arrays"},{"count","arrays"},
                {"index_of","arrays"},{"is_empty","arrays"},{"contains","arrays"},
                /* @maps */
                {"has_key","maps"},{"keys","maps"},{"values","maps"},{"get_keys","maps"},
                {"get_values","maps"},{"remove_key","maps"},{"clear","maps"},
                {"is_empty","maps"},{"merge","maps"},{"contains_value","maps"},
                {"get_or_default","maps"},
                /* @random */
                {"rand_float","random"},{"rand_int","random"},{"rand_bool","random"},
                {"rand_byte","random"},{"rand_char","random"},{"random_hex","random"},
                {"choice","random"},{"shuffle","random"},{"sample","random"},{"seed","random"},
                /* @encoding */
                {"base64_encode","encoding"},{"base64_decode","encoding"},
                {"hex_encode","encoding"},{"hex_decode","encoding"},
                {"url_encode","encoding"},{"url_decode","encoding"},
                /* @crypto */
                {"sha256","crypto"},{"md5","crypto"},{"random_hex","crypto"},
                /* @regex */
                {"is_valid","regex"},{"is_match","regex"},{"find","regex"},
                {"find_all","regex"},{"replace","regex"},{"split","regex"},
                /* @json */
                {"parse","json"},{"stringify","json"},{"encode","json"},
                {"decode","json"},{"is_valid","json"},{"pretty_print","json"},
                /* @io */
                {"read_file","io"},{"write_file","io"},{"append_file","io"},
                {"delete_file","io"},{"rename_file","io"},{"file_exists","io"},
                {"is_file","io"},{"is_directory","io"},{"file_size","io"},{"glob","io"},
                /* @os */
                {"args","os"},{"get_env","os"},{"set_env","os"},{"current_dir","os"},
                {"hostname","os"},{"arch","os"},{"current_os","os"},{"pid","os"},
                {"exec","os"},{"exit","os"},
                /* @time */
                {"now","time"},{"now_ms","time"},{"now_ns","time"},{"tick","time"},
                {"elapsed_ms","time"},{"year","time"},{"month","time"},{"day","time"},
                {"hour","time"},{"minute","time"},{"second","time"},{"weekday","time"},
                {"format","time"},{"to_iso","time"},{"date","time"},{"to_time","time"},
                /* @uuid */
                {"generate_hyphenated","uuid"},{"generate","uuid"},{"is_valid","uuid"},
                /* @bytes */
                {"from_string","bytes"},{"from_hex","bytes"},{"from_base64","bytes"},
                {"to_string","bytes"},{"to_hex","bytes"},{"to_base64","bytes"},
                /* @binary */
                {"encode_i8","binary"},{"encode_u8","binary"},
                {"encode_i16_le","binary"},{"encode_i16_be","binary"},
                {"encode_u16_le","binary"},{"encode_u16_be","binary"},
                {"encode_i32_le","binary"},{"encode_i32_be","binary"},
                {"encode_u32_le","binary"},{"encode_u32_be","binary"},
                {"encode_i64_le","binary"},{"encode_i64_be","binary"},
                {"encode_u64_le","binary"},{"encode_u64_be","binary"},
                {"encode_i128_le","binary"},{"encode_i128_be","binary"},
                {"encode_u128_le","binary"},{"encode_u128_be","binary"},
                {"encode_i256_le","binary"},{"encode_i256_be","binary"},
                {"encode_u256_le","binary"},{"encode_u256_be","binary"},
                {"encode_f32_le","binary"},{"encode_f32_be","binary"},
                {"encode_f64_le","binary"},{"encode_f64_be","binary"},
                {"decode_i8","binary"},{"decode_u8","binary"},
                {"decode_i16_le","binary"},{"decode_i16_be","binary"},
                {"decode_u16_le","binary"},{"decode_u16_be","binary"},
                {"decode_i32_le","binary"},{"decode_i32_be","binary"},
                {"decode_u32_le","binary"},{"decode_u32_be","binary"},
                {"decode_i64_le","binary"},{"decode_i64_be","binary"},
                {"decode_u64_le","binary"},{"decode_u64_be","binary"},
                {"decode_i128_le","binary"},{"decode_i128_be","binary"},
                {"decode_u128_le","binary"},{"decode_u128_be","binary"},
                {"decode_i256_le","binary"},{"decode_i256_be","binary"},
                {"decode_u256_le","binary"},{"decode_u256_be","binary"},
                {"decode_f32_le","binary"},{"decode_f32_be","binary"},
                {"decode_f64_le","binary"},{"decode_f64_be","binary"},
                /* @csv */
                {"parse","csv"},{"read_file","csv"},{"headers","csv"},
                {"write_file","csv"},{"format","csv"},{"encode","csv"},
                /* @sqlite */
                {"open","sqlite"},{"close","sqlite"},{"exec","sqlite"},{"query","sqlite"},
                /* @threads */
                {"spawn","threads"},{"join","threads"},{"get_id","threads"},
                /* @sync */
                {"mutex","sync"},{"lock","sync"},{"unlock","sync"},
                {"try_lock","sync"},{"destroy","sync"},
                /* @atomic */
                {"load","atomic"},{"store","atomic"},{"add","atomic"},{"sub","atomic"},
                {"exchange","atomic"},{"cas","atomic"},{"and","atomic"},{"or","atomic"},
                {"xor","atomic"},{"spinlock","atomic"},{"spin_lock","atomic"},
                {"spin_trylock","atomic"},{"spin_unlock","atomic"},{"fence","atomic"},
                /* @channels */
                {"open","channels"},{"send","channels"},{"receive","channels"},{"close","channels"},
                /* @server */
                {"add_router","server"},{"add_route","server"},{"listen","server"},
                {"cors","server"},{"use","server"},{"text","server"},{"json","server"},
                {"html","server"},{"redirect","server"},{"parse_json","server"},
                /* @http */
                {"get","http"},{"post","http"},{"put","http"},{"delete","http"},
                {"head","http"},{"patch","http"},{"request","http"},{"json_body","http"},
                /* @net */
                {"listen","net"},{"connect","net"},{"accept","net"},{"send","net"},
                {"receive","net"},{"resolve","net"},{"close","net"},
                /* @fmt */
                {"sprintf","fmt"},{"format","fmt"},{"printf","fmt"},
                {"pad_left","fmt"},{"pad_right","fmt"},{"center","fmt"},
                {"int_to_hex","fmt"},{"int_to_binary","fmt"},{"int_to_octal","fmt"},
                {"float_fixed","fmt"},{"float_sci","fmt"},
                /* @mem */
                {"arena","mem"},{"usage","mem"},{"make","mem"},{"alloc","mem"},
                {"init","mem"},{"free","mem"},{"reset","mem"},{"destroy","mem"},
                {NULL,NULL}
            };
            for (int ui = 0; ui < cg->using_module_count; ui++) {
                const char *umod = cg->using_modules[ui];
                const char *real_mod = resolve_alias(cg, umod);
                /* 1) Check stdlib table */
                bool found = false;
                for (int fi = 0; func_to_mod[fi].func; fi++) {
                    if (strcmp(func, func_to_mod[fi].func) == 0 &&
                        strcmp(real_mod, func_to_mod[fi].mod) == 0) {
                        found = true;
                        break;
                    }
                }
                if (found) {
                    /* Dispatch to the stdlib module handler */
                    for (int mi = 0; mi < (int)(sizeof(modules) / sizeof(modules[0])); mi++) {
                        if (strcmp(real_mod, modules[mi].name) == 0) {
                            if (modules[mi].handler(cg, node, func)) return;
                            break;
                        }
                    }
                }
                /* 2) Try user-defined module: <module>_<func> */
                if (!found) {
                    size_t pf_len = strlen(real_mod) + 1 + strlen(func) + 1;
                    char *prefixed = malloc(pf_len);
                    snprintf(prefixed, pf_len, "%s_%s", real_mod, func);
                    AstNode *uf = find_func(cg, prefixed);
                    if (uf) {
                        emitf(cg, "ez_fn_%s_%s(", real_mod, func);
                        for (int i = 0; i < node->data.call.arg_count; i++) {
                            if (i > 0) emit(cg, ", ");
                            emit_expression(cg, node->data.call.args[i]);
                        }
                        emit(cg, ")");
                        return;
                    }
                }
            }
        }
    }

    /* Check for struct-namespaced or user-module function call: Name.func() */
    if (node->data.call.function->kind == NODE_MEMBER_EXPR) {
        AstNode *obj = node->data.call.function->data.member.object;
        const char *member = node->data.call.function->data.member.member;
        /* Handle mod.Struct.func() triple chain: geometry.Vec2.create() */
        if (obj->kind == NODE_MEMBER_EXPR &&
            obj->data.member.object->kind == NODE_LABEL) {
            const char *mod = obj->data.member.object->data.label.value;
            const char *type_name = obj->data.member.member;
            /* Build full prefixed name: mod_Struct_func */
            char full_name[256];
            snprintf(full_name, sizeof(full_name), "%s_%s_%s", mod, type_name, member);
            AstNode *ns_func = find_func(cg, full_name);
            if (ns_func) {
                emitf(cg, "ez_fn_%s(", full_name);
                for (int i = 0; i < node->data.call.arg_count; i++) {
                    if (i > 0) emit(cg, ", ");
                    bool mut_param = i < ns_func->data.func_decl.param_count &&
                        ns_func->data.func_decl.params[i].mutable;
                    if (mut_param && node->data.call.args[i]->kind == NODE_LABEL) {
                        const char *vn = node->data.call.args[i]->data.label.value;
                        if (is_mutable_param(cg, vn)) emit(cg, vn);
                        else emitf(cg, "&%s", vn);
                    } else if (mut_param && node->data.call.args[i]->kind == NODE_MEMBER_EXPR) {
                        emit(cg, "&"); emit_expression(cg, node->data.call.args[i]);
                    } else {
                        emit_expression(cg, node->data.call.args[i]);
                    }
                }
                emit(cg, ")");
                return;
            }
        }
        if (obj->kind == NODE_LABEL) {
            const char *raw_name = obj->data.label.value;

            /* C interop: c.func() — emit raw C function call */
            if (strcmp(raw_name, "c") == 0 && cg->has_c_imports) {
                emitf(cg, "%s(", member);
                for (int i = 0; i < node->data.call.arg_count; i++) {
                    if (i > 0) emit(cg, ", ");
                    /* Auto-convert EzString to char* for C functions */
                    EzType *arg_t = cg->type_table
                        ? typetable_get(cg->type_table, node->data.call.args[i])
                        : NULL;
                    if (arg_t && arg_t->kind == TK_STRING) {
                        emit_expression(cg, node->data.call.args[i]);
                        emit(cg, ".data");
                    } else {
                        emit_expression(cg, node->data.call.args[i]);
                    }
                }
                emit(cg, ")");
                return;
            }

            const char *resolved_name = resolve_alias(cg, raw_name);
            /* Try to find as a namespaced function: Name_func or ResolvedAlias_func */
            size_t ns_len = strlen(resolved_name) + 1 + strlen(member) + 1;
            char *ns_name = malloc(ns_len);
            snprintf(ns_name, ns_len, "%s_%s", resolved_name, member);
            AstNode *ns_func = find_func(cg, ns_name);
            if (!ns_func) {
                /* #1505: check if `member` is a func-typed data field
                 * on the struct. If so, emit as a function-pointer call
                 * through the field access. We get here when the variable
                 * has a struct type but neither <struct>_<member> nor
                 * bare <member> is a registered function. */
                EzType *inst_t = cg->type_table ? typetable_get(cg->type_table, obj) : NULL;
                /* Fall back to scanning struct decls if the type_table
                 * doesn't have a hit for the label. */
                if (!inst_t || inst_t->kind == TK_UNKNOWN) {
                    for (int si = 0; si < cg->struct_decl_count; si++) {
                        const char *sn = cg->struct_decls[si]->data.struct_decl.name;
                        for (int fi = 0; fi < cg->struct_decls[si]->data.struct_decl.field_count; fi++) {
                            StructField *sf = &cg->struct_decls[si]->data.struct_decl.fields[fi];
                            if (strcmp(sf->name, member) == 0 && sf->type_name &&
                                strcmp(sf->type_name, "func") == 0) {
                                inst_t = type_struct(sn);
                                break;
                            }
                        }
                        if (inst_t && inst_t->kind == TK_STRUCT) break;
                    }
                }
                if (inst_t && (inst_t->kind == TK_STRUCT || inst_t->kind == TK_POINTER)) {
                    const char *sn = (inst_t->kind == TK_POINTER && inst_t->element_type)
                        ? inst_t->element_type : inst_t->name;
                    AstNode *sdecl = sn ? find_struct_decl(cg, sn) : NULL;
                    if (sdecl) {
                        for (int fi = 0; fi < sdecl->data.struct_decl.field_count; fi++) {
                            if (strcmp(sdecl->data.struct_decl.fields[fi].name, member) == 0 &&
                                sdecl->data.struct_decl.fields[fi].type_name &&
                                strcmp(sdecl->data.struct_decl.fields[fi].type_name, "func") == 0) {
                                int nargs = node->data.call.arg_count;
                                if (nargs > 0 && !node->data.call.args) nargs = 0;
                                EzType *ret_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
                                const char *c_ret = (ret_t && ret_t->kind != TK_UNKNOWN && ret_t->kind != TK_VOID)
                                    ? ez_type_to_c_cg(cg, type_name(ret_t)) : "int64_t";
                                if (ret_t && ret_t->kind == TK_VOID) c_ret = "void";
                                emitf(cg, "((%s (*)(", c_ret);
                                for (int ai = 0; ai < nargs; ai++) {
                                    if (ai > 0) emit(cg, ", ");
                                    EzType *at = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[ai]) : NULL;
                                    emit(cg, at ? ez_type_to_c_cg(cg, type_name(at)) : "int64_t");
                                }
                                emit(cg, "))");
                                emit_expression(cg, obj);
                                emitf(cg, ".%s)(", member);
                                for (int ai = 0; ai < nargs; ai++) {
                                    if (ai > 0) emit(cg, ", ");
                                    emit_expression(cg, node->data.call.args[ai]);
                                }
                                emit(cg, ")");
                                return;
                            }
                        }
                    }
                }
                /* Try bare function name (for main-file functions in circular imports) */
                ns_func = find_func(cg, member);
                if (ns_func) {
                    emitf(cg, "ez_fn_%s(", member);
                    for (int i = 0; i < node->data.call.arg_count; i++) {
                        if (i > 0) emit(cg, ", ");
                        bool mut_param = i < ns_func->data.func_decl.param_count &&
                            ns_func->data.func_decl.params[i].mutable;
                        if (mut_param && node->data.call.args[i]->kind == NODE_LABEL) {
                            const char *vn = node->data.call.args[i]->data.label.value;
                            if (is_mutable_param(cg, vn)) emit(cg, vn);
                            else emitf(cg, "&%s", vn);
                        } else if (mut_param && node->data.call.args[i]->kind == NODE_MEMBER_EXPR) {
                            emit(cg, "&"); emit_expression(cg, node->data.call.args[i]);
                        } else {
                            emit_expression(cg, node->data.call.args[i]);
                        }
                    }
                    emit(cg, ")");
                    return;
                }
            }
            if (ns_func) {
                emitf(cg, "ez_fn_%s_%s(", resolved_name, member);
                for (int i = 0; i < node->data.call.arg_count; i++) {
                    if (i > 0) emit(cg, ", ");
                    bool mut_param = i < ns_func->data.func_decl.param_count &&
                        ns_func->data.func_decl.params[i].mutable;
                    if (mut_param && node->data.call.args[i]->kind == NODE_LABEL) {
                        const char *vn = node->data.call.args[i]->data.label.value;
                        if (is_mutable_param(cg, vn)) emit(cg, vn);
                        else emitf(cg, "&%s", vn);
                    } else if (mut_param && node->data.call.args[i]->kind == NODE_MEMBER_EXPR) {
                        emit(cg, "&"); emit_expression(cg, node->data.call.args[i]);
                    } else {
                        emit_expression(cg, node->data.call.args[i]);
                    }
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

    /* If not found, try module-prefixed names (for internal cross-references in imported files) */
    const char *resolved_fn_name = fn_name;
    if (!target_func && fn_name) {
        for (int fi = 0; fi < cg->func_count; fi++) {
            const char *registered = cg->all_funcs[fi]->data.func_decl.name;
            const char *us = strchr(registered, '_');
            if (us && strcmp(us + 1, fn_name) == 0) {
                target_func = cg->all_funcs[fi];
                resolved_fn_name = registered;
                break;
            }
        }
    }

    if (fn_name && target_func) {
        /* Known function — use ez_fn_ prefix. For generic functions,
         * rewrite to the mangled instantiation name derived from the
         * first wildcard parameter's argument type (#1443). */
        bool tf_generic = false;
        for (int pi = 0; pi < target_func->data.func_decl.param_count && !tf_generic; pi++) {
            if (target_func->data.func_decl.params[pi].type_name &&
                strchr(target_func->data.func_decl.params[pi].type_name, '?')) tf_generic = true;
        }
        for (int pi = 0; pi < target_func->data.func_decl.return_type_count && !tf_generic; pi++) {
            if (target_func->data.func_decl.return_types[pi] &&
                strchr(target_func->data.func_decl.return_types[pi], '?')) tf_generic = true;
        }
        if (tf_generic) {
            /* Derive the concrete binding by scanning params for the
             * first '?' slot and reading the matching arg's type. */
            const char *binding = NULL;
            int pc = target_func->data.func_decl.param_count;
            int ac = node->data.call.arg_count;
            int cc = pc < ac ? pc : ac;
            for (int pi = 0; pi < cc && !binding; pi++) {
                const char *ptn = target_func->data.func_decl.params[pi].type_name;
                if (!ptn || !strchr(ptn, '?')) continue;
                EzType *at = cg->type_table
                    ? typetable_get(cg->type_table, node->data.call.args[pi]) : NULL;
                if (!at) continue;
                if (strcmp(ptn, "?") == 0) {
                    /* #1506: inside a generic body, the arg's typetable
                     * entry is still TK_UNKNOWN ("unknown") from the
                     * main-pass walk. If the outer wildcard_binding is
                     * active, use it as the binding — that's the
                     * concrete type this instantiation was stamped with. */
                    const char *tn = type_name(at);
                    if ((at->kind == TK_UNKNOWN || strcmp(tn, "unknown") == 0) &&
                        cg->wildcard_binding) {
                        binding = cg->wildcard_binding;
                    } else {
                        binding = tn;
                    }
                } else if (ptn[0] == '[' && strncmp(ptn + 1, "?]", 2) == 0 &&
                           at->kind == TK_ARRAY && at->element_type) {
                    binding = at->element_type;
                } else if (strncmp(ptn, "map[", 4) == 0 && at->kind == TK_MAP &&
                           at->key_type && at->value_type) {
                    /* map[K:V] with '?' in the key, value, or both slots
                     * (#1463). Mirror bind_wildcard() in the typechecker:
                     * the concrete side (if any) must match the arg's
                     * corresponding slot; the wildcard side binds to the
                     * arg's slot. */
                    const char *start = ptn + 4;
                    const char *colon = strchr(start, ':');
                    if (!colon) continue;
                    size_t klen = (size_t)(colon - start);
                    const char *vstart = colon + 1;
                    size_t vlen = strlen(vstart);
                    if (vlen == 0 || vstart[vlen - 1] != ']') continue;
                    vlen--;
                    bool k_wild = (klen == 1 && start[0] == '?');
                    bool v_wild = (vlen == 1 && vstart[0] == '?');
                    if (!k_wild && (strlen(at->key_type) != klen ||
                                    memcmp(at->key_type, start, klen) != 0)) continue;
                    if (!v_wild && (strlen(at->value_type) != vlen ||
                                    memcmp(at->value_type, vstart, vlen) != 0)) continue;
                    if (k_wild && v_wild) {
                        if (strcmp(at->key_type, at->value_type) == 0) {
                            binding = at->key_type;
                        }
                    } else if (k_wild) {
                        binding = at->key_type;
                    } else {
                        binding = at->value_type;
                    }
                }
            }
            size_t mn_need = strlen("ez_fn_") + strlen(resolved_fn_name) + 2
                + (binding ? strlen(binding) : 0) + 1;
            char *mangled = malloc(mn_need);
            size_t pos = (size_t)snprintf(mangled, mn_need, "ez_fn_%s__",
                resolved_fn_name);
            if (binding) {
                for (const char *c = binding; *c && pos < mn_need - 1; c++) {
                    mangled[pos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
                }
            }
            mangled[pos] = '\0';
            emit(cg, mangled);
        } else {
            emit(cg, "ez_fn_");
            emit(cg, resolved_fn_name);
        }
    } else if (fn_name) {
        /* Not a known function — variable holding a function pointer (void *).
         * Cast to appropriate function pointer type based on arg types. */
        int nargs = node->data.call.arg_count;
        /* Try to find the referenced function declaration for mutable param info.
         * Scan var_decls for this variable, check if its init is NODE_FUNC_REF,
         * and look up the referenced function. */
        AstNode *ref_func = NULL;
        for (int si = 0; si < cg->func_count && !ref_func; si++) {
            AstNode *fn_decl = cg->all_funcs[si];
            if (!fn_decl->data.func_decl.body) continue;
            for (int bi = 0; bi < fn_decl->data.func_decl.body->data.block.count && !ref_func; bi++) {
                AstNode *st = fn_decl->data.func_decl.body->data.block.stmts[bi];
                if (st->kind == NODE_VAR_DECL &&
                    strcmp(st->data.var_decl.name, fn_name) == 0 &&
                    st->data.var_decl.value &&
                    st->data.var_decl.value->kind == NODE_FUNC_REF) {
                    AstNode *fref = st->data.var_decl.value->data.func_ref.function;
                    if (fref->kind == NODE_LABEL) {
                        ref_func = find_func(cg, fref->data.label.value);
                    } else if (fref->kind == NODE_MEMBER_EXPR &&
                               fref->data.member.object &&
                               fref->data.member.object->kind == NODE_LABEL) {
                        const char *rn_a = fref->data.member.object->data.label.value;
                        const char *rn_b = fref->data.member.member;
                        size_t rn_len = strlen(rn_a) + 1 + strlen(rn_b) + 1;
                        char *rn = malloc(rn_len);
                        snprintf(rn, rn_len, "%s_%s", rn_a, rn_b);
                        ref_func = find_func(cg, rn);
                    }
                }
            }
        }
        if (ref_func) target_func = ref_func;
        /* Determine return type from the call expression's type table entry */
        EzType *ret_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
        const char *c_ret = (ret_t && ret_t->kind != TK_UNKNOWN) ? ez_type_to_c_cg(cg, type_name(ret_t)) : "int64_t";
        if (ret_t && ret_t->kind == TK_VOID) c_ret = "void";
        /* #1503: the cast must include param types for ALL parameters
         * (including default-valued ones), not just the provided args.
         * When defaults fill the gap, nargs < param_count and the
         * default values are emitted as explicit args below. The cast
         * must match the FULL arity. */
        int cast_count = nargs;
        if (ref_func && ref_func->data.func_decl.param_count > nargs) {
            cast_count = ref_func->data.func_decl.param_count;
        }
        emitf(cg, "((%s (*)(", c_ret);
        for (int i = 0; i < cast_count; i++) {
            if (i > 0) emit(cg, ", ");
            bool mut_p = ref_func && i < ref_func->data.func_decl.param_count &&
                ref_func->data.func_decl.params[i].mutable;
            if (i < nargs) {
                EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[i]) : NULL;
                if (arg_t && arg_t->kind != TK_UNKNOWN) {
                    emit(cg, ez_type_to_c_cg(cg, type_name(arg_t)));
                    if (mut_p) emit(cg, " *");
                } else {
                    emit(cg, "int64_t");
                    if (mut_p) emit(cg, " *");
                }
            } else if (ref_func && i < ref_func->data.func_decl.param_count) {
                /* Default-value position — use the declared param type */
                const char *ptn = ref_func->data.func_decl.params[i].type_name;
                emit(cg, ptn ? ez_type_to_c_cg(cg, ptn) : "int64_t");
                if (mut_p) emit(cg, " *");
            } else {
                emit(cg, "int64_t");
            }
        }
        if (cast_count == 0) emit(cg, "void");
        emit(cg, "))");
        emit(cg, safe_name(fn_name));
        emit(cg, ")");
    } else if (node->data.call.function->kind == NODE_MEMBER_EXPR) {
        /* Module-qualified call fallback: module.func() → ez_module_func() */
        AstNode *obj = node->data.call.function->data.member.object;
        const char *member = node->data.call.function->data.member.member;
        if (obj->kind == NODE_LABEL) {
            const char *mod_name = resolve_alias(cg, obj->data.label.value);
            emitf(cg, "ez_%s_%s", mod_name, member);
        } else {
            emit_expression(cg, node->data.call.function);
        }
    } else if (node->data.call.function->kind == NODE_INDEX_EXPR) {
        /* Indexed callee (e.g. ops[0](x, y)). The index expression yields a
         * void * for [func] arrays (see NODE_INDEX_EXPR emitter), which isn't
         * directly callable — wrap with a function-pointer cast derived from
         * the call site's arg types and return type. */
        int nargs = node->data.call.arg_count;
        EzType *ret_t = cg->type_table ? typetable_get(cg->type_table, node) : NULL;
        const char *c_ret = (ret_t && ret_t->kind != TK_UNKNOWN) ? ez_type_to_c_cg(cg, type_name(ret_t)) : "int64_t";
        if (ret_t && ret_t->kind == TK_VOID) c_ret = "void";
        emitf(cg, "((%s (*)(", c_ret);
        for (int i = 0; i < nargs; i++) {
            if (i > 0) emit(cg, ", ");
            EzType *arg_t = cg->type_table ? typetable_get(cg->type_table, node->data.call.args[i]) : NULL;
            if (arg_t && arg_t->kind != TK_UNKNOWN) {
                emit(cg, ez_type_to_c_cg(cg, type_name(arg_t)));
            } else {
                emit(cg, "int64_t");
            }
        }
        if (nargs == 0) emit(cg, "void");
        emit(cg, "))");
        emit_expression(cg, node->data.call.function);
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
                /* Mutable param on indexed element: array or map */
                AstNode *idx_node = node->data.call.args[i];
                EzType *left_t = cg->type_table
                    ? typetable_get(cg->type_table, idx_node->data.index_expr.left) : NULL;
                if (left_t && left_t->kind == TK_MAP) {
                    /* Map: get pointer to value via ez_map_get */
                    const char *c_key = "EzString";
                    if (left_t->key_type) c_key = ez_map_elem_c_type(cg, left_t->key_type);
                    emitf(cg, "({ %s _mk = ", c_key);
                    emit_expression(cg, idx_node->data.index_expr.index);
                    emit(cg, "; void *_mv = ez_map_get(&");
                    emit_expression(cg, idx_node->data.index_expr.left);
                    emitf(cg, ", &_mk); if (!_mv) { fflush(stdout); ez_panic(__FILE__, %d, \"key not found in map\"); } ",
                        idx_node->token.line);
                    emit(cg, "(int64_t *)_mv; })");
                } else {
                    /* Array: pass pointer to element */
                    emit(cg, "(int64_t *)ez_array_get_ptr(&");
                    emit_expression(cg, idx_node->data.index_expr.left);
                    emit(cg, ", ");
                    emit_expression(cg, idx_node->data.index_expr.index);
                    emit(cg, ", __FILE__, __LINE__)");
                }
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
        /* [func] with a single func ref init: emit as void* (function pointer),
         * not EzArray. Array-literal inits still use EzArray. */
        if (strcmp(elem_type, "func") == 0 &&
            node->data.var_decl.value &&
            node->data.var_decl.value->kind == NODE_FUNC_REF) {
            emitf(cg, "void *%s = ", safe_name(node->data.var_decl.name));
            emit_expression(cg, node->data.var_decl.value);
            emit(cg, ";\n");
            return;
        }
        int fixed_size = extract_array_size(type_name);
        if (fixed_size > 0) {
            /* Fixed-size array: use EzArray but initialized with exact capacity */
            if (cg->indent == 0) {
                /* File scope: emit uninitialized global, defer init to ez_init_globals */
                emitf(cg, "EzArray %s;\n", safe_name(node->data.var_decl.name));
                /* Store deferred init in the init buffer */
                if (node->data.var_decl.value) {
                    buf_appendf(&cg->global_init, "    %s = ", safe_name(node->data.var_decl.name));
                    /* Temporarily redirect output to global_init buffer */
                    Buf saved = cg->output;
                    cg->output = cg->global_init;
                    cg->indent = 1;
                    emit_expression(cg, node->data.var_decl.value);
                    emit(cg, ";\n");
                    cg->global_init = cg->output;
                    cg->output = saved;
                    cg->indent = 0;
                }
            } else {
                emitf(cg, "EzArray %s = ", safe_name(node->data.var_decl.name));
                if (node->data.var_decl.value) {
                    emit_expression(cg, node->data.var_decl.value);
                } else {
                    const char *c_elem_type = ez_type_to_c_cg(cg, elem_type);
                    emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), %d)", c_elem_type, fixed_size);
                }
                emit(cg, ";\n");
            }
            return;
        }

        if (is_nested_array_type(type_name)) {
            cg->current_var_type = type_name;
            AstNode *init = node->data.var_decl.value;
            bool label_init = init && init->kind == NODE_LABEL;
            const char *label_elem_tn = NULL;
            if (label_init) {
                EzType *src_t = cg->type_table
                    ? typetable_get(cg->type_table, init) : NULL;
                if (src_t && src_t->kind == TK_ARRAY) {
                    label_elem_tn = src_t->element_type;
                }
            }
            if (cg->indent == 0) {
                emitf(cg, "EzArray %s;\n", safe_name(node->data.var_decl.name));
                Buf saved = cg->output; cg->output = cg->global_init; cg->indent = 1;
                emitf(cg, "    %s = ", safe_name(node->data.var_decl.name));
                if (label_init) emit_deep_array_copy(cg, init, label_elem_tn);
                else if (init) emit_expression(cg, init);
                else emit(cg, "ez_array_new(ez_default_arena, sizeof(EzArray), 4)");
                emit(cg, ";\n");
                cg->global_init = cg->output; cg->output = saved; cg->indent = 0;
            } else {
                emitf(cg, "EzArray %s = ", safe_name(node->data.var_decl.name));
                if (label_init) emit_deep_array_copy(cg, init, label_elem_tn);
                else if (init) emit_expression(cg, init);
                else emitf(cg, "ez_array_new(ez_default_arena, sizeof(EzArray), 4)");
                emit(cg, ";\n");
            }
            return;
        }

        /* Dynamic array: use EzArray */
        const char *c_elem_type = ez_type_to_c_cg(cg, elem_type);
        if (cg->indent == 0) {
            emitf(cg, "EzArray %s;\n", safe_name(node->data.var_decl.name));
            Buf saved = cg->output; cg->output = cg->global_init; cg->indent = 1;
            emitf(cg, "    %s = ", safe_name(node->data.var_decl.name));
            if (node->data.var_decl.value) emit_expression(cg, node->data.var_decl.value);
            else emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", c_elem_type);
            emit(cg, ";\n");
            cg->global_init = cg->output; cg->output = saved; cg->indent = 0;
        } else {
        emitf(cg, "EzArray %s = ", safe_name(node->data.var_decl.name));
        if (node->data.var_decl.value &&
            node->data.var_decl.value->kind == NODE_ARRAY_VALUE &&
            node->data.var_decl.value->data.array_value.count == 0) {
            /* Empty array literal with type annotation — use correct elem size */
            emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", c_elem_type);
        } else if (node->data.var_decl.value &&
                   node->data.var_decl.value->kind == NODE_LABEL) {
            /* Copy-by-default: deep copy when assigning from another
             * variable. Route through emit_deep_array_copy so nested
             * array elements get independent backing storage (#1465). */
            EzType *src_t = cg->type_table
                ? typetable_get(cg->type_table, node->data.var_decl.value) : NULL;
            const char *elem_tn = (src_t && src_t->kind == TK_ARRAY)
                ? src_t->element_type : NULL;
            emit_deep_array_copy(cg, node->data.var_decl.value, elem_tn);
        } else if (node->data.var_decl.value) {
            emit_expression(cg, node->data.var_decl.value);
        } else {
            emitf(cg, "ez_array_new(ez_default_arena, sizeof(%s), 4)", c_elem_type);
        }
        emit(cg, ";\n");
        } /* end else (indent > 0) */
        return;
    }

    /* Map type: map[K:V] */
    if (type_name && strncmp(type_name, "map[", 4) == 0) {
        /* Parse K:V from type string to determine C types */
        EzType *mt = type_from_name(type_name);
        const char *c_kt = "EzString";
        const char *c_vt = "int64_t";
        if (mt && mt->key_type) c_kt = ez_map_elem_c_type(cg, mt->key_type);
        if (mt && mt->value_type) c_vt = ez_map_elem_c_type(cg, mt->value_type);

        emitf(cg, "EzMap %s = ", safe_name(node->data.var_decl.name));
        if (node->data.var_decl.value) {
            const char *saved_var_type = cg->current_var_type;
            cg->current_var_type = type_name;
            emit_expression(cg, node->data.var_decl.value);
            cg->current_var_type = saved_var_type;
        } else {
            /* No initializer — create empty map */
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
            /* #1520: use mangled name for generic struct instantiations */
            if (val->data.struct_value.wildcard_binding) {
                const char *binding = val->data.struct_value.wildcard_binding;
                const char *base = val->data.struct_value.name;
                static char sv_buf[256];
                size_t sp = snprintf(sv_buf, sizeof(sv_buf), "%s__", base);
                for (const char *c = binding; *c && sp < sizeof(sv_buf) - 1; c++)
                    sv_buf[sp++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
                sv_buf[sp] = '\0';
                c_type = ez_type_to_c_cg(cg, sv_buf);
            } else {
                c_type = ez_type_to_c_cg(cg, val->data.struct_value.name);
            }
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
        } else if (val->kind == NODE_LABEL) {
            /* Variable reference — use __auto_type to propagate the source type */
            c_type = "__auto_type";
        } else if (val->kind == NODE_CAST_EXPR) {
            /* Cast expression — use the target type */
            c_type = ez_type_to_c_cg(cg, val->data.cast.target_type);
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

    emitf(cg, "%s %s", c_type, safe_name(node->data.var_decl.name));

    if (node->data.var_decl.value) {
        emit(cg, " = ");
        cg->current_var_name = node->data.var_decl.name;
        cg->current_var_type = node->data.var_decl.type_name;
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
            /* Integer literal for bigint type */
            const char *pfx = bigint_prefix(type_name);
            if (node->data.var_decl.value->data.int_value.overflow) {
                /* Overflowed literal: parse from decimal string at runtime */
                emitf(cg, "%s_from_decimal(\"%s\")", pfx,
                    node->data.var_decl.value->data.int_value.literal);
            } else {
                /* Fits in 64-bit: use direct constructor */
                int64_t v = node->data.var_decl.value->data.int_value.value;
                const char *from_suffix = (strcmp(type_name, "u128") == 0 || strcmp(type_name, "u256") == 0) ? "u64" : "i64";
                emitf(cg, "%s_from_%s(%lldLL)", pfx, from_suffix, (long long)v);
            }
        } else if (type_name && is_bigint_type(type_name) &&
                   node->data.var_decl.value->kind == NODE_PREFIX_EXPR &&
                   strcmp(node->data.var_decl.value->data.prefix.op, "-") == 0 &&
                   node->data.var_decl.value->data.prefix.right->kind == NODE_INT_VALUE) {
            /* Negated integer literal for bigint: -N → from_i64(-N) or from_decimal("-N") */
            const char *pfx = bigint_prefix(type_name);
            AstNode *inner = node->data.var_decl.value->data.prefix.right;
            if (inner->data.int_value.overflow) {
                emitf(cg, "%s_from_decimal(\"-%s\")", pfx, inner->data.int_value.literal);
            } else {
                emitf(cg, "%s_from_i64(%lldLL)", pfx,
                    -(long long)inner->data.int_value.value);
            }
        } else if (type_name && is_bigint_type(type_name) &&
                   node->data.var_decl.value->kind == NODE_INFIX_EXPR) {
            /* Infix expression assigned to bigint — use bigint operations.
             * Emit the infix manually with bigint operand wrapping since
             * resolve_bigint_type won't detect raw literals as bigint. */
            AstNode *infix = node->data.var_decl.value;
            const char *pfx = bigint_prefix(type_name);
            const char *op = infix->data.infix.op;
            const char *fn_op = NULL;
            if (strcmp(op, "+") == 0) fn_op = "add";
            else if (strcmp(op, "-") == 0) fn_op = "sub";
            else if (strcmp(op, "*") == 0) fn_op = "mul";
            else if (strcmp(op, "/") == 0) fn_op = "div";
            else if (strcmp(op, "%") == 0) fn_op = "mod";
            if (fn_op) {
                bool is_checked = (strcmp(fn_op, "add") == 0 || strcmp(fn_op, "sub") == 0 || strcmp(fn_op, "mul") == 0);
                if (is_checked)
                    emitf(cg, "%s_%s_checked(", pfx, fn_op);
                else
                    emitf(cg, "%s_%s(", pfx, fn_op);
                EMIT_BIGINT_OPERAND(cg, infix->data.infix.left, pfx, type_name);
                emit(cg, ", ");
                EMIT_BIGINT_OPERAND(cg, infix->data.infix.right, pfx, type_name);
                if (is_checked)
                    emitf(cg, ", __FILE__, %d)", node->token.line);
                else
                    emit(cg, ")");
            } else {
                emit_expression(cg, node->data.var_decl.value);
            }
        } else if (type_name && type_name[0] == '^' &&
                   node->data.var_decl.value->kind == NODE_LABEL &&
                   is_ref_var(cg, node->data.var_decl.value->data.label.value)) {
            /* Assigning a ref variable to a ^T pointer — pass the pointer through
             * without auto-dereferencing */
            emit(cg, node->data.var_decl.value->data.label.value);
        } else if (node->data.var_decl.value->kind == NODE_CALL_EXPR &&
                   node->data.var_decl.value->data.call.function->kind == NODE_LABEL &&
                   strcmp(node->data.var_decl.value->data.call.function->data.label.value, "addr") == 0 &&
                   type_name && (strcmp(type_name, "uint") == 0 || strcmp(type_name, "int") == 0 ||
                                 strcmp(type_name, "u64") == 0 || strcmp(type_name, "i64") == 0)) {
            /* addr() assigned to integer type — cast pointer to uintptr_t */
            emit(cg, "(uintptr_t)");
            emit_expression(cg, node->data.var_decl.value);
        } else {
            emit_expression(cg, node->data.var_decl.value);
        }
        cg->current_var_name = NULL;
        cg->current_var_type = NULL;
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
                if (strcmp(left_t->element_type, "func") == 0) {
                    c_elem = "void *";
                } else {
                    c_elem = ez_type_to_c_cg(cg, left_t->element_type);
                }
            }
            /* Compound assignment on array element with sized-type overflow check */
            const char *aop = node->data.assign.op;
            bool is_compound = (strcmp(aop, "+=") == 0 || strcmp(aop, "-=") == 0 || strcmp(aop, "*=") == 0);
            if (is_compound && left_t->element_type) {
                const char *sn = left_t->element_type;
                const char *smin = NULL, *smax = NULL;
                bool su = false;
                if (strcmp(sn, "i8") == 0) { smin = "-128"; smax = "127"; }
                else if (strcmp(sn, "i16") == 0) { smin = "-32768"; smax = "32767"; }
                else if (strcmp(sn, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
                else if (strcmp(sn, "u8") == 0 || strcmp(sn, "byte") == 0) { su = true; smax = "255"; }
                else if (strcmp(sn, "u16") == 0) { su = true; smax = "65535"; }
                else if (strcmp(sn, "u32") == 0) { su = true; smax = "4294967295ULL"; }
                if (smax) {
                    const char *fn = NULL;
                    if (su) {
                        if (strcmp(aop, "+=") == 0) fn = "ez_usized_add_check";
                        else if (strcmp(aop, "-=") == 0) fn = "ez_usized_sub_check";
                        else if (strcmp(aop, "*=") == 0) fn = "ez_usized_mul_check";
                    } else {
                        if (strcmp(aop, "+=") == 0) fn = "ez_sized_add_check";
                        else if (strcmp(aop, "-=") == 0) fn = "ez_sized_sub_check";
                        else if (strcmp(aop, "*=") == 0) fn = "ez_sized_mul_check";
                    }
                    if (fn) {
                        /* EZ_ARRAY_SET/GET use int64_t (internal storage width) */
                        emitf(cg, "EZ_ARRAY_SET(");
                        emit_expression(cg, left);
                        emit(cg, ", int64_t, ");
                        emit_expression(cg, node->data.assign.target->data.index_expr.index);
                        emitf(cg, ", %s(EZ_ARRAY_GET(", fn);
                        emit_expression(cg, left);
                        emit(cg, ", int64_t, ");
                        emit_expression(cg, node->data.assign.target->data.index_expr.index);
                        emit(cg, "), ");
                        emit_expression(cg, node->data.assign.value);
                        if (su) {
                            emitf(cg, ", %s, \"%s\", __FILE__, %d));\n", smax, sn, node->token.line);
                        } else {
                            emitf(cg, ", %s, %s, \"%s\", __FILE__, %d));\n", smin, smax, sn, node->token.line);
                        }
                        return;
                    }
                }
            }
            emitf(cg, "EZ_ARRAY_SET(");
            emit_expression(cg, left);
            emitf(cg, ", %s, ", c_elem);
            emit_expression(cg, node->data.assign.target->data.index_expr.index);
            emit(cg, ", ");
            /* Non-sized compound assignment on array element: read-modify-write */
            if (is_compound) {
                const char *binop = "+";
                if (strcmp(aop, "-=") == 0) binop = "-";
                else if (strcmp(aop, "*=") == 0) binop = "*";
                emitf(cg, "EZ_ARRAY_GET(");
                emit_expression(cg, left);
                emitf(cg, ", %s, ", c_elem);
                emit_expression(cg, node->data.assign.target->data.index_expr.index);
                emitf(cg, ") %s (", binop);
                emit_expression(cg, node->data.assign.value);
                emit(cg, ")");
            } else {
                emit_expression(cg, node->data.assign.value);
            }
            emit(cg, ");\n");
            return;
        }
        if (left_t && left_t->kind == TK_MAP) {
            /* Map key assignment: ez_map_set(arena, &m, &key, &value) */
            const char *c_val = "int64_t";
            if (left_t->value_type) c_val = ez_map_elem_c_type(cg, left_t->value_type);
            const char *c_key = "EzString";
            if (left_t->key_type) c_key = ez_map_elem_c_type(cg, left_t->key_type);
            const char *ms_arena = cg->loop_scope_depth > 0 ? "_ez_outer_arena" : "ez_default_arena";
            bool ms_str_key = left_t->key_type && strcmp(left_t->key_type, "string") == 0;
            bool ms_str_val = left_t->value_type && strcmp(left_t->value_type, "string") == 0;
            emitf(cg, "{ %s _mk = ", c_key);
            emit_expression(cg, node->data.assign.target->data.index_expr.index);
            emit(cg, "; ");
            if (cg->loop_scope_depth > 0) {
                if (ms_str_key) {
                    emitf(cg, "_mk = ez_string_new(%s, _mk.data, _mk.len); ", ms_arena);
                } else if (left_t->key_type && type_needs_deep_copy(cg, left_t->key_type)) {
                    emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _mk = ", ms_arena);
                    emit_value_deep_copy(cg, left_t->key_type, "_mk");
                    emit(cg, "; ez_default_arena = _esc; } ");
                }
            }
            emitf(cg, "%s _mv = ", c_val);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; ");
            if (cg->loop_scope_depth > 0) {
                if (ms_str_val) {
                    emitf(cg, "_mv = ez_string_new(%s, _mv.data, _mv.len); ", ms_arena);
                } else if (left_t->value_type && type_needs_deep_copy(cg, left_t->value_type)) {
                    emitf(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = %s; _mv = ", ms_arena);
                    emit_value_deep_copy(cg, left_t->value_type, "_mv");
                    emit(cg, "; ez_default_arena = _esc; } ");
                }
            }
            emitf(cg, "ez_map_set(%s, &", ms_arena);
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
    /* Nested pointer field assignment: o.inner.val = value (where some ancestor is ptr<T>)
     * Walk the member chain to find the pointer root, then emit nil-check + chain. */
    if (node->data.assign.target->kind == NODE_MEMBER_EXPR) {
        const char *chain[32];
        int depth = 0;
        AstNode *cur = node->data.assign.target;
        AstNode *ptr_root = NULL;
        while (cur->kind == NODE_MEMBER_EXPR && depth < 32) {
            chain[depth++] = cur->data.member.member;
            AstNode *obj = cur->data.member.object;
            EzType *obj_t = cg->type_table ? typetable_get(cg->type_table, obj) : NULL;
            if (obj_t && obj_t->kind == TK_POINTER &&
                !(obj->kind == NODE_LABEL && is_ref_var(cg, obj->data.label.value))) {
                ptr_root = obj;
                break;
            }
            cur = obj;
        }
        if (ptr_root && depth > 1) {
            emit(cg, "{ __auto_type _dp = ");
            emit_expression(cg, ptr_root);
            emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
                "\"nil pointer dereference\"); } _dp->", node->token.line);
            /* Emit chain in reverse: chain[depth-1] is closest to root */
            for (int i = depth - 1; i >= 0; i--) {
                emit(cg, safe_name(chain[i]));
                if (i > 0) emit(cg, ".");
            }
            emitf(cg, " %s ", node->data.assign.op);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; }\n");
            return;
        }
    }
    /* Pointer field assignment: p.field = value (where p is ptr<T>) → nil check + p->field = value */
    if (node->data.assign.target->kind == NODE_MEMBER_EXPR) {
        AstNode *obj = node->data.assign.target->data.member.object;
        EzType *obj_t = cg->type_table ? typetable_get(cg->type_table, obj) : NULL;
        bool is_ref = (obj->kind == NODE_LABEL && is_ref_var(cg, obj->data.label.value));
        if (!is_ref && obj_t && obj_t->kind == TK_POINTER) {
            const char *field = node->data.assign.target->data.member.member;
            emit(cg, "{ __auto_type _dp = ");
            emit_expression(cg, obj);
            emitf(cg, "; if (!_dp) { fflush(stdout); ez_panic(__FILE__, %d, "
                "\"nil pointer dereference\"); } _dp->%s", node->token.line, safe_name(field));
            emitf(cg, " %s ", node->data.assign.op);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; }\n");
            return;
        }
    }

    /* Compound assignment with sized-type overflow check */
    {
        const char *aop = node->data.assign.op;
        bool is_compound = (strcmp(aop, "+=") == 0 || strcmp(aop, "-=") == 0 || strcmp(aop, "*=") == 0);
        if (is_compound) {
            EzType *tgt_t = cg->type_table ? typetable_get(cg->type_table, node->data.assign.target) : NULL;
            const char *sn = (tgt_t && tgt_t->name) ? tgt_t->name : NULL;
            const char *smin = NULL, *smax = NULL;
            bool su = false;
            if (sn) {
                if (strcmp(sn, "i8") == 0) { smin = "-128"; smax = "127"; }
                else if (strcmp(sn, "i16") == 0) { smin = "-32768"; smax = "32767"; }
                else if (strcmp(sn, "i32") == 0) { smin = "-2147483648LL"; smax = "2147483647LL"; }
                else if (strcmp(sn, "u8") == 0 || strcmp(sn, "byte") == 0) { su = true; smax = "255"; }
                else if (strcmp(sn, "u16") == 0) { su = true; smax = "65535"; }
                else if (strcmp(sn, "u32") == 0) { su = true; smax = "4294967295ULL"; }
            }
            if (smax) {
                /* Rewrite x += y as x = ez_sized_add_check(x, y, ...) */
                const char *fn = NULL;
                if (su) {
                    if (strcmp(aop, "+=") == 0) fn = "ez_usized_add_check";
                    else if (strcmp(aop, "-=") == 0) fn = "ez_usized_sub_check";
                    else if (strcmp(aop, "*=") == 0) fn = "ez_usized_mul_check";
                } else {
                    if (strcmp(aop, "+=") == 0) fn = "ez_sized_add_check";
                    else if (strcmp(aop, "-=") == 0) fn = "ez_sized_sub_check";
                    else if (strcmp(aop, "*=") == 0) fn = "ez_sized_mul_check";
                }
                if (fn) {
                    /* Cache target address to avoid double-evaluation of side effects */
                    emitf(cg, "{ %s *_tgt = &(", ez_type_to_c_cg(cg, sn));
                    emit_expression(cg, node->data.assign.target);
                    emitf(cg, "); *_tgt = %s(*_tgt, ", fn);
                    emit_expression(cg, node->data.assign.value);
                    if (su) {
                        emitf(cg, ", %s, \"%s\", __FILE__, %d); }\n", smax, sn, node->token.line);
                    } else {
                        emitf(cg, ", %s, %s, \"%s\", __FILE__, %d); }\n", smin, smax, sn, node->token.line);
                    }
                    return;
                }
            }
        }
    }

    /* Array copy-by-default: arr2 = arr1 deep-copies the array.
     * Routes through emit_deep_array_copy so nested inner arrays get
     * independent backing storage (#1465). */
    if (strcmp(node->data.assign.op, "=") == 0 &&
        node->data.assign.value->kind == NODE_LABEL) {
        EzType *tgt_t = cg->type_table ? typetable_get(cg->type_table, node->data.assign.target) : NULL;
        if (tgt_t && tgt_t->kind == TK_ARRAY) {
            emit_expression(cg, node->data.assign.target);
            emit(cg, " = ");
            emit_deep_array_copy(cg, node->data.assign.value, tgt_t->element_type);
            emit(cg, ";\n");
            return;
        }
    }

    /* Default assignment — suppress ref auto-deref when assigning to a pointer target */

    /* #1521: when inside a loop scope and assigning a string/container
     * value to a plain variable with =, escape the value to the outer
     * arena so it survives the iteration arena's destruction. */
    if (cg->loop_scope_depth > 0 && strcmp(node->data.assign.op, "=") == 0 &&
        node->data.assign.target->kind == NODE_LABEL) {
        EzType *tgt_t = cg->type_table ? typetable_get(cg->type_table, node->data.assign.target) : NULL;
        if (tgt_t && tgt_t->kind == TK_STRING) {
            emit(cg, "{ EzString _esc_v = ");
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; ");
            emit_expression(cg, node->data.assign.target);
            emit(cg, " = ez_string_new(_ez_outer_arena, _esc_v.data, _esc_v.len); }\n");
            return;
        }
        if (tgt_t && tgt_t->name && type_needs_deep_copy(cg, tgt_t->name)) {
            const char *c_type = ez_type_to_c_cg(cg, tgt_t->name);
            emitf(cg, "{ %s _esc_v = ", c_type);
            emit_expression(cg, node->data.assign.value);
            emit(cg, "; EzArena *_esc_a = ez_default_arena; ez_default_arena = _ez_outer_arena; ");
            emit_expression(cg, node->data.assign.target);
            emit(cg, " = ");
            emit_value_deep_copy(cg, tgt_t->name, "_esc_v");
            emit(cg, "; ez_default_arena = _esc_a; }\n");
            return;
        }
    }

    emit_expression(cg, node->data.assign.target);
    emitf(cg, " %s ", node->data.assign.op);
    if (node->data.assign.value->kind == NODE_LABEL &&
        is_ref_var(cg, node->data.assign.value->data.label.value)) {
        EzType *tgt_t = cg->type_table ? typetable_get(cg->type_table, node->data.assign.target) : NULL;
        if (tgt_t && tgt_t->kind == TK_POINTER) {
            emit(cg, node->data.assign.value->data.label.value);
        } else {
            emit_expression(cg, node->data.assign.value);
        }
    } else {
        emit_expression(cg, node->data.assign.value);
    }
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

/* #1521: emit escape + cleanup for a non-void function return.
 * Escapes the return value (_ret) to _func_saved, then destroys
 * the function arena and returns. */
static void emit_func_return_escape(CodeGen *cg, const char *ret_type_name) {
    if (!ret_type_name) return;
    EzType *rt = type_from_name(ret_type_name);
    if (rt->kind == TK_STRING) {
        emit(cg, "_ret = ez_string_new(_func_saved, _ret.data, _ret.len); ");
    } else if (type_needs_deep_copy(cg, ret_type_name)) {
        emit(cg, "{ EzArena *_esc = ez_default_arena; ez_default_arena = _func_saved; _ret = ");
        emit_value_deep_copy(cg, ret_type_name, "_ret");
        emit(cg, "; ez_default_arena = _esc; } ");
    }
    emit(cg, "ez_default_arena = _func_saved; ");
    emit(cg, "ez_arena_destroy(_func_arena, __FILE__, __LINE__); free(_func_arena); ");
}

static void emit_return_statement(CodeGen *cg, AstNode *node) {
    /* Emit ensure cleanup before return */
    emit_ensure_cleanup(cg);

    /* Guard against malformed AST: count > 0 but NULL values array */
    if (node->data.return_stmt.count > 0 && !node->data.return_stmt.values) {
        emit_indent(cg);
        emit(cg, "ez_scope_restore(ez_default_arena, _scope_mark); ez_exit_func(); return;\n");
        return;
    }

    if (node->data.return_stmt.count > 1 && cg->current_func) {
        /* Multi-return: evaluate into temp, then exit and return */
        emit_indent(cg);
        const char *mbn = multi_ret_name(cg->current_func);
        emitf(cg, "{ EzMulti_%s _ret = (EzMulti_%s){", mbn, mbn);
        for (int i = 0; i < node->data.return_stmt.count; i++) {
            if (i > 0) emit(cg, ", ");
            emit_expression(cg, node->data.return_stmt.values[i]);
        }
        emit(cg, "}; ez_exit_func(); return _ret; }\n");
    } else if (node->data.return_stmt.count == 1 && cg->current_func &&
               cg->current_func->data.func_decl.return_type_count > 1) {
        /* Single value returned from multi-return function (or_return propagation) */
        int rc = cg->current_func->data.func_decl.return_type_count;
        emit_indent(cg);
        const char *mbn2 = multi_ret_name(cg->current_func);
        emitf(cg, "{ EzMulti_%s _ret = (EzMulti_%s){", mbn2, mbn2);
        for (int i = 0; i < rc - 1; i++) {
            emit(cg, "{0}, ");
        }
        emit_expression(cg, node->data.return_stmt.values[0]);
        emit(cg, "}; ez_exit_func(); return _ret; }\n");
    } else if (cg->current_func &&
               cg->current_func->data.func_decl.return_type_count == 0) {
        /* Void function */
        emit_indent(cg);
        emit(cg, "ez_scope_restore(ez_default_arena, _scope_mark); ez_exit_func(); return;\n");
    } else if (node->data.return_stmt.count == 1) {
        /* Single return value: evaluate into temp, then exit and return */
        /* Check if we need to dereference a pointer return (new() returns pointer,
         * but function may return struct by value) */
        bool needs_deref = false;
        if (cg->current_func && cg->type_table) {
            EzType *val_t = cg->type_table ? typetable_get(cg->type_table, node->data.return_stmt.values[0]) : NULL;
            if (val_t && val_t->kind == TK_POINTER &&
                cg->current_func->data.func_decl.return_type_count > 0) {
                const char *ret_tn = cg->current_func->data.func_decl.return_types[0];
                EzType *ret_t = type_from_name(ret_tn);
                if (ret_t->kind == TK_STRUCT) needs_deref = true;
            }
        }
        emit_indent(cg);
        if (needs_deref && cg->current_func) {
            const char *ret_tn = cg->current_func->data.func_decl.return_types[0];
            const char *c_ret = ez_type_to_c_cg(cg, ret_tn);
            emitf(cg, "{ %s _ret = *(", c_ret);
            emit_expression(cg, node->data.return_stmt.values[0]);
            emit(cg, "); ");
            emit_func_return_escape(cg, ret_tn);
        } else {
            emit(cg, "{ __auto_type _ret = ");
            emit_expression(cg, node->data.return_stmt.values[0]);
            emit(cg, "; ");
            if (cg->current_func && cg->current_func->data.func_decl.return_type_count > 0) {
                const char *ret_tn = cg->current_func->data.func_decl.return_types[0];
                emit_func_return_escape(cg, ret_tn);
            }
        }
        emit(cg, "ez_exit_func(); return _ret; }\n");
    } else if (node->data.return_stmt.count == 0 && cg->current_func &&
               cg->current_func->data.func_decl.return_names &&
               cg->current_func->data.func_decl.return_type_count > 0) {
        /* Bare return in function with named return values — collect named vars */
        int rc = cg->current_func->data.func_decl.return_type_count;
        if (rc == 1 && cg->current_func->data.func_decl.return_names[0]) {
            emit_indent(cg);
            emitf(cg, "{ __auto_type _ret = %s; ez_exit_func(); return _ret; }\n",
                safe_name(cg->current_func->data.func_decl.return_names[0]));
        } else {
            emit_indent(cg);
            const char *mbn3 = multi_ret_name(cg->current_func);
            emitf(cg, "{ EzMulti_%s _ret = (EzMulti_%s){", mbn3, mbn3);
            for (int i = 0; i < rc; i++) {
                if (i > 0) emit(cg, ", ");
                if (cg->current_func->data.func_decl.return_names[i]) {
                    emitf(cg, "%s", safe_name(cg->current_func->data.func_decl.return_names[i]));
                } else {
                    emit(cg, "0");
                }
            }
            emit(cg, "}; ez_exit_func(); return _ret; }\n");
        }
    } else {
        /* Bare return (no value, non-void — shouldn't happen but handle gracefully) */
        emit_indent(cg);
        emit(cg, "ez_scope_restore(ez_default_arena, _scope_mark); ez_exit_func(); return;\n");
    }
}

static void emit_block(CodeGen *cg, AstNode *node) {
    for (int i = 0; i < node->data.block.count; i++) {
        emit_statement(cg, node->data.block.stmts[i]);
    }
}

static void emit_if_statement(CodeGen *cg, AstNode *node) {
    /* #1521: per-block arena for if/otherwise so temporaries are freed */
    static int if_scope_counter = 0;
    int isc = if_scope_counter++;
    emit_indent(cg);
    emitf(cg, "{ ");
    if (cg->loop_scope_depth == 0) {
        emit(cg, "EzArena *_ez_outer_arena = ez_default_arena; ");
    }
    int if_depth = cg->loop_scope_depth;
    emitf(cg, "EzArena *_if_arena_%d = ez_arena_create(4096); ", isc);
    emitf(cg, "EzArena *_if_saved_%d = ez_default_arena; ", isc);
    emitf(cg, "ez_default_arena = _if_arena_%d;\n", isc);
    cg->loop_scope_depth++;

    emit_indent(cg);
    emit(cg, "if (");
    emit_expression(cg, node->data.if_stmt.condition);
    emit(cg, ") {\n");

    cg->indent++;
    emit_block(cg, node->data.if_stmt.consequence);
    cg->indent--;

    if (node->data.if_stmt.alternative) {
        if (node->data.if_stmt.alternative->kind == NODE_IF_STMT) {
            emit_indent(cg);
            emit(cg, "} else if (");
            emit_expression(cg, node->data.if_stmt.alternative->data.if_stmt.condition);
            emit(cg, ") {\n");
            cg->indent++;
            emit_block(cg, node->data.if_stmt.alternative->data.if_stmt.consequence);
            cg->indent--;
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

    cg->loop_scope_depth--;
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _if_saved_%d; ", isc);
    emitf(cg, "ez_arena_destroy(_if_arena_%d, __FILE__, __LINE__); free(_if_arena_%d); }\n", isc, isc);
}

static void emit_for_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    AstNode *iter = node->data.for_stmt.iterable;
    if (iter && iter->kind == NODE_RANGE_EXPR) {
        /* for i in range(start, end) or range(start, end, step) */
        const char *var = safe_name(node->data.for_stmt.var_name);

        if (iter->data.range_expr.start) {
            /* range(start, end) or range(start, end, step) */
            /* Determine comparison direction based on step sign */
            bool neg_step = false;
            if (iter->data.range_expr.step) {
                AstNode *s = iter->data.range_expr.step;
                if (s->kind == NODE_INT_VALUE && s->data.int_value.value < 0)
                    neg_step = true;
                else if (s->kind == NODE_PREFIX_EXPR && strcmp(s->data.prefix.op, "-") == 0)
                    neg_step = true;
            }
            emitf(cg, "for (int64_t %s = ", var);
            emit_expression(cg, iter->data.range_expr.start);
            emitf(cg, "; %s %s ", var, neg_step ? ">" : "<");
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
    /* #1521 phase 2: per-iteration scratch arena */
    if (cg->loop_scope_depth == 0) {
        emit_indent(cg);
        emit(cg, "EzArena *_ez_outer_arena = ez_default_arena;\n");
    }
    int f_depth = cg->loop_scope_depth;
    emit_indent(cg);
    emitf(cg, "EzArena *_iter_arena_%d = ez_arena_create(16384);\n", f_depth);
    emit_indent(cg);
    emitf(cg, "EzArena *_saved_arena_%d = ez_default_arena;\n", f_depth);
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _iter_arena_%d;\n", f_depth);
    cg->loop_scope_depth++;
    emit_block(cg, node->data.for_stmt.body);
    cg->loop_scope_depth--;
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _saved_arena_%d;\n", f_depth);
    emit_indent(cg);
    emitf(cg, "ez_arena_destroy(_iter_arena_%d, __FILE__, __LINE__); free(_iter_arena_%d);\n", f_depth, f_depth);
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
    /* #1521 phase 2: per-iteration scratch arena */
    if (cg->loop_scope_depth == 0) {
        emit_indent(cg);
        emit(cg, "EzArena *_ez_outer_arena = ez_default_arena;\n");
    }
    int w_depth = cg->loop_scope_depth;
    emit_indent(cg);
    emitf(cg, "EzArena *_iter_arena_%d = ez_arena_create(16384);\n", w_depth);
    emit_indent(cg);
    emitf(cg, "EzArena *_saved_arena_%d = ez_default_arena;\n", w_depth);
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _iter_arena_%d;\n", w_depth);
    cg->loop_scope_depth++;
    emit_block(cg, node->data.while_stmt.body);
    cg->loop_scope_depth--;
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _saved_arena_%d;\n", w_depth);
    emit_indent(cg);
    emitf(cg, "ez_arena_destroy(_iter_arena_%d, __FILE__, __LINE__); free(_iter_arena_%d);\n", w_depth, w_depth);
    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
}

static void emit_loop_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit(cg, "for (;;) {\n");

    cg->indent++;
    /* #1521 phase 2: per-iteration scratch arena */
    if (cg->loop_scope_depth == 0) {
        emit_indent(cg);
        emit(cg, "EzArena *_ez_outer_arena = ez_default_arena;\n");
    }
    int l_depth = cg->loop_scope_depth;
    emit_indent(cg);
    emitf(cg, "EzArena *_iter_arena_%d = ez_arena_create(16384);\n", l_depth);
    emit_indent(cg);
    emitf(cg, "EzArena *_saved_arena_%d = ez_default_arena;\n", l_depth);
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _iter_arena_%d;\n", l_depth);
    cg->loop_scope_depth++;
    emit_block(cg, node->data.loop_stmt.body);
    cg->loop_scope_depth--;
    emit_indent(cg);
    emitf(cg, "ez_default_arena = _saved_arena_%d;\n", l_depth);
    emit_indent(cg);
    emitf(cg, "ez_arena_destroy(_iter_arena_%d, __FILE__, __LINE__); free(_iter_arena_%d);\n", l_depth, l_depth);
    cg->indent--;
    emit_indent(cg);
    emit(cg, "}\n");
}

/* #1508: extract the base (unmangled) function name for multi-return
 * struct references. The monomorphiser temporarily renames functions
 * to `<name>__<binding>`, but the EzMulti typedef is emitted once
 * under the original name. Returns a pointer to a small ring of
 * static buffers so a few concurrent uses stay alive. */
static const char *multi_base_name(const char *fn_name) {
    static char bufs[4][256];
    static int bi = 0;
    char *out = bufs[bi]; bi = (bi + 1) & 3;
    const char *dunder = strstr(fn_name, "__");
    if (dunder) {
        size_t n = (size_t)(dunder - fn_name);
        if (n >= sizeof(bufs[0])) n = sizeof(bufs[0]) - 1;
        memcpy(out, fn_name, n);
        out[n] = '\0';
    } else {
        snprintf(out, sizeof(bufs[0]), "%s", fn_name);
    }
    return out;
}

/* #1502 + #1508: pick the right multi-return struct name. Use
 * the full mangled name when return types contain '?'
 * (per-instantiation struct), base name otherwise (shared). */
static const char *multi_ret_name(AstNode *func) {
    bool has_wc = false;
    for (int i = 0; i < func->data.func_decl.return_type_count; i++) {
        if (func->data.func_decl.return_types[i] &&
            strchr(func->data.func_decl.return_types[i], '?')) {
            has_wc = true;
            break;
        }
    }
    return has_wc ? func->data.func_decl.name
                  : multi_base_name(func->data.func_decl.name);
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
    /* #1508 + #1502: for multi-return, use `EzMulti_<name>`. The
     * monomorphiser temporarily renames the func to `<name>__<binding>`.
     * When return types DON'T contain '?', all instantiations share one
     * struct → use the base name (#1508). When return types DO contain
     * '?', each instantiation gets its own struct → use the full
     * mangled name (#1502). */
    static char buf[256];
    const char *fn_name = node->data.func_decl.name;
    bool has_wc_ret = false;
    for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
        if (node->data.func_decl.return_types[i] &&
            strchr(node->data.func_decl.return_types[i], '?')) {
            has_wc_ret = true;
            break;
        }
    }
    if (!has_wc_ret) {
        /* #1508: strip __<binding> suffix — shared struct. */
        snprintf(buf, sizeof(buf), "EzMulti_%s", multi_base_name(fn_name));
    } else {
        /* #1502: use the full (possibly mangled) name — per-instantiation
         * struct. The wildcard_binding is active so ez_type_to_c_cg will
         * substitute '?' in the struct fields. */
        snprintf(buf, sizeof(buf), "EzMulti_%s", fn_name);
    }
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
                emitf(cg, "%s *%s", ez_type_to_c_cg(cg,param->type_name), safe_name(param->name));
            } else {
                emitf(cg, "%s %s", ez_type_to_c_cg(cg,param->type_name), safe_name(param->name));
            }
        }

        if (node->data.func_decl.param_count == 0) {
            emit(cg, "void");
        }
        emit(cg, ")");
    }

    emit(cg, " {\n");
    cg->indent++;

    /* #1521: scope-based memory management.
     * Void functions: save/restore arena watermark to free temporaries.
     * Non-void functions: create a per-function arena so temporaries
     * are freed, and escape the return value to the caller's arena. */
    bool is_void_fn = (node->data.func_decl.return_type_count == 0);
    if (!is_main) {
        if (is_void_fn) {
            emit_indent(cg);
            emit(cg, "EzScopeMark _scope_mark = ez_scope_save(ez_default_arena);\n");
        } else {
            emit_indent(cg);
            emit(cg, "EzArena *_func_arena = ez_arena_create(65536);\n");
            emit_indent(cg);
            emit(cg, "EzArena *_func_saved = ez_default_arena;\n");
            emit_indent(cg);
            emit(cg, "ez_default_arena = _func_arena;\n");
        }
    }

    AstNode *prev_func = cg->current_func;
    int prev_using_count = cg->using_module_count;
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
                        safe_name(node->data.func_decl.return_names[i]));
                } else if (strncmp(c_t, "EzStruct_", 9) == 0 || strncmp(c_t, "EzEnum_", 7) == 0) {
                    emitf(cg, "%s %s = {0};\n", c_t,
                        safe_name(node->data.func_decl.return_names[i]));
                } else {
                    emitf(cg, "%s %s = 0;\n", c_t,
                        safe_name(node->data.func_decl.return_names[i]));
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
        /* #1521: cleanup function-scoped memory */
        if (!is_main) {
            if (is_void_fn) {
                emit_indent(cg);
                emit(cg, "ez_scope_restore(ez_default_arena, _scope_mark);\n");
            } else {
                emit_indent(cg);
                emit(cg, "ez_default_arena = _func_saved;\n");
                emit_indent(cg);
                emit(cg, "ez_arena_destroy(_func_arena, __FILE__, __LINE__); free(_func_arena);\n");
            }
        }
        emit_indent(cg);
        emit(cg, "ez_exit_func();\n");

        /* Implicit return for named return values (fall-through without explicit return) */
        if (node->data.func_decl.return_names &&
            node->data.func_decl.return_type_count > 0) {
            bool has_named = false;
            for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
                if (node->data.func_decl.return_names[i]) { has_named = true; break; }
            }
            if (has_named) {
                int rc = node->data.func_decl.return_type_count;
                emit_indent(cg);
                if (rc == 1 && node->data.func_decl.return_names[0]) {
                    emitf(cg, "return %s;\n", safe_name(node->data.func_decl.return_names[0]));
                } else {
                    emitf(cg, "return (EzMulti_%s){", multi_ret_name(node));
                    for (int i = 0; i < rc; i++) {
                        if (i > 0) emit(cg, ", ");
                        if (node->data.func_decl.return_names[i]) {
                            emitf(cg, "%s", safe_name(node->data.func_decl.return_names[i]));
                        } else {
                            emit(cg, "0");
                        }
                    }
                    emit(cg, "};\n");
                }
            }
        }
    }
    cg->current_func = prev_func;
    cg->using_module_count = prev_using_count;
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
        bool is_map_iter = (coll_t && coll_t->kind == TK_MAP);

        if (is_map_iter) {
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
            if (coll_t->value_type) c_val = ez_map_elem_c_type(cg, coll_t->value_type);

            /* Iterate in insertion order using the order array */
            char slot_name[32];
            snprintf(slot_name, sizeof(slot_name), "_ez_sl%d", map_iter_counter - 1);
            /* Guard against mutation during iteration */
            emit_expression(cg, coll);
            emit(cg, ".iterating++;\n");
            emit_indent(cg);
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
                        c_key, safe_name(node->data.for_each.index_name), c_key);
                    emit_expression(cg, coll);
                    emitf(cg, ", %s);\n", slot_name);
                }
                if (strcmp(node->data.for_each.var_name, "_") != 0) {
                    emit_indent(cg);
                    emitf(cg, "%s %s = *(%s *)ez_map_value_at(&",
                        c_val, safe_name(node->data.for_each.var_name), c_val);
                    emit_expression(cg, coll);
                    emitf(cg, ", %s);\n", slot_name);
                }
            } else {
                /* One-var form: for_each key in map (keys only) */
                emit_indent(cg);
                emitf(cg, "%s %s = *(%s *)ez_map_key_at(&",
                    c_key, safe_name(node->data.for_each.var_name), c_key);
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
            emitf(cg, "int32_t %s = _ez_str.data[%s];\n", safe_name(node->data.for_each.var_name), idx_name);
        } else {
            /* for_each item in array → iterate EzArray */
            const char *c_elem = "int64_t";
            if (coll_t && coll_t->kind == TK_ARRAY && coll_t->element_type) {
                /* Wildcard substitution (#1443): a generic parameter
                 * typed `[?]` stores "?" as its element_type; swap it
                 * out for the active instantiation's concrete binding
                 * before resolving to a C type. */
                const char *elem_tn = cg_effective_type_str(cg, coll_t->element_type);
                EzType *et = type_from_name(elem_tn);
                if (et->kind == TK_FLOAT) c_elem = "double";
                else if (et->kind == TK_BOOL) c_elem = "bool";
                else if (et->kind == TK_STRING) c_elem = "EzString";
                else if (et->kind == TK_ARRAY) c_elem = "EzArray";
                else if (et->kind == TK_MAP) c_elem = "EzMap";
                else if (et->kind == TK_STRUCT) c_elem = ez_type_to_c_cg(cg, elem_tn);
                else if (et->kind == TK_POINTER) c_elem = ez_type_to_c_cg(cg, elem_tn);
                else if (et->kind == TK_CHAR) c_elem = "int32_t";
                else if (et->kind == TK_BYTE) c_elem = "uint8_t";
            }

            emitf(cg, "for (int32_t %s = 0; %s < ", idx_name, idx_name);
            emit_expression(cg, coll);
            emitf(cg, ".len; %s++) {\n", idx_name);
            cg->indent++;
            emit_indent(cg);
            emitf(cg, "%s %s = EZ_ARRAY_GET(", c_elem, safe_name(node->data.for_each.var_name));
            emit_expression(cg, coll);
            emitf(cg, ", %s, %s);\n", c_elem, idx_name);
        }

        /* #1521 phase 2: per-iteration scratch arena */
        if (cg->loop_scope_depth == 0) {
            emit_indent(cg);
            emit(cg, "EzArena *_ez_outer_arena = ez_default_arena;\n");
        }
        int fe_depth = cg->loop_scope_depth;
        emit_indent(cg);
        emitf(cg, "EzArena *_iter_arena_%d = ez_arena_create(16384);\n", fe_depth);
        emit_indent(cg);
        emitf(cg, "EzArena *_saved_arena_%d = ez_default_arena;\n", fe_depth);
        emit_indent(cg);
        emitf(cg, "ez_default_arena = _iter_arena_%d;\n", fe_depth);
        cg->loop_scope_depth++;
        emit_block(cg, node->data.for_each.body);
        cg->loop_scope_depth--;
        emit_indent(cg);
        emitf(cg, "ez_default_arena = _saved_arena_%d;\n", fe_depth);
        emit_indent(cg);
        emitf(cg, "ez_arena_destroy(_iter_arena_%d, __FILE__, __LINE__); free(_iter_arena_%d);\n", fe_depth, fe_depth);
        cg->indent--;
        emit_indent(cg);
        emit(cg, "}\n");
        /* Decrement map iteration guard */
        if (is_map_iter) {
            emit_indent(cg);
            emit_expression(cg, coll);
            emit(cg, ".iterating--;\n");
        }
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
                    /* Check if step is a negative literal to reverse comparison direction */
                    bool neg_step = (r->data.range_expr.step &&
                        r->data.range_expr.step->kind == NODE_PREFIX_EXPR &&
                        strcmp(r->data.range_expr.step->data.prefix.op, "-") == 0);
                    emit(cg, "(");
                    emit_expression(cg, val);
                    emit(cg, neg_step ? " <= " : " >= ");
                    emit_expression(cg, r->data.range_expr.start);
                    emit(cg, " && ");
                    emit_expression(cg, val);
                    emit(cg, neg_step ? " > " : " < ");
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
    case NODE_FUNC_DECL: {
        /* Generic function (#1443): emit one specialised copy per
         * concrete instantiation the typechecker recorded. If a
         * generic function was declared but never called, there are
         * no instantiations and we skip emission entirely — the
         * un-specialised form has '?' in its signature and can't be
         * compiled as C. */
        bool has_wc = false;
        for (int i = 0; i < node->data.func_decl.param_count && !has_wc; i++) {
            if (node->data.func_decl.params[i].type_name &&
                strchr(node->data.func_decl.params[i].type_name, '?')) has_wc = true;
        }
        for (int i = 0; i < node->data.func_decl.return_type_count && !has_wc; i++) {
            if (node->data.func_decl.return_types[i] &&
                strchr(node->data.func_decl.return_types[i], '?')) has_wc = true;
        }
        if (has_wc) {
            const char *orig_name = node->data.func_decl.name;
            for (int ii = 0; ii < node->data.func_decl.instantiation_count; ii++) {
                const char *concrete = node->data.func_decl.instantiations[ii];
                /* Build the mangled name: `<name>__<concrete>` with
                 * non-alnum chars replaced by underscores so array/map
                 * bindings stay legal C identifiers. */
                char mangled[256];
                size_t pos = snprintf(mangled, sizeof(mangled), "%s__", orig_name);
                for (const char *c = concrete; *c && pos < sizeof(mangled) - 1; c++) {
                    mangled[pos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
                }
                mangled[pos] = '\0';

                node->data.func_decl.name = mangled;
                const char *saved = cg->wildcard_binding;
                cg->wildcard_binding = concrete;
                /* Per-instantiation multi-return typedefs were already
                 * emitted in the forward-declaration loop (#1502). */
                emit_func_declaration(cg, node, false);
                cg->wildcard_binding = saved;
            }
            node->data.func_decl.name = orig_name;
        } else {
            emit_func_declaration(cg, node,
                strcmp(node->data.func_decl.name, "main") == 0);
        }
        break;
    }
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
        /* Function-scoped using: add to using_modules so bare-name
         * dispatch works for the rest of this function body. */
        for (int j = 0; j < node->data.using_stmt.count; j++) {
            if (cg->using_module_count >= cg->using_module_cap) {
                cg->using_module_cap = cg->using_module_cap ? cg->using_module_cap * 2 : 8;
                cg->using_modules = realloc(cg->using_modules,
                    sizeof(const char *) * (size_t)cg->using_module_cap);
            }
            cg->using_modules[cg->using_module_count++] = node->data.using_stmt.modules[j];
        }
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
    cg.global_init = buf_create(256);
    cg.indent = 0;
    cg.has_mem = false;
    cg.has_fmt = false;
    cg.file = file;
    cg.enum_names = NULL;
    cg.enum_count = 0;
    cg.enum_cap = 0;
    cg.current_func = NULL;
    cg.loop_scope_depth = 0;
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
    cg.struct_decls = NULL;
    cg.struct_decl_count = 0;
    cg.struct_decl_cap = 0;
    cg.using_modules = NULL;
    cg.using_module_count = 0;
    cg.using_module_cap = 0;
    cg.alias_names = NULL;
    cg.alias_modules = NULL;
    cg.alias_count = 0;
    cg.alias_cap = 0;
    cg.imported_modules = NULL;
    cg.imported_module_count = 0;
    cg.imported_module_cap = 0;
    cg.c_headers = NULL;
    cg.c_header_count = 0;
    cg.c_header_cap = 0;
    cg.has_c_imports = false;
    cg.wildcard_binding = NULL;
    return cg;
}

void codegen_generate(CodeGen *cg, AstNode *program) {
    if (program->kind != NODE_PROGRAM) return;

    /* First pass: scan for imports, using declarations, and struct declarations */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_IMPORT_STMT) {
            for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                ImportItem *item = &stmt->data.import_stmt.items[j];
                if (item->is_stdlib && item->module) {
                    if (strcmp(item->module, "mem") == 0) cg->has_mem = true;
                    if (strcmp(item->module, "fmt") == 0) cg->has_fmt = true;
                }
                /* Collect C interop headers */
                if (item->is_c_import && item->path) {
                    cg->has_c_imports = true;
                    if (cg->c_header_count >= cg->c_header_cap) {
                        cg->c_header_cap = cg->c_header_cap ? cg->c_header_cap * 2 : 4;
                        cg->c_headers = realloc(cg->c_headers,
                            sizeof(const char *) * (size_t)cg->c_header_cap);
                    }
                    cg->c_headers[cg->c_header_count++] = item->path;
                }
                /* Track all imported module names */
                if (item->module) {
                    const char *mname = item->alias ? item->alias : item->module;
                    if (cg->imported_module_count >= cg->imported_module_cap) {
                        cg->imported_module_cap = cg->imported_module_cap ? cg->imported_module_cap * 2 : 8;
                        cg->imported_modules = realloc(cg->imported_modules,
                            sizeof(const char *) * (size_t)cg->imported_module_cap);
                    }
                    cg->imported_modules[cg->imported_module_count++] = mname;
                }
                /* Track alias → module mapping */
                if (item->alias && item->module && strcmp(item->alias, item->module) != 0) {
                    if (cg->alias_count >= cg->alias_cap) {
                        cg->alias_cap = cg->alias_cap ? cg->alias_cap * 2 : 8;
                        cg->alias_names = realloc(cg->alias_names,
                            sizeof(const char *) * (size_t)cg->alias_cap);
                        cg->alias_modules = realloc(cg->alias_modules,
                            sizeof(const char *) * (size_t)cg->alias_cap);
                    }
                    cg->alias_names[cg->alias_count] = item->alias;
                    cg->alias_modules[cg->alias_count] = item->module;
                    cg->alias_count++;
                }
            }
            /* import and use — register all modules for using */
            if (stmt->data.import_stmt.auto_use) {
                for (int j = 0; j < stmt->data.import_stmt.count; j++) {
                    ImportItem *item = &stmt->data.import_stmt.items[j];
                    if (item->module) {
                        if (cg->using_module_count >= cg->using_module_cap) {
                            cg->using_module_cap = cg->using_module_cap ? cg->using_module_cap * 2 : 8;
                            cg->using_modules = realloc(cg->using_modules,
                                sizeof(const char *) * (size_t)cg->using_module_cap);
                        }
                        cg->using_modules[cg->using_module_count++] = item->module;
                    }
                }
            }
        }
        if (stmt->kind == NODE_USING_STMT) {
            for (int j = 0; j < stmt->data.using_stmt.count; j++) {
                if (cg->using_module_count >= cg->using_module_cap) {
                    cg->using_module_cap = cg->using_module_cap ? cg->using_module_cap * 2 : 8;
                    cg->using_modules = realloc(cg->using_modules,
                        sizeof(const char *) * (size_t)cg->using_module_cap);
                }
                cg->using_modules[cg->using_module_count++] = stmt->data.using_stmt.modules[j];
            }
        }
        if (stmt->kind == NODE_STRUCT_DECL) {
            if (cg->struct_decl_count >= cg->struct_decl_cap) {
                cg->struct_decl_cap = cg->struct_decl_cap ? cg->struct_decl_cap * 2 : 16;
                cg->struct_decls = realloc(cg->struct_decls,
                    sizeof(AstNode *) * (size_t)cg->struct_decl_cap);
            }
            cg->struct_decls[cg->struct_decl_count++] = stmt;
        }
    }

    /* Emit preamble */
    emit(cg, "/* Generated by EZC */\n");
    emit(cg, "#include \"ez_runtime.h\"\n");
    emit(cg, "#include \"ez_array.h\"\n");
    emit(cg, "#include \"ez_map.h\"\n");
    emit(cg, "#include \"ez_builtins.h\"\n");
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
    emit(cg, "#include \"ez_sync.h\"\n");
    emit(cg, "#include \"ez_atomic_mod.h\"\n");
    emit(cg, "#include \"ez_channels.h\"\n");
    emit(cg, "#include \"ez_regex.h\"\n");
    emit(cg, "#include \"ez_net.h\"\n");
    emit(cg, "#include \"ez_http.h\"\n");
    emit(cg, "#include \"ez_server.h\"\n");
    emit(cg, "#include \"ez_bigint.h\"\n");

    /* Emit user C interop headers (after EZ internals to prevent collisions) */
    if (cg->c_header_count > 0) {
        emit(cg, "\n/* C interop headers */\n");
        for (int i = 0; i < cg->c_header_count; i++) {
            const char *hdr = cg->c_headers[i];
            if (strncmp(hdr, "./", 2) == 0 || strncmp(hdr, "../", 3) == 0) {
                emitf(cg, "#include \"%s\"\n", hdr);
            } else {
                emitf(cg, "#include <%s>\n", hdr);
            }
        }
    }
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

            /* Check if this is a string enum (auto-detect from values) */
            bool is_string_enum = false;
            for (int j = 0; j < stmt->data.enum_decl.value_count; j++) {
                if (stmt->data.enum_decl.values[j].value &&
                    stmt->data.enum_decl.values[j].value->kind == NODE_STRING_VALUE) {
                    is_string_enum = true;
                    break;
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
                    }
                    /* Non-flags without explicit value: omit `= N` and
                     * let C's enum auto-increment continue from the
                     * last explicit value (#1511). The old code emitted
                     * `= j` (0-based position) which ignored preceding
                     * explicit values entirely. */
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

        /* Emit forward declarations so pointer fields can reference any struct.
         * Skip generic structs — their forward decls are per-instantiation. */
        for (int i = 0; i < struct_count; i++) {
            if (structs[i]->data.struct_decl.is_generic) continue;
            emitf(cg, "typedef struct EzStruct_%s EzStruct_%s;\n",
                structs[i]->data.struct_decl.name,
                structs[i]->data.struct_decl.name);
        }
        if (struct_count > 0) emit(cg, "\n");

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
                    /* #1520: skip generic structs here — they're
                     * emitted per-instantiation below. */
                    if (s->data.struct_decl.is_generic) continue;
                    emitf(cg, "struct EzStruct_%s {\n", s->data.struct_decl.name);
                    for (int j = 0; j < s->data.struct_decl.field_count; j++) {
                        StructField *f = &s->data.struct_decl.fields[j];
                        emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg, f->type_name), safe_name(f->name));
                    }
                    emit(cg, "};\n\n");
                }
            }
        }
        /* If any structs couldn't be emitted (circular deps), emit them anyway */
        for (int i = 0; i < struct_count; i++) {
            if (!emitted[i]) {
                AstNode *s = structs[i];
                emitf(cg, "struct EzStruct_%s {\n", s->data.struct_decl.name);
                for (int j = 0; j < s->data.struct_decl.field_count; j++) {
                    StructField *f = &s->data.struct_decl.fields[j];
                    emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg, f->type_name), safe_name(f->name));
                }
                emit(cg, "};\n\n");
            }
        }
    }

    /* #1520: emit per-instantiation typedefs for generic (wildcard) structs.
     * For each recorded binding, substitute ? → concrete in field types
     * and emit under a mangled name (e.g. EzStruct_Pair__int). */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_STRUCT_DECL || !stmt->data.struct_decl.is_generic) continue;
        for (int ii = 0; ii < stmt->data.struct_decl.instantiation_count; ii++) {
            const char *concrete = stmt->data.struct_decl.instantiations[ii];
            char mangled[256];
            size_t pos = snprintf(mangled, sizeof(mangled), "%s__", stmt->data.struct_decl.name);
            for (const char *c = concrete; *c && pos < sizeof(mangled) - 1; c++) {
                mangled[pos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
            }
            mangled[pos] = '\0';
            /* Forward declaration */
            emitf(cg, "typedef struct EzStruct_%s EzStruct_%s;\n", mangled, mangled);
            emitf(cg, "struct EzStruct_%s {\n", mangled);
            const char *saved = cg->wildcard_binding;
            cg->wildcard_binding = concrete;
            for (int j = 0; j < stmt->data.struct_decl.field_count; j++) {
                StructField *f = &stmt->data.struct_decl.fields[j];
                emitf(cg, "    %s %s;\n", ez_type_to_c_cg(cg, f->type_name), safe_name(f->name));
            }
            cg->wildcard_binding = saved;
            emit(cg, "};\n\n");
        }
    }

    /* #1496: emit JSON parse/stringify helpers for #json structs. Each
     * #json struct gets two static functions:
     *   - ez_json_parse_<Name>(arena, json_string) → EzStruct_<Name>
     *   - ez_json_stringify_<Name>(arena, struct_value) → EzString
     * These are called by json.parse() / json.stringify() which the
     * typechecker dispatches based on the target/argument struct type. */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind != NODE_STRUCT_DECL || !stmt->data.struct_decl.is_json) continue;
        const char *sn = stmt->data.struct_decl.name;
        int fc = stmt->data.struct_decl.field_count;

        /* --- parse: JSON string → struct --- */
        emitf(cg, "static EzStruct_%s ez_json_parse_%s(EzArena *arena, EzString text) {\n", sn, sn);
        emitf(cg, "    EzStruct_%s _r = {0};\n", sn);
        emitf(cg, "    EzMap _m = ez_json_decode(arena, text);\n");
        for (int j = 0; j < fc; j++) {
            StructField *f = &stmt->data.struct_decl.fields[j];
            const char *ct = ez_type_to_c_cg(cg, f->type_name);
            if (strcmp(f->type_name, "string") == 0) {
                emitf(cg, "    { EzString _k = ez_string_lit(\"%s\"); void *_v = ez_map_get(&_m, &_k);\n", f->name);
                emitf(cg, "      if (_v) _r.%s = *(EzString *)_v; }\n", safe_name(f->name));
            } else if (strcmp(f->type_name, "int") == 0 || strcmp(f->type_name, "i64") == 0) {
                emitf(cg, "    { EzString _k = ez_string_lit(\"%s\"); void *_v = ez_map_get(&_m, &_k);\n", f->name);
                emitf(cg, "      if (_v) { EzString _sv = *(EzString *)_v; _r.%s = ez_builtin_string_to_int(_sv); } }\n", safe_name(f->name));
            } else if (strcmp(f->type_name, "float") == 0 || strcmp(f->type_name, "f64") == 0) {
                emitf(cg, "    { EzString _k = ez_string_lit(\"%s\"); void *_v = ez_map_get(&_m, &_k);\n", f->name);
                emitf(cg, "      if (_v) { EzString _sv = *(EzString *)_v; _r.%s = ez_builtin_string_to_float(_sv); } }\n", safe_name(f->name));
            } else if (strcmp(f->type_name, "bool") == 0) {
                emitf(cg, "    { EzString _k = ez_string_lit(\"%s\"); void *_v = ez_map_get(&_m, &_k);\n", f->name);
                emitf(cg, "      if (_v) { EzString _sv = *(EzString *)_v; _r.%s = (_sv.len == 4 && memcmp(_sv.data, \"true\", 4) == 0); } }\n", safe_name(f->name));
            }
        }
        emitf(cg, "    return _r;\n}\n\n");

        /* --- stringify: struct → JSON string --- */
        emitf(cg, "static EzString ez_json_stringify_%s(EzArena *arena, EzStruct_%s _s) {\n", sn, sn);

        /* Pass 1: compute exact buffer size at runtime.
         * Field names are identifiers (no escaping), so their contribution
         * is compile-time constant.  String values need json_escaped_len().
         * Numeric types use safe upper bounds (21 for int64, 24 for double). */
        {
            /* Compile-time constant part: braces + separators + all key literals */
            int fixed = 2; /* { } */
            for (int j = 0; j < fc; j++) {
                StructField *f = &stmt->data.struct_decl.fields[j];
                if (j > 0) fixed += 2; /* ", " */
                fixed += 2 + (int)strlen(f->name) + 2; /* "key": */
                /* Value upper bound for non-string types */
                if (strcmp(f->type_name, "int") == 0 || strcmp(f->type_name, "i64") == 0) {
                    fixed += 21;
                } else if (strcmp(f->type_name, "float") == 0 || strcmp(f->type_name, "f64") == 0) {
                    fixed += 24;
                } else if (strcmp(f->type_name, "bool") == 0) {
                    fixed += 5;
                }
                /* string fields are added at runtime below */
            }
            emitf(cg, "    size_t _need = %d;\n", fixed);
        }
        /* Add runtime string field sizes */
        for (int j = 0; j < fc; j++) {
            StructField *f = &stmt->data.struct_decl.fields[j];
            if (strcmp(f->type_name, "string") == 0) {
                emitf(cg, "    _need += json_escaped_len(_s.%s);\n", safe_name(f->name));
            }
        }

        /* Pass 2: allocate and write */
        emitf(cg, "    char *_buf = ez_arena_alloc(arena, _need + 1);\n");
        emitf(cg, "    int _pos = 0;\n");
        emitf(cg, "    _buf[_pos++] = '{';\n");
        for (int j = 0; j < fc; j++) {
            StructField *f = &stmt->data.struct_decl.fields[j];
            if (j > 0) emitf(cg, "    _buf[_pos++] = ','; _buf[_pos++] = ' ';\n");
            /* Key */
            emitf(cg, "    _buf[_pos++] = '\"';\n");
            emitf(cg, "    memcpy(_buf + _pos, \"%s\", %d); _pos += %d;\n",
                f->name, (int)strlen(f->name), (int)strlen(f->name));
            emitf(cg, "    _buf[_pos++] = '\"'; _buf[_pos++] = ':'; _buf[_pos++] = ' ';\n");
            /* Value */
            if (strcmp(f->type_name, "string") == 0) {
                emitf(cg, "    json_append_escaped(_buf, &_pos, _s.%s);\n", safe_name(f->name));
            } else if (strcmp(f->type_name, "int") == 0 || strcmp(f->type_name, "i64") == 0) {
                emitf(cg, "    _pos += snprintf(_buf + _pos, _need + 1 - (size_t)_pos, \"%%lld\", (long long)_s.%s);\n",
                    safe_name(f->name));
            } else if (strcmp(f->type_name, "float") == 0 || strcmp(f->type_name, "f64") == 0) {
                emitf(cg, "    _pos += snprintf(_buf + _pos, _need + 1 - (size_t)_pos, \"%%g\", _s.%s);\n",
                    safe_name(f->name));
            } else if (strcmp(f->type_name, "bool") == 0) {
                emitf(cg, "    { const char *_bv = _s.%s ? \"true\" : \"false\"; int _bl = _s.%s ? 4 : 5;\n",
                    safe_name(f->name), safe_name(f->name));
                emitf(cg, "      memcpy(_buf + _pos, _bv, (size_t)_bl); _pos += _bl; }\n");
            }
        }
        emitf(cg, "    _buf[_pos++] = '}';\n");
        emitf(cg, "    _buf[_pos] = '\\0';\n");
        emitf(cg, "    return (EzString){_buf, (int32_t)_pos};\n");
        emitf(cg, "}\n\n");
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

    /* Emit multi-return type definitions. Skip generic functions whose
     * return types contain '?' — those need per-instantiation typedefs
     * emitted during monomorphisation (#1502). */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL &&
            stmt->data.func_decl.return_type_count > 1) {
            bool has_wc = false;
            for (int ri = 0; ri < stmt->data.func_decl.return_type_count; ri++) {
                if (stmt->data.func_decl.return_types[ri] &&
                    strchr(stmt->data.func_decl.return_types[ri], '?')) {
                    has_wc = true;
                    break;
                }
            }
            if (!has_wc) emit_multi_return_typedef(cg, stmt);
        }
    }

    /* Emit forward declarations for all functions (including struct-namespaced) */
    for (int i = 0; i < cg->func_count; i++) {
        AstNode *stmt = cg->all_funcs[i];
        if (stmt->kind != NODE_FUNC_DECL) continue;
        if (strcmp(stmt->data.func_decl.name, "main") == 0) {
            emit(cg, "static void ez_fn_main(void);\n");
            continue;
        }

        /* Detect wildcard generics (#1443) — emit one forward per
         * instantiation under a mangled name, skipping the un-specialised
         * signature which would contain '?' in C. */
        bool has_wc = false;
        for (int j = 0; j < stmt->data.func_decl.param_count && !has_wc; j++) {
            if (stmt->data.func_decl.params[j].type_name &&
                strchr(stmt->data.func_decl.params[j].type_name, '?')) has_wc = true;
        }
        for (int j = 0; j < stmt->data.func_decl.return_type_count && !has_wc; j++) {
            if (stmt->data.func_decl.return_types[j] &&
                strchr(stmt->data.func_decl.return_types[j], '?')) has_wc = true;
        }

        int emit_rounds = has_wc ? stmt->data.func_decl.instantiation_count : 1;
        const char *orig_name = stmt->data.func_decl.name;
        for (int r = 0; r < emit_rounds; r++) {
            const char *saved_binding = cg->wildcard_binding;
            char mangled[256];
            if (has_wc) {
                const char *concrete = stmt->data.func_decl.instantiations[r];
                cg->wildcard_binding = concrete;
                size_t pos = snprintf(mangled, sizeof(mangled), "%s__", orig_name);
                for (const char *c = concrete; *c && pos < sizeof(mangled) - 1; c++) {
                    mangled[pos++] = (isalnum((unsigned char)*c) || *c == '_') ? *c : '_';
                }
                mangled[pos] = '\0';
            }
            const char *emit_name = has_wc ? mangled : orig_name;
            /* Temporarily set the func name to the mangled version so
             * func_return_type sees the right name for multi-return
             * structs (#1502 + #1508). */
            if (has_wc) stmt->data.func_decl.name = mangled;
            /* #1502: emit per-instantiation multi-return typedef
             * before the forward declaration that references it. */
            if (has_wc && stmt->data.func_decl.return_type_count > 1) {
                bool has_wc_ret = false;
                for (int ri = 0; ri < stmt->data.func_decl.return_type_count; ri++) {
                    if (stmt->data.func_decl.return_types[ri] &&
                        strchr(stmt->data.func_decl.return_types[ri], '?')) {
                        has_wc_ret = true;
                        break;
                    }
                }
                if (has_wc_ret) emit_multi_return_typedef(cg, stmt);
            }
            emitf(cg, "static %s ", func_return_type(cg, stmt));
            if (has_wc) stmt->data.func_decl.name = orig_name;
            emitf(cg, "ez_fn_%s(", emit_name);
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
            cg->wildcard_binding = saved_binding;
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
    /* Initialize file-scope arrays that can't use C static initializers */
    if (cg->global_init.len > 0) {
        buf_append(&cg->output, cg->global_init.data);
    }
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
    free(cg->enum_names);
    free(cg->all_funcs);
    free(cg->ref_vars);
    free(cg->bigint_var_names);
    free(cg->bigint_var_types);
    free(cg->struct_decls);
    free(cg->using_modules);
    free(cg->alias_names);
    free(cg->alias_modules);
    free(cg->imported_modules);
    free(cg->c_headers);
}
