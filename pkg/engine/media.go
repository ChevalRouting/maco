package engine

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/storage"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

type Media struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	SizeGiB int    `json:"size_gib"`
	InUse   bool   `json:"in_use"`
	Path    string `json:"-"`
}

func (e *Engine) mediaDir() string { return filepath.Join(e.paths.ImagesDir(), "media") }

func (e *Engine) lockMedia(ctx context.Context) (*os.File, error) {
	return storage.Lock(ctx, filepath.Join(e.paths.Root, "media.lock"))
}

func (e *Engine) GetMedia(id string) (Media, error) {
	var m Media
	if _, err := uuid.Parse(id); err != nil {
		return m, fmt.Errorf("invalid media ID")
	}

	data, err := os.ReadFile(filepath.Join(e.mediaDir(), id+".json"))
	if err != nil {
		return m, err
	}

	if err = json.Unmarshal(data, &m); err != nil {
		return m, err
	}

	ext := ".qcow2"
	if m.Kind == "iso" {
		ext = ".iso"
	}

	m.Path = filepath.Join(e.mediaDir(), id+ext)
	return m, nil
}
func (e *Engine) ListMedia() ([]Media, error) {
	result := []Media{}
	entries, err := os.ReadDir(e.mediaDir())
	if os.IsNotExist(err) {
		return result, nil
	}

	if err != nil {
		return nil, err
	}

	used, err := e.usedMedia()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			m, err := e.GetMedia(strings.TrimSuffix(entry.Name(), ".json"))
			if err != nil {
				return nil, err
			}

			m.InUse = used[m.ID]
			result = append(result, m)
		}
	}

	return result, nil
}
func (e *Engine) usedMedia() (map[string]bool, error) {
	used := map[string]bool{}
	manifests, err := e.vms.List()
	if err != nil {
		return nil, err
	}

	for _, m := range manifests {
		if strings.HasPrefix(m.Image, "media:") {
			used[strings.TrimPrefix(m.Image, "media:")] = true
		}

		for _, id := range m.ISOs {
			used[id] = true
		}

		for _, disk := range m.Disks {
			backing := qcow2Backing(filepath.Join(e.paths.VMDiskDir(m.ID), disk.ID+".qcow2"))
			if id := e.mediaIDForPath(backing); id != "" {
				used[id] = true
			}
		}
	}

	return used, nil
}
func (e *Engine) mediaIDForPath(p string) string {
	if p == "" || filepath.Dir(p) != e.mediaDir() {
		return ""
	}

	if !strings.HasSuffix(p, ".qcow2") {
		return ""
	}

	return strings.TrimSuffix(filepath.Base(p), ".qcow2")
}
func qcow2Backing(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}

	defer func() { _ = f.Close() }()
	var header [20]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return ""
	}

	if string(header[:4]) != "QFI\xfb" {
		return ""
	}

	offset := binary.BigEndian.Uint64(header[8:16])
	size := binary.BigEndian.Uint32(header[16:20])
	if offset == 0 || size == 0 || size > 4096 {
		return ""
	}

	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, int64(offset)); err != nil {
		return ""
	}

	return string(buf)
}
func (e *Engine) DeleteMedia(id string) error {
	lock, err := e.lockMedia(context.Background())
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	m, err := e.GetMedia(id)
	if err != nil {
		return err
	}

	used, err := e.usedMedia()
	if err != nil {
		return err
	}

	if used[id] {
		return fmt.Errorf("media is in use by a virtual machine")
	}

	if err := os.Remove(filepath.Join(e.mediaDir(), id+".json")); err != nil {
		return err
	}

	_ = os.Remove(m.Path)
	return nil
}
func (e *Engine) MediaFreeBytes() (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(e.paths.ImagesDir(), &stat); err != nil {
		return 0, err
	}

	return stat.Bavail * uint64(stat.Bsize), nil
}
func (e *Engine) CreateMedia(ctx context.Context, name string, size int, iso io.Reader) (Media, error) {
	m := Media{ID: uuid.NewString(), Name: strings.TrimSpace(name), Kind: "image", SizeGiB: size}
	if m.Name == "" {
		return m, fmt.Errorf("name is required")
	}

	if iso == nil && (size < 1 || size > 65536) {
		return m, fmt.Errorf("size must be between 1 and 65536 GiB")
	}

	if err := os.MkdirAll(e.mediaDir(), 0o700); err != nil {
		return m, err
	}

	ext := ".qcow2"
	if iso != nil {
		m.Kind = "iso"
		m.SizeGiB = 0
		ext = ".iso"
	}

	m.Path = filepath.Join(e.mediaDir(), m.ID+ext)
	if iso != nil {
		part := m.Path + ".part"
		defer func() { _ = os.Remove(part) }()
		f, err := os.OpenFile(part, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return m, err
		}

		_, err = io.Copy(f, iso)
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}

		if err == nil {
			err = os.Rename(part, m.Path)
		}

		if err != nil {
			_ = os.Remove(part)
			_ = os.Remove(m.Path)
			return m, err
		}
	} else {
		out, err := exec.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", m.Path, fmt.Sprintf("%dG", size)).CombinedOutput()
		if err != nil {
			_ = os.Remove(m.Path)
			return m, fmt.Errorf("create image: %w: %s", err, out)
		}
	}

	data, _ := json.Marshal(m)
	if err := storage.WriteFile(filepath.Join(e.mediaDir(), m.ID+".json"), data, 0o600); err != nil {
		_ = os.Remove(m.Path)
		return m, err
	}

	return m, nil
}
func (e *Engine) UploadImage(ctx context.Context, name string, r io.Reader) (Media, error) {
	m := Media{ID: uuid.NewString(), Name: strings.TrimSpace(name), Kind: "image"}
	if m.Name == "" {
		return m, fmt.Errorf("name is required")
	}

	if err := os.MkdirAll(e.mediaDir(), 0o700); err != nil {
		return m, err
	}

	m.Path = filepath.Join(e.mediaDir(), m.ID+".qcow2")
	part := m.Path + ".part"
	defer func() { _ = os.Remove(part) }()
	f, err := os.OpenFile(part, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return m, err
	}

	_, err = io.Copy(f, r)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}

	if err != nil {
		_ = os.Remove(part)
		return m, err
	}

	info, err := image.InspectStandalone(ctx, part)
	if err != nil {
		return m, err
	}

	free, err := e.MediaFreeBytes()
	if err != nil {
		return m, err
	}

	needed := info.ActualSize
	if needed <= 0 {
		if fi, statErr := os.Stat(part); statErr == nil {
			needed = fi.Size()
		}
	}

	if needed > 0 && uint64(needed) > free {
		return m, fmt.Errorf("not enough free disk space to convert this image")
	}

	normalized := m.Path + ".normalized"
	defer func() { _ = os.Remove(normalized) }()
	out, err := exec.CommandContext(ctx, "qemu-img", "convert", "-f", info.Format, "-O", "qcow2", part, normalized).CombinedOutput()
	if err != nil {
		return m, fmt.Errorf("normalize image: %w: %s", err, out)
	}

	if err := os.Chmod(normalized, 0o600); err != nil {
		return m, err
	}

	m.SizeGiB = int((info.VirtualSize + (1 << 30) - 1) >> 30)
	if err := os.Rename(normalized, m.Path); err != nil {
		_ = os.Remove(part)
		return m, err
	}

	data, _ := json.Marshal(m)
	if err := storage.WriteFile(filepath.Join(e.mediaDir(), m.ID+".json"), data, 0o600); err != nil {
		_ = os.Remove(m.Path)
		return m, err
	}

	return m, nil
}

func (e *Engine) imageBase(ctx context.Context, ref string, pull bool) (string, error) {
	if strings.HasPrefix(ref, "media:") {
		m, err := e.GetMedia(strings.TrimPrefix(ref, "media:"))
		if err != nil {
			return "", err
		}

		if m.Kind != "image" {
			return "", fmt.Errorf("boot image must be a disk image")
		}

		return m.Path, nil
	}

	img, err := image.Lookup(ref)
	if err != nil {
		return "", err
	}

	if !pull {
		return "", nil
	}

	return image.PullContext(ctx, e.paths.ImagesDir(), img)
}

type MediaSettings struct {
	ISOs      []string `json:"isos" binding:"optional"`
	BootOrder []string `json:"boot_order" binding:"optional"`
}

func (e *Engine) validateMedia(isos, order []string, disks []types.VMDisk) error {
	valid := map[string]bool{"disk": true}
	seen := map[string]bool{}
	for _, id := range isos {
		m, err := e.GetMedia(id)
		if err != nil {
			return err
		}

		if m.Kind != "iso" || seen[id] {
			return fmt.Errorf("invalid or duplicate ISO")
		}

		seen[id] = true
		valid["iso:"+id] = true
	}

	for _, disk := range disks {
		valid["disk:"+disk.ID] = true
	}

	seen = map[string]bool{}
	for _, id := range order {
		if !valid[id] || seen[id] {
			return fmt.Errorf("invalid or duplicate boot device")
		}

		seen[id] = true
	}

	return nil
}
func (e *Engine) UpdateMediaContext(ctx context.Context, ref string, settings MediaSettings) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	if e.driver.Status(m.ID).Phase == vm.PhaseRunning {
		return fmt.Errorf("stop the VM before changing media")
	}

	m, err = e.vms.Load(m.ID)
	if err != nil {
		return err
	}

	mediaLock, err := e.lockMedia(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = mediaLock.Close() }()
	if err = e.validateMedia(settings.ISOs, settings.BootOrder, m.Disks); err != nil {
		return err
	}

	m.ISOs = settings.ISOs
	m.BootOrder = settings.BootOrder
	return e.vms.Save(m)
}
