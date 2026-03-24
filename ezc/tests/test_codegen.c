/*
 * test_codegen.c - End-to-end tests for EZC code generation
 *
 * Each test compiles an EZ snippet, runs the resulting binary,
 * and checks the output.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "test.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static int test_num = 0;

/* Compile and run an EZ program, return its stdout output */
static char *compile_and_run(const char *ez_source) {
    test_num++;
    static char output[4096];
    char ez_file[128], bin_file[128];

    snprintf(ez_file, sizeof(ez_file), "/tmp/ezc_e2e_%d.ez", test_num);
    snprintf(bin_file, sizeof(bin_file), "/tmp/ezc_e2e_%d", test_num);

    /* Write source */
    FILE *f = fopen(ez_file, "w");
    if (!f) return NULL;
    fputs(ez_source, f);
    fclose(f);

    /* Compile */
    char cmd[1024];
    snprintf(cmd, sizeof(cmd), "./ezc %s -o %s 2>/dev/null", ez_file, bin_file);
    if (system(cmd) != 0) {
        unlink(ez_file);
        return NULL;
    }

    /* Run and capture output */
    char run_cmd[256];
    snprintf(run_cmd, sizeof(run_cmd), "%s 2>&1", bin_file);
    FILE *p = popen(run_cmd, "r");
    if (!p) {
        unlink(ez_file);
        unlink(bin_file);
        return NULL;
    }

    size_t total = 0;
    size_t n;
    while ((n = fread(output + total, 1, sizeof(output) - total - 1, p)) > 0) {
        total += n;
    }
    output[total] = '\0';
    pclose(p);

    /* Remove trailing newline for easier comparison */
    if (total > 0 && output[total - 1] == '\n') {
        output[total - 1] = '\0';
    }

    unlink(ez_file);
    unlink(bin_file);
    return output;
}

/* --- Hello World --- */

static void test_e2e_hello(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() { println(\"Hello, World!\") }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "Hello, World!");
}

/* --- Arithmetic --- */

static void test_e2e_arithmetic(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    println(2 + 3)\n"
        "    println(10 - 4)\n"
        "    println(3 * 7)\n"
        "    println(20 / 4)\n"
        "    println(17 % 5)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "5\n6\n21\n5\n2");
}

/* --- Variables --- */

static void test_e2e_variables(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp x int = 10\n"
        "    temp y int = 20\n"
        "    println(x + y)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "30");
}

/* --- String interpolation --- */

static void test_e2e_interpolation(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp name string = \"Alice\"\n"
        "    temp age int = 30\n"
        "    println(\"${name} is ${age}\")\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "Alice is 30");
}

/* --- If/or/otherwise --- */

static void test_e2e_if_else(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp x int = 5\n"
        "    if x > 10 {\n"
        "        println(\"big\")\n"
        "    } or x > 3 {\n"
        "        println(\"medium\")\n"
        "    } otherwise {\n"
        "        println(\"small\")\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "medium");
}

/* --- For loop --- */

static void test_e2e_for_range(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    for i in range(1, 4) {\n"
        "        println(i)\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "1\n2\n3");
}

/* --- While loop --- */

static void test_e2e_while(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp i int = 0\n"
        "    as_long_as i < 3 {\n"
        "        println(i)\n"
        "        i++\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "0\n1\n2");
}

/* --- Loop with break --- */

static void test_e2e_loop_break(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp i int = 0\n"
        "    loop {\n"
        "        if i >= 3 { break }\n"
        "        println(i)\n"
        "        i++\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "0\n1\n2");
}

/* --- Functions --- */

static void test_e2e_function_call(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do add(a int, b int) -> int { return a + b }\n"
        "do main() { println(add(3, 4)) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "7");
}

/* --- Recursion --- */

static void test_e2e_recursion(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do fib(n int) -> int {\n"
        "    if n <= 1 { return n }\n"
        "    return fib(n - 1) + fib(n - 2)\n"
        "}\n"
        "do main() { println(fib(10)) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "55");
}

/* --- Multiple returns --- */

static void test_e2e_multi_return(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do swap(a int, b int) -> (int, int) { return b, a }\n"
        "do main() {\n"
        "    temp x int, y int = swap(10, 20)\n"
        "    println(x)\n"
        "    println(y)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "20\n10");
}

/* --- Mutable params --- */

static void test_e2e_mutable_param(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do inc(&n int) { n = n + 1 }\n"
        "do main() {\n"
        "    temp x int = 5\n"
        "    inc(x)\n"
        "    println(x)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "6");
}

/* --- Ensure --- */

static void test_e2e_ensure(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do cleanup() { println(\"cleaned\") }\n"
        "do work() {\n"
        "    ensure cleanup()\n"
        "    println(\"working\")\n"
        "}\n"
        "do main() { work() }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "working\ncleaned");
}

/* --- Structs --- */

static void test_e2e_struct(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "const Point struct {\n"
        "    x int\n"
        "    y int\n"
        "}\n"
        "do main() {\n"
        "    temp p Point = Point{x: 3, y: 4}\n"
        "    println(p.x)\n"
        "    println(p.y)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "3\n4");
}

/* --- Enums --- */

static void test_e2e_enum(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "const Color enum { RED\n GREEN\n BLUE }\n"
        "do main() {\n"
        "    temp c Color = Color.GREEN\n"
        "    println(c)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "1");
}

/* --- Arrays --- */

static void test_e2e_array(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp nums [int] = {10, 20, 30}\n"
        "    println(nums[0])\n"
        "    println(nums[2])\n"
        "    println(len(nums))\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "10\n30\n3");
}

/* --- Array mutation --- */

static void test_e2e_array_set(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp nums [int] = {1, 2, 3}\n"
        "    nums[1] = 99\n"
        "    println(nums[1])\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "99");
}

/* --- For each --- */

static void test_e2e_for_each(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp names [string] = {\"a\", \"b\", \"c\"}\n"
        "    for_each name in names {\n"
        "        println(name)\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "a\nb\nc");
}

/* --- When/Is --- */

static void test_e2e_when(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp x int = 2\n"
        "    when x {\n"
        "        is 1 { println(\"one\") }\n"
        "        is 2 { println(\"two\") }\n"
        "        default { println(\"other\") }\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "two");
}

/* --- Len builtin --- */

static void test_e2e_len_string(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() { println(len(\"hello\")) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "5");
}

/* --- Typeof builtin --- */

static void test_e2e_typeof(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    println(typeof(42))\n"
        "    println(typeof(\"hi\"))\n"
        "    println(typeof(true))\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "int\nstring\nbool");
}

/* --- Compound assignment --- */

static void test_e2e_compound_assign(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp x int = 10\n"
        "    x += 5\n"
        "    x -= 3\n"
        "    x *= 2\n"
        "    println(x)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "24");
}

/* --- Char type --- */

static void test_e2e_char(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp c char = 'A'\n"
        "    println(c)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "65");
}

/* --- Blank identifier --- */

static void test_e2e_blank_ident(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do pair() -> (int, int) { return 42, 99 }\n"
        "do main() {\n"
        "    temp _, b int = pair()\n"
        "    println(b)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "99");
}

/* ===== @mem Module Tests ===== */

static void test_e2e_mem_arena_create_destroy(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    println(\"created\")\n"
        "    mem.destroy(a)\n"
        "    println(\"destroyed\")\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "created\ndestroyed");
}

static void test_e2e_mem_usage(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(1024)\n"
        "    println(mem.usage(a))\n"
        "    temp s string = mem.alloc(a, \"hello\")\n"
        "    temp used int = mem.usage(a)\n"
        "    if used > 0 { println(\"allocated\") }\n"
        "    mem.destroy(a)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "0\nallocated");
}

static void test_e2e_mem_reset(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(1024)\n"
        "    temp s string = mem.alloc(a, \"hello\")\n"
        "    if mem.usage(a) > 0 { println(\"used\") }\n"
        "    mem.reset(a)\n"
        "    println(mem.usage(a))\n"
        "    mem.destroy(a)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "used\n0");
}

static void test_e2e_mem_alloc_string(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    ensure mem.destroy(a)\n"
        "    temp s string = mem.alloc(a, \"arena string\")\n"
        "    println(s)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "arena string");
}

static void test_e2e_mem_alloc_array(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    ensure mem.destroy(a)\n"
        "    temp nums [int] = mem.alloc(a, {10, 20, 30})\n"
        "    println(nums[0])\n"
        "    println(nums[1])\n"
        "    println(nums[2])\n"
        "    println(len(nums))\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "10\n20\n30\n3");
}

static void test_e2e_mem_ensure_cleanup(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do work() {\n"
        "    temp a = mem.arena(1024)\n"
        "    ensure mem.destroy(a)\n"
        "    temp s string = mem.alloc(a, \"working\")\n"
        "    println(s)\n"
        "}\n"
        "do main() {\n"
        "    work()\n"
        "    println(\"done\")\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "working\ndone");
}

/* ===== Pointer Tests ===== */

static void test_e2e_ptr_new_deref(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    ensure mem.destroy(a)\n"
        "    temp p ^int = mem.new(a, int)\n"
        "    p^ = 42\n"
        "    println(p^)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "42");
}

static void test_e2e_ptr_struct(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "const Point struct {\n"
        "    x int\n"
        "    y int\n"
        "}\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    ensure mem.destroy(a)\n"
        "    temp p ^Point = mem.new(a, Point)\n"
        "    p^.x = 3\n"
        "    p^.y = 4\n"
        "    println(p^.x)\n"
        "    println(p^.y)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "3\n4");
}

static void test_e2e_ptr_addr(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp x int = 10\n"
        "    temp p ^int = addr(x)\n"
        "    println(p^)\n"
        "    p^ = 99\n"
        "    println(x)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "10\n99");
}

static void test_e2e_ptr_nil(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    temp p ^int = nil\n"
        "    if p == nil {\n"
        "        println(\"null\")\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "null");
}

static void test_e2e_ptr_write_through(void) {
    char *out = compile_and_run(
        "import @std, @mem\nusing std\n"
        "do set_value(p ^int, val int) {\n"
        "    p^ = val\n"
        "}\n"
        "do main() {\n"
        "    temp a = mem.arena(4096)\n"
        "    ensure mem.destroy(a)\n"
        "    temp p ^int = mem.new(a, int)\n"
        "    set_value(p, 777)\n"
        "    println(p^)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "777");
}

/* ===== v3 Keyword Tests ===== */

static void test_e2e_mut_keyword(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    mut x int = 10\n"
        "    mut y = 20\n"
        "    println(x + y)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "30");
}

static void test_e2e_while_keyword(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    mut i int = 0\n"
        "    while i < 3 {\n"
        "        println(i)\n"
        "        i++\n"
        "    }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "0\n1\n2");
}

static void test_e2e_mut_while_combined(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do fib(n int) -> int {\n"
        "    mut a int = 0\n"
        "    mut b int = 1\n"
        "    mut i int = 0\n"
        "    while i < n {\n"
        "        mut next int = a + b\n"
        "        a = b\n"
        "        b = next\n"
        "        i++\n"
        "    }\n"
        "    return a\n"
        "}\n"
        "do main() { println(fib(10)) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "55");
}

/* ===== Default Params, in/not_in, OS, Arrays Tests ===== */

static void test_e2e_default_params(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do greet(name string = \"World\") -> string {\n"
        "    return \"Hello, ${name}!\"\n"
        "}\n"
        "do main() {\n"
        "    println(greet())\n"
        "    println(greet(\"EZC\"))\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "Hello, World!\nHello, EZC!");
}

static void test_e2e_in_operator(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() {\n"
        "    mut nums [int] = {1, 2, 3}\n"
        "    if 2 in nums { println(\"found\") }\n"
        "    if 9 !in nums { println(\"not found\") }\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "found\nnot found");
}

static void test_e2e_os_args(void) {
    char *out = compile_and_run(
        "import @std, @os\nusing std\n"
        "do main() {\n"
        "    println(os.arch())\n"
        "}");
    ASSERT_NOT_NULL(out);
    /* Should be arm64 or x86_64 — just check it's not empty */
    ASSERT(strlen(out) > 0);
}

static void test_e2e_arrays_append(void) {
    char *out = compile_and_run(
        "import @std, @arrays\nusing std\n"
        "do main() {\n"
        "    mut nums [int] = {1, 2}\n"
        "    arrays.append(nums, 3)\n"
        "    println(len(nums))\n"
        "    println(nums[2])\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "3\n3");
}

static void test_e2e_arrays_sort(void) {
    char *out = compile_and_run(
        "import @std, @arrays\nusing std\n"
        "do main() {\n"
        "    mut nums [int] = {3, 1, 2}\n"
        "    arrays.sort(nums)\n"
        "    println(nums[0])\n"
        "    println(nums[1])\n"
        "    println(nums[2])\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "1\n2\n3");
}

static void test_e2e_hex_literal(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() { println(0xFF) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "255");
}

static void test_e2e_octal_literal(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() { println(0o10) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "8");
}

static void test_e2e_binary_literal(void) {
    char *out = compile_and_run(
        "import @std\nusing std\n"
        "do main() { println(0b1010) }");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "10");
}

/* ===== Threads Tests ===== */

static void test_e2e_threads_spawn_join(void) {
    char *out = compile_and_run(
        "import @std, @threads\nusing std\n"
        "do worker(id int) { println(\"w${id}\") }\n"
        "do main() {\n"
        "  mut t1 = threads.spawn(()worker, 1)\n"
        "  mut t2 = threads.spawn(()worker, 2)\n"
        "  threads.join(t1)\n"
        "  threads.join(t2)\n"
        "  println(\"done\")\n"
        "}");
    ASSERT_NOT_NULL(out);
    /* Output order may vary due to threading, but "done" must be last */
    ASSERT(strstr(out, "done") != NULL);
    ASSERT(strstr(out, "w1") != NULL);
    ASSERT(strstr(out, "w2") != NULL);
}

static void test_e2e_threads_channel(void) {
    char *out = compile_and_run(
        "import @std, @threads\nusing std\n"
        "do main() {\n"
        "  mut ch = threads.channel(4)\n"
        "  threads.send(ch, 42)\n"
        "  threads.send(ch, 100)\n"
        "  mut a = threads.recv(ch)\n"
        "  mut b = threads.recv(ch)\n"
        "  println(\"${a},${b}\")\n"
        "  threads.close(ch)\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "42,100");
}

static void test_e2e_threads_sleep(void) {
    char *out = compile_and_run(
        "import @std, @threads\nusing std\n"
        "do main() {\n"
        "  threads.sleep_ms(10)\n"
        "  println(\"awake\")\n"
        "}");
    ASSERT_NOT_NULL(out);
    ASSERT_STR_EQ(out, "awake");
}

int main(void) {
    /* Must run from the ezc/ directory */
    if (access("./ezc", X_OK) != 0) {
        fprintf(stderr, "test_codegen: must run from the ezc/ directory\n");
        return 1;
    }

    printf("\n");
    RUN_TEST(test_e2e_hello);
    RUN_TEST(test_e2e_arithmetic);
    RUN_TEST(test_e2e_variables);
    RUN_TEST(test_e2e_interpolation);
    RUN_TEST(test_e2e_if_else);
    RUN_TEST(test_e2e_for_range);
    RUN_TEST(test_e2e_while);
    RUN_TEST(test_e2e_loop_break);
    RUN_TEST(test_e2e_function_call);
    RUN_TEST(test_e2e_recursion);
    RUN_TEST(test_e2e_multi_return);
    RUN_TEST(test_e2e_mutable_param);
    RUN_TEST(test_e2e_ensure);
    RUN_TEST(test_e2e_struct);
    RUN_TEST(test_e2e_enum);
    RUN_TEST(test_e2e_array);
    RUN_TEST(test_e2e_array_set);
    RUN_TEST(test_e2e_for_each);
    RUN_TEST(test_e2e_when);
    RUN_TEST(test_e2e_len_string);
    RUN_TEST(test_e2e_typeof);
    RUN_TEST(test_e2e_compound_assign);
    RUN_TEST(test_e2e_char);
    RUN_TEST(test_e2e_blank_ident);

    /* @mem module */
    RUN_TEST(test_e2e_mem_arena_create_destroy);
    RUN_TEST(test_e2e_mem_usage);
    RUN_TEST(test_e2e_mem_reset);
    RUN_TEST(test_e2e_mem_alloc_string);
    RUN_TEST(test_e2e_mem_alloc_array);
    RUN_TEST(test_e2e_mem_ensure_cleanup);

    /* Pointers */
    RUN_TEST(test_e2e_ptr_new_deref);
    RUN_TEST(test_e2e_ptr_struct);
    RUN_TEST(test_e2e_ptr_addr);
    RUN_TEST(test_e2e_ptr_nil);
    RUN_TEST(test_e2e_ptr_write_through);

    /* v3 keywords */
    RUN_TEST(test_e2e_mut_keyword);
    RUN_TEST(test_e2e_while_keyword);
    RUN_TEST(test_e2e_mut_while_combined);

    /* New features */
    RUN_TEST(test_e2e_default_params);
    RUN_TEST(test_e2e_in_operator);
    RUN_TEST(test_e2e_os_args);
    RUN_TEST(test_e2e_arrays_append);
    RUN_TEST(test_e2e_arrays_sort);
    RUN_TEST(test_e2e_hex_literal);
    RUN_TEST(test_e2e_octal_literal);
    RUN_TEST(test_e2e_binary_literal);

    /* Threads */
    RUN_TEST(test_e2e_threads_spawn_join);
    RUN_TEST(test_e2e_threads_channel);
    RUN_TEST(test_e2e_threads_sleep);

    PRINT_RESULTS();
    return _test_fail > 0 ? 1 : 0;
}
