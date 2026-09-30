package vm

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type NetworkMode string

const (
	NetworkUser         NetworkMode = "user"
	NetworkBridge       NetworkMode = "bridge"
	NetworkSwitch       NetworkMode = "switch"
	NetworkVmnetBridged NetworkMode = "vmnet-bridged"
)

func (m NetworkMode) NeedsRoot() bool {
	return m == NetworkVmnetBridged
}

type DiskSpec struct {
	ID   string
	Path string
}

type InterfaceSpec struct {
	Reference string
	ID        string
	MAC       string
	Network   NetworkMode
	Uplink    string
	Bridge    string
	Socket    string
	Group     string
}

type Spec struct {
	Interfaces   []InterfaceSpec
	ISOs         []DiskSpec
	BootOrder    []string
	Disks        []DiskSpec
	ID           string
	Name         string
	CPUs         int
	MemoryMiB    int
	DiskPath     string
	SeedPath     string
	IgnitionPath string
	Firmware     string
}

const QEMUBinary = "qemu-system-aarch64"

func LocateQEMU() (string, error) {
	path, err := exec.LookPath(QEMUBinary)
	if err != nil {
		return "", fmt.Errorf("%s not found in PATH (try `brew install qemu`): %w", QEMUBinary, err)
	}

	return path, nil
}

func (s *Spec) buildArgs(runDir string) ([]string, error) {
	if s.Firmware == "" {
		return nil, fmt.Errorf("spec %s: firmware is required", s.ID)
	}

	if s.DiskPath == "" {
		return nil, fmt.Errorf("spec %s: disk path is required", s.ID)
	}

	cpus := s.CPUs
	if cpus <= 0 {
		cpus = 1
	}

	mem := s.MemoryMiB
	if mem <= 0 {
		mem = 1024
	}

	args := []string{
		"-name", s.Name,
		"-machine", "virt,highmem=on,gic-version=3",
		"-accel", "hvf",
		"-cpu", "host",
		"-smp", strconv.Itoa(cpus),
		"-m", strconv.Itoa(mem),
		"-drive", "if=pflash,format=raw,readonly=on,file=" + s.Firmware,
		"-device", "virtio-scsi-pci,id=scsi",
		"-drive", "if=none,id=bootdisk,format=qcow2,file=" + s.DiskPath,
		"-device", "scsi-hd,bus=scsi.0,drive=bootdisk" + s.bootIndex("disk"),
	}

	if s.SeedPath != "" {
		args = append(args,
			"-drive", "if=none,id=seed,format=raw,file="+s.SeedPath,
			"-device", "scsi-hd,bus=scsi.0,drive=seed",
		)
	}

	if s.IgnitionPath != "" {
		args = append(args, "-fw_cfg", "name=opt/org.flatcar-linux/config,file="+s.IgnitionPath)
	}

	for _, iso := range s.ISOs {
		node := diskNodeName(iso.ID)
		args = append(args, "-drive", "if=none,id="+node+",format=raw,readonly=on,file="+iso.Path, "-device", "scsi-cd,bus=scsi.0,drive="+node+s.bootIndex("iso:"+iso.ID))
	}
	for _, disk := range s.Disks {
		deviceID := diskNodeName(disk.ID)
		args = append(args,
			"-drive", "if=none,id="+deviceID+",format=qcow2,file="+disk.Path,
			"-device", "scsi-hd,bus=scsi.0,id=device-"+disk.ID+",drive="+deviceID+",serial="+disk.ID+",device_id="+deviceID[:20]+s.bootIndex("disk:"+disk.ID),
		)
	}

	args = append(args, networkPortArgs()...)
	for _, nic := range s.Interfaces {
		if nic.MAC == "" {
			nic.MAC = InterfaceMAC(s.ID, nic.ID)
		}
		netArgs, err := networkArgs(nic)
		if err != nil {
			return nil, err
		}
		args = append(args, netArgs...)
	}

	console := "socket,id=serial0,path=" + filepath.Join(runDir, "console.sock") +
		",server=on,wait=off,logfile=" + filepath.Join(runDir, "serial.log")

	args = append(args,
		"-display", "none",
		"-vnc", "unix:"+filepath.Join(runDir, "vnc.sock"),
		"-device", "virtio-gpu-pci",
		"-device", "qemu-xhci,id=maco-usb",
		"-device", "usb-kbd",
		"-device", "usb-tablet",
		"-chardev", console,
		"-serial", "chardev:serial0",
		"-qmp", "unix:"+filepath.Join(runDir, "qmp.sock")+",server,nowait",
		"-chardev", "socket,path="+filepath.Join(runDir, "guest-agent.sock")+",server=on,wait=off,id=qga0",
		"-device", "virtio-serial-pci,id=maco-qga-serial",
		"-device", "virtserialport,chardev=qga0,name=org.qemu.guest_agent.0",
		"-device", "virtio-rng-pci",
	)

	return args, nil
}

const DefaultMACPrefix = "02:00:00"

var macPrefix = DefaultMACPrefix

func SetMACPrefix(base string) error {
	normalized, err := NormalizeMACPrefix(base)
	if err != nil {
		return err
	}
	macPrefix = normalized
	return nil
}

func NormalizeMACPrefix(base string) (string, error) {
	base = strings.TrimSpace(strings.ToLower(base))
	if base == "" {
		return DefaultMACPrefix, nil
	}
	parts := strings.Split(base, ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("base MAC %q needs at least three octets", base)
	}
	octets := [3]byte{}
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(parts[i], 16, 8)
		if err != nil {
			return "", fmt.Errorf("base MAC %q has an invalid octet %q", base, parts[i])
		}
		octets[i] = byte(v)
	}
	if octets[0]&1 != 0 {
		return "", fmt.Errorf("base MAC %q must be unicast (an even first octet)", base)
	}
	return fmt.Sprintf("%02x:%02x:%02x", octets[0], octets[1], octets[2]), nil
}

func MAC(id string) string {
	sum := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s:%02x:%02x:%02x", macPrefix, sum[0], sum[1], sum[2])
}

func MACPrefix() string { return macPrefix }

func RandomMAC() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s:%02x:%02x:%02x", macPrefix, b[0], b[1], b[2])
}

func (s *Spec) bootIndex(id string) string {
	order := s.BootOrder
	if len(order) == 0 {
		for _, iso := range s.ISOs {
			order = append(order, "iso:"+iso.ID)
		}
		order = append(order, "disk")
		for _, disk := range s.Disks {
			order = append(order, "disk:"+disk.ID)
		}
	}
	for i, item := range order {
		if item == id {
			return fmt.Sprintf(",bootindex=%d", i+1)
		}
	}
	return ""
}

func networkArgs(nic InterfaceSpec) ([]string, error) {
	args := []string{}
	switch nic.Network {
	case NetworkUser:
		args = append(args,
			"-netdev", "user,id=net0",
			"-device", "virtio-net-pci,netdev=net0",
		)
	case NetworkBridge:
		if nic.Bridge == "" || nic.Socket == "" {
			return nil, fmt.Errorf("spec %s: bridge network requires a bridge and helper socket", nic.ID)
		}

		args = append(args,
			"-netdev", "stream,id=net0,server=off,addr.type=unix,addr.path="+nic.Socket,
			"-device", "virtio-net-pci,netdev=net0,csum=off,guest_csum=off,gso=off,guest_tso4=off,guest_tso6=off,guest_ecn=off",
		)
	case NetworkSwitch:
		if nic.Group == "" {
			return nil, fmt.Errorf("spec %s: switch nic.Network requires a multicast group", nic.ID)
		}

		args = append(args,
			"-netdev", "socket,id=net0,mcast="+nic.Group,
			"-device", "virtio-net-pci,netdev=net0",
		)
	case NetworkVmnetBridged:
		if nic.Uplink == "" {
			return nil, fmt.Errorf("spec %s: vmnet-bridged requires an uplink interface", nic.ID)
		}

		args = append(args,
			"-netdev", "vmnet-bridged,id=net0,ifname="+nic.Uplink,
			"-device", "virtio-net-pci,netdev=net0",
		)
	default:
		return nil, fmt.Errorf("spec %s: unknown network mode %q", nic.ID, nic.Network)
	}

	for index := range args {
		args[index] = strings.ReplaceAll(args[index], "net0", nic.ID)
		if index > 0 && args[index-1] == "-device" {
			args[index] += ",id=" + nic.ID + ",mac=" + nic.MAC + ",bus=" + networkBus(nic.ID) + ",addr=0,disable-legacy=on"
		}
	}
	return args, nil
}
func InterfaceMAC(vmID, id string) string {
	if id == "net0" {
		return MAC(vmID)
	}
	return MAC(vmID + ":" + id)
}

func networkBus(id string) string { return "maco-" + id }
func networkPortArgs() []string {
	args := []string{}
	for i := 0; i < 32; i++ {
		port := fmt.Sprintf("pcie-root-port,id=%s,bus=pcie.0,addr=0x%x.%d,chassis=%d,slot=%d,port=%d,io-reserve=0", networkBus("net"+strconv.Itoa(i)), 16+i/8, i%8, i+1, i+1, i+1)
		if i%8 == 0 {
			port += ",multifunction=on"
		}
		args = append(args, "-device", port)
	}
	return args
}
