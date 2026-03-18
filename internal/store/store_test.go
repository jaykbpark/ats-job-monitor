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

	if len(records) != 5 {
		t.Fatalf("expected 5 applied migrations, got %d", len(records))
	}

	if records[0].Version != "0001_init.sql" {
		t.Fatalf("unexpected migration version: %q", records[0].Version)
	}

	if records[1].Version != "0002_job_signals.sql" {
		t.Fatalf("unexpected second migration version: %q", records[1].Version)
	}

	if records[2].Version != "0003_job_match_state.sql" {
		t.Fatalf("unexpected third migration version: %q", records[2].Version)
	}

	if records[3].Version != "0004_notification_kind.sql" {
		t.Fatalf("unexpected fourth migration version: %q", records[3].Version)
	}

	if records[4].Version != "0005_watch_target_notification_email.sql" {
		t.Fatalf("unexpected fifth migration version: %q", records[4].Version)
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
		Name:              "Greenhouse",
		Provider:          "greenhouse",
		BoardKey:          "greenhouse",
		SourceURL:         "https://job-boards.greenhouse.io/greenhouse",
		NotificationEmail: "jobs@example.com",
		FiltersJSON:       `{"includeKeywordsAny":["platform"]}`,
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

	if targets[0].NotificationEmail != "jobs@example.com" {
		t.Fatalf("unexpected notification email: %q", targets[0].NotificationEmail)
	}

	if len(targets[0].Filters.IncludeKeywordsAny) != 1 || targets[0].Filters.IncludeKeywordsAny[0] != "platform" {
		t.Fatalf("unexpected parsed filters: %#v", targets[0].Filters)
	}
}

func TestCreateWatchTargetRejectsInvalidNotificationEmail(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	_, err := store.CreateWatchTarget(ctx, CreateWatchTargetParams{
		Name:              "Greenhouse",
		Provider:          "greenhouse",
		BoardKey:          "greenhouse",
		NotificationEmail: "not-an-email",
	})
	if err == nil {
		t.Fatal("expected invalid notification email to be rejected")
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

func TestUpdateWatchTarget(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer func() {
		_ = store.Close()
	}()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	target, err := store.CreateWatchTarget(ctx, CreateWatchTargetParams{
		Name:              "Greenhouse",
		Provider:          "greenhouse",
		BoardKey:          "greenhouse",
		SourceURL:         "https://job-boards.greenhouse.io/greenhouse",
		NotificationEmail: "jobs@example.com",
		FiltersJSON:       `{"includeKeywordsAny":["platform"]}`,
		Status:            "active",
	})
	if err != nil {
		t.Fatalf("create watch target failed: %v", err)
	}

	updatedName := "Greenhouse Backend"
	updatedSourceURL := ""
	updatedEmail := ""
	updatedFilters := `{"includeKeywordsAny":["backend"],"remoteOnly":true}`
	updatedStatus := "paused"

	updated, err := store.UpdateWatchTarget(ctx, UpdateWatchTargetParams{
		ID:                target.ID,
		Name:              &updatedName,
		SourceURL:         &updatedSourceURL,
		NotificationEmail: &updatedEmail,
		FiltersJSON:       &updatedFilters,
		Status:            &updatedStatus,
	})
	if err != nil {
		t.Fatalf("update watch target failed: %v", err)
	}

	if updated.Name != "Greenhouse Backend" {
		t.Fatalf("unexpected updated name: %q", updated.Name)
	}

	if updated.SourceURL != "" {
		t.Fatalf("expected source URL to be cleared, got %q", updated.SourceURL)
	}

	if updated.NotificationEmail != "" {
		t.Fatalf("expected notification email to be cleared, got %q", updated.NotificationEmail)
	}

	if updated.Status != "paused" {
		t.Fatalf("unexpected updated status: %q", updated.Status)
	}

	if !updated.Filters.RemoteOnly || len(updated.Filters.IncludeKeywordsAny) != 1 || updated.Filters.IncludeKeywordsAny[0] != "backend" {
		t.Fatalf("unexpected updated filters: %#v", updated.Filters)
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
