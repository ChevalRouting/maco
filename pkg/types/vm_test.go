package types

import (
	"reflect"
	"testing"
)

func TestEffectiveBootOrder(t *testing.T) {
	m := VMManifest{DiskSizeGiB: 20, Disks: []VMDisk{{ID: "second"}, {ID: "third"}}, ISOs: []string{"installer"}}
	if got, want := m.EffectiveBootOrder(), []string{"iso:installer", "disk", "disk:second", "disk:third"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("default order %v, want %v", got, want)
	}

	diskless := VMManifest{Disks: []VMDisk{{ID: "second"}}, ISOs: []string{"installer"}}
	if got, want := diskless.EffectiveBootOrder(), []string{"iso:installer", "disk:second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diskless order %v, want %v", got, want)
	}

	m.BootOrder = []string{"disk:third", "disk:second", "disk", "iso:installer"}
	if got := m.EffectiveBootOrder(); !reflect.DeepEqual(got, m.BootOrder) {
		t.Fatalf("configured order changed: %v", got)
	}

	m.BootOrder = []string{"disk:second", "disk:removed", "disk:second"}
	if got, want := m.EffectiveBootOrder(), []string{"disk:second", "disk", "disk:third", "iso:installer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("incomplete order %v, want %v", got, want)
	}
}
