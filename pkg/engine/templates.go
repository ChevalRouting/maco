package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/storage"
)

type Template struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Spec        CreateVMParams `json:"spec"`
}

func (e *Engine) templatesDir() string { return e.paths.TemplatesDir() }

func (e *Engine) lockTemplates(ctx context.Context) (*os.File, error) {
	return storage.Lock(ctx, filepath.Join(e.paths.Root, "templates.lock"))
}

func (e *Engine) GetTemplate(id string) (Template, error) {
	var t Template
	if _, err := uuid.Parse(id); err != nil {
		return t, fmt.Errorf("invalid template ID")
	}

	data, err := os.ReadFile(filepath.Join(e.templatesDir(), id+".json"))
	if err != nil {
		return t, err
	}

	if err = json.Unmarshal(data, &t); err != nil {
		return t, err
	}

	return t, nil
}

func (e *Engine) ListTemplates() ([]Template, error) {
	result := []Template{}
	entries, err := os.ReadDir(e.templatesDir())
	if os.IsNotExist(err) {
		return result, nil
	}

	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		t, err := e.GetTemplate(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}

		result = append(result, t)
	}

	return result, nil
}

func (e *Engine) CreateTemplate(ctx context.Context, name, description string, spec CreateVMParams) (Template, error) {
	lock, err := e.lockTemplates(ctx)
	if err != nil {
		return Template{}, err
	}

	defer func() { _ = lock.Close() }()

	t := Template{ID: uuid.NewString(), Name: strings.TrimSpace(name), Description: strings.TrimSpace(description), Spec: spec}
	if err := validateTemplate(t); err != nil {
		return Template{}, err
	}

	t.Spec.Name = ""
	if err := e.writeTemplate(t); err != nil {
		return Template{}, err
	}

	return t, nil
}

func (e *Engine) UpdateTemplate(ctx context.Context, id, name, description string, spec CreateVMParams) (Template, error) {
	lock, err := e.lockTemplates(ctx)
	if err != nil {
		return Template{}, err
	}

	defer func() { _ = lock.Close() }()

	existing, err := e.GetTemplate(id)
	if err != nil {
		return Template{}, err
	}

	existing.Name = strings.TrimSpace(name)
	existing.Description = strings.TrimSpace(description)
	existing.Spec = spec
	existing.Spec.Name = ""
	if err := validateTemplate(existing); err != nil {
		return Template{}, err
	}

	if err := e.writeTemplate(existing); err != nil {
		return Template{}, err
	}

	return existing, nil
}

type TemplateInstanceParams struct {
	Name        string `json:"name" extensions:"x-maco-description=Unique VM name. Use an RFC 1123 hostname such as agent-vm." minLength:"1" maxLength:"63"`
	CPUs        int    `json:"cpus,omitempty" binding:"optional" extensions:"x-maco-description=Optional vCPU override. Omit to use the template value."`
	MemoryMiB   int    `json:"memory_mib,omitempty" binding:"optional" extensions:"x-maco-description=Optional memory override in MiB. Omit to use the template value."`
	DiskSizeGiB int    `json:"disk_size_gib,omitempty" binding:"optional" extensions:"x-maco-description=Optional boot disk override in GiB. Omit to use the template value."`
	Autostart   *bool  `json:"autostart,omitempty" binding:"optional" extensions:"x-maco-description=Optional autostart override. Omit to use the template value."`
}

func (e *Engine) TemplateInstance(id string, p TemplateInstanceParams) (CreateVMParams, error) {
	t, err := e.GetTemplate(id)
	if err != nil {
		return CreateVMParams{}, err
	}

	spec := t.Spec
	spec.Name = strings.TrimSpace(p.Name)
	if spec.Name == "" {
		return CreateVMParams{}, fmt.Errorf("name is required")
	}

	if !vmName.MatchString(spec.Name) {
		return CreateVMParams{}, fmt.Errorf("name must be 1-63 letters, digits, dots, underscores or hyphens")
	}

	if p.CPUs > 0 {
		spec.CPUs = p.CPUs
	}

	if p.MemoryMiB > 0 {
		spec.MemoryMiB = p.MemoryMiB
	}

	if p.DiskSizeGiB > 0 {
		spec.DiskSizeGiB = p.DiskSizeGiB
	}

	if p.Autostart != nil {
		spec.Autostart = *p.Autostart
	}

	return spec, nil
}

func (e *Engine) DeleteTemplate(ctx context.Context, id string) error {
	lock, err := e.lockTemplates(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()

	if _, err := e.GetTemplate(id); err != nil {
		return err
	}

	return os.Remove(filepath.Join(e.templatesDir(), id+".json"))
}

func (e *Engine) writeTemplate(t Template) error {
	if err := os.MkdirAll(e.templatesDir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(e.templatesDir(), t.ID+".json"), data, 0o600)
}

func validateTemplate(t Template) error {
	if t.Name == "" {
		return fmt.Errorf("name is required")
	}

	if !vmName.MatchString(t.Name) {
		return fmt.Errorf("name must be 1-63 letters, digits, dots, underscores or hyphens")
	}

	return nil
}
