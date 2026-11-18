package cloudinit

import (
	"gopkg.in/yaml.v3"
)

type Seed struct {
	Hostname string
}

type instanceMetadata struct {
	InstanceID string `yaml:"instance-id"`
	Hostname   string `yaml:"local-hostname"`
}

func (s Seed) metaData() string {
	data, _ := yaml.Marshal(instanceMetadata{s.Hostname, s.Hostname})
	return string(data)
}
