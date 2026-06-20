package dbutil

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"2026-06-19T10:30:00.000000Z", false},
		{"2026-06-19T10:30:00.000Z", false},
		{"2026-06-19T10:30:00Z", false},
		{"2026-06-19T10:30:00-03:00", false},
		{"", true},
		{"garbage", true},
	}
	for _, tt := range tests {
		_, err := ParseTime(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTime(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
		}
	}
}

func TestParseDueDate(t *testing.T) {
	cases := map[string]bool{
		"2026-07-01":               false,
		"2026-07-01T12:00:00Z":     false,
		"2026-07-01T12:00:00-03:00": false,
		"01/07/2026":               true,
		"":                         true,
	}
	for in, wantErr := range cases {
		_, err := ParseDueDate(in)
		if (err != nil) != wantErr {
			t.Errorf("ParseDueDate(%q) err=%v wantErr=%v", in, err, wantErr)
		}
	}
}

func TestFormatDueDate_RoundTrip(t *testing.T) {
	in := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	s := FormatDueDate(in)
	out, err := ParseDueDate(s)
	if err != nil {
		t.Fatalf("ParseDueDate: %v", err)
	}
	if !out.Equal(in) {
		t.Errorf("round-trip mismatch: in=%v out=%v", in, out)
	}
}

func TestBoolToInt(t *testing.T) {
	if BoolToInt(true) != 1 {
		t.Error("BoolToInt(true) should be 1")
	}
	if BoolToInt(false) != 0 {
		t.Error("BoolToInt(false) should be 0")
	}
}

func TestIsDigits(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"123":   true,
		"0":     true,
		"12a":   false,
		" 123 ": false,
		"-1":    false,
	}
	for in, want := range cases {
		if got := IsDigits(in); got != want {
			t.Errorf("IsDigits(%q)=%v want %v", in, got, want)
		}
	}
}

func TestNullableInt(t *testing.T) {
	if NullableInt(nil) != nil {
		t.Error("NullableInt(nil) should be nil")
	}
	n := int64(42)
	if got := NullableInt(&n); got != int64(42) {
		t.Errorf("NullableInt(&42)=%v", got)
	}
}