package monitor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type fakeGreenhouseFetcher struct {
	jobs []providers.Job
	err  error
}

func (f *fakeGreenhouseFetcher) FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error) {
	return f.jobs, f.err
}

func TestSyncWatchTargetPersistsJobsAndRun(t *testing.T) {
	ctx := context.Background()
	dbStore := openMonitorTestStore(t)
	defer func() {
		_ = dbStore.Close()
	}()

	target, err := dbStore.CreateWatchTarget(ctx, store.CreateWatchTargetParams{
		Name:      "Greenhouse",
		Provider:  "greenhouse",
		BoardKey:  "greenhouse",
		SourceURL: "https://job-boards.greenhouse.io/greenhouse",
	})
	if err != nil {
		t.Fatalf("create watch target: %v", err)
	}

	service := NewService(dbStore, &fakeGreenhouseFetcher{
		jobs: []providers.Job{
			{
				ExternalJobID: "42",
				Title:         "Backend Engineer",
				Location:      "Remote",
				Department:    "Engineering",
				JobURL:        "https://job-boards.greenhouse.io/greenhouse/jobs/42",
				MetadataJSON:  `{"team":"Platform"}`,
				RawJSON:       `{"id":42}`,
			},
		},
	})

	run, err := service.SyncWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("sync watch target: %v", err)
	}

	if run.Status != "succeeded" {
		t.Fatalf("expected succeeded run, got %q", run.Status)
	}

	if run.NewJobsCount != 1 {
		t.Fatalf("expected 1 new job, got %d", run.NewJobsCount)
	}

	jobs, err := dbStore.ListJobsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list synced jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 persisted job, got %d", len(jobs))
	}

	runs, err := dbStore.ListSyncRunsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list sync runs: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("expected 1 sync run, got %d", len(runs))
	}
}

func openMonitorTestStore(t *testing.T) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "monitor-test.sqlite")
	dbStore, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	if err := dbStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	return dbStore
}
