package vm

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"
)

var ErrGuestAgentUnavailable = errors.New("qemu guest agent is not responding")

func (d *Driver) qgaPath(id string) string {
	return filepath.Join(d.vmRunDir(id), "guest-agent.sock")
}

type GuestOSInfo struct {
	ID            string `json:"id,omitempty" binding:"optional"`
	Name          string `json:"name,omitempty" binding:"optional"`
	PrettyName    string `json:"pretty_name,omitempty" binding:"optional"`
	Version       string `json:"version,omitempty" binding:"optional"`
	VersionID     string `json:"version_id,omitempty" binding:"optional"`
	KernelRelease string `json:"kernel_release,omitempty" binding:"optional"`
	KernelVersion string `json:"kernel_version,omitempty" binding:"optional"`
	Machine       string `json:"machine,omitempty" binding:"optional"`
}

type GuestIPAddress struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Prefix  int    `json:"prefix"`
}

type GuestInterface struct {
	Name            string           `json:"name"`
	HardwareAddress string           `json:"hardware_address,omitempty" binding:"optional"`
	IPAddresses     []GuestIPAddress `json:"ip_addresses,omitempty" binding:"optional"`
}

type GuestFilesystem struct {
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
	Type       string `json:"type"`
	TotalBytes int64  `json:"total_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
}

type GuestAgentInfo struct {
	Version     string            `json:"version" binding:"optional"`
	Hostname    string            `json:"hostname,omitempty" binding:"optional"`
	OS          *GuestOSInfo      `json:"os,omitempty" binding:"optional"`
	Interfaces  []GuestInterface  `json:"interfaces,omitempty" binding:"optional"`
	Filesystems []GuestFilesystem `json:"filesystems,omitempty" binding:"optional"`
}

func (d *Driver) GuestAgentInfo(id string) (*GuestAgentInfo, error) {
	if d.Status(id).Phase != PhaseRunning {
		return nil, ErrGuestAgentUnavailable
	}

	client, err := dialQGA(d.qgaPath(id), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGuestAgentUnavailable, err)
	}
	defer func() { _ = client.close() }()

	var base struct {
		Version string `json:"version"`
	}
	if err := client.execute("guest-info", nil, &base); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGuestAgentUnavailable, err)
	}

	info := &GuestAgentInfo{Version: base.Version}
	info.Hostname = client.hostname()
	info.OS = client.osInfo()
	info.Interfaces = client.interfaces()
	info.Filesystems = client.filesystems()
	return info, nil
}

const (
	freezeTimeout      = 60 * time.Second
	thawTimeout        = 2 * time.Minute
	thawCommandTimeout = 30 * time.Second
)

func (d *Driver) freezeGuest(id string) (froze bool, err error) {
	client, err := dialQGA(d.qgaPath(id), 3*time.Second)
	if err != nil {
		return false, nil
	}
	defer func() { _ = client.close() }()
	if err := client.conn.SetDeadline(time.Now().Add(freezeTimeout)); err != nil {
		return false, err
	}
	if err := client.execute("guest-fsfreeze-freeze", nil, nil); err != nil {
		return true, err
	}
	return true, nil
}

func (d *Driver) thawGuest(id string) error {
	deadline := time.Now().Add(thawTimeout)
	var last error
	for {
		last = d.thawOnce(id)
		if last == nil {
			return nil
		}
		if !time.Now().Before(deadline) {
			return last
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (d *Driver) thawOnce(id string) error {
	client, err := dialQGA(d.qgaPath(id), 5*time.Second)
	if err != nil {
		return err
	}
	defer func() { _ = client.close() }()
	if err := client.conn.SetDeadline(time.Now().Add(thawCommandTimeout)); err != nil {
		return err
	}
	return client.execute("guest-fsfreeze-thaw", nil, nil)
}

type qgaClient struct {
	conn net.Conn
	r    *bufio.Reader
}

func dialQGA(socket string, timeout time.Duration) (*qgaClient, error) {
	conn, err := net.DialTimeout("unix", socket, timeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	client := &qgaClient{conn: conn, r: bufio.NewReader(conn)}
	if err := client.sync(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func (c *qgaClient) close() error { return c.conn.Close() }

func (c *qgaClient) sync() error {
	token := time.Now().UnixNano() & 0x7fffffff
	if _, err := c.conn.Write([]byte{0xff}); err != nil {
		return err
	}
	if err := c.send("guest-sync-delimited", map[string]any{"id": token}); err != nil {
		return err
	}

	for {
		b, err := c.r.ReadByte()
		if err != nil {
			return err
		}
		if b == 0xff {
			break
		}
	}

	var reply struct {
		Return int64 `json:"return"`
	}
	if err := c.read(&reply); err != nil {
		return err
	}
	if reply.Return != token {
		return fmt.Errorf("guest agent sync mismatch")
	}
	return nil
}

func (c *qgaClient) send(cmd string, args any) error {
	payload := map[string]any{"execute": cmd}
	if args != nil {
		payload["arguments"] = args
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(append(data, '\n'))
	return err
}

func (c *qgaClient) read(out any) error {
	line := make([]byte, 0, 4096)
	for {
		fragment, err := c.r.ReadSlice('\n')
		if len(line)+len(fragment) > 1<<20 {
			return fmt.Errorf("guest agent response exceeds 1 MiB")
		}
		line = append(line, fragment...)
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			return err
		}
		break
	}
	return json.Unmarshal(line, out)
}

func (c *qgaClient) execute(cmd string, args, out any) error {
	if err := c.send(cmd, args); err != nil {
		return err
	}

	var reply struct {
		Return json.RawMessage `json:"return"`
		Error  *qmpError       `json:"error"`
	}
	if err := c.read(&reply); err != nil {
		return err
	}
	if reply.Error != nil {
		return reply.Error
	}
	if out != nil && len(reply.Return) > 0 {
		return json.Unmarshal(reply.Return, out)
	}
	return nil
}

func (c *qgaClient) hostname() string {
	var reply struct {
		HostName string `json:"host-name"`
	}
	if c.execute("guest-get-host-name", nil, &reply) != nil {
		return ""
	}
	return reply.HostName
}

func (c *qgaClient) osInfo() *GuestOSInfo {
	var raw struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		PrettyName    string `json:"pretty-name"`
		Version       string `json:"version"`
		VersionID     string `json:"version-id"`
		KernelRelease string `json:"kernel-release"`
		KernelVersion string `json:"kernel-version"`
		Machine       string `json:"machine"`
	}
	if c.execute("guest-get-osinfo", nil, &raw) != nil {
		return nil
	}
	return &GuestOSInfo{
		ID:            raw.ID,
		Name:          raw.Name,
		PrettyName:    raw.PrettyName,
		Version:       raw.Version,
		VersionID:     raw.VersionID,
		KernelRelease: raw.KernelRelease,
		KernelVersion: raw.KernelVersion,
		Machine:       raw.Machine,
	}
}

func (c *qgaClient) interfaces() []GuestInterface {
	var raw []struct {
		Name            string `json:"name"`
		HardwareAddress string `json:"hardware-address"`
		IPAddresses     []struct {
			Address string `json:"ip-address"`
			Type    string `json:"ip-address-type"`
			Prefix  int    `json:"prefix"`
		} `json:"ip-addresses"`
	}
	if c.execute("guest-network-get-interfaces", nil, &raw) != nil {
		return nil
	}

	result := make([]GuestInterface, 0, len(raw))
	for _, nic := range raw {
		iface := GuestInterface{Name: nic.Name, HardwareAddress: nic.HardwareAddress}
		for _, addr := range nic.IPAddresses {
			iface.IPAddresses = append(iface.IPAddresses, GuestIPAddress{Address: addr.Address, Type: addr.Type, Prefix: addr.Prefix})
		}
		result = append(result, iface)
	}
	return result
}

func (c *qgaClient) filesystems() []GuestFilesystem {
	var raw []struct {
		Name       string `json:"name"`
		Mountpoint string `json:"mountpoint"`
		Type       string `json:"type"`
		TotalBytes int64  `json:"total-bytes"`
		UsedBytes  int64  `json:"used-bytes"`
	}
	if c.execute("guest-get-fsinfo", nil, &raw) != nil {
		return nil
	}

	result := make([]GuestFilesystem, 0, len(raw))
	for _, fs := range raw {
		result = append(result, GuestFilesystem{Name: fs.Name, Mountpoint: fs.Mountpoint, Type: fs.Type, TotalBytes: fs.TotalBytes, UsedBytes: fs.UsedBytes})
	}
	return result
}
