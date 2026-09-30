package main

import (
	"fmt"

	"github.com/m-vinc/maco/mcp/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

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

	cmd.Flags().StringVar(&configPath, "config", "maco-mcp.yml", "path to the Maco MCP instance registry")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	return cmd
}
