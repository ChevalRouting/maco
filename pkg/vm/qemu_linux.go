//go:build linux

package vm

const QEMUBinary = "qemu-system-x86_64"

func machineArgs(firmware string) []string {
	return []string{
		"-machine", "q35",
		"-accel", "kvm",
		"-bios", firmware,
	}
}
