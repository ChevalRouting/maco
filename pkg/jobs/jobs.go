package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/m-vinc/maco/pkg/engine"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type State string

const (
	Pending   State = "pending"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
)

const historyLimit = 2000

const privateJobTTL = 48 * time.Hour

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
	engine    *engine.Engine
	redis     *redis.Client
	client    *asynq.Client
	inspector *asynq.Inspector
	worker    *asynq.Server
	prefix    string
	queue     string
	cancel    context.CancelFunc
	scheduled chan struct{}
}

func New(e *engine.Engine, addr string) (*Service, error) {
	option, err := asynq.ParseRedisURI(addr)
	if err != nil {
		return nil, fmt.Errorf("redis URL: %w", err)
	}

	redisOption, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256([]byte(e.Paths().Root))
	queue := "maco-" + hex.EncodeToString(sum[:8])
	s := &Service{engine: e, redis: redis.NewClient(redisOption), client: asynq.NewClient(option), inspector: asynq.NewInspector(option), prefix: queue + ":jobs:", queue: queue}
	s.worker = asynq.NewServer(option, asynq.Config{Concurrency: 1, Queues: map[string]int{queue: 1}, ShutdownTimeout: 35 * time.Second})
	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}

	if err := s.worker.Start(s); err != nil {
		return err
	}

	ctx, s.cancel = context.WithCancel(ctx)
	s.scheduled = make(chan struct{})
	go s.runMaintenance(ctx)
	return nil
}

func (s *Service) Close() {
	if s.cancel != nil {
		s.cancel()
		<-s.scheduled
	}

	s.worker.Shutdown()
	_ = s.client.Close()
	_ = s.inspector.Close()
	_ = s.redis.Close()
}

func (s *Service) save(ctx context.Context, j *Job) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	pipeline := s.redis.TxPipeline()
	if j.Private() {
		pipeline.Set(ctx, s.prefix+j.ID, data, privateJobTTL)
		pipeline.Expire(ctx, s.prefix+j.ID+":logs", privateJobTTL)
	} else {
		pipeline.Set(ctx, s.prefix+j.ID, data, 0)
	}

	pipeline.Publish(ctx, s.prefix+"events", data)
	_, err = pipeline.Exec(ctx)
	return err
}

func (s *Service) trimHistory(ctx context.Context) {
	stale, err := s.redis.ZRange(ctx, s.prefix+"index", 0, -(historyLimit + 1)).Result()
	if err != nil || len(stale) == 0 {
		return
	}

	keys := make([]string, len(stale))
	for i, id := range stale {
		keys[i] = s.prefix + id
	}

	values, err := s.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return
	}

	pipeline := s.redis.TxPipeline()
	trimmed := 0
	for i, id := range stale {
		if data, ok := values[i].(string); ok {
			var job Job
			if json.Unmarshal([]byte(data), &job) == nil && job.State != Succeeded && job.State != Failed {
				continue
			}
		}

		pipeline.Del(ctx, s.prefix+id, s.prefix+id+":logs")
		pipeline.ZRem(ctx, s.prefix+"index", id)
		trimmed++
	}

	if trimmed == 0 {
		return
	}

	_, _ = pipeline.Exec(ctx)
}

func (s *Service) Submit(ctx context.Context, payload Payload) (*Job, error) {
	payload.ID = uuid.NewString()
	j := &Job{ID: payload.ID, Action: payload.Action, Label: Label(payload.Action), Target: payload.Target, State: Pending, CreatedAt: time.Now().UTC(), Logs: []string{}}
	if err := s.save(ctx, j); err != nil {
		return nil, err
	}

	if !j.Private() {
		if err := s.redis.ZAdd(ctx, s.prefix+"index", redis.Z{Score: float64(j.CreatedAt.UnixMilli()), Member: j.ID}).Err(); err != nil {
			return nil, err
		}

		s.trimHistory(ctx)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	_, err = s.client.EnqueueContext(ctx, asynq.NewTask("maco:action", data), asynq.Queue(s.queue), asynq.TaskID(j.ID), asynq.MaxRetry(0), asynq.Timeout(2*time.Hour))
	if err != nil {
		j.State, j.Error = Failed, "could not enqueue job"
		now := time.Now().UTC()
		j.FinishedAt = &now
		_ = s.save(context.Background(), j)
		return nil, err
	}

	return j, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Job, error) {
	data, err := s.redis.Get(ctx, s.prefix+id).Bytes()
	if err != nil {
		return nil, err
	}

	var j Job
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}

	j.Label = Label(j.Action)

	if j.State == Pending || j.State == Running {
		info, err := s.inspector.GetTaskInfo(s.queue, id)
		if err == nil && info.State == asynq.TaskStateArchived {
			j.State, j.Error = Failed, info.LastErr
			if err := s.appendLog(ctx, j.ID, "Failed: "+info.LastErr); err != nil {
				return nil, err
			}

			now := time.Now().UTC()
			j.FinishedAt = &now
			if err := s.save(ctx, &j); err != nil {
				return nil, err
			}
		}
	}

	j.Logs, err = s.redis.LRange(ctx, s.prefix+id+":logs", 0, -1).Result()
	return &j, err
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	ids, err := s.redis.ZRevRange(ctx, s.prefix+"index", 0, 199).Result()
	if err != nil {
		return nil, err
	}

	ids, err = s.includeActive(ids)
	if err != nil {
		return nil, err
	}

	list := make([]Job, 0, len(ids))
	for _, id := range ids {
		j, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}

		j.Logs = []string{}
		list = append(list, *j)
	}

	return list, nil
}

func (s *Service) appendLog(ctx context.Context, id, message string) error {
	return s.redis.RPush(ctx, s.prefix+id+":logs", time.Now().UTC().Format(time.RFC3339)+" "+message).Err()
}

func (s *Service) ProcessTask(ctx context.Context, task *asynq.Task) error {
	lock, err := acquireLock(ctx, s.redis, s.prefix+"execlock")
	if err != nil {
		return err
	}

	defer lock.release()

	var payload Payload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode task: %w", err)
	}

	j, err := s.Get(ctx, payload.ID)
	if err != nil {
		return err
	}

	if j.State == Succeeded || j.State == Failed {
		return nil
	}

	if j.StartedAt != nil {
		return fmt.Errorf("job interrupted; inspect resources before submitting again")
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

	logger := log.Logger.Output(&jobWriter{service: s, id: j.ID})
	ctx = logger.WithContext(ctx)
	result, actionErr := s.execute(ctx, payload)
	now = time.Now().UTC()
	j.FinishedAt, j.Result = &now, result
	message := "Completed successfully"
	j.State = Succeeded
	if actionErr != nil {
		j.State, j.Error = Failed, actionErr.Error()
		message = "Failed: " + actionErr.Error()
		log.Error().Err(actionErr).Str("job", j.ID).Str("action", j.Action).Str("target", j.Target).Msg("job failed")
	} else {
		log.Info().Str("job", j.ID).Str("action", j.Action).Str("target", j.Target).Dur("took", now.Sub(*j.StartedAt)).Msg("job completed")
	}

	persistCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.appendLog(persistCtx, j.ID, message); err != nil {
		return err
	}

	if err := s.save(persistCtx, j); err != nil {
		return err
	}

	return actionErr
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
		return p.Target, s.engine.RebootVM(p.Target)
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
		if _, err := s.engine.PruneBackups(ctx, p.Target, p.Backup.KeepLast, p.Backup.MaxAgeDays); err != nil {
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

type taskLister func(string, ...asynq.ListOption) ([]*asynq.TaskInfo, error)

func (s *Service) includeActive(ids []string) ([]string, error) {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		seen[id] = true
	}

	listers := []taskLister{s.inspector.ListActiveTasks, s.inspector.ListPendingTasks}
	for _, listTasks := range listers {
		for page := 1; ; page++ {
			tasks, err := listTasks(s.queue, asynq.Page(page), asynq.PageSize(100))
			if errors.Is(err, asynq.ErrQueueNotFound) {
				break
			}

			if err != nil {
				return nil, err
			}

			for _, task := range tasks {
				if !seen[task.ID] {
					ids = append(ids, task.ID)
					seen[task.ID] = true
				}
			}

			if len(tasks) < 100 {
				break
			}
		}
	}

	return ids, nil
}

func (s *Service) reconcileOrphans(ctx context.Context) {
	ids, err := s.redis.ZRange(ctx, s.prefix+"index", 0, -1).Result()
	if err != nil {
		return
	}

	for offset := 0; offset < len(ids); offset += 200 {
		end := min(offset+200, len(ids))
		keys := make([]string, 0, end-offset)
		for _, id := range ids[offset:end] {
			keys = append(keys, s.prefix+id)
		}

		values, err := s.redis.MGet(ctx, keys...).Result()
		if err != nil {
			return
		}

		for _, value := range values {
			data, ok := value.(string)
			if !ok {
				continue
			}

			var job Job
			if json.Unmarshal([]byte(data), &job) != nil {
				continue
			}

			if job.State != Pending && job.State != Running {
				continue
			}

			if time.Since(job.CreatedAt) < time.Minute {
				continue
			}

			info, err := s.inspector.GetTaskInfo(s.queue, job.ID)
			if err == nil {
				if info.State != asynq.TaskStateArchived {
					continue
				}

				job.Error = info.LastErr
			} else if !errors.Is(err, asynq.ErrTaskNotFound) && !errors.Is(err, asynq.ErrQueueNotFound) {
				continue
			}

			if job.Error == "" {
				job.Error = "job was interrupted before it started"
			}

			job.State = Failed
			now := time.Now().UTC()
			job.FinishedAt = &now
			_ = s.appendLog(ctx, job.ID, "Failed: "+job.Error)
			_ = s.save(ctx, &job)
		}
	}
}

func (s *Service) Subscribe(ctx context.Context) (*redis.PubSub, error) {
	subscription := s.redis.Subscribe(ctx, s.prefix+"events")
	if _, err := subscription.Receive(ctx); err != nil {
		_ = subscription.Close()
		return nil, err
	}

	return subscription, nil
}
