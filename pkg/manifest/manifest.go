package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/m-vinc/maco/pkg/storage"
	"github.com/m-vinc/maco/pkg/types"
	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("vm not found")

type Store struct {
	dir      string
	validate *validator.Validate
}

func NewStore(dir string) *Store {
	return &Store{dir: dir, validate: validator.New(validator.WithRequiredStructEnabled())}
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".yml")
}

func (s *Store) Parse(data []byte) (*types.VMManifest, error) {
	var m types.VMManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	if err := s.validate.Struct(&m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &m, nil
}

func (s *Store) Save(m *types.VMManifest) error {
	if err := storage.ValidateID(m.ID); err != nil {
		return err
	}

	if err := s.validate.Struct(m); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}

	return storage.WriteFile(s.path(m.ID), data, 0o600)
}

func (s *Store) Load(id string) (*types.VMManifest, error) {
	if err := storage.ValidateID(id); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	m, err := s.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path(id), err)
	}

	if m.ID != id {
		return nil, fmt.Errorf("manifest ID does not match filename")
	}

	return m, nil
}

func (s *Store) List() ([]*types.VMManifest, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	manifests := make([]*types.VMManifest, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yml" {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".yml")
		m, err := s.Load(id)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, m)
	}

	return manifests, nil
}

func (s *Store) Delete(id string) error {
	err := os.Remove(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}

	return err
}

func (s *Store) Resolve(ref string) (*types.VMManifest, error) {
	if m, err := s.Load(ref); err == nil {
		return m, nil
	}

	manifests, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, m := range manifests {
		if m.Name == ref {
			return m, nil
		}
	}

	return nil, ErrNotFound
}
