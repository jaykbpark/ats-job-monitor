package store

import (
	"context"
	"fmt"
	"strings"
)

type SyncRun struct {
	ID               int64  `json:"id"`
	WatchTargetID    int64  `json:"watchTargetId"`
	Status           string `json:"status"`
	StartedAt        string `json:"startedAt"`
	FinishedAt       string `json:"finishedAt,omitempty"`
	FetchedJobsCount int    `json:"fetchedJobsCount"`
	MatchedJobsCount int    `json:"matchedJobsCount"`
	NewJobsCount     int    `json:"newJobsCount"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type RecordSyncRunParams struct {
	WatchTargetID    int64
	Status           string
	FetchedJobsCount int
	MatchedJobsCount int
	NewJobsCount     int
	ErrorMessage     string
}

func (s *Store) RecordSyncRun(ctx context.Context, params RecordSyncRunParams) (SyncRun, error) {
	status := strings.TrimSpace(params.Status)
	if status == "" {
		status = "succeeded"
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_runs (
			watch_target_id,
			status,
			finished_at,
			fetched_jobs_count,
			matched_jobs_count,
			new_jobs_count,
			error_message
		) VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?, ?, NULLIF(?, ''))
	`, params.WatchTargetID, status, params.FetchedJobsCount, params.MatchedJobsCount, params.NewJobsCount, strings.TrimSpace(params.ErrorMessage))
	if err != nil {
		return SyncRun{}, fmt.Errorf("insert sync run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return SyncRun{}, fmt.Errorf("read inserted sync run id: %w", err)
	}

	return s.GetSyncRun(ctx, id)
}

func (s *Store) GetSyncRun(ctx context.Context, id int64) (SyncRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id,
			watch_target_id,
			status,
			started_at,
			COALESCE(finished_at, ''),
			fetched_jobs_count,
			matched_jobs_count,
			new_jobs_count,
			COALESCE(error_message, '')
		FROM sync_runs
		WHERE id = ?
	`, id)

	run, err := scanSyncRun(row)
	if err != nil {
		return SyncRun{}, fmt.Errorf("get sync run %d: %w", id, err)
	}

	return run, nil
}

func (s *Store) ListSyncRunsByWatchTarget(ctx context.Context, watchTargetID int64) ([]SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			watch_target_id,
			status,
			started_at,
			COALESCE(finished_at, ''),
			fetched_jobs_count,
			matched_jobs_count,
			new_jobs_count,
			COALESCE(error_message, '')
		FROM sync_runs
		WHERE watch_target_id = ?
		ORDER BY id DESC
	`, watchTargetID)
	if err != nil {
		return nil, fmt.Errorf("list sync runs: %w", err)
	}
	defer rows.Close()

	runs := make([]SyncRun, 0)
	for rows.Next() {
		run, err := scanSyncRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sync run row: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync runs: %w", err)
	}

	return runs, nil
}

func scanSyncRun(row scanner) (SyncRun, error) {
	var run SyncRun
	if err := row.Scan(
		&run.ID,
		&run.WatchTargetID,
		&run.Status,
		&run.StartedAt,
		&run.FinishedAt,
		&run.FetchedJobsCount,
		&run.MatchedJobsCount,
		&run.NewJobsCount,
		&run.ErrorMessage,
	); err != nil {
		return SyncRun{}, err
	}

	return run, nil
}
