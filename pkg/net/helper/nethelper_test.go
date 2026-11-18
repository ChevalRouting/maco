package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall(t *testing.T) {
	dst := filepath.Join(t.TempDir(), Name)
	err := Install(dst)

	if !Embedded() {
		if err == nil {
			t.Fatal("expected an error when the helper is not embedded")
		}

		return
	}

	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("helper not executable: %v", info.Mode())
	}

	if err := Install(dst); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
}
