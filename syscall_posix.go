// +build !windows
package main

import (
	"syscall"
)

func statfs(path string) int64 {
	var statfs syscall.Statfs_t
	err := syscall.Statfs(path, &statfs)
	if err != nil {
		return 4096
	}
	return int64(statfs.Bsize)
}
