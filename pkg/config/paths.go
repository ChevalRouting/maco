package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/m-vinc/maco/pkg/storage"
	"os"
	"path/filepath"
)

const EnvDataDir = "MACO_DATA_DIR"
const EnvWorkDir = "MACO_WORKDIR"

const globalDataDir = "/Library/Application Support/maco"

type Paths struct {
	Root string
}

func DefaultDataDir() (string, error) {
	if v := os.Getenv(EnvDataDir); v != "" {
		return v, nil
	}

	return globalDataDir, nil
}

func Resolve(dir string) (*Paths, error) {
	if dir == "" {
		var err error
		if dir, err = DefaultDataDir(); err != nil {
			return nil, err
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir %q: %w", dir, err)
	}

	return &Paths{Root: abs}, nil
}

func (p *Paths) EnsureDirs() error {
	dirs := []string{p.Root, p.VMsDir(), p.NetworksDir(), p.ImagesDir(), p.TemplatesDir(), p.DisksDir(), p.BackupsDir(), p.FirmwareDir(), p.RunDir()}
	for _, d := range dirs {
		if err := storage.EnsurePrivateDir(d); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	return nil
}

func (p *Paths) VMsDir() string       { return filepath.Join(p.Root, "vms") }
func (p *Paths) NetworksDir() string  { return filepath.Join(p.Root, "networks") }
func (p *Paths) ImagesDir() string    { return filepath.Join(p.Root, "images") }
func (p *Paths) TemplatesDir() string { return filepath.Join(p.Root, "templates") }
func (p *Paths) DisksDir() string     { return filepath.Join(p.Root, "disks") }
func (p *Paths) BackupsDir() string   { return filepath.Join(p.Root, "backups") }
func (p *Paths) FirmwareDir() string  { return filepath.Join(p.Root, "firmware") }

func (p *Paths) RunDir() string {
	if v := os.Getenv(EnvWorkDir); v != "" {
		return v
	}
	sum := sha256.Sum256([]byte(p.Root))
	return filepath.Join("/tmp", "maco-"+hex.EncodeToString(sum[:4]))
}

func (p *Paths) DBPath() string { return filepath.Join(p.Root, "web.db") }

func (p *Paths) InitMarkerPath() string { return filepath.Join(p.Root, ".initialized") }

func (p *Paths) TLSCertPath() string { return filepath.Join(p.Root, "tls.crt") }
func (p *Paths) TLSKeyPath() string  { return filepath.Join(p.Root, "tls.key") }

func (p *Paths) ManifestPath(id string) string {
	return filepath.Join(p.VMsDir(), id+".yml")
}

func (p *Paths) VMDiskDir(id string) string { return filepath.Join(p.DisksDir(), id) }

func (p *Paths) VMBackupDir(id string) string { return filepath.Join(p.BackupsDir(), id) }

func (p *Paths) VMRunDir(id string) string { return filepath.Join(p.RunDir(), id) }
