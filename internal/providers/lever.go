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

type LeverClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

type leverJob struct {
	ID          string          `json:"id"`
	Text        string          `json:"text"`
	Categories  leverCategories `json:"categories"`
	HostedURL   string          `json:"hostedUrl"`
	Description string          `json:"descriptionPlain"`
	Additional  string          `json:"additionalPlain"`
	Lists       []leverList     `json:"lists"`
	Meta        map[string]any  `json:"meta"`
	Workplace   string          `json:"workplaceType"`
}

type leverCategories struct {
	Location   string `json:"location"`
	Commitment string `json:"commitment"`
	Team       string `json:"team"`
	Department string `json:"department"`
}

type leverList struct {
	Text string `json:"text"`
}

func NewLeverClient() *LeverClient {
	return &LeverClient{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    "https://api.lever.co",
	}
}

func (c *LeverClient) FetchJobs(ctx context.Context, boardKey string) ([]Job, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.lever.co"
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	requestURL := fmt.Sprintf("%s/v0/postings/%s?mode=json", baseURL, url.PathEscape(boardKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build lever request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch lever jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lever jobs returned status %d", resp.StatusCode)
	}

	var payload []leverJob
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode lever jobs: %w", err)
	}

	jobs := make([]Job, 0, len(payload))
	for _, job := range payload {
		raw, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("encode lever raw job: %w", err)
		}

		metadata := map[string]any{
			"lists":         job.Lists,
			"meta":          job.Meta,
			"workplaceType": job.Workplace,
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("encode lever metadata: %w", err)
		}

		jobs = append(jobs, Job{
			ExternalJobID:  strings.TrimSpace(job.ID),
			Title:          strings.TrimSpace(job.Text),
			Location:       strings.TrimSpace(job.Categories.Location),
			Department:     firstNonEmpty(job.Categories.Department, job.Categories.Team),
			Team:           strings.TrimSpace(job.Categories.Team),
			EmploymentType: strings.TrimSpace(job.Categories.Commitment),
			JobURL:         strings.TrimSpace(job.HostedURL),
			MetadataJSON:   string(metadataJSON),
			RawJSON:        string(raw),
		})
	}

	return jobs, nil
}
