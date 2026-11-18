package ignition

import (
	"os"
	"path/filepath"

	"github.com/m-vinc/maco/pkg/provision"
)

type Provisioner struct{}

func (Provisioner) Name() string { return "ignition" }

func (Provisioner) Render(p provision.Params) ([]provision.File, error) {
	doc, err := Config{
		Hostname:    p.Hostname,
		MAC:         p.MAC,
		Addresses:   p.Addresses,
		Gateway:     p.Gateway,
		Nameservers: p.Nameservers,
	}.easyButane()
	if err != nil {
		return nil, err
	}

	return []provision.File{{Name: "config.bu", Content: string(doc)}}, nil
}

func (Provisioner) Deliver(dir string, files []provision.File) (provision.Delivery, error) {
	butane := ""
	for _, file := range files {
		if file.Name == "config.bu" {
			butane = file.Content
		}
	}

	ign, err := transpile([]byte(butane))
	if err != nil {
		return provision.Delivery{}, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return provision.Delivery{}, err
	}

	ignPath := filepath.Join(dir, "config.ign")
	if err := os.WriteFile(ignPath, ign, 0o600); err != nil {
		return provision.Delivery{}, err
	}

	return provision.Delivery{IgnitionPath: ignPath}, nil
}
