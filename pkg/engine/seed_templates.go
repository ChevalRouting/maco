package engine

import (
	"context"
	"os"

	"github.com/m-vinc/maco/pkg/types"
	"github.com/rs/zerolog/log"
)

type defaultTemplate struct {
	Name        string
	Description string
	Spec        CreateVMParams
}

func (e *Engine) FirstRun() bool {
	_, err := os.Stat(e.paths.InitMarkerPath())
	return os.IsNotExist(err)
}

func (e *Engine) markInitialized() error {
	return os.WriteFile(e.paths.InitMarkerPath(), []byte("initialized\n"), 0o600)
}

func (e *Engine) SeedDefaultTemplates(ctx context.Context) (int, error) {
	if !e.FirstRun() {
		return 0, nil
	}
	existing, err := e.ListTemplates()
	if err != nil {
		return 0, err
	}
	have := map[string]bool{}
	for _, t := range existing {
		have[t.Name] = true
	}
	seeded := 0
	for _, d := range defaultTemplates() {
		if have[d.Name] {
			continue
		}
		if _, err := e.CreateTemplate(ctx, d.Name, d.Description, d.Spec); err != nil {
			return seeded, err
		}
		seeded++
	}
	if err := e.markInitialized(); err != nil {
		return seeded, err
	}
	log.Ctx(ctx).Info().Int("count", seeded).Msg("seeded default templates on first run")
	return seeded, nil
}

func defaultTemplates() []defaultTemplate {
	return []defaultTemplate{
		{
			Name:        "ubuntu-cloud-init",
			Description: "Ubuntu 24.04 starter (cloud-init): maco user with a dummy password and profile SSH keys, qemu-guest-agent, hostname from the VM name.",
			Spec: CreateVMParams{
				Image:       "ubuntu-24.04-arm64",
				CPUs:        2,
				MemoryMiB:   2048,
				DiskSizeGiB: 20,
				Network:     "user",
				GuestSetup:  &types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: types.GuestModeRaw, Raw: ubuntuCloudInitDefault},
			},
		},
		{
			Name:        "flatcar-podman",
			Description: "Flatcar starter (Ignition): core user with password and profile SSH keys, podman enabled, hostname from the VM name.",
			Spec: CreateVMParams{
				Image:       "flatcar-stable-arm64",
				CPUs:        2,
				MemoryMiB:   2048,
				DiskSizeGiB: 20,
				Network:     "user",
				GuestSetup:  &types.GuestSetup{Provisioner: types.ProvisionerIgnition, Mode: types.GuestModeRaw, Raw: flatcarPodmanDefault},
			},
		},
	}
}

const ubuntuCloudInitDefault = `#cloud-config
hostname: [[ .Name ]]
users:
  - name: maco
    groups: [sudo]
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    shell: /bin/bash
    lock_passwd: false
    ssh_authorized_keys:
[[- range .SSHKeys ]]
      - [[ . ]]
[[- end ]]
# Change this password, or drop ssh_pwauth + chpasswd to disable password login.
ssh_pwauth: true
chpasswd:
  expire: false
  users:
    - name: maco
      password: changeme
      type: text
package_update: true
packages:
  - qemu-guest-agent
runcmd:
  - [systemctl, enable, --now, qemu-guest-agent]
`

const flatcarPodmanDefault = `variant: flatcar
version: 1.1.0
passwd:
  users:
    - name: core
      groups:
        - sudo
      password_hash: "$6$MA/WL4wmgEPGBDqS$lxybEkr7.26X8TR9hDzl6CrC9UhvvZTqPJL/qvozkIFxbFSg7UMg440v8L0taCSuIJlCQyxB9ln2sIGVPO6dQ1"
      ssh_authorized_keys:
[[- range .SSHKeys ]]
        - [[ . ]]
[[- end ]]
storage:
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: |
          [[ .Name ]]
    - path: /etc/flatcar/enabled-sysext.conf
      mode: 0644
      contents:
        inline: |
          podman
    - path: /etc/systemd/network/10-maco-dhcp.network
      mode: 0644
      contents:
        inline: |
          [Match]
          Name=en* eth*

          [Network]
          DHCP=yes
          IPv6AcceptRA=yes

          [DHCP]
          UseMTU=yes

          [DHCPv6]
          WithoutRA=solicit
systemd:
  units:
    - name: podman.socket
      enabled: true
`
