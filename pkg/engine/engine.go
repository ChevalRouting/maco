package engine

import (
	"context"
	"path/filepath"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/db"
	"github.com/m-vinc/maco/pkg/manifest"
	"github.com/m-vinc/maco/pkg/net/vnet"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/usb"
	"github.com/m-vinc/maco/pkg/vm"
	"github.com/rs/zerolog/log"
)

type Engine struct {
	ensureNetwork func(*vnet.Store, *types.NetworkManifest, bool) error
	usbDevices    func() ([]usb.Device, error)
	usbClaimsDir  string
	paths         *config.Paths
	vms           *manifest.Store
	nets          *vnet.Store
	driver        *vm.Driver
}

func New(paths *config.Paths) *Engine {
	if settings, err := config.LoadSettings(paths); err != nil {
		log.Warn().Err(err).Msg("read maco config; using default base MAC")
	} else if err := vm.SetMACPrefix(settings.BaseMAC); err != nil {
		log.Warn().Err(err).Str("base_mac", settings.BaseMAC).Msg("invalid base MAC; using default")
	}

	return &Engine{
		paths:         paths,
		ensureNetwork: vnet.Ensure,
		vms:           manifest.NewStore(paths.VMsDir()),
		nets:          vnet.NewStore(paths.NetworksDir()),
		driver:        vm.NewDriver(paths.RunDir()),
		usbClaimsDir:  filepath.Join(paths.RunDir(), "usb"),
	}
}

func (e *Engine) Paths() *config.Paths     { return e.paths }
func (e *Engine) VMStore() *manifest.Store { return e.vms }
func (e *Engine) NetStore() *vnet.Store    { return e.nets }
func (e *Engine) Driver() *vm.Driver       { return e.driver }

func (e *Engine) OpenDB(ctx context.Context) (*db.DB, error) {
	return db.Shared(ctx, e.paths.DBPath())
}
