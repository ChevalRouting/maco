package image

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

const MaxDiskSizeGiB = 65536

func InspectStandalone(ctx context.Context, path string) (DiskInfo, error) {
	out, err := exec.CommandContext(ctx, "qemu-img", "info", "--output=json", path).CombinedOutput()
	if err != nil {
		return DiskInfo{}, fmt.Errorf("inspect disk: %w: %s", err, out)
	}

	return parseStandalone(out)
}

func parseStandalone(data []byte) (DiskInfo, error) {
	var info DiskInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return info, err
	}

	if info.Format != "qcow2" && info.Format != "raw" {
		return info, fmt.Errorf("only raw and qcow2 images are supported")
	}

	if info.VirtualSize <= 0 || info.VirtualSize > int64(MaxDiskSizeGiB)*(1<<30) {
		return info, fmt.Errorf("image capacity must be between 1 byte and %d GiB", MaxDiskSizeGiB)
	}

	var fields any
	if err := json.Unmarshal(data, &fields); err != nil {
		return info, err
	}

	if hasExternalFile(fields) {
		return info, fmt.Errorf("images with backing files or external data files are not supported")
	}

	return info, nil
}

func hasExternalFile(value any) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "backing-filename" || key == "full-backing-filename" || key == "data-file" {
				if name, ok := child.(string); !ok || name != "" {
					return true
				}
			}

			if hasExternalFile(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if hasExternalFile(child) {
				return true
			}
		}
	}

	return false
}

func ValidateCapacity(sizeGiB int) error {
	if sizeGiB < 1 || sizeGiB > MaxDiskSizeGiB {
		return fmt.Errorf("disk capacity must be between 1 and %d GiB", MaxDiskSizeGiB)
	}

	return nil
}
