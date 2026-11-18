package engine

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

const APIKeyPrefix = "maco_"

func generateAPIKey() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}

	return APIKeyPrefix + base64.RawURLEncoding.EncodeToString(secret), nil
}

func (e *Engine) CreateAPIKey(ctx context.Context, userID, name string) (types.APIKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.APIKey{}, "", fmt.Errorf("name is required")
	}

	token, err := generateAPIKey()
	if err != nil {
		return types.APIKey{}, "", err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return types.APIKey{}, "", err
	}

	key := types.APIKey{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		Prefix:    token[:12],
		CreatedAt: time.Now().Unix(),
	}
	if err := database.CreateAPIKey(ctx, key, token); err != nil {
		return types.APIKey{}, "", err
	}

	return key, token, nil
}

func (e *Engine) ListAPIKeys(ctx context.Context, userID string) ([]types.APIKey, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	return database.ListAPIKeysByUser(ctx, userID)
}

func (e *Engine) RevokeAPIKey(ctx context.Context, userID, id string) (bool, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return false, err
	}

	return database.RevokeAPIKey(ctx, userID, id)
}

func (e *Engine) AuthenticateAPIKey(ctx context.Context, token string) (types.User, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return types.User{}, err
	}

	key, ok, err := database.GetAPIKeyByToken(ctx, token)
	if err != nil {
		return types.User{}, err
	}

	if !ok {
		return types.User{}, ErrInvalidCredentials
	}

	u, ok, err := database.GetUserByID(ctx, key.UserID)
	if err != nil {
		return types.User{}, err
	}

	if !ok {
		return types.User{}, ErrInvalidCredentials
	}

	_ = database.TouchAPIKey(ctx, key.ID)
	return u, nil
}
