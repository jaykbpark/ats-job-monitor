package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGreenhouseClientFetchJobsSupportsArrayMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 99,
		      "title": "Platform Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/99",
		      "location": { "name": "Remote" },
		      "metadata": ["remote", "us-only"],
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

	if jobs[0].MetadataJSON != `["remote", "us-only"]` {
		t.Fatalf("unexpected metadata json: %q", jobs[0].MetadataJSON)
	}
}

func TestGreenhouseClientFetchJobsWithContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("content"); got != "true" {
			t.Fatalf("expected content=true query param, got %q", got)
		}

		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 100,
		      "title": "Backend Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/100",
		      "content": "<p>Requires 4+ years of Go experience</p>",
		      "location": { "name": "Remote" },
		      "metadata": null,
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

	jobs, err := client.FetchJobsWithContent(context.Background(), "acme")
	if err != nil {
		t.Fatalf("fetch jobs with content: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	if !strings.Contains(jobs[0].RawJSON, `"content":"\u003cp\u003eRequires 4+ years of Go experience\u003c/p\u003e"`) {
		t.Fatalf("expected raw json to contain fetched content, got %q", jobs[0].RawJSON)
	}
}

func TestGreenhouseClientExtractsEmploymentTypeFromStructuredMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 101,
		      "title": "Software Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/101",
		      "location": { "name": "Remote" },
		      "metadata": [
		        {
		          "name": "Time Type",
		          "value": "Full Time"
		        }
		      ],
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

	if got := jobs[0].EmploymentType; got != "full-time" {
		t.Fatalf("unexpected employment type: %q", got)
	}
}

func TestGreenhouseClientExtractsEmploymentTypeFromAnchoredContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 102,
		      "title": "Software Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/102",
		      "content": "<p>This is a full-time role based in our SF office.</p>",
		      "location": { "name": "San Francisco" },
		      "metadata": null,
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

	jobs, err := client.FetchJobsWithContent(context.Background(), "acme")
	if err != nil {
		t.Fatalf("fetch jobs with content: %v", err)
	}

	if got := jobs[0].EmploymentType; got != "full-time" {
		t.Fatalf("unexpected employment type: %q", got)
	}
}

func TestGreenhouseClientIgnoresInternDisclaimerForEmploymentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
		  "jobs": [
		    {
		      "id": 103,
		      "title": "Backend Engineer",
		      "absolute_url": "https://job-boards.greenhouse.io/acme/jobs/103",
		      "content": "<p>If you are an intern, new grad, staff, frontend or fullstack applicant, please do not apply using this link.</p>",
		      "location": { "name": "Remote" },
		      "metadata": null,
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

	jobs, err := client.FetchJobsWithContent(context.Background(), "acme")
	if err != nil {
		t.Fatalf("fetch jobs with content: %v", err)
	}

	if got := jobs[0].EmploymentType; got != "" {
		t.Fatalf("expected disclaimer text to be ignored, got %q", got)
	}
}
