package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func OpenDefault(ctx context.Context) (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate application data directory: %w", err)
	}
	databaseDir := filepath.Join(configDir, "KasirKita")
	if err := os.MkdirAll(databaseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	return Open(ctx, filepath.Join(databaseDir, "kasir-kita.sqlite"))
}

func Open(ctx context.Context, databasePath string) (*Store, error) {
	db, err := sql.Open("sqlite", dataSourceName(databasePath))
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}
	if err := applyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate SQLite database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func dataSourceName(databasePath string) string {
	absPath, err := filepath.Abs(databasePath)
	if err == nil {
		databasePath = absPath
	}
	path := filepath.ToSlash(databasePath)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	uri := url.URL{Scheme: "file", Path: path}
	query := url.Values{}
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	uri.RawQuery = query.Encode()
	return uri.String()
}
