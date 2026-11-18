package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

type State string

const (
	Pending   State = "pending"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
)

const (
	historyLimit    = 2000
	privateJobTTL   = 48 * time.Hour
	eventRetention  = 10 * time.Minute
	jobTimeout      = 2 * time.Hour
	shutdownTimeout = 35 * time.Second
	actionJob       = "maco:action"
)

type Job struct {
	ID         string     `json:"id"`
	Action     string     `json:"action"`
	Label      string     `json:"label"`
	Target     string     `json:"target"`
	State      State      `json:"state"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty" binding:"optional"`
	FinishedAt *time.Time `json:"finished_at,omitempty" binding:"optional"`
	Error      string     `json:"error,omitempty" binding:"optional"`
	Result     string     `json:"result,omitempty" binding:"optional"`
	Logs       []string   `json:"logs"`
}

func (j Job) Private() bool {
	return j.Action == "vm.screenshot"
}

type Payload struct {
	Interface  engine.InterfaceParams      `json:"interface"`
	USB        engine.USBParams            `json:"usb"`
	Hardware   engine.UpdateHardwareParams `json:"hardware"`
	Disk       engine.DiskParams           `json:"disk"`
	ID         string                      `json:"id"`
	Action     string                      `json:"action"`
	Target     string                      `json:"target"`
	Force      bool                        `json:"force"`
	VM         engine.CreateVMParams       `json:"vm"`
	SSHKeys    []string                    `json:"ssh_keys"`
	Network    engine.CreateNetworkParams  `json:"network"`
	Backup     engine.BackupParams         `json:"backup"`
	Snapshot   engine.SnapshotParams       `json:"snapshot"`
	GuestSetup engine.GuestSetupParams     `json:"guest_setup"`
}

type Service struct {
	engine  *engine.Engine
	db      *sql.DB
	queue   *goqite.Queue
	runner  *jobs.Runner
	cancel  context.CancelFunc
	stopped chan struct{}
}

type runnerLog struct{}

func (runnerLog) Info(message string, fields ...any) {
	log.Debug().Fields(fields).Msg(message)
}

func New(e *engine.Engine) (*Service, error) {
	database, err := e.OpenDB(context.Background())
	if err != nil {
		return nil, fmt.Errorf("open job database: %w", err)
	}

	queue := goqite.New(goqite.NewOpts{DB: database.DB, Name: "maco", MaxReceive: 1, Timeout: 30 * time.Second, SQLFlavor: goqite.SQLFlavorSQLite})
	runner := jobs.NewRunner(jobs.NewRunnerOpts{Queue: queue, Limit: 1, Extend: 30 * time.Second, PollInterval: 250 * time.Millisecond, Log: runnerLog{}})
	s := &Service{engine: e, db: database.DB, queue: queue, runner: runner}
	runner.Register(actionJob, s.process)
	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("open job database: %w", err)
	}

	ctx, s.cancel = context.WithCancel(ctx)
	s.stopped = make(chan struct{})
	workers := make(chan struct{})
	go func() {
		defer close(workers)
		s.runner.Start(ctx)
	}()

	go func() {
		defer close(s.stopped)
		s.runMaintenance(ctx)
		<-workers
	}()

	return nil
}

func (s *Service) Close() {
	if s.cancel == nil {
		return
	}

	s.cancel()
	select {
	case <-s.stopped:
	case <-time.After(shutdownTimeout):
		log.Warn().Msg("job runner did not stop in time")
	}
}

func (s *Service) Submit(ctx context.Context, payload Payload) (*Job, error) {
	payload.ID = uuid.NewString()
	j := &Job{ID: payload.ID, Action: payload.Action, Label: Label(payload.Action), Target: payload.Target, State: Pending, CreatedAt: time.Now().UTC(), Logs: []string{}}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := s.insert(ctx, j, body); err != nil {
		return nil, err
	}

	if !j.Private() {
		s.trimHistory(ctx)
	}

	return j, nil
}

func (s *Service) process(ctx context.Context, body []byte) error {
	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode task: %w", err)
	}

	j, err := s.Get(ctx, payload.ID)
	if err != nil {
		return err
	}

	if j.State == Succeeded || j.State == Failed {
		return nil
	}

	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if j.StartedAt != nil {
		return s.finish(persistCtx, j, "", errors.New("job interrupted; inspect resources before submitting again"))
	}

	now := time.Now().UTC()
	j.State, j.StartedAt = Running, &now
	if err := s.save(ctx, j); err != nil {
		return err
	}

	if err := s.appendLog(ctx, j.ID, "Starting "+j.Action+" "+j.Target); err != nil {
		return err
	}

	log.Info().Str("job", j.ID).Str("action", j.Action).Str("target", j.Target).Msg("job started")

	runCtx, stop := context.WithTimeout(ctx, jobTimeout)
	defer stop()
	logger := log.Logger.Output(&jobWriter{service: s, id: j.ID})
	result, actionErr := s.execute(logger.WithContext(runCtx), payload)
	return s.finish(persistCtx, j, result, actionErr)
}

func (s *Service) finish(ctx context.Context, j *Job, result string, actionErr error) error {
	now := time.Now().UTC()
	j.FinishedAt, j.Result, j.State = &now, result, Succeeded
	message := "Completed successfully"
	if actionErr != nil {
		j.State, j.Error = Failed, actionErr.Error()
		message = "Failed: " + actionErr.Error()
		log.Error().Err(actionErr).Str("job", j.ID).Str("action", j.Action).Str("target", j.Target).Msg("job failed")
	} else if j.StartedAt != nil {
		log.Info().Str("job", j.ID).Str("action", j.Action).Str("target", j.Target).Dur("took", now.Sub(*j.StartedAt)).Msg("job completed")
	}

	if err := s.appendLog(ctx, j.ID, message); err != nil {
		return err
	}

	return s.save(ctx, j)
}

func (s *Service) execute(ctx context.Context, p Payload) (string, error) {
	switch p.Action {
	case "vm.usb.assign":
		_, err := s.engine.AssignUSBSelection(ctx, p.Target, p.USB)
		return p.Target, err
	case "vm.usb.unassign":
		return p.Target, s.engine.UnassignUSB(ctx, p.Target, p.USB.AssignmentKey)
	case "vm.usb.attach":
		_, err := s.engine.AttachUSB(ctx, p.Target, p.USB)
		return p.Target, err
	case "vm.usb.detach":
		return p.Target, s.engine.DetachUSB(ctx, p.Target, p.USB.AttachmentID)
	case "vm.create":
		m, err := s.engine.CreateVM(p.VM, p.SSHKeys)
		if err != nil {
			return "", err
		}

		return m.ID, nil
	case "vm.start":
		_, err := s.engine.StartVM(ctx, p.Target)
		return p.Target, err
	case "vm.stop":
		return p.Target, s.engine.ForceStopVM(ctx, p.Target)
	case "vm.shutdown":
		return p.Target, s.engine.ShutdownVM(p.Target)
	case "vm.reboot":
		return p.Target, s.engine.RebootVM(p.Target, p.Force)
	case "vm.disk.add", "vm.disk.remove", "vm.disk.grow", "vm.disk.wipe", "vm.disk.replace":
		return p.Target, s.engine.ManageDisk(ctx, p.Target, p.Action, p.Disk)
	case "vm.interface.add", "vm.interface.update", "vm.interface.remove":
		return p.Target, s.engine.ManageInterfaceContext(ctx, p.Target, p.Action, p.Interface)
	case "vm.hardware":
		return p.Target, s.engine.UpdateHardware(ctx, p.Target, p.Hardware)
	case "vm.guest-setup":
		return p.Target, s.engine.SetGuestSetup(ctx, p.Target, p.GuestSetup)
	case "vm.screenshot":
		return p.Target, s.engine.ScreenshotVM(p.Target)
	case "vm.delete":
		return p.Target, s.engine.DeleteVM(ctx, p.Target)
	case "vm.backup":
		res, err := s.engine.BackupVM(ctx, p.Target)
		if err != nil {
			return "", err
		}

		if p.Backup.Prune {
			if _, err := s.engine.PruneBackups(ctx, res.VMID); err != nil {
				return "", fmt.Errorf("prune backups: %w", err)
			}
		}

		return res.VMID, nil
	case "vm.backup.restore":
		m, err := s.engine.RestoreVM(ctx, p.Target, p.Backup.Timestamp, p.Backup.AsNew)
		if err != nil {
			return "", err
		}

		return m.ID, nil
	case "vm.backup.delete":
		return p.Target, s.engine.DeleteBackup(p.Target, p.Backup.Timestamp)
	case "vm.backup.prune":
		if _, err := s.engine.PruneBackups(ctx, p.Target); err != nil {
			return "", err
		}

		return p.Target, nil
	case "vm.snapshot.create":
		if _, err := s.engine.CreateSnapshot(ctx, p.Target, p.Snapshot); err != nil {
			return "", err
		}

		return p.Target, nil
	case "vm.snapshot.restore":
		return p.Target, s.engine.RestoreSnapshot(ctx, p.Target, p.Snapshot)
	case "vm.snapshot.delete":
		return p.Target, s.engine.DeleteSnapshot(ctx, p.Target, p.Snapshot)
	case "network.create":
		n, err := s.engine.CreateNetwork(p.Network)
		if err != nil {
			return "", err
		}

		return n.ID, nil
	case "network.update":
		_, err := s.engine.UpdateNetwork(p.Target, p.Network)
		return p.Target, err
	case "network.apply":
		return p.Target, s.engine.ApplyNetwork(p.Target, false)
	case "network.destroy":
		return p.Target, s.engine.DestroyNetwork(p.Target)
	case "catalog.download":
		return p.Target, s.engine.DownloadCatalogImage(ctx, p.Target)
	case "boot.reconcile":
		return s.bootReconcile(ctx)
	case "host.poweroff":
		return p.Target, s.engine.PowerOffHost(ctx, p.Force)
	case "host.reboot":
		return p.Target, s.engine.RebootHost(ctx, p.Force)
	default:
		return "", fmt.Errorf("unknown action %q", p.Action)
	}
}

func (s *Service) bootReconcile(ctx context.Context) (string, error) {
	err := s.engine.BootReconcile(ctx)
	targets, listErr := s.engine.AutostartTargets()
	if listErr != nil {
		return "host", errors.Join(err, listErr)
	}

	for _, id := range targets {
		log.Ctx(ctx).Info().Str("vm", id).Msg("Queuing autostart")
		if _, submitErr := s.Submit(ctx, Payload{Action: "vm.start", Target: id}); submitErr != nil {
			err = errors.Join(err, fmt.Errorf("enqueue start %s: %w", id, submitErr))
		}
	}

	return "host", err
}

type jobWriter struct {
	service *Service
	id      string
}

func (w *jobWriter) Write(data []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		return 0, err
	}

	message, _ := event[zerolog.MessageFieldName].(string)
	if err := w.service.appendLog(ctx, w.id, message); err != nil {
		return 0, err
	}

	return len(data), nil
}
