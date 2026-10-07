//go:build prod && !darwin

package helper

func embeddedBinary() ([]byte, bool) {
	return nil, false
}
