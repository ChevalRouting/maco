package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/m-vinc/maco/pkg/net/datapath"
)

var ErrNetworkStateUncertain = errors.New("network change needs recovery; stop/start the VM to reconcile")
var validNetworkID = regexp.MustCompile(`^net([0-9]|[1-2][0-9]|3[0-1])$`)

type networkChange struct {
	Before *InterfaceSpec `json:"before,omitempty" binding:"optional"`
	After  *InterfaceSpec `json:"after,omitempty" binding:"optional"`
}

func (d *Driver) networkChangePath(id string) string {
	return filepath.Join(d.vmRunDir(id), "network-change.json")
}

func (d *Driver) PendingNetworkReferences(id string) ([]string, error) {
	data, err := os.ReadFile(d.networkChangePath(id))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var change networkChange
	if err := json.Unmarshal(data, &change); err != nil {
		return nil, err
	}
	refs := []string{}
	for _, nic := range []*InterfaceSpec{change.Before, change.After} {
		if nic != nil && nic.Reference != "" {
			refs = append(refs, nic.Reference)
		}
	}
	return refs, nil
}

func networkBackend(nic InterfaceSpec) (map[string]any, error) {
	if !validNetworkID.MatchString(nic.ID) {
		return nil, fmt.Errorf("invalid network interface ID")
	}
	backend := map[string]any{"id": nic.ID}
	switch nic.Network {
	case NetworkUser:
		backend["type"] = "user"
	case NetworkSwitch:
		if nic.Group == "" {
			return nil, fmt.Errorf("switch requires a multicast group")
		}
		backend["type"], backend["mcast"] = "socket", nic.Group
	case NetworkBridge:
		if nic.Bridge == "" || nic.Socket == "" {
			return nil, fmt.Errorf("bridge requires a bridge and helper socket")
		}
		backend["type"], backend["server"] = "stream", false
		backend["addr"] = map[string]string{"type": "unix", "path": nic.Socket}
	case NetworkVmnetBridged:
		if nic.Uplink == "" {
			return nil, fmt.Errorf("vmnet-bridged requires an uplink")
		}
		backend["type"], backend["ifname"] = "vmnet-bridged", nic.Uplink
	default:
		return nil, fmt.Errorf("unknown network mode %q", nic.Network)
	}
	return backend, nil
}

func networkDevice(nic InterfaceSpec) map[string]any {
	device := map[string]any{"driver": "virtio-net-pci", "id": nic.ID, "netdev": nic.ID, "mac": nic.MAC, "bus": networkBus(nic.ID), "addr": "0", "disable-legacy": "on"}
	if nic.Network == NetworkBridge {
		for _, property := range []string{"csum", "guest_csum", "gso", "guest_tso4", "guest_tso6", "guest_ecn"} {
			device[property] = false
		}
	}
	return device
}

func uncertain(err error) error  { return fmt.Errorf("%w: %v", ErrNetworkStateUncertain, err) }
func qmpRejected(err error) bool { var reply *qmpError; return errors.As(err, &reply) }

func (d *Driver) ChangeInterface(ctx context.Context, id string, before, after *InterfaceSpec, commit func() error) error {
	if _, err := os.Stat(d.networkChangePath(id)); err == nil {
		return ErrNetworkStateUncertain
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, nic := range []*InterfaceSpec{before, after} {
		if nic != nil && !validNetworkID.MatchString(nic.ID) {
			return fmt.Errorf("invalid interface ID")
		}
	}
	client, err := dialQMP(d.qmpPath(id), 3*time.Second)
	if err != nil {
		return err
	}
	defer client.close()
	cancelIO := context.AfterFunc(ctx, func() { _ = client.conn.Close() })
	defer cancelIO()
	prepare := func(nic *InterfaceSpec) *InterfaceSpec {
		if nic == nil {
			return nil
		}
		copy := *nic
		if copy.MAC == "" {
			copy.MAC = InterfaceMAC(id, copy.ID)
		}
		if copy.Network == NetworkBridge {
			copy.Socket = datapath.Socket(filepath.Join(d.vmRunDir(id), "interfaces", copy.ID))
		}
		return &copy
	}
	before, after = prepare(before), prepare(after)
	if after != nil {
		if _, err := networkBackend(*after); err != nil {
			return err
		}
	}
	data, err := json.Marshal(networkChange{before, after})
	if err != nil {
		return err
	}
	if err := d.writeNetworkChange(id, data); err != nil {
		return err
	}
	clear := func(err error) error {
		if errors.Is(err, ErrNetworkStateUncertain) {
			return err
		}
		if cleanupErr := os.Remove(d.networkChangePath(id)); cleanupErr != nil {
			return uncertain(fmt.Errorf("%v; clear journal: %w", err, cleanupErr))
		}
		return err
	}
	if before != nil {
		if err := d.detachInterface(client, id, *before); err != nil {
			return clear(err)
		}
	}
	rollback := func(cause error) error {
		if before != nil {
			if err := d.attachInterface(client, id, *before); err != nil {
				return uncertain(fmt.Errorf("%v; restore original interface: %w", cause, err))
			}
		}
		return clear(cause)
	}
	if after != nil {
		if err := d.attachInterface(client, id, *after); err != nil {
			if errors.Is(err, ErrNetworkStateUncertain) {
				return err
			}
			return rollback(err)
		}
	}
	if err := commit(); err != nil {
		if after != nil {
			if cleanupErr := d.detachInterface(client, id, *after); cleanupErr != nil {
				return uncertain(fmt.Errorf("save manifest: %v; undo attachment: %w", err, cleanupErr))
			}
		}
		return rollback(err)
	}
	return clear(nil)
}

func (d *Driver) attachInterface(client *qmpClient, id string, nic InterfaceSpec) error {
	backend, err := networkBackend(nic)
	if err != nil {
		return err
	}
	dir := filepath.Join(d.vmRunDir(id), "interfaces", nic.ID)
	stop := func() error {
		if nic.Network == NetworkBridge {
			return d.stopInterfaceHelper(dir)
		}
		return nil
	}
	if nic.Network == NetworkBridge {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := d.startInterfaceHelper(dir, nic.Bridge); err != nil {
			if cleanupErr := stop(); cleanupErr != nil {
				return uncertain(fmt.Errorf("helper start: %v; cleanup: %w", err, cleanupErr))
			}
			return err
		}
	}
	_ = client.conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := client.executeArguments("netdev_add", backend); err != nil {
		if !qmpRejected(err) {
			return uncertain(err)
		}
		if cleanupErr := stop(); cleanupErr != nil {
			return uncertain(cleanupErr)
		}
		return fmt.Errorf("create network backend: %w", err)
	}
	if _, err := client.executeArguments("device_add", networkDevice(nic)); err != nil {
		if !qmpRejected(err) {
			return uncertain(err)
		}
		if _, cleanupErr := client.executeArguments("netdev_del", map[string]string{"id": nic.ID}); cleanupErr != nil {
			return uncertain(fmt.Errorf("attach NIC: %v; delete backend: %w", err, cleanupErr))
		}
		if cleanupErr := stop(); cleanupErr != nil {
			return uncertain(cleanupErr)
		}
		return fmt.Errorf("attach NIC (older VMs need a stop/start to reserve PCIe slots): %w", err)
	}
	return nil
}

func (d *Driver) detachInterface(client *qmpClient, id string, nic InterfaceSpec) error {
	_ = client.conn.SetDeadline(time.Now().Add(15 * time.Second))
	client.events = nil
	retryDeadline := time.Now().Add(5 * time.Second)
	for {
		_, err := client.executeArguments("device_del", map[string]string{"id": nic.ID})
		if err == nil {
			break
		}
		if !qmpRejected(err) {
			return uncertain(err)
		}
		var reply *qmpError
		if !errors.As(err, &reply) || !strings.Contains(reply.Desc, "guest is busy (power indicator blinking)") || time.Now().After(retryDeadline) {
			return fmt.Errorf("remove NIC (older VMs need a stop/start to reserve PCIe slots): %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	for {
		var event qmpMessage
		if len(client.events) > 0 {
			event, client.events = client.events[0], client.events[1:]
		} else if err := client.dec.Decode(&event); err != nil {
			return uncertain(fmt.Errorf("NIC removal not confirmed: %w", err))
		}
		var data qmpDeviceEvent
		if json.Unmarshal(event.Data, &data) != nil || data.Device != nic.ID {
			continue
		}
		if event.Event == "DEVICE_UNPLUG_GUEST_ERROR" {
			return fmt.Errorf("guest rejected NIC removal")
		}
		if event.Event == "DEVICE_DELETED" {
			break
		}
	}
	if _, err := client.executeArguments("netdev_del", map[string]string{"id": nic.ID}); err != nil {
		return uncertain(err)
	}
	if nic.Network == NetworkBridge {
		if err := d.stopInterfaceHelper(filepath.Join(d.vmRunDir(id), "interfaces", nic.ID)); err != nil {
			return uncertain(err)
		}
	}
	return nil
}

func (d *Driver) startInterfaceHelper(dir, bridge string) error {
	if d.startNetworkHelper != nil {
		return d.startNetworkHelper(dir, bridge)
	}
	return datapath.Start(dir, bridge)
}
func (d *Driver) stopInterfaceHelper(dir string) error {
	if d.stopNetworkHelper != nil {
		return d.stopNetworkHelper(dir)
	}
	return datapath.Stop(dir)
}

func (d *Driver) writeNetworkChange(id string, data []byte) error {
	file, err := os.CreateTemp(d.vmRunDir(id), ".network-change-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), d.networkChangePath(id))
}
