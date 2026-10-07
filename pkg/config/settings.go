package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	BaseMAC string `yaml:"base_mac"`
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
