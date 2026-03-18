package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type Sink interface {
	Deliver(ctx context.Context, notification store.NotificationRecord) error
}

type Service struct {
	store *store.Store
	sink  Sink
}

type Result struct {
	LoadedCount    int `json:"loadedCount"`
	DeliveredCount int `json:"deliveredCount"`
	FailedCount    int `json:"failedCount"`
}

func NewService(store *store.Store, sink Sink) *Service {
	if sink == nil {
		sink = NewConsoleSink(io.Discard)
	}

	return &Service{
		store: store,
		sink:  sink,
	}
}

func (s *Service) DeliverPending(ctx context.Context, limit int) (Result, error) {
	notifications, err := s.store.ListPendingNotifications(ctx, limit)
	if err != nil {
		return Result{}, err
	}

	result := Result{LoadedCount: len(notifications)}
	for _, notification := range notifications {
		if err := s.sink.Deliver(ctx, notification); err != nil {
			if updateErr := s.store.MarkNotificationFailed(ctx, notification.ID, err.Error()); updateErr != nil {
				return Result{}, updateErr
			}
			result.FailedCount++
			continue
		}

		if err := s.store.MarkNotificationSent(ctx, notification.ID); err != nil {
			return Result{}, err
		}
		result.DeliveredCount++
	}

	return result, nil
}

type ConsoleSink struct {
	Writer io.Writer
}

func NewConsoleSink(writer io.Writer) *ConsoleSink {
	if writer == nil {
		writer = io.Discard
	}

	return &ConsoleSink{Writer: writer}
}

func (s *ConsoleSink) Deliver(ctx context.Context, notification store.NotificationRecord) error {
	payload := map[string]any{
		"notificationId": notification.ID,
		"kind":           notification.Kind,
		"channel":        notification.Channel,
		"jobTitle":       notification.JobTitle,
		"jobUrl":         notification.JobURL,
		"externalJobId":  notification.ExternalJobID,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode notification payload: %w", err)
	}

	if _, err := fmt.Fprintln(s.Writer, string(encoded)); err != nil {
		return fmt.Errorf("write notification payload: %w", err)
	}

	return nil
}
