package helper

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

const Name = "maco-net-helper"

func Embedded() bool {
	_, ok := embeddedBinary()
	return ok
}

func Install(dst string) error {
	data, ok := embeddedBinary()
	if !ok {
		return fmt.Errorf("this maco build has no embedded %s; build with task build", Name)
	}

	if current, err := os.ReadFile(dst); err == nil && sha256.Sum256(current) == sha256.Sum256(data) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".maco-net-helper-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	return os.Rename(tmpPath, dst)
}
