package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/m-vinc/maco/pkg/engine"
	"github.com/spf13/cobra"
)

func newUSBCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "usb", Short: "list physical USB devices on this host"}
	var asJSON bool
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		inventory, err := eng().ListUSBDevices(cmd.Context())
		if err != nil {
			return err
		}

		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(inventory)
		}

		if !inventory.Supported {
			return fmt.Errorf("%s", inventory.Reason)
		}

		out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(out, "ID\tPRODUCT\tMANUFACTURER\tVID:PID\tSERIAL\tBUS/PORT\tCLASS\tSTATE")
		for _, d := range inventory.Devices {
			serial := d.Serial
			if serial == "" {
				serial = "unavailable"
			}

			state := d.State
			if d.Reason != "" {
				state += " (" + d.Reason + ")"
			}

			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%04x:%04x\t%s\t%d/%s\t%s\t%s\n", d.ID, d.Product, d.Manufacturer, d.VendorID, d.ProductID, serial, d.Bus, d.Port, strings.Join(d.Classes, ", "), state)
		}

		return out.Flush()
	}}
	list.Flags().BoolVar(&asJSON, "json", false, "print full inventory as JSON")
	cmd.AddCommand(list)
	return cmd
}

func newVMUSBCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "usb", Short: "manage session and persistent USB attachments"}
	list := &cobra.Command{Use: "list VM", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		attachments, err := eng().ListVMUSB(cmd.Context(), args[0])
		if err != nil {
			return err
		}

		return json.NewEncoder(cmd.OutOrStdout()).Encode(attachments)
	}}
	attach := &cobra.Command{Use: "attach VM DEVICE-ID", Short: "attach a device from maco usb list", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		inventory, err := eng().ListUSBDevices(cmd.Context())
		if err != nil {
			return err
		}

		for _, device := range inventory.Devices {
			if device.ID != args[1] {
				continue
			}

			id, err := eng().AttachUSB(cmd.Context(), args[0], engine.USBParams{DeviceID: device.ID, Fingerprint: device.Fingerprint})
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), id)
			return err
		}

		return fmt.Errorf("USB selection is unavailable; run maco usb list again")
	}}
	detach := &cobra.Command{Use: "detach VM ATTACHMENT-ID", Short: "release a device after ejecting it in the guest", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return eng().DetachUSB(cmd.Context(), args[0], args[1])
	}}
	assign := &cobra.Command{Use: "assign VM DEVICE-ID", Short: "persist a device to the manifest so it reattaches on every start", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		assignment, err := eng().AssignUSB(cmd.Context(), args[0], args[1])
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), engine.AssignmentKey(assignment))
		return err
	}}
	unassign := &cobra.Command{Use: "unassign VM VENDOR:PRODUCT[:SERIAL]", Short: "remove a persistent assignment from the manifest", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return eng().UnassignUSB(cmd.Context(), args[0], args[1])
	}}
	assigned := &cobra.Command{Use: "assigned VM", Short: "list persistent USB assignments in the manifest", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		assignments, err := eng().ListUSBAssignments(args[0])
		if err != nil {
			return err
		}

		out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		_, _ = fmt.Fprintln(out, "KEY\tPRODUCT\tVID:PID\tSERIAL")
		for _, a := range assignments {
			serial := a.Serial
			if serial == "" {
				serial = "unavailable"
			}

			_, _ = fmt.Fprintf(out, "%s\t%s\t%04x:%04x\t%s\n", engine.AssignmentKey(a), a.Product, a.VendorID, a.ProductID, serial)
		}

		return out.Flush()
	}}
	cmd.AddCommand(list, attach, detach, assign, unassign, assigned)
	return cmd
}
