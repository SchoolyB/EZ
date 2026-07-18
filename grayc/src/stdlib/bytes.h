/*
 * bytes.h — Public interface for the bytes stdlib module.
 * Declares functions for converting between strings, byte arrays,
 * hex, and base64 representations.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_BYTES_H
#define GRAY_BYTES_H

#include "../runtime/runtime.h"
#include "../runtime/array.h"

/*@man from_string
 *@module bytes
 *@group Conversion
 *@sig from_string(s string) -> [byte]
 *@desc Converts a UTF-8 string into a byte array.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_string("hello")
 *@end
 */

/*@man from_hex
 *@module bytes
 *@group Conversion
 *@sig from_hex(hex string) -> [byte]
 *@desc Decodes a hex-encoded string into a byte array.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_hex("48656c6c6f")
 *@end
 */

/*@man from_base64
 *@module bytes
 *@group Conversion
 *@sig from_base64(b64 string) -> [byte]
 *@desc Decodes a base64-encoded string into a byte array.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_base64("SGVsbG8=")
 *@end
 */

/*@man to_string
 *@module bytes
 *@group Encoding
 *@sig to_string(bytes [byte]) -> string
 *@desc Converts a byte array to a UTF-8 string.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_string("hello")
 *   println(bytes.to_string(b))
 *@end
 */

/*@man to_hex
 *@module bytes
 *@group Encoding
 *@sig to_hex(bytes [byte]) -> string
 *@desc Encodes a byte array as a lowercase hex string.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_string("hi")
 *   println(bytes.to_hex(b))
 *@end
 */

/*@man to_base64
 *@module bytes
 *@group Encoding
 *@sig to_base64(bytes [byte]) -> string
 *@desc Encodes a byte array as a base64 string.
 *@example
 *   import @bytes
 *   mut b [byte] = bytes.from_string("hello")
 *   println(bytes.to_base64(b))
 *@end
 */

GrayArray gray_bytes_from_string(GrayArena *arena, GrayString s);
GrayString gray_bytes_to_string(GrayArena *arena, GrayArray *bytes);
GrayArray gray_bytes_from_hex(GrayArena *arena, GrayString hex);
GrayString gray_bytes_to_hex(GrayArena *arena, GrayArray *bytes);
GrayArray gray_bytes_from_base64(GrayArena *arena, GrayString b64);
GrayString gray_bytes_to_base64(GrayArena *arena, GrayArray *bytes);

#endif
