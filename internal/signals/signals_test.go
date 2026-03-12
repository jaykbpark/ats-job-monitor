package signals

import (
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
)

func TestDeriveSignals(t *testing.T) {
	job := providers.Job{
		Title:          "Senior Backend Engineer",
		Department:     "Engineering",
		Team:           "Platform",
		Location:       "Remote - US",
		EmploymentType: "Regular Full Time (Salary)",
		MetadataJSON:   `{"workplaceType":"remote"}`,
		RawJSON:        `{"description":"Requires 5+ years of experience building backend systems"}`,
	}

	signals := Derive(job)

	if !signals.IsRemote {
		t.Fatal("expected job to be remote")
	}

	if signals.NormalizedEmploymentType != "full-time" {
		t.Fatalf("unexpected employment type: %q", signals.NormalizedEmploymentType)
	}

	if signals.Seniority != "senior" {
		t.Fatalf("unexpected seniority: %q", signals.Seniority)
	}

	if signals.MinYearsExperience == nil || *signals.MinYearsExperience != 5 {
		t.Fatalf("unexpected min years experience: %#v", signals.MinYearsExperience)
	}

	if signals.ExperienceConfidence != "high" {
		t.Fatalf("unexpected experience confidence: %q", signals.ExperienceConfidence)
	}
}

func TestDeriveSignalsRangeYears(t *testing.T) {
	job := providers.Job{
		Title:   "Backend Engineer",
		RawJSON: `{"description":"Looking for 2-4 years experience in backend systems"}`,
	}

	signals := Derive(job)

	if signals.MinYearsExperience == nil || *signals.MinYearsExperience != 2 {
		t.Fatalf("unexpected min years experience: %#v", signals.MinYearsExperience)
	}

	if signals.MaxYearsExperience == nil || *signals.MaxYearsExperience != 4 {
		t.Fatalf("unexpected max years experience: %#v", signals.MaxYearsExperience)
	}
}
