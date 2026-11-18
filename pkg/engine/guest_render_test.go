package engine

import (
	"strings"
	"testing"

	"github.com/m-vinc/maco/pkg/types"
)

func TestRenderGuestSetup(t *testing.T) {
	data := guestTemplateData{
		Name:        "web-01",
		Hostname:    "web-01",
		CPUs:        4,
		MemoryMiB:   4096,
		DiskSizeGiB: 30,
		Disks:       []guestTemplateDisk{{Name: "data", SizeGiB: 10}},
	}

	setup := types.GuestSetup{
		Provisioner: types.ProvisionerIgnition,
		Mode:        types.GuestModeRaw,
		Raw:         "hostname: [[ .Name ]]\ncpus: [[ .CPUs ]]\nmem: [[ .MemoryMiB ]]\n[[ range .Disks ]]disk: [[ .Name ]] [[ .SizeGiB ]]G\n[[ end ]]",
		Files:       []types.GuestFile{{Name: "extra", Content: "host=[[ .Hostname ]]"}},
	}

	out, err := renderGuestSetup(setup, data)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"hostname: web-01", "cpus: 4", "mem: 4096", "disk: data 10G"} {
		if !strings.Contains(out.Raw, want) {
			t.Fatalf("raw missing %q:\n%s", want, out.Raw)
		}
	}

	if out.Files[0].Content != "host=web-01" {
		t.Fatalf("file not rendered: %q", out.Files[0].Content)
	}

	plain := types.GuestSetup{Raw: "#cloud-config\nruncmd:\n  - echo {{ v1.local_hostname }}\n"}
	passed, err := renderGuestSetup(plain, data)
	if err != nil {
		t.Fatal(err)
	}

	if passed.Raw != plain.Raw {
		t.Fatalf("plain content changed: %q", passed.Raw)
	}

	bad := types.GuestSetup{Raw: "name: [[ .Nope ]]"}
	if _, err := renderGuestSetup(bad, data); err == nil {
		t.Fatal("expected error for unknown variable")
	}
}
