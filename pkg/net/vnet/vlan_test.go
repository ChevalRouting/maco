package vnet

import (
	"errors"
	"testing"
)

type vlanCase struct {
	name, device, matching        string
	owned, exists, configureFails bool
	want                          string
	wantOwned                     bool
	creates, destroys             int
}

type fakeVLANHost struct {
	t                 *testing.T
	tc                vlanCase
	creates, destroys int
}

func (f *fakeVLANHost) exists(string) bool { return f.tc.exists }

func (f *fakeVLANHost) find(parent string, tag int) (string, error) {
	if parent != "en10" || tag != 123 {
		f.t.Fatal("wrong lookup")
	}

	return f.tc.matching, nil
}

func (f *fakeVLANHost) create(string) (string, error) { f.creates++; return "vlan9", nil }

func (f *fakeVLANHost) named(_, name string) (string, error) { f.creates++; return name, nil }

func (f *fakeVLANHost) configure(string, string, int) error {
	if f.tc.configureFails {
		return errors.New("File exists")
	}

	return nil
}

func (f *fakeVLANHost) destroy(string) error { f.destroys++; return nil }

func checkResolveVLAN(t *testing.T, tc vlanCase) {
	fake := &fakeVLANHost{t: t, tc: tc}
	host := vlanHost{exists: fake.exists, find: fake.find, create: fake.create, named: fake.named, configure: fake.configure, destroy: fake.destroy}
	device, owned, err := resolveVLANWith(host, tc.device, tc.owned, "en10", 123)
	if err != nil || device != tc.want || owned != tc.wantOwned || fake.creates != tc.creates || fake.destroys != tc.destroys {
		t.Fatalf("got %s owned=%v creates=%d destroys=%d err=%v", device, owned, fake.creates, fake.destroys, err)
	}
}

func TestResolveVLANReuseAndOwnership(t *testing.T) {
	for _, tc := range []vlanCase{
		{name: "borrow existing", matching: "vlan7", want: "vlan7"},
		{name: "retain owned", device: "vlan7", owned: true, exists: true, want: "vlan7", wantOwned: true},
		{name: "retain borrowed", device: "vlan7", exists: true, want: "vlan7"},
		{name: "missing device reuse", device: "vlan2", owned: true, matching: "vlan7", want: "vlan7"},
		{name: "recover failed clone", device: "vlan2", owned: true, exists: true, configureFails: true, matching: "vlan7", want: "vlan7"},
		{name: "create", want: "vlan9", wantOwned: true, creates: 1},
		{name: "recreate owned under its name", device: "vlan2", owned: true, want: "vlan2", wantOwned: true, creates: 1},
	} {
		t.Run(tc.name, func(t *testing.T) { checkResolveVLAN(t, tc) })
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
	destroyed := ""
	host := vlanHost{
		exists:    func(string) bool { return false },
		find:      (&racingVLANFind{}).find,
		create:    func(string) (string, error) { return "vlan9", nil },
		configure: func(string, string, int) error { return errors.New("File exists") },
		destroy:   func(device string) error { destroyed = device; return nil },
	}
	device, owned, err := resolveVLANWith(host, "", false, "en10", 123)
	if err != nil || device != "vlan7" || owned || destroyed != "vlan9" {
		t.Fatalf("device=%s owned=%v cleanup=%s err=%v", device, owned, destroyed, err)
	}
}

type racingVLANFind struct {
	finds int
}

func (r *racingVLANFind) find(string, int) (string, error) {
	r.finds++
	if r.finds == 1 {
		return "", nil
	}

	return "vlan7", nil
}
