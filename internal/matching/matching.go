package matching

import (
	"strings"
	"unicode"

	"github.com/jaykbpark/ats-job-monitor/internal/filters"
	"github.com/jaykbpark/ats-job-monitor/internal/signals"
)

type Result struct {
	Matched      bool     `json:"matched"`
	MatchReasons []string `json:"matchReasons,omitempty"`
	HardFailures []string `json:"hardFailures,omitempty"`
}

func Evaluate(hardFilters filters.HardFilters, jobSignals signals.JobSignals) Result {
	matchReasons := make([]string, 0)
	hardFailures := make([]string, 0)

	if len(hardFilters.LocationsAny) > 0 {
		matchedLocation := ""
		for _, candidate := range hardFilters.LocationsAny {
			normalizedCandidate := normalizeText(candidate)
			if normalizedCandidate == "" {
				continue
			}
			if strings.Contains(jobSignals.NormalizedLocation, normalizedCandidate) {
				matchedLocation = candidate
				break
			}
		}

		if matchedLocation == "" {
			hardFailures = append(hardFailures, "location did not match any allowed locations")
		} else {
			matchReasons = append(matchReasons, "matched location: "+matchedLocation)
		}
	}

	if hardFilters.RemoteOnly {
		if !jobSignals.IsRemote {
			hardFailures = append(hardFailures, "job is not remote")
		} else {
			matchReasons = append(matchReasons, "matched remote-only requirement")
		}
	}

	if len(hardFilters.EmploymentTypes) > 0 {
		if jobSignals.NormalizedEmploymentType == "unknown" {
			hardFailures = append(hardFailures, "employment type is unknown")
		} else if !containsNormalizedEmploymentType(hardFilters.EmploymentTypes, jobSignals.NormalizedEmploymentType) {
			hardFailures = append(hardFailures, "employment type did not match allowed values")
		} else {
			matchReasons = append(matchReasons, "matched employment type: "+jobSignals.NormalizedEmploymentType)
		}
	}

	if len(hardFilters.SeniorityAny) > 0 {
		if jobSignals.Seniority == "unknown" {
			hardFailures = append(hardFailures, "seniority is unknown")
		} else if !containsNormalizedText(hardFilters.SeniorityAny, jobSignals.Seniority) {
			hardFailures = append(hardFailures, "seniority did not match allowed values")
		} else {
			matchReasons = append(matchReasons, "matched seniority: "+jobSignals.Seniority)
		}
	}

	hasExperienceFilters := hardFilters.MinYearsExperience != nil || hardFilters.MaxYearsExperience != nil
	if hasExperienceFilters {
		if jobSignals.ExperienceConfidence == "unknown" || (jobSignals.MinYearsExperience == nil && jobSignals.MaxYearsExperience == nil) {
			if !hardFilters.AllowUnknownExperience {
				hardFailures = append(hardFailures, "experience requirements are unknown")
			}
		} else {
			if hardFilters.MinYearsExperience != nil && jobSignals.MaxYearsExperience != nil && *jobSignals.MaxYearsExperience < *hardFilters.MinYearsExperience {
				hardFailures = append(hardFailures, "experience range is below the minimum allowed years")
			}

			if hardFilters.MaxYearsExperience != nil && jobSignals.MinYearsExperience != nil && *jobSignals.MinYearsExperience > *hardFilters.MaxYearsExperience {
				hardFailures = append(hardFailures, "experience range exceeds the maximum allowed years")
			}

			if len(hardFailures) == 0 || !containsExperienceFailure(hardFailures) {
				matchReasons = append(matchReasons, "matched years-of-experience requirements")
			}
		}
	}

	for _, keyword := range hardFilters.ExcludeKeywords {
		normalizedKeyword := normalizeText(keyword)
		if normalizedKeyword == "" {
			continue
		}
		if strings.Contains(jobSignals.SearchText, normalizedKeyword) {
			hardFailures = append(hardFailures, "matched excluded keyword: "+keyword)
		}
	}

	if len(hardFilters.IncludeKeywordsAny) > 0 {
		matchedKeywords := make([]string, 0)
		for _, keyword := range hardFilters.IncludeKeywordsAny {
			normalizedKeyword := normalizeText(keyword)
			if normalizedKeyword == "" {
				continue
			}
			if strings.Contains(jobSignals.SearchText, normalizedKeyword) {
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}

		if len(matchedKeywords) == 0 {
			hardFailures = append(hardFailures, "did not match any required keywords")
		} else {
			matchReasons = append(matchReasons, "matched keywords: "+strings.Join(matchedKeywords, ", "))
		}
	}

	return Result{
		Matched:      len(hardFailures) == 0,
		MatchReasons: matchReasons,
		HardFailures: hardFailures,
	}
}

func containsNormalizedText(values []string, target string) bool {
	normalizedTarget := normalizeText(target)
	for _, value := range values {
		if normalizeText(value) == normalizedTarget {
			return true
		}
	}

	return false
}

func containsNormalizedEmploymentType(values []string, target string) bool {
	normalizedTarget := normalizeEmploymentType(target)
	for _, value := range values {
		if normalizeEmploymentType(value) == normalizedTarget {
			return true
		}
	}

	return false
}

func containsExperienceFailure(failures []string) bool {
	for _, failure := range failures {
		if strings.Contains(failure, "experience") {
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
		return normalized
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
