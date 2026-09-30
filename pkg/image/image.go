package image

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/m-vinc/maco/pkg/types"
	"github.com/rs/zerolog/log"
)

type Image struct {
	Name         string
	Distro       string
	Version      string
	Arch         string
	DisplayName  string
	Description  string
	Tags         []string
	URL          string
	SHA256       string
	Provisioning []types.GuestCapability
}

var ignitionProvisioning = []types.GuestCapability{{Provisioner: types.ProvisionerIgnition, Raw: true}}

func DefaultProvisioning() []types.GuestCapability {
	return []types.GuestCapability{{Provisioner: types.ProvisionerCloudInit, Raw: true}}
}

func Provisioning(img Image) []types.GuestCapability {
	if len(img.Provisioning) == 0 {
		return DefaultProvisioning()
	}
	return img.Provisioning
}

type DiskInfo struct {
	Format      string `json:"format"`
	VirtualSize int64  `json:"virtual-size"`
	ActualSize  int64  `json:"actual-size"`
}

var Catalog = map[string]Image{
	"ubuntu-24.04-arm64": {
		Name:        "ubuntu-24.04-arm64",
		Distro:      "ubuntu",
		Version:     "24.04 LTS",
		Arch:        "arm64",
		DisplayName: "Ubuntu 24.04 LTS",
		Tags:        []string{"Debian-like", "LTS"},
		Description: "Noble Numbat cloud image",
		URL:         "https://cloud-images.ubuntu.com/releases/noble/release/ubuntu-24.04-server-cloudimg-arm64.img",
	},
	"ubuntu-22.04-arm64": {
		Name:        "ubuntu-22.04-arm64",
		Distro:      "ubuntu",
		Version:     "22.04 LTS",
		Arch:        "arm64",
		DisplayName: "Ubuntu 22.04 LTS",
		Tags:        []string{"Debian-like", "LTS"},
		Description: "Jammy Jellyfish cloud image",
		URL:         "https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img",
	},
	"debian-12-arm64": {
		Name:        "debian-12-arm64",
		Distro:      "debian",
		Version:     "12",
		Arch:        "arm64",
		DisplayName: "Debian 12",
		Tags:        []string{"Debian-like"},
		Description: "Bookworm generic cloud image",
		URL:         "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-genericcloud-arm64.qcow2",
	},
	"fedora-44-arm64": {
		Name:        "fedora-44-arm64",
		Distro:      "fedora",
		Version:     "44",
		Arch:        "arm64",
		DisplayName: "Fedora Cloud 44",
		Tags:        []string{"RPM"},
		Description: "Fedora Cloud Base generic image",
		URL:         "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Cloud/aarch64/images/Fedora-Cloud-Base-Generic-44-1.7.aarch64.qcow2",
	},
	"rocky-9-arm64": {
		Name:        "rocky-9-arm64",
		Distro:      "rocky-linux",
		Version:     "9",
		Arch:        "arm64",
		DisplayName: "Rocky Linux 9",
		Tags:        []string{"RHEL-like"},
		Description: "Generic cloud image",
		URL:         "https://dl.rockylinux.org/pub/rocky/9/images/aarch64/Rocky-9-GenericCloud.latest.aarch64.qcow2",
	},
	"rocky-10-arm64": {
		Name:        "rocky-10-arm64",
		Distro:      "rocky-linux",
		Version:     "10",
		Arch:        "arm64",
		DisplayName: "Rocky Linux 10",
		Tags:        []string{"RHEL-like"},
		Description: "Generic cloud image",
		URL:         "https://dl.rockylinux.org/pub/rocky/10/images/aarch64/Rocky-10-GenericCloud.latest.aarch64.qcow2",
	},
	"almalinux-9-arm64": {
		Name:        "almalinux-9-arm64",
		Distro:      "almalinux",
		Version:     "9",
		Arch:        "arm64",
		DisplayName: "AlmaLinux 9",
		Tags:        []string{"RHEL-like"},
		Description: "Generic cloud image",
		URL:         "https://repo.almalinux.org/almalinux/9/cloud/aarch64/images/AlmaLinux-9-GenericCloud-latest.aarch64.qcow2",
	},
	"alpine-3.23-arm64": {
		Name:        "alpine-3.23-arm64",
		Distro:      "alpine",
		Version:     "3.23",
		Arch:        "arm64",
		DisplayName: "Alpine Linux 3.23",
		Tags:        []string{"Lightweight", "musl"},
		Description: "Cloud image with cloud-init",
		URL:         "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/cloud/nocloud_alpine-3.23.4-aarch64-uefi-cloudinit-r0.qcow2",
	},
	"flatcar-stable-arm64": {
		Name:         "flatcar-stable-arm64",
		Distro:       "flatcar",
		Version:      "stable",
		Arch:         "arm64",
		DisplayName:  "Flatcar Container Linux",
		Tags:         []string{"Container", "Ignition", "podman"},
		Description:  "Immutable container host provisioned with Butane/Ignition",
		URL:          "https://stable.release.flatcar-linux.net/arm64-usr/current/flatcar_production_qemu_uefi_image.img",
		Provisioning: ignitionProvisioning,
	},
}

func Lookup(name string) (Image, error) {
	img, ok := Catalog[name]
	if !ok {
		return Image{}, fmt.Errorf("unknown image %q", name)
	}

	return img, nil
}

func cachePath(imagesDir string, img Image) string {
	return filepath.Join(imagesDir, img.Name+".qcow2")
}

func CachedSize(imagesDir string, img Image) (int64, bool) {
	fi, err := os.Stat(cachePath(imagesDir, img))
	if err != nil || fi.Size() == 0 {
		return 0, false
	}
	return fi.Size(), true
}

func RemoveCached(imagesDir string, img Image) error {
	err := os.Remove(cachePath(imagesDir, img))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func Pull(imagesDir string, img Image) (string, error) {
	return PullContext(log.Logger.WithContext(context.Background()), imagesDir, img)
}

func PullContext(ctx context.Context, imagesDir string, img Image) (string, error) {
	if err := os.MkdirAll(imagesDir, 0o700); err != nil {
		return "", err
	}

	dst := cachePath(imagesDir, img)
	if fi, err := os.Stat(dst); err == nil && fi.Size() > 0 {
		log.Ctx(ctx).Info().Str("image", img.Name).Str("path", dst).Msg("image already cached")
		return dst, nil
	}

	log.Ctx(ctx).Info().Str("image", img.Name).Str("url", img.URL).Msg("downloading cloud image")
	if err := download(ctx, img.URL, dst); err != nil {
		return "", err
	}

	if img.SHA256 != "" {
		if err := verifySHA256(dst, img.SHA256); err != nil {
			_ = os.Remove(dst)
			return "", err
		}
	}

	log.Ctx(ctx).Info().Str("image", img.Name).Str("path", dst).Msg("image ready")
	return dst, nil
}

func download(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", tmp, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dst)
}

func verifySHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s want %s", path, got, want)
	}

	return nil
}
