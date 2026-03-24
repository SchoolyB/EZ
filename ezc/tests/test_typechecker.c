/*
 * test_typechecker.c - Unit tests for the EZC type checker
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include "../src/util/arena.h"
#include "../src/util/error.h"
#include "../src/lexer/lexer.h"
#include "../src/parser/parser.h"
#include "../src/typechecker/typechecker.h"

static Arena *arena;

static TypeTable *check(const char *input) {
    DiagnosticList *diag = diag_create();
    diag->use_color = false;
    Lexer *l = lexer_create(arena, input, "test.ez");
    Parser *p = parser_create(arena, l, "test.ez", diag);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(diag, "test.ez");
    typechecker_check(tc, prog);
    return typechecker_get_table(tc);
}

/* Helper: parse, type check, and get the type of the first expression in main */
static EzType *expr_type(const char *expr_code) {
    char buf[512];
    snprintf(buf, sizeof(buf),
        "do main() { temp _result = %s }", expr_code);
    DiagnosticList *diag = diag_create();
    diag->use_color = false;
    Lexer *l = lexer_create(arena, buf, "test.ez");
    Parser *p = parser_create(arena, l, "test.ez", diag);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(diag, "test.ez");
    typechecker_check(tc, prog);
    TypeTable *tt = typechecker_get_table(tc);

    /* Find the var decl's value and look up its type */
    AstNode *main_fn = prog->data.program.stmts[0];
    AstNode *var_decl = main_fn->data.func_decl.body->data.block.stmts[0];
    if (var_decl->kind == NODE_VAR_DECL && var_decl->data.var_decl.value) {
        return typetable_get(tt, var_decl->data.var_decl.value);
    }
    return NULL;
}

/* --- Scope and Symbol Table --- */

static void test_scope_define_lookup(void) {
    Scope *s = scope_create(NULL);
    scope_define(s, "x", &TYPE_INT, true);
    Symbol *sym = scope_lookup(s, "x");
    ASSERT_NOT_NULL(sym);
    ASSERT_EQ(sym->type->kind, TK_INT);
    ASSERT(sym->mutable);
}

static void test_scope_nested(void) {
    Scope *outer = scope_create(NULL);
    scope_define(outer, "x", &TYPE_INT, true);
    Scope *inner = scope_create(outer);
    scope_define(inner, "y", &TYPE_STRING, false);

    /* inner can see both */
    ASSERT_NOT_NULL(scope_lookup(inner, "x"));
    ASSERT_NOT_NULL(scope_lookup(inner, "y"));

    /* outer can only see x */
    ASSERT_NOT_NULL(scope_lookup(outer, "x"));
    ASSERT(scope_lookup(outer, "y") == NULL);
}

static void test_scope_shadow(void) {
    Scope *outer = scope_create(NULL);
    scope_define(outer, "x", &TYPE_INT, true);
    Scope *inner = scope_create(outer);
    scope_define(inner, "x", &TYPE_STRING, false);

    /* inner sees the shadowed string version */
    Symbol *sym = scope_lookup_local(inner, "x");
    ASSERT_NOT_NULL(sym);
    ASSERT_EQ(sym->type->kind, TK_STRING);
}

static void test_scope_undefined(void) {
    Scope *s = scope_create(NULL);
    ASSERT(scope_lookup(s, "nonexistent") == NULL);
}

/* --- Type Constructors --- */

static void test_type_from_name_primitives(void) {
    ASSERT_EQ(type_from_name("int")->kind, TK_INT);
    ASSERT_EQ(type_from_name("i8")->kind, TK_INT);
    ASSERT_EQ(type_from_name("i16")->kind, TK_INT);
    ASSERT_EQ(type_from_name("i32")->kind, TK_INT);
    ASSERT_EQ(type_from_name("i64")->kind, TK_INT);
    ASSERT_EQ(type_from_name("u8")->kind, TK_INT);
    ASSERT_EQ(type_from_name("u16")->kind, TK_INT);
    ASSERT_EQ(type_from_name("u32")->kind, TK_INT);
    ASSERT_EQ(type_from_name("u64")->kind, TK_INT);
    ASSERT_EQ(type_from_name("uint")->kind, TK_INT);
    ASSERT_EQ(type_from_name("float")->kind, TK_FLOAT);
    ASSERT_EQ(type_from_name("f32")->kind, TK_FLOAT);
    ASSERT_EQ(type_from_name("f64")->kind, TK_FLOAT);
    ASSERT_EQ(type_from_name("bool")->kind, TK_BOOL);
    ASSERT_EQ(type_from_name("char")->kind, TK_CHAR);
    ASSERT_EQ(type_from_name("byte")->kind, TK_BYTE);
    ASSERT_EQ(type_from_name("string")->kind, TK_STRING);
    ASSERT_EQ(type_from_name("void")->kind, TK_VOID);
    ASSERT_EQ(type_from_name("nil")->kind, TK_NIL);
}

static void test_type_from_name_array(void) {
    EzType *t = type_from_name("[int]");
    ASSERT_EQ(t->kind, TK_ARRAY);
    ASSERT_STR_EQ(t->element_type, "int");
}

static void test_type_from_name_struct(void) {
    EzType *t = type_from_name("Person");
    ASSERT_EQ(t->kind, TK_STRUCT);
    ASSERT_STR_EQ(t->name, "Person");
}

static void test_type_is_numeric(void) {
    ASSERT(type_is_numeric(&TYPE_INT));
    ASSERT(type_is_numeric(&TYPE_FLOAT));
    ASSERT(type_is_numeric(&TYPE_CHAR));
    ASSERT(type_is_numeric(&TYPE_BYTE));
    ASSERT(!type_is_numeric(&TYPE_STRING));
    ASSERT(!type_is_numeric(&TYPE_BOOL));
}

/* --- Expression Type Resolution --- */

static void test_resolve_int_literal(void) {
    EzType *t = expr_type("42");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_float_literal(void) {
    EzType *t = expr_type("3.14");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

static void test_resolve_string_literal(void) {
    EzType *t = expr_type("\"hello\"");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_STRING);
}

static void test_resolve_bool_literal(void) {
    EzType *t = expr_type("true");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_nil(void) {
    EzType *t = expr_type("nil");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_NIL);
}

static void test_resolve_arithmetic(void) {
    EzType *t = expr_type("1 + 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_float_arithmetic(void) {
    EzType *t = expr_type("1.0 + 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

static void test_resolve_comparison(void) {
    EzType *t = expr_type("1 < 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_logical(void) {
    EzType *t = expr_type("true && false");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_negation(void) {
    EzType *t = expr_type("-42");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_not(void) {
    EzType *t = expr_type("!true");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_array_literal(void) {
    EzType *t = expr_type("{1, 2, 3}");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_ARRAY);
}

/* --- Variable Type Resolution --- */

static void test_resolve_typed_var(void) {
    TypeTable *tt = check(
        "do main() {\n"
        "    temp x int = 42\n"
        "    temp y string = \"hello\"\n"
        "}");
    (void)tt;
    /* If it doesn't crash, the scope correctly handled the declarations */
    ASSERT_NOT_NULL(tt);
}

/* --- Builtin Function Types --- */

static void test_resolve_len(void) {
    EzType *t = expr_type("len(\"hello\")");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_typeof(void) {
    EzType *t = expr_type("typeof(42)");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_STRING);
}

static void test_resolve_to_float(void) {
    EzType *t = expr_type("to_float(42)");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

/* --- Struct Type Resolution --- */

static void test_resolve_struct_field(void) {
    TypeTable *tt = check(
        "const Person struct {\n"
        "    name string\n"
        "    age int\n"
        "}\n"
        "do main() {\n"
        "    temp p Person = Person{name: \"Alice\", age: 30}\n"
        "    temp n = p.name\n"
        "}");
    (void)tt;
    ASSERT_NOT_NULL(tt);
}

/* --- Function Return Type Resolution --- */

static void test_resolve_func_return(void) {
    TypeTable *tt = check(
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() {\n"
        "    temp result = add(1, 2)\n"
        "}");
    (void)tt;
    ASSERT_NOT_NULL(tt);
}

/* --- Pointer Type Tests --- */

static void test_type_from_name_pointer(void) {
    EzType *t = type_from_name("^int");
    ASSERT_EQ(t->kind, TK_POINTER);
    ASSERT_STR_EQ(t->element_type, "int");
}

static void test_type_pointer_constructor(void) {
    EzType *t = type_pointer("Person");
    ASSERT_EQ(t->kind, TK_POINTER);
    ASSERT_STR_EQ(t->element_type, "Person");
}

static void test_resolve_addr(void) {
    EzType *t = expr_type("addr(42)");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_POINTER);
}

/* Helper: parse and typecheck, return diagnostics */
static DiagnosticList *check_diag(const char *input) {
    DiagnosticList *d = diag_create();
    d->use_color = false;
    Lexer *l = lexer_create(arena, input, "test.ez");
    Parser *p = parser_create(arena, l, "test.ez", d);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(d, "test.ez");
    typechecker_check(tc, prog);
    return d;
}

static void test_error_type_mismatch(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x string = 42 }");
    ASSERT(diag_has_errors(d));
    diag_destroy(d);
}

static void test_error_wrong_arg_count(void) {
    DiagnosticList *d = check_diag(
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() { add(1, 2, 3) }");
    ASSERT(diag_has_errors(d));
    diag_destroy(d);
}

static void test_error_deref_non_pointer(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 42\n mut y = x^ }");
    ASSERT(diag_has_errors(d));
    diag_destroy(d);
}

static void test_resolve_map_type(void) {
    EzType *t = expr_type("{\"a\": 1}");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_MAP);
}

static void test_resolve_string_enum(void) {
    TypeTable *tt = check(
        "const Status enum { TODO = \"todo\" DONE = \"done\" }\n"
        "do main() { mut s = Status.TODO }");
    (void)tt;
    /* Should not crash — string enum member resolves correctly */
}

static void test_type_from_name_map(void) {
    EzType *t = type_from_name("map[string:int]");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_MAP);
}

int main(void) {
    arena = arena_create(256 * 1024);
    printf("\n");

    /* Scope tests */
    RUN_TEST(test_scope_define_lookup);
    RUN_TEST(test_scope_nested);
    RUN_TEST(test_scope_shadow);
    RUN_TEST(test_scope_undefined);

    /* Type constructor tests */
    RUN_TEST(test_type_from_name_primitives);
    RUN_TEST(test_type_from_name_array);
    RUN_TEST(test_type_from_name_struct);
    RUN_TEST(test_type_is_numeric);

    /* Expression resolution tests */
    RUN_TEST(test_resolve_int_literal);
    RUN_TEST(test_resolve_float_literal);
    RUN_TEST(test_resolve_string_literal);
    RUN_TEST(test_resolve_bool_literal);
    RUN_TEST(test_resolve_nil);
    RUN_TEST(test_resolve_arithmetic);
    RUN_TEST(test_resolve_float_arithmetic);
    RUN_TEST(test_resolve_comparison);
    RUN_TEST(test_resolve_logical);
    RUN_TEST(test_resolve_negation);
    RUN_TEST(test_resolve_not);
    RUN_TEST(test_resolve_array_literal);

    /* Variable tests */
    RUN_TEST(test_resolve_typed_var);

    /* Builtin tests */
    RUN_TEST(test_resolve_len);
    RUN_TEST(test_resolve_typeof);
    RUN_TEST(test_resolve_to_float);

    /* Struct tests */
    RUN_TEST(test_resolve_struct_field);

    /* Function tests */
    RUN_TEST(test_resolve_func_return);

    /* Pointer tests */
    RUN_TEST(test_type_from_name_pointer);
    RUN_TEST(test_type_pointer_constructor);
    RUN_TEST(test_resolve_addr);

    /* Error detection tests */
    RUN_TEST(test_error_type_mismatch);
    RUN_TEST(test_error_wrong_arg_count);
    RUN_TEST(test_error_deref_non_pointer);

    /* Map/enum type tests */
    RUN_TEST(test_resolve_map_type);
    RUN_TEST(test_resolve_string_enum);
    RUN_TEST(test_type_from_name_map);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
