package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	return New(paths)
}

func TestResolveNetwork(t *testing.T) {
	e := testEngine(t)
	for _, n := range []*types.NetworkManifest{
		{ID: uuid.NewString(), Name: "switch-lab", Mode: types.NetworkSwitch, Group: "239.1.2.3:1234"},
		{ID: uuid.NewString(), Name: "physical", Mode: types.NetworkBridged, Uplink: "en10"},
	} {
		if err := e.nets.Save(n); err != nil {
			t.Fatal(err)
		}

		spec, err := e.resolveNetwork(n.Name)
		if err != nil || spec.Group != n.Group || spec.Uplink != n.Uplink {
			t.Fatalf("named network was not resolved: %+v, %v", spec, err)
		}
	}

	if _, err := e.resolveNetwork("missing"); err == nil {
		t.Fatal("unknown named network accepted")
	}
}

func TestGuestNetworkConfig(t *testing.T) {
	config := guestNetworkConfig(vm.MAC("guest"), []string{"192.0.2.1/24", "fd00::1/64"}, []string{"1.1.1.1"}, "192.0.2.254")
	if !strings.Contains(config, "macaddress: "+vm.MAC("guest")) || !strings.Contains(config, "fd00::1/64") || !strings.Contains(config, "dhcp4: false") {
		t.Fatal(config)
	}
	if !strings.Contains(config, "via: 192.0.2.254") || !strings.Contains(config, "- 1.1.1.1") {
		t.Fatal(config)
	}
}

func TestBootReconcileEmpty(t *testing.T) {
	e := testEngine(t)
	if err := e.BootReconcile(context.Background()); err != nil {
		t.Fatalf("boot reconcile on an empty host: %v", err)
	}
}

func TestAutostartTargets(t *testing.T) {
	e := testEngine(t)
	auto := &types.VMManifest{ID: uuid.NewString(), Name: "auto", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10, Autostart: true}
	manual := &types.VMManifest{ID: uuid.NewString(), Name: "manual", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10}
	if err := e.vms.Save(auto); err != nil {
		t.Fatal(err)
	}
	if err := e.vms.Save(manual); err != nil {
		t.Fatal(err)
	}
	targets, err := e.AutostartTargets()
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != auto.ID {
		t.Fatalf("expected only the autostart VM, got %v", targets)
	}
}
