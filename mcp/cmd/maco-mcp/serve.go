package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-vinc/maco/mcp/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func defaultConfigPath() string {
	if p := os.Getenv("MACO_CLIENT_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "maco-client.yml"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "maco", "client.yml")
}

func serveCmd() *cobra.Command {
	var (
		configPath string
		logLevel   string
	)

	cmd := &cobra.Command{
		Use:           "maco-mcp",
		Short:         "Maco MCP server",
		Version:       version(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lvl, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return fmt.Errorf("invalid log level %q", logLevel)
			}

			zerolog.SetGlobalLevel(lvl)

			server, err := mcpserver.New(configPath, version())
			if err != nil {
				return err
			}

			return server.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath(), "path to the shared maco client config")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	return cmd
}
