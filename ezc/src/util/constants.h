/*
 * constants.h - Shared compiler constants
 *
 * Only for values referenced by multiple compilation stages.
 * File-local constants belong in their respective .c files.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_CONSTANTS_H
#define EZ_CONSTANTS_H

/* --- Diagnostic buffer sizes --- */
#define EZ_MSG_BUF_SIZE         256
#define EZ_MSG_BUF_LARGE        512
#define EZ_SOURCE_LINE_MAX      2048
#define EZ_TYPE_NAME_MAX        128

/* --- Time unit conversions --- */
#define NS_PER_SEC              1000000000LL
#define NS_PER_MS               1000000LL
#define MS_PER_SEC              1000LL

#endif
