package vnet

import (
	"fmt"
	"slices"

	"github.com/m-vinc/maco/pkg/net/host"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/rs/zerolog/log"
)

func Reconcile(store *Store, dryRun bool) error {
	networks, err := store.List()
	if err != nil {
		return err
	}

	for _, n := range networks {
		if err := Ensure(store, n, dryRun); err != nil {
			return fmt.Errorf("network %s: %w", n.Name, err)
		}
	}

	return nil
}

func Ensure(store *Store, n *types.NetworkManifest, dryRun bool) error {
	if n.Mode == types.NetworkVLAN {
		return ensureVLAN(store, n, dryRun)
	}

	if n.Mode == types.NetworkBridged || n.Mode == types.NetworkVmnetBridged {
		if !host.Exists(n.Uplink) {
			return fmt.Errorf("physical uplink %s is missing", n.Uplink)
		}
		return nil
	}
	if n.Mode != types.NetworkBridge {
		return nil
	}

	lock, err := store.Lock()
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()

	current, err := store.Load(n.ID)
	if err != nil {
		return err
	}

	*n = *current
	if n.Mode != types.NetworkBridge {
		return fmt.Errorf("network mode changed during apply; retry")
	}

	if dryRun {
		log.Info().Str("network", n.Name).Str("device", n.Device).Strs("members", n.Members).Int("vlans", len(n.VLANs)).Msg("would reconcile native bridge")
		return nil
	}

	for _, member := range n.Members {
		if !host.Exists(member) {
			return fmt.Errorf("bridge member %s is missing", member)
		}
	}
	for _, vlan := range n.VLANs {
		if !host.Exists(vlan.Parent) {
			return fmt.Errorf("VLAN parent %s is missing", vlan.Parent)
		}
	}

	if n.Device != "" && host.Exists(n.Device) && !host.IsBridge(n.Device) {
		return fmt.Errorf("%s is not a native bridge", n.Device)
	}

	if n.Device == "" || !host.Exists(n.Device) {
		if n.Device != "" && !n.Owned {
			return fmt.Errorf("existing bridge %s is missing", n.Device)
		}

		device, err := host.CreateBridge()
		if err != nil {
			return err
		}

		n.Device, n.Owned = device, true
		n.AppliedMembers = nil
		if err := store.Save(n); err != nil {
			_ = host.Destroy(device)
			return err
		}
	}

	desired := slices.Clone(n.Members)
	for index := range n.VLANs {
		vlan := &n.VLANs[index]
		for _, applied := range n.AppliedVLANs {
			if applied.Parent == vlan.Parent && applied.Tag == vlan.Tag {
				vlan.Device = applied.Device
				vlan.Borrowed = applied.Borrowed
			}
		}

		if !host.Exists(vlan.Parent) {
			return fmt.Errorf("VLAN parent %s is missing", vlan.Parent)
		}

		previousDevice := vlan.Device
		device, owned, err := resolveVLAN(vlan.Device, !vlan.Borrowed, vlan.Parent, vlan.Tag)
		if err != nil {
			return err
		}
		vlan.Device, vlan.Borrowed = device, !owned
		n.AppliedVLANs = slices.DeleteFunc(n.AppliedVLANs, func(applied types.VLAN) bool { return applied.Parent == vlan.Parent && applied.Tag == vlan.Tag })
		n.AppliedVLANs = append(n.AppliedVLANs, *vlan)
		if err := store.Save(n); err != nil {
			if owned && device != previousDevice {
				_ = host.Destroy(device)
			}
			return err
		}

		if err := host.Up(vlan.Parent); err != nil {
			return err
		}

		desired = append(desired, vlan.Device)
	}

	members, err := host.Members(n.Device)
	if err != nil {
		return err
	}

	for _, previous := range slices.Clone(n.AppliedMembers) {
		if !slices.Contains(desired, previous) {
			if slices.Contains(members, previous) {
				if err := host.RemoveMember(n.Device, previous); err != nil {
					return err
				}
			}

			n.AppliedMembers = slices.DeleteFunc(n.AppliedMembers, func(member string) bool { return member == previous })
		}
	}

	for _, member := range desired {
		if !host.Exists(member) {
			return fmt.Errorf("bridge member %s is missing", member)
		}

		if !slices.Contains(members, member) {
			if err := host.AddMember(n.Device, member); err != nil {
				return err
			}

			if !slices.Contains(n.AppliedMembers, member) {
				n.AppliedMembers = append(n.AppliedMembers, member)
				if err := store.Save(n); err != nil {
					_ = host.RemoveMember(n.Device, member)
					return err
				}
			}
		}

		if err := host.Up(member); err != nil {
			return err
		}
	}

	for _, applied := range slices.Clone(n.AppliedVLANs) {
		if !slices.Contains(desired, applied.Device) {
			if !applied.Borrowed && host.Exists(applied.Device) {
				if err := host.Destroy(applied.Device); err != nil {
					return err
				}
			}

			n.AppliedVLANs = slices.DeleteFunc(n.AppliedVLANs, func(vlan types.VLAN) bool { return vlan.Device == applied.Device })
			if err := store.Save(n); err != nil {
				return err
			}
		}
	}

	if n.AppliedAddress != "" && n.AppliedAddress != n.Address {
		if err := host.RemoveAddress(n.Device, n.AppliedAddress); err != nil {
			return err
		}

		n.AppliedAddress = ""
	}

	if n.Address != "" && !host.HasAddress(n.Device, n.Address) {
		if err := host.SetAddress(n.Device, n.Address); err != nil {
			return err
		}

		n.AppliedAddress = n.Address
	}

	if err := host.Up(n.Device); err != nil {
		return err
	}

	return store.Save(n)
}

func ensureVLAN(store *Store, n *types.NetworkManifest, dryRun bool) error {
	lock, err := store.Lock()
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()

	current, err := store.Load(n.ID)
	if err != nil {
		return err
	}

	*n = *current
	if n.Mode != types.NetworkVLAN {
		return fmt.Errorf("network mode changed during apply; retry")
	}

	if dryRun {
		log.Info().Str("network", n.Name).Str("parent", n.Parent).Int("tag", n.Tag).Msg("would reconcile VLAN interface")
		return nil
	}

	if !host.Exists(n.Parent) {
		return fmt.Errorf("VLAN parent %s is missing", n.Parent)
	}

	previousDevice := n.Device
	device, owned, err := resolveVLAN(n.Device, n.Owned, n.Parent, n.Tag)
	if err != nil {
		return err
	}
	n.Device, n.Owned = device, owned
	if err := store.Save(n); err != nil {
		if owned && device != previousDevice {
			_ = host.Destroy(device)
		}
		return err
	}

	if err := host.Up(n.Parent); err != nil {
		return err
	}

	if err := host.Up(n.Device); err != nil {
		return err
	}

	return store.Save(n)
}

func Destroy(n *types.NetworkManifest) error {
	if n.Mode == types.NetworkVLAN {
		if n.Owned && n.Device != "" && host.Exists(n.Device) {
			return host.Destroy(n.Device)
		}

		return nil
	}

	if n.Mode != types.NetworkBridge {
		return nil
	}

	if host.Exists(n.Device) {
		if !host.IsBridge(n.Device) {
			return fmt.Errorf("%s is not a native bridge", n.Device)
		}

		members, err := host.Members(n.Device)
		if err != nil {
			return err
		}

		for _, member := range members {
			if !slices.Contains(n.AppliedMembers, member) && (n.Owned || len(member) >= 4 && member[:4] == "feth") {
				return fmt.Errorf("bridge %s still has unmanaged or active VM member %s", n.Device, member)
			}
		}

		for _, member := range n.AppliedMembers {
			if slices.Contains(members, member) {
				if err := host.RemoveMember(n.Device, member); err != nil {
					return err
				}
			}
		}
	}

	for _, vlan := range n.AppliedVLANs {
		if !vlan.Borrowed && vlan.Device != "" && host.Exists(vlan.Device) {
			if err := host.Destroy(vlan.Device); err != nil {
				return err
			}
		}
	}

	if !n.Owned && n.AppliedAddress != "" && host.Exists(n.Device) {
		if err := host.RemoveAddress(n.Device, n.AppliedAddress); err != nil {
			return err
		}
	}

	if n.Owned && host.Exists(n.Device) {
		return host.Destroy(n.Device)
	}

	return nil
}
