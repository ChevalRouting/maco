package main

import "github.com/spf13/cobra"

func newReconcileCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reconcile",
		Short: "start autostart VMs and refresh the state cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return eng().Reconcile(cmd.Context())
		},
	}
}
