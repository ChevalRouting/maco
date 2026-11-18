package service

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type archiver struct {
	t       *testing.T
	fixture *launchdFixture
	err     error
}

func (a *archiver) archive(context.Context, string) error {
	if a.fixture.loaded {
		a.t.Fatal("archiving while service is loaded")
	}

	a.fixture.events = append(a.fixture.events, "archive")
	return a.err
}

type cancellingLaunchd struct {
	fixture *launchdFixture
}

func (c cancellingLaunchd) run(ctx context.Context, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return c.fixture.run(ctx, args...)
}

func TestRotationStopsBeforeArchive(t *testing.T) {
	testRotationOrder(t, nil)
}

func TestRotationRestoresOnFailure(t *testing.T) {
	testRotationOrder(t, errors.New("disk full"))
}

func testRotationOrder(t *testing.T, rotationErr error) {
	f := &launchdFixture{loaded: true}
	a := &archiver{t: t, fixture: f, err: rotationErr}
	err := rotateServiceLog(context.Background(), launchd{run: f.run, alive: f.alive}, DaemonLabel, "/unused", a.archive)
	if !errors.Is(err, rotationErr) {
		t.Fatalf("err=%v want %v", err, rotationErr)
	}

	if !f.loaded {
		t.Fatal("service not restored")
	}

	events := strings.Join(f.events, "\n")
	if strings.Index(events, "bootout") > strings.Index(events, "archive") || strings.Index(events, "archive") > strings.Index(events, "bootstrap") {
		t.Fatal(events)
	}
}

func TestRotationDoesNotStartStoppedService(t *testing.T) {
	f := &launchdFixture{}
	err := rotateServiceLog(context.Background(), launchd{run: f.run, alive: f.alive}, DaemonLabel, "/unused", func(context.Context, string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	if f.loaded || strings.Contains(strings.Join(f.events, "\n"), "bootstrap") {
		t.Fatal(f.events)
	}
}

func TestRotationRestoresAfterCancellation(t *testing.T) {
	f := &launchdFixture{loaded: true}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := launchd{alive: f.alive, run: cancellingLaunchd{fixture: f}.run}
	err := rotateServiceLog(ctx, l, DaemonLabel, "/unused", func(context.Context, string) error { cancel(); return context.Canceled })
	if !errors.Is(err, context.Canceled) || !f.loaded {
		t.Fatalf("err=%v loaded=%v", err, f.loaded)
	}
}

func TestNewsyslogArchivesAndRetainsSeven(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.log")
	config := filepath.Join(dir, "rotation.conf")
	if err := os.WriteFile(config, fmt.Appendf(nil, rotationRule, path), 0o600); err != nil {
		t.Fatal(err)
	}

	for i := range 9 {
		rotateOnce(t, config, path, fmt.Sprintf("log %d\n", i))
	}

	archives, err := filepath.Glob(path + ".*.gz")
	if err != nil {
		t.Fatal(err)
	}

	if len(archives) != 7 {
		t.Fatalf("archives=%v", archives)
	}
}

func rotateOnce(t *testing.T, config, path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("/usr/sbin/newsyslog", "-r", "-F", "-s", "-f", config).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}

	if data := readArchive(t, path+".0.gz"); data != content {
		t.Fatalf("archive=%q want %q", data, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() != 0 || info.Mode().Perm() != 0o600 {
		t.Fatalf("new log: %v", info)
	}
}

func readArchive(t *testing.T, path string) string {
	archive, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = archive.Close() }()

	reader, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = reader.Close() }()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestRotationLaunchdPlist(t *testing.T) {
	m := NewManager(Config{})
	data, err := m.render(RotationLabel+".plist", plistData{Binary: "/tmp/a&b/maco", LogDir: "/Library/Logs/maco"})
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/usr/bin/plutil", "-lint", "-")
	cmd.Stdin = strings.NewReader(string(data))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("invalid plist: %v: %s", err, out)
	}

	for _, want := range []string{"/tmp/a&amp;b/maco", "<integer>3600</integer>", "rotate-logs", "--log-dir"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %q in %s", want, data)
		}
	}
}
