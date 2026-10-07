package vm

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrDiskBackendOpen = errors.New("disk backend remains open")

func diskNodeName(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("d%x", digest[:14])
}

type blockImage struct {
	Filename    string `json:"filename"`
	VirtualSize int64  `json:"virtual-size"`
}

type blockInserted struct {
	NodeName string     `json:"node-name"`
	Image    blockImage `json:"image"`
}

type blockDevice struct {
	Device   string         `json:"device"`
	Inserted *blockInserted `json:"inserted"`
}

func (d *Driver) AttachDisk(id, diskID, path string) error {
	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return err
	}

	defer func() { _ = client.close() }()
	if err := client.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	node := diskNodeName(diskID)
	arguments := map[string]any{"driver": "qcow2", "node-name": node, "file": map[string]string{"driver": "file", "filename": path}}
	if _, err := client.executeArguments("blockdev-add", arguments); err != nil {
		if !qmpRejected(err) {
			return fmt.Errorf("%w: blockdev-add outcome unknown: %v", ErrDiskBackendOpen, err)
		}

		return fmt.Errorf("attach disk: %w", err)
	}

	if _, err := client.executeArguments("device_add", map[string]string{"driver": "scsi-hd", "bus": "scsi.0", "id": "device-" + diskID, "drive": node, "serial": diskID, "device_id": node[:20]}); err != nil {
		if !qmpRejected(err) {
			return fmt.Errorf("%w: device_add outcome unknown: %v", ErrDiskBackendOpen, err)
		}

		if _, cleanupErr := client.executeArguments("blockdev-del", map[string]string{"node-name": node}); cleanupErr != nil {
			return fmt.Errorf("%w: attach disk: %v; close backend: %v", ErrDiskBackendOpen, err, cleanupErr)
		}

		return fmt.Errorf("attach disk (restart older VMs to enable the SCSI controller): %w", err)
	}

	return nil
}

func (d *Driver) GrowDisk(id, path string, size int) error {
	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return err
	}

	defer func() { _ = client.close() }()
	if err := client.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	data, err := client.execute("query-block")
	if err != nil {
		return err
	}

	var devices []blockDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		return err
	}

	for _, device := range devices {
		if device.Inserted == nil || device.Inserted.Image.Filename != path {
			continue
		}

		capacity := int64(size) * (1 << 30)
		if size < 1 || capacity < device.Inserted.Image.VirtualSize {
			return fmt.Errorf("disk shrinking is not supported")
		}

		_, err := client.executeArguments("block_resize", map[string]any{"node-name": device.Inserted.NodeName, "size": capacity})
		return err
	}

	return fmt.Errorf("attached disk not found")
}
