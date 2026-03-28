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
EzArray ez_maps_keys(EzArena *arena, EzMap *m);

/* maps.values(m) — return array of values */
EzArray ez_maps_values(EzArena *arena, EzMap *m);

/* maps.has_key(m, key) — check if key exists */
bool ez_maps_has_key(EzMap *m, const void *key);

/* maps.is_empty(m) — true if map has no entries */
bool ez_maps_is_empty(EzMap *m);

#endif
