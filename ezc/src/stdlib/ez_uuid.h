/*
 * ez_uuid.h - @uuid module for EZC
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_UUID_H
#define EZ_UUID_H

#include "../runtime/ez_runtime.h"

EzString ez_uuid_generate(EzArena *arena);
EzString ez_uuid_generate_compact(EzArena *arena);
bool ez_uuid_is_valid(EzString s);

#endif
