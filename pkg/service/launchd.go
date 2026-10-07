//go:build darwin

package service

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type launchd struct {
	run   func(context.Context, ...string) ([]byte, error)
	alive func(int) (bool, error)
}

var systemLaunchd = launchd{run: runLaunchctl, alive: processAlive}

func runLaunchctl(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "launchctl", args...).CombinedOutput()
}

func processAlive(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (l launchd) status(ctx context.Context, label string) (bool, int, error) {
	out, err := l.run(ctx, "print", "system/"+label)
	if err != nil {
		if ctx.Err() == nil && strings.Contains(string(out), fmt.Sprintf("Could not find service %q", label)) {
			return false, 0, nil
		}

		return false, 0, fmt.Errorf("inspect %s: %w: %s", label, err, out)
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "pid" && fields[1] == "=" {
			pid, err := strconv.Atoi(fields[2])
			if err != nil || pid <= 0 {
				return false, 0, fmt.Errorf("invalid PID for %s: %q", label, fields[2])
			}

			return true, pid, nil
		}
	}

	return true, 0, nil
}

func (l launchd) stop(ctx context.Context, label string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	loaded, pid, err := l.status(ctx, label)
	if err != nil {
		return err
	}

	if !loaded {
		return nil
	}

	out, err := l.run(ctx, "bootout", "system/"+label)
	if err != nil {
		return fmt.Errorf("bootout %s: %w: %s", label, err, out)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		loaded, _, err = l.status(ctx, label)
		if err != nil {
			return err
		}

		alive := false
		if pid > 0 {
			alive, err = l.alive(pid)
			if err != nil {
				return fmt.Errorf("check stopped %s PID %d: %w", label, pid, err)
			}
		}

		if !loaded && !alive {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s (PID %d) to stop: %w", label, pid, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (l launchd) reload(ctx context.Context, label string) error {
	if err := l.stop(ctx, label); err != nil {
		return err
	}

	out, err := l.run(ctx, "bootstrap", "system", plistPath(label))
	if err != nil {
		return fmt.Errorf("bootstrap %s: %w: %s", label, err, out)
	}

	return nil
}
