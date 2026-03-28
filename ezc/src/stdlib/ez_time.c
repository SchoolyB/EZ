/*
 * ez_time.c - @time module implementation
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#include "ez_time.h"
#include <time.h>
#include <unistd.h>
#include <string.h>

int64_t ez_time_now(void) { return (int64_t)time(NULL); }

int64_t ez_time_now_ms(void) {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)ts.tv_sec * 1000 + (int64_t)ts.tv_nsec / 1000000;
}

int64_t ez_time_now_ns(void) {
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)ts.tv_sec * 1000000000 + (int64_t)ts.tv_nsec;
}

static struct tm *get_tm(int64_t ts) {
    time_t t = (time_t)ts;
    return localtime(&t);
}

int64_t ez_time_year(int64_t ts) { return get_tm(ts)->tm_year + 1900; }
int64_t ez_time_month(int64_t ts) { return get_tm(ts)->tm_mon + 1; }
int64_t ez_time_day(int64_t ts) { return get_tm(ts)->tm_mday; }
int64_t ez_time_hour(int64_t ts) { return get_tm(ts)->tm_hour; }
int64_t ez_time_minute(int64_t ts) { return get_tm(ts)->tm_min; }
int64_t ez_time_second(int64_t ts) { return get_tm(ts)->tm_sec; }
int64_t ez_time_weekday(int64_t ts) { return get_tm(ts)->tm_wday; }

EzString ez_time_format(EzArena *arena, EzString fmt, int64_t ts) {
    char buf[256];
    struct tm *tm = get_tm(ts);
    int len = (int)strftime(buf, sizeof(buf), fmt.data, tm);
    return ez_string_new(arena, buf, len);
}

EzString ez_time_to_iso(EzArena *arena, int64_t ts) {
    return ez_time_format(arena, ez_string_lit("%Y-%m-%dT%H:%M:%S"), ts);
}

EzString ez_time_date(EzArena *arena, int64_t ts) {
    return ez_time_format(arena, ez_string_lit("%Y-%m-%d"), ts);
}

EzString ez_time_to_time(EzArena *arena, int64_t ts) {
    return ez_time_format(arena, ez_string_lit("%H:%M:%S"), ts);
}

int64_t ez_time_tick(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000000000 + (int64_t)ts.tv_nsec;
}

int64_t ez_time_elapsed_ms(int64_t start_tick) {
    return (ez_time_tick() - start_tick) / 1000000;
}
