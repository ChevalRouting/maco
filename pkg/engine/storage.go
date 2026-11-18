package engine

import "syscall"

type StorageStats struct {
	Path       string `json:"path"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
	DisksBytes int64  `json:"disks_bytes"`
	DiskCount  int    `json:"disk_count"`
}

func (e *Engine) StorageStats() (*StorageStats, error) {
	path := e.paths.DisksDir()

	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return nil, err
	}

	block := uint64(fs.Bsize)
	total := fs.Blocks * block
	free := fs.Bavail * block
	used := total - fs.Bfree*block

	disks, err := e.ListDisks()
	if err != nil {
		return nil, err
	}

	var diskBytes int64
	for _, disk := range disks {
		diskBytes += disk.SizeBytes
	}

	return &StorageStats{
		Path:       path,
		TotalBytes: total,
		FreeBytes:  free,
		UsedBytes:  used,
		DisksBytes: diskBytes,
		DiskCount:  len(disks),
	}, nil
}
