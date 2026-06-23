package models

import "time"

// TimeFormat is the canonical timestamp format stored in the database and sent
// over the API: RFC3339 in UTC with fixed millisecond precision. Fixed-width
// fractional seconds make the strings sort lexically the same as chronologically,
// which the sync protocol relies on for "changed since" comparisons.
const TimeFormat = "2006-01-02T15:04:05.000Z07:00"

// DateFormat is the calendar-date format used for a net's date.
const DateFormat = "2006-01-02"

// FormatTime renders t in the canonical UTC format.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeFormat)
}

// ParseTime parses a canonical timestamp string.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}

// Now returns the current time in the canonical format.
func Now() string {
	return FormatTime(time.Now())
}
