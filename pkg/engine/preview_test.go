package engine

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestInitialPreviewCapture(t *testing.T) {
	e := testEngine(t)
	m, err := e.CreateVM(CreateVMParams{Name: "preview", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	dir := e.paths.VMRunDir(m.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(e.paths.RunDir()) })
	if err := os.WriteFile(filepath.Join(dir, "qemu.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", filepath.Join(dir, "qmp.sock"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = listener.Close() }()
	result := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			result <- err
			return
		}

		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		decoder, encoder := json.NewDecoder(conn), json.NewEncoder(conn)
		if err := encoder.Encode(map[string]any{"QMP": map[string]any{}}); err != nil {
			result <- err
			return
		}

		for i := 0; i < 2; i++ {
			var command struct {
				Execute   string            `json:"execute"`
				Arguments map[string]string `json:"arguments"`
			}
			if err := decoder.Decode(&command); err != nil {
				result <- err
				return
			}

			if command.Execute == "screendump" {
				if err := os.WriteFile(command.Arguments["filename"], []byte("captured"), 0o600); err != nil {
					result <- err
					return
				}
			}

			if err := encoder.Encode(map[string]any{"return": map[string]any{}}); err != nil {
				result <- err
				return
			}
		}

		result <- nil
	}()
	start := time.Now()
	e.captureInitialPreview(context.Background(), m.ID)
	if time.Since(start) < time.Second {
		t.Fatal("captured before one second")
	}

	if err := <-result; err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "preview.png"))
	if err != nil || string(data) != "captured" {
		t.Fatalf("preview not published: %s %v", data, err)
	}
}

func TestInitialPreviewCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := testEngine(t)
	start := time.Now()
	e.captureInitialPreview(ctx, "missing")
	if time.Since(start) > time.Second {
		t.Fatal("cancelled capture waited")
	}
}
