package main

import (
	"github.com/google/uuid"
	"testing"

	"github.com/m-vinc/maco/pkg/types"
)

func runCLI(t *testing.T, directory string, args ...string) {
	t.Helper()
	root := newRootCommand()
	root.SetArgs(append([]string{"--data-dir", directory}, args...))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkAndVMAttachment(t *testing.T) {
	directory := t.TempDir()
	runCLI(t, directory, "networks", "list")
	if err := eng().NetStore().Save(&types.NetworkManifest{ID: uuid.NewString(), Name: "lab", Mode: types.NetworkBridge, Address: "192.0.2.254/24", VLANs: []types.VLAN{{Parent: "en10", Tag: 123}}}); err != nil {
		t.Fatal(err)
	}

	n, err := eng().NetStore().Resolve("lab")
	if err != nil || n.Mode != types.NetworkBridge || n.VLANs[0].Tag != 123 {
		t.Fatalf("bridge definition: %+v, %v", n, err)
	}

	runCLI(t, directory, "vm", "new", "guest", "--network", "lab", "--address", "192.0.2.1/24")
	m, err := eng().VMStore().Resolve("guest")
	if err != nil || m.Network != n.ID || m.Addresses[0] != "192.0.2.1/24" {
		t.Fatalf("VM attachment: %+v, %v", m, err)
	}

	runCLI(t, directory, "net", "apply", "lab", "--dry-run")
	n, err = eng().NetStore().Resolve("lab")
	if err != nil || n.Device != "" {
		t.Fatalf("dry-run changed network: %+v, %v", n, err)
	}
}

func TestNetworkModeCLI(t *testing.T) {
	directory := t.TempDir()
	runCLI(t, directory, "networks", "create", "nat", "--mode", "user")
	runCLI(t, directory, "networks", "create", "isolated", "--mode", "switch")
	for name, mode := range map[string]types.NetworkMode{"nat": types.NetworkUser, "isolated": types.NetworkSwitch} {
		n, err := eng().NetStore().Resolve(name)
		if err != nil || n.Mode != mode {
			t.Fatalf("%s: %+v, %v", name, n, err)
		}
	}
}
