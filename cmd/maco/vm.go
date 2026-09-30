package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/m-vinc/maco/pkg/engine"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newVMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "manage virtual machines",
	}

	cmd.AddCommand(
		newVMNewCommand(),
		newVMUSBCommand(),
		newVMListCommand(),
		newVMEditCommand(),
		newVMStartCommand(),
		newVMStopCommand(),
		newVMStatusCommand(),
		newVMConsoleCommand(),
		newVMBackupCommand(),
		newVMBackupsCommand(),
		newVMRestoreCommand(),
		newVMBackupRemoveCommand(),
		newVMSnapshotCommand(),
		newVMGuestAgentCommand(),
		newVMDestroyCommand(),
	)

	return cmd
}

func newVMNewCommand() *cobra.Command {
	var params engine.CreateVMParams

	cmd := &cobra.Command{
		Use:   "new NAME",
		Short: "create a VM manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			params.Name = args[0]
			m, err := eng().CreateVM(params, nil)
			if err != nil {
				return err
			}

			log.Info().Str("vm", m.Name).Str("id", m.ID).
				Str("manifest", cli.Paths.ManifestPath(m.ID)).Msg("vm created")
			return nil
		},
	}

	cmd.Flags().StringVar(&params.Image, "image", "ubuntu-24.04-arm64", "base cloud image")
	cmd.Flags().IntVar(&params.CPUs, "cpus", 2, "number of vCPUs")
	cmd.Flags().IntVar(&params.MemoryMiB, "memory", 2048, "memory in MiB")
	cmd.Flags().IntVar(&params.DiskSizeGiB, "disk-size", 20, "virtual disk size in GiB")
	cmd.Flags().StringVar(&params.Network, "network", "", "maco network to attach (empty = user-mode NAT)")
	cmd.Flags().StringSliceVar(&params.Addresses, "address", nil, "static guest IPv4/IPv6 CIDRs (default DHCP)")
	cmd.Flags().BoolVar(&params.Autostart, "autostart", false, "start this VM automatically on maco reconcile")
	return cmd
}

func newVMListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list VMs with their state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			views, err := eng().ListVMs(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Printf("%-36s  %-16s  %-9s  %-9s  %s\n", "ID", "NAME", "AUTOSTART", "STATE", "PID")
			for _, v := range views {
				autostart := "no"
				if v.Manifest.Autostart {
					autostart = "yes"
				}

				pid := "-"
				if v.PID > 0 {
					pid = strconv.Itoa(v.PID)
				}

				fmt.Printf("%-36s  %-16s  %-9s  %-9s  %s\n", v.Manifest.ID, v.Manifest.Name, autostart, v.Phase, pid)
			}

			return nil
		},
	}
}

func newVMEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit REF",
		Short: "edit a VM manifest in $EDITOR and save on exit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := eng().VMStore()
			m, err := store.Resolve(args[0])
			if err != nil {
				return err
			}

			original, err := os.ReadFile(cli.Paths.ManifestPath(m.ID))
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}

			return editManifestLoop(cmd.Context(), original, func(edited []byte) (bool, error) {
				updated, err := store.Parse(edited)
				if err != nil {
					return true, err
				}

				if updated.ID != m.ID {
					return true, fmt.Errorf("manifest id must not change (expected %s)", m.ID)
				}

				if err := store.Save(updated); err != nil {
					return false, err
				}

				log.Info().Str("vm", updated.Name).Str("id", updated.ID).Msg("manifest saved")
				if string(eng().Driver().Status(m.ID).Phase) == "running" {
					log.Warn().Msg("vm is running, restart to apply changes")
				}

				return false, nil
			})
		},
	}
}

func newVMStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start REF",
		Short: "boot a VM under HVF",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := eng().VMStore().Resolve(args[0])
			if err != nil {
				return err
			}

			st, err := eng().StartVM(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			log.Info().Str("vm", m.Name).Int("pid", st.PID).
				Str("serial", eng().Driver().SerialLogPath(m.ID)).Msg("vm running")
			return nil
		},
	}
}

func newVMStopCommand() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "stop REF",
		Short: "shut down a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := eng().StopVM(cmd.Context(), args[0], timeout); err != nil {
				return err
			}

			log.Info().Str("vm", args[0]).Msg("vm stopped")
			return nil
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "graceful shutdown timeout")
	return cmd
}

func newVMStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status REF",
		Short: "show a VM's runtime status",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			m, st, err := eng().StatusVM(args[0])
			if err != nil {
				return err
			}

			if st.PID > 0 {
				fmt.Printf("%s\t%s\tpid %d\n", m.Name, st.Phase, st.PID)
			} else {
				fmt.Printf("%s\t%s\n", m.Name, st.Phase)
			}

			return nil
		},
	}
}

func newVMConsoleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "console REF",
		Short: "attach to a VM's serial console (Ctrl+A Q to quit)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return eng().ConsoleVM(args[0])
		},
	}
}

func newVMBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup REF",
		Short: "snapshot a VM's disks into the maco backups folder (works while running)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := eng().BackupVM(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			log.Info().Str("vm", res.VMName).Str("path", res.Path).
				Int("disks", len(res.Disks)).Msg("vm backed up")
			return nil
		},
	}
}

func newVMBackupsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backups REF",
		Short: "list a VM's backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			backups, err := eng().ListBackups(args[0])
			if err != nil {
				return err
			}

			for _, backup := range backups {
				fmt.Printf("%s\t%d bytes\tlive=%t\tconsistent=%t\n", backup.Timestamp, backup.SizeBytes, backup.Live, backup.Consistent)
			}
			return nil
		},
	}
}

func newVMRestoreCommand() *cobra.Command {
	var asNew bool
	cmd := &cobra.Command{
		Use:   "restore REF TIMESTAMP",
		Short: "restore a VM from a backup (replaces the VM in place, or creates a new one with --as-new)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := eng().RestoreVM(cmd.Context(), args[0], args[1], asNew)
			if err != nil {
				return err
			}

			log.Info().Str("vm", m.Name).Str("id", m.ID).Bool("as_new", asNew).Msg("vm restored")
			return nil
		},
	}
	cmd.Flags().BoolVar(&asNew, "as-new", false, "restore into a brand new VM instead of replacing the existing one")
	return cmd
}

func newVMBackupRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup-rm REF TIMESTAMP",
		Short: "delete a VM backup",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := eng().DeleteBackup(args[0], args[1]); err != nil {
				return err
			}

			log.Info().Str("backup", args[1]).Msg("backup deleted")
			return nil
		},
	}
}

func newVMSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "manage VM snapshots",
	}

	var includeRAM bool
	create := &cobra.Command{
		Use:   "create REF TAG",
		Short: "create a snapshot (include the live RAM state with --ram while running)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshot, err := eng().CreateSnapshot(cmd.Context(), args[0], engine.SnapshotParams{Tag: args[1], IncludeRAM: includeRAM})
			if err != nil {
				return err
			}

			log.Info().Str("tag", snapshot.Tag).Bool("ram", snapshot.HasRAM).Msg("snapshot created")
			return nil
		},
	}
	create.Flags().BoolVar(&includeRAM, "ram", false, "include the live RAM state (running VMs only)")

	list := &cobra.Command{
		Use:   "list REF",
		Short: "list a VM's snapshots",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			snapshots, err := eng().ListSnapshots(args[0])
			if err != nil {
				return err
			}

			for _, snapshot := range snapshots {
				fmt.Printf("%s\tram=%t\t%s\n", snapshot.Tag, snapshot.HasRAM, snapshot.CreatedAt)
			}
			return nil
		},
	}

	restore := &cobra.Command{
		Use:   "restore REF TAG",
		Short: "restore a VM to a snapshot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := eng().RestoreSnapshot(cmd.Context(), args[0], engine.SnapshotParams{Tag: args[1]}); err != nil {
				return err
			}

			log.Info().Str("tag", args[1]).Msg("snapshot restored")
			return nil
		},
	}

	remove := &cobra.Command{
		Use:   "rm REF TAG",
		Short: "delete a VM snapshot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := eng().DeleteSnapshot(cmd.Context(), args[0], engine.SnapshotParams{Tag: args[1]}); err != nil {
				return err
			}

			log.Info().Str("tag", args[1]).Msg("snapshot deleted")
			return nil
		},
	}

	cmd.AddCommand(create, list, restore, remove)
	return cmd
}

func newVMGuestAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "guest-agent REF",
		Short: "show guest OS details reported by the qemu guest agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			info, err := eng().GuestAgent(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("agent version\t%s\n", info.Version)
			if info.Hostname != "" {
				fmt.Printf("hostname\t%s\n", info.Hostname)
			}
			if info.OS != nil {
				fmt.Printf("os\t%s\n", info.OS.PrettyName)
				fmt.Printf("kernel\t%s %s\n", info.OS.KernelRelease, info.OS.Machine)
			}
			for _, nic := range info.Interfaces {
				addresses := make([]string, 0, len(nic.IPAddresses))
				for _, addr := range nic.IPAddresses {
					addresses = append(addresses, fmt.Sprintf("%s/%d", addr.Address, addr.Prefix))
				}
				fmt.Printf("iface %s\t%s\n", nic.Name, strings.Join(addresses, " "))
			}
			for _, fs := range info.Filesystems {
				fmt.Printf("fs %s\t%s\t%s used of %s\n", fs.Mountpoint, fs.Type,
					humanBytes(fs.UsedBytes), humanBytes(fs.TotalBytes))
			}
			return nil
		},
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for size := n / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func newVMDestroyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy REF",
		Short: "delete a stopped VM and its disks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := eng().DeleteVM(cmd.Context(), args[0]); err != nil {
				return err
			}

			log.Info().Str("vm", args[0]).Msg("vm destroyed")
			return nil
		},
	}
}
