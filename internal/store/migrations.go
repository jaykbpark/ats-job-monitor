package store

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/db/migrations"
)

type MigrationRecord struct {
	Version   string `json:"version"`
	AppliedAt string `json:"appliedAt"`
}

func (s *Store) Migrate(ctx context.Context) error {
	if err := s.ensureMigrationTable(ctx); err != nil {
		return err
	}

	applied, err := s.appliedMigrationSet(ctx)
	if err != nil {
		return err
	}

	filenames, err := migrationFilenames()
	if err != nil {
		return err
	}

	for _, filename := range filenames {
		if _, exists := applied[filename]; exists {
			continue
		}

		contents, err := migrations.Files.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", filename, err)
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration transaction for %q: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %q: %w", filename, err)
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, filename); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %q: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %q: %w", filename, err)
		}
	}

	return nil
}

func (s *Store) AppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	if err := s.ensureMigrationTable(ctx); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT version, applied_at
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	records := make([]MigrationRecord, 0)
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.AppliedAt); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}

	return records, nil
}

func (s *Store) ensureMigrationTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func (s *Store) appliedMigrationSet(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	versions := map[string]struct{}{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan schema_migration version: %w", err)
		}
		versions[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}

	return versions, nil
}

func migrationFilenames() ([]string, error) {
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".sql") {
			filenames = append(filenames, name)
		}
	}

	sort.Strings(filenames)
	return filenames, nil
}
