package types

type VMDisk struct {
	ID      string `yaml:"id" json:"id" validate:"required,uuid"`
	Name    string `yaml:"name" json:"name" validate:"required"`
	SizeGiB int    `yaml:"size_gib" json:"size_gib" validate:"min=1"`
}

type VMUSBAssignment struct {
	VendorID  uint16 `yaml:"vendor_id" json:"vendor_id" validate:"required"`
	ProductID uint16 `yaml:"product_id" json:"product_id" validate:"required"`
	Serial    string `yaml:"serial,omitempty" json:"serial,omitempty" binding:"optional"`
	Product   string `yaml:"product,omitempty" json:"product,omitempty" binding:"optional"`
}

type VMInterface struct {
	ID          string   `yaml:"id" json:"id" validate:"required"`
	Network     string   `yaml:"network" json:"network"`
	MAC         string   `yaml:"mac,omitempty" json:"mac,omitempty" validate:"omitempty,mac" binding:"optional"`
	Addresses   []string `yaml:"addresses,omitempty" json:"addresses,omitempty" validate:"dive,cidr" binding:"optional"`
	Gateway     string   `yaml:"gateway,omitempty" json:"gateway,omitempty" validate:"omitempty,ip" binding:"optional"`
	Nameservers []string `yaml:"nameservers,omitempty" json:"nameservers,omitempty" validate:"dive,ip" binding:"optional"`
}

const (
	ProvisionerNone      = "none"
	ProvisionerCloudInit = "cloud-init"
	ProvisionerIgnition  = "ignition"
	GuestModeRaw         = "raw"
)

// @Description Guest provisioning configuration. Discover supported provisioners and modes from listCatalog before creating a VM.
type GuestSetup struct {
	Provisioner string      `yaml:"provisioner,omitempty" json:"provisioner,omitempty" binding:"optional"`
	Mode        string      `yaml:"mode,omitempty" json:"mode,omitempty" binding:"optional"`
	Hostname    string      `yaml:"hostname,omitempty" json:"hostname,omitempty" validate:"omitempty,hostname_rfc1123" binding:"optional"`
	Raw         string      `yaml:"raw,omitempty" json:"raw,omitempty" binding:"optional"`
	Files       []GuestFile `yaml:"files,omitempty" json:"files,omitempty" validate:"dive" binding:"optional"`
}

type GuestFile struct {
	Name    string `yaml:"name" json:"name" validate:"required"`
	Content string `yaml:"content" json:"content" binding:"optional"`
}

type GuestCapability struct {
	Provisioner string `json:"provisioner"`
	Raw         bool   `json:"raw"`
}

type VMManifest struct {
	Interfaces  *[]VMInterface    `yaml:"interfaces,omitempty" json:"interfaces,omitempty" validate:"omitempty,dive" binding:"optional"`
	ISOs        []string          `yaml:"isos,omitempty" json:"isos,omitempty" binding:"optional"`
	BootOrder   []string          `yaml:"boot_order,omitempty" json:"boot_order,omitempty" binding:"optional"`
	Disks       []VMDisk          `yaml:"disks,omitempty" json:"disks,omitempty" validate:"dive" binding:"optional"`
	USB         []VMUSBAssignment `yaml:"usb,omitempty" json:"usb,omitempty" validate:"dive" binding:"optional"`
	GuestSetup  *GuestSetup       `yaml:"guest_setup,omitempty" json:"guest_setup,omitempty" binding:"optional"`
	ID          string            `yaml:"id" json:"id" validate:"required"`
	Name        string            `yaml:"name" json:"name" validate:"required,hostname_rfc1123"`
	Image       string            `yaml:"image,omitempty" json:"image"`
	CPUs        int               `yaml:"cpus" json:"cpus" validate:"min=1"`
	MemoryMiB   int               `yaml:"memory_mib" json:"memory_mib" validate:"min=64"`
	DiskSizeGiB int               `yaml:"disk_size_gib" json:"disk_size_gib" validate:"min=1"`
	Network     string            `yaml:"network,omitempty" json:"network,omitempty" binding:"optional"`
	Addresses   []string          `yaml:"addresses,omitempty" json:"addresses,omitempty" validate:"dive,cidr" binding:"optional"`
	Gateway     string            `yaml:"gateway,omitempty" json:"gateway,omitempty" validate:"omitempty,ip" binding:"optional"`
	Nameservers []string          `yaml:"nameservers,omitempty" json:"nameservers,omitempty" validate:"dive,ip" binding:"optional"`
	SSHKeys     []string          `yaml:"ssh_keys,omitempty" json:"ssh_keys,omitempty" binding:"optional"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty" binding:"optional"`
	Autostart   bool              `yaml:"autostart,omitempty" json:"autostart,omitempty" binding:"optional"`
}

type BackupSchedule struct {
	VMID          string `json:"vm_id"`
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"interval_hours"`
	KeepLast      int    `json:"keep_last"`
	MaxAgeDays    int    `json:"max_age_days"`
	LastRunAt     int64  `json:"last_run_at"`
}

type VMState struct {
	ID       string
	Phase    string
	PID      int
	BootTime int64
	SeenAt   int64
}

func (m *VMManifest) EffectiveGuestSetup() GuestSetup {
	setup := GuestSetup{}
	if m.GuestSetup != nil {
		setup = *m.GuestSetup
	}
	if setup.Provisioner == "" {
		setup.Provisioner = ProvisionerCloudInit
	}
	if setup.Mode == "" {
		setup.Mode = GuestModeRaw
	}
	return setup
}

func (m *VMManifest) EffectiveBootOrder() []string {
	devices := []string{"disk"}
	for _, disk := range m.Disks {
		devices = append(devices, "disk:"+disk.ID)
	}
	for _, id := range m.ISOs {
		devices = append(devices, "iso:"+id)
	}
	preferred := m.BootOrder
	if len(preferred) == 0 {
		preferred = []string{}
		for _, id := range m.ISOs {
			preferred = append(preferred, "iso:"+id)
		}
		preferred = append(preferred, "disk")
	}
	valid := map[string]bool{}
	for _, id := range devices {
		valid[id] = true
	}
	seen := map[string]bool{}
	result := []string{}
	for _, id := range append(append([]string{}, preferred...), devices...) {
		if valid[id] && !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result
}

func (m *VMManifest) EffectiveInterfaces() []VMInterface {
	if m.Interfaces != nil {
		return *m.Interfaces
	}
	return []VMInterface{{ID: "net0", Network: m.Network, Addresses: m.Addresses, Gateway: m.Gateway, Nameservers: m.Nameservers}}
}
