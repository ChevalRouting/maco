package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

const snapshotJobTimeout = 2 * time.Hour

type Snapshot struct {
	Tag       string `json:"tag"`
	HasRAM    bool   `json:"has_ram"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

type imageSnapshot struct {
	Name        string `json:"name"`
	VMStateSize int64  `json:"vm-state-size"`
	DateSec     int64  `json:"date-sec"`
}

type imageInfo struct {
	Snapshots []imageSnapshot `json:"snapshots"`
}

func (d *Driver) CreateSnapshot(ctx context.Context, id, tag string, includeRAM bool, diskPaths []string) (*Snapshot, error) {
	live := d.Status(id).Phase == PhaseRunning
	if !live {
		if err := offlineSnapshot(ctx, "-c", tag, diskPaths); err != nil {
			return nil, err
		}

		return &Snapshot{Tag: tag}, nil
	}

	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return nil, err
	}

	defer client.close()
	if err := client.conn.SetDeadline(time.Now().Add(snapshotJobTimeout)); err != nil {
		return nil, err
	}

	nodes, err := client.snapshotNodes(diskPaths)
	if err != nil {
		return nil, err
	}

	if includeRAM {
		jobID := "snapshot-save-" + uuid.NewString()
		if _, err := client.executeArguments("snapshot-save", map[string]any{"job-id": jobID, "tag": tag, "vmstate": nodes[0], "devices": nodes}); err != nil {
			return nil, err
		}

		if err := client.awaitJob(ctx, jobID); err != nil {
			return nil, err
		}

		return &Snapshot{Tag: tag, HasRAM: true}, nil
	}

	actions := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		actions = append(actions, map[string]any{"type": "blockdev-snapshot-internal-sync", "data": map[string]any{"device": node, "name": tag}})
	}

	if _, err := client.executeArguments("transaction", map[string]any{"actions": actions}); err != nil {
		return nil, err
	}

	return &Snapshot{Tag: tag}, nil
}

func (d *Driver) RestoreSnapshot(ctx context.Context, id, tag string, diskPaths []string) error {
	if d.Status(id).Phase != PhaseRunning {
		return offlineSnapshot(ctx, "-a", tag, diskPaths)
	}

	hasRAM, err := snapshotHasRAM(ctx, diskPaths[0], tag)
	if err != nil {
		return err
	}

	if !hasRAM {
		return fmt.Errorf("snapshot %q has no saved RAM; stop the VM before restoring it", tag)
	}

	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return err
	}

	defer client.close()
	if err := client.conn.SetDeadline(time.Now().Add(snapshotJobTimeout)); err != nil {
		return err
	}

	nodes, err := client.snapshotNodes(diskPaths)
	if err != nil {
		return err
	}

	if _, err := client.execute("stop"); err != nil {
		return err
	}

	jobID := "snapshot-load-" + uuid.NewString()
	if _, err := client.executeArguments("snapshot-load", map[string]any{"job-id": jobID, "tag": tag, "vmstate": nodes[0], "devices": nodes}); err != nil {
		_, _ = client.execute("cont")
		return err
	}

	if err := client.awaitJob(ctx, jobID); err != nil {
		_, _ = client.execute("cont")
		return err
	}

	_, err = client.execute("cont")
	return err
}

func (d *Driver) DeleteSnapshot(ctx context.Context, id, tag string, diskPaths []string) error {
	if d.Status(id).Phase != PhaseRunning {
		return offlineSnapshot(ctx, "-d", tag, diskPaths)
	}

	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return err
	}

	defer client.close()
	if err := client.conn.SetDeadline(time.Now().Add(snapshotJobTimeout)); err != nil {
		return err
	}

	nodes, err := client.snapshotNodes(diskPaths)
	if err != nil {
		return err
	}

	jobID := "snapshot-delete-" + uuid.NewString()
	if _, err := client.executeArguments("snapshot-delete", map[string]any{"job-id": jobID, "tag": tag, "devices": nodes}); err != nil {
		return err
	}

	return client.awaitJob(ctx, jobID)
}

func (c *qmpClient) snapshotNodes(diskPaths []string) ([]string, error) {
	nodes := make([]string, 0, len(diskPaths))
	for _, path := range diskPaths {
		node, err := c.blockNodeForFile(path)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no snapshot-capable disks attached")
	}

	return nodes, nil
}

func readImageInfo(ctx context.Context, disk string) (imageInfo, error) {
	qemu, err := exec.LookPath("qemu-img")
	if err != nil {
		return imageInfo{}, err
	}

	out, err := exec.CommandContext(ctx, qemu, "info", "--output=json", "--force-share", disk).Output()
	if err != nil {
		return imageInfo{}, fmt.Errorf("read snapshots: %w", err)
	}

	var info imageInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return imageInfo{}, err
	}

	return info, nil
}

func snapshotHasRAM(ctx context.Context, primaryDisk, tag string) (bool, error) {
	info, err := readImageInfo(ctx, primaryDisk)
	if err != nil {
		return false, err
	}

	for _, snapshot := range info.Snapshots {
		if snapshot.Name == tag {
			return snapshot.VMStateSize > 0, nil
		}
	}

	return false, fmt.Errorf("snapshot %q not found", tag)
}

type jobInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (c *qmpClient) awaitJob(ctx context.Context, jobID string) error {
	return c.awaitJobStatus(ctx, "query-jobs", jobID, "snapshot", snapshotJobTimeout, func(data json.RawMessage) ([]qmpJob, error) {
		var jobs []jobInfo
		if err := json.Unmarshal(data, &jobs); err != nil {
			return nil, err
		}

		out := make([]qmpJob, len(jobs))
		for i, job := range jobs {
			out[i] = qmpJob{id: job.ID, status: job.Status, failure: job.Error}
		}

		return out, nil
	})
}

func (d *Driver) ListSnapshots(ctx context.Context, primaryDisk string) ([]Snapshot, error) {
	info, err := readImageInfo(ctx, primaryDisk)
	if err != nil {
		return nil, err
	}

	snapshots := make([]Snapshot, 0, len(info.Snapshots))
	for _, snapshot := range info.Snapshots {
		created := ""
		if snapshot.DateSec > 0 {
			created = time.Unix(snapshot.DateSec, 0).UTC().Format(time.RFC3339)
		}

		snapshots = append(snapshots, Snapshot{Tag: snapshot.Name, HasRAM: snapshot.VMStateSize > 0, SizeBytes: snapshot.VMStateSize, CreatedAt: created})
	}

	return snapshots, nil
}

func offlineSnapshot(ctx context.Context, flag, tag string, diskPaths []string) error {
	qemu, err := exec.LookPath("qemu-img")
	if err != nil {
		return err
	}

	for _, path := range diskPaths {
		if out, err := exec.CommandContext(ctx, qemu, "snapshot", flag, tag, path).CombinedOutput(); err != nil {
			return fmt.Errorf("snapshot %s: %w: %s", path, err, out)
		}
	}

	return nil
}
