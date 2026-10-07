package vm

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestGuestShutdownOnlyRequestsPowerdown(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "maco-power-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	d := NewDriver(dir)
	id := "guest"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := d.writePID(id, os.Getpid()); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", d.qmpPath(id))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	commands := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			commands <- err.Error()
			return
		}

		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		encoder, decoder := json.NewEncoder(conn), json.NewDecoder(conn)
		_ = encoder.Encode(map[string]any{"QMP": map[string]any{}})
		var command qmpCommand
		if err := decoder.Decode(&command); err != nil {
			commands <- err.Error()
			return
		}

		_ = encoder.Encode(map[string]any{"return": map[string]any{}})
		if err := decoder.Decode(&command); err != nil {
			commands <- err.Error()
			return
		}

		commands <- command.Execute
		_ = encoder.Encode(map[string]any{"return": map[string]any{}})
	}()
	if err := d.Shutdown(id); err != nil {
		t.Fatal(err)
	}

	if command := <-commands; command != "system_powerdown" {
		t.Fatalf("unexpected command %s", command)
	}

	if d.Status(id).Phase != PhaseRunning {
		t.Fatal("guest shutdown changed running state before guest exited")
	}
}

func TestForceStopDoesNotRequestGuestShutdown(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "maco-power-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	d := NewDriver(dir)
	id := "force"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	defer func() { _ = cmd.Process.Kill(); <-done }()
	if err := d.writePID(id, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", filepath.Join(d.vmRunDir(id), "qmp.sock"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	_ = listener.(*net.UnixListener).SetDeadline(time.Now().Add(2 * time.Second))
	connected := make(chan bool, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}

		connected <- err == nil
	}()
	if err := d.ForceStop(id); err != nil {
		t.Fatal(err)
	}

	_ = listener.Close()
	if <-connected {
		t.Fatal("force stop contacted QMP for guest shutdown")
	}

	if d.Status(id).Phase != PhaseStopped {
		t.Fatal("VM still running after stop")
	}
}
