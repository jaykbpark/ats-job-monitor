package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLeverClientFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/postings/acme" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_, _ = w.Write([]byte(`[
		  {
		    "id": "job-1",
		    "text": "Software Engineer",
		    "hostedUrl": "https://jobs.lever.co/acme/job-1",
		    "categories": {
		      "location": "Remote",
		      "commitment": "Full-time",
		      "team": "Platform",
		      "department": "Engineering"
		    },
		    "lists": [{"text": "List A"}],
		    "meta": {"level": "Senior"},
		    "workplaceType": "remote"
		  }
		]`))
	}))
	defer server.Close()

	client := &LeverClient{
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

	if jobs[0].ExternalJobID != "job-1" {
		t.Fatalf("unexpected external job id: %q", jobs[0].ExternalJobID)
	}

	if jobs[0].EmploymentType != "Full-time" {
		t.Fatalf("unexpected employment type: %q", jobs[0].EmploymentType)
	}
}
