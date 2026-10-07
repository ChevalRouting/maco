package engine

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

var interfaceID = regexp.MustCompile(`^net([0-9]|[1-2][0-9]|3[0-1])$`)

type InterfaceParams struct {
	ID      string `json:"id" binding:"optional" extensions:"x-maco-description=VM interface ID such as net0. Discover existing IDs with getVM."`
	Network string `json:"network" binding:"optional" extensions:"x-maco-description=Network ID from listNetworks or user for NAT."`
	MAC     string `json:"mac" binding:"optional" extensions:"x-maco-description=Optional guest MAC address. Use suggestMAC for a generated suggestion or omit for a stable generated MAC."`
}

func exposeInterfaces(m *types.VMManifest) {
	interfaces := append([]types.VMInterface{}, m.EffectiveInterfaces()...)
	for i := range interfaces {
		if interfaces[i].MAC == "" {
			interfaces[i].MAC = vm.InterfaceMAC(m.ID, interfaces[i].ID)
		}

		if interfaces[i].Network == "" {
			interfaces[i].Network = "user"
		}
	}

	m.Interfaces = &interfaces
}

func (e *Engine) normalizedInterfaces(m *types.VMManifest) ([]types.VMInterface, error) {
	interfaces := append([]types.VMInterface{}, m.EffectiveInterfaces()...)
	if len(interfaces) > 32 {
		return nil, fmt.Errorf("at most 32 interfaces are supported")
	}

	ids, macs := map[string]bool{}, map[string]bool{}
	for i := range interfaces {
		nic := &interfaces[i]
		if !interfaceID.MatchString(nic.ID) || ids[nic.ID] {
			return nil, fmt.Errorf("invalid or duplicate interface ID")
		}

		ids[nic.ID] = true
		if nic.MAC == "" {
			nic.MAC = vm.InterfaceMAC(m.ID, nic.ID)
		}

		mac, err := net.ParseMAC(nic.MAC)
		if err != nil || len(mac) != 6 || mac[0]&1 != 0 {
			return nil, fmt.Errorf("interface MAC must be a unicast Ethernet address")
		}

		nic.MAC = mac.String()
		if macs[nic.MAC] {
			return nil, fmt.Errorf("duplicate interface MAC")
		}

		macs[nic.MAC] = true
		if nic.Network == "" {
			nic.Network = "user"
		}

		if nic.Network != "user" {
			n, err := e.nets.Resolve(nic.Network)
			if err != nil {
				return nil, err
			}

			if n.Mode == types.NetworkVLAN {
				return nil, fmt.Errorf("connect to a bridge containing the VLAN")
			}

			nic.Network = n.ID
		}
	}

	return interfaces, nil
}

func (e *Engine) ManageInterfaceContext(ctx context.Context, ref, action string, p InterfaceParams) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	m, err = e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	if e.driver.Status(m.ID).Phase == vm.PhaseRunning {
		refs, err := e.driver.PendingNetworkReferences(m.ID)
		if err != nil {
			return err
		}

		if len(refs) != 0 {
			return vm.ErrNetworkStateUncertain
		}
	}

	previous := append([]types.VMInterface{}, m.EffectiveInterfaces()...)
	for i := range previous {
		if previous[i].MAC == "" {
			previous[i].MAC = vm.InterfaceMAC(m.ID, previous[i].ID)
		}

		if previous[i].Network == "" {
			previous[i].Network = "user"
		}
	}

	interfaces := append([]types.VMInterface{}, previous...)
	changedID := p.ID
	switch action {
	case "vm.interface.add":
		used := map[string]bool{}
		for _, nic := range interfaces {
			used[nic.ID] = true
		}

		id := ""
		for i := 0; i < 32; i++ {
			candidate := "net" + strconv.Itoa(i)
			if !used[candidate] {
				id = candidate
				break
			}
		}

		if id == "" {
			return fmt.Errorf("at most 32 interfaces are supported")
		}

		changedID = id
		interfaces = append(interfaces, types.VMInterface{ID: id, Network: p.Network, MAC: p.MAC})
	case "vm.interface.update", "vm.interface.remove":
		found := false
		for i := range interfaces {
			if interfaces[i].ID == p.ID {
				found = true
				if action == "vm.interface.remove" {
					interfaces = append(interfaces[:i], interfaces[i+1:]...)
				} else {
					interfaces[i].Network = p.Network
					if p.MAC != "" {
						interfaces[i].MAC = p.MAC
					}
				}

				break
			}
		}

		if !found {
			return fmt.Errorf("interface not found")
		}
	default:
		return fmt.Errorf("unknown interface action")
	}

	m.Interfaces = &interfaces
	normalized, err := e.normalizedInterfaces(m)
	if err != nil {
		return err
	}

	m.Interfaces = &normalized
	m.Network = ""
	m.Addresses = nil
	if len(normalized) > 0 {
		m.Network = normalized[0].Network
		m.Addresses = normalized[0].Addresses
	}

	if e.driver.Status(m.ID).Phase != vm.PhaseRunning {
		return e.vms.Save(m)
	}

	resolve := func(list []types.VMInterface) (*vm.InterfaceSpec, error) {
		for _, nic := range list {
			if nic.ID != changedID {
				continue
			}

			network, err := e.resolveNetwork(nic.Network)
			if err != nil {
				return nil, err
			}

			network.ID = nic.ID
			network.MAC = nic.MAC
			network.Reference = nic.Network
			return &network, nil
		}

		return nil, nil
	}
	before, err := resolve(previous)
	if err != nil {
		return err
	}

	after, err := resolve(normalized)
	if err != nil {
		return err
	}

	networkLock, err := e.nets.LockContext(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = networkLock.Close() }()
	for _, nic := range []*vm.InterfaceSpec{before, after} {
		if nic == nil {
			continue
		}

		if nic.Reference == "user" {
			continue
		}

		if _, err := e.nets.Resolve(nic.Reference); err != nil {
			return fmt.Errorf("network changed before hotplug: %w", err)
		}
	}

	if before != nil && after != nil && *before == *after {
		return e.vms.Save(m)
	}

	return e.driver.ChangeInterface(ctx, m.ID, before, after, func() error { return e.vms.Save(m) })
}

func guestInterfacesConfig(interfaces []types.VMInterface) string {
	config := "version: 2\nrenderer: networkd\nethernets:\n"
	for _, nic := range interfaces {
		part := guestNetworkConfig(nic.MAC, nic.Addresses, nic.Nameservers, nic.Gateway)
		part = strings.TrimPrefix(part, "version: 2\nrenderer: networkd\nethernets:\n")
		part = strings.Replace(part, "  lab:\n", "  "+nic.ID+":\n", 1)
		part = strings.Replace(part, "set-name: lab0", "set-name: lab"+strings.TrimPrefix(nic.ID, "net"), 1)
		config += part
	}

	if len(interfaces) == 0 {
		return "version: 2\nrenderer: networkd\nethernets: {}\n"
	}

	return config
}
