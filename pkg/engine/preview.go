package engine

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func (e *Engine) captureInitialPreview(ctx context.Context, id string) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	if err := e.ScreenshotVM(id); err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("vm", id).Msg("Initial VM preview unavailable")
	}
}
