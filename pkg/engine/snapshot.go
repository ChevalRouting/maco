package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/m-vinc/maco/pkg/vm"
)

type SnapshotParams struct {
	Tag        string `json:"tag" binding:"optional"`
	IncludeRAM bool   `json:"include_ram" binding:"optional"`
}

var snapshotTag = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (e *Engine) vmDiskPaths(id string) ([]string, error) {
	dir := e.paths.VMDiskDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read vm disks: %w", err)
	}

	primary := ""
	data := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".qcow2") {
			continue
		}

		if entry.Name() == "disk.qcow2" {
			primary = filepath.Join(dir, entry.Name())
			continue
		}

		data = append(data, filepath.Join(dir, entry.Name()))
	}

	sort.Strings(data)
	paths := make([]string, 0, len(data)+1)
	if primary != "" {
		paths = append(paths, primary)
	}

	paths = append(paths, data...)
	if len(paths) == 0 {
		return nil, fmt.Errorf("vm has no disks to snapshot")
	}

	return paths, nil
}

func (e *Engine) CreateSnapshot(ctx context.Context, ref string, params SnapshotParams) (*vm.Snapshot, error) {
	if !snapshotTag.MatchString(params.Tag) {
		return nil, fmt.Errorf("snapshot name must be 1 to 128 characters of letters, digits, dot, dash or underscore")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	defer func() { _ = lock.Close() }()
	paths, err := e.vmDiskPaths(m.ID)
	if err != nil {
		return nil, err
	}

	return e.driver.CreateSnapshot(ctx, m.ID, params.Tag, params.IncludeRAM, paths)
}

func (e *Engine) ListSnapshots(ref string) ([]vm.Snapshot, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	paths, err := e.vmDiskPaths(m.ID)
	if err != nil {
		return []vm.Snapshot{}, nil
	}

	return e.driver.ListSnapshots(context.Background(), paths[0])
}

func (e *Engine) RestoreSnapshot(ctx context.Context, ref string, params SnapshotParams) error {
	if !snapshotTag.MatchString(params.Tag) {
		return fmt.Errorf("invalid snapshot name")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	paths, err := e.vmDiskPaths(m.ID)
	if err != nil {
		return err
	}

	return e.driver.RestoreSnapshot(ctx, m.ID, params.Tag, paths)
}

func (e *Engine) DeleteSnapshot(ctx context.Context, ref string, params SnapshotParams) error {
	if !snapshotTag.MatchString(params.Tag) {
		return fmt.Errorf("invalid snapshot name")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	paths, err := e.vmDiskPaths(m.ID)
	if err != nil {
		return err
	}

	return e.driver.DeleteSnapshot(ctx, m.ID, params.Tag, paths)
}
