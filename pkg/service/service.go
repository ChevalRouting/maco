package service

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed assets/*.plist
var assets embed.FS

const (
	DaemonLabel      = "com.maco.daemon"
	legacyRedisLabel = "com.maco.redis"
	LaunchDaemons    = "/Library/LaunchDaemons"
	LogDir           = "/Library/Logs/maco"
)

type Config struct {
	Binary  string
	DataDir string
	Addr    string
	LogDir  string
	TLSCert string
	TLSKey  string
}

type Manager struct {
	cfg Config
}

func NewManager(cfg Config) *Manager {
	if cfg.LogDir == "" {
		cfg.LogDir = LogDir
	}

	return &Manager{cfg: cfg}
}

type plistData struct {
	DaemonLabel string
	Binary      string
	DataDir     string
	Addr        string
	LogDir      string
	Path        string
	TLSCert     string
	TLSKey      string
}

func (m *Manager) data() plistData {
	return plistData{
		DaemonLabel: DaemonLabel,
		Binary:      m.cfg.Binary,
		DataDir:     m.cfg.DataDir,
		Addr:        m.cfg.Addr,
		LogDir:      m.cfg.LogDir,
		Path:        daemonPath(),
		TLSCert:     m.cfg.TLSCert,
		TLSKey:      m.cfg.TLSKey,
	}
}

func daemonPath() string {
	return strings.Join([]string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, ":")
}

func (m *Manager) render(name string, data plistData) ([]byte, error) {
	raw, err := assets.ReadFile("assets/" + name)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *Manager) RenderDaemon() ([]byte, error) {
	return m.render(DaemonLabel+".plist", m.data())
}

func plistPath(label string) string {
	return filepath.Join(LaunchDaemons, label+".plist")
}

func (m *Manager) Install(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service install requires root; re-run with sudo")
	}

	if err := os.MkdirAll(m.cfg.LogDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", m.cfg.LogDir, err)
	}

	if err := m.removeLegacyRedis(ctx); err != nil {
		return err
	}

	daemon, err := m.RenderDaemon()
	if err != nil {
		return err
	}

	if err := writePlist(DaemonLabel, daemon); err != nil {
		return err
	}

	if err := systemLaunchd.reload(ctx, DaemonLabel); err != nil {
		return err
	}

	rotation, err := m.render(RotationLabel+".plist", plistData{Binary: m.cfg.Binary, LogDir: m.cfg.LogDir})
	if err != nil {
		return err
	}

	if err := writePlist(RotationLabel, rotation); err != nil {
		return err
	}

	return systemLaunchd.reload(ctx, RotationLabel)
}

func (m *Manager) Uninstall(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service uninstall requires root; re-run with sudo")
	}

	for _, label := range []string{RotationLabel, DaemonLabel} {
		if err := remove(ctx, label); err != nil {
			return err
		}
	}

	return m.removeLegacyRedis(ctx)
}

func (m *Manager) removeLegacyRedis(ctx context.Context) error {
	if _, err := os.Stat(plistPath(legacyRedisLabel)); os.IsNotExist(err) {
		return nil
	}

	return remove(ctx, legacyRedisLabel)
}

func remove(ctx context.Context, label string) error {
	if err := systemLaunchd.stop(ctx, label); err != nil {
		return err
	}

	if err := os.Remove(plistPath(label)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", plistPath(label), err)
	}

	return nil
}

func (m *Manager) Status(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	for _, label := range []string{DaemonLabel, RotationLabel} {
		cmd := exec.CommandContext(ctx, "launchctl", "print", "system/"+label)
		out, _ := cmd.CombinedOutput()
		_, _ = fmt.Fprintf(&buf, "=== %s ===\n%s\n", label, out)
	}

	return buf.String(), nil
}

func writePlist(label string, content []byte) error {
	path := plistPath(label)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
