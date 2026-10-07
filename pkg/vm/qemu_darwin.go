//go:build darwin

package vm

const QEMUBinary = "qemu-system-aarch64"

func machineArgs(firmware string) []string {
	return []string{
		"-machine", "virt,highmem=on,gic-version=3",
		"-accel", "hvf",
		"-drive", "if=pflash,format=raw,readonly=on,file=" + firmware,
	}
}
