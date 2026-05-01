/*
 * ez_encoding.h - @encoding module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_ENCODING_H
#define EZ_ENCODING_H

#include "../runtime/ez_runtime.h"

EzString ez_encoding_base64_encode(EzArena *arena, EzString s);
EzString ez_encoding_base64_decode(EzArena *arena, EzString s);
EzString ez_encoding_hex_encode(EzArena *arena, EzString s);
EzString ez_encoding_hex_decode(EzArena *arena, EzString s);
EzString ez_encoding_url_encode(EzArena *arena, EzString s);
EzString ez_encoding_url_decode(EzArena *arena, EzString s);

#endif
