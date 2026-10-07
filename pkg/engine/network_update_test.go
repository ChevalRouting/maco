package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/m-vinc/maco/pkg/net/vnet"
	"github.com/m-vinc/maco/pkg/types"
)

func TestUpdateNetworkPreservesOwnershipAndApplies(t *testing.T) {
	e := testEngine(t)
	n := &types.NetworkManifest{ID: "de305d54-75b4-431b-adb2-eb6b9e546014", Name: "lan", Mode: types.NetworkBridge, Device: "bridge10", Owned: true, AppliedAddress: "192.168.1.1/24", AppliedMembers: []string{"en0"}, AppliedVLANs: []types.VLAN{{Parent: "en0", Tag: 100, Device: "vlan10"}}}
	if err := e.nets.Save(n); err != nil {
		t.Fatal(err)
	}

	called := false
	failure := errors.New("host unavailable")
	e.ensureNetwork = func(store *vnet.Store, updated *types.NetworkManifest, dry bool) error {
		called = true
		if dry || updated.ID != n.ID || updated.Device != n.Device || !updated.Owned || updated.AppliedAddress != n.AppliedAddress || !reflect.DeepEqual(updated.AppliedVLANs, n.AppliedVLANs) || !reflect.DeepEqual(updated.AppliedMembers, n.AppliedMembers) {
			t.Fatalf("lost runtime state: %+v", updated)
		}

		return failure
	}
	params := CreateNetworkParams{Name: n.Name, Mode: string(n.Mode), Address: "192.168.2.1/24", Members: []string{"en1"}}
	if _, err := e.UpdateNetwork(n.ID, params); !errors.Is(err, failure) {
		t.Fatalf("error: %v", err)
	}

	saved, err := e.nets.Load(n.ID)
	if err != nil || !called || saved.Address != params.Address || !reflect.DeepEqual(saved.Members, params.Members) {
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
