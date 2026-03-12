package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/store"
)

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
	    "keywords": ["platform", "backend"]
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

	return NewServer(dbStore)
}
