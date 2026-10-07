package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/vm"
)

type BackupResult struct {
	VMID       string   `json:"vm_id"`
	VMName     string   `json:"vm_name"`
	Timestamp  string   `json:"timestamp"`
	Path       string   `json:"path"`
	Disks      []string `json:"disks"`
	Live       bool     `json:"live"`
	Consistent bool     `json:"consistent"`
	SizeBytes  int64    `json:"size_bytes"`
}

type BackupInfo struct {
	Timestamp  string   `json:"timestamp"`
	VMName     string   `json:"vm_name"`
	Disks      []string `json:"disks"`
	Live       bool     `json:"live"`
	Consistent bool     `json:"consistent"`
	SizeBytes  int64    `json:"size_bytes"`
	CreatedAt  string   `json:"created_at"`
}

type BackupParams struct {
	Timestamp  string `json:"timestamp" binding:"optional"`
	AsNew      bool   `json:"as_new" binding:"optional"`
	KeepLast   int    `json:"keep_last" binding:"optional"`
	MaxAgeDays int    `json:"max_age_days" binding:"optional"`
}

var backupTimestamp = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}\.[0-9]{9}-[0-9a-fA-F-]{36}$`)

func (e *Engine) BackupVM(ctx context.Context, ref string) (*BackupResult, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	defer func() { _ = lock.Close() }()
	m, err = e.vms.Load(m.ID)
	if err != nil {
		return nil, err
	}

	live := e.driver.Status(m.ID).Phase == vm.PhaseRunning

	source := e.paths.VMDiskDir(m.ID)
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, fmt.Errorf("read vm disks: %w", err)
	}

	disks := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".qcow2") {
			disks = append(disks, entry.Name())
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("vm %s has no disks to back up", m.Name)
	}

	qemu := ""
	if !live {
		if qemu, err = exec.LookPath("qemu-img"); err != nil {
			return nil, err
		}
	}

	timestamp := time.Now().UTC().Format("20060102-150405.000000000") + "-" + uuid.NewString()
	staging := filepath.Join(e.paths.VMRunDir(m.ID), "backup-"+timestamp)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return nil, err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(staging)
		}
	}()

	consistent := false
	if live {
		backups := make([]vm.BackupJob, len(disks))
		for i, disk := range disks {
			backups[i] = vm.BackupJob{Source: filepath.Join(source, disk), Target: filepath.Join(staging, disk)}
		}

		frozen, err := e.driver.BackupDisksContext(ctx, m.ID, backups)
		if err != nil {
			if errors.Is(err, vm.ErrBackupCleanup) {
				cleanup = false
			}

			return nil, fmt.Errorf("snapshot disks: %w", err)
		}

		consistent = frozen
	} else {
		for _, disk := range disks {
			src, target := filepath.Join(source, disk), filepath.Join(staging, disk)
			out, err := exec.CommandContext(ctx, qemu, "convert", "-O", "qcow2", src, target).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("snapshot %s: %w: %s", disk, err, out)
			}
		}
	}

	backupRoot := e.paths.VMBackupDir(m.ID)
	if err := os.MkdirAll(backupRoot, 0o700); err != nil {
		return nil, err
	}

	publication, err := os.MkdirTemp(backupRoot, ".backup-*")
	if err != nil {
		return nil, err
	}

	defer func() { _ = os.RemoveAll(publication) }()
	for _, disk := range disks {
		if err := copyFileContext(ctx, filepath.Join(staging, disk), filepath.Join(publication, disk)); err != nil {
			return nil, fmt.Errorf("copy %s: %w", disk, err)
		}
	}

	if err := copyFileContext(ctx, e.paths.ManifestPath(m.ID), filepath.Join(publication, "manifest.yml")); err != nil {
		return nil, err
	}

	size, err := dirSize(publication)
	if err != nil {
		return nil, err
	}

	meta := BackupInfo{Timestamp: timestamp, VMName: m.Name, Disks: disks, Live: live, Consistent: consistent, SizeBytes: size, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := writeBackupMetadata(publication, meta); err != nil {
		return nil, err
	}

	dest := filepath.Join(backupRoot, timestamp)
	if err := os.Rename(publication, dest); err != nil {
		return nil, err
	}

	return &BackupResult{
		VMID:       m.ID,
		VMName:     m.Name,
		Timestamp:  timestamp,
		Path:       dest,
		Disks:      disks,
		Live:       live,
		Consistent: consistent,
		SizeBytes:  size,
	}, nil
}

func writeBackupMetadata(dir string, meta BackupInfo) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0o600)
}

func dirSize(dir string) (int64, error) {
	var total int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}

		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}

	return total, nil
}

func (e *Engine) ListBackups(ref string) ([]BackupInfo, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	root := e.paths.VMBackupDir(m.ID)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []BackupInfo{}, nil
	}

	if err != nil {
		return nil, err
	}

	backups := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info := readBackupMetadata(filepath.Join(root, entry.Name()), entry.Name())
		backups = append(backups, info)
	}

	sort.Slice(backups, func(i, j int) bool { return backups[i].Timestamp > backups[j].Timestamp })
	return backups, nil
}

func readBackupMetadata(dir, timestamp string) BackupInfo {
	info := BackupInfo{Timestamp: timestamp}
	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err == nil {
		if err := json.Unmarshal(data, &info); err == nil {
			info.Timestamp = timestamp
			return info
		}
	}

	if size, err := dirSize(dir); err == nil {
		info.SizeBytes = size
	}

	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".qcow2") {
				info.Disks = append(info.Disks, entry.Name())
			}
		}
	}

	return info
}

func (e *Engine) DeleteBackup(ref, timestamp string) error {
	if !backupTimestamp.MatchString(timestamp) {
		return fmt.Errorf("invalid backup id")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	dest := filepath.Join(e.paths.VMBackupDir(m.ID), timestamp)
	if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("backup %s not found", timestamp)
	}

	return os.RemoveAll(dest)
}

func (e *Engine) RestoreVM(ctx context.Context, ref, timestamp string, asNew bool) (*types.VMManifest, error) {
	if !backupTimestamp.MatchString(timestamp) {
		return nil, fmt.Errorf("invalid backup id")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	backupDir := filepath.Join(e.paths.VMBackupDir(m.ID), timestamp)
	entries, err := os.ReadDir(backupDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("backup %s not found", timestamp)
	}

	if err != nil {
		return nil, err
	}

	disks := make([]string, 0)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".qcow2") {
			disks = append(disks, entry.Name())
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("backup %s has no disks", timestamp)
	}

	backupManifest, err := os.ReadFile(filepath.Join(backupDir, "manifest.yml"))
	if err != nil {
		return nil, err
	}

	restored, err := e.vms.Parse(backupManifest)
	if err != nil {
		return nil, fmt.Errorf("parse backup manifest: %w", err)
	}

	if asNew {
		return e.restoreAsNew(ctx, restored, backupDir, disks)
	}

	return e.restoreInPlace(ctx, m.ID, restored, backupDir, disks)
}

func (e *Engine) restoreInPlace(ctx context.Context, id string, restored *types.VMManifest, backupDir string, disks []string) (*types.VMManifest, error) {
	lock, err := e.driver.LockContext(ctx, id)
	if err != nil {
		return nil, err
	}

	defer func() { _ = lock.Close() }()
	if e.driver.Status(id).Phase == vm.PhaseRunning {
		return nil, fmt.Errorf("vm is running; stop it before restoring")
	}

	diskDir := e.paths.VMDiskDir(id)
	staging, err := os.MkdirTemp(filepath.Dir(diskDir), ".restore-*")
	if err != nil {
		return nil, err
	}

	defer func() { _ = os.RemoveAll(staging) }()
	for _, disk := range disks {
		if err := copyFileContext(ctx, filepath.Join(backupDir, disk), filepath.Join(staging, disk)); err != nil {
			return nil, fmt.Errorf("copy %s: %w", disk, err)
		}
	}

	restored.ID = id
	if err := os.RemoveAll(diskDir); err != nil {
		return nil, err
	}

	if err := os.Rename(staging, diskDir); err != nil {
		return nil, err
	}

	if err := e.vms.Save(restored); err != nil {
		return nil, err
	}

	exposeInterfaces(restored)
	return restored, nil
}

func (e *Engine) restoreAsNew(ctx context.Context, restored *types.VMManifest, backupDir string, disks []string) (*types.VMManifest, error) {
	restored.ID = uuid.NewString()
	restored.Name = uniqueVMName(e.vms, restored.Name+"-restored")
	restored.Autostart = false
	if restored.Interfaces != nil {
		for i := range *restored.Interfaces {
			(*restored.Interfaces)[i].MAC = ""
		}
	}

	diskDir := e.paths.VMDiskDir(restored.ID)
	staging, err := os.MkdirTemp(filepath.Dir(diskDir), ".restore-*")
	if err != nil {
		return nil, err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(staging)
		}
	}()
	for _, disk := range disks {
		if err := copyFileContext(ctx, filepath.Join(backupDir, disk), filepath.Join(staging, disk)); err != nil {
			return nil, fmt.Errorf("copy %s: %w", disk, err)
		}
	}

	exposeInterfaces(restored)
	if err := e.vms.Save(restored); err != nil {
		return nil, err
	}

	if err := os.Rename(staging, diskDir); err != nil {
		_ = e.vms.Delete(restored.ID)
		return nil, err
	}

	cleanup = false
	return restored, nil
}

func uniqueVMName(store interface {
	List() ([]*types.VMManifest, error)
}, base string) string {
	existing := map[string]bool{}
	if manifests, err := store.List(); err == nil {
		for _, m := range manifests {
			existing[m.Name] = true
		}
	}

	if len(base) > 63 {
		base = base[:63]
	}

	if !existing[base] {
		return base
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if len(candidate) > 63 {
			candidate = candidate[:63]
		}

		if !existing[candidate] {
			return candidate
		}
	}
}

func copyFileContext(ctx context.Context, src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, &contextReader{ctx: ctx, reader: in}); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.reader.Read(data)
}
