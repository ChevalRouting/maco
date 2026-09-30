package datapath

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Port struct {
	Bridge   string `json:"bridge"`
	Endpoint string `json:"endpoint"`
	Peer     string `json:"peer"`
}

func Socket(runDir string) string     { return filepath.Join(runDir, "network.sock") }
func statePath(runDir string) string  { return filepath.Join(runDir, "network.json") }
func readyPath(runDir string) string  { return filepath.Join(runDir, "network.ready") }
func failedPath(runDir string) string { return filepath.Join(runDir, "network.failed") }
func stopPath(runDir string) string   { return filepath.Join(runDir, "network.stop") }

func Helper() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	path := filepath.Join(filepath.Dir(executable), "maco-net-helper")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("maco-net-helper missing beside %s; run task build or install both binaries", executable)
	}

	return path, nil
}

func Start(runDir, bridge string) error {
	if _, err := Helper(); err != nil {
		return err
	}

	if err := Stop(runDir); err != nil {
		return err
	}

	for _, path := range []string{Socket(runDir), readyPath(runDir), stopPath(runDir), failedPath(runDir)} {
		_ = os.Remove(path)
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"network-port", "--run-dir", runDir, "--bridge", bridge, "--uid", strconv.Itoa(os.Getuid()), "--gid", strconv.Itoa(os.Getgid())}
	command := exec.Command(executable, args...)

	logFile, err := os.OpenFile(filepath.Join(runDir, "network.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	command.Stdin = os.Stdin
	command.Stdout = logFile
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return err
	}

	finished := make(chan error, 1)
	go waitCommand(command, finished)
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-finished:
			if err == nil {
				finished = nil
				continue
			}

			return fmt.Errorf("network worker exited before readiness: %v; see %s", err, filepath.Join(runDir, "network.log"))
		case <-deadline.C:
			_ = os.WriteFile(stopPath(runDir), nil, 0o600)
			return fmt.Errorf("network worker readiness timeout; see %s", filepath.Join(runDir, "network.log"))
		case <-ticker.C:
			if data, err := os.ReadFile(failedPath(runDir)); err == nil {
				return fmt.Errorf("network worker: %s; see %s", data, filepath.Join(runDir, "network.log"))
			}

			if _, err := os.Stat(readyPath(runDir)); err == nil {
				return nil
			}
		}
	}
}

func waitCommand(command *exec.Cmd, finished chan<- error) {
	finished <- command.Wait()
}

func Stop(runDir string) error {
	if _, err := os.Stat(statePath(runDir)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	if err := os.WriteFile(stopPath(runDir), nil, 0o600); err != nil {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(statePath(runDir)); errors.Is(err, os.ErrNotExist) {
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("network worker did not clean up; inspect %s", filepath.Join(runDir, "network.log"))
}
