package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type networkStep struct {
	command string
	reply   string
	check   func(map[string]any) error
}

func networkFixture(t *testing.T, steps []networkStep) (*Driver, string) {
	t.Helper()
	d := NewDriver(networkTestDir(t))
	id := "fixture"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", d.qmpPath(id))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}

		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		_, _ = fmt.Fprintln(conn, `{"QMP":{}}`)
		decoder := json.NewDecoder(conn)
		for _, step := range append([]networkStep{{command: "qmp_capabilities"}}, steps...) {
			var command struct {
				Execute   string
				Arguments map[string]any
			}
			if err := decoder.Decode(&command); err != nil {
				done <- err
				return
			}

			if command.Execute != step.command {
				done <- fmt.Errorf("wanted %s got %s", step.command, command.Execute)
				return
			}

			if step.check != nil {
				if err := step.check(command.Arguments); err != nil {
					done <- err
					return
				}
			}

			if step.reply == "close" {
				done <- nil
				return
			}

			reply := step.reply
			if reply == "" {
				reply = `{"return":{}}`
			}

			if _, err := fmt.Fprintln(conn, reply); err != nil {
				done <- err
				return
			}
		}

		done <- nil
	}()
	t.Cleanup(func() {
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(6 * time.Second):
			t.Error("QMP fixture did not finish")
		}
	})
	return d, id
}

const deletedNIC = `{"event":"DEVICE_DELETED","data":{"device":"net1"}}`
const rejectedNIC = `{"error":{"class":"GenericError","desc":"slot unavailable"}}`

func TestNetworkAttachQMP(t *testing.T) {
	d, id := networkFixture(t, []networkStep{
		{command: "netdev_add", check: func(args map[string]any) error {
			if args["type"] != "user" || args["id"] != "net1" {
				return fmt.Errorf("wrong backend: %v", args)
			}

			return nil
		}},
		{command: "device_add", check: func(args map[string]any) error {
			if args["bus"] != networkBus("net1") || args["netdev"] != "net1" || args["mac"] != InterfaceMAC("fixture", "net1") {
				return fmt.Errorf("wrong NIC: %v", args)
			}

			return nil
		}},
	})
	committed := false
	err := d.ChangeInterface(context.Background(), id, nil, &InterfaceSpec{ID: "net1", Network: NetworkUser}, func() error { committed = true; return nil })
	if err != nil || !committed {
		t.Fatalf("attach: %v committed=%v", err, committed)
	}

	if _, err := os.Stat(d.networkChangePath(id)); !os.IsNotExist(err) {
		t.Fatal("journal survived completed attach")
	}
}

func TestNetworkDetachWaitsForMatchingEvent(t *testing.T) {
	d, id := networkFixture(t, []networkStep{
		{command: "device_del", reply: `{"event":"DEVICE_DELETED","data":{"device":"other"}}` + "\n" + deletedNIC + "\n" + `{"return":{}}`},
		{command: "netdev_del"},
	})
	committed := false
	if err := d.ChangeInterface(context.Background(), id, &InterfaceSpec{ID: "net1", Network: NetworkUser}, nil, func() error { committed = true; return nil }); err != nil || !committed {
		t.Fatalf("detach: %v committed=%v", err, committed)
	}
}

func TestNetworkReplacementRollsBackRejectedDevice(t *testing.T) {
	d, id := networkFixture(t, []networkStep{
		{command: "device_del", reply: `{"return":{}}` + "\n" + deletedNIC},
		{command: "netdev_del"},
		{command: "netdev_add"},
		{command: "device_add", reply: rejectedNIC},
		{command: "netdev_del"},
		{command: "netdev_add", check: func(args map[string]any) error {
			if args["type"] != "user" {
				return fmt.Errorf("wrong rollback backend")
			}

			return nil
		}},
		{command: "device_add"},
	})
	before := &InterfaceSpec{ID: "net1", Network: NetworkUser, Reference: "original"}
	after := &InterfaceSpec{ID: "net1", Network: NetworkSwitch, Group: "239.1.2.3:1234", Reference: "replacement"}
	err := d.ChangeInterface(context.Background(), id, before, after, func() error { t.Error("committed rejected change"); return nil })
	if err == nil || errors.Is(err, ErrNetworkStateUncertain) {
		t.Fatalf("wrong rejection: %v", err)
	}

	if _, err := os.Stat(d.networkChangePath(id)); !os.IsNotExist(err) {
		t.Fatal("journal survived successful rollback")
	}
}

func TestNetworkUnconfirmedDeleteProtectsReferences(t *testing.T) {
	d, id := networkFixture(t, []networkStep{{command: "device_del", reply: "close"}})
	before := &InterfaceSpec{ID: "net1", Network: NetworkBridge, Bridge: "bridge1", Reference: "original"}
	after := &InterfaceSpec{ID: "net1", Network: NetworkUser, Reference: "replacement"}
	stopped := false
	d.stopNetworkHelper = func(string) error { stopped = true; return nil }
	err := d.ChangeInterface(context.Background(), id, before, after, func() error { t.Error("committed unknown state"); return nil })
	if !errors.Is(err, ErrNetworkStateUncertain) || stopped {
		t.Fatalf("err=%v stopped=%v", err, stopped)
	}

	refs, err := d.PendingNetworkReferences(id)
	if err != nil || strings.Join(refs, ",") != "original,replacement" {
		t.Fatalf("unprotected references: %v %v", refs, err)
	}

	if err := d.ChangeInterface(context.Background(), id, before, after, func() error { return nil }); !errors.Is(err, ErrNetworkStateUncertain) {
		t.Fatal("allowed change before recovery")
	}
}

func TestNetworkBridgeRejectedAttachCleansHelper(t *testing.T) {
	d, id := networkFixture(t, []networkStep{{command: "netdev_add"}, {command: "device_add", reply: rejectedNIC}, {command: "netdev_del"}})
	started, stopped := false, false
	d.startNetworkHelper = func(dir, bridge string) error {
		started = bridge == "bridge1" && filepath.Base(dir) == "net1"
		return nil
	}
	d.stopNetworkHelper = func(string) error { stopped = true; return nil }
	err := d.ChangeInterface(context.Background(), id, nil, &InterfaceSpec{ID: "net1", Network: NetworkBridge, Bridge: "bridge1"}, func() error { t.Error("committed rejection"); return nil })
	if err == nil || !started || !stopped || errors.Is(err, ErrNetworkStateUncertain) {
		t.Fatalf("err=%v start=%v stop=%v", err, started, stopped)
	}
}

func TestNetworkSaveFailureUndoesAttachment(t *testing.T) {
	d, id := networkFixture(t, []networkStep{{command: "netdev_add"}, {command: "device_add"}, {command: "device_del", reply: `{"return":{}}` + "\n" + deletedNIC}, {command: "netdev_del"}})
	failure := errors.New("disk full")
	err := d.ChangeInterface(context.Background(), id, nil, &InterfaceSpec{ID: "net1", Network: NetworkUser}, func() error { return failure })
	if !errors.Is(err, failure) || errors.Is(err, ErrNetworkStateUncertain) {
		t.Fatalf("save rollback: %v", err)
	}
}

func TestLiveQEMUNetworkHotAdd(t *testing.T) {
	qemu, err := exec.LookPath(QEMUBinary)
	if err != nil {
		t.Skip("QEMU unavailable")
	}

	d := NewDriver(networkTestDir(t))
	id := "live"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	args := []string{"-machine", "virt", "-accel", "tcg", "-cpu", "cortex-a72", "-m", "128", "-S", "-display", "none", "-nodefaults", "-qmp", "unix:" + d.qmpPath(id) + ",server=on,wait=off"}
	args = append(args, networkPortArgs()...)
	process := exec.Command(qemu, args...)
	log, err := os.Create(filepath.Join(d.vmRunDir(id), "qemu.log"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = log.Close() }()
	process.Stderr = log
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() { _ = process.Process.Kill(); _ = process.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(d.qmpPath(id)); err == nil {
			break
		}

		if time.Now().After(deadline) {
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("QMP not ready: %s", data)
		}

		time.Sleep(10 * time.Millisecond)
	}

	if err := d.ChangeInterface(context.Background(), id, nil, &InterfaceSpec{ID: "net31", Network: NetworkUser}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}

	client, err := dialQMP(d.qmpPath(id), time.Second)
	if err != nil {
		t.Fatal(err)
	}

	defer client.close()
	data, err := client.executeArguments("qom-list", map[string]string{"path": "/machine/peripheral"})
	if err != nil || !strings.Contains(string(data), `"name": "net31"`) && !strings.Contains(string(data), `"name":"net31"`) {
		t.Fatalf("NIC absent: %s %v", data, err)
	}
}

func networkTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/private/tmp", "maco-net-test-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestNetworkDetachRetriesTransientGuestBusy(t *testing.T) {
	d, id := networkFixture(t, []networkStep{
		{command: "device_del", reply: `{"error":{"class":"GenericError","desc":"Hot-unplug failed: guest is busy (power indicator blinking)"}}`},
		{command: "device_del", reply: `{"return":{}}` + "\n" + deletedNIC},
		{command: "netdev_del"},
	})
	if err := d.ChangeInterface(context.Background(), id, &InterfaceSpec{ID: "net1", Network: NetworkUser}, nil, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkStopClearsRecoveryJournal(t *testing.T) {
	d := NewDriver(networkTestDir(t))
	id := "stopped"
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(d.networkChangePath(id), []byte(`{"after":{"Reference":"protected-network"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := d.ForceStop(id); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(d.networkChangePath(id)); !os.IsNotExist(err) {
		t.Fatal("stopped VM retained recovery journal")
	}
}
