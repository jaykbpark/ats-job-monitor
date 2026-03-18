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
		FiltersJSON: `{
			"includeKeywordsAny": ["backend"],
			"remoteOnly": true
		}`,
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
			{
				ExternalJobID: "43",
				Title:         "Office IT Engineer",
				Location:      "San Francisco, CA",
				Department:    "Engineering",
				JobURL:        "https://job-boards.greenhouse.io/greenhouse/jobs/43",
				MetadataJSON:  `{"team":"Internal Systems"}`,
				RawJSON:       `{"id":43}`,
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

	if run.NewJobsCount != 2 {
		t.Fatalf("expected 2 new jobs, got %d", run.NewJobsCount)
	}

	if run.MatchedJobsCount != 1 {
		t.Fatalf("expected 1 matched job, got %d", run.MatchedJobsCount)
	}

	jobs, err := dbStore.ListJobsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list synced jobs: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 persisted jobs, got %d", len(jobs))
	}

	if jobs[0].SearchText == "" {
		t.Fatal("expected derived search text to be persisted")
	}

	if !jobs[0].IsMatch {
		t.Fatalf("expected first job to match filters: %#v", jobs[0])
	}

	if jobs[1].IsMatch {
		t.Fatalf("expected second job not to match filters: %#v", jobs[1])
	}

	if len(jobs[1].HardFailures) == 0 {
		t.Fatalf("expected hard failures for unmatched job: %#v", jobs[1])
	}

	runs, err := dbStore.ListSyncRunsByWatchTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("list sync runs: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("expected 1 sync run, got %d", len(runs))
	}

	if runs[0].MatchedJobsCount != 1 {
		t.Fatalf("expected persisted matched job count to be 1, got %d", runs[0].MatchedJobsCount)
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

	if jobs[0].NormalizedEmploymentType != "full-time" {
		t.Fatalf("unexpected normalized employment type: %q", jobs[0].NormalizedEmploymentType)
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

	if jobs[0].Seniority != "unknown" {
		t.Fatalf("unexpected seniority: %q", jobs[0].Seniority)
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
