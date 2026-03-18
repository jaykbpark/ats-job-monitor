package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/filters"
)

type WatchTarget struct {
	ID                int64               `json:"id"`
	Name              string              `json:"name"`
	Provider          string              `json:"provider"`
	BoardKey          string              `json:"boardKey"`
	SourceURL         string              `json:"sourceUrl,omitempty"`
	NotificationEmail string              `json:"notificationEmail,omitempty"`
	FiltersJSON       string              `json:"filtersJson"`
	Filters           filters.HardFilters `json:"filters"`
	Status            string              `json:"status"`
	LastSyncedAt      string              `json:"lastSyncedAt,omitempty"`
	CreatedAt         string              `json:"createdAt"`
	UpdatedAt         string              `json:"updatedAt"`
}

type CreateWatchTargetParams struct {
	Name              string
	Provider          string
	BoardKey          string
	SourceURL         string
	NotificationEmail string
	FiltersJSON       string
	Status            string
}

type UpdateWatchTargetParams struct {
	ID                int64
	Name              *string
	SourceURL         *string
	NotificationEmail *string
	FiltersJSON       *string
	Status            *string
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

	normalizedFiltersJSON, _, err := filters.NormalizeHardFiltersJSON(filtersJSON)
	if err != nil {
		return WatchTarget{}, fmt.Errorf("invalid hard filters: %w", err)
	}
	filtersJSON = normalizedFiltersJSON

	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = "active"
	}

	notificationEmail, err := normalizeNotificationEmail(params.NotificationEmail)
	if err != nil {
		return WatchTarget{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO watch_targets (name, provider, board_key, source_url, notification_email, filters_json, status)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)
	`, name, provider, boardKey, strings.TrimSpace(params.SourceURL), notificationEmail, filtersJSON, status)
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
			COALESCE(notification_email, ''),
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

func (s *Store) UpdateWatchTarget(ctx context.Context, params UpdateWatchTargetParams) (WatchTarget, error) {
	target, err := s.GetWatchTarget(ctx, params.ID)
	if err != nil {
		return WatchTarget{}, err
	}

	name := target.Name
	if params.Name != nil {
		name = strings.TrimSpace(*params.Name)
		if name == "" {
			return WatchTarget{}, fmt.Errorf("watch target name is required")
		}
	}

	sourceURL := target.SourceURL
	if params.SourceURL != nil {
		sourceURL = strings.TrimSpace(*params.SourceURL)
	}

	notificationEmail := target.NotificationEmail
	if params.NotificationEmail != nil {
		notificationEmail, err = normalizeNotificationEmail(*params.NotificationEmail)
		if err != nil {
			return WatchTarget{}, err
		}
	}

	filtersJSON := target.FiltersJSON
	if params.FiltersJSON != nil {
		filtersJSON = strings.TrimSpace(*params.FiltersJSON)
		if filtersJSON == "" {
			filtersJSON = "{}"
		}

		normalizedFiltersJSON, _, err := filters.NormalizeHardFiltersJSON(filtersJSON)
		if err != nil {
			return WatchTarget{}, fmt.Errorf("invalid hard filters: %w", err)
		}
		filtersJSON = normalizedFiltersJSON
	}

	status := target.Status
	if params.Status != nil {
		status = strings.TrimSpace(*params.Status)
		if status == "" {
			return WatchTarget{}, fmt.Errorf("watch target status is required")
		}
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE watch_targets
		SET name = ?,
			source_url = NULLIF(?, ''),
			notification_email = NULLIF(?, ''),
			filters_json = ?,
			status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, sourceURL, notificationEmail, filtersJSON, status, params.ID); err != nil {
		return WatchTarget{}, fmt.Errorf("update watch target %d: %w", params.ID, err)
	}

	return s.GetWatchTarget(ctx, params.ID)
}

func (s *Store) ListWatchTargets(ctx context.Context) ([]WatchTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			name,
			provider,
			board_key,
			COALESCE(source_url, ''),
			COALESCE(notification_email, ''),
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
		&target.NotificationEmail,
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

	parsedFilters, err := filters.ParseHardFilters(target.FiltersJSON)
	if err != nil {
		return WatchTarget{}, fmt.Errorf("parse stored hard filters: %w", err)
	}
	target.Filters = parsedFilters

	return target, nil
}

func normalizeNotificationEmail(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	address, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", fmt.Errorf("notification email is invalid")
	}

	return strings.TrimSpace(address.Address), nil
}
