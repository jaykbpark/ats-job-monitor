package signals

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
)

type JobSignals struct {
	SearchText               string `json:"searchText"`
	NormalizedLocation       string `json:"normalizedLocation"`
	IsRemote                 bool   `json:"isRemote"`
	NormalizedEmploymentType string `json:"normalizedEmploymentType"`
	Seniority                string `json:"seniority"`
	MinYearsExperience       *int   `json:"minYearsExperience,omitempty"`
	MaxYearsExperience       *int   `json:"maxYearsExperience,omitempty"`
	ExperienceConfidence     string `json:"experienceConfidence"`
}

var (
	rangeYearsPattern   = regexp.MustCompile(`\b(\d{1,2})\s*(?:-|to)\s*(\d{1,2})\s*(?:years|year|yrs|yr)\b`)
	minYearsPattern     = regexp.MustCompile(`\b(\d{1,2})\+?\s*(?:years|year|yrs|yr)(?:\s+(?:of\s+)experience)?\b`)
	atLeastYearsPattern = regexp.MustCompile(`\b(?:at least|minimum of)\s+(\d{1,2})\s*(?:years|year|yrs|yr)\b`)
)

func Derive(job providers.Job) JobSignals {
	rawSearchSource := strings.Join([]string{
		job.Title,
		job.Department,
		job.Team,
		job.Location,
		job.MetadataJSON,
		job.RawJSON,
	}, " ")

	searchText := normalizeText(rawSearchSource)
	normalizedLocation := normalizeText(job.Location)
	minYears, maxYears, experienceConfidence := deriveExperience(rawSearchSource)

	return JobSignals{
		SearchText:               searchText,
		NormalizedLocation:       normalizedLocation,
		IsRemote:                 deriveIsRemote(job.Location, job.MetadataJSON, job.RawJSON),
		NormalizedEmploymentType: normalizeEmploymentType(job.EmploymentType),
		Seniority:                deriveSeniority(searchText),
		MinYearsExperience:       minYears,
		MaxYearsExperience:       maxYears,
		ExperienceConfidence:     experienceConfidence,
	}
}

func normalizeText(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	lastSpace := false
	for _, r := range strings.ToLower(value) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			builder.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r) || isCommonSeparator(r):
			if !lastSpace {
				builder.WriteByte(' ')
				lastSpace = true
			}
		}
	}

	return strings.TrimSpace(builder.String())
}

func isCommonSeparator(r rune) bool {
	switch r {
	case ',', '-', '/', '_', '.', ':', ';', '(', ')', '[', ']', '{', '}', '|':
		return true
	default:
		return false
	}
}

func deriveIsRemote(location string, metadataJSON string, rawJSON string) bool {
	searchSpace := normalizeText(strings.Join([]string{location, metadataJSON, rawJSON}, " "))

	remoteMarkers := []string{
		"remote",
		"anywhere",
		"work from home",
		"distributed",
	}

	for _, marker := range remoteMarkers {
		if strings.Contains(searchSpace, marker) {
			return true
		}
	}

	return false
}

func normalizeEmploymentType(value string) string {
	normalized := normalizeText(value)

	switch {
	case normalized == "":
		return "unknown"
	case strings.Contains(normalized, "intern"):
		return "internship"
	case strings.Contains(normalized, "contract"):
		return "contract"
	case strings.Contains(normalized, "temporary") || strings.Contains(normalized, "temp"):
		return "temporary"
	case strings.Contains(normalized, "part time") || strings.Contains(normalized, "parttime"):
		return "part-time"
	case strings.Contains(normalized, "full time") || strings.Contains(normalized, "fulltime"):
		return "full-time"
	default:
		return "unknown"
	}
}

func deriveSeniority(searchText string) string {
	type pattern struct {
		regex *regexp.Regexp
		value string
	}

	patterns := []pattern{
		{regexp.MustCompile(`\bprincipal\b`), "principal"},
		{regexp.MustCompile(`\bstaff\b`), "staff"},
		{regexp.MustCompile(`\bdirector\b|\bhead of\b|\bvp\b`), "director"},
		{regexp.MustCompile(`\bmanager\b`), "manager"},
		{regexp.MustCompile(`\bsenior\b|\bsr\b`), "senior"},
		{regexp.MustCompile(`\bjunior\b|\bjr\b|\bassociate\b`), "junior"},
		{regexp.MustCompile(`\bintern\b|\binternship\b`), "intern"},
		{regexp.MustCompile(`\bnew grad\b|\bgraduate\b|\bentry level\b`), "entry"},
		{regexp.MustCompile(`\bmid\b|\bmid level\b`), "mid"},
	}

	for _, candidate := range patterns {
		if candidate.regex.MatchString(searchText) {
			return candidate.value
		}
	}

	return "unknown"
}

func deriveExperience(rawSearchSource string) (*int, *int, string) {
	raw := strings.ToLower(rawSearchSource)

	if matches := rangeYearsPattern.FindStringSubmatch(raw); len(matches) == 3 {
		minValue := parseInt(matches[1])
		maxValue := parseInt(matches[2])
		return &minValue, &maxValue, "high"
	}

	if matches := atLeastYearsPattern.FindStringSubmatch(raw); len(matches) == 2 {
		minValue := parseInt(matches[1])
		return &minValue, nil, "high"
	}

	if matches := minYearsPattern.FindStringSubmatch(raw); len(matches) == 2 {
		minValue := parseInt(matches[1])
		return &minValue, nil, "high"
	}

	return nil, nil, "unknown"
}

func parseInt(value string) int {
	var parsed int
	_, _ = fmt.Sscanf(value, "%d", &parsed)
	return parsed
}
