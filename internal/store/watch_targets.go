package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type WatchTarget struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	BoardKey     string `json:"boardKey"`
	SourceURL    string `json:"sourceUrl,omitempty"`
	FiltersJSON  string `json:"filtersJson"`
	Status       string `json:"status"`
	LastSyncedAt string `json:"lastSyncedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type CreateWatchTargetParams struct {
	Name        string
	Provider    string
	BoardKey    string
	SourceURL   string
	FiltersJSON string
	Status      string
}

func (s *Store) CreateWatchTarget(ctx context.Context, params CreateWatchTargetParams) (WatchTarget, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return WatchTarget{}, fmt.Errorf("watch target name is required")
	}

	provider := strings.TrimSpace(params.Provider)
	if provider == "" {
		return WatchTarget{}, fmt.Errorf("watch target provider is required")
	}

	boardKey := strings.TrimSpace(params.BoardKey)
	if boardKey == "" {
		return WatchTarget{}, fmt.Errorf("watch target board key is required")
	}

	filtersJSON := strings.TrimSpace(params.FiltersJSON)
	if filtersJSON == "" {
		filtersJSON = "{}"
	}

	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = "active"
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO watch_targets (name, provider, board_key, source_url, filters_json, status)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, ?)
	`, name, provider, boardKey, strings.TrimSpace(params.SourceURL), filtersJSON, status)
	if err != nil {
		return WatchTarget{}, fmt.Errorf("insert watch target: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return WatchTarget{}, fmt.Errorf("read inserted watch target id: %w", err)
	}

	return s.GetWatchTarget(ctx, id)
}

func (s *Store) GetWatchTarget(ctx context.Context, id int64) (WatchTarget, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			name,
			provider,
			board_key,
			COALESCE(source_url, ''),
			filters_json,
			status,
			COALESCE(last_synced_at, ''),
			created_at,
			updated_at
		FROM watch_targets
		WHERE id = ?
	`, id)

	target, err := scanWatchTarget(row)
	if err != nil {
		return WatchTarget{}, fmt.Errorf("get watch target %d: %w", id, err)
	}

	return target, nil
}

func (s *Store) ListWatchTargets(ctx context.Context) ([]WatchTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			provider,
			board_key,
			COALESCE(source_url, ''),
			filters_json,
			status,
			COALESCE(last_synced_at, ''),
			created_at,
			updated_at
		FROM watch_targets
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list watch targets: %w", err)
	}
	defer rows.Close()

	targets := make([]WatchTarget, 0)
	for rows.Next() {
		target, err := scanWatchTarget(rows)
		if err != nil {
			return nil, fmt.Errorf("scan watch target row: %w", err)
		}
		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watch targets: %w", err)
	}

	return targets, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanWatchTarget(row scanner) (WatchTarget, error) {
	var target WatchTarget
	if err := row.Scan(
		&target.ID,
		&target.Name,
		&target.Provider,
		&target.BoardKey,
		&target.SourceURL,
		&target.FiltersJSON,
		&target.Status,
		&target.LastSyncedAt,
		&target.CreatedAt,
		&target.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return WatchTarget{}, err
		}
		return WatchTarget{}, err
	}

	return target, nil
}
