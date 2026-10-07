//go:build darwin

package vm

import (
	"fmt"

	"github.com/m-vinc/maco/pkg/net/datapath"
)

func setBridgeEndpoint(nic *InterfaceSpec, dir string) {
	nic.Socket = datapath.Socket(dir)
}

func bridgedNetworkArgs(nic InterfaceSpec) ([]string, error) {
	if nic.Bridge == "" || nic.Socket == "" {
		return nil, fmt.Errorf("spec %s: bridge network requires a bridge and helper socket", nic.ID)
	}

	return []string{
		"-netdev", "stream,id=net0,server=off,addr.type=unix,addr.path=" + nic.Socket,
		"-device", "virtio-net-pci,netdev=net0,csum=off,guest_csum=off,gso=off,guest_tso4=off,guest_tso6=off,guest_ecn=off",
	}, nil
}
