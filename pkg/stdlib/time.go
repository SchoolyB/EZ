package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"math/big"
	"strconv"
	"time"

	"github.com/marshallburns/ez/pkg/object"
)

// TimeBuiltins contains the time module functions
var TimeBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// Current Time
	// ============================================================================

	// now returns the current Unix timestamp in seconds.
	// Takes no arguments. Returns int.
	"time.now": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(time.Now().Unix())}
		},
	},

	// now_ms returns the current Unix timestamp in milliseconds.
	// Takes no arguments. Returns int.
	"time.now_ms": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(time.Now().UnixMilli())}
		},
	},

	// now_ns returns the current Unix timestamp in nanoseconds.
	// Takes no arguments. Returns int.
	"time.now_ns": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(time.Now().UnixNano())}
		},
	},

	// ============================================================================
	// Sleep/Delay
	// ============================================================================

	// sleep pauses execution for the specified number of seconds.
	// Takes seconds (int or float). Returns nil.
	"time.sleep": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.sleep() takes exactly 1 argument (seconds)"}
			}
			switch v := args[0].(type) {
			case *object.Integer:
				time.Sleep(time.Duration(v.Value.Int64()) * time.Second)
			case *object.Float:
				time.Sleep(time.Duration(v.Value * float64(time.Second)))
			default:
				return &object.Error{Code: "E7005", Message: "time.sleep() requires a number"}
			}
			return object.NIL
		},
	},
	// sleep_ms pauses execution for the specified number of milliseconds.
	// Takes milliseconds (int). Returns nil.
	"time.sleep_ms": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.sleep_ms() takes exactly 1 argument (milliseconds)"}
			}
			ms, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.sleep_ms() requires an integer"}
			}
			time.Sleep(time.Duration(ms.Value.Int64()) * time.Millisecond)
			return object.NIL
		},
	},

	// ============================================================================
	// Time Components
	// ============================================================================

	// year extracts the year from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.year": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Year()))}
		},
	},
	// month extracts the month (1-12) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.month": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Month()))}
		},
	},

	// day extracts the day of month (1-31) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.day": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Day()))}
		},
	},
	// hour extracts the hour (0-23) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.hour": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Hour()))}
		},
	},

	// minute extracts the minute (0-59) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.minute": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Minute()))}
		},
	},
	// second extracts the second (0-59) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.second": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Second()))}
		},
	},

	// weekday extracts the day of week (0=Sunday, 6=Saturday).
	// Takes optional timestamp. Returns int.
	"time.weekday": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.Weekday()))}
		},
	},
	// weekday_name returns the name of the weekday (e.g., "Monday").
	// Takes optional timestamp. Returns string.
	"time.weekday_name": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.String{Value: t.Weekday().String()}
		},
	},

	// month_name returns the name of the month (e.g., "January").
	// Takes optional timestamp. Returns string.
	"time.month_name": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.String{Value: t.Month().String()}
		},
	},
	// day_of_year extracts the day of year (1-366) from a timestamp.
	// Takes optional timestamp. Returns int.
	"time.day_of_year": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.Integer{Value: big.NewInt(int64(t.YearDay()))}
		},
	},

	// ============================================================================
	// Formatting
	// ============================================================================

	// format formats a timestamp using a format string.
	// Takes format and optional timestamp. Returns string.
	"time.format": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Code: "E7001", Message: "time.format() takes 1 or 2 arguments: format or format, timestamp"}
			}

			var t time.Time
			var format string

			// First argument is always the format string
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "time.format() requires a string format as first argument"}
			}
			format = str.Value

			if len(args) == 1 {
				// No timestamp provided, use current time
				t = time.Now()
			} else {
				// Second argument is the timestamp
				ts, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.format() requires an integer timestamp as second argument"}
				}
				t = time.Unix(ts.Value.Int64(), 0)
			}

			goFormat := convertFormat(format)
			return &object.String{Value: t.Format(goFormat)}
		},
	},
	// iso formats a timestamp as ISO 8601 (RFC 3339).
	// Takes optional timestamp. Returns string.
	"time.iso": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.String{Value: t.Format(time.RFC3339)}
		},
	},

	// date formats a timestamp as YYYY-MM-DD.
	// Takes optional timestamp. Returns string.
	"time.date": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.String{Value: t.Format("2006-01-02")}
		},
	},
	// clock formats a timestamp as HH:MM:SS.
	// Takes optional timestamp. Returns string.
	"time.clock": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			return &object.String{Value: t.Format("15:04:05")}
		},
	},

	// ============================================================================
	// Parsing
	// ============================================================================

	// parse parses a time string into a Unix timestamp.
	// Takes string and format. Returns int or Error.
	"time.parse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.parse() takes exactly 2 arguments (string, format)"}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "time.parse() requires a string"}
			}
			format, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "time.parse() requires a format string"}
			}

			goFormat := convertFormat(format.Value)
			t, err := time.Parse(goFormat, str.Value)
			if err != nil {
				return &object.Error{Code: "E11001", Message: "time.parse() failed: " + err.Error()}
			}
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// ============================================================================
	// Creating Timestamps
	// ============================================================================

	// make creates a timestamp from year, month, day, and optional time.
	// Takes year, month, day, and optional hour, minute, second. Returns int.
	"time.make": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 || len(args) > 6 {
				return &object.Error{Code: "E7001", Message: "time.make() takes 3 to 6 arguments (year, month, day, [hour, minute, second])"}
			}

			year, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
			}
			month, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
			}
			day, ok := args[2].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
			}

			hour, minute, second := 0, 0, 0
			if len(args) > 3 {
				h, ok := args[3].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
				}
				hour = int(h.Value.Int64())
			}
			if len(args) > 4 {
				m, ok := args[4].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
				}
				minute = int(m.Value.Int64())
			}
			if len(args) > 5 {
				s, ok := args[5].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.make() requires integer arguments"}
				}
				second = int(s.Value.Int64())
			}

			t := time.Date(int(year.Value.Int64()), time.Month(month.Value.Int64()), int(day.Value.Int64()),
				hour, minute, second, 0, time.Local)
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// ============================================================================
	// Arithmetic
	// ============================================================================

	// add_seconds adds seconds to a timestamp.
	// Takes timestamp and seconds. Returns int.
	"time.add_seconds": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_seconds() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_seconds() requires integer timestamp"}
			}
			secs, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_seconds() requires integer seconds"}
			}
			result := new(big.Int).Add(ts.Value, secs.Value)
			return &object.Integer{Value: result}
		},
	},
	// add_minutes adds minutes to a timestamp.
	// Takes timestamp and minutes. Returns int.
	"time.add_minutes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_minutes() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_minutes() requires integer timestamp"}
			}
			mins, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_minutes() requires integer minutes"}
			}
			result := new(big.Int).Add(ts.Value, new(big.Int).Mul(mins.Value, big.NewInt(60)))
			return &object.Integer{Value: result}
		},
	},
	// add_hours adds hours to a timestamp.
	// Takes timestamp and hours. Returns int.
	"time.add_hours": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_hours() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_hours() requires integer timestamp"}
			}
			hours, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_hours() requires integer hours"}
			}
			result := new(big.Int).Add(ts.Value, new(big.Int).Mul(hours.Value, big.NewInt(3600)))
			return &object.Integer{Value: result}
		},
	},
	// add_days adds days to a timestamp.
	// Takes timestamp and days. Returns int.
	"time.add_days": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_days() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_days() requires integer timestamp"}
			}
			days, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_days() requires integer days"}
			}
			result := new(big.Int).Add(ts.Value, new(big.Int).Mul(days.Value, big.NewInt(86400)))
			return &object.Integer{Value: result}
		},
	},
	// add_weeks adds weeks to a timestamp.
	// Takes timestamp and weeks. Returns int.
	"time.add_weeks": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_weeks() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_weeks() requires integer timestamp"}
			}
			weeks, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_weeks() requires integer weeks"}
			}
			result := new(big.Int).Add(ts.Value, new(big.Int).Mul(weeks.Value, big.NewInt(604800)))
			return &object.Integer{Value: result}
		},
	},
	// add_months adds months to a timestamp (handles month-end correctly).
	// Takes timestamp and months. Returns int.
	"time.add_months": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_months() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_months() requires integer timestamp"}
			}
			months, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_months() requires integer months"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			originalDay := t.Day()

			// Move to target month (first of month to avoid overflow issues)
			targetYear := t.Year()
			targetMonth := int(t.Month()) + int(months.Value.Int64())

			// Normalize month/year
			for targetMonth > 12 {
				targetMonth -= 12
				targetYear++
			}
			for targetMonth < 1 {
				targetMonth += 12
				targetYear--
			}

			// Get last day of target month
			lastDay := daysInMonth(targetYear, time.Month(targetMonth))

			// Clamp day to valid range
			day := originalDay
			if day > lastDay {
				day = lastDay
			}

			result := time.Date(targetYear, time.Month(targetMonth), day,
				t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
			return &object.Integer{Value: big.NewInt(result.Unix())}
		},
	},
	// add_years adds years to a timestamp.
	// Takes timestamp and years. Returns int.
	"time.add_years": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.add_years() takes exactly 2 arguments"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_years() requires integer timestamp"}
			}
			years, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.add_years() requires integer years"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			t = t.AddDate(int(years.Value.Int64()), 0, 0)
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// ============================================================================
	// Differences
	// ============================================================================

	// diff calculates the difference in seconds between two timestamps.
	// Takes two timestamps. Returns int.
	"time.diff": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.diff() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff() requires integer timestamps"}
			}
			result := new(big.Int).Sub(ts1.Value, ts2.Value)
			return &object.Integer{Value: result}
		},
	},
	// diff_days calculates the difference in days between two timestamps.
	// Takes two timestamps. Returns int.
	"time.diff_days": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.diff_days() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_days() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_days() requires integer timestamps"}
			}
			result := new(big.Int).Quo(new(big.Int).Sub(ts1.Value, ts2.Value), big.NewInt(86400))
			return &object.Integer{Value: result}
		},
	},
	// diff_hours calculates the difference in hours between two timestamps.
	// Takes two timestamps. Returns int.
	"time.diff_hours": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.diff_hours() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_hours() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_hours() requires integer timestamps"}
			}
			result := new(big.Int).Quo(new(big.Int).Sub(ts1.Value, ts2.Value), big.NewInt(3600))
			return &object.Integer{Value: result}
		},
	},
	// diff_minutes calculates the difference in minutes between two timestamps.
	// Takes two timestamps. Returns int.
	"time.diff_minutes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.diff_minutes() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_minutes() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.diff_minutes() requires integer timestamps"}
			}
			result := new(big.Int).Quo(new(big.Int).Sub(ts1.Value, ts2.Value), big.NewInt(60))
			return &object.Integer{Value: result}
		},
	},

	// ============================================================================
	// Comparisons
	// ============================================================================

	// is_before checks if the first timestamp is before the second.
	// Takes two timestamps. Returns bool.
	"time.is_before": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.is_before() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_before() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_before() requires integer timestamps"}
			}
			if ts1.Value.Cmp(ts2.Value) < 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},
	// is_after checks if the first timestamp is after the second.
	// Takes two timestamps. Returns bool.
	"time.is_after": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.is_after() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_after() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_after() requires integer timestamps"}
			}
			if ts1.Value.Cmp(ts2.Value) > 0 {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// ============================================================================
	// Timezone
	// ============================================================================

	// timezone returns the local timezone name (e.g., "EST").
	// Takes no arguments. Returns string.
	"time.timezone": {
		Fn: func(args ...object.Object) object.Object {
			name, _ := time.Now().Zone()
			return &object.String{Value: name}
		},
	},

	// utc_offset returns the local UTC offset in seconds.
	// Takes no arguments. Returns int.
	"time.utc_offset": {
		Fn: func(args ...object.Object) object.Object {
			_, offset := time.Now().Zone()
			return &object.Integer{Value: big.NewInt(int64(offset))}
		},
	},

	// ============================================================================
	// Special Checks
	// ============================================================================

	// is_leap_year checks if a year is a leap year.
	// Takes optional year. Returns bool.
	"time.is_leap_year": {
		Fn: func(args ...object.Object) object.Object {
			var year int
			if len(args) == 0 {
				year = time.Now().Year()
			} else {
				y, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.is_leap_year() requires an integer year"}
				}
				year = int(y.Value.Int64())
			}
			if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
				return object.TRUE
			}
			return object.FALSE
		},
	},
	// days_in_month returns the number of days in a month.
	// Takes optional year and month. Returns int.
	"time.days_in_month": {
		Fn: func(args ...object.Object) object.Object {
			var year, month int
			if len(args) == 0 {
				now := time.Now()
				year = now.Year()
				month = int(now.Month())
			} else if len(args) == 2 {
				y, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.days_in_month() requires integer arguments"}
				}
				m, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Code: "E7004", Message: "time.days_in_month() requires integer arguments"}
				}
				year = int(y.Value.Int64())
				month = int(m.Value.Int64())
			} else {
				return &object.Error{Code: "E7001", Message: "time.days_in_month() takes 0 or 2 arguments"}
			}

			t := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC)
			return &object.Integer{Value: big.NewInt(int64(t.Day()))}
		},
	},

	// ============================================================================
	// Start/End of Periods
	// ============================================================================

	// start_of_day returns the timestamp for midnight of that day.
	// Takes optional timestamp. Returns int.
	"time.start_of_day": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			return &object.Integer{Value: big.NewInt(start.Unix())}
		},
	},
	// end_of_day returns the timestamp for 23:59:59 of that day.
	// Takes optional timestamp. Returns int.
	"time.end_of_day": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			return &object.Integer{Value: big.NewInt(end.Unix())}
		},
	},

	// start_of_month returns the timestamp for the first day of the month.
	// Takes optional timestamp. Returns int.
	"time.start_of_month": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			return &object.Integer{Value: big.NewInt(start.Unix())}
		},
	},
	// end_of_month returns the timestamp for the last day of the month.
	// Takes optional timestamp. Returns int.
	"time.end_of_month": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			end := time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 0, t.Location())
			return &object.Integer{Value: big.NewInt(end.Unix())}
		},
	},

	// start_of_year returns the timestamp for January 1st of that year.
	// Takes optional timestamp. Returns int.
	"time.start_of_year": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			start := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
			return &object.Integer{Value: big.NewInt(start.Unix())}
		},
	},
	// end_of_year returns the timestamp for December 31st of that year.
	// Takes optional timestamp. Returns int.
	"time.end_of_year": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			end := time.Date(t.Year(), 12, 31, 23, 59, 59, 0, t.Location())
			return &object.Integer{Value: big.NewInt(end.Unix())}
		},
	},

	// ============================================================================
	// Calendar Utilities
	// ============================================================================

	// quarter returns the quarter (1-4) for a timestamp.
	// Takes optional timestamp. Returns int.
	"time.quarter": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			month := int(t.Month())
			quarter := (month-1)/3 + 1
			return &object.Integer{Value: big.NewInt(int64(quarter))}
		},
	},
	// week_of_year returns the ISO week number (1-53).
	// Takes optional timestamp. Returns int.
	"time.week_of_year": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			_, week := t.ISOWeek()
			return &object.Integer{Value: big.NewInt(int64(week))}
		},
	},

	// ============================================================================
	// Timing/Benchmarking
	// ============================================================================

	// tick returns a high-resolution timestamp in nanoseconds for timing.
	// Takes no arguments. Returns int.
	"time.tick": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(time.Now().UnixNano())}
		},
	},
	// elapsed_ms calculates elapsed time in milliseconds since a tick.
	// Takes a start tick. Returns float.
	"time.elapsed_ms": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.elapsed_ms() takes exactly 1 argument (start tick)"}
			}
			start, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.elapsed_ms() requires integer tick"}
			}
			elapsed := time.Now().UnixNano() - start.Value.Int64()
			return &object.Float{Value: float64(elapsed) / 1e6}
		},
	},

	// ============================================================================
	// Weekday Constants
	// ============================================================================

	// SUNDAY weekday constant (0).
	"time.SUNDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(0)}
		},
		IsConstant: true,
	},
	// MONDAY weekday constant (1).
	"time.MONDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},

	// TUESDAY weekday constant (2).
	"time.TUESDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},
	// WEDNESDAY weekday constant (3).
	"time.WEDNESDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(3)}
		},
		IsConstant: true,
	},

	// THURSDAY weekday constant (4).
	"time.THURSDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(4)}
		},
		IsConstant: true,
	},
	// FRIDAY weekday constant (5).
	"time.FRIDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(5)}
		},
		IsConstant: true,
	},

	// SATURDAY weekday constant (6).
	"time.SATURDAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(6)}
		},
		IsConstant: true,
	},

	// ============================================================================
	// Month Constants
	// ============================================================================

	// JANUARY month constant (1).
	"time.JANUARY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},
	// FEBRUARY month constant (2).
	"time.FEBRUARY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(2)}
		},
		IsConstant: true,
	},

	// MARCH month constant (3).
	"time.MARCH": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(3)}
		},
		IsConstant: true,
	},
	// APRIL month constant (4).
	"time.APRIL": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(4)}
		},
		IsConstant: true,
	},

	// MAY month constant (5).
	"time.MAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(5)}
		},
		IsConstant: true,
	},
	// JUNE month constant (6).
	"time.JUNE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(6)}
		},
		IsConstant: true,
	},

	// JULY month constant (7).
	"time.JULY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(7)}
		},
		IsConstant: true,
	},
	// AUGUST month constant (8).
	"time.AUGUST": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(8)}
		},
		IsConstant: true,
	},

	// SEPTEMBER month constant (9).
	"time.SEPTEMBER": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(9)}
		},
		IsConstant: true,
	},
	// OCTOBER month constant (10).
	"time.OCTOBER": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(10)}
		},
		IsConstant: true,
	},

	// NOVEMBER month constant (11).
	"time.NOVEMBER": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(11)}
		},
		IsConstant: true,
	},
	// DECEMBER month constant (12).
	"time.DECEMBER": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(12)}
		},
		IsConstant: true,
	},

	// ============================================================================
	// Duration Constants (in seconds)
	// ============================================================================

	// SECOND duration constant (1 second).
	"time.SECOND": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(1)}
		},
		IsConstant: true,
	},
	// MINUTE duration constant (60 seconds).
	"time.MINUTE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(60)}
		},
		IsConstant: true,
	},

	// HOUR duration constant (3600 seconds).
	"time.HOUR": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(3600)}
		},
		IsConstant: true,
	},
	// DAY duration constant (86400 seconds).
	"time.DAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(86400)}
		},
		IsConstant: true,
	},

	// WEEK duration constant (604800 seconds).
	"time.WEEK": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(604800)}
		},
		IsConstant: true,
	},

	// ============================================================================
	// Unix Conversion
	// ============================================================================

	// from_unix converts a Unix timestamp (seconds) to a timestamp.
	// Takes seconds. Returns int.
	"time.from_unix": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.from_unix() takes exactly 1 argument"}
			}
			secs, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.from_unix() requires an integer"}
			}
			t := time.Unix(secs.Value.Int64(), 0)
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// from_unix_ms converts a Unix timestamp (milliseconds) to a timestamp.
	// Takes milliseconds. Returns int.
	"time.from_unix_ms": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.from_unix_ms() takes exactly 1 argument"}
			}
			ms, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.from_unix_ms() requires an integer"}
			}

			t := time.UnixMilli(ms.Value.Int64())
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// to_unix converts a timestamp to Unix seconds.
	// Takes timestamp. Returns int.
	"time.to_unix": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.to_unix() takes exactly 1 argument"}
			}
			ts, ok := args[0].(*object.Integer)

			if !ok {
				return &object.Error{Code: "E7004", Message: "time.to_unix() requires an integer timestamp"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			return &object.Integer{Value: big.NewInt(t.Unix())}
		},
	},

	// to_unix_ms converts a timestamp to Unix milliseconds.
	// Takes timestamp. Returns int.
	"time.to_unix_ms": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.to_unix_ms() takes exactly 1 argument"}
			}

			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.to_unix_ms() requires an integer timestamp"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			return &object.Integer{Value: big.NewInt(t.UnixMilli())}
		},
	},

	// ============================================================================
	// Day Checks
	// ============================================================================

	// is_weekend checks if a timestamp falls on Saturday or Sunday.
	// Takes optional timestamp. Returns bool.
	"time.is_weekend": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)

			wd := t.Weekday()
			if wd == time.Sunday || wd == time.Saturday {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_weekday checks if a timestamp falls on Monday through Friday.
	// Takes optional timestamp. Returns bool.
	"time.is_weekday": {
		Fn: func(args ...object.Object) object.Object {
			t := getTime(args)
			wd := t.Weekday()
			if wd >= time.Monday && wd <= time.Friday {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// is_today checks if a timestamp falls on today's date.
	// Takes timestamp. Returns bool.
	"time.is_today": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.is_today() takes exactly 1 argument"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_today() requires an integer timestamp"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			now := time.Now()

			return &object.Boolean{Value: t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()}
		},
	},

	// is_same_day checks if two timestamps fall on the same calendar day.
	// Takes two timestamps. Returns bool.
	"time.is_same_day": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "time.is_same_day() takes exactly 2 arguments"}
			}
			ts1, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_same_day() requires integer timestamps"}
			}
			ts2, ok := args[1].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.is_same_day() requires integer timestamps"}
			}
			t1 := time.Unix(ts1.Value.Int64(), 0)
			t2 := time.Unix(ts2.Value.Int64(), 0)

			return &object.Boolean{Value: t1.Year() == t2.Year() &&
				t1.Month() == t2.Month() &&
				t1.Day() == t2.Day()}
		},
	},

	// ============================================================================
	// Relative Time
	// ============================================================================

	// relative formats a timestamp as a human-readable relative time.
	// Takes timestamp. Returns string (e.g., "2 hours ago").
	"time.relative": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "time.relative() takes exactly 1 argument"}
			}
			ts, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "time.relative() requires an integer timestamp"}
			}
			t := time.Unix(ts.Value.Int64(), 0)
			now := time.Now()
			diff := now.Sub(t)
			// Handle future times
			if diff < 0 {
				diff = -diff
				if diff < time.Second {
					return &object.String{Value: "just now"}
				} else if diff < time.Minute {
					secs := int(diff.Seconds())
					return &object.String{Value: "in " + strconv.Itoa(secs) + " seconds"}
				} else if diff < time.Hour {
					mins := int(diff.Minutes())
					if mins == 1 {
						return &object.String{Value: "in 1 minute"}
					}
					return &object.String{Value: "in " + strconv.Itoa(mins) + " minutes"}

				} else if diff < 24*time.Hour {
					hours := int(diff.Hours())
					if hours == 1 {
						return &object.String{Value: "in 1 hour"}
					}
					return &object.String{Value: "in " + strconv.Itoa(hours) + " hours"}
				} else {
					days := int(diff.Hours() / 24)
					if days == 1 {
						return &object.String{Value: "in 1 day"}
					}
					return &object.String{Value: "in " + strconv.Itoa(days) + " days"}
				}
			}

			// Handle past time
			if diff < time.Second {
				return &object.String{Value: "just now"}
			} else if diff < time.Minute {
				secs := int(diff.Seconds())
				if secs == 1 {
					return &object.String{Value: "1 second ago"}
				}
				return &object.String{Value: strconv.Itoa(secs) + " seconds ago"}
			} else if diff < time.Hour {
				mins := int(diff.Minutes())
				if mins == 1 {
					return &object.String{Value: "1 minute ago"}
				}
				return &object.String{Value: strconv.Itoa(mins) + " minutes ago"}
			} else if diff < 24*time.Hour {
				hours := int(diff.Hours())
				if hours == 1 {
					return &object.String{Value: "1 hour ago"}
				}
				return &object.String{Value: strconv.Itoa(hours) + " hours ago"}
			} else {
				days := int(diff.Hours() / 24)
				if days == 1 {
					return &object.String{Value: "1 day ago"}
				}
				return &object.String{Value: strconv.Itoa(days) + " days ago"}
			}
		},
	},
}

// Helper to get time from args (current time if no args)
func getTime(args []object.Object) time.Time {
	if len(args) == 0 {
		return time.Now()
	}
	if ts, ok := args[0].(*object.Integer); ok {
		return time.Unix(ts.Value.Int64(), 0)
	}
	return time.Now()
}

// Convert common format patterns to Go format
// Uses ordered slice to ensure longer patterns (YYYY) are replaced before shorter ones (YY)
func convertFormat(format string) string {
	// Order matters: longer patterns must come first to avoid partial replacements
	replacements := []struct{ from, to string }{
		{"YYYY", "2006"},
		{"YY", "06"},
		{"MM", "01"},
		{"DD", "02"},
		{"HH", "15"},
		{"hh", "03"},
		{"mm", "04"},
		{"ss", "05"},
		{"SSS", "000"},
		{"ZZ", "-07:00"}, // ZZ before Z
		{"Z", "-0700"},
		{"A", "PM"},
		{"a", "pm"},
	}

	result := format
	for _, r := range replacements {
		result = replaceAll(result, r.from, r.to)
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// daysInMonth returns the number of days in the given month
func daysInMonth(year int, month time.Month) int {
	// Use Go's time normalization: day 0 of month+1 is the last day of month
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
