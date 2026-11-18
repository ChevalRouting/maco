//go:build !prod

package qemu

func embeddedArchive() ([]byte, bool) {
	return nil, false
}
