//go:build linux

package host

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"

	vnl "github.com/vishvananda/netlink"
)

type Interface struct {
	Name  string
	Up    bool
	Addrs []string
}

type Port struct {
	Device       string   `json:"device"`
	HardwarePort string   `json:"hardware_port"`
	Up           bool     `json:"up"`
	Addrs        []string `json:"addresses"`
}

func Ports() ([]Port, error) {
	ifaces, err := List()
	if err != nil {
		return nil, err
	}

	ports := make([]Port, 0, len(ifaces))
	for name, iface := range ifaces {
		ports = append(ports, Port{
			Device: name,
			Up:     iface.Up,
			Addrs:  iface.Addrs,
		})
	}

	sort.Slice(ports, func(a, b int) bool { return ports[a].Device < ports[b].Device })
	return ports, nil
}

func List() (map[string]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	out := make(map[string]Interface, len(ifaces))
	for _, ifi := range ifaces {
		addrs, _ := ifi.Addrs()
		strs := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			strs = append(strs, addr.String())
		}

		out[ifi.Name] = Interface{Name: ifi.Name, Up: ifi.Flags&net.FlagUp != 0, Addrs: strs}
	}

	return out, nil
}

func Exists(name string) bool {
	_, err := vnl.LinkByName(name)
	return err == nil
}

func CreateBridge() (string, error) {
	name, err := uniqueName("mbr")
	if err != nil {
		return "", err
	}

	bridge := &vnl.Bridge{LinkAttrs: vnl.LinkAttrs{Name: name}}
	if err := vnl.LinkAdd(bridge); err != nil {
		return "", fmt.Errorf("create bridge %s: %w", name, err)
	}

	return name, nil
}

func Create(kind string) (string, error) {
	switch kind {
	case "tap":
		return createTap()
	case "bridge":
		return CreateBridge()
	case "vlan":
		return uniqueName("mvl")
	default:
		return "", fmt.Errorf("unsupported interface kind %q on linux", kind)
	}
}

func createTap() (string, error) {
	name, err := uniqueName("mtap")
	if err != nil {
		return "", err
	}

	tap := &vnl.Tuntap{LinkAttrs: vnl.LinkAttrs{Name: name}, Mode: vnl.TUNTAP_MODE_TAP}
	if err := vnl.LinkAdd(tap); err != nil {
		return "", fmt.Errorf("create tap %s: %w", name, err)
	}

	return name, nil
}

func Destroy(device string) error {
	link, err := vnl.LinkByName(device)
	if err != nil {
		return nil
	}

	if err := vnl.LinkDel(link); err != nil {
		return fmt.Errorf("destroy %s: %w", device, err)
	}

	return nil
}

func SetAddress(device, cidr string) error {
	addr, err := vnl.ParseAddr(cidr)
	if err != nil {
		return fmt.Errorf("parse address %q: %w", cidr, err)
	}

	link, err := vnl.LinkByName(device)
	if err != nil {
		return fmt.Errorf("lookup %s: %w", device, err)
	}

	if err := vnl.AddrReplace(link, addr); err != nil {
		return fmt.Errorf("set address %s on %s: %w", cidr, device, err)
	}

	return nil
}

func HasAddress(device, address string) bool {
	iface, err := net.InterfaceByName(device)
	if err != nil {
		return false
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		if addr.String() == address {
			return true
		}
	}

	return false
}

func RemoveAddress(device, address string) error {
	link, err := vnl.LinkByName(device)
	if err != nil {
		return nil
	}

	addr, err := vnl.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("parse address %q: %w", address, err)
	}

	if err := vnl.AddrDel(link, addr); err != nil && !strings.Contains(err.Error(), "cannot assign") {
		return fmt.Errorf("remove address %s from %s: %w", address, device, err)
	}

	return nil
}

func Up(device string) error {
	link, err := vnl.LinkByName(device)
	if err != nil {
		return fmt.Errorf("lookup %s: %w", device, err)
	}

	if err := vnl.LinkSetUp(link); err != nil {
		return fmt.Errorf("bring up %s: %w", device, err)
	}

	return nil
}

func Pair(left, right string) error {
	return fmt.Errorf("interface pairing is darwin-only; linux uses tap devices")
}

func IsBridge(device string) bool {
	link, err := vnl.LinkByName(device)
	if err != nil {
		return false
	}

	_, ok := link.(*vnl.Bridge)
	return ok
}

func Members(device string) ([]string, error) {
	bridge, err := vnl.LinkByName(device)
	if err != nil {
		return nil, fmt.Errorf("lookup bridge %s: %w", device, err)
	}

	links, err := vnl.LinkList()
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}

	index := bridge.Attrs().Index
	members := make([]string, 0)
	for _, link := range links {
		if link.Attrs().MasterIndex == index {
			members = append(members, link.Attrs().Name)
		}
	}

	sort.Strings(members)
	return members, nil
}

func AddMember(bridge, member string) error {
	master, err := vnl.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("lookup bridge %s: %w", bridge, err)
	}

	link, err := vnl.LinkByName(member)
	if err != nil {
		return fmt.Errorf("lookup member %s: %w", member, err)
	}

	if err := vnl.LinkSetMaster(link, master); err != nil {
		return fmt.Errorf("add %s to bridge %s: %w", member, bridge, err)
	}

	return nil
}

func RemoveMember(bridge, member string) error {
	link, err := vnl.LinkByName(member)
	if err != nil {
		return nil
	}

	if err := vnl.LinkSetNoMaster(link); err != nil {
		return fmt.Errorf("remove %s from bridge %s: %w", member, bridge, err)
	}

	return nil
}

func ConfigureVLAN(device, parent string, tag int) error {
	if link, err := vnl.LinkByName(device); err == nil {
		vlan, ok := link.(*vnl.Vlan)
		if !ok {
			return fmt.Errorf("%s exists and is not a VLAN", device)
		}

		parentLink, err := vnl.LinkByName(parent)
		if err != nil {
			return fmt.Errorf("lookup parent %s: %w", parent, err)
		}

		if vlan.VlanId == tag && vlan.ParentIndex == parentLink.Attrs().Index {
			return nil
		}

		return fmt.Errorf("VLAN %s already has different configuration; recreate it to change its parent or tag", device)
	}

	parentLink, err := vnl.LinkByName(parent)
	if err != nil {
		return fmt.Errorf("lookup parent %s: %w", parent, err)
	}

	vlan := &vnl.Vlan{
		LinkAttrs: vnl.LinkAttrs{Name: device, ParentIndex: parentLink.Attrs().Index},
		VlanId:    tag,
	}
	if err := vnl.LinkAdd(vlan); err != nil {
		return fmt.Errorf("create VLAN %s on %s tag %d: %w", device, parent, tag, err)
	}

	return nil
}

func FindVLAN(parent string, tag int) (string, error) {
	parentLink, err := vnl.LinkByName(parent)
	if err != nil {
		return "", nil
	}

	links, err := vnl.LinkList()
	if err != nil {
		return "", fmt.Errorf("list links: %w", err)
	}

	index := parentLink.Attrs().Index
	for _, link := range links {
		vlan, ok := link.(*vnl.Vlan)
		if ok && vlan.VlanId == tag && vlan.ParentIndex == index {
			return vlan.Name, nil
		}
	}

	return "", nil
}

func uniqueName(prefix string) (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate interface name: %w", err)
	}

	return prefix + hex.EncodeToString(buf), nil
}
