package vm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func serveMockAgent(t *testing.T, sock string) {
	t.Helper()
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	replies := map[string]string{
		"guest-info":                   `{"return":{"version":"8.2.2"}}`,
		"guest-get-host-name":          `{"return":{"host-name":"testhost"}}`,
		"guest-get-osinfo":             `{"return":{"id":"ubuntu","pretty-name":"Ubuntu 24.04 LTS","kernel-release":"6.8.0-40-generic","machine":"aarch64"}}`,
		"guest-network-get-interfaces": `{"return":[{"name":"lo","ip-addresses":[]},{"name":"enp0s1","hardware-address":"02:aa:bb:cc:dd:ee","ip-addresses":[{"ip-address":"10.0.0.5","ip-address-type":"ipv4","prefix":24}]}]}`,
		"guest-get-fsinfo":             `{"return":[{"name":"vda1","mountpoint":"/","type":"ext4","total-bytes":1000,"used-bytes":400}]}`,
		"guest-fsfreeze-freeze":        `{"return":2}`,
		"guest-fsfreeze-thaw":          `{"return":2}`,
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		defer func() { _ = conn.Close() }()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			var cmd struct {
				Execute   string `json:"execute"`
				Arguments struct {
					ID json.RawMessage `json:"id"`
				} `json:"arguments"`
			}
			if json.Unmarshal(bytes.TrimLeft(line, "\xff"), &cmd) != nil {
				return
			}

			if cmd.Execute == "guest-sync-delimited" {
				_, _ = conn.Write([]byte{0xff})
				_, _ = conn.Write([]byte(`{"return":` + string(cmd.Arguments.ID) + "}\n"))
				continue
			}

			if reply, ok := replies[cmd.Execute]; ok {
				_, _ = conn.Write([]byte(reply + "\n"))
			}
		}
	}()
}

func TestGuestAgentClient(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "qga.sock")
	serveMockAgent(t, sock)

	client, err := dialQGA(sock, 2*time.Second)
	if err != nil {
		t.Fatalf("dial/sync: %v", err)
	}

	defer func() { _ = client.close() }()

	var base struct {
		Version string `json:"version"`
	}
	if err := client.execute("guest-info", nil, &base); err != nil || base.Version != "8.2.2" {
		t.Fatalf("guest-info: %v version=%q", err, base.Version)
	}

	if host := client.hostname(); host != "testhost" {
		t.Fatalf("hostname %q", host)
	}

	os := client.osInfo()
	if os == nil || os.PrettyName != "Ubuntu 24.04 LTS" || os.KernelRelease != "6.8.0-40-generic" || os.Machine != "aarch64" {
		t.Fatalf("osinfo %+v", os)
	}

	ifaces := client.interfaces()
	if len(ifaces) != 2 || ifaces[1].Name != "enp0s1" || len(ifaces[1].IPAddresses) != 1 ||
		ifaces[1].IPAddresses[0].Address != "10.0.0.5" || ifaces[1].IPAddresses[0].Prefix != 24 {
		t.Fatalf("interfaces %+v", ifaces)
	}

	fs := client.filesystems()
	if len(fs) != 1 || fs[0].Mountpoint != "/" || fs[0].Type != "ext4" || fs[0].UsedBytes != 400 || fs[0].TotalBytes != 1000 {
		t.Fatalf("filesystems %+v", fs)
	}

	if err := client.execute("guest-fsfreeze-freeze", nil, nil); err != nil {
		t.Fatalf("freeze: %v", err)
	}

	if err := client.execute("guest-fsfreeze-thaw", nil, nil); err != nil {
		t.Fatalf("thaw: %v", err)
	}
}

func serveFreezeAgent(t *testing.T, sock string, freezeFailures int32) *int32 {
	t.Helper()
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	remaining := freezeFailures
	var thaws int32

	handle := func(conn net.Conn) {
		defer func() { _ = conn.Close() }()
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			var cmd struct {
				Execute   string `json:"execute"`
				Arguments struct {
					ID json.RawMessage `json:"id"`
				} `json:"arguments"`
			}
			if json.Unmarshal(bytes.TrimLeft(line, "\xff"), &cmd) != nil {
				return
			}

			switch cmd.Execute {
			case "guest-sync-delimited":
				_, _ = conn.Write([]byte{0xff})
				_, _ = conn.Write([]byte(`{"return":` + string(cmd.Arguments.ID) + "}\n"))
			case "guest-fsfreeze-freeze":
				if atomic.AddInt32(&remaining, -1) >= 0 {
					_, _ = conn.Write([]byte(`{"error":{"class":"GenericError","desc":"busy"}}` + "\n"))
				} else {
					_, _ = conn.Write([]byte(`{"return":1}` + "\n"))
				}
			case "guest-fsfreeze-thaw":
				atomic.AddInt32(&thaws, 1)
				_, _ = conn.Write([]byte(`{"return":1}` + "\n"))
			}
		}
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go handle(conn)
		}
	}()
	return &thaws
}

func newFreezeTestDriver(t *testing.T) (*Driver, string) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "maco-qga-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	d := NewDriver(dir)
	id := "vm"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	return d, id
}

func TestFreezeGuestRequiresThawOnError(t *testing.T) {
	d, id := newFreezeTestDriver(t)
	serveFreezeAgent(t, d.qgaPath(id), 1)

	froze, err := d.freezeGuest(id)
	if err == nil {
		t.Fatal("expected freeze error")
	}

	if !froze {
		t.Fatal("freeze error must still report froze=true so the caller thaws the guest")
	}
}

func TestFreezeGuestNoAgent(t *testing.T) {
	d, id := newFreezeTestDriver(t)
	froze, err := d.freezeGuest(id)
	if err != nil {
		t.Fatalf("no-agent freeze should be a no-op: %v", err)
	}

	if froze {
		t.Fatal("without an agent no freeze happens and no thaw is required")
	}
}

func TestThawGuestRetriesUntilAgentRecovers(t *testing.T) {
	d, id := newFreezeTestDriver(t)
	sock := d.qgaPath(id)

	go func() {
		time.Sleep(1200 * time.Millisecond)
		serveFreezeAgent(t, sock, 0)
	}()

	if err := d.thawGuest(id); err != nil {
		t.Fatalf("thaw should succeed once the agent comes back: %v", err)
	}
}
