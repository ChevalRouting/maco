//go:build prod

package helper

import "embed"

//go:embed all:assets
var embedded embed.FS

func embeddedBinary() ([]byte, bool) {
	data, err := embedded.ReadFile("assets/" + Name)
	if err != nil || len(data) == 0 {
		return nil, false
	}

	return data, true
}
