package stdlib

import (
	"math/big"
	"testing"
	"time"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// time.now / time.now_ms / time.now_ns
// ============================================================================

func TestTimeNowReturnsInteger(t *testing.T) {
	fn := TimeBuiltins["time.now"]
	result := fn.Fn()
	if _, ok := result.(*object.Integer); !ok {
		t.Errorf("expected Integer, got %T", result)
	}
}

func TestTimeNowMsReturnsInteger(t *testing.T) {
	fn := TimeBuiltins["time.now_ms"]
	result := fn.Fn()
	if _, ok := result.(*object.Integer); !ok {
		t.Errorf("expected Integer, got %T", result)
	}
}

func TestTimeNowNsReturnsInteger(t *testing.T) {
	fn := TimeBuiltins["time.now_ns"]
	result := fn.Fn()
	if _, ok := result.(*object.Integer); !ok {
		t.Errorf("expected Integer, got %T", result)
	}
}

func TestTimeNowMsIsGreater(t *testing.T) {
	fnNow := TimeBuiltins["time.now"]
	fnMs := TimeBuiltins["time.now_ms"]

	now := fnNow.Fn().(*object.Integer).Value.Int64()
	ms := fnMs.Fn().(*object.Integer).Value.Int64()

	// ms should be roughly now * 1000
	if ms < now*1000-1000 || ms > now*1000+1000 {
		t.Errorf("now_ms should be ~= now * 1000, got now=%d, ms=%d", now, ms)
	}
}

// ============================================================================
// time.year / time.month / time.day / time.hour / time.minute / time.second
// ============================================================================

func TestTimeYearFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.year"]
	// 2024-01-15 12:30:45 UTC
	ts := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 2024 {
		t.Errorf("expected 2024, got %v", result)
	}
}

func TestTimeMonthFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.month"]
	ts := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 6 {
		t.Errorf("expected 6, got %v", result)
	}
}

func TestTimeDayFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.day"]
	ts := time.Date(2024, 6, 25, 12, 30, 45, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 25 {
		t.Errorf("expected 25, got %v", result)
	}
}

func TestTimeHourFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.hour"]
	// Use local time to avoid timezone issues
	ts := time.Date(2024, 6, 15, 14, 30, 45, 0, time.Local).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 14 {
		t.Errorf("expected 14, got %v", result)
	}
}

func TestTimeMinuteFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.minute"]
	ts := time.Date(2024, 6, 15, 12, 45, 30, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 45 {
		t.Errorf("expected 45, got %v", result)
	}
}

func TestTimeSecondFromTimestamp(t *testing.T) {
	fn := TimeBuiltins["time.second"]
	ts := time.Date(2024, 6, 15, 12, 30, 59, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if intVal, ok := result.(*object.Integer); !ok || intVal.Value.Int64() != 59 {
		t.Errorf("expected 59, got %v", result)
	}
}

// ============================================================================
// time.weekday / time.weekday_name
// ============================================================================

func TestTimeWeekday(t *testing.T) {
	fn := TimeBuiltins["time.weekday"]
	// Monday = 1 in EZ
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix() // Monday
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	// Monday should be 1
	if intVal.Value.Int64() != 1 {
		t.Errorf("expected 1 (Monday), got %d", intVal.Value.Int64())
	}
}

func TestTimeWeekdayName(t *testing.T) {
	fn := TimeBuiltins["time.weekday_name"]
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix() // Monday
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "Monday" {
		t.Errorf("expected 'Monday', got '%s'", strVal.Value)
	}
}

// ============================================================================
// time.month_name
// ============================================================================

func TestTimeMonthName(t *testing.T) {
	fn := TimeBuiltins["time.month_name"]
	ts := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC).Unix() // March
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}
	if strVal.Value != "March" {
		t.Errorf("expected 'March', got '%s'", strVal.Value)
	}
}

// ============================================================================
// time.day_of_year / time.week_of_year / time.quarter
// ============================================================================

func TestTimeDayOfYearUnit(t *testing.T) {
	fn := TimeBuiltins["time.day_of_year"]
	ts := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC).Unix() // Feb 1 = day 32
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 32 {
		t.Errorf("expected 32, got %d", intVal.Value.Int64())
	}
}

func TestTimeWeekOfYear(t *testing.T) {
	fn := TimeBuiltins["time.week_of_year"]
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if _, ok := result.(*object.Integer); !ok {
		t.Errorf("expected Integer, got %T", result)
	}
}

func TestTimeQuarter(t *testing.T) {
	fn := TimeBuiltins["time.quarter"]
	tests := []struct {
		month    int
		expected int64
	}{
		{1, 1}, {2, 1}, {3, 1},
		{4, 2}, {5, 2}, {6, 2},
		{7, 3}, {8, 3}, {9, 3},
		{10, 4}, {11, 4}, {12, 4},
	}
	for _, tt := range tests {
		ts := time.Date(2024, time.Month(tt.month), 15, 12, 0, 0, 0, time.UTC).Unix()
		result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
		intVal, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T", result)
		}
		if intVal.Value.Int64() != tt.expected {
			t.Errorf("month %d: expected Q%d, got Q%d", tt.month, tt.expected, intVal.Value.Int64())
		}
	}
}

// ============================================================================
// time.is_leap_year
// ============================================================================

func TestTimeIsLeapYear(t *testing.T) {
	fn := TimeBuiltins["time.is_leap_year"]
	tests := []struct {
		year     int64
		expected bool
	}{
		{2024, true},  // divisible by 4
		{2023, false}, // not divisible by 4
		{2000, true},  // divisible by 400
		{1900, false}, // divisible by 100 but not 400
	}
	for _, tt := range tests {
		// time.is_leap_year takes a year integer, not a timestamp
		result := fn.Fn(&object.Integer{Value: big.NewInt(tt.year)})
		expected := object.FALSE
		if tt.expected {
			expected = object.TRUE
		}
		if result != expected {
			t.Errorf("year %d: expected %v, got %v", tt.year, expected, result)
		}
	}
}

// ============================================================================
// time.days_in_month
// ============================================================================

func TestTimeDaysInMonth(t *testing.T) {
	fn := TimeBuiltins["time.days_in_month"]
	tests := []struct {
		year     int64
		month    int64
		expected int64
	}{
		{2024, 1, 31},  // January
		{2024, 2, 29},  // February (leap year)
		{2023, 2, 28},  // February (non-leap year)
		{2024, 4, 30},  // April
		{2024, 12, 31}, // December
	}
	for _, tt := range tests {
		// time.days_in_month takes (year, month) as integer arguments
		result := fn.Fn(
			&object.Integer{Value: big.NewInt(tt.year)},
			&object.Integer{Value: big.NewInt(tt.month)},
		)
		intVal, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("expected Integer, got %T: %v", result, result)
		}
		if intVal.Value.Int64() != tt.expected {
			t.Errorf("year %d month %d: expected %d days, got %d", tt.year, tt.month, tt.expected, intVal.Value.Int64())
		}
	}
}

// ============================================================================
// time.is_weekend / time.is_weekday
// ============================================================================

func TestTimeIsWeekendTrue(t *testing.T) {
	fn := TimeBuiltins["time.is_weekend"]
	// Saturday
	ts := time.Date(2024, 1, 13, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if result != object.TRUE {
		t.Errorf("expected TRUE for Saturday, got %v", result)
	}
}

func TestTimeIsWeekendFalse(t *testing.T) {
	fn := TimeBuiltins["time.is_weekend"]
	// Monday
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if result != object.FALSE {
		t.Errorf("expected FALSE for Monday, got %v", result)
	}
}

func TestTimeIsWeekdayTrue(t *testing.T) {
	fn := TimeBuiltins["time.is_weekday"]
	// Wednesday
	ts := time.Date(2024, 1, 17, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	if result != object.TRUE {
		t.Errorf("expected TRUE for Wednesday, got %v", result)
	}
}

// ============================================================================
// time.from_unix / time.to_unix / time.from_unix_ms / time.to_unix_ms
// ============================================================================

func TestTimeFromUnixUnit(t *testing.T) {
	fn := TimeBuiltins["time.from_unix"]
	result := fn.Fn(&object.Integer{Value: big.NewInt(1704067200)}) // 2024-01-01 00:00:00 UTC
	if _, ok := result.(*object.Integer); !ok {
		t.Errorf("expected Integer, got %T", result)
	}
}

func TestTimeToUnixUnit(t *testing.T) {
	fn := TimeBuiltins["time.to_unix"]
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != ts {
		t.Errorf("expected %d, got %d", ts, intVal.Value.Int64())
	}
}

// ============================================================================
// time.add_days / time.add_hours / time.add_minutes / time.add_seconds
// ============================================================================

func TestTimeAddDays(t *testing.T) {
	fn := TimeBuiltins["time.add_days"]
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(5)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	expected := time.Date(2024, 1, 6, 12, 0, 0, 0, time.UTC).Unix()
	if intVal.Value.Int64() != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value.Int64())
	}
}

func TestTimeAddHours(t *testing.T) {
	fn := TimeBuiltins["time.add_hours"]
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(3)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	expected := time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC).Unix()
	if intVal.Value.Int64() != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value.Int64())
	}
}

func TestTimeAddMinutes(t *testing.T) {
	fn := TimeBuiltins["time.add_minutes"]
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(30)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	expected := time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC).Unix()
	if intVal.Value.Int64() != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value.Int64())
	}
}

func TestTimeAddSeconds(t *testing.T) {
	fn := TimeBuiltins["time.add_seconds"]
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(45)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	expected := time.Date(2024, 1, 1, 12, 0, 45, 0, time.UTC).Unix()
	if intVal.Value.Int64() != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value.Int64())
	}
}

// ============================================================================
// time.add_months / time.add_years / time.add_weeks
// ============================================================================

func TestTimeAddMonths(t *testing.T) {
	fn := TimeBuiltins["time.add_months"]
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(2)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	resultTime := time.Unix(intVal.Value.Int64(), 0).UTC()
	if resultTime.Month() != 3 {
		t.Errorf("expected March, got %v", resultTime.Month())
	}
}

func TestTimeAddYears(t *testing.T) {
	fn := TimeBuiltins["time.add_years"]
	base := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(1)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	resultTime := time.Unix(intVal.Value.Int64(), 0).UTC()
	if resultTime.Year() != 2025 {
		t.Errorf("expected 2025, got %d", resultTime.Year())
	}
}

func TestTimeAddWeeks(t *testing.T) {
	fn := TimeBuiltins["time.add_weeks"]
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(base)}, &object.Integer{Value: big.NewInt(2)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	if intVal.Value.Int64() != expected {
		t.Errorf("expected %d, got %d", expected, intVal.Value.Int64())
	}
}

// ============================================================================
// time.diff / time.diff_days / time.diff_hours / time.diff_minutes
// ============================================================================

func TestTimeDiff(t *testing.T) {
	fn := TimeBuiltins["time.diff"]
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, 1, 1, 12, 0, 10, 0, time.UTC).Unix()
	// diff calculates first - second, so pass later time first
	result := fn.Fn(&object.Integer{Value: big.NewInt(t2)}, &object.Integer{Value: big.NewInt(t1)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 10 {
		t.Errorf("expected 10 seconds, got %d", intVal.Value.Int64())
	}
}

func TestTimeDiffDays(t *testing.T) {
	fn := TimeBuiltins["time.diff_days"]
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC).Unix()
	// diff_days calculates first - second, so pass later time first
	result := fn.Fn(&object.Integer{Value: big.NewInt(t2)}, &object.Integer{Value: big.NewInt(t1)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intVal.Value.Int64() != 10 {
		t.Errorf("expected 10 days, got %d", intVal.Value.Int64())
	}
}

// ============================================================================
// time.is_before / time.is_after / time.is_same_day
// ============================================================================

func TestTimeIsBefore(t *testing.T) {
	fn := TimeBuiltins["time.is_before"]
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(t1)}, &object.Integer{Value: big.NewInt(t2)})
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestTimeIsAfter(t *testing.T) {
	fn := TimeBuiltins["time.is_after"]
	t1 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(t1)}, &object.Integer{Value: big.NewInt(t2)})
	if result != object.TRUE {
		t.Errorf("expected TRUE, got %v", result)
	}
}

func TestTimeIsSameDayUnit(t *testing.T) {
	fn := TimeBuiltins["time.is_same_day"]
	t1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(t1)}, &object.Integer{Value: big.NewInt(t2)})
	// is_same_day returns *object.Boolean, not object.TRUE
	boolVal, ok := result.(*object.Boolean)
	if !ok || !boolVal.Value {
		t.Errorf("expected true, got %v", result)
	}
}

// ============================================================================
// time.format / time.parse
// ============================================================================

func TestTimeFormat(t *testing.T) {
	fn := TimeBuiltins["time.format"]
	ts := time.Date(2024, 1, 15, 14, 30, 45, 0, time.Local).Unix()
	// time.format takes format string first, timestamp second
	result := fn.Fn(&object.String{Value: "2006-01-02"}, &object.Integer{Value: big.NewInt(ts)})
	strVal, ok := result.(*object.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if strVal.Value != "2024-01-15" {
		t.Errorf("expected '2024-01-15', got '%s'", strVal.Value)
	}
}

func TestTimeParse(t *testing.T) {
	fn := TimeBuiltins["time.parse"]
	result := fn.Fn(&object.String{Value: "2024-01-15"}, &object.String{Value: "2006-01-02"})
	intVal, ok := result.(*object.Integer)
	if !ok {
		// Could return tuple with error
		if arr, ok := result.(*object.Array); ok && len(arr.Elements) > 0 {
			if _, ok := arr.Elements[0].(*object.Integer); ok {
				return // OK
			}
		}
		t.Fatalf("expected Integer or tuple, got %T", result)
	}
	parsed := time.Unix(intVal.Value.Int64(), 0).UTC()
	if parsed.Day() != 15 || parsed.Month() != 1 || parsed.Year() != 2024 {
		t.Errorf("parse mismatch: got %v", parsed)
	}
}

// ============================================================================
// time.start_of_day / time.end_of_day
// ============================================================================

func TestTimeStartOfDay(t *testing.T) {
	fn := TimeBuiltins["time.start_of_day"]
	ts := time.Date(2024, 1, 15, 14, 30, 45, 0, time.Local).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	// Use Local timezone since implementation uses Local
	resultTime := time.Unix(intVal.Value.Int64(), 0).Local()
	if resultTime.Hour() != 0 || resultTime.Minute() != 0 || resultTime.Second() != 0 {
		t.Errorf("expected 00:00:00, got %v", resultTime)
	}
}

func TestTimeEndOfDay(t *testing.T) {
	fn := TimeBuiltins["time.end_of_day"]
	ts := time.Date(2024, 1, 15, 14, 30, 45, 0, time.Local).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	// Use Local timezone since implementation uses Local
	resultTime := time.Unix(intVal.Value.Int64(), 0).Local()
	if resultTime.Hour() != 23 || resultTime.Minute() != 59 || resultTime.Second() != 59 {
		t.Errorf("expected 23:59:59, got %v", resultTime)
	}
}

// ============================================================================
// time.start_of_month / time.end_of_month
// ============================================================================

func TestTimeStartOfMonth(t *testing.T) {
	fn := TimeBuiltins["time.start_of_month"]
	ts := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC).Unix()
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	resultTime := time.Unix(intVal.Value.Int64(), 0).UTC()
	if resultTime.Day() != 1 {
		t.Errorf("expected day 1, got %d", resultTime.Day())
	}
}

func TestTimeEndOfMonth(t *testing.T) {
	fn := TimeBuiltins["time.end_of_month"]
	ts := time.Date(2024, 2, 15, 14, 30, 45, 0, time.Local).Unix() // Feb 2024 (leap year)
	result := fn.Fn(&object.Integer{Value: big.NewInt(ts)})
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	// Use Local timezone since implementation uses Local
	resultTime := time.Unix(intVal.Value.Int64(), 0).Local()
	if resultTime.Day() != 29 {
		t.Errorf("expected day 29 (leap year Feb), got %d", resultTime.Day())
	}
}

// ============================================================================
// time.make
// ============================================================================

func TestTimeMake(t *testing.T) {
	fn := TimeBuiltins["time.make"]
	result := fn.Fn(
		&object.Integer{Value: big.NewInt(2024)},
		&object.Integer{Value: big.NewInt(6)},
		&object.Integer{Value: big.NewInt(15)},
		&object.Integer{Value: big.NewInt(14)},
		&object.Integer{Value: big.NewInt(30)},
		&object.Integer{Value: big.NewInt(45)},
	)
	intVal, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	resultTime := time.Unix(intVal.Value.Int64(), 0).UTC()
	if resultTime.Year() != 2024 || resultTime.Month() != 6 || resultTime.Day() != 15 {
		t.Errorf("expected 2024-06-15, got %v", resultTime)
	}
}

// ============================================================================
// time constants
// ============================================================================

func TestTimeConstants(t *testing.T) {
	tests := []struct {
		name     string
		expected int64
	}{
		{"time.SECOND", 1},
		{"time.MINUTE", 60},
		{"time.HOUR", 3600},
		{"time.DAY", 86400},
		{"time.WEEK", 604800},
	}
	for _, tt := range tests {
		fn := TimeBuiltins[tt.name]
		result := fn.Fn()
		intVal, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("%s: expected Integer, got %T", tt.name, result)
		}
		if intVal.Value.Int64() != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, intVal.Value.Int64())
		}
	}
}

func TestTimeWeekdayConstants(t *testing.T) {
	// Note: These match Go's time.Weekday constants (Sunday=0)
	tests := []struct {
		name     string
		expected int64
	}{
		{"time.SUNDAY", 0},
		{"time.MONDAY", 1},
		{"time.TUESDAY", 2},
		{"time.WEDNESDAY", 3},
		{"time.THURSDAY", 4},
		{"time.FRIDAY", 5},
		{"time.SATURDAY", 6},
	}
	for _, tt := range tests {
		fn := TimeBuiltins[tt.name]
		result := fn.Fn()
		intVal, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("%s: expected Integer, got %T", tt.name, result)
		}
		if intVal.Value.Int64() != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, intVal.Value.Int64())
		}
	}
}

func TestTimeMonthConstants(t *testing.T) {
	tests := []struct {
		name     string
		expected int64
	}{
		{"time.JANUARY", 1},
		{"time.FEBRUARY", 2},
		{"time.MARCH", 3},
		{"time.APRIL", 4},
		{"time.MAY", 5},
		{"time.JUNE", 6},
		{"time.JULY", 7},
		{"time.AUGUST", 8},
		{"time.SEPTEMBER", 9},
		{"time.OCTOBER", 10},
		{"time.NOVEMBER", 11},
		{"time.DECEMBER", 12},
	}
	for _, tt := range tests {
		fn := TimeBuiltins[tt.name]
		result := fn.Fn()
		intVal, ok := result.(*object.Integer)
		if !ok {
			t.Fatalf("%s: expected Integer, got %T", tt.name, result)
		}
		if intVal.Value.Int64() != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, intVal.Value.Int64())
		}
	}
}
