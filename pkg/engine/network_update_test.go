package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/m-vinc/maco/pkg/net/vnet"
	"github.com/m-vinc/maco/pkg/types"
)

type preservingApply struct {
	t        *testing.T
	original *types.NetworkManifest
	failure  error
	called   bool
}

func (a *preservingApply) ensure(_ *vnet.Store, updated *types.NetworkManifest, dry bool) error {
	a.called = true
	n := a.original
	if dry || updated.ID != n.ID || updated.Device != n.Device || !updated.Owned || updated.AppliedAddress != n.AppliedAddress || !reflect.DeepEqual(updated.AppliedVLANs, n.AppliedVLANs) || !reflect.DeepEqual(updated.AppliedMembers, n.AppliedMembers) {
		a.t.Fatalf("lost runtime state: %+v", updated)
	}

	return a.failure
}

func TestUpdateNetworkPreservesOwnershipAndApplies(t *testing.T) {
	e := testEngine(t)
	n := &types.NetworkManifest{ID: "de305d54-75b4-431b-adb2-eb6b9e546014", Name: "lan", Mode: types.NetworkBridge, Device: "bridge10", Owned: true, AppliedAddress: "192.168.1.1/24", AppliedMembers: []string{"en0"}, AppliedVLANs: []types.VLAN{{Parent: "en0", Tag: 100, Device: "vlan10"}}}
	if err := e.nets.Save(n); err != nil {
		t.Fatal(err)
	}

	failure := errors.New("host unavailable")
	apply := &preservingApply{t: t, original: n, failure: failure}
	e.ensureNetwork = apply.ensure
	params := CreateNetworkParams{Name: n.Name, Mode: string(n.Mode), Address: "192.168.2.1/24", Members: []string{"en1"}}
	if _, err := e.UpdateNetwork(n.ID, params); !errors.Is(err, failure) {
		t.Fatalf("error: %v", err)
	}

	saved, err := e.nets.Load(n.ID)
	if err != nil || !apply.called || saved.Address != params.Address || !reflect.DeepEqual(saved.Members, params.Members) {
		t.Fatalf("saved: %+v, %v", saved, err)
	}

	for _, invalid := range []CreateNetworkParams{{Name: "renamed", Mode: "bridge"}, {Name: "lan", Mode: "user"}, {Name: "lan", Mode: "bridge", Address: "bad"}, {Name: "lan", Mode: "bridge", Device: "bridge20"}} {
		if _, err := e.UpdateNetwork(n.ID, invalid); err == nil {
			t.Fatalf("accepted: %+v", invalid)
		}

		current, _ := e.nets.Load(n.ID)
		if !reflect.DeepEqual(saved, current) {
			t.Fatal("invalid edit changed manifest")
		}
	}

	e.ensureNetwork = func(_ *vnet.Store, _ *types.NetworkManifest, _ bool) error { return nil }
	if _, err := e.UpdateNetwork(n.ID, params); err != nil {
		t.Fatal(err)
	}
}

func TestColdNetworkChange(t *testing.T) {
	bridge := &types.NetworkManifest{Mode: types.NetworkBridge, Members: []string{"en0"}}
	if coldNetworkChange(bridge, CreateNetworkParams{Members: []string{"en1"}, VLANs: []types.VLAN{{Parent: "en1", Tag: 10}}, Address: "192.0.2.1/24"}) {
		t.Fatal("bridge member change needs stopped VMs")
	}

	bridged := &types.NetworkManifest{Mode: types.NetworkVmnetBridged, Uplink: "en0"}
	if coldNetworkChange(bridged, CreateNetworkParams{Uplink: "en0"}) || !coldNetworkChange(bridged, CreateNetworkParams{Uplink: "en1"}) {
		t.Fatal("uplink change detection")
	}
}
