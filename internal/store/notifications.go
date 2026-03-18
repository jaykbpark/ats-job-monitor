package store

import (
	"context"
	"fmt"
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
		var notification NotificationRecord
		if err := rows.Scan(
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
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}
