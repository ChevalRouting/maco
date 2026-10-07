//go:build darwin

package engine

import (
	"golang.org/x/sys/unix"
)

func hostMemoryBytes() uint64 {
	if mem, err := unix.SysctlUint64("hw.memsize"); err == nil {
		return mem
	}

	return 0
}
