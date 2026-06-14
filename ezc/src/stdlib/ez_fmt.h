/*
 * ez_fmt.h - @fmt module for EZ (formatted output)
 *
 * EZC-only module for C-style formatted I/O.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_FMT_H
#define EZ_FMT_H

#include "../runtime/ez_runtime.h"
#include <stdio.h>

/*
 * fmt.printf and fmt.sprintf are handled directly by the codegen —
 * the compiler translates EZ format strings to C printf format strings
 * at compile time, so no runtime function is needed for those.
 *
 * These helpers are for operations that need runtime support.
 */

/*@man printf
 *@module fmt
 *@group Output
 *@sig printf(format string, ...args T)
 *@desc Prints a formatted string to stdout. Uses C-style format directives: %d (int), %f (float), %s (string), %b (bool), %c (char). Accepts string, int, float, and bool arguments. Composite types are not supported.
 *@example
 *   import @fmt
 *   fmt.printf("hello %s, you are %d years old\n", "alice", 30)
 *@end
 */
/* fmt.printf — handled directly by codegen */

/*@man sprintf
 *@module fmt
 *@group Output
 *@sig sprintf(format string, ...args T) -> string
 *@desc Returns a formatted string without printing it. Uses the same format directives as printf.
 *@example
 *   import @fmt
 *   mut s string = fmt.sprintf("x = %d", 42)
 *   println(s)
 *@end
 */
/* fmt.sprintf — handled directly by codegen */

/*@man format
 *@module fmt
 *@group Output
 *@sig format(format string, ...args T) -> string
 *@desc Returns a formatted string. Alias for sprintf.
 *@example
 *   import @fmt
 *   mut s string = fmt.format("pi is %.2f", 3.14159)
 *   println(s)
 *@end
 */
/* fmt.format — handled via ez_string_format with ez_default_arena */

/*@man printfln
 *@module fmt
 *@group Output
 *@sig printfln(format string, ...args T)
 *@desc Prints a formatted string to stdout with a trailing newline. Uses the same format directives as printf.
 *@example
 *   import @fmt
 *   fmt.printfln("hello %s, you are %d years old", "alice", 30)
 *@end
 */
/* fmt.printfln — handled directly by codegen */

/*@man eprintf
 *@module fmt
 *@group Output
 *@sig eprintf(format string, ...args T)
 *@desc Prints a formatted string to stderr. Uses the same format directives as printf.
 *@example
 *   import @fmt
 *   fmt.eprintf("error: %s\n", "something went wrong")
 *@end
 */
/* fmt.eprintf — handled directly by codegen */

/*@man eprintfln
 *@module fmt
 *@group Output
 *@sig eprintfln(format string, ...args T)
 *@desc Prints a formatted string to stderr with a trailing newline. Uses the same format directives as printf.
 *@example
 *   import @fmt
 *   fmt.eprintfln("error: %s", "something went wrong")
 *@end
 */
/* fmt.eprintfln — handled directly by codegen */

/*@man sprintfln
 *@module fmt
 *@group Output
 *@sig sprintfln(format string, ...args T) -> string
 *@desc Returns a formatted string with a trailing newline. Uses the same format directives as sprintf.
 *@example
 *   import @fmt
 *   mut s string = fmt.sprintfln("x = %d", 42)
 *   println(s)
 *@end
 */
/* fmt.sprintfln — handled directly by codegen */

/*@man pad_left
 *@module fmt
 *@group Padding
 *@sig pad_left(s string, width int, ch char) -> string
 *@desc Returns s padded on the left with ch until the total length reaches width. Returns s unchanged if it is already at least width characters long.
 *@example
 *   import @fmt
 *   println(fmt.pad_left("42", 5, '0'))
 *@end
 */
EzString ez_fmt_pad_left(EzArena *arena, EzString s, int64_t width, char ch);

/*@man pad_right
 *@module fmt
 *@group Padding
 *@sig pad_right(s string, width int, ch char) -> string
 *@desc Returns s padded on the right with ch until the total length reaches width. Returns s unchanged if it is already at least width characters long.
 *@example
 *   import @fmt
 *   println(fmt.pad_right("hi", 6, '.'))
 *@end
 */
EzString ez_fmt_pad_right(EzArena *arena, EzString s, int64_t width, char ch);

/*@man center
 *@module fmt
 *@group Padding
 *@sig center(s string, width int, ch char) -> string
 *@desc Returns s centered within width, padded on both sides with ch. If padding is uneven, the extra character goes on the right.
 *@example
 *   import @fmt
 *   println(fmt.center("hi", 8, '-'))
 *@end
 */
EzString ez_fmt_center(EzArena *arena, EzString s, int64_t width, char ch);

/*@man int_to_hex
 *@module fmt
 *@group Number Formatting
 *@sig int_to_hex(n int) -> string
 *@desc Returns the integer n formatted as a lowercase hexadecimal string with no "0x" prefix.
 *@example
 *   import @fmt
 *   println(fmt.int_to_hex(255))
 *@end
 */
EzString ez_fmt_int_to_hex(EzArena *arena, int64_t n);

/*@man int_to_binary
 *@module fmt
 *@group Number Formatting
 *@sig int_to_binary(n int) -> string
 *@desc Returns the integer n formatted as a binary string with no "0b" prefix.
 *@example
 *   import @fmt
 *   println(fmt.int_to_binary(10))
 *@end
 */
EzString ez_fmt_int_to_binary(EzArena *arena, int64_t n);

/*@man int_to_octal
 *@module fmt
 *@group Number Formatting
 *@sig int_to_octal(n int) -> string
 *@desc Returns the integer n formatted as an octal string with no "0o" prefix.
 *@example
 *   import @fmt
 *   println(fmt.int_to_octal(8))
 *@end
 */
EzString ez_fmt_int_to_octal(EzArena *arena, int64_t n);

/*@man float_fixed
 *@module fmt
 *@group Number Formatting
 *@sig float_fixed(f float, decimals int) -> string
 *@desc Returns f formatted with exactly decimals digits after the decimal point.
 *@example
 *   import @fmt
 *   println(fmt.float_fixed(3.14159, 2))
 *@end
 */
EzString ez_fmt_float_fixed(EzArena *arena, double f, int64_t decimals);

/*@man float_sci
 *@module fmt
 *@group Number Formatting
 *@sig float_sci(f float) -> string
 *@desc Returns f formatted in scientific notation (e.g. "3.14e+00").
 *@example
 *   import @fmt
 *   println(fmt.float_sci(0.000123))
 *@end
 */
EzString ez_fmt_float_sci(EzArena *arena, double f);

#endif
