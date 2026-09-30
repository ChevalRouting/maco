package main

import (
	"github.com/m-vinc/maco/pkg/net/datapath"
	"github.com/spf13/cobra"
)

func newNetworkPortCommand() *cobra.Command {
	var runDir, bridge string
	var uid, gid int
	cmd := &cobra.Command{
		Use:               "network-port",
		Hidden:            true,
		Args:              cobra.NoArgs,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE:              func(_ *cobra.Command, _ []string) error { return datapath.RunWorker(runDir, bridge, uid, gid) },
	}
	cmd.Flags().StringVar(&runDir, "run-dir", "", "private VM runtime directory")
	cmd.Flags().StringVar(&bridge, "bridge", "", "native bridge")
	cmd.Flags().IntVar(&uid, "uid", -1, "VM owner")
	cmd.Flags().IntVar(&gid, "gid", -1, "VM group")
	return cmd
}
