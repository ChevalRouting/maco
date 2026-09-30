package engine

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/m-vinc/maco/pkg/types"
)

const (
	guestTemplateLeft  = "[["
	guestTemplateRight = "]]"
)

type guestTemplateDisk struct {
	Name    string
	SizeGiB int
}

type guestTemplateInterface struct {
	Network     string
	MAC         string
	Addresses   []string
	Gateway     string
	Nameservers []string
}

type guestTemplateData struct {
	Name        string
	Hostname    string
	CPUs        int
	MemoryMiB   int
	DiskSizeGiB int
	SSHKey      string
	SSHKeys     []string
	Network     string
	Addresses   []string
	Gateway     string
	Nameservers []string
	Disks       []guestTemplateDisk
	Interfaces  []guestTemplateInterface
}

func renderGuestValue(field, content string, data guestTemplateData) (string, error) {
	if !strings.Contains(content, guestTemplateLeft) {
		return content, nil
	}
	tmpl, err := template.New(field).
		Delims(guestTemplateLeft, guestTemplateRight).
		Option("missingkey=error").
		Parse(content)
	if err != nil {
		return "", fmt.Errorf("guest %s template: %w", field, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("guest %s template: %w", field, err)
	}
	return buf.String(), nil
}

func renderGuestSetup(setup types.GuestSetup, data guestTemplateData) (types.GuestSetup, error) {
	rendered := setup
	var err error
	if rendered.Raw, err = renderGuestValue("raw", setup.Raw, data); err != nil {
		return setup, err
	}
	if len(setup.Files) > 0 {
		files := make([]types.GuestFile, len(setup.Files))
		for i, file := range setup.Files {
			content, err := renderGuestValue("file "+file.Name, file.Content, data)
			if err != nil {
				return setup, err
			}
			files[i] = types.GuestFile{Name: file.Name, Content: content}
		}
		rendered.Files = files
	}
	return rendered, nil
}

func guestDataFromManifest(m *types.VMManifest, hostname string, interfaces []types.VMInterface) guestTemplateData {
	data := guestTemplateData{
		Name:        m.Name,
		Hostname:    hostname,
		CPUs:        m.CPUs,
		MemoryMiB:   m.MemoryMiB,
		DiskSizeGiB: m.DiskSizeGiB,
		SSHKeys:     m.SSHKeys,
		Network:     m.Network,
	}
	if len(m.SSHKeys) > 0 {
		data.SSHKey = m.SSHKeys[0]
	}
	for _, disk := range m.Disks {
		data.Disks = append(data.Disks, guestTemplateDisk{Name: disk.Name, SizeGiB: disk.SizeGiB})
	}
	for _, iface := range interfaces {
		data.Interfaces = append(data.Interfaces, guestTemplateInterface{
			Network:     iface.Network,
			MAC:         iface.MAC,
			Addresses:   iface.Addresses,
			Gateway:     iface.Gateway,
			Nameservers: iface.Nameservers,
		})
	}
	if len(interfaces) > 0 {
		primary := interfaces[0]
		data.Addresses = primary.Addresses
		data.Gateway = primary.Gateway
		data.Nameservers = primary.Nameservers
	}
	return data
}
