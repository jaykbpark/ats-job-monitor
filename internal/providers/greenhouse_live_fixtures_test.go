package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGreenhouseClientLiveEmploymentFixtures(t *testing.T) {
	tests := []struct {
		name               string
		boardKey           string
		fixtureFile        string
		fetchWithContent   bool
		wantTitle          string
		wantEmploymentType string
	}{
		{
			name:               "figma manager billing anchored role text",
			boardKey:           "figma",
			fixtureFile:        "greenhouse_figma_manager_billing.json",
			fetchWithContent:   true,
			wantTitle:          "Manager, Software Engineering - Billing",
			wantEmploymentType: "full-time",
		},
		{
			name:               "stripe disclaimer does not imply internship",
			boardKey:           "stripe",
			fixtureFile:        "greenhouse_stripe_backend_api_engineer_mas.json",
			fetchWithContent:   true,
			wantTitle:          "Backend/API Engineer, Money as a Service",
			wantEmploymentType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := mustReadFixture(t, tt.fixtureFile)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(payload)
			}))
			defer server.Close()

			client := &GreenhouseClient{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			var (
				jobs []Job
				err  error
			)
			if tt.fetchWithContent {
				jobs, err = client.FetchJobsWithContent(context.Background(), tt.boardKey)
			} else {
				jobs, err = client.FetchJobs(context.Background(), tt.boardKey)
			}
			if err != nil {
				t.Fatalf("fetch jobs: %v", err)
			}

			if len(jobs) != 1 {
				t.Fatalf("expected 1 job, got %d", len(jobs))
			}

			if jobs[0].Title != tt.wantTitle {
				t.Fatalf("Title = %q, want %q", jobs[0].Title, tt.wantTitle)
			}

			if jobs[0].EmploymentType != tt.wantEmploymentType {
				t.Fatalf("EmploymentType = %q, want %q", jobs[0].EmploymentType, tt.wantEmploymentType)
			}
		})
	}
}

func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}

	return data
}
