package qemu

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const digestFile = ".archive-sha256"

func Dir(executable string) string {
	return filepath.Join(filepath.Dir(executable), "..", "lib", "maco", "qemu")
}

func Locate(name string) (string, error) {
	if executable, err := os.Executable(); err == nil {
		bundled := filepath.Join(Dir(executable), "bin", name)
		if info, err := os.Stat(bundled); err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return filepath.Clean(bundled), nil
		}
	}

	found, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s not found in the bundled QEMU or PATH: %w", name, err)
	}

	return found, nil
}

func Embedded() bool {
	_, ok := embeddedArchive()
	return ok
}

func Install(prefix string) (bool, error) {
	data, ok := embeddedArchive()
	if !ok {
		return false, nil
	}

	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if current, err := os.ReadFile(filepath.Join(prefix, digestFile)); err == nil && strings.TrimSpace(string(current)) == digest {
		return false, nil
	}

	staging := prefix + ".new"
	_ = os.RemoveAll(staging)
	if err := extract(data, staging); err != nil {
		_ = os.RemoveAll(staging)
		return false, err
	}

	if err := os.WriteFile(filepath.Join(staging, digestFile), []byte(digest+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(staging)
		return false, err
	}

	previous := prefix + ".old"
	_ = os.RemoveAll(previous)
	if err := os.Rename(prefix, previous); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.RemoveAll(staging)
		return false, err
	}

	if err := os.Rename(staging, prefix); err != nil {
		_ = os.Rename(previous, prefix)
		_ = os.RemoveAll(staging)
		return false, err
	}

	_ = os.RemoveAll(previous)
	return true, nil
}

func extract(data []byte, dst string) error {
	compressed, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	defer func() { _ = compressed.Close() }()
	archive := tar.NewReader(compressed)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		name, ok := entryPath(header.Name)
		if !ok {
			continue
		}

		target := filepath.Join(dst, name)
		if err := extractEntry(archive, header, dst, name, target); err != nil {
			return fmt.Errorf("extract %s: %w", header.Name, err)
		}
	}
}

func entryPath(name string) (string, bool) {
	clean := path.Clean(strings.TrimPrefix(name, "./"))
	_, rest, found := strings.Cut(clean, "/")
	if !found || rest == "" || path.IsAbs(clean) || rest == ".." || strings.HasPrefix(rest, "../") {
		return "", false
	}

	return rest, true
}

func extractEntry(archive *tar.Reader, header *tar.Header, root, name, target string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0o755)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode).Perm())
		if err != nil {
			return err
		}

		if _, err := io.Copy(file, archive); err != nil {
			_ = file.Close()
			return err
		}

		return file.Close()
	case tar.TypeSymlink:
		resolved := path.Join(path.Dir(name), header.Linkname)
		if path.IsAbs(header.Linkname) || resolved == ".." || strings.HasPrefix(resolved, "../") {
			return fmt.Errorf("symlink %s escapes %s", header.Linkname, root)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.Symlink(header.Linkname, target)
	default:
		return nil
	}
}
