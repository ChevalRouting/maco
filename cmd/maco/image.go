package main

import (
	"fmt"

	"github.com/m-vinc/maco/pkg/image"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "manage cached guest cloud images",
	}

	cmd.AddCommand(newImagePullCommand(), newImageListCommand())
	return cmd
}

func newImagePullCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pull NAME",
		Short: "download and cache a guest cloud image",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			img, err := image.Lookup(args[0])
			if err != nil {
				return err
			}

			path, err := image.Pull(cli.Paths.ImagesDir(), img)
			if err != nil {
				return err
			}

			log.Info().Str("path", path).Msg("image cached")
			return nil
		},
	}
}

func newImageListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list known images in the catalog",
		RunE: func(_ *cobra.Command, _ []string) error {
			for name, img := range image.Catalog {
				fmt.Printf("%-22s %s\n", name, img.URL)
			}

			return nil
		},
	}
}
