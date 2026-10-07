//go:build linux

package firmware

import (
	"context"
	"fmt"
	"os"
)

var ovmfCandidates = []string{
	"/usr/share/ovmf/OVMF.fd",
	"/usr/share/OVMF/OVMF.fd",
	"/usr/share/edk2/ovmf/OVMF.fd",
	"/usr/share/edk2-ovmf/x64/OVMF.fd",
	"/usr/share/qemu/OVMF.fd",
}

func EnsureContext(_ context.Context, _ string) (string, error) {
	if env := os.Getenv("MACO_OVMF"); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", fmt.Errorf("MACO_OVMF=%s: %w", env, err)
		}

		return env, nil
	}

	for _, path := range ovmfCandidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("OVMF firmware not found; install the OVMF/edk2 package or set MACO_OVMF (searched %v)", ovmfCandidates)
}
