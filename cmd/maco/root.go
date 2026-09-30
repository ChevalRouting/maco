package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(VERSION)
		},
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:               "maco",
		Short:             "lightweight macOS VMM (qemu + Apple Hypervisor.framework)",
		Version:           VERSION,
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: injectContext,
	}

	rootFlags(root)
	root.AddCommand(
		newVersionCommand(),
		newVMCommand(),
		newUSBCommand(),
		newNetCommand(),
		newNetworkPortCommand(),
		newReconcileCommand(),
		newUserCommand(),
		newServeCommand(),
		newServiceCommand(),
		newInstallCommand(),
		newImageCommand(),
	)

	return root
}

func rootFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cli.LogLevel, "log-level", "v", "info", "log level (debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&cli.DataDir, "data-dir", "", "data directory (default $MACO_DATA_DIR or /Library/Application Support/maco)")
}
