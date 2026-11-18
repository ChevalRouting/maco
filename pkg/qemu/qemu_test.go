package qemu

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type entry struct {
	name     string
	body     string
	link     string
	typeflag byte
	mode     int64
}

func archiveOf(t *testing.T, entries []entry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	compressed := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compressed)
	for _, e := range entries {
		header := &tar.Header{Name: e.name, Typeflag: e.typeflag, Mode: e.mode, Linkname: e.link, Size: int64(len(e.body))}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}

		if _, err := writer.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}

func TestExtractKeepsLayoutModesAndLinks(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "qemu")
	data := archiveOf(t, []entry{
		{name: "qemu/", typeflag: tar.TypeDir, mode: 0o755},
		{name: "qemu/bin/qemu-system-aarch64", body: "binary", typeflag: tar.TypeReg, mode: 0o755},
		{name: "qemu/lib/libglib-2.0.0.dylib", body: "library", typeflag: tar.TypeReg, mode: 0o644},
		{name: "qemu/lib/libglib.dylib", link: "libglib-2.0.0.dylib", typeflag: tar.TypeSymlink},
		{name: "qemu/../escape", body: "outside", typeflag: tar.TypeReg, mode: 0o644},
	})

	if err := extract(data, dst); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dst, "bin", "qemu-system-aarch64"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("binary: %v %v", info, err)
	}

	link, err := os.Readlink(filepath.Join(dst, "lib", "libglib.dylib"))
	if err != nil || link != "libglib-2.0.0.dylib" {
		t.Fatalf("symlink: %q %v", link, err)
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(dst), "escape")); !os.IsNotExist(err) {
		t.Fatalf("entry escaped the destination: %v", err)
	}
}

func TestExtractRejectsEscapingSymlinks(t *testing.T) {
	data := archiveOf(t, []entry{{name: "qemu/lib/evil", link: "../../../etc/passwd", typeflag: tar.TypeSymlink}})
	if err := extract(data, filepath.Join(t.TempDir(), "qemu")); err == nil {
		t.Fatal("escaping symlink was extracted")
	}
}
