package service

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed assets/*.plist
var assets embed.FS

const (
	DaemonLabel   = "com.maco.daemon"
	RedisLabel    = "com.maco.redis"
	LaunchDaemons = "/Library/LaunchDaemons"
	LogDir        = "/Library/Logs/maco"
)

type Config struct {
	Binary        string
	DataDir       string
	Addr          string
	RedisURL      string
	RedisServer   string
	RedisPassword string
	LogDir        string
	TLSCert       string
	TLSKey        string
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
	DaemonLabel   string
	RedisLabel    string
	Binary        string
	DataDir       string
	Addr          string
	RedisURL      string
	RedisServer   string
	RedisPassword string
	RedisHost     string
	RedisPort     string
	RedisDir      string
	LogDir        string
	Path          string
	TLSCert       string
	TLSKey        string
}

func (m *Manager) data() (plistData, error) {
	host, port, err := redisHostPort(m.cfg.RedisURL)
	if err != nil {
		return plistData{}, err
	}

	return plistData{
		DaemonLabel:   DaemonLabel,
		RedisLabel:    RedisLabel,
		Binary:        m.cfg.Binary,
		DataDir:       m.cfg.DataDir,
		Addr:          m.cfg.Addr,
		RedisURL:      m.cfg.RedisURL,
		RedisServer:   m.cfg.RedisServer,
		RedisPassword: m.cfg.RedisPassword,
		RedisHost:     host,
		RedisPort:     port,
		RedisDir:      filepath.Join(m.cfg.DataDir, "redis"),
		LogDir:        m.cfg.LogDir,
		Path:          daemonPath(),
		TLSCert:       m.cfg.TLSCert,
		TLSKey:        m.cfg.TLSKey,
	}, nil
}

func daemonPath() string {
	return strings.Join([]string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, ":")
}

func redisHostPort(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("redis URL: %w", err)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return u.Host, "6379", nil
	}

	if host == "" {
		host = "127.0.0.1"
	}

	return host, port, nil
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
	data, err := m.data()
	if err != nil {
		return nil, err
	}

	return m.render(DaemonLabel+".plist", data)
}

func (m *Manager) RenderRedis() ([]byte, error) {
	data, err := m.data()
	if err != nil {
		return nil, err
	}

	return m.render(RedisLabel+".plist", data)
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

	if err := os.MkdirAll(filepath.Join(m.cfg.DataDir, "redis"), 0o700); err != nil {
		return fmt.Errorf("mkdir redis dir: %w", err)
	}

	if m.cfg.RedisServer != "" {
		redis, err := m.RenderRedis()
		if err != nil {
			return err
		}

		if err := writePlist(RedisLabel, redis); err != nil {
			return err
		}

		if err := systemLaunchd.reload(ctx, RedisLabel); err != nil {
			return err
		}
	}

	daemon, err := m.RenderDaemon()
	if err != nil {
		return err
	}

	if err := writePlist(DaemonLabel, daemon); err != nil {
		return err
	}

	return systemLaunchd.reload(ctx, DaemonLabel)
}

func (m *Manager) Uninstall(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service uninstall requires root; re-run with sudo")
	}

	for _, label := range []string{DaemonLabel, RedisLabel} {
		if err := systemLaunchd.stop(ctx, label); err != nil {
			return err
		}

		if err := os.Remove(plistPath(label)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", plistPath(label), err)
		}
	}

	return nil
}

func (m *Manager) Status(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	for _, label := range []string{RedisLabel, DaemonLabel} {
		cmd := exec.CommandContext(ctx, "launchctl", "print", "system/"+label)
		out, _ := cmd.CombinedOutput()
		fmt.Fprintf(&buf, "=== %s ===\n%s\n", label, out)
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
