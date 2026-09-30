package firmware

import (
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const Filename = "maco-aarch64-code.fd"

const Size = 67108864

const assetPath = "assets/maco-aarch64-code.fd.bz2"

//go:embed assets/maco-aarch64-code.fd.bz2
var assets embed.FS

func EnsureContext(ctx context.Context, firmwareDir string) (string, error) {
	compressed, err := assets.ReadFile(assetPath)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(compressed)
	version := hex.EncodeToString(sum[:])

	dst := filepath.Join(firmwareDir, Filename)
	stamp := dst + ".sha256"

	if current, err := os.ReadFile(stamp); err == nil && string(current) == version {
		if fi, err := os.Stat(dst); err == nil && fi.Size() == Size {
			return dst, nil
		}
	}

	if err := os.MkdirAll(firmwareDir, 0o700); err != nil {
		return "", err
	}

	log.Ctx(ctx).Info().Str("path", dst).Msg("extracting maco uefi firmware")
	if err := extract(dst, compressed); err != nil {
		return "", err
	}

	if err := os.WriteFile(stamp, []byte(version), 0o600); err != nil {
		return "", err
	}

	log.Ctx(ctx).Info().Str("path", dst).Msg("firmware ready")
	return dst, nil
}

func extract(dst string, compressed []byte) error {
	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, bzip2.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	if n != Size {
		_ = os.Remove(tmp)
		return fmt.Errorf("firmware size mismatch: got %d want %d", n, Size)
	}

	return os.Rename(tmp, dst)
}
