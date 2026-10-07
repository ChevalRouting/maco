package image

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func CreateVMDisk(ctx context.Context, base, dst string, sizeGiB int) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := ValidateCapacity(sizeGiB); err != nil {
		return err
	}

	qemu, err := exec.LookPath("qemu-img")
	if err != nil {
		return err
	}

	format := "qcow2"
	if base != "" {
		info, err := InspectStandalone(ctx, base)
		if err != nil {
			return err
		}

		format = info.Format

		if int64(sizeGiB)*(1<<30) < info.VirtualSize {
			return fmt.Errorf("disk size is smaller than the image; use at least %d GiB", (info.VirtualSize+(1<<30)-1)>>30)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(dst), ".vm-disk-*.qcow2")
	if err != nil {
		return err
	}

	path := temporary.Name()
	defer func() { _ = os.Remove(path) }()
	if err := temporary.Close(); err != nil {
		return err
	}

	capacity := fmt.Sprintf("%dG", sizeGiB)
	if base == "" {
		out, err := exec.CommandContext(ctx, qemu, "create", "-f", "qcow2", path, capacity).CombinedOutput()
		if err != nil {
			return fmt.Errorf("create disk: %w: %s", err, out)
		}
	} else {
		out, err := exec.CommandContext(ctx, qemu, "convert", "-f", format, "-O", "qcow2", base, path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("copy image: %w: %s", err, out)
		}

		out, err = exec.CommandContext(ctx, qemu, "resize", path, capacity).CombinedOutput()
		if err != nil {
			return fmt.Errorf("expand image: %w: %s", err, out)
		}
	}

	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}

	return os.Rename(path, dst)
}
