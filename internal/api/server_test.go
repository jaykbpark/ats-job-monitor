package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	monitorpkg "github.com/jaykbpark/ats-job-monitor/internal/monitor"
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

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestListCompaniesEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/companies?q=greenhouse", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode companies response: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("expected 1 company, got %d", len(payload))
	}
}

func TestCreateAndListWatchTargetsEndpoints(t *testing.T) {
	server := newTestServer(t)

	body := []byte(`{
	  "name": "Greenhouse",
	  "provider": "greenhouse",
	  "boardKey": "greenhouse",
	  "sourceUrl": "https://job-boards.greenhouse.io/greenhouse",
	  "filters": {
	    "includeKeywordsAny": ["platform", "backend"]
	  }
	}`)

	createReq := httptest.NewRequest(http.MethodPost, "/api/watch-targets", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(createRecorder, createReq)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	if created["provider"] != "greenhouse" {
		t.Fatalf("unexpected provider: %#v", created["provider"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/watch-targets", nil)
	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, listReq)

	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRecorder.Code)
	}

	var targets []map[string]any
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &targets); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 watch target, got %d", len(targets))
	}
}

func TestCreateWatchTargetRejectsInvalidFiltersJSON(t *testing.T) {
	server := newTestServer(t)

	body := []byte(`{
	  "name": "Bad Filters",
	  "provider": "greenhouse",
	  "boardKey": "greenhouse",
	  "filtersJson": "{bad json}"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/watch-targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateWatchTargetRejectsUnknownFilterFields(t *testing.T) {
	server := newTestServer(t)

	body := []byte(`{
	  "name": "Bad Filters",
	  "provider": "greenhouse",
	  "boardKey": "greenhouse",
	  "filters": {
	    "unknownField": true
	  }
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/watch-targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreateWatchTargetReturnsTypedFilters(t *testing.T) {
	server := newTestServer(t)

	body := []byte(`{
	  "name": "Typed Filters",
	  "provider": "greenhouse",
	  "boardKey": "greenhouse",
	  "filters": {
	    "locationsAny": [" Remote ", "Vancouver"],
	    "includeKeywordsAny": ["backend"]
	  }
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/watch-targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var target map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &target); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	filters, ok := target["filters"].(map[string]any)
	if !ok {
		t.Fatalf("expected typed filters in response, got %#v", target["filters"])
	}

	if _, ok := filters["locationsAny"]; !ok {
		t.Fatalf("expected locationsAny in typed filters response: %#v", filters)
	}
}

func TestSyncWatchTargetAndListJobs(t *testing.T) {
	server := newTestServer(t)

	createWatchTargetForTest(t, server, `{
	  "includeKeywordsAny": ["backend"],
	  "remoteOnly": true
	}`)

	syncReq := httptest.NewRequest(http.MethodPost, "/api/watch-targets/1/sync", nil)
	syncRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(syncRecorder, syncReq)

	if syncRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from sync, got %d: %s", syncRecorder.Code, syncRecorder.Body.String())
	}

	jobsReq := httptest.NewRequest(http.MethodGet, "/api/watch-targets/1/jobs", nil)
	jobsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(jobsRecorder, jobsReq)

	if jobsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from jobs list, got %d", jobsRecorder.Code)
	}

	var jobs []map[string]any
	if err := json.Unmarshal(jobsRecorder.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode jobs response: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 synced jobs, got %d", len(jobs))
	}

	if jobs[0]["isMatch"] != true {
		t.Fatalf("expected first synced job to match filters, got %#v", jobs[0]["isMatch"])
	}

	if jobs[1]["isMatch"] != false {
		t.Fatalf("expected second synced job not to match filters, got %#v", jobs[1]["isMatch"])
	}

	runsReq := httptest.NewRequest(http.MethodGet, "/api/watch-targets/1/sync-runs", nil)
	runsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(runsRecorder, runsReq)

	if runsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from sync-runs list, got %d", runsRecorder.Code)
	}

	var runs []map[string]any
	if err := json.Unmarshal(runsRecorder.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode sync-runs response: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("expected 1 sync run, got %d", len(runs))
	}

	if runs[0]["matchedJobsCount"] != float64(1) {
		t.Fatalf("expected matchedJobsCount=1, got %#v", runs[0]["matchedJobsCount"])
	}

	notificationsReq := httptest.NewRequest(http.MethodGet, "/api/watch-targets/1/notifications", nil)
	notificationsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(notificationsRecorder, notificationsReq)

	if notificationsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from notifications list, got %d", notificationsRecorder.Code)
	}

	var notifications []map[string]any
	if err := json.Unmarshal(notificationsRecorder.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("decode notifications response: %v", err)
	}

	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	if notifications[0]["kind"] != "new_match" {
		t.Fatalf("unexpected notification kind: %#v", notifications[0]["kind"])
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "api-test.sqlite")
	dbStore, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	t.Cleanup(func() {
		_ = dbStore.Close()
	})

	if err := dbStore.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	return NewServer(dbStore, monitorpkg.NewService(dbStore, &fakeGreenhouseFetcher{
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
	}, &fakeLeverFetcher{}, &fakeAshbyFetcher{}))
}

func createWatchTargetForTest(t *testing.T, server *Server, filtersJSON string) {
	t.Helper()

	body := []byte(`{
	  "name": "Greenhouse",
	  "provider": "greenhouse",
	  "boardKey": "greenhouse",
	  "sourceUrl": "https://job-boards.greenhouse.io/greenhouse",
	  "filters": ` + filtersJSON + `
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/watch-targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201 from watch target creation, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
