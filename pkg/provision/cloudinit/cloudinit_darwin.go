//go:build darwin

package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
)

func makeHybridISO(isoPath, srcDir string) error {
	hdiutil, err := exec.LookPath("hdiutil")
	if err != nil {
		return fmt.Errorf("hdiutil not found: %w", err)
	}

	_ = os.Remove(isoPath)

	args := []string{
		"makehybrid",
		"-o", isoPath,
		"-iso", "-joliet",
		"-default-volume-name", "CIDATA",
		srcDir,
	}

	out, err := exec.Command(hdiutil, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hdiutil makehybrid: %w: %s", err, out)
	}

	return nil
}
