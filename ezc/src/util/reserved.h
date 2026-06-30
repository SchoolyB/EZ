/*
 * reserved.h - Canonical reserved name lists for the EZ compiler
 *
 * Single authoritative source for every identifier the user may not shadow:
 * reserved type names, builtin function names, stdlib module names, and
 * stdlib struct names reserved by the runtime. Both the parser and typechecker
 * include this header so that additions require a single change.
 *
 * Every array must remain sorted in strcmp order (ASCII: digits, uppercase,
 * lowercase) for bsearch to work correctly.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_RESERVED_H
#define EZ_RESERVED_H

#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

static inline int ez_strptr_cmp(const void *a, const void *b) {
    return strcmp(*(const char *const *)a, *(const char *const *)b);
}

/* --- Reserved type names (STANDARD.md §2.5) --- */

static const char *const ez_reserved_type_names[] = {
    "Error",
    "SourceLocation",
    "bool",
    "byte",
    "char",
    "f32",
    "f64",
    "float",
    "func",
    "i128",
    "i16",
    "i256",
    "i32",
    "i64",
    "i8",
    "int",
    "map",
    "nil",
    "string",
    "u128",
    "u16",
    "u256",
    "u32",
    "u64",
    "u8",
    "uint",
    "void",
};

#define EZ_RESERVED_TYPE_NAMES_COUNT \
    ((int)(sizeof(ez_reserved_type_names) / sizeof(ez_reserved_type_names[0])))

static inline bool is_reserved_type_name(const char *name) {
    return bsearch(&name, ez_reserved_type_names,
                   (size_t)EZ_RESERVED_TYPE_NAMES_COUNT,
                   sizeof(const char *), ez_strptr_cmp) != NULL;
}

/* --- Builtin function names --- */

static const char *const ez_builtin_func_names[] = {
    "addr",
    "assert",
    "c_string",
    "cast",
    "char_count",
    "copy",
    "embed",
    "eprint",
    "eprintln",
    "error",
    "exit",
    "here",
    "input",
    "len",
    "panic",
    "print",
    "println",
    "ref",
    "size_of",
    "sleep_ms",
    "sleep_ns",
    "sleep_s",
    "to_char",
    "type_of",
};

#define EZ_BUILTIN_FUNC_NAMES_COUNT \
    ((int)(sizeof(ez_builtin_func_names) / sizeof(ez_builtin_func_names[0])))

static inline bool is_reserved_builtin_func_name(const char *name) {
    return bsearch(&name, ez_builtin_func_names,
                   (size_t)EZ_BUILTIN_FUNC_NAMES_COUNT,
                   sizeof(const char *), ez_strptr_cmp) != NULL;
}

/* --- Standard library module names --- */

static const char *const ez_stdlib_module_names[] = {
    "arrays",
    "atomic",
    "binary",
    "bytes",
    "channels",
    "crypto",
    "csv",
    "encoding",
    "fmt",
    "http",
    "io",
    "json",
    "maps",
    "math",
    "mem",
    "net",
    "os",
    "random",
    "regex",
    "server",
    "sqlite",
    "strconv",
    "strings",
    "sync",
    "threads",
    "time",
    "uuid",
};

#define EZ_STDLIB_MODULE_NAMES_COUNT \
    ((int)(sizeof(ez_stdlib_module_names) / sizeof(ez_stdlib_module_names[0])))

static inline bool is_stdlib_module_name(const char *name) {
    return bsearch(&name, ez_stdlib_module_names,
                   (size_t)EZ_STDLIB_MODULE_NAMES_COUNT,
                   sizeof(const char *), ez_strptr_cmp) != NULL;
}

/* --- Reserved stdlib struct names (map to internal C types) --- */

static const char *const ez_reserved_stdlib_struct_names[] = {
    "Channel",
    "Database",
    "HttpRequest",
    "HttpResponse",
    "Listener",
    "Mutex",
    "Router",
    "Socket",
    "SpinLock",
    "Thread",
    "UUID",
};

#define EZ_RESERVED_STDLIB_STRUCT_NAMES_COUNT \
    ((int)(sizeof(ez_reserved_stdlib_struct_names) / sizeof(ez_reserved_stdlib_struct_names[0])))

static inline bool is_reserved_stdlib_struct_name(const char *name) {
    return bsearch(&name, ez_reserved_stdlib_struct_names,
                   (size_t)EZ_RESERVED_STDLIB_STRUCT_NAMES_COUNT,
                   sizeof(const char *), ez_strptr_cmp) != NULL;
}

#endif /* EZ_RESERVED_H */
