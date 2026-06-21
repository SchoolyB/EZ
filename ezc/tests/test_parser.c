/*
 * test_parser.c - Unit tests for the EZC parser
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include "../src/util/arena.h"
#include "../src/util/error.h"
#include "../src/lexer/lexer.h"
#include "../src/parser/parser.h"

static Arena *arena;
static DiagnosticList *diag;

static AstNode *parse(const char *input) {
    diag = diag_create();
    diag->use_color = false;
    Lexer *l = lexer_create(arena, input, "test.ez");
    Parser *p = parser_create(arena, l, "test.ez", diag);
    return parser_parse_program(p);
}

static AstNode *first_stmt(AstNode *prog) {
    if (!prog || prog->kind != NODE_PROGRAM) return NULL;
    if (prog->data.program.stmt_count == 0) return NULL;
    return prog->data.program.stmts[0];
}

static void test_parse_var_decl(void) {
    AstNode *prog = parse("mut x int = 42");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT_STR_EQ(stmt->data.var_decl.name, "x");
    ASSERT_STR_EQ(stmt->data.var_decl.type_name, "int");
    ASSERT(stmt->data.var_decl.mutable);
    ASSERT_NOT_NULL(stmt->data.var_decl.value);
    ASSERT_EQ(stmt->data.var_decl.value->kind, NODE_INT_VALUE);
}

static void test_parse_const_decl(void) {
    AstNode *prog = parse("const PI float = 3.14");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT(!stmt->data.var_decl.mutable);
    ASSERT_STR_EQ(stmt->data.var_decl.name, "PI");
}

static void test_parse_func_decl(void) {
    AstNode *prog = parse("do add(a int, b int) -> int { return a + b }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
    ASSERT_STR_EQ(stmt->data.func_decl.name, "add");
    ASSERT_EQ(stmt->data.func_decl.param_count, 2);
    ASSERT_EQ(stmt->data.func_decl.return_type_count, 1);
    ASSERT_STR_EQ(stmt->data.func_decl.return_types[0], "int");
}

static void test_parse_func_no_return(void) {
    AstNode *prog = parse("do greet() { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
    ASSERT_STR_EQ(stmt->data.func_decl.name, "greet");
    ASSERT_EQ(stmt->data.func_decl.param_count, 0);
    ASSERT_EQ(stmt->data.func_decl.return_type_count, 0);
}

static void test_parse_import_single(void) {
    AstNode *prog = parse("import @math");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_IMPORT_STMT);
    ASSERT_EQ(stmt->data.import_stmt.count, 1);
    ASSERT(stmt->data.import_stmt.items[0].is_stdlib);
    ASSERT_STR_EQ(stmt->data.import_stmt.items[0].module, "math");
}

static void test_parse_import_multi(void) {
    AstNode *prog = parse("import @math, @strings");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_IMPORT_STMT);
    ASSERT_EQ(stmt->data.import_stmt.count, 2);
    ASSERT_STR_EQ(stmt->data.import_stmt.items[0].module, "math");
    ASSERT_STR_EQ(stmt->data.import_stmt.items[1].module, "strings");
}

static void test_parse_if_or_otherwise(void) {
    AstNode *prog = parse("if x > 0 { } or x == 0 { } otherwise { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_IF_STMT);
    ASSERT_NOT_NULL(stmt->data.if_stmt.consequence);
    ASSERT_NOT_NULL(stmt->data.if_stmt.alternative);
    ASSERT_EQ(stmt->data.if_stmt.alternative->kind, NODE_IF_STMT);
}

static void test_parse_for_range(void) {
    AstNode *prog = parse("for i in range(0, 10) { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FOR_STMT);
    ASSERT_STR_EQ(stmt->data.for_stmt.var_name, "i");
    ASSERT_NOT_NULL(stmt->data.for_stmt.iterable);
    ASSERT_EQ(stmt->data.for_stmt.iterable->kind, NODE_RANGE_EXPR);
}

static void test_parse_while(void) {
    AstNode *prog = parse("as_long_as x < 10 { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_WHILE_STMT);
}

static void test_parse_loop(void) {
    AstNode *prog = parse("loop { break }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_LOOP_STMT);
}

static void test_parse_struct_decl(void) {
    AstNode *prog = parse("const Person struct { name string\n age int }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_STRUCT_DECL);
    ASSERT_STR_EQ(stmt->data.struct_decl.name, "Person");
    ASSERT_EQ(stmt->data.struct_decl.field_count, 2);
}

static void test_parse_enum_decl(void) {
    AstNode *prog = parse("const Color enum { RED\n GREEN\n BLUE }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ENUM_DECL);
    ASSERT_STR_EQ(stmt->data.enum_decl.name, "Color");
    ASSERT_EQ(stmt->data.enum_decl.value_count, 3);
}

static void test_parse_struct_literal(void) {
    AstNode *prog = parse("mut p = Person{name: \"Alice\", age: 30}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT_NOT_NULL(stmt->data.var_decl.value);
    ASSERT_EQ(stmt->data.var_decl.value->kind, NODE_STRUCT_VALUE);
    ASSERT_EQ(stmt->data.var_decl.value->data.struct_value.count, 2);
}

static void test_parse_array_literal(void) {
    AstNode *prog = parse("mut nums [int] = {1, 2, 3}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT_STR_EQ(stmt->data.var_decl.type_name, "[int]");
    ASSERT_NOT_NULL(stmt->data.var_decl.value);
    ASSERT_EQ(stmt->data.var_decl.value->kind, NODE_ARRAY_VALUE);
    ASSERT_EQ(stmt->data.var_decl.value->data.array_value.count, 3);
}

static void test_parse_ensure(void) {
    AstNode *prog = parse("ensure cleanup()");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ENSURE_STMT);
}

static void test_parse_when(void) {
    AstNode *prog = parse("when x { is 1 { } is 2 { } default { } }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_WHEN_STMT);
    ASSERT_EQ(stmt->data.when_stmt.case_count, 2);
    ASSERT_NOT_NULL(stmt->data.when_stmt.default_body);
}

static void test_parse_default_params(void) {
    AstNode *prog = parse("do greet(name string = \"World\") { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
    ASSERT_EQ(stmt->data.func_decl.param_count, 1);
    ASSERT_NOT_NULL(stmt->data.func_decl.params[0].default_value);
}

static void test_parse_hex_int(void) {
    AstNode *prog = parse("mut x int = 0xFF");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->data.var_decl.value->kind, NODE_INT_VALUE);
    ASSERT_EQ(stmt->data.var_decl.value->data.int_value.value, 255);
}

static void test_parse_octal_int(void) {
    AstNode *prog = parse("mut x int = 0o10");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->data.var_decl.value->data.int_value.value, 8);
}

static void test_parse_binary_int(void) {
    AstNode *prog = parse("mut x int = 0b1010");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->data.var_decl.value->data.int_value.value, 10);
}

static void test_parse_mut_keyword(void) {
    AstNode *prog = parse("mut x int = 42");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT(stmt->data.var_decl.mutable);
}

static void test_parse_array_return_type(void) {
    AstNode *prog = parse("do get() -> [int] { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
    ASSERT_EQ(stmt->data.func_decl.return_type_count, 1);
    ASSERT_STR_EQ(stmt->data.func_decl.return_types[0], "[int]");
}

static void test_parse_error_reports(void) {
    AstNode *prog = parse("mut x int =");
    (void)prog;
    ASSERT(diag_has_errors(diag));
}

static void test_parse_func_ref(void) {
    AstNode *prog = parse("do main() { mut fn = ()double }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
}

static void test_parse_struct_func(void) {
    AstNode *prog = parse(
        "const Point struct {\n"
        "  x int\n"
        "  do create(x int) -> Point { return Point{x: x} }\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_STRUCT_DECL);
    ASSERT_EQ(stmt->data.struct_decl.field_count, 1);
    ASSERT_EQ(stmt->data.struct_decl.func_count, 1);
}

static void test_parse_or_return(void) {
    AstNode *prog = parse(
        "do wrapper() -> (string, Error) {\n"
        "  mut val = fallible() or_return\n"
        "  return val, nil\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
}

static void test_parse_flags_enum(void) {
    AstNode *prog = parse(
        "#flags\n"
        "const Perms enum { READ\n WRITE\n EXEC }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ENUM_DECL);
    ASSERT(stmt->data.enum_decl.is_flags);
}

static void test_parse_string_enum(void) {
    AstNode *prog = parse(
        "const Status enum {\n"
        "  TODO = \"todo\"\n"
        "  DONE = \"done\"\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ENUM_DECL);
    ASSERT_EQ(stmt->data.enum_decl.value_count, 2);
}

static void test_parse_map_type(void) {
    AstNode *prog = parse("mut m map[string:int] = {:}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT(strstr(stmt->data.var_decl.type_name, "map") != NULL);
}

static void test_parse_fixed_array(void) {
    AstNode *prog = parse("const arr [int, 3] = {1, 2, 3}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT(strstr(stmt->data.var_decl.type_name, "int,3") != NULL);
}

static void test_parse_nested_array(void) {
    AstNode *prog = parse("mut m [[int]] = {{1}, {2}}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_VAR_DECL);
    ASSERT(strstr(stmt->data.var_decl.type_name, "[[int]]") != NULL);
}

static void test_parse_private_struct_func(void) {
    /* Private functions inside structs */
    AstNode *prog = parse(
        "const Foo struct {\n"
        "  private do secret() -> int { return 42 }\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_STRUCT_DECL);
    ASSERT_EQ(stmt->data.struct_decl.func_count, 1);
}

static void test_parse_for_each_index(void) {
    AstNode *prog = parse(
        "do main() {\n"
        "  for_each i, item in arr { }\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FUNC_DECL);
}

/* Helper: get the first statement inside main()'s body */
static AstNode *body_stmt(AstNode *prog, int index) {
    AstNode *fn = first_stmt(prog);
    if (!fn || fn->kind != NODE_FUNC_DECL) return NULL;
    AstNode *body = fn->data.func_decl.body;
    if (!body || body->kind != NODE_BLOCK_STMT) return NULL;
    if (index >= body->data.block.count) return NULL;
    return body->data.block.stmts[index];
}

/* Helper: get the value expression from first var decl in main */
static AstNode *var_value(AstNode *prog) {
    AstNode *stmt = body_stmt(prog, 0);
    if (!stmt || stmt->kind != NODE_VAR_DECL) return NULL;
    return stmt->data.var_decl.value;
}

/* --- Expression AST Tests --- */

static void test_parse_infix_expr(void) {
    AstNode *prog = parse("do main() { mut x = 1 + 2 * 3 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_INFIX_EXPR);
    /* 1 + (2 * 3) — left is 1, right is 2*3 */
    ASSERT_EQ(val->data.infix.op, TOK_PLUS);
    ASSERT_EQ(val->data.infix.left->kind, NODE_INT_VALUE);
    ASSERT_EQ(val->data.infix.right->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.right->data.infix.op, TOK_ASTERISK);
}

static void test_parse_prefix_expr(void) {
    AstNode *prog = parse("do main() { mut x = -42 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_PREFIX_EXPR);
    ASSERT_EQ(val->data.prefix.op, TOK_MINUS);
    ASSERT_EQ(val->data.prefix.right->kind, NODE_INT_VALUE);
}

static void test_parse_not_expr(void) {
    AstNode *prog = parse("do main() { mut x = !true }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_PREFIX_EXPR);
    ASSERT_EQ(val->data.prefix.op, TOK_BANG);
    ASSERT_EQ(val->data.prefix.right->kind, NODE_BOOL_VALUE);
}

static void test_parse_postfix_expr(void) {
    AstNode *prog = parse("do main() { mut x int = 0\n x++ }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_EXPR_STMT);
    ASSERT_EQ(stmt->data.expr_stmt.expr->kind, NODE_POSTFIX_EXPR);
    ASSERT_EQ(stmt->data.expr_stmt.expr->data.postfix.op, TOK_INCREMENT);
}

static void test_parse_index_expr(void) {
    AstNode *prog = parse("do main() { mut a [int] = {1,2,3}\n mut x = a[1] }");
    AstNode *val = body_stmt(prog, 1);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_VAR_DECL);
    ASSERT_EQ(val->data.var_decl.value->kind, NODE_INDEX_EXPR);
}

static void test_parse_member_expr(void) {
    AstNode *prog = parse(
        "const P struct { x int }\n"
        "do main() { mut p P = P{x: 1}\n mut v = p.x }");
    /* main is stmt[1] (after struct) */
    AstNode *fn = prog->data.program.stmts[1];
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    AstNode *vdecl = fn->data.func_decl.body->data.block.stmts[1];
    ASSERT_EQ(vdecl->data.var_decl.value->kind, NODE_MEMBER_EXPR);
}

static void test_parse_call_expr(void) {
    AstNode *prog = parse("do foo() -> int { return 1 }\n do main() { mut x = foo() }");
    AstNode *fn = prog->data.program.stmts[1];
    AstNode *vdecl = fn->data.func_decl.body->data.block.stmts[0];
    ASSERT_NOT_NULL(vdecl);
    ASSERT_EQ(vdecl->data.var_decl.value->kind, NODE_CALL_EXPR);
}

static void test_parse_cast_expr(void) {
    AstNode *prog = parse("do main() { mut x = cast(42, u8) }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_CAST_EXPR);
    ASSERT_STR_EQ(val->data.cast.target_type, "u8");
}

static void test_parse_new_expr(void) {
    AstNode *prog = parse(
        "const Foo struct { x int }\n"
        "do main() { mut p = new(Foo) }");
    AstNode *fn = prog->data.program.stmts[1];
    AstNode *vdecl = fn->data.func_decl.body->data.block.stmts[0];
    ASSERT_NOT_NULL(vdecl);
    ASSERT_EQ(vdecl->data.var_decl.value->kind, NODE_NEW_EXPR);
    ASSERT_STR_EQ(vdecl->data.var_decl.value->data.new_expr.type_name, "Foo");
}

static void test_parse_comparison_operators(void) {
    AstNode *prog = parse("do main() { mut x = 1 < 2 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.op, TOK_LT);
}

static void test_parse_logical_and_or(void) {
    AstNode *prog = parse("do main() { mut x = true && false || true }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_INFIX_EXPR);
    /* || has lower precedence than && */
    ASSERT_EQ(val->data.infix.op, TOK_OR);
}

/* --- Literal AST Tests --- */

static void test_parse_bool_literal(void) {
    AstNode *prog = parse("do main() { mut x = true }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_BOOL_VALUE);
}

static void test_parse_float_literal(void) {
    AstNode *prog = parse("do main() { mut x = 3.14 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_FLOAT_VALUE);
}

static void test_parse_string_literal(void) {
    AstNode *prog = parse("do main() { mut x = \"hello\" }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_STRING_VALUE);
}

static void test_parse_char_literal(void) {
    AstNode *prog = parse("do main() { mut x = 'A' }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_CHAR_VALUE);
}

static void test_parse_nil_literal(void) {
    AstNode *prog = parse("do main() { mut x = nil }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_NIL_VALUE);
}

/* --- Statement AST Tests --- */

static void test_parse_assign_stmt(void) {
    AstNode *prog = parse("do main() { mut x int = 1\n x = 2 }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ASSIGN_STMT);
}

static void test_parse_compound_assign(void) {
    AstNode *prog = parse("do main() { mut x int = 1\n x += 5 }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ASSIGN_STMT);
}

static void test_parse_break_continue(void) {
    AstNode *prog = parse("do main() { loop { break } }");
    AstNode *loop = body_stmt(prog, 0);
    ASSERT_NOT_NULL(loop);
    ASSERT_EQ(loop->kind, NODE_LOOP_STMT);
    AstNode *brk = loop->data.loop_stmt.body->data.block.stmts[0];
    ASSERT_EQ(brk->kind, NODE_BREAK_STMT);
}

static void test_parse_continue_stmt(void) {
    AstNode *prog = parse("do main() { loop { continue } }");
    AstNode *loop = body_stmt(prog, 0);
    AstNode *cont = loop->data.loop_stmt.body->data.block.stmts[0];
    ASSERT_EQ(cont->kind, NODE_CONTINUE_STMT);
}

static void test_parse_map_literal(void) {
    AstNode *prog = parse("do main() { mut m = {\"a\": 1, \"b\": 2} }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_MAP_VALUE);
}

static void test_parse_multi_return(void) {
    AstNode *prog = parse("do pair() -> (int, int) { return 1, 2 }");
    AstNode *fn = first_stmt(prog);
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    ASSERT_EQ(fn->data.func_decl.return_type_count, 2);
}

static void test_parse_using_stmt(void) {
    AstNode *prog = parse("import @math\n using math");
    /* using should be stmt[1] */
    ASSERT(prog->data.program.stmt_count >= 2);
    ASSERT_EQ(prog->data.program.stmts[1]->kind, NODE_USING_STMT);
}

static void test_parse_blank_ident(void) {
    AstNode *prog = parse("do pair() -> (int, int) { return 1, 2 }\n do main() { mut _, b = pair() }");
    AstNode *fn = prog->data.program.stmts[1];
    /* Just check it parses without crashing */
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
}

/* --- P3: Remaining parser coverage gaps --- */

static void test_parse_range_with_step(void) {
    AstNode *prog = parse("for i in range(0, 10, 2) { }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FOR_STMT);
    AstNode *r = stmt->data.for_stmt.iterable;
    ASSERT_EQ(r->kind, NODE_RANGE_EXPR);
    ASSERT_NOT_NULL(r->data.range_expr.start);
    ASSERT_NOT_NULL(r->data.range_expr.end);
    ASSERT_NOT_NULL(r->data.range_expr.step);
    ASSERT_EQ(r->data.range_expr.step->kind, NODE_INT_VALUE);
    ASSERT_EQ(r->data.range_expr.step->data.int_value.value, 2);
}

static void test_parse_pointer_deref(void) {
    AstNode *prog = parse(
        "const Node struct { val int }\n"
        "do main() { mut p = new(Node)\n mut v = p^.val }");
    AstNode *fn = prog->data.program.stmts[1];
    AstNode *vdecl = fn->data.func_decl.body->data.block.stmts[1];
    ASSERT_NOT_NULL(vdecl);
    /* p^.val — should parse as member access on a deref */
    ASSERT_EQ(vdecl->data.var_decl.value->kind, NODE_MEMBER_EXPR);
}

/* module keyword removed in v3.0 — test removed */

static void test_parse_local_import(void) {
    AstNode *prog = parse("import \"./mylib\"");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_IMPORT_STMT);
    ASSERT(!stmt->data.import_stmt.items[0].is_stdlib);
    ASSERT_STR_EQ(stmt->data.import_stmt.items[0].path, "./mylib");
}


static void test_parse_precedence_add_mul(void) {
    /* 1 + 2 * 3 should parse as 1 + (2 * 3) */
    AstNode *prog = parse("do main() { mut x = 1 + 2 * 3 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.op, TOK_PLUS);
    ASSERT_EQ(val->data.infix.left->kind, NODE_INT_VALUE);
    ASSERT_EQ(val->data.infix.left->data.int_value.value, 1);
    ASSERT_EQ(val->data.infix.right->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.right->data.infix.op, TOK_ASTERISK);
}

static void test_parse_precedence_comparison_logical(void) {
    /* a > 0 && b < 10 should parse as (a > 0) && (b < 10) */
    AstNode *prog = parse("do main() { mut x = 1 > 0 && 2 < 10 }");
    AstNode *val = var_value(prog);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.op, TOK_AND);
    ASSERT_EQ(val->data.infix.left->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.left->data.infix.op, TOK_GT);
    ASSERT_EQ(val->data.infix.right->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.infix.right->data.infix.op, TOK_LT);
}

static void test_parse_named_return(void) {
    AstNode *prog = parse("do foo() -> (name string, age int) { }");
    AstNode *fn = first_stmt(prog);
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    ASSERT_EQ(fn->data.func_decl.return_type_count, 2);
    ASSERT_STR_EQ(fn->data.func_decl.return_names[0], "name");
    ASSERT_STR_EQ(fn->data.func_decl.return_names[1], "age");
}

static void test_parse_error_bad_var_decl(void) {
    AstNode *prog = parse("mut = 42");
    (void)prog;
    ASSERT(diag_has_errors(diag));
}

static void test_parse_error_unexpected_token(void) {
    AstNode *prog = parse("do 42() { }");
    (void)prog;
    ASSERT(diag_has_errors(diag));
}

/* --- Additional parser coverage --- */

static void test_parse_for_each_stmt(void) {
    AstNode *prog = parse("do main() { mut arr [int] = {1,2,3}\n for_each x in arr { } }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_FOR_EACH_STMT);
}

static void test_parse_while_alias(void) {
    /* while is an alias for as_long_as */
    AstNode *prog = parse("while true { break }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_WHILE_STMT);
}

static void test_parse_empty_block(void) {
    AstNode *prog = parse("do main() { }");
    AstNode *fn = first_stmt(prog);
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    ASSERT_NOT_NULL(fn->data.func_decl.body);
    ASSERT_EQ(fn->data.func_decl.body->kind, NODE_BLOCK_STMT);
    ASSERT_EQ(fn->data.func_decl.body->data.block.count, 0);
}

static void test_parse_grouped_params(void) {
    AstNode *prog = parse("do add(a, b int) -> int { return a + b }");
    AstNode *fn = first_stmt(prog);
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    ASSERT_EQ(fn->data.func_decl.param_count, 2);
    ASSERT_STR_EQ(fn->data.func_decl.params[0].type_name, "int");
    ASSERT_STR_EQ(fn->data.func_decl.params[1].type_name, "int");
}

static void test_parse_compound_assign_mul(void) {
    AstNode *prog = parse("do main() { mut x int = 2\n x *= 5 }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ASSIGN_STMT);
}

static void test_parse_compound_assign_div(void) {
    AstNode *prog = parse("do main() { mut x int = 10\n x /= 2 }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ASSIGN_STMT);
}

static void test_parse_compound_assign_mod(void) {
    AstNode *prog = parse("do main() { mut x int = 10\n x %= 3 }");
    AstNode *stmt = body_stmt(prog, 1);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ASSIGN_STMT);
}

static void test_parse_multiple_when_cases(void) {
    AstNode *prog = parse(
        "when x { is 1 { } is 2 { } is 3 { } is 4 { } default { } }");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_WHEN_STMT);
    ASSERT_EQ(stmt->data.when_stmt.case_count, 4);
}

static void test_parse_nested_if(void) {
    AstNode *prog = parse(
        "do main() {\n"
        "  if true {\n"
        "    if false { }\n"
        "  }\n"
        "}");
    AstNode *outer_if = body_stmt(prog, 0);
    ASSERT_NOT_NULL(outer_if);
    ASSERT_EQ(outer_if->kind, NODE_IF_STMT);
    /* The inner if is inside the consequence block */
    AstNode *inner = outer_if->data.if_stmt.consequence->data.block.stmts[0];
    ASSERT_EQ(inner->kind, NODE_IF_STMT);
}

static void test_parse_struct_nested_field(void) {
    AstNode *prog = parse(
        "const Inner struct { val int }\n"
        "const Outer struct { inner Inner }");
    ASSERT_EQ(prog->data.program.stmt_count, 2);
    AstNode *outer = prog->data.program.stmts[1];
    ASSERT_EQ(outer->kind, NODE_STRUCT_DECL);
    ASSERT_STR_EQ(outer->data.struct_decl.fields[0].type_name, "Inner");
}

static void test_parse_in_operator(void) {
    AstNode *prog = parse("do main() { mut arr [int] = {1,2,3}\n mut x = 1 in arr }");
    AstNode *val = body_stmt(prog, 1);
    ASSERT_NOT_NULL(val);
    ASSERT_EQ(val->kind, NODE_VAR_DECL);
    ASSERT_EQ(val->data.var_decl.value->kind, NODE_INFIX_EXPR);
    ASSERT_EQ(val->data.var_decl.value->data.infix.op, TOK_IN);
}


static void test_parse_return_multiple_values(void) {
    AstNode *prog = parse(
        "do pair() -> (int, string) {\n"
        "  return 42, \"hello\"\n"
        "}");
    AstNode *fn = first_stmt(prog);
    ASSERT_NOT_NULL(fn);
    ASSERT_EQ(fn->kind, NODE_FUNC_DECL);
    ASSERT_EQ(fn->data.func_decl.return_type_count, 2);
    ASSERT_STR_EQ(fn->data.func_decl.return_types[0], "int");
    ASSERT_STR_EQ(fn->data.func_decl.return_types[1], "string");
}

static void test_parse_import_alias(void) {
    /* EZ alias syntax: import alias @module */
    AstNode *prog = parse("import m @math");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_IMPORT_STMT);
    ASSERT_STR_EQ(stmt->data.import_stmt.items[0].module, "math");
    ASSERT_STR_EQ(stmt->data.import_stmt.items[0].alias, "m");
}

static void test_parse_enum_with_values(void) {
    AstNode *prog = parse(
        "const Priority enum {\n"
        "  LOW = 1\n"
        "  MED = 2\n"
        "  HIGH = 3\n"
        "}");
    AstNode *stmt = first_stmt(prog);
    ASSERT_NOT_NULL(stmt);
    ASSERT_EQ(stmt->kind, NODE_ENUM_DECL);
    ASSERT_EQ(stmt->data.enum_decl.value_count, 3);
}

/* Helper: check if a specific error code was emitted */
static bool parser_has_code(DiagnosticList *d, const char *code) {
    for (int i = 0; i < d->count; i++) {
        if (d->items[i].code && strcmp(d->items[i].code, code) == 0) return true;
    }
    return false;
}

/* ===== Parser error code tests ===== */

static void test_parse_error_E2025_non_int_array_size(void) {
    AstNode *prog = parse("do main() { mut a [int, \"five\"] = {} }");
    (void)prog;
    ASSERT(diag_has_errors(diag));
}

static void test_parse_error_E2059_empty_when(void) {
    AstNode *prog = parse("do main() { when 1 { } }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2059"));
}

static void test_parse_error_E2060_too_many_returns(void) {
    AstNode *prog = parse(
        "do bad() -> (int, int, int, int, int, int, int, int, int, int, int, int, int, int, int, int, int) { }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2060"));
}

static void test_parse_error_E2068_mut_struct(void) {
    AstNode *prog = parse("mut S struct { x int }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2068"));
}

static void test_parse_error_E2069_semicolon_in_struct(void) {
    AstNode *prog = parse("const S struct { x int; y int }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2069"));
}

static void test_parse_error_E2070_wildcard_in_var(void) {
    AstNode *prog = parse("do main() { mut x ? = 42 }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2070"));
}

static void test_parse_error_E2071_empty_interpolation(void) {
    AstNode *prog = parse("do main() { println(\"${}\") }");
    (void)prog;
    ASSERT(parser_has_code(diag, "E2071"));
}

int main(void) {
    arena = arena_create(256 * 1024);
    printf("\n");
    RUN_TEST(test_parse_var_decl);
    RUN_TEST(test_parse_const_decl);
    RUN_TEST(test_parse_func_decl);
    RUN_TEST(test_parse_func_no_return);
    RUN_TEST(test_parse_import_single);
    RUN_TEST(test_parse_import_multi);
    RUN_TEST(test_parse_if_or_otherwise);
    RUN_TEST(test_parse_for_range);
    RUN_TEST(test_parse_while);
    RUN_TEST(test_parse_loop);
    RUN_TEST(test_parse_struct_decl);
    RUN_TEST(test_parse_enum_decl);
    RUN_TEST(test_parse_struct_literal);
    RUN_TEST(test_parse_array_literal);
    RUN_TEST(test_parse_ensure);
    RUN_TEST(test_parse_when);
    RUN_TEST(test_parse_default_params);
    RUN_TEST(test_parse_hex_int);
    RUN_TEST(test_parse_octal_int);
    RUN_TEST(test_parse_binary_int);
    RUN_TEST(test_parse_mut_keyword);
    RUN_TEST(test_parse_array_return_type);
    RUN_TEST(test_parse_error_reports);
    RUN_TEST(test_parse_func_ref);
    RUN_TEST(test_parse_struct_func);
    RUN_TEST(test_parse_or_return);
    RUN_TEST(test_parse_flags_enum);
    RUN_TEST(test_parse_string_enum);
    RUN_TEST(test_parse_map_type);
    RUN_TEST(test_parse_fixed_array);
    RUN_TEST(test_parse_nested_array);
    RUN_TEST(test_parse_private_struct_func);
    RUN_TEST(test_parse_for_each_index);

    /* Expression AST tests */
    RUN_TEST(test_parse_infix_expr);
    RUN_TEST(test_parse_prefix_expr);
    RUN_TEST(test_parse_not_expr);
    RUN_TEST(test_parse_postfix_expr);
    RUN_TEST(test_parse_index_expr);
    RUN_TEST(test_parse_member_expr);
    RUN_TEST(test_parse_call_expr);
    RUN_TEST(test_parse_cast_expr);
    RUN_TEST(test_parse_new_expr);
    RUN_TEST(test_parse_comparison_operators);
    RUN_TEST(test_parse_logical_and_or);

    /* Literal AST tests */
    RUN_TEST(test_parse_bool_literal);
    RUN_TEST(test_parse_float_literal);
    RUN_TEST(test_parse_string_literal);
    RUN_TEST(test_parse_char_literal);
    RUN_TEST(test_parse_nil_literal);

    /* Statement AST tests */
    RUN_TEST(test_parse_assign_stmt);
    RUN_TEST(test_parse_compound_assign);
    RUN_TEST(test_parse_break_continue);
    RUN_TEST(test_parse_continue_stmt);
    RUN_TEST(test_parse_map_literal);
    RUN_TEST(test_parse_multi_return);
    RUN_TEST(test_parse_using_stmt);
    RUN_TEST(test_parse_blank_ident);

    /* P3: Remaining parser coverage */
    RUN_TEST(test_parse_range_with_step);
    RUN_TEST(test_parse_pointer_deref);
    /* test_parse_module_decl removed — module keyword removed in v3.0 */
    RUN_TEST(test_parse_local_import);
    RUN_TEST(test_parse_precedence_add_mul);
    RUN_TEST(test_parse_precedence_comparison_logical);
    RUN_TEST(test_parse_named_return);
    RUN_TEST(test_parse_error_bad_var_decl);
    RUN_TEST(test_parse_error_unexpected_token);

    /* Additional parser coverage */
    RUN_TEST(test_parse_for_each_stmt);
    RUN_TEST(test_parse_while_alias);
    RUN_TEST(test_parse_empty_block);
    RUN_TEST(test_parse_grouped_params);
    RUN_TEST(test_parse_compound_assign_mul);
    RUN_TEST(test_parse_compound_assign_div);
    RUN_TEST(test_parse_compound_assign_mod);
    RUN_TEST(test_parse_multiple_when_cases);
    RUN_TEST(test_parse_nested_if);
    RUN_TEST(test_parse_struct_nested_field);
    RUN_TEST(test_parse_in_operator);
    RUN_TEST(test_parse_return_multiple_values);
    RUN_TEST(test_parse_import_alias);
    RUN_TEST(test_parse_enum_with_values);

    /* Parser error code tests */
    RUN_TEST(test_parse_error_E2025_non_int_array_size);
    RUN_TEST(test_parse_error_E2059_empty_when);
    RUN_TEST(test_parse_error_E2060_too_many_returns);
    RUN_TEST(test_parse_error_E2068_mut_struct);
    RUN_TEST(test_parse_error_E2069_semicolon_in_struct);
    RUN_TEST(test_parse_error_E2070_wildcard_in_var);
    RUN_TEST(test_parse_error_E2071_empty_interpolation);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
