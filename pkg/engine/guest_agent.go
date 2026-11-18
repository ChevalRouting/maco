package engine

import "github.com/m-vinc/maco/pkg/vm"

func (e *Engine) GuestAgent(ref string) (*vm.GuestAgentInfo, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	return e.driver.GuestAgentInfo(m.ID)
}
