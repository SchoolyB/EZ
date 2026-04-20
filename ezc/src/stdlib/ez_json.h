/*
 * ez_json.h - @json module for EZC
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

/* json.decode(text) — parse JSON string to map */
EzMap ez_json_decode(EzArena *arena, EzString text);

/* json.is_valid(text) — check if string is valid JSON */
bool ez_json_is_valid(EzString text);

/* json.pretty(value, indent) — pretty-print JSON */
EzString ez_json_pretty_map(EzArena *arena, EzMap *m, int64_t indent);

#endif
