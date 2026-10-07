package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/net/vnet"
	"github.com/m-vinc/maco/pkg/provision"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
	"github.com/m-vinc/maco/pkg/vm/firmware"
	"github.com/rs/zerolog/log"
)

type VMView struct {
	Manifest *types.VMManifest `json:"manifest"`
	Phase    string            `json:"phase"`
	PID      int               `json:"pid"`
	BootTime int64             `json:"boot_time"`
}

type CreateVMParams struct {
	ISOs        []string `json:"isos,omitempty" binding:"optional" extensions:"x-maco-description=Exactly one media ID from listMedia for an ISO installation. Cannot be combined with image."`
	BootOrder   []string `json:"boot_order,omitempty" binding:"optional" extensions:"x-maco-description=Ordered boot entries: disk or disk:<disk-id> or iso:<media-id>. Omission uses the server default order."`
	Name        string   `json:"name" extensions:"x-maco-description=Unique VM name. Use an RFC 1123 hostname such as agent-vm." minLength:"1" maxLength:"63"`
	Image       string   `json:"image" binding:"optional" extensions:"x-maco-description=Image key from listCatalog or disk image ID from listMedia. Supply this or exactly one installation ISO."`
	CPUs        int      `json:"cpus" extensions:"x-maco-description=Virtual CPU count. At least 1." minimum:"1" example:"2"`
	MemoryMiB   int      `json:"memory_mib" extensions:"x-maco-description=Guest memory in MiB. At least 64." minimum:"64" example:"2048"`
	DiskSizeGiB int      `json:"disk_size_gib" extensions:"x-maco-description=Primary disk capacity in GiB. At least 1." minimum:"1" example:"20"`
	Network     string   `json:"network" binding:"optional" extensions:"x-maco-description=Default interface network ID from listNetworks or user for NAT. Omission uses user networking."`
	Addresses   []string `json:"addresses" binding:"optional" extensions:"x-maco-description=Static IPv4 or IPv6 CIDR addresses for the primary guest interface. Omission uses guest provisioning defaults."`
	Gateway     string   `json:"gateway" binding:"optional" extensions:"x-maco-description=Optional primary-interface gateway IP address."`
	Nameservers []string `json:"nameservers" binding:"optional" extensions:"x-maco-description=DNS server IP addresses for guest provisioning."`
	Tags        []string `json:"tags,omitempty" binding:"optional" extensions:"x-maco-description=Freeform labels stored on the VM for grouping and filtering, such as role or environment. Trimmed and de-duplicated."`
	Autostart   bool     `json:"autostart" binding:"optional" extensions:"x-maco-description=Whether host reconciliation should automatically start this VM. Default false."`

	Disks      []CreateDiskParams `json:"disks,omitempty" binding:"optional" extensions:"x-maco-description=Additional named data disks with capacities in GiB."`
	Interfaces []InterfaceParams  `json:"interfaces,omitempty" binding:"optional" extensions:"x-maco-description=Explicit interface list. Omit to use network; an empty list creates no interfaces. Maximum 32."`

	GuestSetup *types.GuestSetup `json:"guest_setup,omitempty" binding:"optional" extensions:"x-maco-description=Guest provisioning configuration. Discover supported provisioners and modes from listCatalog."`
}

type CreateDiskParams struct {
	Name    string `json:"name" extensions:"x-maco-description=Name for this additional disk." minLength:"1"`
	SizeGiB int    `json:"size_gib" extensions:"x-maco-description=Disk capacity in GiB." minimum:"1"`
}

type GuestSetupParams struct {
	Provisioner string            `json:"provisioner" binding:"optional"`
	Mode        string            `json:"mode" binding:"optional"`
	Hostname    string            `json:"hostname" binding:"optional"`
	Addresses   []string          `json:"addresses" binding:"optional"`
	Gateway     string            `json:"gateway" binding:"optional"`
	Nameservers []string          `json:"nameservers" binding:"optional"`
	Files       []types.GuestFile `json:"files" binding:"optional"`
}

type GuestSetupView struct {
	Provisioner  string                  `json:"provisioner"`
	Mode         string                  `json:"mode"`
	Hostname     string                  `json:"hostname"`
	Addresses    []string                `json:"addresses"`
	Gateway      string                  `json:"gateway"`
	Nameservers  []string                `json:"nameservers"`
	Files        []types.GuestFile       `json:"files"`
	Provisioning []types.GuestCapability `json:"provisioning"`
}

var vmName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)

var guestHostname = regexp.MustCompile(`^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(\.([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*$`)

func trimAddresses(addresses []string) []string {
	trimmed := []string{}
	for _, address := range addresses {
		if value := strings.TrimSpace(address); value != "" {
			trimmed = append(trimmed, value)
		}
	}

	return trimmed
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		out = append(out, value)
	}

	return out
}

func validateGuestNetwork(addresses []string, gateway string, nameservers []string) error {
	for _, address := range trimAddresses(addresses) {
		if _, _, err := net.ParseCIDR(address); err != nil {
			return fmt.Errorf("address %q must use CIDR notation", address)
		}
	}

	if gateway = strings.TrimSpace(gateway); gateway != "" && net.ParseIP(gateway) == nil {
		return fmt.Errorf("gateway %q must be an IP address", gateway)
	}

	for _, ns := range trimAddresses(nameservers) {
		if net.ParseIP(ns) == nil {
			return fmt.Errorf("DNS server %q must be an IP address", ns)
		}
	}

	return nil
}

func (e *Engine) CreateVM(params CreateVMParams, sshKeys []string) (*types.VMManifest, error) {
	if !vmName.MatchString(params.Name) {
		return nil, fmt.Errorf("name must be 1 to 63 characters of letters, digits, dot, dash or underscore")
	}

	if err := validateGuestNetwork(params.Addresses, params.Gateway, params.Nameservers); err != nil {
		return nil, err
	}

	if (params.Image != "" && len(params.ISOs) != 0) || (params.Image == "" && len(params.ISOs) != 1) {
		return nil, fmt.Errorf("select either one image or one installation ISO")
	}

	lock, err := e.lockMedia(context.Background())
	if err != nil {
		return nil, err
	}

	defer func() { _ = lock.Close() }()

	if params.Image != "" {
		if _, err := e.imageBase(context.Background(), params.Image, false); err != nil {
			return nil, err
		}
	}

	networkRef := params.Network
	if networkRef != "" && networkRef != "user" {
		n, err := e.nets.Resolve(networkRef)
		if err != nil {
			return nil, fmt.Errorf("resolve network %q: %w", networkRef, err)
		}

		networkRef = n.ID
	}

	if err := e.validateMedia(params.ISOs, params.BootOrder, nil); err != nil {
		return nil, err
	}

	m := &types.VMManifest{
		ISOs: params.ISOs, BootOrder: params.BootOrder,
		ID:          uuid.NewString(),
		Name:        params.Name,
		Image:       params.Image,
		CPUs:        params.CPUs,
		MemoryMiB:   params.MemoryMiB,
		DiskSizeGiB: params.DiskSizeGiB,
		Network:     networkRef,
		Addresses:   params.Addresses,
		Gateway:     strings.TrimSpace(params.Gateway),
		Nameservers: trimAddresses(params.Nameservers),
		SSHKeys:     sshKeys,
		Tags:        normalizeTags(params.Tags),
		Autostart:   params.Autostart,
	}

	if params.Interfaces != nil {
		interfaces := make([]types.VMInterface, 0, len(params.Interfaces))
		for i := range params.Interfaces {
			interfaces = append(interfaces, types.VMInterface{
				ID:      fmt.Sprintf("net%d", i),
				Network: params.Interfaces[i].Network,
				MAC:     strings.TrimSpace(params.Interfaces[i].MAC),
			})
		}

		m.Interfaces = &interfaces
		setPrimaryNetwork(m, m.Addresses, m.Gateway, m.Nameservers)
		normalized, err := e.normalizedInterfaces(m)
		if err != nil {
			return nil, err
		}

		m.Interfaces = &normalized
	}

	if params.GuestSetup == nil || params.GuestSetup.Provisioner == types.ProvisionerNone {
		m.GuestSetup = &types.GuestSetup{Provisioner: types.ProvisionerNone}
	} else {
		setup := *params.GuestSetup
		caps := e.guestCapabilities(m)
		if setup.Provisioner == "" {
			setup.Provisioner = caps[0].Provisioner
		}

		if setup.Mode == "" {
			setup.Mode = types.GuestModeRaw
		}

		if err := validateGuestSetup(caps, setup); err != nil {
			return nil, err
		}

		m.GuestSetup = &setup
	}

	for i := range params.Disks {
		name := strings.TrimSpace(params.Disks[i].Name)
		if name == "" || params.Disks[i].SizeGiB < 1 {
			_ = os.RemoveAll(e.paths.VMDiskDir(m.ID))
			return nil, fmt.Errorf("each additional disk needs a name and capacity of at least 1 GiB")
		}

		disk := types.VMDisk{ID: uuid.NewString(), Name: name, SizeGiB: params.Disks[i].SizeGiB}
		path := filepath.Join(e.paths.VMDiskDir(m.ID), disk.ID+".qcow2")
		if err := image.CreateVMDisk(context.Background(), "", path, disk.SizeGiB); err != nil {
			_ = os.RemoveAll(e.paths.VMDiskDir(m.ID))
			return nil, fmt.Errorf("create data disk %q: %w", name, err)
		}

		m.Disks = append(m.Disks, disk)
	}

	exposeInterfaces(m)
	if err := e.vms.Save(m); err != nil {
		_ = os.RemoveAll(e.paths.VMDiskDir(m.ID))
		return nil, err
	}

	return m, nil
}

func (e *Engine) guestCapabilities(m *types.VMManifest) []types.GuestCapability {
	if img, err := image.Lookup(m.Image); err == nil {
		return image.Provisioning(img)
	}

	return image.DefaultProvisioning()
}

func validateGuestSetup(caps []types.GuestCapability, setup types.GuestSetup) error {
	var supported *types.GuestCapability
	for i := range caps {
		if caps[i].Provisioner == setup.Provisioner {
			supported = &caps[i]
			break
		}
	}

	if supported == nil {
		return fmt.Errorf("image does not support %s provisioning", setup.Provisioner)
	}

	if setup.Hostname != "" && !guestHostname.MatchString(setup.Hostname) {
		return fmt.Errorf("hostname must be a valid RFC 1123 hostname")
	}

	switch setup.Mode {
	case types.GuestModeRaw:
		if !supported.Raw {
			return fmt.Errorf("%s does not support raw mode for this image", setup.Provisioner)
		}

		if !guestFilesHaveContent(setup) {
			return fmt.Errorf("a manual configuration is required")
		}
	default:
		return fmt.Errorf("invalid guest setup mode %q", setup.Mode)
	}

	return nil
}

func (e *Engine) GuestSetup(ref string) (*GuestSetupView, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	setup := m.EffectiveGuestSetup()
	primary := primaryInterface(m)
	files := setup.Files
	if len(files) == 0 && strings.TrimSpace(setup.Raw) != "" {
		files = []types.GuestFile{{Name: primaryFile(setup.Provisioner), Content: setup.Raw}}
	}

	return &GuestSetupView{
		Provisioner:  setup.Provisioner,
		Mode:         setup.Mode,
		Hostname:     setup.Hostname,
		Addresses:    primary.Addresses,
		Gateway:      primary.Gateway,
		Nameservers:  primary.Nameservers,
		Files:        files,
		Provisioning: e.guestCapabilities(m),
	}, nil
}

func primaryInterface(m *types.VMManifest) types.VMInterface {
	interfaces := m.EffectiveInterfaces()
	if len(interfaces) == 0 {
		return types.VMInterface{}
	}

	return interfaces[0]
}

func setPrimaryNetwork(m *types.VMManifest, addresses []string, gateway string, nameservers []string) {
	if m.Interfaces == nil || len(*m.Interfaces) == 0 {
		m.Addresses, m.Gateway, m.Nameservers = addresses, gateway, nameservers
		return
	}

	interfaces := *m.Interfaces
	interfaces[0].Addresses = addresses
	interfaces[0].Gateway = gateway
	interfaces[0].Nameservers = nameservers
	m.Interfaces = &interfaces
}

func (e *Engine) SetGuestSetup(ctx context.Context, ref string, p GuestSetupParams) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	setup := types.GuestSetup{Provisioner: p.Provisioner, Mode: p.Mode, Hostname: strings.TrimSpace(p.Hostname), Files: p.Files}
	if setup.Provisioner == "" {
		setup.Provisioner = m.EffectiveGuestSetup().Provisioner
	}

	if setup.Mode == "" {
		setup.Mode = types.GuestModeRaw
	}

	if err := validateGuestSetup(e.guestCapabilities(m), setup); err != nil {
		return err
	}

	if err := validateGuestNetwork(p.Addresses, p.Gateway, p.Nameservers); err != nil {
		return err
	}

	m.GuestSetup = &setup
	setPrimaryNetwork(m, trimAddresses(p.Addresses), strings.TrimSpace(p.Gateway), trimAddresses(p.Nameservers))
	return e.vms.Save(m)
}

func (e *Engine) ListVMs(ctx context.Context) ([]VMView, error) {
	manifests, err := e.vms.List()
	if err != nil {
		return nil, err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	cached, err := database.ListVMStates(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]VMView, 0, len(manifests))
	for _, m := range manifests {
		st := e.driver.Status(m.ID)
		boot := int64(0)
		if st.Phase == vm.PhaseRunning {
			if prev, ok := cached[m.ID]; ok && prev.Phase == string(vm.PhaseRunning) {
				boot = prev.BootTime
			}

			if boot == 0 {
				boot = time.Now().Unix()
			}
		}

		if err := database.SetVMState(ctx, types.VMState{
			ID: m.ID, Phase: string(st.Phase), PID: st.PID, BootTime: boot, SeenAt: time.Now().Unix(),
		}); err != nil {
			return nil, err
		}

		exposeInterfaces(m)
		views = append(views, VMView{Manifest: m, Phase: string(st.Phase), PID: st.PID, BootTime: boot})
	}

	return views, nil
}

func (e *Engine) StartVM(ctx context.Context, ref string) (vm.Status, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return vm.Status{}, err
	}

	if st := e.driver.Status(m.ID); st.Phase == vm.PhaseRunning {
		return st, nil
	}

	st, err := e.driver.StartPreparedContext(ctx, m.ID, func() (vm.Spec, error) {
		current, err := e.vms.Load(m.ID)
		if err != nil {
			return vm.Spec{}, err
		}

		m = current
		if err := e.checkUSBAssignments(ctx, m); err != nil {
			return vm.Spec{}, err
		}

		spec, err := e.prepareSpec(ctx, m)
		if err != nil {
			return vm.Spec{}, err
		}

		log.Ctx(ctx).Info().Msg("Starting QEMU")
		return spec, nil
	})
	if err != nil {
		return vm.Status{}, err
	}

	if err := e.attachAssignedUSB(ctx, m); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Stopping VM after USB assignment failed")
		_ = e.driver.Stop(m.ID, 5*time.Second)
		_ = e.cacheState(ctx, m.ID, vm.Status{Phase: vm.PhaseStopped}, 0)
		return vm.Status{}, err
	}

	_ = e.cacheState(ctx, m.ID, st, time.Now().Unix())
	e.captureInitialPreview(ctx, m.ID)
	return st, nil
}

func (e *Engine) StopVM(ctx context.Context, ref string, timeout time.Duration) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msg("Requesting guest shutdown")
	if err := e.driver.Stop(m.ID, timeout); err != nil {
		return err
	}

	_ = e.cacheState(ctx, m.ID, vm.Status{Phase: vm.PhaseStopped}, 0)
	return nil
}

func (e *Engine) StatusVM(ref string) (*types.VMManifest, vm.Status, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, vm.Status{}, err
	}

	exposeInterfaces(m)
	return m, e.driver.Status(m.ID), nil
}

func (e *Engine) ConsoleVM(ref string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	return e.driver.Console(m.ID)
}

func (e *Engine) DeleteVM(ctx context.Context, ref string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	m, err = e.vms.Load(m.ID)
	if err != nil {
		return err
	}

	if e.driver.Status(m.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("vm %s is running; stop it first", m.Name)
	}

	if err := os.RemoveAll(e.paths.VMDiskDir(m.ID)); err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(e.paths.VMRunDir(m.ID), "preview.png")); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := e.vms.Delete(m.ID); err != nil {
		return err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	if err := database.DeleteBackupSchedule(ctx, m.ID); err != nil {
		return err
	}

	return database.DeleteVMState(ctx, m.ID)
}

func (e *Engine) BootReconcile(ctx context.Context) error {
	var errs error
	log.Ctx(ctx).Info().Msg("Applying network configuration")
	if err := e.ApplyNetworks(false); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Applying network configuration")
		errs = errors.Join(errs, fmt.Errorf("apply networks: %w", err))
	}

	if err := e.refreshStates(ctx); err != nil {
		errs = errors.Join(errs, fmt.Errorf("refresh virtual machine states: %w", err))
	}

	return errs
}

func (e *Engine) AutostartTargets() ([]string, error) {
	manifests, err := e.vms.List()
	if err != nil {
		return nil, err
	}

	targets := []string{}
	for _, m := range manifests {
		if m.Autostart && e.driver.Status(m.ID).Phase != vm.PhaseRunning {
			targets = append(targets, m.ID)
		}
	}

	return targets, nil
}

func (e *Engine) refreshStates(ctx context.Context) error {
	manifests, err := e.vms.List()
	if err != nil {
		return err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	cached, err := database.ListVMStates(ctx)
	if err != nil {
		return err
	}

	for _, m := range manifests {
		st := e.driver.Status(m.ID)
		boot := int64(0)
		if st.Phase == vm.PhaseRunning {
			if prev, ok := cached[m.ID]; ok && prev.Phase == string(vm.PhaseRunning) {
				boot = prev.BootTime
			}

			if boot == 0 {
				boot = time.Now().Unix()
			}
		}

		if err := database.SetVMState(ctx, types.VMState{
			ID: m.ID, Phase: string(st.Phase), PID: st.PID, BootTime: boot, SeenAt: time.Now().Unix(),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Reconcile(ctx context.Context) error {
	manifests, err := e.vms.List()
	if err != nil {
		return err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	cached, err := database.ListVMStates(ctx)
	if err != nil {
		return err
	}

	for _, m := range manifests {
		st := e.driver.Status(m.ID)
		startedAt := int64(0)
		if st.Phase != vm.PhaseRunning && m.Autostart {
			st, err = e.StartVM(ctx, m.ID)
			if err != nil {
				return fmt.Errorf("autostart %s: %w", m.Name, err)
			}

			startedAt = time.Now().Unix()
		}

		boot := startedAt
		if st.Phase == vm.PhaseRunning {
			if prev, ok := cached[m.ID]; boot == 0 && ok && prev.Phase == string(vm.PhaseRunning) {
				boot = prev.BootTime
			}

			if boot == 0 {
				boot = time.Now().Unix()
			}
		}

		if err := database.SetVMState(ctx, types.VMState{
			ID: m.ID, Phase: string(st.Phase), PID: st.PID, BootTime: boot, SeenAt: time.Now().Unix(),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) cacheState(ctx context.Context, id string, st vm.Status, boot int64) error {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	return database.SetVMState(ctx, types.VMState{
		ID: id, Phase: string(st.Phase), PID: st.PID, BootTime: boot, SeenAt: time.Now().Unix(),
	})
}

func (e *Engine) prepareSpec(ctx context.Context, m *types.VMManifest) (vm.Spec, error) {
	log.Ctx(ctx).Info().Msg("Resolving VM network")
	spec := vm.Spec{Interfaces: []vm.InterfaceSpec{}}
	interfaces, err := e.normalizedInterfaces(m)
	if err == nil {
		for _, nic := range interfaces {
			network, resolveErr := e.resolveNetwork(nic.Network)
			if resolveErr != nil {
				err = resolveErr
				break
			}

			network.ID = nic.ID
			network.MAC = nic.MAC
			spec.Interfaces = append(spec.Interfaces, network)
		}
	}

	if err != nil {
		return vm.Spec{}, err
	}

	fw, err := firmware.EnsureContext(ctx, e.paths.FirmwareDir())
	if err != nil {
		return vm.Spec{}, err
	}

	diskPath, err := e.prepareVMDisk(ctx, m)
	if err != nil {
		return vm.Spec{}, err
	}

	setup := m.EffectiveGuestSetup()
	hostname := setup.Hostname
	if hostname == "" {
		hostname = m.Name
	}

	seedPath := ""
	ignitionPath := ""
	manual := setup.Mode == types.GuestModeRaw || len(setup.Files) > 0
	if setup.Provisioner != types.ProvisionerNone && (m.Image != "" || manual) {
		prov := provision.Get(setup.Provisioner)
		if prov == nil {
			return vm.Spec{}, fmt.Errorf("unknown provisioner %q", setup.Provisioner)
		}

		setup, err = renderGuestSetup(setup, guestDataFromManifest(m, hostname, interfaces))
		if err != nil {
			return vm.Spec{}, err
		}

		primary := types.VMInterface{}
		if len(interfaces) > 0 {
			primary = interfaces[0]
		}

		params := provision.Params{
			Hostname:      hostname,
			MAC:           primary.MAC,
			Addresses:     primary.Addresses,
			Gateway:       primary.Gateway,
			Nameservers:   primary.Nameservers,
			NetworkConfig: guestInterfacesConfig(interfaces),
		}
		files, err := guestDeliveryFiles(prov, setup, params)
		if err != nil {
			return vm.Spec{}, err
		}

		log.Ctx(ctx).Info().Str("provisioner", setup.Provisioner).Msg("Preparing guest provisioning")
		delivery, err := prov.Deliver(e.paths.VMDiskDir(m.ID), files)
		if err != nil {
			return vm.Spec{}, err
		}

		seedPath = delivery.SeedPath
		ignitionPath = delivery.IgnitionPath
	} else if setup.Provisioner == types.ProvisionerNone {
		log.Ctx(ctx).Info().Msg("Guest provisioning disabled; no seed or Ignition config")
	}

	for _, disk := range m.Disks {
		if _, err := uuid.Parse(disk.ID); err != nil {
			return vm.Spec{}, fmt.Errorf("invalid data disk ID: %w", err)
		}

		path := filepath.Join(e.paths.VMDiskDir(m.ID), disk.ID+".qcow2")
		if _, err := os.Stat(path); err != nil {
			return vm.Spec{}, fmt.Errorf("data disk %s unavailable: %w", disk.Name, err)
		}

		spec.Disks = append(spec.Disks, vm.DiskSpec{ID: disk.ID, Path: path})
	}

	for _, id := range m.ISOs {
		media, err := e.GetMedia(id)
		if err != nil {
			return vm.Spec{}, err
		}

		spec.ISOs = append(spec.ISOs, vm.DiskSpec{ID: id, Path: media.Path})
	}

	spec.BootOrder = m.EffectiveBootOrder()
	spec.ID, spec.Name = m.ID, m.Name
	spec.CPUs, spec.MemoryMiB = m.CPUs, m.MemoryMiB
	spec.DiskPath, spec.SeedPath, spec.Firmware = diskPath, seedPath, fw
	spec.IgnitionPath = ignitionPath
	return spec, nil
}

func (e *Engine) resolveNetwork(ref string) (vm.InterfaceSpec, error) {
	switch ref {
	case "", "user":
		return vm.InterfaceSpec{Network: vm.NetworkUser}, nil
	}

	n, err := e.nets.Resolve(ref)
	if err != nil {
		return vm.InterfaceSpec{}, fmt.Errorf("resolve network %q: %w", ref, err)
	}

	switch n.Mode {
	case types.NetworkBridge:
		if err := vnet.Ensure(e.nets, n, false); err != nil {
			return vm.InterfaceSpec{}, err
		}

		return vm.InterfaceSpec{Network: vm.NetworkBridge, Bridge: n.Device}, nil
	case types.NetworkSwitch:
		return vm.InterfaceSpec{Network: vm.NetworkSwitch, Group: n.Group}, nil
	case types.NetworkUser:
		return vm.InterfaceSpec{Network: vm.NetworkUser}, nil
	case types.NetworkBridged, types.NetworkVmnetBridged:
		return vm.InterfaceSpec{Network: vm.NetworkVmnetBridged, Uplink: n.Uplink}, nil
	default:
		return vm.InterfaceSpec{}, fmt.Errorf("unknown network mode %q", n.Mode)
	}
}

func guestNetworkConfig(mac string, addresses, nameservers []string, gateway string) string {
	config := "version: 2\nrenderer: networkd\nethernets:\n  lab:\n"
	if mac != "" {
		config += "    match:\n      macaddress: " + mac + "\n    set-name: lab0\n"
	}

	config += "    optional: true\n"
	if len(addresses) == 0 {
		config += "    dhcp4: true\n"
		if len(nameservers) > 0 {
			config += "    nameservers:\n      addresses:\n"
			for _, ns := range nameservers {
				config += "        - " + ns + "\n"
			}
		}

		return config
	}

	config += "    dhcp4: false\n    addresses:\n"
	for _, address := range addresses {
		config += "      - " + address + "\n"
	}

	if gateway != "" {
		config += "    routes:\n      - to: default\n        via: " + gateway + "\n"
	}

	if len(nameservers) > 0 {
		config += "    nameservers:\n      addresses:\n"
		for _, ns := range nameservers {
			config += "        - " + ns + "\n"
		}
	}

	return config
}

func (e *Engine) ScreenshotVM(ref string) error {
	manifest, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	return e.driver.Screenshot(manifest.ID)
}

func (e *Engine) ShutdownVM(ref string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	return e.driver.Shutdown(m.ID)
}

func (e *Engine) RebootVM(ref string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	return e.driver.Reboot(m.ID)
}

func (e *Engine) ForceStopVM(ctx context.Context, ref string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	if err := e.driver.ForceStop(m.ID); err != nil {
		return err
	}

	_ = e.cacheState(ctx, m.ID, vm.Status{Phase: vm.PhaseStopped}, 0)
	return nil
}

func (e *Engine) prepareVMDisk(ctx context.Context, m *types.VMManifest) (string, error) {
	if m.DiskSizeGiB <= 0 {
		return "", nil
	}

	diskPath := filepath.Join(e.paths.VMDiskDir(m.ID), "disk.qcow2")
	if _, err := os.Stat(diskPath); err == nil {
		return diskPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	base := ""
	if m.Image != "" {
		var err error
		base, err = e.imageBase(ctx, m.Image, true)
		if err != nil {
			return "", err
		}
	}

	if err := image.CreateVMDisk(ctx, base, diskPath, m.DiskSizeGiB); err != nil {
		return "", err
	}

	return diskPath, nil
}
