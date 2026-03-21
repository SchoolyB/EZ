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

static void emit_newline(CodeGen *cg) {
    emit(cg, "\n");
}

/* Map EZ type name to C type */
static const char *ez_type_to_c(const char *type_name) {
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

    /* Default: assume it's a struct type */
    return type_name;
}

static bool is_string_type(const char *type_name) {
    return type_name && strcmp(type_name, "string") == 0;
}

/* --- Expression Emission --- */

static void emit_expression(CodeGen *cg, AstNode *node) {
    if (!node) return;

    switch (node->kind) {
    case NODE_LABEL:
        emit(cg, node->data.label.value);
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

    case NODE_NIL_VALUE:
        emit(cg, "NULL");
        break;

    case NODE_PREFIX_EXPR:
        emit(cg, "(");
        emit(cg, node->data.prefix.op);
        emit_expression(cg, node->data.prefix.right);
        emit(cg, ")");
        break;

    case NODE_INFIX_EXPR:
        emit(cg, "(");
        emit_expression(cg, node->data.infix.left);
        emitf(cg, " %s ", node->data.infix.op);
        emit_expression(cg, node->data.infix.right);
        emit(cg, ")");
        break;

    case NODE_POSTFIX_EXPR:
        emit_expression(cg, node->data.postfix.left);
        emit(cg, node->data.postfix.op);
        break;

    case NODE_CALL_EXPR:
        emit_call_expression(cg, node);
        break;

    case NODE_MEMBER_EXPR:
        emit_expression(cg, node->data.member.object);
        emitf(cg, ".%s", node->data.member.member);
        break;

    case NODE_INDEX_EXPR:
        emit_expression(cg, node->data.index_expr.left);
        emit(cg, "[");
        emit_expression(cg, node->data.index_expr.index);
        emit(cg, "]");
        break;

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
                if (arg->kind == NODE_STRING_VALUE) suffix = "_str";
                else if (arg->kind == NODE_FLOAT_VALUE) suffix = "_float";
                else if (arg->kind == NODE_BOOL_VALUE) suffix = "_bool";
                emitf(cg, "ez_std_println%s(", suffix);
                emit_expression(cg, arg);
                emit(cg, ")");
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
    }

    /* Generic function call */
    emit(cg, "ez_fn_");
    if (node->data.call.function->kind == NODE_LABEL) {
        emit(cg, node->data.call.function->data.label.value);
    } else {
        emit_expression(cg, node->data.call.function);
    }
    emit(cg, "(");
    for (int i = 0; i < node->data.call.arg_count; i++) {
        if (i > 0) emit(cg, ", ");
        emit_expression(cg, node->data.call.args[i]);
    }
    emit(cg, ")");
}

/* --- Statement Emission --- */

static void emit_var_declaration(CodeGen *cg, AstNode *node) {
    emit_indent(cg);

    const char *c_type = ez_type_to_c(node->data.var_decl.type_name);

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
    emit_expression(cg, node->data.assign.target);
    emitf(cg, " %s ", node->data.assign.op);
    emit_expression(cg, node->data.assign.value);
    emit(cg, ";\n");
}

static void emit_return_statement(CodeGen *cg, AstNode *node) {
    emit_indent(cg);
    emit(cg, "return");
    if (node->data.return_stmt.count > 0) {
        emit(cg, " ");
        emit_expression(cg, node->data.return_stmt.values[0]);
    }
    emit(cg, ";\n");
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

static void emit_func_declaration(CodeGen *cg, AstNode *node, bool is_main) {
    /* Return type */
    if (is_main) {
        emit(cg, "static void ez_fn_main(void)");
    } else {
        if (node->data.func_decl.return_type_count == 0) {
            emit(cg, "static void ");
        } else {
            emitf(cg, "static %s ", ez_type_to_c(node->data.func_decl.return_types[0]));
        }
        emitf(cg, "ez_fn_%s(", node->data.func_decl.name);

        /* Parameters */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            if (i > 0) emit(cg, ", ");
            Param *param = &node->data.func_decl.params[i];
            if (param->mutable) {
                emitf(cg, "%s *%s", ez_type_to_c(param->type_name), param->name);
            } else {
                emitf(cg, "%s %s", ez_type_to_c(param->type_name), param->name);
            }
        }

        if (node->data.func_decl.param_count == 0) {
            emit(cg, "void");
        }
        emit(cg, ")");
    }

    emit(cg, " {\n");
    cg->indent++;
    if (node->data.func_decl.body) {
        emit_block(cg, node->data.func_decl.body);
    }
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
    case NODE_FUNC_DECL:
        emit_func_declaration(cg, node,
            strcmp(node->data.func_decl.name, "main") == 0);
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

CodeGen codegen_create(const char *file) {
    CodeGen cg;
    cg.output = buf_create(4096);
    cg.indent = 0;
    cg.has_std = false;
    cg.file = file;
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
                if (item->is_stdlib && item->module &&
                    strcmp(item->module, "std") == 0) {
                    cg->has_std = true;
                }
            }
        }
    }

    /* Emit preamble */
    emit(cg, "/* Generated by EZC */\n");
    emit(cg, "#include \"ez_runtime.h\"\n");
    if (cg->has_std) {
        emit(cg, "#include \"ez_std.h\"\n");
    }
    emit(cg, "\n");

    /* Emit forward declarations for user functions */
    for (int i = 0; i < program->data.program.stmt_count; i++) {
        AstNode *stmt = program->data.program.stmts[i];
        if (stmt->kind == NODE_FUNC_DECL) {
            if (strcmp(stmt->data.func_decl.name, "main") == 0) {
                emit(cg, "static void ez_fn_main(void);\n");
            } else {
                emit(cg, "static ");
                if (stmt->data.func_decl.return_type_count == 0) {
                    emit(cg, "void ");
                } else {
                    emitf(cg, "%s ", ez_type_to_c(stmt->data.func_decl.return_types[0]));
                }
                emitf(cg, "ez_fn_%s(", stmt->data.func_decl.name);
                for (int j = 0; j < stmt->data.func_decl.param_count; j++) {
                    if (j > 0) emit(cg, ", ");
                    Param *param = &stmt->data.func_decl.params[j];
                    if (param->mutable) {
                        emitf(cg, "%s *", ez_type_to_c(param->type_name));
                    } else {
                        emit(cg, ez_type_to_c(param->type_name));
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
