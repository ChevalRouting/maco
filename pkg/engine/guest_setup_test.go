package engine

import (
	"testing"

	"github.com/m-vinc/maco/pkg/types"
)

func TestValidateGuestSetup(t *testing.T) {
	cloudInit := []types.GuestCapability{{Provisioner: types.ProvisionerCloudInit, Raw: true}}
	ignition := []types.GuestCapability{{Provisioner: types.ProvisionerIgnition, Raw: true}}

	cases := []struct {
		name  string
		caps  []types.GuestCapability
		setup types.GuestSetup
		ok    bool
	}{
		{"ignition not supported by cloud-init image", cloudInit, types.GuestSetup{Provisioner: types.ProvisionerIgnition, Mode: types.GuestModeRaw, Raw: "variant: flatcar"}, false},
		{"raw required in raw mode", ignition, types.GuestSetup{Provisioner: types.ProvisionerIgnition, Mode: types.GuestModeRaw}, false},
		{"raw provided", ignition, types.GuestSetup{Provisioner: types.ProvisionerIgnition, Mode: types.GuestModeRaw, Raw: "variant: flatcar"}, true},
		{"cloud-init raw provided", cloudInit, types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: types.GuestModeRaw, Raw: "#cloud-config"}, true},
		{"invalid mode", cloudInit, types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: "weird"}, false},
		{"valid hostname", cloudInit, types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: types.GuestModeRaw, Raw: "#cloud-config", Hostname: "web-01"}, true},
		{"invalid hostname", cloudInit, types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: types.GuestModeRaw, Raw: "#cloud-config", Hostname: "bad host"}, false},
	}

	for _, c := range cases {
		err := validateGuestSetup(c.caps, c.setup)
		if c.ok && err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
		}

		if !c.ok && err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
	}
}
