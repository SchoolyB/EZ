/*
 * ez_regex.h - @regex module for EZC
 *
 * Regular expression operations using POSIX regex.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_REGEX_H
#define EZ_REGEX_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* regex.match(pattern, text) -> bool */
bool ez_regex_match(EzString pattern, EzString text);

/* regex.find(pattern, text) -> string (first match, or empty) */
EzString ez_regex_find(EzArena *arena, EzString pattern, EzString text);

/* regex.find_all(pattern, text) -> [string] */
EzArray ez_regex_find_all(EzArena *arena, EzString pattern, EzString text);

/* regex.replace(pattern, text, replacement) -> string */
EzString ez_regex_replace(EzArena *arena, EzString pattern, EzString text, EzString replacement);

/* regex.split(pattern, text) -> [string] */
EzArray ez_regex_split(EzArena *arena, EzString pattern, EzString text);

#endif
