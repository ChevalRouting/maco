//go:build darwin

package host

import "testing"

func TestVLANMatches(t *testing.T) {
	out := "vlan7: flags=8843<UP,BROADCAST> mtu 1500\n\tvlan: 123 parent interface: en10\n"
	if !vlanMatches(out, "en10", 123) || vlanMatches(out, "en1", 123) || vlanMatches(out, "en10", 12) || vlanMatches("vlan9: flags=0\n", "en10", 123) {
		t.Fatal("incorrect VLAN match")
	}
}
