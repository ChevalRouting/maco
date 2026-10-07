package engine

import (
	"errors"
	"github.com/m-vinc/maco/pkg/net/vnet"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
	"testing"
)

func TestCreateNetworkModes(t *testing.T) {
	e := testEngine(t)
	e.ensureNetwork = func(_ *vnet.Store, _ *types.NetworkManifest, _ bool) error { return nil }
	for _, tc := range []struct {
		params CreateNetworkParams
		want   vm.NetworkMode
	}{
		{CreateNetworkParams{Name: "nat", Mode: "user"}, vm.NetworkUser},
		{CreateNetworkParams{Name: "physical-new", Mode: "vmnet-bridged", Uplink: "en10"}, vm.NetworkVmnetBridged},
		{CreateNetworkParams{Name: "physical-old", Mode: "bridged", Uplink: "en10"}, vm.NetworkVmnetBridged},
	} {
		n, err := e.CreateNetwork(tc.params)
		if err != nil {
			t.Fatal(err)
		}

		for _, ref := range []string{n.Name, n.ID} {
			spec, err := e.resolveNetwork(ref)
			if err != nil || spec.Network != tc.want || spec.Uplink != tc.params.Uplink {
				t.Fatalf("resolve %s: %+v, %v", ref, spec, err)
			}
		}

		if err := e.DestroyNetwork(n.ID); err != nil {
			t.Fatal(err)
		}
	}

	n, err := e.CreateNetwork(CreateNetworkParams{Name: "access", VLANs: []types.VLAN{{Parent: "en10", Tag: 123}}})
	if err != nil || n.Mode != types.NetworkBridge || len(n.VLANs) != 1 {
		t.Fatalf("bridge VLAN: %+v, %v", n, err)
	}

	if err := e.ApplyNetwork(n.ID, true); err != nil {
		t.Fatal(err)
	}

	for _, params := range []CreateNetworkParams{
		{Name: "invalid", Mode: "user", Uplink: "en10"},
		{Name: "invalid", Mode: "user", Parent: "en10", Tag: 123},
		{Name: "invalid", Mode: "user", Members: []string{"en10"}},
		{Name: "invalid", Mode: "vmnet-bridged"},
		{Name: "invalid", Mode: "vmnet-bridged", Uplink: "en0,foo=bar"},
	} {
		if _, err := e.CreateNetwork(params); err == nil {
			t.Fatalf("accepted invalid network: %+v", params)
		}
	}
}

func TestCreateNetworkAppliesAndRollsBack(t *testing.T) {
	e := testEngine(t)
	failure := errors.New("apply failed")
	called := false
	e.ensureNetwork = func(store *vnet.Store, n *types.NetworkManifest, dryRun bool) error {
		called = true
		if dryRun {
			t.Fatal("creation must apply")
		}

		if _, err := store.Load(n.ID); err != nil {
			t.Fatal(err)
		}

		return failure
	}
	if _, err := e.CreateNetwork(CreateNetworkParams{Name: "broken", Mode: "bridge"}); !errors.Is(err, failure) {
		t.Fatalf("creation error: %v", err)
	}

	if !called {
		t.Fatal("apply never ran")
	}

	if _, err := e.nets.Resolve("broken"); !errors.Is(err, vnet.ErrNotFound) {
		t.Fatalf("failed manifest retained: %v", err)
	}

	e.ensureNetwork = vnet.Ensure
	if _, err := e.CreateNetwork(CreateNetworkParams{Name: "missing", VLANs: []types.VLAN{{Parent: "missing999", Tag: 123}}}); err == nil {
		t.Fatal("missing parent succeeded")
	}

	if _, err := e.nets.Resolve("missing"); !errors.Is(err, vnet.ErrNotFound) {
		t.Fatalf("failed manifest retained: %v", err)
	}
}
