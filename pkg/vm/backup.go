package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const backupTimeout = 2 * time.Hour

type blockJob struct {
	Device string `json:"device"`
	Len    int64  `json:"len"`
	Offset int64  `json:"offset"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty" binding:"optional"`
}

type BackupJob struct {
	Source string
	Target string
}

var ErrBackupCleanup = errors.New("backup jobs could not be confirmed stopped; staging files retained")

func (d *Driver) BackupDisksContext(ctx context.Context, id string, backups []BackupJob) (consistent bool, retErr error) {
	ctx, cancel := context.WithTimeout(ctx, backupTimeout)
	defer cancel()
	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return false, err
	}
	defer client.close()
	stopWatch := context.AfterFunc(ctx, func() { client.conn.Close() })
	defer stopWatch()
	if err := client.conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return false, err
	}
	actions := make([]map[string]any, len(backups))
	jobIDs := make([]string, len(backups))
	for i, backup := range backups {
		node, err := client.blockNodeForFile(backup.Source)
		if err != nil {
			return false, err
		}
		jobIDs[i] = "backup-" + uuid.NewString()
		actions[i] = map[string]any{"type": "drive-backup", "data": map[string]any{
			"job-id": jobIDs[i], "device": node, "sync": "full", "target": backup.Target, "format": "qcow2", "auto-dismiss": false,
		}}
	}
	froze, err := d.freezeGuest(id)
	thawed := !froze
	thaw := func() error {
		if thawed {
			return nil
		}
		if err := d.thawGuest(id); err != nil {
			return fmt.Errorf("thaw guest: %w", err)
		}
		thawed = true
		return nil
	}
	defer func() {
		if err := thaw(); err != nil {
			consistent = false
			retErr = errors.Join(retErr, err)
		}
	}()
	if err != nil {
		return false, fmt.Errorf("freeze guest: %w", err)
	}
	attempted := false
	defer func() {
		if retErr != nil && attempted {
			if err := d.cancelBackups(id, jobIDs); err != nil {
				retErr = errors.Join(retErr, ErrBackupCleanup, err)
			}
		}
	}()
	attempted = true
	if _, err := client.executeArguments("transaction", map[string]any{"actions": actions}); err != nil {
		return false, err
	}
	if err := thaw(); err != nil {
		return false, err
	}
	for _, jobID := range jobIDs {
		if err := client.awaitBackup(ctx, jobID); err != nil {
			return false, err
		}
	}
	return froze, nil
}

func (d *Driver) cancelBackups(id string, jobIDs []string) error {
	client, err := dialQMP(d.qmpPath(id), 3*time.Second)
	if err != nil {
		return err
	}
	defer client.close()
	deadline := time.Now().Add(15 * time.Second)
	wanted := make(map[string]bool, len(jobIDs))
	for _, id := range jobIDs {
		wanted[id] = true
	}
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("backup cancellation timed out")
		}
		client.conn.SetDeadline(time.Now().Add(3 * time.Second))
		data, err := client.execute("query-block-jobs")
		if err != nil {
			return err
		}
		var jobs []blockJob
		if err := json.Unmarshal(data, &jobs); err != nil {
			return err
		}
		active := false
		for _, job := range jobs {
			if !wanted[job.Device] {
				continue
			}
			active = true
			if job.Status == "concluded" {
				if _, err := client.executeArguments("job-dismiss", map[string]any{"id": job.Device}); err != nil {
					return err
				}
			} else {
				if _, err := client.executeArguments("job-cancel", map[string]any{"id": job.Device}); err != nil {
					return err
				}
			}
		}
		if !active {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *qmpClient) blockNodeForFile(path string) (string, error) {
	data, err := c.execute("query-block")
	if err != nil {
		return "", err
	}

	var devices []blockDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		return "", err
	}

	for _, device := range devices {
		if device.Inserted != nil && device.Inserted.Image.Filename == path {
			return device.Inserted.NodeName, nil
		}
	}

	return "", fmt.Errorf("no attached disk for %s", path)
}

func (c *qmpClient) awaitBackup(ctx context.Context, jobID string) error {
	return c.awaitJobStatus(ctx, "query-block-jobs", jobID, "backup", backupTimeout, func(data json.RawMessage) ([]qmpJob, error) {
		var jobs []blockJob
		if err := json.Unmarshal(data, &jobs); err != nil {
			return nil, err
		}
		out := make([]qmpJob, len(jobs))
		for i, job := range jobs {
			out[i] = qmpJob{id: job.Device, status: job.Status, failure: job.Error}
		}
		return out, nil
	})
}
