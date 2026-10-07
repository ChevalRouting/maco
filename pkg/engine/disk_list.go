package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m-vinc/maco/pkg/types"
)

type DiskView struct {
	BootFirst   bool   `json:"boot_first"`
	DiskID      string `json:"disk_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	VMID        string `json:"vm_id"`
	VMName      string `json:"vm_name"`
	Path        string `json:"path"`
	SizeBytes   int64  `json:"size_bytes"`
	CapacityGiB int    `json:"capacity_gib"`
	Orphaned    bool   `json:"orphaned"`
}

func (e *Engine) ListDisks() ([]DiskView, error) {
	manifests, err := e.vms.List()
	if err != nil {
		return nil, err
	}

	owners := make(map[string]*types.VMManifest, len(manifests))
	for _, m := range manifests {
		owners[m.ID] = m
	}

	root := e.paths.DisksDir()
	dirs, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []DiskView{}, nil
		}

		return nil, err
	}

	views := make([]DiskView, 0)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		vmID := dir.Name()
		files, err := os.ReadDir(filepath.Join(root, vmID))
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".qcow2") {
				continue
			}

			info, err := file.Info()
			if err != nil {
				return nil, err
			}

			path := filepath.Join(root, vmID, file.Name())
			views = append(views, describeDisk(owners[vmID], vmID, file.Name(), path, info.Size()))
		}
	}

	sort.Slice(views, func(a, b int) bool {
		if views[a].Orphaned != views[b].Orphaned {
			return views[a].Orphaned
		}

		if views[a].VMName != views[b].VMName {
			return views[a].VMName < views[b].VMName
		}

		return views[a].Name < views[b].Name
	})

	return views, nil
}

func describeDisk(owner *types.VMManifest, vmID, file, path string, size int64) DiskView {
	view := DiskView{
		DiskID:    strings.TrimSuffix(file, ".qcow2"),
		Path:      path,
		SizeBytes: size,
		VMID:      vmID,
		Kind:      "data",
	}

	if file == "disk.qcow2" {
		view.Kind = "boot"
	}

	if owner == nil {
		view.Orphaned = true
		view.Name = file
		return view
	}

	view.VMName = owner.Name
	device := "disk:" + view.DiskID
	if file == "disk.qcow2" {
		device = "disk"
	}

	order := owner.EffectiveBootOrder()
	view.BootFirst = len(order) > 0 && order[0] == device
	if view.Kind == "boot" {
		view.Name = "Disk 1"
		view.CapacityGiB = owner.DiskSizeGiB
		return view
	}

	for _, disk := range owner.Disks {
		if disk.ID == view.DiskID {
			view.Name = disk.Name
			view.CapacityGiB = disk.SizeGiB
			return view
		}
	}

	view.Name = file
	view.Orphaned = true
	return view
}
