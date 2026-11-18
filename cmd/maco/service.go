package main

import (
	"fmt"
	"os"

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

	addServiceRotationCommand(cmd)
	return cmd
}

func serviceManager(addr, tlsCert, tlsKey string) (*service.Manager, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return serviceManagerFor(binary, addr, tlsCert, tlsKey), nil
}

func serviceManagerFor(binary, addr, tlsCert, tlsKey string) *service.Manager {
	return service.NewManager(service.Config{
		Binary:  binary,
		DataDir: cli.Paths.Root,
		Addr:    addr,
		TLSCert: tlsCert,
		TLSKey:  tlsKey,
	})
}

func newServiceInstallCommand() *cobra.Command {
	var (
		addr    string
		tlsCert string
		tlsKey  string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "write the launchd plists and load the daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if (tlsCert == "") != (tlsKey == "") {
				return fmt.Errorf("--tls-cert and --tls-key must be provided together")
			}

			mgr, err := serviceManager(addr, tlsCert, tlsKey)
			if err != nil {
				return err
			}

			if err := mgr.Install(cmd.Context()); err != nil {
				return err
			}

			log.Info().Str("label", service.DaemonLabel).Str("addr", addr).Msg("maco daemon installed")
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address for the web UI and API")
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
			mgr, err := serviceManager(":8080", "", "")
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
			mgr, err := serviceManager(":8080", "", "")
			if err != nil {
				return err
			}

			out, err := mgr.Status(cmd.Context())
			if err != nil {
				return err
			}

			_, _ = fmt.Print(out)
			return nil
		},
	}
}

func addServiceRotationCommand(parent *cobra.Command) {
	var logDir string
	cmd := &cobra.Command{
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		Use:               "rotate-logs", Short: "rotate service logs over 10 MiB (briefly restarts affected services)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return service.NewManager(service.Config{LogDir: logDir}).RotateLogs(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&logDir, "log-dir", service.LogDir, "service log directory")
	parent.AddCommand(cmd)
}
