package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	Content     string                `json:"content"`
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

var greenhouseEmploymentTypePatterns = []struct {
	pattern *regexp.Regexp
	value   string
}{
	{regexp.MustCompile(`(?i)\b(?:this is|we are hiring for|we are seeking|we are looking for|this position is|this role is|the role is|the position is)\s+(?:a|an)?\s*full[- ]time\s+(?:role|position|job)\b`), "full-time"},
	{regexp.MustCompile(`(?i)\bfull[- ]time\s+(?:role|position|job)\b`), "full-time"},
	{regexp.MustCompile(`(?i)\b(?:this is|we are hiring for|we are seeking|we are looking for|this position is|this role is|the role is|the position is)\s+(?:a|an)?\s*part[- ]time\s+(?:role|position|job)\b`), "part-time"},
	{regexp.MustCompile(`(?i)\bpart[- ]time\s+(?:role|position|job)\b`), "part-time"},
	{regexp.MustCompile(`(?i)\b(?:this is|we are hiring for|we are seeking|we are looking for|this position is|this role is|the role is|the position is)\s+(?:a|an)?\s*contract(?:or)?\s+(?:role|position|job)\b`), "contract"},
	{regexp.MustCompile(`(?i)\bcontract(?:or)?\s+(?:role|position|job)\b`), "contract"},
	{regexp.MustCompile(`(?i)\b(?:this is|we are hiring for|we are seeking|we are looking for|this position is|this role is|the role is|the position is)\s+(?:a|an)?\s*(?:internship|intern)\s+(?:role|position|job)\b`), "internship"},
	{regexp.MustCompile(`(?i)\b(?:internship|intern)\s+(?:role|position|job)\b`), "internship"},
	{regexp.MustCompile(`(?i)\b(?:this is|we are hiring for|we are seeking|we are looking for|this position is|this role is|the role is|the position is)\s+(?:a|an)?\s*temporary\s+(?:role|position|job)\b`), "temporary"},
	{regexp.MustCompile(`(?i)\btemporary\s+(?:role|position|job)\b`), "temporary"},
}

func NewGreenhouseClient() *GreenhouseClient {
	return &GreenhouseClient{
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		BaseURL:    "https://boards-api.greenhouse.io",
	}
}

func (c *GreenhouseClient) FetchJobs(ctx context.Context, boardKey string) ([]Job, error) {
	return c.fetchJobs(ctx, boardKey, false)
}

func (c *GreenhouseClient) FetchJobsWithContent(ctx context.Context, boardKey string) ([]Job, error) {
	return c.fetchJobs(ctx, boardKey, true)
}

func (c *GreenhouseClient) fetchJobs(ctx context.Context, boardKey string, includeContent bool) ([]Job, error) {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://boards-api.greenhouse.io"
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	requestURL := fmt.Sprintf("%s/v1/boards/%s/jobs?content=%t", baseURL, url.PathEscape(boardKey), includeContent)
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
		employmentType := deriveGreenhouseEmploymentType(job.Metadata, job.Content)

		jobs = append(jobs, Job{
			ExternalJobID:  fmt.Sprintf("%d", job.ID),
			Title:          job.Title,
			Location:       firstNonEmpty(job.Location.Name, firstNamedItem(job.Offices)),
			Department:     firstNamedItem(job.Departments),
			EmploymentType: employmentType,
			Team:           "",
			JobURL:         job.AbsoluteURL,
			MetadataJSON:   string(metadataJSON),
			RawJSON:        string(raw),
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

func deriveGreenhouseEmploymentType(metadataJSON []byte, content string) string {
	if employmentType := greenhouseEmploymentTypeFromMetadata(metadataJSON); employmentType != "" {
		return employmentType
	}

	return greenhouseEmploymentTypeFromContent(content)
}

func greenhouseEmploymentTypeFromMetadata(metadataJSON []byte) string {
	var payload any
	if err := json.Unmarshal(metadataJSON, &payload); err != nil {
		return ""
	}

	return greenhouseEmploymentTypeFromValue(payload)
}

func greenhouseEmploymentTypeFromValue(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if employmentType := greenhouseEmploymentTypeFromMap(typed); employmentType != "" {
			return employmentType
		}
		for _, child := range typed {
			if employmentType := greenhouseEmploymentTypeFromValue(child); employmentType != "" {
				return employmentType
			}
		}
	case []any:
		for _, child := range typed {
			if employmentType := greenhouseEmploymentTypeFromValue(child); employmentType != "" {
				return employmentType
			}
		}
	case string:
		return greenhouseEmploymentTypeFromString(typed)
	}

	return ""
}

func greenhouseEmploymentTypeFromMap(value map[string]any) string {
	if nameValue, ok := greenhouseStringValue(value, "name"); ok {
		if greenhouseIsEmploymentFieldName(nameValue) {
			if candidate, ok := greenhouseStringValue(value, "value"); ok {
				if normalized := greenhouseEmploymentTypeFromString(candidate); normalized != "" {
					return normalized
				}
			}
		}
	}

	for key, candidate := range value {
		if matched := greenhouseEmploymentTypeFromField(key, candidate); matched != "" {
			return matched
		}
	}

	return ""
}

func greenhouseEmploymentTypeFromField(key string, value any) string {
	switch normalizeGreenhouseKey(key) {
	case "timetype", "employmenttype", "employment", "employmentstatus", "jobtype", "commitment":
		return greenhouseEmploymentTypeFromValue(value)
	}

	return ""
}

func greenhouseIsEmploymentFieldName(value string) bool {
	switch normalizeGreenhouseKey(value) {
	case "timetype", "employmenttype", "employment", "employmentstatus", "jobtype", "commitment":
		return true
	default:
		return false
	}
}

func greenhouseEmploymentTypeFromContent(content string) string {
	text := greenhouseCleanText(content)
	if text == "" {
		return ""
	}

	for _, candidate := range greenhouseEmploymentTypePatterns {
		if candidate.pattern.MatchString(text) {
			return candidate.value
		}
	}

	return ""
}

func greenhouseEmploymentTypeFromString(value string) string {
	normalized := normalizeGreenhouseKey(value)
	switch normalized {
	case "fulltime", "fulltimeemployment", "fulltimeposition", "fulltimerole":
		return "full-time"
	case "parttime", "parttimeemployment", "parttimeposition", "parttimerole":
		return "part-time"
	case "contract", "contractor", "contractposition", "contractrole":
		return "contract"
	case "intern", "internship", "internposition", "internshipposition", "internshiprole":
		return "internship"
	case "temporary", "temp", "temprole", "tempposition":
		return "temporary"
	default:
		return ""
	}
}

func greenhouseStringValue(value map[string]any, key string) (string, bool) {
	raw, ok := value[key]
	if !ok {
		return "", false
	}

	str, ok := raw.(string)
	if !ok {
		return "", false
	}

	return str, true
}

func normalizeGreenhouseKey(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func greenhouseCleanText(value string) string {
	value = strings.ToLower(value)
	value = htmlUnescape(value)
	value = greenhouseHTMLTagPattern.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	return strings.Join(strings.Fields(value), " ")
}

var greenhouseHTMLTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)

func htmlUnescape(value string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&#34;", `"`,
	)
	return replacer.Replace(value)
}
