package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sync"
	"time"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/storage"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	*sql.DB
	queries *generated.Queries
}

func DSN(path string) string {
	return path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(on)&_txlock=immediate"
}

func Open(path string) (*DB, error) {
	database, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(8)
	database.SetMaxIdleConns(8)
	database.SetConnMaxIdleTime(5 * time.Minute)
	return New(database), nil
}

func New(database *sql.DB) *DB {
	return &DB{DB: database, queries: generated.New(database)}
}

func InitDB(ctx context.Context, path string) (*DB, error) {
	database, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("open migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, database.DB, migrationFS)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return database, nil
}

var (
	sharedMu sync.Mutex
	shared   = map[string]*DB{}
)

func Shared(ctx context.Context, path string) (*DB, error) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if database, ok := shared[path]; ok {
		return database, nil
	}

	lock, err := storage.Lock(ctx, path+".init.lock")
	if err != nil {
		return nil, err
	}

	defer func() { _ = lock.Close() }()

	database, err := InitDB(ctx, path)
	if err != nil {
		return nil, err
	}

	shared[path] = database
	return database, nil
}
