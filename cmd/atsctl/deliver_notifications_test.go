package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/matching"
	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

func TestParseDeliverNotificationsArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLimit  int
		wantDBPath string
		wantErr    bool
	}{
		{
			name:       "db path before limit",
			args:       []string{"/tmp/app.sqlite", "--limit", "5"},
			wantLimit:  5,
			wantDBPath: "/tmp/app.sqlite",
		},
		{
			name:       "limit before db path",
			args:       []string{"--limit", "2", "/tmp/app.sqlite"},
			wantLimit:  2,
			wantDBPath: "/tmp/app.sqlite",
		},
		{
			name:    "missing db path",
			args:    []string{"--limit", "2"},
			wantErr: true,
		},
		{
			name:    "unsupported flag",
			args:    []string{"--dry-run", "/tmp/app.sqlite"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotDBPath, err := parseDeliverNotificationsArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseDeliverNotificationsArgs() error = %v", err)
			}

			if gotLimit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", gotLimit, tt.wantLimit)
			}

			if gotDBPath != tt.wantDBPath {
				t.Fatalf("dbPath = %q, want %q", gotDBPath, tt.wantDBPath)
			}
		})
	}
}

func TestBuildDeliverySinkRequiresSMTPForEmailNotifications(t *testing.T) {
	dbStore := openCommandTestStore(t)
	defer func() {
		_ = dbStore.Close()
	}()

	ctx := context.Background()
	target, err := dbStore.CreateWatchTarget(ctx, store.CreateWatchTargetParams{
		Name:              "Demo",
		Provider:          "greenhouse",
		BoardKey:          "demo",
		NotificationEmail: "jobs@example.com",
	})
	if err != nil {
		t.Fatalf("create watch target: %v", err)
	}

	if _, err := dbStore.SyncJobs(ctx, target.ID, []store.SyncedJob{
		{
			Job: providers.Job{
				ExternalJobID: "job-1",
				Title:         "Software Engineer",
				JobURL:        "https://example.com/jobs/1",
				RawJSON:       `{"descriptionPlain":"Build backend software."}`,
			},
			Match: matching.Result{Matched: true, MatchReasons: []string{"matched keywords: software"}},
		},
	}); err != nil {
		t.Fatalf("sync jobs: %v", err)
	}

	if _, err := buildDeliverySink(ctx, dbStore, 0); err == nil {
		t.Fatal("expected SMTP configuration error for email notifications")
	}
}

func openCommandTestStore(t *testing.T) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "atsctl.sqlite")
	dbStore, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := dbStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	return dbStore
}
