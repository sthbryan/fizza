package dbutil

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Time struct {
	time.Time
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

func (t *Time) Ptr() *time.Time {
	if t == nil || t.Time.IsZero() {
		return nil
	}
	v := t.Time
	return &v
}

func (t *Time) FromPtr(p *time.Time) {
	if p == nil {
		t.Time = time.Time{}
		return
	}
	t.Time = *p
}