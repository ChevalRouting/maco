package engine

import (
	"strings"

	"github.com/m-vinc/maco/pkg/provision"
	"github.com/m-vinc/maco/pkg/provision/cloudinit"
	"github.com/m-vinc/maco/pkg/provision/ignition"
	"github.com/m-vinc/maco/pkg/types"
)

func init() {
	provision.Register(cloudinit.Provisioner{})
	provision.Register(ignition.Provisioner{})
}

func provisionFiles(files []types.GuestFile) []provision.File {
	out := make([]provision.File, 0, len(files))
	for _, file := range files {
		out = append(out, provision.File{Name: file.Name, Content: file.Content})
	}
	return out
}

func primaryFile(provisioner string) string {
	if provisioner == types.ProvisionerIgnition {
		return "config.bu"
	}
	return "user-data"
}

func guestDeliveryFiles(prov provision.Provisioner, setup types.GuestSetup, params provision.Params) ([]provision.File, error) {
	if setup.Mode == types.GuestModeRaw {
		if len(setup.Files) > 0 {
			return provisionFiles(setup.Files), nil
		}
		if strings.TrimSpace(setup.Raw) != "" {
			files, err := prov.Render(params)
			if err != nil {
				return nil, err
			}
			return overlayFile(files, primaryFile(setup.Provisioner), setup.Raw), nil
		}
	}
	return prov.Render(params)
}

func guestFilesHaveContent(setup types.GuestSetup) bool {
	for _, file := range setup.Files {
		if strings.TrimSpace(file.Content) != "" {
			return true
		}
	}
	return strings.TrimSpace(setup.Raw) != ""
}

func overlayFile(files []provision.File, name, content string) []provision.File {
	for i := range files {
		if files[i].Name == name {
			files[i].Content = content
			return files
		}
	}
	return append(files, provision.File{Name: name, Content: content})
}
