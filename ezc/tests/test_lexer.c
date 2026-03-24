/*
 * test_lexer.c - Unit tests for the EZC lexer
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include "../src/util/arena.h"
#include "../src/lexer/lexer.h"

static Arena *arena;

static Lexer *lex(const char *input) {
    return lexer_create(arena, input, "test.ez");
}

static Token next(Lexer *l) {
    return lexer_next_token(l);
}

static void test_empty_input(void) {
    Lexer *l = lex("");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_EOF);
}

static void test_single_char_tokens(void) {
    Lexer *l = lex("(){}[],.:@");
    ASSERT_EQ(next(l).type, TOK_LPAREN);
    ASSERT_EQ(next(l).type, TOK_RPAREN);
    ASSERT_EQ(next(l).type, TOK_LBRACE);
    ASSERT_EQ(next(l).type, TOK_RBRACE);
    ASSERT_EQ(next(l).type, TOK_LBRACKET);
    ASSERT_EQ(next(l).type, TOK_RBRACKET);
    ASSERT_EQ(next(l).type, TOK_COMMA);
    ASSERT_EQ(next(l).type, TOK_DOT);
    ASSERT_EQ(next(l).type, TOK_COLON);
    ASSERT_EQ(next(l).type, TOK_AT);
    ASSERT_EQ(next(l).type, TOK_EOF);
}

static void test_two_char_tokens(void) {
    Lexer *l = lex("== != <= >= += -= *= /= ++ -- -> && ||");
    ASSERT_EQ(next(l).type, TOK_EQ);
    ASSERT_EQ(next(l).type, TOK_NOT_EQ);
    ASSERT_EQ(next(l).type, TOK_LT_EQ);
    ASSERT_EQ(next(l).type, TOK_GT_EQ);
    ASSERT_EQ(next(l).type, TOK_PLUS_ASSIGN);
    ASSERT_EQ(next(l).type, TOK_MINUS_ASSIGN);
    ASSERT_EQ(next(l).type, TOK_ASTERISK_ASSIGN);
    ASSERT_EQ(next(l).type, TOK_SLASH_ASSIGN);
    ASSERT_EQ(next(l).type, TOK_INCREMENT);
    ASSERT_EQ(next(l).type, TOK_DECREMENT);
    ASSERT_EQ(next(l).type, TOK_ARROW);
    ASSERT_EQ(next(l).type, TOK_AND);
    ASSERT_EQ(next(l).type, TOK_OR);
}

static void test_integer_literal(void) {
    Lexer *l = lex("42");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "42");
}

static void test_integer_with_separators(void) {
    Lexer *l = lex("1_000_000");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "1_000_000");
}

static void test_float_literal(void) {
    Lexer *l = lex("3.14");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_FLOAT);
    ASSERT_STR_EQ(t.literal, "3.14");
}

static void test_string_literal(void) {
    Lexer *l = lex("\"hello world\"");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_STRING);
    ASSERT_STR_EQ(t.literal, "hello world");
}

static void test_string_with_escapes(void) {
    Lexer *l = lex("\"hello\\nworld\"");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_STRING);
    ASSERT_STR_EQ(t.literal, "hello\\nworld");
}

static void test_char_literal(void) {
    Lexer *l = lex("'A'");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_CHAR);
    ASSERT_STR_EQ(t.literal, "A");
}

static void test_raw_string(void) {
    Lexer *l = lex("`raw string`");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_RAW_STRING);
    ASSERT_STR_EQ(t.literal, "raw string");
}

static void test_keywords(void) {
    Lexer *l = lex("temp const do return if or otherwise for for_each as_long_as loop break continue");
    ASSERT_EQ(next(l).type, TOK_TEMP);
    ASSERT_EQ(next(l).type, TOK_CONST);
    ASSERT_EQ(next(l).type, TOK_DO);
    ASSERT_EQ(next(l).type, TOK_RETURN);
    ASSERT_EQ(next(l).type, TOK_IF);
    ASSERT_EQ(next(l).type, TOK_OR_KW);
    ASSERT_EQ(next(l).type, TOK_OTHERWISE);
    ASSERT_EQ(next(l).type, TOK_FOR);
    ASSERT_EQ(next(l).type, TOK_FOR_EACH);
    ASSERT_EQ(next(l).type, TOK_AS_LONG_AS);
    ASSERT_EQ(next(l).type, TOK_LOOP);
    ASSERT_EQ(next(l).type, TOK_BREAK);
    ASSERT_EQ(next(l).type, TOK_CONTINUE);
}

static void test_more_keywords(void) {
    Lexer *l = lex("import using struct enum nil new true false when is default cast ensure module private");
    ASSERT_EQ(next(l).type, TOK_IMPORT);
    ASSERT_EQ(next(l).type, TOK_USING);
    ASSERT_EQ(next(l).type, TOK_STRUCT);
    ASSERT_EQ(next(l).type, TOK_ENUM);
    ASSERT_EQ(next(l).type, TOK_NIL);
    ASSERT_EQ(next(l).type, TOK_NEW);
    ASSERT_EQ(next(l).type, TOK_TRUE);
    ASSERT_EQ(next(l).type, TOK_FALSE);
    ASSERT_EQ(next(l).type, TOK_WHEN);
    ASSERT_EQ(next(l).type, TOK_IS);
    ASSERT_EQ(next(l).type, TOK_DEFAULT);
    ASSERT_EQ(next(l).type, TOK_CAST);
    ASSERT_EQ(next(l).type, TOK_ENSURE);
    ASSERT_EQ(next(l).type, TOK_MODULE);
    ASSERT_EQ(next(l).type, TOK_PRIVATE);
}

static void test_identifiers(void) {
    Lexer *l = lex("foo bar_baz count123");
    Token t1 = next(l);
    ASSERT_EQ(t1.type, TOK_IDENT);
    ASSERT_STR_EQ(t1.literal, "foo");
    Token t2 = next(l);
    ASSERT_EQ(t2.type, TOK_IDENT);
    ASSERT_STR_EQ(t2.literal, "bar_baz");
    Token t3 = next(l);
    ASSERT_EQ(t3.type, TOK_IDENT);
    ASSERT_STR_EQ(t3.literal, "count123");
}

static void test_skip_comments(void) {
    Lexer *l = lex("a // line comment\nb");
    Token t1 = next(l);
    ASSERT_EQ(t1.type, TOK_IDENT);
    ASSERT_STR_EQ(t1.literal, "a");
    Token t2 = next(l);
    ASSERT_EQ(t2.type, TOK_IDENT);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_skip_block_comments(void) {
    Lexer *l = lex("a /* block */ b");
    Token t1 = next(l);
    ASSERT_STR_EQ(t1.literal, "a");
    Token t2 = next(l);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_line_tracking(void) {
    Lexer *l = lex("a\nb\nc");
    Token t1 = next(l);
    ASSERT_EQ(t1.line, 1);
    Token t2 = next(l);
    ASSERT_EQ(t2.line, 2);
    Token t3 = next(l);
    ASSERT_EQ(t3.line, 3);
}

static void test_hash_attributes(void) {
    Lexer *l = lex("#suppress #strict #flags #doc");
    ASSERT_EQ(next(l).type, TOK_SUPPRESS);
    ASSERT_EQ(next(l).type, TOK_STRICT);
    ASSERT_EQ(next(l).type, TOK_FLAGS);
    ASSERT_EQ(next(l).type, TOK_DOC);
}

static void test_hello_world_tokens(void) {
    Lexer *l = lex("import @std\nusing std\n\ndo main() {\n    println(\"Hello\")\n}");
    ASSERT_EQ(next(l).type, TOK_IMPORT);
    ASSERT_EQ(next(l).type, TOK_AT);
    ASSERT_EQ(next(l).type, TOK_IDENT);  /* std */
    ASSERT_EQ(next(l).type, TOK_USING);
    ASSERT_EQ(next(l).type, TOK_IDENT);  /* std */
    ASSERT_EQ(next(l).type, TOK_DO);
    ASSERT_EQ(next(l).type, TOK_IDENT);  /* main */
    ASSERT_EQ(next(l).type, TOK_LPAREN);
    ASSERT_EQ(next(l).type, TOK_RPAREN);
    ASSERT_EQ(next(l).type, TOK_LBRACE);
    ASSERT_EQ(next(l).type, TOK_IDENT);  /* println */
    ASSERT_EQ(next(l).type, TOK_LPAREN);
    ASSERT_EQ(next(l).type, TOK_STRING); /* "Hello" */
    ASSERT_EQ(next(l).type, TOK_RPAREN);
    ASSERT_EQ(next(l).type, TOK_RBRACE);
    ASSERT_EQ(next(l).type, TOK_EOF);
}

static void test_v3_keyword_aliases(void) {
    /* mut maps to TOK_TEMP, while maps to TOK_AS_LONG_AS */
    Lexer *l = lex("mut while");
    ASSERT_EQ(next(l).type, TOK_TEMP);
    ASSERT_EQ(next(l).type, TOK_AS_LONG_AS);
}

static void test_v3_keywords_coexist(void) {
    /* old and new keywords produce the same tokens */
    Lexer *l = lex("temp mut as_long_as while");
    Token t1 = next(l);
    Token t2 = next(l);
    ASSERT_EQ(t1.type, t2.type);
    Token t3 = next(l);
    Token t4 = next(l);
    ASSERT_EQ(t3.type, t4.type);
}

static void test_not_in_operator(void) {
    Lexer *l = lex("!in");
    ASSERT_EQ(next(l).type, TOK_NOT_IN);
}

static void test_not_in_vs_bang(void) {
    /* !in should be NOT_IN, !inside should be BANG + IDENT */
    Lexer *l = lex("!in !inside");
    ASSERT_EQ(next(l).type, TOK_NOT_IN);
    ASSERT_EQ(next(l).type, TOK_BANG);
    ASSERT_EQ(next(l).type, TOK_IDENT);
}

static void test_hex_literal(void) {
    Lexer *l = lex("0xFF");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0xFF");
}

static void test_octal_literal(void) {
    Lexer *l = lex("0o77");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0o77");
}

static void test_binary_literal(void) {
    Lexer *l = lex("0b1010");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0b1010");
}

static void test_or_return_keyword(void) {
    Lexer *l = lex("or_return");
    ASSERT_EQ(next(l).type, TOK_OR_RETURN);
}

static void test_caret_token(void) {
    Lexer *l = lex("^int");
    ASSERT_EQ(next(l).type, TOK_CARET);
    ASSERT_EQ(next(l).type, TOK_IDENT);
}

static void test_float_with_underscores(void) {
    Lexer *l = lex("3.141_592_653");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_FLOAT);
    ASSERT_STR_EQ(t.literal, "3.141_592_653");
}

static void test_interpolation_tokens(void) {
    /* Interpolated string starts as a string token */
    Lexer *l = lex("\"hello ${name}\"");
    Token t = next(l);
    ASSERT_EQ(t.type, TOK_STRING);
}

static void test_bang_in_vs_bang(void) {
    /* !in should be NOT_IN, ! alone should be BANG */
    Lexer *l = lex("!in x !y");
    ASSERT_EQ(next(l).type, TOK_NOT_IN);
    ASSERT_EQ(next(l).type, TOK_IDENT); /* x */
    ASSERT_EQ(next(l).type, TOK_BANG);
    ASSERT_EQ(next(l).type, TOK_IDENT); /* y */
}

int main(void) {
    arena = arena_create(64 * 1024);
    printf("\n");
    RUN_TEST(test_empty_input);
    RUN_TEST(test_single_char_tokens);
    RUN_TEST(test_two_char_tokens);
    RUN_TEST(test_integer_literal);
    RUN_TEST(test_integer_with_separators);
    RUN_TEST(test_float_literal);
    RUN_TEST(test_string_literal);
    RUN_TEST(test_string_with_escapes);
    RUN_TEST(test_char_literal);
    RUN_TEST(test_raw_string);
    RUN_TEST(test_keywords);
    RUN_TEST(test_more_keywords);
    RUN_TEST(test_identifiers);
    RUN_TEST(test_skip_comments);
    RUN_TEST(test_skip_block_comments);
    RUN_TEST(test_line_tracking);
    RUN_TEST(test_hash_attributes);
    RUN_TEST(test_hello_world_tokens);
    RUN_TEST(test_v3_keyword_aliases);
    RUN_TEST(test_v3_keywords_coexist);
    RUN_TEST(test_not_in_operator);
    RUN_TEST(test_not_in_vs_bang);
    RUN_TEST(test_hex_literal);
    RUN_TEST(test_octal_literal);
    RUN_TEST(test_binary_literal);
    RUN_TEST(test_or_return_keyword);
    RUN_TEST(test_caret_token);
    RUN_TEST(test_float_with_underscores);
    RUN_TEST(test_interpolation_tokens);
    RUN_TEST(test_bang_in_vs_bang);
    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
