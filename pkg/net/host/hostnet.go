//go:build darwin

package host

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strings"
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

	labels := hardwarePorts()
	ports := make([]Port, 0, len(ifaces))
	for name, iface := range ifaces {
		ports = append(ports, Port{
			Device:       name,
			HardwarePort: labels[name],
			Up:           iface.Up,
			Addrs:        iface.Addrs,
		})
	}

	sort.Slice(ports, func(a, b int) bool { return ports[a].Device < ports[b].Device })
	return ports, nil
}

func hardwarePorts() map[string]string {
	out, err := exec.Command("/usr/sbin/networksetup", "-listallhardwareports").CombinedOutput()
	if err != nil {
		return map[string]string{}
	}

	labels := make(map[string]string)
	current := ""
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "Hardware Port: "):
			current = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port: "))
		case strings.HasPrefix(line, "Device: "):
			device := strings.TrimSpace(strings.TrimPrefix(line, "Device: "))
			if device != "" {
				labels[device] = current
			}
		}
	}

	return labels
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
	_, err := net.InterfaceByName(name)
	return err == nil
}

func CreateBridge() (string, error) { return create("bridge") }

func Create(kind string) (string, error) {
	if kind != "feth" && kind != "vlan" {
		return "", fmt.Errorf("unsupported interface kind %q", kind)
	}

	return create(kind)
}

func create(kind string) (string, error) {
	out, err := ifconfig(kind, "create")
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(out)
	if name == "" {
		return "", fmt.Errorf("ifconfig %s create returned no device", kind)
	}

	return name, nil
}

func Destroy(device string) error {
	_, err := ifconfig(device, "destroy")
	return err
}

func SetAddress(device, cidr string) error {
	ip, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("parse address %q: %w", cidr, err)
	}

	mask := subnet.Mask
	if len(mask) != net.IPv4len {
		return fmt.Errorf("address %q: only IPv4 is supported", cidr)
	}

	netmask := fmt.Sprintf("0x%02x%02x%02x%02x", mask[0], mask[1], mask[2], mask[3])
	out, err := ifconfig(device, "inet", ip.String(), "netmask", netmask, "alias")
	if err != nil && !strings.Contains(out, "File exists") {
		return err
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
	if !HasAddress(device, address) {
		return nil
	}

	ip, _, err := net.ParseCIDR(address)
	if err != nil {
		return err
	}

	_, err = ifconfig(device, "inet", ip.String(), "-alias")
	return err
}

func Up(device string) error {
	_, err := ifconfig(device, "up")
	return err
}

func Pair(left, right string) error {
	_, err := ifconfig(left, "peer", right)
	return err
}

func IsBridge(device string) bool {
	out, err := inspect(device)
	return err == nil && strings.Contains(out, "Configuration:") && strings.HasPrefix(device, "bridge")
}

func Members(device string) ([]string, error) {
	out, err := inspect(device)
	if err != nil {
		return nil, err
	}

	matches := regexp.MustCompile(`member: ([a-zA-Z0-9]+)`).FindAllStringSubmatch(out, -1)
	members := make([]string, 0, len(matches))
	for _, match := range matches {
		members = append(members, match[1])
	}

	return members, nil
}

func AddMember(bridge, member string) error {
	_, err := ifconfig(bridge, "addm", member)
	return err
}

func RemoveMember(bridge, member string) error {
	_, err := ifconfig(bridge, "deletem", member)
	return err
}

func ConfigureVLAN(device, parent string, tag int) error {
	out, err := inspect(device)
	if err != nil {
		return err
	}

	match := regexp.MustCompile(`vlan: ([0-9]+) parent interface: ([a-zA-Z0-9]+)`).FindStringSubmatch(out)
	if len(match) != 0 {
		if match[1] == fmt.Sprint(tag) && match[2] == parent {
			return nil
		}

		return fmt.Errorf("VLAN %s already has different configuration; recreate it to change its parent or tag", device)
	}

	_, err = ifconfig(device, "vlan", fmt.Sprint(tag), "vlandev", parent)
	return err
}

func inspect(device string) (string, error) {
	out, err := exec.Command("/sbin/ifconfig", device).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect interface %s: %w: %s", device, err, out)
	}

	return string(out), nil
}

func ifconfig(args ...string) (string, error) {
	out, err := exec.Command("/sbin/ifconfig", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ifconfig %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}

	return string(out), nil
}

func FindVLAN(parent string, tag int) (string, error) {
	interfaces, err := List()
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(interfaces))
	for name := range interfaces {
		if strings.HasPrefix(name, "vlan") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		out, err := inspect(name)
		if err != nil {
			return "", err
		}
		if vlanMatches(out, parent, tag) {
			return name, nil
		}
	}
	return "", nil
}

func vlanMatches(out, parent string, tag int) bool {
	match := regexp.MustCompile(`vlan: ([0-9]+) parent interface: ([a-zA-Z0-9]+)`).FindStringSubmatch(out)
	return len(match) != 0 && match[1] == fmt.Sprint(tag) && match[2] == parent
}
