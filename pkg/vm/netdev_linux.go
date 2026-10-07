//go:build linux

package vm

import (
	"fmt"

	"github.com/m-vinc/maco/pkg/net/datapath"
)

func setBridgeEndpoint(nic *InterfaceSpec, dir string) {
	nic.Tap = datapath.TapName(dir)
}

func bridgedNetworkArgs(nic InterfaceSpec) ([]string, error) {
	if nic.Bridge == "" || nic.Tap == "" {
		return nil, fmt.Errorf("spec %s: bridge network requires a bridge and tap device", nic.ID)
	}

	return []string{
		"-netdev", "tap,id=net0,ifname=" + nic.Tap + ",script=no,downscript=no",
		"-device", "virtio-net-pci,netdev=net0",
	}, nil
}
