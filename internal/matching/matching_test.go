package matching

import (
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/filters"
	"github.com/jaykbpark/ats-job-monitor/internal/signals"
)

func TestEvaluateMatchesAllHardFilters(t *testing.T) {
	filters := filters.HardFilters{
		LocationsAny:       []string{"Remote", "Vancouver"},
		RemoteOnly:         true,
		EmploymentTypes:    []string{"Full Time"},
		IncludeKeywordsAny: []string{"backend", "platform"},
		ExcludeKeywords:    []string{"manager"},
		SeniorityAny:       []string{"senior"},
		MaxYearsExperience: intPtr(6),
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:               "senior backend platform engineer remote us",
		NormalizedLocation:       "remote us",
		IsRemote:                 true,
		NormalizedEmploymentType: "full-time",
		Seniority:                "senior",
		MinYearsExperience:       intPtr(5),
		ExperienceConfidence:     "high",
	})

	if !result.Matched {
		t.Fatalf("expected match, got failures: %#v", result.HardFailures)
	}

	if len(result.MatchReasons) == 0 {
		t.Fatal("expected non-empty match reasons")
	}
}

func TestEvaluateRejectsUnknownExperienceWhenRequired(t *testing.T) {
	filters := filters.HardFilters{
		MaxYearsExperience: intPtr(5),
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "backend engineer remote",
		NormalizedLocation:   "remote",
		Seniority:            "unknown",
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "experience requirements are unknown")
}

func TestEvaluateAllowsUnknownExperienceWhenConfigured(t *testing.T) {
	filters := filters.HardFilters{
		MaxYearsExperience:     intPtr(5),
		AllowUnknownExperience: true,
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "backend engineer remote",
		NormalizedLocation:   "remote",
		Seniority:            "unknown",
		ExperienceConfidence: "unknown",
	})

	if !result.Matched {
		t.Fatalf("expected match, got failures: %#v", result.HardFailures)
	}
}

func TestEvaluateRejectsExperienceAboveMaximum(t *testing.T) {
	filters := filters.HardFilters{
		MaxYearsExperience: intPtr(5),
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "staff backend engineer",
		MinYearsExperience:   intPtr(7),
		ExperienceConfidence: "high",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "experience range exceeds the maximum allowed years")
}

func TestEvaluateRejectsUnknownSeniorityWhenFiltered(t *testing.T) {
	filters := filters.HardFilters{
		SeniorityAny: []string{"senior", "staff"},
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "software engineer",
		Seniority:            "unknown",
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "seniority is unknown")
}

func TestEvaluateRejectsExcludedKeyword(t *testing.T) {
	filters := filters.HardFilters{
		ExcludeKeywords: []string{"manager"},
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "engineering manager remote us",
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "matched excluded keyword: manager")
}

func TestEvaluateRejectsMissingRequiredKeyword(t *testing.T) {
	filters := filters.HardFilters{
		IncludeKeywordsAny: []string{"golang", "distributed systems"},
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "frontend engineer remote",
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "did not match any required keywords")
}

func TestEvaluateRejectsEmploymentTypeMismatch(t *testing.T) {
	filters := filters.HardFilters{
		EmploymentTypes: []string{"full time"},
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:               "backend engineer remote",
		NormalizedEmploymentType: "contract",
		ExperienceConfidence:     "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "employment type did not match allowed values")
}

func TestEvaluateRejectsLocationMismatch(t *testing.T) {
	filters := filters.HardFilters{
		LocationsAny: []string{"Vancouver"},
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "backend engineer remote us",
		NormalizedLocation:   "san francisco ca",
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "location did not match any allowed locations")
}

func TestEvaluateRejectsNonRemoteWhenRemoteOnly(t *testing.T) {
	filters := filters.HardFilters{
		RemoteOnly: true,
	}

	result := Evaluate(filters, signals.JobSignals{
		SearchText:           "backend engineer san francisco",
		NormalizedLocation:   "san francisco ca",
		IsRemote:             false,
		ExperienceConfidence: "unknown",
	})

	if result.Matched {
		t.Fatal("expected match to fail")
	}

	assertFailureContains(t, result.HardFailures, "job is not remote")
}

func assertFailureContains(t *testing.T, failures []string, want string) {
	t.Helper()

	for _, failure := range failures {
		if failure == want {
			return
		}
	}

	t.Fatalf("expected failure %q in %#v", want, failures)
}

func intPtr(value int) *int {
	return &value
}
