package dyn

import (
	"fmt"
	"time"
)

// Time represents a time-like primitive value.
//
// It represents a timestamp and includes the original string value
// that was parsed to create the timestamp. This makes it possible
// to coalesce a value that YAML interprets as a timestamp back into
// a string without losing information.
type Time struct {
	t time.Time
	s string
}

// NewTime creates a new Time from the given string.
func NewTime(str string) (Time, error) {
	// Try a couple of layouts
	for _, layout := range []string{
		"2006-1-2T15:4:5.999999999Z07:00", // RCF3339Nano with short date fields.
		"2006-1-2t15:4:5.999999999Z07:00", // RFC3339Nano with short date fields and lower-case "t".
		"2006-1-2 15:4:5.999999999",       // space separated with no time zone
		"2006-1-2",                        // date only
	} {
		t, terr := time.Parse(layout, str)
		if terr == nil {
			return Time{t: t, s: str}, nil
		}
	}

	return Time{}, fmt.Errorf("invalid time value: %q", str)
}

// MustTime creates a new Time from the given string.
// It panics if the string cannot be parsed.
func MustTime(str string) Time {
	t, err := NewTime(str)
	if err != nil {
		panic(err)
	}
	return t
}

// FromTime creates a new Time from the given time.Time.
// It uses the RFC3339Nano format for its string representation.
// This guarantees that it can roundtrip into a string without losing information.
func FromTime(t time.Time) Time {
	return Time{t: t, s: t.Format(time.RFC3339Nano)}
}

// Time returns the time.Time value.
func (t Time) Time() time.Time {
	return t.t
}

// String returns the original string value that was parsed to create the timestamp.
func (t Time) String() string {
	return t.s
}
