package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GreenhouseClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

type greenhouseJobsResponse struct {
	Jobs []greenhouseJob `json:"jobs"`
}

type greenhouseJob struct {
	ID          int64                 `json:"id"`
	Title       string                `json:"title"`
	AbsoluteURL string                `json:"absolute_url"`
	Location    greenhouseLocation    `json:"location"`
	Metadata    json.RawMessage       `json:"metadata"`
	Departments []greenhouseNamedItem `json:"departments"`
	Offices     []greenhouseNamedItem `json:"offices"`
}

type greenhouseLocation struct {
	Name string `json:"name"`
}

type greenhouseNamedItem struct {
	Name string `json:"name"`
}

func NewGreenhouseClient() *GreenhouseClient {
	return &GreenhouseClient{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		BaseURL:    "https://boards-api.greenhouse.io",
	}
}

func (c *GreenhouseClient) FetchJobs(ctx context.Context, boardKey string) ([]Job, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://boards-api.greenhouse.io"
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	requestURL := fmt.Sprintf("%s/v1/boards/%s/jobs?content=false", baseURL, url.PathEscape(boardKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build greenhouse request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch greenhouse jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("greenhouse jobs returned status %d", resp.StatusCode)
	}

	var payload greenhouseJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode greenhouse jobs: %w", err)
	}

	jobs := make([]Job, 0, len(payload.Jobs))
	for _, job := range payload.Jobs {
		raw, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("encode greenhouse raw job: %w", err)
		}

		metadataJSON := normalizeRawJSON(job.Metadata, []byte(`{}`))

		jobs = append(jobs, Job{
			ExternalJobID: fmt.Sprintf("%d", job.ID),
			Title:         job.Title,
			Location:      firstNonEmpty(job.Location.Name, firstNamedItem(job.Offices)),
			Department:    firstNamedItem(job.Departments),
			Team:          "",
			JobURL:        job.AbsoluteURL,
			MetadataJSON:  string(metadataJSON),
			RawJSON:       string(raw),
		})
	}

	return jobs, nil
}

func firstNamedItem(items []greenhouseNamedItem) string {
	for _, item := range items {
		if name := strings.TrimSpace(item.Name); name != "" {
			return name
		}
	}

	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func normalizeRawJSON(value []byte, fallback []byte) string {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return string(fallback)
	}

	return string(trimmed)
}
