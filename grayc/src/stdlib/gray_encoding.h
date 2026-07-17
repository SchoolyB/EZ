/*
 * gray_encoding.h — Public interface for the encoding stdlib module.
 * Declares base64, hex, and URL encode/decode functions.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_ENCODING_H
#define GRAY_ENCODING_H

#include "../runtime/gray_runtime.h"

GrayString gray_encoding_base64_encode(GrayArena *arena, GrayString s);
GrayString gray_encoding_base64_decode(GrayArena *arena, GrayString s);
GrayString gray_encoding_hex_encode(GrayArena *arena, GrayString s);
GrayString gray_encoding_hex_decode(GrayArena *arena, GrayString s);
GrayString gray_encoding_url_encode(GrayArena *arena, GrayString s);
GrayString gray_encoding_url_decode(GrayArena *arena, GrayString s);

#endif
