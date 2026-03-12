package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGreenhouseClientFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/boards/acme/jobs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 42,
		      "title": "Backend Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/42",
		      "location": { "name": "Remote" },
		      "metadata": { "team": "Platform" },
		      "departments": [{ "name": "Engineering" }]
		    }
		  ]
		}`))
	}))
	defer server.Close()

	client := &GreenhouseClient{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}

	jobs, err := client.FetchJobs(context.Background(), "acme")
	if err != nil {
		t.Fatalf("fetch jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	if jobs[0].ExternalJobID != "42" {
		t.Fatalf("unexpected external job id: %q", jobs[0].ExternalJobID)
	}

	if jobs[0].Department != "Engineering" {
		t.Fatalf("unexpected department: %q", jobs[0].Department)
	}
}
