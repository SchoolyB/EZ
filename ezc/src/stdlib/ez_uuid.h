/*
 * ez_uuid.h - @uuid module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_UUID_H
#define EZ_UUID_H

#include "../runtime/ez_runtime.h"

typedef struct {
    EzString value;
} EzUUID;

EzUUID ez_uuid_generate(EzArena *arena);
EzString ez_uuid_generate_compact(EzArena *arena, EzUUID id);
EzUUID ez_uuid_generate_random(EzArena *arena);
EzUUID ez_uuid_generate_time_ordered(EzArena *arena);
bool ez_uuid_is_valid(EzString s);
EzUUID ez_uuid_parse(EzArena *arena, EzString s);
EzString ez_uuid_to_string(EzUUID id);
EzUUID ez_uuid_nil(void);

#endif
