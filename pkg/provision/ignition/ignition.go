package ignition

import (
	"fmt"

	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Hostname    string
	Raw         string
	MAC         string
	Addresses   []string
	Gateway     string
	Nameservers []string
}

type butaneUser struct {
	Name              string   `yaml:"name"`
	SSHAuthorizedKeys []string `yaml:"ssh_authorized_keys,omitempty"`
	PasswordHash      string   `yaml:"password_hash,omitempty"`
	Groups            []string `yaml:"groups,omitempty"`
}

type butanePasswd struct {
	Users []butaneUser `yaml:"users,omitempty"`
}

type butaneContents struct {
	Inline string `yaml:"inline,omitempty"`
	Source string `yaml:"source,omitempty"`
}

type butaneFile struct {
	Path     string         `yaml:"path"`
	Mode     int            `yaml:"mode,omitempty"`
	Contents butaneContents `yaml:"contents"`
}

type butaneLink struct {
	Path      string `yaml:"path"`
	Target    string `yaml:"target"`
	Overwrite *bool  `yaml:"overwrite,omitempty"`
}

type butaneStorage struct {
	Files []butaneFile `yaml:"files,omitempty"`
	Links []butaneLink `yaml:"links,omitempty"`
}

type butaneUnit struct {
	Name     string `yaml:"name"`
	Enabled  *bool  `yaml:"enabled,omitempty"`
	Contents string `yaml:"contents,omitempty"`
}

type butaneSystemd struct {
	Units []butaneUnit `yaml:"units,omitempty"`
}

type butaneDoc struct {
	Variant string         `yaml:"variant"`
	Version string         `yaml:"version"`
	Passwd  *butanePasswd  `yaml:"passwd,omitempty"`
	Storage *butaneStorage `yaml:"storage,omitempty"`
	Systemd *butaneSystemd `yaml:"systemd,omitempty"`
}

const enabledSysextPath = "/etc/flatcar/enabled-sysext.conf"

const enabledSysext = "podman\npython\n"

const policyPath = "/etc/containers/policy.json"

const policySource = "https://raw.githubusercontent.com/containers/podman/main/test/policy.json"

var maskedExtensions = []string{
	"/etc/extensions/docker-flatcar.raw",
	"/etc/extensions/containerd-flatcar.raw",
}

func (c Config) easyButane() ([]byte, error) {
	enabled := true
	doc := butaneDoc{
		Variant: "flatcar",
		Version: "1.1.0",
		Systemd: &butaneSystemd{Units: []butaneUnit{{Name: "podman.socket", Enabled: &enabled}}},
	}

	files := []butaneFile{
		{Path: enabledSysextPath, Mode: 0o644, Contents: butaneContents{Inline: enabledSysext}},
		{Path: policyPath, Contents: butaneContents{Source: policySource}},
	}
	if c.Hostname != "" {
		files = append(files, butaneFile{Path: "/etc/hostname", Mode: 0o644, Contents: butaneContents{Inline: c.Hostname + "\n"}})
	}

	if network := c.networkUnit(); network != "" {
		files = append(files, butaneFile{Path: "/etc/systemd/network/00-maco.network", Mode: 0o644, Contents: butaneContents{Inline: network}})
	}

	overwrite := true
	links := make([]butaneLink, 0, len(maskedExtensions))
	for _, path := range maskedExtensions {
		links = append(links, butaneLink{Path: path, Target: "/dev/null", Overwrite: &overwrite})
	}

	doc.Storage = &butaneStorage{Files: files, Links: links}

	return yaml.Marshal(doc)
}

func (c Config) networkUnit() string {
	if len(c.Addresses) == 0 || c.MAC == "" {
		return ""
	}

	unit := "[Match]\nMACAddress=" + c.MAC + "\n\n[Network]\n"
	for _, address := range c.Addresses {
		unit += "Address=" + address + "\n"
	}

	if c.Gateway != "" {
		unit += "Gateway=" + c.Gateway + "\n"
	}

	for _, ns := range c.Nameservers {
		unit += "DNS=" + ns + "\n"
	}

	return unit
}

func transpile(butaneYAML []byte) ([]byte, error) {
	ign, report, err := config.TranslateBytes(butaneYAML, common.TranslateBytesOptions{Raw: true, Pretty: true})
	if err != nil {
		return nil, fmt.Errorf("butane: %w", err)
	}

	if report.IsFatal() {
		return nil, fmt.Errorf("butane: %s", report.String())
	}

	return ign, nil
}
