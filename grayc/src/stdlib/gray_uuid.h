/*
 * gray_uuid.h - @uuid module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_UUID_H
#define GRAY_UUID_H

#include "../runtime/gray_runtime.h"

typedef struct {
    EzString value;
} EzUUID;

EzUUID gray_uuid_generate(EzArena *arena);
EzString gray_uuid_generate_compact(EzArena *arena, EzUUID id);
EzUUID gray_uuid_generate_random(EzArena *arena);
EzUUID gray_uuid_generate_time_ordered(EzArena *arena);
bool gray_uuid_is_valid(EzString s);
EzUUID gray_uuid_parse(EzArena *arena, EzString s);
EzString gray_uuid_to_string(EzUUID id);
EzUUID gray_uuid_nil(void);

#endif
