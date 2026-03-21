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

# Include paths for compiler source
INCLUDES = -Isrc

.PHONY: all build clean install test

all: build

build: $(BIN)

$(BIN): $(OBJ)
	$(CC) $(CFLAGS) $(INCLUDES) -o $@ $^ $(LDFLAGS)

%.o: %.c
	$(CC) $(CFLAGS) $(INCLUDES) -c -o $@ $<

clean:
	rm -f $(OBJ) $(BIN)

install: build
	install -d /usr/local/bin
	install -m 755 $(BIN) /usr/local/bin/
	install -d /usr/local/lib/ezc/runtime
	install -d /usr/local/lib/ezc/stdlib
	install -m 644 src/runtime/ez_runtime.h /usr/local/lib/ezc/runtime/
	install -m 644 src/runtime/ez_runtime.c /usr/local/lib/ezc/runtime/
	install -m 644 src/stdlib/ez_std.h /usr/local/lib/ezc/stdlib/
	install -m 644 src/stdlib/ez_std.c /usr/local/lib/ezc/stdlib/

test: build
	@echo "Running hello world test..."
	@echo 'import @std' > /tmp/ezc_test_hello.ez
	@echo 'using std' >> /tmp/ezc_test_hello.ez
	@echo '' >> /tmp/ezc_test_hello.ez
	@echo 'do main() {' >> /tmp/ezc_test_hello.ez
	@echo '    println("Hello, World!")' >> /tmp/ezc_test_hello.ez
	@echo '}' >> /tmp/ezc_test_hello.ez
	cd .. && ./ezc/ezc /tmp/ezc_test_hello.ez -o /tmp/ezc_test_hello
	@/tmp/ezc_test_hello
	@rm -f /tmp/ezc_test_hello /tmp/ezc_test_hello.ez
	@echo "Test passed!"
