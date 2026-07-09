//go:build windows

package db

import "errors"

func diskFree(path string) (uint64, error) {
	return 0, errors.ErrUnsupported
}
