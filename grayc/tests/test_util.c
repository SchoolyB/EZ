/*
 * test_util.c — Unit tests for the arena allocator, growable buffer,
 * and scope/symbol-table utilities.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include "../src/util/arena.h"
#include "../src/util/buf.h"
#include "../src/typechecker/scope.h"
#include "../src/typechecker/types.h"
#include <stdint.h>

/* ===== Arena Tests ===== */

static void test_arena_create(void) {
    Arena *a = arena_create(4096);
    ASSERT_NOT_NULL(a);
    ASSERT_NOT_NULL(a->first);
    ASSERT_EQ(a->default_block_size, 4096);
    arena_destroy(a);
}

static void test_arena_alloc_basic(void) {
    Arena *a = arena_create(4096);
    void *p = arena_alloc(a, 32);
    ASSERT_NOT_NULL(p);
    arena_destroy(a);
}

static void test_arena_alloc_alignment(void) {
    Arena *a = arena_create(4096);
    arena_alloc(a, 1);  /* 1 byte, gets aligned up to 8 */
    void *p2 = arena_alloc(a, 8);
    ASSERT_NOT_NULL(p2);
    ASSERT_EQ((uintptr_t)p2 % 8, 0);
    arena_destroy(a);
}

static void test_arena_alloc_second_block(void) {
    Arena *a = arena_create(64);
    /* Exhaust the first block */
    arena_alloc(a, 64);
    /* This should trigger a new block */
    void *p = arena_alloc(a, 16);
    ASSERT_NOT_NULL(p);
    ASSERT(a->first->next != NULL);
    arena_destroy(a);
}

static void test_arena_alloc_oversized(void) {
    Arena *a = arena_create(64);
    void *p = arena_alloc(a, 256);
    ASSERT_NOT_NULL(p);
    arena_destroy(a);
}

static void test_arena_copy_string(void) {
    Arena *a = arena_create(4096);
    char *s = arena_copy_string(a, "hello world");
    ASSERT_NOT_NULL(s);
    ASSERT_STR_EQ(s, "hello world");
    arena_destroy(a);
}

static void test_arena_copy_string_empty(void) {
    Arena *a = arena_create(4096);
    char *s = arena_copy_string(a, "");
    ASSERT_NOT_NULL(s);
    ASSERT_STR_EQ(s, "");
    ASSERT_EQ(strlen(s), 0);
    arena_destroy(a);
}

static void test_arena_copy_string_with_length(void) {
    Arena *a = arena_create(4096);
    char *s = arena_copy_string_with_length(a, "hello world", 5);
    ASSERT_NOT_NULL(s);
    ASSERT_STR_EQ(s, "hello");
    arena_destroy(a);
}

/* ===== Buf Tests ===== */

static void test_buffer_create(void) {
    Buf b = buffer_create(64);
    ASSERT_EQ(b.len, 0);
    ASSERT_STR_EQ(buffer_to_string(&b), "");
    buffer_destroy(&b);
}

static void test_append_string_to_buffer(void) {
    Buf b = buffer_create(64);
    append_string_to_buffer(&b, "hello");
    ASSERT_EQ(b.len, 5);
    ASSERT_STR_EQ(buffer_to_string(&b), "hello");
    buffer_destroy(&b);
}

static void test_append_bytes_to_buffer(void) {
    Buf b = buffer_create(64);
    append_bytes_to_buffer(&b, "hello", 3);
    ASSERT_EQ(b.len, 3);
    ASSERT_STR_EQ(buffer_to_string(&b), "hel");
    buffer_destroy(&b);
}

static void test_buf_append_growth(void) {
    Buf b = buffer_create(8);
    append_string_to_buffer(&b, "this string is much longer than the initial capacity");
    ASSERT_STR_EQ(buffer_to_string(&b), "this string is much longer than the initial capacity");
    buffer_destroy(&b);
}

static void test_append_format_to_buffer(void) {
    Buf b = buffer_create(64);
    append_format_to_buffer(&b, "%d items", 42);
    ASSERT_STR_EQ(buffer_to_string(&b), "42 items");
    buffer_destroy(&b);
}

static void test_buf_appendf_large(void) {
    Buf b = buffer_create(8);
    append_format_to_buffer(&b, "value=%d, name=%s, flag=%s", 12345, "test_variable", "true");
    ASSERT_STR_EQ(buffer_to_string(&b), "value=12345, name=test_variable, flag=true");
    buffer_destroy(&b);
}

static void test_append_char_to_buffer(void) {
    Buf b = buffer_create(64);
    append_char_to_buffer(&b, 'X');
    ASSERT_EQ(b.len, 1);
    ASSERT_STR_EQ(buffer_to_string(&b), "X");
    buffer_destroy(&b);
}

static void test_buf_append_char_boundary(void) {
    Buf b = buffer_create(4);
    for (int i = 0; i < 20; i++)
        append_char_to_buffer(&b, 'A');
    ASSERT_EQ(b.len, 20);
    /* Verify all chars are 'A' */
    const char *s = buffer_to_string(&b);
    for (int i = 0; i < 20; i++)
        ASSERT_EQ(s[i], 'A');
    buffer_destroy(&b);
}

static void test_buf_append_indent_zero(void) {
    Buf b = buffer_create(64);
    append_indent_to_buffer(&b, 0);
    ASSERT_EQ(b.len, 0);
    ASSERT_STR_EQ(buffer_to_string(&b), "");
    buffer_destroy(&b);
}

static void test_append_indent_to_buffer(void) {
    Buf b = buffer_create(64);
    append_indent_to_buffer(&b, 2);
    ASSERT_EQ(b.len, 8);  /* 2 * 4 spaces */
    ASSERT_STR_EQ(buffer_to_string(&b), "        ");
    buffer_destroy(&b);
}

static void test_buf_append_indent_large(void) {
    Buf b = buffer_create(128);
    append_indent_to_buffer(&b, 20);
    ASSERT_EQ(b.len, 80);  /* 20 * 4 spaces */
    /* Verify all spaces */
    const char *s = buffer_to_string(&b);
    for (int i = 0; i < 80; i++)
        ASSERT_EQ(s[i], ' ');
    buffer_destroy(&b);
}

/* ===== Scope Tests ===== */

static void test_scope_empty_lookup(void) {
    Scope *scope = scope_create(NULL);
    Symbol *sym = scope_lookup(scope, "nonexistent");
    ASSERT(sym == NULL);
    scope_destroy(scope);
}

static void test_scope_define_update(void) {
    Scope *scope = scope_create(NULL);
    GrayType *t_int = type_from_name("int");
    GrayType *t_str = type_from_name("string");
    scope_define(scope, "x", t_int, true);
    scope_define(scope, "x", t_str, false);
    Symbol *sym = scope_lookup(scope, "x");
    ASSERT_NOT_NULL(sym);
    ASSERT(sym->type == t_str);
    ASSERT_EQ(sym->mutable, false);
    ASSERT_EQ(scope->count, 1);  /* still one symbol, not two */
    scope_destroy(scope);
}

static void test_scope_hash_rebuild(void) {
    Scope *scope = scope_create(NULL);
    GrayType *t = type_from_name("int");
    char names[12][16];
    for (int i = 0; i < 12; i++) {
        snprintf(names[i], sizeof(names[i]), "var_%d", i);
        scope_define(scope, names[i], t, true);
    }
    /* 12 symbols should have triggered at least one hash rebuild (initial hash_cap=16, rebuild when count*2 > hash_cap) */
    ASSERT(scope->hash_cap >= 32);
    /* Verify all symbols are still findable */
    for (int i = 0; i < 12; i++) {
        Symbol *sym = scope_lookup(scope, names[i]);
        ASSERT_NOT_NULL(sym);
    }
    scope_destroy(scope);
}

static void test_scope_many_symbols(void) {
    Scope *scope = scope_create(NULL);
    GrayType *t = type_from_name("int");
    char names[60][16];
    for (int i = 0; i < 60; i++) {
        snprintf(names[i], sizeof(names[i]), "sym_%d", i);
        scope_define(scope, names[i], t, true);
    }
    ASSERT_EQ(scope->count, 60);
    /* Verify all symbols are retrievable */
    for (int i = 0; i < 60; i++) {
        Symbol *sym = scope_lookup(scope, names[i]);
        ASSERT_NOT_NULL(sym);
    }
    scope_destroy(scope);
}

static void test_scope_destroy(void) {
    Scope *scope = scope_create(NULL);
    GrayType *t = type_from_name("int");
    scope_define(scope, "a", t, true);
    scope_define(scope, "b", t, false);
    scope_define(scope, "c", t, true);
    scope_destroy(scope);
    /* If we get here without crashing, the test passes */
    ASSERT(1);
}

static void test_scope_immutable(void) {
    Scope *scope = scope_create(NULL);
    GrayType *t = type_from_name("int");
    scope_define(scope, "constant", t, false);
    Symbol *sym = scope_lookup(scope, "constant");
    ASSERT_NOT_NULL(sym);
    ASSERT_EQ(sym->mutable, false);
    scope_destroy(scope);
}

int main(void) {
    printf("\n");

    printf("--- Arena ---\n");
    RUN_TEST(test_arena_create);
    RUN_TEST(test_arena_alloc_basic);
    RUN_TEST(test_arena_alloc_alignment);
    RUN_TEST(test_arena_alloc_second_block);
    RUN_TEST(test_arena_alloc_oversized);
    RUN_TEST(test_arena_copy_string);
    RUN_TEST(test_arena_copy_string_empty);
    RUN_TEST(test_arena_copy_string_with_length);

    printf("--- Buf ---\n");
    RUN_TEST(test_buffer_create);
    RUN_TEST(test_buf_append_growth);
    RUN_TEST(test_buf_appendf_large);
    RUN_TEST(test_buf_append_char_boundary);
    RUN_TEST(test_buf_append_indent_zero);
    RUN_TEST(test_buf_append_indent_large);

    printf("--- Scope ---\n");
    RUN_TEST(test_scope_empty_lookup);
    RUN_TEST(test_scope_define_update);
    RUN_TEST(test_scope_hash_rebuild);
    RUN_TEST(test_scope_many_symbols);
    RUN_TEST(test_scope_destroy);
    RUN_TEST(test_scope_immutable);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
