/*
 * ez_json.h - @json module for EZ
 *
 * Minimal JSON parser and emitter.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_JSON_H
#define EZ_JSON_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "../runtime/ez_map.h"

/* JSON string escaping helpers (used by generated #json struct code) */
size_t json_escaped_len(EzString s);
void json_append_escaped(char *buf, int *pos, EzString s);

/*@man encode
 *@module json
 *@group Encoding
 *@sig encode(value T) -> string
 *@desc Encodes a value as a JSON string. Accepts int, float, bool, string, map, and array. For #json structs use stringify() instead.
 *@example
 *   import @json
 *   mut m map[string:string] = {"name": "Alice"}
 *   println(json.encode(m))
 *   println(json.encode(42))
 *   println(json.encode(true))
 *   mut arr [int] = {1, 2, 3}
 *   println(json.encode(arr))
 *@end
 */
/* json.encode(value) — convert map to JSON string */
EzString ez_json_encode_map(EzArena *arena, EzMap *m);

/* json.encode(array) — convert typed arrays to JSON */
EzString ez_json_encode_array_int(EzArena *arena, EzArray *arr);
EzString ez_json_encode_array_float(EzArena *arena, EzArray *arr);
EzString ez_json_encode_array_string(EzArena *arena, EzArray *arr);
EzString ez_json_encode_array_bool(EzArena *arena, EzArray *arr);

/* json.encode(map) — convert typed maps to JSON */
EzString ez_json_encode_map_int(EzArena *arena, EzMap *m);
EzString ez_json_encode_map_float(EzArena *arena, EzMap *m);
EzString ez_json_encode_map_bool(EzArena *arena, EzMap *m);

/*@man stringify
 *@module json
 *@group Encoding
 *@sig stringify(value T) -> string
 *@desc Encodes a #json struct to a JSON string. Also accepts maps and other types as a fallback. Use on structs marked with the #json attribute.
 *@example
 *   import @json
 *   #json
 *   const User struct {
 *       name string
 *       age int
 *   }
 *   do main() {
 *       mut u User = json.parse("{\"name\": \"Alice\", \"age\": 25}")
 *       println(json.stringify(u))
 *   }
 *@end
 */
/* json.stringify — codegen-dispatched to per-struct helper for #json structs */

/*@man decode
 *@module json
 *@group Decoding
 *@sig decode(text string) -> (map[string:string], Error)
 *@desc Parses a JSON object string into a map[string:string]. All values are returned as strings. Panics on single-var assignment; use destructuring to receive the Error instead.
 *@example
 *   import @json
 *   mut m, err = json.decode("{\"key\": \"value\"}")
 *   mut m2, _ = json.decode("{\"x\": \"1\"}")
 *@end
 */
/* json.decode(text) — parse JSON string to map */
EzMap ez_json_decode(EzArena *arena, EzString text);

/*@man parse
 *@module json
 *@group Decoding
 *@sig parse(text string) -> T
 *@desc Parses a JSON string directly into a #json struct. The target type is inferred from the variable declaration. The struct must be marked with the #json attribute.
 *@example
 *   import @json
 *   #json
 *   const User struct {
 *       name string
 *       age int
 *   }
 *   do main() {
 *       mut u User = json.parse("{\"name\": \"Alice\", \"age\": 25}")
 *       println(u.name)
 *   }
 *@end
 */
/* json.parse — codegen-dispatched to per-struct helper for #json structs */

/*@man is_valid
 *@module json
 *@group Query
 *@sig is_valid(text string) -> bool
 *@desc Returns true if text is a valid JSON string, false otherwise.
 *@example
 *   import @json
 *   println(json.is_valid("{\"ok\": true}"))
 *   println(json.is_valid("not json"))
 *@end
 */
/* json.is_valid(text) — check if string is valid JSON */
bool ez_json_is_valid(EzString text);

/*@man pretty_print
 *@module json
 *@group Formatting
 *@sig pretty_print(m map, indent int) -> string
 *@desc Returns a pretty-printed JSON string from a map, indented by indent spaces per level.
 *@example
 *   import @json
 *   mut m map[string:string] = {"name": "Alice", "city": "NYC"}
 *   println(json.pretty_print(m, 2))
 *@end
 */
/* json.pretty(value, indent) — pretty-print JSON */
EzString ez_json_pretty_map(EzArena *arena, EzMap *m, int64_t indent);

/* json array splitting: returns an EzArray of EzString, each being one
 * top-level JSON element from a JSON array string like "[{...},{...}]". */
EzArray ez_json_split_array(EzArena *arena, EzString text);

/* _result variant */
typedef struct { EzMap v0; EzError *v1; } EzResult_map;

EzResult_map ez_json_decode_result(EzArena *arena, EzString text);

#endif
