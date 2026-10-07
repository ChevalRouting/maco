//go:build linux

package datapath

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	vnl "github.com/vishvananda/netlink"
)

func TapName(runDir string) string {
	sum := sha256.Sum256([]byte(runDir))
	return "mt" + hex.EncodeToString(sum[:5])
}

func Start(runDir, bridge string) error {
	master, err := vnl.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("lookup bridge %s: %w", bridge, err)
	}

	if _, ok := master.(*vnl.Bridge); !ok {
		return fmt.Errorf("%s is not a bridge", bridge)
	}

	name := TapName(runDir)
	tap, err := vnl.LinkByName(name)
	if err != nil {
		tuntap := &vnl.Tuntap{LinkAttrs: vnl.LinkAttrs{Name: name}, Mode: vnl.TUNTAP_MODE_TAP}
		if err := vnl.LinkAdd(tuntap); err != nil {
			return fmt.Errorf("create tap %s: %w", name, err)
		}

		tap, err = vnl.LinkByName(name)
		if err != nil {
			return fmt.Errorf("lookup tap %s: %w", name, err)
		}
	}

	if err := vnl.LinkSetMaster(tap, master); err != nil {
		return fmt.Errorf("enslave tap %s to bridge %s: %w", name, bridge, err)
	}

	if err := vnl.LinkSetUp(tap); err != nil {
		return fmt.Errorf("bring up tap %s: %w", name, err)
	}

	return nil
}

func Stop(runDir string) error {
	tap, err := vnl.LinkByName(TapName(runDir))
	if err != nil {
		return nil
	}

	if err := vnl.LinkDel(tap); err != nil {
		return fmt.Errorf("delete tap %s: %w", tap.Attrs().Name, err)
	}

	return nil
}

func RunWorker(runDir, bridge string, uid, gid int) error {
	return fmt.Errorf("network-port worker is darwin-only; linux configures taps in-process")
}
