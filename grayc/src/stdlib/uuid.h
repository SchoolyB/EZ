/*
 * uuid.h — Public interface for the uuid stdlib module.
 * Declares UUID v4 generation, parsing, comparison, and
 * string conversion functions.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_UUID_H
#define GRAY_UUID_H

#include "../runtime/runtime.h"

typedef struct {
    GrayString value;
} GrayUUID;

GrayUUID gray_uuid_generate(GrayArena *arena);
GrayString gray_uuid_generate_compact(GrayArena *arena, GrayUUID id);
GrayUUID gray_uuid_generate_random(GrayArena *arena);
GrayUUID gray_uuid_generate_time_ordered(GrayArena *arena);
bool gray_uuid_is_valid(GrayString s);
GrayUUID gray_uuid_parse(GrayArena *arena, GrayString s);
GrayString gray_uuid_to_string(GrayUUID id);
GrayUUID gray_uuid_nil(void);

#endif
