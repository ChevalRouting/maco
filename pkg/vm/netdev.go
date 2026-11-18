package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/m-vinc/maco/pkg/net/datapath"
)

func (d *Driver) bridgePortDir(nic InterfaceSpec, vmRunDir string) string {
	return filepath.Join(d.runDir, "bridges", nic.Bridge)
}

func (d *Driver) StopBridge(bridge string) error {
	return datapath.Stop(filepath.Join(d.runDir, "bridges", bridge))
}

func setBridgeEndpoint(nic *InterfaceSpec, dir string) {
	nic.Socket = datapath.Socket(dir)
}

func bridgeEndpointFile(nic InterfaceSpec) (*os.File, error) {
	return datapath.Connect(nic.Socket)
}

func bridgedNetworkArgs(nic InterfaceSpec) ([]string, error) {
	if nic.Bridge == "" || nic.Socket == "" || nic.FD < 3 {
		return nil, fmt.Errorf("spec %s: bridge network requires a bridge and helper socket", nic.ID)
	}

	return []string{
		"-netdev", "stream,id=net0,server=off,addr.type=fd,addr.str=" + strconv.Itoa(nic.FD),
		"-device", "virtio-net-pci,netdev=net0,csum=off,guest_csum=off,gso=off,guest_tso4=off,guest_tso6=off,guest_ecn=off",
	}, nil
}
