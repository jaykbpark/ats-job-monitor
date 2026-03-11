package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	db, err := sql.Open("sqlite", trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.configure(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) configure(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure sqlite pragma %q: %w", pragma, err)
		}
	}

	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite database: %w", err)
	}

	return nil
}
