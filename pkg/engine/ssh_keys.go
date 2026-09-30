package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

var sshPublicKey = regexp.MustCompile(`^(ssh-|ecdsa-|sk-)[^\s]+\s+[A-Za-z0-9+/]+={0,3}(\s.*)?$`)

func (e *Engine) AddSSHKey(ctx context.Context, userID, name, key string) (types.SSHKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.SSHKey{}, fmt.Errorf("name is required")
	}

	key = strings.TrimSpace(key)
	if !sshPublicKey.MatchString(key) {
		return types.SSHKey{}, fmt.Errorf("a complete SSH public key is required")
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return types.SSHKey{}, err
	}

	sshKey := types.SSHKey{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		PublicKey: key,
		CreatedAt: time.Now().Unix(),
	}
	if err := database.CreateUserSSHKey(ctx, sshKey); err != nil {
		return types.SSHKey{}, err
	}

	return sshKey, nil
}

func (e *Engine) ListSSHKeys(ctx context.Context, userID string) ([]types.SSHKey, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	return database.ListUserSSHKeys(ctx, userID)
}

func (e *Engine) DeleteSSHKey(ctx context.Context, userID, id string) (bool, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return false, err
	}

	return database.DeleteUserSSHKey(ctx, userID, id)
}

func (e *Engine) SSHKeyValues(ctx context.Context, userID string) ([]string, error) {
	keys, err := e.ListSSHKeys(ctx, userID)
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key.PublicKey)
	}

	return values, nil
}
