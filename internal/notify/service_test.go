package notify

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/matching"
	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type fakeSink struct {
	failForID int64
	delivered []int64
}

func (f *fakeSink) Deliver(ctx context.Context, notification store.NotificationRecord) error {
	if notification.ID == f.failForID {
		return errors.New("sink failed")
	}

	f.delivered = append(f.delivered, notification.ID)
	return nil
}

func TestDeliverPendingMarksSentAndFailed(t *testing.T) {
	ctx := context.Background()
	dbStore := openNotifyTestStore(t)
	defer func() {
		_ = dbStore.Close()
	}()

	seedPendingNotifications(t, ctx, dbStore)

	sink := &fakeSink{failForID: 2}
	service := NewService(dbStore, sink)

	result, err := service.DeliverPending(ctx, 0)
	if err != nil {
		t.Fatalf("deliver pending: %v", err)
	}

	if result.LoadedCount != 2 {
		t.Fatalf("expected 2 loaded notifications, got %d", result.LoadedCount)
	}

	if result.DeliveredCount != 1 {
		t.Fatalf("expected 1 delivered notification, got %d", result.DeliveredCount)
	}

	if result.FailedCount != 1 {
		t.Fatalf("expected 1 failed notification, got %d", result.FailedCount)
	}

	notifications, err := dbStore.ListPendingNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("list pending notifications after delivery: %v", err)
	}

	if len(notifications) != 0 {
		t.Fatalf("expected no pending notifications left, got %d", len(notifications))
	}

	all, err := dbStore.ListNotificationsByWatchTarget(ctx, 1)
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}

	if all[0].Status != "failed" || all[1].Status != "sent" {
		t.Fatalf("unexpected notification statuses: %#v", all)
	}
}

func TestConsoleSinkWritesJSONLine(t *testing.T) {
	var buf bytes.Buffer
	sink := NewConsoleSink(&buf)

	err := sink.Deliver(context.Background(), store.NotificationRecord{
		ID:            7,
		Kind:          "new_match",
		Channel:       "inbox",
		JobTitle:      "Software Engineer",
		JobURL:        "https://example.com/jobs/7",
		ExternalJobID: "abc-123",
	})
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected console sink to write output")
	}
}

func openNotifyTestStore(t *testing.T) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "notify-test.sqlite")
	dbStore, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	if err := dbStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	return dbStore
}

func seedPendingNotifications(t *testing.T, ctx context.Context, dbStore *store.Store) {
	t.Helper()

	target, err := dbStore.CreateWatchTarget(ctx, store.CreateWatchTargetParams{
		Name:      "Demo",
		Provider:  "greenhouse",
		BoardKey:  "demo",
		SourceURL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("create watch target: %v", err)
	}

	jobs := []providers.Job{
		{
			ExternalJobID: "job-1",
			Title:         "Software Engineer",
			JobURL:        "https://example.com/jobs/1",
			RawJSON:       `{"descriptionPlain":"Build software."}`,
		},
		{
			ExternalJobID: "job-2",
			Title:         "Platform Engineer",
			JobURL:        "https://example.com/jobs/2",
			RawJSON:       `{"descriptionPlain":"Build platform software."}`,
		},
	}

	syncedJobs := make([]store.SyncedJob, 0, len(jobs))
	for _, job := range jobs {
		syncedJobs = append(syncedJobs, store.SyncedJob{
			Job:   job,
			Match: matchingResult(job.ExternalJobID),
		})
	}

	if _, err := dbStore.SyncJobs(ctx, target.ID, syncedJobs); err != nil {
		t.Fatalf("sync jobs: %v", err)
	}
}

func matchingResult(externalJobID string) matching.Result {
	if externalJobID == "job-1" {
		return matching.Result{Matched: true, MatchReasons: []string{"matched keywords: software"}}
	}

	return matching.Result{Matched: true, MatchReasons: []string{"matched keywords: platform"}}
}
