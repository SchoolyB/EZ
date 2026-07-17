/*
 * test_lexer.c — Unit tests for the grayc lexer, covering tokenization
 * of literals, operators, keywords, comments, and edge cases.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include "../src/util/arena.h"
#include "../src/lexer/lexer.h"

static Arena *arena;

static Lexer *lex(const char *input) {
    return lexer_create(arena, input, "test.gray");
}

static Token next(Lexer *lexer) {
    return lexer_next_token(lexer);
}

static void test_empty_input(void) {
    Lexer *lexer = lex("");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_EOF);
}

static void test_single_char_tokens(void) {
    Lexer *lexer = lex("(){}[],.:@");
    ASSERT_EQ(next(lexer).type, TOK_LPAREN);
    ASSERT_EQ(next(lexer).type, TOK_RPAREN);
    ASSERT_EQ(next(lexer).type, TOK_LBRACE);
    ASSERT_EQ(next(lexer).type, TOK_RBRACE);
    ASSERT_EQ(next(lexer).type, TOK_LBRACKET);
    ASSERT_EQ(next(lexer).type, TOK_RBRACKET);
    ASSERT_EQ(next(lexer).type, TOK_COMMA);
    ASSERT_EQ(next(lexer).type, TOK_DOT);
    ASSERT_EQ(next(lexer).type, TOK_COLON);
    ASSERT_EQ(next(lexer).type, TOK_AT);
    ASSERT_EQ(next(lexer).type, TOK_EOF);
}

static void test_two_char_tokens(void) {
    Lexer *lexer = lex("== != <= >= += -= *= /= ++ -- -> && ||");
    ASSERT_EQ(next(lexer).type, TOK_EQ);
    ASSERT_EQ(next(lexer).type, TOK_NOT_EQ);
    ASSERT_EQ(next(lexer).type, TOK_LT_EQ);
    ASSERT_EQ(next(lexer).type, TOK_GT_EQ);
    ASSERT_EQ(next(lexer).type, TOK_PLUS_ASSIGN);
    ASSERT_EQ(next(lexer).type, TOK_MINUS_ASSIGN);
    ASSERT_EQ(next(lexer).type, TOK_ASTERISK_ASSIGN);
    ASSERT_EQ(next(lexer).type, TOK_SLASH_ASSIGN);
    ASSERT_EQ(next(lexer).type, TOK_INCREMENT);
    ASSERT_EQ(next(lexer).type, TOK_DECREMENT);
    ASSERT_EQ(next(lexer).type, TOK_ARROW);
    ASSERT_EQ(next(lexer).type, TOK_AND);
    ASSERT_EQ(next(lexer).type, TOK_OR);
}

static void test_integer_literal(void) {
    Lexer *lexer = lex("42");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "42");
}

static void test_integer_with_separators(void) {
    Lexer *lexer = lex("1_000_000");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "1_000_000");
}

static void test_float_literal(void) {
    Lexer *lexer = lex("3.14");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_FLOAT);
    ASSERT_STR_EQ(t.literal, "3.14");
}

static void test_string_literal(void) {
    Lexer *lexer = lex("\"hello world\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_STRING);
    ASSERT_STR_EQ(t.literal, "hello world");
}

static void test_string_with_escapes(void) {
    Lexer *lexer = lex("\"hello\\nworld\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_STRING);
    ASSERT_STR_EQ(t.literal, "hello\\nworld");
}

static void test_char_literal(void) {
    Lexer *lexer = lex("'A'");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_CHAR);
    ASSERT_STR_EQ(t.literal, "A");
}

static void test_raw_string(void) {
    Lexer *lexer = lex("`raw string`");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_RAW_STRING);
    ASSERT_STR_EQ(t.literal, "raw string");
}

static void test_keywords(void) {
    Lexer *lexer = lex("mut const do return if or otherwise for for_each as_long_as loop break continue");
    ASSERT_EQ(next(lexer).type, TOK_MUT);
    ASSERT_EQ(next(lexer).type, TOK_CONST);
    ASSERT_EQ(next(lexer).type, TOK_DO);
    ASSERT_EQ(next(lexer).type, TOK_RETURN);
    ASSERT_EQ(next(lexer).type, TOK_IF);
    ASSERT_EQ(next(lexer).type, TOK_OR_KW);
    ASSERT_EQ(next(lexer).type, TOK_OTHERWISE);
    ASSERT_EQ(next(lexer).type, TOK_FOR);
    ASSERT_EQ(next(lexer).type, TOK_FOR_EACH);
    ASSERT_EQ(next(lexer).type, TOK_AS_LONG_AS);
    ASSERT_EQ(next(lexer).type, TOK_LOOP);
    ASSERT_EQ(next(lexer).type, TOK_BREAK);
    ASSERT_EQ(next(lexer).type, TOK_CONTINUE);
}

static void test_more_keywords(void) {
    Lexer *lexer = lex("import using struct enum nil new true false when is default cast ensure module private");
    ASSERT_EQ(next(lexer).type, TOK_IMPORT);
    ASSERT_EQ(next(lexer).type, TOK_USING);
    ASSERT_EQ(next(lexer).type, TOK_STRUCT);
    ASSERT_EQ(next(lexer).type, TOK_ENUM);
    ASSERT_EQ(next(lexer).type, TOK_NIL);
    ASSERT_EQ(next(lexer).type, TOK_NEW);
    ASSERT_EQ(next(lexer).type, TOK_TRUE);
    ASSERT_EQ(next(lexer).type, TOK_FALSE);
    ASSERT_EQ(next(lexer).type, TOK_WHEN);
    ASSERT_EQ(next(lexer).type, TOK_IS);
    ASSERT_EQ(next(lexer).type, TOK_DEFAULT);
    ASSERT_EQ(next(lexer).type, TOK_CAST);
    ASSERT_EQ(next(lexer).type, TOK_ENSURE);
    ASSERT_EQ(next(lexer).type, TOK_MODULE);
    ASSERT_EQ(next(lexer).type, TOK_PRIVATE);
}

static void test_identifiers(void) {
    Lexer *lexer = lex("foo bar_baz count123");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.type, TOK_IDENT);
    ASSERT_STR_EQ(t1.literal, "foo");
    Token t2 = next(lexer);
    ASSERT_EQ(t2.type, TOK_IDENT);
    ASSERT_STR_EQ(t2.literal, "bar_baz");
    Token t3 = next(lexer);
    ASSERT_EQ(t3.type, TOK_IDENT);
    ASSERT_STR_EQ(t3.literal, "count123");
}

static void test_skip_comments(void) {
    Lexer *lexer = lex("a // line comment\nb");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.type, TOK_IDENT);
    ASSERT_STR_EQ(t1.literal, "a");
    Token t2 = next(lexer);
    ASSERT_EQ(t2.type, TOK_IDENT);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_skip_block_comments(void) {
    Lexer *lexer = lex("a /* block */ b");
    Token t1 = next(lexer);
    ASSERT_STR_EQ(t1.literal, "a");
    Token t2 = next(lexer);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_line_tracking(void) {
    Lexer *lexer = lex("a\nb\nc");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.line, 1);
    Token t2 = next(lexer);
    ASSERT_EQ(t2.line, 2);
    Token t3 = next(lexer);
    ASSERT_EQ(t3.line, 3);
}

static void test_hash_attributes(void) {
    Lexer *lexer = lex("#suppress #strict #flags #doc");
    ASSERT_EQ(next(lexer).type, TOK_SUPPRESS);
    ASSERT_EQ(next(lexer).type, TOK_STRICT);
    ASSERT_EQ(next(lexer).type, TOK_FLAGS);
    ASSERT_EQ(next(lexer).type, TOK_DOC);
}

static void test_hello_world_tokens(void) {
    Lexer *lexer = lex("do main() {\n    println(\"Hello\")\n}");
    ASSERT_EQ(next(lexer).type, TOK_DO);
    ASSERT_EQ(next(lexer).type, TOK_IDENT);  /* main */
    ASSERT_EQ(next(lexer).type, TOK_LPAREN);
    ASSERT_EQ(next(lexer).type, TOK_RPAREN);
    ASSERT_EQ(next(lexer).type, TOK_LBRACE);
    ASSERT_EQ(next(lexer).type, TOK_IDENT);  /* println */
    ASSERT_EQ(next(lexer).type, TOK_LPAREN);
    ASSERT_EQ(next(lexer).type, TOK_STRING); /* "Hello" */
    ASSERT_EQ(next(lexer).type, TOK_RPAREN);
    ASSERT_EQ(next(lexer).type, TOK_RBRACE);
    ASSERT_EQ(next(lexer).type, TOK_EOF);
}

static void test_v3_keyword_aliases(void) {
    /* mut maps to TOK_MUT, while maps to TOK_AS_LONG_AS */
    Lexer *lexer = lex("mut while");
    ASSERT_EQ(next(lexer).type, TOK_MUT);
    ASSERT_EQ(next(lexer).type, TOK_AS_LONG_AS);
}

static void test_v3_keywords_coexist(void) {
    /* as_long_as and while produce the same token */
    Lexer *lexer = lex("as_long_as while");
    Token t1 = next(lexer);
    Token t2 = next(lexer);
    ASSERT_EQ(t1.type, t2.type);
}

static void test_not_in_operator(void) {
    Lexer *lexer = lex("!in");
    ASSERT_EQ(next(lexer).type, TOK_NOT_IN);
}

static void test_not_in_vs_bang(void) {
    /* !in should be NOT_IN, !inside should be BANG + IDENT */
    Lexer *lexer = lex("!in !inside");
    ASSERT_EQ(next(lexer).type, TOK_NOT_IN);
    ASSERT_EQ(next(lexer).type, TOK_BANG);
    ASSERT_EQ(next(lexer).type, TOK_IDENT);
}

static void test_hex_literal(void) {
    Lexer *lexer = lex("0xFF");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0xFF");
}

static void test_octal_literal(void) {
    Lexer *lexer = lex("0o77");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0o77");
}

static void test_binary_literal(void) {
    Lexer *lexer = lex("0b1010");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0b1010");
}

static void test_or_return_keyword(void) {
    Lexer *lexer = lex("or_return");
    ASSERT_EQ(next(lexer).type, TOK_OR_RETURN);
}

static void test_caret_token(void) {
    Lexer *lexer = lex("^int");
    ASSERT_EQ(next(lexer).type, TOK_CARET);
    ASSERT_EQ(next(lexer).type, TOK_IDENT);
}

static void test_float_with_underscores(void) {
    Lexer *lexer = lex("3.141_592_653");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_FLOAT);
    ASSERT_STR_EQ(t.literal, "3.141_592_653");
}

static void test_interpolation_tokens(void) {
    /* Interpolated string starts as a string token */
    Lexer *lexer = lex("\"hello ${name}\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_STRING);
}

static void test_bang_in_vs_bang(void) {
    /* !in should be NOT_IN, ! alone should be BANG */
    Lexer *lexer = lex("!in x !y");
    ASSERT_EQ(next(lexer).type, TOK_NOT_IN);
    ASSERT_EQ(next(lexer).type, TOK_IDENT); /* x */
    ASSERT_EQ(next(lexer).type, TOK_BANG);
    ASSERT_EQ(next(lexer).type, TOK_IDENT); /* y */
}

/* --- Lexer Error Path Tests --- */

static void test_error_E1003_unclosed_comment(void) {
    Lexer *lexer = lex("/* unclosed comment");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1003");
}

static void test_error_E1005_unclosed_char(void) {
    Lexer *lexer = lex("'a");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1005");
}

static void test_error_E1006_bad_escape_string(void) {
    Lexer *lexer = lex("\"hello\\q\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1006");
}

static void test_error_E1007_bad_escape_char(void) {
    Lexer *lexer = lex("'\\q'");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1007");
}

static void test_error_E1010_bad_number_format(void) {
    Lexer *lexer = lex("0x");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1010");
}

static void test_error_E1011_consecutive_underscores(void) {
    Lexer *lexer = lex("1__000");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1011");
}

static void test_error_E1013_trailing_underscore(void) {
    Lexer *lexer = lex("100_");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1013");
}

static void test_error_E1014_underscore_before_decimal(void) {
    Lexer *lexer = lex("1_.5");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1014");
}

static void test_error_E1017_unclosed_raw_string(void) {
    Lexer *lexer = lex("`unclosed raw");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1017");
}

static void test_error_E1010_bad_octal(void) {
    Lexer *lexer = lex("0o");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1010");
}

static void test_error_E1010_bad_binary(void) {
    Lexer *lexer = lex("0b");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_ILLEGAL);
    ASSERT_NOT_NULL(l->error_code);
    ASSERT_STR_EQ(l->error_code, "E1010");
}

/* --- Single-character operator tokens --- */

static void test_single_operators(void) {
    Lexer *lexer = lex("+ - * / % < > =");
    ASSERT_EQ(next(lexer).type, TOK_PLUS);
    ASSERT_EQ(next(lexer).type, TOK_MINUS);
    ASSERT_EQ(next(lexer).type, TOK_ASTERISK);
    ASSERT_EQ(next(lexer).type, TOK_SLASH);
    ASSERT_EQ(next(lexer).type, TOK_PERCENT);
    ASSERT_EQ(next(lexer).type, TOK_LT);
    ASSERT_EQ(next(lexer).type, TOK_GT);
    ASSERT_EQ(next(lexer).type, TOK_ASSIGN);
}

static void test_double_asterisk(void) {
    Lexer *lexer = lex("**");
    ASSERT_EQ(next(lexer).type, TOK_ASTERISK);
    ASSERT_EQ(next(lexer).type, TOK_ASTERISK);
}

static void test_percent_assign(void) {
    Lexer *lexer = lex("%=");
    ASSERT_EQ(next(lexer).type, TOK_PERCENT_ASSIGN);
}

static void test_ampersand(void) {
    Lexer *lexer = lex("&x");
    ASSERT_EQ(next(lexer).type, TOK_AMPERSAND);
    ASSERT_EQ(next(lexer).type, TOK_IDENT);
}

static void test_semicolon(void) {
    Lexer *lexer = lex(";");
    ASSERT_EQ(next(lexer).type, TOK_SEMICOLON);
}

/* --- Missing keyword tokens --- */

static void test_keyword_in(void) {
    Lexer *lexer = lex("in");
    ASSERT_EQ(next(lexer).type, TOK_IN);
}

static void test_keyword_range(void) {
    Lexer *lexer = lex("range");
    ASSERT_EQ(next(lexer).type, TOK_RANGE);
}

static void test_keyword_use(void) {
    Lexer *lexer = lex("use");
    ASSERT_EQ(next(lexer).type, TOK_USE);
}

static void test_keyword_blank(void) {
    Lexer *lexer = lex("_");
    ASSERT_EQ(next(lexer).type, TOK_BLANK);
}


/* --- Column tracking --- */

static void test_column_tracking(void) {
    Lexer *lexer = lex("ab cd");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.column, 1);
    Token t2 = next(lexer);
    ASSERT_EQ(t2.column, 4);
}

static void test_column_resets_on_newline(void) {
    Lexer *lexer = lex("ab\ncd");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.line, 1);
    ASSERT_EQ(t1.column, 1);
    Token t2 = next(lexer);
    ASSERT_EQ(t2.line, 2);
    ASSERT_EQ(t2.column, 1);
}

/* --- Remaining lexer error paths --- */

/* Note: E1015 (underscore after decimal) and E1016 (trailing decimal) are
 * unreachable — read_number() only consumes the decimal point when followed
 * by a digit, so these validation checks are dead code. */

/* --- Char escape sequences --- */

static void test_char_escape_newline(void) {
    Lexer *lexer = lex("'\\n'");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_CHAR);
    ASSERT_STR_EQ(t.literal, "\\n");
}

static void test_char_escape_tab(void) {
    Lexer *lexer = lex("'\\t'");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_CHAR);
    ASSERT_STR_EQ(t.literal, "\\t");
}

/* --- Additional lexer coverage --- */

static void test_zero_literal(void) {
    Lexer *lexer = lex("0");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0");
}

static void test_negative_float_tokens(void) {
    /* -3.14 should tokenize as MINUS then FLOAT */
    Lexer *lexer = lex("-3.14");
    ASSERT_EQ(next(lexer).type, TOK_MINUS);
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_FLOAT);
    ASSERT_STR_EQ(t.literal, "3.14");
}

static void test_adjacent_operators(void) {
    /* >= should be GT_EQ, not GT + ASSIGN */
    Lexer *lexer = lex(">= <= == !=");
    ASSERT_EQ(next(lexer).type, TOK_GT_EQ);
    ASSERT_EQ(next(lexer).type, TOK_LT_EQ);
    ASSERT_EQ(next(lexer).type, TOK_EQ);
    ASSERT_EQ(next(lexer).type, TOK_NOT_EQ);
}

static void test_arrow_vs_minus(void) {
    /* -> should be ARROW, - > should be MINUS GT */
    Lexer *lexer = lex("->");
    ASSERT_EQ(next(lexer).type, TOK_ARROW);
}

static void test_string_with_interpolation_marker(void) {
    /* String containing ${...} is still a string token */
    Lexer *lexer = lex("\"hello ${name} world\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_STRING);
}

static void test_multiline_block_comment(void) {
    Lexer *lexer = lex("a /* this is\na multiline\ncomment */ b");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.type, TOK_IDENT);
    ASSERT_STR_EQ(t1.literal, "a");
    Token t2 = next(lexer);
    ASSERT_EQ(t2.type, TOK_IDENT);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_empty_string(void) {
    Lexer *lexer = lex("\"\"");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_STRING);
    ASSERT_STR_EQ(t.literal, "");
}

static void test_multiple_strings(void) {
    Lexer *lexer = lex("\"hello\" \"world\"");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.type, TOK_STRING);
    ASSERT_STR_EQ(t1.literal, "hello");
    Token t2 = next(lexer);
    ASSERT_EQ(t2.type, TOK_STRING);
    ASSERT_STR_EQ(t2.literal, "world");
}

static void test_consecutive_operators(void) {
    /* ++ -- should not be confused */
    Lexer *lexer = lex("++ --");
    ASSERT_EQ(next(lexer).type, TOK_INCREMENT);
    ASSERT_EQ(next(lexer).type, TOK_DECREMENT);
}

static void test_large_integer_literal(void) {
    Lexer *lexer = lex("9999999999");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "9999999999");
}

static void test_char_single_quote(void) {
    Lexer *lexer = lex("'x'");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_CHAR);
    ASSERT_STR_EQ(t.literal, "x");
}

static void test_keyword_panic(void) {
    Lexer *lexer = lex("panic");
    ASSERT_EQ(next(lexer).type, TOK_IDENT); /* panic is a builtin func, not keyword */
}

static void test_mixed_tokens_line(void) {
    /* Realistic code line */
    Lexer *lexer = lex("mut x int = 42");
    ASSERT_EQ(next(lexer).type, TOK_MUT);      /* mut */
    ASSERT_EQ(next(lexer).type, TOK_IDENT);     /* x */
    ASSERT_EQ(next(lexer).type, TOK_IDENT);     /* int */
    ASSERT_EQ(next(lexer).type, TOK_ASSIGN);    /* = */
    ASSERT_EQ(next(lexer).type, TOK_INT);       /* 42 */
    ASSERT_EQ(next(lexer).type, TOK_EOF);
}

static void test_line_tracking_with_comment(void) {
    /* Comments should not affect line tracking */
    Lexer *lexer = lex("a\n// comment\nb");
    Token t1 = next(lexer);
    ASSERT_EQ(t1.line, 1);
    Token t2 = next(lexer);
    ASSERT_EQ(t2.line, 3);
    ASSERT_STR_EQ(t2.literal, "b");
}

static void test_hex_upper_lower(void) {
    Lexer *lexer = lex("0xABCD");
    Token t = next(lexer);
    ASSERT_EQ(t.type, TOK_INT);
    ASSERT_STR_EQ(t.literal, "0xABCD");
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

    /* Single-character operators */
    RUN_TEST(test_single_operators);
    RUN_TEST(test_double_asterisk);
    RUN_TEST(test_percent_assign);
    RUN_TEST(test_ampersand);
    RUN_TEST(test_semicolon);

    /* Missing keywords */
    RUN_TEST(test_keyword_in);
    RUN_TEST(test_keyword_range);
    RUN_TEST(test_keyword_use);
    RUN_TEST(test_keyword_blank);

    /* Column tracking */
    RUN_TEST(test_column_tracking);
    RUN_TEST(test_column_resets_on_newline);

    /* Char escape sequences */
    RUN_TEST(test_char_escape_newline);
    RUN_TEST(test_char_escape_tab);

    /* Lexer error path tests */
    RUN_TEST(test_error_E1003_unclosed_comment);
    RUN_TEST(test_error_E1005_unclosed_char);
    RUN_TEST(test_error_E1006_bad_escape_string);
    RUN_TEST(test_error_E1007_bad_escape_char);
    RUN_TEST(test_error_E1010_bad_number_format);
    RUN_TEST(test_error_E1011_consecutive_underscores);
    RUN_TEST(test_error_E1013_trailing_underscore);
    RUN_TEST(test_error_E1014_underscore_before_decimal);
    /* E1015 and E1016 are unreachable (see comment above) */
    RUN_TEST(test_error_E1017_unclosed_raw_string);
    RUN_TEST(test_error_E1010_bad_octal);
    RUN_TEST(test_error_E1010_bad_binary);

    /* Additional lexer coverage */
    RUN_TEST(test_zero_literal);
    RUN_TEST(test_negative_float_tokens);
    RUN_TEST(test_adjacent_operators);
    RUN_TEST(test_arrow_vs_minus);
    RUN_TEST(test_string_with_interpolation_marker);
    RUN_TEST(test_multiline_block_comment);
    RUN_TEST(test_empty_string);
    RUN_TEST(test_multiple_strings);
    RUN_TEST(test_consecutive_operators);
    RUN_TEST(test_large_integer_literal);
    RUN_TEST(test_char_single_quote);
    RUN_TEST(test_keyword_panic);
    RUN_TEST(test_mixed_tokens_line);
    RUN_TEST(test_line_tracking_with_comment);
    RUN_TEST(test_hex_upper_lower);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
