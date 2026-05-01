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

EzArray ez_bytes_from_string(EzArena *arena, EzString s);
EzString ez_bytes_to_string(EzArena *arena, EzArray *bytes);
EzArray ez_bytes_from_hex(EzArena *arena, EzString hex);
EzString ez_bytes_to_hex(EzArena *arena, EzArray *bytes);
EzArray ez_bytes_from_base64(EzArena *arena, EzString b64);
EzString ez_bytes_to_base64(EzArena *arena, EzArray *bytes);

#endif
