/*
 * ez_maps.h - @maps module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_MAPS_H
#define EZ_MAPS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "../runtime/ez_map.h"

/* maps.keys(m) — return array of keys */
EzArray ez_maps_get_keys(EzArena *arena, EzMap *m);

/* maps.values(m) — return array of values */
EzArray ez_maps_get_values(EzArena *arena, EzMap *m);

/* maps.has_key(m, key) — check if key exists */
bool ez_maps_has_key(EzMap *m, const void *key);

/* maps.is_empty(m) — true if map has no entries */
bool ez_maps_is_empty(EzMap *m);

/* maps.merge(m1, m2) — combine two maps (m2 overwrites m1 on conflict) */
EzMap ez_maps_merge(EzArena *arena, EzMap *m1, EzMap *m2);

/* maps.contains_value(m, value) — check if any entry has the given value */
bool ez_maps_contains_value(EzMap *m, const void *value);

/* maps.is_equal(a, b) — structural equality.
 * str_keys: true if keys are EzString (look up via ez_map_get_str)
 * str_values: true if values are EzString (compare contents, not blob)
 * Composite key/value types (nested arrays, maps, structs) are rejected
 * at typecheck and never reach this function. */
bool ez_maps_is_equal(EzMap *a, EzMap *b, bool str_keys, bool str_values);

/* maps.get_or_default(m, key, default) — return value or fallback */
/* NOTE: implemented inline in codegen due to type-generic return */

#endif
