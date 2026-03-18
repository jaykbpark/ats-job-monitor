package audit

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/signals"
)

//go:embed targets.json
var defaultTargetsJSON []byte

type SignalAuditTarget struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	BoardKey string `json:"boardKey"`
}

type SignalAuditOptions struct {
	ProviderFilter string
	SampleSize     int
	TargetLimit    int
}

type SignalAuditReport struct {
	GeneratedAt string                   `json:"generatedAt"`
	SampleSize  int                      `json:"sampleSize"`
	Summary     SignalAuditSummary       `json:"summary"`
	Targets     []SignalAuditTargetAudit `json:"targets"`
}

type SignalAuditSummary struct {
	TotalTargets        int `json:"totalTargets"`
	SuccessfulTargets   int `json:"successfulTargets"`
	FailedTargets       int `json:"failedTargets"`
	TotalFetchedJobs    int `json:"totalFetchedJobs"`
	TotalEngineering    int `json:"totalEngineeringJobs"`
	TotalSampledJobs    int `json:"totalSampledJobs"`
	AutomaticChecks     int `json:"automaticChecks"`
	AutomaticCheckFails int `json:"automaticCheckFails"`
}

type SignalAuditTargetAudit struct {
	Name            string                 `json:"name"`
	Provider        string                 `json:"provider"`
	BoardKey        string                 `json:"boardKey"`
	FetchedJobs     int                    `json:"fetchedJobs"`
	EngineeringJobs int                    `json:"engineeringJobs"`
	SampledJobs     int                    `json:"sampledJobs"`
	Error           string                 `json:"error,omitempty"`
	Jobs            []SignalAuditJobReport `json:"jobs"`
}

type SignalAuditJobReport struct {
	ExternalJobID  string             `json:"externalJobId"`
	Title          string             `json:"title"`
	JobURL         string             `json:"jobUrl"`
	Location       string             `json:"location,omitempty"`
	Department     string             `json:"department,omitempty"`
	Team           string             `json:"team,omitempty"`
	EmploymentType string             `json:"employmentType,omitempty"`
	EvidenceText   string             `json:"evidenceText,omitempty"`
	Signals        signals.JobSignals `json:"signals"`
	Checks         []SignalAuditCheck `json:"checks"`
}

type SignalAuditCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type jobFetcher interface {
	FetchJobs(ctx context.Context, boardKey string) ([]providers.Job, error)
}

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var (
	auditRangeYearsPattern   = regexp.MustCompile(`\b(\d{1,2})\s*(?:-|to)\s*(\d{1,2})\s*(?:years|year|yrs|yr)\b`)
	auditMinYearsPattern     = regexp.MustCompile(`\b(\d{1,2})\+?\s*(?:years|year|yrs|yr)(?:\s+(?:of\s+)?experience)?\b`)
	auditAtLeastYearsPattern = regexp.MustCompile(`\b(?:at least|minimum of)\s+(\d{1,2})\s*(?:years|year|yrs|yr)\b`)
)

func DefaultTargets() ([]SignalAuditTarget, error) {
	var targets []SignalAuditTarget
	if err := json.Unmarshal(defaultTargetsJSON, &targets); err != nil {
		return nil, fmt.Errorf("decode default audit targets: %w", err)
	}

	return targets, nil
}

func RunSignalAudit(ctx context.Context, options SignalAuditOptions) (SignalAuditReport, error) {
	targets, err := DefaultTargets()
	if err != nil {
		return SignalAuditReport{}, err
	}

	filteredTargets := filterTargets(targets, options.ProviderFilter, options.TargetLimit)
	sampleSize := options.SampleSize
	if sampleSize <= 0 {
		sampleSize = 3
	}

	report := SignalAuditReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SampleSize:  sampleSize,
		Targets:     make([]SignalAuditTargetAudit, 0, len(filteredTargets)),
	}

	fetchers := map[string]jobFetcher{
		"greenhouse": providers.NewGreenhouseClient(),
		"lever":      providers.NewLeverClient(),
		"ashby":      providers.NewAshbyClient(),
	}

	for _, target := range filteredTargets {
		targetReport := SignalAuditTargetAudit{
			Name:     target.Name,
			Provider: target.Provider,
			BoardKey: target.BoardKey,
			Jobs:     []SignalAuditJobReport{},
		}

		report.Summary.TotalTargets++

		fetcher, ok := fetchers[target.Provider]
		if !ok {
			targetReport.Error = fmt.Sprintf("unsupported provider %q", target.Provider)
			report.Summary.FailedTargets++
			report.Targets = append(report.Targets, targetReport)
			continue
		}

		jobs, err := fetchJobsForAudit(ctx, target, fetcher)
		if err != nil {
			targetReport.Error = err.Error()
			report.Summary.FailedTargets++
			report.Targets = append(report.Targets, targetReport)
			continue
		}

		targetReport.FetchedJobs = len(jobs)
		report.Summary.TotalFetchedJobs += len(jobs)

		engineeringJobs := sampleEngineeringJobs(jobs, 0)
		targetReport.EngineeringJobs = len(engineeringJobs)
		report.Summary.TotalEngineering += len(engineeringJobs)

		sampledJobs := engineeringJobs
		if len(sampledJobs) > sampleSize {
			sampledJobs = sampledJobs[:sampleSize]
		}

		targetReport.SampledJobs = len(sampledJobs)
		report.Summary.TotalSampledJobs += len(sampledJobs)

		for _, job := range sampledJobs {
			derived := signals.Derive(job)
			checks := buildChecks(target.Provider, job, derived)

			for _, check := range checks {
				if check.Status == "skip" {
					continue
				}
				report.Summary.AutomaticChecks++
				if check.Status == "fail" {
					report.Summary.AutomaticCheckFails++
				}
			}

			targetReport.Jobs = append(targetReport.Jobs, SignalAuditJobReport{
				ExternalJobID:  job.ExternalJobID,
				Title:          job.Title,
				JobURL:         job.JobURL,
				Location:       job.Location,
				Department:     job.Department,
				Team:           job.Team,
				EmploymentType: job.EmploymentType,
				EvidenceText:   extractEvidenceText(job),
				Signals:        derived,
				Checks:         checks,
			})
		}

		report.Summary.SuccessfulTargets++
		report.Targets = append(report.Targets, targetReport)
	}

	return report, nil
}

func RenderMarkdownReport(report SignalAuditReport) string {
	var builder strings.Builder

	builder.WriteString("# Signal Audit\n\n")
	builder.WriteString(fmt.Sprintf("- Generated at: `%s`\n", report.GeneratedAt))
	builder.WriteString(fmt.Sprintf("- Targets audited: `%d`\n", report.Summary.TotalTargets))
	builder.WriteString(fmt.Sprintf("- Successful targets: `%d`\n", report.Summary.SuccessfulTargets))
	builder.WriteString(fmt.Sprintf("- Failed targets: `%d`\n", report.Summary.FailedTargets))
	builder.WriteString(fmt.Sprintf("- Jobs fetched: `%d`\n", report.Summary.TotalFetchedJobs))
	builder.WriteString(fmt.Sprintf("- Engineering jobs found: `%d`\n", report.Summary.TotalEngineering))
	builder.WriteString(fmt.Sprintf("- Sampled jobs: `%d`\n", report.Summary.TotalSampledJobs))
	builder.WriteString(fmt.Sprintf("- Automatic checks: `%d`\n", report.Summary.AutomaticChecks))
	builder.WriteString(fmt.Sprintf("- Automatic check failures: `%d`\n", report.Summary.AutomaticCheckFails))

	for _, target := range report.Targets {
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("## %s\n\n", target.Name))
		builder.WriteString(fmt.Sprintf("- Provider: `%s`\n", target.Provider))
		builder.WriteString(fmt.Sprintf("- Board key: `%s`\n", target.BoardKey))
		builder.WriteString(fmt.Sprintf("- Fetched jobs: `%d`\n", target.FetchedJobs))
		builder.WriteString(fmt.Sprintf("- Engineering jobs: `%d`\n", target.EngineeringJobs))
		builder.WriteString(fmt.Sprintf("- Sampled jobs: `%d`\n", target.SampledJobs))

		if target.Error != "" {
			builder.WriteString(fmt.Sprintf("- Error: `%s`\n", target.Error))
			continue
		}

		for _, job := range target.Jobs {
			builder.WriteString("\n")
			builder.WriteString(fmt.Sprintf("### %s\n\n", job.Title))
			builder.WriteString(fmt.Sprintf("- URL: %s\n", job.JobURL))
			if job.Location != "" {
				builder.WriteString(fmt.Sprintf("- Source location: `%s`\n", job.Location))
			}
			if job.EmploymentType != "" {
				builder.WriteString(fmt.Sprintf("- Source employment type: `%s`\n", job.EmploymentType))
			}
			if job.Department != "" {
				builder.WriteString(fmt.Sprintf("- Department: `%s`\n", job.Department))
			}
			if job.Team != "" {
				builder.WriteString(fmt.Sprintf("- Team: `%s`\n", job.Team))
			}

			builder.WriteString(fmt.Sprintf("- Derived location: `%s`\n", job.Signals.NormalizedLocation))
			builder.WriteString(fmt.Sprintf("- Derived remote: `%t`\n", job.Signals.IsRemote))
			builder.WriteString(fmt.Sprintf("- Derived employment type: `%s`\n", job.Signals.NormalizedEmploymentType))
			builder.WriteString(fmt.Sprintf("- Derived seniority: `%s`\n", job.Signals.Seniority))
			builder.WriteString(fmt.Sprintf("- Derived experience confidence: `%s`\n", job.Signals.ExperienceConfidence))
			if job.Signals.MinYearsExperience != nil {
				builder.WriteString(fmt.Sprintf("- Derived min years: `%d`\n", *job.Signals.MinYearsExperience))
			}
			if job.Signals.MaxYearsExperience != nil {
				builder.WriteString(fmt.Sprintf("- Derived max years: `%d`\n", *job.Signals.MaxYearsExperience))
			}

			if job.EvidenceText != "" {
				builder.WriteString(fmt.Sprintf("- Evidence snippet: %s\n", job.EvidenceText))
			}

			builder.WriteString("- Checks:\n")
			for _, check := range job.Checks {
				line := fmt.Sprintf("  - `%s`: `%s`", check.Name, check.Status)
				if check.Expected != "" || check.Actual != "" {
					line += fmt.Sprintf(" (expected `%s`, actual `%s`)", check.Expected, check.Actual)
				}
				if check.Notes != "" {
					line += fmt.Sprintf(" - %s", check.Notes)
				}
				builder.WriteString(line + "\n")
			}
		}
	}

	return builder.String()
}

func filterTargets(targets []SignalAuditTarget, providerFilter string, limit int) []SignalAuditTarget {
	filtered := make([]SignalAuditTarget, 0, len(targets))
	normalizedProviderFilter := strings.TrimSpace(strings.ToLower(providerFilter))
	for _, target := range targets {
		if normalizedProviderFilter != "" && strings.ToLower(target.Provider) != normalizedProviderFilter {
			continue
		}

		filtered = append(filtered, target)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func sampleEngineeringJobs(jobs []providers.Job, limit int) []providers.Job {
	sampled := make([]providers.Job, 0, len(jobs))
	for _, job := range jobs {
		if !isEngineeringJob(job) {
			continue
		}

		sampled = append(sampled, job)
		if limit > 0 && len(sampled) >= limit {
			break
		}
	}

	return sampled
}

func isEngineeringJob(job providers.Job) bool {
	title := normalizeForMatch(job.Title)
	scope := normalizeForMatch(strings.Join([]string{job.Title, job.Department, job.Team}, " "))

	excludedTitles := []string{
		"sales engineer",
		"solutions engineer",
		"support engineer",
		"customer engineer",
		"forward deployed engineer",
		"developer advocacy",
		"account executive",
		"account manager",
		"customer success",
		"recruiter",
		"sourcer",
		"designer",
		"marketing",
		"partnerships",
		"people partner",
		"finance",
		"legal",
		"attorney",
		"counsel",
		"physical security",
	}

	for _, candidate := range excludedTitles {
		if strings.Contains(title, candidate) {
			return false
		}
	}

	titleKeywords := []string{
		"engineer",
		"developer",
		"software",
		"frontend",
		"backend",
		"full stack",
		"fullstack",
		"infrastructure",
		"devops",
		"sre",
		"mobile",
		"ios",
		"android",
		"machine learning",
		"firmware",
		"embedded",
		"architect",
	}

	for _, candidate := range titleKeywords {
		if strings.Contains(title, candidate) {
			return true
		}
	}

	if strings.Contains(scope, "engineering") {
		leadershipKeywords := []string{
			"manager",
			"director",
			"lead",
			"head",
			"scientist",
		}
		for _, candidate := range leadershipKeywords {
			if strings.Contains(title, candidate) {
				return true
			}
		}
	}

	return false
}

func normalizeForMatch(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ",", " ", ".", " ", "(", " ", ")", " ")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func fetchJobsForAudit(ctx context.Context, target SignalAuditTarget, fetcher jobFetcher) ([]providers.Job, error) {
	if target.Provider == "greenhouse" {
		if greenhouseClient, ok := fetcher.(*providers.GreenhouseClient); ok {
			return greenhouseClient.FetchJobsWithContent(ctx, target.BoardKey)
		}
	}

	return fetcher.FetchJobs(ctx, target.BoardKey)
}

func buildChecks(provider string, job providers.Job, derived signals.JobSignals) []SignalAuditCheck {
	checks := []SignalAuditCheck{
		{
			Name:     "search_text_populated",
			Status:   passOrFail(derived.SearchText != ""),
			Expected: "non-empty",
			Actual:   presenceValue(derived.SearchText),
		},
		{
			Name:     "normalized_location_populated",
			Status:   locationCheckStatus(job.Location, derived.NormalizedLocation),
			Expected: expectedLocation(job.Location),
			Actual:   emptyOrValue(derived.NormalizedLocation),
		},
		{
			Name:     "employment_type_captured",
			Status:   employmentTypeCheckStatus(job.EmploymentType, derived.NormalizedEmploymentType),
			Expected: expectedEmploymentType(job.EmploymentType),
			Actual:   derived.NormalizedEmploymentType,
		},
	}

	if expectsEvidenceText(job) {
		evidenceText := extractEvidenceFullText(job)
		checks = append(checks, SignalAuditCheck{
			Name:     "evidence_text_present",
			Status:   passOrFail(strings.TrimSpace(evidenceText) != ""),
			Expected: "non-empty",
			Actual:   presenceValue(evidenceText),
		})
	} else {
		checks = append(checks, SignalAuditCheck{
			Name:   "evidence_text_present",
			Status: "skip",
			Notes:  "provider payload does not expose a description/content field for this job",
		})
	}

	if expected, notes, ok := expectedRemote(provider, job); ok {
		checks = append(checks, SignalAuditCheck{
			Name:     "remote_matches_provider_signal",
			Status:   passOrFail(derived.IsRemote == expected),
			Expected: fmt.Sprintf("%t", expected),
			Actual:   fmt.Sprintf("%t", derived.IsRemote),
			Notes:    notes,
		})
	} else {
		checks = append(checks, SignalAuditCheck{
			Name:   "remote_matches_provider_signal",
			Status: "skip",
			Notes:  "provider payload does not expose explicit remote ground truth for this audit",
		})
	}

	if expected, ok := explicitTitleSeniority(job.Title); ok {
		checks = append(checks, SignalAuditCheck{
			Name:     "seniority_matches_title",
			Status:   passOrFail(derived.Seniority == expected),
			Expected: expected,
			Actual:   derived.Seniority,
		})
	} else {
		checks = append(checks, SignalAuditCheck{
			Name:   "seniority_matches_title",
			Status: "skip",
			Notes:  "title does not contain an explicit seniority token",
		})
	}

	if expectedMin, expectedMax, ok := explicitExperienceFromEvidence(job); ok {
		checks = append(checks, SignalAuditCheck{
			Name:     "experience_matches_evidence",
			Status:   passOrFail(intPointersEqual(expectedMin, derived.MinYearsExperience) && intPointersEqual(expectedMax, derived.MaxYearsExperience)),
			Expected: formatExperienceRange(expectedMin, expectedMax),
			Actual:   formatExperienceRange(derived.MinYearsExperience, derived.MaxYearsExperience),
		})
	} else {
		checks = append(checks, SignalAuditCheck{
			Name:   "experience_matches_evidence",
			Status: "skip",
			Notes:  "description does not contain an explicit years-of-experience pattern",
		})
	}

	return checks
}

func passOrFail(passed bool) string {
	if passed {
		return "pass"
	}
	return "fail"
}

func locationCheckStatus(sourceLocation string, normalizedLocation string) string {
	if strings.TrimSpace(sourceLocation) == "" {
		return "skip"
	}
	if strings.TrimSpace(normalizedLocation) == "" {
		return "fail"
	}
	return "pass"
}

func employmentTypeCheckStatus(sourceEmploymentType string, normalizedEmploymentType string) string {
	if strings.TrimSpace(sourceEmploymentType) == "" {
		return "skip"
	}
	if strings.TrimSpace(normalizedEmploymentType) == "" || normalizedEmploymentType == "unknown" {
		return "fail"
	}
	return "pass"
}

func expectedLocation(sourceLocation string) string {
	if strings.TrimSpace(sourceLocation) == "" {
		return "n/a"
	}
	return "non-empty"
}

func expectedEmploymentType(sourceEmploymentType string) string {
	if strings.TrimSpace(sourceEmploymentType) == "" {
		return "n/a"
	}
	return "known normalized value"
}

func expectedRemote(provider string, job providers.Job) (bool, string, bool) {
	switch provider {
	case "ashby":
		var metadata struct {
			IsRemote      bool   `json:"isRemote"`
			WorkplaceType string `json:"workplaceType"`
		}
		if err := json.Unmarshal([]byte(job.MetadataJSON), &metadata); err != nil {
			return false, "", false
		}
		return metadata.IsRemote, fmt.Sprintf("Ashby metadata workplaceType=%q", metadata.WorkplaceType), true
	case "lever":
		var metadata struct {
			WorkplaceType string `json:"workplaceType"`
		}
		if err := json.Unmarshal([]byte(job.MetadataJSON), &metadata); err != nil {
			return false, "", false
		}

		workplaceType := strings.ToLower(strings.TrimSpace(metadata.WorkplaceType))
		switch workplaceType {
		case "remote":
			return true, "Lever workplaceType=remote", true
		case "hybrid":
			return false, "Lever workplaceType=hybrid", true
		case "onsite", "on site":
			return false, fmt.Sprintf("Lever workplaceType=%s", workplaceType), true
		default:
			return false, "", false
		}
	default:
		return false, "", false
	}
}

func explicitTitleSeniority(title string) (string, bool) {
	normalized := normalizeForMatch(title)

	patterns := []struct {
		token string
		value string
	}{
		{token: "principal", value: "principal"},
		{token: "staff", value: "staff"},
		{token: "director", value: "director"},
		{token: "manager", value: "manager"},
		{token: "senior", value: "senior"},
		{token: " sr ", value: "senior"},
		{token: "junior", value: "junior"},
		{token: " jr ", value: "junior"},
		{token: "associate", value: "junior"},
		{token: "intern", value: "intern"},
		{token: "new grad", value: "entry"},
	}

	for _, pattern := range patterns {
		if strings.Contains(" "+normalized+" ", pattern.token) {
			return pattern.value, true
		}
	}

	return "", false
}

func extractEvidenceText(job providers.Job) string {
	evidenceText := extractEvidenceFullText(job)
	if evidenceText == "" {
		return ""
	}

	return trimSnippet(evidenceText, 320)
}

func extractEvidenceFullText(job providers.Job) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(job.RawJSON), &payload); err != nil {
		return ""
	}

	candidateFields := []string{
		"descriptionPlain",
		"additionalPlain",
		"descriptionHtml",
		"description",
		"content",
	}

	for _, field := range candidateFields {
		value, ok := payload[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}

		return cleanEvidenceText(value)
	}

	return ""
}

func trimSnippet(value string, limit int) string {
	cleaned := cleanEvidenceText(value)
	if limit <= 0 || len(cleaned) <= limit {
		return cleaned
	}

	return strings.TrimSpace(cleaned[:limit]) + "..."
}

func cleanEvidenceText(value string) string {
	cleaned := html.UnescapeString(value)
	cleaned = htmlTagPattern.ReplaceAllString(cleaned, " ")
	return strings.Join(strings.Fields(cleaned), " ")
}

func emptyOrValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return value
}

func presenceValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return "<present>"
}

func expectsEvidenceText(job providers.Job) bool {
	var payload map[string]any
	if err := json.Unmarshal([]byte(job.RawJSON), &payload); err != nil {
		return false
	}

	for _, field := range []string{"descriptionPlain", "additionalPlain", "descriptionHtml", "description", "content"} {
		if _, ok := payload[field]; ok {
			return true
		}
	}

	return false
}

func explicitExperienceFromEvidence(job providers.Job) (*int, *int, bool) {
	evidenceText := strings.ToLower(extractEvidenceFullText(job))
	if strings.TrimSpace(evidenceText) == "" {
		return nil, nil, false
	}

	if matches := auditRangeYearsPattern.FindStringSubmatch(evidenceText); len(matches) == 3 {
		minValue := parseInt(matches[1])
		maxValue := parseInt(matches[2])
		return &minValue, &maxValue, true
	}

	if matches := auditAtLeastYearsPattern.FindStringSubmatch(evidenceText); len(matches) == 2 {
		minValue := parseInt(matches[1])
		return &minValue, nil, true
	}

	if matches := auditMinYearsPattern.FindStringSubmatch(evidenceText); len(matches) == 2 {
		minValue := parseInt(matches[1])
		return &minValue, nil, true
	}

	return nil, nil, false
}

func formatExperienceRange(minYears *int, maxYears *int) string {
	switch {
	case minYears == nil && maxYears == nil:
		return "unknown"
	case minYears != nil && maxYears != nil:
		return fmt.Sprintf("%d-%d", *minYears, *maxYears)
	case minYears != nil:
		return fmt.Sprintf("%d+", *minYears)
	default:
		return "unknown"
	}
}

func intPointersEqual(left *int, right *int) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func parseInt(value string) int {
	var parsed int
	_, _ = fmt.Sscanf(value, "%d", &parsed)
	return parsed
}
