package vnet

import "github.com/m-vinc/maco/pkg/net/host"

type vlanHost struct {
	exists    func(string) bool
	find      func(string, int) (string, error)
	create    func(string) (string, error)
	configure func(string, string, int) error
	destroy   func(string) error
}

func resolveVLAN(device string, owned bool, parent string, tag int) (string, bool, error) {
	return resolveVLANWith(vlanHost{host.Exists, host.FindVLAN, host.Create, host.ConfigureVLAN, host.Destroy}, device, owned, parent, tag)
}

func resolveVLANWith(host vlanHost, device string, owned bool, parent string, tag int) (string, bool, error) {
	if device != "" && host.exists(device) {
		if err := host.configure(device, parent, tag); err == nil {
			return device, owned, nil
		} else {
			matching, findErr := host.find(parent, tag)
			if findErr != nil {
				return "", false, findErr
			}
			if matching != "" && matching != device {
				return matching, false, nil
			}
			return "", false, err
		}
	}
	matching, err := host.find(parent, tag)
	if err != nil {
		return "", false, err
	}
	if matching != "" {
		return matching, false, nil
	}
	device, err = host.create("vlan")
	if err != nil {
		return "", false, err
	}
	if err := host.configure(device, parent, tag); err != nil {
		_ = host.destroy(device)
		matching, findErr := host.find(parent, tag)
		if findErr != nil {
			return "", false, findErr
		}
		if matching != "" {
			return matching, false, nil
		}
		return "", false, err
	}
	return device, true, nil
}
