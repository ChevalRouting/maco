package vm

import (
	"net"
	"strings"
	"testing"
)

func TestBridgeArgs(t *testing.T) {
	spec := Spec{ID: "vm-a", Name: "vm-a", Firmware: "/firmware", DiskPath: "/disk", Interfaces: []InterfaceSpec{{ID: "net0", Network: NetworkBridge, Bridge: "bridge0", Socket: "/tmp/network.sock"}}}
	args, err := spec.buildArgs("/tmp/run")
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(args, " ")
	for _, expected := range []string{"stream,id=net0,server=off,addr.type=unix,addr.path=/tmp/network.sock", "csum=off", "guest_tso4=off", "mac=" + MAC(spec.ID)} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in %s", expected, joined)
		}
	}

	if strings.Contains(joined, "vmnet") || strings.Contains(joined, "mcast") {
		t.Fatalf("unexpected backend: %s", joined)
	}

	spec.Interfaces[0].Socket = ""
	if _, err := spec.buildArgs("/tmp/run"); err == nil {
		t.Fatal("missing helper socket accepted")
	}
}

func TestMACIdentity(t *testing.T) {
	address := MAC("vm-a")
	first, err := net.ParseMAC(address)
	if err != nil {
		t.Fatal(err)
	}

	if first[0]&3 != 2 || address != MAC("vm-a") || MAC("vm-a") == MAC("vm-b") {
		t.Fatal("MAC must be stable, distinct, locally administered unicast")
	}
}

func TestSerialAndGraphicalConsoleArgs(t *testing.T) {
	spec := Spec{ID: "console-vm", Name: "console-vm", Firmware: "/firmware", DiskPath: "/disk"}
	args, err := spec.buildArgs("/tmp/console-run")
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(args, " ")
	for _, expected := range []string{"-vnc unix:/tmp/console-run/vnc.sock", "-device virtio-gpu-pci", "-device qemu-xhci", "-device usb-kbd", "-device usb-tablet", "-serial chardev:serial0", "path=/tmp/console-run/console.sock", "logfile=/tmp/console-run/serial.log"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing console option %q: %s", expected, joined)
		}
	}

	if strings.Contains(joined, "-nographic") || strings.Contains(joined, "5900") {
		t.Fatalf("unexpected display or public VNC setting: %s", joined)
	}
}

func TestDataDiskArgs(t *testing.T) {
	spec := Spec{ID: "disk-vm", Name: "disk-vm", Firmware: "/firmware", DiskPath: "/boot", Disks: []DiskSpec{{ID: "stable-serial", Path: "/data"}}}
	args, err := spec.buildArgs("/tmp/run")
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(args, " ")
	node := diskNodeName("stable-serial")
	if !strings.Contains(joined, "if=none,id="+node+",format=qcow2,file=/data") || !strings.Contains(joined, "scsi-hd,bus=scsi.0,id=device-stable-serial,drive="+node+",serial=stable-serial") {
		t.Fatalf("data disk attachment missing: %s", joined)
	}
}

func TestMediaBootOrder(t *testing.T) {
	spec := Spec{ID: "media", Firmware: "/firmware", DiskPath: "/disk", ISOs: []DiskSpec{{ID: "installer", Path: "/installer.iso"}}, BootOrder: []string{"iso:installer", "disk"}}
	args, err := spec.buildArgs("/run")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "scsi-hd,bus=scsi.0,drive=bootdisk,bootindex=2") || !strings.Contains(joined, "scsi-cd,bus=scsi.0,drive="+diskNodeName("installer")+",bootindex=1") || !strings.Contains(joined, "format=raw,readonly=on,file=/installer.iso") {
		t.Fatalf("invalid media args: %s", joined)
	}
}

func TestMultipleVMInterfaces(t *testing.T) {
	spec := Spec{ID: "multi", Firmware: "/firmware", DiskPath: "/disk", Interfaces: []InterfaceSpec{{ID: "net0", Network: NetworkUser}, {ID: "net1", Network: NetworkSwitch, Group: "239.1.2.3:1234"}, {ID: "net2", Network: NetworkBridge, Bridge: "bridge0", Socket: "/tmp/net2.sock"}}}
	args, err := spec.buildArgs("/run")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, expected := range []string{"user,id=net0", "socket,id=net1,mcast=239.1.2.3:1234", "stream,id=net2,server=off,addr.type=unix,addr.path=/tmp/net2.sock", "id=net0,mac=" + MAC(spec.ID), "id=net1,mac=" + InterfaceMAC(spec.ID, "net1"), "id=net2,mac=" + InterfaceMAC(spec.ID, "net2")} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s in %s", expected, joined)
		}
	}
	if strings.Count(joined, "virtio-net-pci,") != 3 {
		t.Fatal("incorrect adapter count")
	}
	spec.Interfaces = []InterfaceSpec{}
	args, err = spec.buildArgs("/run")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(args, " "), "virtio-net-pci") {
		t.Fatal("empty interfaces created default NIC")
	}
}

func TestIgnitionFwCfgArgs(t *testing.T) {
	spec := Spec{ID: "flatcar", Name: "flatcar", Firmware: "/firmware", DiskPath: "/disk", IgnitionPath: "/run/config.ign"}
	args, err := spec.buildArgs("/run")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-fw_cfg name=opt/org.flatcar-linux/config,file=/run/config.ign") {
		t.Fatalf("missing ignition fw_cfg: %s", joined)
	}

	spec.IgnitionPath = ""
	if strings.Contains(strings.Join(mustArgs(t, spec), " "), "fw_cfg") {
		t.Fatal("fw_cfg emitted without an ignition path")
	}
}

func mustArgs(t *testing.T, spec Spec) []string {
	t.Helper()
	args, err := spec.buildArgs("/run")
	if err != nil {
		t.Fatal(err)
	}
	return args
}
