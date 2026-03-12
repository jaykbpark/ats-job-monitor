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

type fakeLeverFetcher struct {
	jobs []providers.Job
	err  error
}

func (f *fakeLeverFetcher) FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error) {
	return f.jobs, f.err
}

type fakeAshbyFetcher struct {
	jobs []providers.Job
	err  error
}

func (f *fakeAshbyFetcher) FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error) {
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
	}, nil, nil)

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

func TestSyncWatchTargetSupportsLever(t *testing.T) {
	ctx := context.Background()
	dbStore := openMonitorTestStore(t)
	defer func() {
		_ = dbStore.Close()
	}()

	target, err := dbStore.CreateWatchTarget(ctx, store.CreateWatchTargetParams{
		Name:      "Lever Demo",
		Provider:  "lever",
		BoardKey:  "leverdemo",
		SourceURL: "https://jobs.lever.co/leverdemo",
	})
	if err != nil {
		t.Fatalf("create watch target: %v", err)
	}

	service := NewService(dbStore, nil, &fakeLeverFetcher{
		jobs: []providers.Job{
			{
				ExternalJobID:  "job-1",
				Title:          "Software Engineer",
				Location:       "Remote",
				Department:     "Engineering",
				EmploymentType: "Full-time",
				JobURL:         "https://jobs.lever.co/leverdemo/job-1",
				MetadataJSON:   `{"workplaceType":"remote"}`,
				RawJSON:        `{"id":"job-1"}`,
			},
		},
	}, nil)

	run, err := service.SyncWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("sync lever watch target: %v", err)
	}

	if run.Status != "succeeded" {
		t.Fatalf("expected succeeded run, got %q", run.Status)
	}

	jobs, err := dbStore.ListJobsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list synced jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 persisted lever job, got %d", len(jobs))
	}

	if jobs[0].EmploymentType != "Full-time" {
		t.Fatalf("unexpected employment type: %q", jobs[0].EmploymentType)
	}
}

func TestSyncWatchTargetSupportsAshby(t *testing.T) {
	ctx := context.Background()
	dbStore := openMonitorTestStore(t)
	defer func() {
		_ = dbStore.Close()
	}()

	target, err := dbStore.CreateWatchTarget(ctx, store.CreateWatchTargetParams{
		Name:      "Ashby",
		Provider:  "ashby",
		BoardKey:  "Ashby",
		SourceURL: "https://jobs.ashbyhq.com/Ashby",
	})
	if err != nil {
		t.Fatalf("create watch target: %v", err)
	}

	service := NewService(dbStore, nil, nil, &fakeAshbyFetcher{
		jobs: []providers.Job{
			{
				ExternalJobID:  "job-1",
				Title:          "Product Engineer",
				Location:       "Remote - US",
				Department:     "Engineering",
				Team:           "Platform",
				EmploymentType: "FullTime",
				JobURL:         "https://jobs.ashbyhq.com/Ashby/job-1",
				MetadataJSON:   `{"workplaceType":"Remote"}`,
				RawJSON:        `{"id":"job-1"}`,
			},
		},
	})

	run, err := service.SyncWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("sync ashby watch target: %v", err)
	}

	if run.Status != "succeeded" {
		t.Fatalf("expected succeeded run, got %q", run.Status)
	}

	jobs, err := dbStore.ListJobsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list synced jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 persisted ashby job, got %d", len(jobs))
	}

	if jobs[0].Team != "Platform" {
		t.Fatalf("unexpected team: %q", jobs[0].Team)
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
