package vm

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestStatusAdoptsProcessFromQMPSocket(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "maco-adopt-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	d := NewDriver(dir)
	id := "vm"
	if got := d.Status(id); got.Phase != PhaseStopped {
		t.Fatalf("status without process = %+v", got)
	}

	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", filepath.Join(d.vmRunDir(id), "qmp.sock"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	got := d.Status(id)
	if got.Phase != PhaseRunning || got.PID != os.Getpid() {
		t.Fatalf("status = %+v, want running as %d", got, os.Getpid())
	}

	if pid, err := d.readPID(id); err != nil || pid != os.Getpid() {
		t.Fatalf("pid file = %d, %v", pid, err)
	}
}
