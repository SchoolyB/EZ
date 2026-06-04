/*
 * ez_bytes.h - @bytes module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_BYTES_H
#define EZ_BYTES_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

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

EzArray ez_bytes_from_string(EzArena *arena, EzString s);
EzString ez_bytes_to_string(EzArena *arena, EzArray *bytes);
EzArray ez_bytes_from_hex(EzArena *arena, EzString hex);
EzString ez_bytes_to_hex(EzArena *arena, EzArray *bytes);
EzArray ez_bytes_from_base64(EzArena *arena, EzString b64);
EzString ez_bytes_to_base64(EzArena *arena, EzArray *bytes);

#endif
