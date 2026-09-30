package vnet

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/m-vinc/maco/pkg/storage"
	"github.com/m-vinc/maco/pkg/types"
	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("network not found")

type Store struct {
	dir      string
	validate *validator.Validate
}

func NewStore(dir string) *Store {
	return &Store{dir: dir, validate: validator.New(validator.WithRequiredStructEnabled())}
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".yml")
}

func (s *Store) Parse(data []byte) (*types.NetworkManifest, error) {
	var n types.NetworkManifest
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, err
	}

	if err := s.check(&n); err != nil {
		return nil, fmt.Errorf("invalid network: %w", err)
	}

	return &n, nil
}

func (s *Store) Save(n *types.NetworkManifest) error {
	if err := storage.ValidateID(n.ID); err != nil {
		return err
	}
	if err := s.check(n); err != nil {
		return fmt.Errorf("invalid network: %w", err)
	}

	data, err := yaml.Marshal(n)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}

	return storage.WriteFile(s.path(n.ID), data, 0o600)
}

func (s *Store) Load(id string) (*types.NetworkManifest, error) {
	if err := storage.ValidateID(id); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	n, err := s.Parse(data)
	if err != nil {
		return nil, err
	}
	if n.ID != id {
		return nil, fmt.Errorf("network ID does not match filename")
	}
	return n, nil
}

func (s *Store) List() ([]*types.NetworkManifest, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	networks := make([]*types.NetworkManifest, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yml" {
			continue
		}

		n, err := s.Load(strings.TrimSuffix(e.Name(), ".yml"))
		if err != nil {
			return nil, err
		}

		networks = append(networks, n)
	}

	return networks, nil
}

func (s *Store) Delete(id string) error {
	err := os.Remove(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}

	return err
}

func (s *Store) Resolve(ref string) (*types.NetworkManifest, error) {
	if n, err := s.Load(ref); err == nil {
		return n, nil
	}

	networks, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, n := range networks {
		if n.Name == ref {
			return n, nil
		}
	}

	return nil, ErrNotFound
}

func (s *Store) check(n *types.NetworkManifest) error {
	if err := s.validate.Struct(n); err != nil {
		return err
	}

	for _, address := range []string{n.Address, n.AppliedAddress} {
		if address != "" {
			ip, _, err := net.ParseCIDR(address)
			if err != nil || ip.To4() == nil {
				return fmt.Errorf("bridge address must be an IPv4 CIDR")
			}
		}
	}

	if n.Mode == types.NetworkSwitch && n.Group == "" {
		return fmt.Errorf("switch requires a multicast group")
	}

	if (n.Mode == types.NetworkBridged || n.Mode == types.NetworkVmnetBridged) && n.Uplink == "" {
		return fmt.Errorf("bridged requires an uplink")
	}

	if n.Mode != types.NetworkBridged && n.Mode != types.NetworkVmnetBridged && n.Uplink != "" {
		return fmt.Errorf("uplink requires vmnet-bridged mode")
	}
	if n.Mode != types.NetworkVLAN && (n.Parent != "" || n.Tag != 0) {
		return fmt.Errorf("parent and tag require vlan mode; use vlans for bridge VLAN members")
	}

	if n.Mode == types.NetworkVLAN {
		if n.Parent == "" {
			return fmt.Errorf("vlan requires a parent interface")
		}

		if n.Tag < 1 || n.Tag > 4094 {
			return fmt.Errorf("vlan tag must be between 1 and 4094")
		}

		if n.Address != "" || len(n.Members) != 0 || len(n.VLANs) != 0 || n.Uplink != "" {
			return fmt.Errorf("vlan mode takes only a parent and tag")
		}

		if n.Device != "" && !regexp.MustCompile(`^vlan[0-9]+$`).MatchString(n.Device) {
			return fmt.Errorf("device must name a native VLAN")
		}
	} else {
		if n.Mode != types.NetworkBridge && (n.Device != "" || n.Address != "" || len(n.Members) != 0 || len(n.VLANs) != 0) {
			return fmt.Errorf("device, address, members and vlans require bridge mode")
		}

		if n.Device != "" && !regexp.MustCompile(`^bridge[0-9]+$`).MatchString(n.Device) {
			return fmt.Errorf("device must name a native bridge")
		}
	}

	for _, applied := range n.AppliedVLANs {
		if !regexp.MustCompile(`^vlan[0-9]+$`).MatchString(applied.Device) {
			return fmt.Errorf("applied VLAN device must name a native VLAN")
		}
	}

	for _, vlan := range n.VLANs {
		if vlan.Device != "" {
			owned := false
			for _, applied := range n.AppliedVLANs {
				if applied == vlan {
					owned = true
				}
			}

			if !owned {
				return fmt.Errorf("VLAN device is assigned by apply; use members to attach an existing VLAN")
			}
		}
	}

	seenMembers := make(map[string]bool)
	for _, member := range n.Members {
		if seenMembers[member] || member == n.Device {
			return fmt.Errorf("duplicate or self-referencing bridge member %s", member)
		}

		seenMembers[member] = true
	}

	seenVLANs := make(map[string]bool)
	for _, vlan := range n.VLANs {
		key := fmt.Sprintf("%s:%d", vlan.Parent, vlan.Tag)
		if seenVLANs[key] {
			return fmt.Errorf("duplicate VLAN %s", key)
		}

		seenVLANs[key] = true
	}

	valid := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{0,14}$`)
	interfaces := append([]string{}, n.Members...)
	if n.Uplink != "" {
		interfaces = append(interfaces, n.Uplink)
	}
	interfaces = append(interfaces, n.AppliedMembers...)
	for _, vlan := range n.VLANs {
		interfaces = append(interfaces, vlan.Parent)
		if vlan.Device != "" {
			interfaces = append(interfaces, vlan.Device)
		}
	}

	if n.Device != "" {
		interfaces = append(interfaces, n.Device)
	}

	if n.Parent != "" {
		interfaces = append(interfaces, n.Parent)
	}

	for _, device := range interfaces {
		if !valid.MatchString(device) {
			return fmt.Errorf("invalid interface %q", device)
		}
	}

	return nil
}
