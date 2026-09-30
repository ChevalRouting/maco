package engine

import (
	"context"
	"testing"
)

func TestAutomaticStartupCanBeChangedAfterCreation(t *testing.T) {
	eng := testEngine(t)
	manifest, err := eng.CreateVM(CreateVMParams{Name: "startup-preference", Image: "ubuntu-24.04-arm64", CPUs: 2, MemoryMiB: 2048, DiskSizeGiB: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false} {
		if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{Autostart: &enabled}); err != nil {
			t.Fatal(err)
		}
		got, err := eng.vms.Load(manifest.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Autostart != enabled || got.CPUs != manifest.CPUs || got.MemoryMiB != manifest.MemoryMiB {
			t.Fatalf("unexpected saved startup preferences: %+v", got)
		}
	}
}
