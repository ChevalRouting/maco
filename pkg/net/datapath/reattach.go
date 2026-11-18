package datapath

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/m-vinc/maco/pkg/net/host"
)

type Reattached struct {
	Bridge string
	Peer   string
	Err    error
}

func PortDirs(root string) []string {
	var dirs []string
	for _, pattern := range []string{filepath.Join(root, "bridges", "*"), filepath.Join(root, "*", "interfaces", "*")} {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			dirs = append(dirs, matches...)
		}
	}

	return dirs
}

func ReadPort(runDir string) (Port, bool) {
	data, err := os.ReadFile(statePath(runDir))
	if err != nil {
		return Port{}, false
	}

	var port Port
	if json.Unmarshal(data, &port) != nil || port.Bridge == "" || port.Peer == "" {
		return Port{}, false
	}

	return port, true
}

func ReattachPorts(root string) []Reattached {
	var results []Reattached
	members := make(map[string][]string)
	for _, dir := range PortDirs(root) {
		port, ok := ReadPort(dir)
		if !ok || !running(dir) || !host.Exists(port.Peer) || !host.IsBridge(port.Bridge) {
			continue
		}

		current, cached := members[port.Bridge]
		if !cached {
			list, err := host.Members(port.Bridge)
			if err != nil {
				continue
			}

			current = list
			members[port.Bridge] = current
		}

		if slices.Contains(current, port.Peer) {
			continue
		}

		err := host.AddMember(port.Bridge, port.Peer)
		if err == nil {
			err = host.Up(port.Peer)
			members[port.Bridge] = append(current, port.Peer)
		}

		results = append(results, Reattached{Bridge: port.Bridge, Peer: port.Peer, Err: err})
	}

	return results
}
