package engine

import (
	"golang.org/x/sys/unix"
	"runtime"
)

type HostInfo struct {
	CPUs        int    `json:"cpus"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

func (e *Engine) HostInfo() HostInfo {
	return HostInfo{CPUs: runtime.NumCPU(), MemoryBytes: hostMemoryBytes()}
}

func hostMemoryBytes() uint64 {
	if mem, err := unix.SysctlUint64("hw.memsize"); err == nil {
		return mem
	}

	return 0
}
