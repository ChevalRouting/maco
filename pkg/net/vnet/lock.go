package vnet

import (
	"context"
	"github.com/m-vinc/maco/pkg/storage"
	"os"
	"path/filepath"
)

func (s *Store) Lock() (*os.File, error) { return s.LockContext(context.Background()) }
func (s *Store) LockContext(ctx context.Context) (*os.File, error) {
	return storage.Lock(ctx, filepath.Join(s.dir, ".host.lock"))
}
