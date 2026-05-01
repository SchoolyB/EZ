/*
 * ez_strings.h - @strings module for EZ
 *
 * String manipulation functions.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_STRINGS_H
#define EZ_STRINGS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* Case */
EzString ez_strings_to_upper(EzArena *arena, EzString s);
EzString ez_strings_to_lower(EzArena *arena, EzString s);

/* Trim */
EzString ez_strings_trim(EzArena *arena, EzString s);
EzString ez_strings_trim_left(EzArena *arena, EzString s);
EzString ez_strings_trim_right(EzArena *arena, EzString s);

/* Query */
bool ez_strings_contains(EzString s, EzString sub);
bool ez_strings_starts_with(EzString s, EzString prefix);
bool ez_strings_ends_with(EzString s, EzString suffix);
int64_t ez_strings_index_of(EzString s, EzString sub);
int64_t ez_strings_count(EzString s, EzString sub);
bool ez_strings_is_empty(EzString s);

/* Transformation */
EzString ez_strings_replace(EzArena *arena, EzString s, EzString old_s, EzString new_s);
EzString ez_strings_repeat(EzArena *arena, EzString s, int64_t count);
EzString ez_strings_reverse(EzArena *arena, EzString s);
EzString ez_strings_slice(EzArena *arena, EzString s, int64_t start, int64_t end);

/* Split/Join */
EzArray ez_strings_split(EzArena *arena, EzString s, EzString sep);
EzString ez_strings_join(EzArena *arena, EzArray arr, EzString sep);

#endif
