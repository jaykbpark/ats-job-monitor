package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrateAppliesEmbeddedSQLFiles(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	records, err := store.AppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("failed to read applied migrations: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 applied migration, got %d", len(records))
	}

	if records[0].Version != "0001_init.sql" {
		t.Fatalf("unexpected migration version: %q", records[0].Version)
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second migrate call should be idempotent: %v", err)
	}
}

func TestCreateAndListWatchTargets(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	target, err := store.CreateWatchTarget(ctx, CreateWatchTargetParams{
		Name:        "Greenhouse",
		Provider:    "greenhouse",
		BoardKey:    "greenhouse",
		SourceURL:   "https://job-boards.greenhouse.io/greenhouse",
		FiltersJSON: `{"includeKeywordsAny":["platform"]}`,
	})
	if err != nil {
		t.Fatalf("create watch target failed: %v", err)
	}

	if target.ID == 0 {
		t.Fatal("expected inserted watch target id")
	}

	if target.Status != "active" {
		t.Fatalf("expected default status active, got %q", target.Status)
	}

	targets, err := store.ListWatchTargets(ctx)
	if err != nil {
		t.Fatalf("list watch targets failed: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 watch target, got %d", len(targets))
	}

	if targets[0].BoardKey != "greenhouse" {
		t.Fatalf("unexpected board key: %q", targets[0].BoardKey)
	}

	if targets[0].SourceURL != "https://job-boards.greenhouse.io/greenhouse" {
		t.Fatalf("unexpected source url: %q", targets[0].SourceURL)
	}

	if len(targets[0].Filters.IncludeKeywordsAny) != 1 || targets[0].Filters.IncludeKeywordsAny[0] != "platform" {
		t.Fatalf("unexpected parsed filters: %#v", targets[0].Filters)
	}
}

func TestCreateWatchTargetRejectsInvalidFilters(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	_, err := store.CreateWatchTarget(ctx, CreateWatchTargetParams{
		Name:        "Greenhouse",
		Provider:    "greenhouse",
		BoardKey:    "greenhouse",
		FiltersJSON: `{"minYearsExperience":6,"maxYearsExperience":2}`,
	})
	if err == nil {
		t.Fatal("expected invalid filters to be rejected")
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}

	return store
}
