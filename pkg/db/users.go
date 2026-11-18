package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/types"
)

func user(row generated.User) types.User {
	return types.User{
		ID:           row.ID,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		CreatedAt:    row.CreatedAt,
	}
}

func (db *DB) CreateUser(ctx context.Context, u types.User) error {
	return db.queries.CreateUser(ctx, generated.CreateUserParams{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		CreatedAt:    u.CreatedAt,
	})
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (types.User, bool, error) {
	row, err := db.queries.GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return types.User{}, false, nil
	}

	if err != nil {
		return types.User{}, false, err
	}

	return user(row), true, nil
}

func (db *DB) GetUserByID(ctx context.Context, id string) (types.User, bool, error) {
	row, err := db.queries.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return types.User{}, false, nil
	}

	if err != nil {
		return types.User{}, false, err
	}

	return user(row), true, nil
}

func (db *DB) ListUsers(ctx context.Context) ([]types.User, error) {
	rows, err := db.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]types.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, user(row))
	}

	return users, nil
}

func (db *DB) CountUsers(ctx context.Context) (int64, error) {
	return db.queries.CountUsers(ctx)
}

func (db *DB) UpdateUserPassword(ctx context.Context, username, hash string) error {
	return db.queries.UpdateUserPassword(ctx, generated.UpdateUserPasswordParams{
		PasswordHash: hash,
		Username:     username,
	})
}

func (db *DB) DeleteUser(ctx context.Context, username string) error {
	return db.queries.DeleteUser(ctx, username)
}
