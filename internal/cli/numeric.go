package cli

import "fmt"

func parseInt64(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not numeric: %q", s)
		}
		n = n*10 + int64(r-'0')
		if n > 1<<62 {
			return 0, fmt.Errorf("too large: %q", s)
		}
	}
	return n, nil
}