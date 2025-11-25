package interpreter
// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"time"
)

var timeBuiltins = map[string]*Builtin{
	// Current time
	"time.now": {
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().Unix()}
		},
	},
	"time.now_ms": {
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().UnixMilli()}
		},
	},
	"time.now_ns": {
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().UnixNano()}
		},
	},

	// Sleep/delay
	"time.sleep": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("time.sleep() takes exactly 1 argument (seconds)")
			}
			switch v := args[0].(type) {
			case *Integer:
				time.Sleep(time.Duration(v.Value) * time.Second)
			case *Float:
				time.Sleep(time.Duration(v.Value * float64(time.Second)))
			default:
				return newError("time.sleep() requires a number")
			}
			return NIL
		},
	},
	"time.sleep_ms": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("time.sleep_ms() takes exactly 1 argument (milliseconds)")
			}
			ms, ok := args[0].(*Integer)
			if !ok {
				return newError("time.sleep_ms() requires an integer")
			}
			time.Sleep(time.Duration(ms.Value) * time.Millisecond)
			return NIL
		},
	},

	// Time components
	"time.year": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Year())}
		},
	},
	"time.month": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Month())}
		},
	},
	"time.day": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Day())}
		},
	},
	"time.hour": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Hour())}
		},
	},
	"time.minute": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Minute())}
		},
	},
	"time.second": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Second())}
		},
	},
	"time.weekday": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.Weekday())}
		},
	},
	"time.weekday_name": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &String{Value: t.Weekday().String()}
		},
	},
	"time.month_name": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &String{Value: t.Month().String()}
		},
	},
	"time.day_of_year": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &Integer{Value: int64(t.YearDay())}
		},
	},

	// Formatting
	"time.format": {
		Fn: func(args ...Object) Object {
			if len(args) < 1 || len(args) > 2 {
				return newError("time.format() takes 1 or 2 arguments")
			}

			var t time.Time
			var format string

			if len(args) == 1 {
				// format only, use current time
				str, ok := args[0].(*String)
				if !ok {
					return newError("time.format() requires a string format")
				}
				t = time.Now()
				format = str.Value
			} else {
				// timestamp and format
				ts, ok := args[0].(*Integer)
				if !ok {
					return newError("time.format() requires an integer timestamp")
				}
				str, ok := args[1].(*String)
				if !ok {
					return newError("time.format() requires a string format")
				}
				t = time.Unix(ts.Value, 0)
				format = str.Value
			}

			// Convert common format strings to Go format
			goFormat := convertFormat(format)
			return &String{Value: t.Format(goFormat)}
		},
	},
	"time.iso": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &String{Value: t.Format(time.RFC3339)}
		},
	},
	"time.date": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &String{Value: t.Format("2006-01-02")}
		},
	},
	"time.clock": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			return &String{Value: t.Format("15:04:05")}
		},
	},

	// Parsing
	"time.parse": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.parse() takes exactly 2 arguments (string, format)")
			}
			str, ok := args[0].(*String)
			if !ok {
				return newError("time.parse() requires a string")
			}
			format, ok := args[1].(*String)
			if !ok {
				return newError("time.parse() requires a format string")
			}

			goFormat := convertFormat(format.Value)
			t, err := time.Parse(goFormat, str.Value)
			if err != nil {
				return newError("time.parse() failed: %s", err.Error())
			}
			return &Integer{Value: t.Unix()}
		},
	},

	// Creating timestamps
	"time.make": {
		Fn: func(args ...Object) Object {
			if len(args) < 3 || len(args) > 6 {
				return newError("time.make() takes 3 to 6 arguments (year, month, day, [hour, minute, second])")
			}

			year, ok := args[0].(*Integer)
			if !ok {
				return newError("time.make() requires integer arguments")
			}
			month, ok := args[1].(*Integer)
			if !ok {
				return newError("time.make() requires integer arguments")
			}
			day, ok := args[2].(*Integer)
			if !ok {
				return newError("time.make() requires integer arguments")
			}

			hour, minute, second := 0, 0, 0
			if len(args) > 3 {
				h, ok := args[3].(*Integer)
				if !ok {
					return newError("time.make() requires integer arguments")
				}
				hour = int(h.Value)
			}
			if len(args) > 4 {
				m, ok := args[4].(*Integer)
				if !ok {
					return newError("time.make() requires integer arguments")
				}
				minute = int(m.Value)
			}
			if len(args) > 5 {
				s, ok := args[5].(*Integer)
				if !ok {
					return newError("time.make() requires integer arguments")
				}
				second = int(s.Value)
			}

			t := time.Date(int(year.Value), time.Month(month.Value), int(day.Value),
				hour, minute, second, 0, time.Local)
			return &Integer{Value: t.Unix()}
		},
	},

	// Arithmetic
	"time.add_seconds": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.add_seconds() takes exactly 2 arguments")
			}
			ts, ok := args[0].(*Integer)
			if !ok {
				return newError("time.add_seconds() requires integer timestamp")
			}
			secs, ok := args[1].(*Integer)
			if !ok {
				return newError("time.add_seconds() requires integer seconds")
			}
			return &Integer{Value: ts.Value + secs.Value}
		},
	},
	"time.add_minutes": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.add_minutes() takes exactly 2 arguments")
			}
			ts, ok := args[0].(*Integer)
			if !ok {
				return newError("time.add_minutes() requires integer timestamp")
			}
			mins, ok := args[1].(*Integer)
			if !ok {
				return newError("time.add_minutes() requires integer minutes")
			}
			return &Integer{Value: ts.Value + mins.Value*60}
		},
	},
	"time.add_hours": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.add_hours() takes exactly 2 arguments")
			}
			ts, ok := args[0].(*Integer)
			if !ok {
				return newError("time.add_hours() requires integer timestamp")
			}
			hours, ok := args[1].(*Integer)
			if !ok {
				return newError("time.add_hours() requires integer hours")
			}
			return &Integer{Value: ts.Value + hours.Value*3600}
		},
	},
	"time.add_days": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.add_days() takes exactly 2 arguments")
			}
			ts, ok := args[0].(*Integer)
			if !ok {
				return newError("time.add_days() requires integer timestamp")
			}
			days, ok := args[1].(*Integer)
			if !ok {
				return newError("time.add_days() requires integer days")
			}
			return &Integer{Value: ts.Value + days.Value*86400}
		},
	},

	// Differences
	"time.diff": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.diff() takes exactly 2 arguments")
			}
			ts1, ok := args[0].(*Integer)
			if !ok {
				return newError("time.diff() requires integer timestamps")
			}
			ts2, ok := args[1].(*Integer)
			if !ok {
				return newError("time.diff() requires integer timestamps")
			}
			return &Integer{Value: ts1.Value - ts2.Value}
		},
	},
	"time.diff_days": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.diff_days() takes exactly 2 arguments")
			}
			ts1, ok := args[0].(*Integer)
			if !ok {
				return newError("time.diff_days() requires integer timestamps")
			}
			ts2, ok := args[1].(*Integer)
			if !ok {
				return newError("time.diff_days() requires integer timestamps")
			}
			return &Integer{Value: (ts1.Value - ts2.Value) / 86400}
		},
	},

	// Comparisons
	"time.is_before": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.is_before() takes exactly 2 arguments")
			}
			ts1, ok := args[0].(*Integer)
			if !ok {
				return newError("time.is_before() requires integer timestamps")
			}
			ts2, ok := args[1].(*Integer)
			if !ok {
				return newError("time.is_before() requires integer timestamps")
			}
			if ts1.Value < ts2.Value {
				return TRUE
			}
			return FALSE
		},
	},
	"time.is_after": {
		Fn: func(args ...Object) Object {
			if len(args) != 2 {
				return newError("time.is_after() takes exactly 2 arguments")
			}
			ts1, ok := args[0].(*Integer)
			if !ok {
				return newError("time.is_after() requires integer timestamps")
			}
			ts2, ok := args[1].(*Integer)
			if !ok {
				return newError("time.is_after() requires integer timestamps")
			}
			if ts1.Value > ts2.Value {
				return TRUE
			}
			return FALSE
		},
	},

	// Timezone
	"time.timezone": {
		Fn: func(args ...Object) Object {
			name, _ := time.Now().Zone()
			return &String{Value: name}
		},
	},
	"time.utc_offset": {
		Fn: func(args ...Object) Object {
			_, offset := time.Now().Zone()
			return &Integer{Value: int64(offset)}
		},
	},

	// Special checks
	"time.is_leap_year": {
		Fn: func(args ...Object) Object {
			var year int
			if len(args) == 0 {
				year = time.Now().Year()
			} else {
				y, ok := args[0].(*Integer)
				if !ok {
					return newError("time.is_leap_year() requires an integer year")
				}
				year = int(y.Value)
			}
			if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
				return TRUE
			}
			return FALSE
		},
	},
	"time.days_in_month": {
		Fn: func(args ...Object) Object {
			var year, month int
			if len(args) == 0 {
				now := time.Now()
				year = now.Year()
				month = int(now.Month())
			} else if len(args) == 2 {
				y, ok := args[0].(*Integer)
				if !ok {
					return newError("time.days_in_month() requires integer arguments")
				}
				m, ok := args[1].(*Integer)
				if !ok {
					return newError("time.days_in_month() requires integer arguments")
				}
				year = int(y.Value)
				month = int(m.Value)
			} else {
				return newError("time.days_in_month() takes 0 or 2 arguments")
			}

			// Get first day of next month, then subtract one day
			t := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC)
			return &Integer{Value: int64(t.Day())}
		},
	},

	// Start/end of periods
	"time.start_of_day": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			return &Integer{Value: start.Unix()}
		},
	},
	"time.end_of_day": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			return &Integer{Value: end.Unix()}
		},
	},
	"time.start_of_month": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			return &Integer{Value: start.Unix()}
		},
	},
	"time.end_of_month": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			end := time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 0, t.Location())
			return &Integer{Value: end.Unix()}
		},
	},
	"time.start_of_year": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			start := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
			return &Integer{Value: start.Unix()}
		},
	},
	"time.end_of_year": {
		Fn: func(args ...Object) Object {
			t := getTime(args)
			end := time.Date(t.Year(), 12, 31, 23, 59, 59, 0, t.Location())
			return &Integer{Value: end.Unix()}
		},
	},

	// Timing/benchmarking
	"time.tick": {
		Fn: func(args ...Object) Object {
			return &Integer{Value: time.Now().UnixNano()}
		},
	},
	"time.elapsed_ms": {
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return newError("time.elapsed_ms() takes exactly 1 argument (start tick)")
			}
			start, ok := args[0].(*Integer)
			if !ok {
				return newError("time.elapsed_ms() requires integer tick")
			}
			elapsed := time.Now().UnixNano() - start.Value
			return &Float{Value: float64(elapsed) / 1e6}
		},
	},
}

// Helper to get time from args (current time if no args)
func getTime(args []Object) time.Time {
	if len(args) == 0 {
		return time.Now()
	}
	if ts, ok := args[0].(*Integer); ok {
		return time.Unix(ts.Value, 0)
	}
	return time.Now()
}

// Convert common format patterns to Go format
func convertFormat(format string) string {
	// Map common format specifiers to Go reference time
	replacements := map[string]string{
		"YYYY": "2006",
		"YY":   "06",
		"MM":   "01",
		"DD":   "02",
		"HH":   "15",
		"hh":   "03",
		"mm":   "04",
		"ss":   "05",
		"SSS":  "000",
		"A":    "PM",
		"a":    "pm",
		"Z":    "-0700",
		"ZZ":   "-07:00",
	}

	result := format
	for k, v := range replacements {
		result = replaceAll(result, k, v)
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

func init() {
	for name, builtin := range timeBuiltins {
		builtins[name] = builtin
	}
}
