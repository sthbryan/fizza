package dbutil

import (
	"database/sql/driver"
	"encoding/json"
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

type Time struct {
	time.Time
}

func Now() Time { return Time{Time: time.Now().UTC()} }

func (t Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.UTC())
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	var tt time.Time
	if err := json.Unmarshal(data, &tt); err != nil {
		return err
	}
	t.Time = tt
	return nil
}

func (t *Time) Scan(value any) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case string:
		parsed, err := ParseTime(v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	case int64:
		t.Time = time.Unix(v, 0).UTC()
		return nil
	case time.Time:
		t.Time = v
		return nil
	}
	return fmt.Errorf("dbutil.Time: cannot scan %T", value)
}

func (t Time) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.UTC().Format(SQLiteTimeLayout), nil
}
