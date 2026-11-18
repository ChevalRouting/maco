package engine

import "github.com/m-vinc/maco/pkg/net/host"

func (e *Engine) Interfaces() ([]host.Port, error) {
	return host.Ports()
}
