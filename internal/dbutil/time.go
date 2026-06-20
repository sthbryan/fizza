package dbutil

import (
	"fmt"
	"time"
)

const SQLiteTimeLayout = "2006-01-02T15:04:05.000000Z07:00"

var timeLayouts = []string{
	SQLiteTimeLayout,
	"2006-01-02T15:04:05.000Z07:00",
	time.RFC3339Nano,
	time.RFC3339,
}

func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable timestamp %q", s)
}

func ParseDueDate(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable due_date %q (use YYYY-MM-DD or ISO 8601)", s)
}

func FormatDueDate(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z07:00")
}