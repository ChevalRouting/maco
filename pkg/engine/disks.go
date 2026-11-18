package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

type DiskParams struct {
	ImageID string `json:"image_id,omitempty" binding:"optional"`
	ID      string `json:"id" binding:"optional"`
	Name    string `json:"name" binding:"optional"`
	SizeGiB int    `json:"size_gib" binding:"optional"`
}

func (e *Engine) ManageDisk(ctx context.Context, ref, action string, params DiskParams) error {
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

	if action == "vm.disk.add" {
		return e.addDataDisk(ctx, manifest, params)
	}

	if action == "vm.disk.remove" && params.ID == "disk" {
		return e.removeRootDisk(manifest)
	}

	if action == "vm.disk.replace" {
		return e.replaceBootDisk(ctx, manifest, params)
	}

	if action == "vm.disk.wipe" {
		return e.wipeDisk(ctx, manifest, params.ID)
	}

	for index, disk := range manifest.Disks {
		if disk.ID != params.ID {
			continue
		}

		if _, err := uuid.Parse(disk.ID); err != nil {
			return fmt.Errorf("invalid disk ID: %w", err)
		}

		path := filepath.Join(e.paths.VMDiskDir(manifest.ID), disk.ID+".qcow2")
		switch action {
		case "vm.disk.grow":
			if params.SizeGiB <= disk.SizeGiB {
				return fmt.Errorf("new capacity must be larger than %d GiB", disk.SizeGiB)
			}

			if err := e.resizeDisk(ctx, manifest.ID, path, params.SizeGiB); err != nil {
				return err
			}

			manifest.Disks[index].SizeGiB = params.SizeGiB
			return e.vms.Save(manifest)
		case "vm.disk.remove":
			if e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
				return fmt.Errorf("stop the VM before removing disks")
			}

			removed := path + ".removed"
			if err := os.Rename(path, removed); err != nil {
				return err
			}

			manifest.Disks = append(manifest.Disks[:index], manifest.Disks[index+1:]...)
			order := manifest.BootOrder[:0]
			for _, device := range manifest.BootOrder {
				if device != "disk:"+disk.ID {
					order = append(order, device)
				}
			}

			manifest.BootOrder = order
			if err := e.vms.Save(manifest); err != nil {
				if restoreErr := os.Rename(removed, path); restoreErr != nil {
					return fmt.Errorf("save disk removal: %w; restore disk: %v", err, restoreErr)
				}

				return err
			}

			return os.Remove(removed)
		default:
			return fmt.Errorf("unknown disk action")
		}
	}

	return fmt.Errorf("disk not found")
}

func (e *Engine) wipeDisk(ctx context.Context, manifest *types.VMManifest, id string) error {
	if e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("stop the VM before wiping disks")
	}

	var path string
	var size int
	if id == "disk" {
		if manifest.DiskSizeGiB <= 0 {
			return fmt.Errorf("this VM has no boot disk")
		}

		path = filepath.Join(e.paths.VMDiskDir(manifest.ID), "disk.qcow2")
		size = manifest.DiskSizeGiB
	} else {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("invalid disk ID: %w", err)
		}

		for _, disk := range manifest.Disks {
			if disk.ID == id {
				path = filepath.Join(e.paths.VMDiskDir(manifest.ID), disk.ID+".qcow2")
				size = disk.SizeGiB
				break
			}
		}

		if path == "" {
			return fmt.Errorf("disk not found")
		}
	}

	tmp := path + ".wipe"
	_ = os.Remove(tmp)
	if err := image.CreateVMDisk(ctx, "", tmp, size); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("create blank disk: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}

func (e *Engine) removeRootDisk(manifest *types.VMManifest) error {
	if manifest.DiskSizeGiB <= 0 {
		return fmt.Errorf("this VM has no boot disk")
	}

	if e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("stop the VM before removing disks")
	}

	path := filepath.Join(e.paths.VMDiskDir(manifest.ID), "disk.qcow2")
	removed := path + ".removed"
	renamed := false
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, removed); err != nil {
			return err
		}

		renamed = true
	} else if !os.IsNotExist(err) {
		return err
	}

	previousSize := manifest.DiskSizeGiB
	previousOrder := manifest.BootOrder
	manifest.DiskSizeGiB = 0
	order := make([]string, 0, len(previousOrder))
	for _, device := range previousOrder {
		if device != "disk" {
			order = append(order, device)
		}
	}

	manifest.BootOrder = order
	if err := e.vms.Save(manifest); err != nil {
		manifest.DiskSizeGiB = previousSize
		manifest.BootOrder = previousOrder
		if renamed {
			if restoreErr := os.Rename(removed, path); restoreErr != nil {
				return fmt.Errorf("save disk removal: %w; restore disk: %v", err, restoreErr)
			}
		}

		return err
	}

	if renamed {
		return os.Remove(removed)
	}

	return nil
}

func (e *Engine) addDataDisk(ctx context.Context, manifest *types.VMManifest, params DiskParams) error {
	if params.SizeGiB < 1 || strings.TrimSpace(params.Name) == "" {
		return fmt.Errorf("disk name and capacity of at least 1 GiB are required")
	}

	disk := types.VMDisk{ID: uuid.NewString(), Name: strings.TrimSpace(params.Name), SizeGiB: params.SizeGiB}
	path := filepath.Join(e.paths.VMDiskDir(manifest.ID), disk.ID+".qcow2")

	base := ""
	if params.ImageID != "" {
		mediaLock, err := e.lockMedia(ctx)
		if err != nil {
			return err
		}

		defer func() { _ = mediaLock.Close() }()

		media, err := e.GetMedia(params.ImageID)
		if err != nil {
			return err
		}

		if media.Kind != "image" || params.SizeGiB < media.SizeGiB {
			return fmt.Errorf("select a disk image and sufficient capacity")
		}

		base = media.Path
	}

	if err := image.CreateVMDisk(ctx, base, path, disk.SizeGiB); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("create data disk: %w", err)
	}

	manifest.Disks = append(manifest.Disks, disk)
	if err := e.vms.Save(manifest); err != nil {
		_ = os.Remove(path)
		return err
	}

	if e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
		if err := e.driver.AttachDisk(manifest.ID, disk.ID, path); err != nil {
			manifest.Disks = manifest.Disks[:len(manifest.Disks)-1]
			if saveErr := e.vms.Save(manifest); saveErr != nil {
				return fmt.Errorf("attach disk: %w; restore manifest: %v", err, saveErr)
			}

			if !errors.Is(err, vm.ErrDiskBackendOpen) {
				_ = os.Remove(path)
			}

			return err
		}
	}

	return nil
}

func (e *Engine) replaceBootDisk(ctx context.Context, manifest *types.VMManifest, params DiskParams) error {
	if params.ID != "disk" || manifest.DiskSizeGiB <= 0 {
		return fmt.Errorf("only an existing boot disk can be replaced")
	}

	if e.driver.Status(manifest.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("stop the VM before replacing its boot disk")
	}

	mediaLock, err := e.lockMedia(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = mediaLock.Close() }()
	ref := params.ImageID
	if _, err := uuid.Parse(ref); err == nil {
		ref = "media:" + ref
	}

	base, err := e.imageBase(ctx, ref, true)
	if err != nil {
		return err
	}

	info, err := image.InspectStandalone(ctx, base)
	if err != nil {
		return err
	}

	size := max(manifest.DiskSizeGiB, int((info.VirtualSize+(1<<30)-1)>>30))
	path := filepath.Join(e.paths.VMDiskDir(manifest.ID), "disk.qcow2")
	tmp := path + ".replace"
	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return err
	}

	defer func() { _ = os.Remove(tmp) }()
	if err := image.CreateVMDisk(ctx, base, tmp, size); err != nil {
		return err
	}

	backup := path + ".previous"
	if _, err := os.Stat(backup); err == nil {
		return fmt.Errorf("previous boot disk recovery file exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(path, backup); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err != nil {
		if restoreErr := os.Rename(backup, path); restoreErr != nil {
			return fmt.Errorf("replace disk: %w; restore disk: %v", err, restoreErr)
		}

		return err
	}

	manifest.Image = ref
	manifest.DiskSizeGiB = size
	order := []string{"disk"}
	for _, device := range manifest.BootOrder {
		if device != "disk" {
			order = append(order, device)
		}
	}

	manifest.BootOrder = order
	if err := e.vms.Save(manifest); err != nil {
		if restoreErr := os.Rename(backup, path); restoreErr != nil {
			return fmt.Errorf("save image selection: %w; restore disk: %v", err, restoreErr)
		}

		return err
	}

	return os.Remove(backup)
}
