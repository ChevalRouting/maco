//go:build linux

package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
)

func makeHybridISO(isoPath, srcDir string) error {
	_ = os.Remove(isoPath)

	if tool, err := exec.LookPath("xorriso"); err == nil {
		return runISO(tool, "-as", "mkisofs", "-output", isoPath, "-volid", "CIDATA", "-joliet", "-rock", srcDir)
	}

	for _, name := range []string{"genisoimage", "mkisofs"} {
		if tool, err := exec.LookPath(name); err == nil {
			return runISO(tool, "-output", isoPath, "-volid", "CIDATA", "-joliet", "-rock", srcDir)
		}
	}

	return fmt.Errorf("no ISO tool found; install xorriso or genisoimage to build cloud-init seeds")
}

func runISO(tool string, args ...string) error {
	if out, err := exec.Command(tool, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w: %s", tool, err, out)
	}

	return nil
}
