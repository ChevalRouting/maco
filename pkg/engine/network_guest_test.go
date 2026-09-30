package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGuestNetworkHotplug(t *testing.T) {
	base := os.Getenv("MACO_TEST_NETWORK_GUEST_IMAGE")
	if base == "" {
		t.Skip("set MACO_TEST_NETWORK_GUEST_IMAGE to a cached Ubuntu 24.04 ARM64 qcow2")
	}
	if !filepath.IsAbs(base) {
		t.Fatal("guest image path must be absolute")
	}
	e := testEngine(t)
	t.Cleanup(func() { _ = os.RemoveAll(e.paths.RunDir()) })
	if err := os.Symlink(base, filepath.Join(e.paths.ImagesDir(), "ubuntu-24.04-arm64.qcow2")); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(e.paths.Root, "ssh-key")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("key: %v %s", err, out)
	}
	public, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	m, err := e.CreateVM(CreateVMParams{Name: "hotplug-guest", Image: "ubuntu-24.04-arm64", CPUs: 2, MemoryMiB: 1024, DiskSizeGiB: 4, Network: "user"}, []string{strings.TrimSpace(string(public))})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := e.StartVM(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := e.ForceStopVM(context.Background(), m.ID); err != nil {
			t.Error(err)
		}
	}()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	socket := filepath.Join(e.paths.VMRunDir(m.ID), "qmp.sock")
	result, err := guestTestQMP(socket, "human-monitor-command", map[string]string{"command-line": fmt.Sprintf("hostfwd_add net0 tcp:127.0.0.1:%d-:22", port)})
	if err != nil || string(result) != `""` {
		t.Fatalf("SSH forward: %s %v", result, err)
	}
	ssh := func(script string) ([]byte, error) {
		commandCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return exec.CommandContext(commandCtx, "ssh", "-F", "/dev/null", "-i", key, "-o", "IdentitiesOnly=yes", "-o", "IdentityAgent=none", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile="+filepath.Join(e.paths.Root, "known-hosts"), "-o", "BatchMode=yes", "-o", "ConnectTimeout=2", "-p", strconv.Itoa(port), "maco@127.0.0.1", script).CombinedOutput()
	}
	ready := false
	for deadline := time.Now().Add(90 * time.Second); time.Now().Before(deadline); {
		if _, err := ssh("test -f /var/lib/cloud/instance/boot-finished"); err == nil {
			ready = true
			break
		}
		time.Sleep(time.Second)
	}
	if !ready {
		serial, _ := os.ReadFile(filepath.Join(e.paths.VMRunDir(m.ID), "serial.log"))
		if len(serial) > 6000 {
			serial = serial[len(serial)-6000:]
		}
		t.Fatalf("guest not ready: %s", serial)
	}
	t.Log("guest booted through the production driver")
	if err := e.ManageInterfaceContext(ctx, m.ID, "vm.interface.add", InterfaceParams{Network: "user"}); err != nil {
		t.Fatal(err)
	}
	stored, err := e.vms.Load(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.EffectiveInterfaces()) != 2 {
		t.Fatal("hot-added NIC not persisted")
	}
	nic := stored.EffectiveInterfaces()[1]
	find := fmt.Sprintf(`for p in /sys/class/net/*/address; do if [ "$(cat "$p")" = '%s' ]; then basename "$(dirname "$p")"; fi; done`, nic.MAC)
	check := func(script string) []byte {
		t.Helper()
		output, err := ssh(script)
		if err != nil {
			t.Fatalf("guest check: %v\n%s", err, output)
		}
		return output
	}
	discovered := false
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		out, err := ssh(find)
		if err == nil && strings.TrimSpace(string(out)) != "" {
			discovered = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !discovered {
		t.Fatal("guest did not enumerate hot-added NIC")
	}
	config := fmt.Sprintf("[Match]\nMACAddress=%s\n[Network]\nDHCP=ipv4\n[DHCPv4]\nRouteMetric=200\n", nic.MAC)
	check(fmt.Sprintf("set -e; iface=$(%s); printf '%s' | sudo tee /run/systemd/network/90-hotplug.network >/dev/null; sudo networkctl reload; sudo ip link set \"$iface\" up; sudo networkctl reconfigure \"$iface\"", find, config))
	traffic := false
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		out, err := ssh(fmt.Sprintf("set -e; iface=$(%s); ip -4 addr show dev \"$iface\" | grep 'inet '; ping -I \"$iface\" -c 1 -W 2 10.0.2.2", find))
		if err == nil {
			t.Logf("hot-added NIC traffic:\n%s", out)
			traffic = true
			break
		}
		time.Sleep(time.Second)
	}
	if !traffic {
		t.Fatal("no DHCP/traffic on hot-added NIC")
	}
	isolated, err := e.CreateNetwork(CreateNetworkParams{Name: "isolated-hotplug", Mode: "switch"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ManageInterfaceContext(ctx, m.ID, "vm.interface.update", InterfaceParams{ID: nic.ID, Network: isolated.ID}); err != nil {
		t.Fatal(err)
	}
	stored, err = e.vms.Load(m.ID)
	if err != nil || stored.EffectiveInterfaces()[1].Network != isolated.ID || stored.EffectiveInterfaces()[1].MAC != nic.MAC {
		t.Fatalf("replacement not persisted: %+v %v", stored, err)
	}
	check("test -f /var/lib/cloud/instance/boot-finished")
	t.Log("guest acknowledged replacement while management NIC stayed connected")
	if err := e.ManageInterfaceContext(ctx, m.ID, "vm.interface.remove", InterfaceParams{ID: nic.ID}); err != nil {
		t.Fatal(err)
	}
	check(fmt.Sprintf("test -z \"$(%s)\"", find))
	stored, err = e.vms.Load(m.ID)
	if err != nil || len(stored.EffectiveInterfaces()) != 1 {
		t.Fatalf("removal not persisted: %+v %v", stored, err)
	}
	if err := e.DestroyNetwork(isolated.ID); err != nil {
		t.Fatal(err)
	}
	t.Log("guest acknowledged removal; management NIC and manifest remain consistent")
}

func guestTestQMP(socket, command string, arguments any) (json.RawMessage, error) {
	conn, err := net.DialTimeout("unix", socket, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	decoder, encoder := json.NewDecoder(conn), json.NewEncoder(conn)
	var response map[string]json.RawMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}
	for _, cmd := range []map[string]any{{"execute": "qmp_capabilities"}, {"execute": command, "arguments": arguments}} {
		if err := encoder.Encode(cmd); err != nil {
			return nil, err
		}
		for {
			response = nil
			if err := decoder.Decode(&response); err != nil {
				return nil, err
			}
			if _, ok := response["event"]; ok {
				continue
			}
			if data, ok := response["error"]; ok {
				return nil, fmt.Errorf("QMP: %s", data)
			}
			break
		}
	}
	return response["return"], nil
}
