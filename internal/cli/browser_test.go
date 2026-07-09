package cli

import "testing"

func TestBrowserURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"127.0.0.1:6500", "http://127.0.0.1:6500"},
		{"0.0.0.0:9090", "http://127.0.0.1:9090"},
		{"[::]:6500", "http://127.0.0.1:6500"},
		{"localhost:6500", "http://localhost:6500"},
	}
	for _, tt := range tests {
		if got := browserURL(tt.in); got != tt.want {
			t.Errorf("browserURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
