package types

type NetworkMode string

const (
	NetworkUser         NetworkMode = "user"
	NetworkVmnetBridged NetworkMode = "vmnet-bridged"
	NetworkBridge       NetworkMode = "bridge"
	NetworkSwitch       NetworkMode = "switch"
	NetworkBridged      NetworkMode = "bridged"
	NetworkVLAN         NetworkMode = "vlan"
)

type VLAN struct {
	Borrowed bool   `yaml:"borrowed,omitempty" json:"borrowed,omitempty" binding:"optional"`
	Parent   string `yaml:"parent" json:"parent" validate:"required"`
	Tag      int    `yaml:"tag" json:"tag" validate:"min=1,max=4094"`
	Device   string `yaml:"device,omitempty" json:"device,omitempty" binding:"optional"`
}

type NetworkManifest struct {
	ID             string      `yaml:"id" json:"id" validate:"required,uuid"`
	Name           string      `yaml:"name" json:"name" validate:"required,hostname_rfc1123"`
	Mode           NetworkMode `yaml:"mode" json:"mode" validate:"oneof=bridge switch bridged vmnet-bridged user vlan"`
	Uplink         string      `yaml:"uplink,omitempty" json:"uplink,omitempty" binding:"optional"`
	Parent         string      `yaml:"parent,omitempty" json:"parent,omitempty" binding:"optional"`
	Tag            int         `yaml:"tag,omitempty" json:"tag,omitempty" binding:"optional"`
	Group          string      `yaml:"group,omitempty" json:"group,omitempty" binding:"optional"`
	Address        string      `yaml:"address,omitempty" json:"address,omitempty" validate:"omitempty,cidr" binding:"optional"`
	Device         string      `yaml:"device,omitempty" json:"device,omitempty" binding:"optional"`
	Members        []string    `yaml:"members,omitempty" json:"members,omitempty" binding:"optional"`
	VLANs          []VLAN      `yaml:"vlans,omitempty" json:"vlans,omitempty" validate:"dive" binding:"optional"`
	Owned          bool        `yaml:"owned,omitempty" json:"owned,omitempty" binding:"optional"`
	AppliedVLANs   []VLAN      `yaml:"applied_vlans,omitempty" json:"applied_vlans,omitempty" validate:"dive" binding:"optional"`
	AppliedAddress string      `yaml:"applied_address,omitempty" json:"applied_address,omitempty" validate:"omitempty,cidr" binding:"optional"`
	AppliedMembers []string    `yaml:"applied_members,omitempty" json:"applied_members,omitempty" binding:"optional"`
}
