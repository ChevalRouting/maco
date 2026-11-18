package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/types"
)

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func (db *DB) CreateAPIKey(ctx context.Context, key types.APIKey, token string) error {
	return db.queries.CreateAPIKey(ctx, generated.CreateAPIKeyParams{
		ID:        key.ID,
		UserID:    key.UserID,
		Name:      key.Name,
		TokenHash: hashToken(token),
		Prefix:    key.Prefix,
		CreatedAt: key.CreatedAt,
	})
}

func (db *DB) ListAPIKeysByUser(ctx context.Context, userID string) ([]types.APIKey, error) {
	rows, err := db.queries.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	keys := make([]types.APIKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, types.APIKey{
			ID:         row.ID,
			UserID:     row.UserID,
			Name:       row.Name,
			Prefix:     row.Prefix,
			CreatedAt:  row.CreatedAt,
			LastUsedAt: row.LastUsedAt,
			RevokedAt:  row.RevokedAt,
		})
	}

	return keys, nil
}

func (db *DB) GetAPIKeyByToken(ctx context.Context, token string) (types.APIKey, bool, error) {
	row, err := db.queries.GetActiveAPIKeyByHash(ctx, hashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return types.APIKey{}, false, nil
	}

	if err != nil {
		return types.APIKey{}, false, err
	}

	return types.APIKey{
		ID:         row.ID,
		UserID:     row.UserID,
		Name:       row.Name,
		Prefix:     row.Prefix,
		CreatedAt:  row.CreatedAt,
		LastUsedAt: row.LastUsedAt,
		RevokedAt:  row.RevokedAt,
	}, true, nil
}

func (db *DB) RevokeAPIKey(ctx context.Context, userID, id string) (bool, error) {
	revokedAt := time.Now().Unix()
	rows, err := db.queries.RevokeAPIKey(ctx, generated.RevokeAPIKeyParams{
		RevokedAt: &revokedAt,
		ID:        id,
		UserID:    userID,
	})
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

func (db *DB) TouchAPIKey(ctx context.Context, id string) error {
	usedAt := time.Now().Unix()
	return db.queries.TouchAPIKey(ctx, generated.TouchAPIKeyParams{LastUsedAt: &usedAt, ID: id})
}
