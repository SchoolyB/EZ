/*
 * gray_encoding.h - @encoding module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_ENCODING_H
#define GRAY_ENCODING_H

#include "../runtime/gray_runtime.h"

EzString gray_encoding_base64_encode(EzArena *arena, EzString s);
EzString gray_encoding_base64_decode(EzArena *arena, EzString s);
EzString gray_encoding_hex_encode(EzArena *arena, EzString s);
EzString gray_encoding_hex_decode(EzArena *arena, EzString s);
EzString gray_encoding_url_encode(EzArena *arena, EzString s);
EzString gray_encoding_url_decode(EzArena *arena, EzString s);

#endif
