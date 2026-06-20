package dbutil

func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func IsDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func NullableInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}