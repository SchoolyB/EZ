/*
 * gray_time.h — Public interface for the time stdlib module.
 * Declares current time queries, date/time component extraction,
 * formatting, and elapsed-time measurement functions.
 *
 * Author:  Marshall A Burns (@SchoolyB)
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef GRAY_TIME_H
#define GRAY_TIME_H

#include "../runtime/gray_runtime.h"

/* Current time */

/*@man now
 *@module time
 *@group Current Time
 *@sig now() -> int
 *@desc Returns the current Unix timestamp in seconds.
 *@example
 *   import @time
 *   println(time.now())
 *@end
 */
int64_t gray_time_now(void);        /* Unix timestamp (seconds) */

/*@man now_ms
 *@module time
 *@group Current Time
 *@sig now_ms() -> int
 *@desc Returns the current Unix timestamp in milliseconds.
 *@example
 *   import @time
 *   println(time.now_ms())
 *@end
 */
int64_t gray_time_now_ms(void);     /* Milliseconds since epoch */

/*@man now_ns
 *@module time
 *@group Current Time
 *@sig now_ns() -> int
 *@desc Returns the current Unix timestamp in nanoseconds.
 *@example
 *   import @time
 *   println(time.now_ns())
 *@end
 */
int64_t gray_time_now_ns(void);     /* Nanoseconds since epoch */

/* Components */

/*@man year
 *@module time
 *@group Components
 *@sig year(timestamp int) -> int
 *@desc Returns the year from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.year(time.now()))
 *@end
 */
int64_t gray_time_year(int64_t ts);

/*@man month
 *@module time
 *@group Components
 *@sig month(timestamp int) -> int
 *@desc Returns the month (1–12) from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.month(time.now()))
 *@end
 */
int64_t gray_time_month(int64_t ts);

/*@man day
 *@module time
 *@group Components
 *@sig day(timestamp int) -> int
 *@desc Returns the day of the month from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.day(time.now()))
 *@end
 */
int64_t gray_time_day(int64_t ts);

/*@man hour
 *@module time
 *@group Components
 *@sig hour(timestamp int) -> int
 *@desc Returns the hour (0–23) from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.hour(time.now()))
 *@end
 */
int64_t gray_time_hour(int64_t ts);

/*@man minute
 *@module time
 *@group Components
 *@sig minute(timestamp int) -> int
 *@desc Returns the minute (0–59) from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.minute(time.now()))
 *@end
 */
int64_t gray_time_minute(int64_t ts);

/*@man second
 *@module time
 *@group Components
 *@sig second(timestamp int) -> int
 *@desc Returns the second (0–59) from a Unix timestamp.
 *@example
 *   import @time
 *   println(time.second(time.now()))
 *@end
 */
int64_t gray_time_second(int64_t ts);

/*@man weekday
 *@module time
 *@group Components
 *@sig weekday(timestamp int) -> int
 *@desc Returns the day of the week from a Unix timestamp. 0 = Sunday, 1 = Monday, ..., 6 = Saturday.
 *@example
 *   import @time
 *   println(time.weekday(time.now()))
 *@end
 */
int64_t gray_time_weekday(int64_t ts);

/* Formatting */

/*@man format
 *@module time
 *@group Formatting
 *@sig format(fmt string, timestamp int) -> string
 *@desc Formats a Unix timestamp using a format string. Uses strftime-style directives: %Y (year), %m (month), %d (day), %H (hour), %M (minute), %S (second).
 *@example
 *   import @time
 *   println(time.format("%Y-%m-%d", time.now()))
 *@end
 */
GrayString gray_time_format(GrayArena *arena, GrayString fmt, int64_t ts);

/*@man to_iso
 *@module time
 *@group Formatting
 *@sig to_iso(timestamp int) -> string
 *@desc Returns the timestamp as an ISO 8601 string (e.g. "2025-06-01T14:30:00Z").
 *@example
 *   import @time
 *   println(time.to_iso(time.now()))
 *@end
 */
GrayString gray_time_to_iso(GrayArena *arena, int64_t ts);

/*@man date
 *@module time
 *@group Formatting
 *@sig date(timestamp int) -> string
 *@desc Returns the date portion of a Unix timestamp as "YYYY-MM-DD".
 *@example
 *   import @time
 *   println(time.date(time.now()))
 *@end
 */
GrayString gray_time_date(GrayArena *arena, int64_t ts);

/*@man to_clock
 *@module time
 *@group Formatting
 *@sig to_clock(timestamp int) -> string
 *@desc Returns the time portion of a Unix timestamp as "HH:MM:SS".
 *@example
 *   import @time
 *   println(time.to_clock(time.now()))
 *@end
 */
GrayString gray_time_to_clock(GrayArena *arena, int64_t ts);

/* Arithmetic */

/*@man diff
 *@module time
 *@group Arithmetic
 *@sig diff(t1 int, t2 int) -> int
 *@desc Returns the difference between two Unix timestamps in seconds as t2 - t1. The result is negative if t1 is after t2.
 *@example
 *   import @time
 *   mut start int = time.now()
 *   mut end int = time.now()
 *   mut delta int = time.diff(start, end)
 *   println(delta)
 *@end
 */
int64_t gray_time_diff(int64_t t1, int64_t t2);

/* Performance */

/*@man tick
 *@module time
 *@group Performance
 *@sig tick() -> int
 *@desc Returns a high-resolution timestamp in nanoseconds for performance measurement. Use with elapsed_ms() to measure durations.
 *@example
 *   import @time
 *   mut start int = time.tick()
 *   mut elapsed int = time.elapsed_ms(start)
 *   println(elapsed)
 *@end
 */
int64_t gray_time_tick(void);

/*@man elapsed_ms
 *@module time
 *@group Performance
 *@sig elapsed_ms(start_tick int) -> int
 *@desc Returns the number of milliseconds elapsed since the tick value returned by tick().
 *@example
 *   import @time
 *   mut start int = time.tick()
 *   mut elapsed int = time.elapsed_ms(start)
 *   println(elapsed)
 *@end
 */
int64_t gray_time_elapsed_ms(int64_t start_tick);

#endif
