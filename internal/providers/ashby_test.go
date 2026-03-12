package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAshbyClientFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posting-api/job-board/Acme" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": "job-1",
		      "title": "Product Engineer",
		      "department": "Engineering",
		      "team": "Platform",
		      "employmentType": "FullTime",
		      "location": "Remote - US",
		      "workplaceType": "Remote",
		      "jobUrl": "https://jobs.ashbyhq.com/Acme/job-1",
		      "applyUrl": "https://jobs.ashbyhq.com/Acme/job-1/application",
		      "isRemote": true,
		      "secondaryLocations": [{"location":"Remote - Canada"}]
		    }
		  ]
		}`))
	}))
	defer server.Close()

	client := &AshbyClient{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}

	jobs, err := client.FetchJobs(context.Background(), "Acme")
	if err != nil {
		t.Fatalf("fetch jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	if jobs[0].ExternalJobID != "job-1" {
		t.Fatalf("unexpected external job id: %q", jobs[0].ExternalJobID)
	}

	if jobs[0].Location != "Remote - US" {
		t.Fatalf("unexpected location: %q", jobs[0].Location)
	}
}
