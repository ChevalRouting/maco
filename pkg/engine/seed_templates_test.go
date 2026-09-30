package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
)

func TestSeedDefaultTemplates(t *testing.T) {
	e := New(&config.Paths{Root: t.TempDir()})
	ctx := context.Background()

	if !e.FirstRun() {
		t.Fatal("expected first run on a fresh data dir")
	}

	n, err := e.SeedDefaultTemplates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(defaultTemplates()) || n == 0 {
		t.Fatalf("seeded %d templates, want %d", n, len(defaultTemplates()))
	}
	if e.FirstRun() {
		t.Fatal("expected first-run marker after seeding")
	}

	all, err := e.ListTemplates()
	if err != nil || len(all) != n {
		t.Fatalf("listing after seed: %d %v", len(all), err)
	}

	again, err := e.SeedDefaultTemplates(ctx)
	if err != nil || again != 0 {
		t.Fatalf("re-seed should be a no-op: %d %v", again, err)
	}
	all, _ = e.ListTemplates()
	if len(all) != n {
		t.Fatalf("re-seed changed template count to %d", len(all))
	}

	for _, d := range defaultTemplates() {
		if d.Spec.GuestSetup == nil || d.Spec.GuestSetup.Raw == "" {
			t.Fatalf("default %q missing guest setup", d.Name)
		}
		for _, keys := range [][]string{nil, {"ssh-ed25519 AAAAExample test@host"}} {
			data := guestTemplateData{Name: "web-01", SSHKeys: keys}
			out, err := renderGuestSetup(*d.Spec.GuestSetup, data)
			if err != nil {
				t.Fatalf("default %q render (keys=%v): %v", d.Name, keys, err)
			}
			if !strings.Contains(out.Raw, "web-01") {
				t.Fatalf("default %q did not render hostname:\n%s", d.Name, out.Raw)
			}
			if len(keys) > 0 && !strings.Contains(out.Raw, keys[0]) {
				t.Fatalf("default %q did not render SSH key:\n%s", d.Name, out.Raw)
			}
		}
	}
}
