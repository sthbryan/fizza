//go:build unix

package db

import "golang.org/x/sys/unix"

type unixStat_t = unix.Statfs_t

var unixStatfs = unix.Statfs

func diskFree(path string) (uint64, error) {
	var stat unixStat_t
	if err := unixStatfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
