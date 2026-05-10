/*
 * main.c - EZ compiler entry point
 *
 * Usage: ez <file.ez> [-o output]
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/stat.h>
#include <dirent.h>
#include <time.h>
#ifdef __APPLE__
#include <mach-o/dyld.h>
#endif

#include "util/arena.h"
#include "util/error.h"
#include "util/constants.h"
#include "lexer/lexer.h"
#include "parser/parser.h"
#include "typechecker/typechecker.h"
#include "codegen/codegen.h"

#ifndef EZ_VERSION
#define EZ_VERSION "unknown"
#endif
#define PATH_BUF_SIZE 2048
#define CMD_BUF_SIZE 8192
#define EZ_MAX_IMPORTS 256
#define EZ_COMPILER_ARENA_SIZE (1024 * 1024)

static void print_usage(void) {
    fprintf(stderr, "EZ Programming Language v%s\n", EZ_VERSION);
    fprintf(stderr, "\nUsage:\n");
    fprintf(stderr, "  ez <file.ez> [options]         Compile and run\n");
    fprintf(stderr, "  ez build <file.ez> [options]   Compile to binary\n");
    fprintf(stderr, "  ez check <file.ez>             Type check only\n");
    fprintf(stderr, "  ez version                     Show version\n");
    fprintf(stderr, "\nOptions:\n");
    fprintf(stderr, "  -o <file>       Output binary name (default: based on input filename)\n");
    fprintf(stderr, "  -c              Emit C source only (don't compile)\n");
    fprintf(stderr, "  -O0, -O1, -O2   Optimization level (default: -O2)\n");
    fprintf(stderr, "  -g              Include debug symbols\n");
    fprintf(stderr, "  -v, --verbose   Show compilation commands\n");
    fprintf(stderr, "  --time          Show compilation timing\n");
    fprintf(stderr, "  --quiet         Suppress all warnings\n");
    fprintf(stderr, "  --quiet W1001   Suppress specific warnings (comma-separated)\n");
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

    /* Try the seekable fast path. Fall through to streaming on any
     * failure: pipes, FIFOs, /dev/stdin, /proc, sockets, etc. all make
     * ftell return -1, which previously wrapped (size_t)-1 + 1 to 0
     * and caused fread to write past the heap allocation. */
    long size = -1;
    if (fseek(f, 0, SEEK_END) == 0) {
        size = ftell(f);
        if (size >= 0) (void)fseek(f, 0, SEEK_SET);
    }

    if (size >= 0) {
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

    /* Streaming fallback for non-seekable inputs. */
    clearerr(f);
    size_t cap = 4096;
    size_t len = 0;
    char *buf = malloc(cap);
    if (!buf) {
        fprintf(stderr, "ez: out of memory\n");
        fclose(f);
        return NULL;
    }
    for (;;) {
        if (len == cap) {
            size_t new_cap = cap * 2;
            char *new_buf = realloc(buf, new_cap);
            if (!new_buf) {
                free(buf);
                fclose(f);
                fprintf(stderr, "ez: out of memory\n");
                return NULL;
            }
            buf = new_buf;
            cap = new_cap;
        }
        size_t got = fread(buf + len, 1, cap - len, f);
        if (got == 0) break;
        len += got;
    }
    if (len + 1 > cap) {
        char *grow = realloc(buf, len + 1);
        if (grow) buf = grow;
    }
    buf[len] = '\0';
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
    static char buf[PATH_BUF_SIZE];

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
 *   1. EZ_RUNTIME env var (explicit override)
 *   2. Relative to binary: ../lib/ezc (installed layout)
 *   3. Relative to binary: src (development layout — binary is in ezc/)
 *   4. Relative to CWD: ezc/src (running from project root)
 *   5. /usr/local/lib/ezc (system install)
 */
static const char *find_runtime_dir(const char *argv0) {
    static char path[PATH_BUF_SIZE];

    /* 1. Environment variable override */
    const char *env = getenv("EZ_RUNTIME");
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

    /* 4. Walk up from CWD looking for the project root */
    {
        char cwd[PATH_BUF_SIZE];
        if (getcwd(cwd, sizeof(cwd))) {
            char probe[PATH_BUF_SIZE];
            char *dir = cwd;
            while (*dir) {
                snprintf(probe, sizeof(probe), "%s/ezc/src/runtime/ez_runtime.h", dir);
                if (access(probe, R_OK) == 0) {
                    snprintf(path, sizeof(path), "%s/ezc/src", dir);
                    return path;
                }
                /* Move to parent */
                char *slash = strrchr(dir, '/');
                if (!slash || slash == dir) break;
                *slash = '\0';
            }
        }
    }

    /* 5. System install location */
    if (access("/usr/local/lib/ezc/runtime/ez_runtime.h", R_OK) == 0) {
        return "/usr/local/lib/ezc";
    }

    return NULL;
}

/* Rewrite labels in an AST tree: replace unprefixed names with prefixed versions */
static void rewrite_labels(AstNode *node, const char **orig, const char **prefixed, int count, Arena *arena) {
    if (!node) return;
    if (node->kind == NODE_LABEL) {
        for (int i = 0; i < count; i++) {
            if (strcmp(node->data.label.value, orig[i]) == 0) {
                node->data.label.value = prefixed[i];
                return;
            }
        }
        return;
    }
    /* Recurse into child nodes */
    switch (node->kind) {
    case NODE_PREFIX_EXPR: rewrite_labels(node->data.prefix.right, orig, prefixed, count, arena); break;
    case NODE_INFIX_EXPR:
        rewrite_labels(node->data.infix.left, orig, prefixed, count, arena);
        rewrite_labels(node->data.infix.right, orig, prefixed, count, arena);
        break;
    case NODE_CALL_EXPR:
        rewrite_labels(node->data.call.function, orig, prefixed, count, arena);
        for (int i = 0; i < node->data.call.arg_count; i++)
            rewrite_labels(node->data.call.args[i], orig, prefixed, count, arena);
        break;
    case NODE_RETURN_STMT:
        for (int i = 0; i < node->data.return_stmt.count; i++)
            rewrite_labels(node->data.return_stmt.values[i], orig, prefixed, count, arena);
        break;
    case NODE_VAR_DECL:
        /* Rewrite type annotation: mut req Request → mut req mod_Request */
        if (node->data.var_decl.type_name) {
            for (int i = 0; i < count; i++) {
                if (strcmp(node->data.var_decl.type_name, orig[i]) == 0) {
                    node->data.var_decl.type_name = prefixed[i];
                    break;
                }
            }
        }
        rewrite_labels(node->data.var_decl.value, orig, prefixed, count, arena);
        break;
    case NODE_ASSIGN_STMT:
        rewrite_labels(node->data.assign.target, orig, prefixed, count, arena);
        rewrite_labels(node->data.assign.value, orig, prefixed, count, arena);
        break;
    case NODE_IF_STMT:
        rewrite_labels(node->data.if_stmt.condition, orig, prefixed, count, arena);
        if (node->data.if_stmt.consequence) {
            rewrite_labels(node->data.if_stmt.consequence, orig, prefixed, count, arena);
        }
        if (node->data.if_stmt.alternative) {
            rewrite_labels(node->data.if_stmt.alternative, orig, prefixed, count, arena);
        }
        break;
    case NODE_BLOCK_STMT:
        for (int i = 0; i < node->data.block.count; i++)
            rewrite_labels(node->data.block.stmts[i], orig, prefixed, count, arena);
        break;
    case NODE_EXPR_STMT:
        rewrite_labels(node->data.expr_stmt.expr, orig, prefixed, count, arena);
        break;
    case NODE_POSTFIX_EXPR:
        rewrite_labels(node->data.postfix.left, orig, prefixed, count, arena);
        break;
    case NODE_INDEX_EXPR:
        rewrite_labels(node->data.index_expr.left, orig, prefixed, count, arena);
        rewrite_labels(node->data.index_expr.index, orig, prefixed, count, arena);
        break;
    case NODE_MEMBER_EXPR:
        rewrite_labels(node->data.member.object, orig, prefixed, count, arena);
        break;
    case NODE_FOR_STMT:
        rewrite_labels(node->data.for_stmt.iterable, orig, prefixed, count, arena);
        rewrite_labels(node->data.for_stmt.body, orig, prefixed, count, arena);
        break;
    case NODE_FOR_EACH_STMT:
        rewrite_labels(node->data.for_each.collection, orig, prefixed, count, arena);
        rewrite_labels(node->data.for_each.body, orig, prefixed, count, arena);
        break;
    case NODE_WHILE_STMT:
        rewrite_labels(node->data.while_stmt.condition, orig, prefixed, count, arena);
        rewrite_labels(node->data.while_stmt.body, orig, prefixed, count, arena);
        break;
    case NODE_LOOP_STMT:
        rewrite_labels(node->data.loop_stmt.body, orig, prefixed, count, arena);
        break;
    case NODE_INTERPOLATED_STRING:
        for (int i = 0; i < node->data.interpolated_string.part_count; i++)
            rewrite_labels(node->data.interpolated_string.parts[i], orig, prefixed, count, arena);
        break;
    case NODE_STRUCT_VALUE:
        /* Rewrite struct literal name: Point{...} → shapes_Point{...} */
        for (int i = 0; i < count; i++) {
            if (node->data.struct_value.name && strcmp(node->data.struct_value.name, orig[i]) == 0) {
                node->data.struct_value.name = prefixed[i];
                break;
            }
        }
        /* Rewrite field value expressions */
        for (int i = 0; i < node->data.struct_value.count; i++) {
            rewrite_labels(node->data.struct_value.field_values[i], orig, prefixed, count, arena);
        }
        break;
    case NODE_WHEN_STMT:
        rewrite_labels(node->data.when_stmt.value, orig, prefixed, count, arena);
        for (int i = 0; i < node->data.when_stmt.case_count; i++) {
            WhenCase *wc = &node->data.when_stmt.cases[i];
            for (int j = 0; j < wc->value_count; j++)
                rewrite_labels(wc->values[j], orig, prefixed, count, arena);
            rewrite_labels(wc->body, orig, prefixed, count, arena);
        }
        if (node->data.when_stmt.default_body)
            rewrite_labels(node->data.when_stmt.default_body, orig, prefixed, count, arena);
        break;
    case NODE_ENSURE_STMT:
        rewrite_labels(node->data.ensure_stmt.expr, orig, prefixed, count, arena);
        break;
    case NODE_CAST_EXPR:
        /* Rewrite cast target type: as(Request, x) → as(mod_Request, x) */
        if (node->data.cast.target_type) {
            for (int i = 0; i < count; i++) {
                if (strcmp(node->data.cast.target_type, orig[i]) == 0) {
                    node->data.cast.target_type = prefixed[i];
                    break;
                }
            }
        }
        rewrite_labels(node->data.cast.value, orig, prefixed, count, arena);
        break;
    case NODE_NEW_EXPR:
        /* Rewrite new() type: new(Hero) → new(mod_Hero) */
        if (node->data.new_expr.type_name) {
            for (int i = 0; i < count; i++) {
                if (strcmp(node->data.new_expr.type_name, orig[i]) == 0) {
                    node->data.new_expr.type_name = prefixed[i];
                    break;
                }
            }
        }
        break;
    case NODE_RANGE_EXPR:
        rewrite_labels(node->data.range_expr.start, orig, prefixed, count, arena);
        rewrite_labels(node->data.range_expr.end, orig, prefixed, count, arena);
        rewrite_labels(node->data.range_expr.step, orig, prefixed, count, arena);
        break;
    case NODE_FUNC_DECL:
        /* Rewrite return types in nested function declarations */
        for (int i = 0; i < node->data.func_decl.return_type_count; i++) {
            const char *rt = node->data.func_decl.return_types[i];
            for (int j = 0; j < count; j++) {
                if (strcmp(rt, orig[j]) == 0) {
                    node->data.func_decl.return_types[i] = prefixed[j];
                    break;
                }
            }
        }
        /* Rewrite parameter types in nested function declarations */
        for (int i = 0; i < node->data.func_decl.param_count; i++) {
            const char *pt = node->data.func_decl.params[i].type_name;
            if (!pt) continue;
            for (int j = 0; j < count; j++) {
                if (strcmp(pt, orig[j]) == 0) {
                    node->data.func_decl.params[i].type_name = prefixed[j];
                    break;
                }
            }
        }
        rewrite_labels(node->data.func_decl.body, orig, prefixed, count, arena);
        break;
    default: break;
    }
}

/* Import cache: track already-imported files to avoid duplicates and cycles */
static const char *imported_files[EZ_MAX_IMPORTS];
static const char *imported_modules_map[EZ_MAX_IMPORTS]; /* module name that imported each file */
static int imported_file_count = 0;

static bool already_imported(const char *path) {
    for (int i = 0; i < imported_file_count; i++) {
        if (strcmp(imported_files[i], path) == 0) return true;
    }
    return false;
}

static const char *imported_by_module(const char *path) {
    for (int i = 0; i < imported_file_count; i++) {
        if (strcmp(imported_files[i], path) == 0) return imported_modules_map[i];
    }
    return NULL;
}

static void mark_imported_with_module(const char *path, const char *mod) {
    if (imported_file_count < EZ_MAX_IMPORTS) {
        imported_files[imported_file_count] = path;
        imported_modules_map[imported_file_count] = mod;
        imported_file_count++;
    }
}

static void mark_imported(const char *path) {
    mark_imported_with_module(path, NULL);
}

/* Scan a directory for .ez files. Returns count of files found.
 * Fills paths[] with full file paths (dir_path + "/" + filename). */
#define MAX_DIR_FILES 256
static int scan_ez_files(const char *dir_path, char paths[][PATH_BUF_SIZE], int max_files) {
    DIR *d = opendir(dir_path);
    if (!d) return -1;

    int count = 0;
    struct dirent *ent;
    while ((ent = readdir(d)) != NULL && count < max_files) {
        const char *name = ent->d_name;
        if (name[0] == '.') continue; /* skip hidden files and . / .. */
        size_t nlen = strlen(name);
        if (nlen < 4 || strcmp(name + nlen - 3, ".ez") != 0) continue;
        snprintf(paths[count], PATH_BUF_SIZE, "%s/%s", dir_path, name);
        count++;
    }
    closedir(d);

    /* Sort alphabetically for deterministic import order */
    for (int i = 0; i < count - 1; i++) {
        for (int j = i + 1; j < count; j++) {
            if (strcmp(paths[i], paths[j]) > 0) {
                char tmp[PATH_BUF_SIZE];
                memcpy(tmp, paths[i], PATH_BUF_SIZE);
                memcpy(paths[i], paths[j], PATH_BUF_SIZE);
                memcpy(paths[j], tmp, PATH_BUF_SIZE);
            }
        }
    }
    return count;
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
    bool quiet_all = false;
    const char *quiet_codes_arg = NULL;
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
        if (strcmp(argv[i], "--quiet") == 0 || strcmp(argv[i], "-q") == 0) {
            /* --quiet / -q with optional next argument for specific codes */
            if (i + 1 < argc && argv[i + 1][0] == 'W') {
                quiet_codes_arg = argv[++i];
            } else if (i + 1 < argc && argv[i + 1][0] == 'E') {
                fprintf(stderr, "ez: '-q' only accepts warning codes (W-prefixed), not error code '%s'\n", argv[i + 1]);
                return 1;
            } else {
                quiet_all = true;
            }
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
    Arena *arena = arena_create(EZ_COMPILER_ARENA_SIZE);
    DiagnosticList *diag = diag_create();
    diag_set_source(diag, input_file, source);
    if (no_color) diag->use_color = false;

    /* Configure warning suppression */
    if (quiet_all) {
        diag->suppress_all_warnings = true;
    } else if (quiet_codes_arg) {
        /* Parse comma-separated warning codes */
        char *codes_buf = strdup(quiet_codes_arg);
        int code_cap = 8;
        diag->suppressed_codes = malloc(sizeof(const char *) * code_cap);
        diag->suppressed_count = 0;
        char *tok = strtok(codes_buf, ",");
        while (tok) {
            /* Validate: must start with W */
            if (tok[0] == 'E') {
                fprintf(stderr, "ez: '-q' only accepts warning codes (W-prefixed), not error code '%s'\n", tok);
                free(codes_buf);
                return 1;
            }
            if (tok[0] != 'W') {
                fprintf(stderr, "ez: unknown warning code '%s'\n", tok);
                free(codes_buf);
                return 1;
            }
            if (diag->suppressed_count >= code_cap) {
                code_cap *= 2;
                diag->suppressed_codes = realloc(diag->suppressed_codes, sizeof(const char *) * code_cap);
            }
            diag->suppressed_codes[diag->suppressed_count++] = strdup(tok);
            tok = strtok(NULL, ",");
        }
        free(codes_buf);
    }

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
        /* Mark the main file as already imported (prevents circular import loops).
         * Use realpath so that diamond dependencies reaching the main file via
         * different relative paths are still detected as duplicates. */
        {
            char *rp = realpath(input_file, NULL);
            mark_imported(rp ? arena_strdup(arena, rp) : input_file);
            free(rp);
        }

        /* Derive main file's module name for circular import resolution */
        const char *main_base = input_file;
        const char *main_slash = strrchr(input_file, '/');
        if (main_slash) main_base = main_slash + 1;
        char main_mod_buf[EZ_MSG_BUF_SIZE];
        size_t main_mod_len = strlen(main_base);
        if (main_mod_len > 3 && strcmp(main_base + main_mod_len - 3, ".ez") == 0) {
            memcpy(main_mod_buf, main_base, main_mod_len - 3);
            main_mod_buf[main_mod_len - 3] = '\0';
        } else {
            snprintf(main_mod_buf, sizeof(main_mod_buf), "%s", main_base);
        }

        /* Determine the directory of the input file */
        char input_dir[PATH_BUF_SIZE];
        strncpy(input_dir, input_file, sizeof(input_dir) - 1);
        input_dir[sizeof(input_dir) - 1] = '\0';
        char *last_slash = strrchr(input_dir, '/');
        if (last_slash) *(last_slash + 1) = '\0';
        else { input_dir[0] = '.'; input_dir[1] = '/'; input_dir[2] = '\0'; }

        /* Process imports iteratively — re-scan after each merge to find transitive imports.
         * The already_imported cache prevents infinite loops and duplicate processing. */
        bool found_new_import = true;
        while (found_new_import) {
            found_new_import = false;

            /* Collect current import statements */
            AstNode *import_stmts[EZ_MAX_IMPORTS];
            int import_stmt_count = 0;
            for (int si = 0; si < program->data.program.stmt_count; si++) {
                if (program->data.program.stmts[si]->kind == NODE_IMPORT_STMT &&
                    import_stmt_count < EZ_MAX_IMPORTS) {
                    import_stmts[import_stmt_count++] = program->data.program.stmts[si];
                }
            }

        for (int si = 0; si < import_stmt_count; si++) {
            AstNode *stmt = import_stmts[si];

            for (int ii = 0; ii < stmt->data.import_stmt.count; ii++) {
                ImportItem *item = &stmt->data.import_stmt.items[ii];
                if (item->is_stdlib || item->is_c_import || !item->path) continue;

                /* Resolve path relative to the file that contains the import.
                 * For imports written directly in the entry file, source_dir is NULL
                 * and we fall back to input_dir. For transitive imports injected from
                 * imported files, source_dir points at the importing file's directory. */
                char import_path[PATH_BUF_SIZE];
                const char *rel = item->path;
                if (rel[0] == '.' && rel[1] == '/') rel += 2;
                const char *base_dir = item->source_dir ? item->source_dir : input_dir;
                snprintf(import_path, sizeof(import_path), "%s%s", base_dir, rel);

                /* Determine import kind: direct .ez file, extensionless file, or directory.
                 * Build a list of actual .ez file paths to import. */
                char (*file_list)[PATH_BUF_SIZE] = NULL;
                int file_count = 0;

                size_t iplen = strlen(import_path);
                if (iplen >= 3 && strcmp(import_path + iplen - 3, ".ez") == 0) {
                    /* Case 1: explicit .ez path — direct file import */
                    file_list = arena_alloc(arena, sizeof(char[PATH_BUF_SIZE]));
                    strncpy(file_list[0], import_path, PATH_BUF_SIZE - 1);
                    file_list[0][PATH_BUF_SIZE - 1] = '\0';
                    file_count = 1;
                } else {
                    /* Case 2: try appending .ez (extensionless file import) */
                    char try_file[PATH_BUF_SIZE];
                    snprintf(try_file, sizeof(try_file), "%s.ez", import_path);
                    struct stat st;
                    if (stat(try_file, &st) == 0 && S_ISREG(st.st_mode)) {
                        file_list = arena_alloc(arena, sizeof(char[PATH_BUF_SIZE]));
                        strncpy(file_list[0], try_file, PATH_BUF_SIZE - 1);
                        file_list[0][PATH_BUF_SIZE - 1] = '\0';
                        file_count = 1;
                        /* Update import_path so collision detection uses the resolved path */
                        strncpy(import_path, try_file, sizeof(import_path) - 1);
                        import_path[sizeof(import_path) - 1] = '\0';
                    } else if (stat(import_path, &st) == 0 && S_ISDIR(st.st_mode)) {
                        /* Case 3: directory import — scan for .ez files */

                        /* Self-referential directory import: if the importing file
                         * lives inside the directory it is trying to import, reject. */
                        if (item->source_dir) {
                            char *norm_dir = realpath(import_path, NULL);
                            char *norm_src = realpath(item->source_dir, NULL);
                            if (norm_dir && norm_src && strcmp(norm_dir, norm_src) == 0) {
                                char msg[EZ_MSG_BUF_LARGE];
                                snprintf(msg, sizeof(msg),
                                    "cannot import own module directory '%s'", item->path);
                                diag_error(diag, "E6004", strdup(msg),
                                    input_file, stmt->token.line, stmt->token.column, 0);
                                free(norm_dir);
                                free(norm_src);
                                continue;
                            }
                            free(norm_dir);
                            free(norm_src);
                        }

                        file_list = arena_alloc(arena, sizeof(char[PATH_BUF_SIZE]) * MAX_DIR_FILES);
                        file_count = scan_ez_files(import_path, file_list, MAX_DIR_FILES);
                        if (file_count == 0) {
                            char msg[EZ_MSG_BUF_LARGE];
                            snprintf(msg, sizeof(msg), "directory '%s' contains no .ez files", item->path);
                            diag_error(diag, "E6003", strdup(msg),
                                input_file, stmt->token.line, stmt->token.column, 0);
                            continue;
                        }
                    } else {
                        /* Nothing found */
                        char msg[EZ_MSG_BUF_LARGE];
                        snprintf(msg, sizeof(msg), "cannot find file or directory '%s'", item->path);
                        diag_error(diag, "E6002", strdup(msg),
                            input_file, stmt->token.line, stmt->token.column, 0);
                        continue;
                    }
                }

                /* Derive module name from filename/directory (strip directory and .ez) */
                const char *mod_base = rel;
                const char *slash = strrchr(rel, '/');
                if (slash) mod_base = slash + 1;

                /* For directory imports the path ends with '/' so mod_base
                 * points at the empty string after it.  Back up to extract
                 * the actual directory name (e.g. "engine" from "src/engine/"). */
                if (mod_base[0] == '\0' && slash && slash > rel) {
                    const char *prev = slash - 1;
                    while (prev > rel && *prev != '/') prev--;
                    if (*prev == '/') prev++;
                    size_t dlen = (size_t)(slash - prev);
                    char dir_name[EZ_MSG_BUF_SIZE];
                    memcpy(dir_name, prev, dlen);
                    dir_name[dlen] = '\0';
                    mod_base = arena_strdup(arena, dir_name);
                }

                char mod_name_buf[EZ_MSG_BUF_SIZE];
                size_t mod_len = strlen(mod_base);
                if (mod_len > 3 && strcmp(mod_base + mod_len - 3, ".ez") == 0) {
                    memcpy(mod_name_buf, mod_base, mod_len - 3);
                    mod_name_buf[mod_len - 3] = '\0';
                } else {
                    snprintf(mod_name_buf, sizeof(mod_name_buf), "%s", mod_base);
                }
                const char *mod_name = item->alias ? item->alias : arena_strdup(arena, mod_name_buf);

                /* Normalize import_path so diamond deps resolve to the same canonical path */
                char norm_import[PATH_BUF_SIZE];
                {
                    char *rp = realpath(import_path, NULL);
                    if (rp) {
                        strncpy(norm_import, rp, sizeof(norm_import) - 1);
                        norm_import[sizeof(norm_import) - 1] = '\0';
                        free(rp);
                    } else {
                        strncpy(norm_import, import_path, sizeof(norm_import) - 1);
                        norm_import[sizeof(norm_import) - 1] = '\0';
                    }
                }

                /* Module name collision detection — only for DIRECT imports from the same file.
                 * Don't collide with the main file's own module name.
                 * Diamond dependencies (same file via different paths) emit a warning
                 * and are skipped rather than causing a false E6001 error. */
                static const char *seen_modules[EZ_MAX_IMPORTS];
                static const char *seen_paths[EZ_MAX_IMPORTS];
                static int seen_count = 0;
                bool collision = false;
                for (int sm = 0; sm < seen_count; sm++) {
                    if (strcmp(seen_modules[sm], mod_name) == 0) {
                        if (strcmp(seen_paths[sm], norm_import) == 0) {
                            /* Same file, same module name — diamond dependency.
                             * Only warn for direct imports; transitive diamonds
                             * (from inside directory modules) are silently deduped. */
                            if (!item->source_dir) {
                                char msg[EZ_MSG_BUF_SIZE];
                                snprintf(msg, sizeof(msg),
                                    "module '%s' is already imported; duplicate import ignored",
                                    mod_name);
                                diag_warning(diag, "W2013", strdup(msg),
                                    input_file, stmt->token.line, stmt->token.column, 0);
                            }
                            collision = true;
                            break;
                        }
                        /* Different file, same module name — genuine collision */
                        char msg[EZ_MSG_BUF_SIZE];
                        snprintf(msg, sizeof(msg),
                            "module name '%s' is already imported — use an alias to distinguish them",
                            mod_name);
                        diag_error(diag, "E6001", strdup(msg),
                            input_file, stmt->token.line, stmt->token.column, 0);
                        collision = true;
                        break;
                    }
                }
                if (collision) continue;
                if (seen_count < EZ_MAX_IMPORTS) {
                    seen_modules[seen_count] = mod_name;
                    seen_paths[seen_count] = arena_strdup(arena, norm_import);
                    seen_count++;
                }

                /* Set the alias if not already set */
                if (!item->alias) item->alias = mod_name;
                if (!item->module) item->module = mod_name;

                /* Process each file in the import (1 for single file, N for directory).
                 * For directory imports, we use a two-pass approach:
                 *   Pass 1: Parse all files, collect ALL declaration names across all files
                 *   Pass 2: Rewrite using the combined mapping, then merge
                 * This ensures sibling references (e.g. logic.ez referencing types.ez's
                 * structs) get properly rewritten to their prefixed names. */

                /* Storage for parsed programs in the directory */
                AstNode *parsed_programs[MAX_DIR_FILES];
                const char *parsed_paths[MAX_DIR_FILES];
                int parsed_count = 0;

                /* Sibling import aliases collected during parse pass.
                 * After all names are known, compound mappings (alias_Name → mod_Name)
                 * are generated for each alias × declaration name. */
                const char *sibling_aliases[EZ_MAX_IMPORTS];
                int sibling_alias_count = 0;

                /* Combined name mapping across all files in this import */
                const char *orig_names[EZ_MAX_IMPORTS];
                const char *new_names[EZ_MAX_IMPORTS];
                int name_count = 0;

                /* Parse pass: parse each file, collect names, inject transitive imports */
                for (int fi = 0; fi < file_count; fi++) {
                    const char *cur_file_path = file_list[fi];

                    /* Normalize the path so diamond dependencies (same file
                     * reached via different relative paths) are deduplicated. */
                    char *norm = realpath(cur_file_path, NULL);
                    const char *norm_path = norm ? arena_strdup(arena, norm) : cur_file_path;
                    free(norm);

                    /* Skip if already imported (handles cycles and duplicates) */
                    if (already_imported(norm_path)) {
                        /* If this is a transitive import from inside a directory
                         * module referencing a sibling already pulled in by the
                         * directory import, emit an informational warning. */
                        if (item->source_dir && file_count == 1) {
                            char msg[EZ_MSG_BUF_SIZE];
                            snprintf(msg, sizeof(msg),
                                "import of '%s' is redundant; already included by directory import",
                                item->path);
                            diag_warning(diag, "W2014", strdup(msg),
                                input_file, stmt->token.line, stmt->token.column, 0);
                        } else if (!item->source_dir && file_count == 1) {
                            /* Direct import of a file already pulled in by a directory import */
                            const char *owner_mod = imported_by_module(norm_path);
                            char msg[EZ_MSG_BUF_LARGE];
                            if (owner_mod) {
                                snprintf(msg, sizeof(msg),
                                    "file '%s' was already imported as part of a directory import; "
                                    "use the '%s' namespace (e.g. %s.%s()) instead",
                                    item->path, owner_mod, owner_mod, mod_name_buf);
                            } else {
                                snprintf(msg, sizeof(msg),
                                    "file '%s' was already imported as part of a directory import; "
                                    "this import is redundant",
                                    item->path);
                            }
                            diag_warning(diag, "W2015", strdup(msg),
                                input_file, stmt->token.line, stmt->token.column, 0);
                        }
                        continue;
                    }
                    mark_imported_with_module(norm_path, mod_name);
                    found_new_import = true;

                    /* Read and parse the imported file */
                    char *imp_source = read_file(cur_file_path);
                    if (!imp_source) {
                        char msg[EZ_MSG_BUF_LARGE];
                        snprintf(msg, sizeof(msg), "cannot find file or directory '%s'", cur_file_path);
                        diag_error(diag, "E6002", strdup(msg),
                            input_file, stmt->token.line, stmt->token.column, 0);
                        continue;
                    }

                    Lexer *imp_lexer = lexer_create(arena, imp_source, cur_file_path);
                    Parser *imp_parser = parser_create(arena, imp_lexer, cur_file_path, diag);
                    AstNode *imp_program = parser_parse_program(imp_parser);

                    if (!imp_program || diag_has_errors(diag)) continue;

                    /* Inject transitive import statements into the main program.
                     * Sibling imports (pointing to other files in the same directory)
                     * are NOT injected — instead their module alias is added to the
                     * rewrite mapping so qualified references like types.Item get
                     * rewritten to mylib.Item → resolves as mylib_Item. */
                    {
                        char cur_dir[PATH_BUF_SIZE];
                        strncpy(cur_dir, cur_file_path, sizeof(cur_dir) - 1);
                        cur_dir[sizeof(cur_dir) - 1] = '\0';
                        char *cd_slash = strrchr(cur_dir, '/');
                        if (cd_slash) *(cd_slash + 1) = '\0';
                        else { cur_dir[0] = '.'; cur_dir[1] = '/'; cur_dir[2] = '\0'; }
                        const char *src_dir = arena_strdup(arena, cur_dir);

                        /* Normalize the directory being imported for sibling detection */
                        char *norm_import_dir = realpath(import_path, NULL);

                        for (int ti = 0; ti < imp_program->data.program.stmt_count; ti++) {
                            AstNode *ts = imp_program->data.program.stmts[ti];
                            if (ts->kind != NODE_IMPORT_STMT) continue;

                            bool all_sibling = true;
                            for (int xi = 0; xi < ts->data.import_stmt.count; xi++) {
                                ImportItem *titem = &ts->data.import_stmt.items[xi];
                                if (titem->is_stdlib || titem->is_c_import) {
                                    all_sibling = false;
                                    continue;
                                }
                                if (!titem->path) continue;

                                /* Resolve the transitive import path */
                                const char *trel = titem->path;
                                if (trel[0] == '.' && trel[1] == '/') trel += 2;
                                char tres[PATH_BUF_SIZE];
                                snprintf(tres, sizeof(tres), "%s%s", src_dir, trel);

                                /* Check if it resolves to a file inside the same directory */
                                size_t trlen = strlen(tres);
                                bool is_sibling = false;
                                /* Try with .ez extension if not already present */
                                char tres_ez[PATH_BUF_SIZE];
                                const char *tres_check = tres;
                                if (trlen < 3 || strcmp(tres + trlen - 3, ".ez") != 0) {
                                    snprintf(tres_ez, sizeof(tres_ez), "%s.ez", tres);
                                    tres_check = tres_ez;
                                }
                                char *norm_tres = realpath(tres_check, NULL);
                                if (norm_tres && norm_import_dir) {
                                    /* Check if the file's directory matches import_path */
                                    char *tslash = strrchr(norm_tres, '/');
                                    if (tslash) {
                                        size_t dir_len = (size_t)(tslash - norm_tres);
                                        size_t imp_dir_len = strlen(norm_import_dir);
                                        /* Strip trailing slash from norm_import_dir for comparison */
                                        if (imp_dir_len > 0 && norm_import_dir[imp_dir_len - 1] == '/')
                                            imp_dir_len--;
                                        if (dir_len == imp_dir_len &&
                                            strncmp(norm_tres, norm_import_dir, dir_len) == 0) {
                                            is_sibling = true;
                                        }
                                    }
                                }
                                free(norm_tres);

                                if (is_sibling) {
                                    /* Collect sibling alias for compound mapping generation later */
                                    const char *sib_alias = titem->alias;
                                    if (!sib_alias) {
                                        /* Derive alias from path (filename without .ez) */
                                        const char *sib_base = trel;
                                        const char *sib_slash = strrchr(trel, '/');
                                        if (sib_slash) sib_base = sib_slash + 1;
                                        char sib_buf[EZ_MSG_BUF_SIZE];
                                        size_t sib_len = strlen(sib_base);
                                        if (sib_len > 3 && strcmp(sib_base + sib_len - 3, ".ez") == 0) {
                                            memcpy(sib_buf, sib_base, sib_len - 3);
                                            sib_buf[sib_len - 3] = '\0';
                                        } else {
                                            snprintf(sib_buf, sizeof(sib_buf), "%s", sib_base);
                                        }
                                        sib_alias = arena_strdup(arena, sib_buf);
                                    }
                                    /* Track unique sibling aliases */
                                    bool already_tracked = false;
                                    for (int sa = 0; sa < sibling_alias_count; sa++) {
                                        if (strcmp(sibling_aliases[sa], sib_alias) == 0) {
                                            already_tracked = true;
                                            break;
                                        }
                                    }
                                    if (!already_tracked && sibling_alias_count < EZ_MAX_IMPORTS) {
                                        sibling_aliases[sibling_alias_count++] = sib_alias;
                                    }
                                    /* Null out the sibling import path so it's not injected */
                                    titem->path = NULL;
                                } else {
                                    all_sibling = false;
                                    if (!titem->source_dir) {
                                        titem->source_dir = src_dir;
                                    }
                                }
                            }

                            /* Only inject import stmt if it has non-sibling items */
                            if (!all_sibling) {
                                if (program->data.program.stmt_count >= program->data.program.stmt_cap) {
                                    int nc = program->data.program.stmt_cap * 2;
                                    AstNode **ns = arena_alloc(arena, sizeof(AstNode *) * nc);
                                    memcpy(ns, program->data.program.stmts,
                                           sizeof(AstNode *) * program->data.program.stmt_count);
                                    program->data.program.stmts = ns;
                                    program->data.program.stmt_cap = nc;
                                }
                                program->data.program.stmts[program->data.program.stmt_count++] = ts;
                            }
                        }
                        free(norm_import_dir);
                    }

                    /* Collect declaration names from this file into the combined mapping */
                    for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                        AstNode *s = imp_program->data.program.stmts[mi];
                        const char *oname = NULL;
                        if (s->kind == NODE_FUNC_DECL) oname = s->data.func_decl.name;
                        else if (s->kind == NODE_VAR_DECL) oname = s->data.var_decl.name;
                        else if (s->kind == NODE_STRUCT_DECL) oname = s->data.struct_decl.name;
                        else if (s->kind == NODE_ENUM_DECL) oname = s->data.enum_decl.name;
                        if (oname && name_count < EZ_MAX_IMPORTS) {
                            orig_names[name_count] = oname;
                            char *pn = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                            snprintf(pn, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, oname);
                            new_names[name_count] = pn;
                            name_count++;
                        }
                    }

                    /* Store parsed program for the rewrite pass */
                    parsed_programs[parsed_count] = imp_program;
                    parsed_paths[parsed_count] = cur_file_path;
                    parsed_count++;
                }

                /* Generate compound mappings for sibling imports.
                 * The parser converts types.Item → types_Item at parse time, so we
                 * need mappings like types_Item → mylib_Item for each alias × name. */
                {
                    int base_name_count = name_count;
                    for (int sa = 0; sa < sibling_alias_count; sa++) {
                        const char *alias = sibling_aliases[sa];
                        for (int ni = 0; ni < base_name_count; ni++) {
                            if (name_count >= EZ_MAX_IMPORTS) break;
                            /* Build "alias_OrigName" and map to "mod_OrigName" */
                            char *compound = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                            snprintf(compound, EZ_MSG_BUF_SIZE, "%s_%s", alias, orig_names[ni]);
                            /* Avoid duplicates */
                            bool dup = false;
                            for (int ci = 0; ci < name_count; ci++) {
                                if (strcmp(orig_names[ci], compound) == 0) { dup = true; break; }
                            }
                            if (dup) continue;
                            orig_names[name_count] = compound;
                            new_names[name_count] = new_names[ni]; /* same as mod_OrigName */
                            name_count++;
                        }
                        /* Also add bare alias → mod_name for label references */
                        if (name_count < EZ_MAX_IMPORTS) {
                            bool dup = false;
                            for (int ci = 0; ci < name_count; ci++) {
                                if (strcmp(orig_names[ci], alias) == 0) { dup = true; break; }
                            }
                            if (!dup) {
                                orig_names[name_count] = alias;
                                new_names[name_count] = mod_name;
                                name_count++;
                            }
                        }
                    }
                }

                /* Rewrite+merge pass: use combined mapping to rewrite ALL files */
                for (int pi = 0; pi < parsed_count; pi++) {
                    AstNode *imp_program = parsed_programs[pi];

                /* Merge imported declarations into main program.
                 * Two passes: var_decls first (so they're in scope for function bodies),
                 * then everything else. */

                /* Pass 1: variable declarations */
                for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                    AstNode *imp_stmt = imp_program->data.program.stmts[mi];
                    if (imp_stmt->kind != NODE_VAR_DECL) continue;

                    if (!imp_stmt->data.var_decl.mutable) {
                        imp_stmt->data.var_decl.original_name = imp_stmt->data.var_decl.name;
                        char *prefixed = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                        snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, imp_stmt->data.var_decl.name);
                        imp_stmt->data.var_decl.name = prefixed;
                        if (imp_stmt->data.var_decl.type_name) {
                            for (int ni = 0; ni < name_count; ni++) {
                                if (strcmp(imp_stmt->data.var_decl.type_name, orig_names[ni]) == 0) {
                                    imp_stmt->data.var_decl.type_name = new_names[ni];
                                    break;
                                }
                            }
                        }
                    } else {
                        imp_stmt->data.var_decl.original_name = imp_stmt->data.var_decl.name;
                        char *prefixed = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                        snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, imp_stmt->data.var_decl.name);
                        imp_stmt->data.var_decl.name = prefixed;
                        imp_stmt->data.var_decl.is_private = true;
                    }
                    /* Rewrite initializer expressions */
                    rewrite_labels(imp_stmt->data.var_decl.value, orig_names, new_names, name_count, arena);

                    /* Insert at beginning */
                    if (program->data.program.stmt_count >= program->data.program.stmt_cap) {
                        int new_cap = program->data.program.stmt_cap * 2;
                        AstNode **new_stmts = arena_alloc(arena, sizeof(AstNode *) * new_cap);
                        memcpy(new_stmts, program->data.program.stmts,
                               sizeof(AstNode *) * program->data.program.stmt_count);
                        program->data.program.stmts = new_stmts;
                        program->data.program.stmt_cap = new_cap;
                    }
                    int insert_at = 0;
                    for (int k = 0; k < program->data.program.stmt_count; k++) {
                        if (program->data.program.stmts[k]->kind == NODE_IMPORT_STMT ||
                            program->data.program.stmts[k]->kind == NODE_USING_STMT) {
                            insert_at = k + 1;
                        } else break;
                    }
                    memmove(&program->data.program.stmts[insert_at + 1],
                            &program->data.program.stmts[insert_at],
                            sizeof(AstNode *) * (program->data.program.stmt_count - insert_at));
                    program->data.program.stmts[insert_at] = imp_stmt;
                    program->data.program.stmt_count++;
                }

                /* Pass 2: functions, structs, enums */
                for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                    AstNode *imp_stmt = imp_program->data.program.stmts[mi];
                    /* Skip import/using/module/var declarations (vars handled in Pass 1) */
                    if (imp_stmt->kind == NODE_IMPORT_STMT ||
                        imp_stmt->kind == NODE_USING_STMT ||
                        imp_stmt->kind == NODE_MODULE_DECL ||
                        imp_stmt->kind == NODE_VAR_DECL) continue;

                    /* Prefix function names and rewrite body + type references */
                    if (imp_stmt->kind == NODE_FUNC_DECL) {
                        imp_stmt->data.func_decl.original_name = imp_stmt->data.func_decl.name;
                        char *prefixed = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                        snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, imp_stmt->data.func_decl.name);
                        imp_stmt->data.func_decl.name = prefixed;
                        /* Rewrite internal references in function body */
                        rewrite_labels(imp_stmt->data.func_decl.body, orig_names, new_names, name_count, arena);
                        /* Rewrite return type references */
                        for (int ri = 0; ri < imp_stmt->data.func_decl.return_type_count; ri++) {
                            const char *rt = imp_stmt->data.func_decl.return_types[ri];
                            for (int ni = 0; ni < name_count; ni++) {
                                if (strcmp(rt, orig_names[ni]) == 0) {
                                    imp_stmt->data.func_decl.return_types[ri] = new_names[ni];
                                    break;
                                }
                            }
                        }
                        /* Rewrite parameter type references */
                        for (int pi = 0; pi < imp_stmt->data.func_decl.param_count; pi++) {
                            const char *pt = imp_stmt->data.func_decl.params[pi].type_name;
                            if (!pt) continue;
                            for (int ni = 0; ni < name_count; ni++) {
                                if (strcmp(pt, orig_names[ni]) == 0) {
                                    imp_stmt->data.func_decl.params[pi].type_name = new_names[ni];
                                    break;
                                }
                            }
                        }
                    }

                    /* Prefix struct names and rewrite field type references */
                    if (imp_stmt->kind == NODE_STRUCT_DECL) {
                        imp_stmt->data.struct_decl.original_name = imp_stmt->data.struct_decl.name;
                        char *prefixed = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                        snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, imp_stmt->data.struct_decl.name);
                        imp_stmt->data.struct_decl.name = prefixed;
                        /* Rewrite field types that reference other imported types */
                        for (int fld = 0; fld < imp_stmt->data.struct_decl.field_count; fld++) {
                            const char *ft = imp_stmt->data.struct_decl.fields[fld].type_name;
                            if (!ft) continue;
                            for (int ni = 0; ni < name_count; ni++) {
                                if (strcmp(ft, orig_names[ni]) == 0) {
                                    imp_stmt->data.struct_decl.fields[fld].type_name = new_names[ni];
                                    break;
                                }
                            }
                        }
                        /* Rewrite struct-namespaced function bodies, return types, and param types */
                        for (int si = 0; si < imp_stmt->data.struct_decl.func_count; si++) {
                            AstNode *fn = imp_stmt->data.struct_decl.funcs[si].func_decl;
                            if (!fn || fn->kind != NODE_FUNC_DECL) continue;
                            /* Rewrite function body labels */
                            rewrite_labels(fn->data.func_decl.body, orig_names, new_names, name_count, arena);
                            /* Rewrite return types */
                            for (int ri = 0; ri < fn->data.func_decl.return_type_count; ri++) {
                                const char *rt = fn->data.func_decl.return_types[ri];
                                for (int ni = 0; ni < name_count; ni++) {
                                    if (strcmp(rt, orig_names[ni]) == 0) {
                                        fn->data.func_decl.return_types[ri] = new_names[ni];
                                        break;
                                    }
                                }
                            }
                            /* Rewrite parameter types */
                            for (int pi = 0; pi < fn->data.func_decl.param_count; pi++) {
                                const char *pt = fn->data.func_decl.params[pi].type_name;
                                if (!pt) continue;
                                for (int ni = 0; ni < name_count; ni++) {
                                    if (strcmp(pt, orig_names[ni]) == 0) {
                                        fn->data.func_decl.params[pi].type_name = new_names[ni];
                                        break;
                                    }
                                }
                            }
                        }
                    }

                    /* Prefix enum names with module name */
                    if (imp_stmt->kind == NODE_ENUM_DECL) {
                        imp_stmt->data.enum_decl.original_name = imp_stmt->data.enum_decl.name;
                        char *prefixed = arena_alloc(arena, EZ_MSG_BUF_SIZE);
                        snprintf(prefixed, EZ_MSG_BUF_SIZE, "%s_%s", mod_name, imp_stmt->data.enum_decl.name);
                        imp_stmt->data.enum_decl.name = prefixed;
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
                    /* Find insertion point: after imports/using/var_decls */
                    int insert_at = 0;
                    for (int k = 0; k < program->data.program.stmt_count; k++) {
                        if (program->data.program.stmts[k]->kind == NODE_IMPORT_STMT ||
                            program->data.program.stmts[k]->kind == NODE_USING_STMT ||
                            program->data.program.stmts[k]->kind == NODE_VAR_DECL) {
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
                }
                } /* end for (pi: rewrite+merge pass) */

                /* Mark this import item as fully processed so re-scanning
                 * the import list on the next while iteration doesn't
                 * trigger a spurious W2013 "already imported" warning. */
                item->path = NULL;
            }
        }
        } /* end while (found_new_import) */
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
        snprintf(default_output, PATH_BUF_SIZE, "/tmp/ez_run_%d", (int)getpid());
        output_file = default_output;
    } else if (!output_file) {
        default_output = output_name_from_input(input_file);
        output_file = default_output;
    }

    /* Write generated C to temp file */
    const char *out_base = strrchr(output_file, '/');
    out_base = out_base ? out_base + 1 : output_file;
    char c_file[PATH_BUF_SIZE];
    snprintf(c_file, sizeof(c_file), "/tmp/ez_%s.c", out_base);

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
        fprintf(stderr, "    - $EZ_RUNTIME environment variable\n");
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
    char extra_flags[EZ_TYPE_NAME_MAX] = "";
    if (debug_symbols) {
        snprintf(extra_flags, sizeof(extra_flags), "-g %s", opt_level);
    } else {
        snprintf(extra_flags, sizeof(extra_flags), "%s", opt_level);
    }

    clock_t t_cc_start = clock();

    if (has_archive) {
        snprintf(cmd, sizeof(cmd),
            "cc -std=c11 %s -Wall -Wno-unused-function -Wno-unused-variable -Wno-unused-but-set-variable "
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
        for (int i = 0; i < (int)(sizeof(runtime_srcs)/sizeof(runtime_srcs[0])); i++) {
            int w = snprintf(srcs + off, sizeof(srcs) - (size_t)off, "%s/%s ", runtime_dir, runtime_srcs[i]);
            if (w > 0 && (size_t)w < sizeof(srcs) - (size_t)off) off += w;
            else { off = (int)sizeof(srcs); break; }
        }
        for (int i = 0; i < (int)(sizeof(stdlib_srcs)/sizeof(stdlib_srcs[0])); i++) {
            int w = snprintf(srcs + off, sizeof(srcs) - (size_t)off, "%s/%s ", runtime_dir, stdlib_srcs[i]);
            if (w > 0 && (size_t)w < sizeof(srcs) - (size_t)off) off += w;
            else { off = (int)sizeof(srcs); break; }
        }

        snprintf(cmd, sizeof(cmd),
            "cc -std=c11 %s -Wall -Wno-unused-function -Wno-unused-variable -Wno-unused-but-set-variable "
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
        /* Check if this might be a C interop header issue */
        bool has_c_import = false;
        for (int si = 0; si < program->data.program.stmt_count; si++) {
            AstNode *s = program->data.program.stmts[si];
            if (s->kind == NODE_IMPORT_STMT) {
                for (int ii = 0; ii < s->data.import_stmt.count; ii++) {
                    if (s->data.import_stmt.items[ii].is_c_import) {
                        has_c_import = true;
                        break;
                    }
                }
            }
            if (has_c_import) break;
        }
        if (has_c_import) {
            fprintf(stderr, "ez: hint: check that all C headers in import c\"...\" exist and are installed\n");
        }
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
