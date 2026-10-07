//go:build prod && darwin

package helper

import "embed"

//go:embed assets/maco-net-helper
var embedded embed.FS

func embeddedBinary() ([]byte, bool) {
	data, err := embedded.ReadFile("assets/" + Name)
	if err != nil {
		return nil, false
	}

	return data, true
}
