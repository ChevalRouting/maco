//go:build linux

package engine

import (
	"golang.org/x/sys/unix"
)

func hostMemoryBytes() uint64 {
	var si unix.Sysinfo_t
	if err := unix.Sysinfo(&si); err == nil {
		return uint64(si.Totalram) * uint64(si.Unit)
	}

	return 0
}
