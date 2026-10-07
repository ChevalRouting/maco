package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "manage the maco launchd system daemon",
	}

	cmd.AddCommand(
		newServiceInstallCommand(),
		newServiceUninstallCommand(),
		newServiceStatusCommand(),
	)

	return cmd
}

func serviceManager(addr, redisServer string, noRedis bool, tlsCert, tlsKey string) (*service.Manager, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return serviceManagerFor(binary, addr, redisServer, noRedis, tlsCert, tlsKey)
}

func serviceManagerFor(binary, addr, redisServer string, noRedis bool, tlsCert, tlsKey string) (*service.Manager, error) {
	redisURL := os.Getenv("MACO_REDIS_URL")
	managed := managesRedis() && redisURL == ""
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	if !managesRedis() {
		noRedis = true
	}

	if !noRedis && redisServer == "" {
		redisServer = findRedisServer()
	}

	if noRedis {
		redisServer = ""
	}

	redisPassword := ""
	if managed && redisServer != "" {
		password, err := redisSecret(cli.Paths.Root)
		if err != nil {
			return nil, err
		}

		redisPassword = password
		redisURL = fmt.Sprintf("redis://:%s@127.0.0.1:6379/0", url.QueryEscape(password))
	}

	return service.NewManager(service.Config{
		Binary:        binary,
		DataDir:       cli.Paths.Root,
		Addr:          addr,
		RedisURL:      redisURL,
		RedisServer:   redisServer,
		RedisPassword: redisPassword,
		TLSCert:       tlsCert,
		TLSKey:        tlsKey,
	}), nil
}

func redisSecret(dir string) (string, error) {
	path := filepath.Join(dir, "redis.secret")
	data, err := os.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	if !os.IsNotExist(err) {
		return "", err
	}

	password, err := auth.GeneratePassword()
	if err != nil {
		return "", err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return redisSecret(dir)
		}

		return "", err
	}

	if _, err := file.WriteString(password); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return password, nil
}

func redisMissingWarning(redisServer string, noRedis bool) {
	if !managesRedis() {
		return
	}

	if !noRedis && redisServer == "" && findRedisServer() == "" {
		log.Warn().Msg("no redis-server found; install Redis with `brew install redis`, then re-run. maco serve will retry until Redis is reachable")
	}
}

func findRedisServer() string {
	if found, err := exec.LookPath("redis-server"); err == nil {
		return found
	}

	for _, p := range []string{"/opt/homebrew/bin/redis-server", "/usr/local/bin/redis-server"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}

	return ""
}

func newServiceInstallCommand() *cobra.Command {
	var (
		addr        string
		redisServer string
		noRedis     bool
		tlsCert     string
		tlsKey      string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "write the launchd plists and load the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if (tlsCert == "") != (tlsKey == "") {
				return fmt.Errorf("--tls-cert and --tls-key must be provided together")
			}

			mgr, err := serviceManager(addr, redisServer, noRedis, tlsCert, tlsKey)
			if err != nil {
				return err
			}

			if err := mgr.Install(cmd.Context()); err != nil {
				return err
			}

			redisMissingWarning(redisServer, noRedis)

			log.Info().Str("label", service.DaemonLabel).Str("addr", addr).Msg("maco daemon installed")
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address for the web UI and API")
	cmd.Flags().StringVar(&redisServer, "redis-server", "", "path to a redis-server binary to run as a companion daemon (default: look up on PATH)")
	cmd.Flags().BoolVar(&noRedis, "no-redis", false, "do not manage a Redis daemon (assume an external Redis)")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate for the daemon to serve (default: self-signed, generated in the data dir)")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS private key matching --tls-cert")
	return cmd
}

func newServiceUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "unload the daemon and remove the launchd plists",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mgr, err := serviceManager(":8080", "", false, "", "")
			if err != nil {
				return err
			}

			if err := mgr.Uninstall(cmd.Context()); err != nil {
				return err
			}

			log.Info().Msg("maco daemon uninstalled")
			return nil
		},
	}
}

func newServiceStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "print launchd status for the maco daemons",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mgr, err := serviceManager(":8080", "", false, "", "")
			if err != nil {
				return err
			}

			out, err := mgr.Status(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Print(out)
			return nil
		},
	}
}
