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
#include <time.h>
#ifdef __APPLE__
#include <mach-o/dyld.h>
#endif

#include "util/arena.h"
#include "util/error.h"
#include "lexer/lexer.h"
#include "parser/parser.h"
#include "typechecker/typechecker.h"
#include "codegen/codegen.h"

#define EZ_VERSION "3.0.0"
#define PATH_BUF_SIZE 2048
#define CMD_BUF_SIZE 8192

static void print_usage(void) {
    fprintf(stderr, "EZ Programming Language v%s\n", EZ_VERSION);
    fprintf(stderr, "\nUsage:\n");
    fprintf(stderr, "  ez <file.ez> [options]         Compile to binary (default)\n");
    fprintf(stderr, "  ez build <file.ez> [options]   Compile to binary\n");
    fprintf(stderr, "  ez run <file.ez> [options]     Compile and run (temp binary)\n");
    fprintf(stderr, "  ez check <file.ez>             Type check only\n");
    fprintf(stderr, "  ez version                     Show version\n");
    fprintf(stderr, "\nOptions:\n");
    fprintf(stderr, "  -o <file>       Output binary name (default: based on input filename)\n");
    fprintf(stderr, "  -c              Emit C source only (don't compile)\n");
    fprintf(stderr, "  -O0, -O1, -O2   Optimization level (default: -O2)\n");
    fprintf(stderr, "  -g              Include debug symbols\n");
    fprintf(stderr, "  -v, --verbose   Show compilation commands\n");
    fprintf(stderr, "  --time          Show compilation timing\n");
    fprintf(stderr, "  --no-color      Disable colored output\n");
    fprintf(stderr, "  -h, --help      Show this help\n");
}

static char *read_file(const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) {
        fprintf(stderr, "ez: cannot open '%s': ", path);
        perror("");
        return NULL;
    }

    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *buf = malloc((size_t)size + 1);
    if (!buf) {
        fprintf(stderr, "ez: out of memory\n");
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
        fprintf(stderr, "ez: cannot write '%s': ", path);
        perror("");
        return false;
    }
    fputs(content, f);
    fclose(f);
    return true;
}

/* Resolve the directory containing the ezc binary itself */
static const char *get_self_dir(const char *argv0) {
    static char buf[1024];

#ifdef __APPLE__
    /* macOS: use _NSGetExecutablePath */
    uint32_t size = sizeof(buf);
    if (_NSGetExecutablePath(buf, &size) == 0) {
        char *resolved = realpath(buf, NULL);
        if (resolved) {
            strncpy(buf, resolved, sizeof(buf) - 1);
            free(resolved);
            char *slash = strrchr(buf, '/');
            if (slash) *slash = '\0';
            return buf;
        }
    }
#endif

#ifdef __linux__
    /* Linux: read /proc/self/exe */
    ssize_t len = readlink("/proc/self/exe", buf, sizeof(buf) - 1);
    if (len > 0) {
        buf[len] = '\0';
        char *slash = strrchr(buf, '/');
        if (slash) *slash = '\0';
        return buf;
    }
#endif

    /* Fallback: try to resolve argv[0] */
    if (argv0) {
        char *resolved = realpath(argv0, NULL);
        if (resolved) {
            strncpy(buf, resolved, sizeof(buf) - 1);
            free(resolved);
            char *slash = strrchr(buf, '/');
            if (slash) *slash = '\0';
            return buf;
        }
    }

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

/*
 * Find the runtime directory containing ez_runtime.h and ez_std.h.
 *
 * Search order:
 *   1. EZC_RUNTIME env var (explicit override)
 *   2. Relative to binary: ../lib/ezc (installed layout)
 *   3. Relative to binary: src (development layout — binary is in ezc/)
 *   4. Relative to CWD: ezc/src (running from project root)
 *   5. /usr/local/lib/ezc (system install)
 */
static const char *find_runtime_dir(const char *argv0) {
    static char path[PATH_BUF_SIZE];

    /* 1. Environment variable override */
    const char *env = getenv("EZC_RUNTIME");
    if (env && access(env, R_OK) == 0) {
        snprintf(path, sizeof(path), "%s/runtime/ez_runtime.h", env);
        if (access(path, R_OK) == 0) return env;
    }

    /* 2-3. Relative to binary location */
    const char *self_dir = get_self_dir(argv0);
    if (self_dir) {
        /* Installed layout: binary in /usr/local/bin, runtime in /usr/local/lib/ezc */
        snprintf(path, sizeof(path), "%s/../lib/ezc/runtime/ez_runtime.h", self_dir);
        if (access(path, R_OK) == 0) {
            snprintf(path, sizeof(path), "%s/../lib/ezc", self_dir);
            return path;
        }

        /* Development layout: binary in ezc/, runtime in ezc/src/runtime */
        snprintf(path, sizeof(path), "%s/src/runtime/ez_runtime.h", self_dir);
        if (access(path, R_OK) == 0) {
            snprintf(path, sizeof(path), "%s/src", self_dir);
            return path;
        }
    }

    /* 4. Relative to CWD (running from project root) */
    if (access("ezc/src/runtime/ez_runtime.h", R_OK) == 0) {
        return "ezc/src";
    }

    /* 5. System install location */
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
    bool check_only = false;
    bool run_mode = false;
    bool verbose = false;
    bool show_time = false;
    bool no_color = false;
    bool debug_symbols = false;
    const char *opt_level = "-O2";

    /* Parse arguments */
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "version") == 0 || strcmp(argv[i], "--version") == 0) {
            printf("ez %s\n", EZ_VERSION);
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
        if (strcmp(argv[i], "-O0") == 0) { opt_level = "-O0"; continue; }
        if (strcmp(argv[i], "-O1") == 0) { opt_level = "-O1"; continue; }
        if (strcmp(argv[i], "-O2") == 0) { opt_level = "-O2"; continue; }
        if (strcmp(argv[i], "-O3") == 0) { opt_level = "-O3"; continue; }
        if (strcmp(argv[i], "-g") == 0) {
            debug_symbols = true;
            continue;
        }
        if (strcmp(argv[i], "-v") == 0 || strcmp(argv[i], "--verbose") == 0) {
            verbose = true;
            continue;
        }
        if (strcmp(argv[i], "--time") == 0) {
            show_time = true;
            continue;
        }
        if (strcmp(argv[i], "--no-color") == 0) {
            no_color = true;
            continue;
        }
        /* Subcommands */
        if (strcmp(argv[i], "check") == 0 && !input_file) {
            check_only = true;
            continue;
        }
        if (strcmp(argv[i], "build") == 0 && !input_file) {
            /* build is the default — just skip the keyword */
            continue;
        }
        if (strcmp(argv[i], "run") == 0 && !input_file) {
            run_mode = true;
            continue;
        }
        if (argv[i][0] == '-') {
            fprintf(stderr, "ez: unknown option '%s'\n", argv[i]);
            return 1;
        }
        input_file = argv[i];
    }

    if (!input_file) {
        fprintf(stderr, "ez: no input file\n");
        return 1;
    }

    /* Read source file */
    char *source = read_file(input_file);
    if (!source) return 1;

    /* Create compiler arena and diagnostics */
    Arena *arena = arena_create(1024 * 1024);
    DiagnosticList *diag = diag_create();
    diag_set_source(diag, input_file, source);
    if (no_color) diag->use_color = false;

    clock_t t_start = clock();

    /* Lex */
    Lexer *lexer = lexer_create(arena, source, input_file);

    /* Parse */
    Parser *parser = parser_create(arena, lexer, input_file, diag);
    AstNode *program = parser_parse_program(parser);

    if (diag_has_errors(diag)) {
        diag_print_all(diag);
        diag_print_summary(diag);
        diag_destroy(diag);
        arena_destroy(arena);
        free(source);
        return 1;
    }

    /* Resolve local imports: parse imported .ez files and merge declarations */
    {
        /* Determine the directory of the input file */
        char input_dir[PATH_BUF_SIZE];
        strncpy(input_dir, input_file, sizeof(input_dir) - 1);
        input_dir[sizeof(input_dir) - 1] = '\0';
        char *last_slash = strrchr(input_dir, '/');
        if (last_slash) *(last_slash + 1) = '\0';
        else strcpy(input_dir, "./");

        for (int si = 0; si < program->data.program.stmt_count; si++) {
            AstNode *stmt = program->data.program.stmts[si];
            if (stmt->kind != NODE_IMPORT_STMT) continue;

            for (int ii = 0; ii < stmt->data.import_stmt.count; ii++) {
                ImportItem *item = &stmt->data.import_stmt.items[ii];
                if (item->is_stdlib || !item->path) continue;

                /* Resolve path relative to input file directory */
                char import_path[PATH_BUF_SIZE];
                const char *rel = item->path;
                if (rel[0] == '.' && rel[1] == '/') rel += 2;
                /* Path already includes .ez extension from parser */
                snprintf(import_path, sizeof(import_path), "%s%s", input_dir, rel);

                /* Derive module name from filename (strip directory and .ez) */
                const char *mod_base = rel;
                const char *slash = strrchr(rel, '/');
                if (slash) mod_base = slash + 1;
                /* Strip .ez extension for module name */
                char mod_name_buf[256];
                size_t mod_len = strlen(mod_base);
                if (mod_len > 3 && strcmp(mod_base + mod_len - 3, ".ez") == 0) {
                    memcpy(mod_name_buf, mod_base, mod_len - 3);
                    mod_name_buf[mod_len - 3] = '\0';
                } else {
                    snprintf(mod_name_buf, sizeof(mod_name_buf), "%s", mod_base);
                }
                const char *mod_name = arena_strdup(arena, mod_name_buf);

                /* Set the alias if not already set */
                if (!item->alias) item->alias = mod_name;
                if (!item->module) item->module = mod_name;

                /* Read and parse the imported file */
                char *imp_source = read_file(import_path);
                if (!imp_source) {
                    fprintf(stderr, "ez: cannot open imported file '%s'\n", import_path);
                    continue;
                }

                Lexer *imp_lexer = lexer_create(arena, imp_source, import_path);
                Parser *imp_parser = parser_create(arena, imp_lexer, import_path, diag);
                AstNode *imp_program = parser_parse_program(imp_parser);

                if (!imp_program || diag_has_errors(diag)) continue;

                /* Merge imported declarations into main program.
                 * Prefix function names with the module name. */
                for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                    AstNode *imp_stmt = imp_program->data.program.stmts[mi];
                    /* Skip import/using/module declarations from imported file */
                    if (imp_stmt->kind == NODE_IMPORT_STMT ||
                        imp_stmt->kind == NODE_USING_STMT ||
                        imp_stmt->kind == NODE_MODULE_DECL) continue;

                    /* Prefix function names with module name */
                    if (imp_stmt->kind == NODE_FUNC_DECL) {
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.func_decl.name);
                        imp_stmt->data.func_decl.name = prefixed;
                    }

                    /* Prefix constant names with module name */
                    if (imp_stmt->kind == NODE_VAR_DECL && !imp_stmt->data.var_decl.mutable) {
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.var_decl.name);
                        imp_stmt->data.var_decl.name = prefixed;
                    }

                    /* Insert into main program BEFORE existing declarations.
                     * This ensures imported constants/functions are visible to all code. */
                    if (program->data.program.stmt_count >= program->data.program.stmt_cap) {
                        int new_cap = program->data.program.stmt_cap * 2;
                        AstNode **new_stmts = arena_alloc(arena, sizeof(AstNode *) * new_cap);
                        memcpy(new_stmts, program->data.program.stmts,
                               sizeof(AstNode *) * program->data.program.stmt_count);
                        program->data.program.stmts = new_stmts;
                        program->data.program.stmt_cap = new_cap;
                    }
                    /* Find insertion point: after imports/using, before other declarations */
                    int insert_at = 0;
                    for (int k = 0; k < program->data.program.stmt_count; k++) {
                        if (program->data.program.stmts[k]->kind == NODE_IMPORT_STMT ||
                            program->data.program.stmts[k]->kind == NODE_USING_STMT) {
                            insert_at = k + 1;
                        } else {
                            break;
                        }
                    }
                    /* Shift existing stmts to make room */
                    memmove(&program->data.program.stmts[insert_at + 1],
                            &program->data.program.stmts[insert_at],
                            sizeof(AstNode *) * (program->data.program.stmt_count - insert_at));
                    program->data.program.stmts[insert_at] = imp_stmt;
                    program->data.program.stmt_count++;
                    /* Adjust si since we shifted statements */
                    si++;
                }
            }
        }
    }

    /* Type check */
    TypeChecker *tc = typechecker_create(diag, input_file);
    typechecker_check(tc, program);

    if (diag_has_errors(diag)) {
        diag_print_all(diag);
        diag_print_summary(diag);
        diag_destroy(diag);
        arena_destroy(arena);
        free(source);
        return 1;
    }

    /* Print warnings even if no errors */
    if (diag_warning_count(diag) > 0 && !diag_has_errors(diag)) {
        diag_print_all(diag);
    }

    /* Check-only mode: stop after type checking */
    if (check_only) {
        clock_t t_end = clock();
        if (show_time) {
            double ms = (double)(t_end - t_start) / CLOCKS_PER_SEC * 1000.0;
            fprintf(stderr, "ez: check completed in %.1fms\n", ms);
        }
        fprintf(stderr, "ez: %s — no errors\n", input_file);
        diag_destroy(diag);
        arena_destroy(arena);
        free(source);
        return 0;
    }

    /* Generate C code */
    CodeGen cg = codegen_create(input_file);
    cg.type_table = typechecker_get_table(tc);
    codegen_generate(&cg, program);
    const char *c_code = codegen_result(&cg);

    /* Determine output name */
    char *default_output = NULL;
    if (run_mode && !output_file) {
        /* Run mode: use temp file */
        default_output = malloc(PATH_BUF_SIZE);
        snprintf(default_output, PATH_BUF_SIZE, "/tmp/ezc_run_%d", (int)getpid());
        output_file = default_output;
    } else if (!output_file) {
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

    /* Check that a C compiler is available */
    if (system("cc --version >/dev/null 2>&1") != 0 &&
        system("gcc --version >/dev/null 2>&1") != 0 &&
        system("clang --version >/dev/null 2>&1") != 0) {
        fprintf(stderr, "ez: no C compiler found.\n");
        fprintf(stderr, "  Install gcc or clang to compile EZ programs.\n");
        fprintf(stderr, "  On macOS: xcode-select --install\n");
        fprintf(stderr, "  On Ubuntu: sudo apt install gcc\n");
        codegen_destroy(&cg);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 1;
    }

    /* Find runtime directory */
    const char *runtime_dir = find_runtime_dir(argv[0]);
    if (!runtime_dir) {
        fprintf(stderr, "ez: cannot find runtime headers.\n");
        fprintf(stderr, "  Searched:\n");
        fprintf(stderr, "    - $EZC_RUNTIME environment variable\n");
        fprintf(stderr, "    - relative to ez binary\n");
        fprintf(stderr, "    - ./ezc/src/ (project root)\n");
        fprintf(stderr, "    - /usr/local/lib/ezc/\n");
        fprintf(stderr, "  Try: cd <project-root> && make -C ezc install\n");
        codegen_destroy(&cg);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 1;
    }

    /* Compile the generated C code.
     * Try linking against pre-compiled libezrt.a first (fast path).
     * Fall back to compiling runtime from source if archive not found. */
    char cmd[CMD_BUF_SIZE];
    char lib_path[PATH_BUF_SIZE];
    bool has_archive = false;

    /* Check for libezrt.a relative to binary */
    snprintf(lib_path, sizeof(lib_path), "%s/../libezrt.a", runtime_dir);
    if (access(lib_path, R_OK) == 0) {
        has_archive = true;
    } else {
        /* Check in install location */
        snprintf(lib_path, sizeof(lib_path), "%s/../libezrt.a", runtime_dir);
        if (access(lib_path, R_OK) != 0) {
            /* Check next to the runtime dir */
            const char *self = get_self_dir(NULL);
            if (self) {
                snprintf(lib_path, sizeof(lib_path), "%s/libezrt.a", self);
                if (access(lib_path, R_OK) == 0) has_archive = true;
            }
        }
    }

    /* Build debug/optimization flags */
    char extra_flags[128] = "";
    if (debug_symbols) {
        snprintf(extra_flags, sizeof(extra_flags), "-g %s", opt_level);
    } else {
        snprintf(extra_flags, sizeof(extra_flags), "%s", opt_level);
    }

    clock_t t_cc_start = clock();

    if (has_archive) {
        snprintf(cmd, sizeof(cmd),
            "cc -std=c11 %s -Wall -Wno-unused-function -Wno-unused-variable "
            "-I%s/runtime -I%s/stdlib "
            "-o %s %s %s "
            "-lm -lpthread 2>&1",
            extra_flags,
            runtime_dir, runtime_dir,
            output_file, c_file, lib_path);
    } else {
        /* Build source list from all runtime and stdlib .c files */
        static const char *runtime_srcs[] = {
            "runtime/ez_runtime.c", "runtime/ez_array.c", "runtime/ez_map.c",
        };
        static const char *stdlib_srcs[] = {
            "stdlib/ez_arrays.c",   "stdlib/ez_binary.c",   "stdlib/ez_builtins.c",
            "stdlib/ez_bytes.c",    "stdlib/ez_channels.c", "stdlib/ez_crypto.c",
            "stdlib/ez_csv.c",      "stdlib/ez_encoding.c", "stdlib/ez_fmt.c",
            "stdlib/ez_http.c",     "stdlib/ez_io.c",       "stdlib/ez_json.c",
            "stdlib/ez_maps.c",     "stdlib/ez_math.c",     "stdlib/ez_mem.c",
            "stdlib/ez_net.c",      "stdlib/ez_os.c",       "stdlib/ez_random.c",
            "stdlib/ez_regex.c",    "stdlib/ez_server.c",   "stdlib/ez_sqlite.c",
            "stdlib/ez_strings.c",  "stdlib/ez_sync.c",     "stdlib/ez_atomic.c",
            "stdlib/ez_threads.c",
            "stdlib/ez_time.c",     "stdlib/ez_uuid.c",
        };
        char srcs[CMD_BUF_SIZE];
        int off = 0;
        for (int i = 0; i < (int)(sizeof(runtime_srcs)/sizeof(runtime_srcs[0])); i++)
            off += snprintf(srcs + off, sizeof(srcs) - (size_t)off, "%s/%s ", runtime_dir, runtime_srcs[i]);
        for (int i = 0; i < (int)(sizeof(stdlib_srcs)/sizeof(stdlib_srcs[0])); i++)
            off += snprintf(srcs + off, sizeof(srcs) - (size_t)off, "%s/%s ", runtime_dir, stdlib_srcs[i]);

        snprintf(cmd, sizeof(cmd),
            "cc -std=c11 %s -Wall -Wno-unused-function -Wno-unused-variable "
            "-I%s/runtime -I%s/stdlib "
            "-o %s %s %s"
            "-lm -lpthread 2>&1",
            extra_flags,
            runtime_dir, runtime_dir,
            output_file, c_file, srcs);
    }

    if (verbose) {
        fprintf(stderr, "ez: %s\n", cmd);
    }

    if (strlen(cmd) >= CMD_BUF_SIZE - 1) {
        fprintf(stderr, "ez: compile command too long (paths may be too deep)\n");
        codegen_destroy(&cg);
        diag_destroy(diag);
        arena_destroy(arena);
        free(source);
        free(default_output);
        return 1;
    }

    int ret = system(cmd);

    clock_t t_cc_end = clock();

    if (ret != 0) {
        fprintf(stderr, "ez: C compilation failed\n");
        fprintf(stderr, "ez: generated C source at %s\n", c_file);
    } else {
        unlink(c_file);

        if (show_time) {
            double frontend_ms = (double)(t_cc_start - t_start) / CLOCKS_PER_SEC * 1000.0;
            double cc_ms = (double)(t_cc_end - t_cc_start) / CLOCKS_PER_SEC * 1000.0;
            double total_ms = (double)(t_cc_end - t_start) / CLOCKS_PER_SEC * 1000.0;
            fprintf(stderr, "ez: compiled %s → %s (%.0fms)\n", input_file, output_file, total_ms);
            fprintf(stderr, "  frontend:  %.1fms (lex + parse + typecheck + codegen)\n", frontend_ms);
            fprintf(stderr, "  cc:        %.1fms (compile + link)\n", cc_ms);
        }
    }

    /* Run mode: execute the binary and clean up */
    if (ret == 0 && run_mode) {
        char run_cmd[CMD_BUF_SIZE];
        snprintf(run_cmd, sizeof(run_cmd), "%s", output_file);
        ret = system(run_cmd);
        unlink(output_file);
    }

    codegen_destroy(&cg);
    diag_destroy(diag);
    arena_destroy(arena);
    free(source);
    free(default_output);

    return ret != 0 ? 1 : 0;
}
