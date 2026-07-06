package db

import (
	"github.com/fizza/fizza/internal/dbutil"
)

func parseTimeAsDBUtil(s string) (dbutil.Time, error) {
	t, err := dbutil.ParseTime(s)
	if err != nil {
		return dbutil.Time{}, err
	}
	return dbutil.Time{Time: t}, nil
}