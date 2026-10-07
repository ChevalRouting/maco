package vm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type qmpClient struct {
	conn   net.Conn
	dec    *json.Decoder
	enc    *json.Encoder
	events []qmpMessage
}

type qmpMessage struct {
	QMP    json.RawMessage `json:"QMP,omitempty" binding:"optional"`
	Return json.RawMessage `json:"return,omitempty" binding:"optional"`
	Error  *qmpError       `json:"error,omitempty" binding:"optional"`
	Event  string          `json:"event,omitempty" binding:"optional"`
	Data   json.RawMessage `json:"data,omitempty" binding:"optional"`
}

type qmpError struct {
	Class string `json:"class"`
	Desc  string `json:"desc"`
}

func (e *qmpError) Error() string { return fmt.Sprintf("qmp %s: %s", e.Class, e.Desc) }

func dialQMP(socket string, timeout time.Duration) (*qmpClient, error) {
	conn, err := net.DialTimeout("unix", socket, timeout)
	if err != nil {
		return nil, err
	}

	_ = conn.SetDeadline(time.Now().Add(timeout))

	c := &qmpClient{
		conn: conn,
		dec:  json.NewDecoder(bufio.NewReader(conn)),
		enc:  json.NewEncoder(conn),
	}

	var greeting qmpMessage
	if err := c.dec.Decode(&greeting); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("qmp greeting: %w", err)
	}

	if _, err := c.execute("qmp_capabilities"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("qmp capabilities: %w", err)
	}

	_ = conn.SetDeadline(time.Time{})
	return c, nil
}

func (c *qmpClient) close() error { return c.conn.Close() }

type qmpCommand struct {
	Execute   string `json:"execute"`
	Arguments any    `json:"arguments,omitempty" binding:"optional"`
}

func (c *qmpClient) execute(cmd string) (json.RawMessage, error) {
	return c.executeArguments(cmd, nil)
}

func (c *qmpClient) executeArguments(cmd string, arguments any) (json.RawMessage, error) {
	if err := c.enc.Encode(qmpCommand{Execute: cmd, Arguments: arguments}); err != nil {
		return nil, err
	}

	for {
		var msg qmpMessage
		if err := c.dec.Decode(&msg); err != nil {
			return nil, err
		}

		if msg.Error != nil {
			return nil, msg.Error
		}

		if msg.Event != "" {
			if len(c.events) < 256 {
				c.events = append(c.events, msg)
			}

			continue
		}

		return msg.Return, nil
	}
}

type qmpJob struct {
	id      string
	status  string
	failure string
}

func (c *qmpClient) awaitJobStatus(ctx context.Context, query, jobID, noun string, timeout time.Duration, extract func(json.RawMessage) ([]qmpJob, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := c.conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return err
		}

		data, err := c.execute(query)
		if err != nil {
			return err
		}

		jobs, err := extract(data)
		if err != nil {
			return err
		}

		var job *qmpJob
		for i := range jobs {
			if jobs[i].id == jobID {
				job = &jobs[i]
				break
			}
		}

		if job == nil {
			return fmt.Errorf("%s job %s vanished before completing", noun, jobID)
		}

		if job.status == "concluded" {
			failure := job.failure
			if _, err := c.executeArguments("job-dismiss", map[string]string{"id": jobID}); err != nil {
				return err
			}

			if failure != "" {
				return fmt.Errorf("%s job failed: %s", noun, failure)
			}

			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not finish within %s", noun, timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (c *qmpClient) powerdown() error {
	if err := c.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	defer func() { _ = c.conn.SetDeadline(time.Time{}) }()
	_, err := c.execute("system_powerdown")
	return err
}

func (c *qmpClient) reset() error {
	if err := c.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	defer func() { _ = c.conn.SetDeadline(time.Time{}) }()
	_, err := c.execute("system_reset")
	return err
}

func (d *Driver) Shutdown(id string) error {
	lock, err := d.lock(id)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	if d.Status(id).Phase != PhaseRunning {
		return nil
	}

	client, err := dialQMP(d.qmpPath(id), 3*time.Second)
	if err != nil {
		return err
	}

	defer client.close()
	return client.powerdown()
}

func (d *Driver) Reboot(id string) error {
	lock, err := d.lock(id)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	if d.Status(id).Phase != PhaseRunning {
		return nil
	}

	client, err := dialQMP(d.qmpPath(id), 3*time.Second)
	if err != nil {
		return err
	}

	defer client.close()
	return client.reset()
}
