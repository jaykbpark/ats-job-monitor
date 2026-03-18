package store

import (
	"context"
	"fmt"
	"strings"
)

type NotificationRecord struct {
	ID            int64  `json:"id"`
	WatchTargetID int64  `json:"watchTargetId"`
	JobID         int64  `json:"jobId"`
	ExternalJobID string `json:"externalJobId"`
	JobTitle      string `json:"jobTitle"`
	JobURL        string `json:"jobUrl"`
	Kind          string `json:"kind"`
	Channel       string `json:"channel"`
	Status        string `json:"status"`
	SentAt        string `json:"sentAt,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

func (s *Store) ListPendingNotifications(ctx context.Context, limit int) ([]NotificationRecord, error) {
	query := `
		SELECT
			n.id,
			n.watch_target_id,
			n.job_id,
			j.external_job_id,
			j.title,
			j.job_url,
			n.kind,
			n.channel,
			n.status,
			COALESCE(n.sent_at, ''),
			COALESCE(n.error_message, ''),
			n.created_at
		FROM notifications n
		JOIN jobs j ON j.id = n.job_id
		WHERE n.status = 'pending'
		ORDER BY n.id ASC
	`
	args := []any{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]NotificationRecord, 0)
	for rows.Next() {
		notification, err := scanNotificationRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending notifications: %w", err)
	}

	return notifications, nil
}

func (s *Store) ListNotificationsByWatchTarget(ctx context.Context, watchTargetID int64) ([]NotificationRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			n.id,
			n.watch_target_id,
			n.job_id,
			j.external_job_id,
			j.title,
			j.job_url,
			n.kind,
			n.channel,
			n.status,
			COALESCE(n.sent_at, ''),
			COALESCE(n.error_message, ''),
			n.created_at
		FROM notifications n
		JOIN jobs j ON j.id = n.job_id
		WHERE n.watch_target_id = ?
		ORDER BY n.id DESC
	`, watchTargetID)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]NotificationRecord, 0)
	for rows.Next() {
		notification, err := scanNotificationRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}

func (s *Store) MarkNotificationSent(ctx context.Context, notificationID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications
		SET status = 'sent',
			sent_at = CURRENT_TIMESTAMP,
			error_message = NULL
		WHERE id = ?
	`, notificationID)
	if err != nil {
		return fmt.Errorf("mark notification %d sent: %w", notificationID, err)
	}

	return nil
}

func (s *Store) MarkNotificationFailed(ctx context.Context, notificationID int64, errorMessage string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications
		SET status = 'failed',
			sent_at = NULL,
			error_message = NULLIF(?, '')
		WHERE id = ?
	`, strings.TrimSpace(errorMessage), notificationID)
	if err != nil {
		return fmt.Errorf("mark notification %d failed: %w", notificationID, err)
	}

	return nil
}

func scanNotificationRecord(row scanner) (NotificationRecord, error) {
	var notification NotificationRecord
	if err := row.Scan(
		&notification.ID,
		&notification.WatchTargetID,
		&notification.JobID,
		&notification.ExternalJobID,
		&notification.JobTitle,
		&notification.JobURL,
		&notification.Kind,
		&notification.Channel,
		&notification.Status,
		&notification.SentAt,
		&notification.ErrorMessage,
		&notification.CreatedAt,
	); err != nil {
		return NotificationRecord{}, err
	}

	return notification, nil
}
