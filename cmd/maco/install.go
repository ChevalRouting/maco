package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	nethelper "github.com/m-vinc/maco/pkg/net/helper"
	"github.com/m-vinc/maco/pkg/service"
	"github.com/m-vinc/maco/pkg/tlscert"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const installPrefix = "/usr/local/bin"

func newInstallCommand() *cobra.Command {
	var (
		addr        string
		redisServer string
		noRedis     bool
		helper      string
		tlsCert     string
		tlsKey      string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install the binaries, generate a TLS certificate, and load the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("install requires root; re-run with sudo")
			}

			if (tlsCert == "") != (tlsKey == "") {
				return fmt.Errorf("--tls-cert and --tls-key must be provided together")
			}

			if err := cli.Paths.EnsureDirs(); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}

			source, err := os.Executable()
			if err != nil {
				return err
			}

			binary := filepath.Join(installPrefix, "maco")
			if err := installBinary(source, binary); err != nil {
				return fmt.Errorf("install maco: %w", err)
			}

			log.Info().Str("path", binary).Msg("installed maco")

			if err := installNetHelper(source, helper); err != nil {
				return err
			}

			if tlsCert == "" {
				if err := tlscert.EnsureSelfSigned(cli.Paths.TLSCertPath(), cli.Paths.TLSKeyPath()); err != nil {
					return fmt.Errorf("generate TLS certificate: %w", err)
				}

				log.Info().Str("cert", cli.Paths.TLSCertPath()).Msg("generated self-signed TLS certificate")
			} else {
				log.Info().Str("cert", tlsCert).Msg("using provided TLS certificate")
			}

			mgr, err := serviceManagerFor(binary, addr, redisServer, noRedis, tlsCert, tlsKey)
			if err != nil {
				return err
			}

			if err := mgr.Install(cmd.Context()); err != nil {
				return err
			}

			redisMissingWarning(redisServer, noRedis)

			log.Info().Str("label", service.DaemonLabel).Str("addr", addr).Msg("maco installed")
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address for the web UI and API")
	cmd.Flags().StringVar(&redisServer, "redis-server", "", "path to a redis-server binary to run as a companion daemon (default: look up on PATH)")
	cmd.Flags().BoolVar(&noRedis, "no-redis", false, "do not manage a Redis daemon (assume an external Redis)")
	cmd.Flags().StringVar(&helper, "helper", "", "path to the maco-net-helper binary (default: embedded, or alongside maco)")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS certificate for the daemon to serve (default: self-signed, generated in the data dir)")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS private key matching --tls-cert")
	return cmd
}

func installHelper(source, override, dst string) error {
	if override != "" {
		return installBinary(override, dst)
	}

	if nethelper.Embedded() {
		return nethelper.Install(dst)
	}

	sibling := filepath.Join(filepath.Dir(source), nethelper.Name)
	if _, err := os.Stat(sibling); err != nil {
		return fmt.Errorf("no embedded helper in this build and none beside %s; build with task build or pass --helper", source)
	}

	return installBinary(sibling, dst)
}

func installBinary(source, dst string) error {
	abs, err := filepath.Abs(source)
	if err != nil {
		return err
	}

	if abs == dst {
		return nil
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}

	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".maco-install-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	return os.Rename(tmpPath, dst)
}
