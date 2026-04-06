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
        rewrite_labels(node->data.var_decl.value, orig, prefixed, count, arena);
        break;
    case NODE_ASSIGN_STMT:
        rewrite_labels(node->data.assign.target, orig, prefixed, count, arena);
        rewrite_labels(node->data.assign.value, orig, prefixed, count, arena);
        break;
    case NODE_IF_STMT:
        rewrite_labels(node->data.if_stmt.condition, orig, prefixed, count, arena);
        if (node->data.if_stmt.consequence) {
            for (int i = 0; i < node->data.if_stmt.consequence->data.block.count; i++)
                rewrite_labels(node->data.if_stmt.consequence->data.block.stmts[i], orig, prefixed, count, arena);
        }
        if (node->data.if_stmt.alternative) {
            for (int i = 0; i < node->data.if_stmt.alternative->data.block.count; i++)
                rewrite_labels(node->data.if_stmt.alternative->data.block.stmts[i], orig, prefixed, count, arena);
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
    case NODE_CAST_EXPR:
        rewrite_labels(node->data.cast.value, orig, prefixed, count, arena);
        break;
    default: break;
    }
}

/* Import cache: track already-imported files to avoid duplicates and cycles */
static const char *imported_files[256];
static int imported_file_count = 0;

static bool already_imported(const char *path) {
    for (int i = 0; i < imported_file_count; i++) {
        if (strcmp(imported_files[i], path) == 0) return true;
    }
    return false;
}

static void mark_imported(const char *path) {
    if (imported_file_count < 256) {
        imported_files[imported_file_count++] = path;
    }
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
        /* Mark the main file as already imported (prevents circular import loops) */
        mark_imported(input_file);

        /* Derive main file's module name for circular import resolution */
        const char *main_base = input_file;
        const char *main_slash = strrchr(input_file, '/');
        if (main_slash) main_base = main_slash + 1;
        char main_mod_buf[256];
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
        else strcpy(input_dir, "./");

        /* Collect all import statements first (inserting during iteration shifts indices) */
        AstNode *import_stmts[256];
        int import_stmt_count = 0;
        for (int si = 0; si < program->data.program.stmt_count; si++) {
            if (program->data.program.stmts[si]->kind == NODE_IMPORT_STMT &&
                import_stmt_count < 256) {
                import_stmts[import_stmt_count++] = program->data.program.stmts[si];
            }
        }

        for (int si = 0; si < import_stmt_count; si++) {
            AstNode *stmt = import_stmts[si];

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
                const char *mod_name = item->alias ? item->alias : arena_strdup(arena, mod_name_buf);

                /* Module name collision detection — only for DIRECT imports from the same file.
                 * Don't collide with the main file's own module name. */
                static const char *seen_modules[256];
                static const char *seen_paths[256];
                static int seen_count = 0;
                bool collision = false;
                for (int sm = 0; sm < seen_count; sm++) {
                    if (strcmp(seen_modules[sm], mod_name) == 0 &&
                        strcmp(seen_paths[sm], import_path) != 0) {
                        char msg[256];
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
                if (seen_count < 256) {
                    seen_modules[seen_count] = mod_name;
                    seen_paths[seen_count] = import_path;
                    seen_count++;
                }

                /* Set the alias if not already set */
                if (!item->alias) item->alias = mod_name;
                if (!item->module) item->module = mod_name;

                /* Skip if already imported (handles cycles and duplicates) */
                if (already_imported(import_path)) continue;
                mark_imported(arena_strdup(arena, import_path));

                /* Read and parse the imported file */
                char *imp_source = read_file(import_path);
                if (!imp_source) {
                    char msg[512];
                    snprintf(msg, sizeof(msg), "cannot find imported file '%s'", import_path);
                    diag_error(diag, "E6001", strdup(msg),
                        input_file, stmt->token.line, stmt->token.column, 0);
                    continue;
                }

                Lexer *imp_lexer = lexer_create(arena, imp_source, import_path);
                Parser *imp_parser = parser_create(arena, imp_lexer, import_path, diag);
                AstNode *imp_program = parser_parse_program(imp_parser);

                if (!imp_program || diag_has_errors(diag)) continue;

                /* Recursively process imports within the imported file */
                {
                    char imp_dir[PATH_BUF_SIZE];
                    strncpy(imp_dir, import_path, sizeof(imp_dir) - 1);
                    imp_dir[sizeof(imp_dir) - 1] = '\0';
                    char *imp_slash = strrchr(imp_dir, '/');
                    if (imp_slash) *(imp_slash + 1) = '\0';
                    else strcpy(imp_dir, "./");

                    for (int ti = 0; ti < imp_program->data.program.stmt_count; ti++) {
                        AstNode *ts = imp_program->data.program.stmts[ti];
                        if (ts->kind != NODE_IMPORT_STMT) continue;
                        for (int tj = 0; tj < ts->data.import_stmt.count; tj++) {
                            ImportItem *titem = &ts->data.import_stmt.items[tj];
                            if (titem->is_stdlib || !titem->path) continue;
                            const char *trel = titem->path;
                            if (trel[0] == '.' && trel[1] == '/') trel += 2;
                            char trans_path[PATH_BUF_SIZE];
                            snprintf(trans_path, sizeof(trans_path), "%s%s", imp_dir, trel);
                            if (already_imported(trans_path)) continue;
                            mark_imported(arena_strdup(arena, trans_path));

                            char *tsrc = read_file(trans_path);
                            if (!tsrc) continue;
                            /* Derive transitive module name */
                            const char *tbase = trel;
                            const char *tslash = strrchr(trel, '/');
                            if (tslash) tbase = tslash + 1;
                            char tmod_buf[256];
                            size_t tblen = strlen(tbase);
                            if (tblen > 3 && strcmp(tbase + tblen - 3, ".ez") == 0) {
                                memcpy(tmod_buf, tbase, tblen - 3);
                                tmod_buf[tblen - 3] = '\0';
                            } else {
                                snprintf(tmod_buf, sizeof(tmod_buf), "%s", tbase);
                            }
                            const char *tmod = titem->alias ? titem->alias : arena_strdup(arena, tmod_buf);
                            if (!titem->alias) { titem->alias = tmod; titem->module = tmod; }

                            Lexer *tlex = lexer_create(arena, tsrc, trans_path);
                            Parser *tpar = parser_create(arena, tlex, trans_path, diag);
                            AstNode *tprog = parser_parse_program(tpar);
                            if (!tprog || diag_has_errors(diag)) continue;

                            /* Merge transitive declarations — prefix with transitive module name,
                             * but these are only visible within the importing file, NOT the main file */
                            for (int tk = 0; tk < tprog->data.program.stmt_count; tk++) {
                                AstNode *tdecl = tprog->data.program.stmts[tk];
                                if (tdecl->kind == NODE_IMPORT_STMT ||
                                    tdecl->kind == NODE_USING_STMT ||
                                    tdecl->kind == NODE_MODULE_DECL) continue;
                                if (tdecl->kind == NODE_VAR_DECL && tdecl->data.var_decl.mutable) continue;
                                if (tdecl->kind == NODE_FUNC_DECL && tdecl->data.func_decl.is_private) continue;
                                if (tdecl->kind == NODE_VAR_DECL && tdecl->data.var_decl.is_private) continue;

                                if (tdecl->kind == NODE_FUNC_DECL) {
                                    char *tp = arena_alloc(arena, 256);
                                    snprintf(tp, 256, "%s_%s", tmod, tdecl->data.func_decl.name);
                                    tdecl->data.func_decl.name = tp;
                                }
                                if (tdecl->kind == NODE_VAR_DECL && !tdecl->data.var_decl.mutable) {
                                    char *tp = arena_alloc(arena, 256);
                                    snprintf(tp, 256, "%s_%s", tmod, tdecl->data.var_decl.name);
                                    tdecl->data.var_decl.name = tp;
                                }
                                if (tdecl->kind == NODE_STRUCT_DECL) {
                                    char *tp = arena_alloc(arena, 256);
                                    snprintf(tp, 256, "%s_%s", tmod, tdecl->data.struct_decl.name);
                                    tdecl->data.struct_decl.name = tp;
                                }
                                if (tdecl->kind == NODE_ENUM_DECL) {
                                    char *tp = arena_alloc(arena, 256);
                                    snprintf(tp, 256, "%s_%s", tmod, tdecl->data.enum_decl.name);
                                    tdecl->data.enum_decl.name = tp;
                                }

                                /* Insert into main program at the beginning */
                                if (program->data.program.stmt_count >= program->data.program.stmt_cap) {
                                    int nc = program->data.program.stmt_cap * 2;
                                    AstNode **ns = arena_alloc(arena, sizeof(AstNode *) * nc);
                                    memcpy(ns, program->data.program.stmts,
                                           sizeof(AstNode *) * program->data.program.stmt_count);
                                    program->data.program.stmts = ns;
                                    program->data.program.stmt_cap = nc;
                                }
                                memmove(program->data.program.stmts + 1, program->data.program.stmts,
                                    sizeof(AstNode *) * program->data.program.stmt_count);
                                program->data.program.stmts[0] = tdecl;
                                program->data.program.stmt_count++;
                            }
                        }
                    }
                }

                /* Collect original names before prefixing */
                const char *orig_names[256];
                const char *new_names[256];
                int name_count = 0;
                for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                    AstNode *s = imp_program->data.program.stmts[mi];
                    const char *oname = NULL;
                    if (s->kind == NODE_FUNC_DECL) oname = s->data.func_decl.name;
                    else if (s->kind == NODE_VAR_DECL) oname = s->data.var_decl.name;
                    else if (s->kind == NODE_STRUCT_DECL) oname = s->data.struct_decl.name;
                    else if (s->kind == NODE_ENUM_DECL) oname = s->data.enum_decl.name;
                    if (oname && name_count < 256) {
                        orig_names[name_count] = oname;
                        char *pn = arena_alloc(arena, 256);
                        snprintf(pn, 256, "%s_%s", mod_name, oname);
                        new_names[name_count] = pn;
                        name_count++;
                    }
                }

                /* Merge imported declarations into main program.
                 * Two passes: var_decls first (so they're in scope for function bodies),
                 * then everything else. */

                /* Pass 1: variable declarations */
                for (int mi = 0; mi < imp_program->data.program.stmt_count; mi++) {
                    AstNode *imp_stmt = imp_program->data.program.stmts[mi];
                    if (imp_stmt->kind != NODE_VAR_DECL) continue;

                    if (!imp_stmt->data.var_decl.mutable) {
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.var_decl.name);
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
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.var_decl.name);
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
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.func_decl.name);
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
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.struct_decl.name);
                        imp_stmt->data.struct_decl.name = prefixed;
                        /* Rewrite field types that reference other imported types */
                        for (int fi = 0; fi < imp_stmt->data.struct_decl.field_count; fi++) {
                            const char *ft = imp_stmt->data.struct_decl.fields[fi].type_name;
                            if (!ft) continue;
                            for (int ni = 0; ni < name_count; ni++) {
                                if (strcmp(ft, orig_names[ni]) == 0) {
                                    imp_stmt->data.struct_decl.fields[fi].type_name = new_names[ni];
                                    break;
                                }
                            }
                        }
                    }

                    /* Prefix enum names with module name */
                    if (imp_stmt->kind == NODE_ENUM_DECL) {
                        char *prefixed = arena_alloc(arena, 256);
                        snprintf(prefixed, 256, "%s_%s", mod_name, imp_stmt->data.enum_decl.name);
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
