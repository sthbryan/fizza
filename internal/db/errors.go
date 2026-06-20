package db

import (
	"errors"
	"strings"
)

var (
	ErrNotFound  = errors.New("db: not found")
	ErrDuplicate = errors.New("db: duplicate")
)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func IsNotFound(err error) bool  { return errors.Is(err, ErrNotFound) }
func IsDuplicate(err error) bool { return errors.Is(err, ErrDuplicate) }