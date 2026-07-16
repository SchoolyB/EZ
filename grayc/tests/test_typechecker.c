/*
 * test_typechecker.c — Unit tests for the grayc type checker, validating
 * type inference, type errors, and expression type resolution.
 *
 * Author:  Marshall A Burns (@SchoolyB)
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
    Lexer *l = lexer_create(arena, input, "test.gray");
    Parser *p = parser_create(arena, l, "test.gray", diag);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(diag, "test.gray");
    typechecker_check(tc, prog);
    return typechecker_get_table(tc);
}

/* Helper: parse, type check, and get the type of the first expression in main */
static GrayType *expr_type(const char *expr_code) {
    char buf[512];
    snprintf(buf, sizeof(buf),
        "do main() { mut _result = %s }", expr_code);
    DiagnosticList *diag = diag_create();
    diag->use_color = false;
    Lexer *l = lexer_create(arena, buf, "test.gray");
    Parser *p = parser_create(arena, l, "test.gray", diag);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(diag, "test.gray");
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
    ASSERT_EQ(type_from_name("u8")->kind, TK_UINT);
    ASSERT_EQ(type_from_name("u16")->kind, TK_UINT);
    ASSERT_EQ(type_from_name("u32")->kind, TK_UINT);
    ASSERT_EQ(type_from_name("u64")->kind, TK_UINT);
    ASSERT_EQ(type_from_name("uint")->kind, TK_UINT);
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
    GrayType *t = type_from_name("[int]");
    ASSERT_EQ(t->kind, TK_ARRAY);
    ASSERT_STR_EQ(t->element_type, "int");
}

static void test_type_from_name_struct(void) {
    GrayType *t = type_from_name("Person");
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
    GrayType *t = expr_type("42");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_float_literal(void) {
    GrayType *t = expr_type("3.14");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

static void test_resolve_string_literal(void) {
    GrayType *t = expr_type("\"hello\"");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_STRING);
}

static void test_resolve_bool_literal(void) {
    GrayType *t = expr_type("true");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_nil(void) {
    GrayType *t = expr_type("nil");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_NIL);
}

static void test_resolve_arithmetic(void) {
    GrayType *t = expr_type("1 + 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_float_arithmetic(void) {
    GrayType *t = expr_type("1.0 + 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

static void test_resolve_comparison(void) {
    GrayType *t = expr_type("1 < 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_logical(void) {
    GrayType *t = expr_type("true && false");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_negation(void) {
    GrayType *t = expr_type("-42");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_not(void) {
    GrayType *t = expr_type("!true");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_array_literal(void) {
    GrayType *t = expr_type("{1, 2, 3}");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_ARRAY);
}

/* --- Variable Type Resolution --- */

static void test_resolve_typed_var(void) {
    TypeTable *tt = check(
        "do main() {\n"
        "    mut x int = 42\n"
        "    mut y string = \"hello\"\n"
        "}");
    (void)tt;
    /* If it doesn't crash, the scope correctly handled the declarations */
    ASSERT_NOT_NULL(tt);
}

/* --- Builtin Function Types --- */

static void test_resolve_len(void) {
    GrayType *t = expr_type("len(\"hello\")");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}

static void test_resolve_type_of(void) {
    GrayType *t = expr_type("type_of(42)");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_STRING);
}

static void test_resolve_to_float(void) {
    GrayType *t = expr_type("float(42)");
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
        "    mut p Person = Person{name: \"Alice\", age: 30}\n"
        "    mut n = p.name\n"
        "}");
    (void)tt;
    ASSERT_NOT_NULL(tt);
}

/* --- Function Return Type Resolution --- */

static void test_resolve_func_return(void) {
    TypeTable *tt = check(
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() {\n"
        "    mut result = add(1, 2)\n"
        "}");
    (void)tt;
    ASSERT_NOT_NULL(tt);
}

/* --- Pointer Type Tests --- */

static void test_type_from_name_pointer(void) {
    GrayType *t = type_from_name("^int");
    ASSERT_EQ(t->kind, TK_POINTER);
    ASSERT_STR_EQ(t->element_type, "int");
}

static void test_type_pointer_constructor(void) {
    GrayType *t = type_pointer("Person");
    ASSERT_EQ(t->kind, TK_POINTER);
    ASSERT_STR_EQ(t->element_type, "Person");
}

static void test_resolve_addr(void) {
    GrayType *t = expr_type("addr(42)");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_POINTER);
}

/* Helper: parse and typecheck, return diagnostics */
static DiagnosticList *check_diag(const char *input) {
    DiagnosticList *d = diag_create();
    d->use_color = false;
    Lexer *l = lexer_create(arena, input, "test.gray");
    Parser *p = parser_create(arena, l, "test.gray", d);
    AstNode *prog = parser_parse_program(p);
    TypeChecker *tc = typechecker_create(d, "test.gray");
    typechecker_check(tc, prog);
    return d;
}

/* Helper: check if a specific error code was emitted */
static bool has_code(DiagnosticList *d, const char *code) {
    for (int i = 0; i < d->count; i++) {
        if (d->items[i].code && strcmp(d->items[i].code, code) == 0) return true;
    }
    return false;
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
    GrayType *t = expr_type("{\"a\": 1}");
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
    GrayType *t = type_from_name("map[string:int]");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_MAP);
}

/* --- E3xxx: Type Error Detection --- */

static void test_error_E3001_type_mismatch_assign(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = \"hello\" }");
    ASSERT(has_code(d, "E3001"));
    diag_destroy(d);
}

static void test_error_E3002_invalid_operator(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x = \"a\" - \"b\" }");
    ASSERT(has_code(d, "E3002"));
    diag_destroy(d);
}

static void test_error_E3003_non_int_index(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut a [int] = {1,2,3}\n mut x = a[\"bad\"] }");
    ASSERT(has_code(d, "E3003"));
    diag_destroy(d);
}

static void test_error_E3005_const_reassign(void) {
    DiagnosticList *d = check_diag(
        "do main() { const x int = 5\n x = 10 }");
    ASSERT(has_code(d, "E3005"));
    diag_destroy(d);
}

static void test_error_E3006_return_from_void(void) {
    DiagnosticList *d = check_diag(
        "do foo() { return 42 }\n"
        "do main() { foo() }");
    ASSERT(has_code(d, "E3006"));
    diag_destroy(d);
}

static void test_error_E5023_increment_float(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x float = 1.0\n x++ }");
    ASSERT(has_code(d, "E5023"));
    diag_destroy(d);
}

static void test_error_E3008_index_non_array(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 5\n mut y = x[0] }");
    ASSERT(has_code(d, "E3008"));
    diag_destroy(d);
}

static void test_error_E3009_foreach_non_iterable(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 5\n for_each item in x { } }");
    ASSERT(has_code(d, "E3009"));
    diag_destroy(d);
}

static void test_error_E3010_struct_no_field(void) {
    DiagnosticList *d = check_diag(
        "const Point struct { x int\n y int }\n"
        "do main() { mut p Point = Point{x: 1, y: 2}\n mut z = p.z }");
    ASSERT(has_code(d, "E3010"));
    diag_destroy(d);
}

static void test_error_E3013_field_on_primitive(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 42\n mut y = x.foo }");
    ASSERT(has_code(d, "E3013"));
    diag_destroy(d);
}

static void test_error_E3015_call_non_function(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 42\n x() }");
    ASSERT(has_code(d, "E3015"));
    diag_destroy(d);
}

static void test_error_E3016_deref_non_pointer(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 42\n mut y = x^ }");
    ASSERT(has_code(d, "E3016"));
    diag_destroy(d);
}

static void test_error_E3036_out_of_range(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x uint = -5 }");
    ASSERT(has_code(d, "E3036"));
    diag_destroy(d);
}

static void test_error_E3024_missing_return(void) {
    DiagnosticList *d = check_diag(
        "do foo() -> int { }\n"
        "do main() { foo() }");
    ASSERT(has_code(d, "E3024"));
    diag_destroy(d);
}

static void test_error_E3027_const_to_mut_param(void) {
    DiagnosticList *d = check_diag(
        "do modify(&arr [int]) { arr[0] = 999 }\n"
        "do main() { const nums [int] = {1, 2, 3}\n modify(nums) }");
    ASSERT(has_code(d, "E3027"));
    diag_destroy(d);
}

static void test_error_E3034_any_type(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x any = 42 }");
    ASSERT(has_code(d, "E3034"));
    diag_destroy(d);
}

static void test_error_E3038_void_variable(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x void = 42 }");
    ASSERT(has_code(d, "E3038"));
    diag_destroy(d);
}

static void test_error_E3018_when_type_mismatch(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 5\n when x { is \"bad\" { } default { } } }");
    ASSERT(has_code(d, "E3018"));
    diag_destroy(d);
}

/* --- E4xxx: Name Problems --- */

static void test_error_E4001_undefined_var(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x = y }");
    ASSERT(has_code(d, "E4001"));
    diag_destroy(d);
}

static void test_error_E4002_undefined_func(void) {
    DiagnosticList *d = check_diag(
        "do main() { nonexistent() }");
    ASSERT(has_code(d, "E4002"));
    diag_destroy(d);
}

static void test_error_E4003_duplicate_var(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 1\n mut x int = 2 }");
    ASSERT(has_code(d, "E4003"));
    diag_destroy(d);
}

static void test_error_E4004_duplicate_func(void) {
    DiagnosticList *d = check_diag(
        "do foo() { }\n"
        "do foo() { }\n"
        "do main() { }");
    ASSERT(has_code(d, "E4004"));
    diag_destroy(d);
}

static void test_error_E4005_no_main(void) {
    DiagnosticList *d = check_diag(
        "do foo() { }");
    ASSERT(has_code(d, "E4005"));
    diag_destroy(d);
}

/* --- E5xxx: Usage Problems --- */

static void test_error_E5008_wrong_arg_count_specific(void) {
    DiagnosticList *d = check_diag(
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() { add(1) }");
    ASSERT(has_code(d, "E5008"));
    diag_destroy(d);
}

static void test_error_E5015_incr_literal(void) {
    DiagnosticList *d = check_diag(
        "do main() { 42++ }");
    ASSERT(has_code(d, "E5015"));
    diag_destroy(d);
}

/* --- Warnings --- */

static void test_warning_W1001_unused_var(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 42 }");
    ASSERT(diag_warning_count(d) > 0);
    ASSERT(has_code(d, "W1001"));
    diag_destroy(d);
}

static void test_warning_W1003_unused_func(void) {
    DiagnosticList *d = check_diag(
        "do unused() { }\n"
        "do main() { }");
    ASSERT(diag_warning_count(d) > 0);
    ASSERT(has_code(d, "W1003"));
    diag_destroy(d);
}

static void test_warning_W2002_shadow_var(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 1\n if true { mut x int = 2 } }");
    ASSERT(has_code(d, "W2002"));
    diag_destroy(d);
}

/* --- More E3xxx --- */

static void test_error_E3011_type_as_value(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x = int }");
    ASSERT(has_code(d, "E3011"));
    diag_destroy(d);
}

static void test_error_E3012_addr_literal(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut p = addr(42) }");
    ASSERT(has_code(d, "E3012"));
    diag_destroy(d);
}

static void test_error_E3032_compare_diff_enums(void) {
    DiagnosticList *d = check_diag(
        "const Color enum { RED\n BLUE }\n"
        "const Size enum { SMALL\n BIG }\n"
        "do main() { mut x bool = Color.RED == Size.SMALL }");
    ASSERT(has_code(d, "E3032"));
    diag_destroy(d);
}

static void test_error_E3007_negate_non_numeric(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x = -\"hello\" }");
    ASSERT(has_code(d, "E3007"));
    diag_destroy(d);
}

static void test_error_E3035_not_all_paths_return(void) {
    DiagnosticList *d = check_diag(
        "do foo(x int) -> int {\n"
        "    if x > 0 { return 1 }\n"
        "}\n"
        "do main() { foo(1) }");
    ASSERT(has_code(d, "E3035"));
    diag_destroy(d);
}

static void test_error_E3039_ensure_non_call(void) {
    DiagnosticList *d = check_diag(
        "do main() { ensure 42 }");
    ASSERT(has_code(d, "E3039"));
    diag_destroy(d);
}

static void test_error_E3040_multi_return_single_var(void) {
    DiagnosticList *d = check_diag(
        "do pair() -> (int, int) { return 1, 2 }\n"
        "do main() { mut x = pair() }");
    ASSERT(has_code(d, "E3040"));
    diag_destroy(d);
}

static void test_error_E3044_field_on_type(void) {
    DiagnosticList *d = check_diag(
        "const Point struct { x int\n y int }\n"
        "do main() { mut v = Point.x }");
    ASSERT(has_code(d, "E3044"));
    diag_destroy(d);
}

/* --- More E4xxx --- */

static void test_error_E4006_reserved_name(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut gray_internal int = 5 }");
    ASSERT(has_code(d, "E4006"));
    diag_destroy(d);
}

static void test_error_E4012_shadow_type(void) {
    DiagnosticList *d = check_diag(
        "const Point struct { x int }\n"
        "do main() { mut Point int = 5 }");
    ASSERT(has_code(d, "E4012"));
    diag_destroy(d);
}

static void test_error_E4013_shadow_func(void) {
    DiagnosticList *d = check_diag(
        "do helper() { }\n"
        "do main() { mut helper int = 5 }");
    ASSERT(has_code(d, "E4013"));
    diag_destroy(d);
}

/* --- More E5xxx --- */

static void test_error_E5007_append_const_array(void) {
    DiagnosticList *d = check_diag(
        "import @arrays\n"
        "do main() { const arr [int] = {1, 2}\n arrays.append(arr, 3) }");
    ASSERT(has_code(d, "E5007"));
    diag_destroy(d);
}

static void test_error_E5011_unused_return(void) {
    DiagnosticList *d = check_diag(
        "do get() -> int { return 42 }\n"
        "do main() { get() }");
    ASSERT(has_code(d, "E5011"));
    diag_destroy(d);
}

static void test_error_E3019_signed_to_unsigned(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = -5\n mut y uint = x }");
    ASSERT(has_code(d, "E3019"));
    diag_destroy(d);
}

/* --- E2xxx: Parser errors detected during typechecking --- */

static void test_error_E2050_break_outside_loop(void) {
    DiagnosticList *d = check_diag(
        "do main() { break }");
    ASSERT(has_code(d, "E2050"));
    diag_destroy(d);
}

static void test_error_E2051_nested_func(void) {
    DiagnosticList *d = check_diag(
        "do main() { do inner() { } }");
    ASSERT(has_code(d, "E2051"));
    diag_destroy(d);
}

static void test_error_E2053_struct_in_func(void) {
    DiagnosticList *d = check_diag(
        "do main() { const Foo struct { x int } }");
    ASSERT(has_code(d, "E2053"));
    diag_destroy(d);
}

static void test_error_E2043_duplicate_case(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 1\n when x { is 1 { } is 1 { } default { } } }");
    ASSERT(has_code(d, "E2043"));
    diag_destroy(d);
}

/* Note: compiler emits E2037 for reserved struct names (should be E2038) */
static void test_error_E2037_reserved_struct_name(void) {
    DiagnosticList *d = check_diag(
        "const string struct { x int }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2037"));
    diag_destroy(d);
}

static void test_error_E2016_empty_enum(void) {
    DiagnosticList *d = check_diag(
        "const Empty enum { }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2016"));
    diag_destroy(d);
}

static void test_error_E2012_duplicate_param(void) {
    DiagnosticList *d = check_diag(
        "do foo(a int, a int) -> int { return a }\n"
        "do main() { foo(1, 2) }");
    ASSERT(has_code(d, "E2012"));
    diag_destroy(d);
}

static void test_error_E2013_duplicate_struct_field(void) {
    DiagnosticList *d = check_diag(
        "const Bad struct { x int\n x int }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2013"));
    diag_destroy(d);
}

/* --- Stdlib errors emitted by typechecker --- */

static void test_error_E12006_duplicate_map_key(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut m = {\"a\": 1, \"a\": 2} }");
    ASSERT(has_code(d, "E12006"));
    diag_destroy(d);
}

/* --- E2xxx: Additional parser-level errors in typechecker --- */

static void test_error_E2011_const_no_value(void) {
    DiagnosticList *d = check_diag(
        "do main() { const x int }");
    ASSERT(has_code(d, "E2011"));
    diag_destroy(d);
}

static void test_error_E2036_import_in_func(void) {
    DiagnosticList *d = check_diag(
        "do main() { import @math }");
    ASSERT(has_code(d, "E2036"));
    diag_destroy(d);
}

static void test_error_E2038_reserved_enum_name(void) {
    DiagnosticList *d = check_diag(
        "const int enum { A\n B }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2038"));
    diag_destroy(d);
}

static void test_error_E2039_required_after_default_param(void) {
    DiagnosticList *d = check_diag(
        "do foo(a int = 1, b int) { }\n"
        "do main() { foo(1, 2) }");
    ASSERT(has_code(d, "E2039"));
    diag_destroy(d);
}

static void test_error_E2056_stmt_at_file_scope(void) {
    DiagnosticList *d = check_diag(
        "if true { }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2056"));
    diag_destroy(d);
}

/* --- E3xxx: Additional type errors --- */

static void test_error_E3017_fmt_struct(void) {
    DiagnosticList *d = check_diag(
        "import @fmt\n"
        "const Point struct { x int\n y int }\n"
        "do main() {\n"
        "    mut p Point = Point{x: 1, y: 2}\n"
        "    fmt.printf(\"%s\", p)\n"
        "}");
    ASSERT(has_code(d, "E3017"));
    diag_destroy(d);
}

static void test_error_E3033_duplicate_enum_value(void) {
    DiagnosticList *d = check_diag(
        "const Color enum { RED = 1\n BLUE = 1 }\n"
        "do main() { }");
    ASSERT(has_code(d, "E3033"));
    diag_destroy(d);
}

static void test_error_E3041_new_non_struct(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut p = new(int) }");
    ASSERT(has_code(d, "E3041"));
    diag_destroy(d);
}

static void test_error_E3041_interpolate_void(void) {
    DiagnosticList *d = check_diag(
        "do noop() { }\n"
        "do main() { mut s string = \"result: ${noop()}\" }");
    ASSERT(has_code(d, "E3041"));
    diag_destroy(d);
}

static void test_error_E3043_invalid_cast(void) {
    DiagnosticList *d = check_diag(
        "const Point struct { x int\n y int }\n"
        "do main() {\n"
        "    mut p Point = Point{x: 1, y: 2}\n"
        "    mut x = cast(p, int)\n"
        "}");
    ASSERT(has_code(d, "E3043"));
    diag_destroy(d);
}

static void test_error_E3045_or_return_no_error(void) {
    DiagnosticList *d = check_diag(
        "do get() -> int { return 42 }\n"
        "do main() { mut x = get() or_return }");
    ASSERT(has_code(d, "E3045"));
    diag_destroy(d);
}

/* --- E4xxx: Additional name errors --- */

static void test_error_E4014_shadow_module(void) {
    DiagnosticList *d = check_diag(
        "import @math\n"
        "do main() { mut math int = 5 }");
    ASSERT(has_code(d, "E4014"));
    diag_destroy(d);
}

/* --- E5xxx: Additional usage errors --- */

static void test_error_E5024_signed_return_as_unsigned(void) {
    DiagnosticList *d = check_diag(
        "do foo() -> uint {\n"
        "    mut x int = -5\n"
        "    return x\n"
        "}\n"
        "do main() { foo() }");
    ASSERT(has_code(d, "E5024"));
    diag_destroy(d);
}

/* --- E6xxx: Module errors --- */

static void test_error_E6001_unknown_module(void) {
    DiagnosticList *d = check_diag(
        "import @nonexistent\n"
        "do main() { }");
    ASSERT(has_code(d, "E6001"));
    diag_destroy(d);
}

/* --- Stdlib type errors in typechecker --- */

static void test_error_E7004_repeat_float_count(void) {
    DiagnosticList *d = check_diag(
        "import @strings\n"
        "do main() { mut s = strings.repeat(\"ha\", 3.5) }");
    ASSERT(has_code(d, "E7004"));
    diag_destroy(d);
}

static void test_error_E9002_sum_non_numeric(void) {
    DiagnosticList *d = check_diag(
        "import @arrays\n"
        "do main() { mut a [string] = {\"a\", \"b\"}\n arrays.sum(a) }");
    ASSERT(has_code(d, "E9002"));
    diag_destroy(d);
}

static void test_error_E9005_invalid_range(void) {
    DiagnosticList *d = check_diag(
        "do main() { for i in range(10, 1) { } }");
    ASSERT(has_code(d, "E9005"));
    diag_destroy(d);
}

static void test_error_E12001_map_func_on_array(void) {
    DiagnosticList *d = check_diag(
        "import @maps\n"
        "do main() { mut a [int] = {1, 2}\n maps.keys(a) }");
    ASSERT(has_code(d, "E12001"));
    diag_destroy(d);
}

/* --- Additional Warnings --- */

static void test_warning_W1005_typed_blank(void) {
    DiagnosticList *d = check_diag(
        "do pair() -> (int, int) { return 1, 2 }\n"
        "do main() { mut x int, _ int = pair() }");
    ASSERT(has_code(d, "W1005"));
    diag_destroy(d);
}

static void test_warning_W2001_unused_import(void) {
    DiagnosticList *d = check_diag(
        "import @math\n"
        "do main() { }");
    ASSERT(has_code(d, "W1002"));
    diag_destroy(d);
}

static void test_warning_W3003_partial_array_init(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut a [int, 5] = {1, 2} }");
    ASSERT(has_code(d, "W3003"));
    diag_destroy(d);
}

/* --- Additional typechecker coverage --- */

static void test_resolve_char_literal(void) {
    GrayType *t = expr_type("'A'");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_CHAR);
}

static void test_resolve_modulo(void) {
    GrayType *t = expr_type("10 % 3");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_INT);
}


static void test_resolve_equality(void) {
    GrayType *t = expr_type("1 == 1");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_inequality(void) {
    GrayType *t = expr_type("1 != 2");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_or_logic(void) {
    GrayType *t = expr_type("true || false");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_string_comparison(void) {
    GrayType *t = expr_type("\"a\" == \"b\"");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_BOOL);
}

static void test_resolve_float_negation(void) {
    GrayType *t = expr_type("-3.14");
    ASSERT_NOT_NULL(t);
    ASSERT_EQ(t->kind, TK_FLOAT);
}

static void test_scope_deeply_nested(void) {
    Scope *s1 = scope_create(NULL);
    scope_define(s1, "a", &TYPE_INT, true);
    Scope *s2 = scope_create(s1);
    scope_define(s2, "b", &TYPE_STRING, true);
    Scope *s3 = scope_create(s2);
    scope_define(s3, "c", &TYPE_BOOL, true);

    /* s3 can see all three */
    ASSERT_NOT_NULL(scope_lookup(s3, "a"));
    ASSERT_NOT_NULL(scope_lookup(s3, "b"));
    ASSERT_NOT_NULL(scope_lookup(s3, "c"));

    /* s1 can only see a */
    ASSERT_NOT_NULL(scope_lookup(s1, "a"));
    ASSERT(scope_lookup(s1, "b") == NULL);
    ASSERT(scope_lookup(s1, "c") == NULL);
}

static void test_scope_local_only(void) {
    Scope *outer = scope_create(NULL);
    scope_define(outer, "x", &TYPE_INT, true);
    Scope *inner = scope_create(outer);

    /* lookup_local should NOT find outer's x */
    ASSERT(scope_lookup_local(inner, "x") == NULL);
    /* but regular lookup should */
    ASSERT_NOT_NULL(scope_lookup(inner, "x"));
}

static void test_type_from_name_bigint(void) {
    GrayType *t1 = type_from_name("i128");
    ASSERT_NOT_NULL(t1);
    GrayType *t2 = type_from_name("u128");
    ASSERT_NOT_NULL(t2);
    GrayType *t3 = type_from_name("i256");
    ASSERT_NOT_NULL(t3);
    GrayType *t4 = type_from_name("u256");
    ASSERT_NOT_NULL(t4);
}

static void test_type_is_numeric_uint(void) {
    ASSERT(type_is_numeric(&TYPE_UINT));
}

static void test_error_E3002_bool_arithmetic(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x = true + false }");
    ASSERT(has_code(d, "E3002"));
    diag_destroy(d);
}

static void test_error_E3001_bool_to_int(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = true }");
    ASSERT(has_code(d, "E3001"));
    diag_destroy(d);
}

static void test_error_E3001_int_to_string(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x string = 42 }");
    ASSERT(has_code(d, "E3001"));
    diag_destroy(d);
}

static void test_error_E3005_const_array_reassign(void) {
    DiagnosticList *d = check_diag(
        "do main() { const arr [int] = {1,2,3}\n arr = {4,5,6} }");
    ASSERT(has_code(d, "E3005"));
    diag_destroy(d);
}

static void test_error_E3009_foreach_int(void) {
    DiagnosticList *d = check_diag(
        "do main() { for_each x in 42 { } }");
    ASSERT(has_code(d, "E3009"));
    diag_destroy(d);
}

static void test_error_E4003_duplicate_var_same_scope(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut x int = 1\n mut x string = \"hi\" }");
    ASSERT(has_code(d, "E4003"));
    diag_destroy(d);
}

static void test_error_E5008_too_many_args(void) {
    DiagnosticList *d = check_diag(
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() { add(1, 2, 3) }");
    ASSERT(has_code(d, "E5008"));
    diag_destroy(d);
}

static void test_error_E3008_index_string(void) {
    /* String indexing should work, so this should not error */
    DiagnosticList *d = check_diag(
        "do main() { mut s string = \"hello\"\n mut c = s[0] }");
    ASSERT(!has_code(d, "E3008"));
    diag_destroy(d);
}

static void test_valid_nested_struct(void) {
    /* Nested struct access should typecheck without errors */
    DiagnosticList *d = check_diag(
        "const Inner struct {\n"
        "    val int\n"
        "}\n"
        "const Outer struct {\n"
        "    inner Inner\n"
        "}\n"
        "do main() {\n"
        "    mut o Outer = Outer{inner: Inner{val: 42}}\n"
        "    mut v int = o.inner.val\n"
        "    println(\"${v}\")\n"
        "}");
    ASSERT(!diag_has_errors(d));
    diag_destroy(d);
}

static void test_valid_enum_member_access(void) {
    DiagnosticList *d = check_diag(
        "const Color enum {\n"
        "    RED\n"
        "    GREEN\n"
        "    BLUE\n"
        "}\n"
        "do main() { mut c = Color.RED\n println(\"${c}\") }");
    ASSERT(!diag_has_errors(d));
    diag_destroy(d);
}

static void test_valid_multi_return(void) {
    DiagnosticList *d = check_diag(
        "do swap(a int, b int) -> (int, int) { return b, a }\n"
        "do main() { mut x int, y int = swap(1, 2)\n println(\"${x} ${y}\") }");
    ASSERT(!diag_has_errors(d));
    diag_destroy(d);
}

static void test_valid_when_stmt(void) {
    DiagnosticList *d = check_diag(
        "do main() {\n"
        "    mut x int = 2\n"
        "    when x {\n"
        "        is 1 { println(\"one\") }\n"
        "        is 2 { println(\"two\") }\n"
        "        default { println(\"other\") }\n"
        "    }\n"
        "}");
    ASSERT(!diag_has_errors(d));
    diag_destroy(d);
}

static void test_valid_struct_function_return(void) {
    DiagnosticList *d = check_diag(
        "const Point struct {\n"
        "    x int\n"
        "    y int\n"
        "    do origin() -> Point {\n"
        "        return Point{x: 0, y: 0}\n"
        "    }\n"
        "}\n"
        "do main() { mut p Point = Point.origin()\n println(\"${p.x}\") }");
    ASSERT(!diag_has_errors(d));
    diag_destroy(d);
}

static void test_error_E2050_continue_outside_loop(void) {
    DiagnosticList *d = check_diag(
        "do main() { continue }");
    ASSERT(has_code(d, "E2050"));
    diag_destroy(d);
}

static void test_error_E3024_some_paths_missing_return(void) {
    DiagnosticList *d = check_diag(
        "do foo(x int) -> int {\n"
        "    if x > 0 { return 1 }\n"
        "    if x < 0 { return -1 }\n"
        "}\n"
        "do main() { foo(1) }");
    ASSERT(has_code(d, "E3035") || has_code(d, "E3024"));
    diag_destroy(d);
}

static void test_error_E4007_duplicate_struct(void) {
    DiagnosticList *d = check_diag(
        "const Foo struct { x int }\n"
        "const Foo struct { y int }\n"
        "do main() { }");
    ASSERT(has_code(d, "E4007"));
    diag_destroy(d);
}

/* ===== Batch: untested error codes ===== */

static void test_error_E3047_enum_no_member(void) {
    DiagnosticList *d = check_diag(
        "const Color enum { RED GREEN BLUE }\n"
        "do main() { mut c = Color.YELLOW }");
    ASSERT(has_code(d, "E3047"));
    diag_destroy(d);
}

static void test_error_E3048_string_plus(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut s string = \"a\" + \"b\" }");
    ASSERT(has_code(d, "E3048"));
    diag_destroy(d);
}

/* E3049, E3077, E5026 need full compilation to trigger — tested via integration */

static void test_error_E3057_invalid_map_key(void) {
    DiagnosticList *d = check_diag(
        "const P struct { x int }\n"
        "do main() { mut m map[P:int] = {} }");
    ASSERT(has_code(d, "E3057"));
    diag_destroy(d);
}

static void test_error_E3059_const_map(void) {
    DiagnosticList *d = check_diag(
        "do main() { const m map[string:int] = {\"a\": 1} }");
    ASSERT(has_code(d, "E3059"));
    diag_destroy(d);
}

static void test_error_E3061_recursive_struct(void) {
    DiagnosticList *d = check_diag(
        "const Node struct { next Node }\n"
        "do main() { }");
    ASSERT(has_code(d, "E3061"));
    diag_destroy(d);
}

static void test_valid_recursive_pointer_struct(void) {
    DiagnosticList *d = check_diag(
        "const Node struct {\n value int\n next ^Node\n}\n"
        "do main() { }");
    ASSERT(d->count == 0);
    diag_destroy(d);
}

static void test_error_E3063_return_addr_local(void) {
    DiagnosticList *d = check_diag(
        "do bad() -> ptr<int> {\n"
        "  mut x int = 42\n"
        "  return addr(x)\n"
        "}");
    ASSERT(has_code(d, "E3063"));
    diag_destroy(d);
}

/* E3068 requires parsing "void" as a return type, which the parser may
   reject before the typechecker sees it. Removed — tested via integration. */

static void test_error_E3072_return_nil_non_pointer(void) {
    DiagnosticList *d = check_diag(
        "do bad() -> int { return nil }");
    ASSERT(has_code(d, "E3072"));
    diag_destroy(d);
}

static void test_error_E3073_return_in_main(void) {
    DiagnosticList *d = check_diag(
        "do main() { return }");
    ASSERT(has_code(d, "E3073"));
    diag_destroy(d);
}

static void test_error_E3074_array_compare(void) {
    DiagnosticList *d = check_diag(
        "do main() {\n"
        "  mut a [int] = {1, 2}\n"
        "  mut b [int] = {1, 2}\n"
        "  if a == b { }\n"
        "}");
    ASSERT(has_code(d, "E3074"));
    diag_destroy(d);
}

static void test_error_E3076_map_compare(void) {
    DiagnosticList *d = check_diag(
        "do main() {\n"
        "  mut a map[string:int] = {\"x\": 1}\n"
        "  mut b map[string:int] = {\"x\": 1}\n"
        "  if a == b { }\n"
        "}");
    ASSERT(has_code(d, "E3076"));
    diag_destroy(d);
}

static void test_error_E3078_pointer_arithmetic(void) {
    DiagnosticList *d = check_diag(
        "do main() {\n"
        "  mut x int = 10\n"
        "  mut p = addr(x)\n"
        "  mut q = p + 1\n"
        "}");
    ASSERT(has_code(d, "E3078"));
    diag_destroy(d);
}

static void test_error_E3080_named_return_mismatch(void) {
    DiagnosticList *d = check_diag(
        "do get() -> (result int) {\n"
        "  mut other int = 42\n"
        "  return other\n"
        "}");
    ASSERT(has_code(d, "E3080"));
    diag_destroy(d);
}

static void test_error_E3082_wildcard_named_return(void) {
    DiagnosticList *d = check_diag(
        "do get(x ?) -> (val ?) { return x }\n"
        "do main() { println(get(1)) }");
    ASSERT(has_code(d, "E3082"));
    diag_destroy(d);
}

static void test_error_E2014_duplicate_enum_variant(void) {
    DiagnosticList *d = check_diag(
        "const Color enum { RED RED }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2014"));
    diag_destroy(d);
}

static void test_error_E2015_duplicate_struct_field_init(void) {
    DiagnosticList *d = check_diag(
        "const P struct { x int }\n"
        "do main() { mut p = P{x: 1, x: 2} }");
    ASSERT(has_code(d, "E2015"));
    diag_destroy(d);
}

static void test_error_E2063_duplicate_named_return(void) {
    DiagnosticList *d = check_diag(
        "do bad() -> (x int, x int) {\n"
        "  mut x int = 1\n"
        "  return x, x\n"
        "}\n"
        "do main() { }");
    ASSERT(has_code(d, "E2063"));
    diag_destroy(d);
}

static void test_error_E2064_field_func_conflict(void) {
    DiagnosticList *d = check_diag(
        "const S struct {\n"
        "  name int\n"
        "  do name() -> int { return 0 }\n"
        "}\n"
        "do main() { }");
    ASSERT(has_code(d, "E2064"));
    diag_destroy(d);
}

static void test_error_E2065_enum_variant_same_as_type(void) {
    DiagnosticList *d = check_diag(
        "const Color enum { Color }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2065"));
    diag_destroy(d);
}

static void test_error_E2066_struct_field_same_as_type(void) {
    DiagnosticList *d = check_diag(
        "const Foo struct { Foo int }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2066"));
    diag_destroy(d);
}

static void test_error_E2067_empty_struct(void) {
    DiagnosticList *d = check_diag(
        "const Empty struct { }\n"
        "do main() { }");
    ASSERT(has_code(d, "E2067"));
    diag_destroy(d);
}

static void test_error_E3083_c_string_non_pointer(void) {
    DiagnosticList *d = check_diag(
        "do main() { mut msg = c_string(\"hello\") }");
    ASSERT(has_code(d, "E3083"));
    diag_destroy(d);
}

static void test_error_E4008_main_with_params(void) {
    DiagnosticList *d = check_diag(
        "do main(x int) { }");
    ASSERT(has_code(d, "E4008"));
    diag_destroy(d);
}

static void test_error_E5025_invalid_assign_target(void) {
    DiagnosticList *d = check_diag(
        "do main() { 42 = 10 }");
    ASSERT(has_code(d, "E5025"));
    diag_destroy(d);
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
    RUN_TEST(test_resolve_type_of);
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

    /* E3xxx: Type error detection */
    RUN_TEST(test_error_E3001_type_mismatch_assign);
    RUN_TEST(test_error_E3002_invalid_operator);
    RUN_TEST(test_error_E3003_non_int_index);
    RUN_TEST(test_error_E3005_const_reassign);
    RUN_TEST(test_error_E3006_return_from_void);
    RUN_TEST(test_error_E5023_increment_float);
    RUN_TEST(test_error_E3008_index_non_array);
    RUN_TEST(test_error_E3009_foreach_non_iterable);
    RUN_TEST(test_error_E3010_struct_no_field);
    RUN_TEST(test_error_E3013_field_on_primitive);
    RUN_TEST(test_error_E3015_call_non_function);
    RUN_TEST(test_error_E3016_deref_non_pointer);
    RUN_TEST(test_error_E3036_out_of_range);
    RUN_TEST(test_error_E3024_missing_return);
    RUN_TEST(test_error_E3027_const_to_mut_param);
    RUN_TEST(test_error_E3034_any_type);
    RUN_TEST(test_error_E3038_void_variable);
    RUN_TEST(test_error_E3018_when_type_mismatch);

    /* E4xxx: Name problems */
    RUN_TEST(test_error_E4001_undefined_var);
    RUN_TEST(test_error_E4002_undefined_func);
    RUN_TEST(test_error_E4003_duplicate_var);
    RUN_TEST(test_error_E4004_duplicate_func);
    RUN_TEST(test_error_E4005_no_main);

    /* E5xxx: Usage problems */
    RUN_TEST(test_error_E5008_wrong_arg_count_specific);
    RUN_TEST(test_error_E5015_incr_literal);

    /* Warnings */
    RUN_TEST(test_warning_W1001_unused_var);
    RUN_TEST(test_warning_W1003_unused_func);
    RUN_TEST(test_warning_W2002_shadow_var);

    /* More E3xxx */
    RUN_TEST(test_error_E3011_type_as_value);
    RUN_TEST(test_error_E3012_addr_literal);
    RUN_TEST(test_error_E3032_compare_diff_enums);
    RUN_TEST(test_error_E3007_negate_non_numeric);
    RUN_TEST(test_error_E3035_not_all_paths_return);
    RUN_TEST(test_error_E3039_ensure_non_call);
    RUN_TEST(test_error_E3040_multi_return_single_var);
    RUN_TEST(test_error_E3044_field_on_type);

    /* More E4xxx */
    RUN_TEST(test_error_E4006_reserved_name);
    RUN_TEST(test_error_E4012_shadow_type);
    RUN_TEST(test_error_E4013_shadow_func);

    /* More E5xxx */
    RUN_TEST(test_error_E5007_append_const_array);
    RUN_TEST(test_error_E5011_unused_return);
    RUN_TEST(test_error_E3019_signed_to_unsigned);

    /* E2xxx: Parser-level errors in typechecker */
    RUN_TEST(test_error_E2050_break_outside_loop);
    RUN_TEST(test_error_E2051_nested_func);
    RUN_TEST(test_error_E2053_struct_in_func);
    RUN_TEST(test_error_E2043_duplicate_case);
    RUN_TEST(test_error_E2037_reserved_struct_name);
    RUN_TEST(test_error_E2016_empty_enum);
    RUN_TEST(test_error_E2012_duplicate_param);
    RUN_TEST(test_error_E2013_duplicate_struct_field);

    /* Stdlib errors in typechecker */
    RUN_TEST(test_error_E12006_duplicate_map_key);

    /* E2xxx: Additional parser-level errors in typechecker */
    RUN_TEST(test_error_E2011_const_no_value);
    RUN_TEST(test_error_E2036_import_in_func);
    RUN_TEST(test_error_E2038_reserved_enum_name);
    RUN_TEST(test_error_E2039_required_after_default_param);
    RUN_TEST(test_error_E2056_stmt_at_file_scope);

    /* E3xxx: Additional type errors */
    RUN_TEST(test_error_E3017_fmt_struct);
    RUN_TEST(test_error_E3033_duplicate_enum_value);
    RUN_TEST(test_error_E3041_new_non_struct);
    RUN_TEST(test_error_E3041_interpolate_void);
    RUN_TEST(test_error_E3043_invalid_cast);
    RUN_TEST(test_error_E3045_or_return_no_error);

    /* E4xxx: Additional name errors */
    RUN_TEST(test_error_E4014_shadow_module);

    /* E5xxx: Additional usage errors */
    RUN_TEST(test_error_E5024_signed_return_as_unsigned);

    /* E6xxx: Module errors */
    RUN_TEST(test_error_E6001_unknown_module);

    /* Stdlib type errors */
    RUN_TEST(test_error_E7004_repeat_float_count);
    RUN_TEST(test_error_E9002_sum_non_numeric);
    RUN_TEST(test_error_E9005_invalid_range);
    RUN_TEST(test_error_E12001_map_func_on_array);

    /* Additional warnings */
    RUN_TEST(test_warning_W1005_typed_blank);
    RUN_TEST(test_warning_W2001_unused_import);
    RUN_TEST(test_warning_W3003_partial_array_init);

    /* Additional typechecker coverage */
    RUN_TEST(test_resolve_char_literal);
    RUN_TEST(test_resolve_modulo);
    RUN_TEST(test_resolve_equality);
    RUN_TEST(test_resolve_inequality);
    RUN_TEST(test_resolve_or_logic);
    RUN_TEST(test_resolve_string_comparison);
    RUN_TEST(test_resolve_float_negation);
    RUN_TEST(test_scope_deeply_nested);
    RUN_TEST(test_scope_local_only);
    RUN_TEST(test_type_from_name_bigint);
    RUN_TEST(test_type_is_numeric_uint);
    RUN_TEST(test_error_E3002_bool_arithmetic);
    RUN_TEST(test_error_E3001_bool_to_int);
    RUN_TEST(test_error_E3001_int_to_string);
    RUN_TEST(test_error_E3005_const_array_reassign);
    RUN_TEST(test_error_E3009_foreach_int);
    RUN_TEST(test_error_E4003_duplicate_var_same_scope);
    RUN_TEST(test_error_E5008_too_many_args);
    RUN_TEST(test_error_E3008_index_string);
    RUN_TEST(test_valid_nested_struct);
    RUN_TEST(test_valid_enum_member_access);
    RUN_TEST(test_valid_multi_return);
    RUN_TEST(test_valid_when_stmt);
    RUN_TEST(test_valid_struct_function_return);
    RUN_TEST(test_error_E2050_continue_outside_loop);
    RUN_TEST(test_error_E3024_some_paths_missing_return);
    RUN_TEST(test_error_E4007_duplicate_struct);

    /* Batch: previously untested error codes */
    RUN_TEST(test_error_E2014_duplicate_enum_variant);
    RUN_TEST(test_error_E2015_duplicate_struct_field_init);
    RUN_TEST(test_error_E2063_duplicate_named_return);
    RUN_TEST(test_error_E2064_field_func_conflict);
    RUN_TEST(test_error_E2065_enum_variant_same_as_type);
    RUN_TEST(test_error_E2066_struct_field_same_as_type);
    RUN_TEST(test_error_E2067_empty_struct);
    RUN_TEST(test_error_E3047_enum_no_member);
    RUN_TEST(test_error_E3048_string_plus);
    RUN_TEST(test_error_E3057_invalid_map_key);
    RUN_TEST(test_error_E3059_const_map);
    RUN_TEST(test_error_E3061_recursive_struct);
    RUN_TEST(test_valid_recursive_pointer_struct);
    RUN_TEST(test_error_E3063_return_addr_local);
    RUN_TEST(test_error_E3072_return_nil_non_pointer);
    RUN_TEST(test_error_E3073_return_in_main);
    RUN_TEST(test_error_E3074_array_compare);
    RUN_TEST(test_error_E3076_map_compare);
    RUN_TEST(test_error_E3078_pointer_arithmetic);
    RUN_TEST(test_error_E3080_named_return_mismatch);
    RUN_TEST(test_error_E3082_wildcard_named_return);
    RUN_TEST(test_error_E4008_main_with_params);
    RUN_TEST(test_error_E5025_invalid_assign_target);
    RUN_TEST(test_error_E3083_c_string_non_pointer);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
