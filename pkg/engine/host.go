package engine

import (
	"runtime"
)

type HostInfo struct {
	CPUs        int    `json:"cpus"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

func (e *Engine) HostInfo() HostInfo {
	return HostInfo{CPUs: runtime.NumCPU(), MemoryBytes: hostMemoryBytes()}
}
