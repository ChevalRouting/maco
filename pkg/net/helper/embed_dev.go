//go:build !prod

package helper

func embeddedBinary() ([]byte, bool) {
	return nil, false
}
