/*
 * regex.h — Public interface for the regex stdlib module.
 * Declares match, find, find_all, replace, and split operations
 * using POSIX extended regular expressions.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_REGEX_H
#define GRAY_REGEX_H

#include "../runtime/runtime.h"
#include "../runtime/array.h"
#include "io.h" /* GrayResult_string, GrayResult_array */

/* regex.is_valid(pattern) -> bool */
bool gray_regex_is_valid(GrayString pattern);

/* regex.match(pattern, text) -> bool */
bool gray_regex_match(GrayString pattern, GrayString text);

/* regex.find(pattern, text) -> string (first match, or empty) */
GrayString gray_regex_find(GrayArena *arena, GrayString pattern, GrayString text);

/* regex.find_all(pattern, text) -> [string] */
GrayArray gray_regex_find_all(GrayArena *arena, GrayString pattern, GrayString text);

/* regex.replace(pattern, text, replacement) -> string */
GrayString gray_regex_replace(GrayArena *arena, GrayString pattern, GrayString text, GrayString replacement);

/* regex.split(pattern, text) -> [string] */
GrayArray gray_regex_split(GrayArena *arena, GrayString pattern, GrayString text);

/* _result variants */
GrayResult_string gray_regex_find_result(GrayArena *arena, GrayString pattern, GrayString text);
GrayResult_array gray_regex_find_all_result(GrayArena *arena, GrayString pattern, GrayString text);
GrayResult_string gray_regex_replace_result(GrayArena *arena, GrayString pattern, GrayString text, GrayString replacement);
GrayResult_array gray_regex_split_result(GrayArena *arena, GrayString pattern, GrayString text);

#endif
