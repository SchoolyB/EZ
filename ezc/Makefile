# EZC Makefile
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

CC ?= cc
CFLAGS = -std=c11 -Wall -Wextra -Wpedantic -Wno-unused-parameter
LDFLAGS = -lm

# Build mode
ifdef DEBUG
	CFLAGS += -g -O0 -DDEBUG
else
	CFLAGS += -O2
endif

# Source files (compiler only, not runtime/stdlib which are compiled into user binaries)
SRC = src/main.c \
      src/util/arena.c \
      src/util/buf.c \
      src/util/error.c \
      src/lexer/token.c \
      src/lexer/lexer.c \
      src/parser/ast.c \
      src/parser/parser.c \
      src/typechecker/types.c \
      src/typechecker/scope.c \
      src/typechecker/typechecker.c \
      src/codegen/codegen.c

OBJ = $(SRC:.c=.o)
BIN = ezc

# Runtime/stdlib files compiled into a static library for fast linking
RT_SRC = src/runtime/ez_runtime.c \
         src/runtime/ez_array.c \
         src/runtime/ez_map.c \
         src/stdlib/ez_std.c \
         src/stdlib/ez_mem.c \
         src/stdlib/ez_fmt.c \
         src/stdlib/ez_math.c \
         src/stdlib/ez_strings.c \
         src/stdlib/ez_io.c \
         src/stdlib/ez_maps.c \
         src/stdlib/ez_os.c \
         src/stdlib/ez_arrays.c \
         src/stdlib/ez_random.c \
         src/stdlib/ez_time.c \
         src/stdlib/ez_uuid.c \
         src/stdlib/ez_encoding.c \
         src/stdlib/ez_crypto.c \
         src/stdlib/ez_bytes.c \
         src/stdlib/ez_binary.c \
         src/stdlib/ez_csv.c \
         src/stdlib/ez_json.c \
         src/stdlib/ez_sqlite.c \
         src/stdlib/ez_threads.c \
         src/stdlib/ez_regex.c \
         src/stdlib/ez_net.c \
         src/stdlib/ez_http.c \
         src/stdlib/ez_server.c \
         src/vendor/sqlite3.c
RT_OBJ = $(RT_SRC:.c=.o)
RT_LIB = libezrt.a

# Include paths for compiler source
INCLUDES = -Isrc

.PHONY: all build clean install test

all: build

build: $(BIN) $(RT_LIB)

$(BIN): $(OBJ)
	$(CC) $(CFLAGS) $(INCLUDES) -o $@ $^ $(LDFLAGS)

$(RT_LIB): $(RT_OBJ)
	ar rcs $@ $^

# Runtime objects need different flags (no -Wpedantic for GCC statement expressions)
src/runtime/%.o: src/runtime/%.c
	$(CC) -std=c11 -D_POSIX_C_SOURCE=200809L -Wall -Wextra -Wno-unused-parameter -O2 -Isrc -c -o $@ $<

src/stdlib/%.o: src/stdlib/%.c
	$(CC) -std=c11 -D_POSIX_C_SOURCE=200809L -Wall -Wextra -Wno-unused-parameter -O2 -Isrc -c -o $@ $<

# SQLite amalgamation — relaxed warnings, single-threaded mode
src/vendor/sqlite3.o: src/vendor/sqlite3.c
	$(CC) -std=c11 -w -O2 -DSQLITE_THREADSAFE=0 -c -o $@ $<

%.o: %.c
	$(CC) $(CFLAGS) $(INCLUDES) -c -o $@ $<

clean:
	rm -f $(OBJ) $(RT_OBJ) $(BIN) $(RT_LIB) tests/test_lexer tests/test_parser tests/test_typechecker tests/test_codegen tests/*_san

install: build
	install -d /usr/local/bin
	install -m 755 $(BIN) /usr/local/bin/
	install -d /usr/local/lib/ezc/runtime
	install -d /usr/local/lib/ezc/stdlib
	install -m 644 src/runtime/ez_runtime.h /usr/local/lib/ezc/runtime/
	install -m 644 src/runtime/ez_array.h /usr/local/lib/ezc/runtime/
	install -m 644 src/runtime/ez_map.h /usr/local/lib/ezc/runtime/
	install -m 644 src/stdlib/ez_std.h /usr/local/lib/ezc/stdlib/
	install -m 644 src/stdlib/ez_mem.h /usr/local/lib/ezc/stdlib/
	install -m 644 $(RT_LIB) /usr/local/lib/ezc/

# Shared objects needed by test binaries (everything except main.o)
TEST_DEPS = src/util/arena.o src/util/buf.o src/util/error.o \
            src/lexer/token.o src/lexer/lexer.o \
            src/parser/ast.o src/parser/parser.o \
            src/typechecker/types.o src/typechecker/scope.o src/typechecker/typechecker.o

tests/test_lexer: tests/test_lexer.c $(TEST_DEPS)
	$(CC) $(CFLAGS) $(INCLUDES) -o $@ $^ $(LDFLAGS)

tests/test_parser: tests/test_parser.c $(TEST_DEPS)
	$(CC) $(CFLAGS) $(INCLUDES) -o $@ $^ $(LDFLAGS)

tests/test_typechecker: tests/test_typechecker.c $(TEST_DEPS)
	$(CC) $(CFLAGS) $(INCLUDES) -o $@ $^ $(LDFLAGS)

tests/test_codegen: tests/test_codegen.c
	$(CC) $(CFLAGS) -D_POSIX_C_SOURCE=200809L $(INCLUDES) -o $@ $< $(LDFLAGS)

test-unit: $(TEST_DEPS) tests/test_lexer tests/test_parser tests/test_typechecker
	@echo ""
	@echo "=== Lexer Tests ==="
	@./tests/test_lexer
	@echo "=== Parser Tests ==="
	@./tests/test_parser
	@echo "=== Type Checker Tests ==="
	@./tests/test_typechecker

test-e2e: build tests/test_codegen
	@echo ""
	@echo "=== End-to-End Codegen Tests ==="
	@./tests/test_codegen

test-parity: build
	@./tests/run_parity.sh

test-integration: build
	@./tests/run_integration.sh

# Sanitizer builds — catch undefined behavior (and memory bugs on Linux)
# Must compile ALL source files with sanitizer flags
TEST_SRC_FILES = src/util/arena.c src/util/buf.c src/util/error.c \
                 src/lexer/token.c src/lexer/lexer.c \
                 src/parser/ast.c src/parser/parser.c \
                 src/typechecker/types.c src/typechecker/scope.c src/typechecker/typechecker.c

# UBSan works everywhere; ASan needs Linux for reliable CI
UBSAN_FLAGS = -fsanitize=undefined -fno-omit-frame-pointer -g -O1
ASAN_FLAGS = -fsanitize=address,undefined -fno-omit-frame-pointer -g -O1

test-ubsan:
	@echo ""
	@echo "=== UBSan Lexer Tests ==="
	@$(CC) $(UBSAN_FLAGS) $(INCLUDES) -o tests/test_lexer_san tests/test_lexer.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_lexer_san
	@echo "=== UBSan Parser Tests ==="
	@$(CC) $(UBSAN_FLAGS) $(INCLUDES) -o tests/test_parser_san tests/test_parser.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_parser_san
	@echo "=== UBSan Type Checker Tests ==="
	@$(CC) $(UBSAN_FLAGS) $(INCLUDES) -o tests/test_typechecker_san tests/test_typechecker.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_typechecker_san
	@rm -f tests/test_lexer_san tests/test_parser_san tests/test_typechecker_san
	@echo ""
	@echo "All UBSan tests passed — no undefined behavior detected."

test-asan:
	@echo ""
	@echo "=== ASan+UBSan Tests (Linux recommended) ==="
	@$(CC) $(ASAN_FLAGS) $(INCLUDES) -o tests/test_lexer_san tests/test_lexer.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_lexer_san
	@$(CC) $(ASAN_FLAGS) $(INCLUDES) -o tests/test_parser_san tests/test_parser.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_parser_san
	@$(CC) $(ASAN_FLAGS) $(INCLUDES) -o tests/test_typechecker_san tests/test_typechecker.c $(TEST_SRC_FILES) $(LDFLAGS)
	@./tests/test_typechecker_san
	@rm -f tests/test_lexer_san tests/test_parser_san tests/test_typechecker_san
	@echo ""
	@echo "All ASan+UBSan tests passed — no memory errors or undefined behavior."

# Format check
format-check:
	@echo "Checking C code formatting..."
	@find src -name '*.c' -o -name '*.h' | xargs clang-format --dry-run --Werror 2>&1 && echo "All files formatted correctly" || echo "Formatting issues found — run 'make format' to fix"

format:
	@find src -name '*.c' -o -name '*.h' | xargs clang-format -i
	@echo "Formatted all C files"

test: test-unit test-e2e test-parity
