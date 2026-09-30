package vnet

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

func TestBridgeManifest(t *testing.T) {
	store := NewStore(t.TempDir())
	n := &types.NetworkManifest{ID: uuid.NewString(), Name: "lab", Mode: types.NetworkBridge, Members: []string{"vlan9"}, VLANs: []types.VLAN{{Parent: "en10", Tag: 123}}}
	if err := store.Save(n); err != nil {
		t.Fatal(err)
	}

	resolved, err := store.Resolve("lab")
	if err != nil || resolved.ID != n.ID || resolved.VLANs[0].Tag != 123 {
		t.Fatalf("resolve failed: %+v, %v", resolved, err)
	}

	info, err := os.Stat(store.path(n.ID))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("manifest permissions: %v, %v", info, err)
	}
}

func TestInvalidNetworks(t *testing.T) {
	cases := []*types.NetworkManifest{
		{Mode: "bogus"},
		{Mode: types.NetworkSwitch},
		{Mode: types.NetworkBridged},
		{Mode: types.NetworkBridge, VLANs: []types.VLAN{{Parent: "en10", Tag: 4095}}},
		{Mode: types.NetworkBridge, VLANs: []types.VLAN{{Parent: "en10", Tag: 123}, {Parent: "en10", Tag: 123}}},
		{Mode: types.NetworkBridge, Members: []string{"-a"}},
		{Mode: types.NetworkSwitch, Group: "239.1.2.3:1234", Address: "192.0.2.1/24"},
		{Mode: types.NetworkBridge, Address: "fd00::1/64"},
	}
	store := NewStore(t.TempDir())
	for _, n := range cases {
		n.ID, n.Name = uuid.NewString(), "lab"
		if err := store.Save(n); err == nil {
			t.Fatalf("invalid network accepted: %+v", n)
		}
	}
}
