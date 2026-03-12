package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AshbyClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

type ashbyJobsResponse struct {
	Jobs []ashbyJob `json:"jobs"`
}

type ashbyJob struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Department     string          `json:"department"`
	Team           string          `json:"team"`
	EmploymentType string          `json:"employmentType"`
	Location       string          `json:"location"`
	WorkplaceType  string          `json:"workplaceType"`
	JobURL         string          `json:"jobUrl"`
	ApplyURL       string          `json:"applyUrl"`
	IsRemote       bool            `json:"isRemote"`
	Secondary      []ashbyLocation `json:"secondaryLocations"`
}

type ashbyLocation struct {
	Location string `json:"location"`
}

func NewAshbyClient() *AshbyClient {
	return &AshbyClient{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		BaseURL:    "https://api.ashbyhq.com",
	}
}

func (c *AshbyClient) FetchJobs(ctx context.Context, boardKey string) ([]Job, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.ashbyhq.com"
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	requestURL := fmt.Sprintf("%s/posting-api/job-board/%s", baseURL, url.PathEscape(boardKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build ashby request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ashby jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ashby jobs returned status %d", resp.StatusCode)
	}

	var payload ashbyJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode ashby jobs: %w", err)
	}

	jobs := make([]Job, 0, len(payload.Jobs))
	for _, job := range payload.Jobs {
		raw, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("encode ashby raw job: %w", err)
		}

		secondaryLocations := make([]string, 0, len(job.Secondary))
		for _, location := range job.Secondary {
			if trimmed := strings.TrimSpace(location.Location); trimmed != "" {
				secondaryLocations = append(secondaryLocations, trimmed)
			}
		}

		metadata := map[string]any{
			"applyUrl":           job.ApplyURL,
			"isRemote":           job.IsRemote,
			"secondaryLocations": secondaryLocations,
			"workplaceType":      job.WorkplaceType,
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("encode ashby metadata: %w", err)
		}

		jobs = append(jobs, Job{
			ExternalJobID:  strings.TrimSpace(job.ID),
			Title:          strings.TrimSpace(job.Title),
			Location:       strings.TrimSpace(job.Location),
			Department:     strings.TrimSpace(job.Department),
			Team:           strings.TrimSpace(job.Team),
			EmploymentType: strings.TrimSpace(job.EmploymentType),
			JobURL:         strings.TrimSpace(job.JobURL),
			MetadataJSON:   string(metadataJSON),
			RawJSON:        string(raw),
		})
	}

	return jobs, nil
}
