package engine

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
	"github.com/rs/zerolog/log"
)

const hostShutdownGrace = 90 * time.Second

func (e *Engine) PowerOffHost(ctx context.Context, force bool) error {
	return e.hostPower(ctx, force, "-h")
}

func (e *Engine) RebootHost(ctx context.Context, force bool) error {
	return e.hostPower(ctx, force, "-r")
}

func (e *Engine) hostPower(ctx context.Context, force bool, mode string) error {
	if !force {
		e.shutdownAllVMs(ctx)
	}
	log.Ctx(ctx).Info().Bool("force", force).Str("mode", mode).Msg("Powering host")
	bin, err := exec.LookPath("shutdown")
	if err != nil {
		bin = "/sbin/shutdown"
	}
	out, err := exec.CommandContext(ctx, bin, mode, "now").CombinedOutput()
	if err != nil {
		return fmt.Errorf("host power command: %w: %s", err, out)
	}
	return nil
}

func (e *Engine) shutdownAllVMs(ctx context.Context) {
	manifests, err := e.vms.List()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Listing VMs before host power operation")
		return
	}
	var wg sync.WaitGroup
	for _, m := range manifests {
		if e.driver.Status(m.ID).Phase != vm.PhaseRunning {
			continue
		}
		wg.Add(1)
		go func(m *types.VMManifest) {
			defer wg.Done()
			log.Ctx(ctx).Info().Str("vm", m.Name).Msg("Shutting down VM before host power operation")
			if err := e.StopVM(ctx, m.ID, hostShutdownGrace); err != nil {
				log.Ctx(ctx).Error().Err(err).Str("vm", m.Name).Msg("VM did not stop cleanly before host power operation")
			}
		}(m)
	}
	wg.Wait()
}
