package engine

import (
	"context"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/types"
)

func TestTemplateCRUD(t *testing.T) {
	e := New(&config.Paths{Root: t.TempDir()})
	ctx := context.Background()

	if all, err := e.ListTemplates(); err != nil || len(all) != 0 {
		t.Fatalf("empty listing: %v %v", all, err)
	}

	spec := CreateVMParams{
		Name:        "ignored",
		Image:       "flatcar-stable-arm64",
		CPUs:        2,
		MemoryMiB:   2048,
		DiskSizeGiB: 20,
		Disks:       []CreateDiskParams{{Name: "data", SizeGiB: 5}},
		GuestSetup:  &types.GuestSetup{Provisioner: "ignition", Mode: "raw"},
	}
	created, err := e.CreateTemplate(ctx, "flatcar-profile", "nginx quadlet host", spec)
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == "" || created.Name != "flatcar-profile" || created.Spec.Name != "" {
		t.Fatalf("unexpected created template: %+v", created)
	}

	got, err := e.GetTemplate(created.ID)
	if err != nil || got.Spec.Image != "flatcar-stable-arm64" || len(got.Spec.Disks) != 1 {
		t.Fatalf("get template: %+v %v", got, err)
	}

	spec.CPUs = 4
	updated, err := e.UpdateTemplate(ctx, created.ID, "flatcar-profile", "updated", spec)
	if err != nil || updated.Spec.CPUs != 4 || updated.Description != "updated" {
		t.Fatalf("update template: %+v %v", updated, err)
	}

	if all, err := e.ListTemplates(); err != nil || len(all) != 1 {
		t.Fatalf("listing after update: %v %v", all, err)
	}

	if _, err := e.CreateTemplate(ctx, "", "", spec); err == nil {
		t.Fatal("accepted empty name")
	}

	if _, err := e.CreateTemplate(ctx, "bad name!", "", spec); err == nil {
		t.Fatal("accepted invalid name")
	}

	if _, err := e.GetTemplate("../escape"); err == nil {
		t.Fatal("accepted unsafe ID")
	}

	if err := e.DeleteTemplate(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	if all, err := e.ListTemplates(); err != nil || len(all) != 0 {
		t.Fatalf("listing after delete: %v %v", all, err)
	}

	if err := e.DeleteTemplate(ctx, created.ID); err == nil {
		t.Fatal("deleted missing template")
	}
}

func TestTemplateInstance(t *testing.T) {
	e := New(&config.Paths{Root: t.TempDir()})
	ctx := context.Background()

	tpl, err := e.CreateTemplate(ctx, "profile", "", CreateVMParams{
		Image: "flatcar-stable-arm64", CPUs: 2, MemoryMiB: 2048, DiskSizeGiB: 20,
		Disks: []CreateDiskParams{{Name: "data", SizeGiB: 10}},
	})
	if err != nil {
		t.Fatal(err)
	}

	on := true
	spec, err := e.TemplateInstance(tpl.ID, TemplateInstanceParams{Name: "web-01", CPUs: 8, Autostart: &on})
	if err != nil {
		t.Fatal(err)
	}

	if spec.Name != "web-01" || spec.CPUs != 8 || spec.MemoryMiB != 2048 || spec.DiskSizeGiB != 20 {
		t.Fatalf("unexpected spec: %+v", spec)
	}

	if !spec.Autostart || len(spec.Disks) != 1 || spec.Image != "flatcar-stable-arm64" {
		t.Fatalf("inherited fields wrong: %+v", spec)
	}

	if _, err := e.TemplateInstance(tpl.ID, TemplateInstanceParams{Name: ""}); err == nil {
		t.Fatal("accepted empty name")
	}

	if _, err := e.TemplateInstance(tpl.ID, TemplateInstanceParams{Name: "bad name!"}); err == nil {
		t.Fatal("accepted invalid name")
	}

	if _, err := e.TemplateInstance("00000000-0000-0000-0000-000000000000", TemplateInstanceParams{Name: "web-01"}); err == nil {
		t.Fatal("instantiated missing template")
	}
}
