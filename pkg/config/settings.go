package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/m-vinc/maco/pkg/storage"
	"gopkg.in/yaml.v3"
)

type Settings struct {
	BaseMAC string      `yaml:"base_mac,omitempty"`
	TLS     TLSSettings `yaml:"tls,omitempty"`
}

type TLSSettings struct {
	Cert string        `yaml:"cert,omitempty"`
	Key  string        `yaml:"key,omitempty"`
	ACME *ACMESettings `yaml:"acme,omitempty"`
}

type ACMESettings struct {
	Directory string   `yaml:"directory,omitempty"`
	Email     string   `yaml:"email,omitempty"`
	Domains   []string `yaml:"domains"`
	HTTPAddr  string   `yaml:"http_addr,omitempty"`
	CA        string   `yaml:"ca,omitempty"`
	EABKID    string   `yaml:"eab_kid,omitempty"`
	EABKey    string   `yaml:"eab_key,omitempty"`
}

func (p *Paths) ConfigPath() string { return filepath.Join(p.Root, "config.yaml") }

func LoadSettings(p *Paths) (Settings, error) {
	data, err := os.ReadFile(p.ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, nil
	}

	if err != nil {
		return Settings{}, err
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func SaveTLSSettings(p *Paths, tls TLSSettings) error {
	settings, err := LoadSettings(p)
	if err != nil {
		return err
	}

	settings.TLS = tls
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	return storage.WriteFile(p.ConfigPath(), data, 0o600)
}

func (p *Paths) TLSDir() string { return filepath.Join(p.Root, "tls") }
