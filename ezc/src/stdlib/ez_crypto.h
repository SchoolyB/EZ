/*
 * ez_crypto.h - @crypto module for EZ
 *
 * Embedded implementations — no OpenSSL dependency.
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_CRYPTO_H
#define EZ_CRYPTO_H

#include "../runtime/ez_runtime.h"

EzString ez_crypto_sha256(EzArena *arena, EzString data);
EzString ez_crypto_md5(EzArena *arena, EzString data);
EzString ez_crypto_random_hex(EzArena *arena, int64_t length);

#endif
