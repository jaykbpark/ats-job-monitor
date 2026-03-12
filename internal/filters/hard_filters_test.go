package filters

import "testing"

func TestNormalizeHardFiltersJSON(t *testing.T) {
	raw := `{
	  "locationsAny": [" Remote ", "Vancouver", "remote"],
	  "employmentTypes": ["Full-time", " Full-time "],
	  "includeKeywordsAny": ["backend", " platform "]
	}`

	normalizedJSON, parsed, err := NormalizeHardFiltersJSON(raw)
	if err != nil {
		t.Fatalf("normalize hard filters: %v", err)
	}

	if len(parsed.LocationsAny) != 2 {
		t.Fatalf("expected 2 normalized locations, got %d", len(parsed.LocationsAny))
	}

	if len(parsed.EmploymentTypes) != 1 {
		t.Fatalf("expected deduped employment types, got %d", len(parsed.EmploymentTypes))
	}

	if normalizedJSON == "" {
		t.Fatal("expected normalized JSON output")
	}
}

func TestParseHardFiltersRejectsUnknownFields(t *testing.T) {
	_, err := ParseHardFilters(`{"unknownField":true}`)
	if err == nil {
		t.Fatal("expected unknown field validation error")
	}
}

func TestParseHardFiltersRejectsInvalidExperienceRange(t *testing.T) {
	_, err := ParseHardFilters(`{"minYearsExperience":5,"maxYearsExperience":2}`)
	if err == nil {
		t.Fatal("expected invalid experience range error")
	}
}
