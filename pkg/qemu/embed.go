//go:build prod

package qemu

import "embed"

//go:embed all:assets
var assets embed.FS

func embeddedArchive() ([]byte, bool) {
	data, err := assets.ReadFile("assets/qemu.tar.gz")
	if err != nil || len(data) == 0 {
		return nil, false
	}

	return data, true
}
