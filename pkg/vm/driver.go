package vm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/m-vinc/maco/pkg/net/datapath"
	"github.com/m-vinc/maco/pkg/storage"
)

type Phase string

const (
	PhaseStopped Phase = "stopped"
	PhaseRunning Phase = "running"
)

type Status struct {
	Phase Phase
	PID   int
}

type Driver struct {
	startNetworkHelper func(string, string) error
	stopNetworkHelper  func(string) error
	runDir             string
}

func NewDriver(runDir string) *Driver { return &Driver{runDir: runDir} }

func (d *Driver) vmRunDir(id string) string    { return filepath.Join(d.runDir, id) }
func (d *Driver) pidPath(id string) string     { return filepath.Join(d.vmRunDir(id), "qemu.pid") }
func (d *Driver) qmpPath(id string) string     { return filepath.Join(d.vmRunDir(id), "qmp.sock") }
func (d *Driver) consolePath(id string) string { return filepath.Join(d.vmRunDir(id), "console.sock") }

func (d *Driver) SerialLogPath(id string) string {
	return filepath.Join(d.vmRunDir(id), "serial.log")
}

func (d *Driver) StartPreparedContext(ctx context.Context, id string, prepare func() (Spec, error)) (Status, error) {
	lock, err := d.LockContext(ctx, id)
	if err != nil {
		return Status{}, err
	}
	defer lock.Close()
	if st := d.Status(id); st.Phase == PhaseRunning {
		return st, nil
	}
	spec, err := prepare()
	if err != nil {
		return Status{}, err
	}
	if spec.ID != id {
		return Status{}, fmt.Errorf("prepared VM ID changed")
	}

	runDir := d.vmRunDir(spec.ID)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return Status{}, fmt.Errorf("mkdir run dir: %w", err)
	}

	if err := stopNetworkHelpers(runDir); err != nil {
		return Status{}, err
	}
	if err := os.Remove(d.networkChangePath(spec.ID)); err != nil && !os.IsNotExist(err) {
		return Status{}, err
	}
	_ = os.Remove(d.qmpPath(spec.ID))

	qemu, err := LocateQEMU()
	if err != nil {
		return Status{}, err
	}

	for i := range spec.Interfaces {
		if spec.Interfaces[i].Network == NetworkBridge {
			spec.Interfaces[i].Socket = datapath.Socket(filepath.Join(runDir, "interfaces", spec.Interfaces[i].ID))
		}
	}

	args, err := spec.buildArgs(runDir)
	if err != nil {
		return Status{}, err
	}

	logFile, err := os.Create(filepath.Join(runDir, "qemu.log"))
	if err != nil {
		return Status{}, fmt.Errorf("create qemu log: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	for _, nic := range spec.Interfaces {
		if nic.Network == NetworkBridge {
			dir := filepath.Join(runDir, "interfaces", nic.ID)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				_ = stopNetworkHelpers(runDir)
				return Status{}, err
			}
			if err := datapath.Start(dir, nic.Bridge); err != nil {
				_ = stopNetworkHelpers(runDir)
				return Status{}, err
			}
		}
	}

	cmd := exec.Command(qemu, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		_ = stopNetworkHelpers(runDir)
		return Status{}, fmt.Errorf("start qemu: %w", err)
	}

	pid := cmd.Process.Pid
	if err := d.writePID(spec.ID, pid); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = stopNetworkHelpers(runDir)
		return Status{}, err
	}

	go func() { _ = cmd.Wait() }()

	time.Sleep(400 * time.Millisecond)
	if !processAlive(pid) {
		_ = d.clearPID(spec.ID)
		_ = stopNetworkHelpers(runDir)
		return Status{}, fmt.Errorf("qemu exited immediately; see %s\n%s",
			filepath.Join(runDir, "qemu.log"), tailFile(filepath.Join(runDir, "qemu.log"), 20))
	}

	return Status{Phase: PhaseRunning, PID: pid}, nil
}

func (d *Driver) Stop(id string, graceful time.Duration) error {
	return d.stop(id, graceful, true)
}

func (d *Driver) ForceStop(id string) error {
	return d.stop(id, 0, false)
}

func (d *Driver) stop(id string, graceful time.Duration, guestShutdown bool) error {
	lock, err := d.lock(id)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()

	st := d.Status(id)
	if st.Phase != PhaseRunning {
		_ = d.clearPID(id)
		return stopNetworkHelpers(d.vmRunDir(id))
	}

	pid := st.PID

	if guestShutdown {
		if c, err := dialQMP(d.qmpPath(id), 3*time.Second); err == nil {
			_ = c.powerdown()
			_ = c.close()
			if waitExit(pid, graceful) {
				_ = d.clearPID(id)
				return stopNetworkHelpers(d.vmRunDir(id))
			}
		}

	}
	if err := d.signalVM(id, pid, syscall.SIGTERM); err != nil {
		return err
	}
	if waitExit(pid, 5*time.Second) {
		_ = d.clearPID(id)
		return stopNetworkHelpers(d.vmRunDir(id))
	}

	if err := d.signalVM(id, pid, syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill qemu %d: %w", pid, err)
	}

	if !waitExit(pid, 5*time.Second) {
		return fmt.Errorf("qemu %d did not exit after SIGKILL", pid)
	}

	_ = d.clearPID(id)
	return stopNetworkHelpers(d.vmRunDir(id))
}

func (d *Driver) Status(id string) Status {
	pid, err := d.readPID(id)
	if err != nil || pid <= 0 || !processAlive(pid) {
		return Status{Phase: PhaseStopped}
	}

	if data, err := os.ReadFile(d.pidPath(id) + ".identity"); err == nil {
		current, err := processIdentity(pid)
		if err != nil || current != string(data) {
			return Status{Phase: PhaseStopped}
		}
	}
	return Status{Phase: PhaseRunning, PID: pid}
}

func (d *Driver) writePID(id string, pid int) error {
	if err := os.MkdirAll(d.vmRunDir(id), 0o700); err != nil {
		return err
	}

	identity, err := processIdentity(pid)
	if err != nil {
		return err
	}
	if err := storage.WriteFile(d.pidPath(id)+".identity", []byte(identity), 0o600); err != nil {
		return err
	}
	return storage.WriteFile(d.pidPath(id), []byte(strconv.Itoa(pid)), 0o600)
}

func (d *Driver) readPID(id string) (int, error) {
	data, err := os.ReadFile(d.pidPath(id))
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func (d *Driver) clearPID(id string) error {
	_ = os.Remove(d.pidPath(id) + ".identity")
	err := os.Remove(d.pidPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	return errors.Is(err, syscall.EPERM)
}

func (d *Driver) signalVM(id string, pid int, sig syscall.Signal) error {
	expected, err := os.ReadFile(d.pidPath(id) + ".identity")
	if err != nil {
		return fmt.Errorf("cannot safely signal VM without process identity: %w", err)
	}
	current, err := processIdentity(pid)
	if err != nil {
		if !processAlive(pid) {
			return nil
		}
		return err
	}
	if current != string(expected) {
		return fmt.Errorf("VM process identity changed; refusing to signal PID %d", pid)
	}
	return signalPID(pid, sig)
}

func signalPID(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return proc.Signal(sig)
}

func waitExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}

		time.Sleep(100 * time.Millisecond)
	}

	return !processAlive(pid)
}

func tailFile(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return strings.Join(lines, "\n")
}

func (d *Driver) lock(id string) (*os.File, error) {
	return d.LockContext(context.Background(), id)
}
func (d *Driver) LockContext(ctx context.Context, id string) (*os.File, error) {
	if err := storage.ValidateID(id); err != nil {
		return nil, err
	}
	if err := storage.EnsurePrivateDir(d.runDir); err != nil {
		return nil, err
	}
	return storage.Lock(ctx, filepath.Join(d.vmRunDir(id), "lifecycle.lock"))
}

func (d *Driver) LockStopped(id string) (*os.File, error) {
	lock, err := d.lock(id)
	if err != nil {
		return nil, err
	}

	if d.Status(id).Phase == PhaseRunning {
		_ = lock.Close()
		return nil, fmt.Errorf("stop the VM before changing hardware")
	}

	return lock, nil
}

func (d *Driver) LockInterfaces(id string) (*os.File, error) { return d.lock(id) }

func (d *Driver) LockDisks(id string) (*os.File, error) {
	return d.lock(id)
}

func stopNetworkHelpers(runDir string) error {
	var result error
	entries, err := os.ReadDir(filepath.Join(runDir, "interfaces"))
	if err != nil && !os.IsNotExist(err) {
		result = err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := datapath.Stop(filepath.Join(runDir, "interfaces", entry.Name())); err != nil {
				result = err
			}
		}
	}
	if err := datapath.Stop(runDir); err != nil {
		result = err
	}
	if result == nil {
		if err := os.Remove(filepath.Join(runDir, "network-change.json")); err != nil && !os.IsNotExist(err) {
			result = err
		}
	}
	return result
}
