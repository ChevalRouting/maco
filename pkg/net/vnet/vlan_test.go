package vnet

import (
	"errors"
	"testing"
)

func TestResolveVLANReuseAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, device, matching        string
		owned, exists, configureFails bool
		want                          string
		wantOwned                     bool
		creates, destroys             int
	}{
		{name: "borrow existing", matching: "vlan7", want: "vlan7"},
		{name: "retain owned", device: "vlan7", owned: true, exists: true, want: "vlan7", wantOwned: true},
		{name: "retain borrowed", device: "vlan7", exists: true, want: "vlan7"},
		{name: "missing device reuse", device: "vlan2", owned: true, matching: "vlan7", want: "vlan7"},
		{name: "recover failed clone", device: "vlan2", owned: true, exists: true, configureFails: true, matching: "vlan7", want: "vlan7"},
		{name: "create", want: "vlan9", wantOwned: true, creates: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creates, destroys := 0, 0
			host := vlanHost{
				exists: func(string) bool { return tc.exists },
				find: func(parent string, tag int) (string, error) {
					if parent != "en10" || tag != 123 {
						t.Fatal("wrong lookup")
					}

					return tc.matching, nil
				},
				create: func(string) (string, error) { creates++; return "vlan9", nil },
				configure: func(string, string, int) error {
					if tc.configureFails {
						return errors.New("File exists")
					}

					return nil
				},
				destroy: func(string) error { destroys++; return nil },
			}
			device, owned, err := resolveVLANWith(host, tc.device, tc.owned, "en10", 123)
			if err != nil || device != tc.want || owned != tc.wantOwned || creates != tc.creates || destroys != tc.destroys {
				t.Fatalf("got %s owned=%v creates=%d destroys=%d err=%v", device, owned, creates, destroys, err)
			}
		})
	}
}

func TestResolveVLANConfigurationFailureCleanup(t *testing.T) {
	failure := errors.New("invalid VLAN parent")
	destroyed := ""
	host := vlanHost{
		exists:    func(string) bool { return false },
		find:      func(string, int) (string, error) { return "", nil },
		create:    func(string) (string, error) { return "vlan9", nil },
		configure: func(string, string, int) error { return failure },
		destroy:   func(device string) error { destroyed = device; return nil },
	}
	if _, _, err := resolveVLANWith(host, "", false, "en10", 123); !errors.Is(err, failure) || destroyed != "vlan9" {
		t.Fatalf("cleanup=%s, err=%v", destroyed, err)
	}
}

func TestResolveVLANConcurrentCreation(t *testing.T) {
	finds := 0
	destroyed := ""
	host := vlanHost{
		exists: func(string) bool { return false },
		find: func(string, int) (string, error) {
			finds++
			if finds == 1 {
				return "", nil
			}

			return "vlan7", nil
		},
		create:    func(string) (string, error) { return "vlan9", nil },
		configure: func(string, string, int) error { return errors.New("File exists") },
		destroy:   func(device string) error { destroyed = device; return nil },
	}
	device, owned, err := resolveVLANWith(host, "", false, "en10", 123)
	if err != nil || device != "vlan7" || owned || destroyed != "vlan9" {
		t.Fatalf("device=%s owned=%v cleanup=%s err=%v", device, owned, destroyed, err)
	}
}
