//go:build darwin

package main

import (
	"path/filepath"

	nethelper "github.com/m-vinc/maco/pkg/net/helper"
	"github.com/rs/zerolog/log"
)

func managesRedis() bool { return true }

func installNetHelper(source, override string) error {
	dst := filepath.Join(installPrefix, nethelper.Name)
	if err := installHelper(source, override, dst); err != nil {
		return err
	}

	log.Info().Str("path", dst).Msg("installed " + nethelper.Name)
	return nil
}
