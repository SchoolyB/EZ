/*
 * ez_time.h - @time module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_TIME_H
#define EZ_TIME_H

#include "../runtime/ez_runtime.h"

/* Current time */
int64_t ez_time_now(void);        /* Unix timestamp (seconds) */
int64_t ez_time_now_ms(void);     /* Milliseconds since epoch */
int64_t ez_time_now_ns(void);     /* Nanoseconds since epoch */

/* Components */
int64_t ez_time_year(int64_t ts);
int64_t ez_time_month(int64_t ts);
int64_t ez_time_day(int64_t ts);
int64_t ez_time_hour(int64_t ts);
int64_t ez_time_minute(int64_t ts);
int64_t ez_time_second(int64_t ts);
int64_t ez_time_weekday(int64_t ts);

/* Formatting */
EzString ez_time_format(EzArena *arena, EzString fmt, int64_t ts);
EzString ez_time_to_iso(EzArena *arena, int64_t ts);
EzString ez_time_date(EzArena *arena, int64_t ts);
EzString ez_time_to_clock(EzArena *arena, int64_t ts);

/* Performance */
int64_t ez_time_tick(void);
int64_t ez_time_elapsed_ms(int64_t start_tick);

#endif
