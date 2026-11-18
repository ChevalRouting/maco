package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	RotationLabel     = "com.maco.logrotate"
	rotationSize      = 10 * 1024 * 1024
	rotationRule      = "%s 600 6 10240 * BNZ\n"
	newsyslogUnsafe   = " \t\r\n\\#"
	rotationRecoverIn = time.Minute
)

type serviceLog struct {
	label string
	name  string
}

func (m *Manager) RotateLogs(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("log rotation requires root; re-run with sudo")
	}

	lock, err := os.OpenFile(filepath.Join(m.cfg.LogDir, ".rotate.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()

	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil
		}

		return err
	}

	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()

	return m.rotateIfLarge(ctx, serviceLog{DaemonLabel, "daemon.log"})
}

func (m *Manager) rotateIfLarge(ctx context.Context, entry serviceLog) error {
	path := filepath.Join(m.cfg.LogDir, entry.name)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to rotate non-regular file %s", path)
	}

	if info.Size() < rotationSize {
		return nil
	}

	return rotateServiceLog(ctx, systemLaunchd, entry.label, path, runNewsyslog)
}

func rotateServiceLog(ctx context.Context, l launchd, label, path string, rotate func(context.Context, string) error) error {
	loaded, _, err := l.status(ctx, label)
	if err != nil {
		return err
	}

	if loaded {
		if err := l.stop(ctx, label); err != nil {
			return err
		}
	}

	rotationErr := rotate(ctx, path)
	if !loaded {
		return rotationErr
	}

	recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), rotationRecoverIn)
	defer cancel()
	out, err := l.run(recovery, "bootstrap", "system", plistPath(label))
	if err != nil {
		err = fmt.Errorf("restart %s after log rotation: %w: %s", label, err, out)
	}

	return errors.Join(rotationErr, err)
}

func runNewsyslog(ctx context.Context, path string) error {
	if strings.ContainsAny(path, newsyslogUnsafe) {
		return fmt.Errorf("unsupported newsyslog log path %q", path)
	}

	config, err := os.CreateTemp("", "maco-newsyslog-*.conf")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(config.Name()) }()

	_, writeErr := fmt.Fprintf(config, rotationRule, path)
	if err := errors.Join(writeErr, config.Close()); err != nil {
		return err
	}

	out, err := exec.CommandContext(ctx, "/usr/sbin/newsyslog", "-F", "-s", "-f", config.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rotate %s: %w: %s", path, err, out)
	}

	return nil
}
