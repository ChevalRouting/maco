package cloudinit

import (
	"gopkg.in/yaml.v3"
)

type Seed struct {
	Hostname string
}

func (s Seed) metaData() string {
	data, _ := yaml.Marshal(struct {
		InstanceID string `yaml:"instance-id"`
		Hostname   string `yaml:"local-hostname"`
	}{s.Hostname, s.Hostname})
	return string(data)
}
