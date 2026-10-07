package engine

import (
	"context"
	"github.com/m-vinc/maco/pkg/config"
	"strings"
	"testing"
)

func TestMediaLibraryAndSettings(t *testing.T) {
	e := New(&config.Paths{Root: t.TempDir()})
	media, err := e.CreateMedia(context.Background(), "installer.iso", 0, strings.NewReader("test ISO"))
	if err != nil {
		t.Fatal(err)
	}

	all, err := e.ListMedia()
	if err != nil || len(all) != 1 || all[0].Name != "installer.iso" {
		t.Fatalf("media listing: %v %v", all, err)
	}

	if err := e.validateMedia([]string{media.ID}, []string{"iso:" + media.ID, "disk"}, nil); err != nil {
		t.Fatal(err)
	}

	for _, settings := range []MediaSettings{{ISOs: []string{media.ID, media.ID}}, {BootOrder: []string{"iso:" + media.ID}}, {BootOrder: []string{"disk", "disk"}}} {
		if e.validateMedia(settings.ISOs, settings.BootOrder, nil) == nil {
			t.Fatal("accepted invalid media settings")
		}
	}

	if _, err := e.GetMedia("../escape"); err == nil {
		t.Fatal("accepted unsafe ID")
	}

	if _, err := e.imageBase(context.Background(), "media:"+media.ID, false); err == nil {
		t.Fatal("accepted ISO as disk image")
	}
}
