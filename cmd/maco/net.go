package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/m-vinc/maco/pkg/engine"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newNetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "networks",
		Aliases: []string{"net", "network"},
		Short:   "manage VM networks",
	}

	cmd.AddCommand(
		newNetNewCommand(),
		newNetListCommand(),
		newNetEditCommand(),
		newNetApplyCommand(),
		newNetDestroyCommand(),
	)

	return cmd
}

func newNetNewCommand() *cobra.Command {
	var (
		params engine.CreateNetworkParams
		vlans  []string
	)

	cmd := &cobra.Command{
		Use:     "new NAME",
		Aliases: []string{"create"},
		Short:   "create a network manifest",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			params.Name = args[0]
			for _, value := range vlans {
				parent, tagText, ok := strings.Cut(value, ":")
				tag, err := strconv.Atoi(tagText)
				if !ok || err != nil || tag < 1 || tag > 4094 {
					return fmt.Errorf("invalid VLAN %q; use PARENT:TAG with tag 1..4094", value)
				}

				params.VLANs = append(params.VLANs, types.VLAN{Parent: parent, Tag: tag})
			}

			n, err := eng().CreateNetwork(params)
			if err != nil {
				return err
			}

			log.Info().Str("network", n.Name).Str("id", n.ID).Str("mode", string(n.Mode)).Msg("network created")
			return nil
		},
	}

	cmd.Flags().StringVar(&params.Mode, "mode", "bridge", "network mode (bridge, vmnet-bridged, user, switch, vlan; bridged is a legacy alias)")
	cmd.Flags().StringVar(&params.Uplink, "uplink", "", "physical uplink interface for vmnet-bridged mode, e.g. en0")
	cmd.Flags().StringVar(&params.Parent, "parent", "", "parent interface for standalone vlan mode")
	cmd.Flags().IntVar(&params.Tag, "tag", 0, "tag 1..4094 for standalone vlan mode")
	cmd.Flags().StringVar(&params.Address, "address", "", "optional host IPv4 CIDR on the bridge")
	cmd.Flags().StringVar(&params.Device, "device", "", "attach to an existing native bridge instead of creating one")
	cmd.Flags().StringSliceVar(&params.Members, "member", nil, "existing interfaces to join to the bridge")
	cmd.Flags().StringSliceVar(&vlans, "vlan", nil, "native VLAN members, PARENT:TAG (repeatable)")
	return cmd
}

func newNetListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list networks with their state",
		RunE: func(_ *cobra.Command, _ []string) error {
			networks, err := eng().ListNetworks()
			if err != nil {
				return err
			}

			fmt.Printf("%-36s  %-16s  %-8s  %s\n", "ID", "NAME", "MODE", "TARGET")
			for _, n := range networks {
				target := n.Group
				if n.Mode == types.NetworkBridge {
					target = n.Device
				}

				if n.Mode == types.NetworkBridged || n.Mode == types.NetworkVmnetBridged {
					target = n.Uplink
				}

				if target == "" {
					target = "-"
				}

				fmt.Printf("%-36s  %-16s  %-8s  %s\n", n.ID, n.Name, n.Mode, target)
			}

			return nil
		},
	}
}

func newNetEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit REF",
		Short: "edit a network manifest in $EDITOR and save on exit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := eng().NetStore()
			n, err := store.Resolve(args[0])
			if err != nil {
				return err
			}

			original, err := os.ReadFile(cli.Paths.NetworksDir() + "/" + n.ID + ".yml")
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}

			return editManifestLoop(cmd.Context(), original, func(edited []byte) (bool, error) {
				updated, err := store.Parse(edited)
				if err != nil {
					return true, err
				}

				if updated.ID != n.ID {
					return true, fmt.Errorf("network id must not change (expected %s)", n.ID)
				}

				lock, err := store.Lock()
				if err != nil {
					return false, err
				}

				defer func() { _ = lock.Close() }()

				current, err := store.Load(n.ID)
				if err != nil {
					return false, err
				}

				if current.Device != updated.Device || current.Owned != updated.Owned || current.Mode != updated.Mode && current.Device != "" || !reflect.DeepEqual(current.AppliedVLANs, updated.AppliedVLANs) || !reflect.DeepEqual(current.AppliedMembers, updated.AppliedMembers) || current.AppliedAddress != updated.AppliedAddress {
					return true, fmt.Errorf("applied device, ownership and runtime fields must not change; reload if an apply ran while editing")
				}

				if err := store.Save(updated); err != nil {
					return false, err
				}

				log.Info().Str("network", updated.Name).Str("id", updated.ID).Msg("manifest saved")
				log.Warn().Msg("run `maco networks apply` to reconcile")
				return false, nil
			})
		},
	}
}

func newNetApplyCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "apply [REF]",
		Short: "create and reconcile native bridges and VLAN members",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return eng().ApplyNetworks(dryRun)
			}

			return eng().ApplyNetwork(args[0], dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report changes without modifying host interfaces")
	return cmd
}

func newNetDestroyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy REF",
		Short: "tear down a network and delete its manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := eng().DestroyNetwork(args[0]); err != nil {
				return err
			}

			log.Info().Str("network", args[0]).Msg("network destroyed")
			return nil
		},
	}
}
