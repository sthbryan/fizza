package db

import (
	"errors"

	sqlite "modernc.org/sqlite"
	lib "modernc.org/sqlite/lib"
)

var (
	ErrNotFound  = errors.New("db: not found")
	ErrDuplicate = errors.New("db: duplicate")
)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var sqlErr *sqlite.Error
	if errors.As(err, &sqlErr) {
		return sqlErr.Code() == lib.SQLITE_CONSTRAINT_UNIQUE
	}
	return false
}

func IsNotFound(err error) bool  { return errors.Is(err, ErrNotFound) }
func IsDuplicate(err error) bool { return errors.Is(err, ErrDuplicate) }