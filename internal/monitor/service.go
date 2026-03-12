package monitor

import (
	"context"
	"fmt"

	"github.com/jaykbpark/ats-job-monitor/internal/ats"
	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

type GreenhouseFetcher interface {
	FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error)
}

type LeverFetcher interface {
	FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error)
}

type AshbyFetcher interface {
	FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error)
}

type Service struct {
	store      *store.Store
	greenhouse GreenhouseFetcher
	lever      LeverFetcher
	ashby      AshbyFetcher
}

func NewService(store *store.Store, greenhouse GreenhouseFetcher, lever LeverFetcher, ashby AshbyFetcher) *Service {
	if greenhouse == nil {
		greenhouse = providers.NewGreenhouseClient()
	}
	if lever == nil {
		lever = providers.NewLeverClient()
	}
	if ashby == nil {
		ashby = providers.NewAshbyClient()
	}

	return &Service{
		store:      store,
		greenhouse: greenhouse,
		lever:      lever,
		ashby:      ashby,
	}
}

func (s *Service) SyncWatchTarget(ctx context.Context, watchTargetID int64) (store.SyncRun, error) {
	target, err := s.store.GetWatchTarget(ctx, watchTargetID)
	if err != nil {
		return store.SyncRun{}, fmt.Errorf("load watch target: %w", err)
	}

	var jobs []providers.Job
	switch target.Provider {
	case ats.ProviderGreenhouse:
		jobs, err = s.greenhouse.FetchJobs(ctx, target.BoardKey)
	case ats.ProviderLever:
		jobs, err = s.lever.FetchJobs(ctx, target.BoardKey)
	case ats.ProviderAshby:
		jobs, err = s.ashby.FetchJobs(ctx, target.BoardKey)
	default:
		err = fmt.Errorf("provider %q is not supported for sync yet", target.Provider)
	}

	if err != nil {
		run, recordErr := s.store.RecordSyncRun(ctx, store.RecordSyncRunParams{
			WatchTargetID: target.ID,
			Status:        "failed",
			ErrorMessage:  err.Error(),
		})
		if recordErr != nil {
			return store.SyncRun{}, fmt.Errorf("sync watch target failed: %v (also failed to record sync run: %w)", err, recordErr)
		}
		return run, err
	}

	result, err := s.store.SyncJobs(ctx, target.ID, jobs)
	if err != nil {
		run, recordErr := s.store.RecordSyncRun(ctx, store.RecordSyncRunParams{
			WatchTargetID:    target.ID,
			Status:           "failed",
			FetchedJobsCount: len(jobs),
			MatchedJobsCount: len(jobs),
			ErrorMessage:     err.Error(),
		})
		if recordErr != nil {
			return store.SyncRun{}, fmt.Errorf("sync jobs failed: %v (also failed to record sync run: %w)", err, recordErr)
		}
		return run, err
	}

	return s.store.RecordSyncRun(ctx, store.RecordSyncRunParams{
		WatchTargetID:    target.ID,
		Status:           "succeeded",
		FetchedJobsCount: result.FetchedJobsCount,
		MatchedJobsCount: result.MatchedJobsCount,
		NewJobsCount:     result.NewJobsCount,
	})
}
