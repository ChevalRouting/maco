package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (d *Driver) Screenshot(id string) error {
	lock, err := d.lock(id)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()

	if d.Status(id).Phase != PhaseRunning {
		return fmt.Errorf("VM is not running")
	}

	client, err := dialQMP(d.qmpPath(id), 5*time.Second)
	if err != nil {
		return err
	}

	defer func() { _ = client.close() }()

	if err := client.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	dir := d.vmRunDir(id)
	temporary := filepath.Join(dir, "preview.next.png")
	defer func() { _ = os.Remove(temporary) }()
	if _, err := client.executeArguments("screendump", map[string]string{"filename": temporary, "format": "png"}); err != nil {
		return err
	}

	return os.Rename(temporary, filepath.Join(dir, "preview.png"))
}
