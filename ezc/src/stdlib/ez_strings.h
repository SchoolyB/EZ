/*
 * ez_strings.h - @strings module for EZ
 *
 * String manipulation functions.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STRINGS_H
#define EZ_STRINGS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/*@man to_upper
 *@module strings
 *@group Case
 *@sig to_upper(s string) -> string
 *@desc Returns a copy of s with all ASCII letters converted to uppercase.
 *@example
 *   import @strings
 *   println(strings.to_upper("hello"))
 *@end
 */
EzString ez_strings_to_upper(EzArena *arena, EzString s);

/*@man to_lower
 *@module strings
 *@group Case
 *@sig to_lower(s string) -> string
 *@desc Returns a copy of s with all ASCII letters converted to lowercase.
 *@example
 *   import @strings
 *   println(strings.to_lower("HELLO"))
 *@end
 */
EzString ez_strings_to_lower(EzArena *arena, EzString s);

/*@man trim
 *@module strings
 *@group Trim
 *@sig trim(s string) -> string
 *@desc Returns a copy of s with leading and trailing whitespace removed.
 *@example
 *   import @strings
 *   println(strings.trim("  hello  "))
 *@end
 */
EzString ez_strings_trim(EzArena *arena, EzString s);

/*@man trim_left
 *@module strings
 *@group Trim
 *@sig trim_left(s string) -> string
 *@desc Returns a copy of s with leading whitespace removed.
 *@example
 *   import @strings
 *   println(strings.trim_left("  hello  "))
 *@end
 */
EzString ez_strings_trim_left(EzArena *arena, EzString s);

/*@man trim_right
 *@module strings
 *@group Trim
 *@sig trim_right(s string) -> string
 *@desc Returns a copy of s with trailing whitespace removed.
 *@example
 *   import @strings
 *   println(strings.trim_right("  hello  "))
 *@end
 */
EzString ez_strings_trim_right(EzArena *arena, EzString s);

/*@man contains
 *@module strings
 *@group Query
 *@sig contains(s string, sub string) -> bool
 *@desc Returns true if s contains the substring sub.
 *@example
 *   import @strings
 *   println(strings.contains("hello world", "world"))
 *   println(strings.contains("hello world", "xyz"))
 *@end
 */
bool ez_strings_contains(EzString s, EzString sub);

/*@man starts_with
 *@module strings
 *@group Query
 *@sig starts_with(s string, prefix string) -> bool
 *@desc Returns true if s starts with prefix.
 *@example
 *   import @strings
 *   println(strings.starts_with("hello", "hel"))
 *@end
 */
bool ez_strings_starts_with(EzString s, EzString prefix);

/*@man ends_with
 *@module strings
 *@group Query
 *@sig ends_with(s string, suffix string) -> bool
 *@desc Returns true if s ends with suffix.
 *@example
 *   import @strings
 *   println(strings.ends_with("hello", "llo"))
 *@end
 */
bool ez_strings_ends_with(EzString s, EzString suffix);

/*@man index_of
 *@module strings
 *@group Query
 *@sig index_of(s string, sub string) -> int
 *@desc Returns the byte index of the first occurrence of sub in s, or -1 if not found.
 *@example
 *   import @strings
 *   println(strings.index_of("hello world", "world"))
 *   println(strings.index_of("hello world", "xyz"))
 *@end
 */
int64_t ez_strings_index_of(EzString s, EzString sub);

/*@man count
 *@module strings
 *@group Query
 *@sig count(s string, sub string) -> int
 *@desc Returns the number of non-overlapping occurrences of sub in s.
 *@example
 *   import @strings
 *   println(strings.count("banana", "a"))
 *@end
 */
int64_t ez_strings_count(EzString s, EzString sub);

/*@man is_empty
 *@module strings
 *@group Query
 *@sig is_empty(s string) -> bool
 *@desc Returns true if s has zero length. Does not trim whitespace first.
 *@example
 *   import @strings
 *   println(strings.is_empty(""))
 *   println(strings.is_empty("hi"))
 *@end
 */
bool ez_strings_is_empty(EzString s);

/*@man replace
 *@module strings
 *@group Transformation
 *@sig replace(s string, old string, new string) -> string
 *@desc Returns a copy of s with all occurrences of old replaced by new.
 *@example
 *   import @strings
 *   println(strings.replace("hello world", "world", "EZ"))
 *@end
 */
EzString ez_strings_replace(EzArena *arena, EzString s, EzString old_s, EzString new_s);

/*@man repeat
 *@module strings
 *@group Transformation
 *@sig repeat(s string, count int) -> string
 *@desc Returns a string consisting of count copies of s concatenated together.
 *@example
 *   import @strings
 *   println(strings.repeat("ab", 3))
 *@end
 */
EzString ez_strings_repeat(EzArena *arena, EzString s, int64_t count);

/*@man reverse
 *@module strings
 *@group Transformation
 *@sig reverse(s string) -> string
 *@desc Returns a copy of s with the bytes in reverse order.
 *@example
 *   import @strings
 *   println(strings.reverse("hello"))
 *@end
 */
EzString ez_strings_reverse(EzArena *arena, EzString s);

/*@man slice
 *@module strings
 *@group Transformation
 *@sig slice(s string, start int, end int) -> string
 *@desc Returns the substring of s from byte index start (inclusive) to end (exclusive).
 *@example
 *   import @strings
 *   println(strings.slice("hello world", 6, 11))
 *@end
 */
EzString ez_strings_slice(EzArena *arena, EzString s, int64_t start, int64_t end);

/*@man split
 *@module strings
 *@group Split/Join
 *@sig split(s string, sep string) -> [string]
 *@desc Splits s around each occurrence of sep and returns a string array.
 *@example
 *   import @strings
 *   mut parts = strings.split("a,b,c", ",")
 *   println(parts[0])
 *@end
 */
EzArray ez_strings_split(EzArena *arena, EzString s, EzString sep);

/*@man join
 *@module strings
 *@group Split/Join
 *@sig join(arr [string], sep string) -> string
 *@desc Joins the elements of arr into a single string with sep between each element.
 *@example
 *   import @strings
 *   mut parts = strings.split("a,b,c", ",")
 *   println(strings.join(parts, "-"))
 *@end
 */
EzString ez_strings_join(EzArena *arena, EzArray arr, EzString sep);

#endif
