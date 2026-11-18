package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "manage web UI accounts",
	}

	cmd.AddCommand(
		newUserAddCommand(),
		newUserListCommand(),
		newUserPasswdCommand(),
		newUserRemoveCommand(),
	)

	return cmd
}

func newUserAddCommand() *cobra.Command {
	var (
		password string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "add USERNAME",
		Short: "create a web UI account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				return fmt.Errorf("--password is required")
			}

			if err := eng().AddUser(cmd.Context(), args[0], password, role); err != nil {
				return err
			}

			log.Info().Str("user", args[0]).Str("role", role).Msg("user created")
			return nil
		},
	}

	cmd.Flags().StringVar(&password, "password", "", "account password")
	cmd.Flags().StringVar(&role, "role", "admin", "account role (admin or viewer)")
	return cmd
}

func newUserListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list web UI accounts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			users, err := eng().ListUsers(cmd.Context())
			if err != nil {
				return err
			}

			for _, u := range users {
				_, _ = fmt.Printf("%-20s  %s\n", u.Username, u.Role)
			}

			return nil
		},
	}
}

func newUserPasswdCommand() *cobra.Command {
	var password string

	cmd := &cobra.Command{
		Use:   "passwd USERNAME",
		Short: "change an account password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				return fmt.Errorf("--password is required")
			}

			if err := eng().SetPassword(cmd.Context(), args[0], password); err != nil {
				return err
			}

			log.Info().Str("user", args[0]).Msg("password updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&password, "password", "", "new password")
	return cmd
}

func newUserRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rm USERNAME",
		Short: "delete a web UI account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := eng().DeleteUser(cmd.Context(), args[0]); err != nil {
				return err
			}

			log.Info().Str("user", args[0]).Msg("user deleted")
			return nil
		},
	}
}
