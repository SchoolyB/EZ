/*
 * main.c - EZC compiler entry point
 *
 * Usage: ezc <file.ez> [-o output]
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/stat.h>

#include "util/arena.h"
#include "lexer/lexer.h"
#include "parser/parser.h"
#include "codegen/codegen.h"

#define EZC_VERSION "0.1.0"

static void print_usage(void) {
    fprintf(stderr, "EZC - EZ Language Compiler v%s\n", EZC_VERSION);
    fprintf(stderr, "Usage: ezc <file.ez> [-o output]\n");
    fprintf(stderr, "\nOptions:\n");
    fprintf(stderr, "  -o <file>    Output binary name (default: based on input filename)\n");
    fprintf(stderr, "  -c           Emit C source only (don't compile)\n");
    fprintf(stderr, "  --version    Show version\n");
    fprintf(stderr, "  --help       Show this help\n");
}

static char *read_file(const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) {
        fprintf(stderr, "ezc: cannot open '%s': ", path);
        perror("");
        return NULL;
    }

    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *buf = malloc((size_t)size + 1);
    if (!buf) {
        fprintf(stderr, "ezc: out of memory\n");
        fclose(f);
        return NULL;
    }

    size_t read = fread(buf, 1, (size_t)size, f);
    buf[read] = '\0';
    fclose(f);
    return buf;
}

static bool write_file(const char *path, const char *content) {
    FILE *f = fopen(path, "w");
    if (!f) {
        fprintf(stderr, "ezc: cannot write '%s': ", path);
        perror("");
        return false;
    }
    fputs(content, f);
    fclose(f);
    return true;
}

/* Get the directory where the ezc binary lives (for finding runtime headers) */
static const char *get_runtime_dir(const char *argv0) {
    (void)argv0;
    /* For now, look relative to CWD or use a known path */
    /* TODO: resolve relative to binary location */
    return NULL;
}

/* Strip .ez extension and return base name */
static char *output_name_from_input(const char *input) {
    const char *slash = strrchr(input, '/');
    const char *base = slash ? slash + 1 : input;

    size_t len = strlen(base);
    if (len > 3 && strcmp(base + len - 3, ".ez") == 0) {
        len -= 3;
    }

    char *out = malloc(len + 1);
    memcpy(out, base, len);
    out[len] = '\0';
    return out;
}

/* Find the runtime source directory */
static const char *find_runtime_dir(const char *argv0) {
    /* Try relative to binary */
    static char path[1024];

    /* Try the source tree layout first (for development) */
    /* Check if ezc/src/runtime/ez_runtime.h exists relative to CWD */
    if (access("ezc/src/runtime/ez_runtime.h", R_OK) == 0) {
        return "ezc/src";
    }

    /* Check relative to binary location */
    const char *dir = get_runtime_dir(argv0);
    if (dir) {
        snprintf(path, sizeof(path), "%s/../src", dir);
        return path;
    }

    /* Fallback: try common install locations */
    if (access("/usr/local/lib/ezc/runtime/ez_runtime.h", R_OK) == 0) {
        return "/usr/local/lib/ezc";
    }

    return NULL;
}

int main(int argc, char **argv) {
    if (argc < 2) {
        print_usage();
        return 1;
    }

    const char *input_file = NULL;
    const char *output_file = NULL;
    bool emit_c_only = false;

    /* Parse arguments */
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--version") == 0) {
            printf("ezc %s\n", EZC_VERSION);
            return 0;
        }
        if (strcmp(argv[i], "--help") == 0 || strcmp(argv[i], "-h") == 0) {
            print_usage();
            return 0;
        }
        if (strcmp(argv[i], "-o") == 0 && i + 1 < argc) {
            output_file = argv[++i];
            continue;
        }
        if (strcmp(argv[i], "-c") == 0) {
            emit_c_only = true;
            continue;
        }
        if (argv[i][0] == '-') {
            fprintf(stderr, "ezc: unknown option '%s'\n", argv[i]);
            return 1;
        }
        input_file = argv[i];
    }

    if (!input_file) {
        fprintf(stderr, "ezc: no input file\n");
        return 1;
    }

    /* Read source file */
    char *source = read_file(input_file);
    if (!source) return 1;

    /* Create compiler arena */
    Arena *arena = arena_create(1024 * 1024);

    /* Lex */
    Lexer *lexer = lexer_create(arena, source, input_file);

    /* Parse */
    Parser *parser = parser_create(arena, lexer, input_file);
    AstNode *program = parser_parse_program(parser);

    if (parser_has_errors(parser)) {
        parser_print_errors(parser);
        arena_destroy(arena);
        free(source);
        return 1;
    }

    /* Generate C code */
    CodeGen cg = codegen_create(input_file);
    codegen_generate(&cg, program);
    const char *c_code = codegen_result(&cg);

    /* Determine output name */
    char *default_output = NULL;
    if (!output_file) {
        default_output = output_name_from_input(input_file);
        output_file = default_output;
    }

    /* Write generated C to temp file */
    const char *out_base = strrchr(output_file, '/');
    out_base = out_base ? out_base + 1 : output_file;
    char c_file[1024];
    snprintf(c_file, sizeof(c_file), "/tmp/ezc_%s.c", out_base);

    if (!write_file(c_file, c_code)) {
        codegen_destroy(&cg);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 1;
    }

    if (emit_c_only) {
        /* Just print the C source */
        printf("%s", c_code);
        codegen_destroy(&cg);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 0;
    }

    /* Find runtime directory */
    const char *runtime_dir = find_runtime_dir(argv[0]);
    if (!runtime_dir) {
        fprintf(stderr, "ezc: cannot find runtime headers. "
            "Run from the project root or install the runtime.\n");
        codegen_destroy(&cg);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 1;
    }

    /* Compile the generated C code */
    char cmd[4096];
    snprintf(cmd, sizeof(cmd),
        "cc -std=c11 -O2 -Wall -Wno-unused-function "
        "-I%s/runtime -I%s/stdlib "
        "-o %s %s %s/runtime/ez_runtime.c %s/stdlib/ez_std.c "
        "-lm 2>&1",
        runtime_dir, runtime_dir,
        output_file, c_file,
        runtime_dir, runtime_dir);

    int ret = system(cmd);
    if (ret != 0) {
        fprintf(stderr, "ezc: C compilation failed\n");
        /* Keep the generated .c file for debugging */
        fprintf(stderr, "ezc: generated C source at %s\n", c_file);
    } else {
        /* Clean up temp file on success */
        unlink(c_file);
    }

    codegen_destroy(&cg);
    arena_destroy(arena);
    free(source);
    free(default_output);

    return ret != 0 ? 1 : 0;
}
