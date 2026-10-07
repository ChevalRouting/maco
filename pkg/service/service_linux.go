//go:build linux

package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DaemonLabel = "maco"
	unitPath    = "/etc/systemd/system/maco.service"
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
	return &Manager{cfg: cfg}
}

const unitTemplate = `[Unit]
Description=maco virtualization daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.Binary}} --data-dir {{.DataDir}} serve --addr {{.Addr}} --redis-url {{.RedisURL}}{{if .TLSCert}} --tls-cert {{.TLSCert}} --tls-key {{.TLSKey}}{{end}}
Environment=MACO_DATA_DIR={{.DataDir}}
Environment=MACO_REDIS_URL={{.RedisURL}}
Restart=on-failure
RestartSec=2
User=root

[Install]
WantedBy=multi-user.target
`

func (m *Manager) RenderDaemon() ([]byte, error) {
	tmpl, err := template.New("maco.service").Parse(unitTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, m.cfg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *Manager) Install(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service install requires root; re-run with sudo")
	}

	unit, err := m.RenderDaemon()
	if err != nil {
		return err
	}

	if err := os.WriteFile(unitPath, unit, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", unitPath, err)
	}

	if err := run(ctx, "systemctl", "daemon-reload"); err != nil {
		return err
	}

	if err := run(ctx, "systemctl", "enable", "--now", DaemonLabel+".service"); err != nil {
		return err
	}

	if !redisReachable(m.cfg.RedisURL) {
		log.Warn().Str("redis", m.cfg.RedisURL).Msg("redis is not reachable; start redis and ensure maco can reach it (maco serve retries until it connects)")
	}

	return nil
}

func (m *Manager) Uninstall(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("service uninstall requires root; re-run with sudo")
	}

	_ = run(ctx, "systemctl", "disable", "--now", DaemonLabel+".service")
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", unitPath, err)
	}

	return run(ctx, "systemctl", "daemon-reload")
}

func (m *Manager) Status(ctx context.Context) (string, error) {
	out, _ := exec.CommandContext(ctx, "systemctl", "status", DaemonLabel+".service", "--no-pager").CombinedOutput()
	return string(out), nil
}

func run(ctx context.Context, name string, args ...string) error {
	if out, err := exec.CommandContext(ctx, name, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, out)
	}

	return nil
}

func redisReachable(rawURL string) bool {
	host := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		host = u.Host
	}

	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "6379")
	}

	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return false
	}

	_ = conn.Close()
	return true
}
