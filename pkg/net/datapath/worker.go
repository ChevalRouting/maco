package datapath

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/m-vinc/maco/pkg/net/host"
)

func RunWorker(runDir, bridge string, uid, gid int) (workerError error) {
	if os.Geteuid() != 0 {
		return fmt.Errorf("network worker requires root")
	}

	if err := syscall.Dup2(int(os.Stdout.Fd()), int(os.Stderr.Fd())); err != nil {
		return err
	}

	info, err := os.Lstat(runDir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("network runtime directory must be a private directory")
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != uid {
		return fmt.Errorf("network runtime directory has the wrong owner")
	}

	defer reportFailure(runDir, uid, gid, &workerError)

	if !host.IsBridge(bridge) {
		return fmt.Errorf("%s is not a native bridge", bridge)
	}

	helper, err := Helper()
	if err != nil {
		return err
	}

	endpoint, err := host.Create("feth")
	if err != nil {
		return err
	}

	peer, err := host.Create("feth")
	if err != nil {
		destroyInterface(endpoint)
		return err
	}
	defer cleanupWorker(runDir, endpoint, peer)

	if err := host.Pair(endpoint, peer); err != nil {
		return err
	}

	if err := host.AddMember(bridge, peer); err != nil {
		return err
	}

	if err := host.Up(endpoint); err != nil {
		return err
	}

	if err := host.Up(peer); err != nil {
		return err
	}

	port := Port{Bridge: bridge, Endpoint: endpoint, Peer: peer}
	if err := savePort(runDir, port, uid, gid); err != nil {
		return err
	}

	command := exec.Command(helper, endpoint, Socket(runDir), strconv.Itoa(uid), strconv.Itoa(gid), readyPath(runDir))
	command.Stdout, command.Stderr = os.Stdout, os.Stdout
	if err := command.Start(); err != nil {
		return err
	}

	finished := make(chan error, 1)
	go waitCommand(command, finished)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(signals)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-finished:
			return err
		case <-signals:
			_ = command.Process.Kill()
			<-finished
			return nil
		case <-ticker.C:
			if _, err := os.Stat(stopPath(runDir)); err == nil {
				_ = command.Process.Kill()
				<-finished
				return nil
			}
		}
	}
}

func savePort(runDir string, port Port, uid, gid int) error {
	data, err := json.Marshal(port)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(statePath(runDir), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := file.Chown(uid, gid); err != nil {
		return err
	}

	_, err = file.Write(data)
	return err
}

func destroyInterface(device string) {
	if err := host.Destroy(device); err != nil {
		_, _ = fmt.Fprintln(os.Stdout, err)
	}
}

func removeRuntimeFiles(runDir string) {
	for _, path := range []string{Socket(runDir), readyPath(runDir), stopPath(runDir), statePath(runDir)} {
		_ = os.Remove(path)
	}
}

func cleanupWorker(runDir, endpoint, peer string) {
	destroyInterface(peer)
	destroyInterface(endpoint)
	removeRuntimeFiles(runDir)
}

func reportFailure(runDir string, uid, gid int, workerError *error) {
	if *workerError == nil {
		return
	}

	file, err := os.OpenFile(failedPath(runDir), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	_ = file.Chown(uid, gid)
	_, _ = fmt.Fprint(file, *workerError)
}
