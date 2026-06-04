/*
 * ez_strconv.h - strconv module for EZ (string/type conversions)
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STRCONV_H
#define EZ_STRCONV_H

#include "../runtime/ez_runtime.h"

/* Result types for fallible conversions */
#ifndef EZRESULT_BOOL_DEFINED
#define EZRESULT_BOOL_DEFINED
typedef struct { bool v0; EzError *v1; } EzResult_bool;
#endif
typedef struct { uint64_t v0; EzError *v1; } EzResult_uint;
typedef struct { double v0; EzError *v1; } EzResult_float;

/*@man to_int
 *@module strconv
 *@group Parsing
 *@sig to_int(s string, base int = 10) -> (int, Error)
 *@desc Parses s as a signed integer in the given base (2–36). With single-var assignment, panics on invalid input. With destructuring, returns an Error instead. Leading/trailing whitespace is not tolerated.
 *@example
 *   import @strconv
 *   mut n int = strconv.to_int("42")
 *   mut val, err = strconv.to_int("ff", strconv.BASE_16)
 *@end
 */
/* Panicking conversions (single-var assignment) */
int64_t ez_strconv_to_int(EzString s, int base);

/*@man to_uint
 *@module strconv
 *@group Parsing
 *@sig to_uint(s string, base int = 10) -> (uint, Error)
 *@desc Parses s as an unsigned integer in the given base (2–36). Rejects strings with a leading minus sign. With single-var assignment, panics on invalid input. With destructuring, returns an Error instead.
 *@example
 *   import @strconv
 *   mut n uint = strconv.to_uint("255")
 *   mut val, err = strconv.to_uint("ff", strconv.BASE_16)
 *@end
 */
uint64_t ez_strconv_to_uint(EzString s, int base);

/*@man to_float
 *@module strconv
 *@group Parsing
 *@sig to_float(s string) -> (float, Error)
 *@desc Parses s as a floating-point number. Accepts standard decimal notation and "inf", "infinity", "nan" (case-insensitive). With single-var assignment, panics on invalid input. With destructuring, returns an Error instead.
 *@example
 *   import @strconv
 *   mut f float = strconv.to_float("3.14")
 *   mut val, err = strconv.to_float("not a number")
 *@end
 */
double ez_strconv_to_float(EzString s);

/*@man to_bool
 *@module strconv
 *@group Parsing
 *@sig to_bool(s string) -> (bool, Error)
 *@desc Parses s as a boolean. Accepts "true" and "false" (case-insensitive). All other strings produce an error or panic. With single-var assignment, panics on invalid input. With destructuring, returns an Error instead.
 *@example
 *   import @strconv
 *   mut b bool = strconv.to_bool("true")
 *   mut val, err = strconv.to_bool("yes")
 *@end
 */
bool ez_strconv_to_bool(EzString s);

/* Fallible conversions (multi-var destructuring) */
EzResult_int ez_strconv_to_int_result(EzString s, int base);
EzResult_uint ez_strconv_to_uint_result(EzString s, int base);
EzResult_float ez_strconv_to_float_result(EzString s);
EzResult_bool ez_strconv_to_bool_result(EzString s);

/*@man from_int
 *@module strconv
 *@group Formatting
 *@sig from_int(n int) -> string
 *@desc Converts an integer to its decimal string representation. Never fails.
 *@example
 *   import @strconv
 *   println(strconv.from_int(42))
 *@end
 */
/* Type to string conversions */
EzString ez_strconv_from_int(EzArena *arena, int64_t n);

/*@man from_uint
 *@module strconv
 *@group Formatting
 *@sig from_uint(n uint) -> string
 *@desc Converts an unsigned integer to its decimal string representation. Never fails.
 *@example
 *   import @strconv
 *   println(strconv.from_uint(255))
 *@end
 */
EzString ez_strconv_from_uint(EzArena *arena, uint64_t n);

/*@man from_float
 *@module strconv
 *@group Formatting
 *@sig from_float(f float) -> string
 *@desc Converts a float to its shortest accurate string representation. Never fails.
 *@example
 *   import @strconv
 *   println(strconv.from_float(3.14))
 *@end
 */
EzString ez_strconv_from_float(EzArena *arena, double f);

/*@man from_bool
 *@module strconv
 *@group Formatting
 *@sig from_bool(b bool) -> string
 *@desc Converts a boolean to "true" or "false". Never fails.
 *@example
 *   import @strconv
 *   println(strconv.from_bool(true))
 *@end
 */
EzString ez_strconv_from_bool(bool b);

/*@man is_numeric
 *@module strconv
 *@group Query
 *@sig is_numeric(s string) -> bool
 *@desc Returns true if s is a valid numeric representation (integer or decimal). Accepts an optional leading sign. Does not accept scientific notation, hex prefixes, or whitespace.
 *@example
 *   import @strconv
 *   println(strconv.is_numeric("3.14"))
 *   println(strconv.is_numeric("abc"))
 *@end
 */
/* Query functions */
bool ez_strconv_is_numeric(EzString s);

/*@man is_integer
 *@module strconv
 *@group Query
 *@sig is_integer(s string) -> bool
 *@desc Returns true if s is a valid integer (digits only, optional leading sign). Does not validate whether the value fits in an int or uint.
 *@example
 *   import @strconv
 *   println(strconv.is_integer("42"))
 *   println(strconv.is_integer("3.14"))
 *@end
 */
bool ez_strconv_is_integer(EzString s);

/*@man BASE_2
 *@module strconv
 *@group Constants
 *@kind const
 *@sig 2
 *@desc Base constant for binary. Pass to to_int() or to_uint() as the base argument.
 *@end
 */

/*@man BASE_8
 *@module strconv
 *@group Constants
 *@kind const
 *@sig 8
 *@desc Base constant for octal. Pass to to_int() or to_uint() as the base argument.
 *@end
 */

/*@man BASE_10
 *@module strconv
 *@group Constants
 *@kind const
 *@sig 10
 *@desc Base constant for decimal (the default). Pass to to_int() or to_uint() as the base argument.
 *@end
 */

/*@man BASE_16
 *@module strconv
 *@group Constants
 *@kind const
 *@sig 16
 *@desc Base constant for hexadecimal. Pass to to_int() or to_uint() as the base argument.
 *@end
 */

/*@man BASE_36
 *@module strconv
 *@group Constants
 *@kind const
 *@sig 36
 *@desc Base constant for base-36 (digits 0-9 and letters A-Z). Pass to to_int() or to_uint() as the base argument.
 *@end
 */

#endif
