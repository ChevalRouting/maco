package cloudinit

import (
	"os"
	"path/filepath"

	"github.com/m-vinc/maco/pkg/provision"
)

type Provisioner struct{}

func (Provisioner) Name() string { return "cloud-init" }

func (Provisioner) Render(p provision.Params) ([]provision.File, error) {
	seed := Seed{Hostname: p.Hostname}
	files := []provision.File{
		{Name: "meta-data", Content: seed.metaData()},
	}
	if p.NetworkConfig != "" {
		files = append(files, provision.File{Name: "network-config", Content: p.NetworkConfig})
	}

	return files, nil
}

func (Provisioner) Deliver(dir string, files []provision.File) (provision.Delivery, error) {
	isoPath := filepath.Join(dir, "seed.iso")
	if err := BuildSeedFiles(isoPath, files); err != nil {
		return provision.Delivery{}, err
	}

	return provision.Delivery{SeedPath: isoPath}, nil
}

func BuildSeedFiles(isoPath string, files []provision.File) error {
	if err := os.MkdirAll(filepath.Dir(isoPath), 0o700); err != nil {
		return err
	}

	staging, err := os.MkdirTemp("", "maco-cidata-")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(staging) }()

	for _, file := range files {
		if err := os.WriteFile(filepath.Join(staging, file.Name), []byte(file.Content), 0o600); err != nil {
			return err
		}
	}

	return makeHybridISO(isoPath, staging)
}
