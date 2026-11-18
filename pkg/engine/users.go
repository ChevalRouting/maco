package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/auth"
	"github.com/m-vinc/maco/pkg/types"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func (e *Engine) Authenticate(ctx context.Context, username, password string) (types.User, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return types.User{}, err
	}

	u, ok, err := database.GetUserByUsername(ctx, username)
	if err != nil {
		return types.User{}, err
	}

	if !ok || !auth.VerifyPassword(u.PasswordHash, password) {
		return types.User{}, ErrInvalidCredentials
	}

	return u, nil
}

func (e *Engine) AddUser(ctx context.Context, username, password, role string) error {
	if !auth.ValidRole(role) {
		return fmt.Errorf("role must be %q or %q", auth.RoleAdmin, auth.RoleViewer)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	return database.CreateUser(ctx, types.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().Unix(),
	})
}

func (e *Engine) ListUsers(ctx context.Context) ([]types.User, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	return database.ListUsers(ctx)
}

func (e *Engine) SetPassword(ctx context.Context, username, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	return database.UpdateUserPassword(ctx, username, hash)
}

func (e *Engine) DeleteUser(ctx context.Context, username string) error {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}

	return database.DeleteUser(ctx, username)
}

func (e *Engine) EnsureAdmin(ctx context.Context, password string) (created bool, generated string, err error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return false, "", err
	}

	count, err := database.CountUsers(ctx)
	if err != nil {
		return false, "", err
	}

	if count > 0 {
		return false, "", nil
	}

	if password == "" {
		if password, err = auth.GeneratePassword(); err != nil {
			return false, "", err
		}

		generated = password
	}

	if err := e.AddUser(ctx, "admin", password, auth.RoleAdmin); err != nil {
		return false, "", err
	}

	return true, generated, nil
}
