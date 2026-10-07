package jobs

import (
	"context"
	"time"

	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/vm"
	"github.com/rs/zerolog/log"
)

func (s *Service) runMaintenance(ctx context.Context) {
	defer close(s.scheduled)
	screenshots := time.NewTicker(30 * time.Second)
	defer screenshots.Stop()
	orphans := time.NewTicker(5 * time.Minute)
	defer orphans.Stop()
	backups := time.NewTicker(time.Minute)
	defer backups.Stop()
	s.enqueueScreenshots(ctx)
	s.reconcileOrphans(ctx)
	s.enqueueScheduledBackups(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-screenshots.C:
			s.enqueueScreenshots(ctx)
		case <-orphans.C:
			s.reconcileOrphans(ctx)
		case <-backups.C:
			s.enqueueScheduledBackups(ctx)
		}
	}
}

func (s *Service) enqueueScheduledBackups(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	schedules, err := s.engine.ListSchedules(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Error().Err(err).Msg("list backup schedules")
		}

		return
	}

	now := time.Now()
	for _, schedule := range schedules {
		if !schedule.Enabled || schedule.IntervalHours < 1 {
			continue
		}

		if schedule.LastRunAt != 0 {
			due := time.Unix(schedule.LastRunAt, 0).Add(time.Duration(schedule.IntervalHours) * time.Hour)
			if now.Before(due) {
				continue
			}
		}

		claimed, err := s.redis.SetNX(ctx, s.prefix+"schedule:"+schedule.VMID, "1", time.Minute).Result()
		if err != nil || !claimed {
			continue
		}

		if _, err := s.Submit(ctx, Payload{Action: "vm.backup", Target: schedule.VMID}); err != nil {
			log.Error().Err(err).Str("vm", schedule.VMID).Msg("enqueue scheduled backup")
			continue
		}

		if err := s.engine.MarkScheduleRun(ctx, schedule.VMID, now); err != nil {
			log.Error().Err(err).Str("vm", schedule.VMID).Msg("record scheduled backup time")
		}

		if schedule.KeepLast > 0 || schedule.MaxAgeDays > 0 {
			if _, err := s.Submit(ctx, Payload{Action: "vm.backup.prune", Target: schedule.VMID, Backup: engine.BackupParams{KeepLast: schedule.KeepLast, MaxAgeDays: schedule.MaxAgeDays}}); err != nil {
				log.Error().Err(err).Str("vm", schedule.VMID).Msg("enqueue backup prune")
			}
		}
	}
}

func (s *Service) enqueueScreenshots(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	views, err := s.engine.ListVMs(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Error().Err(err).Msg("list VMs for screenshots")
		}

		return
	}

	for _, view := range views {
		if view.Phase != string(vm.PhaseRunning) {
			continue
		}

		id := view.Manifest.ID
		key := s.prefix + "preview:" + id
		previous, _ := s.redis.Get(ctx, key).Result()
		if previous != "" {
			job, err := s.Get(ctx, previous)
			if err != nil || job.State == Pending || job.State == Running {
				continue
			}
		}

		claimed, err := s.redis.SetNX(ctx, key+":lease", "1", 30*time.Second).Result()
		if err != nil || !claimed {
			continue
		}

		job, err := s.Submit(ctx, Payload{Action: "vm.screenshot", Target: id})
		if err != nil {
			log.Error().Err(err).Str("vm", id).Msg("enqueue screenshot")
			continue
		}

		if err := s.redis.Set(ctx, key, job.ID, 24*time.Hour).Err(); err != nil {
			log.Error().Err(err).Str("vm", id).Msg("save screenshot job reference")
		}
	}
}
