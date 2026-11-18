package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type launchdFixture struct {
	loaded           bool
	registeredChecks int
	aliveChecks      int
	stopErr          error
	printErr         error
	events           []string
}

func (f *launchdFixture) run(_ context.Context, args ...string) ([]byte, error) {
	f.events = append(f.events, strings.Join(args, " "))
	switch args[0] {
	case "print":
		if f.printErr != nil {
			return nil, f.printErr
		}

		if !f.loaded && f.registeredChecks > 0 {
			f.registeredChecks--
			return []byte("state = stopping\npid = 42\n"), nil
		}

		if !f.loaded {
			return []byte(fmt.Sprintf("Could not find service %q in domain for system", DaemonLabel)), errors.New("exit 113")
		}

		return []byte("state = running\npid = 42\n"), nil
	case "bootout":
		if f.stopErr != nil {
			return []byte("permission denied"), f.stopErr
		}

		f.loaded = false
	case "bootstrap":
		if f.loaded || f.registeredChecks > 0 || f.aliveChecks > 0 {
			return nil, errors.New("bootstrap before shutdown completed")
		}

		f.loaded = true
	}

	return nil, nil
}

func (f *launchdFixture) alive(pid int) (bool, error) {
	if pid != 42 {
		return false, fmt.Errorf("unexpected PID %d", pid)
	}

	if f.aliveChecks > 0 {
		f.aliveChecks--
		return true, nil
	}

	return false, nil
}

func TestReloadWaitsForRegistrationAndProcess(t *testing.T) {
	f := &launchdFixture{loaded: true, registeredChecks: 1, aliveChecks: 3}
	l := launchd{run: f.run, alive: f.alive}
	if err := l.reload(context.Background(), DaemonLabel); err != nil {
		t.Fatal(err)
	}

	if f.events[len(f.events)-1] != "bootstrap system "+plistPath(DaemonLabel) {
		t.Fatal(f.events)
	}
}

func TestReloadPropagatesBootoutFailure(t *testing.T) {
	f := &launchdFixture{loaded: true, stopErr: errors.New("denied")}
	l := launchd{run: f.run, alive: f.alive}
	if err := l.reload(context.Background(), DaemonLabel); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, event := range f.events {
		if strings.HasPrefix(event, "bootstrap") {
			t.Fatal("bootstrap attempted after stop failure")
		}
	}
}

func TestStopAbsentService(t *testing.T) {
	f := &launchdFixture{}
	l := launchd{run: f.run, alive: f.alive}
	if err := l.stop(context.Background(), DaemonLabel); err != nil {
		t.Fatal(err)
	}

	if len(f.events) != 1 {
		t.Fatal(f.events)
	}
}

func TestStopPropagatesInspectionFailure(t *testing.T) {
	f := &launchdFixture{printErr: errors.New("launchctl unavailable")}
	l := launchd{run: f.run, alive: f.alive}
	if err := l.stop(context.Background(), DaemonLabel); err == nil {
		t.Fatal("inspection failure treated as absent service")
	}
}

func TestReloadCancellationDoesNotBootstrap(t *testing.T) {
	f := &launchdFixture{loaded: true, aliveChecks: 1000}
	l := launchd{run: f.run, alive: f.alive}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := l.reload(ctx, DaemonLabel); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout: %v", err)
	}

	for _, event := range f.events {
		if strings.HasPrefix(event, "bootstrap") {
			t.Fatal("bootstrap attempted before old process exited")
		}
	}
}
