package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jaykbpark/ats-job-monitor/internal/matching"
	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/signals"
)

type JobRecord struct {
	ID                       int64    `json:"id"`
	WatchTargetID            int64    `json:"watchTargetId"`
	ExternalJobID            string   `json:"externalJobId"`
	Title                    string   `json:"title"`
	Location                 string   `json:"location,omitempty"`
	Department               string   `json:"department,omitempty"`
	Team                     string   `json:"team,omitempty"`
	EmploymentType           string   `json:"employmentType,omitempty"`
	JobURL                   string   `json:"jobUrl"`
	MetadataJSON             string   `json:"metadataJson"`
	RawJSON                  string   `json:"rawJson"`
	SearchText               string   `json:"searchText"`
	NormalizedLocation       string   `json:"normalizedLocation"`
	IsRemote                 bool     `json:"isRemote"`
	NormalizedEmploymentType string   `json:"normalizedEmploymentType"`
	Seniority                string   `json:"seniority"`
	MinYearsExperience       *int     `json:"minYearsExperience,omitempty"`
	MaxYearsExperience       *int     `json:"maxYearsExperience,omitempty"`
	ExperienceConfidence     string   `json:"experienceConfidence"`
	IsMatch                  bool     `json:"isMatch"`
	MatchReasons             []string `json:"matchReasons"`
	HardFailures             []string `json:"hardFailures"`
	FirstSeenAt              string   `json:"firstSeenAt"`
	LastSeenAt               string   `json:"lastSeenAt"`
	MatchedAt                string   `json:"matchedAt,omitempty"`
	IsActive                 bool     `json:"isActive"`
}

type SyncedJob struct {
	Job     providers.Job
	Signals signals.JobSignals
	Match   matching.Result
}

type SyncJobsResult struct {
	FetchedJobsCount int `json:"fetchedJobsCount"`
	MatchedJobsCount int `json:"matchedJobsCount"`
	NewJobsCount     int `json:"newJobsCount"`
}

func (s *Store) SyncJobs(ctx context.Context, watchTargetID int64, jobs []SyncedJob) (SyncJobsResult, error) {
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
	matchedJobsCount := 0
	incomingIDs := make([]string, 0, len(jobs))
	for _, syncedJob := range jobs {
		job := syncedJob.Job
		jobSignals := syncedJob.Signals
		matchResult := syncedJob.Match

		incomingIDs = append(incomingIDs, job.ExternalJobID)
		if _, exists := existingIDs[job.ExternalJobID]; !exists {
			newJobsCount++
		}
		if matchResult.Matched {
			matchedJobsCount++
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
				search_text,
				normalized_location,
				is_remote,
				normalized_employment_type,
				seniority,
				min_years_experience,
				max_years_experience,
				experience_confidence,
				is_match,
				match_reasons_json,
				hard_failures_json,
				matched_at,
				last_seen_at,
				is_active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CASE WHEN ? = 1 THEN CURRENT_TIMESTAMP ELSE NULL END, CURRENT_TIMESTAMP, 1)
			ON CONFLICT(watch_target_id, external_job_id) DO UPDATE SET
				title = excluded.title,
				location = excluded.location,
				department = excluded.department,
				team = excluded.team,
				employment_type = excluded.employment_type,
				job_url = excluded.job_url,
				metadata_json = excluded.metadata_json,
				raw_json = excluded.raw_json,
				search_text = excluded.search_text,
				normalized_location = excluded.normalized_location,
				is_remote = excluded.is_remote,
				normalized_employment_type = excluded.normalized_employment_type,
				seniority = excluded.seniority,
				min_years_experience = excluded.min_years_experience,
				max_years_experience = excluded.max_years_experience,
				experience_confidence = excluded.experience_confidence,
				is_match = excluded.is_match,
				match_reasons_json = excluded.match_reasons_json,
				hard_failures_json = excluded.hard_failures_json,
				last_seen_at = CURRENT_TIMESTAMP,
				matched_at = CASE WHEN excluded.is_match = 1 THEN CURRENT_TIMESTAMP ELSE NULL END,
				is_active = 1
		`, watchTargetID, job.ExternalJobID, job.Title, job.Location, job.Department, job.Team, job.EmploymentType, job.JobURL, defaultJSON(job.MetadataJSON), job.RawJSON, jobSignals.SearchText, jobSignals.NormalizedLocation, boolToInt(jobSignals.IsRemote), jobSignals.NormalizedEmploymentType, jobSignals.Seniority, nullableInt(jobSignals.MinYearsExperience), nullableInt(jobSignals.MaxYearsExperience), jobSignals.ExperienceConfidence, boolToInt(matchResult.Matched), defaultJSONArray(matchResult.MatchReasons), defaultJSONArray(matchResult.HardFailures), boolToInt(matchResult.Matched)); err != nil {
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
		MatchedJobsCount: matchedJobsCount,
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
			search_text,
			normalized_location,
			is_remote,
			normalized_employment_type,
			seniority,
			min_years_experience,
			max_years_experience,
			experience_confidence,
			is_match,
			match_reasons_json,
			hard_failures_json,
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
	var isRemote int
	var isMatch int
	var minYearsExperience sql.NullInt64
	var maxYearsExperience sql.NullInt64
	var matchReasonsJSON string
	var hardFailuresJSON string
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
		&job.SearchText,
		&job.NormalizedLocation,
		&isRemote,
		&job.NormalizedEmploymentType,
		&job.Seniority,
		&minYearsExperience,
		&maxYearsExperience,
		&job.ExperienceConfidence,
		&isMatch,
		&matchReasonsJSON,
		&hardFailuresJSON,
		&job.FirstSeenAt,
		&job.LastSeenAt,
		&job.MatchedAt,
		&isActive,
	); err != nil {
		return JobRecord{}, err
	}

	job.IsActive = isActive == 1
	job.IsRemote = isRemote == 1
	job.IsMatch = isMatch == 1
	job.MinYearsExperience = nullInt64ToPtr(minYearsExperience)
	job.MaxYearsExperience = nullInt64ToPtr(maxYearsExperience)
	job.MatchReasons = decodeStringArray(matchReasonsJSON)
	job.HardFailures = decodeStringArray(hardFailuresJSON)
	return job, nil
}

func defaultJSON(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}

func defaultJSONArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullInt64ToPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}

	converted := int(value.Int64)
	return &converted
}

func decodeStringArray(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}

	if len(values) == 0 {
		return nil
	}

	return values
}
