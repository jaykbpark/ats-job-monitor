package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
)

type JobRecord struct {
	ID             int64  `json:"id"`
	WatchTargetID  int64  `json:"watchTargetId"`
	ExternalJobID  string `json:"externalJobId"`
	Title          string `json:"title"`
	Location       string `json:"location,omitempty"`
	Department     string `json:"department,omitempty"`
	Team           string `json:"team,omitempty"`
	EmploymentType string `json:"employmentType,omitempty"`
	JobURL         string `json:"jobUrl"`
	MetadataJSON   string `json:"metadataJson"`
	RawJSON        string `json:"rawJson"`
	FirstSeenAt    string `json:"firstSeenAt"`
	LastSeenAt     string `json:"lastSeenAt"`
	MatchedAt      string `json:"matchedAt,omitempty"`
	IsActive       bool   `json:"isActive"`
}

type SyncJobsResult struct {
	FetchedJobsCount int `json:"fetchedJobsCount"`
	MatchedJobsCount int `json:"matchedJobsCount"`
	NewJobsCount     int `json:"newJobsCount"`
}

func (s *Store) SyncJobs(ctx context.Context, watchTargetID int64, jobs []providers.Job) (SyncJobsResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SyncJobsResult{}, fmt.Errorf("begin sync jobs transaction: %w", err)
	}

	existingIDs, err := existingJobIDs(ctx, tx, watchTargetID)
	if err != nil {
		_ = tx.Rollback()
		return SyncJobsResult{}, err
	}

	newJobsCount := 0
	incomingIDs := make([]string, 0, len(jobs))
	for _, job := range jobs {
		incomingIDs = append(incomingIDs, job.ExternalJobID)
		if _, exists := existingIDs[job.ExternalJobID]; !exists {
			newJobsCount++
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO jobs (
				watch_target_id,
				external_job_id,
				title,
				location,
				department,
				team,
				employment_type,
				job_url,
				metadata_json,
				raw_json,
				matched_at,
				last_seen_at,
				is_active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
			ON CONFLICT(watch_target_id, external_job_id) DO UPDATE SET
				title = excluded.title,
				location = excluded.location,
				department = excluded.department,
				team = excluded.team,
				employment_type = excluded.employment_type,
				job_url = excluded.job_url,
				metadata_json = excluded.metadata_json,
				raw_json = excluded.raw_json,
				last_seen_at = CURRENT_TIMESTAMP,
				matched_at = CURRENT_TIMESTAMP,
				is_active = 1
		`, watchTargetID, job.ExternalJobID, job.Title, job.Location, job.Department, job.Team, job.EmploymentType, job.JobURL, defaultJSON(job.MetadataJSON), job.RawJSON); err != nil {
			_ = tx.Rollback()
			return SyncJobsResult{}, fmt.Errorf("upsert synced job %q: %w", job.ExternalJobID, err)
		}
	}

	if len(incomingIDs) == 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET is_active = 0
			WHERE watch_target_id = ?
		`, watchTargetID); err != nil {
			_ = tx.Rollback()
			return SyncJobsResult{}, fmt.Errorf("deactivate stale jobs: %w", err)
		}
	} else {
		query, args := buildDeactivateMissingJobsQuery(watchTargetID, incomingIDs)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			_ = tx.Rollback()
			return SyncJobsResult{}, fmt.Errorf("deactivate missing jobs: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE watch_targets
		SET last_synced_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, watchTargetID); err != nil {
		_ = tx.Rollback()
		return SyncJobsResult{}, fmt.Errorf("update watch target last_synced_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SyncJobsResult{}, fmt.Errorf("commit sync jobs transaction: %w", err)
	}

	return SyncJobsResult{
		FetchedJobsCount: len(jobs),
		MatchedJobsCount: len(jobs),
		NewJobsCount:     newJobsCount,
	}, nil
}

func (s *Store) ListJobsByWatchTarget(ctx context.Context, watchTargetID int64) ([]JobRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			watch_target_id,
			external_job_id,
			title,
			COALESCE(location, ''),
			COALESCE(department, ''),
			COALESCE(team, ''),
			COALESCE(employment_type, ''),
			job_url,
			metadata_json,
			raw_json,
			first_seen_at,
			last_seen_at,
			COALESCE(matched_at, ''),
			is_active
		FROM jobs
		WHERE watch_target_id = ?
		ORDER BY id ASC
	`, watchTargetID)
	if err != nil {
		return nil, fmt.Errorf("list jobs by watch target: %w", err)
	}
	defer rows.Close()

	jobs := make([]JobRecord, 0)
	for rows.Next() {
		job, err := scanJobRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, nil
}

func existingJobIDs(ctx context.Context, tx *sql.Tx, watchTargetID int64) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT external_job_id
		FROM jobs
		WHERE watch_target_id = ?
	`, watchTargetID)
	if err != nil {
		return nil, fmt.Errorf("query existing job ids: %w", err)
	}
	defer rows.Close()

	ids := map[string]struct{}{}
	for rows.Next() {
		var externalJobID string
		if err := rows.Scan(&externalJobID); err != nil {
			return nil, fmt.Errorf("scan existing job id: %w", err)
		}
		ids[externalJobID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing job ids: %w", err)
	}

	return ids, nil
}

func buildDeactivateMissingJobsQuery(watchTargetID int64, incomingIDs []string) (string, []any) {
	placeholders := make([]string, 0, len(incomingIDs))
	args := make([]any, 0, len(incomingIDs)+1)
	args = append(args, watchTargetID)

	for _, id := range incomingIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE jobs
		SET is_active = 0
		WHERE watch_target_id = ?
		  AND external_job_id NOT IN (%s)
	`, strings.Join(placeholders, ", "))

	return query, args
}

func scanJobRecord(row scanner) (JobRecord, error) {
	var job JobRecord
	var isActive int
	if err := row.Scan(
		&job.ID,
		&job.WatchTargetID,
		&job.ExternalJobID,
		&job.Title,
		&job.Location,
		&job.Department,
		&job.Team,
		&job.EmploymentType,
		&job.JobURL,
		&job.MetadataJSON,
		&job.RawJSON,
		&job.FirstSeenAt,
		&job.LastSeenAt,
		&job.MatchedAt,
		&isActive,
	); err != nil {
		return JobRecord{}, err
	}

	job.IsActive = isActive == 1
	return job, nil
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}
