package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/vm"
)

type UpdateHardwareParams struct {
	Autostart   *bool     `json:"autostart,omitempty" binding:"optional" extensions:"x-maco-description=Whether host reconciliation starts this VM automatically. Omit to preserve."`
	CPUs        *int      `json:"cpus,omitempty" binding:"optional" extensions:"x-maco-description=New CPU count. VM must be stopped. Omit to preserve current setting." minimum:"1"`
	MemoryMiB   *int      `json:"memory_mib,omitempty" binding:"optional" extensions:"x-maco-description=New memory in MiB. VM must be stopped. Omit to preserve current setting." minimum:"64"`
	DiskSizeGiB *int      `json:"disk_size_gib,omitempty" binding:"optional" extensions:"x-maco-description=New primary disk capacity in GiB. Shrinking is not supported." minimum:"1"`
	Tags        *[]string `json:"tags,omitempty" binding:"optional" extensions:"x-maco-description=Replacement set of freeform labels for grouping and filtering. Omit to preserve; pass an empty list to clear."`
}

func (e *Engine) UpdateHardware(ctx context.Context, ref string, params UpdateHardwareParams) error {
	manifest, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, manifest.ID)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	manifest, err = e.vms.Load(manifest.ID)
	if err != nil {
		return err
	}

	if (params.CPUs != nil || params.MemoryMiB != nil) && e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("stop the VM before changing CPU or memory")
	}

	if params.CPUs != nil {
		if *params.CPUs < 1 {
			return fmt.Errorf("CPUs must be at least 1")
		}

		manifest.CPUs = *params.CPUs
	}

	if params.MemoryMiB != nil {
		if *params.MemoryMiB < 64 {
			return fmt.Errorf("memory must be at least 64 MiB")
		}

		manifest.MemoryMiB = *params.MemoryMiB
	}

	if params.DiskSizeGiB != nil {
		if *params.DiskSizeGiB < manifest.DiskSizeGiB {
			return fmt.Errorf("disk shrinking is not supported")
		}

		path := filepath.Join(e.paths.VMDiskDir(manifest.ID), "disk.qcow2")
		if _, err := os.Stat(path); err == nil {
			if err := e.resizeDisk(ctx, manifest.ID, path, *params.DiskSizeGiB); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		manifest.DiskSizeGiB = *params.DiskSizeGiB
	}

	if params.Autostart != nil {
		manifest.Autostart = *params.Autostart
	}

	if params.Tags != nil {
		manifest.Tags = normalizeTags(*params.Tags)
	}

	return e.vms.Save(manifest)
}

func resizeVMDisk(ctx context.Context, path string, size int) error {
	output, err := exec.CommandContext(ctx, "qemu-img", "info", "--output=json", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspect disk: %w: %s", err, output)
	}

	var info image.DiskInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return err
	}

	if int64(size)*(1<<30) < info.VirtualSize {
		return fmt.Errorf("disk shrinking is not supported")
	}

	output, err = exec.CommandContext(ctx, "qemu-img", "resize", path, fmt.Sprintf("%dG", size)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("resize disk: %w: %s", err, output)
	}

	return nil
}

func (e *Engine) resizeDisk(ctx context.Context, id, path string, size int) error {
	if e.driver.Status(id).Phase == vm.PhaseRunning {
		return e.driver.GrowDisk(id, path, size)
	}

	return resizeVMDisk(ctx, path, size)
}
