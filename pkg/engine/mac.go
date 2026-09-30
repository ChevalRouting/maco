package engine

import (
	"strings"

	"github.com/m-vinc/maco/pkg/vm"
)

type MACSuggestion struct {
	OUI string `json:"oui"`
	MAC string `json:"mac"`
}

func (e *Engine) SuggestMAC() MACSuggestion {
	used := e.usedMACs()
	mac := vm.RandomMAC()
	for i := 0; i < 64 && used[strings.ToLower(mac)]; i++ {
		mac = vm.RandomMAC()
	}
	return MACSuggestion{OUI: vm.MACPrefix(), MAC: mac}
}

func (e *Engine) usedMACs() map[string]bool {
	used := map[string]bool{}
	manifests, err := e.vms.List()
	if err != nil {
		return used
	}
	for _, m := range manifests {
		for _, nic := range m.EffectiveInterfaces() {
			mac := nic.MAC
			if mac == "" {
				mac = vm.InterfaceMAC(m.ID, nic.ID)
			}
			used[strings.ToLower(mac)] = true
		}
	}
	return used
}
