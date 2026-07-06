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

/* WARNING: MD5 is a cryptographically broken hash function. It has known
 * collision vulnerabilities and must not be used for password hashing,
 * certificate fingerprinting, integrity verification in security contexts,
 * or HMAC construction. Use crypto.sha256() for any security purpose.
 * crypto.md5() is provided only for legacy compatibility and non-security
 * checksum use cases (e.g. cache keys, non-sensitive deduplication). */
EzString ez_crypto_md5(EzArena *arena, EzString data);
EzString ez_crypto_random_hex(EzArena *arena, int64_t length);

#endif
