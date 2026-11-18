package datapath

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPortDirsFindsBridgeAndInterfacePorts(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{filepath.Join(root, "bridges", "bridge2"), filepath.Join(root, "vm-1", "interfaces", "net1"), filepath.Join(root, "vm-1", "disks")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	dirs := PortDirs(root)
	if len(dirs) != 2 || !slices.Contains(dirs, filepath.Join(root, "bridges", "bridge2")) || !slices.Contains(dirs, filepath.Join(root, "vm-1", "interfaces", "net1")) {
		t.Fatalf("unexpected port dirs %v", dirs)
	}
}

func TestReadPortNeedsBridgeAndPeer(t *testing.T) {
	dir := t.TempDir()
	if _, ok := ReadPort(dir); ok {
		t.Fatal("a directory without network.json is not a port")
	}

	if err := os.WriteFile(statePath(dir), []byte(`{"bridge":"bridge2","endpoint":"feth0","peer":"feth1"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	port, ok := ReadPort(dir)
	if !ok || port.Bridge != "bridge2" || port.Peer != "feth1" {
		t.Fatalf("read %+v %v", port, ok)
	}

	if err := os.WriteFile(statePath(dir), []byte(`{"bridge":"bridge2"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, ok := ReadPort(dir); ok {
		t.Fatal("a port without a peer must be ignored")
	}
}
