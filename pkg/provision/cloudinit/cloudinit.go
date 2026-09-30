package cloudinit

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
)

type Seed struct {
	Hostname string
}

func (s Seed) metaData() string {
	data, _ := yaml.Marshal(struct {
		InstanceID string `yaml:"instance-id"`
		Hostname   string `yaml:"local-hostname"`
	}{s.Hostname, s.Hostname})
	return string(data)
}

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
