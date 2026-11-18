package main

import (
	"fmt"
	"time"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/tlscert"
	"github.com/spf13/cobra"
)

func newTLSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tls",
		Short: "inspect the certificates served by the web UI and API",
	}

	cmd.AddCommand(newTLSStatusCommand())
	return cmd
}

func newTLSStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "print the configured certificates and their expiry",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			settings, err := config.LoadSettings(cli.Paths)
			if err != nil {
				return fmt.Errorf("read %s: %w", cli.Paths.ConfigPath(), err)
			}

			statuses, err := tlscert.Inspect(cli.Paths, settings.TLS)
			if err != nil {
				return err
			}

			for _, status := range statuses {
				printTLSStatus(status)
			}

			return nil
		},
	}
}

func printTLSStatus(status tlscert.Status) {
	if !status.Ready {
		_, _ = fmt.Printf("%-30s  %-11s  pending\n", status.Name, status.Source)
		return
	}

	_, _ = fmt.Printf("%-30s  %-11s  expires %s  issuer %s\n", status.Name, status.Source, status.NotAfter.Format(time.DateOnly), status.Issuer)
}
