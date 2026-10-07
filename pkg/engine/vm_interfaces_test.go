package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

func TestVMInterfaceLifecycle(t *testing.T) {
	e := testEngine(t)
	m, err := e.CreateVM(CreateVMParams{Name: "interfaces", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	n := &types.NetworkManifest{ID: uuid.NewString(), Name: "lab", Mode: types.NetworkSwitch, Group: "239.1.2.3:1234"}
	if err := e.nets.Save(n); err != nil {
		t.Fatal(err)
	}

	if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.add", InterfaceParams{Network: n.Name}); err != nil {
		t.Fatal(err)
	}

	stored, err := e.vms.Resolve(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	interfaces := stored.EffectiveInterfaces()
	if len(interfaces) != 2 || interfaces[0].ID != "net0" || interfaces[0].MAC != vm.MAC(m.ID) || interfaces[1].ID != "net1" || interfaces[1].Network != n.ID || interfaces[0].MAC == interfaces[1].MAC {
		t.Fatalf("invalid interfaces: %+v", interfaces)
	}

	run := e.paths.VMRunDir(m.ID)
	if err := os.MkdirAll(run, 0o700); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(e.paths.RunDir()) })
	pidPath := filepath.Join(run, "qemu.pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := e.DestroyNetwork(n.ID); err == nil {
		t.Fatal("destroyed network attached to second interface of running VM")
	}

	if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.remove", InterfaceParams{ID: "net1"}); err == nil {
		t.Fatal("changed running VM interface without a QMP connection")
	}

	_ = os.Remove(pidPath)
	originalMAC := interfaces[1].MAC
	if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.update", InterfaceParams{ID: "net1", Network: "user"}); err != nil {
		t.Fatal(err)
	}

	stored, _ = e.vms.Resolve(m.ID)
	if stored.EffectiveInterfaces()[1].MAC != originalMAC {
		t.Fatal("network change altered MAC")
	}

	for _, params := range []InterfaceParams{{Network: "missing"}, {Network: "user", MAC: "not-a-mac"}, {Network: "user", MAC: interfaces[0].MAC}, {Network: "user", MAC: "01:00:00:00:00:01"}} {
		if e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.add", params) == nil {
			t.Fatal("accepted invalid interface")
		}
	}

	for _, id := range []string{"net0", "net1"} {
		if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.remove", InterfaceParams{ID: id}); err != nil {
			t.Fatal(err)
		}
	}

	stored, err = e.vms.Resolve(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Interfaces == nil || len(stored.EffectiveInterfaces()) != 0 {
		t.Fatal("removed interfaces resurrected legacy default")
	}

	if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.add", InterfaceParams{Network: "user"}); err != nil {
		t.Fatal(err)
	}

	stored, _ = e.vms.Resolve(m.ID)
	if stored.EffectiveInterfaces()[0].ID != "net0" {
		t.Fatal("interface ID not allocated")
	}
}

func TestGuestInterfacesConfig(t *testing.T) {
	interfaces := []types.VMInterface{{ID: "net0", MAC: "02:00:00:00:00:01", Addresses: []string{"192.0.2.2/24"}}, {ID: "net1", MAC: "02:00:00:00:00:02"}}
	config := guestInterfacesConfig(interfaces)
	for _, expected := range []string{"net0:", "net1:", "set-name: lab0", "set-name: lab1", interfaces[0].MAC, interfaces[1].MAC, "dhcp4: true", "dhcp4: false", "192.0.2.2/24"} {
		if !strings.Contains(config, expected) {
			t.Fatalf("missing %s in %s", expected, config)
		}
	}

	if !strings.Contains(guestInterfacesConfig(nil), "ethernets: {}") {
		t.Fatal("empty network config invalid")
	}
}

func TestPendingHotplugProtectsBothNetworks(t *testing.T) {
	e := testEngine(t)
	m, err := e.CreateVM(CreateVMParams{Name: "pending-network", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	before := &types.NetworkManifest{ID: uuid.NewString(), Name: "before", Mode: types.NetworkSwitch, Group: "239.1.2.3:1234"}
	after := &types.NetworkManifest{ID: uuid.NewString(), Name: "after", Mode: types.NetworkSwitch, Group: "239.1.2.4:1234"}
	for _, n := range []*types.NetworkManifest{before, after} {
		if err := e.nets.Save(n); err != nil {
			t.Fatal(err)
		}
	}

	run := e.paths.VMRunDir(m.ID)
	if err := os.MkdirAll(run, 0o700); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(e.paths.RunDir()) })
	if err := os.WriteFile(filepath.Join(run, "qemu.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	data := fmt.Sprintf(`{"before":{"Reference":%q},"after":{"Reference":%q}}`, before.ID, after.ID)
	if err := os.WriteFile(filepath.Join(run, "network-change.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, n := range []*types.NetworkManifest{before, after} {
		if err := e.DestroyNetwork(n.ID); err == nil {
			t.Fatalf("destroyed pending network %s", n.Name)
		}
	}

	if err := e.ManageInterfaceContext(context.Background(), m.ID, "vm.interface.update", InterfaceParams{ID: "net0", Network: "user"}); !errors.Is(err, vm.ErrNetworkStateUncertain) {
		t.Fatalf("allowed no-op while NIC state uncertain: %v", err)
	}
}
