package signals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
)

type liveSignalFixture struct {
	Name     string        `json:"name"`
	Provider string        `json:"provider"`
	BoardKey string        `json:"boardKey"`
	Job      providers.Job `json:"job"`
	Expected struct {
		NormalizedEmploymentType string `json:"normalizedEmploymentType"`
		Seniority                string `json:"seniority"`
		IsRemote                 bool   `json:"isRemote"`
		MinYearsExperience       *int   `json:"minYearsExperience,omitempty"`
		MaxYearsExperience       *int   `json:"maxYearsExperience,omitempty"`
		ExperienceConfidence     string `json:"experienceConfidence"`
	} `json:"expected"`
}

func TestDeriveLiveFixtures(t *testing.T) {
	fixtures := loadLiveSignalFixtures(t)

	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			got := Derive(fixture.Job)

			if got.NormalizedEmploymentType != fixture.Expected.NormalizedEmploymentType {
				t.Fatalf("NormalizedEmploymentType = %q, want %q", got.NormalizedEmploymentType, fixture.Expected.NormalizedEmploymentType)
			}

			if got.Seniority != fixture.Expected.Seniority {
				t.Fatalf("Seniority = %q, want %q", got.Seniority, fixture.Expected.Seniority)
			}

			if got.IsRemote != fixture.Expected.IsRemote {
				t.Fatalf("IsRemote = %t, want %t", got.IsRemote, fixture.Expected.IsRemote)
			}

			assertIntPtrEqual(t, got.MinYearsExperience, fixture.Expected.MinYearsExperience, "min years")
			assertIntPtrEqual(t, got.MaxYearsExperience, fixture.Expected.MaxYearsExperience, "max years")

			if got.ExperienceConfidence != fixture.Expected.ExperienceConfidence {
				t.Fatalf("ExperienceConfidence = %q, want %q", got.ExperienceConfidence, fixture.Expected.ExperienceConfidence)
			}
		})
	}
}

func loadLiveSignalFixtures(t *testing.T) []liveSignalFixture {
	t.Helper()

	path := filepath.Join("testdata", "live_signal_cases.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live signal fixtures: %v", err)
	}

	var fixtures []liveSignalFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("decode live signal fixtures: %v", err)
	}

	if len(fixtures) == 0 {
		t.Fatal("expected at least one live signal fixture")
	}

	return fixtures
}
