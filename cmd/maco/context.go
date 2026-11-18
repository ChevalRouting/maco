package main

import (
	"github.com/m-vinc/maco/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func injectContext(cmd *cobra.Command, _ []string) error {
	lvl, err := zerolog.ParseLevel(cli.LogLevel)
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(lvl)

	paths, err := config.Resolve(cli.DataDir)
	if err != nil {
		return err
	}

	if err := paths.EnsureDirs(); err != nil {
		return err
	}

	cli.Paths = paths

	logger := log.Logger.Level(lvl)
	cmd.SetContext(logger.WithContext(cmd.Context()))
	return nil
}
