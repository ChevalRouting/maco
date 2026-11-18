package db

import (
	"context"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/types"
)

func (db *DB) CreateUserSSHKey(ctx context.Context, key types.SSHKey) error {
	return db.queries.CreateUserSSHKey(ctx, generated.CreateUserSSHKeyParams{
		ID:        key.ID,
		UserID:    key.UserID,
		Name:      key.Name,
		PublicKey: key.PublicKey,
		CreatedAt: key.CreatedAt,
	})
}

func (db *DB) ListUserSSHKeys(ctx context.Context, userID string) ([]types.SSHKey, error) {
	rows, err := db.queries.ListUserSSHKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	keys := make([]types.SSHKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, types.SSHKey{
			ID:        row.ID,
			UserID:    row.UserID,
			Name:      row.Name,
			PublicKey: row.PublicKey,
			CreatedAt: row.CreatedAt,
		})
	}

	return keys, nil
}

func (db *DB) DeleteUserSSHKey(ctx context.Context, userID, id string) (bool, error) {
	rows, err := db.queries.DeleteUserSSHKey(ctx, generated.DeleteUserSSHKeyParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}
